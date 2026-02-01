package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/josedab/waas/pkg/models"
)

// SelfHealingRepository handles self-healing data persistence
type SelfHealingRepository interface {
	// Predictions
	CreatePrediction(ctx context.Context, pred *models.EndpointHealthPrediction) error
	GetPrediction(ctx context.Context, id uuid.UUID) (*models.EndpointHealthPrediction, error)
	GetPredictionsByEndpoint(ctx context.Context, endpointID uuid.UUID, limit int) ([]*models.EndpointHealthPrediction, error)
	GetRecentPredictions(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.EndpointHealthPrediction, error)
	UpdatePredictionAccuracy(ctx context.Context, id uuid.UUID, wasAccurate bool) error
	UpdatePredictionAction(ctx context.Context, id uuid.UUID, action string) error

	// Behavior patterns
	CreateBehaviorPattern(ctx context.Context, pattern *models.EndpointBehaviorPattern) error
	GetBehaviorPatterns(ctx context.Context, endpointID uuid.UUID) ([]*models.EndpointBehaviorPattern, error)
	DeleteOldPatterns(ctx context.Context, endpointID uuid.UUID, beforeTime time.Time) error

	// Remediation rules
	CreateRemediationRule(ctx context.Context, rule *models.AutoRemediationRule) error
	GetRemediationRule(ctx context.Context, id uuid.UUID) (*models.AutoRemediationRule, error)
	GetRemediationRulesByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.AutoRemediationRule, error)
	GetActiveRemediationRules(ctx context.Context, tenantID uuid.UUID) ([]*models.AutoRemediationRule, error)
	UpdateRemediationRule(ctx context.Context, rule *models.AutoRemediationRule) error
	IncrementRuleTriggerCount(ctx context.Context, id uuid.UUID) error
	DeleteRemediationRule(ctx context.Context, id uuid.UUID) error

	// Remediation actions
	CreateRemediationAction(ctx context.Context, action *models.RemediationAction) error
	GetRemediationAction(ctx context.Context, id uuid.UUID) (*models.RemediationAction, error)
	GetRemediationActionsByEndpoint(ctx context.Context, endpointID uuid.UUID, limit int) ([]*models.RemediationAction, error)
	GetRecentRemediationActions(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.RemediationAction, error)
	UpdateRemediationActionOutcome(ctx context.Context, id uuid.UUID, outcome string) error
	RevertRemediationAction(ctx context.Context, id uuid.UUID) error
	CountActionsToday(ctx context.Context, tenantID uuid.UUID) (int, error)

	// Optimization suggestions
	CreateSuggestion(ctx context.Context, suggestion *models.EndpointOptimizationSuggestion) error
	GetSuggestion(ctx context.Context, id uuid.UUID) (*models.EndpointOptimizationSuggestion, error)
	GetSuggestionsByEndpoint(ctx context.Context, endpointID uuid.UUID) ([]*models.EndpointOptimizationSuggestion, error)
	GetPendingSuggestions(ctx context.Context, tenantID uuid.UUID) ([]*models.EndpointOptimizationSuggestion, error)
	UpdateSuggestionStatus(ctx context.Context, id uuid.UUID, status string) error
	CountPendingSuggestions(ctx context.Context, tenantID uuid.UUID) (int, error)

	// Circuit breakers
	GetOrCreateCircuitBreaker(ctx context.Context, tenantID, endpointID uuid.UUID) (*models.EndpointCircuitBreaker, error)
	UpdateCircuitBreakerState(ctx context.Context, id uuid.UUID, state string, failureCount, successCount int) error
	UpdateCircuitBreakerConfig(ctx context.Context, id uuid.UUID, resetTimeout, failureThreshold, successThreshold int) error
	GetOpenCircuitBreakers(ctx context.Context, tenantID uuid.UUID) ([]*models.EndpointCircuitBreaker, error)
	RecordCircuitBreakerFailure(ctx context.Context, endpointID uuid.UUID) error
	RecordCircuitBreakerSuccess(ctx context.Context, endpointID uuid.UUID) error
	CountOpenCircuitBreakers(ctx context.Context, tenantID uuid.UUID) (int, error)
}

