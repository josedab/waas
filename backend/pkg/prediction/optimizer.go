package prediction

import (
	"math"
	"sort"
	"time"
)

// DeliveryOptimizer uses historical delivery data to optimize retry scheduling
type DeliveryOptimizer struct {
	endpoints map[string]*EndpointProfile
	config    *OptimizerConfig
}

// OptimizerConfig configures the optimizer behavior
type OptimizerConfig struct {
	MinSamplesForOptimization int
	MaxRetryDelaySeconds      int
	BaseRetryDelaySeconds     int
	SuccessRateThreshold      float64 // Below this, mark endpoint as unhealthy
	LearningRate              float64
}

// DefaultOptimizerConfig returns default configuration
func DefaultOptimizerConfig() *OptimizerConfig {
	return &OptimizerConfig{
		MinSamplesForOptimization: 10,
		MaxRetryDelaySeconds:      3600,
		BaseRetryDelaySeconds:     5,
		SuccessRateThreshold:      0.8,
		LearningRate:              0.1,
	}
}

// EndpointProfile stores learned behavior for a specific endpoint
type EndpointProfile struct {
	EndpointID         string                 `json:"endpoint_id"`
	TotalAttempts      int64                  `json:"total_attempts"`
	SuccessCount       int64                  `json:"success_count"`
	FailureCount       int64                  `json:"failure_count"`
	AvgResponseTimeMs  float64                `json:"avg_response_time_ms"`
	P95ResponseTimeMs  float64                `json:"p95_response_time_ms"`
	SuccessRateByHour  [24]float64            `json:"success_rate_by_hour"`
	SamplesByHour      [24]int64              `json:"samples_by_hour"`
	AvgRetriesToSucceed float64               `json:"avg_retries_to_succeed"`
	LastFailurePattern  string                `json:"last_failure_pattern"`
	OptimalRetryDelays  []int                 `json:"optimal_retry_delays"`
	RecommendedTimeout  int                   `json:"recommended_timeout_ms"`
	HealthScore         float64               `json:"health_score"`
	LastUpdated         time.Time             `json:"last_updated"`
	FailureCategories   map[string]int64      `json:"failure_categories"`
	ResponseTimeBuckets []ResponseTimeBucket  `json:"response_time_buckets"`
}

// ResponseTimeBucket tracks response time distribution
type ResponseTimeBucket struct {
	RangeMinMs int     `json:"range_min_ms"`
	RangeMaxMs int     `json:"range_max_ms"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}

// DeliveryOutcome represents a historical delivery result
type DeliveryOutcome struct {
	EndpointID     string    `json:"endpoint_id"`
	Success        bool      `json:"success"`
	ResponseTimeMs int       `json:"response_time_ms"`
	StatusCode     int       `json:"status_code"`
	RetryAttempt   int       `json:"retry_attempt"`
	ErrorCategory  string    `json:"error_category"`
	Timestamp      time.Time `json:"timestamp"`
}

// RetryRecommendation is the optimizer's suggestion for retry behavior
type RetryRecommendation struct {
	EndpointID          string    `json:"endpoint_id"`
	ShouldRetry         bool      `json:"should_retry"`
	RecommendedDelayMs  int       `json:"recommended_delay_ms"`
	MaxRetries          int       `json:"max_retries"`
	PredictedSuccessRate float64  `json:"predicted_success_rate"`
	OptimalTimeWindow   *TimeWindow `json:"optimal_time_window,omitempty"`
	Reasoning           string    `json:"reasoning"`
	Confidence          float64   `json:"confidence"`
}

// TimeWindow represents a recommended delivery window
type TimeWindow struct {
	StartHourUTC int     `json:"start_hour_utc"`
	EndHourUTC   int     `json:"end_hour_utc"`
	SuccessRate  float64 `json:"success_rate"`
}

// AnomalyReport represents a detected anomaly in delivery patterns
type AnomalyReport struct {
	EndpointID  string    `json:"endpoint_id"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"` // low, medium, high, critical
	Description string    `json:"description"`
	Metric      string    `json:"metric"`
	Expected    float64   `json:"expected"`
	Actual      float64   `json:"actual"`
	Deviation   float64   `json:"deviation"`
	DetectedAt  time.Time `json:"detected_at"`
}

// NewDeliveryOptimizer creates a new optimizer
func NewDeliveryOptimizer(config *OptimizerConfig) *DeliveryOptimizer {
	if config == nil {
		config = DefaultOptimizerConfig()
	}
	return &DeliveryOptimizer{
		endpoints: make(map[string]*EndpointProfile),
		config:    config,
	}
}

