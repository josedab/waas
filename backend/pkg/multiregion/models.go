package multiregion

import (
	"time"
)

// Region represents a deployment region
type Region struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Code      string    `json:"code" db:"code"`           // e.g., "us-east-1", "eu-west-1"
	Endpoint  string    `json:"endpoint" db:"endpoint"`   // API endpoint URL
	IsActive  bool      `json:"is_active" db:"is_active"`
	IsPrimary bool      `json:"is_primary" db:"is_primary"`
	Priority  int       `json:"priority" db:"priority"`
	Metadata  Metadata  `json:"metadata" db:"metadata"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Metadata holds region-specific metadata
type Metadata struct {
	Cloud       string   `json:"cloud"`       // aws, gcp, azure
	Datacenter  string   `json:"datacenter"`
	Coordinates *GeoCoord `json:"coordinates,omitempty"`
}

// GeoCoord represents geographical coordinates
type GeoCoord struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// RegionHealth represents the health status of a region
type RegionHealth struct {
	RegionID         string        `json:"region_id"`
	Status           HealthStatus  `json:"status"`
	LastCheck        time.Time     `json:"last_check"`
	Latency          time.Duration `json:"latency"`
	SuccessRate      float64       `json:"success_rate"` // 0-100
	ActiveConnections int          `json:"active_connections"`
	QueueDepth       int           `json:"queue_depth"`
	ErrorRate        float64       `json:"error_rate"`
	Metrics          HealthMetrics `json:"metrics"`
}

// HealthStatus represents region health status
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// HealthMetrics contains detailed health metrics
type HealthMetrics struct {
	CPUUsage     float64 `json:"cpu_usage"`
	MemoryUsage  float64 `json:"memory_usage"`
	DiskUsage    float64 `json:"disk_usage"`
	NetworkIn    int64   `json:"network_in_bytes"`
	NetworkOut   int64   `json:"network_out_bytes"`
	RequestCount int64   `json:"request_count"`
}

// FailoverEvent represents a failover occurrence
type FailoverEvent struct {
	ID           string        `json:"id" db:"id"`
	FromRegion   string        `json:"from_region" db:"from_region"`
	ToRegion     string        `json:"to_region" db:"to_region"`
	Reason       FailoverReason `json:"reason" db:"reason"`
	TriggerType  TriggerType   `json:"trigger_type" db:"trigger_type"`
	Status       FailoverStatus `json:"status" db:"status"`
	StartedAt    time.Time     `json:"started_at" db:"started_at"`
	CompletedAt  *time.Time    `json:"completed_at,omitempty" db:"completed_at"`
	Duration     time.Duration `json:"duration,omitempty"`
	AffectedOps  int64         `json:"affected_ops" db:"affected_ops"`
	Details      string        `json:"details,omitempty" db:"details"`
}

// FailoverReason describes why failover occurred
type FailoverReason string

const (
	FailoverReasonHealthCheck  FailoverReason = "health_check_failure"
	FailoverReasonLatency      FailoverReason = "high_latency"
	FailoverReasonErrorRate    FailoverReason = "high_error_rate"
	FailoverReasonMaintenance  FailoverReason = "planned_maintenance"
	FailoverReasonManual       FailoverReason = "manual_trigger"
	FailoverReasonCapacity     FailoverReason = "capacity_exceeded"
)

// TriggerType describes how failover was triggered
type TriggerType string

const (
	TriggerTypeAutomatic TriggerType = "automatic"
	TriggerTypeManual    TriggerType = "manual"
	TriggerTypeScheduled TriggerType = "scheduled"
)

// FailoverStatus represents failover status
type FailoverStatus string

const (
	FailoverStatusInProgress FailoverStatus = "in_progress"
	FailoverStatusCompleted  FailoverStatus = "completed"
	FailoverStatusFailed     FailoverStatus = "failed"
	FailoverStatusRolledBack FailoverStatus = "rolled_back"
)

// ReplicationConfig holds replication configuration
type ReplicationConfig struct {
	ID               string         `json:"id" db:"id"`
	SourceRegion     string         `json:"source_region" db:"source_region"`
	TargetRegion     string         `json:"target_region" db:"target_region"`
	Mode             ReplicationMode `json:"mode" db:"mode"`
	Enabled          bool           `json:"enabled" db:"enabled"`
	LagThresholdMs   int64          `json:"lag_threshold_ms" db:"lag_threshold_ms"`
	RetentionDays    int            `json:"retention_days" db:"retention_days"`
	Tables           []string       `json:"tables" db:"tables"` // Tables to replicate
	CurrentLagMs     int64          `json:"current_lag_ms"`
	LastSyncAt       *time.Time     `json:"last_sync_at,omitempty" db:"last_sync_at"`
	CreatedAt        time.Time      `json:"created_at" db:"created_at"`
}

// ReplicationMode defines how replication works
type ReplicationMode string

const (
	ReplicationModeAsync        ReplicationMode = "async"         // Eventual consistency
	ReplicationModeSemiSync     ReplicationMode = "semi_sync"     // At least one replica confirmed
	ReplicationModeSync         ReplicationMode = "sync"          // All replicas confirmed
)

// RoutingPolicy defines how requests are routed between regions
type RoutingPolicy struct {
	ID          string          `json:"id" db:"id"`
	TenantID    string          `json:"tenant_id" db:"tenant_id"`
	PolicyType  RoutingType     `json:"policy_type" db:"policy_type"`
	PrimaryRegion string        `json:"primary_region" db:"primary_region"`
	FallbackRegions []string    `json:"fallback_regions" db:"fallback_regions"`
	GeoRules    []GeoRule       `json:"geo_rules,omitempty" db:"geo_rules"`
	Weights     map[string]int  `json:"weights,omitempty" db:"weights"` // For weighted routing
	Enabled     bool            `json:"enabled" db:"enabled"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
}

