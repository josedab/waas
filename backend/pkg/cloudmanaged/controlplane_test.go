package cloudmanaged

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockControlPlaneRepo struct {
	mock.Mock
}

func (m *mockControlPlaneRepo) CreateCloudTenant(ctx context.Context, t *CloudTenant) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *mockControlPlaneRepo) GetCloudTenant(ctx context.Context, tenantID string) (*CloudTenant, error) {
	args := m.Called(ctx, tenantID)
	if v := args.Get(0); v != nil {
		return v.(*CloudTenant), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockControlPlaneRepo) UpdateCloudTenant(ctx context.Context, t *CloudTenant) error {
	return m.Called(ctx, t).Error(0)
}

func (m *mockControlPlaneRepo) ListCloudTenants(ctx context.Context, limit, offset int) ([]CloudTenant, error) {
	return nil, nil
}

func (m *mockControlPlaneRepo) RecordUsage(ctx context.Context, meter *UsageMeter) error {
	return nil
}

func (m *mockControlPlaneRepo) GetUsageSummary(ctx context.Context, tenantID, period string) (*UsageSummary, error) {
	return nil, nil
}

func (m *mockControlPlaneRepo) GetBillingInfo(ctx context.Context, tenantID string) (*BillingInfo, error) {
	return nil, nil
}

func (m *mockControlPlaneRepo) SaveBillingInfo(ctx context.Context, info *BillingInfo) error {
	return nil
}

func (m *mockControlPlaneRepo) GetOnboardingProgress(ctx context.Context, tenantID string) (*OnboardingProgress, error) {
	return nil, nil
}

func (m *mockControlPlaneRepo) SaveOnboardingProgress(ctx context.Context, p *OnboardingProgress) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

func (m *mockControlPlaneRepo) GetUsageHistory(ctx context.Context, tenantID string, limit int) ([]UsageMeter, error) {
	return nil, nil
}

func (m *mockControlPlaneRepo) GetSLAMetrics(ctx context.Context, tenantID string) (*SLAConfig, error) {
	return nil, nil
}

func (m *mockControlPlaneRepo) GetComponentStatuses(ctx context.Context) ([]StatusPageEntry, error) {
	return nil, nil
}

func (m *mockControlPlaneRepo) GetRegionalDeployments(ctx context.Context) ([]RegionalDeployment, error) {
	return nil, nil
}

func TestControlPlaneService_ProvisionTenantFull(t *testing.T) {
	tests := []struct {
		name    string
		req     *TenantProvisionRequest
		wantErr bool
	}{
		{
			name: "valid free tier provision",
			req: &TenantProvisionRequest{
				Email: "test@example.com",
				Org:   "test-org",
				Plan:  "free",
			},
			wantErr: false,
		},
		{
			name: "valid pro tier provision",
			req: &TenantProvisionRequest{
				Email:  "pro@example.com",
				Org:    "pro-org",
				Plan:   "pro",
				Region: "eu-west-1",
			},
			wantErr: false,
		},
		{
			name: "missing email",
			req: &TenantProvisionRequest{
				Org:  "test-org",
				Plan: "free",
			},
			wantErr: true,
		},
		{
			name: "invalid plan",
			req: &TenantProvisionRequest{
				Email: "test@example.com",
				Org:   "test-org",
				Plan:  "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockControlPlaneRepo)
			if !tt.wantErr {
				repo.On("CreateCloudTenant", mock.Anything, mock.Anything).Return(nil)
				repo.On("SaveOnboardingProgress", mock.Anything, mock.Anything).Return(nil)
			}

			svc := NewControlPlaneService(repo)
			result, err := svc.ProvisionTenantFull(context.Background(), tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.NotEmpty(t, result.APIKey)
			assert.NotNil(t, result.Endpoints)
			assert.NotNil(t, result.Isolation)
			assert.True(t, result.Isolation.RowPolicyEnabled)
		})
	}
}

func TestControlPlaneService_GetTenantHealth(t *testing.T) {
	repo := new(mockControlPlaneRepo)
	repo.On("GetCloudTenant", mock.Anything, "tenant-1").Return(&CloudTenant{
		TenantID:      "tenant-1",
		Plan:          PlanTierPro,
		Status:        CloudTenantStatusActive,
		WebhooksUsed:  5000,
		WebhooksLimit: 500000,
	}, nil)

	svc := NewControlPlaneService(repo)
	health, err := svc.GetTenantHealth(context.Background(), "tenant-1")

	assert.NoError(t, err)
	assert.Equal(t, "healthy", health.Status)
	assert.NotEmpty(t, health.Components)
}

func TestControlPlaneService_GetRowLevelIsolation(t *testing.T) {
	tests := []struct {
		name       string
		plan       PlanTier
		wantPolicy string
	}{
		{"free tier shared", PlanTierFree, "shared"},
		{"pro tier dedicated", PlanTierPro, "dedicated"},
		{"enterprise isolated", PlanTierEnterprise, "isolated"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockControlPlaneRepo)
			repo.On("GetCloudTenant", mock.Anything, "tenant-1").Return(&CloudTenant{
				TenantID: "tenant-1",
				Plan:     tt.plan,
				Region:   "us-east-1",
			}, nil)

			svc := NewControlPlaneService(repo)
			iso, err := svc.GetRowLevelIsolation(context.Background(), "tenant-1")

			assert.NoError(t, err)
			assert.Equal(t, tt.wantPolicy, iso.IsolationPolicy)
			assert.True(t, iso.RowPolicyEnabled)
		})
	}
}
