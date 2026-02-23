package mobileinspector

import "time"

// MobileSession represents an authenticated mobile session.
type MobileSession struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	UserID       string    `json:"user_id"`
	DeviceID     string    `json:"device_id"`
	Platform     string    `json:"platform"` // ios, android
	AppVersion   string    `json:"app_version"`
	BiometricKey string    `json:"-"` // stored but never exposed
	Token        string    `json:"token"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
	LastActiveAt time.Time `json:"last_active_at"`
}

// DeviceRegistration is the DTO for registering a mobile device.
type DeviceRegistration struct {
	DeviceID   string `json:"device_id" binding:"required"`
	Platform   string `json:"platform" binding:"required"`
	AppVersion string `json:"app_version"`
	PushToken  string `json:"push_token"`
}

// PushRegistration stores push notification tokens.
type PushRegistration struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	UserID    string    `json:"user_id"`
	DeviceID  string    `json:"device_id"`
	Platform  string    `json:"platform"`
	PushToken string    `json:"push_token"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

// EventFeedItem is a mobile-optimized event for the feed.
type EventFeedItem struct {
	ID          string    `json:"id"`
	EndpointID  string    `json:"endpoint_id"`
	EventType   string    `json:"event_type"`
	StatusCode  int       `json:"status_code"`
	Success     bool      `json:"success"`
	LatencyMs   int64     `json:"latency_ms"`
	PayloadSize int       `json:"payload_size"`
	Timestamp   time.Time `json:"timestamp"`
}

// SyncRequest is used for offline-first sync.
type SyncRequest struct {
	LastSyncAt string `json:"last_sync_at" form:"last_sync_at"` // RFC3339
	Limit      int    `json:"limit" form:"limit"`
}

// SyncResponse contains events since last sync.
type SyncResponse struct {
	Events     []EventFeedItem `json:"events"`
	SyncToken  string          `json:"sync_token"`
	HasMore    bool            `json:"has_more"`
	ServerTime string          `json:"server_time"`
}

// EndpointOverview is a mobile-optimized endpoint summary.
type EndpointOverview struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	URL             string  `json:"url"`
	HealthScore     float64 `json:"health_score"`
	SuccessRate     float64 `json:"success_rate"`
	TotalDeliveries int     `json:"total_deliveries"`
	FailedRecent    int     `json:"failed_recent"`
	AvgLatencyMs    float64 `json:"avg_latency_ms"`
}

// AlertConfig defines configurable alert thresholds.
type AlertConfig struct {
	ID                 string     `json:"id"`
	TenantID           string     `json:"tenant_id"`
	UserID             string     `json:"user_id"`
	FailureThreshold   int        `json:"failure_threshold"`
	LatencyThresholdMs int        `json:"latency_threshold_ms"`
	SuccessRateMin     float64    `json:"success_rate_min"`
	NotifyOnFailure    bool       `json:"notify_on_failure"`
	NotifyOnRecovery   bool       `json:"notify_on_recovery"`
	SnoozedUntil       *time.Time `json:"snoozed_until,omitempty"`
}

// AlertNotification is a mobile push alert.
type AlertNotification struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	UserID       string    `json:"user_id"`
	EndpointID   string    `json:"endpoint_id"`
	AlertType    string    `json:"alert_type"` // failure_spike, latency_high, success_rate_low, recovery
	Title        string    `json:"title"`
	Body         string    `json:"body"`
	Severity     string    `json:"severity"` // info, warning, critical
	Acknowledged bool      `json:"acknowledged"`
	CreatedAt    time.Time `json:"created_at"`
}
