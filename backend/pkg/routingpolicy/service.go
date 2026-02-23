package routingpolicy

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ServiceConfig configures the routing policy service.
type ServiceConfig struct {
	MaxPoliciesPerTenant int
	MaxRulesPerPolicy    int
}

// DefaultServiceConfig returns sensible defaults.
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		MaxPoliciesPerTenant: 50,
		MaxRulesPerPolicy:    100,
	}
}

// Service implements routing policy business logic.
type Service struct {
	repo   Repository
	config *ServiceConfig
}

// NewService creates a new routing policy service.
func NewService(repo Repository, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}
	return &Service{repo: repo, config: config}
}

// CreatePolicy creates a new routing policy.
func (s *Service) CreatePolicy(tenantID string, req *CreatePolicyRequest) (*Policy, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if len(req.Rules) == 0 {
		return nil, fmt.Errorf("at least one rule is required")
	}
	if len(req.Rules) > s.config.MaxRulesPerPolicy {
		return nil, fmt.Errorf("maximum %d rules per policy", s.config.MaxRulesPerPolicy)
	}

	for i, rule := range req.Rules {
		if rule.Logic == "" {
			req.Rules[i].Logic = "and"
		}
	}

	policy := &Policy{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		Version:     1,
		Rules:       req.Rules,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.CreatePolicy(policy); err != nil {
		return nil, fmt.Errorf("failed to create policy: %w", err)
	}

	s.storeVersion(policy)
	s.auditLog(policy.ID, "created", 1, "")
	return policy, nil
}

// UpdatePolicy updates an existing policy and creates a new version.
func (s *Service) UpdatePolicy(policyID string, req *CreatePolicyRequest) (*Policy, error) {
	existing, err := s.repo.GetPolicy(policyID)
	if err != nil {
		return nil, err
	}

	existing.Name = req.Name
	existing.Description = req.Description
	existing.Rules = req.Rules
	existing.Version++
	existing.UpdatedAt = time.Now()

	if err := s.repo.UpdatePolicy(existing); err != nil {
		return nil, fmt.Errorf("failed to update policy: %w", err)
	}

	s.storeVersion(existing)
	s.auditLog(existing.ID, "updated", existing.Version, "")
	return existing, nil
}

// GetPolicy retrieves a policy by ID.
func (s *Service) GetPolicy(id string) (*Policy, error) {
	return s.repo.GetPolicy(id)
}

// ListPolicies returns all policies for a tenant.
func (s *Service) ListPolicies(tenantID string) ([]*Policy, error) {
	return s.repo.ListPolicies(tenantID)
}

// DeletePolicy deletes a policy.
func (s *Service) DeletePolicy(id string) error {
	s.auditLog(id, "deleted", 0, "")
	return s.repo.DeletePolicy(id)
}

// TogglePolicy enables or disables a policy.
func (s *Service) TogglePolicy(id string, enabled bool) (*Policy, error) {
	policy, err := s.repo.GetPolicy(id)
	if err != nil {
		return nil, err
	}
	policy.Enabled = enabled
	policy.UpdatedAt = time.Now()

	if err := s.repo.UpdatePolicy(policy); err != nil {
		return nil, err
	}

	action := "enabled"
	if !enabled {
		action = "disabled"
	}
	s.auditLog(id, action, policy.Version, "")
	return policy, nil
}

// Evaluate evaluates all enabled policies against a context.
func (s *Service) Evaluate(tenantID string, ctx *EvaluationContext) ([]*EvaluationResult, error) {
	policies, err := s.repo.ListPolicies(tenantID)
	if err != nil {
		return nil, err
	}

	var results []*EvaluationResult
	for _, policy := range policies {
		if !policy.Enabled {
			continue
		}
		result := s.evaluatePolicy(policy, ctx)
		if len(result.MatchedRules) > 0 {
			results = append(results, result)
		}
	}
	return results, nil
}

// WhatIf simulates policy evaluation without applying actions.
func (s *Service) WhatIf(req *WhatIfRequest) (*EvaluationResult, error) {
	policy, err := s.repo.GetPolicy(req.PolicyID)
	if err != nil {
		return nil, err
	}
	return s.evaluatePolicy(policy, req.Context), nil
}

// GetVersions returns version history for a policy.
func (s *Service) GetVersions(policyID string) ([]*PolicyVersion, error) {
	return s.repo.ListVersions(policyID)
}

// GetAuditLog returns the audit log for a policy.
func (s *Service) GetAuditLog(policyID string, limit int) ([]*AuditEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListAudit(policyID, limit)
}

func (s *Service) evaluatePolicy(policy *Policy, ctx *EvaluationContext) *EvaluationResult {
	result := &EvaluationResult{
		PolicyID: policy.ID,
	}

	// Sort rules by priority
	rules := make([]Rule, len(policy.Rules))
	copy(rules, policy.Rules)
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority < rules[j].Priority
	})

	for _, rule := range rules {
		if s.evaluateRule(&rule, ctx) {
			result.MatchedRules = append(result.MatchedRules, rule.Name)
			result.Actions = append(result.Actions, rule.Actions...)

			// Extract specific action effects
			for _, action := range rule.Actions {
				switch action.Type {
				case "priority_queue":
					result.RoutingQueue = action.Params["queue"]
				case "rate_adjust":
					if factor, err := strconv.ParseFloat(action.Params["factor"], 64); err == nil {
						result.RateAdjust = factor
					}
				case "tag":
					result.Tags = append(result.Tags, action.Params["tag"])
				}
			}
		}
	}

	return result
}

