package edge

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/httputil"
)

// EdgeDeliveryNetwork manages the global edge delivery infrastructure
type EdgeDeliveryNetwork struct {
	nodes  map[string]*EdgeNode
	router *LatencyRouter
	mu     sync.RWMutex
}

// EdgeNode represents a delivery node at a geographic edge location
type EdgeNode struct {
	ID             string       `json:"id"`
	Name           string       `json:"name"`
	Region         string       `json:"region"`
	Location       GeoLocation  `json:"location"`
	Status         string       `json:"status"` // healthy, degraded, unhealthy, maintenance
	Capacity       NodeCapacity `json:"capacity"`
	Metrics        NodeMetrics  `json:"metrics"`
	DataResidency  []string     `json:"data_residency"` // country codes for compliance
	RetryQueueSize int          `json:"retry_queue_size"`
	LastHealthAt   time.Time    `json:"last_health_check"`
	CreatedAt      time.Time    `json:"created_at"`
}

// GeoLocation represents geographic coordinates
type GeoLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	City      string  `json:"city"`
	Country   string  `json:"country"`
	Continent string  `json:"continent"`
}

// NodeCapacity tracks node capacity
type NodeCapacity struct {
	MaxRPS         int     `json:"max_rps"`
	CurrentRPS     int     `json:"current_rps"`
	UtilizationPct float64 `json:"utilization_pct"`
	QueueDepth     int     `json:"queue_depth"`
	MaxConcurrent  int     `json:"max_concurrent"`
}

// NodeMetrics tracks node performance
type NodeMetrics struct {
	AvgLatencyMs  float64 `json:"avg_latency_ms"`
	P50LatencyMs  float64 `json:"p50_latency_ms"`
	P95LatencyMs  float64 `json:"p95_latency_ms"`
	P99LatencyMs  float64 `json:"p99_latency_ms"`
	SuccessRate   float64 `json:"success_rate"`
	DeliveryCount int64   `json:"delivery_count_24h"`
	ErrorCount    int64   `json:"error_count_24h"`
	RetryCount    int64   `json:"retry_count_24h"`
}

// LatencyRouter makes routing decisions based on latency
type LatencyRouter struct {
	latencyMap map[string]map[string]float64 // source -> target -> latency_ms
	mu         sync.RWMutex
}

// RoutingDecision describes why a particular edge node was selected
type RoutingDecision struct {
	SelectedNode     string            `json:"selected_node"`
	Region           string            `json:"region"`
	Reason           string            `json:"reason"`
	EstimatedLatency float64           `json:"estimated_latency_ms"`
	Alternatives     []AlternativeNode `json:"alternatives,omitempty"`
	DataResidency    bool              `json:"data_residency_compliant"`
}

// AlternativeNode is a candidate node that wasn't selected
type AlternativeNode struct {
	NodeID           string  `json:"node_id"`
	Region           string  `json:"region"`
	EstimatedLatency float64 `json:"estimated_latency_ms"`
	Reason           string  `json:"reason_not_selected"`
}

// EdgeDeliveryRequest represents a delivery routed through the edge
type EdgeDeliveryRequest struct {
	WebhookID      string            `json:"webhook_id"`
	EndpointURL    string            `json:"endpoint_url"`
	Payload        []byte            `json:"payload"`
	Headers        map[string]string `json:"headers,omitempty"`
	RequiredRegion string            `json:"required_region,omitempty"`
	DataResidency  string            `json:"data_residency,omitempty"` // country code
	PriorityLevel  int               `json:"priority_level,omitempty"` // 1-5
}

// EdgeDeliveryResult captures the delivery outcome through edge
type EdgeDeliveryResult struct {
	DeliveryID     string    `json:"delivery_id"`
	NodeID         string    `json:"node_id"`
	Region         string    `json:"region"`
	StatusCode     int       `json:"status_code"`
	Success        bool      `json:"success"`
	LatencyMs      int64     `json:"latency_ms"`
	NetworkLatency int64     `json:"network_latency_ms"`
	ProcessLatency int64     `json:"process_latency_ms"`
	DeliveredAt    time.Time `json:"delivered_at"`
}

// NetworkStatus provides global edge network health
type NetworkStatus struct {
	TotalNodes       int            `json:"total_nodes"`
	HealthyNodes     int            `json:"healthy_nodes"`
	DegradedNodes    int            `json:"degraded_nodes"`
	UnhealthyNodes   int            `json:"unhealthy_nodes"`
	GlobalAvgLatency float64        `json:"global_avg_latency_ms"`
	RegionStatus     []RegionStatus `json:"regions"`
	TotalRPS         int            `json:"total_rps"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

// RegionStatus provides per-region health
type RegionStatus struct {
	Region       string  `json:"region"`
	NodeCount    int     `json:"node_count"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	SuccessRate  float64 `json:"success_rate"`
	Capacity     float64 `json:"capacity_pct"`
	Status       string  `json:"status"`
}

