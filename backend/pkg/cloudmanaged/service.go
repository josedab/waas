package cloudmanaged

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service handles cloud managed offering business logic
type Service struct {
	repo Repository
}

// NewService creates a new cloud managed service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// DefaultPlans returns the predefined plan definitions
func DefaultPlans() []PlanDefinition {
	return []PlanDefinition{
		{
			Tier:          PlanTierFree,
			Name:          "Free",
			Description:   "For individual developers getting started",
			PriceMonthly:  0,
			WebhooksLimit: 1000,
			StorageLimit:  100 * 1024 * 1024, // 100 MB
			RetentionDays: 7,
			SupportLevel:  "community",
			Features:      []string{"api_access", "basic_analytics"},
		},
		{
			Tier:          PlanTierStarter,
			Name:          "Starter",
			Description:   "For small teams and projects",
			PriceMonthly:  2900,
			WebhooksLimit: 50000,
			StorageLimit:  1024 * 1024 * 1024, // 1 GB
			RetentionDays: 30,
			SupportLevel:  "email",
			Features:      []string{"api_access", "basic_analytics", "transformations", "team_members"},
		},
		{
			Tier:          PlanTierPro,
			Name:          "Pro",
			Description:   "For growing businesses with advanced needs",
			PriceMonthly:  9900,
			WebhooksLimit: 500000,
			StorageLimit:  10 * 1024 * 1024 * 1024, // 10 GB
			RetentionDays: 90,
			SupportLevel:  "priority",
			Features:      []string{"api_access", "advanced_analytics", "transformations", "team_members", "custom_domains", "audit_logs"},
		},
		{
			Tier:          PlanTierEnterprise,
			Name:          "Enterprise",
			Description:   "For large organizations with custom requirements",
			PriceMonthly:  49900,
			WebhooksLimit: 0, // Unlimited
			StorageLimit:  0, // Unlimited
			RetentionDays: 365,
			SupportLevel:  "dedicated",
			Features:      []string{"api_access", "advanced_analytics", "transformations", "team_members", "custom_domains", "audit_logs", "sso", "sla_guarantee", "dedicated_infra"},
		},
	}
}

// getPlanDefinition looks up a plan by tier
func getPlanDefinition(tier PlanTier) *PlanDefinition {
	for _, p := range DefaultPlans() {
		if p.Tier == tier {
			return &p
		}
	}
	return nil
}

// Signup creates a new cloud tenant with a trial period
func (s *Service) Signup(ctx context.Context, req *SignupRequest) (*CloudTenant, error) {
	plan := PlanTier(req.Plan)
	if plan == "" {
		plan = PlanTierFree
	}

	planDef := getPlanDefinition(plan)
	if planDef == nil {
		return nil, fmt.Errorf("invalid plan: %s", req.Plan)
	}

	region := req.Region
	if region == "" {
		region = "us-east-1"
	}

	now := time.Now()
	trialEnd := now.AddDate(0, 0, 14)

	tenant := &CloudTenant{
		ID:            uuid.New(),
		TenantID:      uuid.New().String(),
		Email:         req.Email,
		Org:           req.Org,
		Plan:          plan,
		Status:        CloudTenantStatusTrial,
		Region:        region,
		WebhooksUsed:  0,
		WebhooksLimit: planDef.WebhooksLimit,
		StorageUsed:   0,
		StorageLimit:  planDef.StorageLimit,
		CreatedAt:     now,
		TrialEndsAt:   &trialEnd,
	}

	if err := s.repo.CreateCloudTenant(ctx, tenant); err != nil {
		return nil, err
	}

	// Initialize onboarding progress
	progress := defaultOnboardingProgress(tenant.TenantID)
	if err := s.repo.SaveOnboardingProgress(ctx, progress); err != nil {
		return nil, err
	}

	return tenant, nil
}

// GetTenant retrieves a cloud tenant
func (s *Service) GetTenant(ctx context.Context, tenantID string) (*CloudTenant, error) {
	return s.repo.GetCloudTenant(ctx, tenantID)
}

// UpdateTenant updates a cloud tenant
func (s *Service) UpdateTenant(ctx context.Context, tenant *CloudTenant) (*CloudTenant, error) {
	if err := s.repo.UpdateCloudTenant(ctx, tenant); err != nil {
		return nil, err
	}
	return s.repo.GetCloudTenant(ctx, tenant.TenantID)
}

// UpgradePlan upgrades the tenant to a higher tier
func (s *Service) UpgradePlan(ctx context.Context, tenantID string, newTier PlanTier) (*CloudTenant, error) {
	tenant, err := s.repo.GetCloudTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	planDef := getPlanDefinition(newTier)
	if planDef == nil {
		return nil, fmt.Errorf("invalid plan: %s", newTier)
	}

	// Validate upgrade direction
	if tierRank(newTier) <= tierRank(tenant.Plan) {
		return nil, fmt.Errorf("new plan must be a higher tier than current plan")
	}

	tenant.Plan = newTier
	tenant.WebhooksLimit = planDef.WebhooksLimit
	tenant.StorageLimit = planDef.StorageLimit
	tenant.Status = CloudTenantStatusActive

	if err := s.repo.UpdateCloudTenant(ctx, tenant); err != nil {
		return nil, err
	}

	return tenant, nil
}

