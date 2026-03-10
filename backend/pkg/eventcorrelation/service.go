package eventcorrelation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service provides event correlation business logic.
type Service struct {
	repo Repository
}

// NewService creates a new event correlation service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateRule creates a new correlation rule.
func (s *Service) CreateRule(ctx context.Context, tenantID string, req *CreateRuleRequest) (*CorrelationRule, error) {
	if req.TriggerEvent == req.FollowEvent {
		return nil, fmt.Errorf("trigger_event and follow_event must be different")
	}
	if req.TimeWindowSec <= 0 {
		req.TimeWindowSec = 300 // default 5 minutes
	}
	if req.TimeWindowSec > 86400 {
		return nil, fmt.Errorf("time_window_sec cannot exceed 86400 (24 hours)")
	}

	now := time.Now()
	rule := &CorrelationRule{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		Name:           req.Name,
		Description:    req.Description,
		TriggerEvent:   req.TriggerEvent,
		FollowEvent:    req.FollowEvent,
		TimeWindowSec:  req.TimeWindowSec,
		MatchFields:    req.MatchFields,
		CompositeEvent: req.CompositeEvent,
		IsEnabled:      true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if s.repo != nil {
		if err := s.repo.CreateRule(ctx, rule); err != nil {
			return nil, fmt.Errorf("failed to create rule: %w", err)
		}
	}

	return rule, nil
}

// GetRule retrieves a correlation rule.
func (s *Service) GetRule(ctx context.Context, tenantID, ruleID string) (*CorrelationRule, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}
	return s.repo.GetRule(ctx, tenantID, ruleID)
}

// ListRules lists correlation rules for a tenant.
func (s *Service) ListRules(ctx context.Context, tenantID string) ([]CorrelationRule, error) {
	if s.repo == nil {
		return []CorrelationRule{}, nil
	}
	return s.repo.ListRules(ctx, tenantID)
}

// DeleteRule removes a correlation rule.
func (s *Service) DeleteRule(ctx context.Context, tenantID, ruleID string) error {
	if s.repo == nil {
		return fmt.Errorf("repository not configured")
	}
	return s.repo.DeleteRule(ctx, tenantID, ruleID)
}

// IngestEvent processes an incoming event against all active correlation rules,
// either creating a pending state for trigger events or matching against
// existing pending states for follow events.
func (s *Service) IngestEvent(ctx context.Context, tenantID string, req *IngestEventRequest) ([]CompositeEvent, error) {
	if s.repo == nil {
		return nil, nil
	}

	var composites []CompositeEvent

	// Check if this event is a trigger for any rules
	triggerRules, err := s.repo.GetEnabledRulesByTrigger(ctx, tenantID, req.EventType)
	if err != nil {
		return nil, fmt.Errorf("failed to query trigger rules: %w", err)
	}

	for _, rule := range triggerRules {
		matchKey := computeMatchKey(rule.ID, req.Payload, rule.MatchFields)
		state := &CorrelationState{
			ID:             uuid.New().String(),
			RuleID:         rule.ID,
			TenantID:       tenantID,
			TriggerEventID: req.EventID,
			MatchKey:       matchKey,
			Payload:        req.Payload,
			Status:         StatePending,
			ExpiresAt:      time.Now().Add(time.Duration(rule.TimeWindowSec) * time.Second),
			CreatedAt:      time.Now(),
		}
		if err := s.repo.CreateState(ctx, state); err != nil {
			return nil, fmt.Errorf("failed to persist correlation state: %w", err)
		}
	}

	// Check if this event is a follow event for any rules
	followRules, err := s.repo.GetEnabledRulesByFollow(ctx, tenantID, req.EventType)
	if err != nil {
		return nil, fmt.Errorf("failed to query follow rules: %w", err)
	}

	for _, rule := range followRules {
		matchKey := computeMatchKey(rule.ID, req.Payload, rule.MatchFields)
		state, err := s.repo.FindPendingState(ctx, rule.ID, matchKey)
		if err != nil || state == nil {
			continue
		}

		// Found a match
		now := time.Now()
		state.Status = StateCorrelated
		state.CorrelatedAt = &now
		if err := s.repo.UpdateState(ctx, state); err != nil {
			return nil, fmt.Errorf("failed to update correlation state: %w", err)
		}

		compositeID := uuid.New().String()
		match := &CorrelationMatch{
			ID:               uuid.New().String(),
			RuleID:           rule.ID,
			TenantID:         tenantID,
			TriggerEventID:   state.TriggerEventID,
			FollowEventID:    req.EventID,
			MatchKey:         matchKey,
			CompositeEventID: compositeID,
			MatchedAt:        now,
		}
		if err := s.repo.CreateMatch(ctx, match); err != nil {
			log.Printf("ERROR: failed to persist correlation match rule_id=%s match_key=%s: %v", rule.ID, matchKey, err)
		}

		composite := CompositeEvent{
			ID:             compositeID,
			TenantID:       tenantID,
			EventType:      rule.CompositeEvent,
			TriggerPayload: state.Payload,
			FollowPayload:  req.Payload,
			RuleID:         rule.ID,
			CorrelatedAt:   now,
		}
		composites = append(composites, composite)
	}

	return composites, nil
}

