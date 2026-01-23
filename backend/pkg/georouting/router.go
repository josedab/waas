package georouting

import (
	"context"
	"fmt"
	"math"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Router handles geographic routing decisions
type Router struct {
	repo          Repository
	geoIP         GeoIPProvider
	healthTracker *HealthTracker
	mu            sync.RWMutex
	regionLatency map[Region]int // cached latency estimates
}

// GeoIPProvider interface for geo-IP lookups
type GeoIPProvider interface {
	Lookup(ip string) (*GeoLocation, error)
}

// NewRouter creates a new geographic router
func NewRouter(repo Repository, geoIP GeoIPProvider) *Router {
	return &Router{
		repo:          repo,
		geoIP:         geoIP,
		healthTracker: NewHealthTracker(repo),
		regionLatency: make(map[Region]int),
	}
}

// Route determines the best region for a webhook delivery
func (r *Router) Route(ctx context.Context, tenantID, endpointID string, clientIP string) (*RoutingDecision, error) {
	// Get endpoint routing configuration
	routing, err := r.repo.GetEndpointRouting(ctx, tenantID, endpointID)
	if err != nil {
		return nil, fmt.Errorf("failed to get routing config: %w", err)
	}

	// Default routing if not configured
	if routing == nil {
		return &RoutingDecision{
			EndpointID:     endpointID,
			SelectedRegion: RegionUSEast,
			Reason:         "default region (no routing configured)",
			Timestamp:      time.Now(),
		}, nil
	}

	// Get healthy regions
	healthyRegions, err := r.getHealthyRegions(ctx, routing.Regions)
	if err != nil {
		return nil, fmt.Errorf("failed to get healthy regions: %w", err)
	}

	decision := &RoutingDecision{
		EndpointID: endpointID,
		Timestamp:  time.Now(),
	}

	switch routing.Mode {
	case ModeManual:
		decision.SelectedRegion = routing.PrimaryRegion
		decision.Reason = "manual selection"
	case ModeGeo:
		region, latency := r.selectByGeo(ctx, clientIP, healthyRegions)
		decision.SelectedRegion = region
		decision.LatencyMs = latency
		decision.Reason = "geographic proximity"
	case ModeFailover:
		region := r.selectWithFailover(ctx, routing.PrimaryRegion, healthyRegions)
		decision.SelectedRegion = region
		if region != routing.PrimaryRegion {
			decision.Reason = fmt.Sprintf("failover from %s", routing.PrimaryRegion)
		} else {
			decision.Reason = "primary region"
		}
	case ModeAuto:
		region, latency := r.selectByLatency(ctx, healthyRegions)
		decision.SelectedRegion = region
		decision.LatencyMs = latency
		decision.Reason = "lowest latency"
	default:
		decision.SelectedRegion = routing.PrimaryRegion
		decision.Reason = "default"
	}

	// Check data residency compliance
	if routing.DataResidency != ResidencyNone {
		if !r.isCompliant(decision.SelectedRegion, routing.DataResidency) {
			compliantRegion := r.findCompliantRegion(healthyRegions, routing.DataResidency)
			if compliantRegion != "" {
				decision.SelectedRegion = compliantRegion
				decision.Reason = fmt.Sprintf("data residency compliance (%s)", routing.DataResidency)
			}
		}
	}

	// Set alternatives
	decision.Alternatives = r.getAlternatives(decision.SelectedRegion, healthyRegions)

	return decision, nil
}

// selectByGeo selects region based on geographic proximity
func (r *Router) selectByGeo(ctx context.Context, clientIP string, regions []Region) (Region, int) {
	if r.geoIP == nil {
		return RegionUSEast, 0
	}

	location, err := r.geoIP.Lookup(clientIP)
	if err != nil {
		return RegionUSEast, 0
	}

	// Find nearest region
	var nearestRegion Region
	minDistance := math.MaxFloat64

	regionCoords := map[Region]struct{ lat, lon float64 }{
		RegionUSEast:    {39.0438, -77.4874},
		RegionUSWest:    {45.8696, -119.6880},
		RegionEUWest:    {53.3498, -6.2603},
		RegionEUCentral: {50.1109, 8.6821},
		RegionAPSouth:   {19.0760, 72.8777},
		RegionAPEast:    {35.6762, 139.6503},
	}

	for _, region := range regions {
		coords, ok := regionCoords[region]
		if !ok {
			continue
		}

		distance := haversine(location.Latitude, location.Longitude, coords.lat, coords.lon)
		if distance < minDistance {
			minDistance = distance
			nearestRegion = region
		}
	}

	// Estimate latency based on distance (rough approximation)
	latencyMs := int(minDistance / 100) // ~100km per ms (speed of light approximation)

	return nearestRegion, latencyMs
}

// selectByLatency selects region with lowest latency
func (r *Router) selectByLatency(ctx context.Context, regions []Region) (Region, int) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var bestRegion Region
	minLatency := math.MaxInt32

	for _, region := range regions {
		if latency, ok := r.regionLatency[region]; ok {
			if latency < minLatency {
				minLatency = latency
				bestRegion = region
			}
		}
	}

	if bestRegion == "" && len(regions) > 0 {
		return regions[0], 0
	}

	return bestRegion, minLatency
}

