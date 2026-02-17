package pipeline

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// PipelineDSL represents a pipeline defined using the declarative DSL format.
// This allows users to define pipelines using a simplified YAML/JSON syntax.
type PipelineDSL struct {
	Name        string     `json:"name" yaml:"name"`
	Description string     `json:"description,omitempty" yaml:"description"`
	Version     string     `json:"version,omitempty" yaml:"version"`
	Stages      []StageDSL `json:"stages" yaml:"stages"`
}

// StageDSL represents a stage in the declarative DSL
type StageDSL struct {
	ID              string      `json:"id,omitempty" yaml:"id"`
	Name            string      `json:"name,omitempty" yaml:"name"`
	Type            string      `json:"type" yaml:"type"`
	ContinueOnError bool        `json:"continue_on_error,omitempty" yaml:"continue_on_error"`
	Timeout         int         `json:"timeout,omitempty" yaml:"timeout"`
	Condition       string      `json:"condition,omitempty" yaml:"condition"`
	Config          interface{} `json:"config,omitempty" yaml:"config"`
}

// ParseDSL parses a JSON DSL definition into a CreatePipelineRequest
func ParseDSL(dslJSON []byte) (*CreatePipelineRequest, error) {
	var dsl PipelineDSL
	if err := json.Unmarshal(dslJSON, &dsl); err != nil {
		return nil, fmt.Errorf("invalid DSL JSON: %w", err)
	}

	return convertDSLToRequest(&dsl)
}

func convertDSLToRequest(dsl *PipelineDSL) (*CreatePipelineRequest, error) {
	if dsl.Name == "" {
		return nil, fmt.Errorf("pipeline name is required")
	}
	if len(dsl.Stages) == 0 {
		return nil, fmt.Errorf("at least one stage is required")
	}

	req := &CreatePipelineRequest{
		Name:        dsl.Name,
		Description: dsl.Description,
	}

	for i, stageDSL := range dsl.Stages {
		stage, err := convertStageDSL(i, &stageDSL)
		if err != nil {
			return nil, fmt.Errorf("stage %d: %w", i, err)
		}
		req.Stages = append(req.Stages, *stage)
	}

	return req, nil
}

func convertStageDSL(index int, dsl *StageDSL) (*StageDefinition, error) {
	stageType := StageType(strings.ToLower(dsl.Type))
	if err := validateStageType(stageType); err != nil {
		return nil, err
	}

	stageID := dsl.ID
	if stageID == "" {
		stageID = fmt.Sprintf("stage-%d-%s", index, dsl.Type)
	}

	stageName := dsl.Name
	if stageName == "" {
		stageName = fmt.Sprintf("%s stage", dsl.Type)
	}

	var configJSON json.RawMessage
	if dsl.Config != nil {
		var err error
		configJSON, err = json.Marshal(dsl.Config)
		if err != nil {
			return nil, fmt.Errorf("invalid config: %w", err)
		}
	}

	return &StageDefinition{
		ID:              stageID,
		Name:            stageName,
		Type:            stageType,
		Config:          configJSON,
		ContinueOnError: dsl.ContinueOnError,
		Timeout:         dsl.Timeout,
		Condition:       dsl.Condition,
	}, nil
}

// --- Visual Builder Support ---

// VisualPipelineNode represents a node in the visual pipeline builder
type VisualPipelineNode struct {
	ID       string             `json:"id"`
	Type     StageType          `json:"type"`
	Label    string             `json:"label"`
	Config   json.RawMessage    `json:"config,omitempty"`
	Position VisualNodePosition `json:"position"`
	Ports    []VisualPort       `json:"ports"`
}

// VisualNodePosition represents X/Y coordinates for the visual builder
type VisualNodePosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// VisualPort represents a connection port on a node
type VisualPort struct {
	ID        string `json:"id"`
	Type      string `json:"type"` // input, output, error
	Label     string `json:"label,omitempty"`
	Connected string `json:"connected_to,omitempty"`
}

// VisualPipelineEdge represents a connection between two nodes
type VisualPipelineEdge struct {
	ID       string `json:"id"`
	Source   string `json:"source"` // source node:port
	Target   string `json:"target"` // target node:port
	Label    string `json:"label,omitempty"`
	EdgeType string `json:"edge_type"` // normal, error, conditional
}

// VisualPipeline represents the full visual pipeline definition
type VisualPipeline struct {
	Nodes []VisualPipelineNode `json:"nodes"`
	Edges []VisualPipelineEdge `json:"edges"`
}

// ConvertVisualToPipeline converts a visual pipeline definition to a Pipeline
func ConvertVisualToPipeline(name, description string, visual *VisualPipeline) (*CreatePipelineRequest, error) {
	if len(visual.Nodes) == 0 {
		return nil, fmt.Errorf("visual pipeline has no nodes")
	}

	// Build adjacency list from edges
	nodeOrder := resolveNodeOrder(visual)

	req := &CreatePipelineRequest{
		Name:        name,
		Description: description,
	}

	for _, nodeID := range nodeOrder {
		node := findNode(visual.Nodes, nodeID)
		if node == nil {
			continue
		}

		stage := StageDefinition{
			ID:     node.ID,
			Name:   node.Label,
			Type:   node.Type,
			Config: node.Config,
		}

		// Check for error edges (continue_on_error)
		for _, edge := range visual.Edges {
			if edge.Source == nodeID && edge.EdgeType == "error" {
				stage.ContinueOnError = true
				break
			}
		}

		// Check for conditional edges
		for _, edge := range visual.Edges {
			if edge.Source == nodeID && edge.EdgeType == "conditional" {
				stage.Condition = edge.Label
				break
			}
		}

		req.Stages = append(req.Stages, stage)
	}

	return req, nil
}

