package topologysim

import "time"

// Topology defines a complete webhook topology for simulation.
type Topology struct {
	ID          string            `json:"id"`
	TenantID    string            `json:"tenant_id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Endpoints   []SimEndpoint     `json:"endpoints"`
	Traffic     []TrafficSource   `json:"traffic"`
	Constraints *InfraConstraints `json:"constraints,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

// SimEndpoint defines a simulated endpoint with failure/latency profiles.
type SimEndpoint struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	FailureRate    float64         `json:"failure_rate"` // 0.0-1.0
	LatencyMean    float64         `json:"latency_mean_ms"`
	LatencyStdDev  float64         `json:"latency_stddev_ms"`
	MaxConcurrency int             `json:"max_concurrency"`
	RetryPolicy    *SimRetryPolicy `json:"retry_policy,omitempty"`
}

// SimRetryPolicy defines retry behavior for simulation.
type SimRetryPolicy struct {
	MaxRetries  int     `json:"max_retries"`
	BackoffBase float64 `json:"backoff_base_ms"`
	BackoffMax  float64 `json:"backoff_max_ms"`
}

// TrafficSource defines a traffic pattern for the simulation.
type TrafficSource struct {
	EventType string   `json:"event_type"`
	TargetIDs []string `json:"target_ids"`
	RPS       float64  `json:"rps"`
	Pattern   string   `json:"pattern"` // constant, burst, ramp, sine
	Duration  string   `json:"duration"`
}

// InfraConstraints models infrastructure limits.
type InfraConstraints struct {
	MaxQueueDepth    int     `json:"max_queue_depth"`
	MaxWorkers       int     `json:"max_workers"`
	NetworkBandwidth float64 `json:"network_bandwidth_mbps"`
	CPUCores         int     `json:"cpu_cores"`
	MemoryGB         float64 `json:"memory_gb"`
}

// SimulationConfig configures a simulation run.
type SimulationConfig struct {
	TopologyID     string `json:"topology_id" binding:"required"`
	Duration       string `json:"duration" binding:"required"`
	MonteCarloRuns int    `json:"monte_carlo_runs"` // 0 = single run
	Seed           int64  `json:"seed"`             // for reproducibility
}

// SimulationResult contains the output of a simulation run.
type SimulationResult struct {
	ID               string           `json:"id"`
	TopologyID       string           `json:"topology_id"`
	Status           string           `json:"status"` // running, completed, failed
	Duration         string           `json:"duration"`
	TotalEvents      int64            `json:"total_events"`
	DeliveredEvents  int64            `json:"delivered_events"`
	FailedEvents     int64            `json:"failed_events"`
	RetriedEvents    int64            `json:"retried_events"`
	AvgLatencyMs     float64          `json:"avg_latency_ms"`
	P95LatencyMs     float64          `json:"p95_latency_ms"`
	P99LatencyMs     float64          `json:"p99_latency_ms"`
	MaxQueueDepth    int              `json:"max_queue_depth"`
	AvgQueueDepth    float64          `json:"avg_queue_depth"`
	RetryStormEvents int              `json:"retry_storm_events"`
	Bottlenecks      []Bottleneck     `json:"bottlenecks"`
	EndpointMetrics  []EndpointMetric `json:"endpoint_metrics"`
	Timeline         []TimelineEvent  `json:"timeline"`
	CostEstimate     *CostEstimate    `json:"cost_estimate,omitempty"`
	Recommendations  []string         `json:"recommendations"`
	CompletedAt      time.Time        `json:"completed_at"`
}

// MonteCarloResult aggregates results across multiple simulation runs.
type MonteCarloResult struct {
	Runs             int     `json:"runs"`
	AvgThroughput    float64 `json:"avg_throughput"`
	P5Throughput     float64 `json:"p5_throughput"`
	P95Throughput    float64 `json:"p95_throughput"`
	AvgFailureRate   float64 `json:"avg_failure_rate"`
	MaxFailureRate   float64 `json:"max_failure_rate"`
	AvgLatencyP99    float64 `json:"avg_latency_p99_ms"`
	RetryStormProb   float64 `json:"retry_storm_probability"`
	CapacityHeadroom float64 `json:"capacity_headroom_pct"`
	Confidence       float64 `json:"confidence_level"`
}

// Bottleneck identifies a resource constraint.
type Bottleneck struct {
	Resource    string  `json:"resource"`    // queue, workers, network, cpu, memory
	Utilization float64 `json:"utilization"` // 0.0-1.0
	Threshold   float64 `json:"threshold"`
	Impact      string  `json:"impact"` // description of impact
}

// EndpointMetric tracks per-endpoint simulation stats.
type EndpointMetric struct {
	EndpointID     string  `json:"endpoint_id"`
	TotalDelivered int64   `json:"total_delivered"`
	TotalFailed    int64   `json:"total_failed"`
	TotalRetried   int64   `json:"total_retried"`
	AvgLatencyMs   float64 `json:"avg_latency_ms"`
	MaxQueueDepth  int     `json:"max_queue_depth"`
}

// TimelineEvent is a discrete event in the simulation timeline.
type TimelineEvent struct {
	Time      float64 `json:"time_ms"`
	EventType string  `json:"event_type"` // delivery, retry, failure, queue_full, retry_storm
	Endpoint  string  `json:"endpoint"`
	Details   string  `json:"details"`
}

// CostEstimate projects infrastructure costs.
type CostEstimate struct {
	ComputeCostPerHour   float64 `json:"compute_cost_per_hour"`
	NetworkCostPerGB     float64 `json:"network_cost_per_gb"`
	EstimatedMonthlyCost float64 `json:"estimated_monthly_cost"`
	CostPerMillionEvents float64 `json:"cost_per_million_events"`
}

// CreateTopologyRequest is the DTO for creating a topology.
type CreateTopologyRequest struct {
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description"`
	Endpoints   []SimEndpoint     `json:"endpoints" binding:"required"`
	Traffic     []TrafficSource   `json:"traffic" binding:"required"`
	Constraints *InfraConstraints `json:"constraints"`
}