// selectWithFailover selects primary if healthy, otherwise first healthy backup
func (r *Router) selectWithFailover(ctx context.Context, primary Region, healthyRegions []Region) Region {
	for _, region := range healthyRegions {
		if region == primary {
			return primary
		}
	}

	// Primary is unhealthy, return first healthy
	if len(healthyRegions) > 0 {
		return healthyRegions[0]
	}

	// No healthy regions, return primary anyway
	return primary
}

// getHealthyRegions filters to healthy regions
func (r *Router) getHealthyRegions(ctx context.Context, regions []Region) ([]Region, error) {
	if len(regions) == 0 {
		regions = AllRegions()
	}

	var healthy []Region
	for _, region := range regions {
		health, err := r.healthTracker.GetHealth(ctx, string(region))
		if err != nil {
			continue
		}
		if health == nil || health.IsHealthy {
			healthy = append(healthy, region)
		}
	}

	if len(healthy) == 0 {
		return regions, nil // Return all if none healthy (avoid total failure)
	}

	return healthy, nil
}

// isCompliant checks if region meets data residency requirements
func (r *Router) isCompliant(region Region, residency DataResidency) bool {
	switch residency {
	case ResidencyUS:
		return region == RegionUSEast || region == RegionUSWest
	case ResidencyEU:
		return region == RegionEUWest || region == RegionEUCentral
	case ResidencyAPAC:
		return region == RegionAPSouth || region == RegionAPEast
	case ResidencyStrict:
		return false // Must be explicitly specified
	default:
		return true
	}
}

// findCompliantRegion finds a healthy region meeting residency requirements
func (r *Router) findCompliantRegion(regions []Region, residency DataResidency) Region {
	for _, region := range regions {
		if r.isCompliant(region, residency) {
			return region
		}
	}
	return ""
}

// getAlternatives returns fallback regions
func (r *Router) getAlternatives(selected Region, healthy []Region) []Region {
	var alternatives []Region
	for _, region := range healthy {
		if region != selected {
			alternatives = append(alternatives, region)
		}
	}
	return alternatives
}

// UpdateLatency updates cached latency for a region
func (r *Router) UpdateLatency(region Region, latencyMs int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.regionLatency[region] = latencyMs
}

// haversine calculates distance between two points on Earth
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371 // km

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

// DefaultGeoIPProvider provides basic GeoIP lookup
type DefaultGeoIPProvider struct{}

// Lookup performs basic GeoIP lookup (placeholder - use MaxMind in production)
func (p *DefaultGeoIPProvider) Lookup(ipStr string) (*GeoLocation, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP: %s", ipStr)
	}

	// Basic heuristics for demo - in production use MaxMind GeoIP2
	if ip.IsLoopback() || ip.IsPrivate() {
		return &GeoLocation{
			IP:        ipStr,
			Country:   "US",
			Latitude:  39.0438,
			Longitude: -77.4874,
		}, nil
	}

	// Default to US East
	return &GeoLocation{
		IP:        ipStr,
		Country:   "US",
		Latitude:  39.0438,
		Longitude: -77.4874,
	}, nil
}

