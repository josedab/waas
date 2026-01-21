package workflow

import (
	"encoding/json"
	"fmt"
	"time"
)

// Workflow represents a visual workflow definition
type Workflow struct {
	ID          string          `json:"id"`
	TenantID    string          `json:"tenant_id"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Version     int             `json:"version"`
	Status      WorkflowStatus  `json:"status"`
	Trigger     TriggerConfig   `json:"trigger"`
	Nodes       []Node          `json:"nodes"`
	Edges       []Edge          `json:"edges"`
	Variables   []Variable      `json:"variables,omitempty"`
	Settings    WorkflowSettings `json:"settings"`
	Canvas      CanvasState     `json:"canvas"`
	CreatedBy   string          `json:"created_by,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	PublishedAt *time.Time      `json:"published_at,omitempty"`
}

// WorkflowStatus represents workflow lifecycle status
type WorkflowStatus string

const (
	WorkflowDraft     WorkflowStatus = "draft"
	WorkflowPublished WorkflowStatus = "published"
	WorkflowArchived  WorkflowStatus = "archived"
	WorkflowDisabled  WorkflowStatus = "disabled"
)

// TriggerConfig defines how the workflow is triggered
type TriggerConfig struct {
	Type       TriggerType       `json:"type"`
	WebhookID  string            `json:"webhook_id,omitempty"`
	Schedule   *ScheduleConfig   `json:"schedule,omitempty"`
	EventType  string            `json:"event_type,omitempty"`
	Conditions []TriggerCondition `json:"conditions,omitempty"`
}

// TriggerType defines trigger types
type TriggerType string

const (
	TriggerWebhook  TriggerType = "webhook"
	TriggerSchedule TriggerType = "schedule"
	TriggerEvent    TriggerType = "event"
	TriggerManual   TriggerType = "manual"
)

// ScheduleConfig defines schedule-based triggers
type ScheduleConfig struct {
	Cron     string `json:"cron,omitempty"`
	Interval string `json:"interval,omitempty"`
	Timezone string `json:"timezone,omitempty"`
}

// TriggerCondition defines conditional trigger execution
type TriggerCondition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    any    `json:"value"`
}

// Node represents a workflow node
type Node struct {
	ID       string          `json:"id"`
	Type     NodeType        `json:"type"`
	Name     string          `json:"name"`
	Position Position        `json:"position"`
	Config   json.RawMessage `json:"config"`
	Inputs   []Port          `json:"inputs,omitempty"`
	Outputs  []Port          `json:"outputs,omitempty"`
	Metadata NodeMetadata    `json:"metadata,omitempty"`
}

// NodeType defines available node types
type NodeType string

const (
	// Control flow nodes
	NodeStart       NodeType = "start"
	NodeEnd         NodeType = "end"
	NodeCondition   NodeType = "condition"
	NodeSwitch      NodeType = "switch"
	NodeLoop        NodeType = "loop"
	NodeParallel    NodeType = "parallel"
	NodeMerge       NodeType = "merge"
	NodeDelay       NodeType = "delay"
	
	// Action nodes
	NodeWebhook     NodeType = "webhook"
	NodeHTTP        NodeType = "http"
	NodeTransform   NodeType = "transform"
	NodeFilter      NodeType = "filter"
	NodeAggregate   NodeType = "aggregate"
	NodeCache       NodeType = "cache"
	
	// Integration nodes
	NodeDatabase    NodeType = "database"
	NodeQueue       NodeType = "queue"
	NodeEmail       NodeType = "email"
	NodeSlack       NodeType = "slack"
	NodeScript      NodeType = "script"
	
	// Error handling nodes
	NodeTryCatch    NodeType = "try_catch"
	NodeRetry       NodeType = "retry"
	NodeErrorHandler NodeType = "error_handler"
)

// Position defines node position on canvas
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Port defines node input/output ports
type Port struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
}

