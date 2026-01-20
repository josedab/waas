package flow

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestNodeTypes(t *testing.T) {
	types := []NodeType{
		NodeTypeStart,
		NodeTypeEnd,
		NodeTypeHTTP,
		NodeTypeTransform,
		NodeTypeCondition,
		NodeTypeDelay,
		NodeTypeSplit,
		NodeTypeJoin,
		NodeTypeFilter,
		NodeTypeAggregate,
		NodeTypeRetry,
		NodeTypeLog,
	}
	
	for _, nodeType := range types {
		if nodeType == "" {
			t.Error("expected non-empty node type")
		}
	}
}

func TestExecutionStatus(t *testing.T) {
	statuses := []ExecutionStatus{
		StatusPending,
		StatusRunning,
		StatusCompleted,
		StatusFailed,
		StatusCancelled,
	}
	
	for _, status := range statuses {
		if status == "" {
			t.Error("expected non-empty status")
		}
	}
}

func TestDefaultFlowConfig(t *testing.T) {
	config := DefaultFlowConfig()
	
	if config.MaxExecutionTime <= 0 {
		t.Error("expected positive max execution time")
	}
	
	if config.LogLevel == "" {
		t.Error("expected log level to be set")
	}
}

func TestFlowCreation(t *testing.T) {
	flow := &Flow{
		ID:          "flow-1",
		TenantID:    "tenant-1",
		Name:        "Test Flow",
		Description: "A test workflow",
		Nodes: []Node{
			{ID: "start", Type: NodeTypeStart, Name: "Start", Position: Position{X: 0, Y: 0}},
			{ID: "http", Type: NodeTypeHTTP, Name: "HTTP Request", Position: Position{X: 100, Y: 0}},
			{ID: "end", Type: NodeTypeEnd, Name: "End", Position: Position{X: 200, Y: 0}},
		},
		Edges: []Edge{
			{ID: "e1", Source: "start", Target: "http"},
			{ID: "e2", Source: "http", Target: "end"},
		},
		Config:   DefaultFlowConfig(),
		IsActive: true,
		Version:  1,
	}
	
	if flow.Name != "Test Flow" {
		t.Errorf("expected name 'Test Flow', got %s", flow.Name)
	}
	
	if len(flow.Nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(flow.Nodes))
	}
	
	if len(flow.Edges) != 2 {
		t.Errorf("expected 2 edges, got %d", len(flow.Edges))
	}
}

func TestHTTPNodeConfig(t *testing.T) {
	config := HTTPNodeConfig{
		URL:     "https://example.com/webhook",
		Method:  "POST",
		Headers: map[string]string{"Content-Type": "application/json"},
		Timeout: 5000,
	}
	
	if config.URL == "" {
		t.Error("expected URL to be set")
	}
	
	if config.Method != "POST" {
		t.Errorf("expected method POST, got %s", config.Method)
	}
	
	if config.Timeout != 5000 {
		t.Errorf("expected timeout 5000, got %d", config.Timeout)
	}
}

func TestTransformNodeConfig(t *testing.T) {
	config := TransformNodeConfig{
		Script: "return { ...payload, transformed: true };",
	}
	
	if config.Script == "" {
		t.Error("expected script to be set")
	}
}

func TestConditionNodeConfig(t *testing.T) {
	config := ConditionNodeConfig{
		Expression: "payload.status === 'success'",
	}
	
	if config.Expression == "" {
		t.Error("expected expression to be set")
	}
}

func TestDelayNodeConfig(t *testing.T) {
	config := DelayNodeConfig{
		DelayMs: 1000,
		Type:    "fixed",
	}
	
	if config.DelayMs != 1000 {
		t.Errorf("expected delay 1000, got %d", config.DelayMs)
	}
	
	if config.Type != "fixed" {
		t.Errorf("expected type 'fixed', got %s", config.Type)
	}
}

func TestFilterNodeConfig(t *testing.T) {
	config := FilterNodeConfig{
		Expression:  "payload.type === 'important'",
		DropOnFalse: true,
	}
	
	if config.Expression == "" {
		t.Error("expected expression to be set")
	}
	
	if !config.DropOnFalse {
		t.Error("expected drop on false to be true")
	}
}

func TestRetryNodeConfig(t *testing.T) {
	config := RetryNodeConfig{
		MaxAttempts: 3,
		DelayMs:     1000,
		Multiplier:  2,
	}
	
	if config.MaxAttempts != 3 {
		t.Errorf("expected 3 max attempts, got %d", config.MaxAttempts)
	}
	
	if config.Multiplier != 2 {
		t.Errorf("expected multiplier 2, got %d", config.Multiplier)
	}
}

