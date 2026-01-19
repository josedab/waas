package models

import (
	"time"

	"github.com/google/uuid"
)

// Region status constants
const (
	RegionStatusActive      = "active"
	RegionStatusInactive    = "inactive"
	RegionStatusMaintenance = "maintenance"
	RegionStatusDraining    = "draining"
)

// Region health status constants
const (
	RegionHealthHealthy   = "healthy"
	RegionHealthDegraded  = "degraded"
	RegionHealthUnhealthy = "unhealthy"
)

// Failover strategy constants
const (
	FailoverStrategyRoundRobin   = "round_robin"
	FailoverStrategyPriority     = "priority"
	FailoverStrategyLatencyBased = "latency_based"
)

// Data residency policy constants
const (
	DataResidencyStrict   = "strict"
	DataResidencyFlexible = "flexible"
	DataResidencyGlobal   = "global"
)

// Replication mode constants
const (
	ReplicationModeSync  = "sync"
	ReplicationModeAsync = "async"
	ReplicationModeNone  = "none"
)

// Geo routing rule types
const (
	GeoRuleLatency     = "latency"
	GeoRuleGeofence    = "geofence"
	GeoRuleLoadBalance = "load_balance"
	GeoRuleFailover    = "failover"
)

// Replication stream types
const (
	StreamTypeEvents  = "events"
	StreamTypeConfigs = "configs"
	StreamTypeState   = "state"
)

// Regional config sync status constants
const (
	RegionalSyncStatusPending  = "pending"
	RegionalSyncStatusSynced   = "synced"
	RegionalSyncStatusConflict = "conflict"
	RegionalSyncStatusFailed   = "failed"
)

// Compliance status constants
const (
	ComplianceCompliant = "compliant"
	ComplianceViolation = "violation"
	ComplianceWarning   = "warning"
)

// Failover status constants
const (
	FailoverInProgress = "in_progress"
	FailoverCompleted  = "completed"
	FailoverFailed     = "failed"
	FailoverRolledBack = "rolled_back"
)

// Region represents a deployment region
type Region struct {
	ID              uuid.UUID              `json:"id" db:"id"`
	Name            string                 `json:"name" db:"name"`
	Code            string                 `json:"code" db:"code"`
	Provider        string                 `json:"provider" db:"provider"`
	Location        string                 `json:"location" db:"location"`
	Latitude        *float64               `json:"latitude,omitempty" db:"latitude"`
	Longitude       *float64               `json:"longitude,omitempty" db:"longitude"`
	Status          string                 `json:"status" db:"status"`
	IsPrimary       bool                   `json:"is_primary" db:"is_primary"`
	HealthStatus    string                 `json:"health_status" db:"health_status"`
	LastHealthCheck *time.Time             `json:"last_health_check,omitempty" db:"last_health_check"`
	CapacityLimit   int                    `json:"capacity_limit" db:"capacity_limit"`
	CurrentLoad     int                    `json:"current_load" db:"current_load"`
	Metadata        map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at" db:"updated_at"`
}

// RegionCluster represents a group of regions for failover
type RegionCluster struct {
	ID               uuid.UUID              `json:"id" db:"id"`
	Name             string                 `json:"name" db:"name"`
	Description      string                 `json:"description,omitempty" db:"description"`
	PrimaryRegionID  *uuid.UUID             `json:"primary_region_id,omitempty" db:"primary_region_id"`
	FailoverStrategy string                 `json:"failover_strategy" db:"failover_strategy"`
	Metadata         map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	Members          []*RegionClusterMember `json:"members,omitempty" db:"-"`
}

