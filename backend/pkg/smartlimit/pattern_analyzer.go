package smartlimit

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
)

// PatternAnalyzer learns per-endpoint throughput patterns and dynamically
// adjusts rate limits based on observed behavior over time.
type PatternAnalyzer struct {
	repo    Repository
	mu      sync.RWMutex
	cache   map[string]*EndpointPattern // key: tenantID:endpointID
	config  *PatternConfig
}

// PatternConfig configures the pattern analyzer behavior
type PatternConfig struct {
	// AnalysisWindow is the lookback period for pattern detection
	AnalysisWindow time.Duration
	// MinSamplesForPattern is the minimum data points needed to establish a pattern
	MinSamplesForPattern int
	// PatternDecayFactor controls how quickly old patterns lose influence (0-1)
	PatternDecayFactor float64
	// AnomalyThreshold is the standard deviation multiplier for anomaly detection
	AnomalyThreshold float64
	// UpdateInterval is how often patterns are recalculated
	UpdateInterval time.Duration
}

// DefaultPatternConfig returns sensible defaults
func DefaultPatternConfig() *PatternConfig {
	return &PatternConfig{
		AnalysisWindow:       7 * 24 * time.Hour, // 7 days
		MinSamplesForPattern: 50,
		PatternDecayFactor:   0.95,
		AnomalyThreshold:    2.0,
		UpdateInterval:       15 * time.Minute,
	}
}

// EndpointPattern captures learned throughput patterns for an endpoint
type EndpointPattern struct {
	EndpointID       string             `json:"endpoint_id"`
	TenantID         string             `json:"tenant_id"`
	HourlyRates      [24]float64        `json:"hourly_rates"`       // Avg successful rate per hour
	HourlyLatencies  [24]float64        `json:"hourly_latencies"`   // Avg latency per hour
	DayOfWeekFactors [7]float64         `json:"day_of_week_factors"` // Multiplier per day of week
	PeakRate         float64            `json:"peak_rate"`
	BaselineRate     float64            `json:"baseline_rate"`
	AvgSuccessRate   float64            `json:"avg_success_rate"`
	AvgLatencyMs     float64            `json:"avg_latency_ms"`
	ErrorBurstScore  float64            `json:"error_burst_score"`  // 0-1 score of error burstiness
	LastAnalyzedAt   time.Time          `json:"last_analyzed_at"`
	SampleCount      int                `json:"sample_count"`
	ThroughputTrend  ThroughputTrend    `json:"throughput_trend"`
}

// ThroughputTrend indicates the direction of throughput changes
type ThroughputTrend string

const (
	TrendStable     ThroughputTrend = "stable"
	TrendIncreasing ThroughputTrend = "increasing"
	TrendDecreasing ThroughputTrend = "decreasing"
	TrendVolatile   ThroughputTrend = "volatile"
)

// RateRecommendation is the output of the pattern analyzer
type RateRecommendation struct {
	EndpointID      string          `json:"endpoint_id"`
	RecommendedRate float64         `json:"recommended_rate"`
	Confidence      float64         `json:"confidence"` // 0-1
	Reason          string          `json:"reason"`
	Trend           ThroughputTrend `json:"trend"`
	AnomalyDetected bool            `json:"anomaly_detected"`
	ValidUntil      time.Time       `json:"valid_until"`
}

// NewPatternAnalyzer creates a new pattern analyzer
func NewPatternAnalyzer(repo Repository, config *PatternConfig) *PatternAnalyzer {
	if config == nil {
		config = DefaultPatternConfig()
	}
	return &PatternAnalyzer{
		repo:   repo,
		cache:  make(map[string]*EndpointPattern),
		config: config,
	}
}

