package pushbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Service provides push bridge operations
type Service struct {
	repo      Repository
	providers map[ProviderType]PushProvider
	config    *ServiceConfig
}

// ServiceConfig holds service configuration
type ServiceConfig struct {
	MaxDevicesPerTenant  int
	MaxMappingsPerTenant int
	DefaultTTL           time.Duration
	MaxRetries           int
	RetryDelay           time.Duration
}

// DefaultServiceConfig returns default configuration
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		MaxDevicesPerTenant:  10000,
		MaxMappingsPerTenant: 100,
		DefaultTTL:           24 * time.Hour,
		MaxRetries:           3,
		RetryDelay:           time.Minute,
	}
}

// PushProvider defines push provider interface
type PushProvider interface {
	Send(ctx context.Context, notification *PushNotification) (*ProviderResponse, error)
	ValidateToken(ctx context.Context, token string) error
}

// NewService creates a new push bridge service
func NewService(repo Repository, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}

	return &Service{
		repo:      repo,
		providers: make(map[ProviderType]PushProvider),
		config:    config,
	}
}

// RegisterProvider registers a push provider
func (s *Service) RegisterProvider(providerType ProviderType, provider PushProvider) {
	s.providers[providerType] = provider
}

// RegisterDevice registers a mobile device
func (s *Service) RegisterDevice(ctx context.Context, tenantID string, req *RegisterDeviceRequest) (*PushDevice, error) {
	// Check if device with same token exists
	existing, _ := s.repo.GetDeviceByToken(ctx, tenantID, req.PushToken)
	if existing != nil {
		// Update existing device
		existing.Platform = req.Platform
		existing.UserID = req.UserID
		if req.DeviceInfo != nil {
			existing.DeviceInfo = *req.DeviceInfo
		}
		if req.Preferences != nil {
			existing.Preferences = *req.Preferences
		}
		if req.Tags != nil {
			existing.Tags = req.Tags
		}
		if req.Metadata != nil {
			existing.Metadata = req.Metadata
		}
		existing.Status = DeviceActive
		now := time.Now()
		existing.LastActiveAt = &now
		existing.UpdatedAt = now

		if err := s.repo.SaveDevice(ctx, existing); err != nil {
			return nil, err
		}
		return existing, nil
	}

	device := &PushDevice{
		ID:           GenerateDeviceID(),
		TenantID:     tenantID,
		UserID:       req.UserID,
		Platform:     req.Platform,
		PushToken:    req.PushToken,
		Status:       DeviceActive,
		Tags:         req.Tags,
		Metadata:     req.Metadata,
		RegisteredAt: time.Now(),
		UpdatedAt:    time.Now(),
	}

	if req.DeviceInfo != nil {
		device.DeviceInfo = *req.DeviceInfo
	}

	if req.Preferences != nil {
		device.Preferences = *req.Preferences
	} else {
		device.Preferences = GetDefaultPreferences()
	}

	now := time.Now()
	device.LastActiveAt = &now

	if err := s.repo.SaveDevice(ctx, device); err != nil {
		return nil, err
	}

	// Deliver any queued notifications
	go s.deliverQueuedNotifications(context.Background(), device)

	return device, nil
}

// GetDevice retrieves a device
func (s *Service) GetDevice(ctx context.Context, tenantID, deviceID string) (*PushDevice, error) {
	return s.repo.GetDevice(ctx, tenantID, deviceID)
}

// ListDevices lists devices
func (s *Service) ListDevices(ctx context.Context, tenantID string, filter *DeviceFilter) ([]PushDevice, error) {
	return s.repo.ListDevices(ctx, tenantID, filter)
}

// UpdateDevice updates a device
func (s *Service) UpdateDevice(ctx context.Context, tenantID, deviceID string, req *UpdateDeviceRequest) (*PushDevice, error) {
	device, err := s.repo.GetDevice(ctx, tenantID, deviceID)
	if err != nil {
		return nil, err
	}

	if req.PushToken != nil {
		device.PushToken = *req.PushToken
	}
	if req.UserID != nil {
		device.UserID = *req.UserID
	}
	if req.DeviceInfo != nil {
		device.DeviceInfo = *req.DeviceInfo
	}
	if req.Preferences != nil {
		device.Preferences = *req.Preferences
	}
	if req.Tags != nil {
		device.Tags = req.Tags
	}
	if req.Metadata != nil {
		device.Metadata = req.Metadata
	}

	device.UpdatedAt = time.Now()

	if err := s.repo.SaveDevice(ctx, device); err != nil {
		return nil, err
	}

	return device, nil
}

