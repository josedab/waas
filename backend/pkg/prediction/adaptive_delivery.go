package prediction

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
)

// AdaptiveDeliveryEngine provides ML-powered delivery optimization
type AdaptiveDeliveryEngine struct {
	service *Service
}

// NewAdaptiveDeliveryEngine creates the adaptive delivery engine
func NewAdaptiveDeliveryEngine(service *Service) *AdaptiveDeliveryEngine {
	return &AdaptiveDeliveryEngine{service: service}
}

// EndpointPattern represents learned delivery patterns for an endpoint
type EndpointPattern struct {
	ID             string            `json:"id"`
	TenantID       string            `json:"tenant_id"`
	EndpointID     string            `json:"endpoint_id"`
	HourlyStats    [24]HourlyPattern `json:"hourly_stats"`
	DayOfWeekStats [7]DayPattern     `json:"day_of_week_stats"`
	OptimalWindow  *DeliveryWindow   `json:"optimal_window,omitempty"`
	FailureProfile *FailureProfile   `json:"failure_profile,omitempty"`
	LastUpdated    time.Time         `json:"last_updated"`
}

// HourlyPattern represents delivery stats for a specific hour
type HourlyPattern struct {
	Hour         int     `json:"hour"`
	SuccessRate  float64 `json:"success_rate"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	Volume       int64   `json:"volume"`
	ErrorRate    float64 `json:"error_rate"`
}

// DayPattern represents delivery stats for a day of the week
type DayPattern struct {
	Day          string  `json:"day"`
	SuccessRate  float64 `json:"success_rate"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	Volume       int64   `json:"volume"`
}

// DeliveryWindow represents the optimal time window for deliveries
type DeliveryWindow struct {
	BestHourStart int     `json:"best_hour_start"`
	BestHourEnd   int     `json:"best_hour_end"`
	Timezone      string  `json:"timezone"`
	SuccessRate   float64 `json:"predicted_success_rate"`
	AvgLatencyMs  float64 `json:"predicted_avg_latency_ms"`
}

// FailureProfile captures failure patterns for prediction
type FailureProfile struct {
	CommonErrors            []ErrorPattern  `json:"common_errors"`
	FailureWindows          []FailureWindow `json:"failure_windows"`
	MeanTimeBetweenFailures float64         `json:"mtbf_hours"`
	RecoveryTimeAvg         float64         `json:"recovery_time_avg_min"`
}

// ErrorPattern represents a common error type
type ErrorPattern struct {
	ErrorType   string    `json:"error_type"`
	Frequency   float64   `json:"frequency"`
	AvgDuration float64   `json:"avg_duration_min"`
	LastSeen    time.Time `json:"last_seen"`
}

// FailureWindow represents a time window with high failure rate
type FailureWindow struct {
	StartHour   int     `json:"start_hour"`
	EndHour     int     `json:"end_hour"`
	FailureRate float64 `json:"failure_rate"`
	DayOfWeek   string  `json:"day_of_week,omitempty"`
}

// PreFlightScore provides a pre-delivery assessment
type PreFlightScore struct {
	EndpointID       string        `json:"endpoint_id"`
	Score            float64       `json:"score"`      // 0-100, higher is better
	Confidence       float64       `json:"confidence"` // 0-1
	Risk             string        `json:"risk"`       // low, medium, high, critical
	PredictedLatency int64         `json:"predicted_latency_ms"`
	SuccessProb      float64       `json:"success_probability"`
	Factors          []ScoreFactor `json:"factors"`
	Recommendation   string        `json:"recommendation"`
}

// ScoreFactor explains a component of the pre-flight score
type ScoreFactor struct {
	Name        string  `json:"name"`
	Impact      float64 `json:"impact"` // -1 to 1
	Description string  `json:"description"`
}

// AdaptiveRetrySchedule provides optimized retry intervals
type AdaptiveRetrySchedule struct {
	EndpointID string          `json:"endpoint_id"`
	MaxRetries int             `json:"max_retries"`
	Intervals  []RetryInterval `json:"intervals"`
	Strategy   string          `json:"strategy"` // exponential, adaptive, fixed, fibonacci
	BaseDelay  time.Duration   `json:"base_delay"`
	MaxDelay   time.Duration   `json:"max_delay"`
}

// RetryInterval represents a single retry interval with reasoning
type RetryInterval struct {
	Attempt     int           `json:"attempt"`
	Delay       time.Duration `json:"delay"`
	Reason      string        `json:"reason"`
	SuccessProb float64       `json:"predicted_success_prob"`
}

