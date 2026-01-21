package billing

import (
	"fmt"
	"time"
)

// UsageRecord represents webhook usage
type UsageRecord struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	WebhookID     string    `json:"webhook_id,omitempty"`
	ResourceType  string    `json:"resource_type"`
	Quantity      int64     `json:"quantity"`
	UnitCost      float64   `json:"unit_cost"`
	TotalCost     float64   `json:"total_cost"`
	Currency      string    `json:"currency"`
	BillingPeriod string    `json:"billing_period"` // YYYY-MM
	RecordedAt    time.Time `json:"recorded_at"`
}

// SpendTracker tracks real-time spending
type SpendTracker struct {
	ID           string             `json:"id"`
	TenantID     string             `json:"tenant_id"`
	BudgetLimit  float64            `json:"budget_limit"`
	CurrentSpend float64            `json:"current_spend"`
	Currency     string             `json:"currency"`
	Period       BillingPeriod      `json:"period"`
	PeriodStart  time.Time          `json:"period_start"`
	PeriodEnd    time.Time          `json:"period_end"`
	Breakdown    map[string]float64 `json:"breakdown"` // By resource type
	Alerts       []AlertThreshold   `json:"alerts"`
	Status       SpendStatus        `json:"status"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

// BillingPeriod defines billing cycle
type BillingPeriod string

const (
	PeriodDaily   BillingPeriod = "daily"
	PeriodWeekly  BillingPeriod = "weekly"
	PeriodMonthly BillingPeriod = "monthly"
)

// SpendStatus defines spending status
type SpendStatus string

const (
	SpendNormal   SpendStatus = "normal"
	SpendWarning  SpendStatus = "warning"
	SpendCritical SpendStatus = "critical"
	SpendExceeded SpendStatus = "exceeded"
)

// AlertThreshold defines budget alert thresholds
type AlertThreshold struct {
	Percentage float64       `json:"percentage"` // 50, 80, 90, 100
	Triggered  bool          `json:"triggered"`
	TriggeredAt *time.Time   `json:"triggered_at,omitempty"`
	Channels   []AlertChannel `json:"channels"`
}

// Validate validates the alert threshold
func (at *AlertThreshold) Validate() error {
	if at.Percentage <= 0 || at.Percentage > 100 {
		return fmt.Errorf("percentage must be between 1 and 100")
	}
	if len(at.Channels) == 0 {
		return fmt.Errorf("at least one channel is required")
	}
	return nil
}

// AlertChannel defines notification channels
type AlertChannel string

const (
	ChannelEmail   AlertChannel = "email"
	ChannelSlack   AlertChannel = "slack"
	ChannelWebhook AlertChannel = "webhook"
	ChannelSMS     AlertChannel = "sms"
	ChannelPush    AlertChannel = "push"
)

// BillingAlert represents a triggered alert
type BillingAlert struct {
	ID          string        `json:"id"`
	TenantID    string        `json:"tenant_id"`
	Type        AlertType     `json:"type"`
	Severity    AlertSeverity `json:"severity"`
	Title       string        `json:"title"`
	Message     string        `json:"message"`
	Data        AlertData     `json:"data"`
	Status      AlertStatus   `json:"status"`
	Channels    []AlertChannel `json:"channels"`
	SentAt      *time.Time    `json:"sent_at,omitempty"`
	AckedAt     *time.Time    `json:"acked_at,omitempty"`
	AckedBy     string        `json:"acked_by,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
}

// AlertType defines alert types
type AlertType string

const (
	AlertBudgetWarning    AlertType = "budget_warning"
	AlertBudgetCritical   AlertType = "budget_critical"
	AlertBudgetExceeded   AlertType = "budget_exceeded"
	AlertSpendAnomaly     AlertType = "spend_anomaly"
	AlertUsageSpike       AlertType = "usage_spike"
	AlertCostOptimization AlertType = "cost_optimization"
)

// AlertSeverity defines alert severity
type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "info"
	SeverityWarning  AlertSeverity = "warning"
	SeverityCritical AlertSeverity = "critical"
)

// AlertStatus defines alert status
type AlertStatus string

const (
	AlertPending AlertStatus = "pending"
	AlertSent    AlertStatus = "sent"
	AlertAcked   AlertStatus = "acknowledged"
	AlertResolved AlertStatus = "resolved"
)

