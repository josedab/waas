package policyengine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRego_Valid(t *testing.T) {
	t.Parallel()
	source := `package webhook.routing
default allow = false
allow { input.event_type == "order.created" }`

	result := ValidateRego(source)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestValidateRego_Empty(t *testing.T) {
	t.Parallel()
	result := ValidateRego("")
	assert.False(t, result.Valid)
	assert.Contains(t, result.Errors[0], "empty")
}

func TestValidateRego_MissingPackage(t *testing.T) {
	t.Parallel()
	result := ValidateRego("default allow = true")
	assert.False(t, result.Valid)
	assert.Contains(t, result.Errors[0], "package")
}

func TestValidateRego_UnbalancedBraces(t *testing.T) {
	t.Parallel()
	result := ValidateRego("package test\nallow { ")
	assert.False(t, result.Valid)
}

func TestService_CreatePolicy_Validation(t *testing.T) {
	t.Parallel()
	svc := NewService(nil)

	tests := []struct {
		name    string
		req     CreatePolicyRequest
		wantErr bool
	}{
		{
			name: "valid policy",
			req: CreatePolicyRequest{
				Name:       "Route orders",
				RegoSource: "package webhook.routing\ndefault allow = true",
				PolicyType: PolicyTypeRouting,
			},
			wantErr: false,
		},
		{
			name: "invalid policy type",
			req: CreatePolicyRequest{
				Name:       "Bad",
				RegoSource: "package test",
				PolicyType: "unknown",
			},
			wantErr: true,
		},
		{
			name: "invalid rego syntax",
			req: CreatePolicyRequest{
				Name:       "Bad",
				RegoSource: "allow = true",
				PolicyType: PolicyTypeRouting,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := svc.CreatePolicy(context.Background(), "tenant-1", &tt.req)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEvaluateRegoSimple_DefaultAllow(t *testing.T) {
	t.Parallel()
	source := "package test\ndefault allow = true"
	input := EvaluationInput{TenantID: "t1", EventType: "test"}

	decision := evaluateRegoSimple(source, input)
	assert.Equal(t, true, decision["allow"])
}

func TestEvaluateRegoSimple_DefaultDeny(t *testing.T) {
	t.Parallel()
	source := "package test\ndefault allow = false"
	input := EvaluationInput{TenantID: "t1", EventType: "test"}

	decision := evaluateRegoSimple(source, input)
	assert.Equal(t, false, decision["allow"])
}

func TestService_CreatePolicy_ValidReturnsPolicy(t *testing.T) {
	t.Parallel()
	svc := NewService(nil)

	policy, err := svc.CreatePolicy(context.Background(), "tenant-1", &CreatePolicyRequest{
		Name:       "Test Policy",
		RegoSource: "package webhook.test\ndefault allow = true",
		PolicyType: PolicyTypeRouting,
	})

	require.NoError(t, err)
	assert.Equal(t, "tenant-1", policy.TenantID)
	assert.Equal(t, "Test Policy", policy.Name)
	assert.Equal(t, 1, policy.Version)
	assert.True(t, policy.IsActive)
}
