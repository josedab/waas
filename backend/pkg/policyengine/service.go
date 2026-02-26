package policyengine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/utils"
)

// Service provides OPA/Rego policy engine business logic.
type Service struct {
	repo   Repository
	logger *utils.Logger
}

// NewService creates a new policy engine service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo, logger: utils.NewLogger("policyengine-service")}
}

// CreatePolicy creates a new Rego policy with syntax validation.
func (s *Service) CreatePolicy(ctx context.Context, tenantID string, req *CreatePolicyRequest) (*Policy, error) {
	if err := validatePolicyType(req.PolicyType); err != nil {
		return nil, err
	}

	validation := ValidateRego(req.RegoSource)
	if !validation.Valid {
		return nil, fmt.Errorf("rego syntax errors: %s", strings.Join(validation.Errors, "; "))
	}

	now := time.Now()
	policy := &Policy{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		RegoSource:  req.RegoSource,
		Version:     1,
		IsActive:    true,
		PolicyType:  req.PolicyType,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if s.repo != nil {
		if err := s.repo.CreatePolicy(ctx, policy); err != nil {
			return nil, fmt.Errorf("failed to create policy: %w", err)
		}
		// Store initial version
		version := &PolicyVersion{
			ID:         uuid.New().String(),
			PolicyID:   policy.ID,
			Version:    1,
			RegoSource: req.RegoSource,
			ChangeNote: "Initial version",
			CreatedBy:  "system",
			CreatedAt:  now,
		}
		if err := s.repo.CreatePolicyVersion(ctx, version); err != nil {
			s.logger.Error("failed to create policy version", map[string]interface{}{"error": err.Error(), "policy_id": policy.ID})
		}
	}

	return policy, nil
}

// GetPolicy retrieves a policy.
func (s *Service) GetPolicy(ctx context.Context, tenantID, policyID string) (*Policy, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}
	return s.repo.GetPolicy(ctx, tenantID, policyID)
}

// ListPolicies lists all policies for a tenant.
func (s *Service) ListPolicies(ctx context.Context, tenantID string) ([]Policy, error) {
	if s.repo == nil {
		return []Policy{}, nil
	}
	return s.repo.ListPolicies(ctx, tenantID)
}

// UpdatePolicy updates a policy, creating a new version.
func (s *Service) UpdatePolicy(ctx context.Context, tenantID, policyID string, req *UpdatePolicyRequest) (*Policy, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}

	policy, err := s.repo.GetPolicy(ctx, tenantID, policyID)
	if err != nil {
		return nil, err
	}

	if req.RegoSource != nil {
		validation := ValidateRego(*req.RegoSource)
		if !validation.Valid {
			return nil, fmt.Errorf("rego syntax errors: %s", strings.Join(validation.Errors, "; "))
		}
		policy.RegoSource = *req.RegoSource
		policy.Version++
	}
	if req.Description != nil {
		policy.Description = *req.Description
	}
	if req.IsActive != nil {
		policy.IsActive = *req.IsActive
	}
	policy.UpdatedAt = time.Now()

	if err := s.repo.UpdatePolicy(ctx, policy); err != nil {
		return nil, fmt.Errorf("failed to update policy: %w", err)
	}

	// Store version history if rego source changed
	if req.RegoSource != nil {
		version := &PolicyVersion{
			ID:         uuid.New().String(),
			PolicyID:   policy.ID,
			Version:    policy.Version,
			RegoSource: *req.RegoSource,
			ChangeNote: req.ChangeNote,
			CreatedBy:  "system",
			CreatedAt:  time.Now(),
		}
		if err := s.repo.CreatePolicyVersion(ctx, version); err != nil {
			s.logger.Error("failed to create policy version", map[string]interface{}{"error": err.Error(), "policy_id": policy.ID})
		}
	}

	return policy, nil
}

// DeletePolicy removes a policy.
func (s *Service) DeletePolicy(ctx context.Context, tenantID, policyID string) error {
	if s.repo == nil {
		return fmt.Errorf("repository not configured")
	}
	return s.repo.DeletePolicy(ctx, tenantID, policyID)
}

