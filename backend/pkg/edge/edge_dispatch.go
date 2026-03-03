package edge

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Dispatch strategy constants
const (
	DispatchStrategyLatency   = "lowest_latency"
	DispatchStrategyGeo       = "geo_nearest"
	DispatchStrategyLoadBalance = "load_balance"
	DispatchStrategyFailover  = "failover"
)

// Node status constants
const (
	EdgeNodeActive      = "active"
	EdgeNodeDraining    = "draining"
	EdgeNodeMaintenance = "maintenance"
	EdgeNodeOffline     = "offline"
)

// EdgeDispatchConfig configures the edge delivery behavior for a tenant.
type EdgeDispatchConfig struct {
	ID              string   `json:"id"`
	TenantID        string   `json:"tenant_id"`
	Strategy        string   `json:"strategy"`
	PreferredRegions []string `json:"preferred_regions,omitempty"`
	MaxLatencyMs    int      `json:"max_latency_ms"`
	EnableFailover  bool     `json:"enable_failover"`
	FailoverRegions []string `json:"failover_regions,omitempty"`
	Enabled         bool     `json:"enabled"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// EdgeDeliveryMetrics tracks delivery metrics per edge node.
type EdgeDeliveryMetrics struct {
	NodeID           string    `json:"node_id"`
	Region           string    `json:"region"`
	TotalDeliveries  int64     `json:"total_deliveries"`
	SuccessCount     int64     `json:"success_count"`
	FailureCount     int64     `json:"failure_count"`
	AvgLatencyMs     float64   `json:"avg_latency_ms"`
	P50LatencyMs     float64   `json:"p50_latency_ms"`
	P99LatencyMs     float64   `json:"p99_latency_ms"`
	ActiveConnections int      `json:"active_connections"`
	LastDeliveryAt   *time.Time `json:"last_delivery_at,omitempty"`
}

// EdgeDispatchResult captures the result of dispatching via the edge network.
type EdgeDispatchResult struct {
	ID             string    `json:"id"`
	WebhookID      string    `json:"webhook_id"`
	NodeID         string    `json:"node_id"`
	Region         string    `json:"region"`
	Strategy       string    `json:"strategy"`
	LatencyMs      int       `json:"latency_ms"`
	StatusCode     int       `json:"status_code"`
	Success        bool      `json:"success"`
	FailoverUsed   bool      `json:"failover_used"`
	OriginalNode   string    `json:"original_node,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
}

// EdgeNetworkOverview provides a dashboard view of the edge network.
type EdgeNetworkOverview struct {
	TotalNodes       int                   `json:"total_nodes"`
	ActiveNodes      int                   `json:"active_nodes"`
	Regions          []string              `json:"regions"`
	TotalDeliveries  int64                 `json:"total_deliveries"`
	AvgLatencyMs     float64               `json:"avg_latency_ms"`
	NodeMetricsList  []EdgeDeliveryMetrics `json:"node_metrics"`
	HealthScore      float64               `json:"health_score"`
}

// Request DTOs

type CreateEdgeDispatchConfigRequest struct {
	Strategy         string   `json:"strategy,omitempty"`
	PreferredRegions []string `json:"preferred_regions,omitempty"`
	MaxLatencyMs     int      `json:"max_latency_ms,omitempty"`
	EnableFailover   bool     `json:"enable_failover"`
	FailoverRegions  []string `json:"failover_regions,omitempty"`
}

type DispatchWebhookRequest struct {
	WebhookID    string  `json:"webhook_id" binding:"required"`
	EndpointURL  string  `json:"endpoint_url" binding:"required"`
	ReceiverLat  float64 `json:"receiver_lat,omitempty"`
	ReceiverLon  float64 `json:"receiver_lon,omitempty"`
	PayloadSize  int64   `json:"payload_size,omitempty"`
}

type RecordEdgeDeliveryRequest struct {
	NodeID     string `json:"node_id" binding:"required"`
	WebhookID  string `json:"webhook_id" binding:"required"`
	LatencyMs  int    `json:"latency_ms" binding:"required"`
	StatusCode int    `json:"status_code" binding:"required"`
	Success    bool   `json:"success"`
}

// In-memory state for edge dispatch
type edgeDispatchState struct {
	mu       sync.RWMutex
	configs  map[string]*EdgeDispatchConfig
	metrics  map[string]*EdgeDeliveryMetrics
	results  []EdgeDispatchResult
}

var globalEdgeDispatchState = &edgeDispatchState{
	configs: make(map[string]*EdgeDispatchConfig),
	metrics: make(map[string]*EdgeDeliveryMetrics),
}

// newEdgeDispatchState creates a fresh edgeDispatchState instance.
func newEdgeDispatchState() *edgeDispatchState {
	return &edgeDispatchState{
		configs: make(map[string]*EdgeDispatchConfig),
		metrics: make(map[string]*EdgeDeliveryMetrics),
	}
}

