package prediction

import (
	"context"
	"errors"
	"time"
)

var (
	ErrEndpointNotFound    = errors.New("endpoint not found")
	ErrPredictionNotFound  = errors.New("prediction not found")
	ErrAlertNotFound       = errors.New("alert not found")
	ErrAlertRuleNotFound   = errors.New("alert rule not found")
	ErrModelNotFound       = errors.New("model not found")
	ErrInsufficientData    = errors.New("insufficient data for prediction")
	ErrModelNotReady       = errors.New("model not ready for predictions")
)

// Repository defines the interface for prediction data storage
type Repository interface {
	// Endpoint Health
	SaveEndpointHealth(ctx context.Context, health *EndpointHealth) error
	GetEndpointHealth(ctx context.Context, tenantID, endpointID string) (*EndpointHealth, error)
	ListEndpointHealth(ctx context.Context, tenantID string) ([]EndpointHealth, error)
	GetHealthHistory(ctx context.Context, endpointID string, since time.Time) ([]EndpointHealth, error)

	// Metrics
	SaveMetrics(ctx context.Context, dataPoint *MetricDataPoint) error
	GetMetrics(ctx context.Context, endpointID string, since time.Time, resolution string) ([]MetricDataPoint, error)
	GetAggregatedMetrics(ctx context.Context, endpointID string, period string) (*MetricDataPoint, error)

	// Predictions
	SavePrediction(ctx context.Context, prediction *Prediction) error
	GetPrediction(ctx context.Context, predictionID string) (*Prediction, error)
	UpdatePrediction(ctx context.Context, prediction *Prediction) error
	ListPredictions(ctx context.Context, tenantID string, filters *PredictionFilters) ([]Prediction, int, error)
	GetActivePredictions(ctx context.Context, tenantID string) ([]Prediction, error)

	// Alerts
	CreateAlert(ctx context.Context, alert *Alert) error
	GetAlert(ctx context.Context, alertID string) (*Alert, error)
	UpdateAlert(ctx context.Context, alert *Alert) error
	ListAlerts(ctx context.Context, tenantID string, filters *AlertFilters) ([]Alert, int, error)
	GetActiveAlerts(ctx context.Context, tenantID string) ([]Alert, error)
	GetAlertsByEndpoint(ctx context.Context, tenantID, endpointID string) ([]Alert, error)
	GetLastAlertTime(ctx context.Context, tenantID, endpointID string, predictionType PredictionType) (*time.Time, error)

	// Alert Rules
	CreateAlertRule(ctx context.Context, rule *AlertRule) error
	GetAlertRule(ctx context.Context, tenantID, ruleID string) (*AlertRule, error)
	UpdateAlertRule(ctx context.Context, rule *AlertRule) error
	DeleteAlertRule(ctx context.Context, tenantID, ruleID string) error
	ListAlertRules(ctx context.Context, tenantID string) ([]AlertRule, error)
	GetEnabledRules(ctx context.Context, tenantID string, predictionType PredictionType) ([]AlertRule, error)

	// Failure Events
	SaveFailureEvent(ctx context.Context, event *FailureEvent) error
	GetFailureEvents(ctx context.Context, endpointID string, since time.Time) ([]FailureEvent, error)
	GetRecentFailures(ctx context.Context, tenantID string, limit int) ([]FailureEvent, error)

	// Models
	SaveModel(ctx context.Context, model *Model) error
	GetModel(ctx context.Context, modelID string) (*Model, error)
	GetActiveModel(ctx context.Context, predictionType PredictionType) (*Model, error)
	ListModels(ctx context.Context) ([]Model, error)
	SetActiveModel(ctx context.Context, modelID string) error

	// Notification Config
	SaveNotificationConfig(ctx context.Context, config *NotificationConfig) error
	GetNotificationConfig(ctx context.Context, tenantID, channel string) (*NotificationConfig, error)
	ListNotificationConfigs(ctx context.Context, tenantID string) ([]NotificationConfig, error)
	DeleteNotificationConfig(ctx context.Context, tenantID, configID string) error

	// Statistics
	GetDashboardStats(ctx context.Context, tenantID string) (*DashboardStats, error)
	GetPredictionAccuracy(ctx context.Context, since time.Time) (float64, error)
}

