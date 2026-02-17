package flowbuilder

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service implements the visual workflow builder business logic
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateWorkflow(ctx context.Context, tenantID string, req *CreateWorkflowRequest) (*Workflow, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("workflow name is required")
	}

	// Validate the DAG before saving
	if len(req.Nodes) > 0 {
		result := s.ValidateDAG(req.Nodes, req.Edges)
		if !result.Valid {
			return nil, fmt.Errorf("invalid workflow: %v", result.Errors)
		}
	}

	workflow := &Workflow{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		Status:      WorkflowDraft,
		Version:     1,
		MaxTimeout:  req.MaxTimeout,
		MaxRetries:  req.MaxRetries,
		Variables:   req.Variables,
	}
	if workflow.MaxTimeout == 0 {
		workflow.MaxTimeout = 300
	}

	if err := s.repo.CreateWorkflow(ctx, workflow); err != nil {
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}

	if len(req.Nodes) > 0 {
		if err := s.repo.SaveNodes(ctx, workflow.ID, req.Nodes); err != nil {
			return nil, fmt.Errorf("failed to save nodes: %w", err)
		}
		workflow.Nodes = req.Nodes
	}
	if len(req.Edges) > 0 {
		if err := s.repo.SaveEdges(ctx, workflow.ID, req.Edges); err != nil {
			return nil, fmt.Errorf("failed to save edges: %w", err)
		}
		workflow.Edges = req.Edges
	}

	return workflow, nil
}

func (s *Service) GetWorkflow(ctx context.Context, id string) (*Workflow, error) {
	return s.repo.GetWorkflow(ctx, id)
}

func (s *Service) UpdateWorkflow(ctx context.Context, id string, req *CreateWorkflowRequest) (*Workflow, error) {
	workflow, err := s.repo.GetWorkflow(ctx, id)
	if err != nil {
		return nil, err
	}

	if len(req.Nodes) > 0 {
		result := s.ValidateDAG(req.Nodes, req.Edges)
		if !result.Valid {
			return nil, fmt.Errorf("invalid workflow: %v", result.Errors)
		}
	}

	if req.Name != "" {
		workflow.Name = req.Name
	}
	if req.Description != "" {
		workflow.Description = req.Description
	}
	if req.MaxTimeout > 0 {
		workflow.MaxTimeout = req.MaxTimeout
	}
	workflow.MaxRetries = req.MaxRetries

	if err := s.repo.UpdateWorkflow(ctx, workflow); err != nil {
		return nil, fmt.Errorf("failed to update workflow: %w", err)
	}

	if len(req.Nodes) > 0 {
		if err := s.repo.SaveNodes(ctx, id, req.Nodes); err != nil {
			return nil, fmt.Errorf("failed to save nodes: %w", err)
		}
		workflow.Nodes = req.Nodes
	}
	if len(req.Edges) > 0 {
		if err := s.repo.SaveEdges(ctx, id, req.Edges); err != nil {
			return nil, fmt.Errorf("failed to save edges: %w", err)
		}
		workflow.Edges = req.Edges
	}

	return workflow, nil
}

func (s *Service) DeleteWorkflow(ctx context.Context, id string) error {
	return s.repo.DeleteWorkflow(ctx, id)
}

func (s *Service) ListWorkflows(ctx context.Context, tenantID string, status WorkflowStatus, page, pageSize int) ([]Workflow, int, error) {
	return s.repo.ListWorkflows(ctx, tenantID, status, page, pageSize)
}

func (s *Service) ActivateWorkflow(ctx context.Context, id string) (*Workflow, error) {
	workflow, err := s.repo.GetWorkflow(ctx, id)
	if err != nil {
		return nil, err
	}

	if len(workflow.Nodes) == 0 {
		return nil, fmt.Errorf("workflow has no nodes")
	}
	result := s.ValidateDAG(workflow.Nodes, workflow.Edges)
	if !result.Valid {
		return nil, fmt.Errorf("workflow validation failed: %v", result.Errors)
	}

	workflow.Status = WorkflowActive
	if err := s.repo.UpdateWorkflow(ctx, workflow); err != nil {
		return nil, err
	}
	return workflow, nil
}

func (s *Service) PauseWorkflow(ctx context.Context, id string) (*Workflow, error) {
	workflow, err := s.repo.GetWorkflow(ctx, id)
	if err != nil {
		return nil, err
	}
	workflow.Status = WorkflowPaused
	if err := s.repo.UpdateWorkflow(ctx, workflow); err != nil {
		return nil, err
	}
	return workflow, nil
}