// NodeMetadata holds node metadata
type NodeMetadata struct {
	Icon        string            `json:"icon,omitempty"`
	Color       string            `json:"color,omitempty"`
	Description string            `json:"description,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Custom      map[string]string `json:"custom,omitempty"`
}

// Edge represents a connection between nodes
type Edge struct {
	ID         string   `json:"id"`
	Source     string   `json:"source"`
	SourcePort string   `json:"source_port"`
	Target     string   `json:"target"`
	TargetPort string   `json:"target_port"`
	Condition  string   `json:"condition,omitempty"`
	Label      string   `json:"label,omitempty"`
	Style      EdgeStyle `json:"style,omitempty"`
}

// EdgeStyle defines edge visual style
type EdgeStyle struct {
	StrokeColor string `json:"stroke_color,omitempty"`
	StrokeWidth int    `json:"stroke_width,omitempty"`
	Animated    bool   `json:"animated,omitempty"`
}

// Variable defines workflow-level variables
type Variable struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	DefaultValue any    `json:"default_value,omitempty"`
	Description  string `json:"description,omitempty"`
	Required     bool   `json:"required,omitempty"`
	Secret       bool   `json:"secret,omitempty"`
}

// WorkflowSettings holds workflow configuration
type WorkflowSettings struct {
	TimeoutSeconds   int      `json:"timeout_seconds"`
	MaxRetries       int      `json:"max_retries"`
	RetryDelayMs     int      `json:"retry_delay_ms"`
	ConcurrencyLimit int      `json:"concurrency_limit"`
	ErrorHandling    string   `json:"error_handling"` // fail_fast, continue, retry
	LogLevel         string   `json:"log_level"`
	Tags             []string `json:"tags,omitempty"`
}

// CanvasState holds visual editor canvas state
type CanvasState struct {
	Zoom      float64  `json:"zoom"`
	PanX      float64  `json:"pan_x"`
	PanY      float64  `json:"pan_y"`
	GridSize  int      `json:"grid_size"`
	SnapToGrid bool    `json:"snap_to_grid"`
}

// WorkflowExecution represents a workflow run
type WorkflowExecution struct {
	ID           string            `json:"id"`
	WorkflowID   string            `json:"workflow_id"`
	WorkflowName string            `json:"workflow_name"`
	TenantID     string            `json:"tenant_id"`
	Version      int               `json:"version"`
	Status       ExecutionStatus   `json:"status"`
	TriggerType  TriggerType       `json:"trigger_type"`
	TriggerData  json.RawMessage   `json:"trigger_data,omitempty"`
	Input        json.RawMessage   `json:"input,omitempty"`
	Output       json.RawMessage   `json:"output,omitempty"`
	Variables    map[string]any    `json:"variables,omitempty"`
	NodeStates   []NodeExecution   `json:"node_states,omitempty"`
	Error        *ExecutionError   `json:"error,omitempty"`
	StartedAt    time.Time         `json:"started_at"`
	CompletedAt  *time.Time        `json:"completed_at,omitempty"`
	Duration     time.Duration     `json:"duration,omitempty"`
}

// ExecutionStatus represents execution state
type ExecutionStatus string

const (
	ExecutionPending   ExecutionStatus = "pending"
	ExecutionRunning   ExecutionStatus = "running"
	ExecutionCompleted ExecutionStatus = "completed"
	ExecutionFailed    ExecutionStatus = "failed"
	ExecutionCancelled ExecutionStatus = "cancelled"
	ExecutionTimedOut  ExecutionStatus = "timed_out"
	ExecutionPaused    ExecutionStatus = "paused"
)

// NodeExecution represents a node's execution state
type NodeExecution struct {
	NodeID      string          `json:"node_id"`
	NodeType    NodeType        `json:"node_type"`
	Status      ExecutionStatus `json:"status"`
	Input       json.RawMessage `json:"input,omitempty"`
	Output      json.RawMessage `json:"output,omitempty"`
	Error       string          `json:"error,omitempty"`
	StartedAt   *time.Time      `json:"started_at,omitempty"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	Duration    time.Duration   `json:"duration,omitempty"`
	Retries     int             `json:"retries"`
}

// ExecutionError holds error details
type ExecutionError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	NodeID     string `json:"node_id,omitempty"`
	Stacktrace string `json:"stacktrace,omitempty"`
	Retryable  bool   `json:"retryable"`
}

// NodeCatalog represents available node types
type NodeCatalog struct {
	Categories []NodeCategory `json:"categories"`
}

// NodeCategory groups node templates
type NodeCategory struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Icon        string         `json:"icon"`
	Templates   []NodeTemplate `json:"templates"`
}

