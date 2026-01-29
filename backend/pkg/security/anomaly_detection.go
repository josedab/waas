package security

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

// AnomalyType represents the type of detected anomaly
type AnomalyType string

const (
	AnomalyHighFailureRate AnomalyType = "high_failure_rate"
	AnomalyUnusualVolume   AnomalyType = "unusual_volume"
	AnomalyLatencySpike    AnomalyType = "latency_spike"
	AnomalyNewIPAddress    AnomalyType = "new_ip_address"
	AnomalyPayloadSize     AnomalyType = "payload_size_anomaly"
)

// Anomaly represents a detected anomaly in delivery patterns
type Anomaly struct {
	ID          string      `json:"id"`
	TenantID    string      `json:"tenant_id"`
	EndpointID  string      `json:"endpoint_id,omitempty"`
	Type        AnomalyType `json:"type"`
	Severity    string      `json:"severity"` // low, medium, high, critical
	Description string      `json:"description"`
	Value       float64     `json:"value"`
	Threshold   float64     `json:"threshold"`
	DetectedAt  time.Time   `json:"detected_at"`
}

// DeliveryMetrics represents aggregated metrics for anomaly detection
type DeliveryMetrics struct {
	TenantID     string
	EndpointID   string
	Period       time.Duration
	TotalCount   int64
	FailureCount int64
	AvgLatencyMs float64
	P99LatencyMs float64
	AvgPayloadKB float64
}

// AnomalyDetector detects anomalies in webhook delivery patterns
type AnomalyDetector struct {
	mu        sync.RWMutex
	baselines map[string]*DeliveryBaseline
	anomalies []Anomaly
	config    AnomalyConfig
}

// AnomalyConfig configures anomaly detection thresholds
type AnomalyConfig struct {
	FailureRateThreshold    float64 `json:"failure_rate_threshold"`    // e.g., 0.25 = 25%
	VolumeStdDevMultiplier  float64 `json:"volume_std_dev_multiplier"` // e.g., 3.0
	LatencyStdDevMultiplier float64 `json:"latency_std_dev_multiplier"`
	PayloadSizeMultiplier   float64 `json:"payload_size_multiplier"`
	MinSamplesForBaseline   int     `json:"min_samples_for_baseline"`
}

// DeliveryBaseline represents the normal pattern for an endpoint
type DeliveryBaseline struct {
	EndpointID     string
	AvgVolume      float64
	StdDevVolume   float64
	AvgLatencyMs   float64
	StdDevLatency  float64
	AvgFailureRate float64
	AvgPayloadKB   float64
	SampleCount    int
	LastUpdated    time.Time
}

// DefaultAnomalyConfig returns sensible defaults
func DefaultAnomalyConfig() AnomalyConfig {
	return AnomalyConfig{
		FailureRateThreshold:    0.25,
		VolumeStdDevMultiplier:  3.0,
		LatencyStdDevMultiplier: 3.0,
		PayloadSizeMultiplier:   5.0,
		MinSamplesForBaseline:   10,
	}
}

// NewAnomalyDetector creates a new anomaly detector
func NewAnomalyDetector(config AnomalyConfig) *AnomalyDetector {
	return &AnomalyDetector{
		baselines: make(map[string]*DeliveryBaseline),
		anomalies: make([]Anomaly, 0),
		config:    config,
	}
}

