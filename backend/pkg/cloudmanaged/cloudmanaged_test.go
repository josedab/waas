package cloudmanaged

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

// mockRepository implements Repository with in-memory maps
type mockRepository struct {
	tenants     map[string]*CloudTenant
	usage       map[string]*UsageSummary
	usageMeters map[string][]UsageMeter
	billing     map[string]*BillingInfo
	onboarding  map[string]*OnboardingProgress
	slaMetrics  map[string]*SLAConfig
	components  []StatusPageEntry
	deployments []RegionalDeployment
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		tenants:     make(map[string]*CloudTenant),
		usage:       make(map[string]*UsageSummary),
		usageMeters: make(map[string][]UsageMeter),
		billing:     make(map[string]*BillingInfo),
		onboarding:  make(map[string]*OnboardingProgress),
		slaMetrics:  make(map[string]*SLAConfig),
	}
}

func (m *mockRepository) CreateCloudTenant(_ context.Context, tenant *CloudTenant) error {
	m.tenants[tenant.TenantID] = tenant
	return nil
}

func (m *mockRepository) GetCloudTenant(_ context.Context, tenantID string) (*CloudTenant, error) {
	t, ok := m.tenants[tenantID]
	if !ok {
		return nil, fmt.Errorf("cloud tenant not found")
	}
	return t, nil
}

func (m *mockRepository) UpdateCloudTenant(_ context.Context, tenant *CloudTenant) error {
	if _, ok := m.tenants[tenant.TenantID]; !ok {
		return fmt.Errorf("cloud tenant not found")
	}
	m.tenants[tenant.TenantID] = tenant
	return nil
}

func (m *mockRepository) ListCloudTenants(_ context.Context, limit, offset int) ([]CloudTenant, error) {
	var result []CloudTenant
	for _, t := range m.tenants {
		result = append(result, *t)
	}
	return result, nil
}

func (m *mockRepository) RecordUsage(_ context.Context, meter *UsageMeter) error {
	m.usageMeters[meter.TenantID] = append(m.usageMeters[meter.TenantID], *meter)
	return nil
}

func (m *mockRepository) GetUsageSummary(_ context.Context, tenantID, period string) (*UsageSummary, error) {
	key := tenantID + ":" + period
	s, ok := m.usage[key]
	if !ok {
		return &UsageSummary{TenantID: tenantID, Period: period}, nil
	}
	return s, nil
}

func (m *mockRepository) GetUsageHistory(_ context.Context, tenantID string, limit int) ([]UsageMeter, error) {
	return m.usageMeters[tenantID], nil
}

func (m *mockRepository) SaveBillingInfo(_ context.Context, info *BillingInfo) error {
	m.billing[info.TenantID] = info
	return nil
}

func (m *mockRepository) GetBillingInfo(_ context.Context, tenantID string) (*BillingInfo, error) {
	b, ok := m.billing[tenantID]
	if !ok {
		return nil, fmt.Errorf("billing info not found")
	}
	return b, nil
}

func (m *mockRepository) SaveOnboardingProgress(_ context.Context, progress *OnboardingProgress) error {
	m.onboarding[progress.TenantID] = progress
	return nil
}

func (m *mockRepository) GetOnboardingProgress(_ context.Context, tenantID string) (*OnboardingProgress, error) {
	p, ok := m.onboarding[tenantID]
	if !ok {
		return nil, fmt.Errorf("onboarding progress not found")
	}
	return p, nil
}

func (m *mockRepository) GetSLAMetrics(_ context.Context, tenantID string) (*SLAConfig, error) {
	s, ok := m.slaMetrics[tenantID]
	if !ok {
		return nil, fmt.Errorf("sla metrics not found")
	}
	return s, nil
}

func (m *mockRepository) GetComponentStatuses(_ context.Context) ([]StatusPageEntry, error) {
	if len(m.components) == 0 {
		return nil, fmt.Errorf("no component statuses")
	}
	return m.components, nil
}

func (m *mockRepository) GetRegionalDeployments(_ context.Context) ([]RegionalDeployment, error) {
	if len(m.deployments) == 0 {
		return nil, fmt.Errorf("no regional deployments")
	}
	return m.deployments, nil
}

