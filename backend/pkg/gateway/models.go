package gateway

import (
	"encoding/json"
	"time"
)

// Provider represents a webhook provider (e.g., Stripe, GitHub)
type Provider struct {
	ID              string          `json:"id" db:"id"`
	TenantID        string          `json:"tenant_id" db:"tenant_id"`
	Name            string          `json:"name" db:"name"`
	Type            string          `json:"type" db:"type"` // stripe, github, shopify, etc.
	Description     string          `json:"description,omitempty" db:"description"`
	IsActive        bool            `json:"is_active" db:"is_active"`
	SignatureConfig json.RawMessage `json:"signature_config,omitempty" db:"signature_config"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at" db:"updated_at"`
}

// SignatureConfig contains provider-specific signature verification settings
type SignatureConfig struct {
	SecretKey        string `json:"secret_key,omitempty"`        // Encrypted secret for signature verification
	HeaderName       string `json:"header_name,omitempty"`       // Header containing the signature
	TimestampHeader  string `json:"timestamp_header,omitempty"`  // Header containing timestamp (if applicable)
	Algorithm        string `json:"algorithm,omitempty"`         // hmac-sha256, hmac-sha1, etc.
	SignaturePrefix  string `json:"signature_prefix,omitempty"`  // Prefix to strip from signature
	ToleranceSeconds int    `json:"tolerance_seconds,omitempty"` // Timestamp tolerance for replay protection
}

// InboundWebhook represents a received webhook
type InboundWebhook struct {
	ID              string            `json:"id" db:"id"`
	TenantID        string            `json:"tenant_id" db:"tenant_id"`
	ProviderID      string            `json:"provider_id" db:"provider_id"`
	ProviderType    string            `json:"provider_type" db:"provider_type"`
	EventType       string            `json:"event_type,omitempty" db:"event_type"`
	Payload         json.RawMessage   `json:"payload" db:"payload"`
	Headers         map[string]string `json:"headers" db:"-"`
	HeadersJSON     json.RawMessage   `json:"-" db:"headers"`
	RawBody         []byte            `json:"-" db:"raw_body"`
	SignatureValid  bool              `json:"signature_valid" db:"signature_valid"`
	ProcessedAt     *time.Time        `json:"processed_at,omitempty" db:"processed_at"`
	CreatedAt       time.Time         `json:"created_at" db:"created_at"`
}

// RoutingRule defines how to route inbound webhooks
type RoutingRule struct {
	ID           string          `json:"id" db:"id"`
	TenantID     string          `json:"tenant_id" db:"tenant_id"`
	ProviderID   string          `json:"provider_id" db:"provider_id"`
	Name         string          `json:"name" db:"name"`
	Description  string          `json:"description,omitempty" db:"description"`
	Priority     int             `json:"priority" db:"priority"`
	IsActive     bool            `json:"is_active" db:"is_active"`
	Conditions   json.RawMessage `json:"conditions" db:"conditions"`   // Filtering conditions
	Destinations json.RawMessage `json:"destinations" db:"destinations"` // Where to route
	Transform    json.RawMessage `json:"transform,omitempty" db:"transform"` // Optional transformation
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at" db:"updated_at"`
}

// RoutingCondition defines a condition for routing
type RoutingCondition struct {
	Field    string `json:"field"`    // JSON path or header name
	Operator string `json:"operator"` // equals, contains, matches, exists
	Value    string `json:"value"`
}

// RoutingDestination defines where to route a webhook
type RoutingDestination struct {
	Type       string `json:"type"`        // endpoint, url, queue
	EndpointID string `json:"endpoint_id,omitempty"`
	URL        string `json:"url,omitempty"`
	QueueName  string `json:"queue_name,omitempty"`
}

// FanoutResult represents the result of fanning out a webhook
type FanoutResult struct {
	InboundID    string            `json:"inbound_id"`
	TotalRouted  int               `json:"total_routed"`
	TotalFailed  int               `json:"total_failed"`
	Destinations []DestinationResult `json:"destinations"`
}

// DestinationResult represents the result for a single destination
type DestinationResult struct {
	RuleID     string `json:"rule_id"`
	RuleName   string `json:"rule_name"`
	Type       string `json:"type"`
	Target     string `json:"target"`
	DeliveryID string `json:"delivery_id,omitempty"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
}

// ProviderType constants
const (
	ProviderTypeStripe    = "stripe"
	ProviderTypeGitHub    = "github"
	ProviderTypeShopify   = "shopify"
	ProviderTypeTwilio    = "twilio"
	ProviderTypeSlack     = "slack"
	ProviderTypeSendGrid  = "sendgrid"
	ProviderTypePaddle    = "paddle"
	ProviderTypeCustom    = "custom"
)

// CreateProviderRequest represents a request to create a provider
type CreateProviderRequest struct {
	Name            string          `json:"name" binding:"required,min=1,max=255"`
	Type            string          `json:"type" binding:"required"`
	Description     string          `json:"description,omitempty"`
	SignatureConfig *SignatureConfig `json:"signature_config,omitempty"`
}

// CreateRoutingRuleRequest represents a request to create a routing rule
type CreateRoutingRuleRequest struct {
	ProviderID   string               `json:"provider_id" binding:"required"`
	Name         string               `json:"name" binding:"required,min=1,max=255"`
	Description  string               `json:"description,omitempty"`
	Priority     int                  `json:"priority"`
	Conditions   []RoutingCondition   `json:"conditions,omitempty"`
	Destinations []RoutingDestination `json:"destinations" binding:"required,min=1"`
	Transform    json.RawMessage      `json:"transform,omitempty"`
}

// UpdateRoutingRuleRequest represents a request to update a routing rule
type UpdateRoutingRuleRequest struct {
	Name         string               `json:"name,omitempty"`
	Description  string               `json:"description,omitempty"`
	Priority     int                  `json:"priority,omitempty"`
	IsActive     bool                 `json:"is_active"`
	Conditions   []RoutingCondition   `json:"conditions,omitempty"`
	Destinations []RoutingDestination `json:"destinations,omitempty"`
	Transform    json.RawMessage      `json:"transform,omitempty"`
}
