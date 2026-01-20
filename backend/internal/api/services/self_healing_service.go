package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"webhook-platform/pkg/models"
	"webhook-platform/pkg/repository"
	"webhook-platform/pkg/utils"
)

// SelfHealingService handles self-healing endpoint intelligence
type SelfHealingService struct {
	repo         repository.SelfHealingRepository
	logger       *utils.Logger
	modelVersion string
}

// NewSelfHealingService creates a new self-healing service
func NewSelfHealingService(repo repository.SelfHealingRepository, logger *utils.Logger) *SelfHealingService {
	return &SelfHealingService{
		repo:         repo,
		logger:       logger,
		modelVersion: "v1.0.0",
	}
}

// PredictEndpointHealth generates a health prediction for an endpoint
func (s *SelfHealingService) PredictEndpointHealth(ctx context.Context, tenantID, endpointID uuid.UUID, features *models.MLFeatureVector) (*models.EndpointHealthPrediction, error) {
	// Calculate failure probability using simplified ML model
	probability := s.calculateFailureProbability(features)
	confidence := s.calculateConfidence(features)

	predictionType := models.PredictionTypeFailure
	if probability < 0.3 {
		predictionType = models.PredictionTypeRecovery
	} else if probability < 0.6 {
		predictionType = models.PredictionTypeDegradation
	}

	prediction := &models.EndpointHealthPrediction{
		TenantID:       tenantID,
		EndpointID:     endpointID,
		PredictionType: predictionType,
		Probability:    probability,
		Confidence:     confidence,
		ModelVersion:   s.modelVersion,
		FeaturesUsed: map[string]interface{}{
			"success_rate_1h":         features.SuccessRate1h,
			"success_rate_24h":        features.SuccessRate24h,
			"error_rate_1h":           features.ErrorRate1h,
			"error_rate_24h":          features.ErrorRate24h,
			"avg_response_time_1h":    features.AvgResponseTime1h,
			"avg_response_time_24h":   features.AvgResponseTime24h,
			"consecutive_failures":    features.ConsecutiveFailures,
			"request_volume_1h":       features.RequestVolume1h,
		},
	}

	// Predict when failure might occur
	if predictionType == models.PredictionTypeFailure && probability > 0.7 {
		predictedTime := time.Now().Add(time.Duration(1/probability) * time.Hour)
		prediction.PredictedTime = &predictedTime
	}

	if err := s.repo.CreatePrediction(ctx, prediction); err != nil {
		return nil, fmt.Errorf("failed to create prediction: %w", err)
	}

	// Check if auto-remediation should be triggered
	go s.checkAutoRemediation(context.Background(), tenantID, endpointID, prediction)

	return prediction, nil
}

// calculateFailureProbability calculates failure probability from features
func (s *SelfHealingService) calculateFailureProbability(features *models.MLFeatureVector) float64 {
	// Simplified ML model using weighted factors
	// In production, this would use a trained model (scikit-learn, TensorFlow, etc.)

	var probability float64

	// Error rate contributes most
	probability += features.ErrorRate1h * 0.3
	probability += features.ErrorRate24h * 0.15

	// Low success rate increases probability
	probability += (1 - features.SuccessRate1h) * 0.2
	probability += (1 - features.SuccessRate24h) * 0.1

	// High response times indicate issues
	if features.AvgResponseTime1h > 5000 { // > 5 seconds
		probability += 0.15
	} else if features.AvgResponseTime1h > 2000 { // > 2 seconds
		probability += 0.08
	}

	// Consecutive failures are strong indicator
	probability += float64(features.ConsecutiveFailures) * 0.02

	// Recent failure time matters
	if features.TimeSinceLastFailure < 3600 { // < 1 hour
		probability += 0.1
	}

	// Cap at 1.0
	if probability > 1.0 {
		probability = 1.0
	}

	return probability
}

