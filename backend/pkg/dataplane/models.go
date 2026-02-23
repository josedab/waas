package dataplane

import (
	"encoding/json"
	"time"
)

// Plane type constants
const (
	PlaneTypeShared    = "shared"
	PlaneTypeDedicated = "dedicated"
	PlaneTypeIsolated  = "isolated"
)

// Status constants
const (
	StatusProvisioning   = "provisioning"
	StatusReady          = "ready"
	StatusDegraded       = "degraded"
	StatusMigrating      = "migrating"
	StatusDecommissioned = "decommissioned"
)

// DataPlane represents a tenant's data plane configuration.
type DataPlane struct {
	ID             string          `json:"id" db:"id"`
	TenantID       string          `json:"tenant_id" db:"tenant_id"`
	PlaneType      string          `json:"plane_type" db:"plane_type"`
	Status         string          `json:"status" db:"status"`
	DBSchema       string          `json:"db_schema" db:"db_schema"`
	RedisNamespace string          `json:"redis_namespace" db:"redis_namespace"`
	WorkerPoolID   string          `json:"worker_pool_id" db:"worker_pool_id"`
	Config         json.RawMessage `json:"config" db:"config"`
	Region         string          `json:"region" db:"region"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at" db:"updated_at"`
}

// PlaneConfig holds detailed configuration for a data plane.
type PlaneConfig struct {
	MaxConnections  int    `json:"max_connections"`
	MaxWorkers      int    `json:"max_workers"`
	StorageQuotaGB  int    `json:"storage_quota_gb"`
	RateLimitPerSec int    `json:"rate_limit_per_sec"`
	BackupSchedule  string `json:"backup_schedule"`
	EncryptionKey   string `json:"encryption_key,omitempty"`
}

// PlaneHealth holds health metrics for a data plane.
type PlaneHealth struct {
	PlaneID           string    `json:"plane_id"`
	Status            string    `json:"status"`
	DBConnectionsUsed int       `json:"db_connections_used"`
	DBConnectionsMax  int       `json:"db_connections_max"`
	RedisMemoryUsedMB int       `json:"redis_memory_used_mb"`
	WorkerUtilization float64   `json:"worker_utilization"`
	LastHealthCheck   time.Time `json:"last_health_check"`
}

// Request DTOs

type ProvisionPlaneRequest struct {
	TenantID  string      `json:"tenant_id" binding:"required"`
	PlaneType string      `json:"plane_type" binding:"required"`
	Region    string      `json:"region"`
	Config    PlaneConfig `json:"config"`
}

type MigratePlaneRequest struct {
	TargetPlaneType string `json:"target_plane_type" binding:"required"`
}