// NewEdgeDeliveryNetwork creates the global edge network
func NewEdgeDeliveryNetwork() *EdgeDeliveryNetwork {
	network := &EdgeDeliveryNetwork{
		nodes: make(map[string]*EdgeNode),
		router: &LatencyRouter{
			latencyMap: make(map[string]map[string]float64),
		},
	}

	// Initialize with global edge nodes
	for _, node := range defaultEdgeNodes() {
		network.nodes[node.ID] = node
	}

	return network
}

// RouteDelivery selects the optimal edge node for a delivery
func (n *EdgeDeliveryNetwork) RouteDelivery(ctx context.Context, req *EdgeDeliveryRequest) (*RoutingDecision, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	var candidates []*EdgeNode
	for _, node := range n.nodes {
		if node.Status == "unhealthy" || node.Status == "maintenance" {
			continue
		}
		// Check data residency compliance
		if req.DataResidency != "" {
			compliant := false
			for _, dr := range node.DataResidency {
				if dr == req.DataResidency {
					compliant = true
					break
				}
			}
			if !compliant {
				continue
			}
		}
		// Check region preference
		if req.RequiredRegion != "" && node.Region != req.RequiredRegion {
			continue
		}
		// Check capacity
		if node.Capacity.UtilizationPct >= 95 {
			continue
		}
		candidates = append(candidates, node)
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no available edge nodes for delivery")
	}

	// Score and rank candidates
	sort.Slice(candidates, func(i, j int) bool {
		scoreI := scoreNode(candidates[i])
		scoreJ := scoreNode(candidates[j])
		return scoreI > scoreJ
	})

	selected := candidates[0]
	decision := &RoutingDecision{
		SelectedNode:     selected.ID,
		Region:           selected.Region,
		Reason:           fmt.Sprintf("Optimal latency (%.0fms) from %s", selected.Metrics.AvgLatencyMs, selected.Location.City),
		EstimatedLatency: selected.Metrics.AvgLatencyMs,
		DataResidency:    req.DataResidency == "" || containsStr(selected.DataResidency, req.DataResidency),
	}

	// Add alternatives
	for i := 1; i < len(candidates) && i <= 3; i++ {
		alt := candidates[i]
		decision.Alternatives = append(decision.Alternatives, AlternativeNode{
			NodeID:           alt.ID,
			Region:           alt.Region,
			EstimatedLatency: alt.Metrics.AvgLatencyMs,
			Reason:           fmt.Sprintf("Higher latency (%.0fms)", alt.Metrics.AvgLatencyMs),
		})
	}

	return decision, nil
}

// GetNetworkStatus returns global edge network health
func (n *EdgeDeliveryNetwork) GetNetworkStatus() *NetworkStatus {
	n.mu.RLock()
	defer n.mu.RUnlock()

	status := &NetworkStatus{UpdatedAt: time.Now()}
	regionMap := make(map[string]*RegionStatus)
	var totalLatency float64

	for _, node := range n.nodes {
		status.TotalNodes++
		status.TotalRPS += node.Capacity.CurrentRPS
		totalLatency += node.Metrics.AvgLatencyMs

		switch node.Status {
		case "healthy":
			status.HealthyNodes++
		case "degraded":
			status.DegradedNodes++
		default:
			status.UnhealthyNodes++
		}

		if _, ok := regionMap[node.Region]; !ok {
			regionMap[node.Region] = &RegionStatus{Region: node.Region, Status: "healthy"}
		}
		rs := regionMap[node.Region]
		rs.NodeCount++
		rs.AvgLatencyMs += node.Metrics.AvgLatencyMs
		rs.SuccessRate += node.Metrics.SuccessRate
		rs.Capacity += node.Capacity.UtilizationPct
	}

	if status.TotalNodes > 0 {
		status.GlobalAvgLatency = totalLatency / float64(status.TotalNodes)
	}

	for _, rs := range regionMap {
		if rs.NodeCount > 0 {
			rs.AvgLatencyMs /= float64(rs.NodeCount)
			rs.SuccessRate /= float64(rs.NodeCount)
			rs.Capacity /= float64(rs.NodeCount)
		}
		status.RegionStatus = append(status.RegionStatus, *rs)
	}

	return status
}

