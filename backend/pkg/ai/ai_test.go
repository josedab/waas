package ai

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Classifier Tests ---

func TestClassifier_ClassifyNewPatterns(t *testing.T) {
	t.Parallel()
	classifier := NewClassifier()

	tests := []struct {
		name         string
		errorMsg     string
		httpStatus   *int
		responseBody string
		wantCategory ErrorCategory
		wantSub      string
		wantRetry    bool
	}{
		{
			name:         "broken pipe",
			errorMsg:     "write: broken pipe",
			wantCategory: CategoryNetwork,
			wantSub:      "broken_pipe",
			wantRetry:    true,
		},
		{
			name:         "connection closed",
			errorMsg:     "connection closed by remote host",
			wantCategory: CategoryNetwork,
			wantSub:      "connection_closed",
			wantRetry:    true,
		},
		{
			name:         "host is down",
			errorMsg:     "host is down",
			wantCategory: CategoryNetwork,
			wantSub:      "host_down",
			wantRetry:    true,
		},
		{
			name:         "ssl handshake failure",
			errorMsg:     "ssl handshake failed",
			wantCategory: CategoryCertificate,
			wantSub:      "ssl_handshake",
			wantRetry:    false,
		},
		{
			name:         "tls handshake error",
			errorMsg:     "tls handshake failure with remote",
			wantCategory: CategoryCertificate,
			wantSub:      "tls_handshake",
			wantRetry:    false,
		},
		{
			name:         "invalid response",
			errorMsg:     "invalid response from server",
			wantCategory: CategoryServerError,
			wantSub:      "invalid_response",
			wantRetry:    true,
		},
		{
			name:         "unexpected eof",
			errorMsg:     "unexpected EOF during read",
			wantCategory: CategoryServerError,
			wantSub:      "unexpected_eof",
			wantRetry:    true,
		},
		{
			name:         "redirect",
			errorMsg:     "301 moved permanently",
			wantCategory: CategoryClientError,
			wantSub:      "redirect",
			wantRetry:    false,
		},
		{
			name:         "payload too large via pattern",
			errorMsg:     "413 payload too large",
			wantCategory: CategoryClientError,
			wantSub:      "payload_too_large",
			wantRetry:    false,
		},
		{
			name:         "protocol error",
			errorMsg:     "protocol error: unexpected message",
			wantCategory: CategoryNetwork,
			wantSub:      "protocol_error",
			wantRetry:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.Classify(tt.errorMsg, tt.httpStatus, tt.responseBody)
			assert.Equal(t, tt.wantCategory, result.Category, "category mismatch")
			assert.Equal(t, tt.wantSub, result.SubCategory, "subcategory mismatch")
			assert.Equal(t, tt.wantRetry, result.IsRetryable, "retryable mismatch")
		})
	}
}

func TestClassifier_ClassifyFromHTTPResponse(t *testing.T) {
	t.Parallel()
	classifier := NewClassifier()

	t.Run("high latency classified as timeout", func(t *testing.T) {
		result := classifier.ClassifyFromHTTPResponse(200, "", nil, 35*time.Second)
		assert.Equal(t, CategoryTimeout, result.Category)
		assert.Equal(t, "slow_response", result.SubCategory)
		assert.True(t, result.IsRetryable)
	})

	t.Run("retry-after header classified as rate limit", func(t *testing.T) {
		headers := map[string]string{"Retry-After": "60"}
		result := classifier.ClassifyFromHTTPResponse(429, "too many requests", headers, time.Second)
		assert.Equal(t, CategoryRateLimit, result.Category)
		assert.Equal(t, "retry_after", result.SubCategory)
	})

	t.Run("redirect status code", func(t *testing.T) {
		result := classifier.ClassifyFromHTTPResponse(302, "", nil, 100*time.Millisecond)
		assert.Equal(t, CategoryClientError, result.Category)
		assert.Equal(t, "redirect", result.SubCategory)
		assert.False(t, result.IsRetryable)
	})

	t.Run("server error body", func(t *testing.T) {
		result := classifier.ClassifyFromHTTPResponse(500, "internal server error", nil, time.Second)
		assert.Equal(t, CategoryServerError, result.Category)
	})

	t.Run("fallback to status code", func(t *testing.T) {
		result := classifier.ClassifyFromHTTPResponse(404, "", nil, 100*time.Millisecond)
		assert.Equal(t, CategoryClientError, result.Category)
		assert.Equal(t, "not_found", result.SubCategory)
	})
}

func TestClassifier_LearnFromOutcome(t *testing.T) {
	t.Parallel()
	classifier := NewClassifier()

	classification := &ErrorClassification{
		Category:    CategoryNetwork,
		SubCategory: "connection_refused",
		Confidence:  0.95,
	}

	// Record correct outcomes
	classifier.LearnFromOutcome(classification, "network")
	classifier.LearnFromOutcome(classification, "network")
	classifier.LearnFromOutcome(classification, "timeout") // incorrect

	accuracy := classifier.GetAccuracy()
	assert.InDelta(t, 0.666, accuracy, 0.01)
}

