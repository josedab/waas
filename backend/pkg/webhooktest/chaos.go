package webhooktest

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// ChaosInjectionConfig configures advanced chaos injection for webhook testing.
// This extends the base ChaosConfig with richer failure scenarios.
type ChaosInjectionConfig struct {
	Enabled          bool              `json:"enabled"`
	LatencyInjection *LatencyInjection `json:"latency_injection,omitempty"`
	FailureInjection *FailureInjection `json:"failure_injection,omitempty"`
	ThrottleConfig   *ThrottleConfig   `json:"throttle_config,omitempty"`
	PartitionConfig  *PartitionConfig  `json:"partition_config,omitempty"`
}

// LatencyInjection configures artificial latency
type LatencyInjection struct {
	Enabled      bool    `json:"enabled"`
	MinLatencyMs int     `json:"min_latency_ms"`
	MaxLatencyMs int     `json:"max_latency_ms"`
	Probability  float64 `json:"probability"` // 0-1
	Distribution string  `json:"distribution"` // uniform, normal, spike
}

// FailureInjection configures artificial failures
type FailureInjection struct {
	Enabled       bool                `json:"enabled"`
	ErrorRate     float64             `json:"error_rate"` // 0-1
	StatusCodes   []int               `json:"status_codes"` // codes to return
	ErrorBodies   []string            `json:"error_bodies,omitempty"`
	Pattern       string              `json:"pattern"` // random, burst, periodic
	BurstSize     int                 `json:"burst_size,omitempty"`
	BurstInterval int                 `json:"burst_interval_seconds,omitempty"`
	Scenarios     []FailureScenario   `json:"scenarios,omitempty"`
}

// FailureScenario defines a specific failure scenario
type FailureScenario struct {
	Name        string  `json:"name"`
	StatusCode  int     `json:"status_code"`
	Body        string  `json:"body,omitempty"`
	Probability float64 `json:"probability"` // 0-1
	Duration    int     `json:"duration_seconds,omitempty"`
}

// ThrottleConfig simulates rate limiting
type ThrottleConfig struct {
	Enabled         bool `json:"enabled"`
	RequestsPerSec  int  `json:"requests_per_sec"`
	BurstSize       int  `json:"burst_size"`
	RetryAfterSec   int  `json:"retry_after_seconds"`
}

// PartitionConfig simulates network partitions
type PartitionConfig struct {
	Enabled       bool `json:"enabled"`
	DropRate      float64 `json:"drop_rate"` // 0-1
	TimeoutMs     int  `json:"timeout_ms"`
	ResetAfterSec int  `json:"reset_after_seconds"`
}

// ChaosEngine applies chaos injection to webhook delivery testing
type ChaosEngine struct {
	config       *ChaosInjectionConfig
	mu           sync.RWMutex
	requestCount int64
	burstCounter int
	lastBurstAt  time.Time
	throttle     *tokenBucket
	metrics      *ChaosMetrics
}

// ChaosMetrics tracks chaos injection statistics
type ChaosMetrics struct {
	TotalRequests    int64 `json:"total_requests"`
	LatencyInjected  int64 `json:"latency_injected"`
	FailuresInjected int64 `json:"failures_injected"`
	Throttled        int64 `json:"throttled"`
	Dropped          int64 `json:"dropped"`
}

type tokenBucket struct {
	tokens     int
	maxTokens  int
	refillRate int
	lastRefill time.Time
	mu         sync.Mutex
}

func newTokenBucket(maxTokens, refillRate int) *tokenBucket {
	return &tokenBucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (tb *tokenBucket) take() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	refill := int(elapsed * float64(tb.refillRate))
	if refill > 0 {
		tb.tokens += refill
		if tb.tokens > tb.maxTokens {
			tb.tokens = tb.maxTokens
		}
		tb.lastRefill = now
	}

	if tb.tokens > 0 {
		tb.tokens--
		return true
	}
	return false
}

// NewChaosEngine creates a new chaos injection engine
func NewChaosEngine(config *ChaosInjectionConfig) *ChaosEngine {
	engine := &ChaosEngine{
		config:  config,
		metrics: &ChaosMetrics{},
	}

	if config.ThrottleConfig != nil && config.ThrottleConfig.Enabled {
		engine.throttle = newTokenBucket(
			config.ThrottleConfig.BurstSize,
			config.ThrottleConfig.RequestsPerSec,
		)
	}

	return engine
}

