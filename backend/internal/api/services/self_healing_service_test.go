package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Mock SelfHealingRepository ---
type MockSelfHealingRepo struct {
	mock.Mock
}

func (m *MockSelfHealingRepo) CreatePrediction(ctx context.Context, pred *models.EndpointHealthPrediction) error {
	return m.Called(ctx, pred).Error(0)
}
func (m *MockSelfHealingRepo) GetPrediction(ctx context.Context, id uuid.UUID) (*models.EndpointHealthPrediction, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.EndpointHealthPrediction), args.Error(1)
}
func (m *MockSelfHealingRepo) GetPredictionsByEndpoint(ctx context.Context, endpointID uuid.UUID, limit int) ([]*models.EndpointHealthPrediction, error) {
	args := m.Called(ctx, endpointID, limit)
	return args.Get(0).([]*models.EndpointHealthPrediction), args.Error(1)
}
func (m *MockSelfHealingRepo) GetRecentPredictions(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.EndpointHealthPrediction, error) {
	args := m.Called(ctx, tenantID, limit)
	return args.Get(0).([]*models.EndpointHealthPrediction), args.Error(1)
}
func (m *MockSelfHealingRepo) UpdatePredictionAccuracy(ctx context.Context, id uuid.UUID, wasAccurate bool) error {
	return m.Called(ctx, id, wasAccurate).Error(0)
}
func (m *MockSelfHealingRepo) UpdatePredictionAction(ctx context.Context, id uuid.UUID, action string) error {
	return m.Called(ctx, id, action).Error(0)
}
func (m *MockSelfHealingRepo) CreateBehaviorPattern(ctx context.Context, pattern *models.EndpointBehaviorPattern) error {
	return m.Called(ctx, pattern).Error(0)
}
func (m *MockSelfHealingRepo) GetBehaviorPatterns(ctx context.Context, endpointID uuid.UUID) ([]*models.EndpointBehaviorPattern, error) {
	args := m.Called(ctx, endpointID)
	return args.Get(0).([]*models.EndpointBehaviorPattern), args.Error(1)
}
func (m *MockSelfHealingRepo) DeleteOldPatterns(ctx context.Context, endpointID uuid.UUID, beforeTime time.Time) error {
	return m.Called(ctx, endpointID, beforeTime).Error(0)
}
func (m *MockSelfHealingRepo) CreateRemediationRule(ctx context.Context, rule *models.AutoRemediationRule) error {
	return m.Called(ctx, rule).Error(0)
}
func (m *MockSelfHealingRepo) GetRemediationRule(ctx context.Context, id uuid.UUID) (*models.AutoRemediationRule, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.AutoRemediationRule), args.Error(1)
}
func (m *MockSelfHealingRepo) GetRemediationRulesByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.AutoRemediationRule, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*models.AutoRemediationRule), args.Error(1)
}
func (m *MockSelfHealingRepo) GetActiveRemediationRules(ctx context.Context, tenantID uuid.UUID) ([]*models.AutoRemediationRule, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*models.AutoRemediationRule), args.Error(1)
}
func (m *MockSelfHealingRepo) UpdateRemediationRule(ctx context.Context, rule *models.AutoRemediationRule) error {
	return m.Called(ctx, rule).Error(0)
}
func (m *MockSelfHealingRepo) IncrementRuleTriggerCount(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockSelfHealingRepo) DeleteRemediationRule(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockSelfHealingRepo) CreateRemediationAction(ctx context.Context, action *models.RemediationAction) error {
	return m.Called(ctx, action).Error(0)
}
func (m *MockSelfHealingRepo) GetRemediationAction(ctx context.Context, id uuid.UUID) (*models.RemediationAction, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.RemediationAction), args.Error(1)
}
func (m *MockSelfHealingRepo) GetRemediationActionsByEndpoint(ctx context.Context, endpointID uuid.UUID, limit int) ([]*models.RemediationAction, error) {
	args := m.Called(ctx, endpointID, limit)
	return args.Get(0).([]*models.RemediationAction), args.Error(1)
}
func (m *MockSelfHealingRepo) GetRecentRemediationActions(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.RemediationAction, error) {
	args := m.Called(ctx, tenantID, limit)
	return args.Get(0).([]*models.RemediationAction), args.Error(1)
}
func (m *MockSelfHealingRepo) UpdateRemediationActionOutcome(ctx context.Context, id uuid.UUID, outcome string) error {
	return m.Called(ctx, id, outcome).Error(0)
}
func (m *MockSelfHealingRepo) RevertRemediationAction(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockSelfHealingRepo) CountActionsToday(ctx context.Context, tenantID uuid.UUID) (int, error) {
	args := m.Called(ctx, tenantID)
	return args.Int(0), args.Error(1)
}
func (m *MockSelfHealingRepo) CreateSuggestion(ctx context.Context, suggestion *models.EndpointOptimizationSuggestion) error {
	return m.Called(ctx, suggestion).Error(0)
}
func (m *MockSelfHealingRepo) GetSuggestion(ctx context.Context, id uuid.UUID) (*models.EndpointOptimizationSuggestion, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.EndpointOptimizationSuggestion), args.Error(1)
}
func (m *MockSelfHealingRepo) GetSuggestionsByEndpoint(ctx context.Context, endpointID uuid.UUID) ([]*models.EndpointOptimizationSuggestion, error) {
	args := m.Called(ctx, endpointID)
	return args.Get(0).([]*models.EndpointOptimizationSuggestion), args.Error(1)
}
func (m *MockSelfHealingRepo) GetPendingSuggestions(ctx context.Context, tenantID uuid.UUID) ([]*models.EndpointOptimizationSuggestion, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*models.EndpointOptimizationSuggestion), args.Error(1)
}
func (m *MockSelfHealingRepo) UpdateSuggestionStatus(ctx context.Context, id uuid.UUID, status string) error {
	return m.Called(ctx, id, status).Error(0)
}
func (m *MockSelfHealingRepo) CountPendingSuggestions(ctx context.Context, tenantID uuid.UUID) (int, error) {
	args := m.Called(ctx, tenantID)
	return args.Int(0), args.Error(1)
}
func (m *MockSelfHealingRepo) GetOrCreateCircuitBreaker(ctx context.Context, tenantID, endpointID uuid.UUID) (*models.EndpointCircuitBreaker, error) {
	args := m.Called(ctx, tenantID, endpointID)
	return args.Get(0).(*models.EndpointCircuitBreaker), args.Error(1)
}
func (m *MockSelfHealingRepo) UpdateCircuitBreakerState(ctx context.Context, id uuid.UUID, state string, failureCount, successCount int) error {
	return m.Called(ctx, id, state, failureCount, successCount).Error(0)
}
func (m *MockSelfHealingRepo) UpdateCircuitBreakerConfig(ctx context.Context, id uuid.UUID, resetTimeout, failureThreshold, successThreshold int) error {
	return m.Called(ctx, id, resetTimeout, failureThreshold, successThreshold).Error(0)
}
func (m *MockSelfHealingRepo) GetOpenCircuitBreakers(ctx context.Context, tenantID uuid.UUID) ([]*models.EndpointCircuitBreaker, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*models.EndpointCircuitBreaker), args.Error(1)
}
func (m *MockSelfHealingRepo) RecordCircuitBreakerFailure(ctx context.Context, endpointID uuid.UUID) error {
	return m.Called(ctx, endpointID).Error(0)
}
func (m *MockSelfHealingRepo) RecordCircuitBreakerSuccess(ctx context.Context, endpointID uuid.UUID) error {
	return m.Called(ctx, endpointID).Error(0)
}
func (m *MockSelfHealingRepo) CountOpenCircuitBreakers(ctx context.Context, tenantID uuid.UUID) (int, error) {
	args := m.Called(ctx, tenantID)
	return args.Int(0), args.Error(1)
}

