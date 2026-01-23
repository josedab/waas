// Package prediction provides predictive failure prevention using ML-based analysis
package prediction

import (
	"encoding/json"
	"time"
)

// PredictionType represents types of predictions
type PredictionType string

const (
	PredictionEndpointFailure   PredictionType = "endpoint_failure"
	PredictionLatencySpike      PredictionType = "latency_spike"
	PredictionErrorRateIncrease PredictionType = "error_rate_increase"
	PredictionCapacityExhaustion PredictionType = "capacity_exhaustion"
	PredictionCertificateExpiry PredictionType = "certificate_expiry"
	PredictionQuotaExhaustion   PredictionType = "quota_exhaustion"
)

// AlertSeverity represents alert severity levels
type AlertSeverity string

const (
	SeverityCritical AlertSeverity = "critical"
	SeverityHigh     AlertSeverity = "high"
	SeverityMedium   AlertSeverity = "medium"
	SeverityLow      AlertSeverity = "low"
	SeverityInfo     AlertSeverity = "info"
)

// AlertStatus represents the status of an alert
type AlertStatus string

const (
	AlertStatusActive       AlertStatus = "active"
	AlertStatusAcknowledged AlertStatus = "acknowledged"
	AlertStatusResolved     AlertStatus = "resolved"
	AlertStatusSuppressed   AlertStatus = "suppressed"
)

// EndpointHealth represents the health status of an endpoint
type EndpointHealth struct {
	EndpointID       string           `json:"endpoint_id"`
	TenantID         string           `json:"tenant_id"`
	URL              string           `json:"url"`
	CurrentStatus    string           `json:"current_status"` // healthy, degraded, unhealthy
	HealthScore      float64          `json:"health_score"`   // 0-100
	SuccessRate      float64          `json:"success_rate"`   // 0-1
	AverageLatencyMs float64          `json:"average_latency_ms"`
	P95LatencyMs     float64          `json:"p95_latency_ms"`
	P99LatencyMs     float64          `json:"p99_latency_ms"`
	ErrorRate        float64          `json:"error_rate"` // 0-1
	TotalRequests    int64            `json:"total_requests"`
	FailedRequests   int64            `json:"failed_requests"`
	LastChecked      time.Time        `json:"last_checked"`
	LastSuccess      *time.Time       `json:"last_success,omitempty"`
	LastFailure      *time.Time       `json:"last_failure,omitempty"`
	Metrics          []TimeSeriesPoint `json:"metrics,omitempty"`
}

// TimeSeriesPoint represents a point in a time series
type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// Prediction represents a failure prediction
type Prediction struct {
	ID              string            `json:"id"`
	TenantID        string            `json:"tenant_id"`
	EndpointID      string            `json:"endpoint_id"`
	Type            PredictionType    `json:"type"`
	Probability     float64           `json:"probability"`     // 0-1
	Confidence      float64           `json:"confidence"`      // 0-1
	PredictedAt     time.Time         `json:"predicted_at"`    // When prediction was made
	ExpectedTime    time.Time         `json:"expected_time"`   // When failure is expected
	TimeWindow      time.Duration     `json:"time_window"`     // Uncertainty window
	Severity        AlertSeverity     `json:"severity"`
	Description     string            `json:"description"`
	Factors         []PredictionFactor `json:"factors"`
	Recommendations []Recommendation  `json:"recommendations"`
	ModelVersion    string            `json:"model_version"`
	Outcome         *PredictionOutcome `json:"outcome,omitempty"` // Actual outcome
}

// PredictionFactor represents a contributing factor to a prediction
type PredictionFactor struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Weight      float64 `json:"weight"`     // How much this factor contributes
	Value       float64 `json:"value"`      // Current value
	Threshold   float64 `json:"threshold"`  // Threshold that triggered concern
	Trend       string  `json:"trend"`      // increasing, decreasing, stable
}

// Recommendation represents an action recommendation
type Recommendation struct {
	ID          string `json:"id"`
	Action      string `json:"action"`
	Description string `json:"description"`
	Impact      string `json:"impact"`   // high, medium, low
	Effort      string `json:"effort"`   // high, medium, low
	AutomateURL string `json:"automate_url,omitempty"` // API endpoint to automate
}

// PredictionOutcome records the actual outcome of a prediction
type PredictionOutcome struct {
	Occurred      bool       `json:"occurred"`
	ActualTime    *time.Time `json:"actual_time,omitempty"`
	WithinWindow  bool       `json:"within_window"`
	Notes         string     `json:"notes,omitempty"`
	RecordedAt    time.Time  `json:"recorded_at"`
}

