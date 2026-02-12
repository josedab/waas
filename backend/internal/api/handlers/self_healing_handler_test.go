package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/josedab/waas/internal/api/services"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/repository"
	"github.com/josedab/waas/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSelfHealingRepository implements repository.SelfHealingRepository for testing
type MockSelfHealingRepository struct {
	mock.Mock
}

func (m *MockSelfHealingRepository) CreatePrediction(ctx context.Context, pred *models.EndpointHealthPrediction) error {
	args := m.Called(ctx, pred)
	return args.Error(0)
}

func (m *MockSelfHealingRepository) GetPrediction(ctx context.Context, id uuid.UUID) (*models.EndpointHealthPrediction, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EndpointHealthPrediction), args.Error(1)
}

func (m *MockSelfHealingRepository) GetPredictionsByEndpoint(ctx context.Context, endpointID uuid.UUID, limit int) ([]*models.EndpointHealthPrediction, error) {
	args := m.Called(ctx, endpointID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.EndpointHealthPrediction), args.Error(1)
}

func (m *MockSelfHealingRepository) GetRecentPredictions(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.EndpointHealthPrediction, error) {
	args := m.Called(ctx, tenantID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.EndpointHealthPrediction), args.Error(1)
}

func (m *MockSelfHealingRepository) UpdatePredictionAccuracy(ctx context.Context, id uuid.UUID, wasAccurate bool) error {
	args := m.Called(ctx, id, wasAccurate)
	return args.Error(0)
}

func (m *MockSelfHealingRepository) UpdatePredictionAction(ctx context.Context, id uuid.UUID, action string) error {
	args := m.Called(ctx, id, action)
	return args.Error(0)
}

func (m *MockSelfHealingRepository) CreateBehaviorPattern(ctx context.Context, pattern *models.EndpointBehaviorPattern) error {
	args := m.Called(ctx, pattern)
	return args.Error(0)
}

func (m *MockSelfHealingRepository) GetBehaviorPatterns(ctx context.Context, endpointID uuid.UUID) ([]*models.EndpointBehaviorPattern, error) {
	args := m.Called(ctx, endpointID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.EndpointBehaviorPattern), args.Error(1)
}

func (m *MockSelfHealingRepository) DeleteOldPatterns(ctx context.Context, endpointID uuid.UUID, beforeTime time.Time) error {
	args := m.Called(ctx, endpointID, beforeTime)
	return args.Error(0)
}

func (m *MockSelfHealingRepository) CreateRemediationRule(ctx context.Context, rule *models.AutoRemediationRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *MockSelfHealingRepository) GetRemediationRule(ctx context.Context, id uuid.UUID) (*models.AutoRemediationRule, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AutoRemediationRule), args.Error(1)
}

func (m *MockSelfHealingRepository) GetRemediationRulesByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.AutoRemediationRule, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.AutoRemediationRule), args.Error(1)
}

func (m *MockSelfHealingRepository) GetActiveRemediationRules(ctx context.Context, tenantID uuid.UUID) ([]*models.AutoRemediationRule, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.AutoRemediationRule), args.Error(1)
}

func (m *MockSelfHealingRepository) UpdateRemediationRule(ctx context.Context, rule *models.AutoRemediationRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *MockSelfHealingRepository) IncrementRuleTriggerCount(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSelfHealingRepository) DeleteRemediationRule(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSelfHealingRepository) CreateRemediationAction(ctx context.Context, action *models.RemediationAction) error {
	args := m.Called(ctx, action)
	return args.Error(0)
}

func (m *MockSelfHealingRepository) GetRemediationAction(ctx context.Context, id uuid.UUID) (*models.RemediationAction, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RemediationAction), args.Error(1)
}

func (m *MockSelfHealingRepository) GetRemediationActionsByEndpoint(ctx context.Context, endpointID uuid.UUID, limit int) ([]*models.RemediationAction, error) {
	args := m.Called(ctx, endpointID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.RemediationAction), args.Error(1)
}

func (m *MockSelfHealingRepository) GetRecentRemediationActions(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.RemediationAction, error) {
	args := m.Called(ctx, tenantID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.RemediationAction), args.Error(1)
}

func (m *MockSelfHealingRepository) UpdateRemediationActionOutcome(ctx context.Context, id uuid.UUID, outcome string) error {
	args := m.Called(ctx, id, outcome)
	return args.Error(0)
}

