package auth

import (
	"context"
	"testing"
	"time"
	"github.com/josedab/waas/pkg/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSubscriptionService_GetSubscriptionInfo(t *testing.T) {
	mockTenantRepo := &MockTenantRepository{}
	mockQuotaRepo := &MockQuotaRepository{}
	service := NewSubscriptionService(mockTenantRepo, mockQuotaRepo)

	tenantID := uuid.New()
	tenant := &models.Tenant{
		ID:               tenantID,
		Name:             "Test Tenant",
		SubscriptionTier: "starter",
		MonthlyQuota:     10000,
	}

	currentMonth := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
	usage := &models.QuotaUsage{
		TenantID:     tenantID,
		RequestCount: 5000,
		SuccessCount: 4800,
		FailureCount: 200,
	}

	billingHistory := []*models.BillingRecord{
		{
			ID:            uuid.New(),
			TenantID:      tenantID,
			BillingPeriod: currentMonth.AddDate(0, -1, 0),
			TotalAmount:   1500,
		},
	}

	mockTenantRepo.On("GetByID", mock.Anything, tenantID).Return(tenant, nil)
	mockQuotaRepo.On("GetQuotaUsageByTenant", mock.Anything, tenantID, currentMonth).Return(usage, nil)
	mockQuotaRepo.On("GetBillingHistory", mock.Anything, tenantID, 6, 0).Return(billingHistory, nil)

	info, err := service.GetSubscriptionInfo(context.Background(), tenantID)

	assert.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, tenant, info.Tenant)
	assert.Equal(t, "starter", info.TierConfig.Name)
	assert.Equal(t, usage, info.CurrentUsage)
	assert.Len(t, info.BillingHistory, 1)
	assert.NotEmpty(t, info.UpgradeOptions)
	assert.NotEmpty(t, info.DowngradeOptions)

	mockTenantRepo.AssertExpectations(t)
	mockQuotaRepo.AssertExpectations(t)
}

func TestSubscriptionService_UpdateSubscription(t *testing.T) {
	mockTenantRepo := &MockTenantRepository{}
	mockQuotaRepo := &MockQuotaRepository{}
	service := NewSubscriptionService(mockTenantRepo, mockQuotaRepo)

	tenantID := uuid.New()
	tenant := &models.Tenant{
		ID:               tenantID,
		Name:             "Test Tenant",
		SubscriptionTier: "starter",
		MonthlyQuota:     10000,
		RateLimitPerMinute: 100,
	}

	update := &SubscriptionUpdate{
		TenantID:      tenantID,
		NewTier:       "professional",
		PreserveUsage: true,
	}

	mockTenantRepo.On("GetByID", mock.Anything, tenantID).Return(tenant, nil)
	mockTenantRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Tenant")).Return(nil)
	mockQuotaRepo.On("CreateQuotaNotification", mock.Anything, mock.AnythingOfType("*models.QuotaNotification")).Return(nil)

	err := service.UpdateSubscription(context.Background(), update)

	assert.NoError(t, err)

	// Verify tenant was updated with new tier configuration
	mockTenantRepo.AssertCalled(t, "Update", mock.Anything, mock.MatchedBy(func(t *models.Tenant) bool {
		return t.SubscriptionTier == "professional" && t.MonthlyQuota == 100000
	}))

	mockTenantRepo.AssertExpectations(t)
	mockQuotaRepo.AssertExpectations(t)
}

func TestSubscriptionService_ValidateSubscriptionChange_Downgrade(t *testing.T) {
	mockTenantRepo := &MockTenantRepository{}
	mockQuotaRepo := &MockQuotaRepository{}
	service := NewSubscriptionService(mockTenantRepo, mockQuotaRepo)

	tenantID := uuid.New()
	tenant := &models.Tenant{
		ID:               tenantID,
		Name:             "Test Tenant",
		SubscriptionTier: "professional",
		MonthlyQuota:     100000,
	}

	currentMonth := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
	usage := &models.QuotaUsage{
		TenantID:     tenantID,
		RequestCount: 15000, // More than starter tier quota (10000)
	}

	mockTenantRepo.On("GetByID", mock.Anything, tenantID).Return(tenant, nil)
	mockQuotaRepo.On("GetQuotaUsageByTenant", mock.Anything, tenantID, currentMonth).Return(usage, nil)

	err := service.ValidateSubscriptionChange(context.Background(), tenantID, "starter")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot downgrade")

	mockTenantRepo.AssertExpectations(t)
	mockQuotaRepo.AssertExpectations(t)
}

func TestSubscriptionService_ValidateSubscriptionChange_Upgrade(t *testing.T) {
	mockTenantRepo := &MockTenantRepository{}
	mockQuotaRepo := &MockQuotaRepository{}
	service := NewSubscriptionService(mockTenantRepo, mockQuotaRepo)

	tenantID := uuid.New()
	tenant := &models.Tenant{
		ID:               tenantID,
		Name:             "Test Tenant",
		SubscriptionTier: "starter",
		MonthlyQuota:     10000,
	}

	mockTenantRepo.On("GetByID", mock.Anything, tenantID).Return(tenant, nil)

	err := service.ValidateSubscriptionChange(context.Background(), tenantID, "professional")

	assert.NoError(t, err)

	mockTenantRepo.AssertExpectations(t)
}

func TestSubscriptionService_CalculateProration(t *testing.T) {
	mockTenantRepo := &MockTenantRepository{}
	mockQuotaRepo := &MockQuotaRepository{}
	service := NewSubscriptionService(mockTenantRepo, mockQuotaRepo)

	tenantID := uuid.New()
	tenant := &models.Tenant{
		ID:               tenantID,
		Name:             "Test Tenant",
		SubscriptionTier: "starter",
		MonthlyQuota:     10000,
	}

	changeDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC) // Mid-month

	mockTenantRepo.On("GetByID", mock.Anything, tenantID).Return(tenant, nil)

	prorationAmount, err := service.CalculateProration(context.Background(), tenantID, "professional", changeDate)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, prorationAmount, 0) // Should be non-negative

	mockTenantRepo.AssertExpectations(t)
}

func TestSubscriptionService_GetAvailableTiers(t *testing.T) {
	mockTenantRepo := &MockTenantRepository{}
	mockQuotaRepo := &MockQuotaRepository{}
	service := NewSubscriptionService(mockTenantRepo, mockQuotaRepo)

	tiers := service.GetAvailableTiers()

	assert.NotEmpty(t, tiers)
	assert.Contains(t, tiers, "free")
	assert.Contains(t, tiers, "starter")
	assert.Contains(t, tiers, "professional")
	assert.Contains(t, tiers, "enterprise")

	// Verify tier configurations
	freeTier := tiers["free"]
	assert.Equal(t, 1000, freeTier.MonthlyQuota)
	assert.Equal(t, 10, freeTier.RateLimitPerMinute)

	enterpriseTier := tiers["enterprise"]
	assert.Equal(t, 1000000, enterpriseTier.MonthlyQuota)
	assert.Equal(t, 2000, enterpriseTier.RateLimitPerMinute)
}