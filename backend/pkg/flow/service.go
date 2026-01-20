package flow

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Repository defines the interface for flow storage
type Repository interface {
	CreateFlow(ctx context.Context, flow *Flow) error
	GetFlow(ctx context.Context, tenantID, flowID string) (*Flow, error)
	ListFlows(ctx context.Context, tenantID string, limit, offset int) ([]Flow, int, error)
	UpdateFlow(ctx context.Context, flow *Flow) error
	DeleteFlow(ctx context.Context, tenantID, flowID string) error

	SaveExecution(ctx context.Context, execution *FlowExecution) error
	GetExecution(ctx context.Context, tenantID, executionID string) (*FlowExecution, error)
	ListExecutions(ctx context.Context, tenantID, flowID string, limit, offset int) ([]FlowExecution, int, error)

	AssignFlowToEndpoint(ctx context.Context, assignment *EndpointFlow) error
	GetEndpointFlows(ctx context.Context, endpointID string) ([]EndpointFlow, error)
	RemoveFlowFromEndpoint(ctx context.Context, endpointID, flowID string) error
}

// Service provides flow management functionality
type Service struct {
	repo     Repository
	executor *Executor
}

// NewService creates a new flow service
func NewService(repo Repository) *Service {
	return &Service{
		repo:     repo,
		executor: NewExecutor(),
	}
}

