package workflow

import (
	"testing"
)

func TestWorkflow_Validate(t *testing.T) {
	tests := []struct {
		name     string
		workflow *Workflow
		wantErr  bool
	}{
		{
			name: "valid workflow",
			workflow: &Workflow{
				ID:       "wf-1",
				TenantID: "tenant-1",
				Name:     "Test Workflow",
				Nodes: []Node{
					{ID: "start", Type: NodeStart, Position: Position{X: 0, Y: 0}},
					{ID: "end", Type: NodeEnd, Position: Position{X: 100, Y: 0}},
				},
				Edges: []Edge{
					{ID: "e1", Source: "start", Target: "end"},
				},
				Status: WorkflowPublished,
			},
			wantErr: false,
		},
		{
			name: "missing name",
			workflow: &Workflow{
				ID:       "wf-1",
				TenantID: "tenant-1",
				Nodes:    []Node{{ID: "start", Type: NodeStart}},
			},
			wantErr: true,
		},
		{
			name: "no nodes",
			workflow: &Workflow{
				ID:       "wf-1",
				TenantID: "tenant-1",
				Name:     "Test",
				Nodes:    []Node{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.workflow.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetDefaultTemplates(t *testing.T) {
	templates := GetDefaultTemplates()

	if len(templates) != 3 {
		t.Errorf("Expected 3 templates, got %d", len(templates))
	}

	for _, tmpl := range templates {
		if tmpl.ID == "" {
			t.Error("Template ID should not be empty")
		}
		if tmpl.Name == "" {
			t.Error("Template name should not be empty")
		}
	}
}

func TestWorkflowStatus_Constants(t *testing.T) {
	statuses := []WorkflowStatus{
		WorkflowDraft, WorkflowPublished, WorkflowArchived, WorkflowDisabled,
	}

	for _, s := range statuses {
		if s == "" {
			t.Error("Workflow status constant should not be empty")
		}
	}
}

func TestExecutionStatus_Constants(t *testing.T) {
	statuses := []ExecutionStatus{
		ExecutionPending, ExecutionRunning, ExecutionCompleted,
		ExecutionFailed, ExecutionCancelled,
	}

	for _, s := range statuses {
		if s == "" {
			t.Error("Execution status constant should not be empty")
		}
	}
}

func TestNodeType_Constants(t *testing.T) {
	nodeTypes := []NodeType{
		NodeStart, NodeEnd, NodeCondition, NodeSwitch,
		NodeParallel, NodeMerge, NodeDelay, NodeLoop,
		NodeHTTP, NodeEmail, NodeWebhook, NodeTransform,
		NodeScript, NodeDatabase, NodeQueue, NodeCache,
	}

	for _, nt := range nodeTypes {
		if nt == "" {
			t.Error("Node type constant should not be empty")
		}
	}
}

func TestNode_Fields(t *testing.T) {
	node := Node{
		ID:       "node-1",
		Type:     NodeStart,
		Name:     "Start",
		Position: Position{X: 100, Y: 200},
	}

	if node.ID != "node-1" {
		t.Errorf("ID = %s, want node-1", node.ID)
	}
	if node.Position.X != 100 || node.Position.Y != 200 {
		t.Errorf("Position = (%f, %f), want (100, 200)", node.Position.X, node.Position.Y)
	}
}

func TestEdge_Fields(t *testing.T) {
	edge := Edge{
		ID:        "edge-1",
		Source:    "node-1",
		Target:    "node-2",
		Label:     "on success",
		Condition: "status == 200",
	}

	if edge.Source != "node-1" || edge.Target != "node-2" {
		t.Errorf("Edge connection = %s->%s, want node-1->node-2", edge.Source, edge.Target)
	}
}
