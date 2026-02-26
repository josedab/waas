package depgraph

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/utils"
)

// ServiceConfig configures the dependency graph service.
type ServiceConfig struct {
	MaxGraphDepth       int
	HealthThresholdGood float64
	HealthThresholdWarn float64
	AutoRefreshInterval time.Duration
}

// DefaultServiceConfig returns sensible defaults.
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		MaxGraphDepth:       10,
		HealthThresholdGood: 95.0,
		HealthThresholdWarn: 80.0,
		AutoRefreshInterval: 5 * time.Minute,
	}
}

// Service implements the dependency graph business logic.
type Service struct {
	repo   Repository
	config *ServiceConfig
	logger *utils.Logger
}

// NewService creates a new dependency graph service.
func NewService(repo Repository, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}
	return &Service{repo: repo, config: config, logger: utils.NewLogger("depgraph-service")}
}

// RecordDelivery ingests a delivery event to update the dependency graph.
func (s *Service) RecordDelivery(tenantID, producerID, consumerID, eventType string, success bool, latencyMs float64) error {
	if producerID == "" || consumerID == "" {
		return fmt.Errorf("producer_id and consumer_id are required")
	}

	depID := producerID + "→" + consumerID
	dep, err := s.repo.GetDependency(depID)
	if err != nil {
		dep = &Dependency{
			ID:           depID,
			TenantID:     tenantID,
			ProducerID:   producerID,
			ConsumerID:   consumerID,
			EventTypes:   []string{eventType},
			DiscoveredAt: time.Now(),
		}
	}

	dep.DeliveryCount++
	dep.LastDeliveryAt = time.Now()
	dep.LastRefreshedAt = time.Now()

	// Update rolling average latency
	dep.AvgLatencyMs = (dep.AvgLatencyMs*float64(dep.DeliveryCount-1) + latencyMs) / float64(dep.DeliveryCount)

	// Update success rate
	if success {
		dep.SuccessRate = (dep.SuccessRate*float64(dep.DeliveryCount-1) + 100.0) / float64(dep.DeliveryCount)
	} else {
		dep.SuccessRate = dep.SuccessRate * float64(dep.DeliveryCount-1) / float64(dep.DeliveryCount)
	}

	// Compute health status
	dep.HealthStatus = s.computeHealthStatus(dep.SuccessRate)

	// Add event type if new
	found := false
	for _, et := range dep.EventTypes {
		if et == eventType {
			found = true
			break
		}
	}
	if !found && eventType != "" {
		dep.EventTypes = append(dep.EventTypes, eventType)
	}

	if err := s.repo.UpsertDependency(dep); err != nil {
		return fmt.Errorf("failed to upsert dependency: %w", err)
	}

	// Update endpoint nodes
	s.ensureNode(tenantID, producerID, "producer")
	s.ensureNode(tenantID, consumerID, "consumer")

	return nil
}

