package multiregion

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"
)

// HealthChecker performs health checks on regions
type HealthChecker struct {
	client     *http.Client
	repo       Repository
	mu         sync.RWMutex
	healthCache map[string]*RegionHealth
	config     HealthConfig
}

// HealthConfig configures health checking
type HealthConfig struct {
	CheckInterval      time.Duration
	Timeout            time.Duration
	HealthyThreshold   int     // Consecutive successes to become healthy
	UnhealthyThreshold int     // Consecutive failures to become unhealthy
	LatencyThresholdMs int64   // Latency above this is degraded
	ErrorRateThreshold float64 // Error rate above this is degraded
}

// DefaultHealthConfig returns default health check configuration
func DefaultHealthConfig() HealthConfig {
	return HealthConfig{
		CheckInterval:      30 * time.Second,
		Timeout:            5 * time.Second,
		HealthyThreshold:   3,
		UnhealthyThreshold: 3,
		LatencyThresholdMs: 1000,
		ErrorRateThreshold: 5.0,
	}
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(repo Repository, config HealthConfig) *HealthChecker {
	return &HealthChecker{
		client: &http.Client{
			Timeout: config.Timeout,
		},
		repo:        repo,
		healthCache: make(map[string]*RegionHealth),
		config:      config,
	}
}

// Start starts the health checker
func (h *HealthChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(h.config.CheckInterval)
	defer ticker.Stop()

	// Initial check
	h.checkAllRegions(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.checkAllRegions(ctx)
		}
	}
}

func (h *HealthChecker) checkAllRegions(ctx context.Context) {
	regions, err := h.repo.ListActiveRegions(ctx)
	if err != nil {
		return
	}

	var wg sync.WaitGroup
	for _, region := range regions {
		wg.Add(1)
		go func(r *Region) {
			defer wg.Done()
			h.checkRegion(ctx, r)
		}(region)
	}
	wg.Wait()
}

func (h *HealthChecker) checkRegion(ctx context.Context, region *Region) {
	start := time.Now()
	
	healthEndpoint := fmt.Sprintf("%s/health", region.Endpoint)
	req, _ := http.NewRequestWithContext(ctx, "GET", healthEndpoint, nil)
	
	resp, err := h.client.Do(req)
	latency := time.Since(start)
	
	health := &RegionHealth{
		RegionID:  region.ID,
		LastCheck: time.Now(),
		Latency:   latency,
	}

	if err != nil {
		health.Status = HealthStatusUnhealthy
		health.ErrorRate = 100.0
	} else {
		resp.Body.Close()
		
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			health.Status = h.determineStatus(latency, 0)
			health.SuccessRate = 100.0
		} else {
			health.Status = HealthStatusDegraded
			health.ErrorRate = 50.0
			health.SuccessRate = 50.0
		}
	}

	// Update cache
	h.mu.Lock()
	h.healthCache[region.ID] = health
	h.mu.Unlock()

	// Persist to database
	h.repo.RecordRegionHealth(ctx, health)
}

func (h *HealthChecker) determineStatus(latency time.Duration, errorRate float64) HealthStatus {
	if errorRate > h.config.ErrorRateThreshold {
		return HealthStatusUnhealthy
	}
	if latency.Milliseconds() > h.config.LatencyThresholdMs {
		return HealthStatusDegraded
	}
	if errorRate > h.config.ErrorRateThreshold/2 {
		return HealthStatusDegraded
	}
	return HealthStatusHealthy
}

// GetHealth returns cached health for a region
func (h *HealthChecker) GetHealth(regionID string) *RegionHealth {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.healthCache[regionID]
}

// GetAllHealth returns all cached health data
func (h *HealthChecker) GetAllHealth() map[string]*RegionHealth {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	result := make(map[string]*RegionHealth)
	for k, v := range h.healthCache {
		result[k] = v
	}
	return result
}

// Router handles request routing between regions
type Router struct {
	repo          Repository
	healthChecker *HealthChecker
	mu            sync.RWMutex
	policyCache   map[string]*RoutingPolicy // tenantID -> policy
	regionsCache  []*Region
	roundRobinIdx map[string]int // tenantID -> current index
}

// NewRouter creates a new multi-region router
func NewRouter(repo Repository, healthChecker *HealthChecker) *Router {
	return &Router{
		repo:          repo,
		healthChecker: healthChecker,
		policyCache:   make(map[string]*RoutingPolicy),
		roundRobinIdx: make(map[string]int),
	}
}

// RefreshCache refreshes the router cache
func (r *Router) RefreshCache(ctx context.Context) error {
	regions, err := r.repo.ListActiveRegions(ctx)
	if err != nil {
		return err
	}

	r.mu.Lock()
	r.regionsCache = regions
	r.mu.Unlock()

	return nil
}