// RecordOutcome records a delivery outcome and updates the endpoint profile
func (o *DeliveryOptimizer) RecordOutcome(outcome DeliveryOutcome) {
	profile, ok := o.endpoints[outcome.EndpointID]
	if !ok {
		profile = &EndpointProfile{
			EndpointID:        outcome.EndpointID,
			FailureCategories: make(map[string]int64),
		}
		o.endpoints[outcome.EndpointID] = profile
	}

	profile.TotalAttempts++
	if outcome.Success {
		profile.SuccessCount++
	} else {
		profile.FailureCount++
		if outcome.ErrorCategory != "" {
			profile.FailureCategories[outcome.ErrorCategory]++
		}
	}

	// Update exponential moving average for response time
	alpha := o.config.LearningRate
	if profile.AvgResponseTimeMs == 0 {
		profile.AvgResponseTimeMs = float64(outcome.ResponseTimeMs)
	} else {
		profile.AvgResponseTimeMs = alpha*float64(outcome.ResponseTimeMs) + (1-alpha)*profile.AvgResponseTimeMs
	}

	// Update hourly success rate
	hour := outcome.Timestamp.UTC().Hour()
	profile.SamplesByHour[hour]++
	if outcome.Success {
		total := float64(profile.SamplesByHour[hour])
		old := profile.SuccessRateByHour[hour]
		profile.SuccessRateByHour[hour] = old + (1.0-old)/total
	} else {
		total := float64(profile.SamplesByHour[hour])
		old := profile.SuccessRateByHour[hour]
		profile.SuccessRateByHour[hour] = old - old/total
	}

	profile.HealthScore = float64(profile.SuccessCount) / float64(profile.TotalAttempts) * 100
	profile.LastUpdated = time.Now()
}

// GetRecommendation returns a retry recommendation for an endpoint
func (o *DeliveryOptimizer) GetRecommendation(endpointID string, currentAttempt int) RetryRecommendation {
	profile, ok := o.endpoints[endpointID]
	if !ok {
		return RetryRecommendation{
			EndpointID:          endpointID,
			ShouldRetry:         true,
			RecommendedDelayMs:  o.config.BaseRetryDelaySeconds * 1000,
			MaxRetries:          5,
			PredictedSuccessRate: 0.5,
			Reasoning:           "no historical data available, using default retry strategy",
			Confidence:          0.1,
		}
	}

	successRate := float64(profile.SuccessCount) / float64(profile.TotalAttempts)

	// Find optimal time window
	var bestWindow *TimeWindow
	bestRate := 0.0
	for h := 0; h < 24; h++ {
		if profile.SamplesByHour[h] >= 5 && profile.SuccessRateByHour[h] > bestRate {
			bestRate = profile.SuccessRateByHour[h]
			bestWindow = &TimeWindow{
				StartHourUTC: h,
				EndHourUTC:   (h + 1) % 24,
				SuccessRate:  profile.SuccessRateByHour[h],
			}
		}
	}

	// Calculate optimal retry delay using learned patterns
	delay := o.calculateOptimalDelay(profile, currentAttempt)
	maxRetries := o.calculateMaxRetries(profile)

	// Predict success rate for next attempt
	predictedSuccess := o.predictSuccessRate(profile, currentAttempt)
	if successRate < 0.1 && currentAttempt > 5 {
		predictedSuccess = 0.01
	}

	shouldRetry := predictedSuccess > 0.05 && currentAttempt < maxRetries

	reasoning := o.generateReasoning(profile, currentAttempt, predictedSuccess, shouldRetry)
	confidence := math.Min(1.0, float64(profile.TotalAttempts)/float64(o.config.MinSamplesForOptimization*10))

	return RetryRecommendation{
		EndpointID:          endpointID,
		ShouldRetry:         shouldRetry,
		RecommendedDelayMs:  delay,
		MaxRetries:          maxRetries,
		PredictedSuccessRate: math.Round(predictedSuccess*1000) / 1000,
		OptimalTimeWindow:   bestWindow,
		Reasoning:           reasoning,
		Confidence:          math.Round(confidence*100) / 100,
	}
}

