package repository

import (
	"context"
	"testing"
	"time"
	"github.com/josedab/waas/pkg/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyticsRepository_RecordDeliveryMetric(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAnalyticsRepository(db.Pool)
	ctx := context.Background()

	// Create test tenant and endpoint
	tenantID := uuid.New()
	endpointID := uuid.New()
	deliveryID := uuid.New()

	metric := &models.DeliveryMetric{
		TenantID:      tenantID,
		EndpointID:    endpointID,
		DeliveryID:    deliveryID,
		Status:        "success",
		HTTPStatus:    &[]int{200}[0],
		LatencyMs:     250,
		AttemptNumber: 1,
	}

	err := repo.RecordDeliveryMetric(ctx, metric)
	require.NoError(t, err)

	// Verify the metric was recorded
	assert.NotEqual(t, uuid.Nil, metric.ID)
	assert.False(t, metric.CreatedAt.IsZero())
}

func TestAnalyticsRepository_GetDeliveryMetrics(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAnalyticsRepository(db.Pool)
	ctx := context.Background()

	// Create test data
	tenantID := uuid.New()
	endpointID1 := uuid.New()
	endpointID2 := uuid.New()

	// Record multiple metrics
	metrics := []*models.DeliveryMetric{
		{
			TenantID:      tenantID,
			EndpointID:    endpointID1,
			DeliveryID:    uuid.New(),
			Status:        "success",
			HTTPStatus:    &[]int{200}[0],
			LatencyMs:     200,
			AttemptNumber: 1,
		},
		{
			TenantID:      tenantID,
			EndpointID:    endpointID2,
			DeliveryID:    uuid.New(),
			Status:        "failed",
			HTTPStatus:    &[]int{500}[0],
			LatencyMs:     1000,
			AttemptNumber: 1,
			ErrorMessage:  &[]string{"Internal server error"}[0],
		},
		{
			TenantID:      tenantID,
			EndpointID:    endpointID1,
			DeliveryID:    uuid.New(),
			Status:        "success",
			HTTPStatus:    &[]int{200}[0],
			LatencyMs:     150,
			AttemptNumber: 2,
		},
	}

	for _, metric := range metrics {
		err := repo.RecordDeliveryMetric(ctx, metric)
		require.NoError(t, err)
	}

	// Test query all metrics for tenant
	query := &models.MetricsQuery{
		TenantID:  tenantID,
		StartDate: time.Now().Add(-1 * time.Hour),
		EndDate:   time.Now().Add(1 * time.Hour),
		Limit:     100,
	}

	results, err := repo.GetDeliveryMetrics(ctx, query)
	require.NoError(t, err)
	assert.Len(t, results, 3)

	// Test query with endpoint filter
	query.EndpointIDs = []uuid.UUID{endpointID1}
	results, err = repo.GetDeliveryMetrics(ctx, query)
	require.NoError(t, err)
	assert.Len(t, results, 2)

	// Test query with status filter
	query.EndpointIDs = nil
	query.Statuses = []string{"success"}
	results, err = repo.GetDeliveryMetrics(ctx, query)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestAnalyticsRepository_UpsertHourlyMetric(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAnalyticsRepository(db.Pool)
	ctx := context.Background()

	tenantID := uuid.New()
	endpointID := uuid.New()
	hourTimestamp := time.Now().Truncate(time.Hour)

	metric := &models.HourlyMetric{
		TenantID:             tenantID,
		EndpointID:           &endpointID,
		HourTimestamp:        hourTimestamp,
		TotalDeliveries:      100,
		SuccessfulDeliveries: 95,
		FailedDeliveries:     5,
		RetryingDeliveries:   0,
		AvgLatencyMs:         &[]float64{250.5}[0],
		P95LatencyMs:         &[]float64{500.0}[0],
		P99LatencyMs:         &[]float64{750.0}[0],
		TotalRetries:         10,
	}

	// Insert new metric
	err := repo.UpsertHourlyMetric(ctx, metric)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, metric.ID)

	// Update existing metric
	metric.TotalDeliveries = 120
	metric.SuccessfulDeliveries = 110
	err = repo.UpsertHourlyMetric(ctx, metric)
	require.NoError(t, err)

	// Verify the update
	query := &models.MetricsQuery{
		TenantID:    tenantID,
		EndpointIDs: []uuid.UUID{endpointID},
		StartDate:   hourTimestamp.Add(-1 * time.Minute),
		EndDate:     hourTimestamp.Add(1 * time.Minute),
	}

	results, err := repo.GetHourlyMetrics(ctx, query)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, 120, results[0].TotalDeliveries)
	assert.Equal(t, 110, results[0].SuccessfulDeliveries)
}