// resetEdgeDispatchState resets the dispatch state; used by tests to prevent state leakage.
func (s *Service) resetEdgeDispatchState() {
	s.dispatch.mu.Lock()
	defer s.dispatch.mu.Unlock()
	s.dispatch.configs = make(map[string]*EdgeDispatchConfig)
	s.dispatch.metrics = make(map[string]*EdgeDeliveryMetrics)
	s.dispatch.results = nil
}

// CreateEdgeDispatchConfig creates or updates edge dispatch configuration for a tenant.
func (s *Service) CreateEdgeDispatchConfig(ctx context.Context, tenantID string, req *CreateEdgeDispatchConfigRequest) (*EdgeDispatchConfig, error) {
	strategy := req.Strategy
	if strategy == "" {
		strategy = DispatchStrategyLatency
	}

	validStrategies := map[string]bool{
		DispatchStrategyLatency: true, DispatchStrategyGeo: true,
		DispatchStrategyLoadBalance: true, DispatchStrategyFailover: true,
	}
	if !validStrategies[strategy] {
		return nil, fmt.Errorf("invalid strategy %q", strategy)
	}

	config := &EdgeDispatchConfig{
		ID:               uuid.New().String(),
		TenantID:         tenantID,
		Strategy:         strategy,
		PreferredRegions: req.PreferredRegions,
		MaxLatencyMs:     req.MaxLatencyMs,
		EnableFailover:   req.EnableFailover,
		FailoverRegions:  req.FailoverRegions,
		Enabled:          true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if config.MaxLatencyMs <= 0 {
		config.MaxLatencyMs = 500
	}

	s.dispatch.mu.Lock()
	s.dispatch.configs[tenantID] = config
	s.dispatch.mu.Unlock()

	return config, nil
}

// GetEdgeDispatchConfig returns the edge dispatch config for a tenant.
func (s *Service) GetEdgeDispatchConfig(ctx context.Context, tenantID string) (*EdgeDispatchConfig, error) {
	s.dispatch.mu.RLock()
	defer s.dispatch.mu.RUnlock()

	config, exists := s.dispatch.configs[tenantID]
	if !exists {
		return nil, fmt.Errorf("no edge dispatch config for tenant %s", tenantID)
	}
	return config, nil
}

// DispatchWebhook selects the optimal edge node and dispatches a webhook.
func (s *Service) DispatchWebhook(ctx context.Context, tenantID string, req *DispatchWebhookRequest) (*EdgeDispatchResult, error) {
	if req.WebhookID == "" || req.EndpointURL == "" {
		return nil, fmt.Errorf("webhook_id and endpoint_url are required")
	}

	s.dispatch.mu.RLock()
	config := s.dispatch.configs[tenantID]
	s.dispatch.mu.RUnlock()

	strategy := DispatchStrategyLatency
	if config != nil {
		strategy = config.Strategy
	}

	// Select optimal node
	node := s.selectNode(strategy, req.ReceiverLat, req.ReceiverLon, config)

	result := &EdgeDispatchResult{
		ID:        uuid.New().String(),
		WebhookID: req.WebhookID,
		NodeID:    node.nodeID,
		Region:    node.region,
		Strategy:  strategy,
		Timestamp: time.Now(),
	}

	// Simulate delivery (in production, would actually dispatch)
	result.LatencyMs = node.estimatedLatencyMs
	result.StatusCode = 200
	result.Success = true

	s.dispatch.mu.Lock()
	s.dispatch.results = append(s.dispatch.results, *result)
	s.dispatch.mu.Unlock()

	return result, nil
}

// RecordEdgeDelivery records a delivery result from an edge node.
func (s *Service) RecordEdgeDelivery(ctx context.Context, tenantID string, req *RecordEdgeDeliveryRequest) error {
	s.dispatch.mu.Lock()
	defer s.dispatch.mu.Unlock()

	metrics, exists := s.dispatch.metrics[req.NodeID]
	if !exists {
		metrics = &EdgeDeliveryMetrics{NodeID: req.NodeID}
		s.dispatch.metrics[req.NodeID] = metrics
	}

	metrics.TotalDeliveries++
	if req.Success {
		metrics.SuccessCount++
	} else {
		metrics.FailureCount++
	}

	// Update average latency (running average)
	metrics.AvgLatencyMs = (metrics.AvgLatencyMs*float64(metrics.TotalDeliveries-1) + float64(req.LatencyMs)) / float64(metrics.TotalDeliveries)
	now := time.Now()
	metrics.LastDeliveryAt = &now

	return nil
}

// GetEdgeNetworkOverview returns a dashboard view of the edge network.
func (s *Service) GetEdgeNetworkOverview(ctx context.Context, tenantID string) (*EdgeNetworkOverview, error) {
	s.dispatch.mu.RLock()
	defer s.dispatch.mu.RUnlock()

	nodes := defaultEdgeNodeList()
	overview := &EdgeNetworkOverview{
		TotalNodes: len(nodes),
	}

	regionSet := make(map[string]bool)
	var totalLatency float64
	var deliveryCount int64

	for _, node := range nodes {
		regionSet[node.region] = true
		if node.status == EdgeNodeActive {
			overview.ActiveNodes++
		}
	}

	for region := range regionSet {
		overview.Regions = append(overview.Regions, region)
	}

	for _, metrics := range s.dispatch.metrics {
		overview.NodeMetricsList = append(overview.NodeMetricsList, *metrics)
		deliveryCount += metrics.TotalDeliveries
		totalLatency += metrics.AvgLatencyMs * float64(metrics.TotalDeliveries)
	}

	overview.TotalDeliveries = deliveryCount
	if deliveryCount > 0 {
		overview.AvgLatencyMs = totalLatency / float64(deliveryCount)
	}

	if overview.TotalNodes > 0 {
		overview.HealthScore = float64(overview.ActiveNodes) / float64(overview.TotalNodes) * 100
	}

	return overview, nil
}

// Internal types for node selection

type selectedNode struct {
	nodeID            string
	region            string
	estimatedLatencyMs int
}

func (s *Service) selectNode(strategy string, receiverLat, receiverLon float64, config *EdgeDispatchConfig) selectedNode {
	nodes := defaultEdgeNodeList()

	if len(nodes) == 0 {
		return selectedNode{nodeID: "fallback", region: "us-east-1", estimatedLatencyMs: 100}
	}

	switch strategy {
	case DispatchStrategyGeo:
		if receiverLat != 0 || receiverLon != 0 {
			sort.Slice(nodes, func(i, j int) bool {
				distI := haversine(receiverLat, receiverLon, nodes[i].lat, nodes[i].lon)
				distJ := haversine(receiverLat, receiverLon, nodes[j].lat, nodes[j].lon)
				return distI < distJ
			})
		}
	case DispatchStrategyLoadBalance:
		// Snapshot delivery counts under a single lock to avoid nested locking.
		s.dispatch.mu.RLock()
		deliveryCounts := make(map[string]int64, len(nodes))
		for _, n := range nodes {
			if m := s.dispatch.metrics[n.nodeID]; m != nil {
				deliveryCounts[n.nodeID] = m.TotalDeliveries
			}
		}
		s.dispatch.mu.RUnlock()

		sort.Slice(nodes, func(i, j int) bool {
			return deliveryCounts[nodes[i].nodeID] < deliveryCounts[nodes[j].nodeID]
		})
	default:
		// Lowest latency: sort by base latency
		sort.Slice(nodes, func(i, j int) bool {
			return nodes[i].baseLatencyMs < nodes[j].baseLatencyMs
		})
	}

	// Filter by preferred regions if configured
	if config != nil && len(config.PreferredRegions) > 0 {
		for _, node := range nodes {
			for _, region := range config.PreferredRegions {
				if node.region == region && node.status == EdgeNodeActive {
					return selectedNode{nodeID: node.nodeID, region: node.region, estimatedLatencyMs: node.baseLatencyMs}
				}
			}
		}
	}

	// Return best available active node
	for _, node := range nodes {
		if node.status == EdgeNodeActive {
			return selectedNode{nodeID: node.nodeID, region: node.region, estimatedLatencyMs: node.baseLatencyMs}
		}
	}

	return selectedNode{nodeID: "fallback", region: "us-east-1", estimatedLatencyMs: 100}
}

type edgeNodeInfo struct {
	nodeID       string
	region       string
	lat          float64
	lon          float64
	baseLatencyMs int
	status       string
}

func defaultEdgeNodeList() []edgeNodeInfo {
	return []edgeNodeInfo{
		{nodeID: "edge-us-east", region: "us-east-1", lat: 39.0, lon: -77.5, baseLatencyMs: 15, status: EdgeNodeActive},
		{nodeID: "edge-us-west", region: "us-west-2", lat: 45.5, lon: -122.7, baseLatencyMs: 18, status: EdgeNodeActive},
		{nodeID: "edge-eu-west", region: "eu-west-1", lat: 53.3, lon: -6.3, baseLatencyMs: 25, status: EdgeNodeActive},
		{nodeID: "edge-eu-central", region: "eu-central-1", lat: 50.1, lon: 8.7, baseLatencyMs: 22, status: EdgeNodeActive},
		{nodeID: "edge-ap-southeast", region: "ap-southeast-1", lat: 1.3, lon: 103.8, baseLatencyMs: 30, status: EdgeNodeActive},
		{nodeID: "edge-ap-northeast", region: "ap-northeast-1", lat: 35.7, lon: 139.7, baseLatencyMs: 28, status: EdgeNodeActive},
		{nodeID: "edge-sa-east", region: "sa-east-1", lat: -23.5, lon: -46.6, baseLatencyMs: 45, status: EdgeNodeActive},
		{nodeID: "edge-af-south", region: "af-south-1", lat: -33.9, lon: 18.4, baseLatencyMs: 50, status: EdgeNodeActive},
	}
}

// haversine calculates distance between two points in km.
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}
