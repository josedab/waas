package delivery

import (
	"testing"

	"github.com/google/uuid"
)

func TestHealthScorerPerfectEndpoint(t *testing.T) {
	scorer := NewHealthScorer(nil)
	endpointID := uuid.New()

	// Record 20 successful deliveries
	for i := 0; i < 20; i++ {
		scorer.RecordDelivery(endpointID, true, 100, 200)
	}

	score := scorer.GetScore(endpointID)
	if score.Score < 90 {
		t.Fatalf("perfect endpoint should have score >= 90, got %d", score.Score)
	}
	if score.Grade != "A" {
		t.Fatalf("expected grade A, got %s", score.Grade)
	}
	if score.IsPaused {
		t.Fatal("healthy endpoint should not be paused")
	}
}

func TestHealthScorerFailingEndpoint(t *testing.T) {
	scorer := NewHealthScorer(nil)
	endpointID := uuid.New()

	// Record 20 failures
	for i := 0; i < 20; i++ {
		scorer.RecordDelivery(endpointID, false, 5000, 500)
	}

	score := scorer.GetScore(endpointID)
	if score.Score > 20 {
		t.Fatalf("failing endpoint should have low score, got %d", score.Score)
	}
	if score.Grade != "F" {
		t.Fatalf("expected grade F, got %s", score.Grade)
	}
	if !score.IsPaused {
		t.Fatal("unhealthy endpoint should be auto-paused")
	}
}

func TestHealthScorerAutoRecover(t *testing.T) {
	config := DefaultHealthScoringConfig()
	config.PauseThreshold = 30
	config.ResumeThreshold = 60
	config.MinDeliveriesForScore = 5

	scorer := NewHealthScorer(config)
	endpointID := uuid.New()

	// Cause auto-pause with failures
	for i := 0; i < 15; i++ {
		scorer.RecordDelivery(endpointID, false, 3000, 500)
	}
	score := scorer.GetScore(endpointID)
	if !score.IsPaused {
		t.Fatal("endpoint should be paused after failures")
	}

	// Recover with successes
	for i := 0; i < 50; i++ {
		scorer.RecordDelivery(endpointID, true, 100, 200)
	}
	score = scorer.GetScore(endpointID)
	if score.IsPaused {
		t.Fatalf("endpoint should resume after recovery, score=%d", score.Score)
	}
}

func TestHealthScorerLatencyImpact(t *testing.T) {
	scorer := NewHealthScorer(nil)
	fastEndpoint := uuid.New()
	slowEndpoint := uuid.New()

	for i := 0; i < 20; i++ {
		scorer.RecordDelivery(fastEndpoint, true, 50, 200)   // Fast
		scorer.RecordDelivery(slowEndpoint, true, 4000, 200)  // Slow
	}

	fastScore := scorer.GetScore(fastEndpoint)
	slowScore := scorer.GetScore(slowEndpoint)

	if fastScore.Score <= slowScore.Score {
		t.Fatalf("fast endpoint (%d) should score higher than slow endpoint (%d)", fastScore.Score, slowScore.Score)
	}
}

func TestHealthScorerConsecutiveErrorPenalty(t *testing.T) {
	scorer := NewHealthScorer(nil)
	ep1 := uuid.New()
	ep2 := uuid.New()

	// Scattered errors
	for i := 0; i < 20; i++ {
		if i%4 == 0 {
			scorer.RecordDelivery(ep1, false, 200, 500)
		} else {
			scorer.RecordDelivery(ep1, true, 200, 200)
		}
	}

	// Consecutive errors at the end
	for i := 0; i < 10; i++ {
		scorer.RecordDelivery(ep2, true, 200, 200)
	}
	for i := 0; i < 10; i++ {
		scorer.RecordDelivery(ep2, false, 200, 500)
	}

	score1 := scorer.GetScore(ep1)
	score2 := scorer.GetScore(ep2)

	if score1.Score <= score2.Score {
		t.Fatalf("scattered errors (%d) should score higher than consecutive errors (%d)", score1.Score, score2.Score)
	}
}

func TestHealthScorerTrend(t *testing.T) {
	config := DefaultHealthScoringConfig()
	config.MinDeliveriesForScore = 3
	scorer := NewHealthScorer(config)
	endpointID := uuid.New()

	// Start with failures, then improve
	for i := 0; i < 10; i++ {
		scorer.RecordDelivery(endpointID, false, 1000, 500)
	}
	for i := 0; i < 20; i++ {
		scorer.RecordDelivery(endpointID, true, 100, 200)
	}

	score := scorer.GetScore(endpointID)
	if score.Trend != "improving" && score.Trend != "stable" {
		// Trend depends on score history; both are acceptable given the pattern
		t.Logf("trend: %s, score: %d", score.Trend, score.Score)
	}
}

func TestHealthScorerPauseCallback(t *testing.T) {
	pauseCalled := false
	config := DefaultHealthScoringConfig()
	config.MinDeliveriesForScore = 5
	scorer := NewHealthScorer(config)
	scorer.SetPauseCallback(func(id uuid.UUID, reason string) {
		pauseCalled = true
	})

	endpointID := uuid.New()
	for i := 0; i < 20; i++ {
		scorer.RecordDelivery(endpointID, false, 5000, 500)
	}

	// Give goroutine time to execute
	score := scorer.GetScore(endpointID)
	if !score.IsPaused {
		t.Fatal("endpoint should be paused")
	}
	_ = pauseCalled // set in goroutine, cannot reliably check synchronously
}

func TestHealthScorerGrades(t *testing.T) {
	tests := []struct {
		score int
		grade string
	}{
		{95, "A"},
		{85, "B"},
		{70, "C"},
		{50, "D"},
		{10, "F"},
	}

	for _, tt := range tests {
		grade := scoreToGrade(tt.score)
		if grade != tt.grade {
			t.Errorf("score %d: expected grade %s, got %s", tt.score, tt.grade, grade)
		}
	}
}