func TestAnalyticsRepository_GetHourlyMetrics(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAnalyticsRepository(db.Pool)
	ctx := context.Background()

	tenantID := uuid.New()
	endpointID := uuid.New()

	// Create hourly metrics for different hours
	baseTime := time.Now().Truncate(time.Hour)
	for i := 0; i < 3; i++ {
		hourTimestamp := baseTime.Add(time.Duration(-i) * time.Hour)
		metric := &models.HourlyMetric{
			TenantID:             tenantID,
			EndpointID:           &endpointID,
			HourTimestamp:        hourTimestamp,
			TotalDeliveries:      100 + i*10,
			SuccessfulDeliveries: 90 + i*10,
			FailedDeliveries:     10,
			RetryingDeliveries:   0,
			AvgLatencyMs:         &[]float64{250.0 + float64(i)*10}[0],
		}

		err := repo.UpsertHourlyMetric(ctx, metric)
		require.NoError(t, err)
	}

	// Query hourly metrics
	query := &models.MetricsQuery{
		TenantID:    tenantID,
		EndpointIDs: []uuid.UUID{endpointID},
		StartDate:   baseTime.Add(-3 * time.Hour),
		EndDate:     baseTime.Add(1 * time.Hour),
		Limit:       10,
	}

	results, err := repo.GetHourlyMetrics(ctx, query)
	require.NoError(t, err)
	assert.Len(t, results, 3)

	// Verify results are ordered by timestamp DESC
	for i := 0; i < len(results)-1; i++ {
		assert.True(t, results[i].HourTimestamp.After(results[i+1].HourTimestamp))
	}
}

func TestAnalyticsRepository_RecordRealtimeMetric(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAnalyticsRepository(db.Pool)
	ctx := context.Background()

	tenantID := uuid.New()

	metric := &models.RealtimeMetric{
		TenantID:    tenantID,
		MetricType:  "delivery_rate",
		MetricValue: 15.5,
		Metadata: map[string]interface{}{
			"window_minutes": 5,
			"source":         "aggregator",
		},
	}

	err := repo.RecordRealtimeMetric(ctx, metric)
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, metric.ID)
	assert.False(t, metric.Timestamp.IsZero())
}

func TestAnalyticsRepository_GetRealtimeMetrics(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAnalyticsRepository(db.Pool)
	ctx := context.Background()

	tenantID := uuid.New()
	metricType := "delivery_rate"

	// Record multiple real-time metrics
	for i := 0; i < 5; i++ {
		metric := &models.RealtimeMetric{
			TenantID:    tenantID,
			MetricType:  metricType,
			MetricValue: float64(10 + i),
		}

		err := repo.RecordRealtimeMetric(ctx, metric)
		require.NoError(t, err)

		// Small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)
	}

	// Query real-time metrics
	since := time.Now().Add(-1 * time.Minute)
	results, err := repo.GetRealtimeMetrics(ctx, tenantID, metricType, since)
	require.NoError(t, err)
	assert.Len(t, results, 5)

	// Verify results are ordered by timestamp DESC
	for i := 0; i < len(results)-1; i++ {
		assert.True(t, results[i].Timestamp.After(results[i+1].Timestamp) || 
			results[i].Timestamp.Equal(results[i+1].Timestamp))
	}
}

