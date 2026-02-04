package flowbuilder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportImportYAML(t *testing.T) {
	w := &Workflow{
		Name:        "Test Workflow",
		Description: "A test workflow",
		Version:     1,
		Status:      WorkflowDraft,
		MaxTimeout:  300,
		MaxRetries:  3,
		Nodes: []WorkflowNode{
			{ID: "n1", Type: NodeTrigger, Name: "Start", Config: map[string]any{"event": "order.created"}, Position: &Position{X: 100, Y: 200}},
			{ID: "n2", Type: NodeHTTPCall, Name: "Deliver", Config: map[string]any{"url": "https://example.com"}, Position: &Position{X: 400, Y: 200}},
		},
		Edges: []WorkflowEdge{
			{ID: "e1", SourceNode: "n1", TargetNode: "n2", Label: "forward"},
		},
	}

	data, err := ExportYAML(w)
	require.NoError(t, err)
	assert.Contains(t, string(data), "Test Workflow")
	assert.Contains(t, string(data), "trigger")
	assert.Contains(t, string(data), "http_call")

	imported, err := ImportYAML(data)
	require.NoError(t, err)
	assert.Equal(t, w.Name, imported.Name)
	assert.Equal(t, len(w.Nodes), len(imported.Nodes))
	assert.Equal(t, len(w.Edges), len(imported.Edges))
	assert.Equal(t, "n1", imported.Edges[0].SourceNode)
	assert.Equal(t, "n2", imported.Edges[0].TargetNode)
}

func TestExportYAML_Nil(t *testing.T) {
	_, err := ExportYAML(nil)
	assert.Error(t, err)
}

func TestImportYAML_Empty(t *testing.T) {
	_, err := ImportYAML([]byte{})
	assert.Error(t, err)
}

func TestImportYAML_Invalid(t *testing.T) {
	_, err := ImportYAML([]byte("not: [valid: yaml: {{"))
	assert.Error(t, err)
}

func TestGetBuiltInTemplates(t *testing.T) {
	templates := GetBuiltInTemplates()
	assert.True(t, len(templates) >= 3)
	for _, tmpl := range templates {
		assert.NotEmpty(t, tmpl.ID)
		assert.NotEmpty(t, tmpl.Name)
		assert.NotEmpty(t, tmpl.Category)
	}
}

func TestExportJSON(t *testing.T) {
	w := &Workflow{Name: "Test", Version: 1, Status: WorkflowDraft}
	data, err := ExportJSON(w)
	require.NoError(t, err)
	assert.Contains(t, string(data), "Test")
}
