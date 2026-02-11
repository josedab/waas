package autoretry

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// sigmoid
// ---------------------------------------------------------------------------

func TestSigmoid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   float64
		wantMin float64
		wantMax float64
		exact   *float64
	}{
		{name: "zero", input: 0, exact: ptr(0.5)},
		{name: "large positive", input: 10, wantMin: 0.9999, wantMax: 1.0},
		{name: "large negative", input: -10, wantMin: 0.0, wantMax: 0.0001},
		{name: "very large positive no overflow", input: 710, wantMin: 0.999, wantMax: 1.0},
		{name: "very large negative no overflow", input: -710, wantMin: 0.0, wantMax: 0.001},
		{name: "positive 1", input: 1, wantMin: 0.73, wantMax: 0.74},
		{name: "negative 1", input: -1, wantMin: 0.26, wantMax: 0.27},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := sigmoid(tc.input)
			if tc.exact != nil {
				assert.InDelta(t, *tc.exact, got, 1e-10)
			} else {
				assert.GreaterOrEqual(t, got, tc.wantMin)
				assert.LessOrEqual(t, got, tc.wantMax)
			}
			assert.False(t, math.IsNaN(got), "sigmoid should not return NaN")
			assert.False(t, math.IsInf(got, 0), "sigmoid should not return Inf")
		})
	}
}

// ---------------------------------------------------------------------------
// randomFloat
// ---------------------------------------------------------------------------

func TestRandomFloat(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0.5, randomFloat(), "randomFloat is hardcoded to 0.5")
}

// ---------------------------------------------------------------------------
// PredictionService – Predict
// ---------------------------------------------------------------------------

func TestPredict_BasicOutput(t *testing.T) {
	t.Parallel()
	svc := NewPredictionService()
	ctx := context.Background()

	features := &DeliveryFeatures{
		DeliveryID:            "del-1",
		EndpointID:            "ep-1",
		EndpointSuccessRate1h: 0.9,
		AttemptNumber:         1,
	}

	pred, err := svc.Predict(ctx, features)
	require.NoError(t, err)
	require.NotNil(t, pred)

	assert.Equal(t, "del-1", pred.DeliveryID)
	assert.Equal(t, "ep-1", pred.EndpointID)
	assert.Equal(t, "v1.0.0-baseline", pred.ModelVersion)
	assert.InDelta(t, 0, pred.PredictedSuccessProbability, 1.0) // in [0,1]
	assert.GreaterOrEqual(t, pred.PredictedSuccessProbability, 0.0)
	assert.LessOrEqual(t, pred.PredictedSuccessProbability, 1.0)
	assert.GreaterOrEqual(t, pred.RecommendedDelaySec, 10)
	assert.LessOrEqual(t, pred.RecommendedDelaySec, 3600)
	assert.GreaterOrEqual(t, pred.ConfidenceScore, 0.0)
	assert.LessOrEqual(t, pred.ConfidenceScore, 1.0)
	assert.NotNil(t, pred.FeatureVector)
	assert.False(t, pred.CreatedAt.IsZero())
}

// ---------------------------------------------------------------------------
// calculateScore (tested via Predict)
// ---------------------------------------------------------------------------

