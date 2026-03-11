package ai

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests below extend coverage for analyzer.go beyond what ai_test.go provides.
// They use the existing mockAnalyticsRepo from ai_test.go.

func TestAnalyzeTrends_ZeroDataPoints(t *testing.T) {
	repo := &mockAnalyticsRepo{trends: []TrendDataPoint{}}
	analyzer := NewDeliveryAnalyzer(repo, nil)

	insights, err := analyzer.AnalyzeTrends(context.Background(), "t1", 24*time.Hour)
	assert.NoError(t, err)
	assert.Nil(t, insights)
}

func TestAnalyzeTrends_ExactlyMinDataPoints(t *testing.T) {
	// Default MinDataPointsForTrend = 6; with exactly 6 points it should proceed
	trends := make([]TrendDataPoint, 6)
	for i := range trends {
		trends[i] = TrendDataPoint{TotalCount: 100, SuccessCount: 99, FailureCount: 1, AvgLatencyMs: 100}
	}
	repo := &mockAnalyticsRepo{trends: trends}
	analyzer := NewDeliveryAnalyzer(repo, nil)

	insights, err := analyzer.AnalyzeTrends(context.Background(), "t1", 24*time.Hour)
	assert.NoError(t, err)
	// Flat data, so no insights expected - but no error either
	assert.Empty(t, insights)
}

func TestAnalyzeTrends_RepoError(t *testing.T) {
	repo := &mockAnalyticsRepo{err: errors.New("db connection failed")}
	analyzer := NewDeliveryAnalyzer(repo, nil)

	insights, err := analyzer.AnalyzeTrends(context.Background(), "t1", 24*time.Hour)
	assert.Error(t, err)
	assert.Nil(t, insights)
	assert.Contains(t, err.Error(), "failed to get delivery trends")
}

func TestAnalyzeTrends_DivisionByZeroWhenTotalCountZero(t *testing.T) {
	trends := make([]TrendDataPoint, 8)
	for i := range trends {
		trends[i] = TrendDataPoint{TotalCount: 0, SuccessCount: 0, FailureCount: 0}
	}
	repo := &mockAnalyticsRepo{trends: trends}
	analyzer := NewDeliveryAnalyzer(repo, nil)

	// Should not panic when TotalCount == 0
	insights, err := analyzer.AnalyzeTrends(context.Background(), "t1", 24*time.Hour)
	assert.NoError(t, err)
	assert.Empty(t, insights)
}

func TestAnalyzeTrends_FailureRateSeverityThresholds(t *testing.T) {
	tests := []struct {
		name             string
		latestRate       float64
		expectedSeverity string
	}{
		{"29% failure rate => warning", 0.29, "warning"},
		{"30% failure rate => warning (not strictly > 0.3)", 0.30, "warning"},
		{"31% failure rate => critical", 0.31, "critical"},
		{"5% failure rate => info", 0.05, "info"},
		{"15% failure rate => warning", 0.15, "warning"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := 7
			trends := make([]TrendDataPoint, n)
			for i := 0; i < n; i++ {
				rate := 0.01 + (tt.latestRate-0.01)*float64(i)/float64(n-1)
				total := 1000
				failures := int(rate * float64(total))
				trends[i] = TrendDataPoint{
					TotalCount:   total,
					SuccessCount: total - failures,
					FailureCount: failures,
					AvgLatencyMs: 200,
				}
			}
			repo := &mockAnalyticsRepo{trends: trends}
			analyzer := NewDeliveryAnalyzer(repo, nil)

			insights, err := analyzer.AnalyzeTrends(context.Background(), "t1", 24*time.Hour)
			require.NoError(t, err)

			for _, ins := range insights {
				if ins.Title == "Increasing Failure Rate" {
					assert.Equal(t, tt.expectedSeverity, ins.Severity)
				}
			}
		})
	}
}

