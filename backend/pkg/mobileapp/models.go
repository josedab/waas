package mobileapp

import "time"

// DevicePlatform represents a mobile device platform.
type DevicePlatform string

const (
	PlatformIOS     DevicePlatform = "ios"
	PlatformAndroid DevicePlatform = "android"
)

// NotificationType represents the type of push notification.
type NotificationType string

const (
	NotificationDeliveryFailure NotificationType = "delivery_failure"
	NotificationEndpointDown    NotificationType = "endpoint_down"
	NotificationThresholdAlert  NotificationType = "threshold_alert"
	NotificationSecurityAlert   NotificationType = "security_alert"
)

// DeviceRegistration represents a registered mobile device.
type DeviceRegistration struct {
	ID          string             `json:"id"`
	TenantID    string             `json:"tenant_id"`
	DeviceToken string             `json:"device_token"`
	Platform    DevicePlatform     `json:"platform"`
	Name        string             `json:"name"`
	Enabled     bool               `json:"enabled"`
	Filters     NotificationFilter `json:"filters"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

// NotificationFilter defines which notifications a device receives.
type NotificationFilter struct {
	EndpointIDs []string `json:"endpoint_ids,omitempty"`
	EventTypes  []string `json:"event_types,omitempty"`
	MinSeverity string   `json:"min_severity,omitempty"`
}

// PushNotification represents a push notification sent to a device.
type PushNotification struct {
	ID       string                 `json:"id"`
	TenantID string                 `json:"tenant_id"`
	DeviceID string                 `json:"device_id"`
	Type     NotificationType       `json:"type"`
	Title    string                 `json:"title"`
	Body     string                 `json:"body"`
	Data     map[string]interface{} `json:"data,omitempty"`
	SentAt   time.Time              `json:"sent_at"`
	ReadAt   *time.Time             `json:"read_at,omitempty"`
}

// MobileDashboard provides a mobile-optimized overview of webhook activity.
type MobileDashboard struct {
	ActiveEndpoints  int               `json:"active_endpoints"`
	RecentDeliveries int               `json:"recent_deliveries"`
	FailureRate      float64           `json:"failure_rate"`
	AlertCount       int               `json:"alert_count"`
	TopEndpoints     []EndpointSummary `json:"top_endpoints"`
}

// EndpointSummary provides a compact endpoint overview for mobile.
type EndpointSummary struct {
	ID             string     `json:"id"`
	URL            string     `json:"url"`
	Status         string     `json:"status"`
	SuccessRate    float64    `json:"success_rate"`
	LastDeliveryAt *time.Time `json:"last_delivery_at,omitempty"`
}

// LivePayloadEvent represents a recent webhook payload for live viewing.
type LivePayloadEvent struct {
	DeliveryID string    `json:"delivery_id"`
	EndpointID string    `json:"endpoint_id"`
	EventType  string    `json:"event_type"`
	StatusCode int       `json:"status_code"`
	LatencyMs  int64     `json:"latency_ms"`
	Payload    string    `json:"payload"`
	Timestamp  time.Time `json:"timestamp"`
}

// RegisterDeviceRequest is the DTO for registering a mobile device.
type RegisterDeviceRequest struct {
	DeviceToken string         `json:"device_token" binding:"required"`
	Platform    DevicePlatform `json:"platform" binding:"required"`
	Name        string         `json:"name"`
}

// NotificationPreferencesRequest is the DTO for updating notification preferences.
type NotificationPreferencesRequest struct {
	Enabled *bool               `json:"enabled,omitempty"`
	Filters *NotificationFilter `json:"filters,omitempty"`
}

// ReplayRequest is the DTO for replaying a webhook delivery from mobile.
type ReplayRequest struct {
	DeliveryID string `json:"delivery_id" binding:"required"`
	EndpointID string `json:"endpoint_id" binding:"required"`
}

// ReplayResult describes the outcome of a delivery replay.
type ReplayResult struct {
	OriginalDeliveryID string    `json:"original_delivery_id"`
	NewDeliveryID      string    `json:"new_delivery_id"`
	EndpointID         string    `json:"endpoint_id"`
	Status             string    `json:"status"` // queued, delivered, failed
	StatusCode         int       `json:"status_code,omitempty"`
	ReplayedAt         time.Time `json:"replayed_at"`
}

// OfflineCacheEntry stores data for offline access.
type OfflineCacheEntry struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	DataType    string    `json:"data_type"` // delivery_log, endpoint_status, dashboard
	Data        string    `json:"data"`
	CachedAt    time.Time `json:"cached_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	SyncVersion int       `json:"sync_version"`
}

// OnCallIntegration configures PagerDuty/OpsGenie integration for mobile.
type OnCallIntegration struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id"`
	Provider  string `json:"provider"` // pagerduty, opsgenie
	APIKey    string `json:"api_key"`
	ServiceID string `json:"service_id,omitempty"`
	Enabled   bool   `json:"enabled"`
}

// OnCallStatus represents the current on-call status.
type OnCallStatus struct {
	Provider   string    `json:"provider"`
	OnCall     bool      `json:"on_call"`
	ShiftStart time.Time `json:"shift_start"`
	ShiftEnd   time.Time `json:"shift_end"`
	TeamName   string    `json:"team_name,omitempty"`
	EscLevel   int       `json:"escalation_level"`
}

// IncidentSummary provides a mobile-optimized incident view.
type IncidentSummary struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Severity     string    `json:"severity"`
	Status       string    `json:"status"` // triggered, acknowledged, resolved
	EndpointID   string    `json:"endpoint_id,omitempty"`
	FailureCount int       `json:"failure_count"`
	CreatedAt    time.Time `json:"created_at"`
}