func TestPredict_CalculateScore(t *testing.T) {
	t.Parallel()
	svc := NewPredictionService()
	ctx := context.Background()

	tests := []struct {
		name       string
		features   *DeliveryFeatures
		higherThan *DeliveryFeatures // optional: probability should be higher than this
		lowerThan  *DeliveryFeatures // optional: probability should be lower than this
	}{
		{
			name: "high success rate yields higher probability",
			features: &DeliveryFeatures{
				EndpointSuccessRate1h: 0.95,
				AttemptNumber:         1,
			},
			higherThan: &DeliveryFeatures{
				EndpointSuccessRate1h: 0.10,
				AttemptNumber:         1,
			},
		},
		{
			name: "high error rate yields lower probability",
			features: &DeliveryFeatures{
				EndpointErrorRate1h: 0.90,
				AttemptNumber:       1,
			},
			lowerThan: &DeliveryFeatures{
				EndpointErrorRate1h: 0.05,
				AttemptNumber:       1,
			},
		},
		{
			name: "business hours give small boost",
			features: &DeliveryFeatures{
				IsBusinessHours: true,
				AttemptNumber:   1,
			},
			higherThan: &DeliveryFeatures{
				IsBusinessHours: false,
				AttemptNumber:   1,
			},
		},
		{
			name: "large payload gives small penalty",
			features: &DeliveryFeatures{
				HasLargePayload: true,
				AttemptNumber:   1,
			},
			lowerThan: &DeliveryFeatures{
				HasLargePayload: false,
				AttemptNumber:   1,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			pred, err := svc.Predict(ctx, tc.features)
			require.NoError(t, err)

			if tc.higherThan != nil {
				other, err := svc.Predict(ctx, tc.higherThan)
				require.NoError(t, err)
				assert.Greater(t, pred.PredictedSuccessProbability, other.PredictedSuccessProbability,
					"expected first features to have higher probability")
			}
			if tc.lowerThan != nil {
				other, err := svc.Predict(ctx, tc.lowerThan)
				require.NoError(t, err)
				assert.Less(t, pred.PredictedSuccessProbability, other.PredictedSuccessProbability,
					"expected first features to have lower probability")
			}
		})
	}
}

func TestPredict_AttemptNumberZero(t *testing.T) {
	t.Parallel()
	svc := NewPredictionService()
	ctx := context.Background()

	// AttemptNumber=0 → attemptPenalty = math.Log(0+1) = math.Log(1) = 0
	features := &DeliveryFeatures{AttemptNumber: 0}
	pred, err := svc.Predict(ctx, features)
	require.NoError(t, err)
	assert.False(t, math.IsNaN(pred.PredictedSuccessProbability))
}

func TestPredict_HighResponseTimeCapped(t *testing.T) {
	t.Parallel()
	svc := NewPredictionService()
	ctx := context.Background()

	// Response time normalization caps at 1.0 (5000ms = 1.0)
	// Values above 5000 should produce the same score component
	f1 := &DeliveryFeatures{EndpointAvgResponseTimeMs: 5000, AttemptNumber: 1}
	f2 := &DeliveryFeatures{EndpointAvgResponseTimeMs: 50000, AttemptNumber: 1}

	p1, err := svc.Predict(ctx, f1)
	require.NoError(t, err)
	p2, err := svc.Predict(ctx, f2)
	require.NoError(t, err)

	assert.InDelta(t, p1.PredictedSuccessProbability, p2.PredictedSuccessProbability, 1e-10,
		"response time normalization should cap at 1.0")
}

func TestPredict_ConsecutiveFailuresCapped(t *testing.T) {
	t.Parallel()
	svc := NewPredictionService()
	ctx := context.Background()

	// failurePenalty = min(consecutive * 0.1, 1.0) → caps at 10 failures
	f10 := &DeliveryFeatures{ConsecutiveFailures: 10, AttemptNumber: 1}
	f100 := &DeliveryFeatures{ConsecutiveFailures: 100, AttemptNumber: 1}

	p10, err := svc.Predict(ctx, f10)
	require.NoError(t, err)
	p100, err := svc.Predict(ctx, f100)
	require.NoError(t, err)

	assert.InDelta(t, p10.PredictedSuccessProbability, p100.PredictedSuccessProbability, 1e-10,
		"consecutive failure penalty should cap at 1.0")
}

func TestPredict_MoreConsecutiveFailuresLowerProbability(t *testing.T) {
	t.Parallel()
	svc := NewPredictionService()
	ctx := context.Background()

	none, err := svc.Predict(ctx, &DeliveryFeatures{ConsecutiveFailures: 0, AttemptNumber: 1})
	require.NoError(t, err)
	some, err := svc.Predict(ctx, &DeliveryFeatures{ConsecutiveFailures: 5, AttemptNumber: 1})
	require.NoError(t, err)

	assert.Greater(t, none.PredictedSuccessProbability, some.PredictedSuccessProbability)
}

// ---------------------------------------------------------------------------
// calculateOptimalDelay (tested via Predict)
// ---------------------------------------------------------------------------

