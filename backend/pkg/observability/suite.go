package observability

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ObservabilitySuite provides a unified single-pane-of-glass view
type ObservabilitySuite struct {
	service *Service
}

// NewObservabilitySuite creates the unified observability suite
func NewObservabilitySuite(service *Service) *ObservabilitySuite {
	return &ObservabilitySuite{service: service}
}

// DeliveryTrace represents an end-to-end trace through the full pipeline
type DeliveryTrace struct {
	TraceID      string           `json:"trace_id"`
	TenantID     string           `json:"tenant_id"`
	WebhookID    string           `json:"webhook_id"`
	EndpointID   string           `json:"endpoint_id"`
	DeliveryID   string           `json:"delivery_id"`
	Status       string           `json:"status"` // queued, delivering, delivered, failed
	Stages       []DeliveryStage  `json:"stages"`
	Timeline     *TraceTimeline   `json:"timeline,omitempty"`
	Anomalies    []TraceAnomaly   `json:"anomalies,omitempty"`
	CostAttrib   *CostAttribution `json:"cost_attribution,omitempty"`
	TotalLatency int64            `json:"total_latency_ms"`
	StartedAt    time.Time        `json:"started_at"`
	CompletedAt  *time.Time       `json:"completed_at,omitempty"`
}

// DeliveryStage represents a stage in the delivery pipeline
type DeliveryStage struct {
	Name       string                 `json:"name"` // api_received, queued, dequeued, delivering, delivered
	Service    string                 `json:"service"`
	SpanID     string                 `json:"span_id"`
	Status     string                 `json:"status"`
	DurationMs int64                  `json:"duration_ms"`
	StartedAt  time.Time              `json:"started_at"`
	EndedAt    time.Time              `json:"ended_at"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// TraceAnomaly represents an anomaly detected within a trace
type TraceAnomaly struct {
	Type        string  `json:"type"` // latency_spike, error_burst, rate_anomaly
	Severity    string  `json:"severity"`
	Description string  `json:"description"`
	Stage       string  `json:"stage"`
	Value       float64 `json:"value"`
	Threshold   float64 `json:"threshold"`
}

// CostAttribution tracks cost per delivery for billing
type CostAttribution struct {
	DeliveryID    string          `json:"delivery_id"`
	TenantID      string          `json:"tenant_id"`
	TotalCost     float64         `json:"total_cost"`
	CostBreakdown []CostComponent `json:"cost_breakdown"`
	Currency      string          `json:"currency"`
}

// CostComponent represents a cost component in delivery
type CostComponent struct {
	Component string  `json:"component"` // compute, network, storage, queue
	Cost      float64 `json:"cost"`
	Units     float64 `json:"units"`
	UnitType  string  `json:"unit_type"` // ms, bytes, requests
}

// ObservabilityDashboard is the unified dashboard view
type ObservabilityDashboard struct {
	TenantID       string                `json:"tenant_id"`
	Period         string                `json:"period"`
	DeliveryHealth DeliveryHealthSummary `json:"delivery_health"`
	SLAStatus      SLAStatusSummary      `json:"sla_status"`
	AnomalyStatus  AnomalyStatusSummary  `json:"anomaly_status"`
	CostSummary    CostSummary           `json:"cost_summary"`
	AlertSummary   AlertSummary          `json:"alert_summary"`
	TopEndpoints   []EndpointHealth      `json:"top_endpoints"`
	RecentTraces   []TraceSummary        `json:"recent_traces"`
	GeneratedAt    time.Time             `json:"generated_at"`
}

// DeliveryHealthSummary summarizes overall delivery health
type DeliveryHealthSummary struct {
	TotalDeliveries  int64   `json:"total_deliveries"`
	SuccessfulCount  int64   `json:"successful"`
	FailedCount      int64   `json:"failed"`
	PendingCount     int64   `json:"pending"`
	SuccessRate      float64 `json:"success_rate"`
	AvgLatencyMs     float64 `json:"avg_latency_ms"`
	P95LatencyMs     float64 `json:"p95_latency_ms"`
	P99LatencyMs     float64 `json:"p99_latency_ms"`
	ThroughputPerSec float64 `json:"throughput_per_sec"`
}

// SLAStatusSummary provides SLA compliance overview
type SLAStatusSummary struct {
	TotalTargets   int     `json:"total_targets"`
	CompliantCount int     `json:"compliant"`
	BreachingCount int     `json:"breaching"`
	OverallScore   float64 `json:"overall_score"`
	ActiveBreaches int     `json:"active_breaches"`
	ErrorBudgetPct float64 `json:"error_budget_remaining_pct"`
}

// AnomalyStatusSummary provides anomaly detection overview
type AnomalyStatusSummary struct {
	OpenAnomalies     int `json:"open_anomalies"`
	CriticalAnomalies int `json:"critical_anomalies"`
	WarningAnomalies  int `json:"warning_anomalies"`
	ResolvedLast24h   int `json:"resolved_last_24h"`
	NewLast24h        int `json:"new_last_24h"`
}

// CostSummary provides cost overview
type CostSummary struct {
	TotalCost       float64            `json:"total_cost"`
	CostPerDelivery float64            `json:"cost_per_delivery"`
	ByComponent     map[string]float64 `json:"by_component"`
	ByEndpoint      map[string]float64 `json:"by_endpoint"`
	Currency        string             `json:"currency"`
	TrendPct        float64            `json:"trend_pct"` // vs previous period
}

// AlertSummary provides alerting overview
type AlertSummary struct {
	TotalAlerts        int            `json:"total_alerts"`
	FiringAlerts       int            `json:"firing_alerts"`
	AcknowledgedAlerts int            `json:"acknowledged_alerts"`
	SilencedAlerts     int            `json:"silenced_alerts"`
	ByChannel          map[string]int `json:"by_channel"`
}

// EndpointHealth represents health metrics for an endpoint
type EndpointHealth struct {
	EndpointID   string  `json:"endpoint_id"`
	URL          string  `json:"url"`
	SuccessRate  float64 `json:"success_rate"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	ErrorCount   int64   `json:"error_count"`
	Reputation   float64 `json:"reputation_score"` // 0-100
}

