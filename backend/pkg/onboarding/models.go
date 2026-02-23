package onboarding

import (
	"encoding/json"
	"time"
)

// Onboarding step constants
const (
	StepCreateTenant   = "create_tenant"
	StepConfigEndpoint = "configure_endpoint"
	StepSendTest       = "send_test_webhook"
	StepVerifyDelivery = "verify_delivery"
	StepInstallSDK     = "install_sdk"
)

// AllSteps defines the ordered onboarding flow.
var AllSteps = []string{
	StepCreateTenant,
	StepConfigEndpoint,
	StepSendTest,
	StepVerifyDelivery,
	StepInstallSDK,
}

// OnboardingSession tracks a tenant's onboarding progress.
type OnboardingSession struct {
	ID             string          `json:"id" db:"id"`
	TenantID       string          `json:"tenant_id" db:"tenant_id"`
	CurrentStep    string          `json:"current_step" db:"current_step"`
	CompletedSteps []string        `json:"completed_steps"`
	Metadata       json.RawMessage `json:"metadata,omitempty" db:"metadata"`
	IsComplete     bool            `json:"is_complete" db:"is_complete"`
	StartedAt      time.Time       `json:"started_at" db:"started_at"`
	CompletedAt    *time.Time      `json:"completed_at,omitempty" db:"completed_at"`
	UpdatedAt      time.Time       `json:"updated_at" db:"updated_at"`
}

// StepDetail provides information about a single onboarding step.
type StepDetail struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Order       int    `json:"order"`
	IsCompleted bool   `json:"is_completed"`
	IsCurrent   bool   `json:"is_current"`
}

// CodeSnippet represents a language-specific code example.
type CodeSnippet struct {
	Language string `json:"language"`
	Code     string `json:"code"`
	Title    string `json:"title"`
}

// OnboardingProgress represents the overall onboarding status.
type OnboardingProgress struct {
	Session         *OnboardingSession `json:"session"`
	Steps           []StepDetail       `json:"steps"`
	ProgressPercent int                `json:"progress_percent"`
	TimeElapsed     string             `json:"time_elapsed"`
}

// OnboardingAnalytics holds funnel metrics.
type OnboardingAnalytics struct {
	TotalStarted          int64            `json:"total_started"`
	TotalCompleted        int64            `json:"total_completed"`
	CompletionRate        float64          `json:"completion_rate"`
	AvgTimeToFirstWebhook string           `json:"avg_time_to_first_webhook"`
	StepDropoff           map[string]int64 `json:"step_dropoff"`
}

// Request DTOs

type StartOnboardingRequest struct {
	TenantID string          `json:"tenant_id" binding:"required"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

type CompleteStepRequest struct {
	StepID   string          `json:"step_id" binding:"required"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

type GetSnippetsRequest struct {
	Language    string `json:"language" binding:"required"`
	TenantID    string `json:"tenant_id"`
	APIKey      string `json:"api_key"`
	EndpointURL string `json:"endpoint_url"`
}
