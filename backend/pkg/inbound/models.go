package inbound

import (
	"encoding/json"
	"time"
)

// Provider constants
const (
	ProviderStripe   = "stripe"
	ProviderGitHub   = "github"
	ProviderTwilio   = "twilio"
	ProviderShopify  = "shopify"
	ProviderSlack    = "slack"
	ProviderSendGrid = "sendgrid"
	ProviderCustom   = "custom"
)

// Status constants
const (
	SourceStatusActive   = "active"
	SourceStatusPaused   = "paused"
	SourceStatusDisabled = "disabled"
)

// Event status constants
const (
	EventStatusReceived  = "received"
	EventStatusValidated = "validated"
	EventStatusRouted    = "routed"
	EventStatusFailed    = "failed"
)

// Destination type constants
const (
	DestinationHTTP     = "http"
	DestinationQueue    = "queue"
	DestinationInternal = "internal"
)

// InboundSource represents a configured inbound webhook source
type InboundSource struct {
	ID                    string    `json:"id" db:"id"`
	TenantID              string    `json:"tenant_id" db:"tenant_id"`
	Name                  string    `json:"name" db:"name"`
	Provider              string    `json:"provider" db:"provider"`
	VerificationSecret    string    `json:"verification_secret,omitempty" db:"verification_secret"`
	VerificationHeader    string    `json:"verification_header,omitempty" db:"verification_header"`
	VerificationAlgorithm string    `json:"verification_algorithm,omitempty" db:"verification_algorithm"`
	Status                string    `json:"status" db:"status"`
	CreatedAt             time.Time `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time `json:"updated_at" db:"updated_at"`
}

// RoutingRule defines how inbound events are routed to destinations
type RoutingRule struct {
	ID                string `json:"id" db:"id"`
	SourceID          string `json:"source_id" db:"source_id"`
	FilterExpression  string `json:"filter_expression,omitempty" db:"filter_expression"`
	DestinationType   string `json:"destination_type" db:"destination_type"`
	DestinationConfig string `json:"destination_config" db:"destination_config"`
	Priority          int    `json:"priority" db:"priority"`
	Active            bool   `json:"active" db:"active"`
}

// InboundEvent records a received inbound webhook event
type InboundEvent struct {
	ID                string          `json:"id" db:"id"`
	SourceID          string          `json:"source_id" db:"source_id"`
	TenantID          string          `json:"tenant_id" db:"tenant_id"`
	Provider          string          `json:"provider" db:"provider"`
	RawPayload        string          `json:"raw_payload" db:"raw_payload"`
	NormalizedPayload string          `json:"normalized_payload,omitempty" db:"normalized_payload"`
	Headers           json.RawMessage `json:"headers" db:"headers"`
	SignatureValid    bool            `json:"signature_valid" db:"signature_valid"`
	Status            string          `json:"status" db:"status"`
	ErrorMessage      string          `json:"error_message,omitempty" db:"error_message"`
	ProcessedAt       *time.Time      `json:"processed_at,omitempty" db:"processed_at"`
	CreatedAt         time.Time       `json:"created_at" db:"created_at"`
}

// ProviderConfig holds provider-specific signature verification settings
type ProviderConfig struct {
	Secret    string `json:"secret"`
	Header    string `json:"header"`
	Algorithm string `json:"algorithm"`
}

// DestinationConfigData holds destination routing configuration
type DestinationConfigData struct {
	URL     string            `json:"url,omitempty"`
	Method  string            `json:"method,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Queue   string            `json:"queue,omitempty"`
}

// CreateSourceRequest is the request DTO for creating an inbound source
type CreateSourceRequest struct {
	Name                  string `json:"name" binding:"required,min=1,max=255"`
	Provider              string `json:"provider" binding:"required,oneof=stripe github twilio shopify slack sendgrid custom"`
	VerificationSecret    string `json:"verification_secret,omitempty"`
	VerificationHeader    string `json:"verification_header,omitempty"`
	VerificationAlgorithm string `json:"verification_algorithm,omitempty"`
}

// UpdateSourceRequest is the request DTO for updating an inbound source
type UpdateSourceRequest struct {
	Name                  string `json:"name,omitempty"`
	VerificationSecret    string `json:"verification_secret,omitempty"`
	VerificationHeader    string `json:"verification_header,omitempty"`
	VerificationAlgorithm string `json:"verification_algorithm,omitempty"`
	Status                string `json:"status,omitempty"`
}

