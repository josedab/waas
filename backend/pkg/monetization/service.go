// Package monetization provides webhook monetization platform for metered billing
package monetization

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/utils"
)

var (
	ErrPlanNotFound         = errors.New("plan not found")
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrAPIKeyNotFound       = errors.New("API key not found")
	ErrQuotaExceeded        = errors.New("quota exceeded")
	ErrInvalidPlan          = errors.New("invalid plan")
	ErrCustomerNotFound     = errors.New("customer not found")
)

// PricingModel represents pricing models
type PricingModel string

const (
	PricingUsageBased   PricingModel = "usage_based"     // Pay per webhook
	PricingTiered       PricingModel = "tiered"          // Tiered pricing
	PricingFlatRate     PricingModel = "flat_rate"       // Fixed monthly fee
	PricingHybrid       PricingModel = "hybrid"          // Base fee + usage
)

// BillingPeriod represents billing periods
type BillingPeriod string

const (
	BillingMonthly  BillingPeriod = "monthly"
	BillingAnnual   BillingPeriod = "annual"
	BillingWeekly   BillingPeriod = "weekly"
)

// SubscriptionStatus represents subscription status
type SubscriptionStatus string

const (
	SubscriptionActive    SubscriptionStatus = "active"
	SubscriptionPaused    SubscriptionStatus = "paused"
	SubscriptionCancelled SubscriptionStatus = "cancelled"
	SubscriptionTrialing  SubscriptionStatus = "trialing"
	SubscriptionPastDue   SubscriptionStatus = "past_due"
)

