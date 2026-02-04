package smartlimit

import (
	"context"
	"testing"
	"time"
)

func seedLearningData(repo *memoryRepository, tenantID, endpointID string, count int) {
	ctx := context.Background()
	base := time.Now().Add(-7 * 24 * time.Hour)
	for i := 0; i < count; i++ {
		ts := base.Add(time.Duration(i) * time.Hour)
		point := &LearningDataPoint{
			ID:           "ldp-" + string(rune(i)),
			TenantID:     tenantID,
			EndpointID:   endpointID,
			Timestamp:    ts,
			HourOfDay:    ts.Hour(),
			DayOfWeek:    int(ts.Weekday()),
			RequestRate:  float64(10 + i%5),
			SuccessRate:  0.95,
			AvgLatency:   float64(100 + i%50),
			RateLimited:  i%20 == 0,
			ResponseCode: 200,
		}
		if i%20 == 0 {
			point.ResponseCode = 429
			point.SuccessRate = 0.0
		}
		repo.SaveLearningData(ctx, point)
	}
}

func TestPatternAnalyzerInsufficientData(t *testing.T) {
	repo := newMemoryRepository()
	pa := NewPatternAnalyzer(repo, DefaultPatternConfig())
	ctx := context.Background()

	// With no data, should return nil
	pattern, err := pa.AnalyzeEndpoint(ctx, "t1", "ep1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pattern != nil {
		t.Fatal("expected nil pattern with no data")
	}
}

func TestPatternAnalyzerWithData(t *testing.T) {
	repo := newMemoryRepository()
	config := DefaultPatternConfig()
	config.MinSamplesForPattern = 10
	pa := NewPatternAnalyzer(repo, config)
	ctx := context.Background()

	seedLearningData(repo, "t1", "ep1", 100)

	pattern, err := pa.AnalyzeEndpoint(ctx, "t1", "ep1")
	if err != nil {
		t.Fatalf("analysis failed: %v", err)
	}
	if pattern == nil {
		t.Fatal("expected non-nil pattern")
	}
	if pattern.SampleCount < 90 {
		t.Fatalf("expected ~100 samples, got %d", pattern.SampleCount)
	}
	if pattern.AvgSuccessRate <= 0 {
		t.Fatal("expected positive success rate")
	}
	if pattern.PeakRate <= 0 {
		t.Fatal("expected positive peak rate")
	}
	if pattern.ThroughputTrend == "" {
		t.Fatal("expected a throughput trend")
	}
}

func TestPatternAnalyzerHourlyPattern(t *testing.T) {
	repo := newMemoryRepository()
	config := DefaultPatternConfig()
	config.MinSamplesForPattern = 5
	pa := NewPatternAnalyzer(repo, config)
	ctx := context.Background()

	// Seed data with distinct hourly patterns
	seedLearningData(repo, "t1", "ep1", 50)

	pattern, _ := pa.AnalyzeEndpoint(ctx, "t1", "ep1")
	if pattern == nil {
		t.Fatal("expected pattern")
	}

	// At least some hours should have non-zero rates
	nonZero := 0
	for _, r := range pattern.HourlyRates {
		if r > 0 {
			nonZero++
		}
	}
	if nonZero == 0 {
		t.Fatal("expected at least some non-zero hourly rates")
	}
}

func TestGetRecommendationWithPattern(t *testing.T) {
	repo := newMemoryRepository()
	config := DefaultPatternConfig()
	config.MinSamplesForPattern = 10
	config.UpdateInterval = 1 * time.Hour
	pa := NewPatternAnalyzer(repo, config)
	ctx := context.Background()

	seedLearningData(repo, "t1", "ep1", 100)

	rateConfig := &AdaptiveRateConfig{
		EndpointID:     "ep1",
		BaseRatePerSec: 10,
		MinRatePerSec:  1,
		MaxRatePerSec:  100,
	}

	rec, err := pa.GetRecommendation(ctx, "t1", "ep1", rateConfig)
	if err != nil {
		t.Fatalf("recommendation failed: %v", err)
	}
	if rec == nil {
		t.Fatal("expected recommendation")
	}
	if rec.RecommendedRate <= 0 {
		t.Fatal("expected positive recommended rate")
	}
	if rec.RecommendedRate < rateConfig.MinRatePerSec || rec.RecommendedRate > rateConfig.MaxRatePerSec {
		t.Fatalf("rate %f outside bounds [%f, %f]", rec.RecommendedRate, rateConfig.MinRatePerSec, rateConfig.MaxRatePerSec)
	}
	if rec.Confidence <= 0 {
		t.Fatal("expected positive confidence")
	}
}

