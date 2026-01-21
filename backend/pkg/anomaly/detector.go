package anomaly

import (
	"math"
	"sort"
)

// Detector performs anomaly detection on metric data
type Detector struct {
	config *DetectorConfig
}

// DetectorConfig holds detector configuration
type DetectorConfig struct {
	DefaultCriticalThreshold float64 // Default: 3.0 (3 std deviations)
	DefaultWarningThreshold  float64 // Default: 2.0 (2 std deviations)
	MinSamplesForDetection   int     // Default: 30
	DefaultSensitivity       float64 // Default: 1.0
}

// DefaultDetectorConfig returns default configuration
func DefaultDetectorConfig() *DetectorConfig {
	return &DetectorConfig{
		DefaultCriticalThreshold: 3.0,
		DefaultWarningThreshold:  2.0,
		MinSamplesForDetection:   30,
		DefaultSensitivity:       1.0,
	}
}

// NewDetector creates a new anomaly detector
func NewDetector(config *DetectorConfig) *Detector {
	if config == nil {
		config = DefaultDetectorConfig()
	}
	return &Detector{config: config}
}

// Detect checks if a value is anomalous given a baseline
func (d *Detector) Detect(value float64, baseline *Baseline, detectionConfig *DetectionConfig) *DetectionResult {
	result := &DetectionResult{
		IsAnomaly:  false,
		Confidence: 0,
	}

	// Check minimum samples
	minSamples := d.config.MinSamplesForDetection
	if detectionConfig != nil && detectionConfig.MinSamples > 0 {
		minSamples = detectionConfig.MinSamples
	}

	if baseline.SampleSize < int64(minSamples) {
		result.Description = "Insufficient data for detection"
		return result
	}

	// Calculate Z-score
	if baseline.StdDev == 0 {
		// All values were the same
		if value != baseline.Mean {
			result.IsAnomaly = true
			result.Severity = SeverityWarning
			result.Score = math.Abs(value - baseline.Mean)
			result.Confidence = 0.5
			result.Description = "Value deviates from constant baseline"
		}
		return result
	}

	zScore := (value - baseline.Mean) / baseline.StdDev
	absZScore := math.Abs(zScore)

	// Apply sensitivity
	sensitivity := d.config.DefaultSensitivity
	if detectionConfig != nil && detectionConfig.Sensitivity > 0 {
		sensitivity = detectionConfig.Sensitivity
	}
	adjustedZScore := absZScore * sensitivity

	result.Score = adjustedZScore

	// Determine thresholds
	criticalThreshold := d.config.DefaultCriticalThreshold
	warningThreshold := d.config.DefaultWarningThreshold
	if detectionConfig != nil {
		if detectionConfig.CriticalThreshold > 0 {
			criticalThreshold = detectionConfig.CriticalThreshold
		}
		if detectionConfig.WarningThreshold > 0 {
			warningThreshold = detectionConfig.WarningThreshold
		}
	}

	// Check thresholds
	if adjustedZScore >= criticalThreshold {
		result.IsAnomaly = true
		result.Severity = SeverityCritical
		result.Confidence = d.calculateConfidence(adjustedZScore, criticalThreshold)
		result.Description = d.generateDescription(value, baseline, zScore, "critical")
	} else if adjustedZScore >= warningThreshold {
		result.IsAnomaly = true
		result.Severity = SeverityWarning
		result.Confidence = d.calculateConfidence(adjustedZScore, warningThreshold)
		result.Description = d.generateDescription(value, baseline, zScore, "warning")
	}

	return result
}

// DetectMultiple detects anomalies across multiple data points
func (d *Detector) DetectMultiple(points []MetricDataPoint, baseline *Baseline, detectionConfig *DetectionConfig) []*DetectionResult {
	results := make([]*DetectionResult, len(points))
	for i, point := range points {
		results[i] = d.Detect(point.Value, baseline, detectionConfig)
	}
	return results
}

