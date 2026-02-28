package endpointmesh

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/utils"
)

var (
	ErrNodeNotFound = errors.New("mesh node not found")
	ErrCircuitOpen  = errors.New("circuit breaker is open")
	ErrNoFallback   = errors.New("no fallback node configured")
	ErrSelfFallback = errors.New("a node cannot be its own fallback")
)

// Service implements the endpoint mesh business logic.
type Service struct {
	repo   Repository
	config *MeshConfig
	logger *utils.Logger
}

// NewService creates a new endpoint mesh service.
func NewService(repo Repository, config *MeshConfig) *Service {
	if config == nil {
		config = DefaultMeshConfig()
	}
	return &Service{
		repo:   repo,
		config: config,
		logger: utils.NewLogger("endpointmesh-service"),
	}
}

// AddNode adds a new endpoint to the mesh.
func (s *Service) AddNode(tenantID string, req *CreateMeshNodeRequest) (*MeshNode, error) {
	now := time.Now()
	node := &MeshNode{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		EndpointID:  req.EndpointID,
		URL:         req.URL,
		Status:      StatusHealthy,
		HealthScore: 1.0,
		CircuitState: CircuitState{
			State:           CircuitClosed,
			LastEvaluatedAt: now,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.CreateNode(node); err != nil {
		return nil, fmt.Errorf("failed to create node: %w", err)
	}

	s.logger.Info("node added to mesh", map[string]interface{}{
		"node_id":   node.ID,
		"tenant_id": tenantID,
		"url":       req.URL,
	})

	return node, nil
}

// RemoveNode removes an endpoint from the mesh.
func (s *Service) RemoveNode(id string) error {
	if err := s.repo.DeleteNode(id); err != nil {
		return ErrNodeNotFound
	}
	return nil
}

// GetNode returns a single mesh node.
func (s *Service) GetNode(id string) (*MeshNode, error) {
	node, err := s.repo.GetNode(id)
	if err != nil {
		return nil, ErrNodeNotFound
	}
	return node, nil
}

// ListNodes returns all mesh nodes for a tenant.
func (s *Service) ListNodes(tenantID string) ([]*MeshNode, error) {
	return s.repo.ListNodes(tenantID)
}

// SetFallback configures a fallback relationship between two nodes.
func (s *Service) SetFallback(nodeID, fallbackNodeID string) error {
	if nodeID == fallbackNodeID {
		return ErrSelfFallback
	}

	node, err := s.repo.GetNode(nodeID)
	if err != nil {
		return ErrNodeNotFound
	}

	// Verify fallback node exists
	if _, err := s.repo.GetNode(fallbackNodeID); err != nil {
		return fmt.Errorf("fallback node not found: %s", fallbackNodeID)
	}

	node.FallbackNodeID = fallbackNodeID
	node.UpdatedAt = time.Now()
	return s.repo.UpdateNode(node)
}

// RecordHealthCheck updates node health and triggers circuit breaker logic.
func (s *Service) RecordHealthCheck(nodeID string, statusCode int, latencyMs int64, success bool, checkErr string) (*HealthCheck, error) {
	node, err := s.repo.GetNode(nodeID)
	if err != nil {
		return nil, ErrNodeNotFound
	}

	now := time.Now()
	hc := &HealthCheck{
		ID:         uuid.New().String(),
		NodeID:     nodeID,
		StatusCode: statusCode,
		LatencyMs:  latencyMs,
		Success:    success,
		Error:      checkErr,
		CheckedAt:  now,
	}

	if err := s.repo.AppendHealthCheck(hc); err != nil {
		return nil, fmt.Errorf("failed to store health check: %w", err)
	}

	if success {
		s.handleSuccess(node, now)
	} else {
		s.handleFailure(node, now)
	}

	node.UpdatedAt = now
	if err := s.repo.UpdateNode(node); err != nil {
		s.logger.Error("failed to update node", map[string]interface{}{"error": err.Error(), "node_id": nodeID})
	}

	return hc, nil
}

// GetTopology returns the current mesh topology.
func (s *Service) GetTopology(tenantID string) (*MeshTopology, error) {
	nodes, err := s.repo.ListNodes(tenantID)
	if err != nil {
		return nil, err
	}

	var connections []MeshConnection
	healthyCount := 0
	for _, n := range nodes {
		if n.Status == StatusHealthy || n.Status == StatusDegraded {
			healthyCount++
		}
		if n.FallbackNodeID != "" {
			connections = append(connections, MeshConnection{
				SourceID: n.ID,
				TargetID: n.FallbackNodeID,
				Type:     "fallback",
				Active:   n.Status == StatusCircuitOpen || n.Status == StatusUnhealthy,
			})
		}
	}

	return &MeshTopology{
		Nodes:        nodes,
		Connections:  connections,
		HealthyCount: healthyCount,
		TotalCount:   len(nodes),
	}, nil
}

// ListRerouteEvents returns rerouting history for a tenant.
func (s *Service) ListRerouteEvents(tenantID string, limit int) ([]*RerouteEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListRerouteEvents(tenantID, limit)
}

// RouteRequest resolves the best node for delivery, respecting circuit state.
func (s *Service) RouteRequest(nodeID string) (*MeshNode, error) {
	node, err := s.repo.GetNode(nodeID)
	if err != nil {
		return nil, ErrNodeNotFound
	}

	// Evaluate half-open transition
	s.evaluateCircuit(node)

	if node.CircuitState.State == CircuitOpen {
		// Try fallback
		if node.FallbackNodeID == "" {
			return nil, ErrNoFallback
		}
		fallback, err := s.repo.GetNode(node.FallbackNodeID)
		if err != nil {
			return nil, ErrNoFallback
		}
		if fallback.CircuitState.State == CircuitOpen {
			return nil, ErrCircuitOpen
		}

		// Record reroute event
		evt := &RerouteEvent{
			ID:           uuid.New().String(),
			TenantID:     node.TenantID,
			SourceNodeID: node.ID,
			TargetNodeID: fallback.ID,
			Reason:       "circuit_open",
			CreatedAt:    time.Now(),
		}
		if err := s.repo.AppendRerouteEvent(evt); err != nil {
			s.logger.Error("failed to record reroute event", map[string]interface{}{"error": err.Error()})
		}

		return fallback, nil
	}

	return node, nil
}

// RecoverNode attempts to bring a node back online.
func (s *Service) RecoverNode(nodeID string) (*MeshNode, error) {
	node, err := s.repo.GetNode(nodeID)
	if err != nil {
		return nil, ErrNodeNotFound
	}

	now := time.Now()
	node.Status = StatusRecovering
	node.CircuitState.State = CircuitHalfOpen
	node.CircuitState.HalfOpenAt = &now
	node.CircuitState.FailureCount = 0
	node.CircuitState.SuccessCount = 0
	node.CircuitState.LastEvaluatedAt = now
	node.ConsecutiveFailures = 0
	node.UpdatedAt = now

	if err := s.repo.UpdateNode(node); err != nil {
		return nil, fmt.Errorf("failed to update node: %w", err)
	}

	s.logger.Info("node recovery initiated", map[string]interface{}{
		"node_id": nodeID,
		"status":  string(node.Status),
	})

	return node, nil
}

func (s *Service) handleSuccess(node *MeshNode, now time.Time) {
	node.LastSuccessAt = &now
	node.ConsecutiveFailures = 0
	node.CircuitState.SuccessCount++
	node.CircuitState.LastEvaluatedAt = now

	switch node.CircuitState.State {
	case CircuitHalfOpen:
		if node.CircuitState.SuccessCount >= s.config.RecoveryThreshold {
			node.CircuitState.State = CircuitClosed
			node.CircuitState.ClosedAt = &now
			node.CircuitState.FailureCount = 0
			node.Status = StatusHealthy
			node.HealthScore = 1.0
		}
	case CircuitClosed:
		node.Status = StatusHealthy
		node.HealthScore = 1.0
	}
}

func (s *Service) handleFailure(node *MeshNode, now time.Time) {
	node.LastFailureAt = &now
	node.ConsecutiveFailures++
	node.CircuitState.FailureCount++
	node.CircuitState.LastEvaluatedAt = now

	if node.CircuitState.State == CircuitHalfOpen {
		// Any failure in half-open reopens circuit
		node.CircuitState.State = CircuitOpen
		node.CircuitState.OpenedAt = &now
		node.CircuitState.SuccessCount = 0
		node.Status = StatusCircuitOpen
		node.HealthScore = 0
		return
	}

	if node.ConsecutiveFailures >= s.config.FailureThreshold {
		node.CircuitState.State = CircuitOpen
		node.CircuitState.OpenedAt = &now
		node.CircuitState.SuccessCount = 0
		node.Status = StatusCircuitOpen
		node.HealthScore = 0

		s.logger.Warn("circuit opened for node", map[string]interface{}{
			"node_id":  node.ID,
			"failures": node.ConsecutiveFailures,
		})
	} else {
		node.Status = StatusDegraded
		node.HealthScore = 1.0 - float64(node.ConsecutiveFailures)/float64(s.config.FailureThreshold)
	}
}

func (s *Service) evaluateCircuit(node *MeshNode) {
	if node.CircuitState.State != CircuitOpen {
		return
	}
	if node.CircuitState.OpenedAt == nil {
		return
	}
	if time.Since(*node.CircuitState.OpenedAt) >= s.config.CircuitOpenDuration {
		now := time.Now()
		node.CircuitState.State = CircuitHalfOpen
		node.CircuitState.HalfOpenAt = &now
		node.CircuitState.SuccessCount = 0
		node.CircuitState.LastEvaluatedAt = now
		node.Status = StatusRecovering
		_ = s.repo.UpdateNode(node)
	}
}
