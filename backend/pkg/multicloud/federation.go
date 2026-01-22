// federation.go provides multi-cloud federation capabilities
package multicloud

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrClusterNotFound   = errors.New("cluster not found")
	ErrFederationRouteNotFound = errors.New("federation route not found")
	ErrClusterUnhealthy  = errors.New("cluster unhealthy")
	ErrNoHealthyClusters = errors.New("no healthy clusters available")
	ErrFailoverFailed    = errors.New("failover failed")
)

// ClusterStatus represents cluster health status
type ClusterStatus string

const (
	ClusterHealthy   ClusterStatus = "healthy"
	ClusterDegraded  ClusterStatus = "degraded"
	ClusterUnhealthy ClusterStatus = "unhealthy"
	ClusterDraining  ClusterStatus = "draining"
	ClusterOffline   ClusterStatus = "offline"
)

// RoutingStrategy represents routing strategies
type RoutingStrategy string

const (
	RoutingLatency     RoutingStrategy = "latency"
	RoutingGeo         RoutingStrategy = "geo"
	RoutingRoundRobin  RoutingStrategy = "round_robin"
	RoutingWeighted    RoutingStrategy = "weighted"
	RoutingFailover    RoutingStrategy = "failover"
	RoutingCost        RoutingStrategy = "cost"
)

// FailoverMode represents failover modes
type FailoverMode string

const (
	FailoverAutomatic FailoverMode = "automatic"
	FailoverManual    FailoverMode = "manual"
	FailoverNone      FailoverMode = "none"
)

