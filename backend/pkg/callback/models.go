package callback

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CallbackRequest represents a webhook sent with an expected callback response
type CallbackRequest struct {
	ID            uuid.UUID       `json:"id" db:"id"`
	TenantID      uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	WebhookID     uuid.UUID       `json:"webhook_id" db:"webhook_id"`
	CorrelationID string          `json:"correlation_id" db:"correlation_id"`
	CallbackURL   string          `json:"callback_url" db:"callback_url"`
	Payload       json.RawMessage `json:"payload" db:"payload"`
	Headers       json.RawMessage `json:"headers" db:"headers"`
	TimeoutMs     int             `json:"timeout_ms" db:"timeout_ms"`
	Status        CallbackStatus  `json:"status" db:"status"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	ExpiresAt     time.Time       `json:"expires_at" db:"expires_at"`
}

// CallbackResponse represents a response received for a callback request
type CallbackResponse struct {
	ID            uuid.UUID       `json:"id" db:"id"`
	RequestID     uuid.UUID       `json:"request_id" db:"request_id"`
	CorrelationID string          `json:"correlation_id" db:"correlation_id"`
	StatusCode    int             `json:"status_code" db:"status_code"`
	Body          json.RawMessage `json:"body" db:"body"`
	Headers       json.RawMessage `json:"headers" db:"headers"`
	ReceivedAt    time.Time       `json:"received_at" db:"received_at"`
	LatencyMs     int64           `json:"latency_ms" db:"latency_ms"`
}

// CallbackStatus represents the status of a callback request
type CallbackStatus string

const (
	CallbackStatusPending   CallbackStatus = "pending"
	CallbackStatusWaiting   CallbackStatus = "waiting"
	CallbackStatusReceived  CallbackStatus = "received"
	CallbackStatusTimeout   CallbackStatus = "timeout"
	CallbackStatusFailed    CallbackStatus = "failed"
	CallbackStatusCancelled CallbackStatus = "cancelled"
)

// CorrelationEntry tracks the mapping between requests and responses
type CorrelationEntry struct {
	ID            uuid.UUID      `json:"id" db:"id"`
	CorrelationID string         `json:"correlation_id" db:"correlation_id"`
	TenantID      uuid.UUID      `json:"tenant_id" db:"tenant_id"`
	RequestID     uuid.UUID      `json:"request_id" db:"request_id"`
	ResponseID    *uuid.UUID     `json:"response_id" db:"response_id"`
	Status        CallbackStatus `json:"status" db:"status"`
	CreatedAt     time.Time      `json:"created_at" db:"created_at"`
	TTL           int            `json:"ttl" db:"ttl"` // seconds
}

// LongPollSession represents a long-polling session for receiving events
type LongPollSession struct {
	ID         uuid.UUID       `json:"id" db:"id"`
	TenantID   uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	EndpointID uuid.UUID       `json:"endpoint_id" db:"endpoint_id"`
	Filters    json.RawMessage `json:"filters" db:"filters"`
	TimeoutMs  int             `json:"timeout_ms" db:"timeout_ms"`
	Status     string          `json:"status" db:"status"` // active, closed, expired
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`
}

// CallbackPattern defines a reusable request-response webhook pattern
type CallbackPattern struct {
	ID                     uuid.UUID       `json:"id" db:"id"`
	TenantID               uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	Name                   string          `json:"name" db:"name"`
	Description            string          `json:"description" db:"description"`
	RequestTemplate        json.RawMessage `json:"request_template" db:"request_template"`
	ExpectedResponseSchema json.RawMessage `json:"expected_response_schema" db:"expected_response_schema"`
	TimeoutMs              int             `json:"timeout_ms" db:"timeout_ms"`
	MaxRetries             int             `json:"max_retries" db:"max_retries"`
	CreatedAt              time.Time       `json:"created_at" db:"created_at"`
}

// CallbackMetrics contains aggregated metrics for callback operations
type CallbackMetrics struct {
	TotalRequests    int     `json:"total_requests" db:"total_requests"`
	SuccessRate      float64 `json:"success_rate" db:"success_rate"`
	AvgLatencyMs     float64 `json:"avg_latency_ms" db:"avg_latency_ms"`
	TimeoutRate      float64 `json:"timeout_rate" db:"timeout_rate"`
	PendingCallbacks int     `json:"pending_callbacks" db:"pending_callbacks"`
}

// Long-poll session status constants
const (
	LongPollStatusActive  = "active"
	LongPollStatusClosed  = "closed"
	LongPollStatusExpired = "expired"
)

// CreateCallbackRequest is the request body to send a webhook with callback
type CreateCallbackRequest struct {
	WebhookID   string          `json:"webhook_id" binding:"required"`
	CallbackURL string          `json:"callback_url" binding:"required"`
	Payload     json.RawMessage `json:"payload" binding:"required"`
	Headers     json.RawMessage `json:"headers"`
	TimeoutMs   int             `json:"timeout_ms"`
}

// CreateLongPollRequest is the request body to create a long-poll session
type CreateLongPollRequest struct {
	EndpointID string          `json:"endpoint_id" binding:"required"`
	Filters    json.RawMessage `json:"filters"`
	TimeoutMs  int            `json:"timeout_ms"`
}

// RegisterPatternRequest is the request body to register a callback pattern
type RegisterPatternRequest struct {
	Name                   string          `json:"name" binding:"required"`
	Description            string          `json:"description"`
	RequestTemplate        json.RawMessage `json:"request_template"`
	ExpectedResponseSchema json.RawMessage `json:"expected_response_schema"`
	TimeoutMs              int             `json:"timeout_ms"`
	MaxRetries             int             `json:"max_retries"`
}
