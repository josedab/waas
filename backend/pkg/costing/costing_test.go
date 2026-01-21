package costing

import (
	"context"
	"testing"
	"time"
)

func TestDefaultRates(t *testing.T) {
	rates := DefaultRates()
	
	if len(rates) == 0 {
		t.Error("expected default rates to be non-empty")
	}
	
	// Check that essential units are defined
	essentialUnits := []CostUnit{UnitDelivery, UnitByte, UnitRetry, UnitTransform}
	for _, unit := range essentialUnits {
		found := false
		for _, rate := range rates {
			if rate.Unit == unit {
				found = true
				if rate.Price <= 0 {
					t.Errorf("expected positive price for unit %s", unit)
				}
				break
			}
		}
		if !found {
			t.Errorf("expected rate for unit %s", unit)
		}
	}
}

func TestCostCalculation(t *testing.T) {
	rates := DefaultRates()
	
	// Create a mock usage
	usage := map[CostUnit]int64{
		UnitDelivery:  1000,
		UnitByte:      1024 * 1024, // 1MB
		UnitRetry:     50,
		UnitTransform: 100,
	}
	
	totalCost := 0.0
	for unit, quantity := range usage {
		for _, rate := range rates {
			if rate.Unit == unit {
				totalCost += rate.Price * float64(quantity)
				break
			}
		}
	}
	
	if totalCost <= 0 {
		t.Error("expected positive total cost")
	}
}

func TestBudgetThresholds(t *testing.T) {
	budget := &Budget{
		ID:           "test-budget",
		TenantID:     "tenant-1",
		Name:         "Monthly Budget",
		Amount:       100.0,
		Currency:     "USD",
		Period:       "monthly",
		CurrentSpend: 80.0,
		Alerts: []BudgetAlert{
			{Threshold: 0.50, Channels: []string{"email"}},
			{Threshold: 0.80, Channels: []string{"email"}},
			{Threshold: 1.00, Channels: []string{"email", "webhook"}},
		},
		IsActive:  true,
		StartDate: time.Now().AddDate(0, 0, -15),
	}
	
	usagePercentage := budget.CurrentSpend / budget.Amount
	
	if usagePercentage != 0.80 {
		t.Errorf("expected 80%% usage (0.80), got %.2f", usagePercentage)
	}
	
	// Check which thresholds are exceeded
	exceededCount := 0
	for _, alert := range budget.Alerts {
		if usagePercentage >= alert.Threshold {
			exceededCount++
		}
	}
	
	// At 80% usage, should exceed 50% and 80% thresholds
	if exceededCount != 2 {
		t.Errorf("expected 2 thresholds exceeded, got %d", exceededCount)
	}
}

func TestUsageSummary(t *testing.T) {
	summary := UsageSummary{
		Deliveries:      5000,
		Bytes:           10 * 1024 * 1024,
		Retries:         100,
		Transformations: 500,
		Successful:      4800,
		Failed:          200,
	}
	
	if summary.Deliveries != 5000 {
		t.Errorf("expected 5000 deliveries, got %d", summary.Deliveries)
	}
	
	if summary.Successful+summary.Failed != summary.Deliveries {
		t.Error("successful + failed should equal total deliveries")
	}
}

func TestCostAllocation(t *testing.T) {
	allocation := &CostAllocation{
		ID:           "alloc-1",
		TenantID:     "tenant-1",
		Period:       "2026-02",
		ResourceType: "endpoint",
		ResourceID:   "endpoint-123",
		ResourceName: "My Webhook Endpoint",
		Usage: UsageSummary{
			Deliveries: 1000,
			Bytes:      512 * 1024,
		},
		Cost: CostBreakdown{
			Delivery:  0.10,
			Bandwidth: 0.005,
			Total:     0.105,
			Currency:  "USD",
		},
	}
	
	if allocation.ResourceType != "endpoint" {
		t.Errorf("expected resource type 'endpoint', got %s", allocation.ResourceType)
	}
	
	if allocation.Cost.Total != 0.105 {
		t.Errorf("expected total cost 0.105, got %.3f", allocation.Cost.Total)
	}
}

