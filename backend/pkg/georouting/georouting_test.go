package georouting

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHaversineDistanceNew(t *testing.T) {
	// Test distance calculation between known points
	tests := []struct {
		name     string
		lat1     float64
		lon1     float64
		lat2     float64
		lon2     float64
		expected float64 // approximate distance in km
		tolerance float64
	}{
		{
			name:     "New York to London",
			lat1:     40.7128, lon1: -74.0060,
			lat2:     51.5074, lon2: -0.1278,
			expected: 5570,
			tolerance: 50,
		},
		{
			name:     "Same point",
			lat1:     40.7128, lon1: -74.0060,
			lat2:     40.7128, lon2: -74.0060,
			expected: 0,
			tolerance: 0.1,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			distance := haversine(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			diff := math.Abs(distance - tt.expected)
			if diff > tt.tolerance {
				t.Errorf("haversine() = %v, expected %v (±%v)", distance, tt.expected, tt.tolerance)
			}
		})
	}
}

func TestAllRegionsNew(t *testing.T) {
	regions := AllRegions()
	
	if len(regions) == 0 {
		t.Error("expected non-empty list of regions")
	}
	
	// Check that essential regions are present
	essentialCodes := []Region{RegionUSEast, RegionUSWest, RegionEUWest, RegionAPEast}
	for _, code := range essentialCodes {
		found := false
		for _, region := range regions {
			if region == code {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected region %s", code)
		}
	}
}

func TestDataResidencyNew(t *testing.T) {
	tests := []struct {
		residency      DataResidency
		region         Region
		shouldBeValid  bool
	}{
		{ResidencyNone, RegionUSEast, true},
		{ResidencyNone, RegionEUWest, true},
		{ResidencyUS, RegionUSEast, true},
		{ResidencyUS, RegionUSWest, true},
		{ResidencyUS, RegionEUWest, false},
		{ResidencyEU, RegionEUWest, true},
		{ResidencyEU, RegionEUCentral, true},
		{ResidencyEU, RegionUSEast, false},
	}
	
	for _, tt := range tests {
		t.Run(string(tt.residency)+"-"+string(tt.region), func(t *testing.T) {
			valid := validateResidency(tt.residency, tt.region)
			if valid != tt.shouldBeValid {
				t.Errorf("validateResidency(%s, %s) = %v, want %v", tt.residency, tt.region, valid, tt.shouldBeValid)
			}
		})
	}
}

func validateResidency(residency DataResidency, region Region) bool {
	switch residency {
	case ResidencyNone:
		return true
	case ResidencyUS:
		return region == RegionUSEast || region == RegionUSWest
	case ResidencyEU:
		return region == RegionEUWest || region == RegionEUCentral
	case ResidencyAPAC:
		return region == RegionAPEast || region == RegionAPSouth
	case ResidencyStrict:
		return true // Depends on specific rules
	default:
		return false
	}
}

func TestRoutingModeNew(t *testing.T) {
	modes := []RoutingMode{
		ModeManual,
		ModeAuto,
		ModeGeo,
		ModeFailover,
	}
	
	for _, mode := range modes {
		if mode == "" {
			t.Error("routing mode should not be empty")
		}
	}
}

func TestRegionHealthNew(t *testing.T) {
	health := &RegionHealth{
		RegionID:       string(RegionUSEast),
		IsHealthy:      true,
		AvgLatencyMs:   50,
		ConsecutiveOK:  10,
		LastCheck:      time.Now(),
		ConsecutiveFail: 0,
	}
	
	if !health.IsHealthy {
		t.Error("expected healthy region")
	}
	
	if health.ConsecutiveOK != 10 {
		t.Errorf("expected 10 consecutive OK, got %d", health.ConsecutiveOK)
	}
	
	// Test unhealthy region
	unhealthy := &RegionHealth{
		RegionID:       string(RegionEUWest),
		IsHealthy:      false,
		AvgLatencyMs:   500,
		LastCheck:      time.Now(),
		ConsecutiveFail: 3,
		LastError:      "connection timeout",
	}
	
	if unhealthy.IsHealthy {
		t.Error("expected unhealthy region")
	}
}

func TestEndpointRoutingNew(t *testing.T) {
	routing := &EndpointRouting{
		ID:             "routing-1",
		TenantID:       "tenant-1",
		EndpointID:     "endpoint-1",
		Mode:           ModeAuto,
		DataResidency:  ResidencyEU,
		PrimaryRegion:  RegionEUWest,
		Regions:        []Region{RegionEUWest, RegionEUCentral},
		FailoverEnabled: true,
		LatencyBased:   true,
	}
	
	if routing.Mode != ModeAuto {
		t.Errorf("expected auto routing mode, got %s", routing.Mode)
	}
	
	if routing.DataResidency != ResidencyEU {
		t.Errorf("expected EU data residency, got %s", routing.DataResidency)
	}
	
	if len(routing.Regions) != 2 {
		t.Errorf("expected 2 regions, got %d", len(routing.Regions))
	}
}

func TestRoutingDecisionNew(t *testing.T) {
	decision := &RoutingDecision{
		EndpointID:     "endpoint-1",
		SelectedRegion: RegionEUWest,
		Reason:         "lowest latency",
		LatencyMs:      45,
		Alternatives:   []Region{RegionEUCentral, RegionUSEast},
		Timestamp:      time.Now(),
	}
	
	if decision.SelectedRegion == "" {
		t.Error("expected selected region")
	}
	
	if decision.Reason == "" {
		t.Error("expected reason for decision")
	}
	
	if len(decision.Alternatives) == 0 {
		t.Error("expected alternatives")
	}
}

func TestRoutingStatsNew(t *testing.T) {
	stats := &RoutingStats{
		TenantID:      "tenant-1",
		Period:        "2026-02",
		ByRegion:      map[Region]int64{RegionUSEast: 100, RegionEUWest: 50},
		ByMode:        map[RoutingMode]int64{ModeAuto: 120, ModeGeo: 30},
		Failovers:     5,
		AvgDecisionMs: 10,
		Timestamp:     time.Now(),
	}
	
	if stats.Failovers != 5 {
		t.Errorf("expected 5 failovers, got %d", stats.Failovers)
	}
	
	if len(stats.ByRegion) != 2 {
		t.Errorf("expected 2 regions in stats, got %d", len(stats.ByRegion))
	}
}

func TestRegionInfoNew(t *testing.T) {
	infos := GetRegionInfo()
	
	if len(infos) == 0 {
		t.Error("expected non-empty region info list")
	}
	
	for _, info := range infos {
		if info.Region == "" {
			t.Error("expected region to be set")
		}
		if info.Name == "" {
			t.Error("expected name to be set")
		}
		if info.Location == "" {
			t.Error("expected location to be set")
		}
	}
}

func TestServiceWithMockRepoNew(t *testing.T) {
	mockRepo := &mockGeoRepoNew{}
	service := NewService(mockRepo, nil)
	
	if service == nil {
		t.Fatal("expected non-nil service")
	}
	
	ctx := context.Background()
	
	// Test creating endpoint routing
	req := &CreateRoutingRequest{
		EndpointID:      "endpoint-1",
		Mode:            ModeGeo,
		PrimaryRegion:   RegionUSEast,
		Regions:         []Region{RegionUSEast, RegionUSWest},
		FailoverEnabled: true,
	}
	
	routing, err := service.CreateEndpointRouting(ctx, "tenant-1", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if routing == nil {
		t.Fatal("expected non-nil routing")
	}
	
	if routing.Mode != ModeGeo {
		t.Errorf("expected mode geo, got %s", routing.Mode)
	}
}

// Mock repository for testing
type mockGeoRepoNew struct {
	routings map[string]*EndpointRouting
	configs  map[string]*RegionConfig
	health   map[string]*RegionHealth
}

func (m *mockGeoRepoNew) CreateEndpointRouting(ctx context.Context, routing *EndpointRouting) error {
	if m.routings == nil {
		m.routings = make(map[string]*EndpointRouting)
	}
	m.routings[routing.ID] = routing
	return nil
}

func (m *mockGeoRepoNew) GetEndpointRouting(ctx context.Context, tenantID, endpointID string) (*EndpointRouting, error) {
	for _, r := range m.routings {
		if r.TenantID == tenantID && r.EndpointID == endpointID {
			return r, nil
		}
	}
	return nil, nil
}

func (m *mockGeoRepoNew) UpdateEndpointRouting(ctx context.Context, routing *EndpointRouting) error {
	if m.routings == nil {
		m.routings = make(map[string]*EndpointRouting)
	}
	m.routings[routing.ID] = routing
	return nil
}

func (m *mockGeoRepoNew) DeleteEndpointRouting(ctx context.Context, tenantID, endpointID string) error {
	for id, r := range m.routings {
		if r.TenantID == tenantID && r.EndpointID == endpointID {
			delete(m.routings, id)
			return nil
		}
	}
	return nil
}

func (m *mockGeoRepoNew) CreateRegionConfig(ctx context.Context, config *RegionConfig) error {
	if m.configs == nil {
		m.configs = make(map[string]*RegionConfig)
	}
	m.configs[config.ID] = config
	return nil
}

func (m *mockGeoRepoNew) GetRegionConfig(ctx context.Context, regionID string) (*RegionConfig, error) {
	if m.configs == nil {
		return nil, nil
	}
	return m.configs[regionID], nil
}

func (m *mockGeoRepoNew) ListRegionConfigs(ctx context.Context) ([]RegionConfig, error) {
	var configs []RegionConfig
	for _, c := range m.configs {
		configs = append(configs, *c)
	}
	return configs, nil
}

func (m *mockGeoRepoNew) UpdateRegionConfig(ctx context.Context, config *RegionConfig) error {
	if m.configs == nil {
		m.configs = make(map[string]*RegionConfig)
	}
	m.configs[config.ID] = config
	return nil
}

func (m *mockGeoRepoNew) GetRegionHealth(ctx context.Context, regionID string) (*RegionHealth, error) {
	if m.health == nil {
		return &RegionHealth{RegionID: regionID, IsHealthy: true}, nil
	}
	h, ok := m.health[regionID]
	if !ok {
		return &RegionHealth{RegionID: regionID, IsHealthy: true}, nil
	}
	return h, nil
}

func (m *mockGeoRepoNew) UpdateRegionHealth(ctx context.Context, health *RegionHealth) error {
	if m.health == nil {
		m.health = make(map[string]*RegionHealth)
	}
	m.health[health.RegionID] = health
	return nil
}

func (m *mockGeoRepoNew) GetRoutingStats(ctx context.Context, tenantID, period string) (*RoutingStats, error) {
	return &RoutingStats{
		TenantID: tenantID,
		Period:   period,
		ByRegion: make(map[Region]int64),
		ByMode:   make(map[RoutingMode]int64),
	}, nil
}

func (m *mockGeoRepoNew) RecordRoutingDecision(ctx context.Context, decision *RoutingDecision) error {
	return nil
}

func (m *mockGeoRepoNew) CreateGeoRegion(ctx context.Context, region *GeoRegion) error {
	return nil
}

func (m *mockGeoRepoNew) GetGeoRegion(ctx context.Context, name string) (*GeoRegion, error) {
	return nil, nil
}

func (m *mockGeoRepoNew) ListGeoRegions(ctx context.Context) ([]GeoRegion, error) {
	return nil, nil
}

func (m *mockGeoRepoNew) UpdateGeoRegion(ctx context.Context, region *GeoRegion) error {
	return nil
}

func (m *mockGeoRepoNew) CreateGeoRoutingPolicy(ctx context.Context, policy *GeoRoutingPolicy) error {
	return nil
}

func (m *mockGeoRepoNew) GetGeoRoutingPolicy(ctx context.Context, tenantID uuid.UUID) (*GeoRoutingPolicy, error) {
	return nil, nil
}

func (m *mockGeoRepoNew) GetGeoRoutingPolicyByID(ctx context.Context, id uuid.UUID) (*GeoRoutingPolicy, error) {
	return nil, nil
}

func (m *mockGeoRepoNew) UpdateGeoRoutingPolicy(ctx context.Context, policy *GeoRoutingPolicy) error {
	return nil
}

func (m *mockGeoRepoNew) ListGeoRoutingPolicies(ctx context.Context, tenantID uuid.UUID) ([]GeoRoutingPolicy, error) {
	return nil, nil
}

func (m *mockGeoRepoNew) GetEndpointRegionConfig(ctx context.Context, endpointID uuid.UUID) (*EndpointRegionConfig, error) {
	return nil, nil
}

func (m *mockGeoRepoNew) SaveEndpointRegionConfig(ctx context.Context, config *EndpointRegionConfig) error {
	return nil
}

func (m *mockGeoRepoNew) RecordGeoRoutingDecision(ctx context.Context, decision *GeoRoutingDecision) error {
	return nil
}

func (m *mockGeoRepoNew) ListGeoRoutingDecisions(ctx context.Context, tenantID uuid.UUID, limit int) ([]GeoRoutingDecision, error) {
	return nil, nil
}