func TestClassifier_GetRetryabilityScore(t *testing.T) {
	t.Parallel()
	classifier := NewClassifier()

	t.Run("non-retryable returns 0", func(t *testing.T) {
		c := ErrorClassification{Category: CategoryAuth, IsRetryable: false, Confidence: 0.9}
		score := classifier.GetRetryabilityScore(c, 0, 0)
		assert.Equal(t, 0.0, score)
	})

	t.Run("first attempt with healthy endpoint", func(t *testing.T) {
		c := ErrorClassification{Category: CategoryServerError, IsRetryable: true, Confidence: 0.9}
		score := classifier.GetRetryabilityScore(c, 0, 5)
		assert.Greater(t, score, 0.5)
	})

	t.Run("many attempts decrease score", func(t *testing.T) {
		c := ErrorClassification{Category: CategoryServerError, IsRetryable: true, Confidence: 0.9}
		score0 := classifier.GetRetryabilityScore(c, 0, 5)
		score3 := classifier.GetRetryabilityScore(c, 3, 5)
		assert.Greater(t, score0, score3)
	})

	t.Run("high failure count decreases score", func(t *testing.T) {
		c := ErrorClassification{Category: CategoryServerError, IsRetryable: true, Confidence: 0.9}
		scoreLow := classifier.GetRetryabilityScore(c, 0, 5)
		scoreHigh := classifier.GetRetryabilityScore(c, 0, 150)
		assert.Greater(t, scoreLow, scoreHigh)
	})

	t.Run("rate limit gets boost", func(t *testing.T) {
		rl := ErrorClassification{Category: CategoryRateLimit, IsRetryable: true, Confidence: 0.9}
		se := ErrorClassification{Category: CategoryServerError, IsRetryable: true, Confidence: 0.9}
		rlScore := classifier.GetRetryabilityScore(rl, 0, 5)
		seScore := classifier.GetRetryabilityScore(se, 0, 5)
		assert.GreaterOrEqual(t, rlScore, seScore)
	})
}

// --- Recommender Tests ---

type mockHistoryRepo struct {
	deliveries       []DeliveryRecord
	stats            *EndpointStats
	failingEndpoints []FailingEndpoint
	err              error
}

func (m *mockHistoryRepo) GetRecentDeliveries(_ context.Context, _ string, _ time.Time, _ int) ([]DeliveryRecord, error) {
	return m.deliveries, m.err
}

func (m *mockHistoryRepo) GetEndpointStats(_ context.Context, _ string, _ time.Time) (*EndpointStats, error) {
	if m.stats == nil {
		return nil, m.err
	}
	return m.stats, m.err
}

func (m *mockHistoryRepo) GetTopFailingEndpoints(_ context.Context, _ string, _ time.Time, _ int) ([]FailingEndpoint, error) {
	return m.failingEndpoints, m.err
}

func TestRecommender_RecommendRetryStrategy(t *testing.T) {
	t.Parallel()

	t.Run("non-retryable error", func(t *testing.T) {
		repo := &mockHistoryRepo{stats: &EndpointStats{TotalDeliveries: 100, SuccessCount: 90}}
		r := NewRecommender(NewClassifier(), repo, nil)

		rec, err := r.RecommendRetryStrategy(context.Background(), "ep-1", "HTTP 401 unauthorized")
		require.NoError(t, err)
		assert.False(t, rec.ShouldRetry)
		assert.Equal(t, "none", rec.Strategy)
	})

	t.Run("rate limited error", func(t *testing.T) {
		repo := &mockHistoryRepo{stats: &EndpointStats{TotalDeliveries: 100, SuccessCount: 80, FailureCount: 20}}
		r := NewRecommender(NewClassifier(), repo, nil)

		rec, err := r.RecommendRetryStrategy(context.Background(), "ep-1", "429 too many requests")
		require.NoError(t, err)
		assert.True(t, rec.ShouldRetry)
		assert.Equal(t, "fixed", rec.Strategy)
	})

	t.Run("server error with healthy endpoint", func(t *testing.T) {
		repo := &mockHistoryRepo{stats: &EndpointStats{TotalDeliveries: 100, SuccessCount: 95, FailureCount: 5}}
		r := NewRecommender(NewClassifier(), repo, nil)

		rec, err := r.RecommendRetryStrategy(context.Background(), "ep-1", "500 internal server error")
		require.NoError(t, err)
		assert.True(t, rec.ShouldRetry)
		assert.Equal(t, "exponential", rec.Strategy)
	})

	t.Run("error with low success rate endpoint", func(t *testing.T) {
		repo := &mockHistoryRepo{stats: &EndpointStats{TotalDeliveries: 100, SuccessCount: 30, FailureCount: 70}}
		r := NewRecommender(NewClassifier(), repo, nil)

		rec, err := r.RecommendRetryStrategy(context.Background(), "ep-1", "connection refused")
		require.NoError(t, err)
		assert.True(t, rec.ShouldRetry)
		assert.Equal(t, "exponential", rec.Strategy)
		assert.Equal(t, 2, rec.MaxRetries) // Conservative due to low success rate
	})
}

