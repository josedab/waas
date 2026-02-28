package edgenetwork

import (
	"context"
	"errors"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/utils"
)

var (
	ErrNodeNotFound   = errors.New("edge node not found")
	ErrRuleNotFound   = errors.New("routing rule not found")
	ErrNoHealthyNodes = errors.New("no healthy edge nodes available")
	ErrInvalidRegion  = errors.New("invalid region")
)

// ServiceConfig holds configuration for the edge network service.
type ServiceConfig struct {
	HealthCheckInterval time.Duration
	UnhealthyThreshold  int
	MaxNodesPerRegion   int
	DefaultCapacity     int
	LatencyWeightFactor float64
}

// DefaultServiceConfig returns default configuration.
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		HealthCheckInterval: 30 * time.Second,
		UnhealthyThreshold:  3,
		MaxNodesPerRegion:   20,
		DefaultCapacity:     1000,
		LatencyWeightFactor: 0.7,
	}
}

// Service provides edge delivery network operations.
type Service struct {
	repo   Repository
	config *ServiceConfig
	logger *utils.Logger
	mu     sync.RWMutex
}

// NewService creates a new edge network service.
func NewService(repo Repository, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}
	if repo == nil {
		repo = NewMemoryRepository()
	}
	return &Service{
		repo:   repo,
		config: config,
		logger: utils.NewLogger("edgenetwork"),
	}
}