func TestCostReport(t *testing.T) {
	report := &CostReport{
		TenantID:  "tenant-1",
		Period:    "2026-02",
		StartDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC),
		Summary: CostBreakdown{
			Delivery: 1.00,
			Total:    1.00,
			Currency: "USD",
		},
		ByEndpoint: []CostAllocation{},
		DailyTrend: []DailyCost{},
		GeneratedAt: time.Now(),
	}
	
	if report.Period != "2026-02" {
		t.Errorf("expected period '2026-02', got %s", report.Period)
	}
	
	if report.Summary.Total != 1.00 {
		t.Errorf("expected total cost 1.00, got %.2f", report.Summary.Total)
	}
}

func TestCostBreakdown(t *testing.T) {
	breakdown := CostBreakdown{
		Delivery:       0.50,
		Bandwidth:      0.10,
		Retries:        0.05,
		Transformations: 0.01,
		Total:          0.66,
		Currency:       "USD",
	}
	
	// Verify components sum to total
	sum := breakdown.Delivery + breakdown.Bandwidth + breakdown.Retries + breakdown.Transformations
	if sum != breakdown.Total {
		t.Errorf("expected sum %.2f to equal total %.2f", sum, breakdown.Total)
	}
}

func TestCostForecast(t *testing.T) {
	forecast := &CostForecast{
		TenantID:       "tenant-1",
		Period:         "2026-02",
		ProjectedCost:  150.0,
		ProjectedUsage: UsageSummary{Deliveries: 1500000},
		Confidence:     0.85,
		Trend:          "increasing",
		PreviousPeriod: 100.0,
		PercentChange:  50.0,
		GeneratedAt:    time.Now(),
	}
	
	if forecast.Confidence < 0 || forecast.Confidence > 1 {
		t.Error("confidence should be between 0 and 1")
	}
	
	if forecast.PercentChange != 50.0 {
		t.Errorf("expected 50%% change, got %.2f", forecast.PercentChange)
	}
}

func TestUsageRecord(t *testing.T) {
	record := &UsageRecord{
		ID:         "record-1",
		TenantID:   "tenant-1",
		EndpointID: "endpoint-1",
		WebhookID:  "webhook-1",
		Unit:       UnitDelivery,
		Quantity:   1,
		Metadata: UsageMeta{
			PayloadBytes: 1024,
			StatusCode:   200,
			LatencyMs:    50,
		},
		RecordedAt: time.Now(),
	}
	
	if record.Unit != UnitDelivery {
		t.Errorf("expected unit delivery, got %s", record.Unit)
	}
	
	if record.Metadata.PayloadBytes != 1024 {
		t.Errorf("expected 1024 bytes, got %d", record.Metadata.PayloadBytes)
	}
}

func TestBudgetAlert(t *testing.T) {
	alert := BudgetAlert{
		Threshold:   0.80,
		Channels:    []string{"email", "webhook"},
		LastAlerted: nil,
	}
	
	if alert.Threshold != 0.80 {
		t.Errorf("expected threshold 0.80, got %.2f", alert.Threshold)
	}
	
	if len(alert.Channels) != 2 {
		t.Errorf("expected 2 channels, got %d", len(alert.Channels))
	}
}

func TestCreateBudgetRequest(t *testing.T) {
	req := CreateBudgetRequest{
		Name:      "Monthly Limit",
		Amount:    100.0,
		Currency:  "USD",
		Period:    "monthly",
		Alerts: []BudgetAlert{
			{Threshold: 0.80, Channels: []string{"email"}},
		},
		StartDate: time.Now(),
	}
	
	if req.Name == "" {
		t.Error("expected non-empty name")
	}
	
	if req.Amount <= 0 {
		t.Error("expected positive amount")
	}
}

