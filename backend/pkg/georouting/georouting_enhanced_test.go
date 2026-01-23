package georouting

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock repository for enhanced tests ---

type enhancedMockRepo struct {
	geoRegions       map[string]*GeoRegion
	policies         map[uuid.UUID]*GeoRoutingPolicy
	policiesByID     map[uuid.UUID]*GeoRoutingPolicy
	epConfigs        map[uuid.UUID]*EndpointRegionConfig
	decisions        []GeoRoutingDecision
	routings         map[string]*EndpointRouting
	configs          map[string]*RegionConfig
	health           map[string]*RegionHealth
}

func newEnhancedMockRepo() *enhancedMockRepo {
	return &enhancedMockRepo{
		geoRegions:   make(map[string]*GeoRegion),
		policies:     make(map[uuid.UUID]*GeoRoutingPolicy),
		policiesByID: make(map[uuid.UUID]*GeoRoutingPolicy),
		epConfigs:    make(map[uuid.UUID]*EndpointRegionConfig),
		routings:     make(map[string]*EndpointRouting),
		configs:      make(map[string]*RegionConfig),
		health:       make(map[string]*RegionHealth),
	}
}

func (m *enhancedMockRepo) CreateEndpointRouting(ctx context.Context, routing *EndpointRouting) error {
	m.routings[routing.ID] = routing
	return nil
}
func (m *enhancedMockRepo) GetEndpointRouting(ctx context.Context, tenantID, endpointID string) (*EndpointRouting, error) {
	for _, r := range m.routings {
		if r.TenantID == tenantID && r.EndpointID == endpointID {
			return r, nil
		}
	}
	return nil, nil
}
func (m *enhancedMockRepo) UpdateEndpointRouting(ctx context.Context, routing *EndpointRouting) error {
	m.routings[routing.ID] = routing
	return nil
}
func (m *enhancedMockRepo) DeleteEndpointRouting(ctx context.Context, tenantID, endpointID string) error {
	return nil
}
func (m *enhancedMockRepo) CreateRegionConfig(ctx context.Context, config *RegionConfig) error {
	m.configs[config.ID] = config
	return nil
}
func (m *enhancedMockRepo) GetRegionConfig(ctx context.Context, regionID string) (*RegionConfig, error) {
	return m.configs[regionID], nil
}
func (m *enhancedMockRepo) ListRegionConfigs(ctx context.Context) ([]RegionConfig, error) {
	var result []RegionConfig
	for _, c := range m.configs {
		result = append(result, *c)
	}
	return result, nil
}
func (m *enhancedMockRepo) UpdateRegionConfig(ctx context.Context, config *RegionConfig) error {
	m.configs[config.ID] = config
	return nil
}
func (m *enhancedMockRepo) GetRegionHealth(ctx context.Context, regionID string) (*RegionHealth, error) {
	if h, ok := m.health[regionID]; ok {
		return h, nil
	}
	return &RegionHealth{RegionID: regionID, IsHealthy: true}, nil
}
func (m *enhancedMockRepo) UpdateRegionHealth(ctx context.Context, health *RegionHealth) error {
	m.health[health.RegionID] = health
	return nil
}
func (m *enhancedMockRepo) GetRoutingStats(ctx context.Context, tenantID, period string) (*RoutingStats, error) {
	return &RoutingStats{TenantID: tenantID, Period: period, ByRegion: make(map[Region]int64), ByMode: make(map[RoutingMode]int64)}, nil
}
func (m *enhancedMockRepo) RecordRoutingDecision(ctx context.Context, decision *RoutingDecision) error {
	return nil
}

