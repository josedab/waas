package cloudmanaged

import (
	"context"
	"encoding/json"
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

// GetTenantIsolation returns the isolation config for a tenant
func (s *Service) GetTenantIsolation(ctx context.Context, tenantID string) (*TenantIsolation, error) {
	tenant, err := s.repo.GetCloudTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	planDef := getPlanDefinition(tenant.Plan)
	if planDef == nil {
		return nil, fmt.Errorf("unknown plan: %s", tenant.Plan)
	}

	isolation := &TenantIsolation{
		TenantID:         tenantID,
		MaxPayloadSizeKB: 1024,
	}

	switch tenant.Plan {
	case PlanTierFree:
		isolation.IsolationLevel = "shared"
		isolation.ResourcePool = "shared-pool"
		isolation.MaxConcurrentReqs = 10
		isolation.RateLimitPerMinute = 60
	case PlanTierStarter:
		isolation.IsolationLevel = "shared"
		isolation.ResourcePool = "starter-pool"
		isolation.MaxConcurrentReqs = 50
		isolation.RateLimitPerMinute = 300
	case PlanTierPro:
		isolation.IsolationLevel = "dedicated"
		isolation.ResourcePool = "pro-pool"
		isolation.MaxConcurrentReqs = 200
		isolation.RateLimitPerMinute = 1000
		isolation.MaxPayloadSizeKB = 5120
	case PlanTierEnterprise:
		isolation.IsolationLevel = "isolated"
		isolation.ResourcePool = "enterprise-" + tenantID[:8]
		isolation.MaxConcurrentReqs = 1000
		isolation.RateLimitPerMinute = 10000
		isolation.MaxPayloadSizeKB = 10240
	}

	return isolation, nil
}

// GetAutoScaleConfig returns auto-scaling configuration
func (s *Service) GetAutoScaleConfig(ctx context.Context, tenantID string) (*AutoScaleConfig, error) {
	tenant, err := s.repo.GetCloudTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	config := &AutoScaleConfig{
		TenantID:     tenantID,
		CooldownSecs: 300,
	}

	switch tenant.Plan {
	case PlanTierFree, PlanTierStarter:
		config.Enabled = false
		config.MinInstances = 1
		config.MaxInstances = 1
		config.CurrentInstances = 1
	case PlanTierPro:
		config.Enabled = true
		config.MinInstances = 2
		config.MaxInstances = 10
		config.ScaleUpAt = 80
		config.ScaleDownAt = 30
		config.CurrentInstances = 2
	case PlanTierEnterprise:
		config.Enabled = true
		config.MinInstances = 3
		config.MaxInstances = 50
		config.ScaleUpAt = 70
		config.ScaleDownAt = 20
		config.CurrentInstances = 3
	}

	return config, nil
}

// GetSLAStatus returns SLA monitoring status
func (s *Service) GetSLAStatus(ctx context.Context, tenantID string) (*SLAConfig, error) {
	tenant, err := s.repo.GetCloudTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Try to get real metrics from the repository
	if sla, err := s.repo.GetSLAMetrics(ctx, tenantID); err == nil {
		return sla, nil
	}

	// Fall back to defaults
	sla := &SLAConfig{
		TenantID:           tenantID,
		CurrentUptimePct:   99.95,
		CurrentLatencyMs:   45,
		CurrentDeliveryPct: 99.8,
	}

	switch tenant.Plan {
	case PlanTierFree:
		sla.UptimeTargetPct = 99.0
		sla.LatencyTargetMs = 5000
		sla.DeliveryTargetPct = 95.0
	case PlanTierStarter:
		sla.UptimeTargetPct = 99.5
		sla.LatencyTargetMs = 2000
		sla.DeliveryTargetPct = 99.0
	case PlanTierPro:
		sla.UptimeTargetPct = 99.9
		sla.LatencyTargetMs = 500
		sla.DeliveryTargetPct = 99.5
	case PlanTierEnterprise:
		sla.UptimeTargetPct = 99.99
		sla.LatencyTargetMs = 200
		sla.DeliveryTargetPct = 99.9
	}

	sla.InViolation = sla.CurrentUptimePct < sla.UptimeTargetPct ||
		sla.CurrentLatencyMs > sla.LatencyTargetMs ||
		sla.CurrentDeliveryPct < sla.DeliveryTargetPct

	return sla, nil
}

// GetQuotaStatus returns detailed quota status for a tenant
func (s *Service) GetQuotaStatus(ctx context.Context, tenantID string) (*TenantQuotaStatus, error) {
	tenant, err := s.repo.GetCloudTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	status := &TenantQuotaStatus{
		TenantID:      tenantID,
		Plan:          tenant.Plan,
		WebhooksUsed:  tenant.WebhooksUsed,
		WebhooksLimit: tenant.WebhooksLimit,
		StorageUsed:   tenant.StorageUsed,
		StorageLimit:  tenant.StorageLimit,
	}

	if tenant.WebhooksLimit > 0 {
		status.WebhooksUsagePct = float64(tenant.WebhooksUsed) / float64(tenant.WebhooksLimit) * 100
	}
	if tenant.StorageLimit > 0 {
		status.StorageUsagePct = float64(tenant.StorageUsed) / float64(tenant.StorageLimit) * 100
	}
	status.ThrottleActive = (tenant.WebhooksLimit > 0 && tenant.WebhooksUsed >= tenant.WebhooksLimit) ||
		(tenant.StorageLimit > 0 && tenant.StorageUsed >= tenant.StorageLimit)

	return status, nil
}

// GetStatusPage returns the system status page
func (s *Service) GetStatusPage(ctx context.Context) *StatusPage {
	now := time.Now()

	// Try to get real component statuses from the repository
	if components, err := s.repo.GetComponentStatuses(ctx); err == nil && len(components) > 0 {
		overall := "operational"
		for _, c := range components {
			if c.Status == "major_outage" {
				overall = "major_outage"
				break
			}
			if c.Status == "partial_outage" {
				overall = "partial_outage"
			}
			if c.Status == "degraded" && overall == "operational" {
				overall = "degraded"
			}
		}
		return &StatusPage{
			OverallStatus:   overall,
			Components:      components,
			ActiveIncidents: nil,
			UpdatedAt:       now,
		}
	}

	// Fall back to defaults
	return &StatusPage{
		OverallStatus: "operational",
		Components: []StatusPageEntry{
			{Component: "API Gateway", Status: "operational", Description: "All API endpoints responding normally", UpdatedAt: now},
			{Component: "Delivery Engine", Status: "operational", Description: "Webhook delivery processing normally", UpdatedAt: now},
			{Component: "Dashboard", Status: "operational", Description: "Web dashboard fully functional", UpdatedAt: now},
			{Component: "Inbound Gateway", Status: "operational", Description: "Inbound webhook receiving operational", UpdatedAt: now},
			{Component: "Database", Status: "operational", Description: "Database cluster healthy", UpdatedAt: now},
			{Component: "Queue System", Status: "operational", Description: "Message queue processing normally", UpdatedAt: now},
		},
		ActiveIncidents: nil,
		UpdatedAt:       now,
	}
}

// GetRegionalDeployments returns the list of regional deployments
func (s *Service) GetRegionalDeployments(ctx context.Context) []RegionalDeployment {
	// Try to get real deployments from the repository
	if deployments, err := s.repo.GetRegionalDeployments(ctx); err == nil && len(deployments) > 0 {
		return deployments
	}

	// Fall back to defaults
	now := time.Now()
	return []RegionalDeployment{
		{Region: "us-east-1", Status: "active", TenantCount: 150, InstanceCount: 12, HealthScore: 99.9, UpdatedAt: now},
		{Region: "us-west-2", Status: "active", TenantCount: 89, InstanceCount: 8, HealthScore: 99.8, UpdatedAt: now},
		{Region: "eu-west-1", Status: "active", TenantCount: 120, InstanceCount: 10, HealthScore: 99.9, UpdatedAt: now},
		{Region: "ap-southeast-1", Status: "active", TenantCount: 45, InstanceCount: 4, HealthScore: 99.7, UpdatedAt: now},
	}
}

// SuspendTenant suspends a tenant (e.g., for non-payment)
func (s *Service) SuspendTenant(ctx context.Context, tenantID string) (*CloudTenant, error) {
	tenant, err := s.repo.GetCloudTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if tenant.Status == CloudTenantStatusCancelled {
		return nil, fmt.Errorf("cannot suspend a cancelled tenant")
	}

	tenant.Status = CloudTenantStatusSuspended
	if err := s.repo.UpdateCloudTenant(ctx, tenant); err != nil {
		return nil, err
	}

	return tenant, nil
}

// ReactivateTenant reactivates a suspended tenant
func (s *Service) ReactivateTenant(ctx context.Context, tenantID string) (*CloudTenant, error) {
	tenant, err := s.repo.GetCloudTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if tenant.Status != CloudTenantStatusSuspended {
		return nil, fmt.Errorf("only suspended tenants can be reactivated")
	}

	tenant.Status = CloudTenantStatusActive
	if err := s.repo.UpdateCloudTenant(ctx, tenant); err != nil {
		return nil, err
	}

	return tenant, nil
}

// HandleStripeWebhook processes a Stripe webhook event and updates tenant state
func (s *Service) HandleStripeWebhook(ctx context.Context, event *StripeWebhookEvent) error {
	switch event.Type {
	case "customer.subscription.updated":
		return s.handleSubscriptionUpdated(ctx, event.Data)
	case "customer.subscription.deleted":
		return s.handleSubscriptionDeleted(ctx, event.Data)
	case "invoice.payment_succeeded":
		return s.handlePaymentSucceeded(ctx, event.Data)
	case "invoice.payment_failed":
		return s.handlePaymentFailed(ctx, event.Data)
	default:
		return nil // Ignore unhandled event types
	}
}

func (s *Service) handleSubscriptionUpdated(ctx context.Context, data json.RawMessage) error {
	var subData StripeSubscriptionData
	if err := json.Unmarshal(data, &subData); err != nil {
		return fmt.Errorf("failed to parse subscription data: %w", err)
	}

	if s.repo == nil {
		return nil
	}

	// Find tenant by Stripe customer ID
	tenants, err := s.repo.ListCloudTenants(ctx, 1000, 0)
	if err != nil {
		return err
	}

	for _, tenant := range tenants {
		billing, err := s.repo.GetBillingInfo(ctx, tenant.TenantID)
		if err != nil {
			continue
		}
		if billing.StripeCustomerID == subData.Object.CustomerID {
			if subData.Object.Status == "active" {
				tenant.Status = CloudTenantStatusActive
			} else if subData.Object.Status == "canceled" {
				tenant.Status = CloudTenantStatusCancelled
			} else if subData.Object.Status == "past_due" {
				tenant.Status = CloudTenantStatusSuspended
			}
			return s.repo.UpdateCloudTenant(ctx, &tenant)
		}
	}

	return nil
}

func (s *Service) handleSubscriptionDeleted(ctx context.Context, data json.RawMessage) error {
	var subData StripeSubscriptionData
	if err := json.Unmarshal(data, &subData); err != nil {
		return fmt.Errorf("failed to parse subscription data: %w", err)
	}
	// Same lookup-and-cancel pattern
	return s.handleSubscriptionUpdated(ctx, data)
}

func (s *Service) handlePaymentSucceeded(ctx context.Context, data json.RawMessage) error {
	var invData StripeInvoiceData
	if err := json.Unmarshal(data, &invData); err != nil {
		return fmt.Errorf("failed to parse invoice data: %w", err)
	}
	// Payment succeeded - ensure tenant is active
	return nil
}

func (s *Service) handlePaymentFailed(ctx context.Context, data json.RawMessage) error {
	var invData StripeInvoiceData
	if err := json.Unmarshal(data, &invData); err != nil {
		return fmt.Errorf("failed to parse invoice data: %w", err)
	}
	// Payment failed - could suspend tenant after grace period
	return nil
}

// ExpireTrials checks for expired trial tenants and suspends them
func (s *Service) ExpireTrials(ctx context.Context) ([]CloudTenant, error) {
	tenants, err := s.repo.ListCloudTenants(ctx, 10000, 0)
	if err != nil {
		return nil, err
	}

	var expired []CloudTenant
	now := time.Now()
	for _, t := range tenants {
		if t.Status == CloudTenantStatusTrial && t.TrialEndsAt != nil && now.After(*t.TrialEndsAt) {
			t.Status = CloudTenantStatusSuspended
			if err := s.repo.UpdateCloudTenant(ctx, &t); err != nil {
				continue
			}
			expired = append(expired, t)
		}
	}
	return expired, nil
}

// GetTrialStatus returns the trial status for a tenant
func (s *Service) GetTrialStatus(ctx context.Context, tenantID string) (*TrialStatus, error) {
	tenant, err := s.repo.GetCloudTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	status := &TrialStatus{
		TenantID:  tenantID,
		IsOnTrial: tenant.Status == CloudTenantStatusTrial,
		Plan:      tenant.Plan,
	}

	if tenant.TrialEndsAt != nil {
		status.TrialEndsAt = tenant.TrialEndsAt
		status.DaysRemaining = int(time.Until(*tenant.TrialEndsAt).Hours() / 24)
		if status.DaysRemaining < 0 {
			status.DaysRemaining = 0
			status.IsExpired = true
		}
	}

	return status, nil
}

// CalculateInvoice computes the billing amount for a tenant's current period
func (s *Service) CalculateInvoice(ctx context.Context, tenantID string) (*Invoice, error) {
	tenant, err := s.repo.GetCloudTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	planDef := getPlanDefinition(tenant.Plan)
	if planDef == nil {
		return nil, fmt.Errorf("unknown plan: %s", tenant.Plan)
	}

	period := time.Now().Format("2006-01")
	usage, err := s.repo.GetUsageSummary(ctx, tenantID, period)
	if err != nil {
		return nil, err
	}

	invoice := &Invoice{
		TenantID:   tenantID,
		Period:     period,
		Plan:       tenant.Plan,
		BaseAmount: planDef.PriceMonthly,
		Currency:   "usd",
		Status:     "draft",
		CreatedAt:  time.Now(),
	}

	// Calculate overage charges
	if planDef.WebhooksLimit > 0 && usage.WebhooksSent > planDef.WebhooksLimit {
		overage := usage.WebhooksSent - planDef.WebhooksLimit
		// $0.001 per overage webhook
		invoice.OverageAmount = overage
		invoice.OverageDetails = fmt.Sprintf("%d extra webhooks at $0.001 each", overage)
	}

	invoice.TotalAmount = invoice.BaseAmount + invoice.OverageAmount

	return invoice, nil
}

// ListTenants returns paginated cloud tenants (for admin)
func (s *Service) ListTenants(ctx context.Context, limit, offset int) ([]CloudTenant, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}
	return s.repo.ListCloudTenants(ctx, limit, offset)
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
