package prediction

import (
	"testing"
	"time"
)

func TestNewDeliveryOptimizer(t *testing.T) {
	opt := NewDeliveryOptimizer(nil)
	if opt == nil {
		t.Fatal("expected non-nil optimizer")
	}
	if opt.config.MinSamplesForOptimization != 10 {
		t.Errorf("expected default min samples 10, got %d", opt.config.MinSamplesForOptimization)
	}
}

func TestRecordOutcomeAndProfile(t *testing.T) {
	opt := NewDeliveryOptimizer(nil)
	now := time.Now()

	// Record 10 successful deliveries
	for i := 0; i < 10; i++ {
		opt.RecordOutcome(DeliveryOutcome{
			EndpointID:     "ep-1",
			Success:        true,
			ResponseTimeMs: 150 + i*10,
			StatusCode:     200,
			Timestamp:      now,
		})
	}

	profile := opt.GetProfile("ep-1")
	if profile == nil {
		t.Fatal("expected profile to exist")
	}
	if profile.TotalAttempts != 10 {
		t.Errorf("expected 10 attempts, got %d", profile.TotalAttempts)
	}
	if profile.SuccessCount != 10 {
		t.Errorf("expected 10 successes, got %d", profile.SuccessCount)
	}
	if profile.HealthScore != 100 {
		t.Errorf("expected health score 100, got %f", profile.HealthScore)
	}
}

func TestRecordFailure(t *testing.T) {
	opt := NewDeliveryOptimizer(nil)
	now := time.Now()

	opt.RecordOutcome(DeliveryOutcome{
		EndpointID:    "ep-fail",
		Success:       false,
		StatusCode:    503,
		ErrorCategory: "server_error",
		Timestamp:     now,
	})

	profile := opt.GetProfile("ep-fail")
	if profile.FailureCount != 1 {
		t.Errorf("expected 1 failure, got %d", profile.FailureCount)
	}
	if profile.FailureCategories["server_error"] != 1 {
		t.Error("expected server_error category to be tracked")
	}
}

func TestGetRecommendationNoHistory(t *testing.T) {
	opt := NewDeliveryOptimizer(nil)
	rec := opt.GetRecommendation("unknown-ep", 0)

	if !rec.ShouldRetry {
		t.Error("expected recommendation to retry for unknown endpoint")
	}
	if rec.Confidence > 0.5 {
		t.Errorf("expected low confidence for unknown endpoint, got %f", rec.Confidence)
	}
}

func TestGetRecommendationHealthyEndpoint(t *testing.T) {
	opt := NewDeliveryOptimizer(nil)
	now := time.Now()

	// Build profile with high success rate
	for i := 0; i < 100; i++ {
		opt.RecordOutcome(DeliveryOutcome{
			EndpointID:     "ep-healthy",
			Success:        true,
			ResponseTimeMs: 100,
			StatusCode:     200,
			Timestamp:      now,
		})
	}

	rec := opt.GetRecommendation("ep-healthy", 0)
	if !rec.ShouldRetry {
		t.Error("expected should retry for healthy endpoint on first attempt")
	}
	if rec.PredictedSuccessRate < 0.9 {
		t.Errorf("expected high predicted success, got %f", rec.PredictedSuccessRate)
	}
	if rec.MaxRetries > 5 {
		t.Errorf("expected low max retries for healthy endpoint, got %d", rec.MaxRetries)
	}
}

func TestGetRecommendationUnhealthyEndpoint(t *testing.T) {
	opt := NewDeliveryOptimizer(nil)
	now := time.Now()

	// Build profile with low success rate
	for i := 0; i < 50; i++ {
		opt.RecordOutcome(DeliveryOutcome{
			EndpointID:    "ep-unhealthy",
			Success:       i < 10, // Only 20% success
			ResponseTimeMs: 500,
			StatusCode:    503,
			ErrorCategory: "timeout",
			Timestamp:     now,
		})
	}

	rec := opt.GetRecommendation("ep-unhealthy", 0)
	if rec.MaxRetries < 7 {
		t.Errorf("expected higher max retries for unhealthy endpoint, got %d", rec.MaxRetries)
	}
}

func TestDetectAnomalies(t *testing.T) {
	opt := NewDeliveryOptimizer(nil)
	now := time.Now()

	// Build profile with failures
	for i := 0; i < 20; i++ {
		opt.RecordOutcome(DeliveryOutcome{
			EndpointID:    "ep-anomaly",
			Success:       i < 5, // 25% success
			ResponseTimeMs: 200,
			StatusCode:    503,
			ErrorCategory: "timeout",
			Timestamp:     now,
		})
	}

	anomalies := opt.DetectAnomalies("ep-anomaly")
	if len(anomalies) == 0 {
		t.Fatal("expected at least one anomaly")
	}

	foundRateDrop := false
	foundPattern := false
	for _, a := range anomalies {
		if a.Type == "success_rate_drop" {
			foundRateDrop = true
		}
		if a.Type == "dominant_failure_pattern" {
			foundPattern = true
		}
	}
	if !foundRateDrop {
		t.Error("expected success_rate_drop anomaly")
	}
	if !foundPattern {
		t.Error("expected dominant_failure_pattern anomaly")
	}
}

func TestDetectAnomaliesInsufficientData(t *testing.T) {
	opt := NewDeliveryOptimizer(nil)
	anomalies := opt.DetectAnomalies("nonexistent")
	if anomalies != nil {
		t.Error("expected nil anomalies for unknown endpoint")
	}
}

func TestListProfiles(t *testing.T) {
	opt := NewDeliveryOptimizer(nil)
	now := time.Now()

	// Create two profiles with different health scores
	for i := 0; i < 10; i++ {
		opt.RecordOutcome(DeliveryOutcome{
			EndpointID: "ep-good", Success: true, ResponseTimeMs: 100, Timestamp: now,
		})
		opt.RecordOutcome(DeliveryOutcome{
			EndpointID: "ep-bad", Success: i < 3, ResponseTimeMs: 500, Timestamp: now,
		})
	}

	profiles := opt.ListProfiles()
	if len(profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(profiles))
	}
	// Should be sorted by health score (good first)
	if profiles[0].EndpointID != "ep-good" {
		t.Error("expected ep-good to be first (highest health)")
	}
}

func TestSeverityFromRate(t *testing.T) {
	tests := []struct {
		rate     float64
		expected string
	}{
		{0.1, "critical"},
		{0.4, "high"},
		{0.6, "medium"},
		{0.75, "low"},
	}
	for _, tt := range tests {
		got := severityFromRate(tt.rate)
		if got != tt.expected {
			t.Errorf("severityFromRate(%f) = %s, want %s", tt.rate, got, tt.expected)
		}
	}
}
