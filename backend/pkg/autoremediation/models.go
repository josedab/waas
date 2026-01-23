package autoremediation

import "time"

// ActionType constants
const (
	ActionTypeRetryStrategyChange = "retry_strategy_change"
	ActionTypeTransformFix        = "transform_fix"
	ActionTypeEndpointDisable     = "endpoint_disable"
	ActionTypeAlert               = "alert"
)

// PatternStatus constants
const (
	PatternStatusActive   = "active"
	PatternStatusResolved = "resolved"
	PatternStatusIgnored  = "ignored"
)

// ActionStatus constants
const (
	ActionStatusPending  = "pending"
	ActionStatusApplied  = "applied"
	ActionStatusFailed   = "failed"
	ActionStatusReverted = "reverted"
)

// FailurePattern represents a detected failure pattern
type FailurePattern struct {
	ID              string    `json:"id" db:"id"`
	TenantID        string    `json:"tenant_id" db:"tenant_id"`
	PatternName     string    `json:"pattern_name" db:"pattern_name"`
	Description     string    `json:"description" db:"description"`
	EventType       string    `json:"event_type" db:"event_type"`
	ErrorCode       string    `json:"error_code" db:"error_code"`
	ErrorMessage    string    `json:"error_message" db:"error_message"`
	Frequency       int       `json:"frequency" db:"frequency"`
	FirstSeenAt     time.Time `json:"first_seen_at" db:"first_seen_at"`
	LastSeenAt      time.Time `json:"last_seen_at" db:"last_seen_at"`
	OccurrenceCount int       `json:"occurrence_count" db:"occurrence_count"`
	Status          string    `json:"status" db:"status"`
	Confidence      float64   `json:"confidence" db:"confidence"`
}

// RemediationRule defines an automated remediation rule
type RemediationRule struct {
	ID           string    `json:"id" db:"id"`
	TenantID     string    `json:"tenant_id" db:"tenant_id"`
	PatternID    string    `json:"pattern_id" db:"pattern_id"`
	Name         string    `json:"name" db:"name"`
	ActionType   string    `json:"action_type" db:"action_type"`
	ActionConfig string    `json:"action_config" db:"action_config"`
	IsAutomatic  bool      `json:"is_automatic" db:"is_automatic"`
	Priority     int       `json:"priority" db:"priority"`
	SuccessCount int       `json:"success_count" db:"success_count"`
	FailureCount int       `json:"failure_count" db:"failure_count"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// RemediationAction records a remediation action taken
type RemediationAction struct {
	ID            string     `json:"id" db:"id"`
	TenantID      string     `json:"tenant_id" db:"tenant_id"`
	RuleID        string     `json:"rule_id" db:"rule_id"`
	PatternID     string     `json:"pattern_id" db:"pattern_id"`
	ActionType    string     `json:"action_type" db:"action_type"`
	ActionDetails string     `json:"action_details" db:"action_details"`
	Status        string     `json:"status" db:"status"`
	AppliedAt     *time.Time `json:"applied_at,omitempty" db:"applied_at"`
	RevertedAt    *time.Time `json:"reverted_at,omitempty" db:"reverted_at"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
}

// Recommendation suggests a remediation for a detected pattern
type Recommendation struct {
	PatternID       string  `json:"pattern_id"`
	PatternName     string  `json:"pattern_name"`
	Confidence      float64 `json:"confidence"`
	SuggestedAction string  `json:"suggested_action"`
	SuggestedConfig string  `json:"suggested_config"`
	EstimatedImpact string  `json:"estimated_impact"`
	Reasoning       string  `json:"reasoning"`
}

// HealthPrediction predicts the future health of an endpoint
type HealthPrediction struct {
	EndpointID      string   `json:"endpoint_id"`
	EndpointURL     string   `json:"endpoint_url"`
	CurrentHealth   string   `json:"current_health"`
	PredictedHealth string   `json:"predicted_health"`
	PredictedAt     string   `json:"predicted_at"`
	ConfidencePct   float64  `json:"confidence_pct"`
	RiskFactors     []string `json:"risk_factors"`
}

// CreateRuleRequest is the request DTO for creating a remediation rule
type CreateRuleRequest struct {
	PatternID    string `json:"pattern_id" binding:"required"`
	Name         string `json:"name" binding:"required,min=1,max=255"`
	ActionType   string `json:"action_type" binding:"required,oneof=retry_strategy_change transform_fix endpoint_disable alert"`
	ActionConfig string `json:"action_config" binding:"required"`
	IsAutomatic  bool   `json:"is_automatic"`
	Priority     int    `json:"priority" binding:"min=0"`
}

// ApplyActionRequest is the request DTO for applying a remediation action
type ApplyActionRequest struct {
	RuleID        string `json:"rule_id" binding:"required"`
	ActionDetails string `json:"action_details"`
}
