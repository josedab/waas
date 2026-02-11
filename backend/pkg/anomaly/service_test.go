package anomaly

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Mock Repository ---

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetBaseline(ctx context.Context, tenantID, endpointID string, metricType MetricType) (*Baseline, error) {
	args := m.Called(ctx, tenantID, endpointID, metricType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Baseline), args.Error(1)
}

func (m *MockRepository) SaveBaseline(ctx context.Context, baseline *Baseline) error {
	args := m.Called(ctx, baseline)
	return args.Error(0)
}

func (m *MockRepository) SaveAnomaly(ctx context.Context, anomaly *Anomaly) error {
	args := m.Called(ctx, anomaly)
	return args.Error(0)
}

func (m *MockRepository) GetAnomaly(ctx context.Context, tenantID, anomalyID string) (*Anomaly, error) {
	args := m.Called(ctx, tenantID, anomalyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Anomaly), args.Error(1)
}

func (m *MockRepository) ListAnomalies(ctx context.Context, tenantID string, status string, limit, offset int) ([]Anomaly, int, error) {
	args := m.Called(ctx, tenantID, status, limit, offset)
	return args.Get(0).([]Anomaly), args.Int(1), args.Error(2)
}

func (m *MockRepository) UpdateAnomalyStatus(ctx context.Context, tenantID, anomalyID, status string) error {
	args := m.Called(ctx, tenantID, anomalyID, status)
	return args.Error(0)
}

func (m *MockRepository) GetDetectionConfig(ctx context.Context, tenantID, endpointID string, metricType MetricType) (*DetectionConfig, error) {
	args := m.Called(ctx, tenantID, endpointID, metricType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DetectionConfig), args.Error(1)
}

func (m *MockRepository) SaveDetectionConfig(ctx context.Context, config *DetectionConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockRepository) ListDetectionConfigs(ctx context.Context, tenantID string) ([]DetectionConfig, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]DetectionConfig), args.Error(1)
}

func (m *MockRepository) GetAlertConfigs(ctx context.Context, tenantID string) ([]AlertConfig, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]AlertConfig), args.Error(1)
}

func (m *MockRepository) SaveAlertConfig(ctx context.Context, config *AlertConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockRepository) SaveAlert(ctx context.Context, alert *AnomalyAlert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *MockRepository) ListAlerts(ctx context.Context, tenantID string, anomalyID string, limit, offset int) ([]AnomalyAlert, int, error) {
	args := m.Called(ctx, tenantID, anomalyID, limit, offset)
	return args.Get(0).([]AnomalyAlert), args.Int(1), args.Error(2)
}

func (m *MockRepository) GetRecentMetrics(ctx context.Context, tenantID, endpointID string, metricType MetricType, duration time.Duration) ([]MetricDataPoint, error) {
	args := m.Called(ctx, tenantID, endpointID, metricType, duration)
	return args.Get(0).([]MetricDataPoint), args.Error(1)
}

// --- Mock AlertNotifier ---

type MockAlertNotifier struct {
	mock.Mock
}

func (m *MockAlertNotifier) Send(ctx context.Context, alert *AnomalyAlert, anomaly *Anomaly, config *AlertConfig) error {
	args := m.Called(ctx, alert, anomaly, config)
	return args.Error(0)
}

// --- CheckMetric tests ---

func TestCheckMetric_NoBaseline(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo, nil)
	ctx := context.Background()

	repo.On("GetBaseline", ctx, "t1", "ep1", MetricTypeErrorRate).Return(nil, nil)

	result, err := svc.CheckMetric(ctx, "t1", "ep1", MetricTypeErrorRate, 50.0)
	require.NoError(t, err)
	assert.False(t, result.IsAnomaly)
	assert.Equal(t, "No baseline established", result.Description)
	repo.AssertExpectations(t)
}

func TestCheckMetric_AnomalyDetected(t *testing.T) {
	repo := new(MockRepository)
	notifier := new(MockAlertNotifier)
	svc := NewService(repo, notifier)
	ctx := context.Background()

	baseline := baselineWithSamples(100, 10, 50)
	repo.On("GetBaseline", ctx, "t1", "ep1", MetricTypeErrorRate).Return(baseline, nil)
	repo.On("GetDetectionConfig", ctx, "t1", "ep1", MetricTypeErrorRate).Return(nil, nil)
	repo.On("SaveAnomaly", ctx, mock.AnythingOfType("*anomaly.Anomaly")).Return(nil)

	// Async goroutine calls — use Maybe() to avoid flaky tests
	repo.On("GetAlertConfigs", mock.Anything, "t1").Return([]AlertConfig{
		{ID: "ac1", TenantID: "t1", Channel: "slack", MinSeverity: SeverityWarning, Enabled: true},
	}, nil).Maybe()
	notifier.On("Send", mock.Anything, mock.AnythingOfType("*anomaly.AnomalyAlert"), mock.AnythingOfType("*anomaly.Anomaly"), mock.AnythingOfType("*anomaly.AlertConfig")).Return(nil).Maybe()
	repo.On("SaveAlert", mock.Anything, mock.AnythingOfType("*anomaly.AnomalyAlert")).Return(nil).Maybe()

	result, err := svc.CheckMetric(ctx, "t1", "ep1", MetricTypeErrorRate, 135.0) // 3.5σ
	require.NoError(t, err)
	assert.True(t, result.IsAnomaly)
	assert.Equal(t, SeverityCritical, result.Severity)

	// Give goroutine time to complete
	time.Sleep(100 * time.Millisecond)

	repo.AssertCalled(t, "SaveAnomaly", ctx, mock.AnythingOfType("*anomaly.Anomaly"))
}