// Predictor defines the interface for making predictions
type Predictor interface {
	// Predict makes a prediction for an endpoint
	Predict(ctx context.Context, endpointID string, predictionType PredictionType, metrics []MetricDataPoint) (*Prediction, error)

	// PredictAll makes predictions for all types
	PredictAll(ctx context.Context, endpointID string, metrics []MetricDataPoint) ([]Prediction, error)

	// GetModelInfo returns information about the prediction model
	GetModelInfo() *Model

	// IsReady returns whether the predictor is ready
	IsReady() bool
}

// FeatureExtractor defines the interface for extracting features from metrics
type FeatureExtractor interface {
	// ExtractFeatures extracts ML features from metrics
	ExtractFeatures(metrics []MetricDataPoint) (map[string]float64, error)

	// GetFeatureNames returns the names of features
	GetFeatureNames() []string
}

// ModelTrainer defines the interface for training prediction models
type ModelTrainer interface {
	// Train trains a model with the given data
	Train(ctx context.Context, trainingData []TrainingExample) (*Model, error)

	// Evaluate evaluates a model's performance
	Evaluate(ctx context.Context, model *Model, testData []TrainingExample) (*ModelMetrics, error)

	// GetSupportedTypes returns supported prediction types
	GetSupportedTypes() []PredictionType
}

// TrainingExample represents a training data point
type TrainingExample struct {
	Features map[string]float64 `json:"features"`
	Label    bool              `json:"label"` // true = failure occurred
	Timestamp time.Time        `json:"timestamp"`
	EndpointID string          `json:"endpoint_id"`
}

// ModelMetrics represents model evaluation metrics
type ModelMetrics struct {
	Accuracy   float64 `json:"accuracy"`
	Precision  float64 `json:"precision"`
	Recall     float64 `json:"recall"`
	F1Score    float64 `json:"f1_score"`
	AUC        float64 `json:"auc"`
	Samples    int64   `json:"samples"`
}

// Notifier defines the interface for sending notifications
type Notifier interface {
	// SendAlert sends an alert notification
	SendAlert(ctx context.Context, alert *Alert, config *NotificationConfig) error

	// SendPrediction sends a prediction notification
	SendPrediction(ctx context.Context, prediction *Prediction, config *NotificationConfig) error

	// TestConnection tests a notification channel
	TestConnection(ctx context.Context, config *NotificationConfig) error
}

// MetricsCollector defines the interface for collecting endpoint metrics
type MetricsCollector interface {
	// Collect collects metrics for an endpoint
	Collect(ctx context.Context, endpointID string) (*MetricDataPoint, error)

	// CollectAll collects metrics for all endpoints
	CollectAll(ctx context.Context, tenantID string) ([]MetricDataPoint, error)

	// Start starts continuous collection
	Start(ctx context.Context, interval time.Duration) error

	// Stop stops continuous collection
	Stop() error
}

// HealthCalculator defines the interface for calculating endpoint health
type HealthCalculator interface {
	// Calculate calculates health from metrics
	Calculate(metrics []MetricDataPoint) (*EndpointHealth, error)

	// GetThresholds returns the health thresholds
	GetThresholds() HealthThresholds
}

// HealthThresholds defines health calculation thresholds
type HealthThresholds struct {
	HealthySuccessRate    float64 `json:"healthy_success_rate"`    // e.g., 0.99
	DegradedSuccessRate   float64 `json:"degraded_success_rate"`   // e.g., 0.95
	HealthyLatencyMs      float64 `json:"healthy_latency_ms"`      // e.g., 500
	DegradedLatencyMs     float64 `json:"degraded_latency_ms"`     // e.g., 2000
	HealthyErrorRate      float64 `json:"healthy_error_rate"`      // e.g., 0.01
	DegradedErrorRate     float64 `json:"degraded_error_rate"`     // e.g., 0.05
}

// DefaultHealthThresholds returns default health thresholds
func DefaultHealthThresholds() HealthThresholds {
	return HealthThresholds{
		HealthySuccessRate:  0.99,
		DegradedSuccessRate: 0.95,
		HealthyLatencyMs:    500,
		DegradedLatencyMs:   2000,
		HealthyErrorRate:    0.01,
		DegradedErrorRate:   0.05,
	}
}
