package receiverdash

import "time"

// ReceiverToken grants read-only access to delivery data for specific endpoints.
type ReceiverToken struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	Token       string     `json:"token"`
	EndpointIDs []string   `json:"endpoint_ids"`
	Label       string     `json:"label"`
	Scopes      []string   `json:"scopes"`
	ExpiresAt   time.Time  `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
}

// ValidScopes defines the allowed scopes for receiver tokens.
var ValidScopes = []string{
	"read:deliveries",
	"read:retries",
	"read:payloads",
	"read:health",
	"read:metrics",
}

// DeliveryRecord represents a single delivery attempt visible to the receiver.
type DeliveryRecord struct {
	ID           string    `json:"id"`
	EndpointID   string    `json:"endpoint_id"`
	EventType    string    `json:"event_type"`
	StatusCode   int       `json:"status_code"`
	Success      bool      `json:"success"`
	Latency      int64     `json:"latency_ms"`
	AttemptCount int       `json:"attempt_count"`
	PayloadSize  int       `json:"payload_size_bytes"`
	CreatedAt    time.Time `json:"created_at"`
}

// RetryStatus tracks the retry state for a failed delivery.
type RetryStatus struct {
	DeliveryID    string     `json:"delivery_id"`
	EndpointID    string     `json:"endpoint_id"`
	CurrentState  string     `json:"current_state"` // pending, retrying, exhausted, succeeded
	AttemptCount  int        `json:"attempt_count"`
	MaxAttempts   int        `json:"max_attempts"`
	NextRetryAt   *time.Time `json:"next_retry_at,omitempty"`
	LastError     string     `json:"last_error,omitempty"`
	LastAttemptAt time.Time  `json:"last_attempt_at"`
}

// EndpointHealth provides an aggregate health score for an endpoint.
type EndpointHealth struct {
	EndpointID       string  `json:"endpoint_id"`
	HealthScore      float64 `json:"health_score"` // 0.0 - 100.0
	SuccessRate      float64 `json:"success_rate"` // percentage
	AvgLatencyMs     float64 `json:"avg_latency_ms"`
	P95LatencyMs     float64 `json:"p95_latency_ms"`
	TotalDeliveries  int     `json:"total_deliveries"`
	FailedDeliveries int     `json:"failed_deliveries"`
	ActiveRetries    int     `json:"active_retries"`
	Period           string  `json:"period"` // 1h, 24h, 7d
}

// PayloadInspection contains the payload details for a specific delivery.
type PayloadInspection struct {
	DeliveryID  string            `json:"delivery_id"`
	EndpointID  string            `json:"endpoint_id"`
	EventType   string            `json:"event_type"`
	Headers     map[string]string `json:"headers"`
	Body        string            `json:"body"`
	ContentType string            `json:"content_type"`
	Size        int               `json:"size_bytes"`
	ReceivedAt  time.Time         `json:"received_at"`
}

// HealthSummary provides a summary across all endpoints for a receiver token.
type HealthSummary struct {
	Endpoints       []EndpointHealth `json:"endpoints"`
	OverallScore    float64          `json:"overall_score"`
	OverallSuccess  float64          `json:"overall_success_rate"`
	TotalDeliveries int              `json:"total_deliveries"`
	TotalFailed     int              `json:"total_failed"`
	ActiveRetries   int              `json:"active_retries"`
	Period          string           `json:"period"`
}

// CreateTokenRequest is the DTO for creating a new receiver token.
type CreateTokenRequest struct {
	EndpointIDs []string `json:"endpoint_ids" binding:"required"`
	Label       string   `json:"label" binding:"required"`
	Scopes      []string `json:"scopes" binding:"required"`
	ExpiresIn   string   `json:"expires_in"` // e.g. "24h", "7d", "30d"
}

// DeliveryHistoryRequest is the query params for delivery history.
type DeliveryHistoryRequest struct {
	EndpointID string `form:"endpoint_id"`
	EventType  string `form:"event_type"`
	Status     string `form:"status"` // success, failed, all
	Limit      int    `form:"limit"`
	Offset     int    `form:"offset"`
	Period     string `form:"period"` // 1h, 24h, 7d, 30d
}
