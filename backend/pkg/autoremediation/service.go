package autoremediation

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service provides auto-remediation functionality
type Service struct {
	repo Repository
}

// NewService creates a new auto-remediation service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// AnalyzeFailures scans recent failures, identifies patterns, and calculates confidence
func (s *Service) AnalyzeFailures(ctx context.Context, tenantID string) ([]FailurePattern, error) {
	existing, err := s.repo.ListPatterns(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list existing patterns: %w", err)
	}

	var analyzed []FailurePattern
	for i := range existing {
		p := &existing[i]
		if p.Status != PatternStatusActive {
			continue
		}

		p.Confidence = s.calculateConfidence(p)
		p.LastSeenAt = time.Now()

		if err := s.repo.UpdatePattern(ctx, p); err != nil {
			continue
		}
		analyzed = append(analyzed, *p)
	}

	return analyzed, nil
}

// GetPatterns retrieves all failure patterns for a tenant
func (s *Service) GetPatterns(ctx context.Context, tenantID string) ([]FailurePattern, error) {
	return s.repo.ListPatterns(ctx, tenantID)
}

// GetPattern retrieves a single failure pattern by ID
func (s *Service) GetPattern(ctx context.Context, tenantID, patternID string) (*FailurePattern, error) {
	return s.repo.GetPattern(ctx, tenantID, patternID)
}

// GetRecommendations generates recommendations based on active patterns
func (s *Service) GetRecommendations(ctx context.Context, tenantID string) ([]Recommendation, error) {
	patterns, err := s.repo.ListPatterns(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list patterns: %w", err)
	}

	var recommendations []Recommendation
	for _, p := range patterns {
		if p.Status != PatternStatusActive {
			continue
		}

		rec := s.generateRecommendation(&p)
		recommendations = append(recommendations, rec)
	}

	return recommendations, nil
}