// UnregisterDevice unregisters a device
func (s *Service) UnregisterDevice(ctx context.Context, tenantID, deviceID string) error {
	return s.repo.UpdateDeviceStatus(ctx, deviceID, DeviceUnregistered)
}

// DeleteDevice deletes a device
func (s *Service) DeleteDevice(ctx context.Context, tenantID, deviceID string) error {
	return s.repo.DeleteDevice(ctx, tenantID, deviceID)
}

// CreateMapping creates a push mapping
func (s *Service) CreateMapping(ctx context.Context, tenantID string, req *CreateMappingRequest) (*PushMapping, error) {
	mapping := &PushMapping{
		ID:          GenerateMappingID(),
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		WebhookID:   req.WebhookID,
		EventType:   req.EventType,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if req.Enabled != nil {
		mapping.Enabled = *req.Enabled
	}

	if req.Config != nil {
		mapping.Config = *req.Config
	} else {
		mapping.Config = GetDefaultMappingConfig()
	}

	if req.Template != nil {
		mapping.Template = *req.Template
	}

	if req.Targeting != nil {
		mapping.Targeting = *req.Targeting
	} else {
		mapping.Targeting = TargetingRules{Type: TargetAll}
	}

	if err := s.repo.SaveMapping(ctx, mapping); err != nil {
		return nil, err
	}

	return mapping, nil
}

// GetMapping retrieves a mapping
func (s *Service) GetMapping(ctx context.Context, tenantID, mappingID string) (*PushMapping, error) {
	return s.repo.GetMapping(ctx, tenantID, mappingID)
}

// ListMappings lists mappings
func (s *Service) ListMappings(ctx context.Context, tenantID string) ([]PushMapping, error) {
	return s.repo.ListMappings(ctx, tenantID)
}

// DeleteMapping deletes a mapping
func (s *Service) DeleteMapping(ctx context.Context, tenantID, mappingID string) error {
	return s.repo.DeleteMapping(ctx, tenantID, mappingID)
}

// SendPush sends a push notification
func (s *Service) SendPush(ctx context.Context, tenantID string, req *SendPushRequest) ([]PushNotification, error) {
	var mapping *PushMapping
	var err error

	// Get mapping if specified
	if req.MappingID != "" {
		mapping, err = s.repo.GetMapping(ctx, tenantID, req.MappingID)
		if err != nil {
			return nil, err
		}
	}

	// Build template
	template := PushTemplate{
		Title: req.Title,
		Body:  req.Body,
		Data:  req.Data,
		Image: req.Image,
	}
	if mapping != nil {
		template = mapping.Template
		if req.Title != "" {
			template.Title = req.Title
		}
		if req.Body != "" {
			template.Body = req.Body
		}
		if req.Data != nil {
			template.Data = req.Data
		}
	}

	// Get targeting rules
	targeting := &TargetingRules{Type: TargetAll}
	if req.Targeting != nil {
		targeting = req.Targeting
	} else if mapping != nil {
		targeting = &mapping.Targeting
	}

	// Get config
	config := GetDefaultMappingConfig()
	if req.Config != nil {
		config = *req.Config
	} else if mapping != nil {
		config = mapping.Config
	}

	// Resolve target devices
	devices, err := s.resolveTargets(ctx, tenantID, targeting)
	if err != nil {
		return nil, err
	}

	// Filter by preferences
	devices = s.filterByPreferences(devices, &config)

	// Send to each device
	var notifications []PushNotification
	for _, device := range devices {
		notif, err := s.sendToDevice(ctx, tenantID, &device, &template, &config, mapping)
		if err != nil {
			continue
		}
		notifications = append(notifications, *notif)
	}

	return notifications, nil
}

func (s *Service) resolveTargets(ctx context.Context, tenantID string, targeting *TargetingRules) ([]PushDevice, error) {
	var devices []PushDevice
	var err error

	switch targeting.Type {
	case TargetAll:
		devices, err = s.repo.ListDevices(ctx, tenantID, &DeviceFilter{
			Status: statusPtr(DeviceActive),
		})
	case TargetUsers:
		for _, userID := range targeting.UserIDs {
			userDevices, _ := s.repo.ListDevices(ctx, tenantID, &DeviceFilter{
				UserID: &userID,
				Status: statusPtr(DeviceActive),
			})
			devices = append(devices, userDevices...)
		}
	case TargetDevices:
		for _, deviceID := range targeting.DeviceIDs {
			device, _ := s.repo.GetDevice(ctx, tenantID, deviceID)
			if device != nil && device.Status == DeviceActive {
				devices = append(devices, *device)
			}
		}
	case TargetTags:
		devices, err = s.repo.ListDevices(ctx, tenantID, &DeviceFilter{
			Tags:   targeting.Tags,
			Status: statusPtr(DeviceActive),
		})
	case TargetSegments:
		for _, segmentID := range targeting.Segments {
			segDevices, _ := s.repo.GetDevicesInSegment(ctx, tenantID, segmentID)
			devices = append(devices, segDevices...)
		}
	}

	return devices, err
}

func statusPtr(s DeviceStatus) *DeviceStatus {
	return &s
}

func (s *Service) filterByPreferences(devices []PushDevice, config *MappingConfig) []PushDevice {
	var filtered []PushDevice
	now := time.Now()

	for _, device := range devices {
		if !device.Preferences.Enabled {
			continue
		}

		// Check quiet hours
		if config.RespectQuietHours && device.Preferences.QuietHoursStart != "" {
			if s.isInQuietHours(now, device.Preferences.QuietHoursStart, device.Preferences.QuietHoursEnd, device.DeviceInfo.Timezone) {
				continue
			}
		}

		// Check platform filter
		if len(config.Platform) > 0 {
			found := false
			for _, p := range config.Platform {
				if p == device.Platform {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		filtered = append(filtered, device)
	}

	return filtered
}

func (s *Service) isInQuietHours(now time.Time, start, end, timezone string) bool {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}

	localNow := now.In(loc)
	currentTime := localNow.Format("15:04")

	if start <= end {
		return currentTime >= start && currentTime <= end
	}
	return currentTime >= start || currentTime <= end
}

func (s *Service) sendToDevice(ctx context.Context, tenantID string, device *PushDevice, template *PushTemplate, config *MappingConfig, mapping *PushMapping) (*PushNotification, error) {
	// Build notification payload
	payload := s.buildPayload(device.Platform, template, config)

	notif := &PushNotification{
		ID:        GenerateNotificationID(),
		TenantID:  tenantID,
		Platform:  device.Platform,
		DeviceID:  device.ID,
		PushToken: device.PushToken,
		Status:    DeliveryPending,
		Payload:   payload,
		Attempts:  0,
		CreatedAt: time.Now(),
	}

	if mapping != nil {
		notif.MappingID = mapping.ID
		notif.WebhookID = mapping.WebhookID
	}

	// Save notification
	if err := s.repo.SaveNotification(ctx, notif); err != nil {
		return nil, err
	}

	// Get provider
	provider := s.getProviderForPlatform(device.Platform)
	if provider == nil {
		notif.Status = DeliveryFailed
		notif.Error = "No provider configured for platform"
		s.repo.SaveNotification(ctx, notif)
		return notif, nil
	}

	// Send
	response, err := provider.Send(ctx, notif)
	now := time.Now()
	notif.LastAttempt = &now
	notif.Attempts++

	if err != nil {
		notif.Status = DeliveryFailed
		notif.Error = err.Error()
		if response != nil {
			notif.Response = response
		}
	} else {
		notif.Status = DeliverySent
		notif.Response = response
	}

	s.repo.SaveNotification(ctx, notif)
	return notif, nil
}

func (s *Service) buildPayload(platform Platform, template *PushTemplate, config *MappingConfig) json.RawMessage {
	payload := map[string]any{
		"notification": map[string]any{
			"title": template.Title,
			"body":  template.Body,
		},
	}

	if template.Image != "" {
		payload["notification"].(map[string]any)["image"] = template.Image
	}

	if len(template.Data) > 0 {
		payload["data"] = template.Data
	}

	// Platform-specific options
	switch platform {
	case PlatformIOS:
		aps := map[string]any{
			"alert": map[string]any{
				"title": template.Title,
				"body":  template.Body,
			},
		}
		if template.Sound != "" {
			aps["sound"] = template.Sound
		}
		if template.Badge != nil {
			aps["badge"] = *template.Badge
		}
		if template.IOSThreadID != "" {
			aps["thread-id"] = template.IOSThreadID
		}
		if template.IOSInterruptionLevel != "" {
			aps["interruption-level"] = template.IOSInterruptionLevel
		}
		payload["aps"] = aps

	case PlatformAndroid:
		android := map[string]any{}
		if config.Priority == "high" {
			android["priority"] = "high"
		}
		if config.TTLSeconds > 0 {
			android["ttl"] = fmt.Sprintf("%ds", config.TTLSeconds)
		}
		if config.Collapsible && config.CollapseKey != "" {
			android["collapse_key"] = config.CollapseKey
		}
		if template.AndroidChannelID != "" {
			android["notification"] = map[string]any{
				"channel_id": template.AndroidChannelID,
			}
		}
		if template.AndroidColor != "" {
			if android["notification"] == nil {
				android["notification"] = map[string]any{}
			}
			android["notification"].(map[string]any)["color"] = template.AndroidColor
		}
		payload["android"] = android
	}

	jsonPayload, _ := json.Marshal(payload)
	return jsonPayload
}

func (s *Service) getProviderForPlatform(platform Platform) PushProvider {
	switch platform {
	case PlatformIOS:
		return s.providers[ProviderAPNS]
	case PlatformAndroid:
		return s.providers[ProviderFCM]
	case PlatformWeb:
		return s.providers[ProviderWebPush]
	case PlatformHuawei:
		return s.providers[ProviderHuawei]
	default:
		return nil
	}
}

func (s *Service) deliverQueuedNotifications(ctx context.Context, device *PushDevice) {
	queued, err := s.repo.GetQueuedNotifications(ctx, device.ID, 20)
	if err != nil {
		return
	}

	for _, q := range queued {
		var template PushTemplate
		json.Unmarshal(q.Notification, &template)

		config := GetDefaultMappingConfig()
		s.sendToDevice(ctx, device.TenantID, device, &template, &config, nil)
		s.repo.DeleteQueuedNotification(ctx, q.ID)
	}
}

// ProcessWebhook processes a webhook and sends push notifications
func (s *Service) ProcessWebhook(ctx context.Context, tenantID, webhookID string, payload []byte) ([]PushNotification, error) {
	// Get mappings for this webhook
	mappings, err := s.repo.GetMappingByWebhook(ctx, tenantID, webhookID)
	if err != nil || len(mappings) == 0 {
		return nil, nil
	}

	var allNotifications []PushNotification
	for _, mapping := range mappings {
		// Apply template with payload data
		template := s.applyTemplate(&mapping.Template, payload)

		// Get target devices
		devices, err := s.resolveTargets(ctx, tenantID, &mapping.Targeting)
		if err != nil {
			continue
		}

		devices = s.filterByPreferences(devices, &mapping.Config)

		// Send to each device
		for _, device := range devices {
			notif, err := s.sendToDevice(ctx, tenantID, &device, template, &mapping.Config, &mapping)
			if err != nil {
				continue
			}
			allNotifications = append(allNotifications, *notif)
		}
	}

	return allNotifications, nil
}

func (s *Service) applyTemplate(template *PushTemplate, payload []byte) *PushTemplate {
	result := *template

	// Parse payload as JSON
	var data map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		return &result
	}

	// Simple template substitution
	result.Title = s.substituteVariables(result.Title, data)
	result.Body = s.substituteVariables(result.Body, data)

	return &result
}

func (s *Service) substituteVariables(template string, data map[string]any) string {
	result := template
	for key, value := range data {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}
	return result
}

// GetNotification retrieves a notification
func (s *Service) GetNotification(ctx context.Context, tenantID, notifID string) (*PushNotification, error) {
	return s.repo.GetNotification(ctx, notifID)
}

// ListNotifications lists notifications
func (s *Service) ListNotifications(ctx context.Context, tenantID string, filter *NotificationFilter) ([]PushNotification, error) {
	return s.repo.ListNotifications(ctx, tenantID, filter)
}

// RecordOpen records notification open
func (s *Service) RecordOpen(ctx context.Context, tenantID, notifID string) error {
	return s.repo.UpdateNotificationStatus(ctx, notifID, DeliveryOpened, nil)
}

// GetStats retrieves push statistics
func (s *Service) GetStats(ctx context.Context, tenantID string, from, to time.Time) (*PushStats, error) {
	return s.repo.GetStats(ctx, tenantID, from, to)
}

// GetSupportedPlatforms returns supported platforms
func (s *Service) GetSupportedPlatforms() []PlatformInfo {
	return GetSupportedPlatforms()
}

// ListProviders lists configured providers
func (s *Service) ListProviders(ctx context.Context, tenantID string) ([]PushProviderConfig, error) {
	return s.repo.ListProviders(ctx, tenantID)
}