// DetectAnomalies checks for anomalies in an endpoint's behavior
func (o *DeliveryOptimizer) DetectAnomalies(endpointID string) []AnomalyReport {
	profile, ok := o.endpoints[endpointID]
	if !ok || profile.TotalAttempts < int64(o.config.MinSamplesForOptimization) {
		return nil
	}

	var anomalies []AnomalyReport
	now := time.Now()
	overallSuccessRate := float64(profile.SuccessCount) / float64(profile.TotalAttempts)

	// Check if success rate dropped significantly
	if overallSuccessRate < o.config.SuccessRateThreshold {
		anomalies = append(anomalies, AnomalyReport{
			EndpointID:  endpointID,
			Type:        "success_rate_drop",
			Severity:    severityFromRate(overallSuccessRate),
			Description: "endpoint success rate below threshold",
			Metric:      "success_rate",
			Expected:    o.config.SuccessRateThreshold,
			Actual:      overallSuccessRate,
			Deviation:   o.config.SuccessRateThreshold - overallSuccessRate,
			DetectedAt:  now,
		})
	}

	// Check for response time anomaly (> 2x average)
	if profile.P95ResponseTimeMs > 0 && profile.AvgResponseTimeMs > profile.P95ResponseTimeMs*2 {
		anomalies = append(anomalies, AnomalyReport{
			EndpointID:  endpointID,
			Type:        "response_time_spike",
			Severity:    "medium",
			Description: "response time significantly higher than P95",
			Metric:      "response_time_ms",
			Expected:    profile.P95ResponseTimeMs,
			Actual:      profile.AvgResponseTimeMs,
			Deviation:   profile.AvgResponseTimeMs - profile.P95ResponseTimeMs,
			DetectedAt:  now,
		})
	}

	// Check for dominant failure category
	if len(profile.FailureCategories) > 0 {
		maxCategory := ""
		maxCount := int64(0)
		for cat, count := range profile.FailureCategories {
			if count > maxCount {
				maxCategory = cat
				maxCount = count
			}
		}
		if float64(maxCount)/float64(profile.FailureCount) > 0.7 {
			anomalies = append(anomalies, AnomalyReport{
				EndpointID:  endpointID,
				Type:        "dominant_failure_pattern",
				Severity:    "high",
				Description: "single failure category dominates: " + maxCategory,
				Metric:      "failure_category",
				Expected:    0,
				Actual:      float64(maxCount),
				DetectedAt:  now,
			})
		}
	}

	return anomalies
}

// GetProfile returns the learned profile for an endpoint
func (o *DeliveryOptimizer) GetProfile(endpointID string) *EndpointProfile {
	return o.endpoints[endpointID]
}

// ListProfiles returns all endpoint profiles sorted by health score
func (o *DeliveryOptimizer) ListProfiles() []*EndpointProfile {
	profiles := make([]*EndpointProfile, 0, len(o.endpoints))
	for _, p := range o.endpoints {
		profiles = append(profiles, p)
	}
	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].HealthScore > profiles[j].HealthScore
	})
	return profiles
}

func (o *DeliveryOptimizer) calculateOptimalDelay(profile *EndpointProfile, attempt int) int {
	base := o.config.BaseRetryDelaySeconds * 1000

	// Exponential backoff with learned modifier
	backoff := float64(base) * math.Pow(2, float64(attempt))

	// Reduce delay for high-success endpoints, increase for low-success
	successRate := float64(profile.SuccessCount) / float64(profile.TotalAttempts)
	modifier := 1.0
	if successRate > 0.9 {
		modifier = 0.5 // Retry faster for usually-healthy endpoints
	} else if successRate < 0.5 {
		modifier = 2.0 // Wait longer for struggling endpoints
	}

	delay := int(backoff * modifier)
	maxDelay := o.config.MaxRetryDelaySeconds * 1000
	if delay > maxDelay {
		delay = maxDelay
	}
	return delay
}

func (o *DeliveryOptimizer) calculateMaxRetries(profile *EndpointProfile) int {
	successRate := float64(profile.SuccessCount) / float64(profile.TotalAttempts)
	switch {
	case successRate > 0.95:
		return 3 // Usually works, few retries needed
	case successRate > 0.8:
		return 5
	case successRate > 0.5:
		return 7
	default:
		return 10 // Struggling endpoint, give more chances
	}
}

func (o *DeliveryOptimizer) predictSuccessRate(profile *EndpointProfile, attempt int) float64 {
	baseRate := float64(profile.SuccessCount) / float64(profile.TotalAttempts)
	// Success probability increases with retries (transient errors clear)
	// but decreases after many attempts (persistent issue)
	if attempt <= 3 {
		return math.Min(1.0, baseRate+float64(attempt)*0.05)
	}
	return math.Max(0.01, baseRate-float64(attempt-3)*0.1)
}

func (o *DeliveryOptimizer) generateReasoning(profile *EndpointProfile, attempt int, predictedSuccess float64, shouldRetry bool) string {
	successRate := float64(profile.SuccessCount) / float64(profile.TotalAttempts)

	if !shouldRetry {
		if predictedSuccess <= 0.05 {
			return "predicted success rate too low to justify retry"
		}
		return "maximum retry attempts reached"
	}

	switch {
	case successRate > 0.95:
		return "endpoint is highly reliable; brief retry should succeed"
	case successRate > 0.8:
		return "endpoint is generally reliable; standard retry recommended"
	case successRate > 0.5:
		return "endpoint has moderate reliability; extended retry with backoff recommended"
	default:
		return "endpoint is unreliable; aggressive backoff with extended retries"
	}
}

func severityFromRate(rate float64) string {
	switch {
	case rate < 0.3:
		return "critical"
	case rate < 0.5:
		return "high"
	case rate < 0.7:
		return "medium"
	default:
		return "low"
	}
}
