package georouting

import (
	"time"

	"github.com/google/uuid"
)

// Region represents a geographic region
type Region string

const (
	RegionUSEast    Region = "us-east-1"
	RegionUSWest    Region = "us-west-2"
	RegionEUWest    Region = "eu-west-1"
	RegionEUCentral Region = "eu-central-1"
	RegionAPSouth   Region = "ap-south-1"
	RegionAPEast    Region = "ap-northeast-1"
)

// DataResidency defines data residency requirements
type DataResidency string

const (
	ResidencyNone   DataResidency = "none"
	ResidencyUS     DataResidency = "us"
	ResidencyEU     DataResidency = "eu"
	ResidencyAPAC   DataResidency = "apac"
	ResidencyStrict DataResidency = "strict" // Payload never leaves specified region
)

// RegionConfig holds configuration for a region
type RegionConfig struct {
	ID            string       `json:"id" db:"id"`
	Region        Region       `json:"region" db:"region"`
	Name          string       `json:"name" db:"name"`
	Endpoint      string       `json:"endpoint" db:"endpoint"`
	IsActive      bool         `json:"is_active" db:"is_active"`
	IsPrimary     bool         `json:"is_primary" db:"is_primary"`
	Priority      int          `json:"priority" db:"priority"`
	MaxConcurrent int          `json:"max_concurrent" db:"max_concurrent"`
	HealthCheck   HealthConfig `json:"health_check" db:"health_check"`
	Metadata      RegionMeta   `json:"metadata" db:"metadata"`
	CreatedAt     time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at" db:"updated_at"`
}

// HealthConfig defines health check parameters
type HealthConfig struct {
	Interval       time.Duration `json:"interval"`
	Timeout        time.Duration `json:"timeout"`
	UnhealthyAfter int           `json:"unhealthy_after"`
	HealthyAfter   int           `json:"healthy_after"`
}

// RegionMeta contains region metadata
type RegionMeta struct {
	Latitude   float64  `json:"latitude"`
	Longitude  float64  `json:"longitude"`
	Country    string   `json:"country"`
	Continents []string `json:"continents"`
	Provider   string   `json:"provider"` // aws, gcp, azure
}

// RegionHealth tracks health status of a region
type RegionHealth struct {
	RegionID        string     `json:"region_id" db:"region_id"`
	IsHealthy       bool       `json:"is_healthy" db:"is_healthy"`
	LastCheck       time.Time  `json:"last_check" db:"last_check"`
	ConsecutiveOK   int        `json:"consecutive_ok" db:"consecutive_ok"`
	ConsecutiveFail int        `json:"consecutive_fail" db:"consecutive_fail"`
	AvgLatencyMs    int        `json:"avg_latency_ms" db:"avg_latency_ms"`
	ErrorRate       float64    `json:"error_rate" db:"error_rate"`
	LastError       string     `json:"last_error,omitempty" db:"last_error"`
	LastErrorAt     *time.Time `json:"last_error_at,omitempty" db:"last_error_at"`
}

// EndpointRouting defines routing configuration for an endpoint
type EndpointRouting struct {
	ID              string        `json:"id" db:"id"`
	EndpointID      string        `json:"endpoint_id" db:"endpoint_id"`
	TenantID        string        `json:"tenant_id" db:"tenant_id"`
	Mode            RoutingMode   `json:"mode" db:"mode"`
	PrimaryRegion   Region        `json:"primary_region" db:"primary_region"`
	Regions         []Region      `json:"regions" db:"regions"`
	DataResidency   DataResidency `json:"data_residency" db:"data_residency"`
	FailoverEnabled bool          `json:"failover_enabled" db:"failover_enabled"`
	LatencyBased    bool          `json:"latency_based" db:"latency_based"`
	CreatedAt       time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at" db:"updated_at"`
}

// RoutingMode defines how routing decisions are made
type RoutingMode string

