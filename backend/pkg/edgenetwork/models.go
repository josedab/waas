package edgenetwork

import "time"

// NodeStatus represents the operational state of an edge node.
type NodeStatus string

const (
	NodeStatusHealthy     NodeStatus = "healthy"
	NodeStatusDegraded    NodeStatus = "degraded"
	NodeStatusUnhealthy   NodeStatus = "unhealthy"
	NodeStatusMaintenance NodeStatus = "maintenance"
	NodeStatusOffline     NodeStatus = "offline"
)

// Region identifies a geographic deployment region.
type Region string

const (
	RegionUSEast1   Region = "us-east-1"
	RegionUSWest2   Region = "us-west-2"
	RegionEUWest1   Region = "eu-west-1"
	RegionEUCentral Region = "eu-central-1"
	RegionAPSouth   Region = "ap-south-1"
	RegionAPNE1     Region = "ap-northeast-1"
	RegionSAEast    Region = "sa-east-1"
)

// AllRegions returns all supported regions.
func AllRegions() []Region {
	return []Region{RegionUSEast1, RegionUSWest2, RegionEUWest1, RegionEUCentral, RegionAPSouth, RegionAPNE1, RegionSAEast}
}

// EdgeNode represents a single node in the edge delivery network.
type EdgeNode struct {
	ID           string     `json:"id" db:"id"`
	Name         string     `json:"name" db:"name"`
	Region       Region     `json:"region" db:"region"`
	Endpoint     string     `json:"endpoint" db:"endpoint"`
	Status       NodeStatus `json:"status" db:"status"`
	Latitude     float64    `json:"latitude" db:"latitude"`
	Longitude    float64    `json:"longitude" db:"longitude"`
	Capacity     int        `json:"capacity" db:"capacity"`
	ActiveConns  int        `json:"active_connections" db:"active_connections"`
	AvgLatencyMs float64    `json:"avg_latency_ms" db:"avg_latency_ms"`
	SuccessRate  float64    `json:"success_rate" db:"success_rate"`
	LastHealthAt time.Time  `json:"last_health_check" db:"last_health_check"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

// RoutingRule defines how traffic is routed to edge nodes.
type RoutingRule struct {
	ID             string             `json:"id" db:"id"`
	TenantID       string             `json:"tenant_id" db:"tenant_id"`
	Name           string             `json:"name" db:"name"`
	Priority       int                `json:"priority" db:"priority"`
	Strategy       RoutingStrategy    `json:"strategy" db:"strategy"`
	TargetRegions  []Region           `json:"target_regions" db:"-"`
	Conditions     []RoutingCondition `json:"conditions" db:"-"`
	FallbackRegion Region             `json:"fallback_region" db:"fallback_region"`
	Enabled        bool               `json:"enabled" db:"enabled"`
	CreatedAt      time.Time          `json:"created_at" db:"created_at"`
}

// RoutingStrategy determines how requests are distributed.
type RoutingStrategy string

const (
	StrategyLatency    RoutingStrategy = "lowest_latency"
	StrategyGeo        RoutingStrategy = "geo_proximity"
	StrategyRoundRobin RoutingStrategy = "round_robin"
	StrategyWeighted   RoutingStrategy = "weighted"
	StrategyFailover   RoutingStrategy = "failover"
)

// RoutingCondition defines when a routing rule applies.
type RoutingCondition struct {
	Field    string `json:"field"`    // source_region, event_type, endpoint_url
	Operator string `json:"operator"` // eq, ne, contains, prefix
	Value    string `json:"value"`
}

// DeliveryRoute represents a resolved route for a webhook delivery.
type DeliveryRoute struct {
	SourceNode string  `json:"source_node"`
	TargetNode string  `json:"target_node"`
	Region     Region  `json:"region"`
	LatencyMs  float64 `json:"estimated_latency_ms"`
	Hops       int     `json:"hops"`
	Strategy   string  `json:"strategy_used"`
}

// NetworkTopology represents the current state of the edge network.
type NetworkTopology struct {
	Nodes       []EdgeNode       `json:"nodes"`
	Connections []NodeConnection `json:"connections"`
	TotalNodes  int              `json:"total_nodes"`
	HealthyPct  float64          `json:"healthy_percentage"`
	Regions     map[Region]int   `json:"regions"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// NodeConnection represents a link between two edge nodes.
type NodeConnection struct {
	SourceID  string  `json:"source_id"`
	TargetID  string  `json:"target_id"`
	LatencyMs float64 `json:"latency_ms"`
	Bandwidth int     `json:"bandwidth_mbps"`
	Active    bool    `json:"active"`
}

// NetworkMetrics holds aggregate network performance data.
type NetworkMetrics struct {
	TotalDeliveries int64                    `json:"total_deliveries"`
	AvgLatencyMs    float64                  `json:"avg_latency_ms"`
	P99LatencyMs    float64                  `json:"p99_latency_ms"`
	SuccessRate     float64                  `json:"success_rate"`
	RegionMetrics   map[Region]*RegionMetric `json:"region_metrics"`
	MeasuredAt      time.Time                `json:"measured_at"`
}

// RegionMetric holds per-region performance data.
type RegionMetric struct {
	Region           Region  `json:"region"`
	NodeCount        int     `json:"node_count"`
	AvgLatencyMs     float64 `json:"avg_latency_ms"`
	SuccessRate      float64 `json:"success_rate"`
	ActiveDeliveries int     `json:"active_deliveries"`
}

// CreateNodeRequest is the API request for registering an edge node.
type CreateNodeRequest struct {
	Name      string  `json:"name" binding:"required"`
	Region    Region  `json:"region" binding:"required"`
	Endpoint  string  `json:"endpoint" binding:"required"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Capacity  int     `json:"capacity"`
}

// CreateRoutingRuleRequest is the API request for creating a routing rule.
type CreateRoutingRuleRequest struct {
	Name           string             `json:"name" binding:"required"`
	Priority       int                `json:"priority"`
	Strategy       RoutingStrategy    `json:"strategy" binding:"required"`
	TargetRegions  []Region           `json:"target_regions"`
	Conditions     []RoutingCondition `json:"conditions"`
	FallbackRegion Region             `json:"fallback_region"`
}
