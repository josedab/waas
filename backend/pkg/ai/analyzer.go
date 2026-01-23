package ai

import (
	"context"
	"fmt"
	"math"
	"time"
)

// AnalyticsRepository provides access to delivery analytics data
type AnalyticsRepository interface {
	GetDeliveryTrends(ctx context.Context, tenantID string, since time.Time, resolution string) ([]TrendDataPoint, error)
	GetEndpointDeliveryStats(ctx context.Context, endpointID string, since time.Time) (*EndpointDeliveryStats, error)
	GetTenantDeliveryStats(ctx context.Context, tenantID string, since time.Time) (*TenantDeliveryStats, error)
}

// TrendDataPoint represents a point in trend data
type TrendDataPoint struct {
	Timestamp    time.Time `json:"timestamp"`
	SuccessCount int       `json:"success_count"`
	FailureCount int       `json:"failure_count"`
	AvgLatencyMs float64   `json:"avg_latency_ms"`
	TotalCount   int       `json:"total_count"`
}

// EndpointDeliveryStats holds aggregate delivery stats for an endpoint
type EndpointDeliveryStats struct {
	EndpointID     string             `json:"endpoint_id"`
	TotalDeliveries int               `json:"total_deliveries"`
	SuccessCount   int                `json:"success_count"`
	FailureCount   int                `json:"failure_count"`
	AvgLatencyMs   float64            `json:"avg_latency_ms"`
	P95LatencyMs   float64            `json:"p95_latency_ms"`
	ErrorTypes     map[string]int     `json:"error_types"`
	HourlyFailures map[int]int        `json:"hourly_failures"` // hour of day -> failure count
}

// TenantDeliveryStats holds aggregate delivery stats for a tenant
type TenantDeliveryStats struct {
	TenantID        string             `json:"tenant_id"`
	TotalDeliveries int                `json:"total_deliveries"`
	SuccessCount    int                `json:"success_count"`
	FailureCount    int                `json:"failure_count"`
	EndpointCount   int                `json:"endpoint_count"`
	TopErrors       map[string]int     `json:"top_errors"`
}

// AnalyzerConfig holds configuration for the analyzer
type AnalyzerConfig struct {
	TrendWindowHours     int
	AnomalyStdDevFactor  float64
	MinDataPointsForTrend int
	InsightMaxCount      int
}

// DefaultAnalyzerConfig returns default analyzer configuration
func DefaultAnalyzerConfig() AnalyzerConfig {
	return AnalyzerConfig{
		TrendWindowHours:      24,
		AnomalyStdDevFactor:   2.0,
		MinDataPointsForTrend: 6,
		InsightMaxCount:       20,
	}
}

// DeliveryAnalyzer provides delivery pattern analysis
type DeliveryAnalyzer struct {
	repo   AnalyticsRepository
	config AnalyzerConfig
}

// NewDeliveryAnalyzer creates a new DeliveryAnalyzer
func NewDeliveryAnalyzer(repo AnalyticsRepository, config *AnalyzerConfig) *DeliveryAnalyzer {
	cfg := DefaultAnalyzerConfig()
	if config != nil {
		cfg = *config
	}
	return &DeliveryAnalyzer{
		repo:   repo,
		config: cfg,
	}
}

