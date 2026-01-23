package cloudctl

import "time"

// CloudTenant represents a tenant in the managed cloud offering
type CloudTenant struct {
	ID             string    `json:"id" db:"id"`
	Name           string    `json:"name" db:"name"`
	Email          string    `json:"email" db:"email"`
	Plan           PlanTier  `json:"plan" db:"plan"`
	Region         string    `json:"region" db:"region"`
	Status         string    `json:"status" db:"status"`
	APIKeyHash     string    `json:"-" db:"api_key_hash"`
	Namespace      string    `json:"namespace" db:"namespace"`
	ResourceQuota  *ResourceQuota `json:"resource_quota,omitempty"`
	QuotaJSON      string    `json:"-" db:"resource_quota"`
	ProvisionedAt  *time.Time `json:"provisioned_at,omitempty" db:"provisioned_at"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// PlanTier represents a subscription plan
type PlanTier string

const (
	PlanFree       PlanTier = "free"
	PlanStarter    PlanTier = "starter"
	PlanPro        PlanTier = "pro"
	PlanEnterprise PlanTier = "enterprise"
)

// TenantStatus constants
const (
	StatusProvisioning = "provisioning"
	StatusActive       = "active"
	StatusSuspended    = "suspended"
	StatusDeleted      = "deleted"
)

// ResourceQuota defines resource limits for a cloud tenant
type ResourceQuota struct {
	MaxEndpoints     int   `json:"max_endpoints"`
	MaxDeliveriesDay int   `json:"max_deliveries_per_day"`
	MaxPayloadBytes  int   `json:"max_payload_bytes"`
	MaxRetries       int   `json:"max_retries"`
	RatePerMinute    int   `json:"rate_per_minute"`
	StorageMB        int   `json:"storage_mb"`
}

// UsageMetrics captures current usage for a tenant
type UsageMetrics struct {
	TenantID        string    `json:"tenant_id" db:"tenant_id"`
	Period          string    `json:"period" db:"period"`
	DeliveriesCount int       `json:"deliveries_count" db:"deliveries_count"`
	EndpointsCount  int       `json:"endpoints_count" db:"endpoints_count"`
	StorageUsedMB   float64   `json:"storage_used_mb" db:"storage_used_mb"`
	BandwidthMB     float64   `json:"bandwidth_mb" db:"bandwidth_mb"`
	APICallsCount   int       `json:"api_calls_count" db:"api_calls_count"`
	MeasuredAt      time.Time `json:"measured_at" db:"measured_at"`
}

// ScalingConfig defines auto-scaling settings
type ScalingConfig struct {
	ID            string `json:"id" db:"id"`
	TenantID      string `json:"tenant_id" db:"tenant_id"`
	MinWorkers    int    `json:"min_workers" db:"min_workers"`
	MaxWorkers    int    `json:"max_workers" db:"max_workers"`
	TargetCPUPct  int    `json:"target_cpu_pct" db:"target_cpu_pct"`
	ScaleUpDelay  int    `json:"scale_up_delay_secs" db:"scale_up_delay_secs"`
	ScaleDownDelay int   `json:"scale_down_delay_secs" db:"scale_down_delay_secs"`
	Enabled       bool   `json:"enabled" db:"enabled"`
}

// ProvisionRequest is the request DTO for provisioning a new cloud tenant
type ProvisionRequest struct {
	Name   string   `json:"name" binding:"required,min=1,max=100"`
	Email  string   `json:"email" binding:"required,email"`
	Plan   PlanTier `json:"plan" binding:"required"`
	Region string   `json:"region" binding:"required"`
}

// UpdatePlanRequest is the request DTO for changing a tenant's plan
type UpdatePlanRequest struct {
	Plan PlanTier `json:"plan" binding:"required"`
}

// UpdateScalingRequest is the request DTO for updating scaling config
type UpdateScalingRequest struct {
	MinWorkers     int  `json:"min_workers" binding:"min=0"`
	MaxWorkers     int  `json:"max_workers" binding:"min=1"`
	TargetCPUPct   int  `json:"target_cpu_pct" binding:"min=10,max=90"`
	ScaleUpDelay   int  `json:"scale_up_delay_secs" binding:"min=30"`
	ScaleDownDelay int  `json:"scale_down_delay_secs" binding:"min=60"`
	Enabled        bool `json:"enabled"`
}

// CloudDashboard provides an overview of the cloud platform
type CloudDashboard struct {
	TotalTenants   int            `json:"total_tenants"`
	ActiveTenants  int            `json:"active_tenants"`
	TotalEndpoints int            `json:"total_endpoints"`
	DeliveriesToday int           `json:"deliveries_today"`
	PlanDistribution map[string]int `json:"plan_distribution"`
	RegionDistribution map[string]int `json:"region_distribution"`
}

// AvailableRegions lists supported cloud regions
var AvailableRegions = []string{
	"us-east-1",
	"us-west-2",
	"eu-west-1",
	"eu-central-1",
	"ap-southeast-1",
	"ap-northeast-1",
}

// PlanQuotas maps plan tiers to their resource quotas
var PlanQuotas = map[PlanTier]ResourceQuota{
	PlanFree: {
		MaxEndpoints: 3, MaxDeliveriesDay: 1000, MaxPayloadBytes: 65536,
		MaxRetries: 3, RatePerMinute: 60, StorageMB: 100,
	},
	PlanStarter: {
		MaxEndpoints: 25, MaxDeliveriesDay: 50000, MaxPayloadBytes: 262144,
		MaxRetries: 5, RatePerMinute: 300, StorageMB: 1024,
	},
	PlanPro: {
		MaxEndpoints: 100, MaxDeliveriesDay: 500000, MaxPayloadBytes: 1048576,
		MaxRetries: 10, RatePerMinute: 1000, StorageMB: 10240,
	},
	PlanEnterprise: {
		MaxEndpoints: 1000, MaxDeliveriesDay: 10000000, MaxPayloadBytes: 5242880,
		MaxRetries: 20, RatePerMinute: 10000, StorageMB: 102400,
	},
}