// RouterConfig holds configuration for the enhanced router
type RouterConfig struct {
	MaxLatencyThreshold int
	FailoverEnabled     bool
	DataResidencyStrict bool
	HealthCheckInterval time.Duration
}

// DefaultRouterConfig returns sensible defaults
func DefaultRouterConfig() RouterConfig {
	return RouterConfig{
		MaxLatencyThreshold: 500,
		FailoverEnabled:     true,
		DataResidencyStrict: true,
		HealthCheckInterval: 30 * time.Second,
	}
}

// CalculateDistance calculates the Haversine distance (km) between two lat/lon points
func CalculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	return haversine(lat1, lon1, lat2, lon2)
}

// GetClosestRegion returns the GeoRegion closest to the given coordinates
func GetClosestRegion(lat, lon float64, regions []GeoRegion) *GeoRegion {
	if len(regions) == 0 {
		return nil
	}

	var closest *GeoRegion
	minDist := math.MaxFloat64

	for i := range regions {
		d := haversine(lat, lon, regions[i].Latitude, regions[i].Longitude)
		if d < minDist {
			minDist = d
			closest = &regions[i]
		}
	}
	return closest
}

// GetLowestLatencyRegion returns the GeoRegion with the lowest average latency
func GetLowestLatencyRegion(regions []GeoRegion) *GeoRegion {
	if len(regions) == 0 {
		return nil
	}

	var best *GeoRegion
	minLat := math.MaxInt32

	for i := range regions {
		if regions[i].Status != "active" {
			continue
		}
		if regions[i].AvgLatency < minLat {
			minLat = regions[i].AvgLatency
			best = &regions[i]
		}
	}

	if best == nil && len(regions) > 0 {
		best = &regions[0]
	}
	return best
}

// ApplyFailover returns the first healthy region from the failover order.
// If the primary is healthy it is returned; otherwise the first healthy alternative is used.
func (r *Router) ApplyFailover(ctx context.Context, primaryRegion string, failoverOrder []string) (string, error) {
	if r.healthTracker.IsHealthy(primaryRegion) {
		return primaryRegion, nil
	}

	for _, region := range failoverOrder {
		if r.healthTracker.IsHealthy(region) {
			return region, nil
		}
	}
	return "", fmt.Errorf("no healthy region available in failover order")
}

// EnforceDataResidency validates that the selected region satisfies the policy's data residency constraints.
func EnforceDataResidency(policy *GeoRoutingPolicy, selectedRegion string) error {
	if len(policy.DataResidencyReq) == 0 {
		return nil
	}
	for _, allowed := range policy.DataResidencyReq {
		if allowed == selectedRegion {
			return nil
		}
	}
	return fmt.Errorf("region %s violates data residency policy; allowed regions: %v", selectedRegion, policy.DataResidencyReq)
}

