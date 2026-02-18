package multiregion

import (
	"math"
	"sort"
	"sync"
	"time"
)

// GeoRouter routes requests to the optimal region based on geography,
// latency, health, and data residency constraints.
type GeoRouter struct {
	regions      map[string]*Region
	health       map[string]*RegionHealth
	policies     map[string]*RoutingPolicy // key: tenantID
	residency    map[string]*DataResidencyPolicy
	mu           sync.RWMutex
	roundRobinIdx int
}

// NewGeoRouter creates a new geo-aware router
func NewGeoRouter() *GeoRouter {
	return &GeoRouter{
		regions:   make(map[string]*Region),
		health:    make(map[string]*RegionHealth),
		policies:  make(map[string]*RoutingPolicy),
		residency: make(map[string]*DataResidencyPolicy),
	}
}

// RegisterRegion adds a region to the routing table
func (r *GeoRouter) RegisterRegion(region *Region) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.regions[region.ID] = region
}

// UpdateHealth updates the health status for a region
func (r *GeoRouter) UpdateHealth(regionID string, health *RegionHealth) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.health[regionID] = health
}

// SetPolicy sets the routing policy for a tenant
func (r *GeoRouter) SetPolicy(tenantID string, policy *RoutingPolicy) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.policies[tenantID] = policy
}

// SetResidencyPolicy sets data residency constraints for a tenant
func (r *GeoRouter) SetResidencyPolicy(tenantID string, policy *DataResidencyPolicy) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.residency[tenantID] = policy
}

// RouteRequest determines the best region for a request
func (r *GeoRouter) RouteRequest(tenantID string, sourceCoord *GeoCoord, countryCode string) *RouteDecision {
	r.mu.Lock()
	defer r.mu.Unlock()

	decision := &RouteDecision{
		TenantID:  tenantID,
		Timestamp: time.Now(),
	}

	// Get candidate regions (healthy only)
	candidates := r.getHealthyCandidates()
	if len(candidates) == 0 {
		decision.Error = "no healthy regions available"
		return decision
	}

	// Apply data residency constraints
	residencyPolicy := r.residency[tenantID]
	if residencyPolicy != nil {
		candidates = r.applyResidencyFilter(candidates, countryCode, residencyPolicy)
		if len(candidates) == 0 {
			decision.Error = "no regions satisfy data residency requirements"
			return decision
		}
		decision.ResidencyEnforced = true
	}

	// Route based on policy
	policy := r.policies[tenantID]
	if policy == nil {
		// Default: geo-proximity
		decision.Region = r.routeByProximity(candidates, sourceCoord)
		decision.Strategy = string(RoutingTypeGeoProximity)
	} else {
		switch policy.PolicyType {
		case RoutingTypeGeoProximity:
			decision.Region = r.routeByProximity(candidates, sourceCoord)
		case RoutingTypeLatencyBased:
			decision.Region = r.routeByLatency(candidates)
		case RoutingTypeWeighted:
			decision.Region = r.routeByWeight(candidates, policy.Weights)
		case RoutingTypeRoundRobin:
			decision.Region = r.routeByRoundRobin(candidates)
		case RoutingTypePrimaryBackup:
			decision.Region = r.routeByPrimaryBackup(candidates, policy)
		default:
			decision.Region = r.routeByProximity(candidates, sourceCoord)
		}
		decision.Strategy = string(policy.PolicyType)
	}

	if decision.Region != nil {
		decision.RegionID = decision.Region.ID
		decision.Endpoint = decision.Region.Endpoint
	}

	return decision
}

// RouteDecision represents a routing decision
type RouteDecision struct {
	TenantID          string    `json:"tenant_id"`
	RegionID          string    `json:"region_id"`
	Endpoint          string    `json:"endpoint"`
	Strategy          string    `json:"strategy"`
	ResidencyEnforced bool      `json:"residency_enforced"`
	Timestamp         time.Time `json:"timestamp"`
	Error             string    `json:"error,omitempty"`
	Region            *Region   `json:"-"`
}

func (r *GeoRouter) getHealthyCandidates() []*Region {
	var candidates []*Region
	for id, region := range r.regions {
		if !region.IsActive {
			continue
		}
		health, hasHealth := r.health[id]
		if hasHealth && health.Status == HealthStatusUnhealthy {
			continue
		}
		candidates = append(candidates, region)
	}
	return candidates
}

func (r *GeoRouter) applyResidencyFilter(candidates []*Region, countryCode string, policy *DataResidencyPolicy) []*Region {
	if len(policy.AllowedRegions) == 0 && len(policy.BlockedRegions) == 0 {
		return candidates
	}

	var filtered []*Region
	for _, region := range candidates {
		// Check blocked list
		blocked := false
		for _, br := range policy.BlockedRegions {
			if region.Code == br || region.ID == br {
				blocked = true
				break
			}
		}
		if blocked {
			continue
		}

		// Check allowed list
		if len(policy.AllowedRegions) > 0 {
			allowed := false
			for _, ar := range policy.AllowedRegions {
				if region.Code == ar || region.ID == ar {
					allowed = true
					break
				}
			}
			if !allowed {
				continue
			}
		}

		filtered = append(filtered, region)
	}

	return filtered
}

