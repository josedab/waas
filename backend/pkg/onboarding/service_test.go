package onboarding

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartOnboarding(t *testing.T) {
	t.Parallel()
	svc := NewService(nil)

	progress, err := svc.StartOnboarding(context.Background(), &StartOnboardingRequest{
		TenantID: "tenant-new",
	})

	require.NoError(t, err)
	assert.Equal(t, StepCreateTenant, progress.Session.CurrentStep)
	assert.Equal(t, 5, len(progress.Steps))
	assert.False(t, progress.Session.IsComplete)
	assert.Equal(t, 0, progress.ProgressPercent)
}

func TestCompleteStep_AdvancesToNext(t *testing.T) {
	t.Parallel()
	svc := NewService(nil)

	progress, err := svc.CompleteStep(context.Background(), "tenant-1", &CompleteStepRequest{
		StepID: StepCreateTenant,
	})

	require.NoError(t, err)
	assert.Contains(t, progress.Session.CompletedSteps, StepCreateTenant)
	assert.Equal(t, StepConfigEndpoint, progress.Session.CurrentStep)
}

func TestCompleteStep_FinalStepCompletesOnboarding(t *testing.T) {
	t.Parallel()
	svc := NewService(nil)

	progress, err := svc.CompleteStep(context.Background(), "tenant-1", &CompleteStepRequest{
		StepID: StepInstallSDK,
	})

	require.NoError(t, err)
	assert.True(t, progress.Session.IsComplete)
	assert.NotNil(t, progress.Session.CompletedAt)
}

func TestCompleteStep_InvalidStep(t *testing.T) {
	t.Parallel()
	svc := NewService(nil)

	_, err := svc.CompleteStep(context.Background(), "tenant-1", &CompleteStepRequest{
		StepID: "invalid_step",
	})

	assert.Error(t, err)
}

func TestGetSnippets_TypeScript(t *testing.T) {
	t.Parallel()
	svc := NewService(nil)

	snippets, err := svc.GetSnippets(context.Background(), &GetSnippetsRequest{
		Language: "typescript",
		TenantID: "my-tenant",
		APIKey:   "wh_abc123",
	})

	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(snippets), 3)
	assert.Contains(t, snippets[1].Code, "my-tenant")
	assert.Contains(t, snippets[1].Code, "wh_abc123")
}

func TestGetSnippets_Python(t *testing.T) {
	t.Parallel()
	svc := NewService(nil)

	snippets, err := svc.GetSnippets(context.Background(), &GetSnippetsRequest{
		Language: "python",
	})

	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(snippets), 3)
	assert.Equal(t, "python", snippets[0].Language)
}

func TestGetSnippets_Go(t *testing.T) {
	t.Parallel()
	svc := NewService(nil)

	snippets, err := svc.GetSnippets(context.Background(), &GetSnippetsRequest{
		Language: "go",
	})

	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(snippets), 2)
}

func TestIsValidStep(t *testing.T) {
	t.Parallel()

	assert.True(t, isValidStep(StepCreateTenant))
	assert.True(t, isValidStep(StepInstallSDK))
	assert.False(t, isValidStep("nonexistent"))
}

func TestGetNextStep(t *testing.T) {
	t.Parallel()

	assert.Equal(t, StepConfigEndpoint, getNextStep(StepCreateTenant))
	assert.Equal(t, StepSendTest, getNextStep(StepConfigEndpoint))
	assert.Equal(t, "", getNextStep(StepInstallSDK))
}