func (s *Service) evaluateRule(rule *Rule, ctx *EvaluationContext) bool {
	if len(rule.Conditions) == 0 {
		return true
	}

	logic := rule.Logic
	if logic == "" {
		logic = "and"
	}

	for _, cond := range rule.Conditions {
		match := s.evaluateCondition(&cond, ctx)
		if logic == "or" && match {
			return true
		}
		if logic == "and" && !match {
			return false
		}
	}

	return logic == "and"
}

func (s *Service) evaluateCondition(cond *Condition, ctx *EvaluationContext) bool {
	var fieldValue string

	switch cond.Field {
	case "tenant_tier":
		fieldValue = ctx.TenantTier
	case "event_type":
		fieldValue = ctx.EventType
	case "payload_size":
		fieldValue = strconv.Itoa(ctx.PayloadSize)
	case "time_of_day":
		fieldValue = ctx.TimeOfDay
	default:
		if strings.HasPrefix(cond.Field, "header.") {
			headerName := strings.TrimPrefix(cond.Field, "header.")
			fieldValue = ctx.Headers[headerName]
		} else if strings.HasPrefix(cond.Field, "metadata.") {
			metaName := strings.TrimPrefix(cond.Field, "metadata.")
			fieldValue = ctx.Metadata[metaName]
		}
	}

	switch cond.Operator {
	case "eq":
		return fieldValue == cond.Value
	case "neq":
		return fieldValue != cond.Value
	case "gt":
		return compareNumeric(fieldValue, cond.Value) > 0
	case "lt":
		return compareNumeric(fieldValue, cond.Value) < 0
	case "gte":
		return compareNumeric(fieldValue, cond.Value) >= 0
	case "lte":
		return compareNumeric(fieldValue, cond.Value) <= 0
	case "in":
		values := strings.Split(cond.Value, ",")
		for _, v := range values {
			if strings.TrimSpace(v) == fieldValue {
				return true
			}
		}
		return false
	case "matches":
		return strings.Contains(fieldValue, cond.Value)
	default:
		return false
	}
}

func compareNumeric(a, b string) int {
	aNum, aErr := strconv.ParseFloat(a, 64)
	bNum, bErr := strconv.ParseFloat(b, 64)
	if aErr != nil || bErr != nil {
		return strings.Compare(a, b)
	}
	if aNum < bNum {
		return -1
	}
	if aNum > bNum {
		return 1
	}
	return 0
}

func (s *Service) storeVersion(policy *Policy) {
	v := &PolicyVersion{
		PolicyID:  policy.ID,
		Version:   policy.Version,
		Policy:    policy,
		CreatedAt: time.Now(),
	}
	_ = s.repo.StoreVersion(v)
}

func (s *Service) auditLog(policyID, action string, version int, changedBy string) {
	entry := &AuditEntry{
		ID:        uuid.New().String(),
		PolicyID:  policyID,
		Action:    action,
		Version:   version,
		ChangedBy: changedBy,
		Timestamp: time.Now(),
	}
	_ = s.repo.AppendAudit(entry)
}
