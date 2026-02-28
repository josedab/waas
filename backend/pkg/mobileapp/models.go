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