// ExecuteWorkflow runs a workflow with the given trigger data
func (s *Service) ExecuteWorkflow(ctx context.Context, tenantID, workflowID string, req *ExecuteWorkflowRequest) (*WorkflowExecution, error) {
	workflow, err := s.repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	if workflow.Status != WorkflowActive && !req.DryRun {
		return nil, fmt.Errorf("workflow is not active (status: %s)", workflow.Status)
	}

	exec := &WorkflowExecution{
		ID:          uuid.New().String(),
		WorkflowID:  workflowID,
		TenantID:    tenantID,
		Status:      ExecRunning,
		TriggerData: req.TriggerData,
	}

	if err := s.repo.CreateExecution(ctx, exec); err != nil {
		return nil, fmt.Errorf("failed to create execution: %w", err)
	}

	// Execute nodes in topological order
	nodeOrder := s.topologicalSort(workflow.Nodes, workflow.Edges)
	currentData := req.TriggerData
	if currentData == nil {
		currentData = map[string]any{}
	}

	for _, nodeID := range nodeOrder {
		node := s.findNode(workflow.Nodes, nodeID)
		if node == nil {
			continue
		}

		start := time.Now()
		output, err := s.executeNode(ctx, node, currentData)
		duration := time.Since(start).Milliseconds()

		result := &NodeExecResult{
			NodeID:     nodeID,
			ExecID:     exec.ID,
			Input:      currentData,
			StartedAt:  start,
			DurationMs: duration,
		}

		if err != nil {
			result.Status = ExecFailed
			result.Error = err.Error()
			// best-effort: persist failed node result for observability
			_ = s.repo.SaveNodeResult(ctx, result)

			now := time.Now()
			exec.Status = ExecFailed
			exec.Error = fmt.Sprintf("node %s failed: %s", node.Name, err.Error())
			exec.CompletedAt = &now
			exec.DurationMs = time.Since(exec.StartedAt).Milliseconds()
			// best-effort: persist execution failure state
			_ = s.repo.UpdateExecution(ctx, exec)
			return exec, nil
		}

		result.Status = ExecCompleted
		result.Output = output
		// best-effort: persist completed node result for observability
		_ = s.repo.SaveNodeResult(ctx, result)

		if output != nil {
			currentData = output
		}
	}

	now := time.Now()
	exec.Status = ExecCompleted
	exec.Result = currentData
	exec.CompletedAt = &now
	exec.DurationMs = time.Since(exec.StartedAt).Milliseconds()
	// best-effort: persist final execution state; workflow completed successfully
	_ = s.repo.UpdateExecution(ctx, exec)

	return exec, nil
}

func (s *Service) executeNode(ctx context.Context, node *WorkflowNode, input map[string]any) (map[string]any, error) {
	switch node.Type {
	case NodeTransform:
		return s.executeTransform(node, input)
	case NodeFilter:
		return s.executeFilter(node, input)
	case NodeCondition:
		return s.executeCondition(node, input)
	case NodeDelay:
		return s.executeDelay(ctx, node, input)
	case NodeHTTPCall:
		return s.executeHTTPCall(ctx, node, input)
	case NodeTrigger, NodeEnd:
		return input, nil
	default:
		return input, nil
	}
}

func (s *Service) executeTransform(node *WorkflowNode, input map[string]any) (map[string]any, error) {
	output := make(map[string]any)
	for k, v := range input {
		output[k] = v
	}
	// Apply configured transformations from node config
	if mappings, ok := node.Config["mappings"].(map[string]any); ok {
		for key, val := range mappings {
			output[key] = val
		}
	}
	return output, nil
}

func (s *Service) executeFilter(node *WorkflowNode, input map[string]any) (map[string]any, error) {
	field, _ := node.Config["field"].(string)
	value, _ := node.Config["value"]
	if field != "" && input[field] != value {
		return nil, fmt.Errorf("filter rejected: %s != %v", field, value)
	}
	return input, nil
}

func (s *Service) executeCondition(node *WorkflowNode, input map[string]any) (map[string]any, error) {
	field, _ := node.Config["field"].(string)
	value, _ := node.Config["value"]
	result := input[field] == value
	output := make(map[string]any)
	for k, v := range input {
		output[k] = v
	}
	output["_condition_result"] = result
	return output, nil
}