// FederationCluster represents a federated cloud cluster
type FederationCluster struct {
	ID            string            `json:"id"`
	TenantID      string            `json:"tenant_id"`
	Name          string            `json:"name"`
	Provider      Provider          `json:"provider"`
	Region        string            `json:"region"`
	Zone          string            `json:"zone,omitempty"`
	Endpoint      string            `json:"endpoint"`
	APIEndpoint   string            `json:"api_endpoint"`
	Status        ClusterStatus     `json:"status"`
	Priority      int               `json:"priority"`
	Weight        int               `json:"weight"`
	Capacity      ClusterCapacity   `json:"capacity"`
	Metrics       ClusterMetrics    `json:"metrics"`
	Config        ClusterConfig     `json:"config"`
	Tags          map[string]string `json:"tags,omitempty"`
	LastHealthCheck time.Time       `json:"last_health_check"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

// ClusterCapacity represents cluster capacity
type ClusterCapacity struct {
	MaxRPS            int     `json:"max_rps"`
	CurrentRPS        int     `json:"current_rps"`
	MaxConcurrency    int     `json:"max_concurrency"`
	CurrentConcurrency int    `json:"current_concurrency"`
	CPUUtilization    float64 `json:"cpu_utilization"`
	MemoryUtilization float64 `json:"memory_utilization"`
	Available         bool    `json:"available"`
}

// ClusterMetrics represents cluster metrics
type ClusterMetrics struct {
	LatencyP50       time.Duration `json:"latency_p50"`
	LatencyP99       time.Duration `json:"latency_p99"`
	SuccessRate      float64       `json:"success_rate"`
	ErrorRate        float64       `json:"error_rate"`
	RequestsTotal    int64         `json:"requests_total"`
	RequestsSuccess  int64         `json:"requests_success"`
	RequestsFailed   int64         `json:"requests_failed"`
	BytesTransferred int64         `json:"bytes_transferred"`
	LastUpdated      time.Time     `json:"last_updated"`
}

// ClusterConfig represents cluster configuration
type ClusterConfig struct {
	TimeoutSec       int  `json:"timeout_sec"`
	RetryAttempts    int  `json:"retry_attempts"`
	CircuitBreaker   bool `json:"circuit_breaker"`
	TLSVerify        bool `json:"tls_verify"`
	CompressionLevel int  `json:"compression_level"`
	KeepAlive        bool `json:"keep_alive"`
	MaxIdleConns     int  `json:"max_idle_conns"`
}

// FederationRoute represents a routing rule for federation
type FederationRoute struct {
	ID           string             `json:"id"`
	TenantID     string             `json:"tenant_id"`
	Name         string             `json:"name"`
	Description  string             `json:"description,omitempty"`
	Strategy     RoutingStrategy    `json:"strategy"`
	FailoverMode FailoverMode       `json:"failover_mode"`
	Clusters     []FedRouteCluster  `json:"clusters"`
	Rules        []FedRoutingRule   `json:"rules"`
	HealthCheck  HealthCheckConfig  `json:"health_check"`
	Active       bool               `json:"active"`
	Default      bool               `json:"default"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

// FedRouteCluster represents a cluster in a federation route
type FedRouteCluster struct {
	ClusterID  string `json:"cluster_id"`
	Priority   int    `json:"priority"`
	Weight     int    `json:"weight"`
	MaxTraffic int    `json:"max_traffic"`
	Backup     bool   `json:"backup"`
}

// FedRoutingRule represents a routing rule
type FedRoutingRule struct {
	Name      string              `json:"name"`
	Priority  int                 `json:"priority"`
	Condition FedRoutingCondition `json:"condition"`
	Action    FedRoutingAction    `json:"action"`
	Active    bool                `json:"active"`
}

// FedRoutingCondition represents a routing condition
type FedRoutingCondition struct {
	Type     string `json:"type"`
	Field    string `json:"field,omitempty"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

// FedRoutingAction represents a routing action
type FedRoutingAction struct {
	Type      string         `json:"type"`
	ClusterID string         `json:"cluster_id,omitempty"`
	Split     []FedSplit     `json:"split,omitempty"`
}

// FedSplit represents traffic split
type FedSplit struct {
	ClusterID  string `json:"cluster_id"`
	Percentage int    `json:"percentage"`
}

// HealthCheckConfig represents health check configuration
type HealthCheckConfig struct {
	Enabled      bool   `json:"enabled"`
	IntervalSec  int    `json:"interval_sec"`
	TimeoutSec   int    `json:"timeout_sec"`
	Path         string `json:"path"`
	ExpectedCode int    `json:"expected_code"`
	Threshold    int    `json:"threshold"`
}

// FailoverEvent represents a failover event
type FailoverEvent struct {
	ID            string        `json:"id"`
	TenantID      string        `json:"tenant_id"`
	RouteID       string        `json:"route_id"`
	FromClusterID string        `json:"from_cluster_id"`
	ToClusterID   string        `json:"to_cluster_id"`
	Reason        string        `json:"reason"`
	Automatic     bool          `json:"automatic"`
	Duration      time.Duration `json:"duration"`
	Status        string        `json:"status"`
	InitiatedBy   string        `json:"initiated_by,omitempty"`
	InitiatedAt   time.Time     `json:"initiated_at"`
	CompletedAt   *time.Time    `json:"completed_at,omitempty"`
}

// CloudMetrics represents cross-cloud metrics
type CloudMetrics struct {
	TenantID        string                  `json:"tenant_id"`
	TotalClusters   int                     `json:"total_clusters"`
	HealthyClusters int                     `json:"healthy_clusters"`
	TotalRPS        int                     `json:"total_rps"`
	AvgLatency      time.Duration           `json:"avg_latency"`
	ByProvider      map[Provider]ProviderMetrics `json:"by_provider"`
	ByRegion        map[string]RegionMetrics `json:"by_region"`
	RecentFailovers []FailoverEvent         `json:"recent_failovers"`
	LastUpdated     time.Time               `json:"last_updated"`
}

// ProviderMetrics represents metrics per provider
type ProviderMetrics struct {
	Provider        Provider      `json:"provider"`
	Clusters        int           `json:"clusters"`
	HealthyClusters int           `json:"healthy_clusters"`
	RPS             int           `json:"rps"`
	AvgLatency      time.Duration `json:"avg_latency"`
	SuccessRate     float64       `json:"success_rate"`
	CostEstimate    float64       `json:"cost_estimate"`
}

// RegionMetrics represents metrics per region
type RegionMetrics struct {
	Region          string        `json:"region"`
	Provider        Provider      `json:"provider"`
	Clusters        int           `json:"clusters"`
	HealthyClusters int           `json:"healthy_clusters"`
	RPS             int           `json:"rps"`
	AvgLatency      time.Duration `json:"avg_latency"`
}

// FederationRepository defines the interface for federation data storage
type FederationRepository interface {
	CreateCluster(ctx context.Context, cluster *FederationCluster) error
	GetCluster(ctx context.Context, clusterID string) (*FederationCluster, error)
	UpdateCluster(ctx context.Context, cluster *FederationCluster) error
	DeleteCluster(ctx context.Context, clusterID string) error
	ListClusters(ctx context.Context, tenantID string, provider *Provider) ([]FederationCluster, error)
	ListHealthyClusters(ctx context.Context, tenantID string) ([]FederationCluster, error)

	CreateFederationRoute(ctx context.Context, route *FederationRoute) error
	GetFederationRoute(ctx context.Context, routeID string) (*FederationRoute, error)
	GetDefaultFederationRoute(ctx context.Context, tenantID string) (*FederationRoute, error)
	UpdateFederationRoute(ctx context.Context, route *FederationRoute) error
	DeleteFederationRoute(ctx context.Context, routeID string) error
	ListFederationRoutes(ctx context.Context, tenantID string) ([]FederationRoute, error)

	CreateFailoverEvent(ctx context.Context, event *FailoverEvent) error
	UpdateFailoverEvent(ctx context.Context, event *FailoverEvent) error
	ListFailoverEvents(ctx context.Context, tenantID string, since time.Time) ([]FailoverEvent, error)
}

// FederationClient defines the interface for cluster communication
type FederationClient interface {
	HealthCheck(ctx context.Context, cluster *FederationCluster) (*ClusterStatus, error)
	Forward(ctx context.Context, cluster *FederationCluster, request *ForwardRequest) (*ForwardResponse, error)
	GetMetrics(ctx context.Context, cluster *FederationCluster) (*ClusterMetrics, error)
}

// ForwardRequest represents a request to forward
type ForwardRequest struct {
	ID      string            `json:"id"`
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers"`
	Body    []byte            `json:"body"`
	Timeout time.Duration     `json:"timeout"`
}

// ForwardResponse represents a forwarded response
type ForwardResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body"`
	Latency    time.Duration     `json:"latency"`
	ClusterID  string            `json:"cluster_id"`
}

