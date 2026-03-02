package analytics

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/utils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestLogger() *utils.Logger {
	return utils.NewLogger("test")
}

// =====================
// Start / Stop
// =====================

func TestAggregator_StartStop(t *testing.T) {
	mockRepo := &MockAnalyticsRepository{}
	logger := newTestLogger()
	agg := NewAggregator(mockRepo, logger)

	// Set up expectations for the initial processHourlyAggregation call on startup
	mockRepo.On("GetMetricsSummary", mock.Anything, mock.Anything).Return(&models.MetricsSummary{}, nil).Maybe()
	mockRepo.On("CleanupOldMetrics", mock.Anything, mock.Anything).Return(nil).Maybe()
	mockRepo.On("RecordRealtimeMetric", mock.Anything, mock.Anything).Return(nil).Maybe()

	ctx, cancel := context.WithCancel(context.Background())

	agg.Start(ctx)

	// Give workers time to start
	time.Sleep(100 * time.Millisecond)

	// Stop via both mechanisms
	cancel()
	agg.Stop()

	// If we get here without deadlock, graceful shutdown works
}

func TestAggregator_StopViaChannel(t *testing.T) {
	mockRepo := &MockAnalyticsRepository{}
	logger := newTestLogger()
	agg := NewAggregator(mockRepo, logger)

	mockRepo.On("GetMetricsSummary", mock.Anything, mock.Anything).Return(&models.MetricsSummary{}, nil).Maybe()
	mockRepo.On("CleanupOldMetrics", mock.Anything, mock.Anything).Return(nil).Maybe()
	mockRepo.On("RecordRealtimeMetric", mock.Anything, mock.Anything).Return(nil).Maybe()

	ctx := context.Background()
	agg.Start(ctx)

	time.Sleep(50 * time.Millisecond)

	// Stop only via stopCh
	done := make(chan struct{})
	go func() {
		agg.Stop()
		close(done)
	}()

	select {
	case <-done:
		// OK
	case <-time.After(5 * time.Second):
		t.Fatal("Stop did not complete within timeout")
	}
}

// =====================
// processHourlyAggregation
// =====================

func TestProcessHourlyAggregation_Success(t *testing.T) {
	mockRepo := &MockAnalyticsRepository{}
	logger := newTestLogger()
	agg := NewAggregator(mockRepo, logger)
	ctx := context.Background()

	hourTime := time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC)

	// No active tenants — should return quickly
	// getActiveTenantsInPeriod returns empty slice by default implementation

	agg.processHourlyAggregation(ctx, hourTime)
	// No assertions needed — just verifying no panic
}

func TestProcessHourlyAggregation_ContextCancellation(t *testing.T) {
	mockRepo := &MockAnalyticsRepository{}
	logger := newTestLogger()
	agg := NewAggregator(mockRepo, logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	hourTime := time.Now().Add(-time.Hour)
	// Should handle cancelled context gracefully
	agg.processHourlyAggregation(ctx, hourTime)
}

// =====================
// aggregateHourlyMetricsForTenant
// =====================

func TestAggregateHourlyMetrics_ZeroDeliveriesSkip(t *testing.T) {
	mockRepo := &MockAnalyticsRepository{}
	logger := newTestLogger()
	agg := NewAggregator(mockRepo, logger)
	ctx := context.Background()

	tenantID := uuid.New()
	hourStart := time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC)
	hourEnd := hourStart.Add(time.Hour)

	mockRepo.On("GetMetricsSummary", ctx, mock.MatchedBy(func(q *models.MetricsQuery) bool {
		return q.TenantID == tenantID
	})).Return(&models.MetricsSummary{
		TotalDeliveries: 0,
	}, nil)

	err := agg.aggregateHourlyMetricsForTenant(ctx, tenantID, nil, hourStart, hourEnd)
	require.NoError(t, err)

	// UpsertHourlyMetric should NOT be called when deliveries == 0
	mockRepo.AssertNotCalled(t, "UpsertHourlyMetric", mock.Anything, mock.Anything)
}

func TestAggregateHourlyMetrics_WithData(t *testing.T) {
	mockRepo := &MockAnalyticsRepository{}
	logger := newTestLogger()
	agg := NewAggregator(mockRepo, logger)
	ctx := context.Background()

	tenantID := uuid.New()
	hourStart := time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC)
	hourEnd := hourStart.Add(time.Hour)

	mockRepo.On("GetMetricsSummary", ctx, mock.Anything).Return(&models.MetricsSummary{
		TotalDeliveries:      100,
		SuccessfulDeliveries: 90,
		FailedDeliveries:     10,
		AvgLatencyMs:         150.0,
		P95LatencyMs:         300.0,
		P99LatencyMs:         500.0,
	}, nil)

	mockRepo.On("UpsertHourlyMetric", ctx, mock.MatchedBy(func(m *models.HourlyMetric) bool {
		return m.TenantID == tenantID &&
			m.TotalDeliveries == 100 &&
			m.SuccessfulDeliveries == 90 &&
			m.FailedDeliveries == 10
	})).Return(nil)

	err := agg.aggregateHourlyMetricsForTenant(ctx, tenantID, nil, hourStart, hourEnd)
	require.NoError(t, err)

	mockRepo.AssertCalled(t, "UpsertHourlyMetric", ctx, mock.Anything)
}