// Plan represents a monetization plan
type Plan struct {
	ID              string        `json:"id"`
	TenantID        string        `json:"tenant_id"`
	Name            string        `json:"name"`
	Description     string        `json:"description,omitempty"`
	PricingModel    PricingModel  `json:"pricing_model"`
	BillingPeriod   BillingPeriod `json:"billing_period"`
	BasePrice       int64         `json:"base_price"`        // In cents
	PricePerWebhook int64         `json:"price_per_webhook"` // In cents (for usage-based)
	IncludedWebhooks int64        `json:"included_webhooks"` // Free tier
	Tiers           []PricingTier `json:"tiers,omitempty"`
	Features        PlanFeatures  `json:"features"`
	Limits          PlanLimits    `json:"limits"`
	TrialDays       int           `json:"trial_days"`
	Active          bool          `json:"active"`
	Public          bool          `json:"public"` // Visible in pricing page
	SortOrder       int           `json:"sort_order"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

// PricingTier represents a pricing tier for tiered pricing
type PricingTier struct {
	UpTo    int64 `json:"up_to"`    // 0 = unlimited
	Price   int64 `json:"price"`    // Per-unit price in cents
	FlatFee int64 `json:"flat_fee"` // Flat fee for this tier in cents
}

// PlanFeatures represents features included in a plan
type PlanFeatures struct {
	CustomDomains       bool `json:"custom_domains"`
	WebhookRetry        bool `json:"webhook_retry"`
	AdvancedAnalytics   bool `json:"advanced_analytics"`
	PrioritySupport     bool `json:"priority_support"`
	SLAGuarantee        bool `json:"sla_guarantee"`
	CustomTransforms    bool `json:"custom_transforms"`
	WhiteLabelPortal    bool `json:"white_label_portal"`
	APIAccess           bool `json:"api_access"`
	WebhookArchive      bool `json:"webhook_archive"`
	TeamMembers         int  `json:"team_members"`
	DataRetentionDays   int  `json:"data_retention_days"`
}

// PlanLimits represents usage limits
type PlanLimits struct {
	WebhooksPerMonth  int64 `json:"webhooks_per_month"`  // 0 = unlimited
	WebhooksPerDay    int64 `json:"webhooks_per_day"`    // 0 = unlimited
	WebhooksPerSecond int64 `json:"webhooks_per_second"` // Rate limit
	EndpointsMax      int   `json:"endpoints_max"`
	PayloadSizeKB     int   `json:"payload_size_kb"`
	RetentionDays     int   `json:"retention_days"`
}

// Customer represents a monetization customer
type Customer struct {
	ID            string            `json:"id"`
	TenantID      string            `json:"tenant_id"`      // Provider tenant
	ExternalID    string            `json:"external_id"`    // Customer's ID in provider's system
	Name          string            `json:"name"`
	Email         string            `json:"email"`
	Company       string            `json:"company,omitempty"`
	StripeID      string            `json:"stripe_id,omitempty"`
	BillingEmail  string            `json:"billing_email,omitempty"`
	PaymentMethod string            `json:"payment_method,omitempty"`
	Currency      string            `json:"currency"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

// CustomerSubscription represents a customer's subscription
type CustomerSubscription struct {
	ID               string             `json:"id"`
	TenantID         string             `json:"tenant_id"`
	CustomerID       string             `json:"customer_id"`
	PlanID           string             `json:"plan_id"`
	Status           SubscriptionStatus `json:"status"`
	CurrentPeriodStart time.Time        `json:"current_period_start"`
	CurrentPeriodEnd time.Time          `json:"current_period_end"`
	CancelAt         *time.Time         `json:"cancel_at,omitempty"`
	CancelledAt      *time.Time         `json:"cancelled_at,omitempty"`
	TrialStart       *time.Time         `json:"trial_start,omitempty"`
	TrialEnd         *time.Time         `json:"trial_end,omitempty"`
	StripeSubID      string             `json:"stripe_sub_id,omitempty"`
	Quantity         int                `json:"quantity"` // For seat-based billing
	Metadata         map[string]string  `json:"metadata,omitempty"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
}

// APIKey represents a customer's API key
type APIKey struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	CustomerID  string    `json:"customer_id"`
	Name        string    `json:"name"`
	KeyPrefix   string    `json:"key_prefix"` // e.g., "whk_live_"
	KeyHash     string    `json:"-"`          // Hashed key
	LastChars   string    `json:"last_chars"` // Last 4 chars for display
	Scopes      []string  `json:"scopes"`
	RateLimit   int       `json:"rate_limit"` // Requests per second
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UsageRecord represents a usage record
type UsageRecord struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	CustomerID   string    `json:"customer_id"`
	SubscriptionID string  `json:"subscription_id"`
	PeriodStart  time.Time `json:"period_start"`
	PeriodEnd    time.Time `json:"period_end"`
	WebhooksSent int64     `json:"webhooks_sent"`
	WebhooksSuccess int64  `json:"webhooks_success"`
	WebhooksFailed int64   `json:"webhooks_failed"`
	BytesTransferred int64 `json:"bytes_transferred"`
	UniqueEndpoints int    `json:"unique_endpoints"`
	Cost           int64   `json:"cost"` // Calculated cost in cents
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Invoice represents an invoice
type Invoice struct {
	ID             string          `json:"id"`
	TenantID       string          `json:"tenant_id"`
	CustomerID     string          `json:"customer_id"`
	SubscriptionID string          `json:"subscription_id,omitempty"`
	Number         string          `json:"number"` // INV-2024-0001
	Status         string          `json:"status"` // draft, open, paid, void, uncollectible
	Currency       string          `json:"currency"`
	Subtotal       int64           `json:"subtotal"`
	Tax            int64           `json:"tax"`
	Total          int64           `json:"total"`
	AmountPaid     int64           `json:"amount_paid"`
	AmountDue      int64           `json:"amount_due"`
	LineItems      []InvoiceItem   `json:"line_items"`
	PeriodStart    time.Time       `json:"period_start"`
	PeriodEnd      time.Time       `json:"period_end"`
	DueDate        time.Time       `json:"due_date"`
	PaidAt         *time.Time      `json:"paid_at,omitempty"`
	StripeInvoiceID string         `json:"stripe_invoice_id,omitempty"`
	PDF            string          `json:"pdf,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

// InvoiceItem represents a line item on an invoice
type InvoiceItem struct {
	Description string `json:"description"`
	Quantity    int64  `json:"quantity"`
	UnitPrice   int64  `json:"unit_price"`
	Amount      int64  `json:"amount"`
}

// UsageDashboard represents usage dashboard data
type UsageDashboard struct {
	Customer        *Customer             `json:"customer"`
	Subscription    *CustomerSubscription `json:"subscription,omitempty"`
	Plan            *Plan                 `json:"plan,omitempty"`
	CurrentUsage    *UsageRecord          `json:"current_usage"`
	UsageHistory    []UsageRecord         `json:"usage_history"`
	InvoiceHistory  []Invoice             `json:"invoice_history"`
	Limits          *PlanLimits           `json:"limits,omitempty"`
	RemainingQuota  int64                 `json:"remaining_quota"`
	UsagePercent    float64               `json:"usage_percent"`
	ProjectedCost   int64                 `json:"projected_cost"` // In cents
	DaysRemaining   int                   `json:"days_remaining"`
}

// Repository defines the interface for monetization data storage
type Repository interface {
	// Plans
	CreatePlan(ctx context.Context, plan *Plan) error
	GetPlan(ctx context.Context, tenantID, planID string) (*Plan, error)
	UpdatePlan(ctx context.Context, plan *Plan) error
	DeletePlan(ctx context.Context, tenantID, planID string) error
	ListPlans(ctx context.Context, tenantID string, publicOnly bool) ([]Plan, error)

	// Customers
	CreateCustomer(ctx context.Context, customer *Customer) error
	GetCustomer(ctx context.Context, tenantID, customerID string) (*Customer, error)
	GetCustomerByExternalID(ctx context.Context, tenantID, externalID string) (*Customer, error)
	UpdateCustomer(ctx context.Context, customer *Customer) error
	DeleteCustomer(ctx context.Context, tenantID, customerID string) error
	ListCustomers(ctx context.Context, tenantID string, limit, offset int) ([]Customer, int, error)

	// Subscriptions
	CreateSubscription(ctx context.Context, sub *CustomerSubscription) error
	GetSubscription(ctx context.Context, subscriptionID string) (*CustomerSubscription, error)
	GetActiveSubscription(ctx context.Context, tenantID, customerID string) (*CustomerSubscription, error)
	UpdateSubscription(ctx context.Context, sub *CustomerSubscription) error
	ListSubscriptions(ctx context.Context, tenantID string, status *SubscriptionStatus) ([]CustomerSubscription, error)

	// API Keys
	CreateAPIKey(ctx context.Context, key *APIKey) error
	GetAPIKey(ctx context.Context, keyID string) (*APIKey, error)
	GetAPIKeyByHash(ctx context.Context, keyHash string) (*APIKey, error)
	UpdateAPIKey(ctx context.Context, key *APIKey) error
	DeleteAPIKey(ctx context.Context, keyID string) error
	ListAPIKeys(ctx context.Context, tenantID, customerID string) ([]APIKey, error)

	// Usage
	CreateUsageRecord(ctx context.Context, record *UsageRecord) error
	GetUsageRecord(ctx context.Context, subscriptionID string, periodStart time.Time) (*UsageRecord, error)
	UpdateUsageRecord(ctx context.Context, record *UsageRecord) error
	ListUsageRecords(ctx context.Context, customerID string, since time.Time) ([]UsageRecord, error)
	IncrementUsage(ctx context.Context, subscriptionID string, webhooks, bytes int64) error

	// Invoices
	CreateInvoice(ctx context.Context, invoice *Invoice) error
	GetInvoice(ctx context.Context, invoiceID string) (*Invoice, error)
	UpdateInvoice(ctx context.Context, invoice *Invoice) error
	ListInvoices(ctx context.Context, customerID string, limit int) ([]Invoice, error)
	GetNextInvoiceNumber(ctx context.Context, tenantID string) (string, error)
}

// BillingProvider defines the interface for payment processing
type BillingProvider interface {
	CreateCustomer(ctx context.Context, customer *Customer) (string, error)
	UpdateCustomer(ctx context.Context, customer *Customer) error
	DeleteCustomer(ctx context.Context, stripeID string) error
	CreateSubscription(ctx context.Context, sub *CustomerSubscription, plan *Plan) (string, error)
	CancelSubscription(ctx context.Context, stripeSubID string, immediately bool) error
	CreateInvoice(ctx context.Context, invoice *Invoice) (string, error)
	FinalizeInvoice(ctx context.Context, stripeInvoiceID string) error
	ReportUsage(ctx context.Context, stripeSubID string, quantity int64, timestamp time.Time) error
	GetPaymentMethods(ctx context.Context, stripeCustomerID string) ([]PaymentMethod, error)
}

// PaymentMethod represents a payment method
type PaymentMethod struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // card, bank_account
	Last4    string `json:"last4"`
	Brand    string `json:"brand,omitempty"`
	ExpMonth int    `json:"exp_month,omitempty"`
	ExpYear  int    `json:"exp_year,omitempty"`
	Default  bool   `json:"default"`
}

// Service provides monetization operations
type Service struct {
	repo      Repository
	billing   BillingProvider
	logger    *utils.Logger
	mu        sync.RWMutex
	config    *ServiceConfig
	planCache map[string]*Plan
}

// ServiceConfig holds service configuration
type ServiceConfig struct {
	DefaultCurrency     string
	DefaultTrialDays    int
	InvoiceDueDays      int
	UsageSyncIntervalSec int
	EnableStripe        bool
}

// DefaultServiceConfig returns default configuration
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		DefaultCurrency:      "usd",
		DefaultTrialDays:     14,
		InvoiceDueDays:       30,
		UsageSyncIntervalSec: 3600,
		EnableStripe:         false,
	}
}