func (m *enhancedMockRepo) CreateGeoRegion(ctx context.Context, region *GeoRegion) error {
	m.geoRegions[region.Name] = region
	return nil
}
func (m *enhancedMockRepo) GetGeoRegion(ctx context.Context, name string) (*GeoRegion, error) {
	r, ok := m.geoRegions[name]
	if !ok {
		return nil, nil
	}
	return r, nil
}
func (m *enhancedMockRepo) ListGeoRegions(ctx context.Context) ([]GeoRegion, error) {
	var result []GeoRegion
	for _, r := range m.geoRegions {
		result = append(result, *r)
	}
	return result, nil
}
func (m *enhancedMockRepo) UpdateGeoRegion(ctx context.Context, region *GeoRegion) error {
	m.geoRegions[region.Name] = region
	return nil
}
func (m *enhancedMockRepo) CreateGeoRoutingPolicy(ctx context.Context, policy *GeoRoutingPolicy) error {
	m.policies[policy.TenantID] = policy
	m.policiesByID[policy.ID] = policy
	return nil
}
func (m *enhancedMockRepo) GetGeoRoutingPolicy(ctx context.Context, tenantID uuid.UUID) (*GeoRoutingPolicy, error) {
	p, ok := m.policies[tenantID]
	if !ok {
		return nil, nil
	}
	return p, nil
}
func (m *enhancedMockRepo) GetGeoRoutingPolicyByID(ctx context.Context, id uuid.UUID) (*GeoRoutingPolicy, error) {
	p, ok := m.policiesByID[id]
	if !ok {
		return nil, nil
	}
	return p, nil
}
func (m *enhancedMockRepo) UpdateGeoRoutingPolicy(ctx context.Context, policy *GeoRoutingPolicy) error {
	m.policies[policy.TenantID] = policy
	m.policiesByID[policy.ID] = policy
	return nil
}
func (m *enhancedMockRepo) ListGeoRoutingPolicies(ctx context.Context, tenantID uuid.UUID) ([]GeoRoutingPolicy, error) {
	var result []GeoRoutingPolicy
	for _, p := range m.policies {
		if p.TenantID == tenantID {
			result = append(result, *p)
		}
	}
	return result, nil
}
func (m *enhancedMockRepo) GetEndpointRegionConfig(ctx context.Context, endpointID uuid.UUID) (*EndpointRegionConfig, error) {
	c, ok := m.epConfigs[endpointID]
	if !ok {
		return nil, nil
	}
	return c, nil
}
func (m *enhancedMockRepo) SaveEndpointRegionConfig(ctx context.Context, config *EndpointRegionConfig) error {
	m.epConfigs[config.EndpointID] = config
	return nil
}
func (m *enhancedMockRepo) RecordGeoRoutingDecision(ctx context.Context, decision *GeoRoutingDecision) error {
	m.decisions = append(m.decisions, *decision)
	return nil
}
func (m *enhancedMockRepo) ListGeoRoutingDecisions(ctx context.Context, tenantID uuid.UUID, limit int) ([]GeoRoutingDecision, error) {
	if limit <= 0 || limit > len(m.decisions) {
		limit = len(m.decisions)
	}
	return m.decisions[:limit], nil
}

// --- Helper to seed regions ---

func seedRegions(repo *enhancedMockRepo) {
	regions := []GeoRegion{
		{ID: uuid.New(), Name: "us-east-1", DisplayName: "US East", Provider: "aws", Latitude: 39.0438, Longitude: -77.4874, Status: "active", Capacity: 1000, CurrentLoad: 200, AvgLatency: 25, CreatedAt: time.Now()},
		{ID: uuid.New(), Name: "eu-west-1", DisplayName: "EU West", Provider: "aws", Latitude: 53.3498, Longitude: -6.2603, Status: "active", Capacity: 800, CurrentLoad: 300, AvgLatency: 45, CreatedAt: time.Now()},
		{ID: uuid.New(), Name: "ap-southeast-1", DisplayName: "AP Southeast", Provider: "aws", Latitude: 1.3521, Longitude: 103.8198, Status: "active", Capacity: 600, CurrentLoad: 100, AvgLatency: 120, CreatedAt: time.Now()},
		{ID: uuid.New(), Name: "us-west-2", DisplayName: "US West", Provider: "aws", Latitude: 45.8696, Longitude: -119.6880, Status: "active", Capacity: 900, CurrentLoad: 150, AvgLatency: 35, CreatedAt: time.Now()},
		{ID: uuid.New(), Name: "eu-central-1", DisplayName: "EU Central", Provider: "aws", Latitude: 50.1109, Longitude: 8.6821, Status: "offline", Capacity: 700, CurrentLoad: 0, AvgLatency: 50, CreatedAt: time.Now()},
	}
	for i := range regions {
		repo.geoRegions[regions[i].Name] = &regions[i]
	}
}

// --- Tests ---

