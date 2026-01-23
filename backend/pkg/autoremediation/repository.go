package autoremediation

import "context"

// Repository defines the data access interface for auto-remediation
type Repository interface {
	// Patterns
	CreatePattern(ctx context.Context, pattern *FailurePattern) error
	GetPattern(ctx context.Context, tenantID, patternID string) (*FailurePattern, error)
	ListPatterns(ctx context.Context, tenantID string) ([]FailurePattern, error)
	UpdatePattern(ctx context.Context, pattern *FailurePattern) error

	// Rules
	CreateRule(ctx context.Context, rule *RemediationRule) error
	GetRule(ctx context.Context, tenantID, ruleID string) (*RemediationRule, error)
	ListRulesByPattern(ctx context.Context, tenantID, patternID string) ([]RemediationRule, error)
	ListRules(ctx context.Context, tenantID string) ([]RemediationRule, error)
	UpdateRule(ctx context.Context, rule *RemediationRule) error

	// Actions
	CreateAction(ctx context.Context, action *RemediationAction) error
	GetAction(ctx context.Context, tenantID, actionID string) (*RemediationAction, error)
	ListActionsByRule(ctx context.Context, tenantID, ruleID string) ([]RemediationAction, error)
	ListActions(ctx context.Context, tenantID string, limit, offset int) ([]RemediationAction, error)
	UpdateAction(ctx context.Context, action *RemediationAction) error

	// Health stats
	GetEndpointHealthStats(ctx context.Context, tenantID string) ([]HealthPrediction, error)
}
