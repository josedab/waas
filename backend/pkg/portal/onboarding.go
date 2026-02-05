package portal

import (
	"encoding/json"
	"time"
)

// OnboardingWizard represents a guided onboarding flow for new tenants
type OnboardingWizard struct {
	ID          string           `json:"id" db:"id"`
	TenantID    string           `json:"tenant_id" db:"tenant_id"`
	CurrentStep string           `json:"current_step" db:"current_step"`
	Steps       []OnboardingStep `json:"steps"`
	StepsJSON   string           `json:"-" db:"steps_json"`
	StartedAt   time.Time        `json:"started_at" db:"started_at"`
	CompletedAt *time.Time       `json:"completed_at,omitempty" db:"completed_at"`
	UpdatedAt   time.Time        `json:"updated_at" db:"updated_at"`
}

// OnboardingStep represents a single step in the onboarding wizard
type OnboardingStep struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Status      string     `json:"status"` // pending, in_progress, completed, skipped
	Order       int        `json:"order"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// Onboarding step IDs
const (
	StepCreateTenant    = "create_tenant"
	StepConfigEndpoint  = "configure_endpoint"
	StepSendTestWebhook = "send_test_webhook"
	StepVerifyDelivery  = "verify_delivery"
	StepViewAnalytics   = "view_analytics"

	StepStatusPending    = "pending"
	StepStatusInProgress = "in_progress"
	StepStatusCompleted  = "completed"
	StepStatusSkipped    = "skipped"
)

// DefaultOnboardingSteps returns the standard onboarding flow
func DefaultOnboardingSteps() []OnboardingStep {
	return []OnboardingStep{
		{ID: StepCreateTenant, Name: "Create Tenant", Description: "Set up your webhook tenant with API credentials", Order: 1, Status: StepStatusPending},
		{ID: StepConfigEndpoint, Name: "Configure Endpoint", Description: "Add your first webhook endpoint URL", Order: 2, Status: StepStatusPending},
		{ID: StepSendTestWebhook, Name: "Send Test Webhook", Description: "Send a test webhook event to your endpoint", Order: 3, Status: StepStatusPending},
		{ID: StepVerifyDelivery, Name: "Verify Delivery", Description: "Confirm your endpoint received the webhook", Order: 4, Status: StepStatusPending},
		{ID: StepViewAnalytics, Name: "View Analytics", Description: "Explore your webhook delivery dashboard", Order: 5, Status: StepStatusPending},
	}
}

// APIExplorerEndpoint represents an API endpoint in the interactive explorer
type APIExplorerEndpoint struct {
	Method      string              `json:"method"`
	Path        string              `json:"path"`
	Summary     string              `json:"summary"`
	Description string              `json:"description"`
	Tags        []string            `json:"tags"`
	Parameters  []ExplorerParameter `json:"parameters,omitempty"`
	RequestBody *ExplorerBody       `json:"request_body,omitempty"`
	Responses   []ExplorerResponse  `json:"responses,omitempty"`
}

// ExplorerParameter describes an API parameter
type ExplorerParameter struct {
	Name        string `json:"name"`
	In          string `json:"in"` // query, path, header
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Type        string `json:"type"`
	Example     string `json:"example,omitempty"`
}

// ExplorerBody describes a request body
type ExplorerBody struct {
	ContentType string          `json:"content_type"`
	Schema      json.RawMessage `json:"schema,omitempty"`
	Example     json.RawMessage `json:"example,omitempty"`
}

// ExplorerResponse describes an API response
type ExplorerResponse struct {
	StatusCode  int             `json:"status_code"`
	Description string          `json:"description"`
	Example     json.RawMessage `json:"example,omitempty"`
}

// APIExplorerConfig holds the interactive API explorer configuration
type APIExplorerConfig struct {
	BaseURL    string                `json:"base_url"`
	Title      string                `json:"title"`
	Version    string                `json:"version"`
	Endpoints  []APIExplorerEndpoint `json:"endpoints"`
	Categories []string              `json:"categories"`
}

// SDKCodeGenRequest is the request for generating SDK code
type SDKCodeGenRequest struct {
	Language  string `json:"language" binding:"required"`
	EventType string `json:"event_type,omitempty"`
	Framework string `json:"framework,omitempty"`
}

// SDKCodeGenResponse contains generated code for a specific SDK/language
type SDKCodeGenResponse struct {
	Language    string `json:"language"`
	Framework   string `json:"framework"`
	Code        string `json:"code"`
	InstallCmd  string `json:"install_cmd,omitempty"`
	Description string `json:"description"`
}

// UnifiedPortalView combines all portal sub-features into a single view
type UnifiedPortalView struct {
	Portal      *PortalConfig     `json:"portal"`
	Onboarding  *OnboardingWizard `json:"onboarding,omitempty"`
	Stats       *PortalStats      `json:"stats,omitempty"`
	QuickLinks  []QuickLink       `json:"quick_links"`
	RecentItems []RecentActivity  `json:"recent_activity,omitempty"`
}

// QuickLink provides navigation shortcuts
type QuickLink struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Icon        string `json:"icon"`
}

// RecentActivity tracks recent actions in the portal
type RecentActivity struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // delivery, endpoint_created, test_sent, etc.
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
}

// StartOnboardingRequest initiates the onboarding wizard
type StartOnboardingRequest struct {
	TenantName  string `json:"tenant_name" binding:"required,min=1,max=255"`
	CompanyName string `json:"company_name,omitempty"`
}

// CompleteStepRequest marks an onboarding step as completed
type CompleteStepRequest struct {
	StepID string                 `json:"step_id" binding:"required"`
	Data   map[string]interface{} `json:"data,omitempty"`
}

// TryEndpointRequest is used for the live API explorer's "Try It" feature
type TryEndpointRequest struct {
	Method  string            `json:"method" binding:"required"`
	Path    string            `json:"path" binding:"required"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    json.RawMessage   `json:"body,omitempty"`
}

// TryEndpointResponse contains the result of a "Try It" API call
type TryEndpointResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       json.RawMessage   `json:"body,omitempty"`
	LatencyMs  int64             `json:"latency_ms"`
}