// FederationService provides multi-cloud federation operations
type FederationService struct {
	repo   FederationRepository
	client FederationClient
	config *FederationConfig
}

// FederationConfig holds service configuration
type FederationConfig struct {
	HealthCheckIntervalSec int
	FailoverThreshold      int
	MaxFailoverRetries     int
	DefaultTimeoutSec      int
	EnableAutoFailover     bool
}

// DefaultFederationConfig returns default configuration
func DefaultFederationConfig() *FederationConfig {
	return &FederationConfig{
		HealthCheckIntervalSec: 30,
		FailoverThreshold:      3,
		MaxFailoverRetries:     3,
		DefaultTimeoutSec:      30,
		EnableAutoFailover:     true,
	}
}

// NewFederationService creates a new federation service
func NewFederationService(repo FederationRepository, client FederationClient, config *FederationConfig) *FederationService {
	if config == nil {
		config = DefaultFederationConfig()
	}
	return &FederationService{
		repo:   repo,
		client: client,
		config: config,
	}
}

// RegisterCluster registers a new cluster
func (s *FederationService) RegisterCluster(ctx context.Context, tenantID string, cluster *FederationCluster) (*FederationCluster, error) {
	cluster.ID = uuid.New().String()
	cluster.TenantID = tenantID
	cluster.Status = ClusterHealthy
	cluster.CreatedAt = time.Now()
	cluster.UpdatedAt = time.Now()

	if cluster.Priority == 0 {
		cluster.Priority = 100
	}
	if cluster.Weight == 0 {
		cluster.Weight = 100
	}
	if cluster.Config.TimeoutSec == 0 {
		cluster.Config.TimeoutSec = s.config.DefaultTimeoutSec
	}
	if cluster.Config.RetryAttempts == 0 {
		cluster.Config.RetryAttempts = 3
	}

	if s.client != nil {
		status, err := s.client.HealthCheck(ctx, cluster)
		if err == nil && status != nil {
			cluster.Status = *status
		}
	}
	cluster.LastHealthCheck = time.Now()

	if err := s.repo.CreateCluster(ctx, cluster); err != nil {
		return nil, err
	}

	return cluster, nil
}

// GetCluster retrieves a cluster
func (s *FederationService) GetCluster(ctx context.Context, clusterID string) (*FederationCluster, error) {
	return s.repo.GetCluster(ctx, clusterID)
}

// UpdateCluster updates a cluster
func (s *FederationService) UpdateCluster(ctx context.Context, cluster *FederationCluster) error {
	cluster.UpdatedAt = time.Now()
	return s.repo.UpdateCluster(ctx, cluster)
}