// Evaluate evaluates a policy against an input document using a simple
// rule-based engine. For a full OPA integration, this would call the
// github.com/open-policy-agent/opa/rego package.
func (s *Service) Evaluate(ctx context.Context, tenantID string, req *EvaluateRequest) (*EvaluationResult, error) {
	var policy *Policy

	if s.repo != nil {
		var err error
		policy, err = s.repo.GetPolicy(ctx, tenantID, req.PolicyID)
		if err != nil {
			return nil, fmt.Errorf("policy not found: %w", err)
		}
	} else {
		// In-memory fallback for testing
		policy = &Policy{
			ID:         req.PolicyID,
			Name:       "inline",
			RegoSource: "",
			IsActive:   true,
		}
	}

	if !policy.IsActive {
		return nil, fmt.Errorf("policy %q is not active", policy.ID)
	}

	start := time.Now()
	decision := evaluateRegoSimple(policy.RegoSource, req.Input)
	durationMs := int(time.Since(start).Milliseconds())

	result := &EvaluationResult{
		Allowed:    decision["allow"] == true,
		Decision:   decision,
		PolicyID:   policy.ID,
		PolicyName: policy.Name,
		DurationMs: durationMs,
		IsDryRun:   req.DryRun,
	}

	// Log evaluation
	if s.repo != nil && !req.DryRun {
		resultJSON, _ := json.Marshal(decision)
		evalLog := &EvaluationLog{
			ID:         uuid.New().String(),
			TenantID:   tenantID,
			PolicyID:   policy.ID,
			Decision:   result.Allowed,
			InputHash:  hashInput(req.Input),
			DurationMs: durationMs,
			IsDryRun:   req.DryRun,
			Result:     resultJSON,
			CreatedAt:  time.Now(),
		}
		if err := s.repo.StoreEvaluationLog(ctx, evalLog); err != nil {
			s.logger.Error("failed to store evaluation log", map[string]interface{}{"error": err.Error()})
		}
	}

	return result, nil
}

// ValidateRegoSource validates Rego syntax for the API.
func (s *Service) ValidateRegoSource(regoSource string) *ValidationResult {
	return ValidateRego(regoSource)
}

// ListPolicyVersions returns version history for a policy.
func (s *Service) ListPolicyVersions(ctx context.Context, policyID string) ([]PolicyVersion, error) {
	if s.repo == nil {
		return []PolicyVersion{}, nil
	}
	return s.repo.ListPolicyVersions(ctx, policyID)
}

// ListEvaluationLogs returns evaluation logs for a tenant.
func (s *Service) ListEvaluationLogs(ctx context.Context, tenantID string, limit, offset int) ([]EvaluationLog, error) {
	if s.repo == nil {
		return []EvaluationLog{}, nil
	}
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListEvaluationLogs(ctx, tenantID, limit, offset)
}

// ValidateRego performs basic Rego syntax validation.
func ValidateRego(source string) *ValidationResult {
	var errors []string

	if strings.TrimSpace(source) == "" {
		return &ValidationResult{Valid: false, Errors: []string{"rego source cannot be empty"}}
	}

	if !strings.Contains(source, "package ") {
		errors = append(errors, "missing 'package' declaration")
	}

	// Check for balanced braces
	braceCount := 0
	for _, ch := range source {
		if ch == '{' {
			braceCount++
		} else if ch == '}' {
			braceCount--
		}
		if braceCount < 0 {
			errors = append(errors, "unmatched closing brace")
			break
		}
	}
	if braceCount > 0 {
		errors = append(errors, "unmatched opening brace")
	}

	if len(errors) > 0 {
		return &ValidationResult{Valid: false, Errors: errors}
	}
	return &ValidationResult{Valid: true}
}

// evaluateRegoSimple provides a basic rule evaluation engine. In production,
// this would delegate to github.com/open-policy-agent/opa/rego for full
// OPA support. The simple engine parses "default allow = false/true" and
// rule-based conditions from the rego source.
func evaluateRegoSimple(source string, input EvaluationInput) map[string]interface{} {
	decision := map[string]interface{}{
		"allow": true,
	}

	lines := strings.Split(source, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Parse default allow
		if strings.HasPrefix(trimmed, "default allow") {
			if strings.Contains(trimmed, "false") {
				decision["allow"] = false
			} else {
				decision["allow"] = true
			}
		}

		// Parse simple allow rules: allow { input.event_type == "..." }
		if strings.HasPrefix(trimmed, "allow") && strings.Contains(trimmed, "input.event_type") {
			for _, part := range strings.Split(trimmed, "==") {
				cleaned := strings.TrimSpace(part)
				cleaned = strings.Trim(cleaned, `"{}' `)
				if cleaned != "" && !strings.Contains(cleaned, "input.") && !strings.Contains(cleaned, "allow") {
					if cleaned == input.EventType {
						decision["allow"] = true
					}
				}
			}
		}
	}

	decision["evaluated_at"] = time.Now().Format(time.RFC3339)
	return decision
}

func hashInput(input EvaluationInput) string {
	b, _ := json.Marshal(input)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:16])
}

func validatePolicyType(pt string) error {
	switch pt {
	case PolicyTypeRouting, PolicyTypeFiltering, PolicyTypeAuthorization, PolicyTypeDelivery:
		return nil
	}
	return fmt.Errorf("invalid policy_type %q: must be routing, filtering, authorization, or delivery", pt)
}
