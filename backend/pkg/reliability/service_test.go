package reliability

import (
	"testing"
)

func TestComputeReliabilityScore(t *testing.T) {
	tests := []struct {
		name       string
		stats      *DeliveryStats
		wantMin    float64
		wantMax    float64
		wantStatus ScoreStatus
	}{
		{
			name:       "no data returns 100",
			stats:      &DeliveryStats{},
			wantMin:    100,
			wantMax:    100,
			wantStatus: ScoreStatusHealthy,
		},
		{
			name: "perfect delivery",
			stats: &DeliveryStats{
				TotalAttempts:      100,
				SuccessfulAttempts: 100,
				FailedAttempts:     0,
				LatencyP95Ms:       200,
			},
			wantMin:    95,
			wantMax:    100,
			wantStatus: ScoreStatusHealthy,
		},
		{
			name: "degraded delivery",
			stats: &DeliveryStats{
				TotalAttempts:       100,
				SuccessfulAttempts:  80,
				FailedAttempts:      20,
				LatencyP95Ms:        2000,
				ConsecutiveFailures: 2,
			},
			wantMin:    40,
			wantMax:    85,
			wantStatus: ScoreStatusDegraded,
		},
		{
			name: "critical delivery",
			stats: &DeliveryStats{
				TotalAttempts:       100,
				SuccessfulAttempts:  30,
				FailedAttempts:      70,
				LatencyP95Ms:        8000,
				ConsecutiveFailures: 8,
			},
			wantMin:    0,
			wantMax:    40,
			wantStatus: ScoreStatusCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := computeReliabilityScore(tt.stats)
			status := scoreToStatus(score)

			if score < tt.wantMin || score > tt.wantMax {
				t.Errorf("score = %v, want between %v and %v", score, tt.wantMin, tt.wantMax)
			}
			if status != tt.wantStatus {
				t.Errorf("status = %v, want %v (score=%v)", status, tt.wantStatus, score)
			}
		})
	}
}

func TestScoreToStatus(t *testing.T) {
	tests := []struct {
		score float64
		want  ScoreStatus
	}{
		{100, ScoreStatusHealthy},
		{95, ScoreStatusHealthy},
		{94.99, ScoreStatusDegraded},
		{70, ScoreStatusDegraded},
		{69.99, ScoreStatusCritical},
		{1, ScoreStatusCritical},
		{0, ScoreStatusUnknown},
	}

	for _, tt := range tests {
		got := scoreToStatus(tt.score)
		if got != tt.want {
			t.Errorf("scoreToStatus(%v) = %v, want %v", tt.score, got, tt.want)
		}
	}
}

func TestSafePercent(t *testing.T) {
	if got := safePercent(0, 0); got != 100.0 {
		t.Errorf("safePercent(0,0) = %v, want 100.0", got)
	}
	if got := safePercent(50, 100); got != 50.0 {
		t.Errorf("safePercent(50,100) = %v, want 50.0", got)
	}
}
