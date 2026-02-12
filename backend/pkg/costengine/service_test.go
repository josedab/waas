package costengine

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockRepository is a mock implementation of the Repository interface.
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateCostModel(ctx context.Context, model *CostModel) error {
	args := m.Called(ctx, model)
	return args.Error(0)
}

func (m *MockRepository) GetCostModel(ctx context.Context, tenantID, modelID string) (*CostModel, error) {
	args := m.Called(ctx, tenantID, modelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CostModel), args.Error(1)
}

func (m *MockRepository) ListCostModels(ctx context.Context, tenantID string) ([]CostModel, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]CostModel), args.Error(1)
}

func (m *MockRepository) UpdateCostModel(ctx context.Context, model *CostModel) error {
	args := m.Called(ctx, model)
	return args.Error(0)
}

func (m *MockRepository) GetActiveCostModel(ctx context.Context, tenantID string) (*CostModel, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CostModel), args.Error(1)
}

func (m *MockRepository) RecordDeliveryCost(ctx context.Context, cost *DeliveryCost) error {
	args := m.Called(ctx, cost)
	return args.Error(0)
}

func (m *MockRepository) GetDeliveryCostsByPeriod(ctx context.Context, tenantID string, start, end time.Time) ([]DeliveryCost, error) {
	args := m.Called(ctx, tenantID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]DeliveryCost), args.Error(1)
}

func (m *MockRepository) GetTotalCostByPeriod(ctx context.Context, tenantID string, start, end time.Time) (float64, int64, error) {
	args := m.Called(ctx, tenantID, start, end)
	return args.Get(0).(float64), args.Get(1).(int64), args.Error(2)
}

func (m *MockRepository) GetCostByEndpoint(ctx context.Context, tenantID string, start, end time.Time) (map[string]float64, error) {
	args := m.Called(ctx, tenantID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]float64), args.Error(1)
}

func (m *MockRepository) GetCostByEventType(ctx context.Context, tenantID string, start, end time.Time) (map[string]float64, error) {
	args := m.Called(ctx, tenantID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]float64), args.Error(1)
}

func (m *MockRepository) GetCostByDay(ctx context.Context, tenantID string, start, end time.Time) ([]DailyCost, error) {
	args := m.Called(ctx, tenantID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]DailyCost), args.Error(1)
}

func (m *MockRepository) GetTopCostEndpoints(ctx context.Context, tenantID string, start, end time.Time, limit int) ([]EndpointCost, error) {
	args := m.Called(ctx, tenantID, start, end, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]EndpointCost), args.Error(1)
}

func (m *MockRepository) CreateBudget(ctx context.Context, budget *CostBudget) error {
	args := m.Called(ctx, budget)
	return args.Error(0)
}

func (m *MockRepository) GetBudget(ctx context.Context, tenantID, budgetID string) (*CostBudget, error) {
	args := m.Called(ctx, tenantID, budgetID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CostBudget), args.Error(1)
}

func (m *MockRepository) ListBudgets(ctx context.Context, tenantID string) ([]CostBudget, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]CostBudget), args.Error(1)
}

func (m *MockRepository) UpdateBudget(ctx context.Context, budget *CostBudget) error {
	args := m.Called(ctx, budget)
	return args.Error(0)
}

func (m *MockRepository) CreateAnomaly(ctx context.Context, anomaly *CostAnomaly) error {
	args := m.Called(ctx, anomaly)
	return args.Error(0)
}

func (m *MockRepository) ListAnomalies(ctx context.Context, tenantID string) ([]CostAnomaly, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]CostAnomaly), args.Error(1)
}

func (m *MockRepository) GetHistoricalAvgCost(ctx context.Context, tenantID string, days int) (float64, error) {
	args := m.Called(ctx, tenantID, days)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockRepository) GetCurrentSpend(ctx context.Context, tenantID string, periodStart time.Time) (float64, error) {
	args := m.Called(ctx, tenantID, periodStart)
	return args.Get(0).(float64), args.Error(1)
}

// --- Helper ---

func newTestModel() *CostModel {
	return &CostModel{
		ID:                     "model-1",
		TenantID:               "tenant-1",
		Name:                   "default",
		ComputeCostPerDelivery: 0.001,
		BandwidthCostPerKB:     0.0005,
		RetryCostMultiplier:    1.5,
		StorageCostPerGBDay:    0.023,
		Currency:               "USD",
		IsActive:               true,
	}
}

// --- RecordDeliveryCost Tests ---

