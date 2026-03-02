package billing

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Mock Repository ---

type mockRepository struct {
	mock.Mock
}

var _ Repository = (*mockRepository)(nil)

func (m *mockRepository) RecordUsage(ctx context.Context, record *CostUsageRecord) error {
	args := m.Called(ctx, record)
	return args.Error(0)
}

func (m *mockRepository) GetUsageSummary(ctx context.Context, tenantID, period string) (*CostUsageSummary, error) {
	args := m.Called(ctx, tenantID, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CostUsageSummary), args.Error(1)
}

func (m *mockRepository) GetUsageByResource(ctx context.Context, tenantID, resourceType, period string) ([]CostUsageRecord, error) {
	args := m.Called(ctx, tenantID, resourceType, period)
	return args.Get(0).([]CostUsageRecord), args.Error(1)
}

func (m *mockRepository) GetSpendTracker(ctx context.Context, tenantID string, period BillingPeriod) (*SpendTracker, error) {
	args := m.Called(ctx, tenantID, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SpendTracker), args.Error(1)
}

func (m *mockRepository) UpdateSpendTracker(ctx context.Context, tracker *SpendTracker) error {
	args := m.Called(ctx, tracker)
	return args.Error(0)
}

func (m *mockRepository) GetCurrentSpend(ctx context.Context, tenantID string) (float64, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).(float64), args.Error(1)
}

func (m *mockRepository) SaveBudget(ctx context.Context, budget *BudgetConfig) error {
	args := m.Called(ctx, budget)
	return args.Error(0)
}

func (m *mockRepository) GetBudget(ctx context.Context, tenantID, budgetID string) (*BudgetConfig, error) {
	args := m.Called(ctx, tenantID, budgetID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BudgetConfig), args.Error(1)
}

func (m *mockRepository) ListBudgets(ctx context.Context, tenantID string) ([]BudgetConfig, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]BudgetConfig), args.Error(1)
}

func (m *mockRepository) DeleteBudget(ctx context.Context, tenantID, budgetID string) error {
	args := m.Called(ctx, tenantID, budgetID)
	return args.Error(0)
}

func (m *mockRepository) SaveAlert(ctx context.Context, alert *BillingAlert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *mockRepository) GetAlert(ctx context.Context, alertID string) (*BillingAlert, error) {
	args := m.Called(ctx, alertID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BillingAlert), args.Error(1)
}

func (m *mockRepository) ListAlerts(ctx context.Context, tenantID string, status *AlertStatus) ([]BillingAlert, error) {
	args := m.Called(ctx, tenantID, status)
	return args.Get(0).([]BillingAlert), args.Error(1)
}

func (m *mockRepository) UpdateAlertStatus(ctx context.Context, alertID string, status AlertStatus, ackedBy string) error {
	args := m.Called(ctx, alertID, status, ackedBy)
	return args.Error(0)
}

func (m *mockRepository) SaveOptimization(ctx context.Context, opt *CostOptimization) error {
	args := m.Called(ctx, opt)
	return args.Error(0)
}

func (m *mockRepository) ListOptimizations(ctx context.Context, tenantID string) ([]CostOptimization, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]CostOptimization), args.Error(1)
}

func (m *mockRepository) UpdateOptimizationStatus(ctx context.Context, optID string, status OptimizationStatus) error {
	args := m.Called(ctx, optID, status)
	return args.Error(0)
}

func (m *mockRepository) SaveInvoice(ctx context.Context, invoice *CostInvoice) error {
	args := m.Called(ctx, invoice)
	return args.Error(0)
}

func (m *mockRepository) GetInvoice(ctx context.Context, tenantID, invoiceID string) (*CostInvoice, error) {
	args := m.Called(ctx, tenantID, invoiceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CostInvoice), args.Error(1)
}

func (m *mockRepository) ListInvoices(ctx context.Context, tenantID string) ([]CostInvoice, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]CostInvoice), args.Error(1)
}

