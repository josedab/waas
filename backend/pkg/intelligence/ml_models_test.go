package intelligence

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGradientBoostingModel_Train(t *testing.T) {
	model := NewGradientBoostingModel(10, 0.1)

	samples := []TrainingSample{
		{Features: FeatureVector{FailureRate24h: 0.8, ConsecutiveFailures: 10, P99LatencyMs: 8000}, Label: 1.0},
		{Features: FeatureVector{FailureRate24h: 0.9, ConsecutiveFailures: 15, P99LatencyMs: 12000}, Label: 1.0},
		{Features: FeatureVector{FailureRate24h: 0.7, ConsecutiveFailures: 8, P99LatencyMs: 6000}, Label: 1.0},
		{Features: FeatureVector{FailureRate24h: 0.01, ConsecutiveFailures: 0, P99LatencyMs: 100}, Label: 0.0},
		{Features: FeatureVector{FailureRate24h: 0.02, ConsecutiveFailures: 0, P99LatencyMs: 200}, Label: 0.0},
		{Features: FeatureVector{FailureRate24h: 0.05, ConsecutiveFailures: 1, P99LatencyMs: 500}, Label: 0.0},
	}

	err := model.Train(samples)
	require.NoError(t, err)

	metrics := model.GetMetrics()
	assert.True(t, metrics.SampleCount == 6)
	assert.True(t, metrics.Accuracy > 0)
}

func TestGradientBoostingModel_TrainTooFewSamples(t *testing.T) {
	model := NewGradientBoostingModel(5, 0.1)
	err := model.Train([]TrainingSample{
		{Features: FeatureVector{FailureRate24h: 0.5}, Label: 1.0},
	})
	assert.Error(t, err)
}

func TestGradientBoostingModel_Predict(t *testing.T) {
	model := NewGradientBoostingModel(10, 0.1)

	samples := []TrainingSample{
		{Features: FeatureVector{FailureRate24h: 0.9, ConsecutiveFailures: 10}, Label: 1.0},
		{Features: FeatureVector{FailureRate24h: 0.8, ConsecutiveFailures: 8}, Label: 1.0},
		{Features: FeatureVector{FailureRate24h: 0.7, ConsecutiveFailures: 6}, Label: 1.0},
		{Features: FeatureVector{FailureRate24h: 0.01, ConsecutiveFailures: 0}, Label: 0.0},
		{Features: FeatureVector{FailureRate24h: 0.02, ConsecutiveFailures: 0}, Label: 0.0},
		{Features: FeatureVector{FailureRate24h: 0.03, ConsecutiveFailures: 0}, Label: 0.0},
	}

	_ = model.Train(samples)

	// High failure features should predict high probability
	highRisk := model.Predict(&FeatureVector{FailureRate24h: 0.85, ConsecutiveFailures: 12})
	lowRisk := model.Predict(&FeatureVector{FailureRate24h: 0.01, ConsecutiveFailures: 0})

	assert.True(t, highRisk > lowRisk, "high risk (%.2f) should be > low risk (%.2f)", highRisk, lowRisk)
}

func TestGradientBoostingModel_PredictUntrained(t *testing.T) {
	model := NewGradientBoostingModel(5, 0.1)
	prob := model.Predict(&FeatureVector{FailureRate24h: 0.5})
	assert.Equal(t, 0.5, prob) // Neutral when untrained
}

func TestGradientBoostingModel_OnlineUpdate(t *testing.T) {
	model := NewGradientBoostingModel(5, 0.1)

	samples := make([]TrainingSample, 10)
	for i := 0; i < 5; i++ {
		samples[i] = TrainingSample{Features: FeatureVector{FailureRate24h: 0.8 + float64(i)*0.02}, Label: 1.0}
		samples[i+5] = TrainingSample{Features: FeatureVector{FailureRate24h: 0.01 + float64(i)*0.01}, Label: 0.0}
	}

	_ = model.Train(samples)
	initialCount := model.GetMetrics().SampleCount

	model.OnlineUpdate(TrainingSample{
		Features: FeatureVector{FailureRate24h: 0.95, ConsecutiveFailures: 20},
		Label:    1.0,
	})

	assert.Equal(t, initialCount+1, model.GetMetrics().SampleCount)
}