// AnalyzeEndpoint computes throughput patterns for an endpoint from historical data
func (pa *PatternAnalyzer) AnalyzeEndpoint(ctx context.Context, tenantID, endpointID string) (*EndpointPattern, error) {
	end := time.Now()
	start := end.Add(-pa.config.AnalysisWindow)

	data, err := pa.repo.GetLearningData(ctx, tenantID, endpointID, start, end)
	if err != nil {
		return nil, err
	}

	if len(data) < pa.config.MinSamplesForPattern {
		return nil, nil // Not enough data
	}

	pattern := &EndpointPattern{
		EndpointID:     endpointID,
		TenantID:       tenantID,
		SampleCount:    len(data),
		LastAnalyzedAt: time.Now(),
	}

	// Compute hourly rate patterns and latencies
	hourCounts := [24]int{}
	hourSuccessRates := [24]float64{}
	hourLatencies := [24]float64{}
	dayCounts := [7]int{}
	dayRates := [7]float64{}

	var totalSuccessRate, totalLatency float64
	var successCount, errorBurstLen, maxErrorBurst int

	for i, point := range data {
		h := point.HourOfDay
		d := point.DayOfWeek

		hourCounts[h]++
		hourSuccessRates[h] += point.SuccessRate
		hourLatencies[h] += point.AvgLatency
		dayCounts[d]++
		dayRates[d] += point.RequestRate

		totalSuccessRate += point.SuccessRate
		totalLatency += point.AvgLatency

		if point.SuccessRate >= 0.5 {
			successCount++
		}

		// Track error burst patterns
		if point.RateLimited || point.ResponseCode >= 400 {
			errorBurstLen++
			if errorBurstLen > maxErrorBurst {
				maxErrorBurst = errorBurstLen
			}
		} else {
			errorBurstLen = 0
		}

		// Detect throughput trend from recent vs older data
		_ = i
	}

	n := float64(len(data))
	pattern.AvgSuccessRate = totalSuccessRate / n
	pattern.AvgLatencyMs = totalLatency / n
	if len(data) > 0 {
		pattern.ErrorBurstScore = math.Min(1.0, float64(maxErrorBurst)/20.0)
	}

	// Compute hourly averages
	for h := 0; h < 24; h++ {
		if hourCounts[h] > 0 {
			pattern.HourlyRates[h] = hourSuccessRates[h] / float64(hourCounts[h])
			pattern.HourlyLatencies[h] = hourLatencies[h] / float64(hourCounts[h])
		}
	}

	// Compute day-of-week factors (relative to average)
	avgDayRate := 0.0
	activeDays := 0
	for d := 0; d < 7; d++ {
		if dayCounts[d] > 0 {
			dayRates[d] /= float64(dayCounts[d])
			avgDayRate += dayRates[d]
			activeDays++
		}
	}
	if activeDays > 0 {
		avgDayRate /= float64(activeDays)
	}
	for d := 0; d < 7; d++ {
		if avgDayRate > 0 && dayCounts[d] > 0 {
			pattern.DayOfWeekFactors[d] = dayRates[d] / avgDayRate
		} else {
			pattern.DayOfWeekFactors[d] = 1.0
		}
	}

	// Determine peak and baseline rates
	pattern.PeakRate = 0
	pattern.BaselineRate = math.MaxFloat64
	for h := 0; h < 24; h++ {
		if hourCounts[h] > 0 {
			if pattern.HourlyRates[h] > pattern.PeakRate {
				pattern.PeakRate = pattern.HourlyRates[h]
			}
			if pattern.HourlyRates[h] < pattern.BaselineRate {
				pattern.BaselineRate = pattern.HourlyRates[h]
			}
		}
	}
	if pattern.BaselineRate == math.MaxFloat64 {
		pattern.BaselineRate = 0
	}

	// Detect throughput trend (compare first half vs second half)
	half := len(data) / 2
	if half > 0 {
		var firstHalfRate, secondHalfRate float64
		for i := 0; i < half; i++ {
			firstHalfRate += data[i].RequestRate
		}
		for i := half; i < len(data); i++ {
			secondHalfRate += data[i].RequestRate
		}
		firstHalfRate /= float64(half)
		secondHalfRate /= float64(len(data) - half)

		ratio := secondHalfRate / math.Max(firstHalfRate, 0.001)
		switch {
		case ratio > 1.3:
			pattern.ThroughputTrend = TrendIncreasing
		case ratio < 0.7:
			pattern.ThroughputTrend = TrendDecreasing
		case math.Abs(ratio-1.0) > 0.5:
			pattern.ThroughputTrend = TrendVolatile
		default:
			pattern.ThroughputTrend = TrendStable
		}
	} else {
		pattern.ThroughputTrend = TrendStable
	}

	// Cache the pattern
	pa.mu.Lock()
	pa.cache[tenantID+":"+endpointID] = pattern
	pa.mu.Unlock()

	return pattern, nil
}

