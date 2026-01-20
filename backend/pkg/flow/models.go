// Package flow provides visual workflow builder for webhook pipelines
package flow

import (
	"encoding/json"
	"time"
)

// Flow represents a webhook processing workflow
type Flow struct {
	ID          string     `json:"id" db:"id"`
	TenantID    string     `json:"tenant_id" db:"tenant_id"`
	Name        string     `json:"name" db:"name"`
	Description string     `json:"description,omitempty" db:"description"`
	Nodes       []Node     `json:"nodes" db:"nodes"`
	Edges       []Edge     `json:"edges" db:"edges"`
	Config      FlowConfig `json:"config" db:"config"`
	IsActive    bool       `json:"is_active" db:"is_active"`
	Version     int        `json:"version" db:"version"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// FlowConfig holds flow-level configuration
type FlowConfig struct {
	MaxExecutionTime int    `json:"max_execution_time_ms"` // Max 60000ms
	RetryOnError     bool   `json:"retry_on_error"`
	LogLevel         string `json:"log_level"` // debug, info, error
}

// Node represents a processing node in the flow
type Node struct {
	ID       string          `json:"id"`
	Type     NodeType        `json:"type"`
	Name     string          `json:"name"`
	Position Position        `json:"position"`
	Config   json.RawMessage `json:"config"`
}

// NodeType defines the type of processing node
type NodeType string

const (
	NodeTypeStart     NodeType = "start"     // Entry point
	NodeTypeEnd       NodeType = "end"       // Exit point
	NodeTypeHTTP      NodeType = "http"      // HTTP request
	NodeTypeTransform NodeType = "transform" // Payload transformation
	NodeTypeCondition NodeType = "condition" // Conditional branching
	NodeTypeDelay     NodeType = "delay"     // Wait/delay
	NodeTypeSplit     NodeType = "split"     // Parallel split
	NodeTypeJoin      NodeType = "join"      // Parallel join
	NodeTypeFilter    NodeType = "filter"    // Filter/drop events
	NodeTypeAggregate NodeType = "aggregate" // Aggregate multiple events
	NodeTypeRetry     NodeType = "retry"     // Retry logic
	NodeTypeLog       NodeType = "log"       // Logging
)

// Position represents node position in the visual editor
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Edge represents a connection between nodes
type Edge struct {
	ID        string `json:"id"`
	Source    string `json:"source"`
	Target    string `json:"target"`
	Label     string `json:"label,omitempty"`
	Condition string `json:"condition,omitempty"` // For conditional edges
}

// HTTPNodeConfig configures an HTTP request node
type HTTPNodeConfig struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers,omitempty"`
	Timeout int               `json:"timeout_ms"`
}

// TransformNodeConfig configures a transformation node
type TransformNodeConfig struct {
	Script string `json:"script"`
}

// ConditionNodeConfig configures a conditional node
type ConditionNodeConfig struct {
	Expression string `json:"expression"` // JavaScript expression
}

// DelayNodeConfig configures a delay node
type DelayNodeConfig struct {
	DelayMs int    `json:"delay_ms"`
	Type    string `json:"type"` // fixed, exponential
}

// FilterNodeConfig configures a filter node
type FilterNodeConfig struct {
	Expression  string `json:"expression"`
	DropOnFalse bool   `json:"drop_on_false"`
}

// RetryNodeConfig configures a retry node
type RetryNodeConfig struct {
	MaxAttempts int `json:"max_attempts"`
	DelayMs     int `json:"delay_ms"`
	Multiplier  int `json:"multiplier"`
}

// FlowExecution represents an execution of a flow
type FlowExecution struct {
	ID          string          `json:"id" db:"id"`
	FlowID      string          `json:"flow_id" db:"flow_id"`
	TenantID    string          `json:"tenant_id" db:"tenant_id"`
	Status      ExecutionStatus `json:"status" db:"status"`
	Input       json.RawMessage `json:"input" db:"input"`
	Output      json.RawMessage `json:"output,omitempty" db:"output"`
	Error       string          `json:"error,omitempty" db:"error"`
	NodeResults []NodeResult    `json:"node_results" db:"node_results"`
	StartedAt   time.Time       `json:"started_at" db:"started_at"`
	CompletedAt *time.Time      `json:"completed_at,omitempty" db:"completed_at"`
	DurationMs  int64           `json:"duration_ms" db:"duration_ms"`
}

// ExecutionStatus represents the status of a flow execution
type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusCompleted ExecutionStatus = "completed"
	StatusFailed    ExecutionStatus = "failed"
	StatusCancelled ExecutionStatus = "cancelled"
)

// NodeResult represents the result of executing a single node
type NodeResult struct {
	NodeID     string          `json:"node_id"`
	NodeType   NodeType        `json:"node_type"`
	Status     ExecutionStatus `json:"status"`
	Input      json.RawMessage `json:"input,omitempty"`
	Output     json.RawMessage `json:"output,omitempty"`
	Error      string          `json:"error,omitempty"`
	DurationMs int64           `json:"duration_ms"`
	StartedAt  time.Time       `json:"started_at"`
}

// CreateFlowRequest represents a request to create a flow
type CreateFlowRequest struct {
	Name        string     `json:"name" binding:"required,min=1,max=255"`
	Description string     `json:"description,omitempty"`
	Nodes       []Node     `json:"nodes" binding:"required,min=2"`
	Edges       []Edge     `json:"edges" binding:"required,min=1"`
	Config      FlowConfig `json:"config"`
}

// UpdateFlowRequest represents a request to update a flow
type UpdateFlowRequest struct {
	Name        string     `json:"name,omitempty"`
	Description string     `json:"description,omitempty"`
	Nodes       []Node     `json:"nodes,omitempty"`
	Edges       []Edge     `json:"edges,omitempty"`
	Config      FlowConfig `json:"config,omitempty"`
	IsActive    bool       `json:"is_active"`
}

// ExecuteFlowRequest represents a request to execute a flow
type ExecuteFlowRequest struct {
	Input  json.RawMessage `json:"input" binding:"required"`
	DryRun bool            `json:"dry_run"`
}

// FlowTemplate represents a pre-built flow template
type FlowTemplate struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Category    string     `json:"category"`
	Nodes       []Node     `json:"nodes"`
	Edges       []Edge     `json:"edges"`
	Config      FlowConfig `json:"config"`
}

// EndpointFlow links an endpoint to a flow
type EndpointFlow struct {
	EndpointID string    `json:"endpoint_id" db:"endpoint_id"`
	FlowID     string    `json:"flow_id" db:"flow_id"`
	Priority   int       `json:"priority" db:"priority"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// DefaultFlowConfig returns default flow configuration
func DefaultFlowConfig() FlowConfig {
	return FlowConfig{
		MaxExecutionTime: 30000,
		RetryOnError:     false,
		LogLevel:         "info",
	}
}