// AlertData holds alert-specific data
type AlertData struct {
	BudgetLimit    float64 `json:"budget_limit,omitempty"`
	CurrentSpend   float64 `json:"current_spend,omitempty"`
	Percentage     float64 `json:"percentage,omitempty"`
	ExpectedSpend  float64 `json:"expected_spend,omitempty"`
	AnomalyScore   float64 `json:"anomaly_score,omitempty"`
	ResourceType   string  `json:"resource_type,omitempty"`
	Recommendation string  `json:"recommendation,omitempty"`
}

// BudgetConfig defines budget configuration
type BudgetConfig struct {
	ID           string          `json:"id"`
	TenantID     string          `json:"tenant_id"`
	Name         string          `json:"name"`
	Amount       float64         `json:"amount"`
	Currency     string          `json:"currency"`
	Period       BillingPeriod   `json:"period"`
	ResourceType string          `json:"resource_type,omitempty"` // Empty = all
	WebhookID    string          `json:"webhook_id,omitempty"`
	Alerts       []AlertThreshold `json:"alerts"`
	AutoPause    bool            `json:"auto_pause"` // Pause webhooks on exceed
	Enabled      bool            `json:"enabled"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// CostOptimization represents a cost saving recommendation
type CostOptimization struct {
	ID            string            `json:"id"`
	TenantID      string            `json:"tenant_id"`
	Type          OptimizationType  `json:"type"`
	Title         string            `json:"title"`
	Description   string            `json:"description"`
	EstimatedSavings float64        `json:"estimated_savings"`
	Currency      string            `json:"currency"`
	Impact        OptimizationImpact `json:"impact"`
	ResourceID    string            `json:"resource_id,omitempty"`
	ResourceType  string            `json:"resource_type,omitempty"`
	Actions       []OptimizationAction `json:"actions"`
	Status        OptimizationStatus `json:"status"`
	ImplementedAt *time.Time        `json:"implemented_at,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
}

// OptimizationType defines optimization types
type OptimizationType string

const (
	OptUnusedWebhooks    OptimizationType = "unused_webhooks"
	OptRetryReduction    OptimizationType = "retry_reduction"
	OptPayloadCompression OptimizationType = "payload_compression"
	OptBatchDelivery     OptimizationType = "batch_delivery"
	OptRegionOptimization OptimizationType = "region_optimization"
	OptTierDowngrade     OptimizationType = "tier_downgrade"
)

// OptimizationImpact defines impact level
type OptimizationImpact string

const (
	ImpactLow    OptimizationImpact = "low"
	ImpactMedium OptimizationImpact = "medium"
	ImpactHigh   OptimizationImpact = "high"
)

// OptimizationAction defines actionable steps
type OptimizationAction struct {
	Type        string `json:"type"`
	Label       string `json:"label"`
	Action      string `json:"action"`
	Description string `json:"description"`
	AutoApply   bool   `json:"auto_apply"`
}

// OptimizationStatus defines status
type OptimizationStatus string

const (
	OptStatusPending    OptimizationStatus = "pending"
	OptStatusDismissed  OptimizationStatus = "dismissed"
	OptStatusImplemented OptimizationStatus = "implemented"
)