// SmartBatchConfig provides intelligent batching recommendations
type SmartBatchConfig struct {
	EndpointID      string `json:"endpoint_id"`
	BatchSize       int    `json:"batch_size"`
	BatchIntervalMs int    `json:"batch_interval_ms"`
	MaxWaitMs       int    `json:"max_wait_ms"`
	Reason          string `json:"reason"`
}

// EndpointReputation tracks endpoint reliability over time
type EndpointReputation struct {
	EndpointID          string               `json:"endpoint_id"`
	URL                 string               `json:"url"`
	ReputationScore     float64              `json:"reputation_score"` // 0-100
	Tier                string               `json:"tier"`             // platinum, gold, silver, bronze, probation
	SuccessRate30d      float64              `json:"success_rate_30d"`
	AvgLatency30d       float64              `json:"avg_latency_30d_ms"`
	Uptime30d           float64              `json:"uptime_30d_pct"`
	IncidentCount       int                  `json:"incident_count_30d"`
	LastIncident        *time.Time           `json:"last_incident,omitempty"`
	TrendDirection      string               `json:"trend"` // improving, stable, degrading
	RateLimitSuggestion *RateLimitSuggestion `json:"rate_limit_suggestion,omitempty"`
	UpdatedAt           time.Time            `json:"updated_at"`
}

// RateLimitSuggestion provides auto-tuned rate limit recommendations
type RateLimitSuggestion struct {
	RequestsPerSec  int    `json:"requests_per_sec"`
	BurstSize       int    `json:"burst_size"`
	ConcurrentLimit int    `json:"concurrent_limit"`
	Reason          string `json:"reason"`
}

// CalculatePreFlightScore assesses delivery likelihood before sending
func (e *AdaptiveDeliveryEngine) CalculatePreFlightScore(ctx context.Context, tenantID, endpointID string) (*PreFlightScore, error) {
	score := &PreFlightScore{
		EndpointID: endpointID,
		Score:      75.0,
		Confidence: 0.5,
		Risk:       "low",
	}

	// Get historical health from the prediction service
	if e.service != nil {
		health, err := e.service.GetEndpointHealthPrediction(ctx, tenantID, endpointID)
		if err == nil && health != nil {
			score.Score = health.HealthScore
			score.PredictedLatency = int64(health.AverageLatencyMs)
			score.SuccessProb = health.SuccessRate / 100
			score.Confidence = 0.85
		}
	}

	// Calculate risk level
	score.Risk = calculateRiskLevel(score.Score)

	// Build factors
	score.Factors = []ScoreFactor{
		{Name: "historical_success_rate", Impact: normalizeImpact(score.SuccessProb), Description: fmt.Sprintf("%.1f%% historical success rate", score.SuccessProb*100)},
		{Name: "current_hour", Impact: hourImpact(time.Now().Hour()), Description: "Time-of-day factor"},
		{Name: "predicted_latency", Impact: latencyImpact(score.PredictedLatency), Description: fmt.Sprintf("Predicted %dms latency", score.PredictedLatency)},
	}

	// Generate recommendation
	if score.Score >= 90 {
		score.Recommendation = "Deliver immediately - endpoint is highly reliable"
	} else if score.Score >= 70 {
		score.Recommendation = "Deliver with standard retry policy"
	} else if score.Score >= 50 {
		score.Recommendation = "Consider delaying delivery or using enhanced retry policy"
	} else {
		score.Recommendation = "High failure risk - consider batching or deferring delivery"
	}

	return score, nil
}

