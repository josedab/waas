package auth

import (
	"context"
	"fmt"
	"time"
	"webhook-platform/pkg/models"
	"webhook-platform/pkg/repository"

	"github.com/google/uuid"
)

// SubscriptionService manages tenant subscription tiers, including upgrades,
// downgrades, and proration calculations.
type SubscriptionService struct {
	tenantRepo repository.TenantRepository
	quotaRepo  repository.QuotaRepository
}

// SubscriptionUpdate describes a requested tier change for a tenant.
type SubscriptionUpdate struct {
	TenantID         uuid.UUID `json:"tenant_id"`
	NewTier          string    `json:"new_tier"`
	EffectiveDate    time.Time `json:"effective_date,omitempty"`
	PreserveUsage    bool      `json:"preserve_usage"`
}

// SubscriptionInfo aggregates a tenant's current subscription details,
// usage, billing history, and available tier change options.
type SubscriptionInfo struct {
	Tenant           *models.Tenant           `json:"tenant"`
	TierConfig       models.SubscriptionTier  `json:"tier_config"`
	CurrentUsage     *models.QuotaUsage       `json:"current_usage"`
	BillingHistory   []*models.BillingRecord  `json:"billing_history,omitempty"`
	UpgradeOptions   []models.SubscriptionTier `json:"upgrade_options"`
	DowngradeOptions []models.SubscriptionTier `json:"downgrade_options"`
}

// NewSubscriptionService creates a SubscriptionService with the given repositories.
func NewSubscriptionService(tenantRepo repository.TenantRepository, quotaRepo repository.QuotaRepository) *SubscriptionService {
	return &SubscriptionService{
		tenantRepo: tenantRepo,
		quotaRepo:  quotaRepo,
	}
}

// GetSubscriptionInfo returns comprehensive subscription information for a tenant
func (s *SubscriptionService) GetSubscriptionInfo(ctx context.Context, tenantID uuid.UUID) (*SubscriptionInfo, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	tierConfig, exists := models.GetTierConfig(tenant.SubscriptionTier)
	if !exists {
		return nil, fmt.Errorf("invalid subscription tier: %s", tenant.SubscriptionTier)
	}

	// Get current month usage
	now := time.Now().UTC()
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	
	currentUsage, err := s.quotaRepo.GetQuotaUsageByTenant(ctx, tenantID, currentMonth)
	if err != nil {
		// Create empty usage if not found
		currentUsage = &models.QuotaUsage{
			TenantID:     tenantID,
			Month:        currentMonth,
			RequestCount: 0,
			SuccessCount: 0,
			FailureCount: 0,
			OverageCount: 0,
		}
	}

	// Get billing history (last 6 months)
	billingHistory, err := s.quotaRepo.GetBillingHistory(ctx, tenantID, 6, 0)
	if err != nil {
		billingHistory = []*models.BillingRecord{}
	}

	// Get upgrade/downgrade options
	upgradeOptions, downgradeOptions := s.getSubscriptionOptions(tenant.SubscriptionTier)

	return &SubscriptionInfo{
		Tenant:           tenant,
		TierConfig:       tierConfig,
		CurrentUsage:     currentUsage,
		BillingHistory:   billingHistory,
		UpgradeOptions:   upgradeOptions,
		DowngradeOptions: downgradeOptions,
	}, nil
}

// UpdateSubscription changes a tenant's subscription tier
func (s *SubscriptionService) UpdateSubscription(ctx context.Context, update *SubscriptionUpdate) error {
	// Validate new tier
	newTierConfig, exists := models.GetTierConfig(update.NewTier)
	if !exists {
		return fmt.Errorf("invalid subscription tier: %s", update.NewTier)
	}

	// Get current tenant
	tenant, err := s.tenantRepo.GetByID(ctx, update.TenantID)
	if err != nil {
		return fmt.Errorf("failed to get tenant: %w", err)
	}

	oldTier := tenant.SubscriptionTier

	// Update tenant with new tier configuration
	tenant.SubscriptionTier = update.NewTier
	tenant.MonthlyQuota = newTierConfig.MonthlyQuota
	tenant.RateLimitPerMinute = newTierConfig.RateLimitPerMinute
	tenant.UpdatedAt = time.Now().UTC()

	err = s.tenantRepo.Update(ctx, tenant)
	if err != nil {
		return fmt.Errorf("failed to update tenant: %w", err)
	}

	// Handle usage preservation or reset
	if !update.PreserveUsage {
		// Reset current month usage if not preserving
		now := time.Now().UTC()
		currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		
		usage, err := s.quotaRepo.GetQuotaUsageByTenant(ctx, update.TenantID, currentMonth)
		if err == nil {
			usage.RequestCount = 0
			usage.SuccessCount = 0
			usage.FailureCount = 0
			usage.OverageCount = 0
			s.quotaRepo.UpdateQuotaUsage(ctx, usage)
		}
	}

	// Create audit log or notification for subscription change
	s.logSubscriptionChange(ctx, update.TenantID, oldTier, update.NewTier)

	return nil
}

