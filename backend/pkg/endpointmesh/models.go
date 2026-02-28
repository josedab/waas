package endpointmesh

import "time"

// MeshNodeStatus represents the health status of a mesh node.
type MeshNodeStatus string

const (
	StatusHealthy     MeshNodeStatus = "healthy"
	StatusDegraded    MeshNodeStatus = "degraded"
	StatusUnhealthy   MeshNodeStatus = "unhealthy"
	StatusCircuitOpen MeshNodeStatus = "circuit_open"
	StatusRecovering  MeshNodeStatus = "recovering"
)

// CircuitStateValue constants for the circuit breaker.
const (
	CircuitClosed   = "closed"
	CircuitOpen     = "open"
	CircuitHalfOpen = "half_open"
)

// MeshNode represents an endpoint in the mesh.
type MeshNode struct {
	ID                  string         `json:"id"`
	TenantID            string         `json:"tenant_id"`
	EndpointID          string         `json:"endpoint_id"`
	URL                 string         `json:"url"`
	Status              MeshNodeStatus `json:"status"`
	HealthScore         float64        `json:"health_score"`
	ConsecutiveFailures int            `json:"consecutive_failures"`
	LastSuccessAt       *time.Time     `json:"last_success_at,omitempty"`
	LastFailureAt       *time.Time     `json:"last_failure_at,omitempty"`
	CircuitState        CircuitState   `json:"circuit_state"`
	FallbackNodeID      string         `json:"fallback_node_id,omitempty"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
}

// CircuitState tracks the circuit breaker state for a node.
type CircuitState struct {
	State           string     `json:"state"` // closed, open, half_open
	OpenedAt        *time.Time `json:"opened_at,omitempty"`
	ClosedAt        *time.Time `json:"closed_at,omitempty"`
	HalfOpenAt      *time.Time `json:"half_open_at,omitempty"`
	FailureCount    int        `json:"failure_count"`
	SuccessCount    int        `json:"success_count"`
	LastEvaluatedAt time.Time  `json:"last_evaluated_at"`
}

// HealthCheck records a single health check result.
type HealthCheck struct {
	ID         string    `json:"id"`
	NodeID     string    `json:"node_id"`
	StatusCode int       `json:"status_code"`
	LatencyMs  int64     `json:"latency_ms"`
	Success    bool      `json:"success"`
	Error      string    `json:"error,omitempty"`
	CheckedAt  time.Time `json:"checked_at"`
}

// MeshTopology describes the current state of the mesh.
type MeshTopology struct {
	Nodes        []*MeshNode      `json:"nodes"`
	Connections  []MeshConnection `json:"connections"`
	HealthyCount int              `json:"healthy_count"`
	TotalCount   int              `json:"total_count"`
}

// MeshConnection represents a link between two nodes.
type MeshConnection struct {
	SourceID string `json:"source_id"`
	TargetID string `json:"target_id"`
	Type     string `json:"type"` // primary, fallback
	Active   bool   `json:"active"`
}

// RerouteEvent records a traffic reroute action.
type RerouteEvent struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	SourceNodeID  string    `json:"source_node_id"`
	TargetNodeID  string    `json:"target_node_id"`
	Reason        string    `json:"reason"`
	AutoRecovered bool      `json:"auto_recovered"`
	CreatedAt     time.Time `json:"created_at"`
}

// MeshConfig configures the endpoint mesh behaviour.
type MeshConfig struct {
	HealthCheckInterval time.Duration // interval between health checks
	FailureThreshold    int           // consecutive failures before circuit opens
	RecoveryThreshold   int           // consecutive successes before circuit closes
	CircuitOpenDuration time.Duration // how long circuit stays open before half-open
}

// DefaultMeshConfig returns sensible defaults.
func DefaultMeshConfig() *MeshConfig {
	return &MeshConfig{
		HealthCheckInterval: 30 * time.Second,
		FailureThreshold:    5,
		RecoveryThreshold:   3,
		CircuitOpenDuration: 60 * time.Second,
	}
}

// CreateMeshNodeRequest is the DTO for adding a node.
type CreateMeshNodeRequest struct {
	EndpointID string `json:"endpoint_id" binding:"required"`
	URL        string `json:"url" binding:"required"`
}

// SetFallbackRequest is the DTO for setting a fallback node.
type SetFallbackRequest struct {
	FallbackNodeID string `json:"fallback_node_id" binding:"required"`
}
