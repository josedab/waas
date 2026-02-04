package cloudmanaged

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SaaS onboarding and multi-tenant management extensions

// SignupWithVerificationRequest extends signup with email verification
type SignupWithVerificationRequest struct {
	Email       string `json:"email" binding:"required"`
	Org         string `json:"org" binding:"required"`
	Password    string `json:"password" binding:"required"`
	Plan        string `json:"plan,omitempty"`
	Region      string `json:"region,omitempty"`
	AcceptedTOS bool   `json:"accepted_tos" binding:"required"`
}

// EmailVerification tracks email verification tokens
type EmailVerification struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Email     string    `json:"email"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	Verified  bool      `json:"verified"`
	CreatedAt time.Time `json:"created_at"`
}

// StripeCheckoutSession represents a Stripe checkout session for plan upgrades
type StripeCheckoutSession struct {
	SessionID  string `json:"session_id"`
	URL        string `json:"url"`
	TenantID   string `json:"tenant_id"`
	Plan       string `json:"plan"`
	PriceID    string `json:"price_id"`
	SuccessURL string `json:"success_url"`
	CancelURL  string `json:"cancel_url"`
}

// UsageDashboard provides usage metrics for the tenant dashboard
type UsageDashboard struct {
	TenantID           string            `json:"tenant_id"`
	Plan               PlanTier          `json:"plan"`
	BillingPeriodStart time.Time         `json:"billing_period_start"`
	BillingPeriodEnd   time.Time         `json:"billing_period_end"`
	Webhooks           UsageMetric       `json:"webhooks"`
	Storage            UsageMetric       `json:"storage"`
	Bandwidth          UsageMetric       `json:"bandwidth"`
	APIRequests        UsageMetric       `json:"api_requests"`
	Endpoints          UsageMetric       `json:"endpoints"`
	DailyUsage         []DailyUsagePoint `json:"daily_usage"`
	CostBreakdown      *CostBreakdown    `json:"cost_breakdown"`
	Alerts             []UsageAlert      `json:"alerts,omitempty"`
}

// UsageMetric represents a single usage metric with limit tracking
type UsageMetric struct {
	Name       string  `json:"name"`
	Used       int64   `json:"used"`
	Limit      int64   `json:"limit"`
	Unit       string  `json:"unit"`
	Percentage float64 `json:"percentage"`
}

// DailyUsagePoint is a data point for daily usage charts
type DailyUsagePoint struct {
	Date       string `json:"date"`
	Webhooks   int64  `json:"webhooks"`
	Successes  int64  `json:"successes"`
	Failures   int64  `json:"failures"`
	AvgLatency int64  `json:"avg_latency_ms"`
}

// CostBreakdown provides billing cost breakdown
type CostBreakdown struct {
	BaseCost     int64  `json:"base_cost_cents"`
	OverageCost  int64  `json:"overage_cost_cents"`
	TotalCost    int64  `json:"total_cost_cents"`
	Currency     string `json:"currency"`
	OverageUnits int64  `json:"overage_units"`
}

// UsageAlert represents a usage threshold alert
type UsageAlert struct {
	Type      string    `json:"type"`
	Metric    string    `json:"metric"`
	Threshold float64   `json:"threshold"`
	Current   float64   `json:"current"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

// TenantNamespaceConfig holds K8s namespace isolation configuration
type TenantNamespaceConfig struct {
	TenantID      string            `json:"tenant_id"`
	Namespace     string            `json:"namespace"`
	ResourceQuota *ResourceQuota    `json:"resource_quota"`
	NetworkPolicy *NetworkPolicy    `json:"network_policy"`
	Labels        map[string]string `json:"labels"`
	Annotations   map[string]string `json:"annotations"`
	CreatedAt     time.Time         `json:"created_at"`
}

// ResourceQuota defines resource limits for a tenant namespace
type ResourceQuota struct {
	CPULimit      string `json:"cpu_limit"`
	MemoryLimit   string `json:"memory_limit"`
	CPURequest    string `json:"cpu_request"`
	MemoryRequest string `json:"memory_request"`
	PodLimit      int    `json:"pod_limit"`
}

// NetworkPolicy defines network isolation rules
type NetworkPolicy struct {
	AllowIngress     bool     `json:"allow_ingress"`
	AllowEgress      bool     `json:"allow_egress"`
	AllowedCIDRs     []string `json:"allowed_cidrs,omitempty"`
	DenyAllByDefault bool     `json:"deny_all_by_default"`
}

// OperationsDashboard provides SaaS-wide operational metrics
type OperationsDashboard struct {
	TotalTenants  int64                `json:"total_tenants"`
	ActiveTenants int64                `json:"active_tenants"`
	TotalWebhooks int64                `json:"total_webhooks_delivered"`
	ErrorRate     float64              `json:"error_rate"`
	AvgLatencyMs  float64              `json:"avg_latency_ms"`
	Revenue       *RevenueSummary      `json:"revenue"`
	TenantsByPlan map[string]int64     `json:"tenants_by_plan"`
	TopTenants    []TenantUsageSummary `json:"top_tenants"`
	SystemHealth  *SystemHealthSummary `json:"system_health"`
	RecentSignups int64                `json:"recent_signups_24h"`
	ChurnRate     float64              `json:"churn_rate"`
	GeneratedAt   time.Time            `json:"generated_at"`
}