func TestCalculateDistance_Haversine(t *testing.T) {
	tests := []struct {
		name      string
		lat1, lon1, lat2, lon2 float64
		minDist, maxDist       float64
	}{
		{"NYC to London", 40.7128, -74.0060, 51.5074, -0.1278, 5500, 5600},
		{"Same point", 0, 0, 0, 0, 0, 0.01},
		{"Tokyo to Sydney", 35.6762, 139.6503, -33.8688, 151.2093, 7700, 7900},
		{"North Pole to South Pole", 90, 0, -90, 0, 20000, 20100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := CalculateDistance(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			assert.GreaterOrEqual(t, d, tt.minDist, "distance should be >= minDist")
			assert.LessOrEqual(t, d, tt.maxDist, "distance should be <= maxDist")
		})
	}
}

func TestCalculateDistance_Symmetry(t *testing.T) {
	d1 := CalculateDistance(40.7128, -74.0060, 51.5074, -0.1278)
	d2 := CalculateDistance(51.5074, -0.1278, 40.7128, -74.0060)
	assert.InDelta(t, d1, d2, 0.01, "haversine should be symmetric")
}

func TestGetClosestRegion(t *testing.T) {
	regions := []GeoRegion{
		{Name: "us-east-1", Latitude: 39.0438, Longitude: -77.4874},
		{Name: "eu-west-1", Latitude: 53.3498, Longitude: -6.2603},
		{Name: "ap-southeast-1", Latitude: 1.3521, Longitude: 103.8198},
	}

	tests := []struct {
		name     string
		lat, lon float64
		expected string
	}{
		{"NYC -> us-east", 40.7128, -74.0060, "us-east-1"},
		{"London -> eu-west", 51.5074, -0.1278, "eu-west-1"},
		{"Singapore -> ap-southeast", 1.2900, 103.8500, "ap-southeast-1"},
		{"Tokyo -> ap-southeast", 35.6762, 139.6503, "ap-southeast-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			closest := GetClosestRegion(tt.lat, tt.lon, regions)
			require.NotNil(t, closest)
			assert.Equal(t, tt.expected, closest.Name)
		})
	}
}

func TestGetClosestRegion_Empty(t *testing.T) {
	result := GetClosestRegion(0, 0, nil)
	assert.Nil(t, result)
}

func TestGetLowestLatencyRegion(t *testing.T) {
	regions := []GeoRegion{
		{Name: "us-east-1", AvgLatency: 50, Status: "active"},
		{Name: "eu-west-1", AvgLatency: 25, Status: "active"},
		{Name: "ap-southeast-1", AvgLatency: 120, Status: "active"},
	}

	best := GetLowestLatencyRegion(regions)
	require.NotNil(t, best)
	assert.Equal(t, "eu-west-1", best.Name)
}

func TestGetLowestLatencyRegion_SkipsInactive(t *testing.T) {
	regions := []GeoRegion{
		{Name: "us-east-1", AvgLatency: 50, Status: "active"},
		{Name: "eu-west-1", AvgLatency: 10, Status: "offline"},
		{Name: "ap-southeast-1", AvgLatency: 30, Status: "active"},
	}

	best := GetLowestLatencyRegion(regions)
	require.NotNil(t, best)
	assert.Equal(t, "ap-southeast-1", best.Name)
}

func TestGetLowestLatencyRegion_Empty(t *testing.T) {
	result := GetLowestLatencyRegion(nil)
	assert.Nil(t, result)
}

func TestApplyFailover_PrimaryHealthy(t *testing.T) {
	repo := newEnhancedMockRepo()
	router := NewRouter(repo, nil)

	// Mark primary as healthy
	router.healthTracker.health["us-east-1"] = &RegionHealth{RegionID: "us-east-1", IsHealthy: true}

	region, err := router.ApplyFailover(context.Background(), "us-east-1", []string{"eu-west-1", "ap-southeast-1"})
	require.NoError(t, err)
	assert.Equal(t, "us-east-1", region, "should return primary when healthy")
}

func TestApplyFailover_PrimaryUnhealthy(t *testing.T) {
	repo := newEnhancedMockRepo()
	router := NewRouter(repo, nil)

	router.healthTracker.health["us-east-1"] = &RegionHealth{RegionID: "us-east-1", IsHealthy: false}
	router.healthTracker.health["eu-west-1"] = &RegionHealth{RegionID: "eu-west-1", IsHealthy: true}

	region, err := router.ApplyFailover(context.Background(), "us-east-1", []string{"eu-west-1", "ap-southeast-1"})
	require.NoError(t, err)
	assert.Equal(t, "eu-west-1", region, "should failover to first healthy region")
}