func TestRecordDeliveryCost_CorrectCostCalculation(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	model := newTestModel()
	repo.On("GetActiveCostModel", ctx, "tenant-1").Return(model, nil)
	repo.On("RecordDeliveryCost", ctx, mock.AnythingOfType("*costengine.DeliveryCost")).Return(nil)

	req := &RecordDeliveryCostRequest{
		DeliveryID:       "del-1",
		EndpointID:       "ep-1",
		EventType:        "order.created",
		PayloadSizeBytes: 2048,
		RetryCount:       2,
	}

	cost, err := svc.RecordDeliveryCost(ctx, "tenant-1", req)
	require.NoError(t, err)

	expectedCompute := model.ComputeCostPerDelivery
	expectedBandwidth := model.BandwidthCostPerKB * (2048.0 / 1024.0)
	expectedRetry := expectedCompute * model.RetryCostMultiplier * 2
	expectedTotal := expectedCompute + expectedBandwidth + expectedRetry

	assert.Equal(t, math.Round(expectedCompute*1e6)/1e6, cost.ComputeCost)
	assert.Equal(t, math.Round(expectedBandwidth*1e6)/1e6, cost.BandwidthCost)
	assert.Equal(t, math.Round(expectedRetry*1e6)/1e6, cost.RetryCost)
	assert.Equal(t, math.Round(expectedTotal*1e6)/1e6, cost.TotalCost)
	repo.AssertExpectations(t)
}

func TestRecordDeliveryCost_ZeroPayloadSize(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	model := newTestModel()
	repo.On("GetActiveCostModel", ctx, "tenant-1").Return(model, nil)
	repo.On("RecordDeliveryCost", ctx, mock.AnythingOfType("*costengine.DeliveryCost")).Return(nil)

	req := &RecordDeliveryCostRequest{
		DeliveryID:       "del-1",
		EndpointID:       "ep-1",
		EventType:        "order.created",
		PayloadSizeBytes: 0,
		RetryCount:       1,
	}

	cost, err := svc.RecordDeliveryCost(ctx, "tenant-1", req)
	require.NoError(t, err)
	assert.Equal(t, 0.0, cost.BandwidthCost)
	repo.AssertExpectations(t)
}

func TestRecordDeliveryCost_ZeroRetryCount(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	model := newTestModel()
	repo.On("GetActiveCostModel", ctx, "tenant-1").Return(model, nil)
	repo.On("RecordDeliveryCost", ctx, mock.AnythingOfType("*costengine.DeliveryCost")).Return(nil)

	req := &RecordDeliveryCostRequest{
		DeliveryID:       "del-1",
		EndpointID:       "ep-1",
		EventType:        "order.created",
		PayloadSizeBytes: 1024,
		RetryCount:       0,
	}

	cost, err := svc.RecordDeliveryCost(ctx, "tenant-1", req)
	require.NoError(t, err)
	assert.Equal(t, 0.0, cost.RetryCost)
	repo.AssertExpectations(t)
}

func TestRecordDeliveryCost_ActiveModelNotFound(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("GetActiveCostModel", ctx, "tenant-1").Return(nil, errors.New("not found"))

	req := &RecordDeliveryCostRequest{
		DeliveryID:       "del-1",
		EndpointID:       "ep-1",
		EventType:        "order.created",
		PayloadSizeBytes: 1024,
		RetryCount:       0,
	}

	cost, err := svc.RecordDeliveryCost(ctx, "tenant-1", req)
	assert.Nil(t, cost)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get active cost model")
	repo.AssertExpectations(t)
}

func TestRecordDeliveryCost_CostsRoundedTo6Decimals(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	model := &CostModel{
		ID:                     "model-1",
		TenantID:               "tenant-1",
		ComputeCostPerDelivery: 0.0000011,
		BandwidthCostPerKB:     0.0000003,
		RetryCostMultiplier:    1.3333333,
		IsActive:               true,
	}
	repo.On("GetActiveCostModel", ctx, "tenant-1").Return(model, nil)
	repo.On("RecordDeliveryCost", ctx, mock.AnythingOfType("*costengine.DeliveryCost")).Return(nil)

	req := &RecordDeliveryCostRequest{
		DeliveryID:       "del-1",
		EndpointID:       "ep-1",
		EventType:        "test",
		PayloadSizeBytes: 3333,
		RetryCount:       3,
	}

	cost, err := svc.RecordDeliveryCost(ctx, "tenant-1", req)
	require.NoError(t, err)

	assertRounded6 := func(val float64) {
		rounded := math.Round(val*1e6) / 1e6
		assert.Equal(t, rounded, val)
	}
	assertRounded6(cost.ComputeCost)
	assertRounded6(cost.BandwidthCost)
	assertRounded6(cost.RetryCost)
	assertRounded6(cost.TotalCost)
	repo.AssertExpectations(t)
}

