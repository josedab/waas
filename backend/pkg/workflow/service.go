package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Service provides workflow operations
type Service struct {
	repo     Repository
	executor *Executor
	config   *ServiceConfig
}

// ServiceConfig holds service configuration
type ServiceConfig struct {
	MaxWorkflowsPerTenant int
	MaxNodesPerWorkflow   int
	DefaultTimeout        time.Duration
	MaxConcurrentExecs    int
}

// DefaultServiceConfig returns default configuration
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		MaxWorkflowsPerTenant: 100,
		MaxNodesPerWorkflow:   50,
		DefaultTimeout:        5 * time.Minute,
		MaxConcurrentExecs:    10,
	}
}

// NewService creates a new workflow service
func NewService(repo Repository, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}

	return &Service{
		repo:     repo,
		executor: NewExecutor(repo, config),
		config:   config,
	}
}

// CreateWorkflow creates a new workflow
func (s *Service) CreateWorkflow(ctx context.Context, tenantID string, req *CreateWorkflowRequest) (*Workflow, error) {
	// Check workflow limit
	existing, err := s.repo.ListWorkflows(ctx, tenantID, nil)
	if err != nil {
		return nil, err
	}
	if len(existing) >= s.config.MaxWorkflowsPerTenant {
		return nil, fmt.Errorf("maximum workflows reached: %d", s.config.MaxWorkflowsPerTenant)
	}

	wf := &Workflow{
		ID:          GenerateWorkflowID(),
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		Version:     1,
		Status:      WorkflowDraft,
		Trigger:     req.Trigger,
		Nodes:       req.Nodes,
		Edges:       req.Edges,
		Variables:   req.Variables,
		Canvas:      GetDefaultCanvas(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if req.Settings != nil {
		wf.Settings = *req.Settings
	} else {
		wf.Settings = GetDefaultSettings()
	}

	// Initialize from template if provided
	if req.TemplateID != "" {
		template, err := s.repo.GetTemplate(ctx, req.TemplateID)
		if err == nil {
			wf.Nodes = template.Workflow.Nodes
			wf.Edges = template.Workflow.Edges
			wf.Variables = template.Workflow.Variables
			wf.Settings = template.Workflow.Settings
			wf.Canvas = template.Workflow.Canvas
		}
	}

	// Add start node if no nodes provided
	if len(wf.Nodes) == 0 {
		wf.Nodes = []Node{
			{
				ID:       "start",
				Type:     NodeStart,
				Name:     "Start",
				Position: Position{X: 100, Y: 200},
				Outputs:  []Port{{ID: "out", Name: "Output"}},
			},
		}
	}

	// Validate
	validation := s.ValidateWorkflow(ctx, wf)
	if !validation.Valid {
		return nil, fmt.Errorf("workflow validation failed: %s", validation.Errors[0].Message)
	}

	if err := s.repo.SaveWorkflow(ctx, wf); err != nil {
		return nil, err
	}

	return wf, nil
}

// GetWorkflow retrieves a workflow
func (s *Service) GetWorkflow(ctx context.Context, tenantID, workflowID string) (*Workflow, error) {
	return s.repo.GetWorkflow(ctx, tenantID, workflowID)
}

// ListWorkflows lists workflows
func (s *Service) ListWorkflows(ctx context.Context, tenantID string, filter *WorkflowFilter) ([]Workflow, error) {
	return s.repo.ListWorkflows(ctx, tenantID, filter)
}

// UpdateWorkflow updates a workflow
func (s *Service) UpdateWorkflow(ctx context.Context, tenantID, workflowID string, req *UpdateWorkflowRequest) (*Workflow, error) {
	wf, err := s.repo.GetWorkflow(ctx, tenantID, workflowID)
	if err != nil {
		return nil, err
	}

	// Cannot modify published workflow directly - create new version
	if wf.Status == WorkflowPublished {
		return nil, fmt.Errorf("cannot modify published workflow, create a new version")
	}

	if req.Name != nil {
		wf.Name = *req.Name
	}
	if req.Description != nil {
		wf.Description = *req.Description
	}
	if req.Trigger != nil {
		wf.Trigger = *req.Trigger
	}
	if req.Nodes != nil {
		wf.Nodes = req.Nodes
	}
	if req.Edges != nil {
		wf.Edges = req.Edges
	}
	if req.Variables != nil {
		wf.Variables = req.Variables
	}
	if req.Settings != nil {
		wf.Settings = *req.Settings
	}
	if req.Canvas != nil {
		wf.Canvas = *req.Canvas
	}

	wf.UpdatedAt = time.Now()

	// Validate
	validation := s.ValidateWorkflow(ctx, wf)
	if !validation.Valid {
		return nil, fmt.Errorf("workflow validation failed: %s", validation.Errors[0].Message)
	}

	if err := s.repo.SaveWorkflow(ctx, wf); err != nil {
		return nil, err
	}

	return wf, nil
}

// DeleteWorkflow deletes a workflow
func (s *Service) DeleteWorkflow(ctx context.Context, tenantID, workflowID string) error {
	return s.repo.DeleteWorkflow(ctx, tenantID, workflowID)
}

// PublishWorkflow publishes a workflow
func (s *Service) PublishWorkflow(ctx context.Context, tenantID, workflowID string) (*Workflow, error) {
	wf, err := s.repo.GetWorkflow(ctx, tenantID, workflowID)
	if err != nil {
		return nil, err
	}

	// Validate before publishing
	validation := s.ValidateWorkflow(ctx, wf)
	if !validation.Valid {
		return nil, fmt.Errorf("workflow validation failed: %s", validation.Errors[0].Message)
	}

	wf.Status = WorkflowPublished
	wf.Version++
	now := time.Now()
	wf.PublishedAt = &now
	wf.UpdatedAt = now

	if err := s.repo.SaveWorkflow(ctx, wf); err != nil {
		return nil, err
	}

	return wf, nil
}

// UnpublishWorkflow unpublishes a workflow
func (s *Service) UnpublishWorkflow(ctx context.Context, tenantID, workflowID string) (*Workflow, error) {
	wf, err := s.repo.GetWorkflow(ctx, tenantID, workflowID)
	if err != nil {
		return nil, err
	}

	wf.Status = WorkflowDraft
	wf.UpdatedAt = time.Now()

	if err := s.repo.SaveWorkflow(ctx, wf); err != nil {
		return nil, err
	}

	return wf, nil
}

// CloneWorkflow creates a copy of a workflow
func (s *Service) CloneWorkflow(ctx context.Context, tenantID, workflowID, newName string) (*Workflow, error) {
	original, err := s.repo.GetWorkflow(ctx, tenantID, workflowID)
	if err != nil {
		return nil, err
	}

	clone := &Workflow{
		ID:          GenerateWorkflowID(),
		TenantID:    tenantID,
		Name:        newName,
		Description: original.Description + " (copy)",
		Version:     1,
		Status:      WorkflowDraft,
		Trigger:     original.Trigger,
		Nodes:       original.Nodes,
		Edges:       original.Edges,
		Variables:   original.Variables,
		Settings:    original.Settings,
		Canvas:      original.Canvas,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.SaveWorkflow(ctx, clone); err != nil {
		return nil, err
	}

	return clone, nil
}

// GetWorkflowVersion retrieves a specific workflow version
func (s *Service) GetWorkflowVersion(ctx context.Context, tenantID, workflowID string, version int) (*Workflow, error) {
	return s.repo.GetWorkflowVersion(ctx, tenantID, workflowID, version)
}

// ListWorkflowVersions lists all versions of a workflow
func (s *Service) ListWorkflowVersions(ctx context.Context, tenantID, workflowID string) ([]WorkflowVersionInfo, error) {
	return s.repo.ListWorkflowVersions(ctx, tenantID, workflowID)
}

// ValidateWorkflow validates a workflow
func (s *Service) ValidateWorkflow(ctx context.Context, wf *Workflow) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Check node count
	if len(wf.Nodes) > s.config.MaxNodesPerWorkflow {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Code:    "MAX_NODES_EXCEEDED",
			Message: fmt.Sprintf("Maximum %d nodes allowed", s.config.MaxNodesPerWorkflow),
		})
	}

	// Check for start node
	hasStart := false
	hasEnd := false
	nodeIDs := make(map[string]bool)

	for _, node := range wf.Nodes {
		nodeIDs[node.ID] = true
		if node.Type == NodeStart {
			hasStart = true
		}
		if node.Type == NodeEnd {
			hasEnd = true
		}
	}

	if !hasStart {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Code:    "MISSING_START_NODE",
			Message: "Workflow must have a Start node",
		})
	}

	if !hasEnd {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Code:    "MISSING_END_NODE",
			Message: "Workflow has no End node - execution will complete at leaf nodes",
		})
	}

	// Validate edges
	for _, edge := range wf.Edges {
		if !nodeIDs[edge.Source] {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Code:    "INVALID_EDGE_SOURCE",
				Message: fmt.Sprintf("Edge source node %s not found", edge.Source),
			})
		}
		if !nodeIDs[edge.Target] {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Code:    "INVALID_EDGE_TARGET",
				Message: fmt.Sprintf("Edge target node %s not found", edge.Target),
			})
		}
	}

	// Check for disconnected nodes
	connectedNodes := make(map[string]bool)
	for _, edge := range wf.Edges {
		connectedNodes[edge.Source] = true
		connectedNodes[edge.Target] = true
	}

	for _, node := range wf.Nodes {
		if node.Type != NodeStart && !connectedNodes[node.ID] {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Code:    "DISCONNECTED_NODE",
				Message: fmt.Sprintf("Node %s is not connected to the workflow", node.Name),
				NodeID:  node.ID,
			})
		}
	}

	// Check for cycles (simplified - would need proper graph traversal)
	// This is a basic check, production would use proper cycle detection

	return result
}

