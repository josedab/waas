package auth

import (
	"context"
	"fmt"
	"time"
	"webhook-platform/pkg/models"
	"webhook-platform/pkg/repository"

	"github.com/google/uuid"
)

type BillingService struct {
	tenantRepo repository.TenantRepository
	quotaRepo  repository.QuotaRepository
}

type BillingCalculation struct {
	TenantID        uuid.UUID `json:"tenant_id"`
	BillingPeriod   time.Time `json:"billing_period"`
	SubscriptionTier string   `json:"subscription_tier"`
	BaseRequests    int       `json:"base_requests"`
	OverageRequests int       `json:"overage_requests"`
	BaseAmount      int       `json:"base_amount"`      // in cents
	OverageAmount   int       `json:"overage_amount"`   // in cents
	TotalAmount     int       `json:"total_amount"`     // in cents
	UsageDetails    *models.QuotaUsage `json:"usage_details"`
}

type BillingReport struct {
	TenantID       uuid.UUID            `json:"tenant_id"`
	TenantName     string               `json:"tenant_name"`
	Period         time.Time            `json:"period"`
	Calculation    *BillingCalculation  `json:"calculation"`
	PreviousPeriod *BillingCalculation  `json:"previous_period,omitempty"`
	Trend          string               `json:"trend"` // increasing, decreasing, stable
}

func NewBillingService(tenantRepo repository.TenantRepository, quotaRepo repository.QuotaRepository) *BillingService {
	return &BillingService{
		tenantRepo: tenantRepo,
		quotaRepo:  quotaRepo,
	}
}