// DetectTrend detects if there's an anomalous trend
func (d *Detector) DetectTrend(points []MetricDataPoint, baseline *Baseline) *TrendAnalysis {
	if len(points) < 3 {
		return &TrendAnalysis{Trend: "insufficient_data"}
	}

	// Sort by timestamp
	sorted := make([]MetricDataPoint, len(points))
	copy(sorted, points)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.Before(sorted[j].Timestamp)
	})

	// Calculate linear regression
	n := float64(len(sorted))
	var sumX, sumY, sumXY, sumX2 float64

	for i, p := range sorted {
		x := float64(i)
		y := p.Value
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	
	// Determine trend
	trend := "stable"
	threshold := baseline.StdDev * 0.1 // 10% of std dev per point
	if slope > threshold {
		trend = "increasing"
	} else if slope < -threshold {
		trend = "decreasing"
	}

	// Calculate change rate (assuming hourly data points)
	changeRate := (slope / baseline.Mean) * 100 // Percent per point

	analysis := &TrendAnalysis{
		Trend:      trend,
		ChangeRate: changeRate,
		Period:     "hourly",
	}

	// Simple forecast (linear extrapolation)
	lastValue := sorted[len(sorted)-1].Value
	for i := 1; i <= 6; i++ {
		analysis.Forecast = append(analysis.Forecast, lastValue+slope*float64(i))
	}

	return analysis
}

func (d *Detector) calculateConfidence(score, threshold float64) float64 {
	// Confidence increases as score exceeds threshold
	excess := score - threshold
	confidence := 0.5 + (excess / (threshold * 2)) * 0.5
	if confidence > 1.0 {
		confidence = 1.0
	}
	return confidence
}

func (d *Detector) generateDescription(value float64, baseline *Baseline, zScore float64, severity string) string {
	direction := "above"
	if zScore < 0 {
		direction = "below"
	}

	deviationPct := math.Abs((value - baseline.Mean) / baseline.Mean * 100)

	return formatDescription(severity, value, direction, baseline.Mean, deviationPct, math.Abs(zScore))
}

func formatDescription(severity string, value float64, direction string, mean float64, deviationPct float64, zScore float64) string {
	return severity + " anomaly: value " + formatFloat(value) + " is " + direction + 
		" expected " + formatFloat(mean) + " (" + formatFloat(deviationPct) + "% deviation, " + 
		formatFloat(zScore) + " std devs)"
}

func formatFloat(f float64) string {
	if f == float64(int64(f)) {
		return formatInt(int64(f))
	}
	// Simple float formatting without fmt
	intPart := int64(f)
	decPart := int64((f - float64(intPart)) * 100)
	if decPart < 0 {
		decPart = -decPart
	}
	result := formatInt(intPart) + "."
	if decPart < 10 {
		result += "0"
	}
	result += formatInt(decPart)
	return result
}

func formatInt(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	digits := make([]byte, 0, 20)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}
	// Reverse
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	if neg {
		return "-" + string(digits)
	}
	return string(digits)
}

// CalculateBaseline calculates baseline statistics from data points
func CalculateBaseline(points []MetricDataPoint, existing *Baseline) *Baseline {
	if len(points) == 0 {
		return existing
	}

	values := make([]float64, len(points))
	for i, p := range points {
		values[i] = p.Value
	}

	sort.Float64s(values)

	n := float64(len(values))
	
	// Calculate mean
	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / n

	// Calculate std dev
	var sumSquares float64
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	stdDev := math.Sqrt(sumSquares / n)

	baseline := &Baseline{
		Mean:       mean,
		StdDev:     stdDev,
		Min:        values[0],
		Max:        values[len(values)-1],
		P50:        percentile(values, 50),
		P95:        percentile(values, 95),
		P99:        percentile(values, 99),
		SampleSize: int64(len(points)),
	}

	// Merge with existing baseline using exponential moving average
	if existing != nil && existing.SampleSize > 0 {
		alpha := 0.3 // Weight for new data
		baseline.Mean = alpha*baseline.Mean + (1-alpha)*existing.Mean
		baseline.StdDev = alpha*baseline.StdDev + (1-alpha)*existing.StdDev
		baseline.SampleSize += existing.SampleSize
	}

	return baseline
}

func percentile(sorted []float64, p int) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * float64(p) / 100)
	return sorted[idx]
}