// PostgresSelfHealingRepository implements SelfHealingRepository
type PostgresSelfHealingRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresSelfHealingRepository creates a new self-healing repository
func NewPostgresSelfHealingRepository(pool *pgxpool.Pool) *PostgresSelfHealingRepository {
	return &PostgresSelfHealingRepository{pool: pool}
}

// Prediction operations

func (r *PostgresSelfHealingRepository) CreatePrediction(ctx context.Context, pred *models.EndpointHealthPrediction) error {
	query := `
		INSERT INTO endpoint_health_predictions (
			id, tenant_id, endpoint_id, prediction_type, probability, confidence,
			predicted_time, features_used, model_version, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, CURRENT_TIMESTAMP
		)
	`

	if pred.ID == uuid.Nil {
		pred.ID = uuid.New()
	}

	featuresJSON, _ := json.Marshal(pred.FeaturesUsed)

	_, err := r.pool.Exec(ctx, query,
		pred.ID, pred.TenantID, pred.EndpointID, pred.PredictionType,
		pred.Probability, pred.Confidence, pred.PredictedTime,
		featuresJSON, pred.ModelVersion,
	)

	return err
}

func (r *PostgresSelfHealingRepository) GetPrediction(ctx context.Context, id uuid.UUID) (*models.EndpointHealthPrediction, error) {
	query := `
		SELECT id, tenant_id, endpoint_id, prediction_type, probability, confidence,
		       predicted_time, features_used, model_version, action_taken, action_taken_at,
		       was_accurate, created_at
		FROM endpoint_health_predictions WHERE id = $1
	`

	pred := &models.EndpointHealthPrediction{}
	var featuresJSON []byte

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&pred.ID, &pred.TenantID, &pred.EndpointID, &pred.PredictionType,
		&pred.Probability, &pred.Confidence, &pred.PredictedTime,
		&featuresJSON, &pred.ModelVersion, &pred.ActionTaken, &pred.ActionTakenAt,
		&pred.WasAccurate, &pred.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("prediction not found: %w", err)
	}

	json.Unmarshal(featuresJSON, &pred.FeaturesUsed)
	return pred, nil
}

func (r *PostgresSelfHealingRepository) GetPredictionsByEndpoint(ctx context.Context, endpointID uuid.UUID, limit int) ([]*models.EndpointHealthPrediction, error) {
	query := `
		SELECT id, tenant_id, endpoint_id, prediction_type, probability, confidence,
		       predicted_time, features_used, model_version, action_taken, action_taken_at,
		       was_accurate, created_at
		FROM endpoint_health_predictions WHERE endpoint_id = $1
		ORDER BY created_at DESC LIMIT $2
	`

	return r.queryPredictions(ctx, query, endpointID, limit)
}

func (r *PostgresSelfHealingRepository) GetRecentPredictions(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.EndpointHealthPrediction, error) {
	query := `
		SELECT id, tenant_id, endpoint_id, prediction_type, probability, confidence,
		       predicted_time, features_used, model_version, action_taken, action_taken_at,
		       was_accurate, created_at
		FROM endpoint_health_predictions WHERE tenant_id = $1
		ORDER BY created_at DESC LIMIT $2
	`

	return r.queryPredictions(ctx, query, tenantID, limit)
}

