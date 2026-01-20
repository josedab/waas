package analytics

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"webhook-platform/pkg/models"
	"webhook-platform/pkg/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockAnalyticsRepository is a mock implementation of the analytics repository
type MockAnalyticsRepository struct {
	mock.Mock
}

// Ensure MockAnalyticsRepository implements the interface
var _ repository.AnalyticsRepositoryInterface = (*MockAnalyticsRepository)(nil)

func (m *MockAnalyticsRepository) RecordDeliveryMetric(ctx context.Context, metric *models.DeliveryMetric) error {
	args := m.Called(ctx, metric)
	return args.Error(0)
}

func (m *MockAnalyticsRepository) GetDeliveryMetrics(ctx context.Context, query *models.MetricsQuery) ([]models.DeliveryMetric, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]models.DeliveryMetric), args.Error(1)
}

func (m *MockAnalyticsRepository) GetHourlyMetrics(ctx context.Context, query *models.MetricsQuery) ([]models.HourlyMetric, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]models.HourlyMetric), args.Error(1)
}

func (m *MockAnalyticsRepository) UpsertHourlyMetric(ctx context.Context, metric *models.HourlyMetric) error {
	args := m.Called(ctx, metric)
	return args.Error(0)
}

func (m *MockAnalyticsRepository) RecordRealtimeMetric(ctx context.Context, metric *models.RealtimeMetric) error {
	args := m.Called(ctx, metric)
	return args.Error(0)
}

func (m *MockAnalyticsRepository) GetRealtimeMetrics(ctx context.Context, tenantID uuid.UUID, metricType string, since time.Time) ([]models.RealtimeMetric, error) {
	args := m.Called(ctx, tenantID, metricType, since)
	return args.Get(0).([]models.RealtimeMetric), args.Error(1)
}

func (m *MockAnalyticsRepository) GetDashboardMetrics(ctx context.Context, tenantID uuid.UUID, timeWindow time.Duration) (*models.DashboardMetrics, error) {
	args := m.Called(ctx, tenantID, timeWindow)
	return args.Get(0).(*models.DashboardMetrics), args.Error(1)
}

func (m *MockAnalyticsRepository) GetMetricsSummary(ctx context.Context, query *models.MetricsQuery) (*models.MetricsSummary, error) {
	args := m.Called(ctx, query)
	return args.Get(0).(*models.MetricsSummary), args.Error(1)
}

func (m *MockAnalyticsRepository) CleanupOldMetrics(ctx context.Context, retentionDays int) error {
	args := m.Called(ctx, retentionDays)
	return args.Error(0)
}

func setupTestRouter(mockRepo *MockAnalyticsRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// Add middleware to set tenant_id
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", "550e8400-e29b-41d4-a716-446655440000")
		c.Next()
	})
	
	handlers := NewHandlers(mockRepo)
	handlers.RegisterRoutes(router)
	
	return router
}

func TestHandlers_GetDashboard(t *testing.T) {
	mockRepo := &MockAnalyticsRepository{}
	router := setupTestRouter(mockRepo)

	tenantID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	expectedDashboard := &models.DashboardMetrics{
		DeliveryRate:    15.5,
		SuccessRate:     95.2,
		AvgLatency:      250.0,
		ActiveEndpoints: 5,
		QueueSize:       10,
		RecentAlerts:    []models.AlertHistory{},
		TopEndpoints:    []models.EndpointMetrics{},
	}

	mockRepo.On("GetDashboardMetrics", mock.Anything, tenantID, 24*time.Hour).Return(expectedDashboard, nil)

	req := httptest.NewRequest("GET", "/analytics/dashboard", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.DashboardMetrics
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, expectedDashboard.DeliveryRate, response.DeliveryRate)
	assert.Equal(t, expectedDashboard.SuccessRate, response.SuccessRate)
	assert.Equal(t, expectedDashboard.AvgLatency, response.AvgLatency)

	mockRepo.AssertExpectations(t)
}