const (
	ModeManual   RoutingMode = "manual"   // User specifies region
	ModeAuto     RoutingMode = "auto"     // System chooses based on latency
	ModeGeo      RoutingMode = "geo"      // Route to nearest region
	ModeFailover RoutingMode = "failover" // Primary with failover
)

// RoutingDecision represents a routing decision
type RoutingDecision struct {
	EndpointID     string    `json:"endpoint_id"`
	SelectedRegion Region    `json:"selected_region"`
	Reason         string    `json:"reason"`
	LatencyMs      int       `json:"latency_ms,omitempty"`
	Alternatives   []Region  `json:"alternatives,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
}

// GeoLocation represents a geographic location
type GeoLocation struct {
	IP        string  `json:"ip"`
	Country   string  `json:"country"`
	Region    string  `json:"region"`
	City      string  `json:"city"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	ISP       string  `json:"isp,omitempty"`
}

// RoutingStats tracks routing statistics
type RoutingStats struct {
	TenantID      string                `json:"tenant_id"`
	Period        string                `json:"period"`
	ByRegion      map[Region]int64      `json:"by_region"`
	ByMode        map[RoutingMode]int64 `json:"by_mode"`
	Failovers     int64                 `json:"failovers"`
	AvgDecisionMs int                   `json:"avg_decision_ms"`
	Timestamp     time.Time             `json:"timestamp"`
}

// CreateRoutingRequest represents a request to create endpoint routing
type CreateRoutingRequest struct {
	EndpointID      string        `json:"endpoint_id" binding:"required"`
	Mode            RoutingMode   `json:"mode" binding:"required"`
	PrimaryRegion   Region        `json:"primary_region"`
	Regions         []Region      `json:"regions"`
	DataResidency   DataResidency `json:"data_residency"`
	FailoverEnabled bool          `json:"failover_enabled"`
	LatencyBased    bool          `json:"latency_based"`
}

// UpdateRoutingRequest represents a request to update endpoint routing
type UpdateRoutingRequest struct {
	Mode            *RoutingMode   `json:"mode,omitempty"`
	PrimaryRegion   *Region        `json:"primary_region,omitempty"`
	Regions         []Region       `json:"regions,omitempty"`
	DataResidency   *DataResidency `json:"data_residency,omitempty"`
	FailoverEnabled *bool          `json:"failover_enabled,omitempty"`
	LatencyBased    *bool          `json:"latency_based,omitempty"`
}

// AllRegions returns all supported regions
func AllRegions() []Region {
	return []Region{
		RegionUSEast,
		RegionUSWest,
		RegionEUWest,
		RegionEUCentral,
		RegionAPSouth,
		RegionAPEast,
	}
}

// RegionInfo provides details about a region
type RegionInfo struct {
	Region    Region   `json:"region"`
	Name      string   `json:"name"`
	Location  string   `json:"location"`
	Continent string   `json:"continent"`
	Compliant []string `json:"compliant"` // Compliance certifications
}

// GetRegionInfo returns information about all regions
func GetRegionInfo() []RegionInfo {
	return []RegionInfo{
		{Region: RegionUSEast, Name: "US East (N. Virginia)", Location: "Virginia, USA", Continent: "North America", Compliant: []string{"SOC2", "HIPAA"}},
		{Region: RegionUSWest, Name: "US West (Oregon)", Location: "Oregon, USA", Continent: "North America", Compliant: []string{"SOC2", "HIPAA"}},
		{Region: RegionEUWest, Name: "EU West (Ireland)", Location: "Dublin, Ireland", Continent: "Europe", Compliant: []string{"SOC2", "GDPR"}},
		{Region: RegionEUCentral, Name: "EU Central (Frankfurt)", Location: "Frankfurt, Germany", Continent: "Europe", Compliant: []string{"SOC2", "GDPR"}},
		{Region: RegionAPSouth, Name: "Asia Pacific (Mumbai)", Location: "Mumbai, India", Continent: "Asia", Compliant: []string{"SOC2"}},
		{Region: RegionAPEast, Name: "Asia Pacific (Tokyo)", Location: "Tokyo, Japan", Continent: "Asia", Compliant: []string{"SOC2"}},
	}
}

