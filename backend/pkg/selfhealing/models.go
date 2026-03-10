package selfhealing

import "time"

// EndpointDiscovery represents a discovered or updated endpoint URL.
type EndpointDiscovery struct {
	ID            string      `json:"id"`
	TenantID      string      `json:"tenant_id"`
	EndpointID    string      `json:"endpoint_id"`
	OriginalURL   string      `json:"original_url"`
	DiscoveredURL string      `json:"discovered_url"`
	Method        string      `json:"method"` // well-known, dns-txt, http-redirect, manual
	Status        string      `json:"status"` // pending, validated, applied, rejected
	TestResult    *TestResult `json:"test_result,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
	AppliedAt     *time.Time  `json:"applied_at,omitempty"`
}

// TestResult holds the outcome of a test payload delivery to a new URL.
type TestResult struct {
	StatusCode int       `json:"status_code"`
	LatencyMs  int64     `json:"latency_ms"`
	Success    bool      `json:"success"`
	Error      string    `json:"error,omitempty"`
	TestedAt   time.Time `json:"tested_at"`
}

// WellKnownSpec represents the .well-known/waas-webhooks document.
type WellKnownSpec struct {
	Version   string              `json:"version"`
	Endpoints []WellKnownEndpoint `json:"endpoints"`
	UpdatedAt string              `json:"updated_at"`
}

// WellKnownEndpoint is a single entry in the well-known spec.
type WellKnownEndpoint struct {
	Name       string   `json:"name"`
	URL        string   `json:"url"`
	EventTypes []string `json:"event_types"`
	Status     string   `json:"status"` // active, deprecated, migrated
	MigratedTo string   `json:"migrated_to,omitempty"`
}

// FailureTracker records consecutive failures for an endpoint.
type FailureTracker struct {
	EndpointID          string    `json:"endpoint_id"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	LastFailureAt       time.Time `json:"last_failure_at"`
	HealingTriggered    bool      `json:"healing_triggered"`
}

// MigrationEvent is a meta-event emitted when a URL changes.
type MigrationEvent struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	EndpointID string    `json:"endpoint_id"`
	EventType  string    `json:"event_type"` // endpoint.url.changed, endpoint.url.validated
	OldURL     string    `json:"old_url"`
	NewURL     string    `json:"new_url"`
	Method     string    `json:"method"`
	Timestamp  time.Time `json:"timestamp"`
}

// ServiceConfig configures the self-healing service.
type ServiceConfig struct {
	FailureThreshold     int           // consecutive failures before healing
	TestPayload          string        // payload sent to validate new URL
	ValidationTimeout    time.Duration // timeout for test delivery
	MaxDiscoveriesPerDay int
}

// DefaultServiceConfig returns sensible defaults.
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		FailureThreshold:     5,
		TestPayload:          `{"type":"waas.health_check","data":{"test":true}}`,
		ValidationTimeout:    10 * time.Second,
		MaxDiscoveriesPerDay: 100,
	}
}

// EndpointHealthStatus represents the detected health status of an endpoint.
type EndpointHealthStatus struct {
	EndpointID          string    `json:"endpoint_id"`
	TenantID            string    `json:"tenant_id"`
	Status              string    `json:"status"` // healthy, degraded, unhealthy, dead
	SuccessRate         float64   `json:"success_rate"`
	AvgLatencyMs        float64   `json:"avg_latency_ms"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	LastCheckedAt       time.Time `json:"last_checked_at"`
	Mirrors             []string  `json:"mirrors,omitempty"`
}

// RemediationAction represents an autonomous remediation taken by the system.
type RemediationAction struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	EndpointID  string     `json:"endpoint_id"`
	ActionType  string     `json:"action_type"` // reroute, adjust_concurrency, adjust_retry, circuit_break, mirror_activate
	Description string     `json:"description"`
	OldValue    string     `json:"old_value"`
	NewValue    string     `json:"new_value"`
	Confidence  float64    `json:"confidence"`
	Automated   bool       `json:"automated"`
	Status      string     `json:"status"` // applied, reverted, pending
	CreatedAt   time.Time  `json:"created_at"`
	RevertedAt  *time.Time `json:"reverted_at,omitempty"`
}

// RetryTuningResult captures the learning loop's retry policy recommendation.
type RetryTuningResult struct {
	EndpointID            string  `json:"endpoint_id"`
	CurrentMaxRetries     int     `json:"current_max_retries"`
	RecommendedRetries    int     `json:"recommended_retries"`
	CurrentBackoffMs      float64 `json:"current_backoff_ms"`
	RecommendedBackoffMs  float64 `json:"recommended_backoff_ms"`
	HistoricalSuccessRate float64 `json:"historical_success_rate"`
	OptimalConcurrency    int     `json:"optimal_concurrency"`
	Confidence            float64 `json:"confidence"`
}

// ConcurrencyAdjustment records a concurrency change for an endpoint.
type ConcurrencyAdjustment struct {
	EndpointID       string  `json:"endpoint_id"`
	OldConcurrency   int     `json:"old_concurrency"`
	NewConcurrency   int     `json:"new_concurrency"`
	Reason           string  `json:"reason"`
	SuccessRateDelta float64 `json:"success_rate_delta"`
}
