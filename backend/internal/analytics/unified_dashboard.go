package analytics

import (
	"math"
	"sort"
	"time"
)

// UnifiedDashboard aggregates distributed traces, anomaly detection,
// SLA monitoring, and alerting into a single observability view.
type UnifiedDashboard struct {
	TenantID     string              `json:"tenant_id"`
	GeneratedAt  time.Time           `json:"generated_at"`
	TimeWindow   string              `json:"time_window"`
	Overview     DashboardOverview   `json:"overview"`
	Traces       TracesSummary       `json:"traces"`
	Anomalies    AnomalySummary      `json:"anomalies"`
	SLA          SLAStatus           `json:"sla"`
	Alerts       AlertsSummary       `json:"alerts"`
	TopEndpoints []EndpointHealth    `json:"top_endpoints"`
	ErrorTrends  []ErrorTrendPoint   `json:"error_trends"`
}

// DashboardOverview provides high-level system health
type DashboardOverview struct {
	TotalDeliveries    int64   `json:"total_deliveries"`
	SuccessRate        float64 `json:"success_rate"`
	AvgLatencyMs       float64 `json:"avg_latency_ms"`
	P99LatencyMs       float64 `json:"p99_latency_ms"`
	ActiveEndpoints    int     `json:"active_endpoints"`
	ErrorCount         int64   `json:"error_count"`
	ThroughputPerSec   float64 `json:"throughput_per_sec"`
	HealthScore        float64 `json:"health_score"` // 0-100
}

// TracesSummary summarizes distributed trace data
type TracesSummary struct {
	TotalTraces     int64            `json:"total_traces"`
	AvgSpanCount    float64          `json:"avg_span_count"`
	AvgDurationMs   float64          `json:"avg_duration_ms"`
	ErrorTraces     int64            `json:"error_traces"`
	SlowTraces      int64            `json:"slow_traces"`
	ServiceBreakdown map[string]int64 `json:"service_breakdown"`
	LatencyBuckets  []LatencyBucket  `json:"latency_buckets"`
}

// LatencyBucket groups trace counts by latency range
type LatencyBucket struct {
	Label   string `json:"label"`
	MinMs   int64  `json:"min_ms"`
	MaxMs   int64  `json:"max_ms"`
	Count   int64  `json:"count"`
	Percent float64 `json:"percent"`
}

// AnomalySummary summarizes detected anomalies
type AnomalySummary struct {
	TotalAnomalies    int64              `json:"total_anomalies"`
	CriticalCount     int64              `json:"critical_count"`
	WarningCount      int64              `json:"warning_count"`
	RecentAnomalies   []DetectedAnomaly  `json:"recent_anomalies"`
	AnomalyTrend      string             `json:"anomaly_trend"` // increasing, stable, decreasing
}

// DetectedAnomaly represents a single detected anomaly
type DetectedAnomaly struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`  // latency_spike, error_burst, traffic_drop, pattern_change
	Severity    string    `json:"severity"` // critical, warning, info
	Description string    `json:"description"`
	Metric      string    `json:"metric"`
	Expected    float64   `json:"expected"`
	Actual      float64   `json:"actual"`
	Deviation   float64   `json:"deviation"`
	EndpointID  string    `json:"endpoint_id,omitempty"`
	DetectedAt  time.Time `json:"detected_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

// SLAStatus summarizes SLA compliance
type SLAStatus struct {
	OverallCompliance float64         `json:"overall_compliance"` // 0-100
	TargetsMet        int             `json:"targets_met"`
	TargetsBreached   int             `json:"targets_breached"`
	BurnRate          float64         `json:"burn_rate"`
	ErrorBudgetRemain float64         `json:"error_budget_remaining"` // percentage
	ActiveBreaches    []SLABreach     `json:"active_breaches"`
	ComplianceTrend   []CompliancePoint `json:"compliance_trend"`
}

// SLABreach represents an active SLA breach
type SLABreach struct {
	TargetName  string    `json:"target_name"`
	Metric      string    `json:"metric"` // delivery_rate, p99_latency, uptime
	Target      float64   `json:"target"`
	Actual      float64   `json:"actual"`
	Severity    string    `json:"severity"`
	StartedAt   time.Time `json:"started_at"`
	Duration    string    `json:"duration"`
}

// CompliancePoint tracks SLA compliance over time
type CompliancePoint struct {
	Timestamp  time.Time `json:"timestamp"`
	Compliance float64   `json:"compliance"`
}

// AlertsSummary summarizes active and recent alerts
type AlertsSummary struct {
	ActiveAlerts   int              `json:"active_alerts"`
	CriticalAlerts int              `json:"critical_alerts"`
	RecentAlerts   []DashboardAlert `json:"recent_alerts"`
	AlertsByType   map[string]int   `json:"alerts_by_type"`
}

// DashboardAlert represents an alert in the dashboard
type DashboardAlert struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Severity    string    `json:"severity"` // critical, warning, info
	Status      string    `json:"status"`   // firing, resolved, silenced
	Description string    `json:"description"`
	Source      string    `json:"source"` // prometheus, otel, sla, anomaly
	FiredAt     time.Time `json:"fired_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

// EndpointHealth represents health metrics for a single endpoint
type EndpointHealth struct {
	EndpointID   string  `json:"endpoint_id"`
	EndpointURL  string  `json:"endpoint_url"`
	SuccessRate  float64 `json:"success_rate"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	Deliveries   int64   `json:"deliveries"`
	ErrorCount   int64   `json:"error_count"`
	Status       string  `json:"status"` // healthy, degraded, unhealthy
}