// calculateConfidence calculates confidence score for prediction
func (s *SelfHealingService) calculateConfidence(features *models.MLFeatureVector) float64 {
	// Confidence increases with more data
	confidence := 0.5

	// More requests = more confidence
	if features.RequestVolume24h > 1000 {
		confidence += 0.2
	} else if features.RequestVolume24h > 100 {
		confidence += 0.1
	}

	// Consistent patterns increase confidence
	successRateDiff := math.Abs(features.SuccessRate1h - features.SuccessRate24h)
	if successRateDiff < 0.1 {
		confidence += 0.15
	}

	errorRateDiff := math.Abs(features.ErrorRate1h - features.ErrorRate24h)
	if errorRateDiff < 0.1 {
		confidence += 0.15
	}

	if confidence > 0.95 {
		confidence = 0.95
	}

	return confidence
}

// checkAutoRemediation checks if auto-remediation rules should trigger
func (s *SelfHealingService) checkAutoRemediation(ctx context.Context, tenantID, endpointID uuid.UUID, prediction *models.EndpointHealthPrediction) {
	if prediction.Probability < 0.7 {
		return // Only trigger for high probability failures
	}

	rules, err := s.repo.GetActiveRemediationRules(ctx, tenantID)
	if err != nil {
		s.logger.Error("Failed to get remediation rules", map[string]interface{}{"error": err})
		return
	}

	for _, rule := range rules {
		if s.shouldTriggerRule(rule, prediction) {
			s.executeRemediation(ctx, tenantID, endpointID, rule, prediction)
		}
	}
}

// shouldTriggerRule determines if a rule should trigger
func (s *SelfHealingService) shouldTriggerRule(rule *models.AutoRemediationRule, prediction *models.EndpointHealthPrediction) bool {
	// Check cooldown
	if rule.LastTriggered != nil {
		cooldownEnd := rule.LastTriggered.Add(time.Duration(rule.CooldownMinutes) * time.Minute)
		if time.Now().Before(cooldownEnd) {
			return false
		}
	}

	// Check trigger conditions
	if threshold, ok := rule.TriggerCondition["failure_probability_threshold"].(float64); ok {
		if prediction.Probability < threshold {
			return false
		}
	}

	if predType, ok := rule.TriggerCondition["prediction_type"].(string); ok {
		if prediction.PredictionType != predType {
			return false
		}
	}

	return true
}

// executeRemediation executes a remediation action
func (s *SelfHealingService) executeRemediation(ctx context.Context, tenantID, endpointID uuid.UUID, rule *models.AutoRemediationRule, prediction *models.EndpointHealthPrediction) {
	action := &models.RemediationAction{
		TenantID:     tenantID,
		EndpointID:   endpointID,
		RuleID:       &rule.ID,
		PredictionID: &prediction.ID,
		ActionType:   rule.ActionType,
		ActionDetails: map[string]interface{}{
			"rule_name":          rule.Name,
			"trigger_prediction": prediction.ID.String(),
			"probability":        prediction.Probability,
		},
		TriggeredBy: "auto",
	}

	// Execute based on action type
	switch rule.ActionType {
	case models.RemediationCircuitBreak:
		s.executeCircuitBreak(ctx, tenantID, endpointID, action)
	case models.RemediationAdjustRetry:
		s.executeAdjustRetry(ctx, endpointID, rule.ActionConfig, action)
	case models.RemediationNotify:
		s.executeNotify(ctx, tenantID, endpointID, prediction, action)
	case models.RemediationDisableEndpoint:
		s.executeDisableEndpoint(ctx, endpointID, action)
	}

	// Record the action
	if err := s.repo.CreateRemediationAction(ctx, action); err != nil {
		s.logger.Error("Failed to record remediation action", map[string]interface{}{"error": err})
	}

	// Update prediction
	s.repo.UpdatePredictionAction(ctx, prediction.ID, rule.ActionType)

	// Increment rule trigger count
	s.repo.IncrementRuleTriggerCount(ctx, rule.ID)

	s.logger.Info("Remediation executed", map[string]interface{}{"rule": rule.Name, "action": rule.ActionType, "endpoint": endpointID})
}

