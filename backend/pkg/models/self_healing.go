package models

import (
	"time"

	"github.com/google/uuid"
)

// Prediction type constants
const (
	PredictionTypeFailure     = "failure"
	PredictionTypeDegradation = "degradation"
	PredictionTypeRecovery    = "recovery"
)

// Remediation action type constants
const (
	RemediationDisableEndpoint = "disable_endpoint"
	RemediationAdjustRetry     = "adjust_retry"
	RemediationNotify          = "notify"
	RemediationCircuitBreak    = "circuit_break"
	RemediationRateLimit       = "rate_limit"
)

// Remediation outcome constants
const (
	RemediationOutcomePending  = "pending"
	RemediationOutcomeSuccess  = "success"
	RemediationOutcomeFailed   = "failed"
	RemediationOutcomeReverted = "reverted"
)

// Circuit breaker state constants
const (
	CircuitStateClosed   = "closed"
	CircuitStateOpen     = "open"
	CircuitStateHalfOpen = "half_open"
)

// Suggestion status constants
const (
	SuggestionStatusPending   = "pending"
	SuggestionStatusApplied   = "applied"
	SuggestionStatusDismissed = "dismissed"
)

// EndpointHealthPrediction represents a ML prediction for endpoint health
type EndpointHealthPrediction struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	TenantID       uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	EndpointID     uuid.UUID              `json:"endpoint_id" db:"endpoint_id"`
	PredictionType string                 `json:"prediction_type" db:"prediction_type"`
	Probability    float64                `json:"probability" db:"probability"`
	Confidence     float64                `json:"confidence" db:"confidence"`
	PredictedTime  *time.Time             `json:"predicted_time,omitempty" db:"predicted_time"`
	FeaturesUsed   map[string]interface{} `json:"features_used" db:"features_used"`
	ModelVersion   string                 `json:"model_version" db:"model_version"`
	ActionTaken    string                 `json:"action_taken,omitempty" db:"action_taken"`
	ActionTakenAt  *time.Time             `json:"action_taken_at,omitempty" db:"action_taken_at"`
	WasAccurate    *bool                  `json:"was_accurate,omitempty" db:"was_accurate"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
}

// EndpointBehaviorPattern represents behavioral patterns for an endpoint
type EndpointBehaviorPattern struct {
	ID              uuid.UUID              `json:"id" db:"id"`
	TenantID        uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	EndpointID      uuid.UUID              `json:"endpoint_id" db:"endpoint_id"`
	PatternType     string                 `json:"pattern_type" db:"pattern_type"`
	PatternData     map[string]interface{} `json:"pattern_data" db:"pattern_data"`
	TimeWindowHours int                    `json:"time_window_hours" db:"time_window_hours"`
	CalculatedAt    time.Time              `json:"calculated_at" db:"calculated_at"`
}

// AutoRemediationRule defines an automatic remediation rule
type AutoRemediationRule struct {
	ID               uuid.UUID              `json:"id" db:"id"`
	TenantID         uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	Name             string                 `json:"name" db:"name"`
	Description      string                 `json:"description,omitempty" db:"description"`
	TriggerCondition map[string]interface{} `json:"trigger_condition" db:"trigger_condition"`
	ActionType       string                 `json:"action_type" db:"action_type"`
	ActionConfig     map[string]interface{} `json:"action_config" db:"action_config"`
	Enabled          bool                   `json:"enabled" db:"enabled"`
	CooldownMinutes  int                    `json:"cooldown_minutes" db:"cooldown_minutes"`
	LastTriggered    *time.Time             `json:"last_triggered,omitempty" db:"last_triggered"`
	TriggerCount     int                    `json:"trigger_count" db:"trigger_count"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
}

