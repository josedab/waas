package cloudmanaged

import (
	"encoding/json"
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
	TenantID         string     `json:"tenant_id" db:"tenant_id"`
	StripeCustomerID string     `json:"stripe_customer_id" db:"stripe_customer_id"`
	StripePlanID     string     `json:"stripe_plan_id" db:"stripe_plan_id"`
	PaymentMethod    string     `json:"payment_method" db:"payment_method"`
	BillingEmail     string     `json:"billing_email" db:"billing_email"`
	NextBillingDate  *time.Time `json:"next_billing_date,omitempty" db:"next_billing_date"`
	AmountDue        int64      `json:"amount_due" db:"amount_due"`
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
	TenantID      string           `json:"tenant_id"`
	Steps         []OnboardingStep `json:"steps"`
	CompletionPct float64          `json:"completion_pct"`
	AllCompleted  bool             `json:"all_completed"`
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

// TenantIsolation represents tenant isolation configuration
type TenantIsolation struct {
	TenantID           string  `json:"tenant_id" db:"tenant_id"`
	IsolationLevel     string  `json:"isolation_level" db:"isolation_level"` // shared, dedicated, isolated
	ResourcePool       string  `json:"resource_pool" db:"resource_pool"`
	MaxConcurrentReqs  int     `json:"max_concurrent_requests" db:"max_concurrent_requests"`
	MaxPayloadSizeKB   int     `json:"max_payload_size_kb" db:"max_payload_size_kb"`
	RateLimitPerMinute int     `json:"rate_limit_per_minute" db:"rate_limit_per_minute"`
	NoisyNeighborScore float64 `json:"noisy_neighbor_score" db:"noisy_neighbor_score"`
}

// AutoScaleConfig represents auto-scaling configuration
type AutoScaleConfig struct {
	TenantID         string `json:"tenant_id" db:"tenant_id"`
	Enabled          bool   `json:"enabled" db:"enabled"`
	MinInstances     int    `json:"min_instances" db:"min_instances"`
	MaxInstances     int    `json:"max_instances" db:"max_instances"`
	ScaleUpAt        int    `json:"scale_up_at_pct" db:"scale_up_at_pct"`
	ScaleDownAt      int    `json:"scale_down_at_pct" db:"scale_down_at_pct"`
	CooldownSecs     int    `json:"cooldown_secs" db:"cooldown_secs"`
	CurrentInstances int    `json:"current_instances" db:"current_instances"`
}

// SLAConfig represents SLA monitoring configuration
type SLAConfig struct {
	TenantID           string  `json:"tenant_id" db:"tenant_id"`
	UptimeTargetPct    float64 `json:"uptime_target_pct" db:"uptime_target_pct"`
	LatencyTargetMs    int     `json:"latency_target_ms" db:"latency_target_ms"`
	DeliveryTargetPct  float64 `json:"delivery_target_pct" db:"delivery_target_pct"`
	CurrentUptimePct   float64 `json:"current_uptime_pct" db:"current_uptime_pct"`
	CurrentLatencyMs   int     `json:"current_latency_ms" db:"current_latency_ms"`
	CurrentDeliveryPct float64 `json:"current_delivery_pct" db:"current_delivery_pct"`
	InViolation        bool    `json:"in_violation" db:"in_violation"`
}

// StatusPageEntry represents a component status for the status page
type StatusPageEntry struct {
	Component   string    `json:"component"`
	Status      string    `json:"status"` // operational, degraded, partial_outage, major_outage
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// StatusPage represents the overall system status
type StatusPage struct {
	OverallStatus   string            `json:"overall_status"`
	Components      []StatusPageEntry `json:"components"`
	ActiveIncidents []StatusIncident  `json:"active_incidents,omitempty"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// StatusIncident represents a status page incident
type StatusIncident struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Status     string    `json:"status"`   // investigating, identified, monitoring, resolved
	Severity   string    `json:"severity"` // minor, major, critical
	Message    string    `json:"message"`
	Components []string  `json:"components"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TenantQuotaStatus represents the current quota state
type TenantQuotaStatus struct {
	TenantID         string   `json:"tenant_id"`
	Plan             PlanTier `json:"plan"`
	WebhooksUsed     int64    `json:"webhooks_used"`
	WebhooksLimit    int64    `json:"webhooks_limit"`
	WebhooksUsagePct float64  `json:"webhooks_usage_pct"`
	StorageUsed      int64    `json:"storage_used"`
	StorageLimit     int64    `json:"storage_limit"`
	StorageUsagePct  float64  `json:"storage_usage_pct"`
	ThrottleActive   bool     `json:"throttle_active"`
}

// RegionalDeployment represents a regional deployment
type RegionalDeployment struct {
	Region        string    `json:"region"`
	Status        string    `json:"status"` // active, provisioning, draining
	TenantCount   int       `json:"tenant_count"`
	InstanceCount int       `json:"instance_count"`
	HealthScore   float64   `json:"health_score"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// StripeWebhookEvent represents an incoming Stripe webhook event
type StripeWebhookEvent struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Created int64           `json:"created"`
	Data    json.RawMessage `json:"data"`
}

// StripeSubscriptionData represents Stripe subscription data in webhooks
type StripeSubscriptionData struct {
	Object struct {
		ID                string `json:"id"`
		CustomerID        string `json:"customer"`
		Status            string `json:"status"`
		CurrentPeriodEnd  int64  `json:"current_period_end"`
		CancelAtPeriodEnd bool   `json:"cancel_at_period_end"`
		Plan              struct {
			ID       string `json:"id"`
			Nickname string `json:"nickname"`
		} `json:"plan"`
	} `json:"object"`
}

// StripeInvoiceData represents Stripe invoice data in webhooks
type StripeInvoiceData struct {
	Object struct {
		ID               string `json:"id"`
		CustomerID       string `json:"customer"`
		AmountDue        int64  `json:"amount_due"`
		AmountPaid       int64  `json:"amount_paid"`
		Status           string `json:"status"`
		HostedInvoiceURL string `json:"hosted_invoice_url"`
	} `json:"object"`
}

// TrialStatus represents the trial state for a tenant
type TrialStatus struct {
	TenantID      string     `json:"tenant_id"`
	IsOnTrial     bool       `json:"is_on_trial"`
	IsExpired     bool       `json:"is_expired"`
	Plan          PlanTier   `json:"plan"`
	TrialEndsAt   *time.Time `json:"trial_ends_at,omitempty"`
	DaysRemaining int        `json:"days_remaining"`
}

// Invoice represents a billing invoice for a period
type Invoice struct {
	TenantID       string    `json:"tenant_id"`
	Period         string    `json:"period"`
	Plan           PlanTier  `json:"plan"`
	BaseAmount     int64     `json:"base_amount"`
	OverageAmount  int64     `json:"overage_amount"`
	OverageDetails string    `json:"overage_details,omitempty"`
	TotalAmount    int64     `json:"total_amount"`
	Currency       string    `json:"currency"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

// Ensure uuid is used
var _ = uuid.New
