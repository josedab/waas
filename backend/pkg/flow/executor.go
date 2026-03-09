package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/httputil"
)

// Executor executes flow workflows
type Executor struct {
	httpClient *http.Client
	vmPool     sync.Pool
}

// NewExecutor creates a new flow executor
func NewExecutor() *Executor {
	return &Executor{
		httpClient: httputil.NewSSRFSafeClient(30 * time.Second),
		vmPool: sync.Pool{
			New: func() interface{} {
				return goja.New()
			},
		},
	}
}

// ExecutionContext holds the context for flow execution
type ExecutionContext struct {
	ctx         context.Context
	execution   *FlowExecution
	flow        *Flow
	nodeMap     map[string]*Node
	edgeMap     map[string][]Edge
	results     map[string]*NodeResult
	currentData json.RawMessage
	mu          sync.RWMutex
}

// Execute runs a flow with the given input
func (e *Executor) Execute(ctx context.Context, flow *Flow, input json.RawMessage) (*FlowExecution, error) {
	// Validate flow structure
	if err := e.validateFlow(flow); err != nil {
		return nil, fmt.Errorf("invalid flow: %w", err)
	}

	// Create execution context
	execCtx := &ExecutionContext{
		ctx:     ctx,
		flow:    flow,
		nodeMap: make(map[string]*Node),
		edgeMap: make(map[string][]Edge),
		results: make(map[string]*NodeResult),
	}

	// Build lookup maps
	for i := range flow.Nodes {
		execCtx.nodeMap[flow.Nodes[i].ID] = &flow.Nodes[i]
	}
	for _, edge := range flow.Edges {
		execCtx.edgeMap[edge.Source] = append(execCtx.edgeMap[edge.Source], edge)
	}

	// Create execution record
	execution := &FlowExecution{
		ID:          uuid.New().String(),
		FlowID:      flow.ID,
		TenantID:    flow.TenantID,
		Status:      StatusRunning,
		Input:       input,
		NodeResults: make([]NodeResult, 0),
		StartedAt:   time.Now(),
	}
	execCtx.execution = execution
	execCtx.currentData = input

	// Set up timeout
	timeout := time.Duration(flow.Config.MaxExecutionTime) * time.Millisecond
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	var timeoutCancel context.CancelFunc
	execCtx.ctx, timeoutCancel = context.WithTimeout(ctx, timeout)
	defer timeoutCancel()

	// Find start node
	var startNode *Node
	for _, node := range flow.Nodes {
		if node.Type == NodeTypeStart {
			startNode = &node
			break
		}
	}
	if startNode == nil {
		return nil, fmt.Errorf("flow has no start node")
	}

	// Execute from start node
	if err := e.executeNode(execCtx, startNode.ID); err != nil {
		execution.Status = StatusFailed
		execution.Error = err.Error()
	} else {
		execution.Status = StatusCompleted
		execution.Output = execCtx.currentData
	}

	now := time.Now()
	execution.CompletedAt = &now
	execution.DurationMs = now.Sub(execution.StartedAt).Milliseconds()
	execution.NodeResults = e.collectResults(execCtx)

	return execution, nil
}

