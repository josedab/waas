package analytics

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =====================
// AnomalyDetector
// =====================

func TestAnomalyDetector_Detect_InsufficientData(t *testing.T) {
	d := NewAnomalyDetector(30, 2.5)
	series := make([]TimeSeriesPoint, 10) // Less than windowSize
	anomalies := d.Detect(series, "test_metric")
	assert.Nil(t, anomalies)
}

func TestAnomalyDetector_Detect_NoAnomalies(t *testing.T) {
	d := NewAnomalyDetector(5, 2.5)

	// Stable data with no outliers
	series := make([]TimeSeriesPoint, 10)
	for i := range series {
		series[i] = TimeSeriesPoint{
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			Value:     100.0,
		}
	}

	anomalies := d.Detect(series, "latency")
	assert.Empty(t, anomalies)
}

func TestAnomalyDetector_Detect_SpikeDetected(t *testing.T) {
	d := NewAnomalyDetector(5, 2.0)

	series := make([]TimeSeriesPoint, 10)
	for i := range series {
		series[i] = TimeSeriesPoint{
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			Value:     100.0 + float64(i%3), // slight variation for non-zero stddev
		}
	}
	// Inject a spike
	series[7].Value = 500.0

	anomalies := d.Detect(series, "latency")
	require.NotEmpty(t, anomalies)
	assert.Equal(t, "latency_spike", anomalies[0].Type)
	assert.Equal(t, "latency", anomalies[0].Metric)
}

func TestAnomalyDetector_Detect_TrafficDrop(t *testing.T) {
	d := NewAnomalyDetector(5, 2.0)

	series := make([]TimeSeriesPoint, 10)
	for i := range series {
		series[i] = TimeSeriesPoint{
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			Value:     100.0 + float64(i%3), // slight variation for non-zero stddev
		}
	}
	// Inject a significant drop
	series[7].Value = 0.0

	anomalies := d.Detect(series, "traffic")
	require.NotEmpty(t, anomalies)
	assert.Equal(t, "traffic_drop", anomalies[0].Type)
}

func TestAnomalyDetector_Detect_CriticalSeverity(t *testing.T) {
	d := NewAnomalyDetector(5, 2.0) // threshold = 2.0, critical at 3.0 (1.5x)

	series := make([]TimeSeriesPoint, 10)
	for i := range series {
		series[i] = TimeSeriesPoint{
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			Value:     100.0 + float64(i%3), // slight variation for non-zero stddev
		}
	}
	// Very large spike should be critical
	series[7].Value = 10000.0

	anomalies := d.Detect(series, "latency")
	require.NotEmpty(t, anomalies)
	assert.Equal(t, "critical", anomalies[0].Severity)
}

func TestAnomalyDetector_DefaultValues(t *testing.T) {
	d := NewAnomalyDetector(0, 0) // Should default
	assert.Equal(t, 30, d.windowSize)
	assert.Equal(t, 2.5, d.threshold)
}

// =====================
// computeStats
// =====================

func TestComputeStats_Empty(t *testing.T) {
	mean, stddev := computeStats(nil)
	assert.Equal(t, 0.0, mean)
	assert.Equal(t, 0.0, stddev)
}

func TestComputeStats_SingleValue(t *testing.T) {
	points := []TimeSeriesPoint{{Value: 42.0}}
	mean, stddev := computeStats(points)
	assert.Equal(t, 42.0, mean)
	assert.Equal(t, 0.0, stddev)
}

func TestComputeStats_MultipleValues(t *testing.T) {
	points := []TimeSeriesPoint{
		{Value: 10.0},
		{Value: 20.0},
		{Value: 30.0},
	}
	mean, stddev := computeStats(points)
	assert.InDelta(t, 20.0, mean, 0.001)
	assert.InDelta(t, 8.165, stddev, 0.01)
}

// =====================
// ComputeHealthScore
// =====================

func TestComputeHealthScore_PerfectHealth(t *testing.T) {
	score := ComputeHealthScore(1.0, 50.0, 100.0, 0)
	assert.InDelta(t, 100.0, score, 5.0) // Approximately 100
}

func TestComputeHealthScore_DegradedHealth(t *testing.T) {
	score := ComputeHealthScore(0.5, 2000.0, 10.0, 3)
	assert.True(t, score < 50.0, "Degraded score should be low: got %f", score)
}

func TestComputeHealthScore_Clamped(t *testing.T) {
	// Ensure score is clamped between 0-100
	score := ComputeHealthScore(0.0, 10000.0, 0.0, 100)
	assert.True(t, score >= 0, "Score should not be negative")
	assert.True(t, score <= 100, "Score should not exceed 100")
}

// =====================
// BuildLatencyBuckets
// =====================

func TestBuildLatencyBuckets_Empty(t *testing.T) {
	buckets := BuildLatencyBuckets(nil)
	assert.Len(t, buckets, 6)
	for _, b := range buckets {
		assert.Equal(t, int64(0), b.Count)
		assert.Equal(t, 0.0, b.Percent)
	}
}