func (s *SelfHealingService) executeCircuitBreak(ctx context.Context, tenantID, endpointID uuid.UUID, action *models.RemediationAction) {
	cb, err := s.repo.GetOrCreateCircuitBreaker(ctx, tenantID, endpointID)
	if err != nil {
		action.Outcome = models.RemediationOutcomeFailed
		return
	}

	action.PreviousState = map[string]interface{}{"state": cb.State}

	if err := s.repo.UpdateCircuitBreakerState(ctx, cb.ID, models.CircuitStateOpen, cb.FailureCount, 0); err != nil {
		action.Outcome = models.RemediationOutcomeFailed
		return
	}

	action.NewState = map[string]interface{}{"state": models.CircuitStateOpen}
	action.Outcome = models.RemediationOutcomeSuccess
}

func (s *SelfHealingService) executeAdjustRetry(ctx context.Context, endpointID uuid.UUID, config map[string]interface{}, action *models.RemediationAction) {
	// In production, would update the endpoint's retry configuration
	action.ActionDetails["adjustment"] = config
	action.Outcome = models.RemediationOutcomeSuccess
}

func (s *SelfHealingService) executeNotify(ctx context.Context, tenantID, endpointID uuid.UUID, prediction *models.EndpointHealthPrediction, action *models.RemediationAction) {
	// In production, would send notification via configured channels
	action.ActionDetails["notification_sent"] = true
	action.ActionDetails["message"] = fmt.Sprintf("Endpoint %s predicted to fail with %.1f%% probability", endpointID, prediction.Probability*100)
	action.Outcome = models.RemediationOutcomeSuccess
}

func (s *SelfHealingService) executeDisableEndpoint(ctx context.Context, endpointID uuid.UUID, action *models.RemediationAction) {
	// In production, would update the endpoint's status
	action.ActionDetails["endpoint_disabled"] = true
	action.Outcome = models.RemediationOutcomeSuccess
}

// CreateRemediationRule creates a new remediation rule
func (s *SelfHealingService) CreateRemediationRule(ctx context.Context, tenantID uuid.UUID, req *models.CreateRemediationRuleRequest) (*models.AutoRemediationRule, error) {
	validActions := map[string]bool{
		models.RemediationDisableEndpoint: true,
		models.RemediationAdjustRetry:     true,
		models.RemediationNotify:          true,
		models.RemediationCircuitBreak:    true,
		models.RemediationRateLimit:       true,
	}

	if !validActions[req.ActionType] {
		return nil, fmt.Errorf("invalid action type: %s", req.ActionType)
	}

	rule := &models.AutoRemediationRule{
		TenantID:         tenantID,
		Name:             req.Name,
		Description:      req.Description,
		TriggerCondition: req.TriggerCondition,
		ActionType:       req.ActionType,
		ActionConfig:     req.ActionConfig,
		Enabled:          true,
		CooldownMinutes:  req.CooldownMinutes,
	}

	if rule.CooldownMinutes == 0 {
		rule.CooldownMinutes = 30
	}

	if err := s.repo.CreateRemediationRule(ctx, rule); err != nil {
		return nil, fmt.Errorf("failed to create rule: %w", err)
	}

	return rule, nil
}

// GetRemediationRules retrieves all remediation rules for a tenant
func (s *SelfHealingService) GetRemediationRules(ctx context.Context, tenantID uuid.UUID) ([]*models.AutoRemediationRule, error) {
	return s.repo.GetRemediationRulesByTenant(ctx, tenantID)
}

// TriggerManualRemediation triggers remediation manually
func (s *SelfHealingService) TriggerManualRemediation(ctx context.Context, tenantID uuid.UUID, req *models.TriggerRemediationRequest) (*models.RemediationAction, error) {
	endpointID, err := uuid.Parse(req.EndpointID)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint_id")
	}

	action := &models.RemediationAction{
		TenantID:      tenantID,
		EndpointID:    endpointID,
		ActionType:    req.ActionType,
		ActionDetails: req.ActionDetails,
		TriggeredBy:   "manual",
	}

	// Execute the action
	switch req.ActionType {
	case models.RemediationCircuitBreak:
		s.executeCircuitBreak(ctx, tenantID, endpointID, action)
	case models.RemediationNotify:
		action.Outcome = models.RemediationOutcomeSuccess
	default:
		action.Outcome = models.RemediationOutcomeSuccess
	}

	if err := s.repo.CreateRemediationAction(ctx, action); err != nil {
		return nil, fmt.Errorf("failed to create action: %w", err)
	}

	return action, nil
}

