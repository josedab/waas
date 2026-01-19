package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Organization represents a top-level account (can have multiple tenants)
type Organization struct {
	ID               uuid.UUID        `json:"id" db:"id"`
	Name             string           `json:"name" db:"name"`
	Slug             string           `json:"slug" db:"slug"`
	BillingEmail     string           `json:"billing_email" db:"billing_email"`
	BillingAddress   *BillingAddress  `json:"billing_address,omitempty" db:"billing_address"`
	StripeCustomerID string           `json:"-" db:"stripe_customer_id"`
	PlanID           *uuid.UUID       `json:"plan_id,omitempty" db:"plan_id"`
	Status           string           `json:"status" db:"status"`
	TrialEndsAt      *time.Time       `json:"trial_ends_at,omitempty" db:"trial_ends_at"`
	Settings         json.RawMessage  `json:"settings,omitempty" db:"settings"`
	Metadata         json.RawMessage  `json:"metadata,omitempty" db:"metadata"`
	CreatedAt        time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at" db:"updated_at"`
}

type BillingAddress struct {
	Line1      string `json:"line1,omitempty"`
	Line2      string `json:"line2,omitempty"`
	City       string `json:"city,omitempty"`
	State      string `json:"state,omitempty"`
	PostalCode string `json:"postal_code,omitempty"`
	Country    string `json:"country,omitempty"`
}

// SubscriptionPlan defines pricing and limits for a plan tier
type SubscriptionPlan struct {
	ID                   uuid.UUID       `json:"id" db:"id"`
	Name                 string          `json:"name" db:"name"`
	Slug                 string          `json:"slug" db:"slug"`
	Description          string          `json:"description,omitempty" db:"description"`
	PriceMonthlyCents    int             `json:"price_monthly_cents" db:"price_monthly_cents"`
	PriceYearlyCents     int             `json:"price_yearly_cents" db:"price_yearly_cents"`
	StripePriceIDMonthly string          `json:"-" db:"stripe_price_id_monthly"`
	StripePriceIDYearly  string          `json:"-" db:"stripe_price_id_yearly"`
	Limits               *PlanLimits     `json:"limits" db:"limits"`
	IsPublic             bool            `json:"is_public" db:"is_public"`
	SortOrder            int             `json:"sort_order" db:"sort_order"`
	CreatedAt            time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at" db:"updated_at"`
}

type PlanLimits struct {
	MaxEndpoints          int      `json:"max_endpoints"`
	MaxDeliveriesPerMonth int      `json:"max_deliveries_per_month"`
	MaxPayloadSizeKB      int      `json:"max_payload_size_kb"`
	MaxRetentionDays      int      `json:"max_retention_days"`
	MaxTeamMembers        int      `json:"max_team_members"`
	Features              []string `json:"features"`
}

// UsageMeter tracks aggregate usage for billing periods
type UsageMeter struct {
	ID             uuid.UUID       `json:"id" db:"id"`
	OrganizationID uuid.UUID       `json:"organization_id" db:"organization_id"`
	TenantID       *uuid.UUID      `json:"tenant_id,omitempty" db:"tenant_id"`
	MeterType      string          `json:"meter_type" db:"meter_type"`
	PeriodStart    time.Time       `json:"period_start" db:"period_start"`
	PeriodEnd      time.Time       `json:"period_end" db:"period_end"`
	UsageCount     int64           `json:"usage_count" db:"usage_count"`
	UsageLimit     *int64          `json:"usage_limit,omitempty" db:"usage_limit"`
	OverageCount   int64           `json:"overage_count" db:"overage_count"`
	Metadata       json.RawMessage `json:"metadata,omitempty" db:"metadata"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at" db:"updated_at"`
}

// UsageEvent represents a single usage event for metering
type UsageEvent struct {
	ID             uuid.UUID       `json:"id" db:"id"`
	OrganizationID uuid.UUID       `json:"organization_id" db:"organization_id"`
	TenantID       *uuid.UUID      `json:"tenant_id,omitempty" db:"tenant_id"`
	EventType      string          `json:"event_type" db:"event_type"`
	Quantity       int64           `json:"quantity" db:"quantity"`
	Properties     json.RawMessage `json:"properties,omitempty" db:"properties"`
	IdempotencyKey string          `json:"idempotency_key,omitempty" db:"idempotency_key"`
	RecordedAt     time.Time       `json:"recorded_at" db:"recorded_at"`
}