func TestHandlers_GetDashboardWithCustomWindow(t *testing.T) {
	mockRepo := &MockAnalyticsRepository{}
	router := setupTestRouter(mockRepo)

	tenantID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	expectedDashboard := &models.DashboardMetrics{
		DeliveryRate: 20.0,
		SuccessRate:  98.0,
	}

	mockRepo.On("GetDashboardMetrics", mock.Anything, tenantID, 12*time.Hour).Return(expectedDashboard, nil)

	req := httptest.NewRequest("GET", "/analytics/dashboard?window=12h", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandlers_GetMetrics(t *testing.T) {
	mockRepo := &MockAnalyticsRepository{}
	router := setupTestRouter(mockRepo)

	tenantID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	endpointID := uuid.New()
	
	expectedMetrics := []models.DeliveryMetric{
		{
			ID:            uuid.New(),
			TenantID:      tenantID,
			EndpointID:    endpointID,
			Status:        "success",
			LatencyMs:     200,
			AttemptNumber: 1,
		},
	}

	expectedSummary := &models.MetricsSummary{
		TotalDeliveries:      1,
		SuccessfulDeliveries: 1,
		FailedDeliveries:     0,
		SuccessRate:          100.0,
		AvgLatencyMs:         200.0,
	}

	mockRepo.On("GetDeliveryMetrics", mock.Anything, mock.MatchedBy(func(query *models.MetricsQuery) bool {
		return query.TenantID == tenantID
	})).Return(expectedMetrics, nil)

	mockRepo.On("GetMetricsSummary", mock.Anything, mock.MatchedBy(func(query *models.MetricsQuery) bool {
		return query.TenantID == tenantID
	})).Return(expectedSummary, nil)

	req := httptest.NewRequest("GET", "/analytics/metrics", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.MetricsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, 1, response.TotalCount)
	assert.Len(t, response.Data, 1)
	assert.Equal(t, expectedSummary.SuccessRate, response.Summary.SuccessRate)

	mockRepo.AssertExpectations(t)
}

func TestHandlers_GetMetricsWithFilters(t *testing.T) {
	mockRepo := &MockAnalyticsRepository{}
	router := setupTestRouter(mockRepo)

	tenantID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	endpointID := uuid.New()

	mockRepo.On("GetDeliveryMetrics", mock.Anything, mock.MatchedBy(func(query *models.MetricsQuery) bool {
		return query.TenantID == tenantID &&
			len(query.EndpointIDs) == 1 &&
			query.EndpointIDs[0] == endpointID &&
			len(query.Statuses) == 1 &&
			query.Statuses[0] == "success"
	})).Return([]models.DeliveryMetric{}, nil)

	mockRepo.On("GetMetricsSummary", mock.Anything, mock.AnythingOfType("*models.MetricsQuery")).Return(&models.MetricsSummary{}, nil)

	req := httptest.NewRequest("GET", "/analytics/metrics?endpoint_ids="+endpointID.String()+"&statuses=success", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandlers_GetHourlyMetrics(t *testing.T) {
	mockRepo := &MockAnalyticsRepository{}
	router := setupTestRouter(mockRepo)

	tenantID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	
	expectedMetrics := []models.HourlyMetric{
		{
			ID:                   uuid.New(),
			TenantID:             tenantID,
			HourTimestamp:        time.Now().Truncate(time.Hour),
			TotalDeliveries:      100,
			SuccessfulDeliveries: 95,
			FailedDeliveries:     5,
		},
	}

	mockRepo.On("GetHourlyMetrics", mock.Anything, mock.MatchedBy(func(query *models.MetricsQuery) bool {
		return query.TenantID == tenantID
	})).Return(expectedMetrics, nil)

	req := httptest.NewRequest("GET", "/analytics/metrics/hourly", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.MetricsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, 1, response.TotalCount)
	assert.Len(t, response.Data, 1)

	mockRepo.AssertExpectations(t)
}

func TestHandlers_GetRealtimeMetrics(t *testing.T) {
	mockRepo := &MockAnalyticsRepository{}
	router := setupTestRouter(mockRepo)

	tenantID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	
	expectedMetrics := []models.RealtimeMetric{
		{
			ID:          uuid.New(),
			TenantID:    tenantID,
			MetricType:  "delivery_rate",
			MetricValue: 15.5,
			Timestamp:   time.Now(),
		},
	}

	mockRepo.On("GetRealtimeMetrics", mock.Anything, tenantID, "delivery_rate", mock.AnythingOfType("time.Time")).Return(expectedMetrics, nil)

	req := httptest.NewRequest("GET", "/analytics/metrics/realtime", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "delivery_rate", response["type"])
	assert.NotNil(t, response["metrics"])

	mockRepo.AssertExpectations(t)
}

func TestHandlers_GetEndpointMetrics(t *testing.T) {
	mockRepo := &MockAnalyticsRepository{}
	router := setupTestRouter(mockRepo)

	tenantID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	endpointID := uuid.New()
	
	expectedMetrics := []models.DeliveryMetric{
		{
			ID:         uuid.New(),
			TenantID:   tenantID,
			EndpointID: endpointID,
			Status:     "success",
			LatencyMs:  200,
		},
	}

	expectedSummary := &models.MetricsSummary{
		TotalDeliveries:      1,
		SuccessfulDeliveries: 1,
		SuccessRate:          100.0,
	}

	mockRepo.On("GetDeliveryMetrics", mock.Anything, mock.MatchedBy(func(query *models.MetricsQuery) bool {
		return query.TenantID == tenantID &&
			len(query.EndpointIDs) == 1 &&
			query.EndpointIDs[0] == endpointID
	})).Return(expectedMetrics, nil)

	mockRepo.On("GetMetricsSummary", mock.Anything, mock.AnythingOfType("*models.MetricsQuery")).Return(expectedSummary, nil)

	req := httptest.NewRequest("GET", "/analytics/endpoints/"+endpointID.String()+"/metrics", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.MetricsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, 1, response.TotalCount)
	assert.Equal(t, 100.0, response.Summary.SuccessRate)

	mockRepo.AssertExpectations(t)
}

func TestHandlers_GetMetricsSummary(t *testing.T) {
	mockRepo := &MockAnalyticsRepository{}
	router := setupTestRouter(mockRepo)

	tenantID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	
	expectedSummary := &models.MetricsSummary{
		TotalDeliveries:      1000,
		SuccessfulDeliveries: 950,
		FailedDeliveries:     50,
		SuccessRate:          95.0,
		AvgLatencyMs:         250.0,
		P95LatencyMs:         500.0,
		P99LatencyMs:         750.0,
	}

	mockRepo.On("GetMetricsSummary", mock.Anything, mock.MatchedBy(func(query *models.MetricsQuery) bool {
		return query.TenantID == tenantID
	})).Return(expectedSummary, nil)

	req := httptest.NewRequest("GET", "/analytics/metrics/summary", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.MetricsSummary
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, expectedSummary.TotalDeliveries, response.TotalDeliveries)
	assert.Equal(t, expectedSummary.SuccessRate, response.SuccessRate)
	assert.Equal(t, expectedSummary.AvgLatencyMs, response.AvgLatencyMs)

	mockRepo.AssertExpectations(t)
}

func TestHandlers_InvalidTenant(t *testing.T) {
	mockRepo := &MockAnalyticsRepository{}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// Don't set tenant_id in context
	handlers := NewHandlers(mockRepo)
	handlers.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/analytics/dashboard", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandlers_InvalidEndpointID(t *testing.T) {
	mockRepo := &MockAnalyticsRepository{}
	router := setupTestRouter(mockRepo)

	req := httptest.NewRequest("GET", "/analytics/endpoints/invalid-uuid/metrics", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandlers_ParseMetricsQueryWithDates(t *testing.T) {
	mockRepo := &MockAnalyticsRepository{}
	router := setupTestRouter(mockRepo)

	startDate := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	endDate := time.Now().UTC().Format(time.RFC3339)

	mockRepo.On("GetDeliveryMetrics", mock.Anything, mock.MatchedBy(func(query *models.MetricsQuery) bool {
		return !query.StartDate.IsZero() && !query.EndDate.IsZero()
	})).Return([]models.DeliveryMetric{}, nil)

	mockRepo.On("GetMetricsSummary", mock.Anything, mock.AnythingOfType("*models.MetricsQuery")).Return(&models.MetricsSummary{}, nil)

	req := httptest.NewRequest("GET", "/analytics/metrics?start_date="+startDate+"&end_date="+endDate, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Logf("Response body: %s", w.Body.String())
	}
	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}