func TestAnalyzeTrends_LatencyThresholdBoundaries(t *testing.T) {
	tests := []struct {
		name          string
		latencyMs     float64
		expectInsight bool
	}{
		{"999ms => no latency insight", 999, false},
		{"1000ms => no latency insight (requires >1000)", 1000, false},
		{"1001ms => latency insight triggered", 1001, true},
		{"2000ms => latency insight triggered", 2000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := 7
			trends := make([]TrendDataPoint, n)
			for i := 0; i < n; i++ {
				lat := 100.0 + (tt.latencyMs-100.0)*float64(i)/float64(n-1)
				trends[i] = TrendDataPoint{
					TotalCount:   100,
					SuccessCount: 99,
					FailureCount: 1,
					AvgLatencyMs: lat,
				}
			}
			repo := &mockAnalyticsRepo{trends: trends}
			analyzer := NewDeliveryAnalyzer(repo, nil)

			insights, err := analyzer.AnalyzeTrends(context.Background(), "t1", 24*time.Hour)
			require.NoError(t, err)

			found := false
			for _, ins := range insights {
				if ins.Title == "Increasing Delivery Latency" {
					found = true
				}
			}
			assert.Equal(t, tt.expectInsight, found)
		})
	}
}

func TestAnalyzeTrends_DecreasingVolume(t *testing.T) {
	trends := make([]TrendDataPoint, 7)
	for i := 0; i < 7; i++ {
		total := 1000 - i*150
		if total < 10 {
			total = 10
		}
		trends[i] = TrendDataPoint{
			TotalCount:   total,
			SuccessCount: total,
			FailureCount: 0,
			AvgLatencyMs: 100,
		}
	}
	repo := &mockAnalyticsRepo{trends: trends}
	analyzer := NewDeliveryAnalyzer(repo, nil)

	insights, err := analyzer.AnalyzeTrends(context.Background(), "t1", 24*time.Hour)
	require.NoError(t, err)

	found := false
	for _, ins := range insights {
		if ins.Title == "Declining Delivery Volume" {
			found = true
			assert.Equal(t, "info", ins.Severity)
		}
	}
	assert.True(t, found)
}

func TestAnalyzeTrends_FlatData_NoInsights(t *testing.T) {
	trends := make([]TrendDataPoint, 10)
	for i := range trends {
		trends[i] = TrendDataPoint{
			TotalCount:   100,
			SuccessCount: 95,
			FailureCount: 5,
			AvgLatencyMs: 200,
		}
	}
	repo := &mockAnalyticsRepo{trends: trends}
	analyzer := NewDeliveryAnalyzer(repo, nil)

	insights, err := analyzer.AnalyzeTrends(context.Background(), "t1", 24*time.Hour)
	require.NoError(t, err)
	// Flat data should produce no trend insights
	assert.Empty(t, insights)
}

func TestPredictNextHourFailureRate_InsufficientData(t *testing.T) {
	repo := &mockAnalyticsRepo{
		endpointStats: &EndpointDeliveryStats{TotalDeliveries: 3},
	}
	analyzer := NewDeliveryAnalyzer(repo, nil)

	_, err := analyzer.PredictNextHourFailureRate(context.Background(), "ep-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient data")
}

func TestPredictNextHourFailureRate_ConfidenceScaling(t *testing.T) {
	tests := []struct {
		deliveries int
		minConf    float64
	}{
		{10, 0.5},
		{101, 0.7},
		{1001, 0.8},
	}
	for _, tt := range tests {
		repo := &mockAnalyticsRepo{
			endpointStats: &EndpointDeliveryStats{
				TotalDeliveries: tt.deliveries,
				SuccessCount:    tt.deliveries - 1,
				FailureCount:    1,
				HourlyFailures:  map[int]int{},
			},
		}
		analyzer := NewDeliveryAnalyzer(repo, nil)

		pred, err := analyzer.PredictNextHourFailureRate(context.Background(), "ep-1")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, pred.Confidence, tt.minConf)
		assert.Equal(t, "failure_rate", pred.Metric)
	}
}

func TestPredictNextHourFailureRate_RepoError(t *testing.T) {
	repo := &mockAnalyticsRepo{err: errors.New("db down")}
	analyzer := NewDeliveryAnalyzer(repo, nil)

	_, err := analyzer.PredictNextHourFailureRate(context.Background(), "ep-1")
	assert.Error(t, err)
}

