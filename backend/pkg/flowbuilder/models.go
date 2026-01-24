package flowbuilder

import "time"

// NodeType defines the type of workflow node
type NodeType string

const (
	NodeTrigger   NodeType = "trigger"
	NodeTransform NodeType = "transform"
	NodeFilter    NodeType = "filter"
	NodeSplit     NodeType = "split"
	NodeJoin      NodeType = "join"
	NodeDelay     NodeType = "delay"
	NodeHTTPCall  NodeType = "http_call"
	NodeCondition NodeType = "condition"
	NodeSwitch    NodeType = "switch"
	NodeLoop      NodeType = "loop"
	NodeAggregate NodeType = "aggregate"
	NodeNotify    NodeType = "notify"
	NodeEnd       NodeType = "end"
)

// WorkflowStatus represents the lifecycle state of a workflow
type WorkflowStatus string

const (
	WorkflowDraft    WorkflowStatus = "draft"
	WorkflowActive   WorkflowStatus = "active"
	WorkflowPaused   WorkflowStatus = "paused"
	WorkflowArchived WorkflowStatus = "archived"
)

// ExecutionStatus represents the state of a workflow execution
type ExecutionStatus string

const (
	ExecPending   ExecutionStatus = "pending"
	ExecRunning   ExecutionStatus = "running"
	ExecCompleted ExecutionStatus = "completed"
	ExecFailed    ExecutionStatus = "failed"
	ExecCancelled ExecutionStatus = "cancelled"
	ExecTimedOut  ExecutionStatus = "timed_out"
)

// Workflow represents a complete workflow DAG
type Workflow struct {
	ID          string         `json:"id" db:"id"`
	TenantID    string         `json:"tenant_id" db:"tenant_id"`
	Name        string         `json:"name" db:"name"`
	Description string         `json:"description" db:"description"`
	Status      WorkflowStatus `json:"status" db:"status"`
	Version     int            `json:"version" db:"version"`
	Nodes       []WorkflowNode `json:"nodes" db:"-"`
	Edges       []WorkflowEdge `json:"edges" db:"-"`
	Variables   map[string]any `json:"variables,omitempty" db:"-"`
	MaxTimeout  int            `json:"max_timeout_seconds" db:"max_timeout_seconds"`
	MaxRetries  int            `json:"max_retries" db:"max_retries"`
	Executions  int64          `json:"total_executions" db:"total_executions"`
	SuccessRate float64        `json:"success_rate" db:"success_rate"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" db:"updated_at"`
}

// WorkflowNode represents a single node in the workflow DAG
type WorkflowNode struct {
	ID         string         `json:"id" db:"id"`
	WorkflowID string         `json:"workflow_id" db:"workflow_id"`
	Type       NodeType       `json:"type" db:"type"`
	Name       string         `json:"name" db:"name"`
	Config     map[string]any `json:"config" db:"-"`
	Position   *Position      `json:"position" db:"-"`
	Timeout    int            `json:"timeout_seconds" db:"timeout_seconds"`
	RetryCount int            `json:"retry_count" db:"retry_count"`
	CreatedAt  time.Time      `json:"created_at" db:"created_at"`
}

// Position stores the visual position of a node in the builder
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// WorkflowEdge connects two nodes in the DAG
type WorkflowEdge struct {
	ID         string `json:"id" db:"id"`
	WorkflowID string `json:"workflow_id" db:"workflow_id"`
	SourceNode string `json:"source_node_id" db:"source_node_id"`
	TargetNode string `json:"target_node_id" db:"target_node_id"`
	Condition  string `json:"condition,omitempty" db:"condition"`
	Label      string `json:"label,omitempty" db:"label"`
}

// WorkflowExecution represents a single execution of a workflow
type WorkflowExecution struct {
	ID          string           `json:"id" db:"id"`
	WorkflowID  string           `json:"workflow_id" db:"workflow_id"`
	TenantID    string           `json:"tenant_id" db:"tenant_id"`
	Status      ExecutionStatus  `json:"status" db:"status"`
	TriggerData map[string]any   `json:"trigger_data,omitempty" db:"-"`
	Result      map[string]any   `json:"result,omitempty" db:"-"`
	Error       string           `json:"error,omitempty" db:"error"`
	NodeResults []NodeExecResult `json:"node_results,omitempty" db:"-"`
	StartedAt   time.Time        `json:"started_at" db:"started_at"`
	CompletedAt *time.Time       `json:"completed_at,omitempty" db:"completed_at"`
	DurationMs  int64            `json:"duration_ms" db:"duration_ms"`
}

// NodeExecResult captures the result of executing a single node
type NodeExecResult struct {
	NodeID     string          `json:"node_id" db:"node_id"`
	ExecID     string          `json:"execution_id" db:"execution_id"`
	Status     ExecutionStatus `json:"status" db:"status"`
	Input      map[string]any  `json:"input,omitempty" db:"-"`
	Output     map[string]any  `json:"output,omitempty" db:"-"`
	Error      string          `json:"error,omitempty" db:"error"`
	StartedAt  time.Time       `json:"started_at" db:"started_at"`
	DurationMs int64           `json:"duration_ms" db:"duration_ms"`
}

// WorkflowTemplate is a reusable workflow pattern
type WorkflowTemplate struct {
	ID          string         `json:"id" db:"id"`
	Name        string         `json:"name" db:"name"`
	Description string         `json:"description" db:"description"`
	Category    string         `json:"category" db:"category"`
	Nodes       []WorkflowNode `json:"nodes" db:"-"`
	Edges       []WorkflowEdge `json:"edges" db:"-"`
	Variables   map[string]any `json:"variables,omitempty" db:"-"`
	UsageCount  int64          `json:"usage_count" db:"usage_count"`
	Author      string         `json:"author" db:"author"`
	IsPublic    bool           `json:"is_public" db:"is_public"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
}

// WorkflowAnalytics aggregates execution metrics
type WorkflowAnalytics struct {
	WorkflowID      string  `json:"workflow_id"`
	TotalExecutions int64   `json:"total_executions"`
	SuccessfulExecs int64   `json:"successful_executions"`
	FailedExecs     int64   `json:"failed_executions"`
	AvgDurationMs   float64 `json:"avg_duration_ms"`
	P95DurationMs   float64 `json:"p95_duration_ms"`
	SuccessRate     float64 `json:"success_rate"`
	SlowestNode     string  `json:"slowest_node"`
	MostFailedNode  string  `json:"most_failed_node"`
}

// CreateWorkflowRequest for creating a new workflow
type CreateWorkflowRequest struct {
	Name        string         `json:"name" binding:"required"`
	Description string         `json:"description"`
	Nodes       []WorkflowNode `json:"nodes"`
	Edges       []WorkflowEdge `json:"edges"`
	Variables   map[string]any `json:"variables,omitempty"`
	MaxTimeout  int            `json:"max_timeout_seconds"`
	MaxRetries  int            `json:"max_retries"`
}

// ExecuteWorkflowRequest for triggering a workflow
type ExecuteWorkflowRequest struct {
	TriggerData map[string]any `json:"trigger_data"`
	DryRun      bool           `json:"dry_run"`
}

// ValidateWorkflowRequest for validating a workflow DAG
type ValidateWorkflowRequest struct {
	Nodes []WorkflowNode `json:"nodes" binding:"required"`
	Edges []WorkflowEdge `json:"edges" binding:"required"`
}

// ValidationResult holds workflow validation results
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}