// Alert represents a predictive alert
type Alert struct {
	ID              string          `json:"id"`
	TenantID        string          `json:"tenant_id"`
	EndpointID      string          `json:"endpoint_id,omitempty"`
	PredictionID    string          `json:"prediction_id,omitempty"`
	Type            PredictionType  `json:"type"`
	Severity        AlertSeverity   `json:"severity"`
	Status          AlertStatus     `json:"status"`
	Title           string          `json:"title"`
	Description     string          `json:"description"`
	Details         json.RawMessage `json:"details,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	AcknowledgedAt  *time.Time      `json:"acknowledged_at,omitempty"`
	AcknowledgedBy  string          `json:"acknowledged_by,omitempty"`
	ResolvedAt      *time.Time      `json:"resolved_at,omitempty"`
	ResolvedBy      string          `json:"resolved_by,omitempty"`
	ResolutionNotes string          `json:"resolution_notes,omitempty"`
	SuppressedUntil *time.Time      `json:"suppressed_until,omitempty"`
	NotificationsSent []NotificationRecord `json:"notifications_sent,omitempty"`
}

// NotificationRecord tracks sent notifications
type NotificationRecord struct {
	Channel   string    `json:"channel"` // email, slack, pagerduty, webhook
	Recipient string    `json:"recipient"`
	SentAt    time.Time `json:"sent_at"`
	Status    string    `json:"status"` // sent, delivered, failed
}

// MetricDataPoint represents a metric data point for ML training
type MetricDataPoint struct {
	Timestamp       time.Time         `json:"timestamp"`
	EndpointID      string            `json:"endpoint_id"`
	TenantID        string            `json:"tenant_id"`
	SuccessRate     float64           `json:"success_rate"`
	LatencyMs       float64           `json:"latency_ms"`
	ErrorRate       float64           `json:"error_rate"`
	ThroughputRPS   float64           `json:"throughput_rps"`
	StatusCodes     map[int]int       `json:"status_codes"`
	Features        map[string]float64 `json:"features,omitempty"`
}

// FailureEvent records a historical failure
type FailureEvent struct {
	ID           string          `json:"id"`
	EndpointID   string          `json:"endpoint_id"`
	TenantID     string          `json:"tenant_id"`
	Type         string          `json:"type"` // timeout, connection_refused, 5xx, etc.
	StartTime    time.Time       `json:"start_time"`
	EndTime      *time.Time      `json:"end_time,omitempty"`
	Duration     time.Duration   `json:"duration,omitempty"`
	ErrorCount   int64           `json:"error_count"`
	RootCause    string          `json:"root_cause,omitempty"`
	Resolution   string          `json:"resolution,omitempty"`
	MetricsBefore []MetricDataPoint `json:"metrics_before,omitempty"`
}

// Model represents a trained prediction model
type Model struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Type          string          `json:"type"` // random_forest, gradient_boost, lstm
	Version       string          `json:"version"`
	PredictionType PredictionType `json:"prediction_type"`
	Accuracy      float64         `json:"accuracy"`
	Precision     float64         `json:"precision"`
	Recall        float64         `json:"recall"`
	F1Score       float64         `json:"f1_score"`
	TrainedAt     time.Time       `json:"trained_at"`
	TrainingSamples int64         `json:"training_samples"`
	Features      []string        `json:"features"`
	Hyperparameters map[string]interface{} `json:"hyperparameters,omitempty"`
	Active        bool            `json:"active"`
	ModelData     []byte          `json:"-"` // Serialized model
}

// AlertRule defines when to generate alerts from predictions
type AlertRule struct {
	ID                  string         `json:"id"`
	TenantID            string         `json:"tenant_id"`
	Name                string         `json:"name"`
	Description         string         `json:"description,omitempty"`
	PredictionType      PredictionType `json:"prediction_type"`
	ProbabilityThreshold float64       `json:"probability_threshold"` // 0-1
	ConfidenceThreshold float64        `json:"confidence_threshold"`  // 0-1
	TimeWindowMinutes   int            `json:"time_window_minutes"`   // Alert if predicted within this window
	Severity            AlertSeverity  `json:"severity"`
	NotificationChannels []string      `json:"notification_channels"`
	Enabled             bool           `json:"enabled"`
	CooldownMinutes     int            `json:"cooldown_minutes"` // Prevent duplicate alerts
	Conditions          []RuleCondition `json:"conditions,omitempty"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
}

// RuleCondition represents an additional condition for alert rules
type RuleCondition struct {
	Field    string  `json:"field"`    // e.g., "health_score", "error_rate"
	Operator string  `json:"operator"` // lt, gt, eq, lte, gte
	Value    float64 `json:"value"`
}