// Apply applies chaos injection and returns a modified response or error
func (e *ChaosEngine) Apply() *ChaosResult {
	e.mu.Lock()
	e.requestCount++
	count := e.requestCount
	e.mu.Unlock()

	e.metrics.TotalRequests = count

	result := &ChaosResult{
		ShouldProceed: true,
		StatusCode:    http.StatusOK,
	}

	if !e.config.Enabled {
		return result
	}

	// Network partition / drop
	if e.config.PartitionConfig != nil && e.config.PartitionConfig.Enabled {
		if rand.Float64() < e.config.PartitionConfig.DropRate {
			e.metrics.Dropped++
			result.ShouldProceed = false
			result.Delay = time.Duration(e.config.PartitionConfig.TimeoutMs) * time.Millisecond
			result.Reason = "network partition (simulated)"
			return result
		}
	}

	// Throttling
	if e.throttle != nil && e.config.ThrottleConfig.Enabled {
		if !e.throttle.take() {
			e.metrics.Throttled++
			result.StatusCode = http.StatusTooManyRequests
			result.Body = `{"error": "rate limit exceeded"}`
			result.Headers = map[string]string{
				"Retry-After": fmt.Sprintf("%d", e.config.ThrottleConfig.RetryAfterSec),
			}
			result.Reason = "throttled (simulated)"
			return result
		}
	}

	// Latency injection
	if e.config.LatencyInjection != nil && e.config.LatencyInjection.Enabled {
		if rand.Float64() < e.config.LatencyInjection.Probability {
			delay := e.computeLatency()
			result.Delay = delay
			e.metrics.LatencyInjected++
		}
	}

	// Failure injection
	if e.config.FailureInjection != nil && e.config.FailureInjection.Enabled {
		if e.shouldInjectFailure() {
			result.StatusCode = e.pickErrorCode()
			result.Body = e.pickErrorBody()
			result.Reason = "failure injected (simulated)"
			e.metrics.FailuresInjected++
		}
	}

	return result
}

// GetMetrics returns current chaos injection metrics
func (e *ChaosEngine) GetMetrics() ChaosMetrics {
	return *e.metrics
}

// UpdateConfig updates the chaos configuration dynamically
func (e *ChaosEngine) UpdateConfig(config *ChaosInjectionConfig) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.config = config

	if config.ThrottleConfig != nil && config.ThrottleConfig.Enabled {
		e.throttle = newTokenBucket(
			config.ThrottleConfig.BurstSize,
			config.ThrottleConfig.RequestsPerSec,
		)
	}
}

func (e *ChaosEngine) computeLatency() time.Duration {
	cfg := e.config.LatencyInjection
	minMs := cfg.MinLatencyMs
	maxMs := cfg.MaxLatencyMs
	if maxMs <= minMs {
		return time.Duration(minMs) * time.Millisecond
	}

	var delayMs int
	switch cfg.Distribution {
	case "spike":
		// Bimodal: either baseline or large spike
		if rand.Float64() < 0.3 {
			delayMs = maxMs
		} else {
			delayMs = minMs
		}
	case "normal":
		mean := float64(minMs+maxMs) / 2.0
		stddev := float64(maxMs-minMs) / 4.0
		delayMs = int(rand.NormFloat64()*stddev + mean)
		if delayMs < minMs {
			delayMs = minMs
		}
	default: // uniform
		delayMs = minMs + rand.Intn(maxMs-minMs)
	}

	return time.Duration(delayMs) * time.Millisecond
}

func (e *ChaosEngine) shouldInjectFailure() bool {
	cfg := e.config.FailureInjection

	switch cfg.Pattern {
	case "burst":
		now := time.Now()
		if now.Sub(e.lastBurstAt) > time.Duration(cfg.BurstInterval)*time.Second {
			e.burstCounter = 0
			e.lastBurstAt = now
		}
		if e.burstCounter < cfg.BurstSize {
			e.burstCounter++
			return true
		}
		return false
	case "periodic":
		return e.requestCount%10 == 0 // Every 10th request
	default: // random
		return rand.Float64() < cfg.ErrorRate
	}
}

