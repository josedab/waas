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

// --- Direct tests for calculateFailureProbability ---

func TestCalculateFailureProbability_AllZero(t *testing.T) {
	t.Parallel()
	svc := NewSelfHealingService(nil, utils.NewLogger("test"))
	features := &models.MLFeatureVector{
		SuccessRate1h:        1.0,
		SuccessRate24h:       1.0,
		ErrorRate1h:          0,
		ErrorRate24h:         0,
		AvgResponseTime1h:    0,
		ConsecutiveFailures:  0,
		TimeSinceLastFailure: 7200, // > 1 hour
	}
	prob := svc.calculateFailureProbability(features)
	assert.Equal(t, 0.0, prob)
}

func TestCalculateFailureProbability_AllWorst(t *testing.T) {
	t.Parallel()
	svc := NewSelfHealingService(nil, utils.NewLogger("test"))
	features := &models.MLFeatureVector{
		SuccessRate1h:        0.0,
		SuccessRate24h:       0.0,
		ErrorRate1h:          1.0,
		ErrorRate24h:         1.0,
		AvgResponseTime1h:    6000,
		ConsecutiveFailures:  50,
		TimeSinceLastFailure: 300,
	}
	prob := svc.calculateFailureProbability(features)
	assert.Equal(t, 1.0, prob) // capped at 1.0
}

func TestCalculateFailureProbability_ErrorRateWeight(t *testing.T) {
	t.Parallel()
	svc := NewSelfHealingService(nil, utils.NewLogger("test"))

	// High ErrorRate1h (1.0) → contribution = 0.3
	features := &models.MLFeatureVector{
		SuccessRate1h:        1.0,
		SuccessRate24h:       1.0,
		ErrorRate1h:          1.0,
		ErrorRate24h:         0,
		TimeSinceLastFailure: 7200,
	}
	prob := svc.calculateFailureProbability(features)
	assert.InDelta(t, 0.3, prob, 0.001)
}

func TestCalculateFailureProbability_SuccessRateWeight(t *testing.T) {
	t.Parallel()
	svc := NewSelfHealingService(nil, utils.NewLogger("test"))

	// SuccessRate1h = 0.0 → contribution = (1-0)*0.2 = 0.2
	features := &models.MLFeatureVector{
		SuccessRate1h:        0.0,
		SuccessRate24h:       1.0, // no contribution from 24h
		ErrorRate1h:          0,
		ErrorRate24h:         0,
		TimeSinceLastFailure: 7200,
	}
	prob := svc.calculateFailureProbability(features)
	assert.InDelta(t, 0.2, prob, 0.001)

	// SuccessRate1h = 1.0 → contribution = 0
	features2 := &models.MLFeatureVector{
		SuccessRate1h:        1.0,
		SuccessRate24h:       1.0,
		TimeSinceLastFailure: 7200,
	}
	prob2 := svc.calculateFailureProbability(features2)
	assert.Equal(t, 0.0, prob2)
}

func TestCalculateFailureProbability_ResponseTimeThresholds(t *testing.T) {
	t.Parallel()
	svc := NewSelfHealingService(nil, utils.NewLogger("test"))

	tests := []struct {
		name     string
		respTime float64
		expected float64
	}{
		{"above 5s threshold", 5001, 0.15},
		{"above 2s threshold", 2001, 0.08},
		{"below 2s threshold", 1999, 0.0},
		{"exactly 5000", 5000, 0.08}, // not > 5000, so falls to > 2000
		{"exactly 2000", 2000, 0.0},  // not > 2000
		{"zero", 0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features := &models.MLFeatureVector{
				SuccessRate1h:        1.0,
				SuccessRate24h:       1.0,
				AvgResponseTime1h:    tt.respTime,
				TimeSinceLastFailure: 7200,
			}
			prob := svc.calculateFailureProbability(features)
			assert.InDelta(t, tt.expected, prob, 0.001)
		})
	}
}

