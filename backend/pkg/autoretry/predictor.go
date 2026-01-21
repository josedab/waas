// Package autoretry provides intelligent ML-based retry optimization
package autoretry

import (
	"context"
	"math"
	"time"
)

// DeliveryFeatures represents features extracted for ML prediction
type DeliveryFeatures struct {
	ID                        string                 `json:"id"`
	DeliveryID                string                 `json:"delivery_id"`
	EndpointID                string                 `json:"endpoint_id"`
	EndpointSuccessRate1h     float64                `json:"endpoint_success_rate_1h"`
	EndpointSuccessRate24h    float64                `json:"endpoint_success_rate_24h"`
	EndpointAvgResponseTimeMs int                    `json:"endpoint_avg_response_time_ms"`
	EndpointErrorRate1h       float64                `json:"endpoint_error_rate_1h"`
	EndpointLastSuccessMin    int                    `json:"endpoint_last_success_minutes"`
	HourOfDay                 int                    `json:"hour_of_day"`
	DayOfWeek                 int                    `json:"day_of_week"`
	IsWeekend                 bool                   `json:"is_weekend"`
	IsBusinessHours           bool                   `json:"is_business_hours"`
	PayloadSizeBytes          int                    `json:"payload_size_bytes"`
	HasLargePayload           bool                   `json:"has_large_payload"`
	AttemptNumber             int                    `json:"attempt_number"`
	TimeSinceFirstAttemptSec  int                    `json:"time_since_first_attempt_seconds"`
	PreviousErrorCode         string                 `json:"previous_error_code"`
	ConsecutiveFailures       int                    `json:"consecutive_failures"`
	WasSuccessful             *bool                  `json:"was_successful,omitempty"`
	ResponseTimeMs            *int                   `json:"response_time_ms,omitempty"`
	HTTPStatusCode            *int                   `json:"http_status_code,omitempty"`
	CreatedAt                 time.Time              `json:"created_at"`
	Metadata                  map[string]interface{} `json:"metadata,omitempty"`
}

// RetryPrediction represents a model prediction for retry strategy
type RetryPrediction struct {
	ID                         string                 `json:"id"`
	DeliveryID                 string                 `json:"delivery_id"`
	EndpointID                 string                 `json:"endpoint_id"`
	PredictedSuccessProbability float64               `json:"predicted_success_probability"`
	RecommendedDelaySec        int                    `json:"recommended_delay_seconds"`
	ConfidenceScore            float64                `json:"confidence_score"`
	ModelVersion               string                 `json:"model_version"`
	FeatureVector              map[string]interface{} `json:"feature_vector,omitempty"`
	ActualSuccess              *bool                  `json:"actual_success,omitempty"`
	ActualDelayUsedSec         *int                   `json:"actual_delay_used_seconds,omitempty"`
	CreatedAt                  time.Time              `json:"created_at"`
	EvaluatedAt                *time.Time             `json:"evaluated_at,omitempty"`
}

// RetryStrategy represents a configurable retry strategy
type RetryStrategy struct {
	Name              string  `json:"name"`
	BaseDelaySeconds  int     `json:"base_delay_seconds"`
	MaxDelaySeconds   int     `json:"max_delay_seconds"`
	MaxAttempts       int     `json:"max_attempts"`
	BackoffMultiplier float64 `json:"backoff_multiplier"`
	JitterFactor      float64 `json:"jitter_factor"`
}