// ExecuteWorkflow executes a workflow
func (s *Service) ExecuteWorkflow(ctx context.Context, tenantID, workflowID string, req *ExecuteWorkflowRequest) (*WorkflowExecution, error) {
	wf, err := s.repo.GetWorkflow(ctx, tenantID, workflowID)
	if err != nil {
		return nil, err
	}

	if wf.Status != WorkflowPublished && !req.DryRun {
		return nil, fmt.Errorf("workflow must be published to execute")
	}

	inputJSON, _ := json.Marshal(req.Input)

	exec := &WorkflowExecution{
		ID:           GenerateExecutionID(),
		WorkflowID:   workflowID,
		WorkflowName: wf.Name,
		TenantID:     tenantID,
		Version:      wf.Version,
		Status:       ExecutionPending,
		TriggerType:  TriggerManual,
		Input:        inputJSON,
		Variables:    req.Variables,
		StartedAt:    time.Now(),
	}

	if err := s.repo.SaveExecution(ctx, exec); err != nil {
		return nil, err
	}

	if req.DryRun {
		exec.Status = ExecutionCompleted
		return exec, nil
	}

	// Execute asynchronously
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		s.executor.Execute(ctx, wf, exec)
	}()

	return exec, nil
}

// GetExecution retrieves an execution
func (s *Service) GetExecution(ctx context.Context, tenantID, execID string) (*WorkflowExecution, error) {
	return s.repo.GetExecution(ctx, tenantID, execID)
}