func TestCheckMetric_NormalValue(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo, nil)
	ctx := context.Background()

	baseline := baselineWithSamples(100, 10, 50)
	repo.On("GetBaseline", ctx, "t1", "ep1", MetricTypeErrorRate).Return(baseline, nil)
	repo.On("GetDetectionConfig", ctx, "t1", "ep1", MetricTypeErrorRate).Return(nil, nil)

	result, err := svc.CheckMetric(ctx, "t1", "ep1", MetricTypeErrorRate, 105.0) // 0.5σ
	require.NoError(t, err)
	assert.False(t, result.IsAnomaly)
	repo.AssertNotCalled(t, "SaveAnomaly", mock.Anything, mock.Anything)
}

func TestCheckMetric_NilNotifier_NoAlertPanic(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo, nil) // nil notifier
	ctx := context.Background()

	baseline := baselineWithSamples(100, 10, 50)
	repo.On("GetBaseline", ctx, "t1", "ep1", MetricTypeErrorRate).Return(baseline, nil)
	repo.On("GetDetectionConfig", ctx, "t1", "ep1", MetricTypeErrorRate).Return(nil, nil)
	repo.On("SaveAnomaly", ctx, mock.AnythingOfType("*anomaly.Anomaly")).Return(nil)

	result, err := svc.CheckMetric(ctx, "t1", "ep1", MetricTypeErrorRate, 135.0)
	require.NoError(t, err)
	assert.True(t, result.IsAnomaly)

	// sendAlerts with nil notifier is a noop; give goroutine time
	time.Sleep(100 * time.Millisecond)
}

// --- UpdateBaseline tests ---

func TestUpdateBaseline_SavesBaseline(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo, nil)
	ctx := context.Background()

	points := makePoints([]float64{10, 20, 30, 40, 50})
	repo.On("GetRecentMetrics", ctx, "t1", "ep1", MetricTypeLatencyP95, 24*time.Hour).Return(points, nil)
	repo.On("GetBaseline", ctx, "t1", "ep1", MetricTypeLatencyP95).Return(nil, nil)
	repo.On("SaveBaseline", ctx, mock.AnythingOfType("*anomaly.Baseline")).Return(nil)

	err := svc.UpdateBaseline(ctx, "t1", "ep1", MetricTypeLatencyP95, 24*time.Hour)
	require.NoError(t, err)
	repo.AssertCalled(t, "SaveBaseline", ctx, mock.AnythingOfType("*anomaly.Baseline"))
}

func TestUpdateBaseline_NoData_Noop(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo, nil)
	ctx := context.Background()

	repo.On("GetRecentMetrics", ctx, "t1", "ep1", MetricTypeLatencyP95, 24*time.Hour).Return([]MetricDataPoint{}, nil)

	err := svc.UpdateBaseline(ctx, "t1", "ep1", MetricTypeLatencyP95, 24*time.Hour)
	require.NoError(t, err)
	repo.AssertNotCalled(t, "SaveBaseline", mock.Anything, mock.Anything)
}

// --- AcknowledgeAnomaly / ResolveAnomaly tests ---

func TestAcknowledgeAnomaly(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo, nil)
	ctx := context.Background()

	expected := &Anomaly{ID: "a1", TenantID: "t1", Status: "acknowledged"}
	repo.On("UpdateAnomalyStatus", ctx, "t1", "a1", "acknowledged").Return(nil)
	repo.On("GetAnomaly", ctx, "t1", "a1").Return(expected, nil)

	result, err := svc.AcknowledgeAnomaly(ctx, "t1", "a1")
	require.NoError(t, err)
	assert.Equal(t, "acknowledged", result.Status)
	repo.AssertExpectations(t)
}

func TestResolveAnomaly(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo, nil)
	ctx := context.Background()

	expected := &Anomaly{ID: "a1", TenantID: "t1", Status: "resolved"}
	repo.On("UpdateAnomalyStatus", ctx, "t1", "a1", "resolved").Return(nil)
	repo.On("GetAnomaly", ctx, "t1", "a1").Return(expected, nil)

	result, err := svc.ResolveAnomaly(ctx, "t1", "a1")
	require.NoError(t, err)
	assert.Equal(t, "resolved", result.Status)
	repo.AssertExpectations(t)
}