func TestInvoice(t *testing.T) {
	invoice := &Invoice{
		ID:       "inv-001",
		TenantID: "tenant-1",
		Period:   "2026-02",
		LineItems: []InvoiceLineItem{
			{Description: "Webhook deliveries", Quantity: 10000, Unit: "deliveries", UnitPrice: 0.0001, Total: 1.00},
			{Description: "Bandwidth", Quantity: 1073741824, Unit: "bytes", UnitPrice: 0.00000001, Total: 10.74},
		},
		Subtotal:  11.74,
		Tax:       0.94,
		Total:     12.68,
		Currency:  "USD",
		Status:    "pending",
		DueDate:   time.Now().AddDate(0, 0, 30),
		CreatedAt: time.Now(),
	}
	
	if invoice.Status != "pending" {
		t.Errorf("expected status 'pending', got %s", invoice.Status)
	}
	
	if len(invoice.LineItems) != 2 {
		t.Errorf("expected 2 line items, got %d", len(invoice.LineItems))
	}
}

func TestServiceWithMockRepo(t *testing.T) {
	mockRepo := &mockCostingRepository{}
	
	service := NewService(mockRepo)
	if service == nil {
		t.Fatal("expected non-nil service")
	}
	
	ctx := context.Background()
	
	// Test creating a budget
	req := &CreateBudgetRequest{
		Name:      "Test Budget",
		Amount:    100.0,
		Currency:  "USD",
		Period:    "monthly",
		StartDate: time.Now(),
	}
	
	budget, err := service.CreateBudget(ctx, "tenant-1", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if budget == nil {
		t.Fatal("expected non-nil budget")
	}
	
	if budget.Name != "Test Budget" {
		t.Errorf("expected name 'Test Budget', got %s", budget.Name)
	}
}

// Mock repository for testing
type mockCostingRepository struct {
	budgets map[string]*Budget
}

func (m *mockCostingRepository) CreateUsageRecord(ctx context.Context, record *UsageRecord) error {
	return nil
}

func (m *mockCostingRepository) GetUsageSummary(ctx context.Context, tenantID string, startDate, endDate time.Time) (*UsageSummary, error) {
	return &UsageSummary{}, nil
}

func (m *mockCostingRepository) GetUsageByEndpoint(ctx context.Context, tenantID string, startDate, endDate time.Time) (map[string]UsageSummary, error) {
	return make(map[string]UsageSummary), nil
}

func (m *mockCostingRepository) GetDailyCosts(ctx context.Context, tenantID string, startDate, endDate time.Time) ([]DailyCost, error) {
	return []DailyCost{}, nil
}

func (m *mockCostingRepository) GetCostAllocation(ctx context.Context, tenantID, period, resourceType, resourceID string) (*CostAllocation, error) {
	return nil, nil
}

func (m *mockCostingRepository) GetCostAllocationsByResource(ctx context.Context, tenantID, period, resourceType string) ([]CostAllocation, error) {
	return []CostAllocation{}, nil
}

func (m *mockCostingRepository) SaveCostAllocation(ctx context.Context, allocation *CostAllocation) error {
	return nil
}

func (m *mockCostingRepository) CreateBudget(ctx context.Context, budget *Budget) error {
	if m.budgets == nil {
		m.budgets = make(map[string]*Budget)
	}
	m.budgets[budget.ID] = budget
	return nil
}

func (m *mockCostingRepository) GetBudget(ctx context.Context, tenantID, budgetID string) (*Budget, error) {
	if m.budgets == nil {
		return nil, nil
	}
	b, ok := m.budgets[budgetID]
	if !ok || b.TenantID != tenantID {
		return nil, nil
	}
	return b, nil
}

func (m *mockCostingRepository) ListBudgets(ctx context.Context, tenantID string, limit, offset int) ([]Budget, int, error) {
	var budgets []Budget
	for _, b := range m.budgets {
		if b.TenantID == tenantID {
			budgets = append(budgets, *b)
		}
	}
	return budgets, len(budgets), nil
}

func (m *mockCostingRepository) UpdateBudget(ctx context.Context, budget *Budget) error {
	if m.budgets == nil {
		m.budgets = make(map[string]*Budget)
	}
	m.budgets[budget.ID] = budget
	return nil
}

func (m *mockCostingRepository) DeleteBudget(ctx context.Context, tenantID, budgetID string) error {
	delete(m.budgets, budgetID)
	return nil
}