// ListClusters lists clusters for a tenant
func (s *FederationService) ListClusters(ctx context.Context, tenantID string, provider *Provider) ([]FederationCluster, error) {
	return s.repo.ListClusters(ctx, tenantID, provider)
}

// CreateFederationRoute creates a routing rule
func (s *FederationService) CreateFederationRoute(ctx context.Context, tenantID string, route *FederationRoute) (*FederationRoute, error) {
	route.ID = uuid.New().String()
	route.TenantID = tenantID
	route.Active = true
	route.CreatedAt = time.Now()
	route.UpdatedAt = time.Now()

	if route.HealthCheck.IntervalSec == 0 {
		route.HealthCheck = HealthCheckConfig{
			Enabled:      true,
			IntervalSec:  s.config.HealthCheckIntervalSec,
			TimeoutSec:   5,
			Path:         "/health",
			ExpectedCode: 200,
			Threshold:    s.config.FailoverThreshold,
		}
	}

	if err := s.repo.CreateFederationRoute(ctx, route); err != nil {
		return nil, err
	}

	return route, nil
}

// GetFederationRoute retrieves a route
func (s *FederationService) GetFederationRoute(ctx context.Context, routeID string) (*FederationRoute, error) {
	return s.repo.GetFederationRoute(ctx, routeID)
}

// ListFederationRoutes lists routes for a tenant
func (s *FederationService) ListFederationRoutes(ctx context.Context, tenantID string) ([]FederationRoute, error) {
	return s.repo.ListFederationRoutes(ctx, tenantID)
}

// RouteRequest routes a request to the best cluster
func (s *FederationService) RouteRequest(ctx context.Context, tenantID string, request *ForwardRequest) (*ForwardResponse, error) {
	route, err := s.repo.GetDefaultFederationRoute(ctx, tenantID)
	if err != nil {
		clusters, err := s.repo.ListHealthyClusters(ctx, tenantID)
		if err != nil || len(clusters) == 0 {
			return nil, ErrNoHealthyClusters
		}
		return s.forwardToCluster(ctx, &clusters[0], request)
	}

	return s.routeByStrategy(ctx, route, request)
}

// routeByStrategy routes based on the route's strategy
func (s *FederationService) routeByStrategy(ctx context.Context, route *FederationRoute, request *ForwardRequest) (*ForwardResponse, error) {
	switch route.Strategy {
	case RoutingLatency:
		return s.routeByLatency(ctx, route, request)
	case RoutingGeo:
		return s.routeByGeo(ctx, route, request)
	case RoutingRoundRobin:
		return s.routeRoundRobin(ctx, route, request)
	case RoutingWeighted:
		return s.routeWeighted(ctx, route, request)
	case RoutingFailover:
		return s.routeWithFailover(ctx, route, request)
	case RoutingCost:
		return s.routeByCost(ctx, route, request)
	default:
		return s.routeByLatency(ctx, route, request)
	}
}

// routeByLatency routes to lowest latency cluster
func (s *FederationService) routeByLatency(ctx context.Context, route *FederationRoute, request *ForwardRequest) (*ForwardResponse, error) {
	var bestCluster *FederationCluster
	var bestLatency time.Duration = time.Hour

	for _, rc := range route.Clusters {
		cluster, err := s.repo.GetCluster(ctx, rc.ClusterID)
		if err != nil || cluster.Status != ClusterHealthy {
			continue
		}
		if cluster.Metrics.LatencyP50 < bestLatency {
			bestLatency = cluster.Metrics.LatencyP50
			bestCluster = cluster
		}
	}

	if bestCluster == nil {
		return nil, ErrNoHealthyClusters
	}

	return s.forwardToCluster(ctx, bestCluster, request)
}

// routeByGeo routes to geographically closest cluster
func (s *FederationService) routeByGeo(ctx context.Context, route *FederationRoute, request *ForwardRequest) (*ForwardResponse, error) {
	clientRegion := request.Headers["X-Client-Region"]
	if clientRegion == "" {
		clientRegion = "us-east-1"
	}

	for _, rc := range route.Clusters {
		cluster, err := s.repo.GetCluster(ctx, rc.ClusterID)
		if err != nil || cluster.Status != ClusterHealthy {
			continue
		}
		if cluster.Region == clientRegion {
			return s.forwardToCluster(ctx, cluster, request)
		}
	}

	return s.routeByLatency(ctx, route, request)
}

