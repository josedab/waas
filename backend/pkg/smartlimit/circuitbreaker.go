package smartlimit

import (
	"math"
	"strconv"
	"sync"
	"time"
)

// CircuitState represents the state of the circuit breaker
type CircuitState string

const (
	CircuitClosed   CircuitState = "closed"   // Normal operation
	CircuitOpen     CircuitState = "open"     // Failures exceeded threshold, rejecting requests
	CircuitHalfOpen CircuitState = "half_open" // Testing if endpoint has recovered
)

// CircuitBreakerConfig configures the circuit breaker behavior
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of consecutive failures before opening the circuit
	FailureThreshold int `json:"failure_threshold"`
	// SuccessThreshold is the number of successes in half-open before closing
	SuccessThreshold int `json:"success_threshold"`
	// OpenDuration is how long the circuit stays open before testing (half-open)
	OpenDuration time.Duration `json:"open_duration"`
	// MaxOpenDuration caps the exponential backoff for open duration
	MaxOpenDuration time.Duration `json:"max_open_duration"`
	// HalfOpenMaxRequests is the max concurrent requests in half-open state
	HalfOpenMaxRequests int `json:"half_open_max_requests"`
}

// DefaultCircuitBreakerConfig returns sensible defaults
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		FailureThreshold:    5,
		SuccessThreshold:    3,
		OpenDuration:        30 * time.Second,
		MaxOpenDuration:     5 * time.Minute,
		HalfOpenMaxRequests: 1,
	}
}

// CircuitBreaker implements the circuit breaker pattern per endpoint
type CircuitBreaker struct {
	mu              sync.RWMutex
	config          *CircuitBreakerConfig
	state           CircuitState
	failures        int
	successes       int
	halfOpenCount   int
	lastStateChange time.Time
	openAttempts    int // tracks consecutive open→half-open→open cycles for backoff
	retryAfter      *time.Time
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	if config == nil {
		config = DefaultCircuitBreakerConfig()
	}
	return &CircuitBreaker{
		config:          config,
		state:           CircuitClosed,
		lastStateChange: time.Now(),
	}
}

// CircuitDecision represents the circuit breaker's decision
type CircuitDecision struct {
	Allowed     bool          `json:"allowed"`
	State       CircuitState  `json:"state"`
	RetryAfter  time.Duration `json:"retry_after,omitempty"`
	Reason      string        `json:"reason"`
	Failures    int           `json:"failures"`
	LastChanged time.Time     `json:"last_changed"`
}

// Allow checks if a request should proceed based on circuit state
func (cb *CircuitBreaker) Allow() *CircuitDecision {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	// Respect Retry-After from upstream
	if cb.retryAfter != nil && now.Before(*cb.retryAfter) {
		waitDuration := cb.retryAfter.Sub(now)
		return &CircuitDecision{
			Allowed:     false,
			State:       cb.state,
			RetryAfter:  waitDuration,
			Reason:      "respecting upstream Retry-After header",
			Failures:    cb.failures,
			LastChanged: cb.lastStateChange,
		}
	}

	switch cb.state {
	case CircuitClosed:
		return &CircuitDecision{
			Allowed:     true,
			State:       CircuitClosed,
			Reason:      "circuit closed (normal)",
			Failures:    cb.failures,
			LastChanged: cb.lastStateChange,
		}

	case CircuitOpen:
		// Check if open duration has elapsed
		openDuration := cb.currentOpenDuration()
		if now.Sub(cb.lastStateChange) >= openDuration {
			cb.transitionTo(CircuitHalfOpen)
			cb.halfOpenCount = 1
			return &CircuitDecision{
				Allowed:     true,
				State:       CircuitHalfOpen,
				Reason:      "circuit half-open (testing recovery)",
				Failures:    cb.failures,
				LastChanged: cb.lastStateChange,
			}
		}
		remaining := openDuration - now.Sub(cb.lastStateChange)
		return &CircuitDecision{
			Allowed:     false,
			State:       CircuitOpen,
			RetryAfter:  remaining,
			Reason:      "circuit open (endpoint failing)",
			Failures:    cb.failures,
			LastChanged: cb.lastStateChange,
		}

	case CircuitHalfOpen:
		if cb.halfOpenCount < cb.config.HalfOpenMaxRequests {
			cb.halfOpenCount++
			return &CircuitDecision{
				Allowed:     true,
				State:       CircuitHalfOpen,
				Reason:      "circuit half-open (probe request)",
				Failures:    cb.failures,
				LastChanged: cb.lastStateChange,
			}
		}
		return &CircuitDecision{
			Allowed:    false,
			State:      CircuitHalfOpen,
			RetryAfter: 1 * time.Second,
			Reason:     "circuit half-open (max concurrent probes reached)",
			Failures:   cb.failures,
			LastChanged: cb.lastStateChange,
		}
	}

	return &CircuitDecision{Allowed: true, State: cb.state}
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0

	switch cb.state {
	case CircuitHalfOpen:
		cb.successes++
		if cb.successes >= cb.config.SuccessThreshold {
			cb.transitionTo(CircuitClosed)
			cb.openAttempts = 0
			cb.successes = 0
		}
	case CircuitClosed:
		// Already good
	}
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.successes = 0

	switch cb.state {
	case CircuitClosed:
		if cb.failures >= cb.config.FailureThreshold {
			cb.transitionTo(CircuitOpen)
			cb.openAttempts++
		}
	case CircuitHalfOpen:
		// Any failure in half-open sends back to open
		cb.transitionTo(CircuitOpen)
		cb.openAttempts++
	}
}

