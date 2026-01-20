package metaevents

import (
	"encoding/json"
	"time"
)

// EventType represents types of meta-events
type EventType string

const (
	// Delivery lifecycle events
	EventDeliveryAttempted EventType = "delivery.attempted"
	EventDeliverySucceeded EventType = "delivery.succeeded"
	EventDeliveryFailed    EventType = "delivery.failed"
	EventDeliveryRetrying  EventType = "delivery.retrying"
	EventDeliveryExhausted EventType = "delivery.exhausted"

	// Endpoint events
	EventEndpointCreated  EventType = "endpoint.created"
	EventEndpointUpdated  EventType = "endpoint.updated"
	EventEndpointDeleted  EventType = "endpoint.deleted"
	EventEndpointDisabled EventType = "endpoint.disabled"
	EventEndpointEnabled  EventType = "endpoint.enabled"

	// Health events
	EventEndpointHealthy   EventType = "endpoint.healthy"
	EventEndpointUnhealthy EventType = "endpoint.unhealthy"
	EventEndpointRecovered EventType = "endpoint.recovered"

	// Threshold events
	EventThresholdError   EventType = "threshold.error_rate"
	EventThresholdLatency EventType = "threshold.latency"
	EventThresholdVolume  EventType = "threshold.volume"

	// Anomaly events
	EventAnomalyDetected EventType = "anomaly.detected"

	// Security events
	EventSecurityViolation EventType = "security.violation"
	EventAPIKeyRotated     EventType = "security.api_key_rotated"
)

// MetaEvent represents a webhook event about webhooks
type MetaEvent struct {
	ID          string                 `json:"id" db:"id"`
	TenantID    string                 `json:"tenant_id" db:"tenant_id"`
	Type        EventType              `json:"type" db:"type"`
	Source      string                 `json:"source" db:"source"`
	SourceID    string                 `json:"source_id" db:"source_id"`
	Data        map[string]interface{} `json:"data" db:"data"`
	Metadata    map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	OccurredAt  time.Time              `json:"occurred_at" db:"occurred_at"`
	DeliveredAt *time.Time             `json:"delivered_at,omitempty" db:"delivered_at"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
}

// Subscription represents a meta-event subscription
type Subscription struct {
	ID          string       `json:"id" db:"id"`
	TenantID    string       `json:"tenant_id" db:"tenant_id"`
	Name        string       `json:"name" db:"name"`
	URL         string       `json:"url" db:"url"`
	Secret      string       `json:"-" db:"secret"`
	EventTypes  []EventType  `json:"event_types" db:"event_types"`
	Filters     *EventFilter `json:"filters,omitempty" db:"filters"`
	IsActive    bool         `json:"is_active" db:"is_active"`
	Headers     Headers      `json:"headers,omitempty" db:"headers"`
	RetryPolicy *RetryPolicy `json:"retry_policy,omitempty" db:"retry_policy"`
	CreatedAt   time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at" db:"updated_at"`
}

// EventFilter allows filtering meta-events before delivery
type EventFilter struct {
	EndpointIDs []string `json:"endpoint_ids,omitempty"`
	Severities  []string `json:"severities,omitempty"`
	Sources     []string `json:"sources,omitempty"`
}

// Headers represents custom HTTP headers
type Headers map[string]string

// RetryPolicy defines retry behavior for meta-event delivery
type RetryPolicy struct {
	MaxRetries      int           `json:"max_retries"`
	InitialInterval time.Duration `json:"initial_interval"`
	MaxInterval     time.Duration `json:"max_interval"`
	Multiplier      float64       `json:"multiplier"`
}