func TestAnalyticsRepository_GetDashboardMetrics(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAnalyticsRepository(db.Pool)
	ctx := context.Background()

	tenantID := uuid.New()
	endpointID := uuid.New()

	// Record some delivery metrics
	metrics := []*models.DeliveryMetric{
		{
			TenantID:      tenantID,
			EndpointID:    endpointID,
			DeliveryID:    uuid.New(),
			Status:        "success",
			HTTPStatus:    &[]int{200}[0],
			LatencyMs:     200,
			AttemptNumber: 1,
		},
		{
			TenantID:      tenantID,
			EndpointID:    endpointID,
			DeliveryID:    uuid.New(),
			Status:        "success",
			HTTPStatus:    &[]int{200}[0],
			LatencyMs:     300,
			AttemptNumber: 1,
		},
		{
			TenantID:      tenantID,
			EndpointID:    endpointID,
			DeliveryID:    uuid.New(),
			Status:        "failed",
			HTTPStatus:    &[]int{500}[0],
			LatencyMs:     1000,
			AttemptNumber: 1,
		},
	}

	for _, metric := range metrics {
		err := repo.RecordDeliveryMetric(ctx, metric)
		require.NoError(t, err)
	}

	// Get dashboard metrics
	dashboard, err := repo.GetDashboardMetrics(ctx, tenantID, time.Hour)
	require.NoError(t, err)

	assert.NotNil(t, dashboard)
	assert.Equal(t, float64(66.67), dashboard.SuccessRate) // 2/3 * 100, rounded
	assert.Equal(t, float64(250), dashboard.AvgLatency)    // (200+300)/2
	assert.Equal(t, 1, dashboard.ActiveEndpoints)
}

func TestAnalyticsRepository_GetMetricsSummary(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAnalyticsRepository(db.Pool)
	ctx := context.Background()

	tenantID := uuid.New()
	endpointID := uuid.New()

	// Record test metrics with different statuses and latencies
	testData := []struct {
		status    string
		latencyMs int
	}{
		{"success", 100},
		{"success", 200},
		{"success", 300},
		{"success", 400},
		{"success", 500},
		{"failed", 1000},
		{"failed", 2000},
	}

	for _, data := range testData {
		metric := &models.DeliveryMetric{
			TenantID:      tenantID,
			EndpointID:    endpointID,
			DeliveryID:    uuid.New(),
			Status:        data.status,
			LatencyMs:     data.latencyMs,
			AttemptNumber: 1,
		}

		err := repo.RecordDeliveryMetric(ctx, metric)
		require.NoError(t, err)
	}

	// Get summary
	query := &models.MetricsQuery{
		TenantID:  tenantID,
		StartDate: time.Now().Add(-1 * time.Hour),
		EndDate:   time.Now().Add(1 * time.Hour),
	}

	summary, err := repo.GetMetricsSummary(ctx, query)
	require.NoError(t, err)

	assert.Equal(t, 7, summary.TotalDeliveries)
	assert.Equal(t, 5, summary.SuccessfulDeliveries)
	assert.Equal(t, 2, summary.FailedDeliveries)
	assert.InDelta(t, 71.43, summary.SuccessRate, 0.01) // 5/7 * 100
	assert.Equal(t, float64(300), summary.AvgLatencyMs)  // (100+200+300+400+500)/5
}

func TestAnalyticsRepository_CleanupOldMetrics(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAnalyticsRepository(db.Pool)
	ctx := context.Background()

	tenantID := uuid.New()
	endpointID := uuid.New()

	// Record old metric (35 days ago)
	oldMetric := &models.DeliveryMetric{
		TenantID:      tenantID,
		EndpointID:    endpointID,
		DeliveryID:    uuid.New(),
		Status:        "success",
		LatencyMs:     200,
		AttemptNumber: 1,
	}

	err := repo.RecordDeliveryMetric(ctx, oldMetric)
	require.NoError(t, err)

	// Manually update the created_at to be old
	_, err = db.Pool.Exec(ctx, 
		"UPDATE delivery_metrics SET created_at = $1 WHERE id = $2",
		time.Now().AddDate(0, 0, -35), oldMetric.ID)
	require.NoError(t, err)

	// Record recent metric
	recentMetric := &models.DeliveryMetric{
		TenantID:      tenantID,
		EndpointID:    endpointID,
		DeliveryID:    uuid.New(),
		Status:        "success",
		LatencyMs:     200,
		AttemptNumber: 1,
	}

	err = repo.RecordDeliveryMetric(ctx, recentMetric)
	require.NoError(t, err)

	// Cleanup old metrics (30 day retention)
	err = repo.CleanupOldMetrics(ctx, 30)
	require.NoError(t, err)

	// Verify old metric was deleted and recent metric remains
	query := &models.MetricsQuery{
		TenantID:  tenantID,
		StartDate: time.Now().AddDate(0, 0, -40),
		EndDate:   time.Now().Add(1 * time.Hour),
		Limit:     100,
	}

	results, err := repo.GetDeliveryMetrics(ctx, query)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, recentMetric.ID, results[0].ID)
}