func TestAggregateHourlyMetrics_WithEndpointID(t *testing.T) {
	mockRepo := &MockAnalyticsRepository{}
	logger := newTestLogger()
	agg := NewAggregator(mockRepo, logger)
	ctx := context.Background()

	tenantID := uuid.New()
	endpointID := uuid.New()
	hourStart := time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC)
	hourEnd := hourStart.Add(time.Hour)

	mockRepo.On("GetMetricsSummary", ctx, mock.MatchedBy(func(q *models.MetricsQuery) bool {
		return len(q.EndpointIDs) == 1 && q.EndpointIDs[0] == endpointID
	})).Return(&models.MetricsSummary{
		TotalDeliveries:      50,
		SuccessfulDeliveries: 45,
		FailedDeliveries:     5,
		AvgLatencyMs:         100.0,
		P95LatencyMs:         200.0,
		P99LatencyMs:         400.0,
	}, nil)

	mockRepo.On("UpsertHourlyMetric", ctx, mock.MatchedBy(func(m *models.HourlyMetric) bool {
		return m.EndpointID != nil && *m.EndpointID == endpointID
	})).Return(nil)

	err := agg.aggregateHourlyMetricsForTenant(ctx, tenantID, &endpointID, hourStart, hourEnd)
	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAggregateHourlyMetrics_RepoError(t *testing.T) {
	mockRepo := &MockAnalyticsRepository{}
	logger := newTestLogger()
	agg := NewAggregator(mockRepo, logger)
	ctx := context.Background()

	tenantID := uuid.New()
	hourStart := time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC)
	hourEnd := hourStart.Add(time.Hour)

	mockRepo.On("GetMetricsSummary", ctx, mock.Anything).Return(
		(*models.MetricsSummary)(nil), errors.New("db error"),
	)

	err := agg.aggregateHourlyMetricsForTenant(ctx, tenantID, nil, hourStart, hourEnd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get metrics summary")
}

func TestAggregateHourlyMetrics_UpsertError(t *testing.T) {
	mockRepo := &MockAnalyticsRepository{}
	logger := newTestLogger()
	agg := NewAggregator(mockRepo, logger)
	ctx := context.Background()

	tenantID := uuid.New()
	hourStart := time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC)
	hourEnd := hourStart.Add(time.Hour)

	mockRepo.On("GetMetricsSummary", ctx, mock.Anything).Return(&models.MetricsSummary{
		TotalDeliveries:      10,
		SuccessfulDeliveries: 10,
		AvgLatencyMs:         100.0,
		P95LatencyMs:         200.0,
		P99LatencyMs:         300.0,
	}, nil)
	mockRepo.On("UpsertHourlyMetric", ctx, mock.Anything).Return(errors.New("upsert failed"))

	err := agg.aggregateHourlyMetricsForTenant(ctx, tenantID, nil, hourStart, hourEnd)
	require.Error(t, err)
}

// =====================
// generateRealtimeMetrics
// =====================

func TestGenerateRealtimeMetrics_NoActiveTenants(t *testing.T) {
	mockRepo := &MockAnalyticsRepository{}
	logger := newTestLogger()
	agg := NewAggregator(mockRepo, logger)
	ctx := context.Background()

	// getActiveTenantsInPeriod returns empty slice by default
	agg.generateRealtimeMetrics(ctx)
	// Should not call RecordRealtimeMetric
	mockRepo.AssertNotCalled(t, "RecordRealtimeMetric", mock.Anything, mock.Anything)
}

// =====================
// NewAggregator
// =====================

func TestNewAggregator(t *testing.T) {
	mockRepo := &MockAnalyticsRepository{}
	logger := newTestLogger()

	agg := NewAggregator(mockRepo, logger)
	require.NotNil(t, agg)
	assert.NotNil(t, agg.stopCh)
	assert.NotNil(t, agg.analyticsRepo)
	assert.NotNil(t, agg.logger)
}

// =====================
// performCleanup
// =====================

func TestPerformCleanup_Success(t *testing.T) {
	mockRepo := &MockAnalyticsRepository{}
	logger := newTestLogger()
	agg := NewAggregator(mockRepo, logger)
	ctx := context.Background()

	mockRepo.On("CleanupOldMetrics", ctx, 30).Return(nil)

	agg.performCleanup(ctx)

	mockRepo.AssertCalled(t, "CleanupOldMetrics", ctx, 30)
}

func TestPerformCleanup_Error(t *testing.T) {
	mockRepo := &MockAnalyticsRepository{}
	logger := newTestLogger()
	agg := NewAggregator(mockRepo, logger)
	ctx := context.Background()

	mockRepo.On("CleanupOldMetrics", ctx, 30).Return(errors.New("cleanup failed"))

	// Should not panic on error
	agg.performCleanup(ctx)
}