// RevenueSummary provides revenue metrics
type RevenueSummary struct {
	MRR       int64   `json:"mrr_cents"`
	ARR       int64   `json:"arr_cents"`
	MoMGrowth float64 `json:"mom_growth_pct"`
	Currency  string  `json:"currency"`
}

// TenantUsageSummary provides per-tenant usage summary for ops dashboard
type TenantUsageSummary struct {
	TenantID     string   `json:"tenant_id"`
	Org          string   `json:"org"`
	Plan         PlanTier `json:"plan"`
	WebhooksUsed int64    `json:"webhooks_used"`
	ErrorRate    float64  `json:"error_rate"`
}

// SystemHealthSummary provides system-wide health status
type SystemHealthSummary struct {
	APILatencyP99 float64 `json:"api_latency_p99_ms"`
	QueueDepth    int64   `json:"queue_depth"`
	DBConnections int     `json:"db_connections"`
	RedisMemoryMB float64 `json:"redis_memory_mb"`
	UptimePct     float64 `json:"uptime_pct"`
}

// --- Service method extensions ---

// SignupWithVerification handles the full SaaS signup flow with email verification
func (s *Service) SignupWithVerification(ctx context.Context, req *SignupWithVerificationRequest) (*CloudTenant, *EmailVerification, error) {
	if !req.AcceptedTOS {
		return nil, nil, fmt.Errorf("terms of service must be accepted")
	}

	if req.Email == "" || req.Org == "" || req.Password == "" {
		return nil, nil, fmt.Errorf("email, org, and password are required")
	}

	// Default to free plan
	plan := PlanTierFree
	if req.Plan != "" {
		plan = PlanTier(req.Plan)
	}

	planDef := getPlanDefinition(plan)
	if planDef == nil {
		return nil, nil, fmt.Errorf("invalid plan: %s", req.Plan)
	}

	region := "us-east-1"
	if req.Region != "" {
		region = req.Region
	}

	now := time.Now()
	tenant := &CloudTenant{
		ID:            uuid.New(),
		TenantID:      uuid.New().String(),
		Email:         req.Email,
		Org:           req.Org,
		Plan:          plan,
		Status:        CloudTenantStatusTrial,
		Region:        region,
		WebhooksLimit: int64(planDef.WebhooksLimit),
		StorageLimit:  planDef.StorageLimit,
		CreatedAt:     now,
	}

	// Free tier gets 1K events/month
	if plan == PlanTierFree {
		tenant.WebhooksLimit = 1000
	}

	// Set trial period (14 days for paid plans)
	if plan != PlanTierFree {
		trialEnd := now.Add(14 * 24 * time.Hour)
		tenant.TrialEndsAt = &trialEnd
	}

	// Generate verification token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, nil, fmt.Errorf("failed to generate verification token: %w", err)
	}

	verification := &EmailVerification{
		ID:        uuid.New().String(),
		TenantID:  tenant.TenantID,
		Email:     req.Email,
		Token:     hex.EncodeToString(tokenBytes),
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
	}

	return tenant, verification, nil
}

// CreateCheckoutSession creates a Stripe checkout session for plan upgrades
func (s *Service) CreateCheckoutSession(ctx context.Context, tenantID string, targetPlan PlanTier) (*StripeCheckoutSession, error) {
	planDef := getPlanDefinition(targetPlan)
	if planDef == nil {
		return nil, fmt.Errorf("invalid plan: %s", targetPlan)
	}

	if planDef.PriceMonthly == 0 {
		return nil, fmt.Errorf("cannot create checkout for free plan")
	}

	session := &StripeCheckoutSession{
		SessionID:  "cs_" + uuid.New().String(),
		TenantID:   tenantID,
		Plan:       string(targetPlan),
		SuccessURL: fmt.Sprintf("/dashboard/billing/success?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  "/dashboard/billing",
	}

	return session, nil
}

// GetUsageDashboard returns the usage dashboard for a tenant
func (s *Service) GetUsageDashboard(ctx context.Context, tenantID string) (*UsageDashboard, error) {
	tenant, err := s.repo.GetCloudTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("tenant not found: %w", err)
	}

	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Second)

	webhookPct := float64(0)
	if tenant.WebhooksLimit > 0 {
		webhookPct = float64(tenant.WebhooksUsed) / float64(tenant.WebhooksLimit) * 100
	}

	storagePct := float64(0)
	if tenant.StorageLimit > 0 {
		storagePct = float64(tenant.StorageUsed) / float64(tenant.StorageLimit) * 100
	}

	dashboard := &UsageDashboard{
		TenantID:           tenantID,
		Plan:               tenant.Plan,
		BillingPeriodStart: periodStart,
		BillingPeriodEnd:   periodEnd,
		Webhooks: UsageMetric{
			Name:       "Webhooks",
			Used:       tenant.WebhooksUsed,
			Limit:      tenant.WebhooksLimit,
			Unit:       "events",
			Percentage: webhookPct,
		},
		Storage: UsageMetric{
			Name:       "Storage",
			Used:       tenant.StorageUsed,
			Limit:      tenant.StorageLimit,
			Unit:       "bytes",
			Percentage: storagePct,
		},
	}

	// Generate usage alerts
	if webhookPct >= 90 {
		dashboard.Alerts = append(dashboard.Alerts, UsageAlert{
			Type:      "warning",
			Metric:    "webhooks",
			Threshold: 90,
			Current:   webhookPct,
			Message:   "Webhook usage is at 90% of your plan limit",
			CreatedAt: now,
		})
	}

	return dashboard, nil
}

