package mobileinspector

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/utils"
)

// ServiceConfig configures the mobile inspector service.
type ServiceConfig struct {
	SessionDuration   time.Duration
	MaxDevicesPerUser int
	DefaultFeedLimit  int
	MaxAlertSnooze    time.Duration
}

// DefaultServiceConfig returns sensible defaults.
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		SessionDuration:   30 * 24 * time.Hour,
		MaxDevicesPerUser: 5,
		DefaultFeedLimit:  50,
		MaxAlertSnooze:    24 * time.Hour,
	}
}

// Service implements the mobile inspector business logic.
type Service struct {
	repo   Repository
	logger *utils.Logger
	config *ServiceConfig
}

// NewService creates a new mobile inspector service.
func NewService(repo Repository, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}
	return &Service{repo: repo, logger: utils.NewLogger("mobileinspector-service"), config: config}
}

// RegisterDevice creates a new mobile session for a device.
func (s *Service) RegisterDevice(tenantID, userID string, reg *DeviceRegistration) (*MobileSession, error) {
	if tenantID == "" || userID == "" {
		return nil, fmt.Errorf("tenant_id and user_id are required")
	}
	if reg.DeviceID == "" || reg.Platform == "" {
		return nil, fmt.Errorf("device_id and platform are required")
	}
	if reg.Platform != "ios" && reg.Platform != "android" {
		return nil, fmt.Errorf("platform must be ios or android")
	}

	session := &MobileSession{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		UserID:       userID,
		DeviceID:     reg.DeviceID,
		Platform:     reg.Platform,
		AppVersion:   reg.AppVersion,
		Token:        generateMobileToken(),
		ExpiresAt:    time.Now().Add(s.config.SessionDuration),
		CreatedAt:    time.Now(),
		LastActiveAt: time.Now(),
	}

	if err := s.repo.CreateSession(session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Register push token if provided
	if reg.PushToken != "" {
		pushReg := &PushRegistration{
			ID:        uuid.New().String(),
			TenantID:  tenantID,
			UserID:    userID,
			DeviceID:  reg.DeviceID,
			Platform:  reg.Platform,
			PushToken: reg.PushToken,
			Enabled:   true,
			CreatedAt: time.Now(),
		}
		if err := s.repo.CreatePushRegistration(pushReg); err != nil {
			s.logger.Error("failed to create push registration", map[string]interface{}{"error": err.Error(), "device_id": reg.DeviceID})
		}
	}

	return session, nil
}

// ValidateSession checks if a session token is valid.
func (s *Service) ValidateSession(token string) (*MobileSession, error) {
	session, err := s.repo.GetSessionByToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid session: %w", err)
	}
	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}
	session.LastActiveAt = time.Now()
	return session, nil
}

// GetEventFeed returns events since last sync for offline-first support.
func (s *Service) GetEventFeed(tenantID string, req *SyncRequest) (*SyncResponse, error) {
	since := time.Time{}
	if req.LastSyncAt != "" {
		parsed, err := time.Parse(time.RFC3339, req.LastSyncAt)
		if err != nil {
			return nil, fmt.Errorf("invalid last_sync_at: %w", err)
		}
		since = parsed
	}

	limit := req.Limit
	if limit <= 0 {
		limit = s.config.DefaultFeedLimit
	}

	events, err := s.repo.GetEventFeed(tenantID, since, limit+1)
	if err != nil {
		return nil, err
	}

	hasMore := len(events) > limit
	if hasMore {
		events = events[:limit]
	}

	return &SyncResponse{
		Events:     events,
		SyncToken:  uuid.New().String(),
		HasMore:    hasMore,
		ServerTime: time.Now().Format(time.RFC3339),
	}, nil
}

// GetEndpointOverviews returns mobile-optimized endpoint summaries.
func (s *Service) GetEndpointOverviews(tenantID string) ([]EndpointOverview, error) {
	return s.repo.GetEndpointOverviews(tenantID)
}

// UpdateAlertConfig saves alert threshold configuration.
func (s *Service) UpdateAlertConfig(userID string, cfg *AlertConfig) (*AlertConfig, error) {
	cfg.UserID = userID
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.LatencyThresholdMs <= 0 {
		cfg.LatencyThresholdMs = 5000
	}
	if cfg.SuccessRateMin <= 0 {
		cfg.SuccessRateMin = 95.0
	}
	if err := s.repo.SaveAlertConfig(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// GetAlertConfig returns current alert configuration.
func (s *Service) GetAlertConfig(userID string) (*AlertConfig, error) {
	return s.repo.GetAlertConfig(userID)
}

// GetAlerts returns recent alerts for a user.
func (s *Service) GetAlerts(userID string, limit int) ([]*AlertNotification, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListAlerts(userID, limit)
}

// AcknowledgeAlert marks an alert as acknowledged.
func (s *Service) AcknowledgeAlert(alertID string) error {
	return s.repo.AcknowledgeAlert(alertID)
}

// SnoozeAlerts snoozes alerts for a duration.
func (s *Service) SnoozeAlerts(userID string, duration time.Duration) (*AlertConfig, error) {
	if duration > s.config.MaxAlertSnooze {
		duration = s.config.MaxAlertSnooze
	}

	cfg, err := s.repo.GetAlertConfig(userID)
	if err != nil {
		return nil, err
	}

	snoozeUntil := time.Now().Add(duration)
	cfg.SnoozedUntil = &snoozeUntil
	if err := s.repo.SaveAlertConfig(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Logout invalidates a mobile session.
func (s *Service) Logout(sessionID string) error {
	return s.repo.DeleteSession(sessionID)
}

func generateMobileToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return "mob_" + hex.EncodeToString(b)
}