// --- GenerateReport Tests ---

func TestGenerateReport_AggregatesAllDataSources(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

	costByEndpoint := map[string]float64{"ep-1": 1.5, "ep-2": 2.5}
	costByEventType := map[string]float64{"order.created": 3.0}
	costByDay := []DailyCost{{Date: "2024-01-01", Cost: 4.0, Deliveries: 100}}
	topEndpoints := []EndpointCost{{EndpointID: "ep-1", TotalCost: 1.5, DeliveryCount: 50}}

	repo.On("GetTotalCostByPeriod", ctx, "tenant-1", start, end).Return(10.123, int64(500), nil)
	repo.On("GetCostByEndpoint", ctx, "tenant-1", start, end).Return(costByEndpoint, nil)
	repo.On("GetCostByEventType", ctx, "tenant-1", start, end).Return(costByEventType, nil)
	repo.On("GetCostByDay", ctx, "tenant-1", start, end).Return(costByDay, nil)
	repo.On("GetTopCostEndpoints", ctx, "tenant-1", start, end, 10).Return(topEndpoints, nil)

	report, err := svc.GenerateReport(ctx, "tenant-1", &GenerateReportRequest{PeriodStart: start, PeriodEnd: end})
	require.NoError(t, err)

	assert.Equal(t, "tenant-1", report.TenantID)
	assert.Equal(t, math.Round(10.123*100)/100, report.TotalCost)
	assert.Equal(t, int64(500), report.DeliveryCount)
	assert.Equal(t, math.Round((10.123/500)*1e6)/1e6, report.AvgCostPerDelivery)
	assert.Equal(t, costByEndpoint, report.CostByEndpoint)
	assert.Equal(t, costByEventType, report.CostByEventType)
	assert.Equal(t, costByDay, report.CostByDay)
	assert.Equal(t, topEndpoints, report.TopCostEndpoints)
	repo.AssertExpectations(t)
}

func TestGenerateReport_ZeroDeliveries_AvgCostZero(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	repo.On("GetTotalCostByPeriod", ctx, "tenant-1", start, end).Return(0.0, int64(0), nil)
	repo.On("GetCostByEndpoint", ctx, "tenant-1", start, end).Return(map[string]float64{}, nil)
	repo.On("GetCostByEventType", ctx, "tenant-1", start, end).Return(map[string]float64{}, nil)
	repo.On("GetCostByDay", ctx, "tenant-1", start, end).Return([]DailyCost{}, nil)
	repo.On("GetTopCostEndpoints", ctx, "tenant-1", start, end, 10).Return([]EndpointCost{}, nil)

	report, err := svc.GenerateReport(ctx, "tenant-1", &GenerateReportRequest{PeriodStart: start, PeriodEnd: end})
	require.NoError(t, err)

	assert.Equal(t, 0.0, report.TotalCost)
	assert.Equal(t, int64(0), report.DeliveryCount)
	assert.Equal(t, 0.0, report.AvgCostPerDelivery)
	repo.AssertExpectations(t)
}

func TestGenerateReport_TotalCostRoundedTo2Decimals(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	repo.On("GetTotalCostByPeriod", ctx, "tenant-1", start, end).Return(12.3456789, int64(100), nil)
	repo.On("GetCostByEndpoint", ctx, "tenant-1", start, end).Return(map[string]float64{}, nil)
	repo.On("GetCostByEventType", ctx, "tenant-1", start, end).Return(map[string]float64{}, nil)
	repo.On("GetCostByDay", ctx, "tenant-1", start, end).Return([]DailyCost{}, nil)
	repo.On("GetTopCostEndpoints", ctx, "tenant-1", start, end, 10).Return([]EndpointCost{}, nil)

	report, err := svc.GenerateReport(ctx, "tenant-1", &GenerateReportRequest{PeriodStart: start, PeriodEnd: end})
	require.NoError(t, err)

	assert.Equal(t, 12.35, report.TotalCost)
	repo.AssertExpectations(t)
}

func TestGenerateReport_AvgCostRoundedTo6Decimals(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	repo.On("GetTotalCostByPeriod", ctx, "tenant-1", start, end).Return(1.0, int64(3), nil)
	repo.On("GetCostByEndpoint", ctx, "tenant-1", start, end).Return(map[string]float64{}, nil)
	repo.On("GetCostByEventType", ctx, "tenant-1", start, end).Return(map[string]float64{}, nil)
	repo.On("GetCostByDay", ctx, "tenant-1", start, end).Return([]DailyCost{}, nil)
	repo.On("GetTopCostEndpoints", ctx, "tenant-1", start, end, 10).Return([]EndpointCost{}, nil)

	report, err := svc.GenerateReport(ctx, "tenant-1", &GenerateReportRequest{PeriodStart: start, PeriodEnd: end})
	require.NoError(t, err)

	expected := math.Round((1.0/3.0)*1e6) / 1e6
	assert.Equal(t, expected, report.AvgCostPerDelivery)
	repo.AssertExpectations(t)
}