func TestIdentifyBottlenecks_HighFailureRate(t *testing.T) {
	repo := &mockAnalyticsRepo{
		tenantStats: &TenantDeliveryStats{
			TotalDeliveries: 100,
			FailureCount:    25,
			EndpointCount:   3,
			TopErrors:       map[string]int{},
		},
	}
	analyzer := NewDeliveryAnalyzer(repo, nil)

	insights, err := analyzer.IdentifyBottlenecks(context.Background(), "t1")
	require.NoError(t, err)

	found := false
	for _, ins := range insights {
		if ins.Title == "High Overall Failure Rate" {
			found = true
			assert.Equal(t, "critical", ins.Severity)
		}
	}
	assert.True(t, found)
}

func TestIdentifyBottlenecks_NoFailures(t *testing.T) {
	repo := &mockAnalyticsRepo{
		tenantStats: &TenantDeliveryStats{
			TotalDeliveries: 100,
			SuccessCount:    100,
			FailureCount:    0,
			EndpointCount:   2,
			TopErrors:       map[string]int{},
		},
	}
	analyzer := NewDeliveryAnalyzer(repo, nil)

	insights, err := analyzer.IdentifyBottlenecks(context.Background(), "t1")
	require.NoError(t, err)
	assert.Empty(t, insights)
}

func TestGenerateInsights_LimitsResults(t *testing.T) {
	config := DefaultAnalyzerConfig()
	config.InsightMaxCount = 1

	trends := make([]TrendDataPoint, 7)
	for i := 0; i < 7; i++ {
		trends[i] = TrendDataPoint{
			TotalCount:   100,
			SuccessCount: 100 - (i * 5),
			FailureCount: i * 5,
			AvgLatencyMs: float64(500 + i*200),
		}
	}

	repo := &mockAnalyticsRepo{
		trends: trends,
		tenantStats: &TenantDeliveryStats{
			TotalDeliveries: 1000,
			SuccessCount:    700,
			FailureCount:    300,
			EndpointCount:   5,
			TopErrors:       map[string]int{"timeout": 200},
		},
	}

	analyzer := NewDeliveryAnalyzer(repo, &config)
	insights, err := analyzer.GenerateInsights(context.Background(), "t1")
	require.NoError(t, err)
	assert.LessOrEqual(t, len(insights), 1)
}

func TestNewDeliveryAnalyzer_DefaultConfig(t *testing.T) {
	repo := &mockAnalyticsRepo{}
	analyzer := NewDeliveryAnalyzer(repo, nil)
	assert.NotNil(t, analyzer)
	assert.Equal(t, 6, analyzer.config.MinDataPointsForTrend)
	assert.Equal(t, 24, analyzer.config.TrendWindowHours)
	assert.Equal(t, 20, analyzer.config.InsightMaxCount)
}

func TestNewDeliveryAnalyzer_CustomConfig(t *testing.T) {
	config := AnalyzerConfig{
		TrendWindowHours:      48,
		MinDataPointsForTrend: 10,
		InsightMaxCount:       5,
	}
	repo := &mockAnalyticsRepo{}
	analyzer := NewDeliveryAnalyzer(repo, &config)
	assert.Equal(t, 48, analyzer.config.TrendWindowHours)
	assert.Equal(t, 10, analyzer.config.MinDataPointsForTrend)
}

func TestIsIncreasing_Extended(t *testing.T) {
	tests := []struct {
		name     string
		values   []float64
		expected bool
	}{
		{"empty", []float64{}, false},
		{"single value", []float64{1}, false},
		{"two values", []float64{1, 2}, false},
		{"flat zeroes", []float64{0, 0, 0, 0}, false},
		{"slightly increasing (above threshold)", []float64{0, 0.01, 0.02, 0.03}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isIncreasing(tt.values))
		})
	}
}

func TestIsDecreasing_Extended(t *testing.T) {
	tests := []struct {
		name     string
		values   []float64
		expected bool
	}{
		{"empty", []float64{}, false},
		{"flat", []float64{5, 5, 5, 5}, false},
		{"slightly decreasing (above threshold)", []float64{0.03, 0.02, 0.01, 0}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isDecreasing(tt.values))
		})
	}
}