// CreateRoutingRuleRequest is the request DTO for creating a routing rule
type CreateRoutingRuleRequest struct {
	FilterExpression  string `json:"filter_expression,omitempty"`
	DestinationType   string `json:"destination_type" binding:"required,oneof=http queue internal"`
	DestinationConfig string `json:"destination_config" binding:"required"`
	Priority          int    `json:"priority"`
	Active            bool   `json:"active"`
}

// TransformRule defines a payload transformation to apply
type TransformRule struct {
	ID          string `json:"id" db:"id"`
	SourceID    string `json:"source_id" db:"source_id"`
	Name        string `json:"name" db:"name"`
	Expression  string `json:"expression" db:"expression"` // JSONPath or JS expression
	TargetField string `json:"target_field" db:"target_field"`
	Active      bool   `json:"active" db:"active"`
	Priority    int    `json:"priority" db:"priority"`
}

// InboundDLQEntry represents a failed event in the dead letter queue
type InboundDLQEntry struct {
	ID           string    `json:"id" db:"id"`
	EventID      string    `json:"event_id" db:"event_id"`
	SourceID     string    `json:"source_id" db:"source_id"`
	TenantID     string    `json:"tenant_id" db:"tenant_id"`
	RawPayload   string    `json:"raw_payload" db:"raw_payload"`
	ErrorMessage string    `json:"error_message" db:"error_message"`
	AttemptCount int       `json:"attempt_count" db:"attempt_count"`
	LastAttempt  time.Time `json:"last_attempt" db:"last_attempt"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	Replayed     bool      `json:"replayed" db:"replayed"`
}

// ProviderHealth tracks the health of an inbound provider connection
type ProviderHealth struct {
	SourceID          string     `json:"source_id"`
	Provider          string     `json:"provider"`
	Status            string     `json:"status"` // healthy, degraded, down
	SuccessRate       float64    `json:"success_rate"`
	AvgLatencyMs      int        `json:"avg_latency_ms"`
	EventsLast24h     int64      `json:"events_last_24h"`
	ErrorsLast24h     int64      `json:"errors_last_24h"`
	LastEventAt       *time.Time `json:"last_event_at,omitempty"`
	LastErrorAt       *time.Time `json:"last_error_at,omitempty"`
	ConsecutiveErrors int        `json:"consecutive_errors"`
}

// RateLimitConfig configures rate limiting per source
type RateLimitConfig struct {
	SourceID       string `json:"source_id" db:"source_id"`
	RequestsPerMin int    `json:"requests_per_minute" db:"requests_per_minute"`
	BurstSize      int    `json:"burst_size" db:"burst_size"`
	CurrentCount   int    `json:"current_count"`
	ThrottledCount int64  `json:"throttled_count"`
	Enabled        bool   `json:"enabled" db:"enabled"`
}

// ContentRoute represents a content-based routing destination
type ContentRoute struct {
	ID              string `json:"id" db:"id"`
	SourceID        string `json:"source_id" db:"source_id"`
	Name            string `json:"name" db:"name"`
	MatchExpression string `json:"match_expression" db:"match_expression"` // JSONPath condition
	MatchValue      string `json:"match_value" db:"match_value"`
	DestinationType string `json:"destination_type" db:"destination_type"`
	DestinationURL  string `json:"destination_url" db:"destination_url"`
	FanOut          bool   `json:"fan_out" db:"fan_out"` // if true, event goes to ALL matching routes
	Active          bool   `json:"active" db:"active"`
}

// CreateContentRouteRequest is the request DTO for creating a content route
type CreateContentRouteRequest struct {
	Name            string `json:"name" binding:"required"`
	MatchExpression string `json:"match_expression" binding:"required"`
	MatchValue      string `json:"match_value"`
	DestinationType string `json:"destination_type" binding:"required,oneof=http queue internal"`
	DestinationURL  string `json:"destination_url" binding:"required"`
	FanOut          bool   `json:"fan_out"`
}

// CreateTransformRuleRequest is the request DTO for creating a transform rule
type CreateTransformRuleRequest struct {
	Name        string `json:"name" binding:"required"`
	Expression  string `json:"expression" binding:"required"`
	TargetField string `json:"target_field" binding:"required"`
	Priority    int    `json:"priority"`
}

// InboundStats aggregates statistics for a source or tenant
type InboundStats struct {
	TotalEvents    int64   `json:"total_events"`
	ValidatedCount int64   `json:"validated_count"`
	RoutedCount    int64   `json:"routed_count"`
	FailedCount    int64   `json:"failed_count"`
	DLQCount       int64   `json:"dlq_count"`
	AvgLatencyMs   float64 `json:"avg_latency_ms"`
	SuccessRate    float64 `json:"success_rate"`
}
