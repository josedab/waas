package security

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnomalyDetector_KnownMaliciousPatterns(t *testing.T) {
	t.Parallel()
	detector := NewAnomalyDetector(AnomalyConfig{
		FailureRateThreshold:    0.25,
		VolumeStdDevMultiplier:  3.0,
		LatencyStdDevMultiplier: 3.0,
		PayloadSizeMultiplier:   5.0,
		MinSamplesForBaseline:   5,
	})
	ctx := context.Background()

	// Build baseline with normal metrics
	for i := 0; i < 10; i++ {
		detector.Analyze(ctx, &DeliveryMetrics{
			TenantID:     "t1",
			EndpointID:   "ep1",
			TotalCount:   100,
			FailureCount: 2,
			AvgLatencyMs: 50,
		})
	}

	// Inject high failure rate (simulating attack / malicious traffic)
	anomalies := detector.Analyze(ctx, &DeliveryMetrics{
		TenantID:     "t1",
		EndpointID:   "ep1",
		TotalCount:   100,
		FailureCount: 80,
		AvgLatencyMs: 50,
	})

	assert.NotEmpty(t, anomalies, "should detect anomaly for high failure rate")
	found := false
	for _, a := range anomalies {
		if a.Type == AnomalyHighFailureRate {
			found = true
			assert.Equal(t, "t1", a.TenantID)
			assert.Equal(t, "ep1", a.EndpointID)
			assert.Greater(t, a.Value, a.Threshold)
		}
	}
	assert.True(t, found, "should find high_failure_rate anomaly")
}

func TestAnomalyDetector_EncodedBypassAttempts(t *testing.T) {
	t.Parallel()
	detector := NewAnomalyDetector(AnomalyConfig{
		FailureRateThreshold:    0.1,
		VolumeStdDevMultiplier:  2.0,
		LatencyStdDevMultiplier: 2.0,
		PayloadSizeMultiplier:   3.0,
		MinSamplesForBaseline:   5,
	})
	ctx := context.Background()

	// Build baseline with very low volume
	for i := 0; i < 10; i++ {
		detector.Analyze(ctx, &DeliveryMetrics{
			TenantID:   "t2",
			EndpointID: "ep2",
			TotalCount: int64(10 + i), // slight variation to build non-zero std dev
			AvgLatencyMs: 20,
		})
	}

	// Sudden volume spike (potential bypass/abuse)
	anomalies := detector.Analyze(ctx, &DeliveryMetrics{
		TenantID:   "t2",
		EndpointID: "ep2",
		TotalCount: 10000,
		AvgLatencyMs: 20,
	})

	found := false
	for _, a := range anomalies {
		if a.Type == AnomalyUnusualVolume {
			found = true
		}
	}
	assert.True(t, found, "should detect unusual volume anomaly for bypass attempt")
}

func TestAnomalyDetector_FalsePositiveBenignTraffic(t *testing.T) {
	t.Parallel()
	detector := NewAnomalyDetector(DefaultAnomalyConfig())
	ctx := context.Background()

	// Build baseline
	for i := 0; i < 15; i++ {
		detector.Analyze(ctx, &DeliveryMetrics{
			TenantID:     "t3",
			EndpointID:   "ep3",
			TotalCount:   100,
			FailureCount: 2,
			AvgLatencyMs: 50,
		})
	}

	// Normal traffic — should NOT trigger anomalies
	anomalies := detector.Analyze(ctx, &DeliveryMetrics{
		TenantID:     "t3",
		EndpointID:   "ep3",
		TotalCount:   105,
		FailureCount: 3,
		AvgLatencyMs: 52,
	})

	assert.Empty(t, anomalies, "normal traffic should not trigger anomalies")
}

func TestAnomalyDetector_LatencySpike(t *testing.T) {
	t.Parallel()
	detector := NewAnomalyDetector(AnomalyConfig{
		FailureRateThreshold:    0.25,
		VolumeStdDevMultiplier:  3.0,
		LatencyStdDevMultiplier: 2.0,
		PayloadSizeMultiplier:   5.0,
		MinSamplesForBaseline:   5,
	})
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		detector.Analyze(ctx, &DeliveryMetrics{
			TenantID:     "t4",
			EndpointID:   "ep4",
			TotalCount:   100,
			FailureCount: 1,
			AvgLatencyMs: 50 + float64(i), // slight variation for non-zero std dev
		})
	}

	// Inject latency spike
	anomalies := detector.Analyze(ctx, &DeliveryMetrics{
		TenantID:     "t4",
		EndpointID:   "ep4",
		TotalCount:   100,
		FailureCount: 1,
		AvgLatencyMs: 5000,
	})

	found := false
	for _, a := range anomalies {
		if a.Type == AnomalyLatencySpike {
			found = true
		}
	}
	assert.True(t, found, "should detect latency spike anomaly")
}

func TestAnomalyDetector_GetAnomalies(t *testing.T) {
	t.Parallel()
	detector := NewAnomalyDetector(DefaultAnomalyConfig())
	ctx := context.Background()

	// No anomalies yet
	result := detector.GetAnomalies(ctx, "t1", 10)
	assert.Empty(t, result)

	// Default limit
	result = detector.GetAnomalies(ctx, "t1", 0)
	assert.Empty(t, result)
}

func TestAnomalyDetector_InsufficientBaseline(t *testing.T) {
	t.Parallel()
	detector := NewAnomalyDetector(DefaultAnomalyConfig())
	ctx := context.Background()

	// With fewer samples than MinSamplesForBaseline, no anomalies should be detected
	anomalies := detector.Analyze(ctx, &DeliveryMetrics{
		TenantID:     "t5",
		EndpointID:   "ep5",
		TotalCount:   100,
		FailureCount: 90,
		AvgLatencyMs: 5000,
	})

	assert.Nil(t, anomalies, "should not detect anomalies without sufficient baseline")
}

func TestClassifySeverity(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "critical", classifySeverity(1.0, 0.2))
	assert.Equal(t, "high", classifySeverity(0.6, 0.25))
	assert.Equal(t, "medium", classifySeverity(0.4, 0.25))
	assert.Equal(t, "low", classifySeverity(0.26, 0.25))
}
