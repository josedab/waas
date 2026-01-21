package cloud

import (
	"time"
)

// AvailablePlans contains the default available subscription plans
var AvailablePlans = []*Plan{
	{
		ID:           "free",
		Name:         "Free",
		Slug:         "free",
		Description:  "Perfect for getting started",
		PriceMonthly: 0,
		PriceYearly:  0,
		IsFree:       true,
		IsActive:     true,
		Features: PlanFeatures{
			APIAccess:       true,
			WebhooksGateway: true,
		},
		Limits: PlanLimits{
			MonthlyWebhooks:    1000,
			EndpointsPerTenant: 5,
			RequestsPerMinute:  10,
			PayloadSizeBytes:   65536,
			RetentionDays:      7,
			TeamMemberCount:    1,
			MaxRetries:         3,
		},
	},
	{
		ID:           "starter",
		Name:         "Starter",
		Slug:         "starter",
		Description:  "For small teams and projects",
		PriceMonthly: 2900,
		PriceYearly:  29000,
		IsActive:     true,
		TrialDays:    14,
		Features: PlanFeatures{
			APIAccess:         true,
			WebhooksGateway:   true,
			Transformations:   true,
			TeamMembers:       true,
			AdvancedAnalytics: true,
		},
		Limits: PlanLimits{
			MonthlyWebhooks:    50000,
			EndpointsPerTenant: 25,
			RequestsPerMinute:  100,
			PayloadSizeBytes:   1048576,
			RetentionDays:      30,
			TeamMemberCount:    5,
			TransformTimeoutMs: 5000,
			MaxRetries:         5,
		},
	},
	{
		ID:           "pro",
		Name:         "Professional",
		Slug:         "pro",
		Description:  "For growing businesses",
		PriceMonthly: 9900,
		PriceYearly:  99000,
		IsActive:     true,
		TrialDays:    14,
		Features: PlanFeatures{
			APIAccess:         true,
			WebhooksGateway:   true,
			Transformations:   true,
			TeamMembers:       true,
			AdvancedAnalytics: true,
			SchemaRegistry:    true,
			AuditLogs:         true,
			PrioritySupport:   true,
		},
		Limits: PlanLimits{
			MonthlyWebhooks:    500000,
			EndpointsPerTenant: 100,
			RequestsPerMinute:  500,
			PayloadSizeBytes:   5242880,
			RetentionDays:      90,
			TeamMemberCount:    20,
			TransformTimeoutMs: 10000,
			MaxRetries:         10,
		},
	},
	{
		ID:           "enterprise",
		Name:         "Enterprise",
		Slug:         "enterprise",
		Description:  "For large organizations with advanced needs",
		PriceMonthly: 0, // Custom pricing
		PriceYearly:  0,
		IsActive:     true,
		Features: PlanFeatures{
			APIAccess:         true,
			WebhooksGateway:   true,
			Transformations:   true,
			TeamMembers:       true,
			AdvancedAnalytics: true,
			SchemaRegistry:    true,
			AuditLogs:         true,
			PrioritySupport:   true,
			CustomDomains:     true,
			SSO:               true,
			DedicatedInfra:    true,
			MultiRegion:       true,
			SLAGuarantee:      true,
		},
		Limits: PlanLimits{
			MonthlyWebhooks:    0, // Unlimited
			EndpointsPerTenant: 0, // Unlimited
			RequestsPerMinute:  0, // Unlimited
			PayloadSizeBytes:   10485760,
			RetentionDays:      365,
			TeamMemberCount:    0, // Unlimited
			TransformTimeoutMs: 30000,
			MaxRetries:         15,
		},
	},
}

// Plan represents a billing plan
type Plan struct {
	ID              string            `json:"id" db:"id"`
	Name            string            `json:"name" db:"name"`
	Slug            string            `json:"slug" db:"slug"`
	Description     string            `json:"description" db:"description"`
	PriceMonthly    int64             `json:"price_monthly" db:"price_monthly"` // In cents
	PriceYearly     int64             `json:"price_yearly" db:"price_yearly"`   // In cents
	Features        PlanFeatures      `json:"features" db:"features"`
	Limits          PlanLimits        `json:"limits" db:"limits"`
	IsActive        bool              `json:"is_active" db:"is_active"`
	IsFree          bool              `json:"is_free" db:"is_free"`
	TrialDays       int               `json:"trial_days" db:"trial_days"`
	StripePriceID   string            `json:"stripe_price_id,omitempty" db:"stripe_price_id"`
	Metadata        map[string]string `json:"metadata,omitempty" db:"metadata"`
	CreatedAt       time.Time         `json:"created_at" db:"created_at"`
}