// GetRecommendation returns a rate recommendation for the current time
func (pa *PatternAnalyzer) GetRecommendation(ctx context.Context, tenantID, endpointID string, config *AdaptiveRateConfig) (*RateRecommendation, error) {
	key := tenantID + ":" + endpointID

	pa.mu.RLock()
	pattern, exists := pa.cache[key]
	pa.mu.RUnlock()

	if !exists || time.Since(pattern.LastAnalyzedAt) > pa.config.UpdateInterval {
		var err error
		pattern, err = pa.AnalyzeEndpoint(ctx, tenantID, endpointID)
		if err != nil || pattern == nil {
			return &RateRecommendation{
				EndpointID:      endpointID,
				RecommendedRate: config.BaseRatePerSec,
				Confidence:      0.3,
				Reason:          "insufficient data for pattern analysis",
				Trend:           TrendStable,
				ValidUntil:      time.Now().Add(5 * time.Minute),
			}, nil
		}
	}

	now := time.Now()
	hour := now.Hour()
	day := int(now.Weekday())

	// Calculate recommended rate based on pattern
	hourlyFactor := 1.0
	if pattern.HourlyRates[hour] > 0 && pattern.PeakRate > 0 {
		hourlyFactor = pattern.HourlyRates[hour] / pattern.PeakRate
	}

	dayFactor := pattern.DayOfWeekFactors[day]

	// Base recommendation: scale between min and max based on pattern
	rateRange := config.MaxRatePerSec - config.MinRatePerSec
	recommendedRate := config.MinRatePerSec + (rateRange * hourlyFactor * dayFactor)

	// Adjust for error patterns
	if pattern.ErrorBurstScore > 0.5 {
		recommendedRate *= (1.0 - pattern.ErrorBurstScore*0.3)
	}

	// Adjust for latency: if endpoint is slow, reduce rate
	if pattern.HourlyLatencies[hour] > 0 {
		latencyFactor := math.Min(1.0, 1000.0/pattern.HourlyLatencies[hour]) // Slow down if > 1000ms
		recommendedRate *= latencyFactor
	}

	// Clamp to configured bounds
	recommendedRate = math.Max(config.MinRatePerSec, math.Min(config.MaxRatePerSec, recommendedRate))

	// Calculate confidence based on sample count and recency
	confidence := math.Min(1.0, float64(pattern.SampleCount)/float64(pa.config.MinSamplesForPattern*4))
	if time.Since(pattern.LastAnalyzedAt) > pa.config.UpdateInterval*2 {
		confidence *= 0.7 // Reduce confidence for stale patterns
	}

	// Detect anomalies: current conditions deviating significantly from pattern
	anomaly := false
	reason := "rate adjusted based on learned throughput pattern"

	switch pattern.ThroughputTrend {
	case TrendIncreasing:
		reason = "rate increased: endpoint throughput trending upward"
	case TrendDecreasing:
		reason = "rate decreased: endpoint throughput trending downward"
		recommendedRate *= 0.85
	case TrendVolatile:
		reason = "conservative rate: volatile throughput pattern detected"
		recommendedRate *= 0.7
		anomaly = true
	}

	return &RateRecommendation{
		EndpointID:      endpointID,
		RecommendedRate: math.Round(recommendedRate*100) / 100,
		Confidence:      math.Round(confidence*100) / 100,
		Reason:          reason,
		Trend:           pattern.ThroughputTrend,
		AnomalyDetected: anomaly,
		ValidUntil:      now.Add(pa.config.UpdateInterval),
	}, nil
}