// GetGraph returns the full dependency graph for a tenant.
func (s *Service) GetGraph(tenantID string) (*Graph, error) {
	nodes, err := s.repo.ListEndpointNodes(tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	deps, err := s.repo.ListDependencies(tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list dependencies: %w", err)
	}

	graph := &Graph{
		Nodes: make([]EndpointNode, 0, len(nodes)),
		Edges: make([]Dependency, 0, len(deps)),
	}

	for _, n := range nodes {
		// Compute in/out degree
		n.InDegree = 0
		n.OutDegree = 0
		for _, d := range deps {
			if d.ConsumerID == n.ID {
				n.InDegree++
			}
			if d.ProducerID == n.ID {
				n.OutDegree++
			}
		}
		graph.Nodes = append(graph.Nodes, *n)
	}

	for _, d := range deps {
		graph.Edges = append(graph.Edges, *d)
	}

	return graph, nil
}

// AnalyzeImpact computes the blast radius for an endpoint failure.
func (s *Service) AnalyzeImpact(endpointID string) (*ImpactAnalysis, error) {
	if endpointID == "" {
		return nil, fmt.Errorf("endpoint_id is required")
	}

	// Get direct consumers
	directDeps, err := s.repo.GetConsumers(endpointID)
	if err != nil {
		return nil, fmt.Errorf("failed to get consumers: %w", err)
	}

	var directConsumers []EndpointNode
	for _, d := range directDeps {
		node, err := s.repo.GetEndpointNode(d.ConsumerID)
		if err == nil {
			directConsumers = append(directConsumers, *node)
		}
	}

	// Compute transitive closure via BFS
	visited := make(map[string]bool)
	visited[endpointID] = true
	queue := []string{endpointID}
	var transitiveClosure []EndpointNode
	affectedEvents := make(map[string]bool)

	depth := 0
	for len(queue) > 0 && depth < s.config.MaxGraphDepth {
		nextQueue := []string{}
		for _, current := range queue {
			consumers, err := s.repo.GetConsumers(current)
			if err != nil {
				continue
			}
			for _, dep := range consumers {
				if visited[dep.ConsumerID] {
					continue
				}
				visited[dep.ConsumerID] = true
				nextQueue = append(nextQueue, dep.ConsumerID)

				node, err := s.repo.GetEndpointNode(dep.ConsumerID)
				if err == nil {
					transitiveClosure = append(transitiveClosure, *node)
				}
				for _, et := range dep.EventTypes {
					affectedEvents[et] = true
				}
			}
		}
		queue = nextQueue
		depth++
	}

	// Compute risk level
	blastRadius := len(transitiveClosure)
	riskLevel := "low"
	if blastRadius > 10 {
		riskLevel = "critical"
	} else if blastRadius > 5 {
		riskLevel = "high"
	} else if blastRadius > 2 {
		riskLevel = "medium"
	}

	var events []string
	for e := range affectedEvents {
		events = append(events, e)
	}

	recommendations := s.generateRecommendations(blastRadius, riskLevel)

	return &ImpactAnalysis{
		EndpointID:        endpointID,
		DirectConsumers:   directConsumers,
		TransitiveClosure: transitiveClosure,
		BlastRadius:       blastRadius,
		AffectedEvents:    events,
		RiskLevel:         riskLevel,
		Recommendations:   recommendations,
	}, nil
}

// AddDependency manually adds a dependency edge.
func (s *Service) AddDependency(tenantID, producerID, consumerID string, eventTypes []string) (*Dependency, error) {
	if producerID == "" || consumerID == "" {
		return nil, fmt.Errorf("producer_id and consumer_id are required")
	}
	if producerID == consumerID {
		return nil, fmt.Errorf("self-referencing dependency not allowed")
	}

	dep := &Dependency{
		ID:              uuid.New().String(),
		TenantID:        tenantID,
		ProducerID:      producerID,
		ConsumerID:      consumerID,
		EventTypes:      eventTypes,
		HealthStatus:    "unknown",
		DiscoveredAt:    time.Now(),
		LastRefreshedAt: time.Now(),
	}

	if err := s.repo.UpsertDependency(dep); err != nil {
		return nil, fmt.Errorf("failed to create dependency: %w", err)
	}

	s.ensureNode(tenantID, producerID, "producer")
	s.ensureNode(tenantID, consumerID, "consumer")

	return dep, nil
}

func (s *Service) ensureNode(tenantID, endpointID, nodeType string) {
	existing, err := s.repo.GetEndpointNode(endpointID)
	if err != nil {
		node := &EndpointNode{
			ID:           endpointID,
			TenantID:     tenantID,
			Type:         nodeType,
			HealthStatus: "unknown",
		}
		if err := s.repo.UpsertEndpointNode(node); err != nil {
			s.logger.Error("failed to upsert endpoint node", map[string]interface{}{"error": err.Error(), "endpoint_id": endpointID})
		}
		return
	}
	if existing.Type != nodeType && existing.Type != "both" {
		existing.Type = "both"
		if err := s.repo.UpsertEndpointNode(existing); err != nil {
			s.logger.Error("failed to upsert endpoint node", map[string]interface{}{"error": err.Error(), "endpoint_id": endpointID})
		}
	}
}

func (s *Service) computeHealthStatus(successRate float64) string {
	if successRate >= s.config.HealthThresholdGood {
		return "healthy"
	} else if successRate >= s.config.HealthThresholdWarn {
		return "degraded"
	}
	return "critical"
}

func (s *Service) generateRecommendations(blastRadius int, riskLevel string) []string {
	var recs []string
	if riskLevel == "critical" || riskLevel == "high" {
		recs = append(recs, "Consider adding circuit breakers to limit cascade failures")
		recs = append(recs, "Set up alerting for this endpoint's health score")
	}
	if blastRadius > 5 {
		recs = append(recs, "Review whether all transitive dependencies are necessary")
		recs = append(recs, "Consider adding a dead letter queue for critical paths")
	}
	if blastRadius > 0 {
		recs = append(recs, "Ensure retry policies are configured for downstream consumers")
	}
	return recs
}
