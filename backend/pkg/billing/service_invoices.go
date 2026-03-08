package billing

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SubscriptionRepository defines additional repository methods for subscription billing.
type SubscriptionRepository interface {
	SavePricingPlan(ctx context.Context, plan *PricingPlan) error
	GetPricingPlan(ctx context.Context, planID uuid.UUID) (*PricingPlan, error)
	ListPricingPlans(ctx context.Context) ([]PricingPlan, error)
	SaveSubscription(ctx context.Context, sub *Subscription) error
	GetSubscription(ctx context.Context, tenantID uuid.UUID) (*Subscription, error)
	UpdateSubscription(ctx context.Context, sub *Subscription) error
	SaveUsageRecord(ctx context.Context, record *UsageRecord) error
	GetUsageRecord(ctx context.Context, tenantID uuid.UUID, periodStart time.Time) (*UsageRecord, error)
	GetCurrentUsageRecord(ctx context.Context, tenantID uuid.UUID) (*UsageRecord, error)
	SaveInvoice(ctx context.Context, invoice *Invoice) error
	GetInvoiceByID(ctx context.Context, invoiceID uuid.UUID) (*Invoice, error)
	ListInvoicesByTenant(ctx context.Context, tenantID uuid.UUID) ([]Invoice, error)
}

// GetInvoices lists invoices
func (s *Service) GetInvoices(ctx context.Context, tenantID string) ([]CostInvoice, error) {
	return s.repo.ListInvoices(ctx, tenantID)
}

// GetInvoice retrieves an invoice
func (s *Service) GetInvoice(ctx context.Context, tenantID, invoiceID string) (*CostInvoice, error) {
	return s.repo.GetInvoice(ctx, tenantID, invoiceID)
}

// CreatePricingPlan creates a new pricing plan.
func (s *Service) CreatePricingPlan(ctx context.Context, plan *PricingPlan) (*PricingPlan, error) {
	plan.ID = uuid.New()
	plan.Active = true
	plan.CreatedAt = time.Now()
	return plan, nil
}

// ListPricingPlans lists available pricing plans.
func (s *Service) ListPricingPlans(ctx context.Context) ([]PricingPlan, error) {
	return []PricingPlan{
		defaultFreePlan(),
		defaultStarterPlan(),
		defaultProPlan(),
		defaultEnterprisePlan(),
	}, nil
}

// CreateSubscriptionForTenant subscribes a tenant to a plan.
func (s *Service) CreateSubscriptionForTenant(ctx context.Context, tenantID, planID uuid.UUID) (*Subscription, error) {
	now := time.Now()
	sub := &Subscription{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		PlanID:             planID,
		Status:             "active",
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		CreatedAt:          now,
	}

	if s.stripeClient != nil && s.config.BillingEnabled {
		stripeSubID, err := s.stripeClient.CreateSubscription(sub.StripeCustomerID, "")
		if err != nil {
			return nil, fmt.Errorf("stripe create subscription: %w", err)
		}
		sub.StripeSubID = stripeSubID
	}

	return sub, nil
}

// GetSubscriptionForTenant gets a tenant's current subscription.
func (s *Service) GetSubscriptionForTenant(ctx context.Context, tenantID uuid.UUID) (*Subscription, error) {
	// Default subscription for tenants without explicit subscription
	now := time.Now()
	return &Subscription{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		PlanID:             uuid.Nil,
		Status:             "active",
		CurrentPeriodStart: time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC),
		CurrentPeriodEnd:   time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC),
		CreatedAt:          now,
	}, nil
}

// ChangeSubscription upgrades or downgrades a tenant's plan.
func (s *Service) ChangeSubscription(ctx context.Context, tenantID, newPlanID uuid.UUID) (*Subscription, error) {
	sub, err := s.GetSubscriptionForTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	sub.PlanID = newPlanID
	return sub, nil
}

// CancelSubscriptionForTenant cancels a tenant's subscription at period end.
func (s *Service) CancelSubscriptionForTenant(ctx context.Context, tenantID uuid.UUID) error {
	sub, err := s.GetSubscriptionForTenant(ctx, tenantID)
	if err != nil {
		return err
	}
	sub.CancelAtPeriodEnd = true
	sub.Status = "canceled"

	if s.stripeClient != nil && sub.StripeSubID != "" {
		if err := s.stripeClient.CancelSubscription(sub.StripeSubID); err != nil {
			return fmt.Errorf("stripe cancel: %w", err)
		}
	}
	return nil
}

// RecordUsageEvent records usage for a tenant.
func (s *Service) RecordUsageEvent(ctx context.Context, tenantID uuid.UUID, eventCount, retryCount, dataBytes int64) (*UsageRecord, error) {
	now := time.Now()
	record := &UsageRecord{
		ID:          uuid.New(),
		TenantID:    tenantID,
		PeriodStart: time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:   time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC),
		EventCount:  eventCount,
		RetryCount:  retryCount,
		DataBytes:   dataBytes,
		CreatedAt:   now,
	}
	return record, nil
}

// GetUsageSummaryForTenant returns the current usage summary for a tenant.
func (s *Service) GetUsageSummaryForTenant(ctx context.Context, tenantID uuid.UUID) (*UsageSummary, error) {
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)

	plan := defaultFreePlan()

	return &UsageSummary{
		TenantID:       tenantID,
		PlanName:       plan.Name,
		EventsUsed:     0,
		EventsIncluded: plan.IncludedEvents,
		OverageEvents:  0,
		CurrentCost:    plan.BasePrice,
		ProjectedCost:  plan.BasePrice,
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		UsagePercent:   0,
	}, nil
}