// CalculateProration calculates prorated charges for mid-month subscription changes
func (s *SubscriptionService) CalculateProration(ctx context.Context, tenantID uuid.UUID, newTier string, changeDate time.Time) (int, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return 0, fmt.Errorf("failed to get tenant: %w", err)
	}

	oldTierConfig, exists := models.GetTierConfig(tenant.SubscriptionTier)
	if !exists {
		return 0, fmt.Errorf("invalid current tier: %s", tenant.SubscriptionTier)
	}

	newTierConfig, exists := models.GetTierConfig(newTier)
	if !exists {
		return 0, fmt.Errorf("invalid new tier: %s", newTier)
	}

	// Calculate days remaining in month
	now := time.Now().UTC()
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	nextMonth := currentMonth.AddDate(0, 1, 0)
	
	totalDaysInMonth := int(nextMonth.Sub(currentMonth).Hours() / 24)
	daysRemaining := int(nextMonth.Sub(changeDate).Hours() / 24)
	
	if daysRemaining <= 0 {
		return 0, nil
	}

	// Calculate prorated difference (simplified - in real implementation you'd have base prices)
	// This is a placeholder calculation
	oldDailyCost := oldTierConfig.PricePerRequest * oldTierConfig.MonthlyQuota / totalDaysInMonth
	newDailyCost := newTierConfig.PricePerRequest * newTierConfig.MonthlyQuota / totalDaysInMonth
	
	prorationAmount := (newDailyCost - oldDailyCost) * daysRemaining

	return prorationAmount, nil
}

// ValidateSubscriptionChange checks if a subscription change is allowed
func (s *SubscriptionService) ValidateSubscriptionChange(ctx context.Context, tenantID uuid.UUID, newTier string) error {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get tenant: %w", err)
	}

	if tenant.SubscriptionTier == newTier {
		return fmt.Errorf("tenant is already on tier: %s", newTier)
	}

	newTierConfig, exists := models.GetTierConfig(newTier)
	if !exists {
		return fmt.Errorf("invalid subscription tier: %s", newTier)
	}

	// Check if downgrading would violate current usage
	currentTierConfig, _ := models.GetTierConfig(tenant.SubscriptionTier)
	
	if newTierConfig.MonthlyQuota < currentTierConfig.MonthlyQuota {
		// This is a downgrade - check current usage
		now := time.Now().UTC()
		currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		
		usage, err := s.quotaRepo.GetQuotaUsageByTenant(ctx, tenantID, currentMonth)
		if err == nil && usage.RequestCount > newTierConfig.MonthlyQuota {
			return fmt.Errorf("cannot downgrade: current usage (%d) exceeds new tier quota (%d)", 
				usage.RequestCount, newTierConfig.MonthlyQuota)
		}
	}

	return nil
}

// getSubscriptionOptions returns available upgrade and downgrade options
func (s *SubscriptionService) getSubscriptionOptions(currentTier string) ([]models.SubscriptionTier, []models.SubscriptionTier) {
	allTiers := models.GetSubscriptionTiers()
	currentTierConfig, exists := allTiers[currentTier]
	if !exists {
		return []models.SubscriptionTier{}, []models.SubscriptionTier{}
	}

	var upgrades, downgrades []models.SubscriptionTier

	for _, tier := range allTiers {
		if tier.Name == currentTier {
			continue
		}

		if tier.MonthlyQuota > currentTierConfig.MonthlyQuota {
			upgrades = append(upgrades, tier)
		} else if tier.MonthlyQuota < currentTierConfig.MonthlyQuota {
			downgrades = append(downgrades, tier)
		}
	}

	return upgrades, downgrades
}

// logSubscriptionChange creates an audit log for subscription changes
func (s *SubscriptionService) logSubscriptionChange(ctx context.Context, tenantID uuid.UUID, oldTier, newTier string) {
	// In a real implementation, this would write to an audit log
	// For now, we could create a notification
	notification := &models.QuotaNotification{
		TenantID:    tenantID,
		Type:        "subscription_change",
		Threshold:   0,
		UsageCount:  0,
		QuotaLimit:  0,
		Sent:        false,
	}
	
	s.quotaRepo.CreateQuotaNotification(ctx, notification)
}

// GetAvailableTiers returns all available subscription tiers
func (s *SubscriptionService) GetAvailableTiers() map[string]models.SubscriptionTier {
	return models.GetSubscriptionTiers()
}