// ErrorTrendPoint tracks error rate over time
type ErrorTrendPoint struct {
	Timestamp time.Time `json:"timestamp"`
	ErrorRate float64   `json:"error_rate"`
	Count     int64     `json:"count"`
}

// AnomalyDetector detects anomalies in time-series data using statistical methods
type AnomalyDetector struct {
	windowSize int
	threshold  float64 // z-score threshold for anomaly
}

// NewAnomalyDetector creates a detector with configurable sensitivity
func NewAnomalyDetector(windowSize int, threshold float64) *AnomalyDetector {
	if windowSize <= 0 {
		windowSize = 30
	}
	if threshold <= 0 {
		threshold = 2.5
	}
	return &AnomalyDetector{
		windowSize: windowSize,
		threshold:  threshold,
	}
}

// TimeSeriesPoint is a single data point in a time series
type TimeSeriesPoint struct {
	Timestamp time.Time
	Value     float64
}

// Detect runs anomaly detection on a time series and returns detected anomalies
func (d *AnomalyDetector) Detect(series []TimeSeriesPoint, metricName string) []DetectedAnomaly {
	if len(series) < d.windowSize {
		return nil
	}

	var anomalies []DetectedAnomaly

	for i := d.windowSize; i < len(series); i++ {
		window := series[i-d.windowSize : i]
		mean, stddev := computeStats(window)

		if stddev == 0 {
			continue
		}

		zScore := (series[i].Value - mean) / stddev
		if math.Abs(zScore) > d.threshold {
			severity := "warning"
			if math.Abs(zScore) > d.threshold*1.5 {
				severity = "critical"
			}

			anomalyType := "latency_spike"
			if series[i].Value < mean {
				anomalyType = "traffic_drop"
			}

			anomalies = append(anomalies, DetectedAnomaly{
				Type:        anomalyType,
				Severity:    severity,
				Description: metricName + " deviated significantly from expected range",
				Metric:      metricName,
				Expected:    mean,
				Actual:      series[i].Value,
				Deviation:   zScore,
				DetectedAt:  series[i].Timestamp,
			})
		}
	}

	return anomalies
}

func computeStats(points []TimeSeriesPoint) (mean, stddev float64) {
	if len(points) == 0 {
		return 0, 0
	}

	sum := 0.0
	for _, p := range points {
		sum += p.Value
	}
	mean = sum / float64(len(points))

	sumSq := 0.0
	for _, p := range points {
		diff := p.Value - mean
		sumSq += diff * diff
	}
	stddev = math.Sqrt(sumSq / float64(len(points)))
	return
}

// ComputeHealthScore calculates a 0-100 health score from multiple signals
func ComputeHealthScore(successRate, p99LatencyMs, errorBudgetRemaining float64, activeBreaches int) float64 {
	// Weighted scoring
	successComponent := successRate * 40                              // 40% weight
	latencyComponent := math.Max(0, 30-p99LatencyMs/100) * (30.0/30) // 30% weight, penalize high latency
	budgetComponent := errorBudgetRemaining * 0.2                    // 20% weight
	breachComponent := math.Max(0, 10-float64(activeBreaches)*5)     // 10% weight

	score := successComponent + latencyComponent + budgetComponent + breachComponent
	return math.Max(0, math.Min(100, score))
}

// BuildLatencyBuckets creates histogram buckets from latency values
func BuildLatencyBuckets(latencies []float64) []LatencyBucket {
	bucketDefs := []struct {
		Label string
		Min   int64
		Max   int64
	}{
		{"<50ms", 0, 50},
		{"50-100ms", 50, 100},
		{"100-250ms", 100, 250},
		{"250-500ms", 250, 500},
		{"500ms-1s", 500, 1000},
		{">1s", 1000, math.MaxInt64},
	}

	total := float64(len(latencies))
	buckets := make([]LatencyBucket, len(bucketDefs))

	for i, def := range bucketDefs {
		count := int64(0)
		for _, l := range latencies {
			if int64(l) >= def.Min && int64(l) < def.Max {
				count++
			}
		}
		pct := 0.0
		if total > 0 {
			pct = float64(count) / total * 100
		}
		buckets[i] = LatencyBucket{
			Label:   def.Label,
			MinMs:   def.Min,
			MaxMs:   def.Max,
			Count:   count,
			Percent: pct,
		}
	}

	return buckets
}

// ComputeP99 calculates the 99th percentile from a sorted slice
func ComputeP99(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	idx := int(float64(len(sorted)) * 0.99)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// DetermineAnomalyTrend analyzes anomaly counts to determine trend
func DetermineAnomalyTrend(recentCounts []int64) string {
	if len(recentCounts) < 2 {
		return "stable"
	}

	half := len(recentCounts) / 2
	firstHalf := int64(0)
	secondHalf := int64(0)
	for i, c := range recentCounts {
		if i < half {
			firstHalf += c
		} else {
			secondHalf += c
		}
	}

	if secondHalf > firstHalf*2 {
		return "increasing"
	}
	if secondHalf < firstHalf/2 {
		return "decreasing"
	}
	return "stable"
}
