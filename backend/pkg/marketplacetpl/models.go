package marketplacetpl

import "time"

// Template represents a pre-built webhook integration template
type Template struct {
	ID          string         `json:"id" db:"id"`
	Name        string         `json:"name" db:"name"`
	Description string         `json:"description" db:"description"`
	Category    string         `json:"category" db:"category"`
	Source      string         `json:"source" db:"source"`
	Destination string         `json:"destination" db:"destination"`
	Transform   string         `json:"transform,omitempty" db:"transform"`
	RetryPolicy *RetryPolicy   `json:"retry_policy,omitempty"`
	RetryJSON   string         `json:"-" db:"retry_policy"`
	SamplePayload string       `json:"sample_payload,omitempty" db:"sample_payload"`
	Version     string         `json:"version" db:"version"`
	Author      string         `json:"author" db:"author"`
	InstallCount int           `json:"install_count" db:"install_count"`
	Rating      float64        `json:"rating" db:"rating"`
	IsVerified  bool           `json:"is_verified" db:"is_verified"`
	Tags        []string       `json:"tags"`
	TagsJSON    string         `json:"-" db:"tags"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" db:"updated_at"`
}

// RetryPolicy defines retry configuration for a template
type RetryPolicy struct {
	MaxAttempts  int `json:"max_attempts"`
	InitialDelay int `json:"initial_delay_ms"`
	MaxDelay     int `json:"max_delay_ms"`
}

// Installation represents a template installed by a tenant
type Installation struct {
	ID           string    `json:"id" db:"id"`
	TenantID     string    `json:"tenant_id" db:"tenant_id"`
	TemplateID   string    `json:"template_id" db:"template_id"`
	EndpointID   string    `json:"endpoint_id,omitempty" db:"endpoint_id"`
	Config       string    `json:"config,omitempty" db:"config"`
	Status       string    `json:"status" db:"status"`
	InstalledAt  time.Time `json:"installed_at" db:"installed_at"`
}

// InstallationStatus constants
const (
	InstallStatusActive   = "active"
	InstallStatusDisabled = "disabled"
	InstallStatusRemoved  = "removed"
)

// TemplateCategory constants
const (
	CategoryPayments    = "payments"
	CategoryMessaging   = "messaging"
	CategoryDevOps      = "devops"
	CategoryMonitoring  = "monitoring"
	CategoryEcommerce   = "ecommerce"
	CategoryCRM         = "crm"
	CategoryCustom      = "custom"
)

// Review represents a user review of a template
type Review struct {
	ID         string    `json:"id" db:"id"`
	TenantID   string    `json:"tenant_id" db:"tenant_id"`
	TemplateID string    `json:"template_id" db:"template_id"`
	Rating     int       `json:"rating" db:"rating"`
	Comment    string    `json:"comment,omitempty" db:"comment"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// CreateTemplateRequest is the request DTO for submitting a template
type CreateTemplateRequest struct {
	Name          string       `json:"name" binding:"required,min=1,max=255"`
	Description   string       `json:"description" binding:"required"`
	Category      string       `json:"category" binding:"required"`
	Source        string       `json:"source" binding:"required"`
	Destination   string       `json:"destination" binding:"required"`
	Transform     string       `json:"transform,omitempty"`
	RetryPolicy   *RetryPolicy `json:"retry_policy,omitempty"`
	SamplePayload string       `json:"sample_payload,omitempty"`
	Tags          []string     `json:"tags,omitempty"`
}

// InstallTemplateRequest is the request DTO for installing a template
type InstallTemplateRequest struct {
	EndpointID string `json:"endpoint_id,omitempty"`
	Config     string `json:"config,omitempty"`
}

// SubmitReviewRequest is the request DTO for submitting a review
type SubmitReviewRequest struct {
	Rating  int    `json:"rating" binding:"required,min=1,max=5"`
	Comment string `json:"comment,omitempty"`
}

// MarketplaceStats provides marketplace overview statistics
type MarketplaceStats struct {
	TotalTemplates  int `json:"total_templates"`
	VerifiedCount   int `json:"verified_templates"`
	TotalInstalls   int `json:"total_installations"`
	CategoryCounts  map[string]int `json:"categories"`
}
