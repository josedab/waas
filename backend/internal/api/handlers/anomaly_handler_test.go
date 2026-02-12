package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/josedab/waas/pkg/anomaly"
	"github.com/josedab/waas/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Mock Repository ---

type mockAnomalyRepo struct {
	mock.Mock
}

func (m *mockAnomalyRepo) GetBaseline(ctx context.Context, tenantID, endpointID string, metricType anomaly.MetricType) (*anomaly.Baseline, error) {
	args := m.Called(ctx, tenantID, endpointID, metricType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*anomaly.Baseline), args.Error(1)
}

func (m *mockAnomalyRepo) SaveBaseline(ctx context.Context, baseline *anomaly.Baseline) error {
	args := m.Called(ctx, baseline)
	return args.Error(0)
}

func (m *mockAnomalyRepo) SaveAnomaly(ctx context.Context, a *anomaly.Anomaly) error {
	args := m.Called(ctx, a)
	return args.Error(0)
}

func (m *mockAnomalyRepo) GetAnomaly(ctx context.Context, tenantID, anomalyID string) (*anomaly.Anomaly, error) {
	args := m.Called(ctx, tenantID, anomalyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*anomaly.Anomaly), args.Error(1)
}

func (m *mockAnomalyRepo) ListAnomalies(ctx context.Context, tenantID string, status string, limit, offset int) ([]anomaly.Anomaly, int, error) {
	args := m.Called(ctx, tenantID, status, limit, offset)
	return args.Get(0).([]anomaly.Anomaly), args.Int(1), args.Error(2)
}

func (m *mockAnomalyRepo) UpdateAnomalyStatus(ctx context.Context, tenantID, anomalyID, status string) error {
	args := m.Called(ctx, tenantID, anomalyID, status)
	return args.Error(0)
}

func (m *mockAnomalyRepo) GetDetectionConfig(ctx context.Context, tenantID, endpointID string, metricType anomaly.MetricType) (*anomaly.DetectionConfig, error) {
	args := m.Called(ctx, tenantID, endpointID, metricType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*anomaly.DetectionConfig), args.Error(1)
}

func (m *mockAnomalyRepo) SaveDetectionConfig(ctx context.Context, config *anomaly.DetectionConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *mockAnomalyRepo) ListDetectionConfigs(ctx context.Context, tenantID string) ([]anomaly.DetectionConfig, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]anomaly.DetectionConfig), args.Error(1)
}

func (m *mockAnomalyRepo) GetAlertConfigs(ctx context.Context, tenantID string) ([]anomaly.AlertConfig, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]anomaly.AlertConfig), args.Error(1)
}

func (m *mockAnomalyRepo) SaveAlertConfig(ctx context.Context, config *anomaly.AlertConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *mockAnomalyRepo) SaveAlert(ctx context.Context, alert *anomaly.AnomalyAlert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *mockAnomalyRepo) ListAlerts(ctx context.Context, tenantID string, anomalyID string, limit, offset int) ([]anomaly.AnomalyAlert, int, error) {
	args := m.Called(ctx, tenantID, anomalyID, limit, offset)
	return args.Get(0).([]anomaly.AnomalyAlert), args.Int(1), args.Error(2)
}

func (m *mockAnomalyRepo) GetRecentMetrics(ctx context.Context, tenantID, endpointID string, metricType anomaly.MetricType, duration time.Duration) ([]anomaly.MetricDataPoint, error) {
	args := m.Called(ctx, tenantID, endpointID, metricType, duration)
	return args.Get(0).([]anomaly.MetricDataPoint), args.Error(1)
}

// --- Mock AlertNotifier ---

type mockAlertNotifier struct {
	mock.Mock
}

func (m *mockAlertNotifier) Send(ctx context.Context, alert *anomaly.AnomalyAlert, a *anomaly.Anomaly, config *anomaly.AlertConfig) error {
	args := m.Called(ctx, alert, a, config)
	return args.Error(0)
}

// --- Helpers ---

func setupAnomalyRouter(tenantID string, handler *AnomalyHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/")
	if tenantID != "" {
		g.Use(func(c *gin.Context) {
			c.Set("tenant_id", tenantID)
			c.Next()
		})
	}
	RegisterAnomalyRoutes(g, handler)
	return r
}

func newAnomalyHandler(repo anomaly.Repository) *AnomalyHandler {
	svc := anomaly.NewService(repo, nil)
	logger := utils.NewTestLogger()
	return NewAnomalyHandler(svc, logger)
}

// --- ListAnomalies tests ---

