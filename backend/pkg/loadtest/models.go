package loadtest

import "time"

// TrafficPattern defines how load is generated over time.
type TrafficPattern string

const (
	PatternConstant TrafficPattern = "constant"
	PatternRampUp   TrafficPattern = "ramp-up"
	PatternBurst    TrafficPattern = "burst"
	PatternSineWave TrafficPattern = "sine-wave"
)

// TestConfig defines the configuration for a load test.
type TestConfig struct {
	ID              string            `json:"id"`
	TenantID        string            `json:"tenant_id"`
	EndpointURL     string            `json:"endpoint_url" binding:"required"`
	RPS             int               `json:"rps" binding:"required"`      // requests per second
	Duration        string            `json:"duration" binding:"required"` // e.g. "5m", "1h"
	Pattern         TrafficPattern    `json:"pattern"`
	PayloadTemplate string            `json:"payload_template,omitempty"`
	Headers         map[string]string `json:"headers,omitempty"`
	Concurrency     int               `json:"concurrency"`
	EventType       string            `json:"event_type"`
	// Ramp-up specific
	RampUpStart int `json:"ramp_up_start,omitempty"`
	// Burst specific
	BurstMultiplier float64 `json:"burst_multiplier,omitempty"`
	BurstDuration   string  `json:"burst_duration,omitempty"`
}

// TestRun represents an active or completed load test.
type TestRun struct {
	ID          string      `json:"id"`
	TenantID    string      `json:"tenant_id"`
	Config      *TestConfig `json:"config"`
	Status      string      `json:"status"` // pending, running, completed, cancelled
	StartedAt   *time.Time  `json:"started_at,omitempty"`
	CompletedAt *time.Time  `json:"completed_at,omitempty"`
	Report      *TestReport `json:"report,omitempty"`
}

// TestReport contains the results of a completed load test.
type TestReport struct {
	TotalRequests   int64         `json:"total_requests"`
	SuccessCount    int64         `json:"success_count"`
	FailureCount    int64         `json:"failure_count"`
	ErrorRate       float64       `json:"error_rate"`
	Throughput      float64       `json:"throughput_rps"`
	Duration        string        `json:"duration"`
	LatencyP50      float64       `json:"latency_p50_ms"`
	LatencyP95      float64       `json:"latency_p95_ms"`
	LatencyP99      float64       `json:"latency_p99_ms"`
	LatencyMin      float64       `json:"latency_min_ms"`
	LatencyMax      float64       `json:"latency_max_ms"`
	LatencyAvg      float64       `json:"latency_avg_ms"`
	StatusCodes     map[int]int64 `json:"status_codes"`
	Errors          []string      `json:"errors,omitempty"`
	TimeSeries      []TimePoint   `json:"time_series"`
	Recommendations []string      `json:"recommendations,omitempty"`
	GeneratedAt     time.Time     `json:"generated_at"`
}

// TimePoint is a single data point in the time series.
type TimePoint struct {
	Timestamp    time.Time `json:"timestamp"`
	RPS          float64   `json:"rps"`
	LatencyAvg   float64   `json:"latency_avg_ms"`
	ErrorRate    float64   `json:"error_rate"`
	SuccessCount int64     `json:"success_count"`
	FailureCount int64     `json:"failure_count"`
}

// Scenario is a pre-built load test scenario.
type Scenario struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Config      *TestConfig `json:"config"`
}

// DefaultScenarios returns pre-built load test scenarios.
func DefaultScenarios() []Scenario {
	return []Scenario{
		{
			ID:          "smoke",
			Name:        "Smoke Test",
			Description: "Low-volume test to verify endpoint is responsive",
			Config: &TestConfig{
				RPS: 10, Duration: "1m", Pattern: PatternConstant, Concurrency: 2,
			},
		},
		{
			ID:          "baseline",
			Name:        "Baseline Performance",
			Description: "Establish baseline performance metrics at moderate load",
			Config: &TestConfig{
				RPS: 100, Duration: "5m", Pattern: PatternConstant, Concurrency: 10,
			},
		},
		{
			ID:          "spike",
			Name:        "Spike Test",
			Description: "Test endpoint behavior under sudden traffic spikes",
			Config: &TestConfig{
				RPS: 500, Duration: "3m", Pattern: PatternBurst, Concurrency: 50,
				BurstMultiplier: 5.0, BurstDuration: "30s",
			},
		},
		{
			ID:          "ramp",
			Name:        "Ramp-Up Stress Test",
			Description: "Gradually increase load to find breaking point",
			Config: &TestConfig{
				RPS: 1000, Duration: "10m", Pattern: PatternRampUp, Concurrency: 100,
				RampUpStart: 10,
			},
		},
		{
			ID:          "soak",
			Name:        "Soak Test",
			Description: "Sustained load over extended period to detect memory leaks and degradation",
			Config: &TestConfig{
				RPS: 200, Duration: "30m", Pattern: PatternSineWave, Concurrency: 20,
			},
		},
	}
}
