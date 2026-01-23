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
	ID                 string          `json:"id" db:"id"`
	SourceID           string          `json:"source_id" db:"source_id"`
	TenantID           string          `json:"tenant_id" db:"tenant_id"`
	Provider           string          `json:"provider" db:"provider"`
	RawPayload         string          `json:"raw_payload" db:"raw_payload"`
	NormalizedPayload  string          `json:"normalized_payload,omitempty" db:"normalized_payload"`
	Headers            json.RawMessage `json:"headers" db:"headers"`
	SignatureValid     bool            `json:"signature_valid" db:"signature_valid"`
	Status             string          `json:"status" db:"status"`
	ErrorMessage       string          `json:"error_message,omitempty" db:"error_message"`
	ProcessedAt        *time.Time      `json:"processed_at,omitempty" db:"processed_at"`
	CreatedAt          time.Time       `json:"created_at" db:"created_at"`
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