// NewService creates a new monetization service
func NewService(repo Repository, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}

	return &Service{
		repo:      repo,
		logger:    utils.NewLogger("monetization-service"),
		config:    config,
		planCache: make(map[string]*Plan),
	}
}

// SetBillingProvider sets the billing provider
func (s *Service) SetBillingProvider(provider BillingProvider) {
	s.billing = provider
}

// CreatePlan creates a new pricing plan
func (s *Service) CreatePlan(ctx context.Context, tenantID string, plan *Plan) (*Plan, error) {
	plan.ID = uuid.New().String()
	plan.TenantID = tenantID
	plan.Active = true
	plan.CreatedAt = time.Now()
	plan.UpdatedAt = time.Now()

	if err := s.repo.CreatePlan(ctx, plan); err != nil {
		return nil, err
	}

	// Update cache
	s.mu.Lock()
	s.planCache[plan.ID] = plan
	s.mu.Unlock()

	return plan, nil
}

// GetPlan retrieves a plan
func (s *Service) GetPlan(ctx context.Context, tenantID, planID string) (*Plan, error) {
	// Check cache
	s.mu.RLock()
	if plan, ok := s.planCache[planID]; ok {
		s.mu.RUnlock()
		return plan, nil
	}
	s.mu.RUnlock()

	return s.repo.GetPlan(ctx, tenantID, planID)
}