func (m *MockSelfHealingRepository) RevertRemediationAction(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSelfHealingRepository) CountActionsToday(ctx context.Context, tenantID uuid.UUID) (int, error) {
	args := m.Called(ctx, tenantID)
	return args.Int(0), args.Error(1)
}

func (m *MockSelfHealingRepository) CreateSuggestion(ctx context.Context, suggestion *models.EndpointOptimizationSuggestion) error {
	args := m.Called(ctx, suggestion)
	return args.Error(0)
}

func (m *MockSelfHealingRepository) GetSuggestion(ctx context.Context, id uuid.UUID) (*models.EndpointOptimizationSuggestion, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EndpointOptimizationSuggestion), args.Error(1)
}

func (m *MockSelfHealingRepository) GetSuggestionsByEndpoint(ctx context.Context, endpointID uuid.UUID) ([]*models.EndpointOptimizationSuggestion, error) {
	args := m.Called(ctx, endpointID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.EndpointOptimizationSuggestion), args.Error(1)
}

func (m *MockSelfHealingRepository) GetPendingSuggestions(ctx context.Context, tenantID uuid.UUID) ([]*models.EndpointOptimizationSuggestion, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.EndpointOptimizationSuggestion), args.Error(1)
}

func (m *MockSelfHealingRepository) UpdateSuggestionStatus(ctx context.Context, id uuid.UUID, status string) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockSelfHealingRepository) CountPendingSuggestions(ctx context.Context, tenantID uuid.UUID) (int, error) {
	args := m.Called(ctx, tenantID)
	return args.Int(0), args.Error(1)
}

func (m *MockSelfHealingRepository) GetOrCreateCircuitBreaker(ctx context.Context, tenantID, endpointID uuid.UUID) (*models.EndpointCircuitBreaker, error) {
	args := m.Called(ctx, tenantID, endpointID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EndpointCircuitBreaker), args.Error(1)
}

func (m *MockSelfHealingRepository) UpdateCircuitBreakerState(ctx context.Context, id uuid.UUID, state string, failureCount, successCount int) error {
	args := m.Called(ctx, id, state, failureCount, successCount)
	return args.Error(0)
}

func (m *MockSelfHealingRepository) UpdateCircuitBreakerConfig(ctx context.Context, id uuid.UUID, resetTimeout, failureThreshold, successThreshold int) error {
	args := m.Called(ctx, id, resetTimeout, failureThreshold, successThreshold)
	return args.Error(0)
}

func (m *MockSelfHealingRepository) GetOpenCircuitBreakers(ctx context.Context, tenantID uuid.UUID) ([]*models.EndpointCircuitBreaker, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.EndpointCircuitBreaker), args.Error(1)
}

func (m *MockSelfHealingRepository) RecordCircuitBreakerFailure(ctx context.Context, endpointID uuid.UUID) error {
	args := m.Called(ctx, endpointID)
	return args.Error(0)
}

func (m *MockSelfHealingRepository) RecordCircuitBreakerSuccess(ctx context.Context, endpointID uuid.UUID) error {
	args := m.Called(ctx, endpointID)
	return args.Error(0)
}

func (m *MockSelfHealingRepository) CountOpenCircuitBreakers(ctx context.Context, tenantID uuid.UUID) (int, error) {
	args := m.Called(ctx, tenantID)
	return args.Int(0), args.Error(1)
}

// Verify interface compliance
var _ repository.SelfHealingRepository = (*MockSelfHealingRepository)(nil)

func setupSelfHealingHandler(mockRepo *MockSelfHealingRepository) *SelfHealingHandler {
	logger := utils.NewTestLogger()
	svc := services.NewSelfHealingService(mockRepo, logger)
	return NewSelfHealingHandler(svc, logger)
}

func setupSelfHealingRouter(handler *SelfHealingHandler, tenantID *uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	if tenantID != nil {
		tid := *tenantID
		r.Use(func(c *gin.Context) {
			c.Set("tenant_id", tid)
			c.Next()
		})
	}
	return r
}

