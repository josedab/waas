package routingpolicy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreatePolicy(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	policy, err := svc.CreatePolicy("t1", &CreatePolicyRequest{
		Name: "Enterprise Routing",
		Rules: []Rule{
			{
				Name:     "high-priority",
				Priority: 1,
				Conditions: []Condition{
					{Field: "tenant_tier", Operator: "eq", Value: "enterprise"},
				},
				Actions: []Action{
					{Type: "priority_queue", Params: map[string]string{"queue": "high"}},
				},
			},
		},
	})

	require.NoError(t, err)
	assert.NotEmpty(t, policy.ID)
	assert.Equal(t, 1, policy.Version)
	assert.True(t, policy.Enabled)
}

func TestEvaluatePolicy(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	svc.CreatePolicy("t1", &CreatePolicyRequest{
		Name: "Tier Routing",
		Rules: []Rule{
			{
				Name:     "enterprise-priority",
				Priority: 1,
				Conditions: []Condition{
					{Field: "tenant_tier", Operator: "eq", Value: "enterprise"},
				},
				Actions: []Action{
					{Type: "priority_queue", Params: map[string]string{"queue": "high"}},
				},
			},
			{
				Name:     "large-payload",
				Priority: 2,
				Conditions: []Condition{
					{Field: "payload_size", Operator: "gt", Value: "10000"},
				},
				Actions: []Action{
					{Type: "rate_adjust", Params: map[string]string{"factor": "0.5"}},
				},
			},
		},
	})

	results, err := svc.Evaluate("t1", &EvaluationContext{
		TenantTier:  "enterprise",
		EventType:   "order.created",
		PayloadSize: 15000,
	})

	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Len(t, results[0].MatchedRules, 2)
	assert.Equal(t, "high", results[0].RoutingQueue)
	assert.Equal(t, 0.5, results[0].RateAdjust)
}

func TestConditionOperators(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	tests := []struct {
		name     string
		cond     Condition
		ctx      *EvaluationContext
		expected bool
	}{
		{"eq match", Condition{Field: "event_type", Operator: "eq", Value: "order.created"}, &EvaluationContext{EventType: "order.created"}, true},
		{"eq no match", Condition{Field: "event_type", Operator: "eq", Value: "order.updated"}, &EvaluationContext{EventType: "order.created"}, false},
		{"neq", Condition{Field: "tenant_tier", Operator: "neq", Value: "free"}, &EvaluationContext{TenantTier: "enterprise"}, true},
		{"gt", Condition{Field: "payload_size", Operator: "gt", Value: "100"}, &EvaluationContext{PayloadSize: 200}, true},
		{"lt", Condition{Field: "payload_size", Operator: "lt", Value: "100"}, &EvaluationContext{PayloadSize: 50}, true},
		{"in", Condition{Field: "event_type", Operator: "in", Value: "order.created,order.updated"}, &EvaluationContext{EventType: "order.created"}, true},
		{"matches", Condition{Field: "event_type", Operator: "matches", Value: "order"}, &EvaluationContext{EventType: "order.created"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.evaluateCondition(&tt.cond, tt.ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPolicyVersioning(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	policy, _ := svc.CreatePolicy("t1", &CreatePolicyRequest{
		Name:  "V1",
		Rules: []Rule{{Name: "r1", Actions: []Action{{Type: "tag", Params: map[string]string{"tag": "v1"}}}}},
	})

	svc.UpdatePolicy(policy.ID, &CreatePolicyRequest{
		Name:  "V2",
		Rules: []Rule{{Name: "r1", Actions: []Action{{Type: "tag", Params: map[string]string{"tag": "v2"}}}}},
	})

	versions, err := svc.GetVersions(policy.ID)
	require.NoError(t, err)
	assert.Len(t, versions, 2)
}

func TestWhatIf(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	policy, _ := svc.CreatePolicy("t1", &CreatePolicyRequest{
		Name: "Test",
		Rules: []Rule{
			{
				Name:       "always",
				Conditions: []Condition{},
				Actions:    []Action{{Type: "tag", Params: map[string]string{"tag": "tested"}}},
			},
		},
	})

	result, err := svc.WhatIf(&WhatIfRequest{
		PolicyID: policy.ID,
		Context:  &EvaluationContext{TenantTier: "free"},
	})

	require.NoError(t, err)
	assert.Len(t, result.MatchedRules, 1)
	assert.Contains(t, result.Tags, "tested")
}