// ListPlans lists available plans
func (s *Service) ListPlans(ctx context.Context, tenantID string, publicOnly bool) ([]Plan, error) {
	return s.repo.ListPlans(ctx, tenantID, publicOnly)
}

// CreateCustomer creates a new customer
func (s *Service) CreateCustomer(ctx context.Context, tenantID string, customer *Customer) (*Customer, error) {
	customer.ID = uuid.New().String()
	customer.TenantID = tenantID
	if customer.Currency == "" {
		customer.Currency = s.config.DefaultCurrency
	}
	customer.CreatedAt = time.Now()
	customer.UpdatedAt = time.Now()

	// Create in Stripe if enabled
	if s.billing != nil && s.config.EnableStripe {
		stripeID, err := s.billing.CreateCustomer(ctx, customer)
		if err != nil {
			return nil, fmt.Errorf("failed to create Stripe customer: %w", err)
		}
		customer.StripeID = stripeID
	}

	if err := s.repo.CreateCustomer(ctx, customer); err != nil {
		return nil, err
	}

	return customer, nil
}

// GetCustomer retrieves a customer
func (s *Service) GetCustomer(ctx context.Context, tenantID, customerID string) (*Customer, error) {
	return s.repo.GetCustomer(ctx, tenantID, customerID)
}

