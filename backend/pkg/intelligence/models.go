package intelligence

import "time"

// PredictionType classifies the kind of ML prediction
type PredictionType string

const (
	PredictionFailure     PredictionType = "failure"
	PredictionLatency     PredictionType = "latency"
	PredictionRetryNeeded PredictionType = "retry_needed"
	PredictionAnomaly     PredictionType = "anomaly"
)

// RiskLevel categorizes the severity of a prediction
type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

// EventCategory represents ML-classified event types
type EventCategory string

const (
	CategoryPayment       EventCategory = "payment"
	CategoryNotification  EventCategory = "notification"
	CategoryDataSync      EventCategory = "data_sync"
	CategoryUserAction    EventCategory = "user_action"
	CategorySystemEvent   EventCategory = "system_event"
	CategorySecurityAlert EventCategory = "security_alert"
	CategoryUnknown       EventCategory = "unknown"
)

// FailurePrediction represents an ML-based delivery failure prediction
type FailurePrediction struct {
	ID                 string         `json:"id" db:"id"`
	TenantID           string         `json:"tenant_id" db:"tenant_id"`
	EndpointID         string         `json:"endpoint_id" db:"endpoint_id"`
	WebhookID          string         `json:"webhook_id,omitempty" db:"webhook_id"`
	PredictionType     PredictionType `json:"prediction_type" db:"prediction_type"`
	FailureProbability float64        `json:"failure_probability" db:"failure_probability"`
	RiskLevel          RiskLevel      `json:"risk_level" db:"risk_level"`
	Confidence         float64        `json:"confidence" db:"confidence"`
	Reasons            []string       `json:"reasons" db:"-"`
	Recommendation     string         `json:"recommendation" db:"recommendation"`
	Features           *FeatureVector `json:"features,omitempty" db:"-"`
	CreatedAt          time.Time      `json:"created_at" db:"created_at"`
	ExpiresAt          time.Time      `json:"expires_at" db:"expires_at"`
	Resolved           bool           `json:"resolved" db:"resolved"`
}

// FeatureVector holds the ML input features for prediction
type FeatureVector struct {
	AvgLatencyMs        float64 `json:"avg_latency_ms"`
	P99LatencyMs        float64 `json:"p99_latency_ms"`
	FailureRate24h      float64 `json:"failure_rate_24h"`
	FailureRate7d       float64 `json:"failure_rate_7d"`
	ConsecutiveFailures int     `json:"consecutive_failures"`
	ResponseTimetrend   float64 `json:"response_time_trend"`
	PayloadSizeAvg      int64   `json:"payload_size_avg"`
	RequestsPerMinute   float64 `json:"requests_per_minute"`
	LastSuccessAgo      int64   `json:"last_success_ago_seconds"`
	EndpointAge         int64   `json:"endpoint_age_seconds"`
	SSLDaysRemaining    int     `json:"ssl_days_remaining"`
	ErrorDiversity      int     `json:"error_diversity"`
}

// AnomalyDetection represents a detected anomaly in webhook patterns
type AnomalyDetection struct {
	ID           string    `json:"id" db:"id"`
	TenantID     string    `json:"tenant_id" db:"tenant_id"`
	EndpointID   string    `json:"endpoint_id" db:"endpoint_id"`
	AnomalyType  string    `json:"anomaly_type" db:"anomaly_type"`
	Description  string    `json:"description" db:"description"`
	Severity     RiskLevel `json:"severity" db:"severity"`
	Score        float64   `json:"anomaly_score" db:"anomaly_score"`
	Baseline     float64   `json:"baseline_value" db:"baseline_value"`
	Observed     float64   `json:"observed_value" db:"observed_value"`
	Deviation    float64   `json:"deviation" db:"deviation"`
	DetectedAt   time.Time `json:"detected_at" db:"detected_at"`
	Acknowledged bool      `json:"acknowledged" db:"acknowledged"`
}