// StartBackgroundAnalysis periodically recalculates patterns for all active configs
func (pa *PatternAnalyzer) StartBackgroundAnalysis(ctx context.Context, service *Service) {
	ticker := time.NewTicker(pa.config.UpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pa.refreshActivePatterns(ctx, service)
		}
	}
}

func (pa *PatternAnalyzer) refreshActivePatterns(ctx context.Context, service *Service) {
	// Iterate known throttlers and refresh their patterns
	service.throttlers.Range(func(key, value interface{}) bool {
		k := key.(string)
		t := value.(*Throttler)

		parts := splitKey(k)
		if len(parts) != 2 {
			return true
		}
		tenantID, endpointID := parts[0], parts[1]

		recommendation, err := pa.GetRecommendation(ctx, tenantID, endpointID, t.config)
		if err != nil || recommendation == nil {
			return true
		}

		// Apply recommendation if confidence is high enough
		if recommendation.Confidence >= 0.6 {
			t.ApplyRecommendation(recommendation)
		}

		return true
	})
}

func splitKey(key string) []string {
	for i := 0; i < len(key); i++ {
		if key[i] == ':' {
			return []string{key[:i], key[i+1:]}
		}
	}
	return nil
}

// ApplyRecommendation adjusts the throttler based on a pattern recommendation
func (t *Throttler) ApplyRecommendation(rec *RateRecommendation) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Blend current rate with recommendation (smooth transitions)
	blendFactor := rec.Confidence * 0.3 // Max 30% adjustment per cycle
	t.currentRate = t.currentRate*(1-blendFactor) + rec.RecommendedRate*blendFactor

	// Ensure within bounds
	t.currentRate = math.Max(t.config.MinRatePerSec, math.Min(t.config.MaxRatePerSec, t.currentRate))
}

// --- Enhanced Service integration ---

// GetEndpointPattern retrieves the current learned pattern for an endpoint
func (s *Service) GetEndpointPattern(ctx context.Context, tenantID, endpointID string) (*EndpointPattern, error) {
	if s.analyzer == nil {
		return nil, nil
	}
	return s.analyzer.AnalyzeEndpoint(ctx, tenantID, endpointID)
}

// GetRateRecommendation gets an intelligent rate recommendation
func (s *Service) GetRateRecommendation(ctx context.Context, tenantID, endpointID string) (*RateRecommendation, error) {
	if s.analyzer == nil {
		return nil, nil
	}

	config, err := s.repo.GetConfig(ctx, tenantID, endpointID)
	if err != nil {
		return nil, err
	}

	return s.analyzer.GetRecommendation(ctx, tenantID, endpointID, config)
}

// RecordBehaviorSnapshot captures a behavior snapshot for learning
func (s *Service) RecordBehaviorSnapshot(ctx context.Context, tenantID, endpointID, url string,
	totalReqs, successCount, rateLimitCount, timeoutCount, errorCount int64,
	avgLatency, p50, p95, p99, maxLatency float64) error {

	now := time.Now()
	behavior := &EndpointBehavior{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		EndpointID:     endpointID,
		URL:            url,
		WindowStart:    now.Add(-1 * time.Hour),
		WindowEnd:      now,
		TotalRequests:  totalReqs,
		SuccessCount:   successCount,
		RateLimitCount: rateLimitCount,
		TimeoutCount:   timeoutCount,
		ErrorCount:     errorCount,
		AvgLatencyMs:   avgLatency,
		P50LatencyMs:   p50,
		P95LatencyMs:   p95,
		P99LatencyMs:   p99,
		MaxLatencyMs:   maxLatency,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	return s.repo.SaveBehavior(ctx, behavior)
}