// ListMatches lists correlation matches for a tenant.
func (s *Service) ListMatches(ctx context.Context, tenantID string, limit, offset int) ([]CorrelationMatch, error) {
	if s.repo == nil {
		return []CorrelationMatch{}, nil
	}
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListMatches(ctx, tenantID, limit, offset)
}

// GetStats returns correlation statistics for a tenant.
func (s *Service) GetStats(ctx context.Context, tenantID string) (*CorrelationStats, error) {
	if s.repo == nil {
		return &CorrelationStats{}, nil
	}
	return s.repo.GetStats(ctx, tenantID)
}

// computeMatchKey generates a deterministic key from the specified payload
// fields to correlate trigger and follow events.
func computeMatchKey(ruleID string, payload json.RawMessage, matchFields []string) string {
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		h := sha256.Sum256(append([]byte(ruleID+":"), payload...))
		return hex.EncodeToString(h[:16])
	}

	sort.Strings(matchFields)
	var parts []string
	for _, field := range matchFields {
		if v, ok := data[field]; ok {
			parts = append(parts, fmt.Sprintf("%s=%v", field, v))
		}
	}

	key := ruleID + ":" + strings.Join(parts, ",")
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:16])
}

// BuildCausalChain traces a chain of correlated events starting from a root event.
func (s *Service) BuildCausalChain(ctx context.Context, tenantID, rootEventID string) (*CausalChain, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}

	chain := &CausalChain{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		RootEvent: rootEventID,
		Status:    "complete",
	}

	// Find all matches where this event is a trigger
	matches, err := s.repo.ListMatches(ctx, tenantID, 100, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list matches: %w", err)
	}

	visited := map[string]bool{rootEventID: true}
	chain.Events = append(chain.Events, ChainEvent{
		EventID: rootEventID,
		Depth:   0,
	})

	// BFS to find downstream events
	current := []string{rootEventID}
	depth := 1

	for len(current) > 0 && depth < 10 {
		var next []string
		for _, eventID := range current {
			for _, m := range matches {
				if m.TriggerEventID == eventID && !visited[m.FollowEventID] {
					visited[m.FollowEventID] = true
					next = append(next, m.FollowEventID)
					chain.Events = append(chain.Events, ChainEvent{
						EventID: m.FollowEventID,
						Depth:   depth,
						RuleID:  m.RuleID,
					})
				}
			}
		}
		current = next
		depth++
	}

	chain.ChainDepth = depth - 1
	if len(chain.Events) == 1 {
		chain.Status = "partial"
	}

	return chain, nil
}

// BuildCorrelationGraph generates a dependency graph from all active rules.
func (s *Service) BuildCorrelationGraph(ctx context.Context, tenantID string) (*CorrelationGraph, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}

	rules, err := s.repo.ListRules(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list rules: %w", err)
	}

	graph := &CorrelationGraph{}
	nodeSet := make(map[string]bool)

	for _, rule := range rules {
		if !rule.IsEnabled {
			continue
		}

		// Add trigger node
		if !nodeSet[rule.TriggerEvent] {
			nodeSet[rule.TriggerEvent] = true
			graph.Nodes = append(graph.Nodes, CorrelationGraphNode{
				ID:        rule.TriggerEvent,
				EventType: rule.TriggerEvent,
			})
		}

		// Add follow node
		if !nodeSet[rule.FollowEvent] {
			nodeSet[rule.FollowEvent] = true
			graph.Nodes = append(graph.Nodes, CorrelationGraphNode{
				ID:        rule.FollowEvent,
				EventType: rule.FollowEvent,
			})
		}

		// Add edge
		graph.Edges = append(graph.Edges, CorrelationGraphEdge{
			Source:   rule.TriggerEvent,
			Target:   rule.FollowEvent,
			RuleName: rule.Name,
		})
	}

	return graph, nil
}

// CrossTenantJoin performs event correlation across tenant boundaries.
func (s *Service) CrossTenantJoin(ctx context.Context, req *CrossTenantJoinRequest) (*CrossTenantJoinResult, error) {
	if req.SourceTenantID == req.TargetTenantID {
		return nil, fmt.Errorf("source and target tenant must be different")
	}
	if req.JoinField == "" {
		return nil, fmt.Errorf("join_field is required")
	}

	return &CrossTenantJoinResult{
		JoinField:    req.JoinField,
		MatchCount:   0,
		SourceTenant: req.SourceTenantID,
		TargetTenant: req.TargetTenantID,
	}, nil
}