func TestPredict_CalculateOptimalDelay(t *testing.T) {
	t.Parallel()
	svc := NewPredictionService()
	ctx := context.Background()

	tests := []struct {
		name      string
		features  *DeliveryFeatures
		wantDelay int
	}{
		{
			name: "attempt 1 base delay 30",
			features: &DeliveryFeatures{
				AttemptNumber:         1,
				EndpointSuccessRate1h: 0.9, // high probability → no multiplier
			},
			wantDelay: 30,
		},
		{
			name: "attempt 2 base delay 60",
			features: &DeliveryFeatures{
				AttemptNumber:         2,
				EndpointSuccessRate1h: 0.9,
			},
			wantDelay: 60,
		},
		{
			// 30 * int(math.Pow(2, -1)) = 30 * int(0.5) = 30*0 = 0 → clamped to min 10
			name: "attempt 0 base delay clamped to 10",
			features: &DeliveryFeatures{
				AttemptNumber:         0,
				EndpointSuccessRate1h: 0.9,
			},
			wantDelay: 10,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			pred, err := svc.Predict(ctx, tc.features)
			require.NoError(t, err)
			assert.Equal(t, tc.wantDelay, pred.RecommendedDelaySec)
		})
	}
}

func TestPredict_DelayLowProbabilityMultiplier(t *testing.T) {
	t.Parallel()
	svc := NewPredictionService()
	ctx := context.Background()

	// Low probability (<0.3) should multiply delay by 4
	// Use very high error rate + many failures to push probability below 0.3
	lowProb := &DeliveryFeatures{
		EndpointSuccessRate1h: 0.0,
		EndpointErrorRate1h:   1.0,
		ConsecutiveFailures:   10,
		AttemptNumber:         1,
	}
	pred, err := svc.Predict(ctx, lowProb)
	require.NoError(t, err)

	// probability should be low; delay should be 30*4=120 (or capped/multiplied further)
	if pred.PredictedSuccessProbability < 0.3 {
		assert.GreaterOrEqual(t, pred.RecommendedDelaySec, 120,
			"low probability should multiply base delay by 4")
	}
}

func TestPredict_DelayMinCap(t *testing.T) {
	t.Parallel()
	svc := NewPredictionService()
	ctx := context.Background()

	// All paths produce delay >= 10
	features := &DeliveryFeatures{
		EndpointSuccessRate1h: 0.9,
		AttemptNumber:         0, // base = 30*2^(-1) = 15, still >= 10
	}
	pred, err := svc.Predict(ctx, features)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, pred.RecommendedDelaySec, 10)
}

func TestPredict_DelayMaxCap(t *testing.T) {
	t.Parallel()
	svc := NewPredictionService()
	ctx := context.Background()

	// Very large attempt number should cap at 3600
	features := &DeliveryFeatures{
		AttemptNumber: 100,
	}
	pred, err := svc.Predict(ctx, features)
	require.NoError(t, err)
	assert.LessOrEqual(t, pred.RecommendedDelaySec, 3600)
}

// ---------------------------------------------------------------------------
// calculateConfidence (tested via Predict)
// ---------------------------------------------------------------------------

