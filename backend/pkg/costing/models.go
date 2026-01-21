package costing

import (
	"time"
)

// CostUnit represents the unit of cost
type CostUnit string

const (
	UnitDelivery  CostUnit = "delivery"
	UnitByte      CostUnit = "byte"
	UnitRequest   CostUnit = "request"
	UnitTransform CostUnit = "transform"
	UnitRetry     CostUnit = "retry"
)

// PricingTier represents a pricing tier
type PricingTier struct {
	ID          string     `json:"id" db:"id"`
	Name        string     `json:"name" db:"name"`
	Description string     `json:"description,omitempty" db:"description"`
	Rates       []Rate     `json:"rates" db:"rates"`
	Limits      TierLimits `json:"limits" db:"limits"`
	IsDefault   bool       `json:"is_default" db:"is_default"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// Rate represents a pricing rate
type Rate struct {
	Unit     CostUnit `json:"unit"`
	Price    float64  `json:"price"`     // Price per unit
	Currency string   `json:"currency"`  // USD, EUR, etc.
	MinUnits int64    `json:"min_units"` // Minimum for this rate
	MaxUnits int64    `json:"max_units"` // Maximum for this rate (0 = unlimited)
}

// TierLimits defines tier-specific limits
type TierLimits struct {
	MonthlyDeliveries int64 `json:"monthly_deliveries"`
	MonthlyBytes      int64 `json:"monthly_bytes"`
	MaxEndpoints      int   `json:"max_endpoints"`
	MaxRetries        int   `json:"max_retries"`
}

// UsageRecord represents a usage record
type UsageRecord struct {
	ID         string    `json:"id" db:"id"`
	TenantID   string    `json:"tenant_id" db:"tenant_id"`
	EndpointID string    `json:"endpoint_id,omitempty" db:"endpoint_id"`
	WebhookID  string    `json:"webhook_id,omitempty" db:"webhook_id"`
	Unit       CostUnit  `json:"unit" db:"unit"`
	Quantity   int64     `json:"quantity" db:"quantity"`
	Metadata   UsageMeta `json:"metadata,omitempty" db:"metadata"`
	RecordedAt time.Time `json:"recorded_at" db:"recorded_at"`
}

// UsageMeta contains additional usage metadata
type UsageMeta struct {
	PayloadBytes int64  `json:"payload_bytes,omitempty"`
	RetryCount   int    `json:"retry_count,omitempty"`
	Region       string `json:"region,omitempty"`
	StatusCode   int    `json:"status_code,omitempty"`
	LatencyMs    int    `json:"latency_ms,omitempty"`
}

// CostAllocation represents cost attributed to a resource
type CostAllocation struct {
	ID           string        `json:"id" db:"id"`
	TenantID     string        `json:"tenant_id" db:"tenant_id"`
	Period       string        `json:"period" db:"period"`               // e.g., "2024-01"
	ResourceType string        `json:"resource_type" db:"resource_type"` // endpoint, customer, etc.
	ResourceID   string        `json:"resource_id" db:"resource_id"`
	ResourceName string        `json:"resource_name,omitempty" db:"resource_name"`
	Usage        UsageSummary  `json:"usage" db:"usage"`
	Cost         CostBreakdown `json:"cost" db:"cost"`
	CreatedAt    time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at" db:"updated_at"`
}

// UsageSummary summarizes usage
type UsageSummary struct {
	Deliveries      int64 `json:"deliveries"`
	Bytes           int64 `json:"bytes"`
	Retries         int64 `json:"retries"`
	Transformations int64 `json:"transformations"`
	Successful      int64 `json:"successful"`
	Failed          int64 `json:"failed"`
}

// CostBreakdown shows itemized costs
type CostBreakdown struct {
	Delivery        float64 `json:"delivery"`
	Bandwidth       float64 `json:"bandwidth"`
	Retries         float64 `json:"retries"`
	Transformations float64 `json:"transformations"`
	Total           float64 `json:"total"`
	Currency        string  `json:"currency"`
}

// Budget represents a spending budget
type Budget struct {
	ID           string        `json:"id" db:"id"`
	TenantID     string        `json:"tenant_id" db:"tenant_id"`
	Name         string        `json:"name" db:"name"`
	Amount       float64       `json:"amount" db:"amount"`
	Currency     string        `json:"currency" db:"currency"`
	Period       string        `json:"period" db:"period"` // monthly, quarterly, yearly
	ResourceType string        `json:"resource_type,omitempty" db:"resource_type"`
	ResourceID   string        `json:"resource_id,omitempty" db:"resource_id"`
	Alerts       []BudgetAlert `json:"alerts" db:"alerts"`
	CurrentSpend float64       `json:"current_spend" db:"current_spend"`
	IsActive     bool          `json:"is_active" db:"is_active"`
	StartDate    time.Time     `json:"start_date" db:"start_date"`
	EndDate      *time.Time    `json:"end_date,omitempty" db:"end_date"`
	CreatedAt    time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at" db:"updated_at"`
}

