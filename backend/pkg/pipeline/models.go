package pipeline

import (
	"encoding/json"
	"fmt"
	"time"
)

// StageType represents the type of processing stage
type StageType string

const (
	StageTransform StageType = "transform"
	StageValidate  StageType = "validate"
	StageRoute     StageType = "route"
	StageFanOut    StageType = "fan_out"
	StageDeliver   StageType = "deliver"
	StageFilter    StageType = "filter"
	StageEnrich    StageType = "enrich"
	StageDelay     StageType = "delay"
	StageLog       StageType = "log"
)

// Pipeline defines a declarative multi-step webhook delivery pipeline
type Pipeline struct {
	ID          string            `json:"id" db:"id"`
	TenantID    string            `json:"tenant_id" db:"tenant_id"`
	Name        string            `json:"name" db:"name"`
	Description string            `json:"description,omitempty" db:"description"`
	Stages      []StageDefinition `json:"stages"`
	Enabled     bool              `json:"enabled" db:"enabled"`
	Version     int               `json:"version" db:"version"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" db:"updated_at"`
}

// StageDefinition defines a single stage in the pipeline
type StageDefinition struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Type            StageType       `json:"type"`
	Config          json.RawMessage `json:"config"`
	ContinueOnError bool           `json:"continue_on_error"`
	Timeout         int             `json:"timeout_seconds,omitempty"` // 0 = use default
	Condition       string          `json:"condition,omitempty"`       // JS expression for conditional execution
}

// TransformConfig configures a transform stage
type TransformConfig struct {
	Script string `json:"script"` // JavaScript transformation
}

// ValidateConfig configures a validation stage
type ValidateConfig struct {
	Schema     json.RawMessage `json:"schema"`      // JSON Schema to validate against
	Strictness string          `json:"strictness"`   // loose, standard, strict
	RejectOn   string          `json:"reject_on"`    // error, warning
}

// RouteConfig configures a routing stage
type RouteConfig struct {
	Rules []RouteRule `json:"rules"`
}

// RouteRule defines a routing rule
type RouteRule struct {
	Condition   string   `json:"condition"`    // JS expression: payload.type == 'order'
	EndpointIDs []string `json:"endpoint_ids"` // Target endpoints
	Label       string   `json:"label,omitempty"`
}

// FanOutConfig configures a fan-out stage that duplicates delivery to multiple targets
type FanOutConfig struct {
	EndpointIDs []string `json:"endpoint_ids"`
	Parallel    bool     `json:"parallel"`    // Deliver in parallel or sequential
	MaxParallel int      `json:"max_parallel"` // Concurrency limit for parallel delivery
}

// DeliverConfig configures the final delivery stage
type DeliverConfig struct {
	EndpointID     string            `json:"endpoint_id,omitempty"` // Override target
	Timeout        int               `json:"timeout_seconds,omitempty"`
	RetryPolicy    *RetryPolicy      `json:"retry_policy,omitempty"`
	CustomHeaders  map[string]string `json:"custom_headers,omitempty"`
}

// RetryPolicy defines retry behavior for a delivery stage
type RetryPolicy struct {
	MaxAttempts     int `json:"max_attempts"`
	InitialInterval int `json:"initial_interval_seconds"`
	MaxInterval     int `json:"max_interval_seconds"`
	BackoffFactor   float64 `json:"backoff_factor"`
}

// FilterConfig configures a filter stage that can drop messages
type FilterConfig struct {
	Condition string `json:"condition"` // JS expression: return payload.amount > 100
	OnReject  string `json:"on_reject"` // drop, dlq, log
}

// EnrichConfig configures an enrichment stage
type EnrichConfig struct {
	Script string `json:"script"` // JS to add data: payload.enriched_at = new Date().toISOString()
}

// DelayConfig configures a delay stage
type DelayConfig struct {
	DurationMs int    `json:"duration_ms"`
	Until      string `json:"until,omitempty"` // ISO timestamp for scheduled delivery
}

// LogConfig configures a logging stage
type LogConfig struct {
	Message string `json:"message"`
	Level   string `json:"level"` // debug, info, warn, error
	Fields  map[string]string `json:"fields,omitempty"`
}

// --- Execution models ---

// PipelineExecution tracks the execution of a pipeline for a delivery
type PipelineExecution struct {
	ID          string              `json:"id" db:"id"`
	PipelineID  string              `json:"pipeline_id" db:"pipeline_id"`
	TenantID    string              `json:"tenant_id" db:"tenant_id"`
	DeliveryID  string              `json:"delivery_id" db:"delivery_id"`
	Status      ExecutionStatus     `json:"status" db:"status"`
	Stages      []StageExecution    `json:"stages"`
	StartedAt   time.Time           `json:"started_at" db:"started_at"`
	CompletedAt *time.Time          `json:"completed_at,omitempty" db:"completed_at"`
	DurationMs  int64               `json:"duration_ms" db:"duration_ms"`
	Error       string              `json:"error,omitempty" db:"error"`
}

// ExecutionStatus represents pipeline execution state
type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusCompleted ExecutionStatus = "completed"
	StatusFailed    ExecutionStatus = "failed"
	StatusSkipped   ExecutionStatus = "skipped"
)

// StageExecution tracks the execution of a single stage
type StageExecution struct {
	StageID     string          `json:"stage_id"`
	StageName   string          `json:"stage_name"`
	StageType   StageType       `json:"stage_type"`
	Status      ExecutionStatus `json:"status"`
	Input       json.RawMessage `json:"input,omitempty"`
	Output      json.RawMessage `json:"output,omitempty"`
	Error       string          `json:"error,omitempty"`
	StartedAt   time.Time       `json:"started_at"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	DurationMs  int64           `json:"duration_ms"`
}

// --- API request/response models ---

// CreatePipelineRequest defines the request to create a pipeline
type CreatePipelineRequest struct {
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description,omitempty"`
	Stages      []StageDefinition `json:"stages" binding:"required,min=1"`
}

// UpdatePipelineRequest defines the request to update a pipeline
type UpdatePipelineRequest struct {
	Name        *string            `json:"name,omitempty"`
	Description *string            `json:"description,omitempty"`
	Stages      *[]StageDefinition `json:"stages,omitempty"`
	Enabled     *bool              `json:"enabled,omitempty"`
}

// Validate checks if the pipeline definition is valid
func (p *Pipeline) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("pipeline name is required")
	}
	if len(p.Stages) == 0 {
		return fmt.Errorf("pipeline must have at least one stage")
	}

	stageIDs := make(map[string]bool)
	for i, stage := range p.Stages {
		if stage.ID == "" {
			return fmt.Errorf("stage %d: id is required", i)
		}
		if stageIDs[stage.ID] {
			return fmt.Errorf("stage %d: duplicate id '%s'", i, stage.ID)
		}
		stageIDs[stage.ID] = true

		if stage.Type == "" {
			return fmt.Errorf("stage %d (%s): type is required", i, stage.ID)
		}
		if err := validateStageType(stage.Type); err != nil {
			return fmt.Errorf("stage %d (%s): %w", i, stage.ID, err)
		}
	}

	return nil
}

func validateStageType(st StageType) error {
	switch st {
	case StageTransform, StageValidate, StageRoute, StageFanOut, StageDeliver,
		StageFilter, StageEnrich, StageDelay, StageLog:
		return nil
	default:
		return fmt.Errorf("unknown stage type: %s", st)
	}
}