// --- Self-Healing Service Tests ---

func TestSelfHealingService_PredictEndpointHealth_HighFailure(t *testing.T) {
	t.Parallel()
	repo := &MockSelfHealingRepo{}
	logger := utils.NewLogger("test")
	svc := NewSelfHealingService(repo, logger)

	tenantID := uuid.New()
	endpointID := uuid.New()
	features := &models.MLFeatureVector{
		SuccessRate1h:        0.1,
		SuccessRate24h:       0.3,
		ErrorRate1h:          0.9,
		ErrorRate24h:         0.7,
		AvgResponseTime1h:    6000,
		AvgResponseTime24h:   3000,
		ConsecutiveFailures:  10,
		TimeSinceLastFailure: 300,
		RequestVolume1h:      100,
		RequestVolume24h:     2000,
	}

	repo.On("CreatePrediction", mock.Anything, mock.AnythingOfType("*models.EndpointHealthPrediction")).Return(nil)
	repo.On("GetActiveRemediationRules", mock.Anything, mock.Anything).Return([]*models.AutoRemediationRule{}, nil).Maybe()

	prediction, err := svc.PredictEndpointHealth(context.Background(), tenantID, endpointID, features)
	require.NoError(t, err)
	assert.Equal(t, models.PredictionTypeFailure, prediction.PredictionType)
	assert.Greater(t, prediction.Probability, 0.6)
	assert.Greater(t, prediction.Confidence, 0.0)
	assert.Equal(t, "v1.0.0", prediction.ModelVersion)
}

