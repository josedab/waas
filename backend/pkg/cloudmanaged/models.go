package cloudmanaged

import (
	"time"

	"github.com/google/uuid"
)

// PlanTier defines the subscription tier
type PlanTier string

const (
	PlanTierFree       PlanTier = "free"
	PlanTierStarter    PlanTier = "starter"
	PlanTierPro        PlanTier = "pro"
	PlanTierEnterprise PlanTier = "enterprise"
)

// CloudTenantStatus defines the tenant status
type CloudTenantStatus string

const (
	CloudTenantStatusActive    CloudTenantStatus = "active"
	CloudTenantStatusTrial     CloudTenantStatus = "trial"
	CloudTenantStatusSuspended CloudTenantStatus = "suspended"
	CloudTenantStatusCancelled CloudTenantStatus = "cancelled"
)

// CloudTenant represents a managed cloud tenant
type CloudTenant struct {
	ID            uuid.UUID         `json:"id" db:"id"`
	TenantID      string            `json:"tenant_id" db:"tenant_id"`
	Email         string            `json:"email" db:"email"`
	Org           string            `json:"org" db:"org"`
	Plan          PlanTier          `json:"plan" db:"plan"`
	Status        CloudTenantStatus `json:"status" db:"status"`
	Region        string            `json:"region" db:"region"`
	WebhooksUsed  int64             `json:"webhooks_used" db:"webhooks_used"`
	WebhooksLimit int64             `json:"webhooks_limit" db:"webhooks_limit"`
	StorageUsed   int64             `json:"storage_used" db:"storage_used"`
	StorageLimit  int64             `json:"storage_limit" db:"storage_limit"`
	CreatedAt     time.Time         `json:"created_at" db:"created_at"`
	TrialEndsAt   *time.Time        `json:"trial_ends_at,omitempty" db:"trial_ends_at"`
}

// PlanDefinition describes a subscription plan and its limits
type PlanDefinition struct {
	Tier          PlanTier `json:"tier"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	PriceMonthly  int64    `json:"price_monthly"`
	WebhooksLimit int64    `json:"webhooks_limit"`
	StorageLimit  int64    `json:"storage_limit"`
	RetentionDays int      `json:"retention_days"`
	SupportLevel  string   `json:"support_level"`
	Features      []string `json:"features"`
}

// UsageMeter represents a single usage metric record
type UsageMeter struct {
	ID         uuid.UUID `json:"id" db:"id"`
	TenantID   string    `json:"tenant_id" db:"tenant_id"`
	MetricType string    `json:"metric_type" db:"metric_type"`
	Value      int64     `json:"value" db:"value"`
	Period     string    `json:"period" db:"period"`
	RecordedAt time.Time `json:"recorded_at" db:"recorded_at"`
}

// UsageSummary aggregates usage metrics for a period
type UsageSummary struct {
	TenantID        string `json:"tenant_id"`
	Period          string `json:"period"`
	WebhooksSent    int64  `json:"webhooks_sent"`
	WebhooksRecvd   int64  `json:"webhooks_received"`
	Bandwidth       int64  `json:"bandwidth"`
	ActiveEndpoints int64  `json:"active_endpoints"`
	APIRequests     int64  `json:"api_requests"`
	StorageUsed     int64  `json:"storage_used"`
}

// BillingInfo holds billing and payment details for a tenant
type BillingInfo struct {
	TenantID        string     `json:"tenant_id" db:"tenant_id"`
	StripeCustomerID string    `json:"stripe_customer_id" db:"stripe_customer_id"`
	StripePlanID    string     `json:"stripe_plan_id" db:"stripe_plan_id"`
	PaymentMethod   string     `json:"payment_method" db:"payment_method"`
	BillingEmail    string     `json:"billing_email" db:"billing_email"`
	NextBillingDate *time.Time `json:"next_billing_date,omitempty" db:"next_billing_date"`
	AmountDue       int64      `json:"amount_due" db:"amount_due"`
}

// OnboardingStep represents a single onboarding step
type OnboardingStep struct {
	StepID      string     `json:"step_id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Required    bool       `json:"required"`
	Completed   bool       `json:"completed"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// OnboardingProgress tracks the onboarding state for a tenant
type OnboardingProgress struct {
	TenantID      string            `json:"tenant_id"`
	Steps         []OnboardingStep  `json:"steps"`
	CompletionPct float64           `json:"completion_pct"`
	AllCompleted  bool              `json:"all_completed"`
}

// SignupRequest represents a self-service signup request
type SignupRequest struct {
	Email  string `json:"email" binding:"required"`
	Org    string `json:"org" binding:"required"`
	Plan   string `json:"plan,omitempty"`
	Region string `json:"region,omitempty"`
}

// UpgradePlanRequest represents a plan change request
type UpgradePlanRequest struct {
	Plan string `json:"plan" binding:"required"`
}

// UpdateBillingRequest represents a billing info update request
type UpdateBillingRequest struct {
	PaymentMethod string `json:"payment_method,omitempty"`
	BillingEmail  string `json:"billing_email,omitempty"`
}

// Ensure uuid is used
var _ = uuid.New
