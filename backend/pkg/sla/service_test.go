package sla

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockRepository implements Repository for testing.
type mockRepository struct {
	mock.Mock
}

func (m *mockRepository) CreateTarget(ctx context.Context, target *Target) error {
	return m.Called(ctx, target).Error(0)
}

func (m *mockRepository) GetTarget(ctx context.Context, tenantID, targetID string) (*Target, error) {
	args := m.Called(ctx, tenantID, targetID)
	if t := args.Get(0); t != nil {
		return t.(*Target), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockRepository) ListTargets(ctx context.Context, tenantID string) ([]Target, error) {
	args := m.Called(ctx, tenantID)
	if t := args.Get(0); t != nil {
		return t.([]Target), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockRepository) UpdateTarget(ctx context.Context, target *Target) error {
	return m.Called(ctx, target).Error(0)
}

func (m *mockRepository) DeleteTarget(ctx context.Context, tenantID, targetID string) error {
	return m.Called(ctx, tenantID, targetID).Error(0)
}

func (m *mockRepository) GetDeliveryStats(ctx context.Context, tenantID, endpointID string, windowMinutes int) (int, int, int, float64, int, int, error) {
	args := m.Called(ctx, tenantID, endpointID, windowMinutes)
	return args.Int(0), args.Int(1), args.Int(2), args.Get(3).(float64), args.Int(4), args.Int(5), args.Error(6)
}

func (m *mockRepository) CreateBreach(ctx context.Context, breach *Breach) error {
	return m.Called(ctx, breach).Error(0)
}

func (m *mockRepository) ListActiveBreaches(ctx context.Context, tenantID string) ([]Breach, error) {
	args := m.Called(ctx, tenantID)
	if b := args.Get(0); b != nil {
		return b.([]Breach), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockRepository) ListBreachHistory(ctx context.Context, tenantID string, limit, offset int) ([]Breach, error) {
	args := m.Called(ctx, tenantID, limit, offset)
	if b := args.Get(0); b != nil {
		return b.([]Breach), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockRepository) ResolveBreach(ctx context.Context, tenantID, breachID string) error {
	return m.Called(ctx, tenantID, breachID).Error(0)
}

func (m *mockRepository) GetAlertConfig(ctx context.Context, tenantID, targetID string) (*AlertConfig, error) {
	args := m.Called(ctx, tenantID, targetID)
	if c := args.Get(0); c != nil {
		return c.(*AlertConfig), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockRepository) UpsertAlertConfig(ctx context.Context, config *AlertConfig) error {
	return m.Called(ctx, config).Error(0)
}

// ---------------------------------------------------------------------------
// CreateTarget
// ---------------------------------------------------------------------------

func TestCreateTarget_Valid(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("CreateTarget", ctx, mock.AnythingOfType("*sla.Target")).Return(nil)

	req := &CreateTargetRequest{
		Name:            "test-target",
		DeliveryRatePct: 99.9,
		LatencyP50Ms:    200,
		LatencyP99Ms:    500,
		WindowMinutes:   60,
		EndpointID:      "ep-1",
	}

	target, err := svc.CreateTarget(ctx, "tenant-1", req)
	require.NoError(t, err)
	assert.True(t, target.IsActive)
	assert.Equal(t, "tenant-1", target.TenantID)
	assert.Equal(t, "test-target", target.Name)
	assert.Equal(t, 99.9, target.DeliveryRatePct)
	assert.NotEmpty(t, target.ID)
	repo.AssertExpectations(t)
}

func TestCreateTarget_DeliveryRateNegative(t *testing.T) {
	svc := NewService(new(mockRepository))
	_, err := svc.CreateTarget(context.Background(), "t", &CreateTargetRequest{DeliveryRatePct: -1, WindowMinutes: 10})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delivery_rate_pct")
}

func TestCreateTarget_DeliveryRateAbove100(t *testing.T) {
	svc := NewService(new(mockRepository))
	_, err := svc.CreateTarget(context.Background(), "t", &CreateTargetRequest{DeliveryRatePct: 100.1, WindowMinutes: 10})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delivery_rate_pct")
}

func TestCreateTarget_WindowMinutesZero(t *testing.T) {
	svc := NewService(new(mockRepository))
	_, err := svc.CreateTarget(context.Background(), "t", &CreateTargetRequest{DeliveryRatePct: 99, WindowMinutes: 0})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "window_minutes")
}

func TestCreateTarget_RepoError(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("CreateTarget", ctx, mock.Anything).Return(errors.New("db down"))

	_, err := svc.CreateTarget(ctx, "t", &CreateTargetRequest{DeliveryRatePct: 99, WindowMinutes: 10})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db down")
}

// ---------------------------------------------------------------------------
// GetComplianceStatus
// ---------------------------------------------------------------------------

func TestGetComplianceStatus_AllGood(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	target := &Target{
		ID: "t1", TenantID: "ten", EndpointID: "ep",
		DeliveryRatePct: 99.0, LatencyP50Ms: 200, LatencyP99Ms: 500, WindowMinutes: 60,
	}

	repo.On("GetDeliveryStats", ctx, "ten", "ep", 60).
		Return(1000, 999, 1, 50.0, 100, 300, nil)

	status, err := svc.GetComplianceStatus(ctx, "ten", target)
	require.NoError(t, err)
	assert.True(t, status.IsCompliant)
	assert.Equal(t, 99.9, status.CurrentRate) // 999/1000*100 = 99.9
	assert.Equal(t, 1000, status.TotalDeliveries)
	assert.Equal(t, 999, status.SuccessCount)
	assert.Equal(t, 1, status.FailureCount)
}

func TestGetComplianceStatus_CurrentRateRounding(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	target := &Target{
		ID: "t1", TenantID: "ten", EndpointID: "ep",
		DeliveryRatePct: 90.0, WindowMinutes: 10,
	}

	// 333/1000 = 33.3%
	repo.On("GetDeliveryStats", ctx, "ten", "ep", 10).
		Return(1000, 333, 667, 0.0, 0, 0, nil)

	status, err := svc.GetComplianceStatus(ctx, "ten", target)
	require.NoError(t, err)
	assert.Equal(t, 33.3, status.CurrentRate)
}

func TestGetComplianceStatus_RateBelowTarget(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	target := &Target{
		ID: "t1", TenantID: "ten", EndpointID: "ep",
		DeliveryRatePct: 99.0, WindowMinutes: 60,
	}

	repo.On("GetDeliveryStats", ctx, "ten", "ep", 60).
		Return(100, 90, 10, 50.0, 50, 100, nil)

	status, err := svc.GetComplianceStatus(ctx, "ten", target)
	require.NoError(t, err)
	assert.False(t, status.IsCompliant)
}

func TestGetComplianceStatus_P50AboveTarget(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	target := &Target{
		ID: "t1", TenantID: "ten", EndpointID: "ep",
		DeliveryRatePct: 90.0, LatencyP50Ms: 100, WindowMinutes: 60,
	}

	repo.On("GetDeliveryStats", ctx, "ten", "ep", 60).
		Return(100, 100, 0, 50.0, 150, 50, nil)

	status, err := svc.GetComplianceStatus(ctx, "ten", target)
	require.NoError(t, err)
	assert.False(t, status.IsCompliant)
}

func TestGetComplianceStatus_P99AboveTarget(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	target := &Target{
		ID: "t1", TenantID: "ten", EndpointID: "ep",
		DeliveryRatePct: 90.0, LatencyP99Ms: 500, WindowMinutes: 60,
	}

	repo.On("GetDeliveryStats", ctx, "ten", "ep", 60).
		Return(100, 100, 0, 50.0, 50, 600, nil)

	status, err := svc.GetComplianceStatus(ctx, "ten", target)
	require.NoError(t, err)
	assert.False(t, status.IsCompliant)
}

func TestGetComplianceStatus_ZeroDeliveries(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	target := &Target{
		ID: "t1", TenantID: "ten", EndpointID: "ep",
		DeliveryRatePct: 99.0, WindowMinutes: 60,
	}

	repo.On("GetDeliveryStats", ctx, "ten", "ep", 60).
		Return(0, 0, 0, 0.0, 0, 0, nil)

	status, err := svc.GetComplianceStatus(ctx, "ten", target)
	require.NoError(t, err)
	assert.Equal(t, float64(0), status.CurrentRate)
	assert.False(t, status.IsCompliant)
}

func TestGetComplianceStatus_P50ZeroSkipsCheck(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	target := &Target{
		ID: "t1", TenantID: "ten", EndpointID: "ep",
		DeliveryRatePct: 90.0, LatencyP50Ms: 0, LatencyP99Ms: 0, WindowMinutes: 60,
	}

	// p50=9999 but target is 0 so check should be skipped
	repo.On("GetDeliveryStats", ctx, "ten", "ep", 60).
		Return(100, 100, 0, 50.0, 9999, 9999, nil)

	status, err := svc.GetComplianceStatus(ctx, "ten", target)
	require.NoError(t, err)
	assert.True(t, status.IsCompliant)
}

func TestGetComplianceStatus_P99ZeroSkipsCheck(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	target := &Target{
		ID: "t1", TenantID: "ten", EndpointID: "ep",
		DeliveryRatePct: 90.0, LatencyP50Ms: 100, LatencyP99Ms: 0, WindowMinutes: 60,
	}

	// p50=50 (within), p99=9999 but target p99=0 → skipped
	repo.On("GetDeliveryStats", ctx, "ten", "ep", 60).
		Return(100, 100, 0, 50.0, 50, 9999, nil)

	status, err := svc.GetComplianceStatus(ctx, "ten", target)
	require.NoError(t, err)
	assert.True(t, status.IsCompliant)
}

// ---------------------------------------------------------------------------
// CheckAndRecordBreaches
// ---------------------------------------------------------------------------

func TestCheckAndRecordBreaches_DeliveryRateBreach_Critical(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("ListTargets", ctx, "ten").Return([]Target{
		{ID: "t1", TenantID: "ten", EndpointID: "ep", DeliveryRatePct: 99.0, WindowMinutes: 60, IsActive: true},
	}, nil)
	// 80% rate → gap > 10% → critical
	repo.On("GetDeliveryStats", ctx, "ten", "ep", 60).
		Return(100, 80, 20, 0.0, 0, 0, nil)
	repo.On("CreateBreach", ctx, mock.AnythingOfType("*sla.Breach")).Return(nil)

	breaches, err := svc.CheckAndRecordBreaches(ctx, "ten")
	require.NoError(t, err)
	require.Len(t, breaches, 1)
	assert.Equal(t, BreachTypeDeliveryRate, breaches[0].BreachType)
	assert.Equal(t, SeverityCritical, breaches[0].Severity)
}

func TestCheckAndRecordBreaches_DeliveryRateBreach_Warning(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("ListTargets", ctx, "ten").Return([]Target{
		{ID: "t1", TenantID: "ten", EndpointID: "ep", DeliveryRatePct: 99.0, WindowMinutes: 60, IsActive: true},
	}, nil)
	// 95% rate → gap = 4% → warning
	repo.On("GetDeliveryStats", ctx, "ten", "ep", 60).
		Return(100, 95, 5, 0.0, 0, 0, nil)
	repo.On("CreateBreach", ctx, mock.AnythingOfType("*sla.Breach")).Return(nil)

	breaches, err := svc.CheckAndRecordBreaches(ctx, "ten")
	require.NoError(t, err)
	require.Len(t, breaches, 1)
	assert.Equal(t, SeverityWarning, breaches[0].Severity)
}

func TestCheckAndRecordBreaches_LatencyP99Breach(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("ListTargets", ctx, "ten").Return([]Target{
		{ID: "t1", TenantID: "ten", EndpointID: "ep", DeliveryRatePct: 90.0, LatencyP99Ms: 500, WindowMinutes: 60, IsActive: true},
	}, nil)
	// rate ok (100%) but p99 above target
	repo.On("GetDeliveryStats", ctx, "ten", "ep", 60).
		Return(100, 100, 0, 0.0, 0, 600, nil)
	repo.On("CreateBreach", ctx, mock.AnythingOfType("*sla.Breach")).Return(nil)

	breaches, err := svc.CheckAndRecordBreaches(ctx, "ten")
	require.NoError(t, err)
	require.Len(t, breaches, 1)
	assert.Equal(t, BreachTypeLatencyP99, breaches[0].BreachType)
	assert.Equal(t, SeverityWarning, breaches[0].Severity)
}

func TestCheckAndRecordBreaches_WithinSLA(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("ListTargets", ctx, "ten").Return([]Target{
		{ID: "t1", TenantID: "ten", EndpointID: "ep", DeliveryRatePct: 90.0, WindowMinutes: 60, IsActive: true},
	}, nil)
	repo.On("GetDeliveryStats", ctx, "ten", "ep", 60).
		Return(100, 100, 0, 0.0, 0, 0, nil)

	breaches, err := svc.CheckAndRecordBreaches(ctx, "ten")
	require.NoError(t, err)
	assert.Empty(t, breaches)
}

func TestCheckAndRecordBreaches_InactiveSkipped(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("ListTargets", ctx, "ten").Return([]Target{
		{ID: "t1", TenantID: "ten", EndpointID: "ep1", DeliveryRatePct: 99.0, WindowMinutes: 60, IsActive: false},
		{ID: "t2", TenantID: "ten", EndpointID: "ep2", DeliveryRatePct: 90.0, WindowMinutes: 60, IsActive: true},
	}, nil)
	repo.On("GetDeliveryStats", ctx, "ten", "ep2", 60).
		Return(100, 100, 0, 0.0, 0, 0, nil)

	breaches, err := svc.CheckAndRecordBreaches(ctx, "ten")
	require.NoError(t, err)
	assert.Empty(t, breaches)
	// ep1 stats should never be fetched
	repo.AssertNotCalled(t, "GetDeliveryStats", ctx, "ten", "ep1", 60)
}

func TestCheckAndRecordBreaches_BreachRecordedViaRepo(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("ListTargets", ctx, "ten").Return([]Target{
		{ID: "t1", TenantID: "ten", EndpointID: "ep", DeliveryRatePct: 99.0, WindowMinutes: 60, IsActive: true},
	}, nil)
	repo.On("GetDeliveryStats", ctx, "ten", "ep", 60).
		Return(100, 50, 50, 0.0, 0, 0, nil)
	repo.On("CreateBreach", ctx, mock.AnythingOfType("*sla.Breach")).Return(nil)

	_, err := svc.CheckAndRecordBreaches(ctx, "ten")
	require.NoError(t, err)
	repo.AssertCalled(t, "CreateBreach", ctx, mock.AnythingOfType("*sla.Breach"))
}

// ---------------------------------------------------------------------------
// calculateBurnRate (tested via GetDashboard)
// ---------------------------------------------------------------------------

func TestGetDashboard_BurnRate_ZeroErrors(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("ListTargets", ctx, "ten").Return([]Target{
		{ID: "t1", TenantID: "ten", EndpointID: "ep", DeliveryRatePct: 99.0, WindowMinutes: 60, IsActive: true},
	}, nil)
	repo.On("GetDeliveryStats", ctx, "ten", "ep", 60).
		Return(1000, 1000, 0, 0.0, 0, 0, nil)
	repo.On("ListActiveBreaches", ctx, "ten").Return([]Breach{}, nil)

	dash, err := svc.GetDashboard(ctx, "ten")
	require.NoError(t, err)
	require.Len(t, dash.BurnRates, 1)
	assert.Equal(t, 0.0, dash.BurnRates[0].CurrentRate)
	assert.False(t, dash.BurnRates[0].IsAtRisk)
	assert.Greater(t, dash.BurnRates[0].ErrorBudgetPct, 0.0)
}

func TestGetDashboard_BurnRate_TotalFailure(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("ListTargets", ctx, "ten").Return([]Target{
		{ID: "t1", TenantID: "ten", EndpointID: "ep", DeliveryRatePct: 99.0, WindowMinutes: 60, IsActive: true},
	}, nil)
	repo.On("GetDeliveryStats", ctx, "ten", "ep", 60).
		Return(100, 0, 100, 0.0, 0, 0, nil)
	repo.On("ListActiveBreaches", ctx, "ten").Return([]Breach{}, nil)

	dash, err := svc.GetDashboard(ctx, "ten")
	require.NoError(t, err)
	require.Len(t, dash.BurnRates, 1)
	assert.True(t, dash.BurnRates[0].IsAtRisk)
	assert.Greater(t, dash.BurnRates[0].CurrentRate, 1.0)
	assert.Equal(t, "breached", dash.BurnRates[0].ProjectedBreachIn)
}

func TestGetDashboard_BurnRate_PartialFailure(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("ListTargets", ctx, "ten").Return([]Target{
		{ID: "t1", TenantID: "ten", EndpointID: "ep", DeliveryRatePct: 99.0, WindowMinutes: 60, IsActive: true},
	}, nil)
	// 98% → error rate 2%, budget 1% → burn rate 2.0
	repo.On("GetDeliveryStats", ctx, "ten", "ep", 60).
		Return(100, 98, 2, 0.0, 0, 0, nil)
	repo.On("ListActiveBreaches", ctx, "ten").Return([]Breach{}, nil)

	dash, err := svc.GetDashboard(ctx, "ten")
	require.NoError(t, err)
	require.Len(t, dash.BurnRates, 1)
	assert.Equal(t, 2.0, dash.BurnRates[0].CurrentRate)
	assert.True(t, dash.BurnRates[0].IsAtRisk)
}

func TestGetDashboard_BurnRate_ErrorBudgetZero(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("ListTargets", ctx, "ten").Return([]Target{
		{ID: "t1", TenantID: "ten", EndpointID: "ep", DeliveryRatePct: 100.0, WindowMinutes: 60, IsActive: true},
	}, nil)
	repo.On("GetDeliveryStats", ctx, "ten", "ep", 60).
		Return(100, 100, 0, 0.0, 0, 0, nil)
	repo.On("ListActiveBreaches", ctx, "ten").Return([]Breach{}, nil)

	dash, err := svc.GetDashboard(ctx, "ten")
	require.NoError(t, err)
	require.Len(t, dash.BurnRates, 1)
	assert.True(t, dash.BurnRates[0].IsAtRisk)
	assert.Equal(t, 0.0, dash.BurnRates[0].ErrorBudgetPct)
}

// ---------------------------------------------------------------------------
// GetDashboard
// ---------------------------------------------------------------------------

func TestGetDashboard_AggregatesTargets(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("ListTargets", ctx, "ten").Return([]Target{
		{ID: "t1", TenantID: "ten", EndpointID: "ep1", DeliveryRatePct: 90.0, WindowMinutes: 60, IsActive: true},
		{ID: "t2", TenantID: "ten", EndpointID: "ep2", DeliveryRatePct: 95.0, WindowMinutes: 30, IsActive: true},
	}, nil)
	repo.On("GetDeliveryStats", ctx, "ten", "ep1", 60).
		Return(100, 100, 0, 0.0, 0, 0, nil)
	repo.On("GetDeliveryStats", ctx, "ten", "ep2", 30).
		Return(100, 100, 0, 0.0, 0, 0, nil)
	repo.On("ListActiveBreaches", ctx, "ten").Return([]Breach{}, nil)

	dash, err := svc.GetDashboard(ctx, "ten")
	require.NoError(t, err)
	assert.Len(t, dash.Targets, 2)
	assert.Len(t, dash.BurnRates, 2)
}

func TestGetDashboard_OverallScore(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("ListTargets", ctx, "ten").Return([]Target{
		{ID: "t1", TenantID: "ten", EndpointID: "ep1", DeliveryRatePct: 90.0, WindowMinutes: 60, IsActive: true},
		{ID: "t2", TenantID: "ten", EndpointID: "ep2", DeliveryRatePct: 95.0, WindowMinutes: 30, IsActive: true},
	}, nil)
	// t1 compliant
	repo.On("GetDeliveryStats", ctx, "ten", "ep1", 60).
		Return(100, 100, 0, 0.0, 0, 0, nil)
	// t2 not compliant
	repo.On("GetDeliveryStats", ctx, "ten", "ep2", 30).
		Return(100, 50, 50, 0.0, 0, 0, nil)
	repo.On("ListActiveBreaches", ctx, "ten").Return([]Breach{}, nil)

	dash, err := svc.GetDashboard(ctx, "ten")
	require.NoError(t, err)
	assert.Equal(t, 50.0, dash.OverallScore) // 1/2 * 100
}

func TestGetDashboard_IncludesActiveBreaches(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("ListTargets", ctx, "ten").Return([]Target{}, nil)
	repo.On("ListActiveBreaches", ctx, "ten").Return([]Breach{{ID: "b1", TenantID: "ten"}}, nil)

	dash, err := svc.GetDashboard(ctx, "ten")
	require.NoError(t, err)
	assert.Len(t, dash.ActiveBreaches, 1)
}

func TestGetDashboard_EmptyTargets(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("ListTargets", ctx, "ten").Return([]Target{}, nil)
	repo.On("ListActiveBreaches", ctx, "ten").Return([]Breach{}, nil)

	dash, err := svc.GetDashboard(ctx, "ten")
	require.NoError(t, err)
	assert.Equal(t, 0.0, dash.OverallScore)
	assert.Empty(t, dash.Targets)
}

func TestGetDashboard_InactiveTargetsSkipped(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("ListTargets", ctx, "ten").Return([]Target{
		{ID: "t1", TenantID: "ten", EndpointID: "ep1", DeliveryRatePct: 90.0, WindowMinutes: 60, IsActive: false},
	}, nil)
	repo.On("ListActiveBreaches", ctx, "ten").Return([]Breach{}, nil)

	dash, err := svc.GetDashboard(ctx, "ten")
	require.NoError(t, err)
	assert.Empty(t, dash.Targets)
	assert.Equal(t, 0.0, dash.OverallScore)
}

// ---------------------------------------------------------------------------
// GetBreachHistory
// ---------------------------------------------------------------------------

func TestGetBreachHistory_DefaultLimit(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("ListBreachHistory", ctx, "ten", 50, 0).Return([]Breach{}, nil)

	_, err := svc.GetBreachHistory(ctx, "ten", 0, 0)
	require.NoError(t, err)
	repo.AssertCalled(t, "ListBreachHistory", ctx, "ten", 50, 0)
}

func TestGetBreachHistory_NegativeLimit(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("ListBreachHistory", ctx, "ten", 50, 5).Return([]Breach{}, nil)

	_, err := svc.GetBreachHistory(ctx, "ten", -1, 5)
	require.NoError(t, err)
	repo.AssertCalled(t, "ListBreachHistory", ctx, "ten", 50, 5)
}

// ---------------------------------------------------------------------------
// UpdateTarget
// ---------------------------------------------------------------------------

func TestUpdateTarget_Delegates(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	existing := &Target{ID: "t1", TenantID: "ten", Name: "old"}
	repo.On("GetTarget", ctx, "ten", "t1").Return(existing, nil)
	repo.On("UpdateTarget", ctx, mock.AnythingOfType("*sla.Target")).Return(nil)

	req := &CreateTargetRequest{Name: "new", DeliveryRatePct: 99.5, WindowMinutes: 30}
	updated, err := svc.UpdateTarget(ctx, "ten", "t1", req)
	require.NoError(t, err)
	assert.Equal(t, "new", updated.Name)
	assert.Equal(t, 99.5, updated.DeliveryRatePct)
	repo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// DeleteTarget
// ---------------------------------------------------------------------------

func TestDeleteTarget_Delegates(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("DeleteTarget", ctx, "ten", "t1").Return(nil)

	err := svc.DeleteTarget(ctx, "ten", "t1")
	require.NoError(t, err)
	repo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// Additional coverage
// ---------------------------------------------------------------------------

func TestGetDashboard_ZeroDeliveriesBurnRate(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("ListTargets", ctx, "ten").Return([]Target{
		{ID: "t1", TenantID: "ten", EndpointID: "ep", DeliveryRatePct: 99.5, WindowMinutes: 60, IsActive: true},
	}, nil)
	// 0 total deliveries → currentRate=0
	repo.On("GetDeliveryStats", ctx, "ten", "ep", 60).
		Return(0, 0, 0, 0.0, 0, 0, nil)
	repo.On("ListActiveBreaches", ctx, "ten").Return([]Breach{}, nil)

	dash, err := svc.GetDashboard(ctx, "ten")
	require.NoError(t, err)
	require.Len(t, dash.BurnRates, 1)

	br := dash.BurnRates[0]
	// errorBudget=0.5, currentErrorRate=100, burnRate=200
	assert.Equal(t, 0.5, 100.0-99.5)        // error budget sanity
	assert.True(t, br.IsAtRisk)             // burnRate >> 1
	assert.Greater(t, br.CurrentRate, 1.0)  // burn rate > 1
	assert.Equal(t, 0.0, br.ErrorBudgetPct) // budget exhausted
	assert.Empty(t, br.ProjectedBreachIn)   // TotalDeliveries==0 → no projection
}

func TestCheckAndRecordBreaches_InactiveTargetSkipped(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("ListTargets", ctx, "ten").Return([]Target{
		{ID: "t1", TenantID: "ten", EndpointID: "ep", DeliveryRatePct: 99.0, WindowMinutes: 60, IsActive: false},
	}, nil)

	breaches, err := svc.CheckAndRecordBreaches(ctx, "ten")
	require.NoError(t, err)
	assert.Empty(t, breaches)
	repo.AssertNotCalled(t, "GetDeliveryStats", ctx, "ten", "ep", 60)
	repo.AssertNotCalled(t, "CreateBreach", ctx, mock.Anything)
}

func TestCheckAndRecordBreaches_MultipleTargets(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("ListTargets", ctx, "ten").Return([]Target{
		{ID: "t1", TenantID: "ten", EndpointID: "ep1", DeliveryRatePct: 90.0, WindowMinutes: 60, IsActive: true},
		{ID: "t2", TenantID: "ten", EndpointID: "ep2", DeliveryRatePct: 99.0, WindowMinutes: 60, IsActive: true},
	}, nil)
	// t1 compliant (100% >= 90%)
	repo.On("GetDeliveryStats", ctx, "ten", "ep1", 60).
		Return(100, 100, 0, 0.0, 0, 0, nil)
	// t2 not compliant (80% < 99%)
	repo.On("GetDeliveryStats", ctx, "ten", "ep2", 60).
		Return(100, 80, 20, 0.0, 0, 0, nil)
	repo.On("CreateBreach", ctx, mock.AnythingOfType("*sla.Breach")).Return(nil)

	breaches, err := svc.CheckAndRecordBreaches(ctx, "ten")
	require.NoError(t, err)
	require.Len(t, breaches, 1)
	assert.Equal(t, "t2", breaches[0].TargetID)
}

func TestCalculateSeverity_CriticalGap(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("ListTargets", ctx, "ten").Return([]Target{
		{ID: "t1", TenantID: "ten", EndpointID: "ep", DeliveryRatePct: 99.0, WindowMinutes: 60, IsActive: true},
	}, nil)
	// 80% → gap = 99 - 80 = 19 > 10 → critical
	repo.On("GetDeliveryStats", ctx, "ten", "ep", 60).
		Return(100, 80, 20, 0.0, 0, 0, nil)
	repo.On("CreateBreach", ctx, mock.AnythingOfType("*sla.Breach")).Return(nil)

	breaches, err := svc.CheckAndRecordBreaches(ctx, "ten")
	require.NoError(t, err)
	require.Len(t, breaches, 1)
	assert.Equal(t, SeverityCritical, breaches[0].Severity)
}

func TestCalculateSeverity_WarningGap(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("ListTargets", ctx, "ten").Return([]Target{
		{ID: "t1", TenantID: "ten", EndpointID: "ep", DeliveryRatePct: 99.0, WindowMinutes: 60, IsActive: true},
	}, nil)
	// 95% → gap = 99 - 95 = 4 ≤ 10 → warning
	repo.On("GetDeliveryStats", ctx, "ten", "ep", 60).
		Return(100, 95, 5, 0.0, 0, 0, nil)
	repo.On("CreateBreach", ctx, mock.AnythingOfType("*sla.Breach")).Return(nil)

	breaches, err := svc.CheckAndRecordBreaches(ctx, "ten")
	require.NoError(t, err)
	require.Len(t, breaches, 1)
	assert.Equal(t, SeverityWarning, breaches[0].Severity)
}

func TestGetBreachHistory_ZeroLimitDefaultsTo50(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("ListBreachHistory", ctx, "ten", 50, 10).Return([]Breach{}, nil)

	result, err := svc.GetBreachHistory(ctx, "ten", 0, 10)
	require.NoError(t, err)
	assert.Empty(t, result)
	repo.AssertCalled(t, "ListBreachHistory", ctx, "ten", 50, 10)
}

func TestGetDashboard_InactiveTargetsExcluded(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("ListTargets", ctx, "ten").Return([]Target{
		{ID: "t1", TenantID: "ten", EndpointID: "ep1", DeliveryRatePct: 90.0, WindowMinutes: 60, IsActive: true},
		{ID: "t2", TenantID: "ten", EndpointID: "ep2", DeliveryRatePct: 95.0, WindowMinutes: 30, IsActive: false},
		{ID: "t3", TenantID: "ten", EndpointID: "ep3", DeliveryRatePct: 99.0, WindowMinutes: 60, IsActive: true},
	}, nil)
	repo.On("GetDeliveryStats", ctx, "ten", "ep1", 60).
		Return(100, 100, 0, 0.0, 0, 0, nil)
	repo.On("GetDeliveryStats", ctx, "ten", "ep3", 60).
		Return(100, 100, 0, 0.0, 0, 0, nil)
	repo.On("ListActiveBreaches", ctx, "ten").Return([]Breach{}, nil)

	dash, err := svc.GetDashboard(ctx, "ten")
	require.NoError(t, err)
	assert.Len(t, dash.Targets, 2)
	assert.Len(t, dash.BurnRates, 2)
	assert.Equal(t, 100.0, dash.OverallScore)
	// inactive target ep2 stats should never be fetched
	repo.AssertNotCalled(t, "GetDeliveryStats", ctx, "ten", "ep2", 30)
}

// ---------- Concurrent breach recording (sequential simulation) ----------

func TestCheckAndRecordBreaches_MultipleTargetsWithMixedCompliance(t *testing.T) {
	t.Parallel()
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	targets := []Target{
		{ID: "t1", TenantID: "ten", EndpointID: "ep1", DeliveryRatePct: 99.0, WindowMinutes: 60, IsActive: true},
		{ID: "t2", TenantID: "ten", EndpointID: "ep2", DeliveryRatePct: 95.0, WindowMinutes: 60, IsActive: true},
		{ID: "t3", TenantID: "ten", EndpointID: "ep3", DeliveryRatePct: 99.0, WindowMinutes: 60, IsActive: true},
	}

	repo.On("ListTargets", ctx, "ten").Return(targets, nil)
	// ep1: below target (breach)
	repo.On("GetDeliveryStats", ctx, "ten", "ep1", 60).Return(100, 90, 10, 50.0, 10, 50, nil)
	// ep2: above target (compliant)
	repo.On("GetDeliveryStats", ctx, "ten", "ep2", 60).Return(100, 98, 2, 50.0, 10, 50, nil)
	// ep3: below target (breach)
	repo.On("GetDeliveryStats", ctx, "ten", "ep3", 60).Return(100, 80, 20, 50.0, 10, 50, nil)

	repo.On("CreateBreach", ctx, mock.AnythingOfType("*sla.Breach")).Return(nil)

	breaches, err := svc.CheckAndRecordBreaches(ctx, "ten")

	require.NoError(t, err)
	assert.Len(t, breaches, 2, "should have 2 breaches (ep1 and ep3)")

	// Verify both breach endpoint IDs
	endpoints := make(map[string]bool)
	for _, b := range breaches {
		endpoints[b.EndpointID] = true
		assert.Equal(t, BreachTypeDeliveryRate, b.BreachType)
	}
	assert.True(t, endpoints["ep1"])
	assert.True(t, endpoints["ep3"])
}

// ---------- Burn rate projection calculation ----------

func TestGetDashboard_BurnRateProjection(t *testing.T) {
	t.Parallel()
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	// Target: 99% delivery rate, 60 min window
	// Actual: 95% delivery rate → errorBudget=1%, currentError=5%, burnRate=5.0
	target := Target{
		ID: "t1", TenantID: "ten", EndpointID: "ep1",
		DeliveryRatePct: 99.0, WindowMinutes: 60, IsActive: true,
	}
	repo.On("ListTargets", ctx, "ten").Return([]Target{target}, nil)
	repo.On("GetDeliveryStats", ctx, "ten", "ep1", 60).Return(100, 95, 5, 50.0, 10, 50, nil)
	repo.On("ListActiveBreaches", ctx, "ten").Return([]Breach{}, nil)

	dash, err := svc.GetDashboard(ctx, "ten")

	require.NoError(t, err)
	require.Len(t, dash.BurnRates, 1)
	br := dash.BurnRates[0]
	assert.True(t, br.IsAtRisk, "burn rate > 1.0 should be at risk")
	assert.True(t, br.CurrentRate > 1.0, "burn rate should exceed 1.0")
	assert.NotEmpty(t, br.ProjectedBreachIn, "should have projected breach time")
}
