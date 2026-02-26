package georouting

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/utils"
)

// Service manages geographic routing
type Service struct {
	repo   Repository
	router *Router
	logger *utils.Logger
}

// NewService creates a new geo-routing service
func NewService(repo Repository, geoIP GeoIPProvider) *Service {
	return &Service{
		repo:   repo,
		router: NewRouter(repo, geoIP),
		logger: utils.NewLogger("georouting-service"),
	}
}

// GetRouter returns the underlying router
func (s *Service) GetRouter() *Router {
	return s.router
}

// CreateEndpointRouting creates routing configuration for an endpoint
func (s *Service) CreateEndpointRouting(ctx context.Context, tenantID string, req *CreateRoutingRequest) (*EndpointRouting, error) {
	// Validate regions
	if len(req.Regions) > 0 {
		for _, region := range req.Regions {
			if !isValidRegion(region) {
				return nil, fmt.Errorf("invalid region: %s", region)
			}
		}
	}

	if req.PrimaryRegion != "" && !isValidRegion(req.PrimaryRegion) {
		return nil, fmt.Errorf("invalid primary region: %s", req.PrimaryRegion)
	}

	routing := &EndpointRouting{
		ID:              uuid.New().String(),
		EndpointID:      req.EndpointID,
		TenantID:        tenantID,
		Mode:            req.Mode,
		PrimaryRegion:   req.PrimaryRegion,
		Regions:         req.Regions,
		DataResidency:   req.DataResidency,
		FailoverEnabled: req.FailoverEnabled,
		LatencyBased:    req.LatencyBased,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Set defaults
	if routing.PrimaryRegion == "" {
		routing.PrimaryRegion = RegionUSEast
	}
	if len(routing.Regions) == 0 {
		routing.Regions = AllRegions()
	}
	if routing.DataResidency == "" {
		routing.DataResidency = ResidencyNone
	}

	if err := s.repo.CreateEndpointRouting(ctx, routing); err != nil {
		return nil, err
	}

	return routing, nil
}

// GetEndpointRouting retrieves routing configuration for an endpoint
func (s *Service) GetEndpointRouting(ctx context.Context, tenantID, endpointID string) (*EndpointRouting, error) {
	return s.repo.GetEndpointRouting(ctx, tenantID, endpointID)
}

// UpdateEndpointRouting updates routing configuration
func (s *Service) UpdateEndpointRouting(ctx context.Context, tenantID, endpointID string, req *UpdateRoutingRequest) (*EndpointRouting, error) {
	routing, err := s.repo.GetEndpointRouting(ctx, tenantID, endpointID)
	if err != nil {
		return nil, err
	}
	if routing == nil {
		return nil, fmt.Errorf("routing configuration not found")
	}

	if req.Mode != nil {
		routing.Mode = *req.Mode
	}
	if req.PrimaryRegion != nil {
		if !isValidRegion(*req.PrimaryRegion) {
			return nil, fmt.Errorf("invalid primary region: %s", *req.PrimaryRegion)
		}
		routing.PrimaryRegion = *req.PrimaryRegion
	}
	if len(req.Regions) > 0 {
		for _, region := range req.Regions {
			if !isValidRegion(region) {
				return nil, fmt.Errorf("invalid region: %s", region)
			}
		}
		routing.Regions = req.Regions
	}
	if req.DataResidency != nil {
		routing.DataResidency = *req.DataResidency
	}
	if req.FailoverEnabled != nil {
		routing.FailoverEnabled = *req.FailoverEnabled
	}
	if req.LatencyBased != nil {
		routing.LatencyBased = *req.LatencyBased
	}

	routing.UpdatedAt = time.Now()

	if err := s.repo.UpdateEndpointRouting(ctx, routing); err != nil {
		return nil, err
	}

	return routing, nil
}

// DeleteEndpointRouting deletes routing configuration
func (s *Service) DeleteEndpointRouting(ctx context.Context, tenantID, endpointID string) error {
	return s.repo.DeleteEndpointRouting(ctx, tenantID, endpointID)
}

// RouteDelivery determines the best region for a delivery
func (s *Service) RouteDelivery(ctx context.Context, tenantID, endpointID, clientIP string) (*RoutingDecision, error) {
	return s.router.Route(ctx, tenantID, endpointID, clientIP)
}

// GetRegions returns all available regions with their info
func (s *Service) GetRegions() []RegionInfo {
	return GetRegionInfo()
}

// GetRegionHealth returns health status for all regions
func (s *Service) GetRegionHealth(ctx context.Context) (map[string]*RegionHealth, error) {
	return s.router.healthTracker.GetAllHealth(), nil
}

// GetRoutingStats returns routing statistics
func (s *Service) GetRoutingStats(ctx context.Context, tenantID string, period string) (*RoutingStats, error) {
	return s.repo.GetRoutingStats(ctx, tenantID, period)
}

// CreateRegionConfig creates a new region configuration
func (s *Service) CreateRegionConfig(ctx context.Context, config *RegionConfig) error {
	if config.ID == "" {
		config.ID = uuid.New().String()
	}
	config.CreatedAt = time.Now()
	config.UpdatedAt = time.Now()
	return s.repo.CreateRegionConfig(ctx, config)
}

// ListRegionConfigs lists all region configurations
func (s *Service) ListRegionConfigs(ctx context.Context) ([]RegionConfig, error) {
	return s.repo.ListRegionConfigs(ctx)
}

// UpdateRegionConfig updates a region configuration
func (s *Service) UpdateRegionConfig(ctx context.Context, config *RegionConfig) error {
	config.UpdatedAt = time.Now()
	return s.repo.UpdateRegionConfig(ctx, config)
}

func isValidRegion(r Region) bool {
	for _, valid := range AllRegions() {
		if valid == r {
			return true
		}
	}
	return false
}

// CreateGeoRegion registers a new geo region
func (s *Service) CreateGeoRegion(ctx context.Context, region *GeoRegion) error {
	if region.ID == uuid.Nil {
		region.ID = uuid.New()
	}
	if region.Status == "" {
		region.Status = "active"
	}
	region.CreatedAt = time.Now()
	return s.repo.CreateGeoRegion(ctx, region)
}

// ListGeoRegions returns all geo regions with health status
func (s *Service) ListGeoRegions(ctx context.Context) ([]GeoRegion, error) {
	return s.repo.ListGeoRegions(ctx)
}

// CreateGeoRoutingPolicy creates a new geo routing policy for a tenant
func (s *Service) CreateGeoRoutingPolicy(ctx context.Context, tenantID uuid.UUID, policy *GeoRoutingPolicy) error {
	if policy.ID == uuid.Nil {
		policy.ID = uuid.New()
	}
	policy.TenantID = tenantID
	policy.Active = true
	policy.CreatedAt = time.Now()
	return s.repo.CreateGeoRoutingPolicy(ctx, policy)
}

// GetGeoRoutingPolicy returns the active routing policy for a tenant
func (s *Service) GetGeoRoutingPolicy(ctx context.Context, tenantID uuid.UUID) (*GeoRoutingPolicy, error) {
	return s.repo.GetGeoRoutingPolicy(ctx, tenantID)
}

// ListGeoRoutingPolicies lists all routing policies for a tenant
func (s *Service) ListGeoRoutingPolicies(ctx context.Context, tenantID uuid.UUID) ([]GeoRoutingPolicy, error) {
	return s.repo.ListGeoRoutingPolicies(ctx, tenantID)
}

// UpdateGeoRoutingPolicy updates an existing routing policy
func (s *Service) UpdateGeoRoutingPolicy(ctx context.Context, policy *GeoRoutingPolicy) error {
	return s.repo.UpdateGeoRoutingPolicy(ctx, policy)
}

// RouteEvent determines the best region and records the decision
func (s *Service) RouteEvent(ctx context.Context, tenantID, endpointID uuid.UUID, eventPayload []byte) (*GeoRoutingDecision, error) {
	decision, err := s.router.RouteDeliveryEnhanced(ctx, tenantID, endpointID, eventPayload)
	if err != nil {
		return nil, err
	}
	// Persist the decision
	if err := s.repo.RecordGeoRoutingDecision(ctx, decision); err != nil {
		s.logger.Error("failed to record geo-routing decision", map[string]interface{}{"error": err.Error(), "tenant_id": tenantID.String()})
	}
	return decision, nil
}

// GetRoutingHistory returns recent routing decisions for a tenant
func (s *Service) GetRoutingHistory(ctx context.Context, tenantID uuid.UUID, limit int) ([]GeoRoutingDecision, error) {
	return s.repo.ListGeoRoutingDecisions(ctx, tenantID, limit)
}

// ConfigureEndpointRegion sets region preferences for an endpoint
func (s *Service) ConfigureEndpointRegion(ctx context.Context, endpointID uuid.UUID, config *EndpointRegionConfig) error {
	config.EndpointID = endpointID
	return s.repo.SaveEndpointRegionConfig(ctx, config)
}

// SimulateRouting simulates a routing decision without persisting
func (s *Service) SimulateRouting(ctx context.Context, tenantID uuid.UUID, sourceIP string) (*GeoRoutingDecision, error) {
	policy, err := s.repo.GetGeoRoutingPolicy(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get routing policy: %w", err)
	}
	if policy == nil {
		return &GeoRoutingDecision{
			EventID:        uuid.New(),
			SelectedRegion: string(RegionUSEast),
			Reason:         "default (no policy)",
		}, nil
	}

	selected, err := s.router.SelectRegion(ctx, policy, sourceIP)
	if err != nil {
		return nil, err
	}

	decision := &GeoRoutingDecision{
		EventID:        uuid.New(),
		SelectedRegion: selected,
		Reason:         fmt.Sprintf("simulated-%s", policy.Strategy),
	}

	// Collect alternatives
	regions, _ := s.repo.ListGeoRegions(ctx)
	for _, rg := range regions {
		if rg.Name != selected && (rg.Status == "active" || rg.Status == "degraded") {
			decision.AlternativeRegions = append(decision.AlternativeRegions, rg.Name)
		}
		if rg.Name == selected {
			decision.Latency = rg.AvgLatency
		}
	}

	return decision, nil
}

// GetGeoDashboard returns dashboard data for the geo-routing overview
func (s *Service) GetGeoDashboard(ctx context.Context) (*GeoDashboardData, error) {
	regions, err := s.repo.ListGeoRegions(ctx)
	if err != nil {
		return nil, err
	}

	healthList, _ := s.router.healthTracker.GetAllRegionHealthStatus(ctx)

	loadDist := make(map[string]float64)
	latencyMap := make(map[string]int)
	for _, rg := range regions {
		if rg.Capacity > 0 {
			loadDist[rg.Name] = float64(rg.CurrentLoad) / float64(rg.Capacity) * 100.0
		}
		latencyMap[rg.Name] = rg.AvgLatency
	}

	return &GeoDashboardData{
		Regions:          regions,
		Health:           healthList,
		LoadDistribution: loadDist,
		LatencyMap:       latencyMap,
	}, nil
}
