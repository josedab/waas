package billing

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Calculator Tests ---

func TestCalculateFlatRate(t *testing.T) {
	plan := &PricingPlan{
		PricingModel: "flat",
		BasePrice:    2900,
	}

	assert.Equal(t, int64(2900), CalculateFlatRate(plan, 0))
	assert.Equal(t, int64(2900), CalculateFlatRate(plan, 1000))
	assert.Equal(t, int64(2900), CalculateFlatRate(plan, 999999))
}

func TestCalculatePerEvent(t *testing.T) {
	plan := &PricingPlan{
		PricingModel:   "per_event",
		BasePrice:      2900,
		IncludedEvents: 10000,
		OveragePrice:   100, // per 1000 events
	}

	tests := []struct {
		name     string
		usage    int64
		expected int64
	}{
		{"zero usage", 0, 2900},
		{"within included", 5000, 2900},
		{"at included limit", 10000, 2900},
		{"1 event over", 10001, 3000},
		{"1000 events over", 11000, 3000},
		{"5000 events over", 15000, 3400},
		{"large overage", 110000, 12900},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculatePerEvent(plan, tt.usage)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateTiered(t *testing.T) {
	tiers := []PricingTier{
		{UpTo: 1000, PricePerUnit: 10},
		{UpTo: 10000, PricePerUnit: 5},
		{UpTo: -1, PricePerUnit: 2},
	}

	tests := []struct {
		name     string
		usage    int64
		expected int64
	}{
		{"zero usage", 0, 0},
		{"within first tier", 500, 5000},
		{"at first tier limit", 1000, 10000},
		{"into second tier", 2000, 10000 + 5000},
		{"at second tier limit", 10000, 10000 + 45000},
		{"into unlimited tier", 15000, 10000 + 45000 + 10000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateTiered(tiers, tt.usage)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateTiered_EmptyTiers(t *testing.T) {
	assert.Equal(t, int64(0), CalculateTiered(nil, 1000))
	assert.Equal(t, int64(0), CalculateTiered([]PricingTier{}, 1000))
}

func TestCalculateOverage(t *testing.T) {
	plan := &PricingPlan{IncludedEvents: 10000}

	tests := []struct {
		name     string
		usage    int64
		expected int64
	}{
		{"no usage", 0, 0},
		{"within included", 5000, 0},
		{"at limit", 10000, 0},
		{"over limit", 15000, 5000},
		{"way over limit", 100000, 90000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateOverage(plan, tt.usage)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// --- Service Tests ---

func TestCalculateCostForPlan_FlatRate(t *testing.T) {
	svc := NewService(nil, nil, nil)
	plan := &PricingPlan{
		PricingModel: "flat",
		BasePrice:    4900,
	}
	assert.Equal(t, int64(4900), svc.CalculateCostForPlan(plan, 0))
	assert.Equal(t, int64(4900), svc.CalculateCostForPlan(plan, 999999))
}

func TestCalculateCostForPlan_PerEvent(t *testing.T) {
	svc := NewService(nil, nil, nil)
	plan := &PricingPlan{
		PricingModel:   "per_event",
		BasePrice:      2900,
		IncludedEvents: 10000,
		OveragePrice:   100,
	}
	assert.Equal(t, int64(2900), svc.CalculateCostForPlan(plan, 5000))
	assert.Equal(t, int64(3400), svc.CalculateCostForPlan(plan, 15000))
}

func TestCalculateCostForPlan_Tiered(t *testing.T) {
	svc := NewService(nil, nil, nil)
	plan := &PricingPlan{
		PricingModel: "tiered",
		Tiers: []PricingTier{
			{UpTo: 1000, PricePerUnit: 10},
			{UpTo: 5000, PricePerUnit: 5},
			{UpTo: -1, PricePerUnit: 2},
		},
	}
	// 1000*10 + 4000*5 = 30000
	assert.Equal(t, int64(30000), svc.CalculateCostForPlan(plan, 5000))
}

func TestCalculateCostForPlan_Unknown(t *testing.T) {
	svc := NewService(nil, nil, nil)
	plan := &PricingPlan{
		PricingModel: "unknown",
		BasePrice:    1000,
	}
	assert.Equal(t, int64(1000), svc.CalculateCostForPlan(plan, 5000))
}

// --- Usage Recording ---

func TestRecordUsageEvent(t *testing.T) {
	svc := NewService(nil, nil, nil)
	ctx := context.Background()
	tenantID := uuid.New()

	record, err := svc.RecordUsageEvent(ctx, tenantID, 100, 5, 2048)
	require.NoError(t, err)
	require.NotNil(t, record)

	assert.Equal(t, tenantID, record.TenantID)
	assert.Equal(t, int64(100), record.EventCount)
	assert.Equal(t, int64(5), record.RetryCount)
	assert.Equal(t, int64(2048), record.DataBytes)
	assert.False(t, record.ID == uuid.Nil)
	assert.False(t, record.PeriodStart.IsZero())
	assert.True(t, record.PeriodEnd.After(record.PeriodStart))
}

// --- Invoice Generation ---

func TestGenerateInvoiceForTenant(t *testing.T) {
	svc := NewService(nil, nil, nil)
	ctx := context.Background()
	tenantID := uuid.New()

	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)

	invoice, err := svc.GenerateInvoiceForTenant(ctx, tenantID, periodStart, periodEnd)
	require.NoError(t, err)
	require.NotNil(t, invoice)

	assert.Equal(t, tenantID, invoice.TenantID)
	assert.Equal(t, "draft", invoice.Status)
	assert.Equal(t, periodStart, invoice.PeriodStart)
	assert.Equal(t, periodEnd, invoice.PeriodEnd)
	assert.True(t, len(invoice.LineItems) > 0)
	assert.Equal(t, invoice.Subtotal, invoice.Total)
}

// --- Cost Projection ---

func TestProjectCost(t *testing.T) {
	svc := NewService(nil, nil, nil)
	ctx := context.Background()
	tenantID := uuid.New()

	summary, err := svc.ProjectCost(ctx, tenantID, 30)
	require.NoError(t, err)
	require.NotNil(t, summary)

	assert.Equal(t, tenantID, summary.TenantID)
	assert.True(t, summary.ProjectedCost >= 0)
}

// --- Subscription Lifecycle ---

func TestCreateSubscription(t *testing.T) {
	svc := NewService(nil, nil, nil)
	ctx := context.Background()

	tenantID := uuid.New()
	planID := uuid.New()

	sub, err := svc.CreateSubscriptionForTenant(ctx, tenantID, planID)
	require.NoError(t, err)
	require.NotNil(t, sub)

	assert.Equal(t, tenantID, sub.TenantID)
	assert.Equal(t, planID, sub.PlanID)
	assert.Equal(t, "active", sub.Status)
	assert.False(t, sub.CancelAtPeriodEnd)
	assert.True(t, sub.CurrentPeriodEnd.After(sub.CurrentPeriodStart))
}

func TestGetSubscription(t *testing.T) {
	svc := NewService(nil, nil, nil)
	ctx := context.Background()
	tenantID := uuid.New()

	sub, err := svc.GetSubscriptionForTenant(ctx, tenantID)
	require.NoError(t, err)
	require.NotNil(t, sub)
	assert.Equal(t, tenantID, sub.TenantID)
	assert.Equal(t, "active", sub.Status)
}

func TestChangeSubscription(t *testing.T) {
	svc := NewService(nil, nil, nil)
	ctx := context.Background()

	tenantID := uuid.New()
	newPlanID := uuid.New()

	sub, err := svc.ChangeSubscription(ctx, tenantID, newPlanID)
	require.NoError(t, err)
	require.NotNil(t, sub)
	assert.Equal(t, newPlanID, sub.PlanID)
}

func TestCancelSubscription(t *testing.T) {
	svc := NewService(nil, nil, nil)
	ctx := context.Background()
	tenantID := uuid.New()

	err := svc.CancelSubscriptionForTenant(ctx, tenantID)
	require.NoError(t, err)
}

// --- Mock Stripe Client ---

type mockStripeClient struct {
	createCustomerCalled     bool
	createSubscriptionCalled bool
	cancelSubscriptionCalled bool
	createUsageRecordCalled  bool
	getInvoicesCalled        bool
}

func (m *mockStripeClient) CreateCustomer(email, name string) (string, error) {
	m.createCustomerCalled = true
	return "cus_test123", nil
}

func (m *mockStripeClient) CreateSubscription(customerID, priceID string) (string, error) {
	m.createSubscriptionCalled = true
	return "sub_test123", nil
}

func (m *mockStripeClient) CancelSubscription(subID string) error {
	m.cancelSubscriptionCalled = true
	return nil
}

func (m *mockStripeClient) CreateUsageRecord(subItemID string, quantity int64, timestamp int64) error {
	m.createUsageRecordCalled = true
	return nil
}

func (m *mockStripeClient) GetInvoices(customerID string) ([]StripeInvoice, error) {
	m.getInvoicesCalled = true
	return []StripeInvoice{
		{ID: "inv_test", Amount: 2900, Status: "paid"},
	}, nil
}

func TestSubscriptionWithStripeClient(t *testing.T) {
	mock := &mockStripeClient{}
	svc := NewService(nil, nil, nil)
	svc.SetStripeClient(mock)
	svc.SetConfig(ServiceConfig{BillingEnabled: true})

	ctx := context.Background()
	tenantID := uuid.New()
	planID := uuid.New()

	sub, err := svc.CreateSubscriptionForTenant(ctx, tenantID, planID)
	require.NoError(t, err)
	require.NotNil(t, sub)
	assert.True(t, mock.createSubscriptionCalled)
	assert.Equal(t, "sub_test123", sub.StripeSubID)
}

// --- Dashboard ---

func TestGetBillingDashboard(t *testing.T) {
	svc := NewService(nil, nil, nil)
	ctx := context.Background()
	tenantID := uuid.New()

	dashboard, err := svc.GetBillingDashboard(ctx, tenantID)
	require.NoError(t, err)
	require.NotNil(t, dashboard)
	assert.NotNil(t, dashboard.Subscription)
	assert.NotNil(t, dashboard.Plan)
	assert.NotNil(t, dashboard.Usage)
}

// --- Pricing Plans ---

func TestListPricingPlans(t *testing.T) {
	svc := NewService(nil, nil, nil)
	ctx := context.Background()

	plans, err := svc.ListPricingPlans(ctx)
	require.NoError(t, err)
	assert.Len(t, plans, 4)

	names := make([]string, len(plans))
	for i, p := range plans {
		names[i] = p.Name
	}
	assert.Contains(t, names, "free")
	assert.Contains(t, names, "starter")
	assert.Contains(t, names, "pro")
	assert.Contains(t, names, "enterprise")
}

func TestCreatePricingPlan(t *testing.T) {
	svc := NewService(nil, nil, nil)
	ctx := context.Background()

	plan := &PricingPlan{
		Name:           "custom",
		DisplayName:    "Custom Plan",
		PricingModel:   "flat",
		BasePrice:      9900,
		IncludedEvents: 50000,
	}

	result, err := svc.CreatePricingPlan(ctx, plan)
	require.NoError(t, err)
	assert.False(t, result.ID == uuid.Nil)
	assert.True(t, result.Active)
	assert.Equal(t, "custom", result.Name)
}

// --- Usage Summary ---

func TestGetUsageSummaryForTenant(t *testing.T) {
	svc := NewService(nil, nil, nil)
	ctx := context.Background()
	tenantID := uuid.New()

	summary, err := svc.GetUsageSummaryForTenant(ctx, tenantID)
	require.NoError(t, err)
	require.NotNil(t, summary)

	assert.Equal(t, tenantID, summary.TenantID)
	assert.Equal(t, "free", summary.PlanName)
	assert.Equal(t, int64(1000), summary.EventsIncluded)
	assert.True(t, summary.PeriodEnd.After(summary.PeriodStart))
}

// --- Stripe Webhook ---

func TestHandleStripeWebhook(t *testing.T) {
	svc := NewService(nil, nil, nil)
	ctx := context.Background()

	events := []string{
		"invoice.paid",
		"invoice.payment_failed",
		"customer.subscription.updated",
		"customer.subscription.deleted",
		"unknown.event",
	}

	for _, event := range events {
		t.Run(event, func(t *testing.T) {
			err := svc.HandleStripeWebhook(ctx, event, nil)
			assert.NoError(t, err)
		})
	}
}