// AnalyzeTrends detects delivery trends for a tenant
func (a *DeliveryAnalyzer) AnalyzeTrends(ctx context.Context, tenantID string, window time.Duration) ([]DeliveryInsight, error) {
	since := time.Now().Add(-window)
	trends, err := a.repo.GetDeliveryTrends(ctx, tenantID, since, "1h")
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery trends: %w", err)
	}

	if len(trends) < a.config.MinDataPointsForTrend {
		return nil, nil
	}

	var insights []DeliveryInsight

	// Analyze failure rate trend
	failureRates := make([]float64, len(trends))
	for i, t := range trends {
		if t.TotalCount > 0 {
			failureRates[i] = float64(t.FailureCount) / float64(t.TotalCount)
		}
	}

	// Check for increasing failure rate
	if isIncreasing(failureRates) {
		latest := failureRates[len(failureRates)-1]
		severity := "info"
		if latest > 0.3 {
			severity = "critical"
		} else if latest > 0.1 {
			severity = "warning"
		}

		insights = append(insights, DeliveryInsight{
			Type:            "trend",
			Severity:        severity,
			Title:           "Increasing Failure Rate",
			Description:     fmt.Sprintf("Delivery failure rate has been steadily increasing to %.1f%%", latest*100),
			SuggestedAction: "Investigate failing endpoints and check for common error patterns",
			DetectedAt:      time.Now(),
		})
	}

	// Analyze latency trend
	latencies := make([]float64, len(trends))
	for i, t := range trends {
		latencies[i] = t.AvgLatencyMs
	}

	if isIncreasing(latencies) && latencies[len(latencies)-1] > 1000 {
		insights = append(insights, DeliveryInsight{
			Type:            "trend",
			Severity:        "warning",
			Title:           "Increasing Delivery Latency",
			Description:     fmt.Sprintf("Average delivery latency has increased to %.0fms", latencies[len(latencies)-1]),
			SuggestedAction: "Review endpoint response times and consider adjusting timeout settings",
			DetectedAt:      time.Now(),
		})
	}

	// Check for traffic volume changes
	volumes := make([]float64, len(trends))
	for i, t := range trends {
		volumes[i] = float64(t.TotalCount)
	}

	if isDecreasing(volumes) && len(volumes) > 2 && volumes[len(volumes)-1] < volumes[0]*0.5 {
		insights = append(insights, DeliveryInsight{
			Type:            "trend",
			Severity:        "info",
			Title:           "Declining Delivery Volume",
			Description:     "Delivery volume has dropped significantly compared to the start of the window",
			SuggestedAction: "Verify webhook event sources are functioning correctly",
			DetectedAt:      time.Now(),
		})
	}

	return insights, nil
}

// PredictNextHourFailureRate predicts the failure rate for the next hour
func (a *DeliveryAnalyzer) PredictNextHourFailureRate(ctx context.Context, endpointID string) (*HealthPrediction, error) {
	since := time.Now().Add(-24 * time.Hour)
	stats, err := a.repo.GetEndpointDeliveryStats(ctx, endpointID, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint stats: %w", err)
	}

	if stats.TotalDeliveries < a.config.MinDataPointsForTrend {
		return nil, fmt.Errorf("insufficient data for prediction")
	}

	currentFailRate := float64(stats.FailureCount) / float64(stats.TotalDeliveries)

	// Simple weighted prediction based on current rate and time patterns
	currentHour := time.Now().Hour()
	hourlyWeight := 1.0
	if hourlyFails, ok := stats.HourlyFailures[currentHour]; ok && stats.FailureCount > 0 {
		hourlyWeight = float64(hourlyFails) / float64(stats.FailureCount) * 24.0
	}

	predictedRate := currentFailRate * hourlyWeight
	if predictedRate > 1.0 {
		predictedRate = 1.0
	}

	confidence := 0.5
	if stats.TotalDeliveries > 100 {
		confidence = 0.7
	}
	if stats.TotalDeliveries > 1000 {
		confidence = 0.8
	}

	return &HealthPrediction{
		Timestamp:   time.Now().Add(time.Hour),
		Metric:      "failure_rate",
		PredValue:   predictedRate,
		Confidence:  confidence,
		Description: fmt.Sprintf("Predicted failure rate for next hour: %.1f%% (current: %.1f%%)", predictedRate*100, currentFailRate*100),
	}, nil
}

