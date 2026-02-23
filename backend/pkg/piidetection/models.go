package piidetection

import (
	"encoding/json"
	"time"
)

// PII category constants
const (
	CategoryEmail      = "email"
	CategoryPhone      = "phone"
	CategorySSN        = "ssn"
	CategoryCreditCard = "credit_card"
	CategoryName       = "name"
	CategoryAddress    = "address"
	CategoryDOB        = "date_of_birth"
	CategoryIPAddress  = "ip_address"
	CategoryCustom     = "custom"
)

// Sensitivity level constants
const (
	SensitivityLow      = "low"
	SensitivityMedium   = "medium"
	SensitivityHigh     = "high"
	SensitivityCritical = "critical"
)

// Masking action constants
const (
	ActionMask     = "mask"
	ActionRedact   = "redact"
	ActionHash     = "hash"
	ActionTokenize = "tokenize"
)

// Policy represents a per-tenant PII detection policy.
type Policy struct {
	ID             string          `json:"id" db:"id"`
	TenantID       string          `json:"tenant_id" db:"tenant_id"`
	Name           string          `json:"name" db:"name"`
	Description    string          `json:"description" db:"description"`
	Sensitivity    string          `json:"sensitivity" db:"sensitivity"`
	Categories     []string        `json:"categories"`
	CustomPatterns []CustomPattern `json:"custom_patterns"`
	MaskingAction  string          `json:"masking_action" db:"masking_action"`
	IsEnabled      bool            `json:"is_enabled" db:"is_enabled"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at" db:"updated_at"`
}

// CustomPattern defines a user-supplied regex pattern for PII detection.
type CustomPattern struct {
	Name    string `json:"name"`
	Pattern string `json:"pattern"`
	Label   string `json:"label"`
}

// ScanResult records the outcome of a PII scan on a webhook payload.
type ScanResult struct {
	ID             string      `json:"id" db:"id"`
	TenantID       string      `json:"tenant_id" db:"tenant_id"`
	PolicyID       string      `json:"policy_id" db:"policy_id"`
	WebhookID      string      `json:"webhook_id" db:"webhook_id"`
	EndpointID     string      `json:"endpoint_id" db:"endpoint_id"`
	EventType      string      `json:"event_type" db:"event_type"`
	Detections     []Detection `json:"detections"`
	FieldsScanned  int         `json:"fields_scanned" db:"fields_scanned"`
	FieldsMasked   int         `json:"fields_masked" db:"fields_masked"`
	MaskingAction  string      `json:"masking_action" db:"masking_action"`
	OriginalHash   string      `json:"original_hash" db:"original_hash"`
	MaskedHash     string      `json:"masked_hash" db:"masked_hash"`
	ScanDurationMs int         `json:"scan_duration_ms" db:"scan_duration_ms"`
	CreatedAt      time.Time   `json:"created_at" db:"created_at"`
}

// Detection describes a single PII field found in a payload.
type Detection struct {
	FieldPath string `json:"field_path"`
	Category  string `json:"category"`
	Masked    bool   `json:"masked"`
}

// DashboardStats aggregates PII detection metrics for a tenant.
type DashboardStats struct {
	TotalScans        int64           `json:"total_scans"`
	TotalDetections   int64           `json:"total_detections"`
	TotalFieldsMasked int64           `json:"total_fields_masked"`
	ActivePolicies    int             `json:"active_policies"`
	TopCategories     []CategoryCount `json:"top_categories"`
}

// CategoryCount pairs a PII category with its occurrence count.
type CategoryCount struct {
	Category string `json:"category"`
	Count    int64  `json:"count"`
}

// Request DTOs

// CreatePolicyRequest is the payload for creating a new PII detection policy.
type CreatePolicyRequest struct {
	Name           string          `json:"name" binding:"required"`
	Description    string          `json:"description"`
	Sensitivity    string          `json:"sensitivity" binding:"required"`
	Categories     []string        `json:"categories" binding:"required"`
	CustomPatterns []CustomPattern `json:"custom_patterns"`
	MaskingAction  string          `json:"masking_action" binding:"required"`
}

// UpdatePolicyRequest is the payload for updating an existing policy.
type UpdatePolicyRequest struct {
	Name           *string         `json:"name"`
	Description    *string         `json:"description"`
	Sensitivity    *string         `json:"sensitivity"`
	Categories     []string        `json:"categories"`
	CustomPatterns []CustomPattern `json:"custom_patterns"`
	MaskingAction  *string         `json:"masking_action"`
	IsEnabled      *bool           `json:"is_enabled"`
}

// ScanRequest asks the engine to scan a payload for PII.
type ScanRequest struct {
	WebhookID  string          `json:"webhook_id" binding:"required"`
	EndpointID string          `json:"endpoint_id" binding:"required"`
	EventType  string          `json:"event_type"`
	Payload    json.RawMessage `json:"payload" binding:"required"`
}

// ScanResponse returns the masked payload and scan metadata.
type ScanResponse struct {
	MaskedPayload json.RawMessage `json:"masked_payload"`
	Result        *ScanResult     `json:"result"`
}