// RegionClusterMember represents a region in a cluster
type RegionClusterMember struct {
	ID        uuid.UUID `json:"id" db:"id"`
	ClusterID uuid.UUID `json:"cluster_id" db:"cluster_id"`
	RegionID  uuid.UUID `json:"region_id" db:"region_id"`
	Priority  int       `json:"priority" db:"priority"`
	Weight    float64   `json:"weight" db:"weight"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	Region    *Region   `json:"region,omitempty" db:"-"`
}

// MeshTenantRegion represents tenant region assignment and data residency
type MeshTenantRegion struct {
	ID                   uuid.UUID   `json:"id" db:"id"`
	TenantID             uuid.UUID   `json:"tenant_id" db:"tenant_id"`
	PrimaryRegionID      uuid.UUID   `json:"primary_region_id" db:"primary_region_id"`
	AllowedRegions       []uuid.UUID `json:"allowed_regions" db:"allowed_regions"`
	DataResidencyPolicy  string      `json:"data_residency_policy" db:"data_residency_policy"`
	ReplicationMode      string      `json:"replication_mode" db:"replication_mode"`
	ComplianceFrameworks []string    `json:"compliance_frameworks" db:"compliance_frameworks"`
	CreatedAt            time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time   `json:"updated_at" db:"updated_at"`
	PrimaryRegion        *Region     `json:"primary_region,omitempty" db:"-"`
}

// GeoRoutingRule represents a geo-routing rule
type GeoRoutingRule struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	TenantID       uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	Name           string                 `json:"name" db:"name"`
	Description    string                 `json:"description,omitempty" db:"description"`
	RuleType       string                 `json:"rule_type" db:"rule_type"`
	Priority       int                    `json:"priority" db:"priority"`
	SourceRegions  []uuid.UUID            `json:"source_regions" db:"source_regions"`
	TargetRegionID *uuid.UUID             `json:"target_region_id,omitempty" db:"target_region_id"`
	Conditions     map[string]interface{} `json:"conditions" db:"conditions"`
	Enabled        bool                   `json:"enabled" db:"enabled"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at" db:"updated_at"`
}

// ReplicationStream represents cross-region replication
type ReplicationStream struct {
	ID               uuid.UUID              `json:"id" db:"id"`
	TenantID         uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	SourceRegionID   uuid.UUID              `json:"source_region_id" db:"source_region_id"`
	TargetRegionID   uuid.UUID              `json:"target_region_id" db:"target_region_id"`
	StreamType       string                 `json:"stream_type" db:"stream_type"`
	Status           string                 `json:"status" db:"status"`
	LagMs            int64                  `json:"lag_ms" db:"lag_ms"`
	LastReplicatedAt *time.Time             `json:"last_replicated_at,omitempty" db:"last_replicated_at"`
	LastEventID      *uuid.UUID             `json:"last_event_id,omitempty" db:"last_event_id"`
	ErrorMessage     string                 `json:"error_message,omitempty" db:"error_message"`
	Metadata         map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
}

// RegionalRoutingDecision represents a routing decision
type RegionalRoutingDecision struct {
	ID             uuid.UUID `json:"id" db:"id"`
	TenantID       uuid.UUID `json:"tenant_id" db:"tenant_id"`
	EventID        uuid.UUID `json:"event_id" db:"event_id"`
	SourceRegionID uuid.UUID `json:"source_region_id" db:"source_region_id"`
	TargetRegionID uuid.UUID `json:"target_region_id" db:"target_region_id"`
	RoutingRuleID  *uuid.UUID `json:"routing_rule_id,omitempty" db:"routing_rule_id"`
	DecisionReason string    `json:"decision_reason" db:"decision_reason"`
	LatencyMs      int       `json:"latency_ms" db:"latency_ms"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// RegionHealthMetric represents a health metric for a region
type RegionHealthMetric struct {
	ID          uuid.UUID `json:"id" db:"id"`
	RegionID    uuid.UUID `json:"region_id" db:"region_id"`
	MetricType  string    `json:"metric_type" db:"metric_type"`
	MetricValue float64   `json:"metric_value" db:"metric_value"`
	RecordedAt  time.Time `json:"recorded_at" db:"recorded_at"`
}

// DataResidencyAudit represents a data residency audit entry
type DataResidencyAudit struct {
	ID               uuid.UUID              `json:"id" db:"id"`
	TenantID         uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	EventType        string                 `json:"event_type" db:"event_type"`
	SourceRegionID   *uuid.UUID             `json:"source_region_id,omitempty" db:"source_region_id"`
	TargetRegionID   *uuid.UUID             `json:"target_region_id,omitempty" db:"target_region_id"`
	DataType         string                 `json:"data_type,omitempty" db:"data_type"`
	ComplianceStatus string                 `json:"compliance_status" db:"compliance_status"`
	Details          map[string]interface{} `json:"details" db:"details"`
	RecordedAt       time.Time              `json:"recorded_at" db:"recorded_at"`
}

// FailoverEvent represents a failover event
type FailoverEvent struct {
	ID              uuid.UUID              `json:"id" db:"id"`
	ClusterID       *uuid.UUID             `json:"cluster_id,omitempty" db:"cluster_id"`
	FromRegionID    uuid.UUID              `json:"from_region_id" db:"from_region_id"`
	ToRegionID      uuid.UUID              `json:"to_region_id" db:"to_region_id"`
	TriggerReason   string                 `json:"trigger_reason" db:"trigger_reason"`
	Automatic       bool                   `json:"automatic" db:"automatic"`
	StartedAt       time.Time              `json:"started_at" db:"started_at"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
	Status          string                 `json:"status" db:"status"`
	AffectedTenants int                    `json:"affected_tenants" db:"affected_tenants"`
	Metadata        map[string]interface{} `json:"metadata" db:"metadata"`
}