// IdentifyBottlenecks finds delivery bottlenecks for a tenant
func (a *DeliveryAnalyzer) IdentifyBottlenecks(ctx context.Context, tenantID string) ([]DeliveryInsight, error) {
	since := time.Now().Add(-time.Duration(a.config.TrendWindowHours) * time.Hour)

	tenantStats, err := a.repo.GetTenantDeliveryStats(ctx, tenantID, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant stats: %w", err)
	}

	var insights []DeliveryInsight

	// High overall failure rate
	if tenantStats.TotalDeliveries > 0 {
		failRate := float64(tenantStats.FailureCount) / float64(tenantStats.TotalDeliveries)
		if failRate > 0.2 {
			insights = append(insights, DeliveryInsight{
				Type:            "recommendation",
				Severity:        "critical",
				Title:           "High Overall Failure Rate",
				Description:     fmt.Sprintf("%.1f%% of deliveries are failing across %d endpoints", failRate*100, tenantStats.EndpointCount),
				SuggestedAction: "Review top error types and address the most common failures first",
				DetectedAt:      time.Now(),
			})
		}
	}

	// Identify dominant error types
	for errType, count := range tenantStats.TopErrors {
		if tenantStats.FailureCount > 0 && float64(count)/float64(tenantStats.FailureCount) > 0.5 {
			insights = append(insights, DeliveryInsight{
				Type:            "recommendation",
				Severity:        "warning",
				Title:           fmt.Sprintf("Dominant Error Type: %s", errType),
				Description:     fmt.Sprintf("'%s' accounts for %d of %d failures (>50%%)", errType, count, tenantStats.FailureCount),
				SuggestedAction: fmt.Sprintf("Focus remediation efforts on '%s' errors", errType),
				DetectedAt:      time.Now(),
			})
		}
	}

	return insights, nil
}

// GenerateInsights produces an AI-powered summary of insights for a tenant
func (a *DeliveryAnalyzer) GenerateInsights(ctx context.Context, tenantID string) ([]DeliveryInsight, error) {
	var allInsights []DeliveryInsight

	// Collect trends
	window := time.Duration(a.config.TrendWindowHours) * time.Hour
	trendInsights, err := a.AnalyzeTrends(ctx, tenantID, window)
	if err == nil && trendInsights != nil {
		allInsights = append(allInsights, trendInsights...)
	}

	// Collect bottlenecks
	bottleneckInsights, err := a.IdentifyBottlenecks(ctx, tenantID)
	if err == nil && bottleneckInsights != nil {
		allInsights = append(allInsights, bottleneckInsights...)
	}

	// Add tenant-level summary
	since := time.Now().Add(-window)
	tenantStats, err := a.repo.GetTenantDeliveryStats(ctx, tenantID, since)
	if err == nil && tenantStats.TotalDeliveries > 0 {
		successRate := float64(tenantStats.SuccessCount) / float64(tenantStats.TotalDeliveries)
		if successRate >= 0.99 {
			allInsights = append(allInsights, DeliveryInsight{
				Type:        "alert",
				Severity:    "info",
				Title:       "Excellent Delivery Health",
				Description: fmt.Sprintf("%.2f%% delivery success rate across %d deliveries", successRate*100, tenantStats.TotalDeliveries),
				DetectedAt:  time.Now(),
			})
		}
	}

	// Limit results
	if len(allInsights) > a.config.InsightMaxCount {
		allInsights = allInsights[:a.config.InsightMaxCount]
	}

	return allInsights, nil
}

// isIncreasing checks if a series of values shows an increasing trend
func isIncreasing(values []float64) bool {
	if len(values) < 3 {
		return false
	}
	n := float64(len(values))
	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0
	for i, v := range values {
		x := float64(i)
		sumX += x
		sumY += v
		sumXY += x * v
		sumX2 += x * x
	}
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	return slope > 0 && math.Abs(slope) > 0.001
}

// isDecreasing checks if a series of values shows a decreasing trend
func isDecreasing(values []float64) bool {
	if len(values) < 3 {
		return false
	}
	n := float64(len(values))
	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0
	for i, v := range values {
		x := float64(i)
		sumX += x
		sumY += v
		sumXY += x * v
		sumX2 += x * x
	}
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	return slope < 0 && math.Abs(slope) > 0.001
}