// BudgetAlert defines when to alert on budget
type BudgetAlert struct {
	Threshold   float64    `json:"threshold"` // Percentage (0.5 = 50%, 0.8 = 80%)
	Channels    []string   `json:"channels"`  // email, webhook, slack
	LastAlerted *time.Time `json:"last_alerted,omitempty"`
}

// CostForecast represents a cost forecast
type CostForecast struct {
	TenantID       string       `json:"tenant_id"`
	Period         string       `json:"period"`
	ProjectedCost  float64      `json:"projected_cost"`
	ProjectedUsage UsageSummary `json:"projected_usage"`
	Confidence     float64      `json:"confidence"` // 0-1
	Trend          string       `json:"trend"`      // increasing, decreasing, stable
	PreviousPeriod float64      `json:"previous_period"`
	PercentChange  float64      `json:"percent_change"`
	GeneratedAt    time.Time    `json:"generated_at"`
}

// CostReport represents a cost report
type CostReport struct {
	TenantID    string             `json:"tenant_id"`
	Period      string             `json:"period"`
	StartDate   time.Time          `json:"start_date"`
	EndDate     time.Time          `json:"end_date"`
	Summary     CostBreakdown      `json:"summary"`
	ByEndpoint  []CostAllocation   `json:"by_endpoint"`
	ByCustomer  []CostAllocation   `json:"by_customer,omitempty"`
	ByRegion    map[string]float64 `json:"by_region,omitempty"`
	DailyTrend  []DailyCost        `json:"daily_trend"`
	Forecast    *CostForecast      `json:"forecast,omitempty"`
	GeneratedAt time.Time          `json:"generated_at"`
}

// DailyCost represents daily cost data
type DailyCost struct {
	Date  string       `json:"date"`
	Cost  float64      `json:"cost"`
	Usage UsageSummary `json:"usage"`
}

// CreateBudgetRequest represents a request to create a budget
type CreateBudgetRequest struct {
	Name         string        `json:"name" binding:"required"`
	Amount       float64       `json:"amount" binding:"required,gt=0"`
	Currency     string        `json:"currency" binding:"required"`
	Period       string        `json:"period" binding:"required,oneof=monthly quarterly yearly"`
	ResourceType string        `json:"resource_type,omitempty"`
	ResourceID   string        `json:"resource_id,omitempty"`
	Alerts       []BudgetAlert `json:"alerts"`
	StartDate    time.Time     `json:"start_date"`
}

// UpdateBudgetRequest represents a request to update a budget
type UpdateBudgetRequest struct {
	Name     *string       `json:"name,omitempty"`
	Amount   *float64      `json:"amount,omitempty"`
	Alerts   []BudgetAlert `json:"alerts,omitempty"`
	IsActive *bool         `json:"is_active,omitempty"`
	EndDate  *time.Time    `json:"end_date,omitempty"`
}

// InvoiceLineItem represents a line item for invoicing
type InvoiceLineItem struct {
	Description string  `json:"description"`
	Quantity    int64   `json:"quantity"`
	Unit        string  `json:"unit"`
	UnitPrice   float64 `json:"unit_price"`
	Total       float64 `json:"total"`
}

// Invoice represents a tenant invoice
type Invoice struct {
	ID        string            `json:"id" db:"id"`
	TenantID  string            `json:"tenant_id" db:"tenant_id"`
	Period    string            `json:"period" db:"period"`
	LineItems []InvoiceLineItem `json:"line_items" db:"line_items"`
	Subtotal  float64           `json:"subtotal" db:"subtotal"`
	Tax       float64           `json:"tax" db:"tax"`
	Total     float64           `json:"total" db:"total"`
	Currency  string            `json:"currency" db:"currency"`
	Status    string            `json:"status" db:"status"` // draft, pending, paid
	DueDate   time.Time         `json:"due_date" db:"due_date"`
	PaidAt    *time.Time        `json:"paid_at,omitempty" db:"paid_at"`
	CreatedAt time.Time         `json:"created_at" db:"created_at"`
}

// DefaultRates returns default pricing rates
func DefaultRates() []Rate {
	return []Rate{
		{Unit: UnitDelivery, Price: 0.0001, Currency: "USD", MinUnits: 0, MaxUnits: 0},
		{Unit: UnitByte, Price: 0.00000001, Currency: "USD", MinUnits: 0, MaxUnits: 0},
		{Unit: UnitRetry, Price: 0.00005, Currency: "USD", MinUnits: 0, MaxUnits: 0},
		{Unit: UnitTransform, Price: 0.00002, Currency: "USD", MinUnits: 0, MaxUnits: 0},
	}
}
