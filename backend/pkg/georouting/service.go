package georouting

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service manages geographic routing
type Service struct {
	repo   Repository
	router *Router
}

// NewService creates a new geo-routing service
func NewService(repo Repository, geoIP GeoIPProvider) *Service {
	return &Service{
		repo:   repo,
		router: NewRouter(repo, geoIP),
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