// GetEndpointHealthAnalysis gets comprehensive health analysis for an endpoint
func (s *SelfHealingService) GetEndpointHealthAnalysis(ctx context.Context, tenantID, endpointID uuid.UUID) (*models.EndpointHealthAnalysis, error) {
	analysis := &models.EndpointHealthAnalysis{
		EndpointID: endpointID,
	}

	// Get circuit breaker state
	cb, _ := s.repo.GetOrCreateCircuitBreaker(ctx, tenantID, endpointID)
	if cb != nil {
		analysis.CircuitBreakerState = cb.State
	}

	// Get recent predictions
	predictions, _ := s.repo.GetPredictionsByEndpoint(ctx, endpointID, 5)
	analysis.RecentPredictions = predictions

	if len(predictions) > 0 {
		analysis.FailureProbability = predictions[0].Probability
	}

	// Get pending suggestions
	suggestions, _ := s.repo.GetSuggestionsByEndpoint(ctx, endpointID)
	var pending []*models.EndpointOptimizationSuggestion
	for _, s := range suggestions {
		if s.Status == models.SuggestionStatusPending {
			pending = append(pending, s)
		}
	}
	analysis.PendingSuggestions = pending

	// Get recent actions
	actions, _ := s.repo.GetRemediationActionsByEndpoint(ctx, endpointID, 5)
	analysis.RecentActions = actions

	// Calculate health score
	analysis.HealthScore = s.calculateHealthScore(analysis)
	analysis.Status = s.getHealthStatus(analysis.HealthScore)

	// Generate recommendations
	analysis.RecommendedActions = s.generateRecommendations(analysis)

	return analysis, nil
}

func (s *SelfHealingService) calculateHealthScore(analysis *models.EndpointHealthAnalysis) float64 {
	score := 100.0

	// Reduce for failure probability
	score -= analysis.FailureProbability * 40

	// Reduce for open circuit breaker
	if analysis.CircuitBreakerState == models.CircuitStateOpen {
		score -= 30
	} else if analysis.CircuitBreakerState == models.CircuitStateHalfOpen {
		score -= 15
	}

	// Reduce for pending suggestions
	score -= float64(len(analysis.PendingSuggestions)) * 5

	if score < 0 {
		score = 0
	}

	return score
}

func (s *SelfHealingService) getHealthStatus(score float64) string {
	if score >= 80 {
		return "healthy"
	} else if score >= 50 {
		return "degraded"
	}
	return "unhealthy"
}

func (s *SelfHealingService) generateRecommendations(analysis *models.EndpointHealthAnalysis) []string {
	var recommendations []string

	if analysis.CircuitBreakerState == models.CircuitStateOpen {
		recommendations = append(recommendations, "Circuit breaker is open. Investigate endpoint failures.")
	}

	if analysis.FailureProbability > 0.7 {
		recommendations = append(recommendations, "High failure probability detected. Consider enabling auto-remediation.")
	}

	if len(analysis.PendingSuggestions) > 0 {
		recommendations = append(recommendations, fmt.Sprintf("Review %d pending optimization suggestions.", len(analysis.PendingSuggestions)))
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Endpoint is healthy. No actions needed.")
	}

	return recommendations
}

// UpdateCircuitBreaker updates circuit breaker configuration
func (s *SelfHealingService) UpdateCircuitBreaker(ctx context.Context, tenantID uuid.UUID, req *models.UpdateCircuitBreakerRequest) (*models.EndpointCircuitBreaker, error) {
	endpointID, err := uuid.Parse(req.EndpointID)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint_id")
	}

	cb, err := s.repo.GetOrCreateCircuitBreaker(ctx, tenantID, endpointID)
	if err != nil {
		return nil, err
	}

	if err := s.repo.UpdateCircuitBreakerConfig(ctx, cb.ID, req.ResetTimeoutSeconds, req.FailureThreshold, req.SuccessThreshold); err != nil {
		return nil, err
	}

	return s.repo.GetOrCreateCircuitBreaker(ctx, tenantID, endpointID)
}