func (m *mockRepository) SaveAlertConfig(ctx context.Context, config *AlertConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *mockRepository) GetAlertConfig(ctx context.Context, tenantID string) (*AlertConfig, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*AlertConfig), args.Error(1)
}

// --- Mock Notifier ---

type mockNotifier struct {
	mock.Mock
}

func (m *mockNotifier) Send(ctx context.Context, alert *BillingAlert, channels []AlertChannel, recipients []string) error {
	args := m.Called(ctx, alert, channels, recipients)
	return args.Error(0)
}

// --- Mock StripeClient (testify-based) ---

type testifyStripeClient struct {
	mock.Mock
}

func (m *testifyStripeClient) CreateCustomer(email, name string) (string, error) {
	args := m.Called(email, name)
	return args.String(0), args.Error(1)
}

func (m *testifyStripeClient) CreateSubscription(customerID, priceID string) (string, error) {
	args := m.Called(customerID, priceID)
	return args.String(0), args.Error(1)
}

func (m *testifyStripeClient) CancelSubscription(subID string) error {
	args := m.Called(subID)
	return args.Error(0)
}

func (m *testifyStripeClient) CreateUsageRecord(subItemID string, quantity int64, timestamp int64) error {
	args := m.Called(subItemID, quantity, timestamp)
	return args.Error(0)
}

func (m *testifyStripeClient) GetInvoices(customerID string) ([]StripeInvoice, error) {
	args := m.Called(customerID)
	return args.Get(0).([]StripeInvoice), args.Error(1)
}

// --- Helper ---

func newTestService(repo *mockRepository, notifier *mockNotifier) *Service {
	return NewService(repo, nil, notifier)
}

// =====================
// RecordWebhookUsage
// =====================

func TestRecordWebhookUsage_AllThreeRecords(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	// Expect 3 RecordUsage calls (requests, retries, data transfer)
	repo.On("RecordUsage", ctx, mock.MatchedBy(func(r *CostUsageRecord) bool {
		return r.ResourceType == "webhook_requests" && r.Quantity == 100
	})).Return(nil).Once()
	repo.On("RecordUsage", ctx, mock.MatchedBy(func(r *CostUsageRecord) bool {
		return r.ResourceType == "retry_attempts" && r.Quantity == 5
	})).Return(nil).Once()
	repo.On("RecordUsage", ctx, mock.MatchedBy(func(r *CostUsageRecord) bool {
		return r.ResourceType == "data_transfer_bytes" && r.Quantity == 1024
	})).Return(nil).Once()

	// checkBudgetAlerts runs in goroutine — set up expectations
	repo.On("ListBudgets", mock.Anything, "tenant-1").Return([]BudgetConfig{}, nil).Maybe()
	repo.On("GetCurrentSpend", mock.Anything, "tenant-1").Return(0.0, nil).Maybe()

	err := svc.RecordWebhookUsage(ctx, "tenant-1", "wh-1", 100, 5, 1024)
	require.NoError(t, err)

	// Give goroutine time to finish
	time.Sleep(50 * time.Millisecond)
	repo.AssertNumberOfCalls(t, "RecordUsage", 3)
}

func TestRecordWebhookUsage_ZeroBytesTransferred(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	repo.On("RecordUsage", ctx, mock.MatchedBy(func(r *CostUsageRecord) bool {
		return r.ResourceType == "webhook_requests"
	})).Return(nil).Once()

	repo.On("ListBudgets", mock.Anything, "tenant-1").Return([]BudgetConfig{}, nil).Maybe()
	repo.On("GetCurrentSpend", mock.Anything, "tenant-1").Return(0.0, nil).Maybe()

	err := svc.RecordWebhookUsage(ctx, "tenant-1", "wh-1", 10, 0, 0)
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
	// Only 1 record: requests (retries=0, bytes=0 skip)
	repo.AssertNumberOfCalls(t, "RecordUsage", 1)
}