func TestAdaptiveRetryTuner_RecordAndGet(t *testing.T) {
	tuner := NewAdaptiveRetryTuner()

	// Simulate delivery outcomes
	tuner.RecordDeliveryOutcome("ep-1", 1, false)
	tuner.RecordDeliveryOutcome("ep-1", 2, true)
	tuner.RecordDeliveryOutcome("ep-1", 1, false)
	tuner.RecordDeliveryOutcome("ep-1", 2, true)
	tuner.RecordDeliveryOutcome("ep-1", 1, true)

	profile := tuner.GetRetryProfile("ep-1")
	assert.Equal(t, "ep-1", profile.EndpointID)
	assert.True(t, profile.SampleCount == 5)
	assert.True(t, profile.OptimalRetries >= 1)
}

func TestAdaptiveRetryTuner_DefaultProfile(t *testing.T) {
	tuner := NewAdaptiveRetryTuner()
	profile := tuner.GetRetryProfile("unknown-endpoint")

	assert.Equal(t, 3, profile.OptimalRetries)
	assert.Equal(t, "exponential", profile.OptimalBackoff)
}

func TestAutoRemediator_ExecuteRemediation(t *testing.T) {
	ar := NewAutoRemediator()
	ctx := context.Background()

	tests := []struct {
		actionType string
		expectErr  bool
	}{
		{"adjust_retry", false},
		{"circuit_breaker", false},
		{"rate_limit", false},
		{"pause_delivery", false},
		{"alert_owner", false},
		{"unknown_action", true},
	}

	for _, tt := range tests {
		t.Run(tt.actionType, func(t *testing.T) {
			action, err := ar.ExecuteRemediation(ctx, "tenant-1", "ep-1", tt.actionType, "high_failure_rate")
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "completed", action.Status)
				assert.NotEmpty(t, action.Result)
				assert.NotNil(t, action.ExecutedAt)
				assert.NotNil(t, action.CompletedAt)
			}
		})
	}
}

func TestAutoRemediator_GetActions(t *testing.T) {
	ar := NewAutoRemediator()
	ctx := context.Background()

	_, _ = ar.ExecuteRemediation(ctx, "tenant-1", "ep-1", "adjust_retry", "test")
	_, _ = ar.ExecuteRemediation(ctx, "tenant-1", "ep-2", "circuit_breaker", "test")
	_, _ = ar.ExecuteRemediation(ctx, "tenant-2", "ep-3", "rate_limit", "test")

	actions := ar.GetActions("tenant-1")
	assert.Equal(t, 2, len(actions))

	actions2 := ar.GetActions("tenant-2")
	assert.Equal(t, 1, len(actions2))
}

func TestAutoRemediator_NotificationTargets(t *testing.T) {
	ar := NewAutoRemediator()

	ar.RegisterNotificationTarget(NotificationTarget{
		ID:       "nt-1",
		TenantID: "tenant-1",
		Type:     "slack",
		Config:   map[string]string{"webhook_url": "https://hooks.slack.com/..."},
		Enabled:  true,
	})

	ar.RegisterNotificationTarget(NotificationTarget{
		ID:       "nt-2",
		TenantID: "tenant-1",
		Type:     "pagerduty",
		Config:   map[string]string{"routing_key": "pk_..."},
		Enabled:  true,
	})

	ar.mu.RLock()
	targets := ar.targets["tenant-1"]
	ar.mu.RUnlock()
	assert.Equal(t, 2, len(targets))
}

func TestSigmoid(t *testing.T) {
	assert.InDelta(t, 0.5, sigmoid(0), 0.001)
	assert.True(t, sigmoid(5) > 0.99)
	assert.True(t, sigmoid(-5) < 0.01)
}

func TestFeatureToSlice(t *testing.T) {
	f := &FeatureVector{
		AvgLatencyMs:        100,
		P99LatencyMs:        500,
		FailureRate24h:      0.1,
		FailureRate7d:       0.05,
		ConsecutiveFailures: 3,
		ResponseTimetrend:   0.2,
		PayloadSizeAvg:      1024,
		RequestsPerMinute:   50,
		LastSuccessAgo:      300,
		EndpointAge:         86400,
		SSLDaysRemaining:    30,
		ErrorDiversity:      2,
	}

	slice := featureToSlice(f)
	assert.Equal(t, 12, len(slice))
	assert.Equal(t, 100.0, slice[0])
	assert.Equal(t, 0.1, slice[2])
}

// Ensure no unused imports
var _ = time.Now