// GeoRegion represents a geographic region with full metadata
type GeoRegion struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	DisplayName string    `json:"display_name" db:"display_name"`
	Provider    string    `json:"provider" db:"provider"`
	Latitude    float64   `json:"latitude" db:"latitude"`
	Longitude   float64   `json:"longitude" db:"longitude"`
	Status      string    `json:"status" db:"status"`
	Capacity    int       `json:"capacity" db:"capacity"`
	CurrentLoad int       `json:"current_load" db:"current_load"`
	AvgLatency  int       `json:"avg_latency_ms" db:"avg_latency_ms"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// GeoRoutingPolicy defines how events are routed geographically
type GeoRoutingPolicy struct {
	ID               uuid.UUID      `json:"id" db:"id"`
	TenantID         uuid.UUID      `json:"tenant_id" db:"tenant_id"`
	Name             string         `json:"name" db:"name"`
	Strategy         string         `json:"strategy" db:"strategy"`
	DataResidencyReq []string       `json:"data_residency" db:"data_residency"`
	PreferredRegions []string       `json:"preferred_regions" db:"preferred_regions"`
	FailoverOrder    []string       `json:"failover_order" db:"failover_order"`
	Weights          map[string]int `json:"weights" db:"weights"`
	Active           bool           `json:"active" db:"active"`
	CreatedAt        time.Time      `json:"created_at" db:"created_at"`
}

// EndpointRegionConfig defines per-endpoint region preferences
type EndpointRegionConfig struct {
	EndpointID      uuid.UUID `json:"endpoint_id" db:"endpoint_id"`
	PrimaryRegion   string    `json:"primary_region" db:"primary_region"`
	FailoverRegions []string  `json:"failover_regions" db:"failover_regions"`
	DataResidencyRq string    `json:"data_residency" db:"data_residency"`
}

// GeoRoutingDecision captures how and why a routing decision was made
type GeoRoutingDecision struct {
	EventID            uuid.UUID `json:"event_id"`
	SelectedRegion     string    `json:"selected_region"`
	Reason             string    `json:"reason"`
	Latency            int       `json:"estimated_latency_ms"`
	AlternativeRegions []string  `json:"alternative_regions"`
}

// GeoRegionHealth represents enriched health data for a region
type GeoRegionHealth struct {
	RegionName  string    `json:"region_name"`
	Status      string    `json:"status"`
	AvgLatency  int       `json:"avg_latency_ms"`
	SuccessRate float64   `json:"success_rate"`
	Load        float64   `json:"load_percentage"`
	LastChecked time.Time `json:"last_checked"`
}

// GeoDashboardData provides an overview for the geo-routing dashboard
type GeoDashboardData struct {
	Regions          []GeoRegion       `json:"regions"`
	Health           []GeoRegionHealth `json:"health"`
	LoadDistribution map[string]float64 `json:"load_distribution"`
	LatencyMap       map[string]int     `json:"latency_map"`
}

// CreateGeoRoutingPolicyRequest is the request body for creating a routing policy
type CreateGeoRoutingPolicyRequest struct {
	Name             string         `json:"name" binding:"required"`
	Strategy         string         `json:"strategy" binding:"required"`
	DataResidency    []string       `json:"data_residency"`
	PreferredRegions []string       `json:"preferred_regions"`
	FailoverOrder    []string       `json:"failover_order"`
	Weights          map[string]int `json:"weights"`
}

// SimulateRoutingRequest is the request body for routing simulation
type SimulateRoutingRequest struct {
	SourceIP string `json:"source_ip" binding:"required"`
}

// ConfigureEndpointRegionRequest is the request body for endpoint region configuration
type ConfigureEndpointRegionRequest struct {
	PrimaryRegion   string   `json:"primary_region" binding:"required"`
	FailoverRegions []string `json:"failover_regions"`
	DataResidency   string   `json:"data_residency"`
}