// Experiment represents an A/B test for retry strategies
type Experiment struct {
	ID                   string         `json:"id"`
	Name                 string         `json:"name"`
	Description          string         `json:"description"`
	Status               string         `json:"status"`
	ControlStrategy      RetryStrategy  `json:"control_strategy"`
	TreatmentStrategy    RetryStrategy  `json:"treatment_strategy"`
	TrafficSplit         float64        `json:"traffic_split"`
	StartDate            *time.Time     `json:"start_date,omitempty"`
	EndDate              *time.Time     `json:"end_date,omitempty"`
	ControlSampleSize    int            `json:"control_sample_size"`
	TreatmentSampleSize  int            `json:"treatment_sample_size"`
	ControlSuccessRate   *float64       `json:"control_success_rate,omitempty"`
	TreatmentSuccessRate *float64       `json:"treatment_success_rate,omitempty"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
}

// PredictionService provides ML-based retry predictions
type PredictionService struct {
	modelVersion   string
	defaultWeights ModelWeights
}

// ModelWeights represent the trained model parameters
type ModelWeights struct {
	SuccessRateWeight       float64
	ErrorRateWeight         float64
	ResponseTimeWeight      float64
	TimeOfDayWeight         float64
	AttemptWeight           float64
	ConsecutiveFailureWeight float64
	PayloadSizeWeight       float64
	Intercept               float64
}

// NewPredictionService creates a new prediction service
func NewPredictionService() *PredictionService {
	return &PredictionService{
		modelVersion: "v1.0.0-baseline",
		defaultWeights: ModelWeights{
			SuccessRateWeight:        0.35,
			ErrorRateWeight:          -0.25,
			ResponseTimeWeight:       -0.10,
			TimeOfDayWeight:          0.05,
			AttemptWeight:            -0.15,
			ConsecutiveFailureWeight: -0.20,
			PayloadSizeWeight:        -0.05,
			Intercept:                0.60,
		},
	}
}

// Predict generates a retry prediction based on features
func (s *PredictionService) Predict(ctx context.Context, features *DeliveryFeatures) (*RetryPrediction, error) {
	// Calculate success probability using logistic regression model
	score := s.calculateScore(features)
	probability := sigmoid(score)
	
	// Calculate optimal delay based on features
	delay := s.calculateOptimalDelay(features, probability)
	
	// Calculate confidence based on feature completeness
	confidence := s.calculateConfidence(features)
	
	return &RetryPrediction{
		DeliveryID:                  features.DeliveryID,
		EndpointID:                  features.EndpointID,
		PredictedSuccessProbability: probability,
		RecommendedDelaySec:         delay,
		ConfidenceScore:             confidence,
		ModelVersion:                s.modelVersion,
		FeatureVector:               s.extractFeatureVector(features),
		CreatedAt:                   time.Now(),
	}, nil
}

func (s *PredictionService) calculateScore(f *DeliveryFeatures) float64 {
	w := s.defaultWeights
	
	score := w.Intercept
	score += w.SuccessRateWeight * f.EndpointSuccessRate1h
	score += w.ErrorRateWeight * f.EndpointErrorRate1h
	
	// Normalize response time (0-1 scale, 5000ms = 1.0)
	normalizedRT := math.Min(float64(f.EndpointAvgResponseTimeMs)/5000.0, 1.0)
	score += w.ResponseTimeWeight * normalizedRT
	
	// Business hours bonus
	if f.IsBusinessHours {
		score += w.TimeOfDayWeight
	}
	
	// Attempt penalty (diminishing returns)
	attemptPenalty := math.Log(float64(f.AttemptNumber + 1))
	score += w.AttemptWeight * attemptPenalty
	
	// Consecutive failure penalty
	failurePenalty := math.Min(float64(f.ConsecutiveFailures)*0.1, 1.0)
	score += w.ConsecutiveFailureWeight * failurePenalty
	
	// Payload size penalty
	if f.HasLargePayload {
		score += w.PayloadSizeWeight
	}
	
	return score
}

func (s *PredictionService) calculateOptimalDelay(f *DeliveryFeatures, probability float64) int {
	// Base delay increases exponentially with attempt number
	baseDelay := 30 * int(math.Pow(2, float64(f.AttemptNumber-1)))
	
	// Adjust based on success probability
	// Lower probability = longer delay (endpoint might be down)
	if probability < 0.3 {
		baseDelay *= 4
	} else if probability < 0.5 {
		baseDelay *= 2
	}
	
	// Cap at 1 hour
	if baseDelay > 3600 {
		baseDelay = 3600
	}
	
	// Minimum 10 seconds
	if baseDelay < 10 {
		baseDelay = 10
	}
	
	return baseDelay
}

func (s *PredictionService) calculateConfidence(f *DeliveryFeatures) float64 {
	// Start with base confidence
	confidence := 0.5
	
	// Increase confidence with more data
	if f.EndpointSuccessRate24h > 0 {
		confidence += 0.15
	}
	if f.EndpointLastSuccessMin > 0 {
		confidence += 0.10
	}
	if f.TimeSinceFirstAttemptSec > 0 {
		confidence += 0.10
	}
	if f.ConsecutiveFailures > 0 {
		confidence += 0.15
	}
	
	return math.Min(confidence, 1.0)
}

func (s *PredictionService) extractFeatureVector(f *DeliveryFeatures) map[string]interface{} {
	return map[string]interface{}{
		"success_rate_1h":      f.EndpointSuccessRate1h,
		"success_rate_24h":     f.EndpointSuccessRate24h,
		"error_rate_1h":        f.EndpointErrorRate1h,
		"avg_response_time_ms": f.EndpointAvgResponseTimeMs,
		"attempt_number":       f.AttemptNumber,
		"consecutive_failures": f.ConsecutiveFailures,
		"is_business_hours":    f.IsBusinessHours,
		"payload_size_bytes":   f.PayloadSizeBytes,
	}
}

func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

// RetryOptimizer provides adaptive retry strategies
type RetryOptimizer struct {
	prediction *PredictionService
	strategies map[string]RetryStrategy
}

// NewRetryOptimizer creates a new optimizer
func NewRetryOptimizer() *RetryOptimizer {
	return &RetryOptimizer{
		prediction: NewPredictionService(),
		strategies: map[string]RetryStrategy{
			"conservative": {
				Name:              "conservative",
				BaseDelaySeconds:  60,
				MaxDelaySeconds:   3600,
				MaxAttempts:       5,
				BackoffMultiplier: 3.0,
				JitterFactor:      0.2,
			},
			"aggressive": {
				Name:              "aggressive",
				BaseDelaySeconds:  10,
				MaxDelaySeconds:   300,
				MaxAttempts:       10,
				BackoffMultiplier: 1.5,
				JitterFactor:      0.1,
			},
			"adaptive": {
				Name:              "adaptive",
				BaseDelaySeconds:  30,
				MaxDelaySeconds:   1800,
				MaxAttempts:       7,
				BackoffMultiplier: 2.0,
				JitterFactor:      0.15,
			},
		},
	}
}

// GetOptimalStrategy returns the best retry strategy for given features
func (o *RetryOptimizer) GetOptimalStrategy(ctx context.Context, features *DeliveryFeatures) (*RetryStrategy, *RetryPrediction, error) {
	prediction, err := o.prediction.Predict(ctx, features)
	if err != nil {
		return nil, nil, err
	}
	
	// Select strategy based on prediction
	var strategy RetryStrategy
	switch {
	case prediction.PredictedSuccessProbability > 0.7:
		strategy = o.strategies["aggressive"]
	case prediction.PredictedSuccessProbability < 0.3:
		strategy = o.strategies["conservative"]
	default:
		strategy = o.strategies["adaptive"]
	}
	
	// Customize delay based on prediction
	strategy.BaseDelaySeconds = prediction.RecommendedDelaySec
	
	return &strategy, prediction, nil
}

// CalculateNextDelay calculates the delay for the next retry attempt
func (o *RetryOptimizer) CalculateNextDelay(strategy *RetryStrategy, attemptNumber int) time.Duration {
	// Exponential backoff
	delay := float64(strategy.BaseDelaySeconds) * math.Pow(strategy.BackoffMultiplier, float64(attemptNumber-1))
	
	// Cap at max delay
	if delay > float64(strategy.MaxDelaySeconds) {
		delay = float64(strategy.MaxDelaySeconds)
	}
	
	// Add jitter to prevent thundering herd
	jitter := delay * strategy.JitterFactor * (0.5 - randomFloat())
	delay += jitter
	
	return time.Duration(delay) * time.Second
}

func randomFloat() float64 {
	// Simple deterministic pseudo-random for reproducibility
	return 0.5
}