// GenerateAdaptiveRetrySchedule creates optimized retry intervals
func (e *AdaptiveDeliveryEngine) GenerateAdaptiveRetrySchedule(ctx context.Context, tenantID, endpointID string) (*AdaptiveRetrySchedule, error) {
	schedule := &AdaptiveRetrySchedule{
		EndpointID: endpointID,
		MaxRetries: 5,
		Strategy:   "adaptive",
		BaseDelay:  5 * time.Second,
		MaxDelay:   1 * time.Hour,
	}

	// Default adaptive schedule based on common patterns
	intervals := []struct {
		delay       time.Duration
		successProb float64
		reason      string
	}{
		{5 * time.Second, 0.80, "Quick retry for transient errors"},
		{30 * time.Second, 0.70, "Short backoff for temporary issues"},
		{5 * time.Minute, 0.60, "Medium backoff allowing recovery"},
		{30 * time.Minute, 0.50, "Extended wait for longer outages"},
		{2 * time.Hour, 0.40, "Final attempt after significant delay"},
	}

	// Adjust based on endpoint patterns if available
	if e.service != nil {
		health, err := e.service.GetEndpointHealthPrediction(ctx, tenantID, endpointID)
		if err == nil && health != nil {
			if health.SuccessRate > 95 {
				// Highly reliable - use shorter intervals
				intervals[0].delay = 2 * time.Second
				intervals[1].delay = 15 * time.Second
				intervals[2].delay = 2 * time.Minute
			} else if health.SuccessRate < 80 {
				// Less reliable - use longer intervals
				intervals[0].delay = 30 * time.Second
				intervals[1].delay = 2 * time.Minute
				intervals[2].delay = 15 * time.Minute
			}
		}
	}

	for i, interval := range intervals {
		schedule.Intervals = append(schedule.Intervals, RetryInterval{
			Attempt:     i + 1,
			Delay:       interval.delay,
			Reason:      interval.reason,
			SuccessProb: interval.successProb,
		})
	}

	return schedule, nil
}

// GetEndpointReputation calculates the reputation score for an endpoint
func (e *AdaptiveDeliveryEngine) GetEndpointReputation(ctx context.Context, tenantID, endpointID string) (*EndpointReputation, error) {
	rep := &EndpointReputation{
		EndpointID:      endpointID,
		ReputationScore: 80.0,
		Tier:            "gold",
		SuccessRate30d:  99.0,
		AvgLatency30d:   150,
		Uptime30d:       99.5,
		TrendDirection:  "stable",
		UpdatedAt:       time.Now(),
	}

	// Get real data if available
	if e.service != nil {
		health, err := e.service.GetEndpointHealthPrediction(ctx, tenantID, endpointID)
		if err == nil && health != nil {
			rep.ReputationScore = health.HealthScore
			rep.SuccessRate30d = health.SuccessRate
			rep.AvgLatency30d = health.AverageLatencyMs
		}
	}

	// Calculate tier
	rep.Tier = calculateTier(rep.ReputationScore)

	// Generate rate limit suggestion
	rep.RateLimitSuggestion = &RateLimitSuggestion{
		RequestsPerSec:  calculateOptimalRPS(rep.ReputationScore, rep.AvgLatency30d),
		BurstSize:       calculateBurstSize(rep.ReputationScore),
		ConcurrentLimit: calculateConcurrency(rep.ReputationScore),
		Reason:          fmt.Sprintf("Based on %s tier endpoint with %.1f%% success rate", rep.Tier, rep.SuccessRate30d),
	}

	return rep, nil
}

// GetSmartBatchConfig recommends batching parameters for an endpoint
func (e *AdaptiveDeliveryEngine) GetSmartBatchConfig(ctx context.Context, tenantID, endpointID string) (*SmartBatchConfig, error) {
	config := &SmartBatchConfig{
		EndpointID:      endpointID,
		BatchSize:       10,
		BatchIntervalMs: 1000,
		MaxWaitMs:       5000,
		Reason:          "Default batching configuration",
	}

	// Adjust based on endpoint reputation
	rep, err := e.GetEndpointReputation(ctx, tenantID, endpointID)
	if err == nil {
		switch rep.Tier {
		case "platinum":
			config.BatchSize = 50
			config.BatchIntervalMs = 100
			config.MaxWaitMs = 1000
			config.Reason = "High-performance endpoint: aggressive batching"
		case "gold":
			config.BatchSize = 25
			config.BatchIntervalMs = 500
			config.MaxWaitMs = 3000
			config.Reason = "Reliable endpoint: standard batching"
		case "silver":
			config.BatchSize = 10
			config.BatchIntervalMs = 1000
			config.MaxWaitMs = 5000
			config.Reason = "Moderate reliability: conservative batching"
		case "bronze", "probation":
			config.BatchSize = 5
			config.BatchIntervalMs = 2000
			config.MaxWaitMs = 10000
			config.Reason = "Low reliability: minimal batching with longer intervals"
		}
	}

	return config, nil
}

// GetEndpointHealthPrediction returns health prediction for an endpoint (extends existing service)
func (s *Service) GetEndpointHealthPrediction(ctx context.Context, tenantID, endpointID string) (*EndpointHealth, error) {
	if s.repo == nil {
		return &EndpointHealth{
			EndpointID:       endpointID,
			HealthScore:      80.0,
			SuccessRate:      99.0,
			AverageLatencyMs: 150,
			CurrentStatus:    "healthy",
		}, nil
	}
	return s.repo.GetEndpointHealth(ctx, tenantID, endpointID)
}