// routeRoundRobin routes using round-robin
func (s *FederationService) routeRoundRobin(ctx context.Context, route *FederationRoute, request *ForwardRequest) (*ForwardResponse, error) {
	healthyClusters := make([]*FederationCluster, 0)
	for _, rc := range route.Clusters {
		cluster, err := s.repo.GetCluster(ctx, rc.ClusterID)
		if err != nil || cluster.Status != ClusterHealthy {
			continue
		}
		healthyClusters = append(healthyClusters, cluster)
	}

	if len(healthyClusters) == 0 {
		return nil, ErrNoHealthyClusters
	}

	idx := hashString(request.ID) % len(healthyClusters)
	return s.forwardToCluster(ctx, healthyClusters[idx], request)
}

// routeWeighted routes using weighted distribution
func (s *FederationService) routeWeighted(ctx context.Context, route *FederationRoute, request *ForwardRequest) (*ForwardResponse, error) {
	totalWeight := 0
	for _, rc := range route.Clusters {
		cluster, err := s.repo.GetCluster(ctx, rc.ClusterID)
		if err != nil || cluster.Status != ClusterHealthy {
			continue
		}
		totalWeight += rc.Weight
	}

	if totalWeight == 0 {
		return nil, ErrNoHealthyClusters
	}

	target := hashString(request.ID) % totalWeight
	current := 0

	for _, rc := range route.Clusters {
		cluster, err := s.repo.GetCluster(ctx, rc.ClusterID)
		if err != nil || cluster.Status != ClusterHealthy {
			continue
		}
		current += rc.Weight
		if current > target {
			return s.forwardToCluster(ctx, cluster, request)
		}
	}

	return nil, ErrNoHealthyClusters
}

// routeWithFailover routes with automatic failover
func (s *FederationService) routeWithFailover(ctx context.Context, route *FederationRoute, request *ForwardRequest) (*ForwardResponse, error) {
	for _, rc := range route.Clusters {
		if rc.Backup {
			continue
		}
		cluster, err := s.repo.GetCluster(ctx, rc.ClusterID)
		if err != nil || cluster.Status != ClusterHealthy {
			continue
		}

		resp, err := s.forwardToCluster(ctx, cluster, request)
		if err == nil {
			return resp, nil
		}
	}

	for _, rc := range route.Clusters {
		if !rc.Backup {
			continue
		}
		cluster, err := s.repo.GetCluster(ctx, rc.ClusterID)
		if err != nil || cluster.Status != ClusterHealthy {
			continue
		}

		return s.forwardToCluster(ctx, cluster, request)
	}

	return nil, ErrFailoverFailed
}

// routeByCost routes to lowest cost cluster
func (s *FederationService) routeByCost(ctx context.Context, route *FederationRoute, request *ForwardRequest) (*ForwardResponse, error) {
	costOrder := map[Provider]int{
		ProviderCustom:         1,
		ProviderGCPPubSub:      2,
		ProviderAWSEventBridge: 3,
		ProviderAzureEventGrid: 4,
	}

	var bestCluster *FederationCluster
	bestCost := 999

	for _, rc := range route.Clusters {
		cluster, err := s.repo.GetCluster(ctx, rc.ClusterID)
		if err != nil || cluster.Status != ClusterHealthy {
			continue
		}
		cost := costOrder[cluster.Provider]
		if cost < bestCost {
			bestCost = cost
			bestCluster = cluster
		}
	}

	if bestCluster == nil {
		return nil, ErrNoHealthyClusters
	}

	return s.forwardToCluster(ctx, bestCluster, request)
}

// forwardToCluster forwards request to a cluster
func (s *FederationService) forwardToCluster(ctx context.Context, cluster *FederationCluster, request *ForwardRequest) (*ForwardResponse, error) {
	if s.client == nil {
		return nil, errors.New("federation client not configured")
	}
	return s.client.Forward(ctx, cluster, request)
}