func TestRecordWebhookUsage_PartialRecordingOnError(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	// First call succeeds, second fails
	repo.On("RecordUsage", ctx, mock.MatchedBy(func(r *CostUsageRecord) bool {
		return r.ResourceType == "webhook_requests"
	})).Return(nil).Once()
	repo.On("RecordUsage", ctx, mock.MatchedBy(func(r *CostUsageRecord) bool {
		return r.ResourceType == "retry_attempts"
	})).Return(errors.New("db error")).Once()

	err := svc.RecordWebhookUsage(ctx, "tenant-1", "wh-1", 10, 5, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "record retry usage")
}

func TestRecordWebhookUsage_EmptyTenantID(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	// Even with empty tenant ID, the function proceeds (no validation in source)
	repo.On("RecordUsage", ctx, mock.MatchedBy(func(r *CostUsageRecord) bool {
		return r.TenantID == "" && r.ResourceType == "webhook_requests"
	})).Return(nil).Once()

	repo.On("ListBudgets", mock.Anything, "").Return([]BudgetConfig{}, nil).Maybe()
	repo.On("GetCurrentSpend", mock.Anything, "").Return(0.0, nil).Maybe()

	err := svc.RecordWebhookUsage(ctx, "", "wh-1", 1, 0, 0)
	require.NoError(t, err)
}

// =====================
// ForecastSpend
// =====================

func TestForecastSpend_ConfidenceTiers(t *testing.T) {
	tests := []struct {
		name       string
		byDay      []DailyUsage
		wantConf   float64
	}{
		{
			name:     "high confidence with 7+ days",
			byDay:    make([]DailyUsage, 7),
			wantConf: 0.85,
		},
		{
			name:     "medium confidence with 3-6 days",
			byDay:    make([]DailyUsage, 4),
			wantConf: 0.7,
		},
		{
			name:     "low confidence with <3 days",
			byDay:    make([]DailyUsage, 1),
			wantConf: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockRepository)
			svc := newTestService(repo, nil)
			ctx := context.Background()

			repo.On("GetUsageSummary", ctx, "tenant-1", mock.Anything).Return(&CostUsageSummary{
				TotalCost: 100.0,
				Currency:  "USD",
				ByDay:     tt.byDay,
			}, nil)
			repo.On("ListBudgets", ctx, "tenant-1").Return([]BudgetConfig{}, nil)

			forecast, err := svc.ForecastSpend(ctx, "tenant-1", 30)
			require.NoError(t, err)
			assert.Equal(t, tt.wantConf, forecast.Confidence)
		})
	}
}

func TestForecastSpend_TrendDirection(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	// Increasing trend: later costs are >110% of earlier
	repo.On("GetUsageSummary", ctx, "tenant-1", mock.Anything).Return(&CostUsageSummary{
		TotalCost: 100.0,
		Currency:  "USD",
		ByDay: []DailyUsage{
			{Date: "2026-02-01", Cost: 1.0},
			{Date: "2026-02-02", Cost: 2.0},
			{Date: "2026-02-03", Cost: 3.0},
			{Date: "2026-02-04", Cost: 5.0},
			{Date: "2026-02-05", Cost: 8.0},
		},
	}, nil)
	repo.On("ListBudgets", ctx, "tenant-1").Return([]BudgetConfig{}, nil)

	forecast, err := svc.ForecastSpend(ctx, "tenant-1", 30)
	require.NoError(t, err)
	assert.Equal(t, "increasing", forecast.TrendDirection)
}

func TestForecastSpend_RepoError(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	repo.On("GetUsageSummary", ctx, "tenant-1", mock.Anything).Return(nil, errors.New("db error"))

	_, err := svc.ForecastSpend(ctx, "tenant-1", 30)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get usage summary")
}

// =====================
// CreateBudget
// =====================