func TestRecommender_GenerateHealthReport(t *testing.T) {
	t.Parallel()

	deliveries := []DeliveryRecord{
		{EndpointID: "ep-1", Status: "success", LatencyMs: 100, Timestamp: time.Now().Add(-2 * time.Hour)},
		{EndpointID: "ep-1", Status: "success", LatencyMs: 150, Timestamp: time.Now().Add(-1 * time.Hour)},
		{EndpointID: "ep-1", Status: "failed", LatencyMs: 5000, Timestamp: time.Now()},
	}
	stats := &EndpointStats{
		TotalDeliveries: 100,
		SuccessCount:    90,
		FailureCount:    10,
		AvgLatencyMs:    200,
		P95LatencyMs:    1500,
		ErrorBreakdown:  map[string]int{"timeout": 7, "connection_refused": 3},
	}

	repo := &mockHistoryRepo{deliveries: deliveries, stats: stats}
	r := NewRecommender(NewClassifier(), repo, nil)

	report, err := r.GenerateHealthReport(context.Background(), "ep-1", 24*time.Hour)
	require.NoError(t, err)
	assert.Equal(t, "ep-1", report.EndpointID)
	assert.InDelta(t, 0.9, report.SuccessRate, 0.01)
	assert.Greater(t, report.HealthScore, 50.0)
	assert.NotEmpty(t, report.ErrorBreakdown)
	assert.Contains(t, []string{"improving", "stable", "degrading", "critical"}, report.Trend)
}

func TestRecommender_DetectAnomalies(t *testing.T) {
	t.Parallel()

	repo := &mockHistoryRepo{
		failingEndpoints: []FailingEndpoint{
			{EndpointID: "ep-1", URL: "https://api.example.com", FailureCount: 90, FailureRate: 0.9, TopError: "timeout"},
			{EndpointID: "ep-2", URL: "https://api2.example.com", FailureCount: 5, FailureRate: 0.05, TopError: "server_error"},
		},
		stats: &EndpointStats{
			TotalDeliveries: 100,
			SuccessCount:    10,
			FailureCount:    90,
			AvgLatencyMs:    300,
			P95LatencyMs:    1000,
			ErrorBreakdown:  map[string]int{"timeout": 90},
		},
	}

	r := NewRecommender(NewClassifier(), repo, nil)
	anomalies, err := r.DetectAnomalies(context.Background(), "tenant-1", 24*time.Hour)
	require.NoError(t, err)
	assert.NotEmpty(t, anomalies)

	// The high failure rate endpoint should be detected
	found := false
	for _, a := range anomalies {
		if a.EndpointID == "ep-1" && a.Type == "error_spike" {
			found = true
			assert.Equal(t, "critical", a.Severity)
		}
	}
	assert.True(t, found, "expected anomaly for ep-1")
}

// --- Analyzer Tests ---

type mockAnalyticsRepo struct {
	trends        []TrendDataPoint
	endpointStats *EndpointDeliveryStats
	tenantStats   *TenantDeliveryStats
	err           error
}

func (m *mockAnalyticsRepo) GetDeliveryTrends(_ context.Context, _ string, _ time.Time, _ string) ([]TrendDataPoint, error) {
	return m.trends, m.err
}

func (m *mockAnalyticsRepo) GetEndpointDeliveryStats(_ context.Context, _ string, _ time.Time) (*EndpointDeliveryStats, error) {
	if m.endpointStats == nil {
		return nil, m.err
	}
	return m.endpointStats, m.err
}

func (m *mockAnalyticsRepo) GetTenantDeliveryStats(_ context.Context, _ string, _ time.Time) (*TenantDeliveryStats, error) {
	if m.tenantStats == nil {
		return nil, m.err
	}
	return m.tenantStats, m.err
}

