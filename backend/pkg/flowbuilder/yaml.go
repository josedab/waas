package flowbuilder

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// WorkflowYAML represents a workflow in YAML-serializable format
type WorkflowYAML struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Version     int            `yaml:"version"`
	Status      string         `yaml:"status"`
	MaxTimeout  int            `yaml:"max_timeout_seconds,omitempty"`
	MaxRetries  int            `yaml:"max_retries,omitempty"`
	Variables   map[string]any `yaml:"variables,omitempty"`
	Nodes       []NodeYAML     `yaml:"nodes"`
	Edges       []EdgeYAML     `yaml:"edges"`
}

// NodeYAML represents a workflow node in YAML format
type NodeYAML struct {
	ID       string         `yaml:"id"`
	Type     string         `yaml:"type"`
	Name     string         `yaml:"name"`
	Config   map[string]any `yaml:"config,omitempty"`
	Position *PositionYAML  `yaml:"position,omitempty"`
	Timeout  int            `yaml:"timeout_seconds,omitempty"`
	Retries  int            `yaml:"retry_count,omitempty"`
}

// PositionYAML represents visual position
type PositionYAML struct {
	X float64 `yaml:"x"`
	Y float64 `yaml:"y"`
}

// EdgeYAML represents a workflow edge in YAML format
type EdgeYAML struct {
	Source    string `yaml:"source"`
	Target    string `yaml:"target"`
	Label     string `yaml:"label,omitempty"`
	Condition string `yaml:"condition,omitempty"`
	Priority  int    `yaml:"priority,omitempty"`
}

// ExportYAML converts a Workflow to YAML bytes
func ExportYAML(w *Workflow) ([]byte, error) {
	if w == nil {
		return nil, fmt.Errorf("workflow is nil")
	}

	yamlW := WorkflowYAML{
		Name:        w.Name,
		Description: w.Description,
		Version:     w.Version,
		Status:      string(w.Status),
		MaxTimeout:  w.MaxTimeout,
		MaxRetries:  w.MaxRetries,
		Variables:   w.Variables,
		Nodes:       make([]NodeYAML, len(w.Nodes)),
		Edges:       make([]EdgeYAML, len(w.Edges)),
	}

	for i, n := range w.Nodes {
		yamlW.Nodes[i] = NodeYAML{
			ID:      n.ID,
			Type:    string(n.Type),
			Name:    n.Name,
			Config:  n.Config,
			Timeout: n.Timeout,
			Retries: n.RetryCount,
		}
		if n.Position != nil {
			yamlW.Nodes[i].Position = &PositionYAML{X: n.Position.X, Y: n.Position.Y}
		}
	}

	for i, e := range w.Edges {
		yamlW.Edges[i] = EdgeYAML{
			Source:    e.SourceNode,
			Target:    e.TargetNode,
			Label:     e.Label,
			Condition: e.Condition,
		}
	}

	return yaml.Marshal(yamlW)
}

// ImportYAML parses YAML bytes into a Workflow
func ImportYAML(data []byte) (*Workflow, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty YAML data")
	}

	var yamlW WorkflowYAML
	if err := yaml.Unmarshal(data, &yamlW); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}

	w := &Workflow{
		Name:        yamlW.Name,
		Description: yamlW.Description,
		Version:     yamlW.Version,
		Status:      WorkflowStatus(yamlW.Status),
		MaxTimeout:  yamlW.MaxTimeout,
		MaxRetries:  yamlW.MaxRetries,
		Variables:   yamlW.Variables,
		Nodes:       make([]WorkflowNode, len(yamlW.Nodes)),
		Edges:       make([]WorkflowEdge, len(yamlW.Edges)),
	}

	for i, n := range yamlW.Nodes {
		w.Nodes[i] = WorkflowNode{
			ID:         n.ID,
			Type:       NodeType(n.Type),
			Name:       n.Name,
			Config:     n.Config,
			Timeout:    n.Timeout,
			RetryCount: n.Retries,
		}
		if n.Position != nil {
			w.Nodes[i].Position = &Position{X: n.Position.X, Y: n.Position.Y}
		}
	}

	for i, e := range yamlW.Edges {
		w.Edges[i] = WorkflowEdge{
			SourceNode: e.Source,
			TargetNode: e.Target,
			Label:      e.Label,
			Condition:  e.Condition,
		}
	}

	return w, nil
}

// ExportJSON converts a Workflow to JSON bytes
func ExportJSON(w *Workflow) ([]byte, error) {
	if w == nil {
		return nil, fmt.Errorf("workflow is nil")
	}
	return json.MarshalIndent(w, "", "  ")
}

// GetBuiltInTemplates returns the built-in workflow templates
func GetBuiltInTemplates() []WorkflowTemplate {
	return []WorkflowTemplate{
		{
			ID:          "basic-webhook-forward",
			Name:        "Basic Webhook Forward",
			Description: "Receive a webhook and forward it to an HTTP endpoint",
			Category:    "starter",
		},
		{
			ID:          "filter-and-transform",
			Name:        "Filter & Transform",
			Description: "Filter events by condition, transform payload, then deliver",
			Category:    "starter",
		},
		{
			ID:          "fan-out-delivery",
			Name:        "Fan-Out Delivery",
			Description: "Receive event and deliver to multiple endpoints in parallel",
			Category:    "advanced",
		},
		{
			ID:          "conditional-routing",
			Name:        "Conditional Routing",
			Description: "Route events to different endpoints based on payload content",
			Category:    "advanced",
		},
		{
			ID:          "retry-with-transform",
			Name:        "Retry with Transform",
			Description: "On delivery failure, transform payload and retry to fallback endpoint",
			Category:    "resilience",
		},
	}
}
