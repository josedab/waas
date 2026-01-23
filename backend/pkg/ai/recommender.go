package ai

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"
)

// HistoryRepository provides access to delivery history for recommendations
type HistoryRepository interface {
	GetRecentDeliveries(ctx context.Context, endpointID string, since time.Time, limit int) ([]DeliveryRecord, error)
	GetEndpointStats(ctx context.Context, endpointID string, since time.Time) (*EndpointStats, error)
	GetTopFailingEndpoints(ctx context.Context, tenantID string, since time.Time, limit int) ([]FailingEndpoint, error)
}

// EndpointStats holds aggregate stats for an endpoint
type EndpointStats struct {
	TotalDeliveries int     `json:"total_deliveries"`
	SuccessCount    int     `json:"success_count"`
	FailureCount    int     `json:"failure_count"`
	AvgLatencyMs    float64 `json:"avg_latency_ms"`
	P95LatencyMs    float64 `json:"p95_latency_ms"`
	ErrorBreakdown  map[string]int `json:"error_breakdown"`
}

// RecommenderConfig holds configuration for the recommender
type RecommenderConfig struct {
	DefaultMaxRetries       int
	MaxRecommendedDelay     time.Duration
	HealthScoreThreshold    float64
	AnomalyDeviationFactor  float64
	MinDataPoints           int
}

// DefaultRecommenderConfig returns default recommender configuration
func DefaultRecommenderConfig() RecommenderConfig {
	return RecommenderConfig{
		DefaultMaxRetries:      5,
		MaxRecommendedDelay:    30 * time.Minute,
		HealthScoreThreshold:   50.0,
		AnomalyDeviationFactor: 2.0,
		MinDataPoints:          10,
	}
}

// Recommender provides AI-powered delivery recommendations
type Recommender struct {
	classifier  *Classifier
	historyRepo HistoryRepository
	config      RecommenderConfig
}

// NewRecommender creates a new Recommender
func NewRecommender(classifier *Classifier, historyRepo HistoryRepository, config *RecommenderConfig) *Recommender {
	cfg := DefaultRecommenderConfig()
	if config != nil {
		cfg = *config
	}
	return &Recommender{
		classifier:  classifier,
		historyRepo: historyRepo,
		config:      cfg,
	}
}

// RecommendRetryStrategy produces a smart retry recommendation for an endpoint
func (r *Recommender) RecommendRetryStrategy(ctx context.Context, endpointID string, lastError string) (*RetryRecommendation, error) {
	classification := r.classifier.Classify(lastError, nil, "")

	if !classification.IsRetryable {
		return &RetryRecommendation{
			ShouldRetry: false,
			Confidence:  classification.Confidence,
			Strategy:    "none",
			Reasoning:   fmt.Sprintf("Error category '%s' is not retryable: %s", classification.Category, lastError),
		}, nil
	}

	// Get recent stats
	since := time.Now().Add(-24 * time.Hour)
	stats, err := r.historyRepo.GetEndpointStats(ctx, endpointID, since)
	if err != nil {
		// Fallback to classification-only recommendation
		return r.classificationOnlyRecommendation(classification), nil
	}

	// Calculate health-based factors
	successRate := 0.0
	if stats.TotalDeliveries > 0 {
		successRate = float64(stats.SuccessCount) / float64(stats.TotalDeliveries)
	}

	retryScore := r.classifier.GetRetryabilityScore(classification, 0, stats.FailureCount)

	rec := &RetryRecommendation{
		ShouldRetry: retryScore > 0.3,
		Confidence:  retryScore,
	}

	// Determine strategy and delay based on error type and endpoint health
	switch {
	case classification.Category == CategoryRateLimit:
		rec.Strategy = "fixed"
		rec.RecommendedDelay = time.Duration(classification.SuggestedDelay) * time.Second
		rec.MaxRetries = 3
		rec.Reasoning = "Rate-limited: using fixed delay to respect rate limit windows"

	case classification.Category == CategoryTimeout && successRate > 0.8:
		rec.Strategy = "linear"
		rec.RecommendedDelay = 30 * time.Second
		rec.MaxRetries = r.config.DefaultMaxRetries
		rec.Reasoning = "Timeout with generally healthy endpoint: linear backoff recommended"

	case successRate < 0.5:
		rec.Strategy = "exponential"
		rec.RecommendedDelay = 5 * time.Minute
		rec.MaxRetries = 2
		rec.Reasoning = fmt.Sprintf("Endpoint has low success rate (%.0f%%): conservative retry with exponential backoff", successRate*100)

	case classification.Category == CategoryServerError:
		rec.Strategy = "exponential"
		rec.RecommendedDelay = time.Duration(classification.SuggestedDelay) * time.Second
		rec.MaxRetries = r.config.DefaultMaxRetries
		rec.Reasoning = "Server error: exponential backoff to allow recovery"

	default:
		rec.Strategy = "adaptive"
		rec.RecommendedDelay = time.Duration(classification.SuggestedDelay) * time.Second
		rec.MaxRetries = r.config.DefaultMaxRetries
		rec.Reasoning = fmt.Sprintf("Adaptive strategy based on %s error pattern", classification.Category)
	}

	// Cap delay
	if rec.RecommendedDelay > r.config.MaxRecommendedDelay {
		rec.RecommendedDelay = r.config.MaxRecommendedDelay
	}

	return rec, nil
}