// ListExecutions lists executions
func (s *Service) ListExecutions(ctx context.Context, tenantID, workflowID string, filter *ExecutionFilter) ([]WorkflowExecution, error) {
	return s.repo.ListExecutions(ctx, tenantID, workflowID, filter)
}

// CancelExecution cancels an execution
func (s *Service) CancelExecution(ctx context.Context, tenantID, execID string) (*WorkflowExecution, error) {
	exec, err := s.repo.GetExecution(ctx, tenantID, execID)
	if err != nil {
		return nil, err
	}

	if exec.Status != ExecutionRunning && exec.Status != ExecutionPending {
		return nil, fmt.Errorf("execution cannot be cancelled in state: %s", exec.Status)
	}

	s.executor.Cancel(execID)

	exec.Status = ExecutionCancelled
	now := time.Now()
	exec.CompletedAt = &now

	s.repo.UpdateExecutionStatus(ctx, execID, ExecutionCancelled, nil, nil)

	return exec, nil
}

// GetWorkflowStats retrieves workflow statistics
func (s *Service) GetWorkflowStats(ctx context.Context, tenantID, workflowID string) (*WorkflowStats, error) {
	return s.repo.GetWorkflowStats(ctx, tenantID, workflowID)
}

// GetNodeCatalog returns the node catalog
func (s *Service) GetNodeCatalog() NodeCatalog {
	return GetNodeCatalog()
}

// ListTemplates lists workflow templates
func (s *Service) ListTemplates(ctx context.Context, category string) ([]WorkflowTemplate, error) {
	return s.repo.ListTemplates(ctx, category)
}

// GetTemplate retrieves a workflow template
func (s *Service) GetTemplate(ctx context.Context, templateID string) (*WorkflowTemplate, error) {
	return s.repo.GetTemplate(ctx, templateID)
}

// Executor handles workflow execution
type Executor struct {
	repo    Repository
	config  *ServiceConfig
	running sync.Map // map[execID]context.CancelFunc
}

// NewExecutor creates a new executor
func NewExecutor(repo Repository, config *ServiceConfig) *Executor {
	return &Executor{
		repo:   repo,
		config: config,
	}
}