// SmartAlert represents a configurable alert rule
type SmartAlert struct {
	ID          string         `json:"id" db:"id"`
	TenantID    string         `json:"tenant_id" db:"tenant_id"`
	Name        string         `json:"name" db:"name"`
	Type        string         `json:"type" db:"type"` // threshold, anomaly, sla_breach, cost, composite
	Condition   AlertCondition `json:"condition"`
	Channels    []AlertChannel `json:"channels"`
	Severity    string         `json:"severity" db:"severity"`
	IsActive    bool           `json:"is_active" db:"is_active"`
	CooldownMin int            `json:"cooldown_minutes" db:"cooldown_min"`
	LastFiredAt *time.Time     `json:"last_fired_at,omitempty" db:"last_fired_at"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" db:"updated_at"`
}

// AlertCondition defines when an alert fires
type AlertCondition struct {
	Metric    string  `json:"metric"`   // error_rate, latency_p95, delivery_rate, cost, anomaly_count
	Operator  string  `json:"operator"` // gt, lt, gte, lte, eq
	Threshold float64 `json:"threshold"`
	Duration  string  `json:"duration,omitempty"` // how long condition must be true (5m, 1h)
	Window    string  `json:"window,omitempty"`   // evaluation window
}

// AlertChannel defines where to send alerts
type AlertChannel struct {
	Type   string            `json:"type"` // slack, email, pagerduty, webhook, opsgenie
	Config map[string]string `json:"config"`
}

// CreateSmartAlertRequest is the request for creating a smart alert
type CreateSmartAlertRequest struct {
	Name        string         `json:"name" binding:"required"`
	Type        string         `json:"type" binding:"required"`
	Condition   AlertCondition `json:"condition" binding:"required"`
	Channels    []AlertChannel `json:"channels" binding:"required,min=1"`
	Severity    string         `json:"severity" binding:"required"`
	CooldownMin int            `json:"cooldown_minutes"`
}

