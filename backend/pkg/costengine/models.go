package costengine

import "time"

// CostModel defines the pricing model for webhook deliveries
type CostModel struct {
	ID                    string    `json:"id" db:"id"`
	TenantID              string    `json:"tenant_id" db:"tenant_id"`
	Name                  string    `json:"name" db:"name"`
	ComputeCostPerDelivery float64  `json:"compute_cost_per_delivery" db:"compute_cost_per_delivery"`
	BandwidthCostPerKB    float64   `json:"bandwidth_cost_per_kb" db:"bandwidth_cost_per_kb"`
	RetryCostMultiplier   float64   `json:"retry_cost_multiplier" db:"retry_cost_multiplier"`
	StorageCostPerGBDay   float64   `json:"storage_cost_per_gb_day" db:"storage_cost_per_gb_day"`
	Currency              string    `json:"currency" db:"currency"`
	IsActive              bool      `json:"is_active" db:"is_active"`
	CreatedAt             time.Time `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time `json:"updated_at" db:"updated_at"`
}

// DeliveryCost represents the cost breakdown for a single delivery
type DeliveryCost struct {
	ID               string    `json:"id" db:"id"`
	TenantID         string    `json:"tenant_id" db:"tenant_id"`
	DeliveryID       string    `json:"delivery_id" db:"delivery_id"`
	EndpointID       string    `json:"endpoint_id" db:"endpoint_id"`
	EventType        string    `json:"event_type" db:"event_type"`
	ComputeCost      float64   `json:"compute_cost" db:"compute_cost"`
	BandwidthCost    float64   `json:"bandwidth_cost" db:"bandwidth_cost"`
	RetryCost        float64   `json:"retry_cost" db:"retry_cost"`
	TotalCost        float64   `json:"total_cost" db:"total_cost"`
	PayloadSizeBytes int64     `json:"payload_size_bytes" db:"payload_size_bytes"`
	RetryCount       int       `json:"retry_count" db:"retry_count"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// CostReport aggregates cost data for a reporting period
type CostReport struct {
	TenantID          string            `json:"tenant_id"`
	PeriodStart       time.Time         `json:"period_start"`
	PeriodEnd         time.Time         `json:"period_end"`
	TotalCost         float64           `json:"total_cost"`
	DeliveryCount     int64             `json:"delivery_count"`
	AvgCostPerDelivery float64          `json:"avg_cost_per_delivery"`
	CostByEndpoint    map[string]float64 `json:"cost_by_endpoint"`
	CostByEventType   map[string]float64 `json:"cost_by_event_type"`
	CostByDay         []DailyCost       `json:"cost_by_day"`
	TopCostEndpoints  []EndpointCost    `json:"top_cost_endpoints"`
}

// DailyCost represents cost data for a single day
type DailyCost struct {
	Date       string  `json:"date"`
	Cost       float64 `json:"cost"`
	Deliveries int64   `json:"deliveries"`
}

// EndpointCost represents cost data for a single endpoint
type EndpointCost struct {
	EndpointID    string  `json:"endpoint_id"`
	EndpointURL   string  `json:"endpoint_url"`
	TotalCost     float64 `json:"total_cost"`
	DeliveryCount int64   `json:"delivery_count"`
}

// CostAnomaly represents an unusual cost pattern
type CostAnomaly struct {
	ID           string    `json:"id" db:"id"`
	TenantID     string    `json:"tenant_id" db:"tenant_id"`
	EndpointID   string    `json:"endpoint_id" db:"endpoint_id"`
	AnomalyType  string    `json:"anomaly_type" db:"anomaly_type"`
	ExpectedCost float64   `json:"expected_cost" db:"expected_cost"`
	ActualCost   float64   `json:"actual_cost" db:"actual_cost"`
	DeviationPct float64   `json:"deviation_pct" db:"deviation_pct"`
	DetectedAt   time.Time `json:"detected_at" db:"detected_at"`
	Status       string    `json:"status" db:"status"`
}

// AnomalyType constants
const (
	AnomalyTypeSpike             = "spike"
	AnomalyTypeSustainedIncrease = "sustained_increase"
	AnomalyTypeUnusualPattern    = "unusual_pattern"
)

// AnomalyStatus constants
const (
	AnomalyStatusActive       = "active"
	AnomalyStatusAcknowledged = "acknowledged"
	AnomalyStatusResolved     = "resolved"
)

// CostBudget defines a spending budget for a tenant
type CostBudget struct {
	ID                string    `json:"id" db:"id"`
	TenantID          string    `json:"tenant_id" db:"tenant_id"`
	Name              string    `json:"name" db:"name"`
	MonthlyLimit      float64   `json:"monthly_limit" db:"monthly_limit"`
	AlertThresholdPct float64   `json:"alert_threshold_pct" db:"alert_threshold_pct"`
	CurrentSpend      float64   `json:"current_spend" db:"current_spend"`
	Period            string    `json:"period" db:"period"`
	IsActive          bool      `json:"is_active" db:"is_active"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// CreateCostModelRequest is the request DTO for creating a cost model
type CreateCostModelRequest struct {
	Name                   string  `json:"name" binding:"required,min=1,max=255"`
	ComputeCostPerDelivery float64 `json:"compute_cost_per_delivery" binding:"required,min=0"`
	BandwidthCostPerKB     float64 `json:"bandwidth_cost_per_kb" binding:"required,min=0"`
	RetryCostMultiplier    float64 `json:"retry_cost_multiplier" binding:"required,min=0"`
	StorageCostPerGBDay    float64 `json:"storage_cost_per_gb_day" binding:"min=0"`
	Currency               string  `json:"currency" binding:"required,min=3,max=3"`
}

// RecordDeliveryCostRequest is the request DTO for recording a delivery cost
type RecordDeliveryCostRequest struct {
	DeliveryID       string `json:"delivery_id" binding:"required"`
	EndpointID       string `json:"endpoint_id" binding:"required"`
	EventType        string `json:"event_type" binding:"required"`
	PayloadSizeBytes int64  `json:"payload_size_bytes" binding:"min=0"`
	RetryCount       int    `json:"retry_count" binding:"min=0"`
}

// CreateBudgetRequest is the request DTO for creating a cost budget
type CreateBudgetRequest struct {
	Name              string  `json:"name" binding:"required,min=1,max=255"`
	MonthlyLimit      float64 `json:"monthly_limit" binding:"required,min=0"`
	AlertThresholdPct float64 `json:"alert_threshold_pct" binding:"required,min=0,max=100"`
	Period            string  `json:"period" binding:"required"`
}

// GenerateReportRequest is the request DTO for generating a cost report
type GenerateReportRequest struct {
	PeriodStart time.Time `json:"period_start" binding:"required"`
	PeriodEnd   time.Time `json:"period_end" binding:"required"`
}
