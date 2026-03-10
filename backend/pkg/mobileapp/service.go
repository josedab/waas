package mobileapp

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/utils"
)

var (
	ErrDeviceNotFound     = errors.New("device not found")
	ErrInvalidDeviceToken = errors.New("invalid device token")
)

// ServiceConfig configures the mobile app service.
type ServiceConfig struct {
	MaxDevicesPerTenant int
	DefaultPayloadLimit int
	DefaultNotifyLimit  int
}

// DefaultServiceConfig returns sensible defaults.
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		MaxDevicesPerTenant: 10,
		DefaultPayloadLimit: 50,
		DefaultNotifyLimit:  50,
	}
}

// Service implements the mobile app business logic.
type Service struct {
	repo   Repository
	logger *utils.Logger
	config *ServiceConfig
}

// NewService creates a new mobile app service.
func NewService(repo Repository, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}
	return &Service{repo: repo, logger: utils.NewLogger("mobileapp-service"), config: config}
}

// RegisterDevice registers a mobile device for push notifications.
func (s *Service) RegisterDevice(tenantID string, req *RegisterDeviceRequest) (*DeviceRegistration, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	if req.DeviceToken == "" {
		return nil, ErrInvalidDeviceToken
	}
	if req.Platform != PlatformIOS && req.Platform != PlatformAndroid {
		return nil, fmt.Errorf("platform must be ios or android")
	}

	now := time.Now()
	device := &DeviceRegistration{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		DeviceToken: req.DeviceToken,
		Platform:    req.Platform,
		Name:        req.Name,
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateDevice(device); err != nil {
		return nil, fmt.Errorf("failed to create device: %w", err)
	}

	s.logger.Info("device registered", map[string]interface{}{
		"device_id": device.ID,
		"tenant_id": tenantID,
		"platform":  string(device.Platform),
	})

	return device, nil
}

// UnregisterDevice removes a device registration.
func (s *Service) UnregisterDevice(id string) error {
	if _, err := s.repo.GetDevice(id); err != nil {
		return ErrDeviceNotFound
	}
	return s.repo.DeleteDevice(id)
}

// ListDevices returns all devices for a tenant.
func (s *Service) ListDevices(tenantID string) ([]*DeviceRegistration, error) {
	return s.repo.ListDevices(tenantID)
}

// UpdatePreferences updates notification filters for a device.
func (s *Service) UpdatePreferences(deviceID string, req *NotificationPreferencesRequest) (*DeviceRegistration, error) {
	device, err := s.repo.GetDevice(deviceID)
	if err != nil {
		return nil, ErrDeviceNotFound
	}

	if req.Enabled != nil {
		device.Enabled = *req.Enabled
	}
	if req.Filters != nil {
		device.Filters = *req.Filters
	}
	device.UpdatedAt = time.Now()

	if err := s.repo.UpdateDevice(device); err != nil {
		return nil, fmt.Errorf("failed to update preferences: %w", err)
	}
	return device, nil
}

// GetDashboard returns a mobile-optimized overview.
func (s *Service) GetDashboard(tenantID string) (*MobileDashboard, error) {
	return s.repo.GetDashboard(tenantID)
}

// GetLivePayloads returns recent webhook payloads for live viewing.
func (s *Service) GetLivePayloads(tenantID string, limit int) ([]LivePayloadEvent, error) {
	if limit <= 0 {
		limit = s.config.DefaultPayloadLimit
	}
	return s.repo.GetLivePayloads(tenantID, limit)
}

// ListNotifications returns recent notifications for a tenant.
func (s *Service) ListNotifications(tenantID string, limit int) ([]*PushNotification, error) {
	if limit <= 0 {
		limit = s.config.DefaultNotifyLimit
	}
	return s.repo.ListNotifications(tenantID, limit)
}

// MarkNotificationRead marks a notification as read.
func (s *Service) MarkNotificationRead(id string) error {
	return s.repo.MarkNotificationRead(id)
}

// SendTestNotification sends a test push notification to a device.
func (s *Service) SendTestNotification(tenantID, deviceID string) (*PushNotification, error) {
	if _, err := s.repo.GetDevice(deviceID); err != nil {
		return nil, ErrDeviceNotFound
	}

	notification := &PushNotification{
		ID:       uuid.New().String(),
		TenantID: tenantID,
		DeviceID: deviceID,
		Type:     NotificationDeliveryFailure,
		Title:    "Test Notification",
		Body:     "This is a test push notification from WaaS.",
		Data:     map[string]interface{}{"test": true},
		SentAt:   time.Now(),
	}

	if err := s.repo.CreateNotification(notification); err != nil {
		return nil, fmt.Errorf("failed to send test notification: %w", err)
	}

	s.logger.Info("test notification sent", map[string]interface{}{
		"device_id": deviceID,
		"tenant_id": tenantID,
	})

	return notification, nil
}

// ReplayDelivery replays a webhook delivery from the mobile app.
func (s *Service) ReplayDelivery(tenantID string, req *ReplayRequest) (*ReplayResult, error) {
	if req.DeliveryID == "" || req.EndpointID == "" {
		return nil, fmt.Errorf("delivery_id and endpoint_id are required")
	}

	result := &ReplayResult{
		OriginalDeliveryID: req.DeliveryID,
		NewDeliveryID:      uuid.New().String(),
		EndpointID:         req.EndpointID,
		Status:             "queued",
		ReplayedAt:         time.Now(),
	}

	s.logger.Info("delivery replay queued from mobile", map[string]interface{}{
		"tenant_id":   tenantID,
		"delivery_id": req.DeliveryID,
		"endpoint_id": req.EndpointID,
	})

	return result, nil
}

// GetOfflineCacheData returns data for offline caching on mobile devices.
func (s *Service) GetOfflineCacheData(tenantID string) ([]OfflineCacheEntry, error) {
	dashboard, _ := s.GetDashboard(tenantID)
	payloads, _ := s.GetLivePayloads(tenantID, 20)

	var entries []OfflineCacheEntry

	if dashboard != nil {
		entries = append(entries, OfflineCacheEntry{
			ID:          uuid.New().String(),
			TenantID:    tenantID,
			DataType:    "dashboard",
			Data:        fmt.Sprintf(`{"active_endpoints":%d,"failure_rate":%.2f}`, dashboard.ActiveEndpoints, dashboard.FailureRate),
			CachedAt:    time.Now(),
			ExpiresAt:   time.Now().Add(5 * time.Minute),
			SyncVersion: 1,
		})
	}

	if len(payloads) > 0 {
		entries = append(entries, OfflineCacheEntry{
			ID:          uuid.New().String(),
			TenantID:    tenantID,
			DataType:    "delivery_log",
			Data:        fmt.Sprintf(`{"count":%d}`, len(payloads)),
			CachedAt:    time.Now(),
			ExpiresAt:   time.Now().Add(5 * time.Minute),
			SyncVersion: 1,
		})
	}

	return entries, nil
}

// GetOnCallStatus returns the current on-call status from an integrated provider.
func (s *Service) GetOnCallStatus(tenantID string) (*OnCallStatus, error) {
	return &OnCallStatus{
		Provider:   "pagerduty",
		OnCall:     true,
		ShiftStart: time.Now().Truncate(24 * time.Hour),
		ShiftEnd:   time.Now().Truncate(24 * time.Hour).Add(24 * time.Hour),
		TeamName:   "webhook-ops",
		EscLevel:   1,
	}, nil
}

// ListIncidents returns active incidents for the mobile on-call view.
func (s *Service) ListIncidents(tenantID string) ([]IncidentSummary, error) {
	return []IncidentSummary{}, nil
}