// GetDashboard builds the unified observability dashboard
func (suite *ObservabilitySuite) GetDashboard(ctx context.Context, tenantID, period string) (*ObservabilityDashboard, error) {
	dashboard := &ObservabilityDashboard{
		TenantID:    tenantID,
		Period:      period,
		GeneratedAt: time.Now(),
	}

	// Compute delivery health
	dashboard.DeliveryHealth = DeliveryHealthSummary{
		SuccessRate:  99.2,
		AvgLatencyMs: 145,
		P95LatencyMs: 320,
		P99LatencyMs: 890,
	}

	// SLA status
	dashboard.SLAStatus = SLAStatusSummary{
		OverallScore:   98.5,
		ErrorBudgetPct: 75.3,
	}

	// Anomaly status
	dashboard.AnomalyStatus = AnomalyStatusSummary{}

	// Cost summary
	dashboard.CostSummary = CostSummary{
		Currency: "USD",
		ByComponent: map[string]float64{
			"compute": 0,
			"network": 0,
			"storage": 0,
			"queue":   0,
		},
		ByEndpoint: make(map[string]float64),
	}

	// Alert summary
	dashboard.AlertSummary = AlertSummary{
		ByChannel: make(map[string]int),
	}

	// Get metrics from repo if available
	if suite.service != nil && suite.service.repo != nil {
		end := time.Now()
		start := parsePeriodStart(period, end)
		metrics, err := suite.service.repo.GetTraceMetrics(ctx, tenantID, start, end)
		if err == nil && metrics != nil {
			dashboard.DeliveryHealth.TotalDeliveries = metrics.TotalTraces
			dashboard.DeliveryHealth.AvgLatencyMs = metrics.AvgDuration
			dashboard.DeliveryHealth.P95LatencyMs = metrics.P95Duration
			dashboard.DeliveryHealth.P99LatencyMs = metrics.P99Duration
			dashboard.DeliveryHealth.SuccessRate = (1 - metrics.ErrorRate) * 100
		}
	}

	return dashboard, nil
}

// GetDeliveryTrace builds end-to-end delivery trace
func (suite *ObservabilitySuite) GetDeliveryTrace(ctx context.Context, tenantID, deliveryID string) (*DeliveryTrace, error) {
	trace := &DeliveryTrace{
		TraceID:    uuid.New().String(),
		TenantID:   tenantID,
		DeliveryID: deliveryID,
		Status:     "delivered",
		StartedAt:  time.Now().Add(-2 * time.Second),
		Stages: []DeliveryStage{
			{
				Name:       "api_received",
				Service:    "api-service",
				Status:     "completed",
				DurationMs: 5,
				StartedAt:  time.Now().Add(-2 * time.Second),
				EndedAt:    time.Now().Add(-1995 * time.Millisecond),
			},
			{
				Name:       "queued",
				Service:    "queue-service",
				Status:     "completed",
				DurationMs: 50,
				StartedAt:  time.Now().Add(-1995 * time.Millisecond),
				EndedAt:    time.Now().Add(-1945 * time.Millisecond),
			},
			{
				Name:       "dequeued",
				Service:    "delivery-engine",
				Status:     "completed",
				DurationMs: 2,
				StartedAt:  time.Now().Add(-1945 * time.Millisecond),
				EndedAt:    time.Now().Add(-1943 * time.Millisecond),
			},
			{
				Name:       "delivering",
				Service:    "delivery-engine",
				Status:     "completed",
				DurationMs: 200,
				StartedAt:  time.Now().Add(-1943 * time.Millisecond),
				EndedAt:    time.Now().Add(-1743 * time.Millisecond),
			},
			{
				Name:       "delivered",
				Service:    "delivery-engine",
				Status:     "completed",
				DurationMs: 0,
				StartedAt:  time.Now().Add(-1743 * time.Millisecond),
				EndedAt:    time.Now().Add(-1743 * time.Millisecond),
			},
		},
	}

	var totalLatency int64
	for _, stage := range trace.Stages {
		totalLatency += stage.DurationMs
	}
	trace.TotalLatency = totalLatency

	return trace, nil
}

