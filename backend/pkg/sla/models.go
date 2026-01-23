package sla

import "time"

// Target defines an SLA target for a tenant or endpoint
type Target struct {
	ID                string    `json:"id" db:"id"`
	TenantID          string    `json:"tenant_id" db:"tenant_id"`
	EndpointID        string    `json:"endpoint_id,omitempty" db:"endpoint_id"`
	Name              string    `json:"name" db:"name"`
	DeliveryRatePct   float64   `json:"delivery_rate_pct" db:"delivery_rate_pct"`
	LatencyP50Ms      int       `json:"latency_p50_ms" db:"latency_p50_ms"`
	LatencyP99Ms      int       `json:"latency_p99_ms" db:"latency_p99_ms"`
	WindowMinutes     int       `json:"window_minutes" db:"window_minutes"`
	IsActive          bool      `json:"is_active" db:"is_active"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// ComplianceStatus represents the current SLA compliance state
type ComplianceStatus struct {
	TargetID        string    `json:"target_id"`
	TargetName      string    `json:"target_name"`
	EndpointID      string    `json:"endpoint_id,omitempty"`
	IsCompliant     bool      `json:"is_compliant"`
	CurrentRate     float64   `json:"current_delivery_rate_pct"`
	RequiredRate    float64   `json:"required_delivery_rate_pct"`
	CurrentP50Ms    int       `json:"current_latency_p50_ms"`
	CurrentP99Ms    int       `json:"current_latency_p99_ms"`
	TotalDeliveries int       `json:"total_deliveries"`
	SuccessCount    int       `json:"success_count"`
	FailureCount    int       `json:"failure_count"`
	WindowStart     time.Time `json:"window_start"`
	WindowEnd       time.Time `json:"window_end"`
	MeasuredAt      time.Time `json:"measured_at"`
}

// Breach represents an SLA breach event
type Breach struct {
	ID           string    `json:"id" db:"id"`
	TenantID     string    `json:"tenant_id" db:"tenant_id"`
	TargetID     string    `json:"target_id" db:"target_id"`
	EndpointID   string    `json:"endpoint_id,omitempty" db:"endpoint_id"`
	BreachType   string    `json:"breach_type" db:"breach_type"`
	ExpectedVal  float64   `json:"expected_value" db:"expected_value"`
	ActualVal    float64   `json:"actual_value" db:"actual_value"`
	Severity     string    `json:"severity" db:"severity"`
	ResolvedAt   *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
}

// BreachType constants
const (
	BreachTypeDeliveryRate = "delivery_rate"
	BreachTypeLatencyP50   = "latency_p50"
	BreachTypeLatencyP99   = "latency_p99"
)

// SeverityLevel constants
const (
	SeverityWarning  = "warning"
	SeverityCritical = "critical"
)

// AlertConfig defines alerting configuration for an SLA target
type AlertConfig struct {
	ID            string   `json:"id" db:"id"`
	TenantID      string   `json:"tenant_id" db:"tenant_id"`
	TargetID      string   `json:"target_id" db:"target_id"`
	Channels      []string `json:"channels"`
	ChannelsJSON  string   `json:"-" db:"channels"`
	CooldownMins  int      `json:"cooldown_minutes" db:"cooldown_minutes"`
	IsActive      bool     `json:"is_active" db:"is_active"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// AlertChannelType constants
const (
	ChannelSlack     = "slack"
	ChannelPagerDuty = "pagerduty"
	ChannelEmail     = "email"
	ChannelWebhook   = "webhook"
)

// BurnRate measures how fast the error budget is being consumed
type BurnRate struct {
	TargetID       string  `json:"target_id"`
	CurrentRate    float64 `json:"current_burn_rate"`
	ProjectedBreachIn string `json:"projected_breach_in,omitempty"`
	ErrorBudgetPct float64 `json:"error_budget_remaining_pct"`
	IsAtRisk       bool    `json:"is_at_risk"`
}

// CreateTargetRequest is the request DTO for creating an SLA target
type CreateTargetRequest struct {
	EndpointID      string  `json:"endpoint_id,omitempty"`
	Name            string  `json:"name" binding:"required,min=1,max=255"`
	DeliveryRatePct float64 `json:"delivery_rate_pct" binding:"required,min=0,max=100"`
	LatencyP50Ms    int     `json:"latency_p50_ms" binding:"min=0"`
	LatencyP99Ms    int     `json:"latency_p99_ms" binding:"min=0"`
	WindowMinutes   int     `json:"window_minutes" binding:"required,min=1"`
}

// UpdateAlertConfigRequest is the request DTO for configuring alerts
type UpdateAlertConfigRequest struct {
	Channels     []string `json:"channels" binding:"required,min=1"`
	CooldownMins int      `json:"cooldown_minutes" binding:"min=1"`
	IsActive     bool     `json:"is_active"`
}

// Dashboard aggregates SLA data for display
type Dashboard struct {
	TenantID     string             `json:"tenant_id"`
	Targets      []ComplianceStatus `json:"targets"`
	BurnRates    []BurnRate         `json:"burn_rates"`
	ActiveBreaches []Breach         `json:"active_breaches"`
	OverallScore float64            `json:"overall_compliance_score"`
}