func (r *Recommender) classificationOnlyRecommendation(c ErrorClassification) *RetryRecommendation {
	return &RetryRecommendation{
		ShouldRetry:      c.IsRetryable,
		Confidence:       c.Confidence * 0.8,
		RecommendedDelay: time.Duration(c.SuggestedDelay) * time.Second,
		MaxRetries:       r.config.DefaultMaxRetries,
		Strategy:         "exponential",
		Reasoning:        fmt.Sprintf("Classification-based recommendation for %s/%s", c.Category, c.SubCategory),
	}
}

// GenerateHealthReport generates a comprehensive health report for an endpoint
func (r *Recommender) GenerateHealthReport(ctx context.Context, endpointID string, timeRange time.Duration) (*EndpointHealthReport, error) {
	since := time.Now().Add(-timeRange)

	stats, err := r.historyRepo.GetEndpointStats(ctx, endpointID, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint stats: %w", err)
	}

	deliveries, err := r.historyRepo.GetRecentDeliveries(ctx, endpointID, since, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to get deliveries: %w", err)
	}

	report := &EndpointHealthReport{
		EndpointID:     endpointID,
		ErrorBreakdown: stats.ErrorBreakdown,
		AvgLatency:     time.Duration(stats.AvgLatencyMs) * time.Millisecond,
		P95Latency:     time.Duration(stats.P95LatencyMs) * time.Millisecond,
		GeneratedAt:    time.Now(),
	}

	// Calculate success rate
	if stats.TotalDeliveries > 0 {
		report.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalDeliveries)
	}

	// Calculate health score (0-100)
	report.HealthScore = r.calculateHealthScore(report.SuccessRate, stats.AvgLatencyMs, stats.P95LatencyMs)

	// Determine trend from delivery history
	report.Trend = r.detectTrend(deliveries)

	// Generate predictions
	report.Predictions = r.generateHealthPredictions(deliveries, stats)

	return report, nil
}

func (r *Recommender) calculateHealthScore(successRate, avgLatency, p95Latency float64) float64 {
	// Success rate contributes 50%
	successScore := successRate * 100.0

	// Latency contributes 30% (lower is better, max 2000ms)
	latencyScore := math.Max(0, 100.0-(avgLatency/2000.0)*100.0)

	// P95 contributes 20%
	p95Score := math.Max(0, 100.0-(p95Latency/5000.0)*100.0)

	return math.Min(100, successScore*0.5+latencyScore*0.3+p95Score*0.2)
}

func (r *Recommender) detectTrend(deliveries []DeliveryRecord) string {
	if len(deliveries) < r.config.MinDataPoints {
		return "stable"
	}

	// Split into halves and compare failure rates
	mid := len(deliveries) / 2
	firstHalf := deliveries[:mid]
	secondHalf := deliveries[mid:]

	firstFailRate := failureRate(firstHalf)
	secondFailRate := failureRate(secondHalf)

	diff := secondFailRate - firstFailRate
	switch {
	case diff < -0.1:
		return "improving"
	case diff > 0.2:
		return "critical"
	case diff > 0.05:
		return "degrading"
	default:
		return "stable"
	}
}

func failureRate(records []DeliveryRecord) float64 {
	if len(records) == 0 {
		return 0
	}
	failures := 0
	for _, r := range records {
		if r.Status == "failed" {
			failures++
		}
	}
	return float64(failures) / float64(len(records))
}