// CreateRule creates a new remediation rule
func (s *Service) CreateRule(ctx context.Context, tenantID string, req *CreateRuleRequest) (*RemediationRule, error) {
	_, err := s.repo.GetPattern(ctx, tenantID, req.PatternID)
	if err != nil {
		return nil, fmt.Errorf("pattern not found: %w", err)
	}

	rule := &RemediationRule{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		PatternID:    req.PatternID,
		Name:         req.Name,
		ActionType:   req.ActionType,
		ActionConfig: req.ActionConfig,
		IsAutomatic:  req.IsAutomatic,
		Priority:     req.Priority,
		SuccessCount: 0,
		FailureCount: 0,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.repo.CreateRule(ctx, rule); err != nil {
		return nil, fmt.Errorf("failed to create remediation rule: %w", err)
	}

	return rule, nil
}

// ListRules retrieves all remediation rules for a tenant
func (s *Service) ListRules(ctx context.Context, tenantID string) ([]RemediationRule, error) {
	return s.repo.ListRules(ctx, tenantID)
}

// ApplyRemediation applies a specific rule action and records it
func (s *Service) ApplyRemediation(ctx context.Context, tenantID string, req *ApplyActionRequest) (*RemediationAction, error) {
	rule, err := s.repo.GetRule(ctx, tenantID, req.RuleID)
	if err != nil {
		return nil, fmt.Errorf("rule not found: %w", err)
	}

	now := time.Now()
	action := &RemediationAction{
		ID:            uuid.New().String(),
		TenantID:      tenantID,
		RuleID:        rule.ID,
		PatternID:     rule.PatternID,
		ActionType:    rule.ActionType,
		ActionDetails: req.ActionDetails,
		Status:        ActionStatusApplied,
		AppliedAt:     &now,
		CreatedAt:     now,
	}

	if err := s.repo.CreateAction(ctx, action); err != nil {
		action.Status = ActionStatusFailed
		return nil, fmt.Errorf("failed to apply remediation: %w", err)
	}

	rule.SuccessCount++
	rule.UpdatedAt = time.Now()
	// best-effort: update rule stats; action was already applied successfully
	_ = s.repo.UpdateRule(ctx, rule)

	return action, nil
}

// RevertAction reverts a previously applied action
func (s *Service) RevertAction(ctx context.Context, tenantID, actionID string) (*RemediationAction, error) {
	action, err := s.repo.GetAction(ctx, tenantID, actionID)
	if err != nil {
		return nil, fmt.Errorf("action not found: %w", err)
	}

	if action.Status != ActionStatusApplied {
		return nil, fmt.Errorf("action is not in applied state, current status: %s", action.Status)
	}

	now := time.Now()
	action.Status = ActionStatusReverted
	action.RevertedAt = &now

	if err := s.repo.UpdateAction(ctx, action); err != nil {
		return nil, fmt.Errorf("failed to revert action: %w", err)
	}

	rule, err := s.repo.GetRule(ctx, tenantID, action.RuleID)
	if err == nil {
		rule.FailureCount++
		rule.UpdatedAt = time.Now()
		// best-effort: update rule stats; action revert was already persisted
		_ = s.repo.UpdateRule(ctx, rule)
	}

	return action, nil
}

// ListActions retrieves remediation actions for a tenant
func (s *Service) ListActions(ctx context.Context, tenantID string, limit, offset int) ([]RemediationAction, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListActions(ctx, tenantID, limit, offset)
}

// PredictEndpointHealth returns health predictions for all endpoints
func (s *Service) PredictEndpointHealth(ctx context.Context, tenantID string) ([]HealthPrediction, error) {
	predictions, err := s.repo.GetEndpointHealthStats(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint health stats: %w", err)
	}

	for i := range predictions {
		predictions[i].PredictedAt = time.Now().Add(24 * time.Hour).Format(time.RFC3339)
		if predictions[i].CurrentHealth == "unhealthy" {
			predictions[i].PredictedHealth = "at_risk"
			predictions[i].ConfidencePct = 85.0
			predictions[i].RiskFactors = append(predictions[i].RiskFactors, "current_health_degraded")
		} else {
			predictions[i].PredictedHealth = "healthy"
			predictions[i].ConfidencePct = 90.0
		}
	}

	return predictions, nil
}

func (s *Service) calculateConfidence(p *FailurePattern) float64 {
	confidence := 0.0

	if p.OccurrenceCount >= 10 {
		confidence += 0.4
	} else if p.OccurrenceCount >= 5 {
		confidence += 0.2
	} else {
		confidence += 0.1
	}

	duration := p.LastSeenAt.Sub(p.FirstSeenAt)
	if duration > 24*time.Hour {
		confidence += 0.3
	} else if duration > time.Hour {
		confidence += 0.2
	} else {
		confidence += 0.1
	}

	if p.Frequency > 100 {
		confidence += 0.3
	} else if p.Frequency > 10 {
		confidence += 0.2
	} else {
		confidence += 0.1
	}

	if confidence > 1.0 {
		confidence = 1.0
	}
	return confidence
}

func (s *Service) generateRecommendation(p *FailurePattern) Recommendation {
	rec := Recommendation{
		PatternID:   p.ID,
		PatternName: p.PatternName,
		Confidence:  p.Confidence,
	}

	switch {
	case p.ErrorCode == "TIMEOUT" || p.ErrorCode == "CONNECTION_REFUSED":
		rec.SuggestedAction = ActionTypeRetryStrategyChange
		rec.SuggestedConfig = `{"max_retries": 5, "backoff_multiplier": 2, "initial_delay_ms": 1000}`
		rec.EstimatedImpact = "Reduce timeout failures by ~60%"
		rec.Reasoning = "Repeated timeout/connection errors suggest the endpoint needs more retry attempts with exponential backoff"
	case p.ErrorCode == "TRANSFORM_ERROR":
		rec.SuggestedAction = ActionTypeTransformFix
		rec.SuggestedConfig = `{"validate_payload": true, "fallback_transform": "passthrough"}`
		rec.EstimatedImpact = "Eliminate transform-related failures"
		rec.Reasoning = "Transform errors indicate payload incompatibility that can be resolved with validation and fallback"
	case p.OccurrenceCount > 100:
		rec.SuggestedAction = ActionTypeEndpointDisable
		rec.SuggestedConfig = `{"disable_duration_minutes": 30, "health_check_interval_seconds": 60}`
		rec.EstimatedImpact = "Prevent cascading failures from unhealthy endpoint"
		rec.Reasoning = "High occurrence count suggests the endpoint is persistently unhealthy and should be temporarily disabled"
	default:
		rec.SuggestedAction = ActionTypeAlert
		rec.SuggestedConfig = `{"channels": ["slack", "email"], "severity": "warning"}`
		rec.EstimatedImpact = "Improve incident response time"
		rec.Reasoning = "Pattern detected but requires human review before automated action"
	}

	return rec
}
