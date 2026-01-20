package georouting

import (
	"context"
	"fmt"
	"math"
	"net"
	"sync"
	"time"
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