func (r *PostgresSelfHealingRepository) queryPredictions(ctx context.Context, query string, args ...interface{}) ([]*models.EndpointHealthPrediction, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var predictions []*models.EndpointHealthPrediction
	for rows.Next() {
		pred := &models.EndpointHealthPrediction{}
		var featuresJSON []byte

		if err := rows.Scan(
			&pred.ID, &pred.TenantID, &pred.EndpointID, &pred.PredictionType,
			&pred.Probability, &pred.Confidence, &pred.PredictedTime,
			&featuresJSON, &pred.ModelVersion, &pred.ActionTaken, &pred.ActionTakenAt,
			&pred.WasAccurate, &pred.CreatedAt,
		); err != nil {
			return nil, err
		}

		json.Unmarshal(featuresJSON, &pred.FeaturesUsed)
		predictions = append(predictions, pred)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return predictions, nil
}

func (r *PostgresSelfHealingRepository) UpdatePredictionAccuracy(ctx context.Context, id uuid.UUID, wasAccurate bool) error {
	query := `UPDATE endpoint_health_predictions SET was_accurate = $2 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, wasAccurate)
	return err
}

func (r *PostgresSelfHealingRepository) UpdatePredictionAction(ctx context.Context, id uuid.UUID, action string) error {
	query := `UPDATE endpoint_health_predictions SET action_taken = $2, action_taken_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, action)
	return err
}

// Behavior patterns

func (r *PostgresSelfHealingRepository) CreateBehaviorPattern(ctx context.Context, pattern *models.EndpointBehaviorPattern) error {
	query := `
		INSERT INTO endpoint_behavior_patterns (
			id, tenant_id, endpoint_id, pattern_type, pattern_data, time_window_hours, calculated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP
		)
	`

	if pattern.ID == uuid.Nil {
		pattern.ID = uuid.New()
	}

	patternDataJSON, _ := json.Marshal(pattern.PatternData)

	_, err := r.pool.Exec(ctx, query,
		pattern.ID, pattern.TenantID, pattern.EndpointID,
		pattern.PatternType, patternDataJSON, pattern.TimeWindowHours,
	)

	return err
}