func scoreNode(node *EdgeNode) float64 {
	latencyScore := math.Max(0, 100-node.Metrics.AvgLatencyMs)
	successScore := node.Metrics.SuccessRate
	capacityScore := 100 - node.Capacity.UtilizationPct
	healthScore := 0.0
	if node.Status == "healthy" {
		healthScore = 100
	} else if node.Status == "degraded" {
		healthScore = 50
	}
	return (latencyScore * 0.4) + (successScore * 0.3) + (capacityScore * 0.2) + (healthScore * 0.1)
}

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func defaultEdgeNodes() []*EdgeNode {
	nodes := []struct {
		name, region, city, country, continent string
		lat, lon                               float64
		residency                              []string
	}{
		{"US East", "us-east-1", "Virginia", "US", "NA", 39.0, -77.5, []string{"US"}},
		{"US West", "us-west-2", "Oregon", "US", "NA", 45.5, -122.7, []string{"US"}},
		{"EU West", "eu-west-1", "Dublin", "IE", "EU", 53.3, -6.3, []string{"IE", "EU"}},
		{"EU Central", "eu-central-1", "Frankfurt", "DE", "EU", 50.1, 8.7, []string{"DE", "EU"}},
		{"Asia Pacific", "ap-southeast-1", "Singapore", "SG", "AS", 1.3, 103.8, []string{"SG"}},
		{"Asia NE", "ap-northeast-1", "Tokyo", "JP", "AS", 35.7, 139.7, []string{"JP"}},
		{"South America", "sa-east-1", "São Paulo", "BR", "SA", -23.5, -46.6, []string{"BR"}},
		{"Australia", "ap-southeast-2", "Sydney", "AU", "OC", -33.9, 151.2, []string{"AU"}},
		{"Middle East", "me-south-1", "Bahrain", "BH", "AS", 26.1, 50.6, []string{"BH"}},
		{"Africa", "af-south-1", "Cape Town", "ZA", "AF", -33.9, 18.4, []string{"ZA"}},
	}

	var edgeNodes []*EdgeNode
	for _, n := range nodes {
		edgeNodes = append(edgeNodes, &EdgeNode{
			ID:     uuid.New().String(),
			Name:   n.name,
			Region: n.region,
			Location: GeoLocation{
				Latitude: n.lat, Longitude: n.lon,
				City: n.city, Country: n.country, Continent: n.continent,
			},
			Status: "healthy",
			Capacity: NodeCapacity{
				MaxRPS: 10000, CurrentRPS: 2500,
				UtilizationPct: 25, MaxConcurrent: 1000,
			},
			Metrics: NodeMetrics{
				AvgLatencyMs: 45, P50LatencyMs: 30, P95LatencyMs: 120, P99LatencyMs: 250,
				SuccessRate: 99.5, DeliveryCount: 150000, ErrorCount: 750,
			},
			DataResidency: n.residency,
			LastHealthAt:  time.Now(),
			CreatedAt:     time.Now(),
		})
	}
	return edgeNodes
}

// EdgeNetworkHandler provides HTTP handlers for the edge network
type EdgeNetworkHandler struct {
	network *EdgeDeliveryNetwork
}

// NewEdgeNetworkHandler creates a new edge network handler
func NewEdgeNetworkHandler(network *EdgeDeliveryNetwork) *EdgeNetworkHandler {
	return &EdgeNetworkHandler{network: network}
}

// RegisterEdgeRoutes registers edge delivery routes
func (h *EdgeNetworkHandler) RegisterEdgeRoutes(router *gin.RouterGroup) {
	e := router.Group("/edge-network")
	{
		e.GET("/status", h.GetNetworkStatus)
		e.GET("/nodes", h.ListNodes)
		e.POST("/route", h.RouteDelivery)
	}
}

func (h *EdgeNetworkHandler) GetNetworkStatus(c *gin.Context) {
	status := h.network.GetNetworkStatus()
	c.JSON(http.StatusOK, status)
}

func (h *EdgeNetworkHandler) ListNodes(c *gin.Context) {
	h.network.mu.RLock()
	defer h.network.mu.RUnlock()
	var nodes []*EdgeNode
	for _, n := range h.network.nodes {
		nodes = append(nodes, n)
	}
	c.JSON(http.StatusOK, gin.H{"nodes": nodes, "total": len(nodes)})
}

func (h *EdgeNetworkHandler) RouteDelivery(c *gin.Context) {
	var req EdgeDeliveryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httputil.APIErrorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}
	decision, err := h.network.RouteDelivery(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, httputil.APIErrorResponse{Code: "NO_NODES", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, decision)
}