func TestListAnomalies_Success(t *testing.T) {
	repo := new(mockAnomalyRepo)
	handler := newAnomalyHandler(repo)
	router := setupAnomalyRouter("tenant-1", handler)

	expected := []anomaly.Anomaly{
		{ID: "a1", TenantID: "tenant-1", Status: "open", Severity: anomaly.SeverityWarning},
		{ID: "a2", TenantID: "tenant-1", Status: "resolved", Severity: anomaly.SeverityCritical},
	}
	repo.On("ListAnomalies", mock.Anything, "tenant-1", "", 50, 0).Return(expected, 2, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/anomalies", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []anomaly.Anomaly
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Len(t, result, 2)
	assert.Equal(t, "a1", result[0].ID)
	repo.AssertExpectations(t)
}

func TestListAnomalies_MissingTenantID(t *testing.T) {
	repo := new(mockAnomalyRepo)
	handler := newAnomalyHandler(repo)
	router := setupAnomalyRouter("", handler) // no tenant_id

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/anomalies", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "unauthorized")
	repo.AssertNotCalled(t, "ListAnomalies", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// --- GetBaselines tests ---

func TestGetBaselines_Success(t *testing.T) {
	repo := new(mockAnomalyRepo)
	handler := newAnomalyHandler(repo)
	router := setupAnomalyRouter("tenant-1", handler)

	baseline := &anomaly.Baseline{
		ID: "b1", TenantID: "tenant-1", EndpointID: "ep1",
		MetricType: anomaly.MetricTypeErrorRate, Mean: 5.0, StdDev: 1.0, SampleSize: 100,
	}

	// GetBaselines iterates over 4 metric types
	repo.On("GetBaseline", mock.Anything, "tenant-1", "ep1", anomaly.MetricTypeErrorRate).Return(baseline, nil)
	repo.On("GetBaseline", mock.Anything, "tenant-1", "ep1", anomaly.MetricTypeLatencyP95).Return(nil, nil)
	repo.On("GetBaseline", mock.Anything, "tenant-1", "ep1", anomaly.MetricTypeDeliveryRate).Return(nil, nil)
	repo.On("GetBaseline", mock.Anything, "tenant-1", "ep1", anomaly.MetricTypeRetryRate).Return(nil, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/anomalies/baselines?endpoint_id=ep1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var baselines []anomaly.Baseline
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &baselines))
	assert.Len(t, baselines, 1)
	assert.Equal(t, "b1", baselines[0].ID)
	repo.AssertExpectations(t)
}

func TestGetBaselines_MissingTenantID(t *testing.T) {
	repo := new(mockAnomalyRepo)
	handler := newAnomalyHandler(repo)
	router := setupAnomalyRouter("", handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/anomalies/baselines", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "unauthorized")
}

// --- GetDetectionConfig tests ---

func TestGetDetectionConfig_Success(t *testing.T) {
	repo := new(mockAnomalyRepo)
	handler := newAnomalyHandler(repo)
	router := setupAnomalyRouter("tenant-1", handler)

	config := &anomaly.DetectionConfig{
		ID: "dc1", TenantID: "tenant-1", EndpointID: "ep1",
		MetricType: anomaly.MetricTypeErrorRate, Enabled: true,
		Sensitivity: 1.0, MinSamples: 30, CooldownMinutes: 15,
		CriticalThreshold: 3.0, WarningThreshold: 2.0,
	}
	repo.On("GetDetectionConfig", mock.Anything, "tenant-1", "ep1", anomaly.MetricType("error_rate")).Return(config, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/anomalies/config?endpoint_id=ep1&metric_type=error_rate", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result anomaly.DetectionConfig
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Equal(t, "dc1", result.ID)
	assert.True(t, result.Enabled)
	repo.AssertExpectations(t)
}

func TestGetDetectionConfig_MissingTenantID(t *testing.T) {
	repo := new(mockAnomalyRepo)
	handler := newAnomalyHandler(repo)
	router := setupAnomalyRouter("", handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/anomalies/config", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "unauthorized")
}

// --- GetAnomalyStats tests ---

func TestGetAnomalyStats_Success(t *testing.T) {
	repo := new(mockAnomalyRepo)
	handler := newAnomalyHandler(repo)
	router := setupAnomalyRouter("tenant-1", handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/anomalies/stats", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var stats map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &stats))
	assert.Equal(t, float64(45), stats["total_anomalies"])
	assert.Equal(t, float64(5), stats["open"])
}

func TestGetAnomalyStats_MissingTenantID(t *testing.T) {
	repo := new(mockAnomalyRepo)
	handler := newAnomalyHandler(repo)
	router := setupAnomalyRouter("", handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/anomalies/stats", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- ListAlertConfigs tests ---

func TestListAlertConfigs_Success(t *testing.T) {
	repo := new(mockAnomalyRepo)
	handler := newAnomalyHandler(repo)
	router := setupAnomalyRouter("tenant-1", handler)

	configs := []anomaly.AlertConfig{
		{ID: "ac1", TenantID: "tenant-1", Name: "Slack Alert", Channel: "slack", Enabled: true},
	}
	repo.On("GetAlertConfigs", mock.Anything, "tenant-1").Return(configs, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/anomalies/alerts/config", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []anomaly.AlertConfig
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Len(t, result, 1)
	assert.Equal(t, "Slack Alert", result[0].Name)
	repo.AssertExpectations(t)
}

func TestListAlertConfigs_MissingTenantID(t *testing.T) {
	repo := new(mockAnomalyRepo)
	handler := newAnomalyHandler(repo)
	router := setupAnomalyRouter("", handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/anomalies/alerts/config", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