func (r *Recommender) generateHealthPredictions(deliveries []DeliveryRecord, stats *EndpointStats) []HealthPrediction {
	var predictions []HealthPrediction

	if stats.TotalDeliveries == 0 {
		return predictions
	}

	currentFailRate := float64(stats.FailureCount) / float64(stats.TotalDeliveries)

	// Predict failure rate trend
	if currentFailRate > 0.1 {
		predictions = append(predictions, HealthPrediction{
			Timestamp:   time.Now().Add(time.Hour),
			Metric:      "failure_rate",
			PredValue:   currentFailRate * 1.1,
			Confidence:  0.6,
			Description: "Failure rate may continue increasing based on current trend",
		})
	}

	// Predict latency trend
	if stats.P95LatencyMs > 2000 {
		predictions = append(predictions, HealthPrediction{
			Timestamp:   time.Now().Add(time.Hour),
			Metric:      "p95_latency",
			PredValue:   stats.P95LatencyMs * 1.15,
			Confidence:  0.55,
			Description: "P95 latency is elevated and may continue rising",
		})
	}

	return predictions
}

// DetectAnomalies finds unusual delivery patterns for a tenant
func (r *Recommender) DetectAnomalies(ctx context.Context, tenantID string, timeRange time.Duration) ([]AnomalyReport, error) {
	// Get top failing endpoints to find anomalies
	since := time.Now().Add(-timeRange)
	failingEndpoints, err := r.historyRepo.GetTopFailingEndpoints(ctx, tenantID, since, 50)
	if err != nil {
		return nil, fmt.Errorf("failed to get failing endpoints: %w", err)
	}

	var anomalies []AnomalyReport

	for _, ep := range failingEndpoints {
		// Check for error spike anomaly
		if ep.FailureRate > 0.5 {
			anomalies = append(anomalies, AnomalyReport{
				ID:          fmt.Sprintf("anomaly-%s-%d", ep.EndpointID, time.Now().UnixMilli()),
				TenantID:    tenantID,
				EndpointID:  ep.EndpointID,
				Type:        "error_spike",
				Severity:    severityFromFailureRate(ep.FailureRate),
				Description: fmt.Sprintf("Endpoint %s has %.0f%% failure rate (expected <10%%)", ep.URL, ep.FailureRate*100),
				DetectedAt:  time.Now(),
				MetricName:  "failure_rate",
				Expected:    0.1,
				Actual:      ep.FailureRate,
				Deviation:   ep.FailureRate / 0.1,
			})
		}

		// Check for specific error patterns that indicate anomalies
		stats, err := r.historyRepo.GetEndpointStats(ctx, ep.EndpointID, since)
		if err != nil {
			continue
		}

		if stats.P95LatencyMs > 5000 {
			anomalies = append(anomalies, AnomalyReport{
				ID:          fmt.Sprintf("anomaly-latency-%s-%d", ep.EndpointID, time.Now().UnixMilli()),
				TenantID:    tenantID,
				EndpointID:  ep.EndpointID,
				Type:        "latency_spike",
				Severity:    "warning",
				Description: fmt.Sprintf("Endpoint %s P95 latency is %.0fms (expected <2000ms)", ep.URL, stats.P95LatencyMs),
				DetectedAt:  time.Now(),
				MetricName:  "p95_latency_ms",
				Expected:    2000,
				Actual:      stats.P95LatencyMs,
				Deviation:   stats.P95LatencyMs / 2000,
			})
		}
	}

	// Sort by severity
	sort.Slice(anomalies, func(i, j int) bool {
		return severityRank(anomalies[i].Severity) > severityRank(anomalies[j].Severity)
	})

	return anomalies, nil
}

func severityFromFailureRate(rate float64) string {
	switch {
	case rate > 0.8:
		return "critical"
	case rate > 0.5:
		return "warning"
	default:
		return "info"
	}
}

func severityRank(s string) int {
	ranks := map[string]int{"critical": 3, "warning": 2, "info": 1}
	return ranks[s]
}

// GetTopFailingEndpoints surfaces the worst-performing endpoints for a tenant
func (r *Recommender) GetTopFailingEndpoints(ctx context.Context, tenantID string, limit int) ([]FailingEndpoint, error) {
	since := time.Now().Add(-24 * time.Hour)
	return r.historyRepo.GetTopFailingEndpoints(ctx, tenantID, since, limit)
}