func TestGenerateReport_RepoErrorPropagated_TotalCost(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	repo.On("GetTotalCostByPeriod", ctx, "tenant-1", start, end).Return(0.0, int64(0), errors.New("db error"))

	report, err := svc.GenerateReport(ctx, "tenant-1", &GenerateReportRequest{PeriodStart: start, PeriodEnd: end})
	assert.Nil(t, report)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get total cost")
	repo.AssertExpectations(t)
}

func TestGenerateReport_RepoErrorPropagated_CostByEndpoint(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	repo.On("GetTotalCostByPeriod", ctx, "tenant-1", start, end).Return(10.0, int64(100), nil)
	repo.On("GetCostByEndpoint", ctx, "tenant-1", start, end).Return(nil, errors.New("db error"))

	report, err := svc.GenerateReport(ctx, "tenant-1", &GenerateReportRequest{PeriodStart: start, PeriodEnd: end})
	assert.Nil(t, report)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get cost by endpoint")
	repo.AssertExpectations(t)
}

func TestGenerateReport_RepoErrorPropagated_CostByEventType(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	repo.On("GetTotalCostByPeriod", ctx, "tenant-1", start, end).Return(10.0, int64(100), nil)
	repo.On("GetCostByEndpoint", ctx, "tenant-1", start, end).Return(map[string]float64{}, nil)
	repo.On("GetCostByEventType", ctx, "tenant-1", start, end).Return(nil, errors.New("db error"))

	report, err := svc.GenerateReport(ctx, "tenant-1", &GenerateReportRequest{PeriodStart: start, PeriodEnd: end})
	assert.Nil(t, report)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get cost by event type")
	repo.AssertExpectations(t)
}

func TestGenerateReport_RepoErrorPropagated_CostByDay(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	repo.On("GetTotalCostByPeriod", ctx, "tenant-1", start, end).Return(10.0, int64(100), nil)
	repo.On("GetCostByEndpoint", ctx, "tenant-1", start, end).Return(map[string]float64{}, nil)
	repo.On("GetCostByEventType", ctx, "tenant-1", start, end).Return(map[string]float64{}, nil)
	repo.On("GetCostByDay", ctx, "tenant-1", start, end).Return(nil, errors.New("db error"))

	report, err := svc.GenerateReport(ctx, "tenant-1", &GenerateReportRequest{PeriodStart: start, PeriodEnd: end})
	assert.Nil(t, report)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get cost by day")
	repo.AssertExpectations(t)
}

func TestGenerateReport_RepoErrorPropagated_TopEndpoints(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	repo.On("GetTotalCostByPeriod", ctx, "tenant-1", start, end).Return(10.0, int64(100), nil)
	repo.On("GetCostByEndpoint", ctx, "tenant-1", start, end).Return(map[string]float64{}, nil)
	repo.On("GetCostByEventType", ctx, "tenant-1", start, end).Return(map[string]float64{}, nil)
	repo.On("GetCostByDay", ctx, "tenant-1", start, end).Return([]DailyCost{}, nil)
	repo.On("GetTopCostEndpoints", ctx, "tenant-1", start, end, 10).Return(nil, errors.New("db error"))

	report, err := svc.GenerateReport(ctx, "tenant-1", &GenerateReportRequest{PeriodStart: start, PeriodEnd: end})
	assert.Nil(t, report)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get top cost endpoints")
	repo.AssertExpectations(t)
}

// --- CheckBudgetAlerts Tests ---

func TestCheckBudgetAlerts_SpendAboveThreshold(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	budgets := []CostBudget{
		{ID: "b-1", TenantID: "tenant-1", Name: "prod", MonthlyLimit: 100, AlertThresholdPct: 80, IsActive: true},
	}
	repo.On("ListBudgets", ctx, "tenant-1").Return(budgets, nil)
	repo.On("GetCurrentSpend", ctx, "tenant-1", mock.AnythingOfType("time.Time")).Return(85.0, nil)

	alerts, err := svc.CheckBudgetAlerts(ctx, "tenant-1")
	require.NoError(t, err)
	require.Len(t, alerts, 1)
	assert.Equal(t, "b-1", alerts[0].ID)
	assert.Equal(t, 85.0, alerts[0].CurrentSpend)
	repo.AssertExpectations(t)
}

