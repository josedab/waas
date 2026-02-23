package policyengine

import (
	"encoding/json"
	"time"
)

// Policy type constants
const (
	PolicyTypeRouting       = "routing"
	PolicyTypeFiltering     = "filtering"
	PolicyTypeAuthorization = "authorization"
	PolicyTypeDelivery      = "delivery"
)

// Policy represents a stored Rego policy.
type Policy struct {
	ID            string     `json:"id" db:"id"`
	TenantID      string     `json:"tenant_id" db:"tenant_id"`
	Name          string     `json:"name" db:"name"`
	Description   string     `json:"description" db:"description"`
	RegoSource    string     `json:"rego_source" db:"rego_source"`
	Version       int        `json:"version" db:"version"`
	IsActive      bool       `json:"is_active" db:"is_active"`
	PolicyType    string     `json:"policy_type" db:"policy_type"`
	LastEvaluated *time.Time `json:"last_evaluated,omitempty" db:"last_evaluated"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

// PolicyVersion records a historical version of a Rego policy.
type PolicyVersion struct {
	ID         string    `json:"id" db:"id"`
	PolicyID   string    `json:"policy_id" db:"policy_id"`
	Version    int       `json:"version" db:"version"`
	RegoSource string    `json:"rego_source" db:"rego_source"`
	ChangeNote string    `json:"change_note" db:"change_note"`
	CreatedBy  string    `json:"created_by" db:"created_by"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// EvaluationLog records a policy evaluation.
type EvaluationLog struct {
	ID         string          `json:"id" db:"id"`
	TenantID   string          `json:"tenant_id" db:"tenant_id"`
	PolicyID   string          `json:"policy_id" db:"policy_id"`
	Decision   bool            `json:"decision" db:"decision"`
	InputHash  string          `json:"input_hash" db:"input_hash"`
	DurationMs int             `json:"duration_ms" db:"duration_ms"`
	IsDryRun   bool            `json:"is_dry_run" db:"is_dry_run"`
	Result     json.RawMessage `json:"result" db:"result"`
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`
}

// EvaluationInput is the standardized input document for policy evaluation.
type EvaluationInput struct {
	TenantID    string                 `json:"tenant_id"`
	EventType   string                 `json:"event_type"`
	EndpointID  string                 `json:"endpoint_id,omitempty"`
	WebhookID   string                 `json:"webhook_id,omitempty"`
	PayloadSize int                    `json:"payload_size"`
	Headers     map[string]string      `json:"headers,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Payload     json.RawMessage        `json:"payload,omitempty"`
}

// EvaluationResult holds the outcome of a policy evaluation.
type EvaluationResult struct {
	Allowed    bool                   `json:"allowed"`
	Decision   map[string]interface{} `json:"decision"`
	PolicyID   string                 `json:"policy_id"`
	PolicyName string                 `json:"policy_name"`
	DurationMs int                    `json:"duration_ms"`
	IsDryRun   bool                   `json:"is_dry_run"`
}

// ValidationResult holds Rego syntax validation results.
type ValidationResult struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

// Request DTOs

type CreatePolicyRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	RegoSource  string `json:"rego_source" binding:"required"`
	PolicyType  string `json:"policy_type" binding:"required"`
}

type UpdatePolicyRequest struct {
	RegoSource  *string `json:"rego_source"`
	Description *string `json:"description"`
	IsActive    *bool   `json:"is_active"`
	ChangeNote  string  `json:"change_note"`
}

type EvaluateRequest struct {
	PolicyID string          `json:"policy_id" binding:"required"`
	Input    EvaluationInput `json:"input" binding:"required"`
	DryRun   bool            `json:"dry_run"`
}

type ValidateRegoRequest struct {
	RegoSource string `json:"rego_source" binding:"required"`
}
