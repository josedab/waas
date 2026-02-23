package eventcorrelation

import "context"

// Repository defines the data access interface for event correlation.
type Repository interface {
	CreateRule(ctx context.Context, rule *CorrelationRule) error
	GetRule(ctx context.Context, tenantID, ruleID string) (*CorrelationRule, error)
	ListRules(ctx context.Context, tenantID string) ([]CorrelationRule, error)
	UpdateRule(ctx context.Context, rule *CorrelationRule) error
	DeleteRule(ctx context.Context, tenantID, ruleID string) error
	GetEnabledRulesByTrigger(ctx context.Context, tenantID, eventType string) ([]CorrelationRule, error)
	GetEnabledRulesByFollow(ctx context.Context, tenantID, eventType string) ([]CorrelationRule, error)

	CreateState(ctx context.Context, state *CorrelationState) error
	FindPendingState(ctx context.Context, ruleID, matchKey string) (*CorrelationState, error)
	UpdateState(ctx context.Context, state *CorrelationState) error
	ExpireStates(ctx context.Context) (int, error)

	CreateMatch(ctx context.Context, match *CorrelationMatch) error
	ListMatches(ctx context.Context, tenantID string, limit, offset int) ([]CorrelationMatch, error)
	GetStats(ctx context.Context, tenantID string) (*CorrelationStats, error)
}
