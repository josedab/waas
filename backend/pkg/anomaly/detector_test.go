package anomaly

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func baselineWithSamples(mean, stddev float64, samples int64) *Baseline {
	return &Baseline{
		Mean:       mean,
		StdDev:     stddev,
		SampleSize: samples,
	}
}

// --- Detector.Detect tests ---

func TestDetect_ValueWithinBaseline(t *testing.T) {
	d := NewDetector(nil)
	baseline := baselineWithSamples(100, 10, 50)
	result := d.Detect(110, baseline, nil) // 1σ
	assert.False(t, result.IsAnomaly)
}

func TestDetect_CriticalAnomaly(t *testing.T) {
	d := NewDetector(nil)
	baseline := baselineWithSamples(100, 10, 50)
	result := d.Detect(135, baseline, nil) // 3.5σ → critical
	assert.True(t, result.IsAnomaly)
	assert.Equal(t, SeverityCritical, result.Severity)
	assert.GreaterOrEqual(t, result.Confidence, 0.5)
}

func TestDetect_WarningAnomaly(t *testing.T) {
	d := NewDetector(nil)
	baseline := baselineWithSamples(100, 10, 50)
	result := d.Detect(125, baseline, nil) // 2.5σ → warning
	assert.True(t, result.IsAnomaly)
	assert.Equal(t, SeverityWarning, result.Severity)
}

func TestDetect_StdDevZero_ValueNotEqual(t *testing.T) {
	d := NewDetector(nil)
	baseline := baselineWithSamples(100, 0, 50)
	result := d.Detect(105, baseline, nil)
	assert.True(t, result.IsAnomaly)
	assert.Equal(t, SeverityWarning, result.Severity)
	assert.Equal(t, 5.0, result.Score)
	assert.Equal(t, 0.5, result.Confidence)
}

func TestDetect_StdDevZero_ValueEqual(t *testing.T) {
	d := NewDetector(nil)
	baseline := baselineWithSamples(100, 0, 50)
	result := d.Detect(100, baseline, nil)
	assert.False(t, result.IsAnomaly)
}

func TestDetect_InsufficientSamples(t *testing.T) {
	d := NewDetector(nil)
	baseline := baselineWithSamples(100, 10, 5) // < 30
	result := d.Detect(200, baseline, nil)
	assert.False(t, result.IsAnomaly)
	assert.Contains(t, result.Description, "Insufficient data")
}

func TestDetect_NilDetectionConfig_UsesDefaults(t *testing.T) {
	d := NewDetector(nil)
	baseline := baselineWithSamples(100, 10, 50)

	// Just above 3σ → critical with default config
	result := d.Detect(131, baseline, nil)
	assert.True(t, result.IsAnomaly)
	assert.Equal(t, SeverityCritical, result.Severity)

	// Just below 2σ → no anomaly
	result2 := d.Detect(118, baseline, nil)
	assert.False(t, result2.IsAnomaly)
}

func TestDetect_CustomSensitivity(t *testing.T) {
	d := NewDetector(nil)
	baseline := baselineWithSamples(100, 10, 50)
	cfg := &DetectionConfig{Sensitivity: 2.0}

	// 1.5σ raw, but 2.0 sensitivity doubles → adjustedZ = 3.0 → critical
	result := d.Detect(115, baseline, cfg)
	assert.True(t, result.IsAnomaly)
	assert.Equal(t, SeverityCritical, result.Severity)
}

func TestDetect_CustomThresholds(t *testing.T) {
	d := NewDetector(nil)
	baseline := baselineWithSamples(100, 10, 50)
	cfg := &DetectionConfig{
		CriticalThreshold: 4.0,
		WarningThreshold:  1.0,
	}

	// 2.5σ: above custom warning(1.0) but below custom critical(4.0) → warning
	result := d.Detect(125, baseline, cfg)
	assert.True(t, result.IsAnomaly)
	assert.Equal(t, SeverityWarning, result.Severity)
}