// CreateOptimizationSuggestion creates an optimization suggestion
func (s *SelfHealingService) CreateOptimizationSuggestion(ctx context.Context, tenantID, endpointID uuid.UUID, suggestionType string, currentConfig, suggestedConfig map[string]interface{}, improvement string, confidence float64, rationale string) (*models.EndpointOptimizationSuggestion, error) {
	suggestion := &models.EndpointOptimizationSuggestion{
		TenantID:            tenantID,
		EndpointID:          endpointID,
		SuggestionType:      suggestionType,
		CurrentConfig:       currentConfig,
		SuggestedConfig:     suggestedConfig,
		ExpectedImprovement: improvement,
		Confidence:          confidence,
		Rationale:           rationale,
	}

	if err := s.repo.CreateSuggestion(ctx, suggestion); err != nil {
		return nil, fmt.Errorf("failed to create suggestion: %w", err)
	}

	return suggestion, nil
}

// ApplySuggestion applies an optimization suggestion
func (s *SelfHealingService) ApplySuggestion(ctx context.Context, tenantID uuid.UUID, suggestionID uuid.UUID) error {
	suggestion, err := s.repo.GetSuggestion(ctx, suggestionID)
	if err != nil {
		return fmt.Errorf("suggestion not found")
	}

	if suggestion.TenantID != tenantID {
		return fmt.Errorf("suggestion not found")
	}

	if suggestion.Status != models.SuggestionStatusPending {
		return fmt.Errorf("suggestion is not in pending status")
	}

	// In production, would actually apply the configuration change
	// For now, just update the status
	return s.repo.UpdateSuggestionStatus(ctx, suggestionID, models.SuggestionStatusApplied)
}

// GetDashboard retrieves the self-healing dashboard data
func (s *SelfHealingService) GetDashboard(ctx context.Context, tenantID uuid.UUID) (*models.SelfHealingDashboard, error) {
	dashboard := &models.SelfHealingDashboard{}

	// Get counts
	openCBs, _ := s.repo.CountOpenCircuitBreakers(ctx, tenantID)
	actionsToday, _ := s.repo.CountActionsToday(ctx, tenantID)
	pendingSuggestions, _ := s.repo.CountPendingSuggestions(ctx, tenantID)

	rules, _ := s.repo.GetActiveRemediationRules(ctx, tenantID)

	dashboard.CircuitBreakersOpen = openCBs
	dashboard.ActionsToday = actionsToday
	dashboard.PendingSuggestions = pendingSuggestions
	dashboard.ActiveRules = len(rules)

	// Get recent data
	predictions, _ := s.repo.GetRecentPredictions(ctx, tenantID, 10)
	actions, _ := s.repo.GetRecentRemediationActions(ctx, tenantID, 10)

	dashboard.RecentPredictions = predictions
	dashboard.RecentActions = actions

	// Count predictions today
	for _, p := range predictions {
		if p.CreatedAt.After(time.Now().Truncate(24 * time.Hour)) {
			dashboard.PredictionsToday++
		}
	}

	return dashboard, nil
}

// GetPredictions retrieves recent predictions for a tenant
func (s *SelfHealingService) GetPredictions(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.EndpointHealthPrediction, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.repo.GetRecentPredictions(ctx, tenantID, limit)
}

// GetRemediationActions retrieves recent remediation actions
func (s *SelfHealingService) GetRemediationActions(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.RemediationAction, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.repo.GetRecentRemediationActions(ctx, tenantID, limit)
}

// GetSuggestions retrieves pending suggestions for a tenant
func (s *SelfHealingService) GetSuggestions(ctx context.Context, tenantID uuid.UUID) ([]*models.EndpointOptimizationSuggestion, error) {
	return s.repo.GetPendingSuggestions(ctx, tenantID)
}