// helper to create a tenant via the service
func setupTenantWithPlan(t *testing.T, svc *Service, plan string) *CloudTenant {
	t.Helper()
	ctx := context.Background()
	tenant, err := svc.Signup(ctx, &SignupRequest{
		Email:  "test@example.com",
		Org:    "testorg",
		Plan:   plan,
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatalf("Signup failed: %v", err)
	}
	return tenant
}

func TestSignup_HappyPath(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenant := setupTenantWithPlan(t, svc, "starter")

	if tenant.Plan != PlanTierStarter {
		t.Errorf("expected plan starter, got %s", tenant.Plan)
	}
	if tenant.Status != CloudTenantStatusTrial {
		t.Errorf("expected status trial, got %s", tenant.Status)
	}
	if tenant.WebhooksLimit != 50000 {
		t.Errorf("expected webhooks_limit 50000, got %d", tenant.WebhooksLimit)
	}
	if tenant.TrialEndsAt == nil {
		t.Error("expected trial_ends_at to be set")
	}
	// Onboarding should be initialized
	progress, err := repo.GetOnboardingProgress(context.Background(), tenant.TenantID)
	if err != nil {
		t.Fatalf("expected onboarding progress, got error: %v", err)
	}
	if len(progress.Steps) == 0 {
		t.Error("expected onboarding steps to be initialized")
	}
}

func TestSignup_DefaultPlan(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenant := setupTenantWithPlan(t, svc, "")

	if tenant.Plan != PlanTierFree {
		t.Errorf("expected plan free, got %s", tenant.Plan)
	}
}

func TestSignup_InvalidPlan(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	_, err := svc.Signup(ctx, &SignupRequest{
		Email: "test@example.com",
		Org:   "testorg",
		Plan:  "nonexistent",
	})
	if err == nil {
		t.Error("expected error for invalid plan")
	}
}

func TestUpgradePlan_Valid(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tenant := setupTenantWithPlan(t, svc, "free")

	upgraded, err := svc.UpgradePlan(ctx, tenant.TenantID, PlanTierStarter)
	if err != nil {
		t.Fatalf("UpgradePlan failed: %v", err)
	}
	if upgraded.Plan != PlanTierStarter {
		t.Errorf("expected plan starter, got %s", upgraded.Plan)
	}
	if upgraded.Status != CloudTenantStatusActive {
		t.Errorf("expected status active after upgrade, got %s", upgraded.Status)
	}
	if upgraded.WebhooksLimit != 50000 {
		t.Errorf("expected webhooks_limit 50000, got %d", upgraded.WebhooksLimit)
	}
}

func TestUpgradePlan_InvalidDirection(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tenant := setupTenantWithPlan(t, svc, "pro")

	_, err := svc.UpgradePlan(ctx, tenant.TenantID, PlanTierStarter)
	if err == nil {
		t.Error("expected error when downgrading via UpgradePlan")
	}

	// Same tier should also fail
	_, err = svc.UpgradePlan(ctx, tenant.TenantID, PlanTierPro)
	if err == nil {
		t.Error("expected error when upgrading to same tier")
	}
}

func TestDowngradePlan_Valid(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tenant := setupTenantWithPlan(t, svc, "pro")

	downgraded, err := svc.DowngradePlan(ctx, tenant.TenantID, PlanTierStarter)
	if err != nil {
		t.Fatalf("DowngradePlan failed: %v", err)
	}
	if downgraded.Plan != PlanTierStarter {
		t.Errorf("expected plan starter, got %s", downgraded.Plan)
	}
}

func TestDowngradePlan_UsageExceedsLimits(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tenant := setupTenantWithPlan(t, svc, "pro")

	// Set usage that exceeds the free plan limits
	tenant.WebhooksUsed = 5000
	tenant.StorageUsed = 500 * 1024 * 1024 // 500 MB
	if err := repo.UpdateCloudTenant(ctx, tenant); err != nil {
		t.Fatalf("UpdateCloudTenant failed: %v", err)
	}

	_, err := svc.DowngradePlan(ctx, tenant.TenantID, PlanTierFree)
	if err == nil {
		t.Error("expected error when usage exceeds new plan limits")
	}
}

func TestDowngradePlan_InvalidDirection(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tenant := setupTenantWithPlan(t, svc, "starter")

	_, err := svc.DowngradePlan(ctx, tenant.TenantID, PlanTierPro)
	if err == nil {
		t.Error("expected error when upgrading via DowngradePlan")
	}
}

func TestCheckQuota_WithinLimits(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tenant := setupTenantWithPlan(t, svc, "starter")

	tenant.WebhooksUsed = 100
	tenant.StorageUsed = 1024
	if err := repo.UpdateCloudTenant(ctx, tenant); err != nil {
		t.Fatalf("UpdateCloudTenant failed: %v", err)
	}

	ok, err := svc.CheckQuota(ctx, tenant.TenantID)
	if err != nil {
		t.Fatalf("CheckQuota failed: %v", err)
	}
	if !ok {
		t.Error("expected quota check to pass")
	}
}

func TestCheckQuota_Exceeded(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tenant := setupTenantWithPlan(t, svc, "free")

	tenant.WebhooksUsed = 1000 // at the limit
	if err := repo.UpdateCloudTenant(ctx, tenant); err != nil {
		t.Fatalf("UpdateCloudTenant failed: %v", err)
	}

	ok, err := svc.CheckQuota(ctx, tenant.TenantID)
	if err != nil {
		t.Fatalf("CheckQuota failed: %v", err)
	}
	if ok {
		t.Error("expected quota check to fail when at limit")
	}
}

func TestCheckQuota_UnlimitedPlan(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tenant := setupTenantWithPlan(t, svc, "enterprise")

	// Enterprise has unlimited (limit=0), so any usage should pass
	tenant.WebhooksUsed = 999999
	tenant.StorageUsed = 999999999
	if err := repo.UpdateCloudTenant(ctx, tenant); err != nil {
		t.Fatalf("UpdateCloudTenant failed: %v", err)
	}

	ok, err := svc.CheckQuota(ctx, tenant.TenantID)
	if err != nil {
		t.Fatalf("CheckQuota failed: %v", err)
	}
	if !ok {
		t.Error("expected unlimited plan to always pass quota check")
	}
}

func TestGetTenantIsolation_PerTier(t *testing.T) {
	tests := []struct {
		plan           string
		isolationLevel string
		resourcePool   string
	}{
		{"free", "shared", "shared-pool"},
		{"starter", "shared", "starter-pool"},
		{"pro", "dedicated", "pro-pool"},
		{"enterprise", "isolated", "enterprise-"},
	}

	for _, tt := range tests {
		t.Run(tt.plan, func(t *testing.T) {
			repo := newMockRepository()
			svc := NewService(repo)
			ctx := context.Background()

			tenant := setupTenantWithPlan(t, svc, tt.plan)

			isolation, err := svc.GetTenantIsolation(ctx, tenant.TenantID)
			if err != nil {
				t.Fatalf("GetTenantIsolation failed: %v", err)
			}
			if isolation.IsolationLevel != tt.isolationLevel {
				t.Errorf("expected isolation level %s, got %s", tt.isolationLevel, isolation.IsolationLevel)
			}
			if tt.plan == "enterprise" {
				if isolation.ResourcePool[:11] != "enterprise-" {
					t.Errorf("expected resource pool to start with enterprise-, got %s", isolation.ResourcePool)
				}
			} else {
				if isolation.ResourcePool != tt.resourcePool {
					t.Errorf("expected resource pool %s, got %s", tt.resourcePool, isolation.ResourcePool)
				}
			}
		})
	}
}

func TestSuspendTenant(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tenant := setupTenantWithPlan(t, svc, "starter")

	suspended, err := svc.SuspendTenant(ctx, tenant.TenantID)
	if err != nil {
		t.Fatalf("SuspendTenant failed: %v", err)
	}
	if suspended.Status != CloudTenantStatusSuspended {
		t.Errorf("expected status suspended, got %s", suspended.Status)
	}
}

func TestSuspendTenant_CancelledFails(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tenant := setupTenantWithPlan(t, svc, "starter")
	tenant.Status = CloudTenantStatusCancelled
	_ = repo.UpdateCloudTenant(ctx, tenant)

	_, err := svc.SuspendTenant(ctx, tenant.TenantID)
	if err == nil {
		t.Error("expected error when suspending cancelled tenant")
	}
}

func TestReactivateTenant(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tenant := setupTenantWithPlan(t, svc, "starter")

	// First suspend
	_, err := svc.SuspendTenant(ctx, tenant.TenantID)
	if err != nil {
		t.Fatalf("SuspendTenant failed: %v", err)
	}

	// Then reactivate
	reactivated, err := svc.ReactivateTenant(ctx, tenant.TenantID)
	if err != nil {
		t.Fatalf("ReactivateTenant failed: %v", err)
	}
	if reactivated.Status != CloudTenantStatusActive {
		t.Errorf("expected status active, got %s", reactivated.Status)
	}
}

func TestReactivateTenant_NotSuspendedFails(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tenant := setupTenantWithPlan(t, svc, "starter")

	_, err := svc.ReactivateTenant(ctx, tenant.TenantID)
	if err == nil {
		t.Error("expected error when reactivating non-suspended tenant")
	}
}

func TestTierRank_Ordering(t *testing.T) {
	tiers := []PlanTier{PlanTierFree, PlanTierStarter, PlanTierPro, PlanTierEnterprise}
	for i := 0; i < len(tiers)-1; i++ {
		if tierRank(tiers[i]) >= tierRank(tiers[i+1]) {
			t.Errorf("expected tierRank(%s) < tierRank(%s)", tiers[i], tiers[i+1])
		}
	}
}

func TestTierRank_Unknown(t *testing.T) {
	rank := tierRank(PlanTier("unknown"))
	if rank != -1 {
		t.Errorf("expected -1 for unknown tier, got %d", rank)
	}
}

func TestGetSLAStatus_FallbackDefaults(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tenant := setupTenantWithPlan(t, svc, "pro")

	sla, err := svc.GetSLAStatus(ctx, tenant.TenantID)
	if err != nil {
		t.Fatalf("GetSLAStatus failed: %v", err)
	}
	if sla.CurrentUptimePct != 99.95 {
		t.Errorf("expected default uptime 99.95, got %f", sla.CurrentUptimePct)
	}
	if sla.UptimeTargetPct != 99.9 {
		t.Errorf("expected pro uptime target 99.9, got %f", sla.UptimeTargetPct)
	}
}

func TestGetSLAStatus_FromRepo(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tenant := setupTenantWithPlan(t, svc, "pro")

	repo.slaMetrics[tenant.TenantID] = &SLAConfig{
		TenantID:           tenant.TenantID,
		CurrentUptimePct:   99.99,
		CurrentLatencyMs:   20,
		CurrentDeliveryPct: 99.9,
		UptimeTargetPct:    99.9,
		LatencyTargetMs:    500,
		DeliveryTargetPct:  99.5,
	}

	sla, err := svc.GetSLAStatus(ctx, tenant.TenantID)
	if err != nil {
		t.Fatalf("GetSLAStatus failed: %v", err)
	}
	if sla.CurrentUptimePct != 99.99 {
		t.Errorf("expected repo uptime 99.99, got %f", sla.CurrentUptimePct)
	}
}

func TestGetStatusPage_FallbackDefaults(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	page := svc.GetStatusPage(ctx)
	if page.OverallStatus != "operational" {
		t.Errorf("expected operational, got %s", page.OverallStatus)
	}
	if len(page.Components) != 6 {
		t.Errorf("expected 6 default components, got %d", len(page.Components))
	}
}

func TestGetStatusPage_FromRepo(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	repo.components = []StatusPageEntry{
		{Component: "API", Status: "degraded", Description: "Slow responses"},
	}

	page := svc.GetStatusPage(ctx)
	if page.OverallStatus != "degraded" {
		t.Errorf("expected degraded, got %s", page.OverallStatus)
	}
	if len(page.Components) != 1 {
		t.Errorf("expected 1 component from repo, got %d", len(page.Components))
	}
}

func TestGetRegionalDeployments_FallbackDefaults(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	deployments := svc.GetRegionalDeployments(ctx)
	if len(deployments) != 4 {
		t.Errorf("expected 4 default deployments, got %d", len(deployments))
	}
}

func TestGetRegionalDeployments_FromRepo(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	repo.deployments = []RegionalDeployment{
		{Region: "custom-region", Status: "active", TenantCount: 5, InstanceCount: 2, HealthScore: 100.0},
	}

	deployments := svc.GetRegionalDeployments(ctx)
	if len(deployments) != 1 {
		t.Errorf("expected 1 deployment from repo, got %d", len(deployments))
	}
	if deployments[0].Region != "custom-region" {
		t.Errorf("expected custom-region, got %s", deployments[0].Region)
	}
}

func TestHandleStripeWebhook_SubscriptionUpdatedActive(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tenant := setupTenantWithPlan(t, svc, "starter")
	repo.billing[tenant.TenantID] = &BillingInfo{
		TenantID:         tenant.TenantID,
		StripeCustomerID: "cus_test123",
	}

	data, _ := json.Marshal(StripeSubscriptionData{})
	// Build proper JSON with customer ID and status
	data = []byte(`{"object":{"id":"sub_1","customer":"cus_test123","status":"active"}}`)

	event := &StripeWebhookEvent{
		ID:   "evt_1",
		Type: "customer.subscription.updated",
		Data: data,
	}

	if err := svc.HandleStripeWebhook(ctx, event); err != nil {
		t.Fatalf("HandleStripeWebhook failed: %v", err)
	}

	updated, err := repo.GetCloudTenant(ctx, tenant.TenantID)
	if err != nil {
		t.Fatalf("GetCloudTenant failed: %v", err)
	}
	if updated.Status != CloudTenantStatusActive {
		t.Errorf("expected status active, got %s", updated.Status)
	}
}

func TestHandleStripeWebhook_SubscriptionDeleted(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tenant := setupTenantWithPlan(t, svc, "starter")
	repo.billing[tenant.TenantID] = &BillingInfo{
		TenantID:         tenant.TenantID,
		StripeCustomerID: "cus_test456",
	}

	data := []byte(`{"object":{"id":"sub_2","customer":"cus_test456","status":"canceled"}}`)
	event := &StripeWebhookEvent{
		ID:   "evt_2",
		Type: "customer.subscription.deleted",
		Data: data,
	}

	if err := svc.HandleStripeWebhook(ctx, event); err != nil {
		t.Fatalf("HandleStripeWebhook failed: %v", err)
	}

	updated, err := repo.GetCloudTenant(ctx, tenant.TenantID)
	if err != nil {
		t.Fatalf("GetCloudTenant failed: %v", err)
	}
	if updated.Status != CloudTenantStatusCancelled {
		t.Errorf("expected status cancelled, got %s", updated.Status)
	}
}

func TestHandleStripeWebhook_PaymentFailed(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	data := []byte(`{"object":{"id":"inv_1","customer":"cus_test789","amount_due":2900,"status":"open"}}`)
	event := &StripeWebhookEvent{
		ID:   "evt_3",
		Type: "invoice.payment_failed",
		Data: data,
	}

	if err := svc.HandleStripeWebhook(ctx, event); err != nil {
		t.Fatalf("HandleStripeWebhook failed: %v", err)
	}
}

func TestHandleStripeWebhook_UnknownEventType(t *testing.T) {
	svc := NewService(newMockRepository())
	ctx := context.Background()

	event := &StripeWebhookEvent{
		ID:   "evt_4",
		Type: "unknown.event.type",
		Data: []byte(`{}`),
	}

	if err := svc.HandleStripeWebhook(ctx, event); err != nil {
		t.Fatalf("expected no error for unknown event type, got: %v", err)
	}
}

func TestHandleStripeWebhook_InvalidJSON(t *testing.T) {
	svc := NewService(newMockRepository())
	ctx := context.Background()

	event := &StripeWebhookEvent{
		ID:   "evt_5",
		Type: "customer.subscription.updated",
		Data: []byte(`not valid json`),
	}

	if err := svc.HandleStripeWebhook(ctx, event); err == nil {
		t.Error("expected error for invalid JSON data")
	}
}