// NotificationConfig defines how to send notifications
type NotificationConfig struct {
	ID         string            `json:"id"`
	TenantID   string            `json:"tenant_id"`
	Channel    string            `json:"channel"` // email, slack, pagerduty, webhook
	Enabled    bool              `json:"enabled"`
	Config     map[string]string `json:"config"` // Channel-specific config
	Severities []AlertSeverity   `json:"severities"` // Which severities to notify
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

// Request/Response types

// GetHealthRequest represents a health check request
type GetHealthRequest struct {
	EndpointID string `json:"endpoint_id"`
	Period     string `json:"period"` // 1h, 6h, 24h, 7d
}

// PredictRequest represents a prediction request
type PredictRequest struct {
	EndpointID    string   `json:"endpoint_id" binding:"required"`
	Types         []PredictionType `json:"types,omitempty"` // Empty = all types
	TimeHorizon   string   `json:"time_horizon"` // 1h, 6h, 24h
}

// CreateAlertRuleRequest represents a request to create an alert rule
type CreateAlertRuleRequest struct {
	Name                 string         `json:"name" binding:"required"`
	Description          string         `json:"description,omitempty"`
	PredictionType       PredictionType `json:"prediction_type" binding:"required"`
	ProbabilityThreshold float64        `json:"probability_threshold" binding:"required"`
	ConfidenceThreshold  float64        `json:"confidence_threshold"`
	TimeWindowMinutes    int            `json:"time_window_minutes"`
	Severity             AlertSeverity  `json:"severity" binding:"required"`
	NotificationChannels []string       `json:"notification_channels"`
	CooldownMinutes      int            `json:"cooldown_minutes"`
	Conditions           []RuleCondition `json:"conditions,omitempty"`
}

// UpdateAlertRequest represents a request to update an alert
type UpdateAlertRequest struct {
	Status          *AlertStatus `json:"status,omitempty"`
	ResolutionNotes string       `json:"resolution_notes,omitempty"`
	SuppressMinutes int          `json:"suppress_minutes,omitempty"`
}

// AlertFilters for querying alerts
type AlertFilters struct {
	EndpointID  string        `json:"endpoint_id,omitempty"`
	Type        *PredictionType `json:"type,omitempty"`
	Severity    *AlertSeverity `json:"severity,omitempty"`
	Status      *AlertStatus   `json:"status,omitempty"`
	Since       time.Time      `json:"since,omitempty"`
	Page        int            `json:"page"`
	PageSize    int            `json:"page_size"`
}

// PredictionFilters for querying predictions
type PredictionFilters struct {
	EndpointID   string         `json:"endpoint_id,omitempty"`
	Type         *PredictionType `json:"type,omitempty"`
	MinProbability float64      `json:"min_probability,omitempty"`
	Since        time.Time      `json:"since,omitempty"`
	Page         int            `json:"page"`
	PageSize     int            `json:"page_size"`
}

// ListAlertsResponse represents paginated alerts
type ListAlertsResponse struct {
	Alerts     []Alert `json:"alerts"`
	Total      int     `json:"total"`
	Page       int     `json:"page"`
	PageSize   int     `json:"page_size"`
	TotalPages int     `json:"total_pages"`
}

// ListPredictionsResponse represents paginated predictions
type ListPredictionsResponse struct {
	Predictions []Prediction `json:"predictions"`
	Total       int          `json:"total"`
	Page        int          `json:"page"`
	PageSize    int          `json:"page_size"`
	TotalPages  int          `json:"total_pages"`
}

// DashboardStats represents prediction dashboard statistics
type DashboardStats struct {
	TotalEndpoints      int               `json:"total_endpoints"`
	HealthyEndpoints    int               `json:"healthy_endpoints"`
	DegradedEndpoints   int               `json:"degraded_endpoints"`
	UnhealthyEndpoints  int               `json:"unhealthy_endpoints"`
	ActiveAlerts        int               `json:"active_alerts"`
	CriticalAlerts      int               `json:"critical_alerts"`
	PredictionsLast24h  int               `json:"predictions_last_24h"`
	PreventedFailures   int               `json:"prevented_failures"`
	ModelAccuracy       float64           `json:"model_accuracy"`
	TopRiskyEndpoints   []EndpointHealth  `json:"top_risky_endpoints"`
	RecentPredictions   []Prediction      `json:"recent_predictions"`
}

// DeliverySuccessPrediction represents a delivery success prediction
type DeliverySuccessPrediction struct {
	EndpointID  string    `json:"endpoint_id"`
	Probability float64   `json:"probability"`
	Confidence  float64   `json:"confidence"`
	Factors     []string  `json:"factors"`
	PredictedAt time.Time `json:"predicted_at"`
}

// LatencyPrediction represents a predicted latency for an endpoint
type LatencyPrediction struct {
	EndpointID         string    `json:"endpoint_id"`
	PredictedLatencyMs float64   `json:"predicted_latency_ms"`
	P95LatencyMs       float64   `json:"p95_latency_ms"`
	P99LatencyMs       float64   `json:"p99_latency_ms"`
	Confidence         float64   `json:"confidence"`
	Variance           float64   `json:"variance"`
	PredictedAt        time.Time `json:"predicted_at"`
}

// ReliabilityScore represents an overall reliability metric for an endpoint
type ReliabilityScore struct {
	EndpointID   string                `json:"endpoint_id"`
	OverallScore float64               `json:"overall_score"` // 0-100
	Grade        string                `json:"grade"`         // A, B, C, D, F
	Components   ReliabilityComponents `json:"components"`
	CalculatedAt time.Time             `json:"calculated_at"`
}

// ReliabilityComponents breaks down the reliability score
type ReliabilityComponents struct {
	SuccessRateScore  float64 `json:"success_rate_score"`
	LatencyScore      float64 `json:"latency_score"`
	AvailabilityScore float64 `json:"availability_score"`
}