// DowngradePlan downgrades the tenant to a lower tier
func (s *Service) DowngradePlan(ctx context.Context, tenantID string, newTier PlanTier) (*CloudTenant, error) {
	tenant, err := s.repo.GetCloudTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	planDef := getPlanDefinition(newTier)
	if planDef == nil {
		return nil, fmt.Errorf("invalid plan: %s", newTier)
	}

	if tierRank(newTier) >= tierRank(tenant.Plan) {
		return nil, fmt.Errorf("new plan must be a lower tier than current plan")
	}

	// Verify current usage fits within new plan limits
	if planDef.WebhooksLimit > 0 && tenant.WebhooksUsed > planDef.WebhooksLimit {
		return nil, fmt.Errorf("current webhook usage exceeds new plan limit")
	}
	if planDef.StorageLimit > 0 && tenant.StorageUsed > planDef.StorageLimit {
		return nil, fmt.Errorf("current storage usage exceeds new plan limit")
	}

	tenant.Plan = newTier
	tenant.WebhooksLimit = planDef.WebhooksLimit
	tenant.StorageLimit = planDef.StorageLimit

	if err := s.repo.UpdateCloudTenant(ctx, tenant); err != nil {
		return nil, err
	}

	return tenant, nil
}

// GetAvailablePlans returns all plan definitions with limits
func (s *Service) GetAvailablePlans(ctx context.Context) []PlanDefinition {
	return DefaultPlans()
}

// GetUsageSummary returns aggregated usage for the current period
func (s *Service) GetUsageSummary(ctx context.Context, tenantID, period string) (*UsageSummary, error) {
	if period == "" {
		period = time.Now().Format("2006-01")
	}
	return s.repo.GetUsageSummary(ctx, tenantID, period)
}

// RecordUsage records a usage metric
func (s *Service) RecordUsage(ctx context.Context, tenantID, metricType string, value int64) error {
	meter := &UsageMeter{
		ID:         uuid.New(),
		TenantID:   tenantID,
		MetricType: metricType,
		Value:      value,
		Period:     time.Now().Format("2006-01"),
		RecordedAt: time.Now(),
	}

	return s.repo.RecordUsage(ctx, meter)
}

// CheckQuota verifies the tenant is within plan limits
func (s *Service) CheckQuota(ctx context.Context, tenantID string) (bool, error) {
	tenant, err := s.repo.GetCloudTenant(ctx, tenantID)
	if err != nil {
		return false, err
	}

	// Unlimited plans (limit == 0) always pass
	if tenant.WebhooksLimit > 0 && tenant.WebhooksUsed >= tenant.WebhooksLimit {
		return false, nil
	}
	if tenant.StorageLimit > 0 && tenant.StorageUsed >= tenant.StorageLimit {
		return false, nil
	}

	return true, nil
}

// GetBillingInfo retrieves billing information
func (s *Service) GetBillingInfo(ctx context.Context, tenantID string) (*BillingInfo, error) {
	return s.repo.GetBillingInfo(ctx, tenantID)
}

// UpdateBillingInfo updates billing information
func (s *Service) UpdateBillingInfo(ctx context.Context, tenantID string, req *UpdateBillingRequest) (*BillingInfo, error) {
	info, err := s.repo.GetBillingInfo(ctx, tenantID)
	if err != nil {
		// Create new billing info if not found
		info = &BillingInfo{TenantID: tenantID}
	}

	if req.PaymentMethod != "" {
		info.PaymentMethod = req.PaymentMethod
	}
	if req.BillingEmail != "" {
		info.BillingEmail = req.BillingEmail
	}

	if err := s.repo.SaveBillingInfo(ctx, info); err != nil {
		return nil, err
	}

	return info, nil
}

// GetOnboardingProgress retrieves the onboarding progress
func (s *Service) GetOnboardingProgress(ctx context.Context, tenantID string) (*OnboardingProgress, error) {
	return s.repo.GetOnboardingProgress(ctx, tenantID)
}

// CompleteOnboardingStep marks a step as completed
func (s *Service) CompleteOnboardingStep(ctx context.Context, tenantID, stepID string) (*OnboardingProgress, error) {
	progress, err := s.repo.GetOnboardingProgress(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	found := false
	completedCount := 0
	now := time.Now()

	for i, step := range progress.Steps {
		if step.StepID == stepID {
			progress.Steps[i].Completed = true
			progress.Steps[i].CompletedAt = &now
			found = true
		}
		if progress.Steps[i].Completed {
			completedCount++
		}
	}

	if !found {
		return nil, fmt.Errorf("onboarding step not found: %s", stepID)
	}

	if len(progress.Steps) > 0 {
		progress.CompletionPct = float64(completedCount) / float64(len(progress.Steps)) * 100
	}
	progress.AllCompleted = completedCount == len(progress.Steps)

	if err := s.repo.SaveOnboardingProgress(ctx, progress); err != nil {
		return nil, err
	}

	return progress, nil
}

// tierRank returns the numeric rank of a plan tier for comparison
func tierRank(tier PlanTier) int {
	switch tier {
	case PlanTierFree:
		return 0
	case PlanTierStarter:
		return 1
	case PlanTierPro:
		return 2
	case PlanTierEnterprise:
		return 3
	default:
		return -1
	}
}

// defaultOnboardingProgress returns the initial onboarding steps for a new tenant
func defaultOnboardingProgress(tenantID string) *OnboardingProgress {
	return &OnboardingProgress{
		TenantID: tenantID,
		Steps: []OnboardingStep{
			{StepID: "verify_email", Name: "Verify Email", Description: "Confirm your email address", Required: true},
			{StepID: "create_org", Name: "Create Organization", Description: "Set up your organization", Required: true},
			{StepID: "first_webhook", Name: "Create First Webhook", Description: "Set up your first webhook endpoint", Required: true},
			{StepID: "test_delivery", Name: "Test Delivery", Description: "Send a test webhook event", Required: false},
			{StepID: "invite_team", Name: "Invite Team", Description: "Add team members to your organization", Required: false},
		},
		CompletionPct: 0,
		AllCompleted:  false,
	}
}