// Delivery represents a meta-event delivery attempt
type Delivery struct {
	ID             string     `json:"id" db:"id"`
	SubscriptionID string     `json:"subscription_id" db:"subscription_id"`
	EventID        string     `json:"event_id" db:"event_id"`
	TenantID       string     `json:"tenant_id" db:"tenant_id"`
	Status         string     `json:"status" db:"status"`
	Attempt        int        `json:"attempt" db:"attempt"`
	StatusCode     int        `json:"status_code,omitempty" db:"status_code"`
	ResponseBody   string     `json:"response_body,omitempty" db:"response_body"`
	Error          string     `json:"error,omitempty" db:"error"`
	LatencyMs      int        `json:"latency_ms" db:"latency_ms"`
	NextRetry      *time.Time `json:"next_retry,omitempty" db:"next_retry"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
}

// DeliveryPayload is what gets sent to subscribers
type DeliveryPayload struct {
	ID         string                 `json:"id"`
	Type       EventType              `json:"type"`
	Source     string                 `json:"source"`
	SourceID   string                 `json:"source_id"`
	TenantID   string                 `json:"tenant_id"`
	Data       map[string]interface{} `json:"data"`
	OccurredAt time.Time              `json:"occurred_at"`
	Timestamp  time.Time              `json:"timestamp"`
}

// DeliveryData for delivery events
type DeliveryData struct {
	WebhookID    string `json:"webhook_id"`
	EndpointID   string `json:"endpoint_id"`
	EndpointURL  string `json:"endpoint_url"`
	Attempt      int    `json:"attempt"`
	StatusCode   int    `json:"status_code,omitempty"`
	LatencyMs    int    `json:"latency_ms,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	ErrorType    string `json:"error_type,omitempty"`
}

// EndpointData for endpoint events
type EndpointData struct {
	EndpointID  string `json:"endpoint_id"`
	EndpointURL string `json:"endpoint_url"`
	Name        string `json:"name,omitempty"`
	OldStatus   string `json:"old_status,omitempty"`
	NewStatus   string `json:"new_status,omitempty"`
}

// ThresholdData for threshold events
type ThresholdData struct {
	Metric       string  `json:"metric"`
	Value        float64 `json:"value"`
	Threshold    float64 `json:"threshold"`
	EndpointID   string  `json:"endpoint_id,omitempty"`
	WindowPeriod string  `json:"window_period"`
}

// AnomalyData for anomaly events
type AnomalyData struct {
	Metric     string    `json:"metric"`
	Expected   float64   `json:"expected"`
	Actual     float64   `json:"actual"`
	Deviation  float64   `json:"deviation"`
	EndpointID string    `json:"endpoint_id,omitempty"`
	DetectedAt time.Time `json:"detected_at"`
}

// CreateSubscriptionRequest represents a request to create a subscription
type CreateSubscriptionRequest struct {
	Name        string       `json:"name" binding:"required"`
	URL         string       `json:"url" binding:"required,url"`
	EventTypes  []EventType  `json:"event_types" binding:"required,min=1"`
	Filters     *EventFilter `json:"filters,omitempty"`
	Headers     Headers      `json:"headers,omitempty"`
	RetryPolicy *RetryPolicy `json:"retry_policy,omitempty"`
}

// UpdateSubscriptionRequest represents a request to update a subscription
type UpdateSubscriptionRequest struct {
	Name        *string      `json:"name,omitempty"`
	URL         *string      `json:"url,omitempty"`
	EventTypes  []EventType  `json:"event_types,omitempty"`
	Filters     *EventFilter `json:"filters,omitempty"`
	Headers     Headers      `json:"headers,omitempty"`
	IsActive    *bool        `json:"is_active,omitempty"`
	RetryPolicy *RetryPolicy `json:"retry_policy,omitempty"`
}

// MarshalJSON for EventFilter
func (e EventFilter) MarshalJSON() ([]byte, error) {
	type alias EventFilter
	return json.Marshal(alias(e))
}

// DefaultRetryPolicy returns a sensible default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:      5,
		InitialInterval: 30 * time.Second,
		MaxInterval:     1 * time.Hour,
		Multiplier:      2.0,
	}
}
