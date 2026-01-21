package anomaly

import (
	"time"
)

// MetricType represents the type of metric being tracked
type MetricType string

const (
	MetricTypeDeliveryRate  MetricType = "delivery_rate"
	MetricTypeErrorRate     MetricType = "error_rate"
	MetricTypeLatencyP50    MetricType = "latency_p50"
	MetricTypeLatencyP95    MetricType = "latency_p95"
	MetricTypeLatencyP99    MetricType = "latency_p99"
	MetricTypeQueueDepth    MetricType = "queue_depth"
	MetricTypeRetryRate     MetricType = "retry_rate"
)

// Severity levels for anomalies
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityWarning  Severity = "warning"
	SeverityInfo     Severity = "info"
)

// MetricDataPoint represents a single metric measurement
type MetricDataPoint struct {
	Timestamp  time.Time  `json:"timestamp" db:"timestamp"`
	TenantID   string     `json:"tenant_id" db:"tenant_id"`
	EndpointID string     `json:"endpoint_id,omitempty" db:"endpoint_id"`
	MetricType MetricType `json:"metric_type" db:"metric_type"`
	Value      float64    `json:"value" db:"value"`
}

// Baseline represents the normal behavior pattern for a metric
type Baseline struct {
	ID         string     `json:"id" db:"id"`
	TenantID   string     `json:"tenant_id" db:"tenant_id"`
	EndpointID string     `json:"endpoint_id,omitempty" db:"endpoint_id"`
	MetricType MetricType `json:"metric_type" db:"metric_type"`
	Mean       float64    `json:"mean" db:"mean"`
	StdDev     float64    `json:"std_dev" db:"std_dev"`
	Min        float64    `json:"min" db:"min_value"`
	Max        float64    `json:"max" db:"max_value"`
	P50        float64    `json:"p50" db:"p50"`
	P95        float64    `json:"p95" db:"p95"`
	P99        float64    `json:"p99" db:"p99"`
	SampleSize int64      `json:"sample_size" db:"sample_size"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
}

// Anomaly represents a detected anomaly
type Anomaly struct {
	ID             string     `json:"id" db:"id"`
	TenantID       string     `json:"tenant_id" db:"tenant_id"`
	EndpointID     string     `json:"endpoint_id,omitempty" db:"endpoint_id"`
	MetricType     MetricType `json:"metric_type" db:"metric_type"`
	Severity       Severity   `json:"severity" db:"severity"`
	CurrentValue   float64    `json:"current_value" db:"current_value"`
	ExpectedValue  float64    `json:"expected_value" db:"expected_value"`
	Deviation      float64    `json:"deviation" db:"deviation"`
	DeviationPct   float64    `json:"deviation_pct" db:"deviation_pct"`
	Description    string     `json:"description" db:"description"`
	RootCause      string     `json:"root_cause,omitempty" db:"root_cause"`
	Recommendation string     `json:"recommendation,omitempty" db:"recommendation"`
	Status         string     `json:"status" db:"status"` // open, acknowledged, resolved
	DetectedAt     time.Time  `json:"detected_at" db:"detected_at"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
}

// AnomalyAlert represents an alert generated from an anomaly
type AnomalyAlert struct {
	ID          string    `json:"id" db:"id"`
	TenantID    string    `json:"tenant_id" db:"tenant_id"`
	AnomalyID   string    `json:"anomaly_id" db:"anomaly_id"`
	Channel     string    `json:"channel" db:"channel"` // email, slack, webhook, pagerduty
	Recipient   string    `json:"recipient" db:"recipient"`
	Status      string    `json:"status" db:"status"` // pending, sent, failed
	SentAt      *time.Time `json:"sent_at,omitempty" db:"sent_at"`
	Error       string    `json:"error,omitempty" db:"error"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// DetectionConfig holds configuration for anomaly detection
type DetectionConfig struct {
	ID                 string     `json:"id" db:"id"`
	TenantID           string     `json:"tenant_id" db:"tenant_id"`
	EndpointID         string     `json:"endpoint_id,omitempty" db:"endpoint_id"`
	MetricType         MetricType `json:"metric_type" db:"metric_type"`
	Enabled            bool       `json:"enabled" db:"enabled"`
	Sensitivity        float64    `json:"sensitivity" db:"sensitivity"` // 1.0 = normal, 2.0 = more sensitive
	MinSamples         int        `json:"min_samples" db:"min_samples"` // Min samples before alerting
	CooldownMinutes    int        `json:"cooldown_minutes" db:"cooldown_minutes"`
	CriticalThreshold  float64    `json:"critical_threshold" db:"critical_threshold"` // StdDev multiplier
	WarningThreshold   float64    `json:"warning_threshold" db:"warning_threshold"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"`
}

// AlertConfig holds alert notification configuration
type AlertConfig struct {
	ID         string    `json:"id" db:"id"`
	TenantID   string    `json:"tenant_id" db:"tenant_id"`
	Name       string    `json:"name" db:"name"`
	Channel    string    `json:"channel" db:"channel"`
	Config     string    `json:"config" db:"config"` // JSON config for channel
	MinSeverity Severity `json:"min_severity" db:"min_severity"`
	Enabled    bool      `json:"enabled" db:"enabled"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// DetectionResult represents the result of anomaly detection
type DetectionResult struct {
	IsAnomaly   bool     `json:"is_anomaly"`
	Severity    Severity `json:"severity,omitempty"`
	Score       float64  `json:"score"` // Z-score or deviation score
	Confidence  float64  `json:"confidence"`
	Description string   `json:"description,omitempty"`
}

// TrendAnalysis represents trend information for a metric
type TrendAnalysis struct {
	TenantID   string     `json:"tenant_id"`
	EndpointID string     `json:"endpoint_id,omitempty"`
	MetricType MetricType `json:"metric_type"`
	Trend      string     `json:"trend"` // increasing, decreasing, stable
	ChangeRate float64    `json:"change_rate"` // Percent change per hour
	Forecast   []float64  `json:"forecast,omitempty"` // Predicted values
	Period     string     `json:"period"` // hourly, daily, weekly
}

// CreateDetectionConfigRequest for creating detection config
type CreateDetectionConfigRequest struct {
	EndpointID        string     `json:"endpoint_id,omitempty"`
	MetricType        MetricType `json:"metric_type" binding:"required"`
	Sensitivity       float64    `json:"sensitivity"`
	MinSamples        int        `json:"min_samples"`
	CooldownMinutes   int        `json:"cooldown_minutes"`
	CriticalThreshold float64    `json:"critical_threshold"`
	WarningThreshold  float64    `json:"warning_threshold"`
}

// CreateAlertConfigRequest for creating alert config
type CreateAlertConfigRequest struct {
	Name       string   `json:"name" binding:"required"`
	Channel    string   `json:"channel" binding:"required,oneof=email slack webhook pagerduty"`
	Config     string   `json:"config" binding:"required"`
	MinSeverity Severity `json:"min_severity" binding:"required"`
}