func TestGetRecommendationWithoutData(t *testing.T) {
	repo := newMemoryRepository()
	pa := NewPatternAnalyzer(repo, DefaultPatternConfig())
	ctx := context.Background()

	rateConfig := &AdaptiveRateConfig{
		EndpointID:     "ep1",
		BaseRatePerSec: 10,
		MinRatePerSec:  1,
		MaxRatePerSec:  100,
	}

	rec, err := pa.GetRecommendation(ctx, "t1", "ep1", rateConfig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec == nil {
		t.Fatal("expected fallback recommendation")
	}
	if rec.RecommendedRate != 10 {
		t.Fatalf("expected base rate 10, got %f", rec.RecommendedRate)
	}
	if rec.Confidence >= 0.5 {
		t.Fatal("expected low confidence without data")
	}
}

func TestApplyRecommendation(t *testing.T) {
	config := &AdaptiveRateConfig{
		EndpointID:     "ep1",
		BaseRatePerSec: 10,
		MinRatePerSec:  1,
		MaxRatePerSec:  100,
		BurstSize:      10,
	}
	throttler := NewThrottler(config)

	initialRate := throttler.currentRate

	rec := &RateRecommendation{
		RecommendedRate: 50.0,
		Confidence:      1.0,
	}

	throttler.ApplyRecommendation(rec)

	// Should blend toward the recommendation
	if throttler.currentRate == initialRate {
		t.Fatal("rate should have changed after applying recommendation")
	}
	if throttler.currentRate <= initialRate {
		t.Fatal("rate should have increased toward 50")
	}
}

func TestPatternCaching(t *testing.T) {
	repo := newMemoryRepository()
	config := DefaultPatternConfig()
	config.MinSamplesForPattern = 5
	config.UpdateInterval = 1 * time.Hour
	pa := NewPatternAnalyzer(repo, config)
	ctx := context.Background()

	seedLearningData(repo, "t1", "ep1", 50)

	// First call analyzes
	p1, _ := pa.AnalyzeEndpoint(ctx, "t1", "ep1")
	if p1 == nil {
		t.Fatal("expected pattern")
	}

	// Should be cached now
	pa.mu.RLock()
	cached, exists := pa.cache["t1:ep1"]
	pa.mu.RUnlock()
	if !exists || cached == nil {
		t.Fatal("expected cached pattern")
	}
}

func TestErrorBurstDetection(t *testing.T) {
	repo := newMemoryRepository()
	config := DefaultPatternConfig()
	config.MinSamplesForPattern = 5
	pa := NewPatternAnalyzer(repo, config)
	ctx := context.Background()

	// Seed data with a burst of errors
	base := time.Now().Add(-24 * time.Hour)
	for i := 0; i < 30; i++ {
		ts := base.Add(time.Duration(i) * time.Hour)
		point := &LearningDataPoint{
			ID:         "ldp",
			TenantID:   "t1",
			EndpointID: "ep1",
			Timestamp:  ts,
			HourOfDay:  ts.Hour(),
			DayOfWeek:  int(ts.Weekday()),
		}
		if i >= 10 && i < 25 {
			// 15 consecutive errors
			point.RateLimited = true
			point.ResponseCode = 429
			point.SuccessRate = 0
		} else {
			point.ResponseCode = 200
			point.SuccessRate = 1.0
			point.RequestRate = 10
		}
		repo.SaveLearningData(ctx, point)
	}

	pattern, _ := pa.AnalyzeEndpoint(ctx, "t1", "ep1")
	if pattern == nil {
		t.Fatal("expected pattern")
	}
	if pattern.ErrorBurstScore <= 0 {
		t.Fatal("expected non-zero error burst score")
	}
}