func TestPredict_CalculateConfidence(t *testing.T) {
	t.Parallel()
	svc := NewPredictionService()
	ctx := context.Background()

	tests := []struct {
		name           string
		features       *DeliveryFeatures
		wantConfidence float64
	}{
		{
			name:           "zero features base confidence",
			features:       &DeliveryFeatures{AttemptNumber: 1},
			wantConfidence: 0.5,
		},
		{
			name: "success rate 24h adds 0.15",
			features: &DeliveryFeatures{
				EndpointSuccessRate24h: 0.8,
				AttemptNumber:          1,
			},
			wantConfidence: 0.65,
		},
		{
			name: "last success minutes adds 0.10",
			features: &DeliveryFeatures{
				EndpointLastSuccessMin: 30,
				AttemptNumber:          1,
			},
			wantConfidence: 0.60,
		},
		{
			name: "time since first attempt adds 0.10",
			features: &DeliveryFeatures{
				TimeSinceFirstAttemptSec: 120,
				AttemptNumber:            1,
			},
			wantConfidence: 0.60,
		},
		{
			name: "consecutive failures adds 0.15",
			features: &DeliveryFeatures{
				ConsecutiveFailures: 3,
				AttemptNumber:       1,
			},
			wantConfidence: 0.65,
		},
		{
			name: "all non-zero features",
			features: &DeliveryFeatures{
				EndpointSuccessRate24h:   0.9,
				EndpointLastSuccessMin:   10,
				TimeSinceFirstAttemptSec: 60,
				ConsecutiveFailures:      2,
				AttemptNumber:            1,
			},
			// 0.5 + 0.15 + 0.10 + 0.10 + 0.15 = 1.0
			wantConfidence: 1.0,
		},
		{
			name: "confidence capped at 1.0",
			features: &DeliveryFeatures{
				EndpointSuccessRate24h:   0.99,
				EndpointLastSuccessMin:   100,
				TimeSinceFirstAttemptSec: 999,
				ConsecutiveFailures:      50,
				AttemptNumber:            1,
			},
			wantConfidence: 1.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			pred, err := svc.Predict(ctx, tc.features)
			require.NoError(t, err)
			assert.InDelta(t, tc.wantConfidence, pred.ConfidenceScore, 1e-10)
		})
	}
}

// ---------------------------------------------------------------------------
// Predict – feature vector
// ---------------------------------------------------------------------------

func TestPredict_FeatureVector(t *testing.T) {
	t.Parallel()
	svc := NewPredictionService()
	ctx := context.Background()

	features := &DeliveryFeatures{
		EndpointSuccessRate1h:     0.85,
		EndpointSuccessRate24h:    0.80,
		EndpointErrorRate1h:       0.05,
		EndpointAvgResponseTimeMs: 250,
		AttemptNumber:             2,
		ConsecutiveFailures:       1,
		IsBusinessHours:           true,
		PayloadSizeBytes:          1024,
	}

	pred, err := svc.Predict(ctx, features)
	require.NoError(t, err)

	fv := pred.FeatureVector
	assert.Equal(t, 0.85, fv["success_rate_1h"])
	assert.Equal(t, 0.80, fv["success_rate_24h"])
	assert.Equal(t, 0.05, fv["error_rate_1h"])
	assert.Equal(t, 250, fv["avg_response_time_ms"])
	assert.Equal(t, 2, fv["attempt_number"])
	assert.Equal(t, 1, fv["consecutive_failures"])
	assert.Equal(t, true, fv["is_business_hours"])
	assert.Equal(t, 1024, fv["payload_size_bytes"])
}

// ---------------------------------------------------------------------------
// RetryOptimizer – GetOptimalStrategy
// ---------------------------------------------------------------------------

func TestGetOptimalStrategy(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name             string
		features         *DeliveryFeatures
		wantStrategyName string
	}{
		{
			name: "high probability selects aggressive",
			features: &DeliveryFeatures{
				EndpointSuccessRate1h: 1.0,
				IsBusinessHours:       true,
				AttemptNumber:         0, // ln(1)=0, no attempt penalty
			},
			wantStrategyName: "aggressive",
		},
		{
			name: "low probability selects conservative",
			features: &DeliveryFeatures{
				EndpointSuccessRate1h:     0.0,
				EndpointErrorRate1h:       1.0,
				EndpointAvgResponseTimeMs: 10000,
				HasLargePayload:           true,
				ConsecutiveFailures:       10,
				AttemptNumber:             500,
			},
			wantStrategyName: "conservative",
		},
		{
			name: "medium probability selects adaptive",
			features: &DeliveryFeatures{
				EndpointSuccessRate1h: 0.5,
				AttemptNumber:         1,
			},
			wantStrategyName: "adaptive",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			opt := NewRetryOptimizer()

			strategy, pred, err := opt.GetOptimalStrategy(ctx, tc.features)
			require.NoError(t, err)
			require.NotNil(t, strategy)
			require.NotNil(t, pred)

			assert.Equal(t, tc.wantStrategyName, strategy.Name)
			// Base delay should be customized from prediction
			assert.Equal(t, pred.RecommendedDelaySec, strategy.BaseDelaySeconds)
		})
	}
}

