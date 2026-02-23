package eventcorrelation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
		_ = s.repo.CreateState(ctx, state)
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
		_ = s.repo.UpdateState(ctx, state)

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
		_ = s.repo.CreateMatch(ctx, match)

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