// Invoice represents a billing invoice
type Invoice struct {
	ID            string        `json:"id"`
	TenantID      string        `json:"tenant_id"`
	Number        string        `json:"number"`
	Status        InvoiceStatus `json:"status"`
	Period        string        `json:"period"` // YYYY-MM
	Subtotal      float64       `json:"subtotal"`
	Discount      float64       `json:"discount"`
	Tax           float64       `json:"tax"`
	Total         float64       `json:"total"`
	Currency      string        `json:"currency"`
	LineItems     []LineItem    `json:"line_items"`
	DueDate       time.Time     `json:"due_date"`
	PaidAt        *time.Time    `json:"paid_at,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
}

// InvoiceStatus defines invoice status
type InvoiceStatus string

const (
	InvoiceDraft   InvoiceStatus = "draft"
	InvoicePending InvoiceStatus = "pending"
	InvoicePaid    InvoiceStatus = "paid"
	InvoiceOverdue InvoiceStatus = "overdue"
	InvoiceVoid    InvoiceStatus = "void"
)

// LineItem represents invoice line item
type LineItem struct {
	Description  string  `json:"description"`
	ResourceType string  `json:"resource_type"`
	Quantity     int64   `json:"quantity"`
	UnitPrice    float64 `json:"unit_price"`
	Amount       float64 `json:"amount"`
}

// PricingTier defines pricing tiers
type PricingTier struct {
	ID           string                  `json:"id"`
	Name         string                  `json:"name"`
	Description  string                  `json:"description"`
	BasePrice    float64                 `json:"base_price"`
	Currency     string                  `json:"currency"`
	Limits       map[string]int64        `json:"limits"`
	UnitPricing  map[string]float64      `json:"unit_pricing"`
	Features     []string                `json:"features"`
}

// UsageSummary summarizes usage
type UsageSummary struct {
	TenantID      string             `json:"tenant_id"`
	Period        string             `json:"period"`
	TotalRequests int64              `json:"total_requests"`
	TotalDelivered int64             `json:"total_delivered"`
	TotalFailed   int64              `json:"total_failed"`
	TotalBytes    int64              `json:"total_bytes"`
	TotalCost     float64            `json:"total_cost"`
	Currency      string             `json:"currency"`
	ByResource    map[string]ResourceUsage `json:"by_resource"`
	ByDay         []DailyUsage       `json:"by_day"`
}

// ResourceUsage holds per-resource usage
type ResourceUsage struct {
	ResourceType string  `json:"resource_type"`
	Quantity     int64   `json:"quantity"`
	Cost         float64 `json:"cost"`
}

// DailyUsage holds daily usage
type DailyUsage struct {
	Date     string  `json:"date"`
	Requests int64   `json:"requests"`
	Cost     float64 `json:"cost"`
}

// SpendForecast predicts future spending
type SpendForecast struct {
	TenantID          string    `json:"tenant_id"`
	Period            string    `json:"period"`
	CurrentSpend      float64   `json:"current_spend"`
	ProjectedSpend    float64   `json:"projected_spend"`
	Confidence        float64   `json:"confidence"` // 0-1
	DailyAverage      float64   `json:"daily_average"`
	DaysRemaining     int       `json:"days_remaining"`
	Trend             string    `json:"trend"` // up, down, stable
	TrendDirection    string    `json:"trend_direction"`
	TrendPercent      float64   `json:"trend_percent"`
	BudgetRemaining   float64   `json:"budget_remaining"`
	BudgetUtilization float64   `json:"budget_utilization"`
	Currency          string    `json:"currency"`
	GeneratedAt       time.Time `json:"generated_at"`
}

// CreateBudgetRequest represents budget creation
type CreateBudgetRequest struct {
	Name         string          `json:"name" binding:"required"`
	Amount       float64         `json:"amount" binding:"required"`
	Currency     string          `json:"currency,omitempty"`
	Period       BillingPeriod   `json:"period" binding:"required"`
	ResourceType string          `json:"resource_type,omitempty"`
	WebhookID    string          `json:"webhook_id,omitempty"`
	Alerts       []AlertThreshold `json:"alerts,omitempty"`
	AutoPause    bool            `json:"auto_pause,omitempty"`
}

// AlertConfig defines alert configuration
type AlertConfig struct {
	ID         string          `json:"id"`
	TenantID   string          `json:"tenant_id"`
	Enabled    bool            `json:"enabled"`
	Channels   []AlertChannel  `json:"channels"`
	Recipients AlertRecipients `json:"recipients"`
	Schedule   AlertSchedule   `json:"schedule"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// AlertRecipients defines who receives alerts
type AlertRecipients struct {
	Emails     []string `json:"emails,omitempty"`
	SlackURLs  []string `json:"slack_urls,omitempty"`
	WebhookURLs []string `json:"webhook_urls,omitempty"`
	PhoneNumbers []string `json:"phone_numbers,omitempty"`
}

// AlertSchedule defines when alerts are sent
type AlertSchedule struct {
	Timezone    string `json:"timezone"`
	QuietStart  string `json:"quiet_start,omitempty"` // HH:MM
	QuietEnd    string `json:"quiet_end,omitempty"`
	MinInterval int    `json:"min_interval_minutes"` // Min time between alerts
}

// GetDefaultAlertThresholds returns default thresholds
func GetDefaultAlertThresholds() []AlertThreshold {
	return []AlertThreshold{
		{Percentage: 50, Channels: []AlertChannel{ChannelEmail}},
		{Percentage: 80, Channels: []AlertChannel{ChannelEmail, ChannelSlack}},
		{Percentage: 90, Channels: []AlertChannel{ChannelEmail, ChannelSlack}},
		{Percentage: 100, Channels: []AlertChannel{ChannelEmail, ChannelSlack, ChannelSMS}},
	}
}

// GetResourceTypes returns billable resource types
func GetResourceTypes() []string {
	return []string{
		"webhook_requests",
		"webhook_deliveries",
		"retry_attempts",
		"data_transfer_bytes",
		"storage_bytes",
		"api_calls",
		"workflow_executions",
	}
}