// RemediationAction represents a remediation action taken
type RemediationAction struct {
	ID            uuid.UUID              `json:"id" db:"id"`
	TenantID      uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	EndpointID    uuid.UUID              `json:"endpoint_id" db:"endpoint_id"`
	RuleID        *uuid.UUID             `json:"rule_id,omitempty" db:"rule_id"`
	PredictionID  *uuid.UUID             `json:"prediction_id,omitempty" db:"prediction_id"`
	ActionType    string                 `json:"action_type" db:"action_type"`
	ActionDetails map[string]interface{} `json:"action_details" db:"action_details"`
	PreviousState map[string]interface{} `json:"previous_state,omitempty" db:"previous_state"`
	NewState      map[string]interface{} `json:"new_state,omitempty" db:"new_state"`
	TriggeredBy   string                 `json:"triggered_by" db:"triggered_by"`
	Outcome       string                 `json:"outcome" db:"outcome"`
	RevertedAt    *time.Time             `json:"reverted_at,omitempty" db:"reverted_at"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
}

// EndpointOptimizationSuggestion represents an optimization suggestion
type EndpointOptimizationSuggestion struct {
	ID                  uuid.UUID              `json:"id" db:"id"`
	TenantID            uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	EndpointID          uuid.UUID              `json:"endpoint_id" db:"endpoint_id"`
	SuggestionType      string                 `json:"suggestion_type" db:"suggestion_type"`
	CurrentConfig       map[string]interface{} `json:"current_config" db:"current_config"`
	SuggestedConfig     map[string]interface{} `json:"suggested_config" db:"suggested_config"`
	ExpectedImprovement string                 `json:"expected_improvement,omitempty" db:"expected_improvement"`
	Confidence          float64                `json:"confidence" db:"confidence"`
	Rationale           string                 `json:"rationale,omitempty" db:"rationale"`
	Status              string                 `json:"status" db:"status"`
	AppliedAt           *time.Time             `json:"applied_at,omitempty" db:"applied_at"`
	CreatedAt           time.Time              `json:"created_at" db:"created_at"`
}

// EndpointCircuitBreaker represents circuit breaker state
type EndpointCircuitBreaker struct {
	ID                   uuid.UUID  `json:"id" db:"id"`
	TenantID             uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	EndpointID           uuid.UUID  `json:"endpoint_id" db:"endpoint_id"`
	State                string     `json:"state" db:"state"`
	FailureCount         int        `json:"failure_count" db:"failure_count"`
	SuccessCount         int        `json:"success_count" db:"success_count"`
	LastFailureAt        *time.Time `json:"last_failure_at,omitempty" db:"last_failure_at"`
	LastSuccessAt        *time.Time `json:"last_success_at,omitempty" db:"last_success_at"`
	OpenedAt             *time.Time `json:"opened_at,omitempty" db:"opened_at"`
	HalfOpenAt           *time.Time `json:"half_open_at,omitempty" db:"half_open_at"`
	ResetTimeoutSeconds  int        `json:"reset_timeout_seconds" db:"reset_timeout_seconds"`
	FailureThreshold     int        `json:"failure_threshold" db:"failure_threshold"`
	SuccessThreshold     int        `json:"success_threshold" db:"success_threshold"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at" db:"updated_at"`
}

// Request models

// CreateRemediationRuleRequest represents a request to create a remediation rule
type CreateRemediationRuleRequest struct {
	Name             string                 `json:"name" binding:"required"`
	Description      string                 `json:"description"`
	TriggerCondition map[string]interface{} `json:"trigger_condition" binding:"required"`
	ActionType       string                 `json:"action_type" binding:"required"`
	ActionConfig     map[string]interface{} `json:"action_config"`
	CooldownMinutes  int                    `json:"cooldown_minutes"`
}

// TriggerRemediationRequest represents a request to trigger remediation manually
type TriggerRemediationRequest struct {
	EndpointID    string                 `json:"endpoint_id" binding:"required"`
	ActionType    string                 `json:"action_type" binding:"required"`
	ActionDetails map[string]interface{} `json:"action_details"`
}

