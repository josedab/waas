package cloudmanaged

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignupWithVerification(t *testing.T) {
	svc := NewService(nil)

	t.Run("missing TOS acceptance", func(t *testing.T) {
		_, _, err := svc.SignupWithVerification(context.Background(), &SignupWithVerificationRequest{
			Email:    "test@example.com",
			Org:      "TestOrg",
			Password: "securepass",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "terms of service")
	})

	t.Run("missing required fields", func(t *testing.T) {
		_, _, err := svc.SignupWithVerification(context.Background(), &SignupWithVerificationRequest{
			AcceptedTOS: true,
		})
		assert.Error(t, err)
	})

	t.Run("successful signup defaults to free plan", func(t *testing.T) {
		tenant, verification, err := svc.SignupWithVerification(context.Background(), &SignupWithVerificationRequest{
			Email:       "dev@example.com",
			Org:         "TestOrg",
			Password:    "securepass",
			AcceptedTOS: true,
		})
		require.NoError(t, err)
		assert.NotNil(t, tenant)
		assert.NotNil(t, verification)
		assert.Equal(t, PlanTierFree, tenant.Plan)
		assert.Equal(t, int64(1000), tenant.WebhooksLimit)
		assert.Nil(t, tenant.TrialEndsAt)
		assert.NotEmpty(t, verification.Token)
	})

	t.Run("paid plan gets trial period", func(t *testing.T) {
		tenant, _, err := svc.SignupWithVerification(context.Background(), &SignupWithVerificationRequest{
			Email:       "pro@example.com",
			Org:         "ProOrg",
			Password:    "securepass",
			Plan:        "pro",
			AcceptedTOS: true,
		})
		require.NoError(t, err)
		assert.Equal(t, PlanTierPro, tenant.Plan)
		assert.NotNil(t, tenant.TrialEndsAt)
	})
}

func TestCreateCheckoutSession(t *testing.T) {
	svc := NewService(nil)

	t.Run("free plan rejected", func(t *testing.T) {
		_, err := svc.CreateCheckoutSession(context.Background(), "tenant-1", PlanTierFree)
		assert.Error(t, err)
	})

	t.Run("valid plan creates session", func(t *testing.T) {
		session, err := svc.CreateCheckoutSession(context.Background(), "tenant-1", PlanTierPro)
		require.NoError(t, err)
		assert.NotEmpty(t, session.SessionID)
		assert.Equal(t, "pro", session.Plan)
	})
}

func TestGenerateRLSPolicies(t *testing.T) {
	policies := GenerateRLSPolicies("test-tenant-id")
	assert.True(t, len(policies) >= 4)

	for _, p := range policies {
		assert.NotEmpty(t, p.PolicySQL)
		assert.Contains(t, p.PolicySQL, "CREATE POLICY")
		assert.Contains(t, p.PolicySQL, "tenant_isolation")
	}
}

func TestDefaultPlansFreeTier(t *testing.T) {
	plans := DefaultPlans()
	var freePlan *PlanDefinition
	for _, p := range plans {
		if p.Tier == PlanTierFree {
			freePlan = &p
			break
		}
	}
	require.NotNil(t, freePlan)
	assert.Equal(t, int64(1000), int64(freePlan.WebhooksLimit))
	assert.Equal(t, int64(0), int64(freePlan.PriceMonthly))
}
