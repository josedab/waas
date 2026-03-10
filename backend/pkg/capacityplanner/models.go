package capacityplanner

import "time"

// TimeRange represents a start and end time window.
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// TrafficSnapshot captures a point-in-time traffic measurement.
type TrafficSnapshot struct {
	Timestamp       time.Time `json:"timestamp" db:"timestamp"`
	RequestsPerSec  float64   `json:"requests_per_sec" db:"requests_per_sec"`
	AvgLatencyMs    float64   `json:"avg_latency_ms" db:"avg_latency_ms"`
	P99LatencyMs    float64   `json:"p99_latency_ms" db:"p99_latency_ms"`
	ErrorRate       float64   `json:"error_rate" db:"error_rate"`
	ActiveEndpoints int       `json:"active_endpoints" db:"active_endpoints"`
	QueueDepth      int       `json:"queue_depth" db:"queue_depth"`
}

// CapacityReport is the top-level capacity analysis report for a tenant.
type CapacityReport struct {
	ID              string                  `json:"id" db:"id"`
	TenantID        string                  `json:"tenant_id" db:"tenant_id"`
	CurrentUsage    UsageMetrics            `json:"current_usage"`
	PeakUsage       UsageMetrics            `json:"peak_usage"`
	Projections     []GrowthProjection      `json:"projections"`
	Recommendations []ScalingRecommendation `json:"recommendations"`
	Bottlenecks     []Bottleneck            `json:"bottlenecks"`
	GeneratedAt     time.Time               `json:"generated_at" db:"generated_at"`
}

// UsageMetrics holds aggregated usage data for a tenant.
type UsageMetrics struct {
	DailyAvgRequests   float64 `json:"daily_avg_requests"`
	PeakRequestsPerSec float64 `json:"peak_requests_per_sec"`
	AvgPayloadSizeKB   float64 `json:"avg_payload_size_kb"`
	TotalEndpoints     int     `json:"total_endpoints"`
	ActiveEndpoints    int     `json:"active_endpoints"`
	StorageUsedGB      float64 `json:"storage_used_gb"`
	BandwidthUsedGB    float64 `json:"bandwidth_used_gb"`
}

// GrowthProjection estimates future usage for a given period.
type GrowthProjection struct {
	Period             string  `json:"period"`
	ProjectedDailyReqs float64 `json:"projected_daily_reqs"`
	ProjectedPeakRPS   float64 `json:"projected_peak_rps"`
	GrowthRatePct      float64 `json:"growth_rate_pct"`
	ConfidenceLevel    float64 `json:"confidence_level"`
}

// ScalingRecommendation suggests a resource scaling action.
type ScalingRecommendation struct {
	Resource            string  `json:"resource"`
	CurrentCapacity     string  `json:"current_capacity"`
	RecommendedCapacity string  `json:"recommended_capacity"`
	Urgency             string  `json:"urgency"`
	Reason              string  `json:"reason"`
	EstimatedCostImpact float64 `json:"estimated_cost_impact"`
}

// Bottleneck identifies a resource constraint.
type Bottleneck struct {
	Resource       string  `json:"resource"`
	CurrentUtilPct float64 `json:"current_util_pct"`
	ThresholdPct   float64 `json:"threshold_pct"`
	Impact         string  `json:"impact"`
	Mitigation     string  `json:"mitigation"`
}

// CapacityAlert represents an alert triggered by a capacity threshold breach.
type CapacityAlert struct {
	ID             string    `json:"id" db:"id"`
	TenantID       string    `json:"tenant_id" db:"tenant_id"`
	Resource       string    `json:"resource" db:"resource"`
	Severity       string    `json:"severity" db:"severity"`
	Message        string    `json:"message" db:"message"`
	CurrentValue   float64   `json:"current_value" db:"current_value"`
	ThresholdValue float64   `json:"threshold_value" db:"threshold_value"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// AlertStatus constants
const (
	AlertSeverityCritical = "critical"
	AlertSeverityWarning  = "warning"
	AlertSeverityInfo     = "info"
)

// GenerateReportRequest is the request DTO for generating a capacity report.
type GenerateReportRequest struct {
	PeriodStart time.Time `json:"period_start" binding:"required"`
	PeriodEnd   time.Time `json:"period_end" binding:"required"`
}

// SetAlertThresholdRequest is the request DTO for setting an alert threshold.
type SetAlertThresholdRequest struct {
	Resource       string  `json:"resource" binding:"required"`
	ThresholdValue float64 `json:"threshold_value" binding:"required,min=0"`
	Severity       string  `json:"severity" binding:"required"`
}

// Cloud provider constants
const (
	CloudAWS   = "aws"
	CloudGCP   = "gcp"
	CloudAzure = "azure"
)

// CloudCostForecast projects costs for a specific cloud provider.
type CloudCostForecast struct {
	Provider           string          `json:"provider"`
	CurrentMonthlyCost float64         `json:"current_monthly_cost"`
	ProjectedCosts     []ProjectedCost `json:"projected_costs"`
	CostBreakdown      []CostLineItem  `json:"cost_breakdown"`
	Savings            []CostSavingTip `json:"savings,omitempty"`
}

// ProjectedCost is a future cost estimate for a period.
type ProjectedCost struct {
	Period     string  `json:"period"`
	Amount     float64 `json:"amount"`
	Confidence float64 `json:"confidence"`
}

// CostLineItem is a single cost category.
type CostLineItem struct {
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
	Unit     string  `json:"unit"`
}

// CostSavingTip suggests a way to reduce costs.
type CostSavingTip struct {
	Description     string  `json:"description"`
	EstimatedSaving float64 `json:"estimated_saving_pct"`
	Effort          string  `json:"effort"` // low, medium, high
}

// WeeklyDigest is a periodic summary sent to stakeholders.
type WeeklyDigest struct {
	TenantID           string                  `json:"tenant_id"`
	PeriodStart        time.Time               `json:"period_start"`
	PeriodEnd          time.Time               `json:"period_end"`
	UsageSummary       UsageMetrics            `json:"usage_summary"`
	TopRecommendations []ScalingRecommendation `json:"top_recommendations"`
	CostForecast       *CloudCostForecast      `json:"cost_forecast,omitempty"`
	Alerts             []CapacityAlert         `json:"alerts,omitempty"`
	TrendDirection     string                  `json:"trend_direction"` // growing, stable, declining
	GeneratedAt        time.Time               `json:"generated_at"`
}