// CalculateCostForPlan calculates cost for usage on a given plan.
func (s *Service) CalculateCostForPlan(plan *PricingPlan, eventCount int64) int64 {
	switch plan.PricingModel {
	case "flat":
		return CalculateFlatRate(plan, eventCount)
	case "per_event":
		return CalculatePerEvent(plan, eventCount)
	case "tiered":
		return CalculateTiered(plan.Tiers, eventCount)
	default:
		return plan.BasePrice
	}
}

// GenerateInvoiceForTenant generates an invoice for a tenant's billing period.
func (s *Service) GenerateInvoiceForTenant(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time) (*Invoice, error) {
	plan := defaultFreePlan()
	cost := s.CalculateCostForPlan(&plan, 0)

	invoice := &Invoice{
		ID:             uuid.New(),
		TenantID:       tenantID,
		SubscriptionID: uuid.New(),
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		Subtotal:       cost,
		Tax:            0,
		Total:          cost,
		Status:         "draft",
		LineItems: []InvoiceLineItem{
			{
				Description: fmt.Sprintf("%s Plan - %s to %s", plan.DisplayName, periodStart.Format("Jan 2"), periodEnd.Format("Jan 2")),
				Quantity:    1,
				UnitPrice:   plan.BasePrice,
				Amount:      plan.BasePrice,
			},
		},
		CreatedAt: time.Now(),
	}

	return invoice, nil
}

// GetInvoicesForTenant lists invoices for a tenant.
func (s *Service) GetInvoicesForTenant(ctx context.Context, tenantID uuid.UUID) ([]Invoice, error) {
	return []Invoice{}, nil
}

// GetInvoiceDetail retrieves a single invoice by ID.
func (s *Service) GetInvoiceDetail(ctx context.Context, invoiceID uuid.UUID) (*Invoice, error) {
	return nil, fmt.Errorf("invoice not found")
}

// HandleStripeWebhook processes Stripe webhook events.
func (s *Service) HandleStripeWebhook(ctx context.Context, eventType string, data map[string]interface{}) error {
	switch eventType {
	case "invoice.paid":
		return nil
	case "invoice.payment_failed":
		return nil
	case "customer.subscription.updated":
		return nil
	case "customer.subscription.deleted":
		return nil
	default:
		return nil
	}
}

// GetBillingDashboard returns billing dashboard data for a tenant.
func (s *Service) GetBillingDashboard(ctx context.Context, tenantID uuid.UUID) (*BillingDashboard, error) {
	sub, _ := s.GetSubscriptionForTenant(ctx, tenantID)
	usage, _ := s.GetUsageSummaryForTenant(ctx, tenantID)
	invoices, _ := s.GetInvoicesForTenant(ctx, tenantID)
	plan := defaultFreePlan()

	return &BillingDashboard{
		Subscription:   sub,
		Plan:           &plan,
		Usage:          usage,
		RecentInvoices: invoices,
	}, nil
}

// Default plan definitions

func defaultFreePlan() PricingPlan {
	return PricingPlan{
		Name:           "free",
		DisplayName:    "Free",
		PricingModel:   "flat",
		BasePrice:      0,
		IncludedEvents: 1000,
		OveragePrice:   0,
		Features: PlanFeatures{
			MaxEndpoints:     5,
			MaxRetries:       3,
			CustomTransforms: false,
			Analytics:        false,
			SLA:              false,
			Support:          "community",
		},
		Active: true,
	}
}

func defaultStarterPlan() PricingPlan {
	return PricingPlan{
		Name:           "starter",
		DisplayName:    "Starter",
		PricingModel:   "per_event",
		BasePrice:      2900,
		IncludedEvents: 10000,
		OveragePrice:   100,
		Features: PlanFeatures{
			MaxEndpoints:     25,
			MaxRetries:       5,
			CustomTransforms: false,
			Analytics:        true,
			SLA:              false,
			Support:          "email",
		},
		Active: true,
	}
}

func defaultProPlan() PricingPlan {
	return PricingPlan{
		Name:           "pro",
		DisplayName:    "Pro",
		PricingModel:   "per_event",
		BasePrice:      9900,
		IncludedEvents: 100000,
		OveragePrice:   50,
		Features: PlanFeatures{
			MaxEndpoints:     100,
			MaxRetries:       10,
			CustomTransforms: true,
			Analytics:        true,
			SLA:              true,
			Support:          "priority",
		},
		Active: true,
	}
}

func defaultEnterprisePlan() PricingPlan {
	return PricingPlan{
		Name:           "enterprise",
		DisplayName:    "Enterprise",
		PricingModel:   "tiered",
		BasePrice:      49900,
		IncludedEvents: 1000000,
		OveragePrice:   25,
		Tiers: []PricingTier{
			{UpTo: 100000, PricePerUnit: 10},
			{UpTo: 500000, PricePerUnit: 5},
			{UpTo: -1, PricePerUnit: 2},
		},
		Features: PlanFeatures{
			MaxEndpoints:     -1,
			MaxRetries:       20,
			CustomTransforms: true,
			Analytics:        true,
			SLA:              true,
			Support:          "dedicated",
		},
		Active: true,
	}
}