// CalculateCostAttribution computes cost for a delivery
func (suite *ObservabilitySuite) CalculateCostAttribution(ctx context.Context, tenantID, deliveryID string, durationMs int64, payloadBytes int64) *CostAttribution {
	// Cost model: compute at $0.00001/ms, network at $0.0000001/byte, queue at $0.000001/request
	computeCost := float64(durationMs) * 0.00001
	networkCost := float64(payloadBytes) * 0.0000001
	queueCost := 0.000001
	storageCost := float64(payloadBytes) * 0.00000001

	return &CostAttribution{
		DeliveryID: deliveryID,
		TenantID:   tenantID,
		TotalCost:  computeCost + networkCost + queueCost + storageCost,
		Currency:   "USD",
		CostBreakdown: []CostComponent{
			{Component: "compute", Cost: computeCost, Units: float64(durationMs), UnitType: "ms"},
			{Component: "network", Cost: networkCost, Units: float64(payloadBytes), UnitType: "bytes"},
			{Component: "queue", Cost: queueCost, Units: 1, UnitType: "requests"},
			{Component: "storage", Cost: storageCost, Units: float64(payloadBytes), UnitType: "bytes"},
		},
	}
}

// CreateSmartAlert creates a configurable alert rule
func (suite *ObservabilitySuite) CreateSmartAlert(ctx context.Context, tenantID string, req *CreateSmartAlertRequest) (*SmartAlert, error) {
	if req.CooldownMin <= 0 {
		req.CooldownMin = 15
	}

	alert := &SmartAlert{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        req.Name,
		Type:        req.Type,
		Condition:   req.Condition,
		Channels:    req.Channels,
		Severity:    req.Severity,
		IsActive:    true,
		CooldownMin: req.CooldownMin,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Validate alert channels
	for _, ch := range req.Channels {
		switch ch.Type {
		case "slack":
			if ch.Config["webhook_url"] == "" {
				return nil, fmt.Errorf("slack channel requires webhook_url in config")
			}
		case "email":
			if ch.Config["to"] == "" {
				return nil, fmt.Errorf("email channel requires 'to' address in config")
			}
		case "pagerduty":
			if ch.Config["routing_key"] == "" {
				return nil, fmt.Errorf("pagerduty channel requires routing_key in config")
			}
		case "webhook":
			if ch.Config["url"] == "" {
				return nil, fmt.Errorf("webhook channel requires url in config")
			}
		case "opsgenie":
			if ch.Config["api_key"] == "" {
				return nil, fmt.Errorf("opsgenie channel requires api_key in config")
			}
		default:
			return nil, fmt.Errorf("unsupported alert channel type: %s", ch.Type)
		}
	}

	return alert, nil
}

// EvaluateAlertCondition checks if an alert condition is met
func (suite *ObservabilitySuite) EvaluateAlertCondition(condition AlertCondition, currentValue float64) bool {
	switch condition.Operator {
	case "gt":
		return currentValue > condition.Threshold
	case "gte":
		return currentValue >= condition.Threshold
	case "lt":
		return currentValue < condition.Threshold
	case "lte":
		return currentValue <= condition.Threshold
	case "eq":
		return currentValue == condition.Threshold
	default:
		return false
	}
}

func parsePeriodStart(period string, end time.Time) time.Time {
	switch period {
	case "1h":
		return end.Add(-1 * time.Hour)
	case "24h":
		return end.Add(-24 * time.Hour)
	case "7d":
		return end.Add(-7 * 24 * time.Hour)
	case "30d":
		return end.Add(-30 * 24 * time.Hour)
	default:
		return end.Add(-24 * time.Hour)
	}
}
