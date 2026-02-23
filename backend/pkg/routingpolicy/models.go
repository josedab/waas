package routingpolicy

import "time"

// Policy is the top-level routing policy definition.
type Policy struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Version     int       `json:"version"`
	Rules       []Rule    `json:"rules"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Rule defines a single routing rule with conditions and actions.
type Rule struct {
	Name       string      `json:"name"`
	Priority   int         `json:"priority"`
	Conditions []Condition `json:"conditions"`
	Actions    []Action    `json:"actions"`
	Logic      string      `json:"logic"` // and, or
}

// Condition defines when a rule should be applied.
type Condition struct {
	Field    string `json:"field"`    // tenant_tier, event_type, payload_size, time_of_day, header.*
	Operator string `json:"operator"` // eq, neq, gt, lt, gte, lte, in, not_in, between, matches
	Value    string `json:"value"`
}

// Action defines what happens when a rule matches.
type Action struct {
	Type   string            `json:"type"` // priority_queue, compliance_vault, rate_adjust, transform, route_to, delay, tag
	Params map[string]string `json:"params"`
}

// EvaluationContext provides the data against which rules are evaluated.
type EvaluationContext struct {
	TenantTier  string            `json:"tenant_tier"`
	EventType   string            `json:"event_type"`
	PayloadSize int               `json:"payload_size"`
	TimeOfDay   string            `json:"time_of_day"` // HH:MM format
	Headers     map[string]string `json:"headers"`
	Metadata    map[string]string `json:"metadata"`
}

// EvaluationResult describes which rules matched and what actions to take.
type EvaluationResult struct {
	PolicyID     string   `json:"policy_id"`
	MatchedRules []string `json:"matched_rules"`
	Actions      []Action `json:"actions"`
	RoutingQueue string   `json:"routing_queue,omitempty"`
	RateAdjust   float64  `json:"rate_adjust,omitempty"`
	Tags         []string `json:"tags,omitempty"`
}

// PolicyVersion stores a historical version of a policy.
type PolicyVersion struct {
	PolicyID  string    `json:"policy_id"`
	Version   int       `json:"version"`
	Policy    *Policy   `json:"policy"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by"`
}

// WhatIfRequest is the DTO for policy simulation.
type WhatIfRequest struct {
	PolicyID string             `json:"policy_id" binding:"required"`
	Context  *EvaluationContext `json:"context" binding:"required"`
}

// CreatePolicyRequest is the DTO for creating a policy.
type CreatePolicyRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Rules       []Rule `json:"rules" binding:"required"`
}

// AuditEntry records policy changes.
type AuditEntry struct {
	ID        string    `json:"id"`
	PolicyID  string    `json:"policy_id"`
	Action    string    `json:"action"` // created, updated, enabled, disabled, deleted
	Version   int       `json:"version"`
	ChangedBy string    `json:"changed_by"`
	Timestamp time.Time `json:"timestamp"`
}