// OnboardingSession tracks self-service signup progress
type OnboardingSession struct {
	ID                    uuid.UUID       `json:"id" db:"id"`
	OrganizationID        *uuid.UUID      `json:"organization_id,omitempty" db:"organization_id"`
	Email                 string          `json:"email" db:"email"`
	VerificationToken     string          `json:"-" db:"verification_token"`
	VerificationExpiresAt *time.Time      `json:"-" db:"verification_expires_at"`
	CurrentStep           string          `json:"current_step" db:"current_step"`
	CompletedSteps        []string        `json:"completed_steps" db:"completed_steps"`
	FormData              json.RawMessage `json:"form_data,omitempty" db:"form_data"`
	IPAddress             string          `json:"-" db:"ip_address"`
	UserAgent             string          `json:"-" db:"user_agent"`
	ReferralSource        string          `json:"referral_source,omitempty" db:"referral_source"`
	UTMParams             json.RawMessage `json:"utm_params,omitempty" db:"utm_params"`
	CreatedAt             time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at" db:"updated_at"`
	CompletedAt           *time.Time      `json:"completed_at,omitempty" db:"completed_at"`
}

// TeamMember represents a user within an organization
type TeamMember struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	OrganizationID  uuid.UUID       `json:"organization_id" db:"organization_id"`
	Email           string          `json:"email" db:"email"`
	Name            string          `json:"name,omitempty" db:"name"`
	Role            string          `json:"role" db:"role"`
	PasswordHash    string          `json:"-" db:"password_hash"`
	Status          string          `json:"status" db:"status"`
	InviteToken     string          `json:"-" db:"invite_token"`
	InviteExpiresAt *time.Time      `json:"-" db:"invite_expires_at"`
	LastLoginAt     *time.Time      `json:"last_login_at,omitempty" db:"last_login_at"`
	MFAEnabled      bool            `json:"mfa_enabled" db:"mfa_enabled"`
	MFASecret       string          `json:"-" db:"mfa_secret"`
	Settings        json.RawMessage `json:"settings,omitempty" db:"settings"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at" db:"updated_at"`
}

// OrgAPIToken represents an API token for organization access
type OrgAPIToken struct {
	ID             uuid.UUID       `json:"id" db:"id"`
	OrganizationID uuid.UUID       `json:"organization_id" db:"organization_id"`
	TenantID       *uuid.UUID      `json:"tenant_id,omitempty" db:"tenant_id"`
	Name           string          `json:"name" db:"name"`
	TokenHash      string          `json:"-" db:"token_hash"`
	TokenPrefix    string          `json:"token_prefix" db:"token_prefix"`
	Scopes         []string        `json:"scopes" db:"scopes"`
	LastUsedAt     *time.Time      `json:"last_used_at,omitempty" db:"last_used_at"`
	ExpiresAt      *time.Time      `json:"expires_at,omitempty" db:"expires_at"`
	CreatedBy      *uuid.UUID      `json:"created_by,omitempty" db:"created_by"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	RevokedAt      *time.Time      `json:"revoked_at,omitempty" db:"revoked_at"`
}

// CloudRegion represents a deployment region
type CloudRegion struct {
	ID        uuid.UUID       `json:"id" db:"id"`
	Code      string          `json:"code" db:"code"`
	Name      string          `json:"name" db:"name"`
	Provider  string          `json:"provider" db:"provider"`
	Location  string          `json:"location,omitempty" db:"location"`
	IsActive  bool            `json:"is_active" db:"is_active"`
	IsDefault bool            `json:"is_default" db:"is_default"`
	LatencyMs *int            `json:"latency_ms,omitempty" db:"latency_ms"`
	Features  json.RawMessage `json:"features,omitempty" db:"features"`
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
}

// TenantRegion links tenants to their deployment regions
type TenantRegion struct {
	ID                    uuid.UUID `json:"id" db:"id"`
	TenantID              uuid.UUID `json:"tenant_id" db:"tenant_id"`
	RegionID              uuid.UUID `json:"region_id" db:"region_id"`
	IsPrimary             bool      `json:"is_primary" db:"is_primary"`
	DataResidencyRequired bool      `json:"data_residency_required" db:"data_residency_required"`
	CreatedAt             time.Time `json:"created_at" db:"created_at"`
}

// Meter types
const (
	MeterTypeDeliveries      = "deliveries"
	MeterTypeBandwidthBytes  = "bandwidth_bytes"
	MeterTypeEndpoints       = "endpoints"
	MeterTypeTransformations = "transformations"
	MeterTypeAPICalls        = "api_calls"
	MeterTypeStorageBytes    = "storage_bytes"
	MeterTypeTeamMembers     = "team_members"
)

// Organization statuses
const (
	OrgStatusActive    = "active"
	OrgStatusSuspended = "suspended"
	OrgStatusCancelled = "cancelled"
	OrgStatusTrial     = "trial"
)

// Team member roles
const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
	RoleViewer = "viewer"
)

// Onboarding steps
const (
	OnboardingStepEmailVerification = "email_verification"
	OnboardingStepOrganizationSetup = "organization_setup"
	OnboardingStepPlanSelection     = "plan_selection"
	OnboardingStepPaymentSetup      = "payment_setup"
	OnboardingStepFirstEndpoint     = "first_endpoint"
	OnboardingStepCompleted         = "completed"
)