func (e *ChaosEngine) pickErrorCode() int {
	codes := e.config.FailureInjection.StatusCodes
	if len(codes) == 0 {
		codes = []int{500, 502, 503, 504}
	}
	return codes[rand.Intn(len(codes))]
}

func (e *ChaosEngine) pickErrorBody() string {
	bodies := e.config.FailureInjection.ErrorBodies
	if len(bodies) == 0 {
		return `{"error": "internal server error (simulated)"}`
	}
	return bodies[rand.Intn(len(bodies))]
}

// ChaosResult represents the result of chaos injection
type ChaosResult struct {
	ShouldProceed bool              `json:"should_proceed"`
	StatusCode    int               `json:"status_code"`
	Body          string            `json:"body,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	Delay         time.Duration     `json:"delay,omitempty"`
	Reason        string            `json:"reason,omitempty"`
}

// ContractTestSuiteConfig configures webhook contract testing for CI/CD pipelines.
// This extends the base CITestConfig with chaos and parallelism support.
type ContractTestSuiteConfig struct {
	Name         string               `json:"name" yaml:"name"`
	Description  string               `json:"description,omitempty" yaml:"description"`
	BaseURL      string               `json:"base_url" yaml:"base_url"`
	APIKey       string               `json:"api_key,omitempty" yaml:"api_key"`
	Timeout      int                  `json:"timeout_seconds" yaml:"timeout_seconds"`
	TestCases    []ContractTestCase   `json:"test_cases" yaml:"test_cases"`
	FailFast     bool                 `json:"fail_fast" yaml:"fail_fast"`
	Parallelism  int                  `json:"parallelism" yaml:"parallelism"`
	Chaos        *ChaosInjectionConfig `json:"chaos,omitempty" yaml:"chaos"`
}

// ContractTestCase defines a single contract test case for CI
type ContractTestCase struct {
	Name              string            `json:"name" yaml:"name"`
	EventType         string            `json:"event_type" yaml:"event_type"`
	Payload           interface{}       `json:"payload" yaml:"payload"`
	PayloadFile       string            `json:"payload_file,omitempty" yaml:"payload_file"`
	Headers           map[string]string `json:"headers,omitempty" yaml:"headers"`
	ExpectedStatus    int               `json:"expected_status" yaml:"expected_status"`
	ExpectedLatencyMs int               `json:"max_latency_ms,omitempty" yaml:"max_latency_ms"`
	SchemaValidation  bool              `json:"schema_validation" yaml:"schema_validation"`
	Assertions        []ContractAssertion `json:"assertions,omitempty" yaml:"assertions"`
}

// ContractAssertion defines a test assertion for CI contract tests
type ContractAssertion struct {
	Type     string `json:"type" yaml:"type"`
	Field    string `json:"field,omitempty" yaml:"field"`
	Operator string `json:"operator" yaml:"operator"`
	Value    string `json:"value,omitempty" yaml:"value"`
}

// ContractTestResult is the result of running a contract test suite
type ContractTestResult struct {
	SuiteName    string               `json:"suite_name"`
	Passed       bool                 `json:"passed"`
	TotalTests   int                  `json:"total_tests"`
	PassedTests  int                  `json:"passed_tests"`
	FailedTests  int                  `json:"failed_tests"`
	SkippedTests int                  `json:"skipped_tests"`
	Duration     time.Duration        `json:"duration"`
	Results      []ContractCaseResult `json:"results"`
	ChaosMetrics *ChaosMetrics        `json:"chaos_metrics,omitempty"`
	GeneratedAt  time.Time            `json:"generated_at"`
}

// ContractCaseResult is the result of a single contract test case
type ContractCaseResult struct {
	Name        string                    `json:"name"`
	Passed      bool                      `json:"passed"`
	Duration    time.Duration             `json:"duration"`
	StatusCode  int                       `json:"status_code,omitempty"`
	LatencyMs   int64                     `json:"latency_ms,omitempty"`
	Error       string                    `json:"error,omitempty"`
	Assertions  []ContractAssertionResult `json:"assertions,omitempty"`
}

// ContractAssertionResult tracks the result of a single assertion
type ContractAssertionResult struct {
	Type    string `json:"type"`
	Passed  bool   `json:"passed"`
	Message string `json:"message,omitempty"`
}