// Analyze checks delivery metrics against baselines and returns anomalies
func (d *AnomalyDetector) Analyze(ctx context.Context, metrics *DeliveryMetrics) []Anomaly {
	d.mu.Lock()
	defer d.mu.Unlock()

	var detected []Anomaly
	key := metrics.TenantID + ":" + metrics.EndpointID
	baseline, exists := d.baselines[key]

	if !exists || baseline.SampleCount < d.config.MinSamplesForBaseline {
		// Build baseline
		d.updateBaseline(key, metrics)
		return nil
	}

	now := time.Now()

	// Check failure rate
	if metrics.TotalCount > 0 {
		failureRate := float64(metrics.FailureCount) / float64(metrics.TotalCount)
		if failureRate > d.config.FailureRateThreshold {
			detected = append(detected, Anomaly{
				ID:          fmt.Sprintf("anom_%d", now.UnixNano()),
				TenantID:    metrics.TenantID,
				EndpointID:  metrics.EndpointID,
				Type:        AnomalyHighFailureRate,
				Severity:    classifySeverity(failureRate, d.config.FailureRateThreshold),
				Description: fmt.Sprintf("Failure rate %.1f%% exceeds threshold %.1f%%", failureRate*100, d.config.FailureRateThreshold*100),
				Value:       failureRate,
				Threshold:   d.config.FailureRateThreshold,
				DetectedAt:  now,
			})
		}
	}

	// Check volume anomaly
	if baseline.StdDevVolume > 0 {
		volumeZScore := math.Abs(float64(metrics.TotalCount)-baseline.AvgVolume) / baseline.StdDevVolume
		if volumeZScore > d.config.VolumeStdDevMultiplier {
			detected = append(detected, Anomaly{
				ID:          fmt.Sprintf("anom_%d", now.UnixNano()+1),
				TenantID:    metrics.TenantID,
				EndpointID:  metrics.EndpointID,
				Type:        AnomalyUnusualVolume,
				Severity:    "medium",
				Description: fmt.Sprintf("Volume %d is %.1f std devs from mean %.0f", metrics.TotalCount, volumeZScore, baseline.AvgVolume),
				Value:       float64(metrics.TotalCount),
				Threshold:   baseline.AvgVolume + baseline.StdDevVolume*d.config.VolumeStdDevMultiplier,
				DetectedAt:  now,
			})
		}
	}

	// Check latency spike
	if baseline.StdDevLatency > 0 {
		latencyZScore := (metrics.AvgLatencyMs - baseline.AvgLatencyMs) / baseline.StdDevLatency
		if latencyZScore > d.config.LatencyStdDevMultiplier {
			detected = append(detected, Anomaly{
				ID:          fmt.Sprintf("anom_%d", now.UnixNano()+2),
				TenantID:    metrics.TenantID,
				EndpointID:  metrics.EndpointID,
				Type:        AnomalyLatencySpike,
				Severity:    "high",
				Description: fmt.Sprintf("Avg latency %.0fms is %.1f std devs above mean %.0fms", metrics.AvgLatencyMs, latencyZScore, baseline.AvgLatencyMs),
				Value:       metrics.AvgLatencyMs,
				Threshold:   baseline.AvgLatencyMs + baseline.StdDevLatency*d.config.LatencyStdDevMultiplier,
				DetectedAt:  now,
			})
		}
	}

	// Update baseline with new data
	d.updateBaseline(key, metrics)

	// Store detected anomalies
	d.anomalies = append(d.anomalies, detected...)
	// Keep only last 1000
	if len(d.anomalies) > 1000 {
		d.anomalies = d.anomalies[len(d.anomalies)-1000:]
	}

	return detected
}

// GetAnomalies returns recent anomalies for a tenant
func (d *AnomalyDetector) GetAnomalies(ctx context.Context, tenantID string, limit int) []Anomaly {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}

	var result []Anomaly
	for i := len(d.anomalies) - 1; i >= 0 && len(result) < limit; i-- {
		if d.anomalies[i].TenantID == tenantID {
			result = append(result, d.anomalies[i])
		}
	}
	return result
}

func (d *AnomalyDetector) updateBaseline(key string, metrics *DeliveryMetrics) {
	baseline, exists := d.baselines[key]
	if !exists {
		baseline = &DeliveryBaseline{
			EndpointID: metrics.EndpointID,
		}
		d.baselines[key] = baseline
	}

	n := float64(baseline.SampleCount)
	baseline.SampleCount++

	// Welford's online algorithm for running mean and variance
	if baseline.SampleCount == 1 {
		baseline.AvgVolume = float64(metrics.TotalCount)
		baseline.AvgLatencyMs = metrics.AvgLatencyMs
		baseline.AvgPayloadKB = metrics.AvgPayloadKB
		if metrics.TotalCount > 0 {
			baseline.AvgFailureRate = float64(metrics.FailureCount) / float64(metrics.TotalCount)
		}
	} else {
		newVolume := float64(metrics.TotalCount)
		oldMean := baseline.AvgVolume
		baseline.AvgVolume = oldMean + (newVolume-oldMean)/float64(baseline.SampleCount)
		baseline.StdDevVolume = math.Sqrt(((n-1)*baseline.StdDevVolume*baseline.StdDevVolume + (newVolume-oldMean)*(newVolume-baseline.AvgVolume)) / n)

		oldLatency := baseline.AvgLatencyMs
		baseline.AvgLatencyMs = oldLatency + (metrics.AvgLatencyMs-oldLatency)/float64(baseline.SampleCount)
		baseline.StdDevLatency = math.Sqrt(((n-1)*baseline.StdDevLatency*baseline.StdDevLatency + (metrics.AvgLatencyMs-oldLatency)*(metrics.AvgLatencyMs-baseline.AvgLatencyMs)) / n)
	}

	baseline.LastUpdated = time.Now()
}

func classifySeverity(value, threshold float64) string {
	ratio := value / threshold
	switch {
	case ratio > 4:
		return "critical"
	case ratio > 2:
		return "high"
	case ratio > 1.5:
		return "medium"
	default:
		return "low"
	}
}
