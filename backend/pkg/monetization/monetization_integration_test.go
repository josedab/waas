package monetization_test

import (
	"testing"

	"github.com/josedab/waas/pkg/monetization"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanCRUD(t *testing.T) {
	t.Skip("Integration test - requires database")

	service := monetization.NewService(nil, monetization.DefaultServiceConfig())
	require.NotNil(t, service)
}

func TestServiceConfigDefaults(t *testing.T) {
	config := monetization.DefaultServiceConfig()

	assert.NotNil(t, config)
	assert.Equal(t, "usd", config.DefaultCurrency)
	assert.Equal(t, 14, config.DefaultTrialDays)
	assert.Equal(t, 30, config.InvoiceDueDays)
}

func TestPricingModelConstants(t *testing.T) {
	models := []monetization.PricingModel{
		monetization.PricingUsageBased,
		monetization.PricingTiered,
		monetization.PricingFlatRate,
		monetization.PricingHybrid,
	}

	for _, model := range models {
		assert.NotEmpty(t, string(model))
	}
}

func TestBillingPeriodConstants(t *testing.T) {
	periods := []monetization.BillingPeriod{
		monetization.BillingMonthly,
		monetization.BillingAnnual,
		monetization.BillingWeekly,
	}

	for _, period := range periods {
		assert.NotEmpty(t, string(period))
	}
}

func TestSubscriptionStatusConstants(t *testing.T) {
	statuses := []monetization.SubscriptionStatus{
		monetization.SubscriptionActive,
		monetization.SubscriptionPaused,
		monetization.SubscriptionCancelled,
		monetization.SubscriptionTrialing,
		monetization.SubscriptionPastDue,
	}

	for _, status := range statuses {
		assert.NotEmpty(t, string(status))
	}
}

func TestPlanStructure(t *testing.T) {
	plan := &monetization.Plan{
		ID:               "plan-123",
		TenantID:         "tenant-456",
		Name:             "Professional",
		Description:      "Professional tier plan",
		PricingModel:     monetization.PricingTiered,
		BillingPeriod:    monetization.BillingMonthly,
		BasePrice:        9900,
		IncludedWebhooks: 1000,
	}

	assert.Equal(t, "plan-123", plan.ID)
	assert.Equal(t, "Professional", plan.Name)
	assert.Equal(t, int64(9900), plan.BasePrice)
}

func TestAPIKeyStructure(t *testing.T) {
	key := &monetization.APIKey{
		ID:             "key-123",
		TenantID:       "tenant-456",
		CustomerID:     "customer-789",
		Name:           "Production API Key",
	}

	assert.Equal(t, "key-123", key.ID)
	assert.Equal(t, "Production API Key", key.Name)
}

func TestInvoiceItemStructure(t *testing.T) {
	item := monetization.InvoiceItem{
		Description: "Webhook deliveries",
		Quantity:    1000,
		UnitPrice:   10,
		Amount:      10000,
	}

	assert.Equal(t, "Webhook deliveries", item.Description)
	assert.Equal(t, int64(1000), item.Quantity)
	assert.Equal(t, int64(10000), item.Amount)
}