func TestCalculateFailureProbability_ConsecutiveFailures(t *testing.T) {
	t.Parallel()
	svc := NewSelfHealingService(nil, utils.NewLogger("test"))

	tests := []struct {
		name     string
		failures int
		expected float64
	}{
		{"0 failures", 0, 0.0},
		{"1 failure", 1, 0.02},
		{"10 failures", 10, 0.20},
		{"50 failures", 50, 1.0}, // 1.0 due to cap
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features := &models.MLFeatureVector{
				SuccessRate1h:        1.0,
				SuccessRate24h:       1.0,
				ConsecutiveFailures:  tt.failures,
				TimeSinceLastFailure: 7200,
			}
			prob := svc.calculateFailureProbability(features)
			assert.InDelta(t, tt.expected, prob, 0.001)
		})
	}
}

func TestCalculateFailureProbability_RecentFailure(t *testing.T) {
	t.Parallel()
	svc := NewSelfHealingService(nil, utils.NewLogger("test"))

	// < 1 hour → +0.1
	features := &models.MLFeatureVector{
		SuccessRate1h:        1.0,
		SuccessRate24h:       1.0,
		TimeSinceLastFailure: 1800, // 30 minutes
	}
	prob := svc.calculateFailureProbability(features)
	assert.InDelta(t, 0.1, prob, 0.001)

	// > 1 hour → +0
	features2 := &models.MLFeatureVector{
		SuccessRate1h:        1.0,
		SuccessRate24h:       1.0,
		TimeSinceLastFailure: 3601,
	}
	prob2 := svc.calculateFailureProbability(features2)
	assert.Equal(t, 0.0, prob2)
}

func TestCalculateFailureProbability_OutputCapping(t *testing.T) {
	t.Parallel()
	svc := NewSelfHealingService(nil, utils.NewLogger("test"))

	// Sum would exceed 1.0 without capping
	features := &models.MLFeatureVector{
		SuccessRate1h:        0.0,     // +0.2
		SuccessRate24h:       0.0,     // +0.1
		ErrorRate1h:          1.0,     // +0.3
		ErrorRate24h:         1.0,     // +0.15
		AvgResponseTime1h:    6000,    // +0.15
		ConsecutiveFailures:  20,      // +0.40
		TimeSinceLastFailure: 300,     // +0.1
	}
	// Total = 0.2+0.1+0.3+0.15+0.15+0.4+0.1 = 1.40 → capped to 1.0
	prob := svc.calculateFailureProbability(features)
	assert.Equal(t, 1.0, prob)
}

func TestCalculateFailureProbability_CombinedFactors(t *testing.T) {
	t.Parallel()
	svc := NewSelfHealingService(nil, utils.NewLogger("test"))

	features := &models.MLFeatureVector{
		SuccessRate1h:        0.5,    // (1-0.5)*0.2 = 0.1
		SuccessRate24h:       0.8,    // (1-0.8)*0.1 = 0.02
		ErrorRate1h:          0.5,    // 0.5*0.3 = 0.15
		ErrorRate24h:         0.3,    // 0.3*0.15 = 0.045
		AvgResponseTime1h:    3000,   // > 2000 → +0.08
		ConsecutiveFailures:  5,      // 5*0.02 = 0.10
		TimeSinceLastFailure: 1800,   // < 3600 → +0.1
	}
	// Expected: 0.1 + 0.02 + 0.15 + 0.045 + 0.08 + 0.10 + 0.1 = 0.595
	prob := svc.calculateFailureProbability(features)
	assert.InDelta(t, 0.595, prob, 0.001)
}

func TestCalculateFailureProbability_ZeroTimeSinceLastFailure(t *testing.T) {
	t.Parallel()
	svc := NewSelfHealingService(nil, utils.NewLogger("test"))

	features := &models.MLFeatureVector{
		SuccessRate1h:        1.0,
		SuccessRate24h:       1.0,
		TimeSinceLastFailure: 0, // 0 < 3600, so +0.1
	}
	prob := svc.calculateFailureProbability(features)
	assert.InDelta(t, 0.1, prob, 0.001)
}