// InitiateFailover manually initiates failover
func (s *FederationService) InitiateFailover(ctx context.Context, tenantID, routeID, fromClusterID, toClusterID, reason, initiatedBy string) (*FailoverEvent, error) {
	event := &FailoverEvent{
		ID:            uuid.New().String(),
		TenantID:      tenantID,
		RouteID:       routeID,
		FromClusterID: fromClusterID,
		ToClusterID:   toClusterID,
		Reason:        reason,
		Automatic:     false,
		Status:        "initiated",
		InitiatedBy:   initiatedBy,
		InitiatedAt:   time.Now(),
	}

	if err := s.repo.CreateFailoverEvent(ctx, event); err != nil {
		return nil, err
	}

	fromCluster, _ := s.repo.GetCluster(ctx, fromClusterID)
	if fromCluster != nil {
		fromCluster.Status = ClusterDraining
		_ = s.repo.UpdateCluster(ctx, fromCluster)
	}

	now := time.Now()
	event.Status = "completed"
	event.CompletedAt = &now
	event.Duration = now.Sub(event.InitiatedAt)
	_ = s.repo.UpdateFailoverEvent(ctx, event)

	return event, nil
}

// CheckClusterHealth checks health of all clusters
func (s *FederationService) CheckClusterHealth(ctx context.Context, tenantID string) ([]FederationCluster, error) {
	clusters, err := s.repo.ListClusters(ctx, tenantID, nil)
	if err != nil {
		return nil, err
	}

	for i := range clusters {
		if s.client != nil {
			status, err := s.client.HealthCheck(ctx, &clusters[i])
			if err == nil && status != nil {
				clusters[i].Status = *status
			} else {
				clusters[i].Status = ClusterUnhealthy
			}
		}
		clusters[i].LastHealthCheck = time.Now()
		_ = s.repo.UpdateCluster(ctx, &clusters[i])
	}

	return clusters, nil
}

// GetCloudMetrics retrieves cross-cloud metrics
func (s *FederationService) GetCloudMetrics(ctx context.Context, tenantID string) (*CloudMetrics, error) {
	clusters, err := s.repo.ListClusters(ctx, tenantID, nil)
	if err != nil {
		return nil, err
	}

	metrics := &CloudMetrics{
		TenantID:    tenantID,
		ByProvider:  make(map[Provider]ProviderMetrics),
		ByRegion:    make(map[string]RegionMetrics),
		LastUpdated: time.Now(),
	}

	for _, cluster := range clusters {
		metrics.TotalClusters++
		if cluster.Status == ClusterHealthy {
			metrics.HealthyClusters++
		}
		metrics.TotalRPS += cluster.Capacity.CurrentRPS
		metrics.AvgLatency += cluster.Metrics.LatencyP50

		pm := metrics.ByProvider[cluster.Provider]
		pm.Provider = cluster.Provider
		pm.Clusters++
		if cluster.Status == ClusterHealthy {
			pm.HealthyClusters++
		}
		pm.RPS += cluster.Capacity.CurrentRPS
		pm.AvgLatency += cluster.Metrics.LatencyP50
		pm.SuccessRate += cluster.Metrics.SuccessRate
		metrics.ByProvider[cluster.Provider] = pm

		rm := metrics.ByRegion[cluster.Region]
		rm.Region = cluster.Region
		rm.Provider = cluster.Provider
		rm.Clusters++
		if cluster.Status == ClusterHealthy {
			rm.HealthyClusters++
		}
		rm.RPS += cluster.Capacity.CurrentRPS
		rm.AvgLatency += cluster.Metrics.LatencyP50
		metrics.ByRegion[cluster.Region] = rm
	}

	if metrics.TotalClusters > 0 {
		metrics.AvgLatency /= time.Duration(metrics.TotalClusters)
	}
	for provider, pm := range metrics.ByProvider {
		if pm.Clusters > 0 {
			pm.AvgLatency /= time.Duration(pm.Clusters)
			pm.SuccessRate /= float64(pm.Clusters)
			metrics.ByProvider[provider] = pm
		}
	}
	for region, rm := range metrics.ByRegion {
		if rm.Clusters > 0 {
			rm.AvgLatency /= time.Duration(rm.Clusters)
			metrics.ByRegion[region] = rm
		}
	}

	failovers, _ := s.repo.ListFailoverEvents(ctx, tenantID, time.Now().AddDate(0, 0, -7))
	metrics.RecentFailovers = failovers

	return metrics, nil
}

// hashString provides simple hash for load balancing
func hashString(s string) int {
	h := 0
	for _, c := range s {
		h = 31*h + int(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}