func (r *PostgresSelfHealingRepository) GetBehaviorPatterns(ctx context.Context, endpointID uuid.UUID) ([]*models.EndpointBehaviorPattern, error) {
	query := `
		SELECT id, tenant_id, endpoint_id, pattern_type, pattern_data, time_window_hours, calculated_at
		FROM endpoint_behavior_patterns WHERE endpoint_id = $1
		ORDER BY calculated_at DESC
	`

	rows, err := r.pool.Query(ctx, query, endpointID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patterns []*models.EndpointBehaviorPattern
	for rows.Next() {
		p := &models.EndpointBehaviorPattern{}
		var patternDataJSON []byte

		if err := rows.Scan(
			&p.ID, &p.TenantID, &p.EndpointID, &p.PatternType,
			&patternDataJSON, &p.TimeWindowHours, &p.CalculatedAt,
		); err != nil {
			return nil, err
		}

		json.Unmarshal(patternDataJSON, &p.PatternData)
		patterns = append(patterns, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return patterns, nil
}

func (r *PostgresSelfHealingRepository) DeleteOldPatterns(ctx context.Context, endpointID uuid.UUID, beforeTime time.Time) error {
	query := `DELETE FROM endpoint_behavior_patterns WHERE endpoint_id = $1 AND calculated_at < $2`
	_, err := r.pool.Exec(ctx, query, endpointID, beforeTime)
	return err
}

// Remediation rules

func (r *PostgresSelfHealingRepository) CreateRemediationRule(ctx context.Context, rule *models.AutoRemediationRule) error {
	query := `
		INSERT INTO auto_remediation_rules (
			id, tenant_id, name, description, trigger_condition, action_type,
			action_config, enabled, cooldown_minutes, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)
	`

	if rule.ID == uuid.Nil {
		rule.ID = uuid.New()
	}
	if rule.CooldownMinutes == 0 {
		rule.CooldownMinutes = 30
	}

	triggerJSON, _ := json.Marshal(rule.TriggerCondition)
	actionConfigJSON, _ := json.Marshal(rule.ActionConfig)

	_, err := r.pool.Exec(ctx, query,
		rule.ID, rule.TenantID, rule.Name, rule.Description,
		triggerJSON, rule.ActionType, actionConfigJSON,
		rule.Enabled, rule.CooldownMinutes,
	)

	return err
}

func (r *PostgresSelfHealingRepository) GetRemediationRule(ctx context.Context, id uuid.UUID) (*models.AutoRemediationRule, error) {
	query := `
		SELECT id, tenant_id, name, description, trigger_condition, action_type,
		       action_config, enabled, cooldown_minutes, last_triggered, trigger_count,
		       created_at, updated_at
		FROM auto_remediation_rules WHERE id = $1
	`

	rule := &models.AutoRemediationRule{}
	var triggerJSON, actionConfigJSON []byte

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&rule.ID, &rule.TenantID, &rule.Name, &rule.Description,
		&triggerJSON, &rule.ActionType, &actionConfigJSON,
		&rule.Enabled, &rule.CooldownMinutes, &rule.LastTriggered,
		&rule.TriggerCount, &rule.CreatedAt, &rule.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("rule not found: %w", err)
	}

	json.Unmarshal(triggerJSON, &rule.TriggerCondition)
	json.Unmarshal(actionConfigJSON, &rule.ActionConfig)

	return rule, nil
}

func (r *PostgresSelfHealingRepository) GetRemediationRulesByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.AutoRemediationRule, error) {
	query := `
		SELECT id, tenant_id, name, description, trigger_condition, action_type,
		       action_config, enabled, cooldown_minutes, last_triggered, trigger_count,
		       created_at, updated_at
		FROM auto_remediation_rules WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	return r.queryRules(ctx, query, tenantID)
}

func (r *PostgresSelfHealingRepository) GetActiveRemediationRules(ctx context.Context, tenantID uuid.UUID) ([]*models.AutoRemediationRule, error) {
	query := `
		SELECT id, tenant_id, name, description, trigger_condition, action_type,
		       action_config, enabled, cooldown_minutes, last_triggered, trigger_count,
		       created_at, updated_at
		FROM auto_remediation_rules WHERE tenant_id = $1 AND enabled = true
		ORDER BY created_at DESC
	`

	return r.queryRules(ctx, query, tenantID)
}

func (r *PostgresSelfHealingRepository) queryRules(ctx context.Context, query string, arg interface{}) ([]*models.AutoRemediationRule, error) {
	rows, err := r.pool.Query(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*models.AutoRemediationRule
	for rows.Next() {
		rule := &models.AutoRemediationRule{}
		var triggerJSON, actionConfigJSON []byte

		if err := rows.Scan(
			&rule.ID, &rule.TenantID, &rule.Name, &rule.Description,
			&triggerJSON, &rule.ActionType, &actionConfigJSON,
			&rule.Enabled, &rule.CooldownMinutes, &rule.LastTriggered,
			&rule.TriggerCount, &rule.CreatedAt, &rule.UpdatedAt,
		); err != nil {
			return nil, err
		}

		json.Unmarshal(triggerJSON, &rule.TriggerCondition)
		json.Unmarshal(actionConfigJSON, &rule.ActionConfig)
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return rules, nil
}

func (r *PostgresSelfHealingRepository) UpdateRemediationRule(ctx context.Context, rule *models.AutoRemediationRule) error {
	query := `
		UPDATE auto_remediation_rules
		SET name = $2, description = $3, trigger_condition = $4, action_type = $5,
		    action_config = $6, enabled = $7, cooldown_minutes = $8, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	triggerJSON, _ := json.Marshal(rule.TriggerCondition)
	actionConfigJSON, _ := json.Marshal(rule.ActionConfig)

	_, err := r.pool.Exec(ctx, query,
		rule.ID, rule.Name, rule.Description, triggerJSON,
		rule.ActionType, actionConfigJSON, rule.Enabled, rule.CooldownMinutes,
	)

	return err
}

func (r *PostgresSelfHealingRepository) IncrementRuleTriggerCount(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE auto_remediation_rules
		SET trigger_count = trigger_count + 1, last_triggered = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

func (r *PostgresSelfHealingRepository) DeleteRemediationRule(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM auto_remediation_rules WHERE id = $1", id)
	return err
}

// Remediation actions

func (r *PostgresSelfHealingRepository) CreateRemediationAction(ctx context.Context, action *models.RemediationAction) error {
	query := `
		INSERT INTO remediation_actions (
			id, tenant_id, endpoint_id, rule_id, prediction_id, action_type,
			action_details, previous_state, new_state, triggered_by, outcome, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, CURRENT_TIMESTAMP
		)
	`

	if action.ID == uuid.Nil {
		action.ID = uuid.New()
	}
	if action.Outcome == "" {
		action.Outcome = models.RemediationOutcomePending
	}

	actionDetailsJSON, _ := json.Marshal(action.ActionDetails)
	previousStateJSON, _ := json.Marshal(action.PreviousState)
	newStateJSON, _ := json.Marshal(action.NewState)

	_, err := r.pool.Exec(ctx, query,
		action.ID, action.TenantID, action.EndpointID, action.RuleID,
		action.PredictionID, action.ActionType, actionDetailsJSON,
		previousStateJSON, newStateJSON, action.TriggeredBy, action.Outcome,
	)

	return err
}

func (r *PostgresSelfHealingRepository) GetRemediationAction(ctx context.Context, id uuid.UUID) (*models.RemediationAction, error) {
	query := `
		SELECT id, tenant_id, endpoint_id, rule_id, prediction_id, action_type,
		       action_details, previous_state, new_state, triggered_by, outcome,
		       reverted_at, created_at
		FROM remediation_actions WHERE id = $1
	`

	action := &models.RemediationAction{}
	var actionDetailsJSON, previousStateJSON, newStateJSON []byte

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&action.ID, &action.TenantID, &action.EndpointID, &action.RuleID,
		&action.PredictionID, &action.ActionType, &actionDetailsJSON,
		&previousStateJSON, &newStateJSON, &action.TriggeredBy, &action.Outcome,
		&action.RevertedAt, &action.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	json.Unmarshal(actionDetailsJSON, &action.ActionDetails)
	json.Unmarshal(previousStateJSON, &action.PreviousState)
	json.Unmarshal(newStateJSON, &action.NewState)

	return action, nil
}

func (r *PostgresSelfHealingRepository) GetRemediationActionsByEndpoint(ctx context.Context, endpointID uuid.UUID, limit int) ([]*models.RemediationAction, error) {
	query := `
		SELECT id, tenant_id, endpoint_id, rule_id, prediction_id, action_type,
		       action_details, previous_state, new_state, triggered_by, outcome,
		       reverted_at, created_at
		FROM remediation_actions WHERE endpoint_id = $1
		ORDER BY created_at DESC LIMIT $2
	`

	return r.queryActions(ctx, query, endpointID, limit)
}

func (r *PostgresSelfHealingRepository) GetRecentRemediationActions(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.RemediationAction, error) {
	query := `
		SELECT id, tenant_id, endpoint_id, rule_id, prediction_id, action_type,
		       action_details, previous_state, new_state, triggered_by, outcome,
		       reverted_at, created_at
		FROM remediation_actions WHERE tenant_id = $1
		ORDER BY created_at DESC LIMIT $2
	`

	return r.queryActions(ctx, query, tenantID, limit)
}

func (r *PostgresSelfHealingRepository) queryActions(ctx context.Context, query string, args ...interface{}) ([]*models.RemediationAction, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []*models.RemediationAction
	for rows.Next() {
		action := &models.RemediationAction{}
		var actionDetailsJSON, previousStateJSON, newStateJSON []byte

		if err := rows.Scan(
			&action.ID, &action.TenantID, &action.EndpointID, &action.RuleID,
			&action.PredictionID, &action.ActionType, &actionDetailsJSON,
			&previousStateJSON, &newStateJSON, &action.TriggeredBy, &action.Outcome,
			&action.RevertedAt, &action.CreatedAt,
		); err != nil {
			return nil, err
		}

		json.Unmarshal(actionDetailsJSON, &action.ActionDetails)
		json.Unmarshal(previousStateJSON, &action.PreviousState)
		json.Unmarshal(newStateJSON, &action.NewState)
		actions = append(actions, action)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return actions, nil
}

func (r *PostgresSelfHealingRepository) UpdateRemediationActionOutcome(ctx context.Context, id uuid.UUID, outcome string) error {
	query := `UPDATE remediation_actions SET outcome = $2 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, outcome)
	return err
}

func (r *PostgresSelfHealingRepository) RevertRemediationAction(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE remediation_actions SET outcome = 'reverted', reverted_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

func (r *PostgresSelfHealingRepository) CountActionsToday(ctx context.Context, tenantID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM remediation_actions WHERE tenant_id = $1 AND created_at >= CURRENT_DATE`
	var count int
	err := r.pool.QueryRow(ctx, query, tenantID).Scan(&count)
	return count, err
}

// Optimization suggestions

func (r *PostgresSelfHealingRepository) CreateSuggestion(ctx context.Context, suggestion *models.EndpointOptimizationSuggestion) error {
	query := `
		INSERT INTO endpoint_optimization_suggestions (
			id, tenant_id, endpoint_id, suggestion_type, current_config,
			suggested_config, expected_improvement, confidence, rationale,
			status, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, CURRENT_TIMESTAMP
		)
	`

	if suggestion.ID == uuid.Nil {
		suggestion.ID = uuid.New()
	}
	if suggestion.Status == "" {
		suggestion.Status = models.SuggestionStatusPending
	}

	currentConfigJSON, _ := json.Marshal(suggestion.CurrentConfig)
	suggestedConfigJSON, _ := json.Marshal(suggestion.SuggestedConfig)

	_, err := r.pool.Exec(ctx, query,
		suggestion.ID, suggestion.TenantID, suggestion.EndpointID,
		suggestion.SuggestionType, currentConfigJSON, suggestedConfigJSON,
		suggestion.ExpectedImprovement, suggestion.Confidence, suggestion.Rationale,
		suggestion.Status,
	)

	return err
}

func (r *PostgresSelfHealingRepository) GetSuggestion(ctx context.Context, id uuid.UUID) (*models.EndpointOptimizationSuggestion, error) {
	query := `
		SELECT id, tenant_id, endpoint_id, suggestion_type, current_config,
		       suggested_config, expected_improvement, confidence, rationale,
		       status, applied_at, created_at
		FROM endpoint_optimization_suggestions WHERE id = $1
	`

	s := &models.EndpointOptimizationSuggestion{}
	var currentConfigJSON, suggestedConfigJSON []byte

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.TenantID, &s.EndpointID, &s.SuggestionType,
		&currentConfigJSON, &suggestedConfigJSON, &s.ExpectedImprovement,
		&s.Confidence, &s.Rationale, &s.Status, &s.AppliedAt, &s.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	json.Unmarshal(currentConfigJSON, &s.CurrentConfig)
	json.Unmarshal(suggestedConfigJSON, &s.SuggestedConfig)

	return s, nil
}

func (r *PostgresSelfHealingRepository) GetSuggestionsByEndpoint(ctx context.Context, endpointID uuid.UUID) ([]*models.EndpointOptimizationSuggestion, error) {
	query := `
		SELECT id, tenant_id, endpoint_id, suggestion_type, current_config,
		       suggested_config, expected_improvement, confidence, rationale,
		       status, applied_at, created_at
		FROM endpoint_optimization_suggestions WHERE endpoint_id = $1
		ORDER BY created_at DESC
	`

	return r.querySuggestions(ctx, query, endpointID)
}

func (r *PostgresSelfHealingRepository) GetPendingSuggestions(ctx context.Context, tenantID uuid.UUID) ([]*models.EndpointOptimizationSuggestion, error) {
	query := `
		SELECT id, tenant_id, endpoint_id, suggestion_type, current_config,
		       suggested_config, expected_improvement, confidence, rationale,
		       status, applied_at, created_at
		FROM endpoint_optimization_suggestions WHERE tenant_id = $1 AND status = 'pending'
		ORDER BY confidence DESC, created_at DESC
	`

	return r.querySuggestions(ctx, query, tenantID)
}

func (r *PostgresSelfHealingRepository) querySuggestions(ctx context.Context, query string, arg interface{}) ([]*models.EndpointOptimizationSuggestion, error) {
	rows, err := r.pool.Query(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suggestions []*models.EndpointOptimizationSuggestion
	for rows.Next() {
		s := &models.EndpointOptimizationSuggestion{}
		var currentConfigJSON, suggestedConfigJSON []byte

		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.EndpointID, &s.SuggestionType,
			&currentConfigJSON, &suggestedConfigJSON, &s.ExpectedImprovement,
			&s.Confidence, &s.Rationale, &s.Status, &s.AppliedAt, &s.CreatedAt,
		); err != nil {
			return nil, err
		}

		json.Unmarshal(currentConfigJSON, &s.CurrentConfig)
		json.Unmarshal(suggestedConfigJSON, &s.SuggestedConfig)
		suggestions = append(suggestions, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return suggestions, nil
}

func (r *PostgresSelfHealingRepository) UpdateSuggestionStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `UPDATE endpoint_optimization_suggestions SET status = $2, applied_at = CASE WHEN $2 = 'applied' THEN CURRENT_TIMESTAMP ELSE applied_at END WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, status)
	return err
}

func (r *PostgresSelfHealingRepository) CountPendingSuggestions(ctx context.Context, tenantID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM endpoint_optimization_suggestions WHERE tenant_id = $1 AND status = 'pending'`
	var count int
	err := r.pool.QueryRow(ctx, query, tenantID).Scan(&count)
	return count, err
}

// Circuit breakers

func (r *PostgresSelfHealingRepository) GetOrCreateCircuitBreaker(ctx context.Context, tenantID, endpointID uuid.UUID) (*models.EndpointCircuitBreaker, error) {
	// Try to get existing
	query := `
		SELECT id, tenant_id, endpoint_id, state, failure_count, success_count,
		       last_failure_at, last_success_at, opened_at, half_open_at,
		       reset_timeout_seconds, failure_threshold, success_threshold,
		       created_at, updated_at
		FROM endpoint_circuit_breakers WHERE endpoint_id = $1
	`

	cb := &models.EndpointCircuitBreaker{}
	err := r.pool.QueryRow(ctx, query, endpointID).Scan(
		&cb.ID, &cb.TenantID, &cb.EndpointID, &cb.State,
		&cb.FailureCount, &cb.SuccessCount, &cb.LastFailureAt, &cb.LastSuccessAt,
		&cb.OpenedAt, &cb.HalfOpenAt, &cb.ResetTimeoutSeconds,
		&cb.FailureThreshold, &cb.SuccessThreshold, &cb.CreatedAt, &cb.UpdatedAt,
	)

	if err == nil {
		return cb, nil
	}

	// Create new
	cb = &models.EndpointCircuitBreaker{
		ID:                  uuid.New(),
		TenantID:            tenantID,
		EndpointID:          endpointID,
		State:               models.CircuitStateClosed,
		ResetTimeoutSeconds: 60,
		FailureThreshold:    5,
		SuccessThreshold:    3,
	}

	insertQuery := `
		INSERT INTO endpoint_circuit_breakers (
			id, tenant_id, endpoint_id, state, failure_count, success_count,
			reset_timeout_seconds, failure_threshold, success_threshold,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, 0, 0, $5, $6, $7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		) ON CONFLICT (endpoint_id) DO NOTHING
	`

	_, err = r.pool.Exec(ctx, insertQuery,
		cb.ID, cb.TenantID, cb.EndpointID, cb.State,
		cb.ResetTimeoutSeconds, cb.FailureThreshold, cb.SuccessThreshold,
	)

	if err != nil {
		return nil, err
	}

	return cb, nil
}

func (r *PostgresSelfHealingRepository) UpdateCircuitBreakerState(ctx context.Context, id uuid.UUID, state string, failureCount, successCount int) error {
	query := `
		UPDATE endpoint_circuit_breakers
		SET state = $2, failure_count = $3, success_count = $4,
		    opened_at = CASE WHEN $2 = 'open' AND state != 'open' THEN CURRENT_TIMESTAMP ELSE opened_at END,
		    half_open_at = CASE WHEN $2 = 'half_open' AND state != 'half_open' THEN CURRENT_TIMESTAMP ELSE half_open_at END,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, id, state, failureCount, successCount)
	return err
}

func (r *PostgresSelfHealingRepository) UpdateCircuitBreakerConfig(ctx context.Context, id uuid.UUID, resetTimeout, failureThreshold, successThreshold int) error {
	query := `
		UPDATE endpoint_circuit_breakers
		SET reset_timeout_seconds = COALESCE(NULLIF($2, 0), reset_timeout_seconds),
		    failure_threshold = COALESCE(NULLIF($3, 0), failure_threshold),
		    success_threshold = COALESCE(NULLIF($4, 0), success_threshold),
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, id, resetTimeout, failureThreshold, successThreshold)
	return err
}

func (r *PostgresSelfHealingRepository) GetOpenCircuitBreakers(ctx context.Context, tenantID uuid.UUID) ([]*models.EndpointCircuitBreaker, error) {
	query := `
		SELECT id, tenant_id, endpoint_id, state, failure_count, success_count,
		       last_failure_at, last_success_at, opened_at, half_open_at,
		       reset_timeout_seconds, failure_threshold, success_threshold,
		       created_at, updated_at
		FROM endpoint_circuit_breakers WHERE tenant_id = $1 AND state = 'open'
	`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cbs []*models.EndpointCircuitBreaker
	for rows.Next() {
		cb := &models.EndpointCircuitBreaker{}
		if err := rows.Scan(
			&cb.ID, &cb.TenantID, &cb.EndpointID, &cb.State,
			&cb.FailureCount, &cb.SuccessCount, &cb.LastFailureAt, &cb.LastSuccessAt,
			&cb.OpenedAt, &cb.HalfOpenAt, &cb.ResetTimeoutSeconds,
			&cb.FailureThreshold, &cb.SuccessThreshold, &cb.CreatedAt, &cb.UpdatedAt,
		); err != nil {
			return nil, err
		}
		cbs = append(cbs, cb)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return cbs, nil
}

func (r *PostgresSelfHealingRepository) RecordCircuitBreakerFailure(ctx context.Context, endpointID uuid.UUID) error {
	query := `
		UPDATE endpoint_circuit_breakers
		SET failure_count = failure_count + 1, last_failure_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE endpoint_id = $1
	`
	_, err := r.pool.Exec(ctx, query, endpointID)
	return err
}

func (r *PostgresSelfHealingRepository) RecordCircuitBreakerSuccess(ctx context.Context, endpointID uuid.UUID) error {
	query := `
		UPDATE endpoint_circuit_breakers
		SET success_count = success_count + 1, last_success_at = CURRENT_TIMESTAMP, failure_count = 0, updated_at = CURRENT_TIMESTAMP
		WHERE endpoint_id = $1
	`
	_, err := r.pool.Exec(ctx, query, endpointID)
	return err
}

func (r *PostgresSelfHealingRepository) CountOpenCircuitBreakers(ctx context.Context, tenantID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM endpoint_circuit_breakers WHERE tenant_id = $1 AND state = 'open'`
	var count int
	err := r.pool.QueryRow(ctx, query, tenantID).Scan(&count)
	return count, err
}