func resolveNodeOrder(visual *VisualPipeline) []string {
	// Topological sort based on edges
	inDegree := make(map[string]int)
	adjList := make(map[string][]string)

	for _, node := range visual.Nodes {
		inDegree[node.ID] = 0
	}

	for _, edge := range visual.Edges {
		adjList[edge.Source] = append(adjList[edge.Source], edge.Target)
		inDegree[edge.Target]++
	}

	// BFS topological sort
	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	var order []string
	for len(queue) > 0 {
		nodeID := queue[0]
		queue = queue[1:]
		order = append(order, nodeID)

		for _, neighbor := range adjList[nodeID] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	return order
}

func findNode(nodes []VisualPipelineNode, id string) *VisualPipelineNode {
	for i := range nodes {
		if nodes[i].ID == id {
			return &nodes[i]
		}
	}
	return nil
}

// --- Pipeline Analytics ---

// PipelineAnalytics provides execution statistics for a pipeline
type PipelineAnalytics struct {
	PipelineID      string           `json:"pipeline_id"`
	PipelineName    string           `json:"pipeline_name"`
	Period          string           `json:"period"`
	TotalExecutions int64            `json:"total_executions"`
	SuccessCount    int64            `json:"success_count"`
	FailureCount    int64            `json:"failure_count"`
	AvgDurationMs   float64          `json:"avg_duration_ms"`
	P95DurationMs   float64          `json:"p95_duration_ms"`
	SuccessRate     float64          `json:"success_rate"`
	StageAnalytics  []StageAnalytics `json:"stage_analytics"`
	ErrorBreakdown  map[string]int64 `json:"error_breakdown"`
	Throughput      float64          `json:"throughput_per_sec"`
	GeneratedAt     time.Time        `json:"generated_at"`
}

// StageAnalytics provides per-stage execution statistics
type StageAnalytics struct {
	StageID       string  `json:"stage_id"`
	StageName     string  `json:"stage_name"`
	StageType     string  `json:"stage_type"`
	Executions    int64   `json:"executions"`
	Successes     int64   `json:"successes"`
	Failures      int64   `json:"failures"`
	AvgDurationMs float64 `json:"avg_duration_ms"`
	ErrorRate     float64 `json:"error_rate"`
}

// ComputeAnalytics computes analytics from a list of pipeline executions
func ComputeAnalytics(pipelineID, pipelineName, period string, executions []PipelineExecution) *PipelineAnalytics {
	analytics := &PipelineAnalytics{
		PipelineID:     pipelineID,
		PipelineName:   pipelineName,
		Period:         period,
		ErrorBreakdown: make(map[string]int64),
		GeneratedAt:    time.Now(),
	}

	if len(executions) == 0 {
		return analytics
	}

	var totalDuration int64
	stageStats := make(map[string]*StageAnalytics)

	for _, exec := range executions {
		analytics.TotalExecutions++
		totalDuration += exec.DurationMs

		if exec.Status == StatusCompleted {
			analytics.SuccessCount++
		} else if exec.Status == StatusFailed {
			analytics.FailureCount++
			if exec.Error != "" {
				analytics.ErrorBreakdown[exec.Error]++
			}
		}

		for _, stage := range exec.Stages {
			key := stage.StageID
			sa, ok := stageStats[key]
			if !ok {
				sa = &StageAnalytics{
					StageID:   stage.StageID,
					StageName: stage.StageName,
					StageType: string(stage.StageType),
				}
				stageStats[key] = sa
			}
			sa.Executions++
			sa.AvgDurationMs += float64(stage.DurationMs)
			if stage.Status == StatusCompleted {
				sa.Successes++
			} else if stage.Status == StatusFailed {
				sa.Failures++
			}
		}
	}

	analytics.AvgDurationMs = float64(totalDuration) / float64(analytics.TotalExecutions)
	if analytics.TotalExecutions > 0 {
		analytics.SuccessRate = float64(analytics.SuccessCount) / float64(analytics.TotalExecutions)
	}

	// Compute per-stage averages
	for _, sa := range stageStats {
		if sa.Executions > 0 {
			sa.AvgDurationMs /= float64(sa.Executions)
			sa.ErrorRate = float64(sa.Failures) / float64(sa.Executions)
		}
		analytics.StageAnalytics = append(analytics.StageAnalytics, *sa)
	}

	// Estimate P95 (simplified: use 95th percentile of sorted durations)
	if len(executions) > 0 {
		durations := make([]int64, len(executions))
		for i, e := range executions {
			durations[i] = e.DurationMs
		}
		sortInt64s(durations)
		p95Idx := int(float64(len(durations)) * 0.95)
		if p95Idx >= len(durations) {
			p95Idx = len(durations) - 1
		}
		analytics.P95DurationMs = float64(durations[p95Idx])
	}

	return analytics
}

func sortInt64s(arr []int64) {
	for i := 1; i < len(arr); i++ {
		key := arr[i]
		j := i - 1
		for j >= 0 && arr[j] > key {
			arr[j+1] = arr[j]
			j--
		}
		arr[j+1] = key
	}
}
