package cloudmanaged

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOneClickDeploy_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tenant := setupTenantWithPlan(t, svc, "starter")

	deployment, err := svc.OneClickDeploy(ctx, tenant.TenantID, PlanTierStarter, "us-east-1")
	require.NoError(t, err)

	assert.Equal(t, ProvisioningCompleted, deployment.Status)
	assert.NotEmpty(t, deployment.APIURL)
	assert.NotEmpty(t, deployment.DashboardURL)
	assert.NotEmpty(t, deployment.APIKey)
	assert.NotEmpty(t, deployment.WebhookSecret)
	assert.True(t, deployment.ElapsedSeconds >= 0)
	assert.NotNil(t, deployment.CompletedAt)

	// All steps should be completed
	for _, step := range deployment.Steps {
		assert.Equal(t, "completed", step.Status, "step %s should be completed", step.Name)
		assert.NotNil(t, step.StartedAt)
		assert.NotNil(t, step.CompletedAt)
	}
}

func TestOneClickDeploy_InvalidRegion(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tenant := setupTenantWithPlan(t, svc, "starter")

	deployment, err := svc.OneClickDeploy(ctx, tenant.TenantID, PlanTierStarter, "invalid-region")
	require.NoError(t, err) // Returns deployment with failed status, not error

	assert.Equal(t, ProvisioningFailed, deployment.Status)
	assert.Contains(t, deployment.Error, "unsupported region")
}

func TestOneClickDeploy_EmptyTenantID(t *testing.T) {
	svc := NewService(newMockRepository())
	ctx := context.Background()

	_, err := svc.OneClickDeploy(ctx, "", PlanTierFree, "us-east-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tenant_id is required")
}

func TestOneClickDeploy_DefaultRegion(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tenant := setupTenantWithPlan(t, svc, "free")

	deployment, err := svc.OneClickDeploy(ctx, tenant.TenantID, PlanTierFree, "")
	require.NoError(t, err)
	assert.Equal(t, "us-east-1", deployment.Region)
}

func TestOneClickDeploy_DeploymentModeByPlan(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tests := []struct {
		plan PlanTier
		mode DeploymentMode
	}{
		{PlanTierFree, DeploymentModeShared},
		{PlanTierStarter, DeploymentModeShared},
		{PlanTierPro, DeploymentModeDedicated},
		{PlanTierEnterprise, DeploymentModeDedicated},
	}

	for _, tt := range tests {
		t.Run(string(tt.plan), func(t *testing.T) {
			tenant := setupTenantWithPlan(t, svc, string(tt.plan))
			deployment, err := svc.OneClickDeploy(ctx, tenant.TenantID, tt.plan, "us-east-1")
			require.NoError(t, err)
			assert.Equal(t, tt.mode, deployment.Mode)
		})
	}
}

func TestGetControlPlaneConfig(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tenant := setupTenantWithPlan(t, svc, "pro")

	config, err := svc.GetControlPlaneConfig(ctx, tenant.TenantID)
	require.NoError(t, err)
	assert.True(t, config.RLSEnabled)
	assert.True(t, config.EncryptionAtRest)
	assert.True(t, config.AuditLogEnabled)
	assert.Equal(t, 90, config.RetentionDays)
	assert.Equal(t, 100, config.MaxEndpoints)
	assert.Contains(t, config.EnabledFeatures, "custom_domains")
}

func TestGetControlPlaneConfig_FreeTier(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tenant := setupTenantWithPlan(t, svc, "free")

	config, err := svc.GetControlPlaneConfig(ctx, tenant.TenantID)
	require.NoError(t, err)
	assert.False(t, config.AuditLogEnabled)
	assert.Equal(t, 7, config.RetentionDays)
	assert.Equal(t, 5, config.MaxEndpoints)
}

func TestGetDeploymentHealth(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tenant := setupTenantWithPlan(t, svc, "starter")

	health, err := svc.GetDeploymentHealth(ctx, tenant.TenantID)
	require.NoError(t, err)
	assert.Equal(t, "healthy", health.Status)
	assert.True(t, len(health.ComponentStatuses) > 0)
}

func TestGetDeploymentHealth_TenantNotFound(t *testing.T) {
	svc := NewService(newMockRepository())
	ctx := context.Background()

	_, err := svc.GetDeploymentHealth(ctx, "nonexistent")
	assert.Error(t, err)
}

func TestGetSLAReport(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tenant := setupTenantWithPlan(t, svc, "enterprise")

	report, err := svc.GetSLAReport(ctx, tenant.TenantID)
	require.NoError(t, err)
	assert.Equal(t, "compliant", report.OverallStatus)
	assert.True(t, len(report.Targets) >= 3)
	assert.True(t, report.ErrorBudget > 0)
}

func TestProvisionTenant(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tenant := setupTenantWithPlan(t, svc, "pro")

	job, err := svc.ProvisionTenant(ctx, tenant.TenantID, PlanTierPro, "us-east-1")
	require.NoError(t, err)
	assert.Equal(t, ProvisioningCompleted, job.Status)
	assert.NotNil(t, job.CompletedAt)

	for _, step := range job.Steps {
		assert.Equal(t, "completed", step.Status, "step %s", step.Name)
	}
}

func TestDeprovisionTenant(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tenant := setupTenantWithPlan(t, svc, "starter")

	job, err := svc.DeprovisionTenant(ctx, tenant.TenantID)
	require.NoError(t, err)
	assert.Equal(t, ProvisioningCompleted, job.Status)
}