// PlanFeatures defines plan features
type PlanFeatures struct {
	CustomDomains       bool `json:"custom_domains"`
	SSO                 bool `json:"sso"`
	AuditLogs           bool `json:"audit_logs"`
	PrioritySupport     bool `json:"priority_support"`
	DedicatedInfra      bool `json:"dedicated_infrastructure"`
	AdvancedAnalytics   bool `json:"advanced_analytics"`
	SchemaRegistry      bool `json:"schema_registry"`
	Transformations     bool `json:"transformations"`
	MultiRegion         bool `json:"multi_region"`
	SLAGuarantee        bool `json:"sla_guarantee"`
	APIAccess           bool `json:"api_access"`
	WebhooksGateway     bool `json:"webhooks_gateway"`
	TeamMembers         bool `json:"team_members"`
}

// PlanLimits defines plan limits
type PlanLimits struct {
	MonthlyWebhooks     int64         `json:"monthly_webhooks"`       // 0 = unlimited
	EndpointsPerTenant  int           `json:"endpoints_per_tenant"`
	RequestsPerMinute   int           `json:"requests_per_minute"`
	PayloadSizeBytes    int64         `json:"payload_size_bytes"`
	RetentionDays       int           `json:"retention_days"`
	TeamMemberCount     int           `json:"team_member_count"`
	TransformTimeoutMs  int           `json:"transform_timeout_ms"`
	MaxRetries          int           `json:"max_retries"`
}

// Subscription represents a tenant subscription
type Subscription struct {
	ID                   string             `json:"id" db:"id"`
	TenantID             string             `json:"tenant_id" db:"tenant_id"`
	PlanID               string             `json:"plan_id" db:"plan_id"`
	Status               SubscriptionStatus `json:"status" db:"status"`
	BillingCycle         BillingCycle       `json:"billing_cycle" db:"billing_cycle"`
	CurrentPeriodStart   time.Time          `json:"current_period_start" db:"current_period_start"`
	CurrentPeriodEnd     time.Time          `json:"current_period_end" db:"current_period_end"`
	TrialEnd             *time.Time         `json:"trial_end,omitempty" db:"trial_end"`
	CancelAtPeriodEnd    bool               `json:"cancel_at_period_end" db:"cancel_at_period_end"`
	CanceledAt           *time.Time         `json:"canceled_at,omitempty" db:"canceled_at"`
	StripeSubscriptionID string             `json:"stripe_subscription_id,omitempty" db:"stripe_subscription_id"`
	StripeCustomerID     string             `json:"stripe_customer_id,omitempty" db:"stripe_customer_id"`
	CreatedAt            time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time          `json:"updated_at" db:"updated_at"`
}

// SubscriptionStatus represents subscription status
type SubscriptionStatus string

const (
	SubscriptionStatusTrialing       SubscriptionStatus = "trialing"
	SubscriptionStatusActive         SubscriptionStatus = "active"
	SubscriptionStatusPastDue        SubscriptionStatus = "past_due"
	SubscriptionStatusCanceled       SubscriptionStatus = "canceled"
	SubscriptionStatusUnpaid         SubscriptionStatus = "unpaid"
	SubscriptionStatusIncomplete     SubscriptionStatus = "incomplete"
)

// BillingCycle represents billing frequency
type BillingCycle string

const (
	BillingCycleMonthly BillingCycle = "monthly"
	BillingCycleYearly  BillingCycle = "yearly"
)