func (s *Service) executeDelay(ctx context.Context, node *WorkflowNode, input map[string]any) (map[string]any, error) {
	seconds, _ := node.Config["seconds"].(float64)
	if seconds > 0 && seconds <= 60 {
		select {
		case <-time.After(time.Duration(seconds) * time.Second):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return input, nil
}

func (s *Service) executeHTTPCall(_ context.Context, node *WorkflowNode, input map[string]any) (map[string]any, error) {
	// Stub: real implementation would make HTTP calls
	output := make(map[string]any)
	for k, v := range input {
		output[k] = v
	}
	output["_http_status"] = 200
	output["_http_url"] = node.Config["url"]
	return output, nil
}

// ValidateDAG validates the workflow DAG structure
func (s *Service) ValidateDAG(nodes []WorkflowNode, edges []WorkflowEdge) *ValidationResult {
	result := &ValidationResult{Valid: true}

	if len(nodes) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "workflow must have at least one node")
		return result
	}

	// Build node set
	nodeSet := map[string]bool{}
	hasTrigger := false
	hasEnd := false
	for _, n := range nodes {
		if nodeSet[n.ID] {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("duplicate node ID: %s", n.ID))
		}
		nodeSet[n.ID] = true
		if n.Type == NodeTrigger {
			hasTrigger = true
		}
		if n.Type == NodeEnd {
			hasEnd = true
		}
	}

	if !hasTrigger {
		result.Warnings = append(result.Warnings, "workflow has no trigger node")
	}
	if !hasEnd {
		result.Warnings = append(result.Warnings, "workflow has no end node")
	}

	// Validate edges reference valid nodes
	for _, e := range edges {
		if !nodeSet[e.SourceNode] {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("edge references unknown source node: %s", e.SourceNode))
		}
		if !nodeSet[e.TargetNode] {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("edge references unknown target node: %s", e.TargetNode))
		}
	}

	// Detect cycles
	if s.hasCycle(nodes, edges) {
		result.Valid = false
		result.Errors = append(result.Errors, "workflow contains a cycle")
	}

	return result
}

func (s *Service) hasCycle(nodes []WorkflowNode, edges []WorkflowEdge) bool {
	adj := map[string][]string{}
	for _, e := range edges {
		adj[e.SourceNode] = append(adj[e.SourceNode], e.TargetNode)
	}

	visited := map[string]int{} // 0=unvisited, 1=in-progress, 2=done
	for _, n := range nodes {
		visited[n.ID] = 0
	}

	var dfs func(string) bool
	dfs = func(id string) bool {
		visited[id] = 1
		for _, next := range adj[id] {
			if visited[next] == 1 {
				return true
			}
			if visited[next] == 0 && dfs(next) {
				return true
			}
		}
		visited[id] = 2
		return false
	}

	for _, n := range nodes {
		if visited[n.ID] == 0 && dfs(n.ID) {
			return true
		}
	}
	return false
}

func (s *Service) topologicalSort(nodes []WorkflowNode, edges []WorkflowEdge) []string {
	adj := map[string][]string{}
	inDegree := map[string]int{}
	for _, n := range nodes {
		inDegree[n.ID] = 0
	}
	for _, e := range edges {
		adj[e.SourceNode] = append(adj[e.SourceNode], e.TargetNode)
		inDegree[e.TargetNode]++
	}

	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	var order []string
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		order = append(order, current)
		for _, next := range adj[current] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}
	return order
}

func (s *Service) findNode(nodes []WorkflowNode, id string) *WorkflowNode {
	for i := range nodes {
		if nodes[i].ID == id {
			return &nodes[i]
		}
	}
	return nil
}

func (s *Service) GetExecution(ctx context.Context, id string) (*WorkflowExecution, error) {
	return s.repo.GetExecution(ctx, id)
}

func (s *Service) ListExecutions(ctx context.Context, workflowID string, page, pageSize int) ([]WorkflowExecution, int, error) {
	return s.repo.ListExecutions(ctx, workflowID, page, pageSize)
}

func (s *Service) ListTemplates(ctx context.Context, category string, page, pageSize int) ([]WorkflowTemplate, int, error) {
	return s.repo.ListTemplates(ctx, category, page, pageSize)
}

func (s *Service) GetTemplate(ctx context.Context, id string) (*WorkflowTemplate, error) {
	return s.repo.GetTemplate(ctx, id)
}

func (s *Service) GetAnalytics(ctx context.Context, workflowID string) (*WorkflowAnalytics, error) {
	return s.repo.GetAnalytics(ctx, workflowID)
}
