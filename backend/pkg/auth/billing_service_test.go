package auth

import (
	"context"
	"testing"
	"time"
	"webhook-platform/pkg/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)



func TestBillingService_CalculateBilling_WithinQuota(t *testing.T) {
	mockTenantRepo := &MockTenantRepository{}
	mockQuotaRepo := &MockQuotaRepository{}
	service := NewBillingService(mockTenantRepo, mockQuotaRepo)

	tenantID := uuid.New()
	billingPeriod := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	tenant := &models.Tenant{
		ID:               tenantID,
		Name:             "Test Tenant",
		SubscriptionTier: "starter",
		MonthlyQuota:     10000,
	}

	usage := &models.QuotaUsage{
		TenantID:     tenantID,
		RequestCount: 5000,
		SuccessCount: 4800,
		FailureCount: 200,
		OverageCount: 0,
	}

	mockTenantRepo.On("GetByID", mock.Anything, tenantID).Return(tenant, nil)
	mockQuotaRepo.On("GetQuotaUsageByTenant", mock.Anything, tenantID, billingPeriod).Return(usage, nil)

	calculation, err := service.CalculateBilling(context.Background(), tenantID, billingPeriod)

	assert.NoError(t, err)
	assert.NotNil(t, calculation)
	assert.Equal(t, tenantID, calculation.TenantID)
	assert.Equal(t, "starter", calculation.SubscriptionTier)
	assert.Equal(t, 5000, calculation.BaseRequests)
	assert.Equal(t, 0, calculation.OverageRequests)
	assert.Equal(t, 0, calculation.BaseAmount)    // Free tier
	assert.Equal(t, 0, calculation.OverageAmount) // No overage
	assert.Equal(t, 0, calculation.TotalAmount)

	mockTenantRepo.AssertExpectations(t)
	mockQuotaRepo.AssertExpectations(t)
}

func TestBillingService_CalculateBilling_WithOverage(t *testing.T) {
	mockTenantRepo := &MockTenantRepository{}
	mockQuotaRepo := &MockQuotaRepository{}
	service := NewBillingService(mockTenantRepo, mockQuotaRepo)

	tenantID := uuid.New()
	billingPeriod := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	tenant := &models.Tenant{
		ID:               tenantID,
		Name:             "Test Tenant",
		SubscriptionTier: "starter",
		MonthlyQuota:     10000,
	}

	usage := &models.QuotaUsage{
		TenantID:     tenantID,
		RequestCount: 12000,
		SuccessCount: 11500,
		FailureCount: 500,
		OverageCount: 2000,
	}

	mockTenantRepo.On("GetByID", mock.Anything, tenantID).Return(tenant, nil)
	mockQuotaRepo.On("GetQuotaUsageByTenant", mock.Anything, tenantID, billingPeriod).Return(usage, nil)

	calculation, err := service.CalculateBilling(context.Background(), tenantID, billingPeriod)

	assert.NoError(t, err)
	assert.NotNil(t, calculation)
	assert.Equal(t, 10000, calculation.BaseRequests)
	assert.Equal(t, 2000, calculation.OverageRequests)
	assert.Equal(t, 0, calculation.BaseAmount)     // Free tier
	assert.Equal(t, 2000, calculation.OverageAmount) // 2000 * 1 cent
	assert.Equal(t, 2000, calculation.TotalAmount)

	mockTenantRepo.AssertExpectations(t)
	mockQuotaRepo.AssertExpectations(t)
}

func TestBillingService_ProcessTenantBilling(t *testing.T) {
	mockTenantRepo := &MockTenantRepository{}
	mockQuotaRepo := &MockQuotaRepository{}
	service := NewBillingService(mockTenantRepo, mockQuotaRepo)

	tenantID := uuid.New()
	billingPeriod := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	tenant := &models.Tenant{
		ID:               tenantID,
		Name:             "Test Tenant",
		SubscriptionTier: "starter",
		MonthlyQuota:     10000,
	}

	usage := &models.QuotaUsage{
		TenantID:     tenantID,
		RequestCount: 8000,
		SuccessCount: 7800,
		FailureCount: 200,
		OverageCount: 0,
	}

	mockTenantRepo.On("GetByID", mock.Anything, tenantID).Return(tenant, nil)
	mockQuotaRepo.On("GetQuotaUsageByTenant", mock.Anything, tenantID, billingPeriod).Return(usage, nil)
	mockQuotaRepo.On("GetBillingRecord", mock.Anything, tenantID, billingPeriod).Return(nil, assert.AnError)
	mockQuotaRepo.On("CreateBillingRecord", mock.Anything, mock.AnythingOfType("*models.BillingRecord")).Return(nil)

	err := service.ProcessTenantBilling(context.Background(), tenantID, billingPeriod)

	assert.NoError(t, err)

	mockTenantRepo.AssertExpectations(t)
	mockQuotaRepo.AssertExpectations(t)
}

func TestBillingService_EstimateBilling(t *testing.T) {
	mockTenantRepo := &MockTenantRepository{}
	mockQuotaRepo := &MockQuotaRepository{}
	service := NewBillingService(mockTenantRepo, mockQuotaRepo)

	tenantID := uuid.New()

	tenant := &models.Tenant{
		ID:               tenantID,
		Name:             "Test Tenant",
		SubscriptionTier: "starter",
		MonthlyQuota:     10000,
	}

	// Simulate usage for 10 days of a 30-day month
	now := time.Now().UTC()
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	
	usage := &models.QuotaUsage{
		TenantID:     tenantID,
		RequestCount: 3000, // 300 per day * 10 days
		SuccessCount: 2900,
		FailureCount: 100,
		OverageCount: 0,
	}

	mockTenantRepo.On("GetByID", mock.Anything, tenantID).Return(tenant, nil)
	mockQuotaRepo.On("GetQuotaUsageByTenant", mock.Anything, tenantID, currentMonth).Return(usage, nil)

	estimation, err := service.EstimateBilling(context.Background(), tenantID)

	assert.NoError(t, err)
	assert.NotNil(t, estimation)
	assert.Equal(t, tenantID, estimation.TenantID)
	
	// Should project usage for full month (3000 * 30/10 = 9000)
	// This is within quota, so no overage expected
	assert.Equal(t, 0, estimation.OverageAmount)

	mockTenantRepo.AssertExpectations(t)
	mockQuotaRepo.AssertExpectations(t)
}

func TestBillingService_GenerateBillingReport(t *testing.T) {
	mockTenantRepo := &MockTenantRepository{}
	mockQuotaRepo := &MockQuotaRepository{}
	service := NewBillingService(mockTenantRepo, mockQuotaRepo)

	tenantID := uuid.New()
	billingPeriod := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	previousPeriod := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	tenant := &models.Tenant{
		ID:               tenantID,
		Name:             "Test Tenant",
		SubscriptionTier: "starter",
		MonthlyQuota:     10000, // Use actual starter tier quota
	}

	currentUsage := &models.QuotaUsage{
		TenantID:     tenantID,
		RequestCount: 12000, // Over starter quota (10000) - will create overage charges
		SuccessCount: 11800,
		FailureCount: 200,
	}

	previousUsage := &models.QuotaUsage{
		TenantID:     tenantID,
		RequestCount: 8000, // Within starter quota (10000) - no overage charges
		SuccessCount: 7900,
		FailureCount: 100,
	}

	mockTenantRepo.On("GetByID", mock.Anything, tenantID).Return(tenant, nil)
	mockQuotaRepo.On("GetQuotaUsageByTenant", mock.Anything, tenantID, billingPeriod).Return(currentUsage, nil)
	mockQuotaRepo.On("GetQuotaUsageByTenant", mock.Anything, tenantID, previousPeriod).Return(previousUsage, nil)

	report, err := service.GenerateBillingReport(context.Background(), tenantID, billingPeriod)

	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, tenantID, report.TenantID)
	assert.Equal(t, "Test Tenant", report.TenantName)
	
	// Debug: Print the actual amounts
	t.Logf("Current total: %d, Previous total: %d", report.Calculation.TotalAmount, report.PreviousPeriod.TotalAmount)
	
	// Current period: 12000 requests, quota 10000, so 2000 overage * 1 cent = 2000 cents
	// Previous period: 8000 requests, quota 10000, so 0 overage * 1 cent = 0 cents
	assert.Equal(t, 2000, report.Calculation.TotalAmount)
	assert.Equal(t, 0, report.PreviousPeriod.TotalAmount)
	assert.Equal(t, "increasing", report.Trend)
	assert.NotNil(t, report.Calculation)
	assert.NotNil(t, report.PreviousPeriod)

	mockTenantRepo.AssertExpectations(t)
	mockQuotaRepo.AssertExpectations(t)
}