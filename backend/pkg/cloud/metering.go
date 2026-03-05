package cloud

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MeteringService provides usage-based billing with real-time metering
type MeteringService struct {
	repo         Repository
	stripeClient StripeClient
	mu           sync.RWMutex
	alerts       map[string][]UsageAlert
}

// UsageAlert defines a threshold-based usage alert
type UsageAlert struct {
	ID            string     `json:"id"`
	TenantID      string     `json:"tenant_id"`
	MetricName    string     `json:"metric_name"`
	ThresholdPct  float64    `json:"threshold_pct"`
	Triggered     bool       `json:"triggered"`
	TriggeredAt   *time.Time `json:"triggered_at,omitempty"`
	NotifyEmail   bool       `json:"notify_email"`
	NotifyWebhook bool       `json:"notify_webhook"`
}

// UsageMeterEvent represents a single metered event for billing
type UsageMeterEvent struct {
	ID          string            `json:"id"`
	TenantID    string            `json:"tenant_id"`
	MetricName  string            `json:"metric_name"`
	Quantity    int64             `json:"quantity"`
	Properties  map[string]string `json:"properties,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
	Idempotency string            `json:"idempotency_key,omitempty"`
}

// BillingPeriodSummary provides a breakdown of usage-based charges
type BillingPeriodSummary struct {
	TenantID     string            `json:"tenant_id"`
	Period       string            `json:"period"`
	PlanID       string            `json:"plan_id"`
	BaseCharge   int64             `json:"base_charge"`
	UsageCharges []UsageChargeItem `json:"usage_charges"`
	TotalCharge  int64             `json:"total_charge"`
	Currency     string            `json:"currency"`
	CalculatedAt time.Time         `json:"calculated_at"`
}

// UsageChargeItem represents one line item of usage-based billing
type UsageChargeItem struct {
	MetricName     string `json:"metric_name"`
	Quantity       int64  `json:"quantity"`
	FreeAllowance  int64  `json:"free_allowance"`
	Overage        int64  `json:"overage"`
	UnitPriceCents int64  `json:"unit_price_cents"`
	TotalCents     int64  `json:"total_cents"`
}

// Metering pricing: per-unit overage costs in cents (fractional)
var OveragePricing = map[string]int64{
	"webhooks_sent":     1,  // $0.01 per 100 overage webhooks
	"webhooks_received": 1,  // $0.01 per 100 overage webhooks
	"api_requests":      1,  // $0.01 per 100 overage API calls
	"storage_bytes":     10, // $0.10 per GB overage
}

// NewMeteringService creates a new metering service
func NewMeteringService(repo Repository, stripeClient StripeClient) *MeteringService {
	return &MeteringService{
		repo:         repo,
		stripeClient: stripeClient,
		alerts:       make(map[string][]UsageAlert),
	}
}

// RecordUsageEvent records a metered usage event
func (m *MeteringService) RecordUsageEvent(ctx context.Context, event *UsageMeterEvent) error {
	if event.TenantID == "" {
		return fmt.Errorf("tenant_id is required")
	}
	if event.MetricName == "" {
		return fmt.Errorf("metric_name is required")
	}
	if event.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if event.ID == "" {
		event.ID = generateID()
	}

	if err := m.repo.IncrementUsage(ctx, event.TenantID, event.MetricName, event.Quantity); err != nil {
		return fmt.Errorf("failed to record usage: %w", err)
	}

	// Check alerts asynchronously
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		m.checkAlerts(ctx, event.TenantID)
	}()

	return nil
}

// CalculatePeriodBilling calculates usage-based billing for a period
func (m *MeteringService) CalculatePeriodBilling(ctx context.Context, tenantID, period string) (*BillingPeriodSummary, error) {
	if period == "" {
		period = time.Now().Format("2006-01")
	}

	sub, err := m.repo.GetSubscriptionByTenant(ctx, tenantID)
	if err != nil {
		// Default to free plan
		return m.calculateFreeTierBilling(ctx, tenantID, period)
	}

	plan := GetPlanByID(sub.PlanID)
	if plan == nil {
		return nil, ErrPlanNotFound
	}

	usage, err := m.repo.GetUsage(ctx, tenantID, period)
	if err != nil {
		usage = &UsageRecord{}
	}

	summary := &BillingPeriodSummary{
		TenantID:     tenantID,
		Period:       period,
		PlanID:       sub.PlanID,
		BaseCharge:   plan.PriceMonthly,
		Currency:     "usd",
		CalculatedAt: time.Now(),
	}

	// Calculate overage for webhooks sent
	if plan.Limits.MonthlyWebhooks > 0 && usage.WebhooksSent > plan.Limits.MonthlyWebhooks {
		overage := usage.WebhooksSent - plan.Limits.MonthlyWebhooks
		unitPrice := OveragePricing["webhooks_sent"]
		total := (overage / 100) * unitPrice
		summary.UsageCharges = append(summary.UsageCharges, UsageChargeItem{
			MetricName:     "webhooks_sent",
			Quantity:       usage.WebhooksSent,
			FreeAllowance:  plan.Limits.MonthlyWebhooks,
			Overage:        overage,
			UnitPriceCents: unitPrice,
			TotalCents:     total,
		})
		summary.TotalCharge += total
	}

	// Calculate overage for API requests
	if usage.APIRequests > 0 {
		freeAPIRequests := plan.Limits.MonthlyWebhooks * 2 // 2x webhook limit
		if freeAPIRequests > 0 && usage.APIRequests > freeAPIRequests {
			overage := usage.APIRequests - freeAPIRequests
			unitPrice := OveragePricing["api_requests"]
			total := (overage / 100) * unitPrice
			summary.UsageCharges = append(summary.UsageCharges, UsageChargeItem{
				MetricName:     "api_requests",
				Quantity:       usage.APIRequests,
				FreeAllowance:  freeAPIRequests,
				Overage:        overage,
				UnitPriceCents: unitPrice,
				TotalCents:     total,
			})
			summary.TotalCharge += total
		}
	}

	summary.TotalCharge += summary.BaseCharge

	return summary, nil
}

// calculateFreeTierBilling handles billing for the free tier (1K events/month)
func (m *MeteringService) calculateFreeTierBilling(ctx context.Context, tenantID, period string) (*BillingPeriodSummary, error) {
	usage, err := m.repo.GetUsage(ctx, tenantID, period)
	if err != nil {
		usage = &UsageRecord{}
	}

	summary := &BillingPeriodSummary{
		TenantID:     tenantID,
		Period:       period,
		PlanID:       "free",
		BaseCharge:   0,
		Currency:     "usd",
		CalculatedAt: time.Now(),
	}

	// Free tier: 1,000 events/month
	const freeLimit int64 = 1000
	if usage.WebhooksSent > freeLimit {
		overage := usage.WebhooksSent - freeLimit
		unitPrice := OveragePricing["webhooks_sent"]
		total := (overage / 100) * unitPrice
		summary.UsageCharges = append(summary.UsageCharges, UsageChargeItem{
			MetricName:     "webhooks_sent",
			Quantity:       usage.WebhooksSent,
			FreeAllowance:  freeLimit,
			Overage:        overage,
			UnitPriceCents: unitPrice,
			TotalCents:     total,
		})
		summary.TotalCharge += total
	}

	return summary, nil
}

// SetAlert configures a usage alert for a tenant
func (m *MeteringService) SetAlert(ctx context.Context, alert *UsageAlert) error {
	if alert.ThresholdPct <= 0 || alert.ThresholdPct > 100 {
		return fmt.Errorf("threshold_pct must be between 0 and 100")
	}
	if alert.ID == "" {
		alert.ID = generateID()
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.alerts[alert.TenantID] = append(m.alerts[alert.TenantID], *alert)
	return nil
}

// GetAlerts returns configured alerts for a tenant
func (m *MeteringService) GetAlerts(ctx context.Context, tenantID string) []UsageAlert {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.alerts[tenantID]
}

func (m *MeteringService) checkAlerts(ctx context.Context, tenantID string) {
	m.mu.RLock()
	alerts := m.alerts[tenantID]
	m.mu.RUnlock()

	if len(alerts) == 0 {
		return
	}

	sub, err := m.repo.GetSubscriptionByTenant(ctx, tenantID)
	if err != nil {
		return
	}

	plan := GetPlanByID(sub.PlanID)
	if plan == nil || plan.Limits.MonthlyWebhooks <= 0 {
		return
	}

	usage, err := m.repo.GetUsage(ctx, tenantID, time.Now().Format("2006-01"))
	if err != nil {
		return
	}

	currentPct := float64(usage.WebhooksSent) / float64(plan.Limits.MonthlyWebhooks) * 100

	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.alerts[tenantID] {
		if !m.alerts[tenantID][i].Triggered && currentPct >= m.alerts[tenantID][i].ThresholdPct {
			now := time.Now()
			m.alerts[tenantID][i].Triggered = true
			m.alerts[tenantID][i].TriggeredAt = &now
		}
	}
}