func TestCreateBudget_Success(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	repo.On("SaveBudget", ctx, mock.AnythingOfType("*billing.BudgetConfig")).Return(nil)

	budget, err := svc.CreateBudget(ctx, "tenant-1", &CreateBudgetRequest{
		Name:   "Monthly",
		Amount: 1000,
		Period: PeriodMonthly,
	})

	require.NoError(t, err)
	assert.Equal(t, "tenant-1", budget.TenantID)
	assert.Equal(t, "Monthly", budget.Name)
	assert.Equal(t, 1000.0, budget.Amount)
	assert.True(t, budget.Enabled)
	assert.NotEmpty(t, budget.ID)
}

func TestCreateBudget_DefaultCurrency(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	repo.On("SaveBudget", ctx, mock.MatchedBy(func(b *BudgetConfig) bool {
		return b.Currency == "USD"
	})).Return(nil)

	budget, err := svc.CreateBudget(ctx, "tenant-1", &CreateBudgetRequest{
		Name:   "Test",
		Amount: 500,
		Period: PeriodMonthly,
		// Currency omitted — should default to "USD"
	})

	require.NoError(t, err)
	assert.Equal(t, "USD", budget.Currency)
}

func TestCreateBudget_ExplicitCurrency(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	repo.On("SaveBudget", ctx, mock.AnythingOfType("*billing.BudgetConfig")).Return(nil)

	budget, err := svc.CreateBudget(ctx, "tenant-1", &CreateBudgetRequest{
		Name:     "Test",
		Amount:   500,
		Period:   PeriodMonthly,
		Currency: "EUR",
	})

	require.NoError(t, err)
	assert.Equal(t, "EUR", budget.Currency)
}