// NodeTemplate defines a node type template
type NodeTemplate struct {
	Type        NodeType        `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Icon        string          `json:"icon"`
	Category    string          `json:"category"`
	Inputs      []Port          `json:"inputs"`
	Outputs     []Port          `json:"outputs"`
	ConfigSchema json.RawMessage `json:"config_schema"`
	Defaults    json.RawMessage `json:"defaults,omitempty"`
}

// WorkflowTemplate represents a pre-built workflow template
type WorkflowTemplate struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
	Thumbnail   string   `json:"thumbnail,omitempty"`
	Workflow    Workflow `json:"workflow"`
	UsageCount  int      `json:"usage_count"`
}

// CreateWorkflowRequest represents workflow creation request
type CreateWorkflowRequest struct {
	Name        string           `json:"name" binding:"required"`
	Description string           `json:"description,omitempty"`
	Trigger     TriggerConfig    `json:"trigger" binding:"required"`
	Nodes       []Node           `json:"nodes,omitempty"`
	Edges       []Edge           `json:"edges,omitempty"`
	Variables   []Variable       `json:"variables,omitempty"`
	Settings    *WorkflowSettings `json:"settings,omitempty"`
	TemplateID  string           `json:"template_id,omitempty"`
}

// UpdateWorkflowRequest represents workflow update request
type UpdateWorkflowRequest struct {
	Name        *string           `json:"name,omitempty"`
	Description *string           `json:"description,omitempty"`
	Trigger     *TriggerConfig    `json:"trigger,omitempty"`
	Nodes       []Node            `json:"nodes,omitempty"`
	Edges       []Edge            `json:"edges,omitempty"`
	Variables   []Variable        `json:"variables,omitempty"`
	Settings    *WorkflowSettings `json:"settings,omitempty"`
	Canvas      *CanvasState      `json:"canvas,omitempty"`
}

// ExecuteWorkflowRequest represents manual execution request
type ExecuteWorkflowRequest struct {
	Input     map[string]any `json:"input,omitempty"`
	Variables map[string]any `json:"variables,omitempty"`
	DryRun    bool           `json:"dry_run,omitempty"`
}

// ValidateWorkflowRequest represents validation request
type ValidateWorkflowRequest struct {
	Workflow Workflow `json:"workflow"`
}

// ValidationResult represents workflow validation result
type ValidationResult struct {
	Valid    bool               `json:"valid"`
	Errors   []ValidationError  `json:"errors,omitempty"`
	Warnings []ValidationWarning `json:"warnings,omitempty"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	NodeID  string `json:"node_id,omitempty"`
	Field   string `json:"field,omitempty"`
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	NodeID  string `json:"node_id,omitempty"`
}

// WorkflowStats represents workflow statistics
type WorkflowStats struct {
	WorkflowID      string  `json:"workflow_id"`
	TotalExecutions int64   `json:"total_executions"`
	SuccessCount    int64   `json:"success_count"`
	FailureCount    int64   `json:"failure_count"`
	SuccessRate     float64 `json:"success_rate"`
	AvgDurationMs   float64 `json:"avg_duration_ms"`
	P50DurationMs   float64 `json:"p50_duration_ms"`
	P99DurationMs   float64 `json:"p99_duration_ms"`
	LastExecutedAt  *time.Time `json:"last_executed_at,omitempty"`
}

// GetDefaultSettings returns default workflow settings
func GetDefaultSettings() WorkflowSettings {
	return WorkflowSettings{
		TimeoutSeconds:   300,
		MaxRetries:       3,
		RetryDelayMs:     1000,
		ConcurrencyLimit: 10,
		ErrorHandling:    "fail_fast",
		LogLevel:         "info",
	}
}

// GetDefaultCanvas returns default canvas state
func GetDefaultCanvas() CanvasState {
	return CanvasState{
		Zoom:       1.0,
		PanX:       0,
		PanY:       0,
		GridSize:   20,
		SnapToGrid: true,
	}
}

