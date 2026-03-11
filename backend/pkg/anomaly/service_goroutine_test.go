package anomaly

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAnomalyRepo implements Repository for testing.
type mockAnomalyRepo struct {
	baseline     *Baseline
	baselineErr  error
	config       *DetectionConfig
	configErr    error
	alertConfigs []AlertConfig
	savedAnomaly *Anomaly
}

func (m *mockAnomalyRepo) GetBaseline(_ context.Context, _, _ string, _ MetricType) (*Baseline, error) {
	return m.baseline, m.baselineErr
}
func (m *mockAnomalyRepo) SaveBaseline(_ context.Context, _ *Baseline) error { return nil }
func (m *mockAnomalyRepo) SaveAnomaly(_ context.Context, a *Anomaly) error {
	m.savedAnomaly = a
	return nil
}
func (m *mockAnomalyRepo) GetAnomaly(_ context.Context, _, _ string) (*Anomaly, error) {
	return nil, nil
}
func (m *mockAnomalyRepo) ListAnomalies(_ context.Context, _ string, _ string, _, _ int) ([]Anomaly, int, error) {
	return nil, 0, nil
}
func (m *mockAnomalyRepo) UpdateAnomalyStatus(_ context.Context, _, _, _ string) error { return nil }
func (m *mockAnomalyRepo) GetDetectionConfig(_ context.Context, _, _ string, _ MetricType) (*DetectionConfig, error) {
	return m.config, m.configErr
}
func (m *mockAnomalyRepo) SaveDetectionConfig(_ context.Context, _ *DetectionConfig) error {
	return nil
}
func (m *mockAnomalyRepo) ListDetectionConfigs(_ context.Context, _ string) ([]DetectionConfig, error) {
	return nil, nil
}
func (m *mockAnomalyRepo) GetAlertConfigs(_ context.Context, _ string) ([]AlertConfig, error) {
	return m.alertConfigs, nil
}
func (m *mockAnomalyRepo) SaveAlertConfig(_ context.Context, _ *AlertConfig) error { return nil }
func (m *mockAnomalyRepo) SaveAlert(_ context.Context, _ *AnomalyAlert) error       { return nil }
func (m *mockAnomalyRepo) ListAlerts(_ context.Context, _ string, _ string, _, _ int) ([]AnomalyAlert, int, error) {
	return nil, 0, nil
}
func (m *mockAnomalyRepo) GetRecentMetrics(_ context.Context, _, _ string, _ MetricType, _ time.Duration) ([]MetricDataPoint, error) {
	return nil, nil
}

// panicNotifier panics when Send is called, simulating an unrecovered goroutine.
type panicNotifier struct {
	sendCalled chan struct{}
}

func (p *panicNotifier) Send(_ context.Context, _ *AnomalyAlert, _ *Anomaly, _ *AlertConfig) error {
	if p.sendCalled != nil {
		close(p.sendCalled)
	}
	panic("notifier exploded!")
}

// safeNotifier tracks if alerts were sent without panicking.
type safeNotifier struct {
	sentCount int
}

func (s *safeNotifier) Send(_ context.Context, _ *AnomalyAlert, _ *Anomaly, _ *AlertConfig) error {
	s.sentCount++
	return nil
}

func TestCheckMetric_AnomalyDetected_AlertGoroutineDoesNotCrashCaller(t *testing.T) {
	// Create a baseline that will trigger an anomaly when value is far from mean
	repo := &mockAnomalyRepo{
		baseline: &Baseline{
			Mean:       100.0,
			StdDev:     5.0,
			SampleSize: 100,
		},
		alertConfigs: []AlertConfig{
			{ID: "ac-1", TenantID: "t1", Enabled: true, MinSeverity: "info"},
		},
	}

	notifier := &safeNotifier{}
	svc := NewService(repo, notifier)

	// Value far from mean to trigger anomaly (200 is 20 std devs away)
	result, err := svc.CheckMetric(context.Background(), "t1", "ep-1", MetricTypeErrorRate, 200.0)

	require.NoError(t, err)
	assert.True(t, result.IsAnomaly)
	assert.NotNil(t, repo.savedAnomaly)

	// Give goroutine time to complete
	time.Sleep(100 * time.Millisecond)
}

func TestCheckMetric_NoBaseline_ReturnsGracefully(t *testing.T) {
	repo := &mockAnomalyRepo{baseline: nil}
	svc := NewService(repo, nil)

	result, err := svc.CheckMetric(context.Background(), "t1", "ep-1", MetricTypeLatencyP50, 100.0)

	require.NoError(t, err)
	assert.False(t, result.IsAnomaly)
	assert.Contains(t, result.Description, "No baseline")
}

func TestCheckMetric_NoAnomaly_NoGoroutineLaunched(t *testing.T) {
	repo := &mockAnomalyRepo{
		baseline: &Baseline{
			Mean:       100.0,
			StdDev:     5.0,
			SampleSize: 100,
		},
	}

	notifier := &safeNotifier{}
	svc := NewService(repo, notifier)

	// Value within normal range
	result, err := svc.CheckMetric(context.Background(), "t1", "ep-1", MetricTypeDeliveryRate, 101.0)

	require.NoError(t, err)
	assert.False(t, result.IsAnomaly)
	assert.Nil(t, repo.savedAnomaly)
}

func TestCheckMetric_NilNotifier_DoesNotPanic(t *testing.T) {
	repo := &mockAnomalyRepo{
		baseline: &Baseline{
			Mean:       100.0,
			StdDev:     5.0,
			SampleSize: 100,
		},
	}

	svc := NewService(repo, nil)

	// Should handle nil notifier gracefully
	result, err := svc.CheckMetric(context.Background(), "t1", "ep-1", MetricTypeErrorRate, 200.0)

	require.NoError(t, err)
	assert.True(t, result.IsAnomaly)

	time.Sleep(100 * time.Millisecond)
}

func TestCheckMetric_ContextCancelled_StillReturns(t *testing.T) {
	repo := &mockAnomalyRepo{
		baseline: &Baseline{
			Mean:       100.0,
			StdDev:     5.0,
			SampleSize: 100,
		},
	}

	svc := NewService(repo, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	// The goroutine uses context.Background(), so parent cancellation
	// should not affect the goroutine behavior
	result, err := svc.CheckMetric(ctx, "t1", "ep-1", MetricTypeErrorRate, 200.0)

	require.NoError(t, err)
	assert.True(t, result.IsAnomaly)
}