func (e *Executor) executeNode(ctx *ExecutionContext, nodeID string) error {
	// Check context cancellation
	select {
	case <-ctx.ctx.Done():
		return ctx.ctx.Err()
	default:
	}

	node, exists := ctx.nodeMap[nodeID]
	if !exists {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	// Record start
	result := &NodeResult{
		NodeID:    nodeID,
		NodeType:  node.Type,
		Status:    StatusRunning,
		Input:     ctx.currentData,
		StartedAt: time.Now(),
	}

	// Execute based on node type
	var output json.RawMessage
	var err error

	switch node.Type {
	case NodeTypeStart:
		output = ctx.currentData

	case NodeTypeEnd:
		output = ctx.currentData
		result.Status = StatusCompleted
		result.Output = output
		result.DurationMs = time.Since(result.StartedAt).Milliseconds()
		ctx.mu.Lock()
		ctx.results[nodeID] = result
		ctx.mu.Unlock()
		return nil

	case NodeTypeHTTP:
		output, err = e.executeHTTPNode(ctx, node)

	case NodeTypeTransform:
		output, err = e.executeTransformNode(ctx, node)

	case NodeTypeCondition:
		return e.executeConditionNode(ctx, node)

	case NodeTypeDelay:
		output, err = e.executeDelayNode(ctx, node)

	case NodeTypeFilter:
		output, err = e.executeFilterNode(ctx, node)

	case NodeTypeLog:
		output = ctx.currentData
		// Log the current data (in production, this would write to actual logs)

	default:
		output = ctx.currentData
	}

	if err != nil {
		result.Status = StatusFailed
		result.Error = err.Error()
		result.DurationMs = time.Since(result.StartedAt).Milliseconds()
		ctx.mu.Lock()
		ctx.results[nodeID] = result
		ctx.mu.Unlock()
		return err
	}

	result.Status = StatusCompleted
	result.Output = output
	result.DurationMs = time.Since(result.StartedAt).Milliseconds()

	ctx.mu.Lock()
	ctx.results[nodeID] = result
	ctx.currentData = output
	ctx.mu.Unlock()

	// Execute next nodes
	edges := ctx.edgeMap[nodeID]
	for _, edge := range edges {
		if err := e.executeNode(ctx, edge.Target); err != nil {
			return err
		}
	}

	return nil
}

func (e *Executor) executeHTTPNode(ctx *ExecutionContext, node *Node) (json.RawMessage, error) {
	var config HTTPNodeConfig
	if err := json.Unmarshal(node.Config, &config); err != nil {
		return nil, fmt.Errorf("invalid HTTP node config: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx.ctx, config.Method, config.URL, strings.NewReader(string(ctx.currentData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range config.Headers {
		req.Header.Set(k, v)
	}

	// Execute request
	client := e.httpClient
	if config.Timeout > 0 {
		client = httputil.NewSSRFSafeClient(time.Duration(config.Timeout) * time.Millisecond)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	var result json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		// Return status as result if no JSON body
		var marshalErr error
		result, marshalErr = json.Marshal(map[string]interface{}{
			"status_code": resp.StatusCode,
			"status":      resp.Status,
		})
		if marshalErr != nil {
			return nil, fmt.Errorf("failed to marshal status response: %w", marshalErr)
		}
	}

	if resp.StatusCode >= 400 {
		return result, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	return result, nil
}

func (e *Executor) executeTransformNode(ctx *ExecutionContext, node *Node) (json.RawMessage, error) {
	var config TransformNodeConfig
	if err := json.Unmarshal(node.Config, &config); err != nil {
		return nil, fmt.Errorf("invalid transform node config: %w", err)
	}

	// Get VM from pool
	vm := e.vmPool.Get().(*goja.Runtime)
	defer func() {
		// Reset global variables to prevent cross-tenant data leakage
		vm.Set("input", goja.Undefined())
		vm.Set("payload", goja.Undefined())
		vm.Set("env", goja.Undefined())
		vm.ClearInterrupt()
		e.vmPool.Put(vm)
	}()

	// Set up payload
	if err := vm.Set("input", string(ctx.currentData)); err != nil {
		return nil, err
	}

	// Parse input
	if _, err := vm.RunString("var payload = JSON.parse(input);"); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	// Execute transformation
	script := fmt.Sprintf(`(function() { %s })();`, config.Script)
	result, err := vm.RunString(script)
	if err != nil {
		return nil, fmt.Errorf("transformation error: %w", err)
	}

	// Export result
	output := result.Export()
	return json.Marshal(output)
}

func (e *Executor) executeConditionNode(ctx *ExecutionContext, node *Node) error {
	var config ConditionNodeConfig
	if err := json.Unmarshal(node.Config, &config); err != nil {
		return fmt.Errorf("invalid condition node config: %w", err)
	}

	// Evaluate condition
	vm := e.vmPool.Get().(*goja.Runtime)
	defer func() {
		vm.Set("input", goja.Undefined())
		vm.Set("payload", goja.Undefined())
		vm.ClearInterrupt()
		e.vmPool.Put(vm)
	}()

	vm.Set("input", string(ctx.currentData))
	vm.RunString("var payload = JSON.parse(input);")

	result, err := vm.RunString(config.Expression)
	if err != nil {
		return fmt.Errorf("condition evaluation error: %w", err)
	}

	conditionResult := result.ToBoolean()

	// Record result
	ctx.mu.Lock()
	ctx.results[node.ID] = &NodeResult{
		NodeID:    node.ID,
		NodeType:  node.Type,
		Status:    StatusCompleted,
		Input:     ctx.currentData,
		Output:    ctx.currentData,
		StartedAt: time.Now(),
	}
	ctx.mu.Unlock()

	// Find matching edge
	edges := ctx.edgeMap[node.ID]
	for _, edge := range edges {
		if (conditionResult && edge.Label == "true") || (!conditionResult && edge.Label == "false") || edge.Label == "" {
			return e.executeNode(ctx, edge.Target)
		}
	}

	return nil
}

func (e *Executor) executeDelayNode(ctx *ExecutionContext, node *Node) (json.RawMessage, error) {
	var config DelayNodeConfig
	if err := json.Unmarshal(node.Config, &config); err != nil {
		return nil, fmt.Errorf("invalid delay node config: %w", err)
	}

	select {
	case <-time.After(time.Duration(config.DelayMs) * time.Millisecond):
		return ctx.currentData, nil
	case <-ctx.ctx.Done():
		return nil, ctx.ctx.Err()
	}
}

func (e *Executor) executeFilterNode(ctx *ExecutionContext, node *Node) (json.RawMessage, error) {
	var config FilterNodeConfig
	if err := json.Unmarshal(node.Config, &config); err != nil {
		return nil, fmt.Errorf("invalid filter node config: %w", err)
	}

	// Evaluate filter expression
	vm := e.vmPool.Get().(*goja.Runtime)
	defer func() {
		vm.Set("input", goja.Undefined())
		vm.Set("payload", goja.Undefined())
		vm.ClearInterrupt()
		e.vmPool.Put(vm)
	}()

	vm.Set("input", string(ctx.currentData))
	vm.RunString("var payload = JSON.parse(input);")

	result, err := vm.RunString(config.Expression)
	if err != nil {
		return nil, fmt.Errorf("filter evaluation error: %w", err)
	}

	if !result.ToBoolean() && config.DropOnFalse {
		return nil, fmt.Errorf("event filtered out")
	}

	return ctx.currentData, nil
}

func (e *Executor) validateFlow(flow *Flow) error {
	if len(flow.Nodes) < 2 {
		return fmt.Errorf("flow must have at least 2 nodes (start and end)")
	}
	if len(flow.Nodes) > 20 {
		return fmt.Errorf("flow cannot have more than 20 nodes")
	}

	hasStart := false
	hasEnd := false
	nodeIDs := make(map[string]bool)

	for _, node := range flow.Nodes {
		if nodeIDs[node.ID] {
			return fmt.Errorf("duplicate node ID: %s", node.ID)
		}
		nodeIDs[node.ID] = true

		if node.Type == NodeTypeStart {
			hasStart = true
		}
		if node.Type == NodeTypeEnd {
			hasEnd = true
		}
	}

	if !hasStart {
		return fmt.Errorf("flow must have a start node")
	}
	if !hasEnd {
		return fmt.Errorf("flow must have an end node")
	}

	// Validate edges reference valid nodes
	for _, edge := range flow.Edges {
		if !nodeIDs[edge.Source] {
			return fmt.Errorf("edge references unknown source node: %s", edge.Source)
		}
		if !nodeIDs[edge.Target] {
			return fmt.Errorf("edge references unknown target node: %s", edge.Target)
		}
	}

	return nil
}

func (e *Executor) collectResults(ctx *ExecutionContext) []NodeResult {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	results := make([]NodeResult, 0, len(ctx.results))
	for _, r := range ctx.results {
		results = append(results, *r)
	}
	return results
}