// --- GetAnomalies pagination tests ---

func TestGetAnomalies_PaginationClamping(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo, nil)
	ctx := context.Background()

	tests := []struct {
		name          string
		inputLimit    int
		expectedLimit int
	}{
		{"zero limit defaults to 20", 0, 20},
		{"negative limit defaults to 20", -1, 20},
		{"over 100 clamped to 100", 200, 100},
		{"normal limit passed through", 50, 50},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo.On("ListAnomalies", ctx, "t1", "", tc.expectedLimit, 0).Return([]Anomaly{}, 0, nil).Once()
			_, _, err := svc.GetAnomalies(ctx, "t1", "", tc.inputLimit, 0)
			require.NoError(t, err)
		})
	}
	repo.AssertExpectations(t)
}

// --- CreateDetectionConfig tests ---

func TestCreateDetectionConfig_DefaultsApplied(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo, nil)
	ctx := context.Background()

	repo.On("SaveDetectionConfig", ctx, mock.AnythingOfType("*anomaly.DetectionConfig")).Return(nil)

	req := &CreateDetectionConfigRequest{MetricType: MetricTypeErrorRate}
	cfg, err := svc.CreateDetectionConfig(ctx, "t1", req)
	require.NoError(t, err)

	assert.Equal(t, 1.0, cfg.Sensitivity)
	assert.Equal(t, 30, cfg.MinSamples)
	assert.Equal(t, 15, cfg.CooldownMinutes)
	assert.Equal(t, 3.0, cfg.CriticalThreshold)
	assert.Equal(t, 2.0, cfg.WarningThreshold)
	assert.True(t, cfg.Enabled)
	assert.NotEmpty(t, cfg.ID)
	assert.Equal(t, "t1", cfg.TenantID)
}

// --- CreateAlertConfig tests ---

func TestCreateAlertConfig_SavesWithEnabledTrue(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo, nil)
	ctx := context.Background()

	repo.On("SaveAlertConfig", ctx, mock.AnythingOfType("*anomaly.AlertConfig")).Return(nil)

	req := &CreateAlertConfigRequest{
		Name:        "My Alert",
		Channel:     "slack",
		Config:      `{"webhook_url":"https://example.com"}`,
		MinSeverity: SeverityWarning,
	}
	cfg, err := svc.CreateAlertConfig(ctx, "t1", req)
	require.NoError(t, err)

	assert.True(t, cfg.Enabled)
	assert.Equal(t, "t1", cfg.TenantID)
	assert.Equal(t, "slack", cfg.Channel)
	assert.NotEmpty(t, cfg.ID)
	repo.AssertExpectations(t)
}

// --- GetTrendAnalysis tests ---

func TestGetTrendAnalysis_UseDetector(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo, nil)
	ctx := context.Background()

	points := makePoints([]float64{10, 20, 30, 40, 50})
	baseline := baselineWithSamples(30, 1, 50)
	repo.On("GetRecentMetrics", ctx, "t1", "ep1", MetricTypeErrorRate, 24*time.Hour).Return(points, nil)
	repo.On("GetBaseline", ctx, "t1", "ep1", MetricTypeErrorRate).Return(baseline, nil)

	analysis, err := svc.GetTrendAnalysis(ctx, "t1", "ep1", MetricTypeErrorRate)
	require.NoError(t, err)
	assert.Equal(t, "increasing", analysis.Trend)
	assert.Equal(t, "t1", analysis.TenantID)
	assert.Equal(t, "ep1", analysis.EndpointID)
	assert.Equal(t, MetricTypeErrorRate, analysis.MetricType)
}

func TestGetTrendAnalysis_NilBaseline_UsesDefaults(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo, nil)
	ctx := context.Background()

	points := makePoints([]float64{1, 1, 1, 1})
	repo.On("GetRecentMetrics", ctx, "t1", "ep1", MetricTypeErrorRate, 24*time.Hour).Return(points, nil)
	repo.On("GetBaseline", ctx, "t1", "ep1", MetricTypeErrorRate).Return(nil, nil)

	analysis, err := svc.GetTrendAnalysis(ctx, "t1", "ep1", MetricTypeErrorRate)
	require.NoError(t, err)
	assert.Equal(t, "stable", analysis.Trend)
}

func TestGetTrendAnalysis_InsufficientData(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo, nil)
	ctx := context.Background()

	points := makePoints([]float64{10, 20}) // only 2 points
	baseline := baselineWithSamples(15, 5, 50)
	repo.On("GetRecentMetrics", ctx, "t1", "ep1", MetricTypeErrorRate, 24*time.Hour).Return(points, nil)
	repo.On("GetBaseline", ctx, "t1", "ep1", MetricTypeErrorRate).Return(baseline, nil)

	analysis, err := svc.GetTrendAnalysis(ctx, "t1", "ep1", MetricTypeErrorRate)
	require.NoError(t, err)
	assert.Equal(t, "insufficient_data", analysis.Trend)
}