// RetryOptimization suggests optimal retry strategies
type RetryOptimization struct {
	ID                   string    `json:"id" db:"id"`
	TenantID             string    `json:"tenant_id" db:"tenant_id"`
	EndpointID           string    `json:"endpoint_id" db:"endpoint_id"`
	CurrentMaxRetries    int       `json:"current_max_retries" db:"current_max_retries"`
	SuggestedRetries     int       `json:"suggested_max_retries" db:"suggested_max_retries"`
	CurrentBackoff       string    `json:"current_backoff" db:"current_backoff"`
	SuggestedBackoff     string    `json:"suggested_backoff" db:"suggested_backoff"`
	EstimatedImprovement float64   `json:"estimated_improvement_pct" db:"estimated_improvement_pct"`
	Rationale            string    `json:"rationale" db:"rationale"`
	DataPoints           int       `json:"data_points" db:"data_points"`
	CreatedAt            time.Time `json:"created_at" db:"created_at"`
	Applied              bool      `json:"applied" db:"applied"`
}

// EventClassification represents an ML-classified webhook event
type EventClassification struct {
	ID         string        `json:"id" db:"id"`
	TenantID   string        `json:"tenant_id" db:"tenant_id"`
	WebhookID  string        `json:"webhook_id" db:"webhook_id"`
	EventType  string        `json:"event_type" db:"event_type"`
	Category   EventCategory `json:"category" db:"category"`
	Confidence float64       `json:"confidence" db:"confidence"`
	Labels     []string      `json:"labels" db:"-"`
	CreatedAt  time.Time     `json:"created_at" db:"created_at"`
}

// IntelligenceInsight represents a dashboard-ready AI insight
type IntelligenceInsight struct {
	ID          string         `json:"id" db:"id"`
	TenantID    string         `json:"tenant_id" db:"tenant_id"`
	InsightType string         `json:"insight_type" db:"insight_type"`
	Title       string         `json:"title" db:"title"`
	Description string         `json:"description" db:"description"`
	Severity    RiskLevel      `json:"severity" db:"severity"`
	ActionURL   string         `json:"action_url,omitempty" db:"action_url"`
	ActionLabel string         `json:"action_label,omitempty" db:"action_label"`
	Data        map[string]any `json:"data,omitempty" db:"-"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	DismissedAt *time.Time     `json:"dismissed_at,omitempty" db:"dismissed_at"`
}

// EndpointHealthScore aggregates health metrics for an endpoint
type EndpointHealthScore struct {
	EndpointID        string    `json:"endpoint_id" db:"endpoint_id"`
	TenantID          string    `json:"tenant_id" db:"tenant_id"`
	OverallScore      float64   `json:"overall_score" db:"overall_score"`
	ReliabilityScore  float64   `json:"reliability_score" db:"reliability_score"`
	LatencyScore      float64   `json:"latency_score" db:"latency_score"`
	ThroughputScore   float64   `json:"throughput_score" db:"throughput_score"`
	ErrorRateScore    float64   `json:"error_rate_score" db:"error_rate_score"`
	TrendDirection    string    `json:"trend_direction" db:"trend_direction"`
	PredictedScore24h float64   `json:"predicted_score_24h" db:"predicted_score_24h"`
	CalculatedAt      time.Time `json:"calculated_at" db:"calculated_at"`
}

// IntelligenceDashboard aggregates all AI insights for the dashboard
type IntelligenceDashboard struct {
	Predictions   []FailurePrediction   `json:"predictions"`
	Anomalies     []AnomalyDetection    `json:"anomalies"`
	Optimizations []RetryOptimization   `json:"optimizations"`
	Insights      []IntelligenceInsight `json:"insights"`
	HealthScores  []EndpointHealthScore `json:"health_scores"`
	Summary       *IntelligenceSummary  `json:"summary"`
}

// IntelligenceSummary provides high-level AI metrics
type IntelligenceSummary struct {
	TotalPredictions     int     `json:"total_predictions"`
	ActiveAnomalies      int     `json:"active_anomalies"`
	PendingOptimizations int     `json:"pending_optimizations"`
	AvgHealthScore       float64 `json:"avg_health_score"`
	PredictionAccuracy   float64 `json:"prediction_accuracy"`
	AnomaliesDetected7d  int     `json:"anomalies_detected_7d"`
	EstimatedSavings     float64 `json:"estimated_savings_pct"`
}