func TestApplyFailover_AllUnhealthy(t *testing.T) {
	repo := newEnhancedMockRepo()
	router := NewRouter(repo, nil)

	router.healthTracker.health["us-east-1"] = &RegionHealth{RegionID: "us-east-1", IsHealthy: false}
	router.healthTracker.health["eu-west-1"] = &RegionHealth{RegionID: "eu-west-1", IsHealthy: false}

	_, err := router.ApplyFailover(context.Background(), "us-east-1", []string{"eu-west-1"})
	assert.Error(t, err, "should return error when all regions unhealthy")
}

func TestApplyFailover_SkipsUnhealthyFallback(t *testing.T) {
	repo := newEnhancedMockRepo()
	router := NewRouter(repo, nil)

	router.healthTracker.health["us-east-1"] = &RegionHealth{RegionID: "us-east-1", IsHealthy: false}
	router.healthTracker.health["eu-west-1"] = &RegionHealth{RegionID: "eu-west-1", IsHealthy: false}
	router.healthTracker.health["ap-southeast-1"] = &RegionHealth{RegionID: "ap-southeast-1", IsHealthy: true}

	region, err := router.ApplyFailover(context.Background(), "us-east-1", []string{"eu-west-1", "ap-southeast-1"})
	require.NoError(t, err)
	assert.Equal(t, "ap-southeast-1", region)
}

func TestEnforceDataResidency_Allowed(t *testing.T) {
	policy := &GeoRoutingPolicy{
		DataResidencyReq: []string{"eu-west-1", "eu-central-1"},
	}
	err := EnforceDataResidency(policy, "eu-west-1")
	assert.NoError(t, err)
}