// CreateFlow creates a new flow
func (s *Service) CreateFlow(ctx context.Context, tenantID string, req *CreateFlowRequest) (*Flow, error) {
	// Validate the flow structure
	flow := &Flow{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		Nodes:       req.Nodes,
		Edges:       req.Edges,
		Config:      req.Config,
		IsActive:    true,
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if flow.Config.MaxExecutionTime == 0 {
		flow.Config = DefaultFlowConfig()
	}

	// Validate flow structure
	if err := s.executor.validateFlow(flow); err != nil {
		return nil, err
	}

	if err := s.repo.CreateFlow(ctx, flow); err != nil {
		return nil, fmt.Errorf("failed to create flow: %w", err)
	}

	return flow, nil
}

// GetFlow retrieves a flow by ID
func (s *Service) GetFlow(ctx context.Context, tenantID, flowID string) (*Flow, error) {
	return s.repo.GetFlow(ctx, tenantID, flowID)
}

// ListFlows lists all flows for a tenant
func (s *Service) ListFlows(ctx context.Context, tenantID string, limit, offset int) ([]Flow, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListFlows(ctx, tenantID, limit, offset)
}

// UpdateFlow updates an existing flow
func (s *Service) UpdateFlow(ctx context.Context, tenantID, flowID string, req *UpdateFlowRequest) (*Flow, error) {
	flow, err := s.repo.GetFlow(ctx, tenantID, flowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get flow: %w", err)
	}
	if flow == nil {
		return nil, fmt.Errorf("flow not found")
	}

	if req.Name != "" {
		flow.Name = req.Name
	}
	if req.Description != "" {
		flow.Description = req.Description
	}
	if len(req.Nodes) > 0 {
		flow.Nodes = req.Nodes
	}
	if len(req.Edges) > 0 {
		flow.Edges = req.Edges
	}
	if req.Config.MaxExecutionTime > 0 {
		flow.Config = req.Config
	}
	flow.IsActive = req.IsActive
	flow.Version++
	flow.UpdatedAt = time.Now()

	// Validate updated flow
	if err := s.executor.validateFlow(flow); err != nil {
		return nil, err
	}

	if err := s.repo.UpdateFlow(ctx, flow); err != nil {
		return nil, fmt.Errorf("failed to update flow: %w", err)
	}

	return flow, nil
}

// DeleteFlow deletes a flow
func (s *Service) DeleteFlow(ctx context.Context, tenantID, flowID string) error {
	return s.repo.DeleteFlow(ctx, tenantID, flowID)
}

// ExecuteFlow executes a flow with the given input
func (s *Service) ExecuteFlow(ctx context.Context, tenantID, flowID string, req *ExecuteFlowRequest) (*FlowExecution, error) {
	flow, err := s.repo.GetFlow(ctx, tenantID, flowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get flow: %w", err)
	}
	if flow == nil {
		return nil, fmt.Errorf("flow not found")
	}
	if !flow.IsActive {
		return nil, fmt.Errorf("flow is not active")
	}

	execution, err := s.executor.Execute(ctx, flow, req.Input)
	if err != nil {
		return nil, err
	}

	execution.TenantID = tenantID

	// Save execution (unless dry run)
	if !req.DryRun {
		if err := s.repo.SaveExecution(ctx, execution); err != nil {
			// Log but don't fail
		}
	}

	return execution, nil
}

// GetExecution retrieves an execution by ID
func (s *Service) GetExecution(ctx context.Context, tenantID, executionID string) (*FlowExecution, error) {
	return s.repo.GetExecution(ctx, tenantID, executionID)
}

// ListExecutions lists executions for a flow
func (s *Service) ListExecutions(ctx context.Context, tenantID, flowID string, limit, offset int) ([]FlowExecution, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListExecutions(ctx, tenantID, flowID, limit, offset)
}

// AssignFlowToEndpoint assigns a flow to an endpoint
func (s *Service) AssignFlowToEndpoint(ctx context.Context, tenantID, endpointID, flowID string, priority int) error {
	// Validate flow exists
	flow, err := s.repo.GetFlow(ctx, tenantID, flowID)
	if err != nil {
		return fmt.Errorf("failed to get flow: %w", err)
	}
	if flow == nil {
		return fmt.Errorf("flow not found")
	}

	assignment := &EndpointFlow{
		EndpointID: endpointID,
		FlowID:     flowID,
		Priority:   priority,
		CreatedAt:  time.Now(),
	}

	return s.repo.AssignFlowToEndpoint(ctx, assignment)
}

// GetEndpointFlows gets flows assigned to an endpoint
func (s *Service) GetEndpointFlows(ctx context.Context, endpointID string) ([]EndpointFlow, error) {
	return s.repo.GetEndpointFlows(ctx, endpointID)
}

// RemoveFlowFromEndpoint removes a flow assignment
func (s *Service) RemoveFlowFromEndpoint(ctx context.Context, endpointID, flowID string) error {
	return s.repo.RemoveFlowFromEndpoint(ctx, endpointID, flowID)
}

// GetTemplates returns available flow templates
func (s *Service) GetTemplates() []FlowTemplate {
	return []FlowTemplate{
		{
			ID:          "basic-forward",
			Name:        "Basic Forward",
			Description: "Forward webhook to another endpoint with transformation",
			Category:    "basic",
			Nodes: []Node{
				{ID: "start", Type: NodeTypeStart, Name: "Start", Position: Position{X: 100, Y: 100}},
				{ID: "transform", Type: NodeTypeTransform, Name: "Transform", Position: Position{X: 300, Y: 100}},
				{ID: "http", Type: NodeTypeHTTP, Name: "Send HTTP", Position: Position{X: 500, Y: 100}},
				{ID: "end", Type: NodeTypeEnd, Name: "End", Position: Position{X: 700, Y: 100}},
			},
			Edges: []Edge{
				{ID: "e1", Source: "start", Target: "transform"},
				{ID: "e2", Source: "transform", Target: "http"},
				{ID: "e3", Source: "http", Target: "end"},
			},
			Config: DefaultFlowConfig(),
		},
		{
			ID:          "conditional-routing",
			Name:        "Conditional Routing",
			Description: "Route webhooks based on payload content",
			Category:    "routing",
			Nodes: []Node{
				{ID: "start", Type: NodeTypeStart, Name: "Start", Position: Position{X: 100, Y: 200}},
				{ID: "condition", Type: NodeTypeCondition, Name: "Check Event Type", Position: Position{X: 300, Y: 200}},
				{ID: "http-a", Type: NodeTypeHTTP, Name: "Route A", Position: Position{X: 500, Y: 100}},
				{ID: "http-b", Type: NodeTypeHTTP, Name: "Route B", Position: Position{X: 500, Y: 300}},
				{ID: "end", Type: NodeTypeEnd, Name: "End", Position: Position{X: 700, Y: 200}},
			},
			Edges: []Edge{
				{ID: "e1", Source: "start", Target: "condition"},
				{ID: "e2", Source: "condition", Target: "http-a", Label: "true"},
				{ID: "e3", Source: "condition", Target: "http-b", Label: "false"},
				{ID: "e4", Source: "http-a", Target: "end"},
				{ID: "e5", Source: "http-b", Target: "end"},
			},
			Config: DefaultFlowConfig(),
		},
		{
			ID:          "retry-with-backoff",
			Name:        "Retry with Backoff",
			Description: "Send webhook with retry on failure",
			Category:    "reliability",
			Nodes: []Node{
				{ID: "start", Type: NodeTypeStart, Name: "Start", Position: Position{X: 100, Y: 100}},
				{ID: "retry", Type: NodeTypeRetry, Name: "Retry Config", Position: Position{X: 300, Y: 100}},
				{ID: "http", Type: NodeTypeHTTP, Name: "Send HTTP", Position: Position{X: 500, Y: 100}},
				{ID: "end", Type: NodeTypeEnd, Name: "End", Position: Position{X: 700, Y: 100}},
			},
			Edges: []Edge{
				{ID: "e1", Source: "start", Target: "retry"},
				{ID: "e2", Source: "retry", Target: "http"},
				{ID: "e3", Source: "http", Target: "end"},
			},
			Config: DefaultFlowConfig(),
		},
	}
}

// ValidateFlow validates a flow without creating it
func (s *Service) ValidateFlow(flow *Flow) error {
	return s.executor.validateFlow(flow)
}