// GetNodeCatalog returns the available node catalog
func GetNodeCatalog() NodeCatalog {
	return NodeCatalog{
		Categories: []NodeCategory{
			{
				ID:          "control",
				Name:        "Control Flow",
				Description: "Control execution flow",
				Icon:        "git-branch",
				Templates: []NodeTemplate{
					{Type: NodeStart, Name: "Start", Description: "Workflow entry point", Icon: "play", Category: "control", Outputs: []Port{{ID: "out", Name: "Output"}}},
					{Type: NodeEnd, Name: "End", Description: "Workflow exit point", Icon: "stop", Category: "control", Inputs: []Port{{ID: "in", Name: "Input"}}},
					{Type: NodeCondition, Name: "Condition", Description: "Branch based on condition", Icon: "git-branch", Category: "control", Inputs: []Port{{ID: "in", Name: "Input"}}, Outputs: []Port{{ID: "true", Name: "True"}, {ID: "false", Name: "False"}}},
					{Type: NodeSwitch, Name: "Switch", Description: "Multi-way branch", Icon: "shuffle", Category: "control", Inputs: []Port{{ID: "in", Name: "Input"}}, Outputs: []Port{{ID: "default", Name: "Default"}}},
					{Type: NodeLoop, Name: "Loop", Description: "Iterate over items", Icon: "repeat", Category: "control", Inputs: []Port{{ID: "in", Name: "Input"}}, Outputs: []Port{{ID: "item", Name: "Item"}, {ID: "done", Name: "Done"}}},
					{Type: NodeParallel, Name: "Parallel", Description: "Execute branches in parallel", Icon: "git-merge", Category: "control", Inputs: []Port{{ID: "in", Name: "Input"}}, Outputs: []Port{{ID: "out", Name: "Output"}}},
					{Type: NodeDelay, Name: "Delay", Description: "Wait for specified duration", Icon: "clock", Category: "control", Inputs: []Port{{ID: "in", Name: "Input"}}, Outputs: []Port{{ID: "out", Name: "Output"}}},
				},
			},
			{
				ID:          "actions",
				Name:        "Actions",
				Description: "Perform operations",
				Icon:        "zap",
				Templates: []NodeTemplate{
					{Type: NodeWebhook, Name: "Send Webhook", Description: "Send webhook to endpoint", Icon: "send", Category: "actions", Inputs: []Port{{ID: "in", Name: "Input"}}, Outputs: []Port{{ID: "out", Name: "Output"}, {ID: "error", Name: "Error"}}},
					{Type: NodeHTTP, Name: "HTTP Request", Description: "Make HTTP request", Icon: "globe", Category: "actions", Inputs: []Port{{ID: "in", Name: "Input"}}, Outputs: []Port{{ID: "out", Name: "Output"}, {ID: "error", Name: "Error"}}},
					{Type: NodeTransform, Name: "Transform", Description: "Transform data", Icon: "edit", Category: "actions", Inputs: []Port{{ID: "in", Name: "Input"}}, Outputs: []Port{{ID: "out", Name: "Output"}}},
					{Type: NodeFilter, Name: "Filter", Description: "Filter items", Icon: "filter", Category: "actions", Inputs: []Port{{ID: "in", Name: "Input"}}, Outputs: []Port{{ID: "out", Name: "Output"}}},
					{Type: NodeAggregate, Name: "Aggregate", Description: "Aggregate data", Icon: "layers", Category: "actions", Inputs: []Port{{ID: "in", Name: "Input"}}, Outputs: []Port{{ID: "out", Name: "Output"}}},
					{Type: NodeScript, Name: "Script", Description: "Run custom script", Icon: "code", Category: "actions", Inputs: []Port{{ID: "in", Name: "Input"}}, Outputs: []Port{{ID: "out", Name: "Output"}, {ID: "error", Name: "Error"}}},
				},
			},
			{
				ID:          "integrations",
				Name:        "Integrations",
				Description: "Connect to external services",
				Icon:        "plug",
				Templates: []NodeTemplate{
					{Type: NodeDatabase, Name: "Database", Description: "Query database", Icon: "database", Category: "integrations", Inputs: []Port{{ID: "in", Name: "Input"}}, Outputs: []Port{{ID: "out", Name: "Output"}, {ID: "error", Name: "Error"}}},
					{Type: NodeQueue, Name: "Queue", Description: "Publish to queue", Icon: "list", Category: "integrations", Inputs: []Port{{ID: "in", Name: "Input"}}, Outputs: []Port{{ID: "out", Name: "Output"}}},
					{Type: NodeEmail, Name: "Email", Description: "Send email", Icon: "mail", Category: "integrations", Inputs: []Port{{ID: "in", Name: "Input"}}, Outputs: []Port{{ID: "out", Name: "Output"}}},
					{Type: NodeSlack, Name: "Slack", Description: "Send Slack message", Icon: "message-square", Category: "integrations", Inputs: []Port{{ID: "in", Name: "Input"}}, Outputs: []Port{{ID: "out", Name: "Output"}}},
				},
			},
			{
				ID:          "error",
				Name:        "Error Handling",
				Description: "Handle errors and retries",
				Icon:        "alert-triangle",
				Templates: []NodeTemplate{
					{Type: NodeTryCatch, Name: "Try/Catch", Description: "Catch and handle errors", Icon: "shield", Category: "error", Inputs: []Port{{ID: "in", Name: "Input"}}, Outputs: []Port{{ID: "out", Name: "Output"}, {ID: "catch", Name: "Catch"}}},
					{Type: NodeRetry, Name: "Retry", Description: "Retry on failure", Icon: "refresh-cw", Category: "error", Inputs: []Port{{ID: "in", Name: "Input"}}, Outputs: []Port{{ID: "out", Name: "Output"}, {ID: "failed", Name: "Failed"}}},
					{Type: NodeErrorHandler, Name: "Error Handler", Description: "Global error handler", Icon: "alert-circle", Category: "error", Inputs: []Port{{ID: "error", Name: "Error"}}, Outputs: []Port{{ID: "out", Name: "Output"}}},
				},
			},
		},
	}
}