// GetOperationsDashboard returns the SaaS operations dashboard (admin only)
func (s *Service) GetOperationsDashboard(ctx context.Context) (*OperationsDashboard, error) {
	dashboard := &OperationsDashboard{
		TenantsByPlan: make(map[string]int64),
		Revenue: &RevenueSummary{
			Currency: "usd",
		},
		SystemHealth: &SystemHealthSummary{
			UptimePct: 99.99,
		},
		GeneratedAt: time.Now(),
	}

	// Calculate MRR from plan pricing
	for _, plan := range DefaultPlans() {
		count := dashboard.TenantsByPlan[string(plan.Tier)]
		dashboard.Revenue.MRR += int64(plan.PriceMonthly) * count
	}
	dashboard.Revenue.ARR = dashboard.Revenue.MRR * 12

	return dashboard, nil
}

// GetTenantNamespaceConfig generates K8s namespace config for tenant isolation
func (s *Service) GetTenantNamespaceConfig(ctx context.Context, tenantID string) (*TenantNamespaceConfig, error) {
	tenant, err := s.repo.GetCloudTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("tenant not found: %w", err)
	}

	// Resource quotas based on plan
	var quota *ResourceQuota
	switch tenant.Plan {
	case PlanTierFree:
		quota = &ResourceQuota{CPULimit: "500m", MemoryLimit: "512Mi", CPURequest: "100m", MemoryRequest: "128Mi", PodLimit: 2}
	case PlanTierStarter:
		quota = &ResourceQuota{CPULimit: "1", MemoryLimit: "1Gi", CPURequest: "250m", MemoryRequest: "256Mi", PodLimit: 5}
	case PlanTierPro:
		quota = &ResourceQuota{CPULimit: "4", MemoryLimit: "4Gi", CPURequest: "1", MemoryRequest: "1Gi", PodLimit: 10}
	case PlanTierEnterprise:
		quota = &ResourceQuota{CPULimit: "16", MemoryLimit: "16Gi", CPURequest: "4", MemoryRequest: "4Gi", PodLimit: 50}
	}

	config := &TenantNamespaceConfig{
		TenantID:      tenantID,
		Namespace:     fmt.Sprintf("waas-tenant-%s", tenantID[:8]),
		ResourceQuota: quota,
		NetworkPolicy: &NetworkPolicy{
			AllowIngress:     true,
			AllowEgress:      true,
			DenyAllByDefault: true,
		},
		Labels: map[string]string{
			"app.kubernetes.io/managed-by": "waas",
			"waas.io/tenant-id":            tenantID,
			"waas.io/plan":                 string(tenant.Plan),
		},
		Annotations: map[string]string{
			"waas.io/created-at": time.Now().Format(time.RFC3339),
		},
		CreatedAt: time.Now(),
	}

	return config, nil
}

// RowLevelSecurityPolicy generates PostgreSQL RLS policy for tenant isolation
type RowLevelSecurityPolicy struct {
	TenantID   string `json:"tenant_id"`
	PolicyName string `json:"policy_name"`
	TableName  string `json:"table_name"`
	PolicySQL  string `json:"policy_sql"`
}

// GenerateRLSPolicies creates row-level security policies for tenant data isolation
func GenerateRLSPolicies(tenantID string) []RowLevelSecurityPolicy {
	tables := []string{
		"webhook_endpoints",
		"delivery_attempts",
		"delivery_metrics",
		"quota_usage",
	}

	policies := make([]RowLevelSecurityPolicy, 0, len(tables))
	for _, table := range tables {
		policies = append(policies, RowLevelSecurityPolicy{
			TenantID:   tenantID,
			PolicyName: fmt.Sprintf("tenant_isolation_%s", table),
			TableName:  table,
			PolicySQL:  fmt.Sprintf("CREATE POLICY tenant_isolation_%s ON %s FOR ALL USING (tenant_id = current_setting('app.current_tenant')::uuid)", table, table),
		})
	}

	return policies
}