func TestAnalyzer_AnalyzeTrends_IncreasingFailures(t *testing.T) {
	t.Parallel()

	// Create increasing failure trend
	trends := make([]TrendDataPoint, 10)
	for i := range trends {
		total := 100
		failures := 5 + i*3 // increasing failures
		trends[i] = TrendDataPoint{
			Timestamp:    time.Now().Add(-time.Duration(10-i) * time.Hour),
			SuccessCount: total - failures,
			FailureCount: failures,
			TotalCount:   total,
			AvgLatencyMs: 200,
		}
	}

	repo := &mockAnalyticsRepo{trends: trends}
	analyzer := NewDeliveryAnalyzer(repo, nil)

	insights, err := analyzer.AnalyzeTrends(context.Background(), "tenant-1", 24*time.Hour)
	require.NoError(t, err)

	found := false
	for _, insight := range insights {
		if insight.Title == "Increasing Failure Rate" {
			found = true
			assert.Equal(t, "trend", insight.Type)
		}
	}
	assert.True(t, found, "expected increasing failure rate insight")
}

func TestAnalyzer_AnalyzeTrends_InsufficientData(t *testing.T) {
	t.Parallel()

	repo := &mockAnalyticsRepo{trends: []TrendDataPoint{{}, {}}}
	analyzer := NewDeliveryAnalyzer(repo, nil)

	insights, err := analyzer.AnalyzeTrends(context.Background(), "tenant-1", 24*time.Hour)
	require.NoError(t, err)
	assert.Nil(t, insights)
}

func TestAnalyzer_PredictNextHourFailureRate(t *testing.T) {
	t.Parallel()

	repo := &mockAnalyticsRepo{
		endpointStats: &EndpointDeliveryStats{
			EndpointID:      "ep-1",
			TotalDeliveries: 500,
			SuccessCount:    450,
			FailureCount:    50,
			AvgLatencyMs:    200,
			HourlyFailures:  map[int]int{10: 5, 14: 10},
		},
	}
	analyzer := NewDeliveryAnalyzer(repo, nil)

	prediction, err := analyzer.PredictNextHourFailureRate(context.Background(), "ep-1")
	require.NoError(t, err)
	assert.NotNil(t, prediction)
	assert.Equal(t, "failure_rate", prediction.Metric)
	assert.Greater(t, prediction.PredValue, 0.0)
	assert.LessOrEqual(t, prediction.PredValue, 1.0)
	assert.Greater(t, prediction.Confidence, 0.0)
}

func TestAnalyzer_IdentifyBottlenecks(t *testing.T) {
	t.Parallel()

	repo := &mockAnalyticsRepo{
		tenantStats: &TenantDeliveryStats{
			TenantID:        "tenant-1",
			TotalDeliveries: 1000,
			SuccessCount:    700,
			FailureCount:    300,
			EndpointCount:   5,
			TopErrors:       map[string]int{"timeout": 200, "connection_refused": 50, "auth_failed": 50},
		},
	}
	analyzer := NewDeliveryAnalyzer(repo, nil)

	insights, err := analyzer.IdentifyBottlenecks(context.Background(), "tenant-1")
	require.NoError(t, err)
	assert.NotEmpty(t, insights)

	// Should detect high failure rate
	foundHighFailRate := false
	foundDominantError := false
	for _, insight := range insights {
		if insight.Title == "High Overall Failure Rate" {
			foundHighFailRate = true
		}
		if insight.Title == "Dominant Error Type: timeout" {
			foundDominantError = true
		}
	}
	assert.True(t, foundHighFailRate, "expected high failure rate insight")
	assert.True(t, foundDominantError, "expected dominant error type insight")
}

func TestAnalyzer_GenerateInsights(t *testing.T) {
	t.Parallel()

	trends := make([]TrendDataPoint, 8)
	for i := range trends {
		trends[i] = TrendDataPoint{
			Timestamp:    time.Now().Add(-time.Duration(8-i) * time.Hour),
			SuccessCount: 95,
			FailureCount: 5,
			TotalCount:   100,
			AvgLatencyMs: 200,
		}
	}

	repo := &mockAnalyticsRepo{
		trends: trends,
		tenantStats: &TenantDeliveryStats{
			TenantID:        "tenant-1",
			TotalDeliveries: 1000,
			SuccessCount:    990,
			FailureCount:    10,
			EndpointCount:   5,
			TopErrors:       map[string]int{"timeout": 10},
		},
	}
	analyzer := NewDeliveryAnalyzer(repo, nil)

	insights, err := analyzer.GenerateInsights(context.Background(), "tenant-1")
	require.NoError(t, err)
	assert.NotNil(t, insights)
}

// --- Helper function tests ---

func TestIsIncreasing(t *testing.T) {
	t.Parallel()

	assert.True(t, isIncreasing([]float64{1, 2, 3, 4, 5}))
	assert.False(t, isIncreasing([]float64{5, 4, 3, 2, 1}))
	assert.False(t, isIncreasing([]float64{1, 2})) // too few points
}

func TestIsDecreasing(t *testing.T) {
	t.Parallel()

	assert.True(t, isDecreasing([]float64{5, 4, 3, 2, 1}))
	assert.False(t, isDecreasing([]float64{1, 2, 3, 4, 5}))
	assert.False(t, isDecreasing([]float64{1, 2})) // too few points
}