// CalculateBilling calculates billing for a tenant for a specific month
func (bs *BillingService) CalculateBilling(ctx context.Context, tenantID uuid.UUID, billingPeriod time.Time) (*BillingCalculation, error) {
	tenant, err := bs.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	tierConfig, exists := models.GetTierConfig(tenant.SubscriptionTier)
	if !exists {
		return nil, fmt.Errorf("invalid subscription tier: %s", tenant.SubscriptionTier)
	}

	// Normalize billing period to first day of month
	periodStart := time.Date(billingPeriod.Year(), billingPeriod.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Get usage for the billing period
	usage, err := bs.quotaRepo.GetQuotaUsageByTenant(ctx, tenantID, periodStart)
	if err != nil {
		// If no usage found, create empty usage
		usage = &models.QuotaUsage{
			TenantID:     tenantID,
			Month:        periodStart,
			RequestCount: 0,
			SuccessCount: 0,
			FailureCount: 0,
			OverageCount: 0,
		}
	}

	// Calculate base and overage requests
	baseRequests := minInt(usage.RequestCount, tierConfig.MonthlyQuota)
	overageRequests := maxInt(0, usage.RequestCount-tierConfig.MonthlyQuota)

	// Calculate amounts
	baseAmount := baseRequests * tierConfig.PricePerRequest
	overageAmount := overageRequests * tierConfig.OverageRate
	totalAmount := baseAmount + overageAmount

	return &BillingCalculation{
		TenantID:        tenantID,
		BillingPeriod:   periodStart,
		SubscriptionTier: tenant.SubscriptionTier,
		BaseRequests:    baseRequests,
		OverageRequests: overageRequests,
		BaseAmount:      baseAmount,
		OverageAmount:   overageAmount,
		TotalAmount:     totalAmount,
		UsageDetails:    usage,
	}, nil
}

// ProcessMonthlyBilling processes billing for all tenants for a given month
func (bs *BillingService) ProcessMonthlyBilling(ctx context.Context, billingPeriod time.Time) error {
	periodStart := time.Date(billingPeriod.Year(), billingPeriod.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Get all tenants (in a real implementation, you'd paginate this)
	tenants, err := bs.tenantRepo.List(ctx, 1000, 0)
	if err != nil {
		return fmt.Errorf("failed to get tenants: %w", err)
	}

	for _, tenant := range tenants {
		err := bs.ProcessTenantBilling(ctx, tenant.ID, periodStart)
		if err != nil {
			// Log error but continue processing other tenants
			continue
		}
	}

	return nil
}

// ProcessTenantBilling processes billing for a specific tenant
func (bs *BillingService) ProcessTenantBilling(ctx context.Context, tenantID uuid.UUID, billingPeriod time.Time) error {
	// Check if billing record already exists
	existingRecord, err := bs.quotaRepo.GetBillingRecord(ctx, tenantID, billingPeriod)
	if err == nil && existingRecord.Status == "processed" {
		return fmt.Errorf("billing already processed for tenant %s, period %s", tenantID, billingPeriod.Format("2006-01"))
	}

	// Calculate billing
	calculation, err := bs.CalculateBilling(ctx, tenantID, billingPeriod)
	if err != nil {
		return fmt.Errorf("failed to calculate billing: %w", err)
	}

	// Create or update billing record
	billingRecord := &models.BillingRecord{
		TenantID:        tenantID,
		BillingPeriod:   calculation.BillingPeriod,
		BaseRequests:    calculation.BaseRequests,
		OverageRequests: calculation.OverageRequests,
		BaseAmount:      calculation.BaseAmount,
		OverageAmount:   calculation.OverageAmount,
		TotalAmount:     calculation.TotalAmount,
		Status:          "pending",
	}

	if existingRecord != nil {
		// Update existing record
		billingRecord.ID = existingRecord.ID
		err = bs.quotaRepo.UpdateBillingRecord(ctx, billingRecord)
	} else {
		// Create new record
		err = bs.quotaRepo.CreateBillingRecord(ctx, billingRecord)
	}

	if err != nil {
		return fmt.Errorf("failed to save billing record: %w", err)
	}

	return nil
}

// GenerateBillingReport generates a comprehensive billing report for a tenant
func (bs *BillingService) GenerateBillingReport(ctx context.Context, tenantID uuid.UUID, billingPeriod time.Time) (*BillingReport, error) {
	tenant, err := bs.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	// Calculate current period billing
	currentCalculation, err := bs.CalculateBilling(ctx, tenantID, billingPeriod)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate current billing: %w", err)
	}

	// Calculate previous period for trend analysis
	previousPeriod := billingPeriod.AddDate(0, -1, 0)
	previousCalculation, err := bs.CalculateBilling(ctx, tenantID, previousPeriod)
	if err != nil {
		// Previous period might not exist, that's okay
		previousCalculation = nil
	}

	// Determine trend
	trend := "stable"
	if previousCalculation != nil {
		if currentCalculation.TotalAmount > previousCalculation.TotalAmount {
			trend = "increasing"
		} else if currentCalculation.TotalAmount < previousCalculation.TotalAmount {
			trend = "decreasing"
		}
	}

	return &BillingReport{
		TenantID:       tenantID,
		TenantName:     tenant.Name,
		Period:         billingPeriod,
		Calculation:    currentCalculation,
		PreviousPeriod: previousCalculation,
		Trend:          trend,
	}, nil
}

// GetBillingHistory returns billing history for a tenant
func (bs *BillingService) GetBillingHistory(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*models.BillingRecord, error) {
	return bs.quotaRepo.GetBillingHistory(ctx, tenantID, limit, offset)
}

// EstimateBilling estimates billing for current month based on current usage
func (bs *BillingService) EstimateBilling(ctx context.Context, tenantID uuid.UUID) (*BillingCalculation, error) {
	now := time.Now().UTC()
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	
	tenant, err := bs.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	tierConfig, exists := models.GetTierConfig(tenant.SubscriptionTier)
	if !exists {
		return nil, fmt.Errorf("invalid subscription tier: %s", tenant.SubscriptionTier)
	}

	// Get current usage
	usage, err := bs.quotaRepo.GetQuotaUsageByTenant(ctx, tenantID, currentMonth)
	if err != nil {
		usage = &models.QuotaUsage{
			TenantID:     tenantID,
			Month:        currentMonth,
			RequestCount: 0,
			SuccessCount: 0,
			FailureCount: 0,
			OverageCount: 0,
		}
	}

	// Project usage for full month based on current usage and days elapsed
	daysInMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	daysElapsed := now.Day()
	
	projectedRequests := usage.RequestCount
	if daysElapsed > 0 {
		dailyAverage := float64(usage.RequestCount) / float64(daysElapsed)
		projectedRequests = int(dailyAverage * float64(daysInMonth))
	}

	// Calculate projected billing
	baseRequests := minInt(projectedRequests, tierConfig.MonthlyQuota)
	overageRequests := maxInt(0, projectedRequests-tierConfig.MonthlyQuota)
	
	baseAmount := baseRequests * tierConfig.PricePerRequest
	overageAmount := overageRequests * tierConfig.OverageRate
	totalAmount := baseAmount + overageAmount

	return &BillingCalculation{
		TenantID:        tenantID,
		BillingPeriod:   currentMonth,
		SubscriptionTier: tenant.SubscriptionTier,
		BaseRequests:    baseRequests,
		OverageRequests: overageRequests,
		BaseAmount:      baseAmount,
		OverageAmount:   overageAmount,
		TotalAmount:     totalAmount,
		UsageDetails:    usage,
	}, nil
}

// MarkBillingProcessed marks a billing record as processed
func (bs *BillingService) MarkBillingProcessed(ctx context.Context, tenantID uuid.UUID, billingPeriod time.Time) error {
	record, err := bs.quotaRepo.GetBillingRecord(ctx, tenantID, billingPeriod)
	if err != nil {
		return fmt.Errorf("billing record not found: %w", err)
	}

	record.Status = "processed"
	return bs.quotaRepo.UpdateBillingRecord(ctx, record)
}

// Helper functions
