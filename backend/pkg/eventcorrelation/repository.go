package eventcorrelation

import (
	"context"
	"fmt"
)

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

// MemoryRepository provides an in-memory implementation for testing.
type MemoryRepository struct {
	rules   map[string]*CorrelationRule
	states  map[string]*CorrelationState
	matches []*CorrelationMatch
}

// NewMemoryRepository creates a new in-memory repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		rules:  make(map[string]*CorrelationRule),
		states: make(map[string]*CorrelationState),
	}
}

func (r *MemoryRepository) CreateRule(_ context.Context, rule *CorrelationRule) error {
	r.rules[rule.ID] = rule
	return nil
}

func (r *MemoryRepository) GetRule(_ context.Context, tenantID, ruleID string) (*CorrelationRule, error) {
	if rule, ok := r.rules[ruleID]; ok && rule.TenantID == tenantID {
		return rule, nil
	}
	return nil, fmt.Errorf("rule not found: %s", ruleID)
}

func (r *MemoryRepository) ListRules(_ context.Context, tenantID string) ([]CorrelationRule, error) {
	var result []CorrelationRule
	for _, rule := range r.rules {
		if rule.TenantID == tenantID {
			result = append(result, *rule)
		}
	}
	return result, nil
}

func (r *MemoryRepository) UpdateRule(_ context.Context, rule *CorrelationRule) error {
	r.rules[rule.ID] = rule
	return nil
}

func (r *MemoryRepository) DeleteRule(_ context.Context, tenantID, ruleID string) error {
	delete(r.rules, ruleID)
	return nil
}

func (r *MemoryRepository) GetEnabledRulesByTrigger(_ context.Context, tenantID, eventType string) ([]CorrelationRule, error) {
	var result []CorrelationRule
	for _, rule := range r.rules {
		if rule.TenantID == tenantID && rule.TriggerEvent == eventType && rule.IsEnabled {
			result = append(result, *rule)
		}
	}
	return result, nil
}

func (r *MemoryRepository) GetEnabledRulesByFollow(_ context.Context, tenantID, eventType string) ([]CorrelationRule, error) {
	var result []CorrelationRule
	for _, rule := range r.rules {
		if rule.TenantID == tenantID && rule.FollowEvent == eventType && rule.IsEnabled {
			result = append(result, *rule)
		}
	}
	return result, nil
}

func (r *MemoryRepository) CreateState(_ context.Context, state *CorrelationState) error {
	r.states[state.ID] = state
	return nil
}

func (r *MemoryRepository) FindPendingState(_ context.Context, ruleID, matchKey string) (*CorrelationState, error) {
	for _, state := range r.states {
		if state.RuleID == ruleID && state.MatchKey == matchKey && state.Status == StatePending {
			return state, nil
		}
	}
	return nil, fmt.Errorf("no pending state found")
}

func (r *MemoryRepository) UpdateState(_ context.Context, state *CorrelationState) error {
	r.states[state.ID] = state
	return nil
}

func (r *MemoryRepository) ExpireStates(_ context.Context) (int, error) {
	return 0, nil
}

func (r *MemoryRepository) CreateMatch(_ context.Context, match *CorrelationMatch) error {
	r.matches = append(r.matches, match)
	return nil
}

func (r *MemoryRepository) ListMatches(_ context.Context, tenantID string, limit, offset int) ([]CorrelationMatch, error) {
	var result []CorrelationMatch
	for _, m := range r.matches {
		if m.TenantID == tenantID {
			result = append(result, *m)
		}
	}
	return result, nil
}

func (r *MemoryRepository) GetStats(_ context.Context, tenantID string) (*CorrelationStats, error) {
	return &CorrelationStats{}, nil
}