// SelectRegion picks the best region given a routing policy and optional target IP
func (r *Router) SelectRegion(ctx context.Context, policy *GeoRoutingPolicy, targetIP string) (string, error) {
	regions, err := r.repo.ListGeoRegions(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list regions: %w", err)
	}

	// Filter to active regions only
	var active []GeoRegion
	for _, rg := range regions {
		if rg.Status == "active" || rg.Status == "degraded" {
			active = append(active, rg)
		}
	}
	if len(active) == 0 {
		return "", fmt.Errorf("no active regions available")
	}

	var selected string

	switch policy.Strategy {
	case "latency":
		best := GetLowestLatencyRegion(active)
		if best != nil {
			selected = best.Name
		}
	case "geo-proximity":
		if r.geoIP != nil && targetIP != "" {
			loc, err := r.geoIP.Lookup(targetIP)
			if err == nil {
				closest := GetClosestRegion(loc.Latitude, loc.Longitude, active)
				if closest != nil {
					selected = closest.Name
				}
			}
		}
		if selected == "" && len(policy.PreferredRegions) > 0 {
			selected = policy.PreferredRegions[0]
		}
	case "failover":
		if len(policy.FailoverOrder) > 0 {
			primary := policy.FailoverOrder[0]
			rest := policy.FailoverOrder[1:]
			result, err := r.ApplyFailover(ctx, primary, rest)
			if err == nil {
				selected = result
			}
		}
	case "round-robin":
		if len(active) > 0 {
			r.mu.Lock()
			idx := r.regionLatency[Region("_rr_idx")]
			r.regionLatency[Region("_rr_idx")] = (idx + 1) % len(active)
			r.mu.Unlock()
			selected = active[idx%len(active)].Name
		}
	case "weighted":
		selected = selectWeightedRegion(active, policy.Weights)
	default:
		if len(policy.PreferredRegions) > 0 {
			selected = policy.PreferredRegions[0]
		} else if len(active) > 0 {
			selected = active[0].Name
		}
	}

	if selected == "" {
		return "", fmt.Errorf("could not select a region for strategy %s", policy.Strategy)
	}

	// Enforce data residency if set
	if err := EnforceDataResidency(policy, selected); err != nil {
		// Try to find a compliant region
		for _, rg := range active {
			if EnforceDataResidency(policy, rg.Name) == nil {
				return rg.Name, nil
			}
		}
		return "", err
	}

	return selected, nil
}

// selectWeightedRegion picks a region using weight distribution
func selectWeightedRegion(regions []GeoRegion, weights map[string]int) string {
	if len(weights) == 0 && len(regions) > 0 {
		return regions[0].Name
	}

	totalWeight := 0
	for _, rg := range regions {
		if w, ok := weights[rg.Name]; ok {
			totalWeight += w
		}
	}
	if totalWeight == 0 && len(regions) > 0 {
		return regions[0].Name
	}

	// Deterministic mid-point selection for simplicity
	target := totalWeight / 2
	cumulative := 0
	for _, rg := range regions {
		if w, ok := weights[rg.Name]; ok {
			cumulative += w
			if cumulative > target {
				return rg.Name
			}
		}
	}

	if len(regions) > 0 {
		return regions[0].Name
	}
	return ""
}

// RouteDeliveryEnhanced makes an enhanced routing decision using GeoRoutingPolicy
func (r *Router) RouteDeliveryEnhanced(ctx context.Context, tenantID, endpointID uuid.UUID, payload []byte) (*GeoRoutingDecision, error) {
	policy, err := r.repo.GetGeoRoutingPolicy(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get routing policy: %w", err)
	}

	// Check endpoint-level config
	epConfig, _ := r.repo.GetEndpointRegionConfig(ctx, endpointID)

	decision := &GeoRoutingDecision{
		EventID: uuid.New(),
	}

	if policy == nil {
		// No policy; use endpoint config or default
		if epConfig != nil && epConfig.PrimaryRegion != "" {
			decision.SelectedRegion = epConfig.PrimaryRegion
			decision.Reason = "endpoint-config"
		} else {
			decision.SelectedRegion = string(RegionUSEast)
			decision.Reason = "default"
		}
		return decision, nil
	}

	selected, err := r.SelectRegion(ctx, policy, "")
	if err != nil {
		return nil, err
	}

	decision.SelectedRegion = selected
	decision.Reason = policy.Strategy

	// Collect alternatives
	regions, _ := r.repo.ListGeoRegions(ctx)
	for _, rg := range regions {
		if rg.Name != selected && (rg.Status == "active" || rg.Status == "degraded") {
			decision.AlternativeRegions = append(decision.AlternativeRegions, rg.Name)
		}
	}

	// Estimate latency
	for _, rg := range regions {
		if rg.Name == selected {
			decision.Latency = rg.AvgLatency
			break
		}
	}

	return decision, nil
}
