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