func TestDetect_NegativeZScore(t *testing.T) {
	d := NewDetector(nil)
	baseline := baselineWithSamples(100, 10, 50)
	result := d.Detect(65, baseline, nil) // z = -3.5 → |z| = 3.5 → critical
	assert.True(t, result.IsAnomaly)
	assert.Equal(t, SeverityCritical, result.Severity)
}

// --- Detector.DetectTrend tests ---

func makePoints(values []float64) []MetricDataPoint {
	points := make([]MetricDataPoint, len(values))
	base := time.Now()
	for i, v := range values {
		points[i] = MetricDataPoint{
			Timestamp: base.Add(time.Duration(i) * time.Hour),
			Value:     v,
		}
	}
	return points
}

func TestDetectTrend_Increasing(t *testing.T) {
	d := NewDetector(nil)
	points := makePoints([]float64{10, 20, 30, 40, 50})
	baseline := baselineWithSamples(30, 1, 50) // small stddev so slope > threshold
	analysis := d.DetectTrend(points, baseline)
	assert.Equal(t, "increasing", analysis.Trend)
	assert.Len(t, analysis.Forecast, 6)
}

func TestDetectTrend_Stable(t *testing.T) {
	d := NewDetector(nil)
	points := makePoints([]float64{100, 100, 100, 100})
	baseline := baselineWithSamples(100, 10, 50)
	analysis := d.DetectTrend(points, baseline)
	assert.Equal(t, "stable", analysis.Trend)
}

func TestDetectTrend_Decreasing(t *testing.T) {
	d := NewDetector(nil)
	points := makePoints([]float64{50, 40, 30, 20, 10})
	baseline := baselineWithSamples(30, 1, 50)
	analysis := d.DetectTrend(points, baseline)
	assert.Equal(t, "decreasing", analysis.Trend)
}

func TestDetectTrend_InsufficientData(t *testing.T) {
	d := NewDetector(nil)
	points := makePoints([]float64{10, 20})
	baseline := baselineWithSamples(15, 5, 50)
	analysis := d.DetectTrend(points, baseline)
	assert.Equal(t, "insufficient_data", analysis.Trend)
}

func TestDetectTrend_ForecastLength(t *testing.T) {
	d := NewDetector(nil)
	points := makePoints([]float64{10, 20, 30, 40})
	baseline := baselineWithSamples(25, 1, 50)
	analysis := d.DetectTrend(points, baseline)
	assert.Len(t, analysis.Forecast, 6)
}

// --- CalculateBaseline tests ---

func TestCalculateBaseline_Correctness(t *testing.T) {
	points := makePoints([]float64{10, 20, 30, 40, 50})
	baseline := CalculateBaseline(points, nil)

	assert.Equal(t, 30.0, baseline.Mean)
	assert.Equal(t, 10.0, baseline.Min)
	assert.Equal(t, 50.0, baseline.Max)
	assert.Equal(t, int64(5), baseline.SampleSize)

	// Population stddev of {10,20,30,40,50} = sqrt(200) ≈ 14.14
	expectedStdDev := math.Sqrt(200.0)
	assert.InDelta(t, expectedStdDev, baseline.StdDev, 0.01)

	// Percentiles
	assert.Equal(t, 30.0, baseline.P50)
	assert.Equal(t, 40.0, baseline.P95)
	assert.Equal(t, 40.0, baseline.P99)
}

func TestCalculateBaseline_EmptyPoints(t *testing.T) {
	existing := &Baseline{Mean: 50, StdDev: 5, SampleSize: 100}
	result := CalculateBaseline(nil, existing)
	assert.Equal(t, existing, result)
}

func TestCalculateBaseline_EmptyPointsNilExisting(t *testing.T) {
	result := CalculateBaseline(nil, nil)
	assert.Nil(t, result)
}

func TestCalculateBaseline_WithExistingBaseline_EMA(t *testing.T) {
	existing := &Baseline{Mean: 100, StdDev: 10, SampleSize: 100}
	points := makePoints([]float64{50, 50, 50, 50, 50})
	baseline := CalculateBaseline(points, existing)

	// α=0.3, new mean=50, merged = 0.3*50 + 0.7*100 = 85
	assert.InDelta(t, 85.0, baseline.Mean, 0.01)
	// merged samples = 5 + 100 = 105
	assert.Equal(t, int64(105), baseline.SampleSize)
}