// SubscribeToPlan subscribes a customer to a plan
func (s *Service) SubscribeToPlan(ctx context.Context, tenantID, customerID, planID string) (*CustomerSubscription, error) {
	// Get plan
	plan, err := s.GetPlan(ctx, tenantID, planID)
	if err != nil {
		return nil, err
	}

	// Check for existing subscription
	existing, _ := s.repo.GetActiveSubscription(ctx, tenantID, customerID)
	if existing != nil {
		return nil, errors.New("customer already has active subscription")
	}

	now := time.Now()
	periodEnd := calculatePeriodEnd(now, plan.BillingPeriod)

	sub := &CustomerSubscription{
		ID:                 uuid.New().String(),
		TenantID:           tenantID,
		CustomerID:         customerID,
		PlanID:             planID,
		Status:             SubscriptionActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   periodEnd,
		Quantity:           1,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	// Handle trial
	if plan.TrialDays > 0 {
		sub.Status = SubscriptionTrialing
		trialEnd := now.AddDate(0, 0, plan.TrialDays)
		sub.TrialStart = &now
		sub.TrialEnd = &trialEnd
	}

	// Create in Stripe if enabled
	if s.billing != nil && s.config.EnableStripe {
		customer, _ := s.repo.GetCustomer(ctx, tenantID, customerID)
		if customer != nil && customer.StripeID != "" {
			stripeSubID, err := s.billing.CreateSubscription(ctx, sub, plan)
			if err != nil {
				return nil, fmt.Errorf("failed to create Stripe subscription: %w", err)
			}
			sub.StripeSubID = stripeSubID
		}
	}

	if err := s.repo.CreateSubscription(ctx, sub); err != nil {
		return nil, err
	}

	// Create initial usage record
	usage := &UsageRecord{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		CustomerID:     customerID,
		SubscriptionID: sub.ID,
		PeriodStart:    sub.CurrentPeriodStart,
		PeriodEnd:      sub.CurrentPeriodEnd,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.repo.CreateUsageRecord(ctx, usage); err != nil {
		s.logger.Error("failed to create usage record", map[string]interface{}{"error": err.Error(), "subscription_id": sub.ID})
	}

	return sub, nil
}

// CancelSubscription cancels a subscription
func (s *Service) CancelSubscription(ctx context.Context, tenantID, subscriptionID string, immediately bool) error {
	sub, err := s.repo.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return err
	}

	if sub.TenantID != tenantID {
		return ErrSubscriptionNotFound
	}

	now := time.Now()

	if immediately {
		sub.Status = SubscriptionCancelled
		sub.CancelledAt = &now
	} else {
		sub.CancelAt = &sub.CurrentPeriodEnd
	}

	sub.UpdatedAt = now

	// Cancel in Stripe
	if s.billing != nil && sub.StripeSubID != "" {
		if err := s.billing.CancelSubscription(ctx, sub.StripeSubID, immediately); err != nil {
			return fmt.Errorf("failed to cancel Stripe subscription: %w", err)
		}
	}

	return s.repo.UpdateSubscription(ctx, sub)
}

// CreateAPIKey creates a new API key for a customer
func (s *Service) CreateAPIKey(ctx context.Context, tenantID, customerID, name string, scopes []string) (*APIKey, string, error) {
	// Generate key
	rawKey := fmt.Sprintf("whk_live_%s", uuid.New().String())
	keyHash := hashAPIKey(rawKey)

	key := &APIKey{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		CustomerID: customerID,
		Name:       name,
		KeyPrefix:  "whk_live_",
		KeyHash:    keyHash,
		LastChars:  rawKey[len(rawKey)-4:],
		Scopes:     scopes,
		RateLimit:  100, // Default rate limit
		Active:     true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.repo.CreateAPIKey(ctx, key); err != nil {
		return nil, "", err
	}

	return key, rawKey, nil
}

// ValidateAPIKey validates an API key and returns the customer
func (s *Service) ValidateAPIKey(ctx context.Context, rawKey string) (*APIKey, error) {
	keyHash := hashAPIKey(rawKey)
	
	key, err := s.repo.GetAPIKeyByHash(ctx, keyHash)
	if err != nil {
		return nil, ErrAPIKeyNotFound
	}

	if !key.Active {
		return nil, ErrAPIKeyNotFound
	}

	if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
		return nil, ErrAPIKeyNotFound
	}

	// Update last used
	now := time.Now()
	key.LastUsedAt = &now
	if err := s.repo.UpdateAPIKey(ctx, key); err != nil {
		s.logger.Error("failed to update API key", map[string]interface{}{"error": err.Error(), "key_id": key.ID})
	}

	return key, nil
}

// RecordUsage records webhook usage
func (s *Service) RecordUsage(ctx context.Context, subscriptionID string, webhooks, bytes int64) error {
	return s.repo.IncrementUsage(ctx, subscriptionID, webhooks, bytes)
}

// CheckQuota checks if a customer has remaining quota
func (s *Service) CheckQuota(ctx context.Context, tenantID, customerID string) (bool, int64, error) {
	sub, err := s.repo.GetActiveSubscription(ctx, tenantID, customerID)
	if err != nil {
		return false, 0, err
	}

	plan, err := s.GetPlan(ctx, tenantID, sub.PlanID)
	if err != nil {
		return false, 0, err
	}

	// Unlimited
	if plan.Limits.WebhooksPerMonth == 0 {
		return true, -1, nil
	}

	// Get current usage
	usage, err := s.repo.GetUsageRecord(ctx, sub.ID, sub.CurrentPeriodStart)
	if err != nil {
		return true, plan.Limits.WebhooksPerMonth, nil // Allow if no usage record
	}

	remaining := plan.Limits.WebhooksPerMonth + plan.IncludedWebhooks - usage.WebhooksSent
	if remaining < 0 {
		remaining = 0
	}

	return remaining > 0, remaining, nil
}

// CalculateCost calculates the cost for usage
func (s *Service) CalculateCost(ctx context.Context, plan *Plan, usage *UsageRecord) int64 {
	webhooks := usage.WebhooksSent

	switch plan.PricingModel {
	case PricingFlatRate:
		return plan.BasePrice

	case PricingUsageBased:
		// Subtract included webhooks
		billable := webhooks - plan.IncludedWebhooks
		if billable < 0 {
			billable = 0
		}
		return plan.BasePrice + (billable * plan.PricePerWebhook)

	case PricingTiered:
		return s.calculateTieredCost(plan, webhooks)

	case PricingHybrid:
		billable := webhooks - plan.IncludedWebhooks
		if billable < 0 {
			billable = 0
		}
		return plan.BasePrice + (billable * plan.PricePerWebhook)
	}

	return 0
}

// calculateTieredCost calculates cost for tiered pricing
func (s *Service) calculateTieredCost(plan *Plan, webhooks int64) int64 {
	if len(plan.Tiers) == 0 {
		return plan.BasePrice
	}

	var totalCost int64
	remaining := webhooks

	for i, tier := range plan.Tiers {
		tierStart := int64(0)
		if i > 0 {
			tierStart = plan.Tiers[i-1].UpTo
		}

		tierEnd := tier.UpTo
		if tierEnd == 0 {
			tierEnd = remaining + tierStart // Unlimited
		}

		tierSize := tierEnd - tierStart
		if remaining <= 0 {
			break
		}

		usedInTier := min(remaining, tierSize)
		totalCost += tier.FlatFee + (usedInTier * tier.Price)
		remaining -= usedInTier
	}

	return totalCost
}

// GetUsageDashboard retrieves usage dashboard for a customer
func (s *Service) GetUsageDashboard(ctx context.Context, tenantID, customerID string) (*UsageDashboard, error) {
	customer, err := s.repo.GetCustomer(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}

	dashboard := &UsageDashboard{
		Customer: customer,
	}

	// Get subscription
	sub, _ := s.repo.GetActiveSubscription(ctx, tenantID, customerID)
	if sub != nil {
		dashboard.Subscription = sub

		// Get plan
		plan, _ := s.GetPlan(ctx, tenantID, sub.PlanID)
		if plan != nil {
			dashboard.Plan = plan
			dashboard.Limits = &plan.Limits
		}

		// Get current usage
		usage, _ := s.repo.GetUsageRecord(ctx, sub.ID, sub.CurrentPeriodStart)
		if usage != nil {
			dashboard.CurrentUsage = usage

			// Calculate remaining quota
			if plan != nil && plan.Limits.WebhooksPerMonth > 0 {
				total := plan.Limits.WebhooksPerMonth + plan.IncludedWebhooks
				dashboard.RemainingQuota = total - usage.WebhooksSent
				if dashboard.RemainingQuota < 0 {
					dashboard.RemainingQuota = 0
				}
				dashboard.UsagePercent = float64(usage.WebhooksSent) / float64(total) * 100
			}

			// Calculate projected cost
			if plan != nil {
				dashboard.ProjectedCost = s.CalculateCost(ctx, plan, usage)
			}
		}

		// Days remaining in period
		dashboard.DaysRemaining = int(time.Until(sub.CurrentPeriodEnd).Hours() / 24)
	}

	// Get usage history
	since := time.Now().AddDate(0, -6, 0) // Last 6 months
	history, _ := s.repo.ListUsageRecords(ctx, customerID, since)
	dashboard.UsageHistory = history

	// Get invoice history
	invoices, _ := s.repo.ListInvoices(ctx, customerID, 10)
	dashboard.InvoiceHistory = invoices

	return dashboard, nil
}

// GenerateInvoice generates an invoice for a subscription period
func (s *Service) GenerateInvoice(ctx context.Context, tenantID, subscriptionID string) (*Invoice, error) {
	sub, err := s.repo.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}

	customer, err := s.repo.GetCustomer(ctx, tenantID, sub.CustomerID)
	if err != nil {
		return nil, err
	}

	plan, err := s.GetPlan(ctx, tenantID, sub.PlanID)
	if err != nil {
		return nil, err
	}

	// Get usage
	usage, err := s.repo.GetUsageRecord(ctx, subscriptionID, sub.CurrentPeriodStart)
	if err != nil {
		// Create empty usage
		usage = &UsageRecord{WebhooksSent: 0}
	}

	// Calculate cost
	cost := s.CalculateCost(ctx, plan, usage)

	// Get invoice number
	invoiceNumber, _ := s.repo.GetNextInvoiceNumber(ctx, tenantID)

	now := time.Now()
	invoice := &Invoice{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		CustomerID:     customer.ID,
		SubscriptionID: sub.ID,
		Number:         invoiceNumber,
		Status:         "draft",
		Currency:       customer.Currency,
		Subtotal:       cost,
		Tax:            0, // Add tax calculation as needed
		Total:          cost,
		AmountDue:      cost,
		PeriodStart:    sub.CurrentPeriodStart,
		PeriodEnd:      sub.CurrentPeriodEnd,
		DueDate:        now.AddDate(0, 0, s.config.InvoiceDueDays),
		CreatedAt:      now,
	}

	// Build line items
	invoice.LineItems = []InvoiceItem{
		{
			Description: fmt.Sprintf("%s - %s to %s", plan.Name, sub.CurrentPeriodStart.Format("Jan 2"), sub.CurrentPeriodEnd.Format("Jan 2")),
			Quantity:    1,
			UnitPrice:   plan.BasePrice,
			Amount:      plan.BasePrice,
		},
	}

	if plan.PricingModel == PricingUsageBased || plan.PricingModel == PricingHybrid {
		billable := usage.WebhooksSent - plan.IncludedWebhooks
		if billable > 0 {
			invoice.LineItems = append(invoice.LineItems, InvoiceItem{
				Description: fmt.Sprintf("Webhooks (%d @ $%.4f each)", billable, float64(plan.PricePerWebhook)/100),
				Quantity:    billable,
				UnitPrice:   plan.PricePerWebhook,
				Amount:      billable * plan.PricePerWebhook,
			})
		}
	}

	if err := s.repo.CreateInvoice(ctx, invoice); err != nil {
		return nil, err
	}

	return invoice, nil
}

// Helper functions

func calculatePeriodEnd(start time.Time, period BillingPeriod) time.Time {
	switch period {
	case BillingMonthly:
		return start.AddDate(0, 1, 0)
	case BillingAnnual:
		return start.AddDate(1, 0, 0)
	case BillingWeekly:
		return start.AddDate(0, 0, 7)
	default:
		return start.AddDate(0, 1, 0)
	}
}

func hashAPIKey(key string) string {
	// In production, use a proper hash like bcrypt or argon2
	// This is a placeholder
	return fmt.Sprintf("hash_%s", key)
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