func TestFlowExecution(t *testing.T) {
	input := json.RawMessage(`{"event": "test"}`)
	output := json.RawMessage(`{"result": "success"}`)
	completedAt := time.Now()
	
	execution := &FlowExecution{
		ID:          "exec-1",
		FlowID:      "flow-1",
		TenantID:    "tenant-1",
		Status:      StatusCompleted,
		Input:       input,
		Output:      output,
		NodeResults: []NodeResult{
			{
				NodeID:     "start",
				NodeType:   NodeTypeStart,
				Status:     StatusCompleted,
				DurationMs: 1,
				StartedAt:  time.Now(),
			},
			{
				NodeID:     "end",
				NodeType:   NodeTypeEnd,
				Status:     StatusCompleted,
				DurationMs: 1,
				StartedAt:  time.Now(),
			},
		},
		StartedAt:   time.Now().Add(-time.Second),
		CompletedAt: &completedAt,
		DurationMs:  1000,
	}
	
	if execution.Status != StatusCompleted {
		t.Errorf("expected status completed, got %s", execution.Status)
	}
	
	if len(execution.NodeResults) != 2 {
		t.Errorf("expected 2 node results, got %d", len(execution.NodeResults))
	}
}

func TestNodeResult(t *testing.T) {
	result := NodeResult{
		NodeID:     "http-1",
		NodeType:   NodeTypeHTTP,
		Status:     StatusCompleted,
		Input:      json.RawMessage(`{"url": "test"}`),
		Output:     json.RawMessage(`{"status": 200}`),
		DurationMs: 500,
		StartedAt:  time.Now(),
	}
	
	if result.Status != StatusCompleted {
		t.Errorf("expected completed status, got %s", result.Status)
	}
	
	if result.DurationMs != 500 {
		t.Errorf("expected 500ms duration, got %d", result.DurationMs)
	}
}

func TestCreateFlowRequest(t *testing.T) {
	req := CreateFlowRequest{
		Name:        "New Flow",
		Description: "Test flow",
		Nodes: []Node{
			{ID: "start", Type: NodeTypeStart, Name: "Start"},
			{ID: "end", Type: NodeTypeEnd, Name: "End"},
		},
		Edges: []Edge{
			{ID: "e1", Source: "start", Target: "end"},
		},
		Config: DefaultFlowConfig(),
	}
	
	if req.Name == "" {
		t.Error("expected name to be set")
	}
	
	if len(req.Nodes) < 2 {
		t.Error("expected at least 2 nodes")
	}
}

func TestExecuteFlowRequest(t *testing.T) {
	req := ExecuteFlowRequest{
		Input:  json.RawMessage(`{"event": "webhook_received", "data": {"id": "123"}}`),
		DryRun: true,
	}
	
	if len(req.Input) == 0 {
		t.Error("expected input to be set")
	}
	
	if !req.DryRun {
		t.Error("expected dry run to be true")
	}
}

func TestFlowTemplate(t *testing.T) {
	template := FlowTemplate{
		ID:          "template-1",
		Name:        "Basic HTTP Webhook",
		Description: "Simple webhook forwarding template",
		Category:    "basics",
		Nodes: []Node{
			{ID: "start", Type: NodeTypeStart, Name: "Start"},
			{ID: "http", Type: NodeTypeHTTP, Name: "Forward"},
			{ID: "end", Type: NodeTypeEnd, Name: "End"},
		},
		Edges: []Edge{
			{ID: "e1", Source: "start", Target: "http"},
			{ID: "e2", Source: "http", Target: "end"},
		},
		Config: DefaultFlowConfig(),
	}
	
	if template.Category != "basics" {
		t.Errorf("expected category 'basics', got %s", template.Category)
	}
}

func TestEndpointFlow(t *testing.T) {
	ef := EndpointFlow{
		EndpointID: "endpoint-1",
		FlowID:     "flow-1",
		Priority:   1,
		CreatedAt:  time.Now(),
	}
	
	if ef.EndpointID == "" {
		t.Error("expected endpoint ID")
	}
	
	if ef.FlowID == "" {
		t.Error("expected flow ID")
	}
}