func TestSelfHealing_GetDashboard_Success(t *testing.T) {
	mockRepo := new(MockSelfHealingRepository)
	handler := setupSelfHealingHandler(mockRepo)
	tenantID := uuid.New()
	r := setupSelfHealingRouter(handler, &tenantID)
	r.GET("/self-healing/dashboard", handler.GetDashboard)

	mockRepo.On("CountOpenCircuitBreakers", mock.Anything, tenantID).Return(2, nil)
	mockRepo.On("CountActionsToday", mock.Anything, tenantID).Return(5, nil)
	mockRepo.On("CountPendingSuggestions", mock.Anything, tenantID).Return(3, nil)
	mockRepo.On("GetActiveRemediationRules", mock.Anything, tenantID).Return([]*models.AutoRemediationRule{
		{ID: uuid.New(), Name: "rule-1"},
	}, nil)
	mockRepo.On("GetRecentPredictions", mock.Anything, tenantID, 10).Return([]*models.EndpointHealthPrediction{}, nil)
	mockRepo.On("GetRecentRemediationActions", mock.Anything, tenantID, 10).Return([]*models.RemediationAction{}, nil)

	req, _ := http.NewRequest(http.MethodGet, "/self-healing/dashboard", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var dashboard models.SelfHealingDashboard
	err := json.Unmarshal(w.Body.Bytes(), &dashboard)
	assert.NoError(t, err)
	assert.Equal(t, 2, dashboard.CircuitBreakersOpen)
	assert.Equal(t, 5, dashboard.ActionsToday)
	assert.Equal(t, 3, dashboard.PendingSuggestions)
	assert.Equal(t, 1, dashboard.ActiveRules)

	mockRepo.AssertExpectations(t)
}

func TestSelfHealing_GetDashboard_MissingTenantID(t *testing.T) {
	// Without tenant_id in context, the handler panics on type assertion.
	// Gin's recovery middleware catches the panic and returns 500.
	handler := NewSelfHealingHandler(nil, utils.NewTestLogger())
	r := setupSelfHealingRouter(handler, nil)
	r.GET("/self-healing/dashboard", handler.GetDashboard)

	req, _ := http.NewRequest(http.MethodGet, "/self-healing/dashboard", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSelfHealing_GetEndpointAnalysis_Success(t *testing.T) {
	mockRepo := new(MockSelfHealingRepository)
	handler := setupSelfHealingHandler(mockRepo)
	tenantID := uuid.New()
	endpointID := uuid.New()
	r := setupSelfHealingRouter(handler, &tenantID)
	r.GET("/self-healing/endpoints/:endpoint_id/analysis", handler.GetEndpointAnalysis)

	cb := &models.EndpointCircuitBreaker{
		ID:         uuid.New(),
		TenantID:   tenantID,
		EndpointID: endpointID,
		State:      models.CircuitStateClosed,
	}
	mockRepo.On("GetOrCreateCircuitBreaker", mock.Anything, tenantID, endpointID).Return(cb, nil)
	mockRepo.On("GetPredictionsByEndpoint", mock.Anything, endpointID, 5).Return([]*models.EndpointHealthPrediction{}, nil)
	mockRepo.On("GetSuggestionsByEndpoint", mock.Anything, endpointID).Return([]*models.EndpointOptimizationSuggestion{}, nil)
	mockRepo.On("GetRemediationActionsByEndpoint", mock.Anything, endpointID, 5).Return([]*models.RemediationAction{}, nil)

	req, _ := http.NewRequest(http.MethodGet, "/self-healing/endpoints/"+endpointID.String()+"/analysis", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var analysis models.EndpointHealthAnalysis
	err := json.Unmarshal(w.Body.Bytes(), &analysis)
	assert.NoError(t, err)
	assert.Equal(t, endpointID, analysis.EndpointID)
	assert.Equal(t, "healthy", analysis.Status)
	assert.Equal(t, 100.0, analysis.HealthScore)

	mockRepo.AssertExpectations(t)
}

func TestSelfHealing_GetEndpointAnalysis_InvalidEndpointID(t *testing.T) {
	handler := NewSelfHealingHandler(nil, utils.NewTestLogger())
	tenantID := uuid.New()
	r := setupSelfHealingRouter(handler, &tenantID)
	r.GET("/self-healing/endpoints/:endpoint_id/analysis", handler.GetEndpointAnalysis)

	req, _ := http.NewRequest(http.MethodGet, "/self-healing/endpoints/not-a-uuid/analysis", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	assert.Equal(t, "invalid endpoint_id", body["error"])
}

func TestSelfHealing_GetSupportedActionTypes_Success(t *testing.T) {
	handler := NewSelfHealingHandler(nil, utils.NewTestLogger())
	r := setupSelfHealingRouter(handler, nil)
	r.GET("/self-healing/action-types", handler.GetSupportedActionTypes)

	req, _ := http.NewRequest(http.MethodGet, "/self-healing/action-types", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	assert.NoError(t, err)
	actionTypes, ok := body["action_types"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, actionTypes, 5)
}