func TestBuildLatencyBuckets_Distribution(t *testing.T) {
	latencies := []float64{10, 20, 30, 75, 150, 300, 600, 1500}
	buckets := BuildLatencyBuckets(latencies)

	assert.Len(t, buckets, 6)

	// Verify total count matches input
	totalCount := int64(0)
	for _, b := range buckets {
		totalCount += b.Count
	}
	assert.Equal(t, int64(len(latencies)), totalCount)

	// Verify percentages sum to ~100
	totalPct := 0.0
	for _, b := range buckets {
		totalPct += b.Percent
	}
	assert.InDelta(t, 100.0, totalPct, 0.1)
}

func TestBuildLatencyBuckets_AllInOneBucket(t *testing.T) {
	latencies := []float64{10, 20, 30, 40}
	buckets := BuildLatencyBuckets(latencies)

	// All < 50ms
	assert.Equal(t, int64(4), buckets[0].Count)
	assert.InDelta(t, 100.0, buckets[0].Percent, 0.1)
}

// =====================
// ComputeP99
// =====================

func TestComputeP99_Empty(t *testing.T) {
	assert.Equal(t, 0.0, ComputeP99(nil))
	assert.Equal(t, 0.0, ComputeP99([]float64{}))
}

func TestComputeP99_SingleValue(t *testing.T) {
	assert.Equal(t, 42.0, ComputeP99([]float64{42.0}))
}

func TestComputeP99_MultipleValues(t *testing.T) {
	values := make([]float64, 100)
	for i := range values {
		values[i] = float64(i + 1) // 1 to 100
	}
	p99 := ComputeP99(values)
	assert.InDelta(t, 100.0, p99, 1.0) // Should be near 99-100
}

func TestComputeP99_DoesNotModifyInput(t *testing.T) {
	values := []float64{50, 10, 90, 30, 70}
	original := make([]float64, len(values))
	copy(original, values)

	ComputeP99(values)

	// Original should be unchanged
	assert.Equal(t, original, values)
}

// =====================
// DetermineAnomalyTrend
// =====================

func TestDetermineAnomalyTrend(t *testing.T) {
	tests := []struct {
		name   string
		counts []int64
		want   string
	}{
		{"empty", []int64{}, "stable"},
		{"single", []int64{5}, "stable"},
		{"stable", []int64{5, 5, 5, 5}, "stable"},
		{"increasing", []int64{1, 1, 10, 10}, "increasing"},
		{"decreasing", []int64{10, 10, 1, 1}, "decreasing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, DetermineAnomalyTrend(tt.counts))
		})
	}
}

// =====================
// UnifiedDashboard struct
// =====================

func TestUnifiedDashboard_ZeroValue(t *testing.T) {
	var dashboard UnifiedDashboard

	assert.Empty(t, dashboard.TenantID)
	assert.True(t, dashboard.GeneratedAt.IsZero())
	assert.Equal(t, int64(0), dashboard.Overview.TotalDeliveries)
	assert.Equal(t, 0.0, dashboard.Overview.SuccessRate)
	assert.Equal(t, int64(0), dashboard.Traces.TotalTraces)
	assert.Equal(t, int64(0), dashboard.Anomalies.TotalAnomalies)
	assert.Equal(t, 0.0, dashboard.SLA.OverallCompliance)
	assert.Equal(t, 0, dashboard.Alerts.ActiveAlerts)
	assert.Nil(t, dashboard.TopEndpoints)
	assert.Nil(t, dashboard.ErrorTrends)
}

func TestUnifiedDashboard_FullyPopulated(t *testing.T) {
	dashboard := UnifiedDashboard{
		TenantID:    "tenant-1",
		GeneratedAt: time.Now(),
		TimeWindow:  "1h",
		Overview: DashboardOverview{
			TotalDeliveries:  1000,
			SuccessRate:      0.99,
			AvgLatencyMs:     150.0,
			P99LatencyMs:     500.0,
			ActiveEndpoints:  10,
			ErrorCount:       10,
			ThroughputPerSec: 16.7,
			HealthScore:      95.0,
		},
		Traces: TracesSummary{
			TotalTraces:   500,
			AvgSpanCount:  3.5,
			AvgDurationMs: 200.0,
			ErrorTraces:   5,
			SlowTraces:    10,
		},
		Anomalies: AnomalySummary{
			TotalAnomalies: 2,
			CriticalCount:  1,
			WarningCount:   1,
			AnomalyTrend:   "stable",
		},
		SLA: SLAStatus{
			OverallCompliance: 99.5,
			TargetsMet:        5,
			TargetsBreached:   0,
		},
		Alerts: AlertsSummary{
			ActiveAlerts:   3,
			CriticalAlerts: 1,
		},
		TopEndpoints: []EndpointHealth{
			{EndpointID: "ep-1", SuccessRate: 0.99, Status: "healthy"},
		},
	}

	assert.Equal(t, "tenant-1", dashboard.TenantID)
	assert.Equal(t, int64(1000), dashboard.Overview.TotalDeliveries)
	assert.False(t, dashboard.GeneratedAt.IsZero())
	assert.False(t, math.IsNaN(dashboard.Overview.HealthScore))
}