// Additional type aliases for test compatibility
const (
	NodeHTTPRequest   NodeType = NodeHTTP
	NodeNotification  NodeType = NodeEmail
	NodeAI            NodeType = NodeScript
	NodeSplit         NodeType = NodeParallel
	NodeSubworkflow   NodeType = "subworkflow"
)

// Execution is an alias for WorkflowExecution
type Execution = WorkflowExecution

// NodeExecution alias for compatibility
const (
	ExecPending   ExecutionStatus = ExecutionPending
	ExecRunning   ExecutionStatus = ExecutionRunning
	ExecCompleted ExecutionStatus = ExecutionCompleted
	ExecFailed    ExecutionStatus = ExecutionFailed
	ExecCancelled ExecutionStatus = ExecutionCancelled
	ExecPaused    ExecutionStatus = ExecutionPaused
)

// Edge field aliases for test compatibility - add to Edge struct via methods
func (e *Edge) GetSourceID() string { return e.Source }
func (e *Edge) GetTargetID() string { return e.Target }

// SourceID and TargetID are alias getters for Edge

// IsActive for workflow
func (w *Workflow) GetIsActive() bool {
	return w.Status == WorkflowPublished
}

// Validate validates a workflow definition
func (w *Workflow) Validate() error {
	if w.Name == "" {
		return fmt.Errorf("workflow name is required")
	}
	
	hasStart := false
	for _, node := range w.Nodes {
		if node.Type == NodeStart {
			hasStart = true
			break
		}
	}
	
	if !hasStart {
		return fmt.Errorf("workflow must have a start node")
	}
	
	return nil
}

// GetDefaultTemplates returns predefined workflow templates
func GetDefaultTemplates() []WorkflowTemplate {
	return []WorkflowTemplate{
		{
			ID:          "simple-webhook-forward",
			Name:        "Simple Webhook Forward",
			Description: "Receive a webhook and forward to another endpoint",
			Category:    "basic",
			Workflow: Workflow{
				Name: "Simple Forward",
				Nodes: []Node{
					{ID: "start", Type: NodeStart, Name: "Start"},
					{ID: "webhook", Type: NodeWebhook, Name: "Forward"},
					{ID: "end", Type: NodeEnd, Name: "End"},
				},
				Edges: []Edge{
					{Source: "start", Target: "webhook"},
					{Source: "webhook", Target: "end"},
				},
			},
		},
		{
			ID:          "conditional-routing",
			Name:        "Conditional Routing",
			Description: "Route webhooks based on payload content",
			Category:    "routing",
			Workflow: Workflow{
				Name: "Conditional Route",
				Nodes: []Node{
					{ID: "start", Type: NodeStart, Name: "Start"},
					{ID: "condition", Type: NodeCondition, Name: "Check Type"},
					{ID: "webhook1", Type: NodeWebhook, Name: "Route A"},
					{ID: "webhook2", Type: NodeWebhook, Name: "Route B"},
					{ID: "end", Type: NodeEnd, Name: "End"},
				},
				Edges: []Edge{
					{Source: "start", Target: "condition"},
					{Source: "condition", Target: "webhook1", SourcePort: "true"},
					{Source: "condition", Target: "webhook2", SourcePort: "false"},
					{Source: "webhook1", Target: "end"},
					{Source: "webhook2", Target: "end"},
				},
			},
		},
		{
			ID:          "transform-and-send",
			Name:        "Transform and Send",
			Description: "Transform webhook payload before sending",
			Category:    "data",
			Workflow: Workflow{
				Name: "Transform Flow",
				Nodes: []Node{
					{ID: "start", Type: NodeStart, Name: "Start"},
					{ID: "transform", Type: NodeTransform, Name: "Transform"},
					{ID: "webhook", Type: NodeWebhook, Name: "Send"},
					{ID: "end", Type: NodeEnd, Name: "End"},
				},
				Edges: []Edge{
					{Source: "start", Target: "transform"},
					{Source: "transform", Target: "webhook"},
					{Source: "webhook", Target: "end"},
				},
			},
		},
	}
}