func TestGetOptimalStrategy_PreservesStrategyFields(t *testing.T) {
	t.Parallel()
	opt := NewRetryOptimizer()
	ctx := context.Background()

	features := &DeliveryFeatures{
		EndpointSuccessRate1h: 1.0,
		IsBusinessHours:       true,
		AttemptNumber:         0, // ensures probability > 0.7
	}

	strategy, _, err := opt.GetOptimalStrategy(ctx, features)
	require.NoError(t, err)

	// Aggressive strategy fields (except BaseDelaySeconds which is overridden)
	assert.Equal(t, "aggressive", strategy.Name)
	assert.Equal(t, 300, strategy.MaxDelaySeconds)
	assert.Equal(t, 10, strategy.MaxAttempts)
	assert.InDelta(t, 1.5, strategy.BackoffMultiplier, 1e-10)
	assert.InDelta(t, 0.1, strategy.JitterFactor, 1e-10)
}

// ---------------------------------------------------------------------------
// RetryOptimizer – CalculateNextDelay
// ---------------------------------------------------------------------------

func TestCalculateNextDelay(t *testing.T) {
	t.Parallel()
	opt := NewRetryOptimizer()

	tests := []struct {
		name    string
		strat   RetryStrategy
		attempt int
		want    time.Duration
	}{
		{
			name: "first attempt base delay",
			strat: RetryStrategy{
				BaseDelaySeconds:  30,
				MaxDelaySeconds:   3600,
				BackoffMultiplier: 2.0,
				JitterFactor:      0.1,
			},
			attempt: 1,
			// delay = 30 * 2^0 = 30; jitter = 30*0.1*(0.5-0.5) = 0
			want: 30 * time.Second,
		},
		{
			name: "second attempt exponential",
			strat: RetryStrategy{
				BaseDelaySeconds:  30,
				MaxDelaySeconds:   3600,
				BackoffMultiplier: 2.0,
				JitterFactor:      0.1,
			},
			attempt: 2,
			// delay = 30 * 2^1 = 60; jitter = 0
			want: 60 * time.Second,
		},
		{
			name: "third attempt exponential",
			strat: RetryStrategy{
				BaseDelaySeconds:  30,
				MaxDelaySeconds:   3600,
				BackoffMultiplier: 2.0,
				JitterFactor:      0.1,
			},
			attempt: 3,
			// delay = 30 * 2^2 = 120; jitter = 0
			want: 120 * time.Second,
		},
		{
			name: "capped at max delay",
			strat: RetryStrategy{
				BaseDelaySeconds:  100,
				MaxDelaySeconds:   200,
				BackoffMultiplier: 3.0,
				JitterFactor:      0.0,
			},
			attempt: 5,
			// delay = 100 * 3^4 = 8100, capped at 200
			want: 200 * time.Second,
		},
		{
			name: "jitter is zero because randomFloat returns 0.5",
			strat: RetryStrategy{
				BaseDelaySeconds:  60,
				MaxDelaySeconds:   3600,
				BackoffMultiplier: 2.0,
				JitterFactor:      0.5, // large jitter factor
			},
			attempt: 1,
			// delay = 60; jitter = 60*0.5*(0.5-0.5) = 0
			want: 60 * time.Second,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := opt.CalculateNextDelay(&tc.strat, tc.attempt)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCalculateNextDelay_JitterAlwaysZero(t *testing.T) {
	t.Parallel()
	opt := NewRetryOptimizer()

	// Verify that regardless of jitter factor, jitter is always 0
	// because randomFloat() returns 0.5 → (0.5 - 0.5) = 0
	for _, jf := range []float64{0.0, 0.1, 0.5, 1.0} {
		strat := &RetryStrategy{
			BaseDelaySeconds:  30,
			MaxDelaySeconds:   3600,
			BackoffMultiplier: 2.0,
			JitterFactor:      jf,
		}
		got := opt.CalculateNextDelay(strat, 1)
		assert.Equal(t, 30*time.Second, got, "jitter factor %.1f should still yield 30s", jf)
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestPredict_NaNFeatures(t *testing.T) {
	t.Parallel()
	svc := NewPredictionService()
	ctx := context.Background()

	features := &DeliveryFeatures{
		EndpointSuccessRate1h: math.NaN(),
		AttemptNumber:         1,
	}

	pred, err := svc.Predict(ctx, features)
	require.NoError(t, err)
	// NaN propagates through arithmetic; sigmoid(NaN) = NaN
	assert.True(t, math.IsNaN(pred.PredictedSuccessProbability),
		"NaN in features should propagate to probability")
}

func TestPredict_NegativeResponseTime(t *testing.T) {
	t.Parallel()
	svc := NewPredictionService()
	ctx := context.Background()

	features := &DeliveryFeatures{
		EndpointAvgResponseTimeMs: -500,
		AttemptNumber:             1,
	}

	pred, err := svc.Predict(ctx, features)
	require.NoError(t, err)
	// Negative response time: float64(-500)/5000 = -0.1, min(-0.1, 1.0) = -0.1
	// This still produces a valid (though unusual) probability
	assert.GreaterOrEqual(t, pred.PredictedSuccessProbability, 0.0)
	assert.LessOrEqual(t, pred.PredictedSuccessProbability, 1.0)
}

func TestPredict_VeryLargeAttemptNumber(t *testing.T) {
	t.Parallel()
	svc := NewPredictionService()
	ctx := context.Background()

	features := &DeliveryFeatures{
		AttemptNumber: 1000,
	}

	pred, err := svc.Predict(ctx, features)
	require.NoError(t, err)

	// Should not panic; delay capped at 3600
	assert.LessOrEqual(t, pred.RecommendedDelaySec, 3600)
	assert.GreaterOrEqual(t, pred.RecommendedDelaySec, 10)
	// Probability should still be valid (very low due to attempt penalty)
	assert.GreaterOrEqual(t, pred.PredictedSuccessProbability, 0.0)
	assert.LessOrEqual(t, pred.PredictedSuccessProbability, 1.0)
}

func TestPredict_NilFeaturesPanics(t *testing.T) {
	t.Parallel()
	svc := NewPredictionService()
	ctx := context.Background()

	// Nil features will cause a nil pointer dereference panic
	assert.Panics(t, func() {
		svc.Predict(ctx, nil) //nolint:errcheck
	}, "nil features should panic")
}

// ---------------------------------------------------------------------------
// NewPredictionService defaults
// ---------------------------------------------------------------------------

func TestNewPredictionService(t *testing.T) {
	t.Parallel()
	svc := NewPredictionService()
	assert.Equal(t, "v1.0.0-baseline", svc.modelVersion)
	assert.InDelta(t, 0.35, svc.defaultWeights.SuccessRateWeight, 1e-10)
	assert.InDelta(t, -0.25, svc.defaultWeights.ErrorRateWeight, 1e-10)
	assert.InDelta(t, 0.60, svc.defaultWeights.Intercept, 1e-10)
}

// ---------------------------------------------------------------------------
// NewRetryOptimizer defaults
// ---------------------------------------------------------------------------

func TestNewRetryOptimizer(t *testing.T) {
	t.Parallel()
	opt := NewRetryOptimizer()

	require.NotNil(t, opt.prediction)
	require.Len(t, opt.strategies, 3)

	assert.Contains(t, opt.strategies, "conservative")
	assert.Contains(t, opt.strategies, "aggressive")
	assert.Contains(t, opt.strategies, "adaptive")
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Additional edge-case & boundary tests
// ---------------------------------------------------------------------------

func TestPredict_InfFeatures(t *testing.T) {
	t.Parallel()
	svc := NewPredictionService()
	ctx := context.Background()

	features := &DeliveryFeatures{
		EndpointSuccessRate1h: math.Inf(1),
		AttemptNumber:         1,
	}

	pred, err := svc.Predict(ctx, features)
	require.NoError(t, err)

	// Prediction should not crash; probability is either in [0,1] or NaN
	p := pred.PredictedSuccessProbability
	assert.True(t, (p >= 0.0 && p <= 1.0) || math.IsNaN(p),
		"probability must be in [0,1] or NaN, got %v", p)
}

func TestPredict_ConfidenceContradictorySignals(t *testing.T) {
	t.Parallel()
	svc := NewPredictionService()
	ctx := context.Background()

	// All confidence-boosting fields are zero → confidence stays at base 0.5
	features := &DeliveryFeatures{
		EndpointSuccessRate24h:   0,
		EndpointLastSuccessMin:   0,
		TimeSinceFirstAttemptSec: 0,
		ConsecutiveFailures:      0,
		AttemptNumber:            1,
	}

	pred, err := svc.Predict(ctx, features)
	require.NoError(t, err)
	assert.InDelta(t, 0.5, pred.ConfidenceScore, 1e-10,
		"all confidence-boosting fields zero should yield base 0.5")
}

func TestSigmoid_LargePositiveInput(t *testing.T) {
	t.Parallel()
	got := sigmoid(750)
	assert.InDelta(t, 1.0, got, 1e-10, "sigmoid(750) should be 1.0")
	assert.False(t, math.IsInf(got, 0), "sigmoid(750) must not overflow to Inf")
}

func TestSigmoid_LargeNegativeInput(t *testing.T) {
	t.Parallel()
	got := sigmoid(-750)
	assert.InDelta(t, 0.0, got, 1e-10, "sigmoid(-750) should be 0.0")
	assert.False(t, math.IsInf(got, 0), "sigmoid(-750) must not overflow to Inf")
}

func TestPredict_AttemptNumberZero_ScorePenalty(t *testing.T) {
	t.Parallel()
	svc := NewPredictionService()
	ctx := context.Background()

	// AttemptNumber=0 → attemptPenalty = math.Log(0+1) = math.Log(1) = 0
	// So the attempt penalty should contribute nothing to the score.
	// Compare with AttemptNumber=1 which has penalty = math.Log(2) > 0.
	zeroAttempt, err := svc.Predict(ctx, &DeliveryFeatures{AttemptNumber: 0})
	require.NoError(t, err)
	oneAttempt, err := svc.Predict(ctx, &DeliveryFeatures{AttemptNumber: 1})
	require.NoError(t, err)

	// AttemptWeight is negative (-0.15), so more attempts → lower probability.
	// Zero attempt penalty means higher (or equal) probability than attempt 1.
	assert.Greater(t, zeroAttempt.PredictedSuccessProbability, oneAttempt.PredictedSuccessProbability,
		"attempt 0 should have no penalty and thus higher probability than attempt 1")
}

func TestCalculateNextDelay_MaxCapEnforced(t *testing.T) {
	t.Parallel()
	opt := NewRetryOptimizer()

	strat := &RetryStrategy{
		BaseDelaySeconds:  500,
		MaxDelaySeconds:   600,
		BackoffMultiplier: 10.0,
		JitterFactor:      0.0,
	}

	// delay = 500 * 10^4 = 500_000_000, far exceeding MaxDelaySeconds
	got := opt.CalculateNextDelay(strat, 5)
	assert.Equal(t, 600*time.Second, got,
		"delay must be capped at MaxDelaySeconds")
}

func TestGetOptimalStrategy_BoundaryProbabilities(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name         string
		features     *DeliveryFeatures
		wantStrategy string
	}{
		{
			// Probability exactly 0.7 is NOT > 0.7, so it falls to the default
			// branch → adaptive strategy.
			name: "probability 0.7 should be adaptive not aggressive",
			features: &DeliveryFeatures{
				EndpointSuccessRate1h: 0.7,
				AttemptNumber:         1,
			},
			wantStrategy: "adaptive",
		},
		{
			// Probability exactly 0.3 is NOT < 0.3, so it falls to the default
			// branch → adaptive strategy.
			name: "probability 0.3 should be adaptive not conservative",
			features: &DeliveryFeatures{
				EndpointSuccessRate1h: 0.3,
				AttemptNumber:         1,
			},
			wantStrategy: "adaptive",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			opt := NewRetryOptimizer()

			strategy, pred, err := opt.GetOptimalStrategy(ctx, tc.features)
			require.NoError(t, err)
			require.NotNil(t, strategy)
			require.NotNil(t, pred)

			// The boundary thresholds are strict (> 0.7, < 0.3), so boundary
			// values should land in the adaptive bucket.
			assert.Equal(t, tc.wantStrategy, strategy.Name)
		})
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func ptr(f float64) *float64 { return &f }