// RegionalConfigSync represents regional config sync status
type RegionalConfigSync struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	TenantID     uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	ConfigType   string     `json:"config_type" db:"config_type"`
	ConfigID     uuid.UUID  `json:"config_id" db:"config_id"`
	RegionID     uuid.UUID  `json:"region_id" db:"region_id"`
	Version      int        `json:"version" db:"version"`
	SyncStatus   string     `json:"sync_status" db:"sync_status"`
	LastSyncedAt *time.Time `json:"last_synced_at,omitempty" db:"last_synced_at"`
	ConfigHash   string     `json:"config_hash,omitempty" db:"config_hash"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

// Request/Response types

// CreateTenantRegionRequest is the request to assign a tenant to regions
type CreateTenantRegionRequest struct {
	PrimaryRegionID      string   `json:"primary_region_id" binding:"required"`
	AllowedRegions       []string `json:"allowed_regions"`
	DataResidencyPolicy  string   `json:"data_residency_policy"`
	ReplicationMode      string   `json:"replication_mode"`
	ComplianceFrameworks []string `json:"compliance_frameworks"`
}

// CreateGeoRoutingRuleRequest is the request to create a routing rule
type CreateGeoRoutingRuleRequest struct {
	Name           string                 `json:"name" binding:"required"`
	Description    string                 `json:"description"`
	RuleType       string                 `json:"rule_type" binding:"required"`
	Priority       int                    `json:"priority"`
	SourceRegions  []string               `json:"source_regions"`
	TargetRegionID string                 `json:"target_region_id"`
	Conditions     map[string]interface{} `json:"conditions"`
}

// CreateReplicationStreamRequest is the request to create a replication stream
type CreateReplicationStreamRequest struct {
	SourceRegionID string `json:"source_region_id" binding:"required"`
	TargetRegionID string `json:"target_region_id" binding:"required"`
	StreamType     string `json:"stream_type" binding:"required"`
}

// InitiateFailoverRequest is the request to initiate a failover
type InitiateFailoverRequest struct {
	FromRegionID  string `json:"from_region_id" binding:"required"`
	ToRegionID    string `json:"to_region_id" binding:"required"`
	TriggerReason string `json:"trigger_reason"`
}

// RouteEventRequest is the request to route an event
type RouteEventRequest struct {
	EventID        string `json:"event_id" binding:"required"`
	SourceRegionID string `json:"source_region_id" binding:"required"`
}

// MeshDashboard represents the federated mesh dashboard
type MeshDashboard struct {
	TotalRegions        int                    `json:"total_regions"`
	ActiveRegions       int                    `json:"active_regions"`
	HealthyRegions      int                    `json:"healthy_regions"`
	TenantRegion        *MeshTenantRegion      `json:"tenant_region,omitempty"`
	ReplicationStreams  []*ReplicationStream   `json:"replication_streams"`
	ActiveRoutingRules  int                    `json:"active_routing_rules"`
	RecentFailovers     []*FailoverEvent       `json:"recent_failovers"`
	ComplianceStatus    string                 `json:"compliance_status"`
	RegionalHealth      map[string]interface{} `json:"regional_health"`
}

// RegionWithMetrics represents a region with health metrics
type RegionWithMetrics struct {
	*Region
	Latency     float64 `json:"latency_ms"`
	ErrorRate   float64 `json:"error_rate"`
	Throughput  float64 `json:"throughput"`
	Utilization float64 `json:"utilization_percent"`
}