// RegisterNode adds a new edge node to the network.
func (s *Service) RegisterNode(ctx context.Context, req *CreateNodeRequest) (*EdgeNode, error) {
	if req.Name == "" || req.Endpoint == "" {
		return nil, errors.New("name and endpoint are required")
	}
	if !isValidRegion(req.Region) {
		return nil, ErrInvalidRegion
	}

	capacity := req.Capacity
	if capacity <= 0 {
		capacity = s.config.DefaultCapacity
	}

	node := &EdgeNode{
		ID:           uuid.New().String(),
		Name:         req.Name,
		Region:       req.Region,
		Endpoint:     req.Endpoint,
		Status:       NodeStatusHealthy,
		Latitude:     req.Latitude,
		Longitude:    req.Longitude,
		Capacity:     capacity,
		SuccessRate:  100.0,
		LastHealthAt: time.Now().UTC(),
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := s.repo.CreateNode(ctx, node); err != nil {
		return nil, err
	}
	return node, nil
}

// GetNode retrieves an edge node by ID.
func (s *Service) GetNode(ctx context.Context, id string) (*EdgeNode, error) {
	return s.repo.GetNode(ctx, id)
}

// ListNodes returns edge nodes, optionally filtered by region.
func (s *Service) ListNodes(ctx context.Context, region *Region) ([]EdgeNode, error) {
	return s.repo.ListNodes(ctx, region)
}

// RemoveNode removes an edge node from the network.
func (s *Service) RemoveNode(ctx context.Context, id string) error {
	return s.repo.DeleteNode(ctx, id)
}

// ResolveRoute determines the optimal edge node for delivering to a target.
func (s *Service) ResolveRoute(ctx context.Context, tenantID, targetURL string, sourceRegion Region) (*DeliveryRoute, error) {
	nodes, err := s.repo.ListNodes(ctx, nil)
	if err != nil {
		return nil, err
	}

	healthy := filterHealthyNodes(nodes)
	if len(healthy) == 0 {
		return nil, ErrNoHealthyNodes
	}

	// Check tenant routing rules
	rules, _ := s.repo.ListRoutingRules(ctx, tenantID)
	if len(rules) > 0 {
		sort.Slice(rules, func(i, j int) bool { return rules[i].Priority > rules[j].Priority })
		for _, rule := range rules {
			if !rule.Enabled {
				continue
			}
			if route := s.applyRule(healthy, &rule, sourceRegion); route != nil {
				return route, nil
			}
		}
	}

	// Default: lowest latency strategy
	best := s.selectByLatency(healthy, sourceRegion)
	return &DeliveryRoute{
		SourceNode: best.ID,
		TargetNode: targetURL,
		Region:     best.Region,
		LatencyMs:  best.AvgLatencyMs,
		Hops:       1,
		Strategy:   string(StrategyLatency),
	}, nil
}

// GetTopology returns the current network topology.
func (s *Service) GetTopology(ctx context.Context) (*NetworkTopology, error) {
	nodes, err := s.repo.ListNodes(ctx, nil)
	if err != nil {
		return nil, err
	}

	regions := make(map[Region]int)
	healthy := 0
	for _, n := range nodes {
		regions[n.Region]++
		if n.Status == NodeStatusHealthy {
			healthy++
		}
	}

	healthyPct := 0.0
	if len(nodes) > 0 {
		healthyPct = float64(healthy) / float64(len(nodes)) * 100
	}

	return &NetworkTopology{
		Nodes:      nodes,
		TotalNodes: len(nodes),
		HealthyPct: healthyPct,
		Regions:    regions,
		UpdatedAt:  time.Now().UTC(),
	}, nil
}

// GetMetrics returns aggregate network performance metrics.
func (s *Service) GetMetrics(ctx context.Context) (*NetworkMetrics, error) {
	nodes, err := s.repo.ListNodes(ctx, nil)
	if err != nil {
		return nil, err
	}

	regionMetrics := make(map[Region]*RegionMetric)
	var totalLatency float64
	var totalSuccess float64

	for _, n := range nodes {
		rm, ok := regionMetrics[n.Region]
		if !ok {
			rm = &RegionMetric{Region: n.Region}
			regionMetrics[n.Region] = rm
		}
		rm.NodeCount++
		rm.AvgLatencyMs += n.AvgLatencyMs
		rm.SuccessRate += n.SuccessRate
		rm.ActiveDeliveries += n.ActiveConns
		totalLatency += n.AvgLatencyMs
		totalSuccess += n.SuccessRate
	}

	// Average region metrics
	for _, rm := range regionMetrics {
		if rm.NodeCount > 0 {
			rm.AvgLatencyMs /= float64(rm.NodeCount)
			rm.SuccessRate /= float64(rm.NodeCount)
		}
	}

	avgLatency := 0.0
	avgSuccess := 0.0
	if len(nodes) > 0 {
		avgLatency = totalLatency / float64(len(nodes))
		avgSuccess = totalSuccess / float64(len(nodes))
	}

	return &NetworkMetrics{
		AvgLatencyMs:  avgLatency,
		SuccessRate:   avgSuccess,
		RegionMetrics: regionMetrics,
		MeasuredAt:    time.Now().UTC(),
	}, nil
}

// CreateRoutingRule creates a new routing rule.
func (s *Service) CreateRoutingRule(ctx context.Context, tenantID string, req *CreateRoutingRuleRequest) (*RoutingRule, error) {
	if req.Name == "" || req.Strategy == "" {
		return nil, errors.New("name and strategy are required")
	}

	rule := &RoutingRule{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		Name:           req.Name,
		Priority:       req.Priority,
		Strategy:       req.Strategy,
		TargetRegions:  req.TargetRegions,
		Conditions:     req.Conditions,
		FallbackRegion: req.FallbackRegion,
		Enabled:        true,
		CreatedAt:      time.Now().UTC(),
	}

	if err := s.repo.CreateRoutingRule(ctx, rule); err != nil {
		return nil, err
	}
	return rule, nil
}

// ListRoutingRules returns routing rules for a tenant.
func (s *Service) ListRoutingRules(ctx context.Context, tenantID string) ([]RoutingRule, error) {
	return s.repo.ListRoutingRules(ctx, tenantID)
}

// DeleteRoutingRule removes a routing rule.
func (s *Service) DeleteRoutingRule(ctx context.Context, tenantID, id string) error {
	return s.repo.DeleteRoutingRule(ctx, tenantID, id)
}

func (s *Service) selectByLatency(nodes []EdgeNode, sourceRegion Region) *EdgeNode {
	best := &nodes[0]
	for i := range nodes {
		n := &nodes[i]
		score := n.AvgLatencyMs
		if n.Region == sourceRegion {
			score *= 0.5 // Prefer same-region nodes
		}
		bestScore := best.AvgLatencyMs
		if best.Region == sourceRegion {
			bestScore *= 0.5
		}
		if score < bestScore {
			best = n
		}
	}
	return best
}

func (s *Service) applyRule(nodes []EdgeNode, rule *RoutingRule, sourceRegion Region) *DeliveryRoute {
	var candidates []EdgeNode
	if len(rule.TargetRegions) > 0 {
		for _, n := range nodes {
			for _, r := range rule.TargetRegions {
				if n.Region == r {
					candidates = append(candidates, n)
				}
			}
		}
	} else {
		candidates = nodes
	}
	if len(candidates) == 0 {
		return nil
	}

	var selected *EdgeNode
	switch rule.Strategy {
	case StrategyGeo:
		selected = s.selectByGeo(candidates, sourceRegion)
	case StrategyLatency:
		selected = s.selectByLatency(candidates, sourceRegion)
	default:
		selected = &candidates[0]
	}

	return &DeliveryRoute{
		SourceNode: selected.ID,
		Region:     selected.Region,
		LatencyMs:  selected.AvgLatencyMs,
		Hops:       1,
		Strategy:   string(rule.Strategy),
	}
}

func (s *Service) selectByGeo(nodes []EdgeNode, sourceRegion Region) *EdgeNode {
	// Prefer same-region nodes
	for i := range nodes {
		if nodes[i].Region == sourceRegion {
			return &nodes[i]
		}
	}
	return &nodes[0]
}

func filterHealthyNodes(nodes []EdgeNode) []EdgeNode {
	var healthy []EdgeNode
	for _, n := range nodes {
		if n.Status == NodeStatusHealthy || n.Status == NodeStatusDegraded {
			healthy = append(healthy, n)
		}
	}
	return healthy
}

func isValidRegion(r Region) bool {
	for _, valid := range AllRegions() {
		if r == valid {
			return true
		}
	}
	return false
}

// haversineDistance calculates distance in km between two coordinates.
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371.0
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadius * c
}