// SetRetryAfter sets the upstream Retry-After time
func (cb *CircuitBreaker) SetRetryAfter(retryAfterHeader string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if retryAfterHeader == "" {
		return
	}

	// Try parsing as seconds
	if seconds, err := strconv.Atoi(retryAfterHeader); err == nil {
		t := time.Now().Add(time.Duration(seconds) * time.Second)
		cb.retryAfter = &t
		return
	}

	// Try parsing as HTTP date
	if t, err := time.Parse(time.RFC1123, retryAfterHeader); err == nil {
		cb.retryAfter = &t
	}
}

// State returns the current circuit state
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func (cb *CircuitBreaker) transitionTo(newState CircuitState) {
	cb.state = newState
	cb.lastStateChange = time.Now()
	cb.halfOpenCount = 0
}

// Exponential backoff for open duration
func (cb *CircuitBreaker) currentOpenDuration() time.Duration {
	base := cb.config.OpenDuration
	if cb.openAttempts <= 1 {
		return base
	}
	multiplier := math.Pow(2, float64(cb.openAttempts-1))
	d := time.Duration(float64(base) * multiplier)
	if d > cb.config.MaxOpenDuration {
		return cb.config.MaxOpenDuration
	}
	return d
}

// BackpressureManager coordinates circuit breakers and rate limiters
type BackpressureManager struct {
	mu       sync.RWMutex
	breakers map[string]*CircuitBreaker // key: tenantID:endpointID
	config   *CircuitBreakerConfig
}

// NewBackpressureManager creates a new backpressure manager
func NewBackpressureManager(config *CircuitBreakerConfig) *BackpressureManager {
	if config == nil {
		config = DefaultCircuitBreakerConfig()
	}
	return &BackpressureManager{
		breakers: make(map[string]*CircuitBreaker),
		config:   config,
	}
}

// GetCircuitBreaker returns or creates a circuit breaker for an endpoint
func (bm *BackpressureManager) GetCircuitBreaker(tenantID, endpointID string) *CircuitBreaker {
	key := tenantID + ":" + endpointID
	bm.mu.RLock()
	if cb, ok := bm.breakers[key]; ok {
		bm.mu.RUnlock()
		return cb
	}
	bm.mu.RUnlock()

	bm.mu.Lock()
	defer bm.mu.Unlock()

	// Double-check after write lock
	if cb, ok := bm.breakers[key]; ok {
		return cb
	}

	cb := NewCircuitBreaker(bm.config)
	bm.breakers[key] = cb
	return cb
}

// ShouldDeliver checks both circuit breaker and rate limiter
func (bm *BackpressureManager) ShouldDeliver(tenantID, endpointID string) (*CircuitDecision, error) {
	cb := bm.GetCircuitBreaker(tenantID, endpointID)
	return cb.Allow(), nil
}

// RecordResult records a delivery result for circuit breaking
func (bm *BackpressureManager) RecordResult(tenantID, endpointID string, statusCode int, retryAfterHeader string) {
	cb := bm.GetCircuitBreaker(tenantID, endpointID)

	if statusCode == 429 {
		cb.SetRetryAfter(retryAfterHeader)
		cb.RecordFailure()
	} else if statusCode >= 500 {
		cb.RecordFailure()
	} else if statusCode >= 200 && statusCode < 300 {
		cb.RecordSuccess()
	}
}

// Status returns the status of all tracked endpoints
func (bm *BackpressureManager) Status() map[string]CircuitState {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	status := make(map[string]CircuitState, len(bm.breakers))
	for key, cb := range bm.breakers {
		status[key] = cb.State()
	}
	return status
}