// UpdateCircuitBreakerRequest represents a request to update circuit breaker config
type UpdateCircuitBreakerRequest struct {
	EndpointID           string `json:"endpoint_id" binding:"required"`
	ResetTimeoutSeconds  int    `json:"reset_timeout_seconds"`
	FailureThreshold     int    `json:"failure_threshold"`
	SuccessThreshold     int    `json:"success_threshold"`
}

// ApplySuggestionRequest represents a request to apply an optimization suggestion
type ApplySuggestionRequest struct {
	SuggestionID string `json:"suggestion_id" binding:"required"`
}

// EndpointHealthAnalysis represents comprehensive health analysis
type EndpointHealthAnalysis struct {
	EndpointID            uuid.UUID                         `json:"endpoint_id"`
	HealthScore           float64                           `json:"health_score"`
	Status                string                            `json:"status"`
	FailureProbability    float64                           `json:"failure_probability"`
	CircuitBreakerState   string                            `json:"circuit_breaker_state"`
	RecentPredictions     []*EndpointHealthPrediction       `json:"recent_predictions"`
	PendingSuggestions    []*EndpointOptimizationSuggestion `json:"pending_suggestions"`
	RecentActions         []*RemediationAction              `json:"recent_actions"`
	Metrics               *EndpointHealthMetrics            `json:"metrics"`
	RecommendedActions    []string                          `json:"recommended_actions"`
}

// EndpointHealthMetrics represents health-related metrics
type EndpointHealthMetrics struct {
	SuccessRate24h     float64 `json:"success_rate_24h"`
	AvgResponseTimeMs  int64   `json:"avg_response_time_ms"`
	P95ResponseTimeMs  int64   `json:"p95_response_time_ms"`
	ErrorRate24h       float64 `json:"error_rate_24h"`
	TotalRequests24h   int     `json:"total_requests_24h"`
	FailedRequests24h  int     `json:"failed_requests_24h"`
	UptimePercent      float64 `json:"uptime_percent"`
	LastSuccessAt      *time.Time `json:"last_success_at,omitempty"`
	LastFailureAt      *time.Time `json:"last_failure_at,omitempty"`
}

// SelfHealingDashboard represents the self-healing dashboard data
type SelfHealingDashboard struct {
	TotalEndpoints       int                         `json:"total_endpoints"`
	HealthyEndpoints     int                         `json:"healthy_endpoints"`
	DegradedEndpoints    int                         `json:"degraded_endpoints"`
	UnhealthyEndpoints   int                         `json:"unhealthy_endpoints"`
	CircuitBreakersOpen  int                         `json:"circuit_breakers_open"`
	PredictionsToday     int                         `json:"predictions_today"`
	ActionsToday         int                         `json:"actions_today"`
	PendingSuggestions   int                         `json:"pending_suggestions"`
	ActiveRules          int                         `json:"active_rules"`
	RecentPredictions    []*EndpointHealthPrediction `json:"recent_predictions"`
	RecentActions        []*RemediationAction        `json:"recent_actions"`
	TopAtRiskEndpoints   []*EndpointHealthAnalysis   `json:"top_at_risk_endpoints"`
}

// MLFeatureVector represents features for ML prediction
type MLFeatureVector struct {
	EndpointID          uuid.UUID `json:"endpoint_id"`
	SuccessRate1h       float64   `json:"success_rate_1h"`
	SuccessRate24h      float64   `json:"success_rate_24h"`
	AvgResponseTime1h   float64   `json:"avg_response_time_1h"`
	AvgResponseTime24h  float64   `json:"avg_response_time_24h"`
	ErrorRate1h         float64   `json:"error_rate_1h"`
	ErrorRate24h        float64   `json:"error_rate_24h"`
	RequestVolume1h     int       `json:"request_volume_1h"`
	RequestVolume24h    int       `json:"request_volume_24h"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	TimeSinceLastFailure float64  `json:"time_since_last_failure"`
	DayOfWeek           int       `json:"day_of_week"`
	HourOfDay           int       `json:"hour_of_day"`
}