// RouteRequest determines which region should handle a request
func (r *Router) RouteRequest(ctx context.Context, tenantID string, clientIP string) (*Region, error) {
	// Get or load routing policy
	policy, err := r.getPolicy(ctx, tenantID)
	if err != nil {
		// Default to primary region
		return r.getPrimaryRegion(ctx)
	}

	switch policy.PolicyType {
	case RoutingTypePrimaryBackup:
		return r.routePrimaryBackup(ctx, policy)
	case RoutingTypeGeoProximity:
		return r.routeGeoProximity(ctx, policy, clientIP)
	case RoutingTypeWeighted:
		return r.routeWeighted(ctx, policy)
	case RoutingTypeRoundRobin:
		return r.routeRoundRobin(ctx, tenantID, policy)
	case RoutingTypeLatencyBased:
		return r.routeLatencyBased(ctx, policy)
	default:
		return r.getPrimaryRegion(ctx)
	}
}

func (r *Router) getPolicy(ctx context.Context, tenantID string) (*RoutingPolicy, error) {
	r.mu.RLock()
	policy, ok := r.policyCache[tenantID]
	r.mu.RUnlock()

	if ok {
		return policy, nil
	}

	policy, err := r.repo.GetRoutingPolicy(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	r.policyCache[tenantID] = policy
	r.mu.Unlock()

	return policy, nil
}

func (r *Router) getPrimaryRegion(ctx context.Context) (*Region, error) {
	r.mu.RLock()
	regions := r.regionsCache
	r.mu.RUnlock()

	for _, region := range regions {
		if region.IsPrimary {
			health := r.healthChecker.GetHealth(region.ID)
			if health == nil || health.Status != HealthStatusUnhealthy {
				return region, nil
			}
		}
	}

	// Find any healthy region
	for _, region := range regions {
		health := r.healthChecker.GetHealth(region.ID)
		if health == nil || health.Status != HealthStatusUnhealthy {
			return region, nil
		}
	}

	return nil, fmt.Errorf("no healthy regions available")
}

func (r *Router) routePrimaryBackup(ctx context.Context, policy *RoutingPolicy) (*Region, error) {
	// Try primary
	primary, err := r.repo.GetRegionByCode(ctx, policy.PrimaryRegion)
	if err == nil {
		health := r.healthChecker.GetHealth(primary.ID)
		if health == nil || health.Status != HealthStatusUnhealthy {
			return primary, nil
		}
	}

	// Try fallbacks in order
	for _, fallbackCode := range policy.FallbackRegions {
		region, err := r.repo.GetRegionByCode(ctx, fallbackCode)
		if err != nil {
			continue
		}
		health := r.healthChecker.GetHealth(region.ID)
		if health == nil || health.Status != HealthStatusUnhealthy {
			return region, nil
		}
	}

	return nil, fmt.Errorf("no healthy regions in policy")
}

func (r *Router) routeGeoProximity(ctx context.Context, policy *RoutingPolicy, clientIP string) (*Region, error) {
	// For geo-routing, we'd need IP geolocation
	// Simplified: check geo rules for country match
	// In production, use MaxMind or similar

	// Default to primary if no geo match
	return r.routePrimaryBackup(ctx, policy)
}

func (r *Router) routeWeighted(ctx context.Context, policy *RoutingPolicy) (*Region, error) {
	if len(policy.Weights) == 0 {
		return r.getPrimaryRegion(ctx)
	}

	// Calculate total weight of healthy regions
	totalWeight := 0
	healthyRegions := make(map[string]*Region)
	
	for regionCode, weight := range policy.Weights {
		region, err := r.repo.GetRegionByCode(ctx, regionCode)
		if err != nil {
			continue
		}
		health := r.healthChecker.GetHealth(region.ID)
		if health == nil || health.Status != HealthStatusUnhealthy {
			totalWeight += weight
			healthyRegions[regionCode] = region
		}
	}

	if totalWeight == 0 {
		return nil, fmt.Errorf("no healthy regions with weight")
	}

	// Simple deterministic selection (in production, use random)
	currentWeight := 0
	target := totalWeight / 2 // Mid-point selection
	
	for regionCode, weight := range policy.Weights {
		if region, ok := healthyRegions[regionCode]; ok {
			currentWeight += weight
			if currentWeight >= target {
				return region, nil
			}
		}
	}

	// Return first healthy
	for _, region := range healthyRegions {
		return region, nil
	}

	return nil, fmt.Errorf("no regions available")
}

func (r *Router) routeRoundRobin(ctx context.Context, tenantID string, policy *RoutingPolicy) (*Region, error) {
	regions := append([]string{policy.PrimaryRegion}, policy.FallbackRegions...)
	
	// Get healthy regions
	var healthyRegions []*Region
	for _, code := range regions {
		region, err := r.repo.GetRegionByCode(ctx, code)
		if err != nil {
			continue
		}
		health := r.healthChecker.GetHealth(region.ID)
		if health == nil || health.Status != HealthStatusUnhealthy {
			healthyRegions = append(healthyRegions, region)
		}
	}

	if len(healthyRegions) == 0 {
		return nil, fmt.Errorf("no healthy regions")
	}

	// Get and increment index
	r.mu.Lock()
	idx := r.roundRobinIdx[tenantID]
	r.roundRobinIdx[tenantID] = (idx + 1) % len(healthyRegions)
	r.mu.Unlock()

	return healthyRegions[idx%len(healthyRegions)], nil
}

func (r *Router) routeLatencyBased(ctx context.Context, policy *RoutingPolicy) (*Region, error) {
	regions := append([]string{policy.PrimaryRegion}, policy.FallbackRegions...)
	
	var bestRegion *Region
	var bestLatency time.Duration = time.Hour

	for _, code := range regions {
		region, err := r.repo.GetRegionByCode(ctx, code)
		if err != nil {
			continue
		}
		health := r.healthChecker.GetHealth(region.ID)
		if health == nil {
			continue
		}
		if health.Status != HealthStatusUnhealthy && health.Latency < bestLatency {
			bestLatency = health.Latency
			bestRegion = region
		}
	}

	if bestRegion == nil {
		return r.getPrimaryRegion(ctx)
	}

	return bestRegion, nil
}

// FailoverManager handles automatic and manual failovers
type FailoverManager struct {
	repo          Repository
	healthChecker *HealthChecker
	router        *Router
	mu            sync.Mutex
	config        FailoverConfig
}

// FailoverConfig configures failover behavior
type FailoverConfig struct {
	AutoFailoverEnabled bool
	FailoverThreshold   int           // Consecutive unhealthy checks before failover
	CooldownPeriod      time.Duration // Minimum time between failovers
	DrainTimeout        time.Duration // Time to drain connections before switch
}

// DefaultFailoverConfig returns default failover configuration
func DefaultFailoverConfig() FailoverConfig {
	return FailoverConfig{
		AutoFailoverEnabled: true,
		FailoverThreshold:   3,
		CooldownPeriod:      5 * time.Minute,
		DrainTimeout:        30 * time.Second,
	}
}

// NewFailoverManager creates a new failover manager
func NewFailoverManager(repo Repository, healthChecker *HealthChecker, router *Router, config FailoverConfig) *FailoverManager {
	return &FailoverManager{
		repo:          repo,
		healthChecker: healthChecker,
		router:        router,
		config:        config,
	}
}

// TriggerFailover manually triggers a failover
func (f *FailoverManager) TriggerFailover(ctx context.Context, fromRegion, toRegion string, reason string) (*FailoverEvent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Create failover event
	event := &FailoverEvent{
		ID:          generateID(),
		FromRegion:  fromRegion,
		ToRegion:    toRegion,
		Reason:      FailoverReasonManual,
		TriggerType: TriggerTypeManual,
		Status:      FailoverStatusInProgress,
		StartedAt:   time.Now(),
		Details:     reason,
	}

	if err := f.repo.CreateFailoverEvent(ctx, event); err != nil {
		return nil, err
	}

	// Execute failover
	if err := f.executeFailover(ctx, event); err != nil {
		event.Status = FailoverStatusFailed
		event.Details = fmt.Sprintf("%s - Error: %v", event.Details, err)
		f.repo.UpdateFailoverEvent(ctx, event)
		return event, err
	}

	now := time.Now()
	event.Status = FailoverStatusCompleted
	event.CompletedAt = &now
	event.Duration = now.Sub(event.StartedAt)
	f.repo.UpdateFailoverEvent(ctx, event)

	return event, nil
}

func (f *FailoverManager) executeFailover(ctx context.Context, event *FailoverEvent) error {
	// 1. Mark source region as inactive
	fromRegion, err := f.repo.GetRegionByCode(ctx, event.FromRegion)
	if err != nil {
		return fmt.Errorf("failed to get source region: %w", err)
	}

	fromRegion.IsActive = false
	fromRegion.IsPrimary = false
	if err := f.repo.UpdateRegion(ctx, fromRegion); err != nil {
		return fmt.Errorf("failed to deactivate source region: %w", err)
	}

	// 2. Wait for drain (in production, monitor active connections)
	time.Sleep(f.config.DrainTimeout)

	// 3. Promote target region
	toRegion, err := f.repo.GetRegionByCode(ctx, event.ToRegion)
	if err != nil {
		return fmt.Errorf("failed to get target region: %w", err)
	}

	toRegion.IsPrimary = true
	if err := f.repo.UpdateRegion(ctx, toRegion); err != nil {
		return fmt.Errorf("failed to promote target region: %w", err)
	}

	// 4. Refresh router cache
	f.router.RefreshCache(ctx)

	return nil
}

// generateID generates a unique ID
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// Haversine calculates distance between two coordinates in km
func Haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371 // km

	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}