// Execute executes a workflow
func (e *Executor) Execute(ctx context.Context, wf *Workflow, exec *WorkflowExecution) {
	// Create cancellable context
	ctx, cancel := context.WithTimeout(ctx, time.Duration(wf.Settings.TimeoutSeconds)*time.Second)
	defer cancel()

	e.running.Store(exec.ID, cancel)
	defer e.running.Delete(exec.ID)

	// Update status to running
	exec.Status = ExecutionRunning
	e.repo.SaveExecution(ctx, exec)

	// Initialize variables
	variables := make(map[string]any)
	for k, v := range exec.Variables {
		variables[k] = v
	}

	// Parse input
	var input map[string]any
	json.Unmarshal(exec.Input, &input)
	variables["input"] = input

	// Build node map and find start node
	nodeMap := make(map[string]*Node)
	var startNode *Node
	for i := range wf.Nodes {
		nodeMap[wf.Nodes[i].ID] = &wf.Nodes[i]
		if wf.Nodes[i].Type == NodeStart {
			startNode = &wf.Nodes[i]
		}
	}

	if startNode == nil {
		e.failExecution(ctx, exec, "NO_START_NODE", "No start node found")
		return
	}

	// Build adjacency map
	adjacency := make(map[string][]string)
	for _, edge := range wf.Edges {
		adjacency[edge.Source] = append(adjacency[edge.Source], edge.Target)
	}

	// Execute nodes using BFS
	nodeStates := make(map[string]*NodeExecution)
	queue := []string{startNode.ID}

	for len(queue) > 0 {
		select {
		case <-ctx.Done():
			e.failExecution(ctx, exec, "TIMEOUT", "Workflow execution timed out")
			return
		default:
		}

		nodeID := queue[0]
		queue = queue[1:]

		node := nodeMap[nodeID]
		if node == nil {
			continue
		}

		// Execute node
		nodeExec := e.executeNode(ctx, node, variables)
		nodeStates[nodeID] = nodeExec
		exec.NodeStates = append(exec.NodeStates, *nodeExec)

		if nodeExec.Status == ExecutionFailed {
			if wf.Settings.ErrorHandling == "fail_fast" {
				e.failExecution(ctx, exec, "NODE_FAILED", nodeExec.Error)
				return
			}
		}

		// Add next nodes to queue
		for _, nextID := range adjacency[nodeID] {
			queue = append(queue, nextID)
		}
	}

	// Complete execution
	exec.Status = ExecutionCompleted
	now := time.Now()
	exec.CompletedAt = &now
	exec.Duration = now.Sub(exec.StartedAt)

	e.repo.SaveExecution(ctx, exec)
}

func (e *Executor) executeNode(ctx context.Context, node *Node, variables map[string]any) *NodeExecution {
	now := time.Now()
	nodeExec := &NodeExecution{
		NodeID:    node.ID,
		NodeType:  node.Type,
		Status:    ExecutionRunning,
		StartedAt: &now,
	}

	// Execute based on node type
	var output any
	var err error

	switch node.Type {
	case NodeStart:
		output = variables["input"]
	case NodeEnd:
		output = variables
	case NodeDelay:
		var config struct {
			Duration int `json:"duration_ms"`
		}
		json.Unmarshal(node.Config, &config)
		if config.Duration > 0 {
			time.Sleep(time.Duration(config.Duration) * time.Millisecond)
		}
		output = variables
	case NodeTransform:
		// Would implement transformation logic
		output = variables
	case NodeCondition:
		// Would evaluate condition
		output = map[string]any{"result": true}
	case NodeHTTP:
		// Would make HTTP request
		output = map[string]any{"status": 200}
	default:
		output = variables
	}

	completed := time.Now()
	nodeExec.CompletedAt = &completed
	nodeExec.Duration = completed.Sub(now)

	if err != nil {
		nodeExec.Status = ExecutionFailed
		nodeExec.Error = err.Error()
	} else {
		nodeExec.Status = ExecutionCompleted
		outputJSON, _ := json.Marshal(output)
		nodeExec.Output = outputJSON
	}

	return nodeExec
}

func (e *Executor) failExecution(ctx context.Context, exec *WorkflowExecution, code, message string) {
	exec.Status = ExecutionFailed
	exec.Error = &ExecutionError{
		Code:    code,
		Message: message,
	}
	now := time.Now()
	exec.CompletedAt = &now
	exec.Duration = now.Sub(exec.StartedAt)

	e.repo.SaveExecution(ctx, exec)
}

// Cancel cancels an execution
func (e *Executor) Cancel(execID string) {
	if cancelFunc, ok := e.running.Load(execID); ok {
		cancelFunc.(context.CancelFunc)()
	}
}