func TestEnforceDataResidency_Denied(t *testing.T) {
	policy := &GeoRoutingPolicy{
		DataResidencyReq: []string{"eu-west-1", "eu-central-1"},
	}
	err := EnforceDataResidency(policy, "us-east-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "violates data residency")
}

func TestEnforceDataResidency_NoConstraints(t *testing.T) {
	policy := &GeoRoutingPolicy{
		DataResidencyReq: nil,
	}
	err := EnforceDataResidency(policy, "us-east-1")
	assert.NoError(t, err, "no data residency constraints should pass")
}

func TestSelectRegion_LatencyStrategy(t *testing.T) {
	repo := newEnhancedMockRepo()
	seedRegions(repo)
	router := NewRouter(repo, nil)

	policy := &GeoRoutingPolicy{
		Strategy: "latency",
	}

	region, err := router.SelectRegion(context.Background(), policy, "")
	require.NoError(t, err)
	assert.Equal(t, "us-east-1", region, "should select lowest latency active region")
}

func TestSelectRegion_GeoProximityStrategy(t *testing.T) {
	repo := newEnhancedMockRepo()
	seedRegions(repo)
	geoIP := &DefaultGeoIPProvider{}
	router := NewRouter(repo, geoIP)

	policy := &GeoRoutingPolicy{
		Strategy: "geo-proximity",
	}

	// DefaultGeoIPProvider returns US coordinates for any IP
	region, err := router.SelectRegion(context.Background(), policy, "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, "us-east-1", region)
}

func TestSelectRegion_FailoverStrategy(t *testing.T) {
	repo := newEnhancedMockRepo()
	seedRegions(repo)
	router := NewRouter(repo, nil)

	// Mark primary as healthy
	router.healthTracker.health["eu-west-1"] = &RegionHealth{RegionID: "eu-west-1", IsHealthy: true}

	policy := &GeoRoutingPolicy{
		Strategy:      "failover",
		FailoverOrder: []string{"eu-west-1", "us-east-1", "ap-southeast-1"},
	}

	region, err := router.SelectRegion(context.Background(), policy, "")
	require.NoError(t, err)
	assert.Equal(t, "eu-west-1", region)
}

func TestSelectRegion_FailoverFallback(t *testing.T) {
	repo := newEnhancedMockRepo()
	seedRegions(repo)
	router := NewRouter(repo, nil)

	// Primary unhealthy, second healthy
	router.healthTracker.health["eu-west-1"] = &RegionHealth{RegionID: "eu-west-1", IsHealthy: false}
	router.healthTracker.health["us-east-1"] = &RegionHealth{RegionID: "us-east-1", IsHealthy: true}

	policy := &GeoRoutingPolicy{
		Strategy:      "failover",
		FailoverOrder: []string{"eu-west-1", "us-east-1"},
	}

	region, err := router.SelectRegion(context.Background(), policy, "")
	require.NoError(t, err)
	assert.Equal(t, "us-east-1", region)
}

func TestSelectRegion_WeightedStrategy(t *testing.T) {
	repo := newEnhancedMockRepo()
	seedRegions(repo)
	router := NewRouter(repo, nil)

	policy := &GeoRoutingPolicy{
		Strategy: "weighted",
		Weights: map[string]int{
			"us-east-1": 70,
			"eu-west-1": 20,
			"ap-southeast-1": 10,
		},
	}

	region, err := router.SelectRegion(context.Background(), policy, "")
	require.NoError(t, err)
	assert.NotEmpty(t, region, "weighted routing should select a region")
}

func TestSelectRegion_DataResidencyOverride(t *testing.T) {
	repo := newEnhancedMockRepo()
	seedRegions(repo)
	router := NewRouter(repo, nil)

	// Latency would pick us-east-1, but data residency requires EU
	policy := &GeoRoutingPolicy{
		Strategy:         "latency",
		DataResidencyReq: []string{"eu-west-1"},
	}

	region, err := router.SelectRegion(context.Background(), policy, "")
	require.NoError(t, err)
	assert.Equal(t, "eu-west-1", region, "data residency should override latency selection")
}

func TestSelectRegion_NoActiveRegions(t *testing.T) {
	repo := newEnhancedMockRepo()
	// All regions offline
	repo.geoRegions["region-1"] = &GeoRegion{Name: "region-1", Status: "offline"}

	router := NewRouter(repo, nil)
	policy := &GeoRoutingPolicy{Strategy: "latency"}

	_, err := router.SelectRegion(context.Background(), policy, "")
	assert.Error(t, err)
}

func TestWeightedRoutingDistribution(t *testing.T) {
	regions := []GeoRegion{
		{Name: "us-east-1", Status: "active", AvgLatency: 30},
		{Name: "eu-west-1", Status: "active", AvgLatency: 50},
		{Name: "ap-southeast-1", Status: "active", AvgLatency: 100},
	}
	weights := map[string]int{
		"us-east-1":      60,
		"eu-west-1":      30,
		"ap-southeast-1": 10,
	}

	selected := selectWeightedRegion(regions, weights)
	assert.NotEmpty(t, selected)

	// Verify the selected region is one of the known regions
	valid := false
	for _, r := range regions {
		if r.Name == selected {
			valid = true
			break
		}
	}
	assert.True(t, valid, "selected region should be one of the provided regions")
}

func TestWeightedRoutingDistribution_EmptyWeights(t *testing.T) {
	regions := []GeoRegion{
		{Name: "us-east-1", Status: "active"},
	}
	selected := selectWeightedRegion(regions, nil)
	assert.Equal(t, "us-east-1", selected, "should return first region with empty weights")
}

func TestRouteDeliveryEnhanced_NoPolicy(t *testing.T) {
	repo := newEnhancedMockRepo()
	router := NewRouter(repo, nil)

	decision, err := router.RouteDeliveryEnhanced(context.Background(), uuid.New(), uuid.New(), nil)
	require.NoError(t, err)
	assert.Equal(t, string(RegionUSEast), decision.SelectedRegion)
	assert.Equal(t, "default", decision.Reason)
}

func TestRouteDeliveryEnhanced_WithPolicy(t *testing.T) {
	repo := newEnhancedMockRepo()
	seedRegions(repo)

	tenantID := uuid.New()
	repo.policies[tenantID] = &GeoRoutingPolicy{
		ID:       uuid.New(),
		TenantID: tenantID,
		Strategy: "latency",
		Active:   true,
	}

	router := NewRouter(repo, nil)
	decision, err := router.RouteDeliveryEnhanced(context.Background(), tenantID, uuid.New(), nil)
	require.NoError(t, err)
	assert.NotEmpty(t, decision.SelectedRegion)
	assert.Equal(t, "latency", decision.Reason)
}

func TestServiceCreateGeoRegion(t *testing.T) {
	repo := newEnhancedMockRepo()
	svc := NewService(repo, nil)

	region := &GeoRegion{
		Name:        "us-east-1",
		DisplayName: "US East",
		Provider:    "aws",
		Latitude:    39.0438,
		Longitude:   -77.4874,
		Capacity:    1000,
	}

	err := svc.CreateGeoRegion(context.Background(), region)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, region.ID)
	assert.Equal(t, "active", region.Status)
	assert.False(t, region.CreatedAt.IsZero())
}

func TestServiceSimulateRouting(t *testing.T) {
	repo := newEnhancedMockRepo()
	seedRegions(repo)

	tenantID := uuid.New()
	repo.policies[tenantID] = &GeoRoutingPolicy{
		ID:       uuid.New(),
		TenantID: tenantID,
		Strategy: "latency",
		Active:   true,
	}

	svc := NewService(repo, &DefaultGeoIPProvider{})
	decision, err := svc.SimulateRouting(context.Background(), tenantID, "127.0.0.1")
	require.NoError(t, err)
	assert.Contains(t, decision.Reason, "simulated")
	assert.NotEmpty(t, decision.SelectedRegion)
}

func TestServiceSimulateRouting_NoPolicy(t *testing.T) {
	repo := newEnhancedMockRepo()
	svc := NewService(repo, nil)

	decision, err := svc.SimulateRouting(context.Background(), uuid.New(), "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, string(RegionUSEast), decision.SelectedRegion)
	assert.Contains(t, decision.Reason, "default")
}

func TestServiceConfigureEndpointRegion(t *testing.T) {
	repo := newEnhancedMockRepo()
	svc := NewService(repo, nil)

	epID := uuid.New()
	config := &EndpointRegionConfig{
		PrimaryRegion:   "eu-west-1",
		FailoverRegions: []string{"eu-central-1", "us-east-1"},
		DataResidencyRq: "eu",
	}

	err := svc.ConfigureEndpointRegion(context.Background(), epID, config)
	require.NoError(t, err)
	assert.Equal(t, epID, config.EndpointID)

	// Verify it was saved
	saved, err := repo.GetEndpointRegionConfig(context.Background(), epID)
	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.Equal(t, "eu-west-1", saved.PrimaryRegion)
}

func TestServiceRouteEvent(t *testing.T) {
	repo := newEnhancedMockRepo()
	seedRegions(repo)

	tenantID := uuid.New()
	repo.policies[tenantID] = &GeoRoutingPolicy{
		ID:       uuid.New(),
		TenantID: tenantID,
		Strategy: "latency",
		Active:   true,
	}

	svc := NewService(repo, nil)
	decision, err := svc.RouteEvent(context.Background(), tenantID, uuid.New(), []byte(`{"test":true}`))
	require.NoError(t, err)
	assert.NotEmpty(t, decision.SelectedRegion)
	assert.NotEqual(t, uuid.Nil, decision.EventID)

	// Decision should be recorded
	assert.Len(t, repo.decisions, 1)
}

func TestHealthTracker_CheckRegionHealth(t *testing.T) {
	repo := newEnhancedMockRepo()
	tracker := NewHealthTracker(repo)

	tracker.health["us-east-1"] = &RegionHealth{
		RegionID:    "us-east-1",
		IsHealthy:   true,
		AvgLatencyMs: 30,
		LastCheck:    time.Now(),
		ErrorRate:    2.0,
	}

	health, err := tracker.CheckRegionHealth(context.Background(), "us-east-1")
	require.NoError(t, err)
	assert.Equal(t, "active", health.Status)
	assert.Equal(t, 30, health.AvgLatency)
	assert.InDelta(t, 98.0, health.SuccessRate, 0.1)
}

func TestHealthTracker_CheckRegionHealth_Unknown(t *testing.T) {
	repo := newEnhancedMockRepo()
	tracker := NewHealthTracker(repo)

	health, err := tracker.CheckRegionHealth(context.Background(), "nonexistent")
	require.NoError(t, err)
	// The repo returns IsHealthy=true for unknown regions, so status is "active"
	assert.Equal(t, "active", health.Status)
}

func TestHealthTracker_UpdateRegionMetrics(t *testing.T) {
	repo := newEnhancedMockRepo()
	tracker := NewHealthTracker(repo)

	// Success
	err := tracker.UpdateRegionMetrics(context.Background(), "us-east-1", 30, true)
	require.NoError(t, err)

	rh := tracker.health["us-east-1"]
	require.NotNil(t, rh)
	assert.Equal(t, 1, rh.ConsecutiveOK)
	assert.True(t, rh.IsHealthy)

	// Failures
	for i := 0; i < 3; i++ {
		tracker.UpdateRegionMetrics(context.Background(), "us-east-1", 500, false)
	}
	rh = tracker.health["us-east-1"]
	assert.False(t, rh.IsHealthy)
	assert.Equal(t, 3, rh.ConsecutiveFail)
}

func TestHealthTracker_DetectDegradedRegions(t *testing.T) {
	repo := newEnhancedMockRepo()
	tracker := NewHealthTracker(repo)

	tracker.health["us-east-1"] = &RegionHealth{RegionID: "us-east-1", IsHealthy: true, ConsecutiveFail: 0}
	tracker.health["eu-west-1"] = &RegionHealth{RegionID: "eu-west-1", IsHealthy: true, ConsecutiveFail: 1}
	tracker.health["ap-southeast-1"] = &RegionHealth{RegionID: "ap-southeast-1", IsHealthy: false, ConsecutiveFail: 5}

	degraded, err := tracker.DetectDegradedRegions(context.Background())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(degraded), 2)

	// Check that the unhealthy one is marked offline
	for _, d := range degraded {
		if d.RegionName == "ap-southeast-1" {
			assert.Equal(t, "offline", d.Status)
		}
		if d.RegionName == "eu-west-1" {
			assert.Equal(t, "degraded", d.Status)
		}
	}
}

func TestServiceGetGeoDashboard(t *testing.T) {
	repo := newEnhancedMockRepo()
	seedRegions(repo)
	svc := NewService(repo, nil)

	dashboard, err := svc.GetGeoDashboard(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, dashboard.Regions)
	assert.NotNil(t, dashboard.LoadDistribution)
	assert.NotNil(t, dashboard.LatencyMap)
}

func TestHaversineConsistency(t *testing.T) {
	// Verify CalculateDistance matches haversine
	d1 := CalculateDistance(40.7128, -74.0060, 51.5074, -0.1278)
	d2 := haversine(40.7128, -74.0060, 51.5074, -0.1278)
	assert.Equal(t, d1, d2)
}

func TestRoutingPolicyEvaluation_AllStrategies(t *testing.T) {
	strategies := []string{"latency", "geo-proximity", "failover", "weighted", "round-robin"}

	for _, strategy := range strategies {
		t.Run(strategy, func(t *testing.T) {
			repo := newEnhancedMockRepo()
			seedRegions(repo)
			router := NewRouter(repo, &DefaultGeoIPProvider{})

			// Mark regions healthy
			for name := range repo.geoRegions {
				router.healthTracker.health[name] = &RegionHealth{RegionID: name, IsHealthy: true}
			}

			policy := &GeoRoutingPolicy{
				Strategy:         strategy,
				PreferredRegions: []string{"us-east-1"},
				FailoverOrder:    []string{"us-east-1", "eu-west-1", "ap-southeast-1"},
				Weights: map[string]int{
					"us-east-1":      50,
					"eu-west-1":      30,
					"ap-southeast-1": 20,
				},
			}

			region, err := router.SelectRegion(context.Background(), policy, "127.0.0.1")
			require.NoError(t, err)
			assert.NotEmpty(t, region, "strategy %s should select a region", strategy)
		})
	}
}

func TestGetClosestRegion_Accuracy(t *testing.T) {
	// Verify that proximity result is actually the closest using raw distances
	regions := []GeoRegion{
		{Name: "us-east-1", Latitude: 39.0438, Longitude: -77.4874},
		{Name: "eu-west-1", Latitude: 53.3498, Longitude: -6.2603},
		{Name: "ap-southeast-1", Latitude: 1.3521, Longitude: 103.8198},
	}

	// Point close to Dublin
	lat, lon := 53.0, -6.0
	closest := GetClosestRegion(lat, lon, regions)
	require.NotNil(t, closest)
	assert.Equal(t, "eu-west-1", closest.Name)

	// Verify this is actually the minimum
	minDist := math.MaxFloat64
	for _, r := range regions {
		d := CalculateDistance(lat, lon, r.Latitude, r.Longitude)
		if d < minDist {
			minDist = d
		}
	}
	closestDist := CalculateDistance(lat, lon, closest.Latitude, closest.Longitude)
	assert.InDelta(t, minDist, closestDist, 0.01)
}