func TestServiceWithMockRepo(t *testing.T) {
	mockRepo := &mockFlowRepository{}
	service := NewService(mockRepo)
	
	if service == nil {
		t.Fatal("expected non-nil service")
	}
	
	ctx := context.Background()
	
	// Test creating a flow
	req := &CreateFlowRequest{
		Name: "Test Flow",
		Nodes: []Node{
			{ID: "start", Type: NodeTypeStart, Name: "Start"},
			{ID: "end", Type: NodeTypeEnd, Name: "End"},
		},
		Edges: []Edge{
			{ID: "e1", Source: "start", Target: "end"},
		},
		Config: DefaultFlowConfig(),
	}
	
	flow, err := service.CreateFlow(ctx, "tenant-1", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if flow == nil {
		t.Fatal("expected non-nil flow")
	}
	
	if flow.Name != "Test Flow" {
		t.Errorf("expected name 'Test Flow', got %s", flow.Name)
	}
}

func TestFlowValidation(t *testing.T) {
	tests := []struct {
		name    string
		flow    *Flow
		valid   bool
	}{
		{
			name: "valid flow with start and end",
			flow: &Flow{
				Name: "Valid Flow",
				Nodes: []Node{
					{ID: "start", Type: NodeTypeStart},
					{ID: "end", Type: NodeTypeEnd},
				},
				Edges: []Edge{
					{Source: "start", Target: "end"},
				},
			},
			valid: true,
		},
		{
			name: "missing start node",
			flow: &Flow{
				Name: "Invalid Flow",
				Nodes: []Node{
					{ID: "http", Type: NodeTypeHTTP},
					{ID: "end", Type: NodeTypeEnd},
				},
				Edges: []Edge{},
			},
			valid: false,
		},
		{
			name: "missing end node",
			flow: &Flow{
				Name: "Invalid Flow",
				Nodes: []Node{
					{ID: "start", Type: NodeTypeStart},
					{ID: "http", Type: NodeTypeHTTP},
				},
				Edges: []Edge{},
			},
			valid: false,
		},
	}
	
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			isValid := validateFlow(tc.flow)
			if isValid != tc.valid {
				t.Errorf("expected valid=%v, got %v", tc.valid, isValid)
			}
		})
	}
}

func validateFlow(flow *Flow) bool {
	if flow.Name == "" {
		return false
	}
	
	hasStart := false
	hasEnd := false
	for _, node := range flow.Nodes {
		if node.Type == NodeTypeStart {
			hasStart = true
		}
		if node.Type == NodeTypeEnd {
			hasEnd = true
		}
	}
	
	return hasStart && hasEnd
}

// Mock repository for testing
type mockFlowRepository struct {
	flows      map[string]*Flow
	executions map[string]*FlowExecution
	assignments map[string][]EndpointFlow
}

func (m *mockFlowRepository) CreateFlow(ctx context.Context, flow *Flow) error {
	if m.flows == nil {
		m.flows = make(map[string]*Flow)
	}
	m.flows[flow.ID] = flow
	return nil
}

func (m *mockFlowRepository) GetFlow(ctx context.Context, tenantID, flowID string) (*Flow, error) {
	if m.flows == nil {
		return nil, nil
	}
	f, ok := m.flows[flowID]
	if !ok || f.TenantID != tenantID {
		return nil, nil
	}
	return f, nil
}

func (m *mockFlowRepository) ListFlows(ctx context.Context, tenantID string, limit, offset int) ([]Flow, int, error) {
	var flows []Flow
	for _, f := range m.flows {
		if f.TenantID == tenantID {
			flows = append(flows, *f)
		}
	}
	return flows, len(flows), nil
}

func (m *mockFlowRepository) UpdateFlow(ctx context.Context, flow *Flow) error {
	if m.flows == nil {
		m.flows = make(map[string]*Flow)
	}
	m.flows[flow.ID] = flow
	return nil
}

func (m *mockFlowRepository) DeleteFlow(ctx context.Context, tenantID, flowID string) error {
	delete(m.flows, flowID)
	return nil
}

func (m *mockFlowRepository) SaveExecution(ctx context.Context, execution *FlowExecution) error {
	if m.executions == nil {
		m.executions = make(map[string]*FlowExecution)
	}
	m.executions[execution.ID] = execution
	return nil
}

func (m *mockFlowRepository) GetExecution(ctx context.Context, tenantID, executionID string) (*FlowExecution, error) {
	if m.executions == nil {
		return nil, nil
	}
	return m.executions[executionID], nil
}

func (m *mockFlowRepository) ListExecutions(ctx context.Context, tenantID, flowID string, limit, offset int) ([]FlowExecution, int, error) {
	var execs []FlowExecution
	for _, e := range m.executions {
		if e.TenantID == tenantID && (flowID == "" || e.FlowID == flowID) {
			execs = append(execs, *e)
		}
	}
	return execs, len(execs), nil
}

func (m *mockFlowRepository) AssignFlowToEndpoint(ctx context.Context, assignment *EndpointFlow) error {
	if m.assignments == nil {
		m.assignments = make(map[string][]EndpointFlow)
	}
	m.assignments[assignment.EndpointID] = append(m.assignments[assignment.EndpointID], *assignment)
	return nil
}

func (m *mockFlowRepository) GetEndpointFlows(ctx context.Context, endpointID string) ([]EndpointFlow, error) {
	if m.assignments == nil {
		return nil, nil
	}
	return m.assignments[endpointID], nil
}

func (m *mockFlowRepository) RemoveFlowFromEndpoint(ctx context.Context, endpointID, flowID string) error {
	if m.assignments == nil {
		return nil
	}
	flows := m.assignments[endpointID]
	for i, f := range flows {
		if f.FlowID == flowID {
			m.assignments[endpointID] = append(flows[:i], flows[i+1:]...)
			break
		}
	}
	return nil
}