func (r *GeoRouter) routeByProximity(candidates []*Region, sourceCoord *GeoCoord) *Region {
	if sourceCoord == nil || len(candidates) == 0 {
		// Fallback to primary or first candidate
		return r.findPrimaryOrFirst(candidates)
	}

	var nearest *Region
	minDist := math.MaxFloat64

	for _, region := range candidates {
		if region.Metadata.Coordinates == nil {
			continue
		}
		dist := haversine(
			sourceCoord.Latitude, sourceCoord.Longitude,
			region.Metadata.Coordinates.Latitude, region.Metadata.Coordinates.Longitude,
		)
		if dist < minDist {
			minDist = dist
			nearest = region
		}
	}

	if nearest != nil {
		return nearest
	}
	return r.findPrimaryOrFirst(candidates)
}

func (r *GeoRouter) routeByLatency(candidates []*Region) *Region {
	var best *Region
	var bestLatency time.Duration = math.MaxInt64

	for _, region := range candidates {
		health, ok := r.health[region.ID]
		if !ok {
			continue
		}
		if health.Latency < bestLatency {
			bestLatency = health.Latency
			best = region
		}
	}

	if best != nil {
		return best
	}
	return r.findPrimaryOrFirst(candidates)
}

func (r *GeoRouter) routeByWeight(candidates []*Region, weights map[string]int) *Region {
	if len(weights) == 0 {
		return r.findPrimaryOrFirst(candidates)
	}

	totalWeight := 0
	type weightedRegion struct {
		region *Region
		weight int
	}
	var weighted []weightedRegion

	for _, region := range candidates {
		w, ok := weights[region.ID]
		if !ok {
			w = 1
		}
		totalWeight += w
		weighted = append(weighted, weightedRegion{region: region, weight: w})
	}

	if totalWeight == 0 {
		return r.findPrimaryOrFirst(candidates)
	}

	// Deterministic weighted selection based on current time
	target := int(time.Now().UnixNano()) % totalWeight
	cumulative := 0
	for _, wr := range weighted {
		cumulative += wr.weight
		if target < cumulative {
			return wr.region
		}
	}

	return candidates[0]
}

func (r *GeoRouter) routeByRoundRobin(candidates []*Region) *Region {
	if len(candidates) == 0 {
		return nil
	}
	idx := r.roundRobinIdx % len(candidates)
	r.roundRobinIdx++
	return candidates[idx]
}

func (r *GeoRouter) routeByPrimaryBackup(candidates []*Region, policy *RoutingPolicy) *Region {
	// Try primary first
	for _, region := range candidates {
		if region.ID == policy.PrimaryRegion {
			health, ok := r.health[region.ID]
			if !ok || health.Status != HealthStatusUnhealthy {
				return region
			}
		}
	}

	// Try fallback regions in order
	for _, fallbackID := range policy.FallbackRegions {
		for _, region := range candidates {
			if region.ID == fallbackID {
				return region
			}
		}
	}

	return r.findPrimaryOrFirst(candidates)
}

func (r *GeoRouter) findPrimaryOrFirst(candidates []*Region) *Region {
	if len(candidates) == 0 {
		return nil
	}
	// Sort by priority
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Priority < candidates[j].Priority
	})
	for _, region := range candidates {
		if region.IsPrimary {
			return region
		}
	}
	return candidates[0]
}

// haversine calculates the great-circle distance between two points in km
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0

	dLat := degToRad(lat2 - lat1)
	dLon := degToRad(lon2 - lon1)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(degToRad(lat1))*math.Cos(degToRad(lat2))*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}

func degToRad(deg float64) float64 {
	return deg * math.Pi / 180
}

// CrossRegionSyncState tracks synchronization state between regions
type CrossRegionSyncState struct {
	SourceRegion     string    `json:"source_region"`
	TargetRegion     string    `json:"target_region"`
	LastSyncedAt     time.Time `json:"last_synced_at"`
	PendingEvents    int64     `json:"pending_events"`
	ReplicationLagMs int64     `json:"replication_lag_ms"`
	Status           string    `json:"status"` // synced, syncing, lagging, error
	BytesSynced      int64     `json:"bytes_synced"`
	ErrorCount       int64     `json:"error_count"`
}

// MeshTopology describes the federation mesh topology
type MeshTopology struct {
	Regions     []*Region              `json:"regions"`
	Connections []MeshConnection       `json:"connections"`
	SyncStates  []CrossRegionSyncState `json:"sync_states"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// MeshConnection represents a connection between two regions in the mesh
type MeshConnection struct {
	SourceRegion string  `json:"source_region"`
	TargetRegion string  `json:"target_region"`
	LatencyMs    float64 `json:"latency_ms"`
	Bandwidth    int64   `json:"bandwidth_mbps"`
	Status       string  `json:"status"` // active, degraded, down
	Bidirectional bool   `json:"bidirectional"`
}