func TestCalculateBaseline_WithoutExisting(t *testing.T) {
	points := makePoints([]float64{20, 40, 60})
	baseline := CalculateBaseline(points, nil)
	require.NotNil(t, baseline)
	assert.InDelta(t, 40.0, baseline.Mean, 0.01)
	assert.Equal(t, int64(3), baseline.SampleSize)
}

func TestCalculateBaseline_SinglePoint(t *testing.T) {
	points := makePoints([]float64{42})
	baseline := CalculateBaseline(points, nil)
	assert.Equal(t, 42.0, baseline.Mean)
	assert.Equal(t, 0.0, baseline.StdDev)
	assert.Equal(t, int64(1), baseline.SampleSize)
}

// --- DetectMultiple tests ---

func TestDetectMultiple_BatchDetection(t *testing.T) {
	d := NewDetector(nil)
	baseline := baselineWithSamples(100, 10, 50)
	points := makePoints([]float64{100, 135, 125, 110})

	results := d.DetectMultiple(points, baseline, nil)
	require.Len(t, results, 4)

	assert.False(t, results[0].IsAnomaly) // 100 → 0σ
	assert.True(t, results[1].IsAnomaly)  // 135 → 3.5σ critical
	assert.Equal(t, SeverityCritical, results[1].Severity)
	assert.True(t, results[2].IsAnomaly) // 125 → 2.5σ warning
	assert.Equal(t, SeverityWarning, results[2].Severity)
	assert.False(t, results[3].IsAnomaly) // 110 → 1σ
}

func TestDetect_BaselineMeanZero_StdDevNonZero(t *testing.T) {
	d := NewDetector(nil)
	baseline := baselineWithSamples(0, 0.5, 50)
	result := d.Detect(2.0, baseline, nil) // z = (2-0)/0.5 = 4 → critical
	assert.True(t, result.IsAnomaly)
	assert.Equal(t, SeverityCritical, result.Severity)
}

func TestDetect_NegativeMetricValue(t *testing.T) {
	d := NewDetector(nil)
	baseline := baselineWithSamples(50, 10, 50)
	result := d.Detect(-100, baseline, nil) // z = (-100-50)/10 = -15 → critical
	assert.True(t, result.IsAnomaly)
	assert.Equal(t, SeverityCritical, result.Severity)
}

func TestDetect_StdDevZero_NegativeValue(t *testing.T) {
	d := NewDetector(nil)
	baseline := baselineWithSamples(0, 0, 50)
	result := d.Detect(-5.0, baseline, nil)
	assert.True(t, result.IsAnomaly)
	assert.Equal(t, 5.0, result.Score)
}

func TestDetect_NilBaseline_Panics(t *testing.T) {
	d := NewDetector(nil)

	// Calling Detect with nil baseline panics due to nil pointer dereference at baseline.SampleSize
	assert.Panics(t, func() {
		d.Detect(100.0, nil, nil)
	}, "Detect with nil baseline should panic (nil pointer dereference)")
}

func TestDetectTrend_EmptyPoints(t *testing.T) {
	d := NewDetector(nil)
	baseline := &Baseline{Mean: 100, StdDev: 10, SampleSize: 50}

	result := d.DetectTrend([]MetricDataPoint{}, baseline)

	assert.Equal(t, "insufficient_data", result.Trend)
}

func TestDetect_MinSamplesFromConfig(t *testing.T) {
	d := NewDetector(nil)
	baseline := &Baseline{Mean: 100, StdDev: 10, SampleSize: 15}
	// DetectionConfig with MinSamples=10, which baseline exceeds
	config := &DetectionConfig{MinSamples: 10}

	result := d.Detect(200.0, baseline, config)

	// Should detect anomaly since 15 >= 10 (custom MinSamples)
	assert.True(t, result.IsAnomaly)
}