// UsageRecord represents usage tracking
type UsageRecord struct {
	ID                  string    `json:"id" db:"id"`
	TenantID            string    `json:"tenant_id" db:"tenant_id"`
	Period              string    `json:"period" db:"period"` // YYYY-MM
	WebhooksSent        int64     `json:"webhooks_sent" db:"webhooks_sent"`
	WebhooksReceived    int64     `json:"webhooks_received" db:"webhooks_received"`
	SuccessfulDeliveries int64    `json:"successful_deliveries" db:"successful_deliveries"`
	FailedDeliveries    int64     `json:"failed_deliveries" db:"failed_deliveries"`
	TotalBytes          int64     `json:"total_bytes" db:"total_bytes"`
	APIRequests         int64     `json:"api_requests" db:"api_requests"`
	TransformExecutions int64     `json:"transform_executions" db:"transform_executions"`
	StorageBytes        int64     `json:"storage_bytes" db:"storage_bytes"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}

// Invoice represents a billing invoice
type Invoice struct {
	ID               string        `json:"id" db:"id"`
	TenantID         string        `json:"tenant_id" db:"tenant_id"`
	SubscriptionID   string        `json:"subscription_id" db:"subscription_id"`
	Number           string        `json:"number" db:"number"`
	Status           InvoiceStatus `json:"status" db:"status"`
	Currency         string        `json:"currency" db:"currency"`
	Subtotal         int64         `json:"subtotal" db:"subtotal"` // In cents
	Tax              int64         `json:"tax" db:"tax"`
	Total            int64         `json:"total" db:"total"`
	AmountPaid       int64         `json:"amount_paid" db:"amount_paid"`
	AmountDue        int64         `json:"amount_due" db:"amount_due"`
	LineItems        []LineItem    `json:"line_items" db:"line_items"`
	PeriodStart      time.Time     `json:"period_start" db:"period_start"`
	PeriodEnd        time.Time     `json:"period_end" db:"period_end"`
	DueDate          time.Time     `json:"due_date" db:"due_date"`
	PaidAt           *time.Time    `json:"paid_at,omitempty" db:"paid_at"`
	StripeInvoiceID  string        `json:"stripe_invoice_id,omitempty" db:"stripe_invoice_id"`
	PDFUrl           string        `json:"pdf_url,omitempty" db:"pdf_url"`
	CreatedAt        time.Time     `json:"created_at" db:"created_at"`
}

// InvoiceStatus represents invoice status
type InvoiceStatus string

const (
	InvoiceStatusDraft     InvoiceStatus = "draft"
	InvoiceStatusOpen      InvoiceStatus = "open"
	InvoiceStatusPaid      InvoiceStatus = "paid"
	InvoiceStatusUncollectible InvoiceStatus = "uncollectible"
	InvoiceStatusVoid      InvoiceStatus = "void"
)

// LineItem represents an invoice line item
type LineItem struct {
	Description string `json:"description"`
	Quantity    int64  `json:"quantity"`
	UnitPrice   int64  `json:"unit_price"` // In cents
	Amount      int64  `json:"amount"`     // In cents
}

// Customer represents billing customer information
type Customer struct {
	ID              string    `json:"id" db:"id"`
	TenantID        string    `json:"tenant_id" db:"tenant_id"`
	Email           string    `json:"email" db:"email"`
	Name            string    `json:"name" db:"name"`
	Company         string    `json:"company,omitempty" db:"company"`
	AddressLine1    string    `json:"address_line1,omitempty" db:"address_line1"`
	AddressLine2    string    `json:"address_line2,omitempty" db:"address_line2"`
	City            string    `json:"city,omitempty" db:"city"`
	State           string    `json:"state,omitempty" db:"state"`
	PostalCode      string    `json:"postal_code,omitempty" db:"postal_code"`
	Country         string    `json:"country,omitempty" db:"country"`
	TaxID           string    `json:"tax_id,omitempty" db:"tax_id"`
	StripeCustomerID string   `json:"stripe_customer_id,omitempty" db:"stripe_customer_id"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// PaymentMethod represents a saved payment method
type PaymentMethod struct {
	ID                   string    `json:"id" db:"id"`
	TenantID             string    `json:"tenant_id" db:"tenant_id"`
	Type                 string    `json:"type" db:"type"` // card, bank_account
	IsDefault            bool      `json:"is_default" db:"is_default"`
	CardBrand            string    `json:"card_brand,omitempty" db:"card_brand"`
	CardLast4            string    `json:"card_last4,omitempty" db:"card_last4"`
	CardExpMonth         int       `json:"card_exp_month,omitempty" db:"card_exp_month"`
	CardExpYear          int       `json:"card_exp_year,omitempty" db:"card_exp_year"`
	StripePaymentMethodID string   `json:"stripe_payment_method_id,omitempty" db:"stripe_payment_method_id"`
	CreatedAt            time.Time `json:"created_at" db:"created_at"`
}

// TeamMember represents a team member
type TeamMember struct {
	ID        string          `json:"id" db:"id"`
	TenantID  string          `json:"tenant_id" db:"tenant_id"`
	UserID    string          `json:"user_id" db:"user_id"`
	Email     string          `json:"email" db:"email"`
	Name      string          `json:"name" db:"name"`
	Role      TeamRole        `json:"role" db:"role"`
	Status    MemberStatus    `json:"status" db:"status"`
	InvitedBy string          `json:"invited_by,omitempty" db:"invited_by"`
	InvitedAt time.Time       `json:"invited_at" db:"invited_at"`
	JoinedAt  *time.Time      `json:"joined_at,omitempty" db:"joined_at"`
}

// TeamRole represents team member role
type TeamRole string

const (
	TeamRoleOwner   TeamRole = "owner"
	TeamRoleAdmin   TeamRole = "admin"
	TeamRoleMember  TeamRole = "member"
	TeamRoleViewer  TeamRole = "viewer"
)

// MemberStatus represents member status
type MemberStatus string

const (
	MemberStatusPending  MemberStatus = "pending"
	MemberStatusActive   MemberStatus = "active"
	MemberStatusInactive MemberStatus = "inactive"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID         string                 `json:"id" db:"id"`
	TenantID   string                 `json:"tenant_id" db:"tenant_id"`
	UserID     string                 `json:"user_id,omitempty" db:"user_id"`
	Action     string                 `json:"action" db:"action"`
	Resource   string                 `json:"resource" db:"resource"`
	ResourceID string                 `json:"resource_id,omitempty" db:"resource_id"`
	IPAddress  string                 `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent  string                 `json:"user_agent,omitempty" db:"user_agent"`
	Details    map[string]interface{} `json:"details,omitempty" db:"details"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
}

// Predefined plans
var (
	FreePlan = Plan{
		ID:           "free",
		Name:         "Free",
		Slug:         "free",
		Description:  "For individual developers and small projects",
		PriceMonthly: 0,
		PriceYearly:  0,
		IsFree:       true,
		IsActive:     true,
		Features: PlanFeatures{
			APIAccess:       true,
			Transformations: true,
		},
		Limits: PlanLimits{
			MonthlyWebhooks:    10000,
			EndpointsPerTenant: 5,
			RequestsPerMinute:  60,
			PayloadSizeBytes:   64 * 1024, // 64KB
			RetentionDays:      7,
			TeamMemberCount:    1,
			TransformTimeoutMs: 1000,
			MaxRetries:         3,
		},
	}

	StarterPlan = Plan{
		ID:           "starter",
		Name:         "Starter",
		Slug:         "starter",
		Description:  "For growing teams and applications",
		PriceMonthly: 2900,  // $29
		PriceYearly:  29000, // $290
		TrialDays:    14,
		IsActive:     true,
		Features: PlanFeatures{
			APIAccess:         true,
			Transformations:   true,
			SchemaRegistry:    true,
			AdvancedAnalytics: true,
			TeamMembers:       true,
		},
		Limits: PlanLimits{
			MonthlyWebhooks:    100000,
			EndpointsPerTenant: 25,
			RequestsPerMinute:  300,
			PayloadSizeBytes:   256 * 1024, // 256KB
			RetentionDays:      30,
			TeamMemberCount:    5,
			TransformTimeoutMs: 3000,
			MaxRetries:         5,
		},
	}

	ProPlan = Plan{
		ID:           "pro",
		Name:         "Pro",
		Slug:         "pro",
		Description:  "For scaling businesses with advanced needs",
		PriceMonthly: 9900,  // $99
		PriceYearly:  99000, // $990
		TrialDays:    14,
		IsActive:     true,
		Features: PlanFeatures{
			APIAccess:         true,
			Transformations:   true,
			SchemaRegistry:    true,
			AdvancedAnalytics: true,
			TeamMembers:       true,
			CustomDomains:     true,
			AuditLogs:         true,
			WebhooksGateway:   true,
			PrioritySupport:   true,
		},
		Limits: PlanLimits{
			MonthlyWebhooks:    1000000,
			EndpointsPerTenant: 100,
			RequestsPerMinute:  1000,
			PayloadSizeBytes:   1024 * 1024, // 1MB
			RetentionDays:      90,
			TeamMemberCount:    20,
			TransformTimeoutMs: 5000,
			MaxRetries:         10,
		},
	}

	EnterprisePlan = Plan{
		ID:           "enterprise",
		Name:         "Enterprise",
		Slug:         "enterprise",
		Description:  "For large organizations with custom requirements",
		PriceMonthly: 49900,  // $499 (base, custom pricing)
		PriceYearly:  499000, // $4990
		IsActive:     true,
		Features: PlanFeatures{
			APIAccess:          true,
			Transformations:    true,
			SchemaRegistry:     true,
			AdvancedAnalytics:  true,
			TeamMembers:        true,
			CustomDomains:      true,
			AuditLogs:          true,
			WebhooksGateway:    true,
			PrioritySupport:    true,
			SSO:                true,
			DedicatedInfra:     true,
			MultiRegion:        true,
			SLAGuarantee:       true,
		},
		Limits: PlanLimits{
			MonthlyWebhooks:    0, // Unlimited
			EndpointsPerTenant: 0, // Unlimited
			RequestsPerMinute:  10000,
			PayloadSizeBytes:   10 * 1024 * 1024, // 10MB
			RetentionDays:      365,
			TeamMemberCount:    0, // Unlimited
			TransformTimeoutMs: 10000,
			MaxRetries:         20,
		},
	}
)

// GetAllPlans returns all available plans
func GetAllPlans() []Plan {
	return []Plan{FreePlan, StarterPlan, ProPlan, EnterprisePlan}
}

// GetPlanByID returns a plan by ID
func GetPlanByID(id string) *Plan {
	for _, plan := range GetAllPlans() {
		if plan.ID == id {
			return &plan
		}
	}
	return nil
}