func TestSelfHealingService_PredictEndpointHealth_HealthyEndpoint(t *testing.T) {
	t.Parallel()
	repo := &MockSelfHealingRepo{}
	logger := utils.NewLogger("test")
	svc := NewSelfHealingService(repo, logger)

	features := &models.MLFeatureVector{
		SuccessRate1h:        0.99,
		SuccessRate24h:       0.98,
		ErrorRate1h:          0.01,
		ErrorRate24h:         0.02,
		AvgResponseTime1h:    200,
		AvgResponseTime24h:   250,
		ConsecutiveFailures:  0,
		TimeSinceLastFailure: 86400,
		RequestVolume24h:     5000,
	}

	repo.On("CreatePrediction", mock.Anything, mock.Anything).Return(nil)

	prediction, err := svc.PredictEndpointHealth(context.Background(), uuid.New(), uuid.New(), features)
	require.NoError(t, err)
	assert.Equal(t, models.PredictionTypeRecovery, prediction.PredictionType)
	assert.Less(t, prediction.Probability, 0.3)
}

func TestSelfHealingService_PredictEndpointHealth_RepoError(t *testing.T) {
	t.Parallel()
	repo := &MockSelfHealingRepo{}
	logger := utils.NewLogger("test")
	svc := NewSelfHealingService(repo, logger)

	repo.On("CreatePrediction", mock.Anything, mock.Anything).Return(fmt.Errorf("db error"))

	_, err := svc.PredictEndpointHealth(context.Background(), uuid.New(), uuid.New(), &models.MLFeatureVector{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create prediction")
}

func TestSelfHealingService_CreateRemediationRule_Valid(t *testing.T) {
	t.Parallel()
	repo := &MockSelfHealingRepo{}
	logger := utils.NewLogger("test")
	svc := NewSelfHealingService(repo, logger)

	tenantID := uuid.New()
	req := &models.CreateRemediationRuleRequest{
		Name:       "Auto circuit break",
		ActionType: models.RemediationCircuitBreak,
		TriggerCondition: map[string]interface{}{
			"failure_probability_threshold": 0.8,
		},
	}

	repo.On("CreateRemediationRule", mock.Anything, mock.AnythingOfType("*models.AutoRemediationRule")).Return(nil)

	rule, err := svc.CreateRemediationRule(context.Background(), tenantID, req)
	require.NoError(t, err)
	assert.Equal(t, models.RemediationCircuitBreak, rule.ActionType)
	assert.Equal(t, 30, rule.CooldownMinutes) // default
	assert.True(t, rule.Enabled)
}

func TestSelfHealingService_CreateRemediationRule_InvalidAction(t *testing.T) {
	t.Parallel()
	repo := &MockSelfHealingRepo{}
	logger := utils.NewLogger("test")
	svc := NewSelfHealingService(repo, logger)

	_, err := svc.CreateRemediationRule(context.Background(), uuid.New(), &models.CreateRemediationRuleRequest{
		Name:       "Bad Rule",
		ActionType: "invalid_action",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid action type")
}

func TestSelfHealingService_CreateRemediationRule_CustomCooldown(t *testing.T) {
	t.Parallel()
	repo := &MockSelfHealingRepo{}
	logger := utils.NewLogger("test")
	svc := NewSelfHealingService(repo, logger)

	repo.On("CreateRemediationRule", mock.Anything, mock.Anything).Return(nil)

	rule, err := svc.CreateRemediationRule(context.Background(), uuid.New(), &models.CreateRemediationRuleRequest{
		Name:            "Custom",
		ActionType:      models.RemediationNotify,
		CooldownMinutes: 60,
	})
	require.NoError(t, err)
	assert.Equal(t, 60, rule.CooldownMinutes)
}

func TestSelfHealingService_GetEndpointHealthAnalysis(t *testing.T) {
	t.Parallel()
	repo := &MockSelfHealingRepo{}
	logger := utils.NewLogger("test")
	svc := NewSelfHealingService(repo, logger)

	tenantID := uuid.New()
	endpointID := uuid.New()

	repo.On("GetOrCreateCircuitBreaker", mock.Anything, tenantID, endpointID).Return(&models.EndpointCircuitBreaker{
		State: models.CircuitStateClosed,
	}, nil)
	repo.On("GetPredictionsByEndpoint", mock.Anything, endpointID, 5).Return([]*models.EndpointHealthPrediction{
		{Probability: 0.2},
	}, nil)
	repo.On("GetSuggestionsByEndpoint", mock.Anything, endpointID).Return([]*models.EndpointOptimizationSuggestion{}, nil)
	repo.On("GetRemediationActionsByEndpoint", mock.Anything, endpointID, 5).Return([]*models.RemediationAction{}, nil)

	analysis, err := svc.GetEndpointHealthAnalysis(context.Background(), tenantID, endpointID)
	require.NoError(t, err)
	assert.Equal(t, endpointID, analysis.EndpointID)
	assert.Equal(t, "healthy", analysis.Status)
	assert.Greater(t, analysis.HealthScore, 80.0)
	assert.NotEmpty(t, analysis.RecommendedActions)
}

func TestSelfHealingService_GetDashboard(t *testing.T) {
	t.Parallel()
	repo := &MockSelfHealingRepo{}
	logger := utils.NewLogger("test")
	svc := NewSelfHealingService(repo, logger)

	tenantID := uuid.New()
	repo.On("CountOpenCircuitBreakers", mock.Anything, tenantID).Return(2, nil)
	repo.On("CountActionsToday", mock.Anything, tenantID).Return(5, nil)
	repo.On("CountPendingSuggestions", mock.Anything, tenantID).Return(3, nil)
	repo.On("GetActiveRemediationRules", mock.Anything, tenantID).Return([]*models.AutoRemediationRule{{}, {}}, nil)
	repo.On("GetRecentPredictions", mock.Anything, tenantID, 10).Return([]*models.EndpointHealthPrediction{}, nil)
	repo.On("GetRecentRemediationActions", mock.Anything, tenantID, 10).Return([]*models.RemediationAction{}, nil)

	dashboard, err := svc.GetDashboard(context.Background(), tenantID)
	require.NoError(t, err)
	assert.Equal(t, 2, dashboard.CircuitBreakersOpen)
	assert.Equal(t, 5, dashboard.ActionsToday)
	assert.Equal(t, 3, dashboard.PendingSuggestions)
	assert.Equal(t, 2, dashboard.ActiveRules)
}

func TestSelfHealingService_GetPredictions_DefaultLimit(t *testing.T) {
	t.Parallel()
	repo := &MockSelfHealingRepo{}
	logger := utils.NewLogger("test")
	svc := NewSelfHealingService(repo, logger)

	tenantID := uuid.New()
	repo.On("GetRecentPredictions", mock.Anything, tenantID, 20).Return([]*models.EndpointHealthPrediction{}, nil)

	preds, err := svc.GetPredictions(context.Background(), tenantID, 0)
	require.NoError(t, err)
	assert.Empty(t, preds)
	repo.AssertExpectations(t)
}
