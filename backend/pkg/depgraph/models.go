package depgraph

import "time"

// Dependency represents a producer→consumer edge in the webhook graph.
type Dependency struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	ProducerID      string    `json:"producer_id"`
	ConsumerID      string    `json:"consumer_id"`
	EventTypes      []string  `json:"event_types"`
	DeliveryCount   int64     `json:"delivery_count"`
	SuccessRate     float64   `json:"success_rate"`
	AvgLatencyMs    float64   `json:"avg_latency_ms"`
	HealthStatus    string    `json:"health_status"` // healthy, degraded, critical, unknown
	LastDeliveryAt  time.Time `json:"last_delivery_at"`
	DiscoveredAt    time.Time `json:"discovered_at"`
	LastRefreshedAt time.Time `json:"last_refreshed_at"`
}

// EndpointNode represents a node in the dependency graph.
type EndpointNode struct {
	ID           string   `json:"id"`
	TenantID     string   `json:"tenant_id"`
	Name         string   `json:"name"`
	URL          string   `json:"url"`
	Type         string   `json:"type"` // producer, consumer, both
	HealthStatus string   `json:"health_status"`
	EventTypes   []string `json:"event_types"`
	InDegree     int      `json:"in_degree"`
	OutDegree    int      `json:"out_degree"`
}

// Graph is the full dependency graph for visualization.
type Graph struct {
	Nodes []EndpointNode `json:"nodes"`
	Edges []Dependency   `json:"edges"`
}

// ImpactAnalysis is the result of an impact analysis for an endpoint.
type ImpactAnalysis struct {
	EndpointID        string         `json:"endpoint_id"`
	DirectConsumers   []EndpointNode `json:"direct_consumers"`
	TransitiveClosure []EndpointNode `json:"transitive_closure"`
	BlastRadius       int            `json:"blast_radius"`
	AffectedEvents    []string       `json:"affected_events"`
	RiskLevel         string         `json:"risk_level"` // low, medium, high, critical
	Recommendations   []string       `json:"recommendations"`
}

// GraphRefreshResult tracks the outcome of a graph refresh operation.
type GraphRefreshResult struct {
	NewEdges     int       `json:"new_edges"`
	UpdatedEdges int       `json:"updated_edges"`
	RemovedEdges int       `json:"removed_edges"`
	TotalNodes   int       `json:"total_nodes"`
	TotalEdges   int       `json:"total_edges"`
	RefreshedAt  time.Time `json:"refreshed_at"`
	Duration     string    `json:"duration"`
}