// LearnPatterns collects and learns from delivery data
func (e *AdaptiveDeliveryEngine) LearnPatterns(ctx context.Context, tenantID, endpointID string, deliveries []DeliveryDataPoint) (*EndpointPattern, error) {
	pattern := &EndpointPattern{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		EndpointID:  endpointID,
		LastUpdated: time.Now(),
	}

	// Aggregate by hour
	hourData := make(map[int][]DeliveryDataPoint)
	for _, d := range deliveries {
		hour := d.Timestamp.Hour()
		hourData[hour] = append(hourData[hour], d)
	}

	for hour := 0; hour < 24; hour++ {
		data := hourData[hour]
		if len(data) == 0 {
			continue
		}

		var successCount int64
		var totalLatency float64
		for _, d := range data {
			if d.Success {
				successCount++
			}
			totalLatency += d.LatencyMs
		}

		pattern.HourlyStats[hour] = HourlyPattern{
			Hour:         hour,
			SuccessRate:  float64(successCount) / float64(len(data)) * 100,
			AvgLatencyMs: totalLatency / float64(len(data)),
			Volume:       int64(len(data)),
			ErrorRate:    float64(len(data)-int(successCount)) / float64(len(data)) * 100,
		}
	}

	// Find optimal delivery window
	pattern.OptimalWindow = findOptimalWindow(pattern.HourlyStats[:])

	return pattern, nil
}

// DeliveryDataPoint represents a single delivery for learning
type DeliveryDataPoint struct {
	Timestamp  time.Time `json:"timestamp"`
	LatencyMs  float64   `json:"latency_ms"`
	Success    bool      `json:"success"`
	StatusCode int       `json:"status_code"`
	ErrorType  string    `json:"error_type,omitempty"`
}

// Helper functions

func findOptimalWindow(hourly []HourlyPattern) *DeliveryWindow {
	type scored struct {
		hour  int
		score float64
	}
	var scores []scored
	for _, h := range hourly {
		if h.Volume == 0 {
			continue
		}
		score := h.SuccessRate - (h.ErrorRate * 2) - (h.AvgLatencyMs / 100)
		scores = append(scores, scored{h.Hour, score})
	}

	if len(scores) == 0 {
		return nil
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	best := scores[0]
	return &DeliveryWindow{
		BestHourStart: best.hour,
		BestHourEnd:   (best.hour + 4) % 24,
		Timezone:      "UTC",
		SuccessRate:   hourly[best.hour].SuccessRate,
		AvgLatencyMs:  hourly[best.hour].AvgLatencyMs,
	}
}

func calculateRiskLevel(score float64) string {
	if score >= 90 {
		return "low"
	} else if score >= 70 {
		return "medium"
	} else if score >= 50 {
		return "high"
	}
	return "critical"
}

func calculateTier(score float64) string {
	if score >= 95 {
		return "platinum"
	} else if score >= 85 {
		return "gold"
	} else if score >= 70 {
		return "silver"
	} else if score >= 50 {
		return "bronze"
	}
	return "probation"
}

func normalizeImpact(prob float64) float64 {
	return math.Max(-1, math.Min(1, (prob-0.5)*2))
}

func hourImpact(hour int) float64 {
	// Business hours (9-17) get positive impact
	if hour >= 9 && hour <= 17 {
		return 0.3
	}
	// Early morning has lower impact
	if hour >= 2 && hour <= 5 {
		return -0.2
	}
	return 0.0
}

func latencyImpact(latencyMs int64) float64 {
	if latencyMs < 100 {
		return 0.5
	} else if latencyMs < 500 {
		return 0.2
	} else if latencyMs < 2000 {
		return -0.2
	}
	return -0.5
}

func calculateOptimalRPS(score, latency float64) int {
	base := 100
	if score >= 95 {
		base = 500
	} else if score >= 85 {
		base = 200
	} else if score < 70 {
		base = 50
	}
	if latency > 1000 {
		base = base / 2
	}
	return base
}

func calculateBurstSize(score float64) int {
	if score >= 95 {
		return 100
	} else if score >= 85 {
		return 50
	}
	return 20
}

func calculateConcurrency(score float64) int {
	if score >= 95 {
		return 50
	} else if score >= 85 {
		return 25
	} else if score >= 70 {
		return 10
	}
	return 5
}