func TestCreateBudget_RepoError(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	repo.On("SaveBudget", ctx, mock.AnythingOfType("*billing.BudgetConfig")).Return(errors.New("db error"))

	_, err := svc.CreateBudget(ctx, "tenant-1", &CreateBudgetRequest{
		Name:   "Test",
		Amount: 500,
		Period: PeriodMonthly,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "save budget")
}

// =====================
// UpdateBudget
// =====================

func TestUpdateBudget_PartialUpdate(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	existing := &BudgetConfig{
		ID:       "budget-1",
		TenantID: "tenant-1",
		Name:     "Original",
		Amount:   1000,
		Currency: "USD",
		Period:   PeriodMonthly,
		Enabled:  true,
	}

	repo.On("GetBudget", ctx, "tenant-1", "budget-1").Return(existing, nil)
	repo.On("SaveBudget", ctx, mock.AnythingOfType("*billing.BudgetConfig")).Return(nil)

	// Only update name
	updated, err := svc.UpdateBudget(ctx, "tenant-1", "budget-1", &UpdateBudgetRequest{
		Name: "Updated",
	})

	require.NoError(t, err)
	assert.Equal(t, "Updated", updated.Name)
	assert.Equal(t, 1000.0, updated.Amount)  // Unchanged
	assert.Equal(t, "USD", updated.Currency)  // Unchanged
}

func TestUpdateBudget_GetBudgetError(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	repo.On("GetBudget", ctx, "tenant-1", "budget-1").Return(nil, errors.New("not found"))

	_, err := svc.UpdateBudget(ctx, "tenant-1", "budget-1", &UpdateBudgetRequest{
		Name: "Updated",
	})

	require.Error(t, err)
}

func TestUpdateBudget_AutoPauseToggle(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	existing := &BudgetConfig{
		ID:        "budget-1",
		TenantID:  "tenant-1",
		AutoPause: false,
	}

	repo.On("GetBudget", ctx, "tenant-1", "budget-1").Return(existing, nil)
	repo.On("SaveBudget", ctx, mock.AnythingOfType("*billing.BudgetConfig")).Return(nil)

	autoPause := true
	updated, err := svc.UpdateBudget(ctx, "tenant-1", "budget-1", &UpdateBudgetRequest{
		AutoPause: &autoPause,
	})

	require.NoError(t, err)
	assert.True(t, updated.AutoPause)
}

// =====================
// AnalyzeOptimizations
// =====================

func TestAnalyzeOptimizations_EmptyUsage(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	repo.On("GetUsageSummary", ctx, "tenant-1", mock.Anything).Return(&CostUsageSummary{
		TotalRequests: 0,
		TotalBytes:    0,
		TotalFailed:   0,
	}, nil)

	opts, err := svc.AnalyzeOptimizations(ctx, "tenant-1")
	require.NoError(t, err)
	assert.Empty(t, opts)
}

func TestAnalyzeOptimizations_HighRetryRate(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	repo.On("GetUsageSummary", ctx, "tenant-1", mock.Anything).Return(&CostUsageSummary{
		TotalRequests: 1000,
		TotalFailed:   200, // 20% failure rate
		TotalBytes:    0,
	}, nil)
	repo.On("SaveOptimization", ctx, mock.AnythingOfType("*billing.CostOptimization")).Return(nil)

	opts, err := svc.AnalyzeOptimizations(ctx, "tenant-1")
	require.NoError(t, err)
	require.NotEmpty(t, opts)

	found := false
	for _, o := range opts {
		if o.Type == OptRetryReduction {
			found = true
			break
		}
	}
	assert.True(t, found, "should find retry reduction optimization")
}

func TestAnalyzeOptimizations_LargePayloads(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	repo.On("GetUsageSummary", ctx, "tenant-1", mock.Anything).Return(&CostUsageSummary{
		TotalRequests: 100,
		TotalFailed:   0,
		TotalBytes:    100 * 200 * 1024, // 200KB average
	}, nil)
	repo.On("SaveOptimization", ctx, mock.AnythingOfType("*billing.CostOptimization")).Return(nil)

	opts, err := svc.AnalyzeOptimizations(ctx, "tenant-1")
	require.NoError(t, err)

	found := false
	for _, o := range opts {
		if o.Type == OptPayloadCompression {
			found = true
			break
		}
	}
	assert.True(t, found, "should find compression optimization")
}

func TestAnalyzeOptimizations_BatchDelivery(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	repo.On("GetUsageSummary", ctx, "tenant-1", mock.Anything).Return(&CostUsageSummary{
		TotalRequests: 5000,
		TotalFailed:   0,
		TotalBytes:    0,
	}, nil)
	repo.On("SaveOptimization", ctx, mock.AnythingOfType("*billing.CostOptimization")).Return(nil)

	opts, err := svc.AnalyzeOptimizations(ctx, "tenant-1")
	require.NoError(t, err)

	found := false
	for _, o := range opts {
		if o.Type == OptBatchDelivery {
			found = true
			break
		}
	}
	assert.True(t, found, "should find batch delivery optimization")
}

func TestAnalyzeOptimizations_RepoError(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	repo.On("GetUsageSummary", ctx, "tenant-1", mock.Anything).Return(nil, errors.New("db error"))

	_, err := svc.AnalyzeOptimizations(ctx, "tenant-1")
	require.Error(t, err)
}

// =====================
// checkBudgetAlerts
// =====================

func TestCheckBudgetAlerts_ThresholdCrossing(t *testing.T) {
	repo := new(mockRepository)
	notifier := new(mockNotifier)
	svc := NewService(repo, nil, notifier)
	ctx := context.Background()

	repo.On("ListBudgets", ctx, "tenant-1").Return([]BudgetConfig{
		{
			ID:       "b1",
			TenantID: "tenant-1",
			Name:     "Monthly",
			Amount:   100,
			Enabled:  true,
			Alerts: []AlertThreshold{
				{Percentage: 80, Channels: []AlertChannel{ChannelEmail}},
			},
		},
	}, nil)
	repo.On("GetCurrentSpend", ctx, "tenant-1").Return(85.0, nil) // 85% utilization

	repo.On("SaveAlert", ctx, mock.AnythingOfType("*billing.BillingAlert")).Return(nil)
	repo.On("GetAlertConfig", ctx, "tenant-1").Return(&AlertConfig{
		Recipients: AlertRecipients{Emails: []string{"test@example.com"}},
	}, nil)
	notifier.On("Send", ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	svc.checkBudgetAlerts(ctx, "tenant-1")

	repo.AssertCalled(t, "SaveAlert", ctx, mock.AnythingOfType("*billing.BillingAlert"))
	notifier.AssertCalled(t, "Send", ctx, mock.Anything, mock.Anything, mock.Anything)
}

func TestCheckBudgetAlerts_DisabledBudget(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	repo.On("ListBudgets", ctx, "tenant-1").Return([]BudgetConfig{
		{
			ID:      "b1",
			Amount:  100,
			Enabled: false, // disabled
			Alerts: []AlertThreshold{
				{Percentage: 50, Channels: []AlertChannel{ChannelEmail}},
			},
		},
	}, nil)
	repo.On("GetCurrentSpend", ctx, "tenant-1").Return(80.0, nil)

	svc.checkBudgetAlerts(ctx, "tenant-1")

	repo.AssertNotCalled(t, "SaveAlert", mock.Anything, mock.Anything)
}

func TestCheckBudgetAlerts_AutoPauseTrigger(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	repo.On("ListBudgets", ctx, "tenant-1").Return([]BudgetConfig{
		{
			ID:        "b1",
			TenantID:  "tenant-1",
			Name:      "Monthly",
			Amount:    100,
			Enabled:   true,
			AutoPause: true,
			Alerts:    []AlertThreshold{},
		},
	}, nil)
	repo.On("GetCurrentSpend", ctx, "tenant-1").Return(120.0, nil) // Over budget

	repo.On("SaveAlert", ctx, mock.MatchedBy(func(a *BillingAlert) bool {
		return a.Type == AlertBudgetExceeded && a.Severity == SeverityCritical
	})).Return(nil)

	svc.checkBudgetAlerts(ctx, "tenant-1")

	repo.AssertCalled(t, "SaveAlert", ctx, mock.AnythingOfType("*billing.BillingAlert"))
}

// =====================
// NewService
// =====================

func TestNewService_DefaultPricing(t *testing.T) {
	svc := NewService(nil, nil, nil)
	assert.Equal(t, DefaultPricing.WebhookRequestCost, svc.pricing.WebhookRequestCost)
	assert.Equal(t, "USD", svc.pricing.Currency)
}

func TestNewService_CustomPricing(t *testing.T) {
	custom := &PricingConfig{
		WebhookRequestCost: 0.001,
		Currency:           "EUR",
	}
	svc := NewService(nil, custom, nil)
	assert.Equal(t, 0.001, svc.pricing.WebhookRequestCost)
	assert.Equal(t, "EUR", svc.pricing.Currency)
}

// =====================
// HandleStripeWebhook
// =====================

func TestHandleStripeWebhook_KnownEvents(t *testing.T) {
	svc := NewService(nil, nil, nil)
	ctx := context.Background()

	events := []string{"invoice.paid", "invoice.payment_failed", "customer.subscription.updated", "customer.subscription.deleted"}
	for _, evt := range events {
		err := svc.HandleStripeWebhook(ctx, evt, map[string]interface{}{})
		assert.NoError(t, err, "event: %s", evt)
	}
}

func TestHandleStripeWebhook_UnknownEvent(t *testing.T) {
	svc := NewService(nil, nil, nil)
	ctx := context.Background()

	err := svc.HandleStripeWebhook(ctx, "unknown.event", nil)
	assert.NoError(t, err)
}

// =====================
// Goroutine Error Paths
// =====================

func TestRecordWebhookUsage_ListBudgetsFails_UsageStillRecorded(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	// Usage recording succeeds
	repo.On("RecordUsage", ctx, mock.MatchedBy(func(r *CostUsageRecord) bool {
		return r.ResourceType == "webhook_requests" && r.Quantity == 10
	})).Return(nil).Once()

	// Budget check fails in goroutine — should not affect usage recording
	repo.On("ListBudgets", mock.Anything, "tenant-1").Return([]BudgetConfig(nil), errors.New("db error")).Maybe()
	repo.On("GetCurrentSpend", mock.Anything, "tenant-1").Return(0.0, nil).Maybe()

	err := svc.RecordWebhookUsage(ctx, "tenant-1", "wh-1", 10, 0, 0)
	require.NoError(t, err, "usage should be recorded even if budget check fails")

	time.Sleep(100 * time.Millisecond)
	repo.AssertCalled(t, "RecordUsage", ctx, mock.Anything)
}

func TestRecordWebhookUsage_GetCurrentSpendFails(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	repo.On("RecordUsage", ctx, mock.Anything).Return(nil).Once()

	// ListBudgets succeeds but GetCurrentSpend fails
	repo.On("ListBudgets", mock.Anything, "tenant-1").Return([]BudgetConfig{
		{ID: "b1", Amount: 100, Enabled: true},
	}, nil).Maybe()
	repo.On("GetCurrentSpend", mock.Anything, "tenant-1").Return(0.0, errors.New("redis error")).Maybe()

	err := svc.RecordWebhookUsage(ctx, "tenant-1", "wh-1", 5, 0, 0)
	require.NoError(t, err, "usage should be recorded even if spend check fails")

	time.Sleep(100 * time.Millisecond)
}

func TestCheckBudgetAlerts_NilNotifier_NoPanic(t *testing.T) {
	repo := new(mockRepository)
	// notifier is nil
	svc := NewService(repo, nil, nil)
	ctx := context.Background()

	repo.On("ListBudgets", ctx, "tenant-1").Return([]BudgetConfig{
		{
			ID:      "b1",
			Amount:  100,
			Enabled: true,
			Alerts: []AlertThreshold{
				{Percentage: 50, Channels: []AlertChannel{ChannelEmail}},
			},
		},
	}, nil)
	repo.On("GetCurrentSpend", ctx, "tenant-1").Return(80.0, nil)
	repo.On("SaveAlert", ctx, mock.Anything).Return(nil)
	repo.On("GetAlertConfig", ctx, "tenant-1").Return((*AlertConfig)(nil), errors.New("no config"))

	// Should not panic even with nil notifier
	assert.NotPanics(t, func() {
		svc.checkBudgetAlerts(ctx, "tenant-1")
	})
}

func TestRecordWebhookUsage_NegativeQuantity(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	// Negative requests — the service records if quantity > 0, so negative is skipped
	repo.On("ListBudgets", mock.Anything, "tenant-1").Return([]BudgetConfig{}, nil).Maybe()
	repo.On("GetCurrentSpend", mock.Anything, "tenant-1").Return(0.0, nil).Maybe()

	err := svc.RecordWebhookUsage(ctx, "tenant-1", "wh-1", -5, 0, 0)
	require.NoError(t, err)

	// RecordUsage should NOT be called since requests <= 0
	repo.AssertNotCalled(t, "RecordUsage", mock.Anything, mock.Anything)
}

func TestRecordWebhookUsage_Concurrent(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	repo.On("RecordUsage", ctx, mock.Anything).Return(nil)
	repo.On("ListBudgets", mock.Anything, mock.Anything).Return([]BudgetConfig{}, nil).Maybe()
	repo.On("GetCurrentSpend", mock.Anything, mock.Anything).Return(0.0, nil).Maybe()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			tenantID := fmt.Sprintf("tenant-%d", i)
			_ = svc.RecordWebhookUsage(ctx, tenantID, "wh-1", 1, 0, 0)
		}(i)
	}
	wg.Wait()

	time.Sleep(100 * time.Millisecond)
	// Each goroutine records 1 usage call
	repo.AssertNumberOfCalls(t, "RecordUsage", 10)
}