func TestCheckBudgetAlerts_SpendBelowThreshold(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	budgets := []CostBudget{
		{ID: "b-1", TenantID: "tenant-1", Name: "prod", MonthlyLimit: 100, AlertThresholdPct: 80, IsActive: true},
	}
	repo.On("ListBudgets", ctx, "tenant-1").Return(budgets, nil)
	repo.On("GetCurrentSpend", ctx, "tenant-1", mock.AnythingOfType("time.Time")).Return(50.0, nil)

	alerts, err := svc.CheckBudgetAlerts(ctx, "tenant-1")
	require.NoError(t, err)
	assert.Empty(t, alerts)
	repo.AssertExpectations(t)
}

func TestCheckBudgetAlerts_InactiveBudgetsSkipped(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	budgets := []CostBudget{
		{ID: "b-1", TenantID: "tenant-1", MonthlyLimit: 100, AlertThresholdPct: 80, IsActive: false},
		{ID: "b-2", TenantID: "tenant-1", MonthlyLimit: 100, AlertThresholdPct: 80, IsActive: true},
	}
	repo.On("ListBudgets", ctx, "tenant-1").Return(budgets, nil)
	repo.On("GetCurrentSpend", ctx, "tenant-1", mock.AnythingOfType("time.Time")).Return(90.0, nil)

	alerts, err := svc.CheckBudgetAlerts(ctx, "tenant-1")
	require.NoError(t, err)
	require.Len(t, alerts, 1)
	assert.Equal(t, "b-2", alerts[0].ID)
	repo.AssertExpectations(t)
}

func TestCheckBudgetAlerts_MultipleBudgets(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	budgets := []CostBudget{
		{ID: "b-1", TenantID: "tenant-1", MonthlyLimit: 100, AlertThresholdPct: 80, IsActive: true},
		{ID: "b-2", TenantID: "tenant-1", MonthlyLimit: 200, AlertThresholdPct: 50, IsActive: true},
	}
	repo.On("ListBudgets", ctx, "tenant-1").Return(budgets, nil)
	repo.On("GetCurrentSpend", ctx, "tenant-1", mock.AnythingOfType("time.Time")).Return(150.0, nil)

	alerts, err := svc.CheckBudgetAlerts(ctx, "tenant-1")
	require.NoError(t, err)
	// b-1: 150/100*100 = 150% >= 80% → alert
	// b-2: 150/200*100 = 75% >= 50% → alert
	assert.Len(t, alerts, 2)
	repo.AssertExpectations(t)
}

// --- DetectAnomalies Tests ---