// RoutingType defines routing strategy
type RoutingType string

const (
	RoutingTypePrimaryBackup RoutingType = "primary_backup" // All to primary, failover to backup
	RoutingTypeGeoProximity  RoutingType = "geo_proximity"  // Route to nearest region
	RoutingTypeWeighted      RoutingType = "weighted"       // Weighted distribution
	RoutingTypeRoundRobin    RoutingType = "round_robin"    // Equal distribution
	RoutingTypeLatencyBased  RoutingType = "latency_based"  // Route to lowest latency
)

// GeoRule defines geographic routing rules
type GeoRule struct {
	Countries []string `json:"countries"` // ISO country codes
	Region    string   `json:"region"`    // Target region
	Priority  int      `json:"priority"`
}

// ConflictResolution defines how write conflicts are resolved
type ConflictResolution struct {
	Strategy  ConflictStrategy `json:"strategy"`
	CustomKey string           `json:"custom_key,omitempty"` // For custom resolution
}

// ConflictStrategy defines conflict resolution approach
type ConflictStrategy string

const (
	ConflictStrategyLastWrite  ConflictStrategy = "last_write_wins"
	ConflictStrategyFirstWrite ConflictStrategy = "first_write_wins"
	ConflictStrategyMerge      ConflictStrategy = "merge"
	ConflictStrategyCustom     ConflictStrategy = "custom"
)

// RegionStats contains region performance statistics
type RegionStats struct {
	RegionID          string    `json:"region_id"`
	Period            string    `json:"period"` // "hourly", "daily"
	Timestamp         time.Time `json:"timestamp"`
	TotalRequests     int64     `json:"total_requests"`
	SuccessfulRequests int64    `json:"successful_requests"`
	FailedRequests    int64     `json:"failed_requests"`
	AvgLatencyMs      float64   `json:"avg_latency_ms"`
	P95LatencyMs      float64   `json:"p95_latency_ms"`
	P99LatencyMs      float64   `json:"p99_latency_ms"`
	BytesIn           int64     `json:"bytes_in"`
	BytesOut          int64     `json:"bytes_out"`
}