func TestDetectAnomalies_SpikeAnomaly(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("GetHistoricalAvgCost", ctx, "tenant-1", 30).Return(10.0, nil)
	repo.On("GetTotalCostByPeriod", ctx, "tenant-1", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(16.0, int64(100), nil)
	repo.On("CreateAnomaly", ctx, mock.AnythingOfType("*costengine.CostAnomaly")).Return(nil)
	repo.On("ListAnomalies", ctx, "tenant-1").Return([]CostAnomaly{}, nil)

	anomalies, err := svc.DetectAnomalies(ctx, "tenant-1")
	require.NoError(t, err)
	require.Len(t, anomalies, 1)
	assert.Equal(t, AnomalyTypeSpike, anomalies[0].AnomalyType)
	assert.Equal(t, AnomalyStatusActive, anomalies[0].Status)
	repo.AssertExpectations(t)
}

func TestDetectAnomalies_SustainedIncrease(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	// >200% deviation: (35-10)/10*100 = 250%
	repo.On("GetHistoricalAvgCost", ctx, "tenant-1", 30).Return(10.0, nil)
	repo.On("GetTotalCostByPeriod", ctx, "tenant-1", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(35.0, int64(100), nil)
	repo.On("CreateAnomaly", ctx, mock.AnythingOfType("*costengine.CostAnomaly")).Return(nil)
	repo.On("ListAnomalies", ctx, "tenant-1").Return([]CostAnomaly{}, nil)

	anomalies, err := svc.DetectAnomalies(ctx, "tenant-1")
	require.NoError(t, err)
	require.Len(t, anomalies, 1)
	assert.Equal(t, AnomalyTypeSustainedIncrease, anomalies[0].AnomalyType)
	repo.AssertExpectations(t)
}

func TestDetectAnomalies_NoAnomaly_BelowThreshold(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	// 40% deviation ≤ 50%
	repo.On("GetHistoricalAvgCost", ctx, "tenant-1", 30).Return(10.0, nil)
	repo.On("GetTotalCostByPeriod", ctx, "tenant-1", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(14.0, int64(100), nil)
	repo.On("ListAnomalies", ctx, "tenant-1").Return([]CostAnomaly{}, nil)

	anomalies, err := svc.DetectAnomalies(ctx, "tenant-1")
	require.NoError(t, err)
	assert.Empty(t, anomalies)
	repo.AssertExpectations(t)
}

func TestDetectAnomalies_HistoricalAvgZero_NoAnomalies(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("GetHistoricalAvgCost", ctx, "tenant-1", 30).Return(0.0, nil)
	repo.On("GetTotalCostByPeriod", ctx, "tenant-1", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(100.0, int64(50), nil)
	repo.On("ListAnomalies", ctx, "tenant-1").Return([]CostAnomaly{}, nil)

	anomalies, err := svc.DetectAnomalies(ctx, "tenant-1")
	require.NoError(t, err)
	assert.Empty(t, anomalies)
	repo.AssertExpectations(t)
}

func TestDetectAnomalies_ExistingAnomaliesAppended(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	existing := []CostAnomaly{
		{ID: "existing-1", TenantID: "tenant-1", AnomalyType: AnomalyTypeSpike, Status: AnomalyStatusActive},
	}
	// 60% deviation → spike
	repo.On("GetHistoricalAvgCost", ctx, "tenant-1", 30).Return(10.0, nil)
	repo.On("GetTotalCostByPeriod", ctx, "tenant-1", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(16.0, int64(100), nil)
	repo.On("CreateAnomaly", ctx, mock.AnythingOfType("*costengine.CostAnomaly")).Return(nil)
	repo.On("ListAnomalies", ctx, "tenant-1").Return(existing, nil)

	anomalies, err := svc.DetectAnomalies(ctx, "tenant-1")
	require.NoError(t, err)
	// 1 new + 1 existing
	assert.Len(t, anomalies, 2)
	assert.Equal(t, "existing-1", anomalies[1].ID)
	repo.AssertExpectations(t)
}

// --- CreateCostModel Tests ---

func TestCreateCostModel_SetsIsActiveTrue(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("CreateCostModel", ctx, mock.AnythingOfType("*costengine.CostModel")).Return(nil)

	req := &CreateCostModelRequest{
		Name:                   "test-model",
		ComputeCostPerDelivery: 0.001,
		BandwidthCostPerKB:     0.0005,
		RetryCostMultiplier:    1.5,
		StorageCostPerGBDay:    0.023,
		Currency:               "USD",
	}

	model, err := svc.CreateCostModel(ctx, "tenant-1", req)
	require.NoError(t, err)
	assert.True(t, model.IsActive)
	assert.Equal(t, "tenant-1", model.TenantID)
	assert.Equal(t, "test-model", model.Name)
	assert.NotEmpty(t, model.ID)
	assert.Equal(t, 0.001, model.ComputeCostPerDelivery)
	repo.AssertExpectations(t)
}

// --- UpdateCostModel Tests ---

func TestUpdateCostModel_UpdatesFields(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	existing := &CostModel{
		ID:                     "model-1",
		TenantID:               "tenant-1",
		Name:                   "old-name",
		ComputeCostPerDelivery: 0.001,
		BandwidthCostPerKB:     0.0005,
		RetryCostMultiplier:    1.5,
		Currency:               "USD",
		IsActive:               true,
	}
	repo.On("GetCostModel", ctx, "tenant-1", "model-1").Return(existing, nil)
	repo.On("UpdateCostModel", ctx, mock.AnythingOfType("*costengine.CostModel")).Return(nil)

	req := &CreateCostModelRequest{
		Name:                   "new-name",
		ComputeCostPerDelivery: 0.002,
		BandwidthCostPerKB:     0.001,
		RetryCostMultiplier:    2.0,
		StorageCostPerGBDay:    0.05,
		Currency:               "EUR",
	}

	model, err := svc.UpdateCostModel(ctx, "tenant-1", "model-1", req)
	require.NoError(t, err)
	assert.Equal(t, "new-name", model.Name)
	assert.Equal(t, 0.002, model.ComputeCostPerDelivery)
	assert.Equal(t, 0.001, model.BandwidthCostPerKB)
	assert.Equal(t, 2.0, model.RetryCostMultiplier)
	assert.Equal(t, 0.05, model.StorageCostPerGBDay)
	assert.Equal(t, "EUR", model.Currency)
	repo.AssertExpectations(t)
}

// --- CreateBudget Tests ---

func TestCreateBudget_SetsIsActiveAndZeroSpend(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("CreateBudget", ctx, mock.AnythingOfType("*costengine.CostBudget")).Return(nil)

	req := &CreateBudgetRequest{
		Name:              "monthly-budget",
		MonthlyLimit:      500.0,
		AlertThresholdPct: 80,
		Period:            "monthly",
	}

	budget, err := svc.CreateBudget(ctx, "tenant-1", req)
	require.NoError(t, err)
	assert.True(t, budget.IsActive)
	assert.Equal(t, 0.0, budget.CurrentSpend)
	assert.Equal(t, "tenant-1", budget.TenantID)
	assert.Equal(t, "monthly-budget", budget.Name)
	assert.Equal(t, 500.0, budget.MonthlyLimit)
	assert.NotEmpty(t, budget.ID)
	repo.AssertExpectations(t)
}

// --- UpdateBudget Tests ---

func TestUpdateBudget_UpdatesFields(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	existing := &CostBudget{
		ID:                "b-1",
		TenantID:          "tenant-1",
		Name:              "old-budget",
		MonthlyLimit:      100,
		AlertThresholdPct: 80,
		Period:            "monthly",
		IsActive:          true,
	}
	repo.On("GetBudget", ctx, "tenant-1", "b-1").Return(existing, nil)
	repo.On("UpdateBudget", ctx, mock.AnythingOfType("*costengine.CostBudget")).Return(nil)

	req := &CreateBudgetRequest{
		Name:              "new-budget",
		MonthlyLimit:      200,
		AlertThresholdPct: 90,
		Period:            "weekly",
	}

	budget, err := svc.UpdateBudget(ctx, "tenant-1", "b-1", req)
	require.NoError(t, err)
	assert.Equal(t, "new-budget", budget.Name)
	assert.Equal(t, 200.0, budget.MonthlyLimit)
	assert.Equal(t, 90.0, budget.AlertThresholdPct)
	assert.Equal(t, "weekly", budget.Period)
	repo.AssertExpectations(t)
}

// --- GetCurrentSpend Tests ---

func TestGetCurrentSpend_RoundedTo2Decimals(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("GetCurrentSpend", ctx, "tenant-1", mock.AnythingOfType("time.Time")).Return(123.456789, nil)

	spend, err := svc.GetCurrentSpend(ctx, "tenant-1")
	require.NoError(t, err)
	assert.Equal(t, 123.46, spend)
	repo.AssertExpectations(t)
}

func TestGetCurrentSpend_RepoError(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("GetCurrentSpend", ctx, "tenant-1", mock.AnythingOfType("time.Time")).Return(0.0, errors.New("db error"))

	spend, err := svc.GetCurrentSpend(ctx, "tenant-1")
	assert.Equal(t, 0.0, spend)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get current spend")
	repo.AssertExpectations(t)
}

// --- Additional Coverage Tests ---

func TestRecordDeliveryCost_VeryLargeCost(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	model := &CostModel{
		ID:                     "model-1",
		TenantID:               "tenant-1",
		ComputeCostPerDelivery: 1000000,
		BandwidthCostPerKB:     0.0005,
		RetryCostMultiplier:    1.5,
		IsActive:               true,
	}
	repo.On("GetActiveCostModel", ctx, "tenant-1").Return(model, nil)
	repo.On("RecordDeliveryCost", ctx, mock.AnythingOfType("*costengine.DeliveryCost")).Return(nil)

	req := &RecordDeliveryCostRequest{
		DeliveryID:       "del-1",
		EndpointID:       "ep-1",
		EventType:        "order.created",
		PayloadSizeBytes: 1073741824, // 1 GB
		RetryCount:       100,
	}

	cost, err := svc.RecordDeliveryCost(ctx, "tenant-1", req)
	require.NoError(t, err)

	assert.Greater(t, cost.TotalCost, 0.0)
	assert.False(t, math.IsInf(cost.TotalCost, 0))
	assert.False(t, math.IsNaN(cost.TotalCost))
	// Verify rounded to 6 decimal places
	assert.Equal(t, math.Round(cost.TotalCost*1e6)/1e6, cost.TotalCost)
	repo.AssertExpectations(t)
}

func TestRecordDeliveryCost_NegativePayloadSize(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	model := newTestModel()
	repo.On("GetActiveCostModel", ctx, "tenant-1").Return(model, nil)
	repo.On("RecordDeliveryCost", ctx, mock.AnythingOfType("*costengine.DeliveryCost")).Return(nil)

	req := &RecordDeliveryCostRequest{
		DeliveryID:       "del-1",
		EndpointID:       "ep-1",
		EventType:        "order.created",
		PayloadSizeBytes: -1,
		RetryCount:       0,
	}

	cost, err := svc.RecordDeliveryCost(ctx, "tenant-1", req)
	require.NoError(t, err)
	// Negative payload yields non-positive bandwidth cost (rounds to -0.0 at 6 decimals)
	expectedBandwidth := math.Round(model.BandwidthCostPerKB*(-1.0/1024.0)*1e6) / 1e6
	assert.Equal(t, expectedBandwidth, cost.BandwidthCost)
	repo.AssertExpectations(t)
}

func TestRecordDeliveryCost_FloatingPointPrecision(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	model := &CostModel{
		ID:                     "model-1",
		TenantID:               "tenant-1",
		ComputeCostPerDelivery: 0.1,
		BandwidthCostPerKB:     0.2,
		RetryCostMultiplier:    1.5,
		IsActive:               true,
	}
	repo.On("GetActiveCostModel", ctx, "tenant-1").Return(model, nil)
	repo.On("RecordDeliveryCost", ctx, mock.AnythingOfType("*costengine.DeliveryCost")).Return(nil)

	req := &RecordDeliveryCostRequest{
		DeliveryID:       "del-1",
		EndpointID:       "ep-1",
		EventType:        "order.created",
		PayloadSizeBytes: 1024,
		RetryCount:       0,
	}

	cost, err := svc.RecordDeliveryCost(ctx, "tenant-1", req)
	require.NoError(t, err)

	// 0.1 + 0.2*(1024/1024) = 0.1 + 0.2 = 0.3 (classic floating point edge case)
	expectedCompute := math.Round(0.1*1e6) / 1e6
	expectedBandwidth := math.Round(0.2*1e6) / 1e6
	expectedTotal := math.Round(0.3*1e6) / 1e6

	assert.Equal(t, expectedCompute, cost.ComputeCost)
	assert.Equal(t, expectedBandwidth, cost.BandwidthCost)
	assert.Equal(t, expectedTotal, cost.TotalCost)
	repo.AssertExpectations(t)
}

func TestDetectAnomalies_HistoricalAvgZeroWithRecentCost(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	// Historical avg is 0, recent cost is 100 → would cause division by zero
	repo.On("GetHistoricalAvgCost", ctx, "tenant-1", 30).Return(0.0, nil)
	repo.On("GetTotalCostByPeriod", ctx, "tenant-1", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(100.0, int64(50), nil)
	repo.On("ListAnomalies", ctx, "tenant-1").Return([]CostAnomaly{}, nil)

	anomalies, err := svc.DetectAnomalies(ctx, "tenant-1")
	require.NoError(t, err)
	// historicalAvg <= 0 guard prevents infinite/NaN deviation, so no anomaly created
	assert.Empty(t, anomalies)
	repo.AssertExpectations(t)
}

func TestCheckBudgetAlerts_MultipleBudgetsDifferentThresholds(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	budgets := []CostBudget{
		{ID: "b-1", TenantID: "tenant-1", Name: "low-threshold", MonthlyLimit: 100, AlertThresholdPct: 50, IsActive: true},
		{ID: "b-2", TenantID: "tenant-1", Name: "mid-threshold", MonthlyLimit: 100, AlertThresholdPct: 80, IsActive: true},
		{ID: "b-3", TenantID: "tenant-1", Name: "full-threshold", MonthlyLimit: 100, AlertThresholdPct: 100, IsActive: true},
	}
	repo.On("ListBudgets", ctx, "tenant-1").Return(budgets, nil)
	// Current spend = 85% of limit
	repo.On("GetCurrentSpend", ctx, "tenant-1", mock.AnythingOfType("time.Time")).Return(85.0, nil)

	alerts, err := svc.CheckBudgetAlerts(ctx, "tenant-1")
	require.NoError(t, err)
	// b-1: 85/100*100 = 85% >= 50% → alert
	// b-2: 85/100*100 = 85% >= 80% → alert
	// b-3: 85/100*100 = 85% < 100% → no alert
	require.Len(t, alerts, 2)
	assert.Equal(t, "b-1", alerts[0].ID)
	assert.Equal(t, "b-2", alerts[1].ID)
	repo.AssertExpectations(t)
}

func TestGenerateReport_AllErrorsPropagated(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	repo.On("GetTotalCostByPeriod", ctx, "tenant-1", start, end).Return(10.0, int64(100), nil)
	repo.On("GetCostByEndpoint", ctx, "tenant-1", start, end).Return(nil, errors.New("endpoint db error"))

	report, err := svc.GenerateReport(ctx, "tenant-1", &GenerateReportRequest{PeriodStart: start, PeriodEnd: end})
	assert.Nil(t, report)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get cost by endpoint")
	repo.AssertExpectations(t)
}
