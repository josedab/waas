package models

import (
	"time"

	"github.com/google/uuid"
)

// Edge function runtime constants
const (
	RuntimeJavaScript = "javascript"
	RuntimeTypeScript = "typescript"
	RuntimePython     = "python"
)

// Edge function status constants
const (
	FunctionStatusDraft     = "draft"
	FunctionStatusDeploying = "deploying"
	FunctionStatusActive    = "active"
	FunctionStatusDeprecated = "deprecated"
	FunctionStatusFailed    = "failed"
)

// Deployment status constants
const (
	DeploymentStatusPending  = "pending"
	DeploymentStatusDeploying = "deploying"
	DeploymentStatusActive   = "active"
	DeploymentStatusFailed   = "failed"
	DeploymentStatusDraining = "draining"
)

// Trigger type constants
const (
	TriggerPreSend      = "pre_send"
	TriggerPostReceive  = "post_receive"
	TriggerTransform    = "transform"
	TriggerAuthenticate = "authenticate"
	TriggerEnrich       = "enrich"
)

// Invocation status constants
const (
	InvocationStatusSuccess   = "success"
	InvocationStatusError     = "error"
	InvocationStatusTimeout   = "timeout"
	InvocationStatusColdStart = "cold_start"
)

// Health status constants
const (
	HealthStatusUnknown   = "unknown"
	HealthStatusHealthy   = "healthy"
	HealthStatusUnhealthy = "unhealthy"
)

// EdgeFunction represents a serverless edge function
type EdgeFunction struct {
	ID              uuid.UUID              `json:"id" db:"id"`
	TenantID        uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	Name            string                 `json:"name" db:"name"`
	Description     string                 `json:"description,omitempty" db:"description"`
	Runtime         string                 `json:"runtime" db:"runtime"`
	Code            string                 `json:"code" db:"code"`
	EntryPoint      string                 `json:"entry_point" db:"entry_point"`
	Version         int                    `json:"version" db:"version"`
	Status          string                 `json:"status" db:"status"`
	TimeoutMs       int                    `json:"timeout_ms" db:"timeout_ms"`
	MemoryMb        int                    `json:"memory_mb" db:"memory_mb"`
	EnvironmentVars map[string]string      `json:"environment_vars" db:"environment_vars"`
	Dependencies    []string               `json:"dependencies" db:"dependencies"`
	Metadata        map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at" db:"updated_at"`
	DeployedAt      *time.Time             `json:"deployed_at,omitempty" db:"deployed_at"`
}

// EdgeFunctionVersion represents a version of an edge function
type EdgeFunctionVersion struct {
	ID         uuid.UUID `json:"id" db:"id"`
	FunctionID uuid.UUID `json:"function_id" db:"function_id"`
	Version    int       `json:"version" db:"version"`
	Code       string    `json:"code" db:"code"`
	EntryPoint string    `json:"entry_point" db:"entry_point"`
	CodeHash   string    `json:"code_hash" db:"code_hash"`
	ChangeLog  string    `json:"change_log,omitempty" db:"change_log"`
	CreatedBy  string    `json:"created_by,omitempty" db:"created_by"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// EdgeLocation represents an edge location
type EdgeLocation struct {
	ID        uuid.UUID              `json:"id" db:"id"`
	Name      string                 `json:"name" db:"name"`
	Code      string                 `json:"code" db:"code"`
	Region    string                 `json:"region" db:"region"`
	Provider  string                 `json:"provider" db:"provider"`
	Status    string                 `json:"status" db:"status"`
	LatencyMs int                    `json:"latency_ms" db:"latency_ms"`
	Capacity  int                    `json:"capacity" db:"capacity"`
	Metadata  map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
}

// EdgeFunctionDeployment represents a function deployment to an edge location
type EdgeFunctionDeployment struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	FunctionID     uuid.UUID  `json:"function_id" db:"function_id"`
	LocationID     uuid.UUID  `json:"location_id" db:"location_id"`
	Version        int        `json:"version" db:"version"`
	Status         string     `json:"status" db:"status"`
	DeploymentURL  string     `json:"deployment_url,omitempty" db:"deployment_url"`
	HealthCheckURL string     `json:"health_check_url,omitempty" db:"health_check_url"`
	LastHealthCheck *time.Time `json:"last_health_check,omitempty" db:"last_health_check"`
	HealthStatus   string     `json:"health_status" db:"health_status"`
	ErrorMessage   string     `json:"error_message,omitempty" db:"error_message"`
	DeployedAt     *time.Time `json:"deployed_at,omitempty" db:"deployed_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
	Location       *EdgeLocation `json:"location,omitempty" db:"-"`
}

// EdgeFunctionTrigger represents a trigger for an edge function
type EdgeFunctionTrigger struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	FunctionID  uuid.UUID              `json:"function_id" db:"function_id"`
	TriggerType string                 `json:"trigger_type" db:"trigger_type"`
	EventTypes  []string               `json:"event_types" db:"event_types"`
	EndpointIDs []uuid.UUID            `json:"endpoint_ids" db:"endpoint_ids"`
	Conditions  map[string]interface{} `json:"conditions" db:"conditions"`
	Priority    int                    `json:"priority" db:"priority"`
	Enabled     bool                   `json:"enabled" db:"enabled"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
}

// EdgeFunctionInvocation represents a function invocation
type EdgeFunctionInvocation struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	FunctionID     uuid.UUID  `json:"function_id" db:"function_id"`
	DeploymentID   *uuid.UUID `json:"deployment_id,omitempty" db:"deployment_id"`
	TriggerID      *uuid.UUID `json:"trigger_id,omitempty" db:"trigger_id"`
	TenantID       uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	EventID        *uuid.UUID `json:"event_id,omitempty" db:"event_id"`
	EndpointID     *uuid.UUID `json:"endpoint_id,omitempty" db:"endpoint_id"`
	LocationCode   string     `json:"location_code,omitempty" db:"location_code"`
	Status         string     `json:"status" db:"status"`
	DurationMs     int        `json:"duration_ms" db:"duration_ms"`
	MemoryUsedMb   int        `json:"memory_used_mb" db:"memory_used_mb"`
	InputSizeBytes int        `json:"input_size_bytes" db:"input_size_bytes"`
	OutputSizeBytes int       `json:"output_size_bytes" db:"output_size_bytes"`
	ErrorMessage   string     `json:"error_message,omitempty" db:"error_message"`
	ColdStart      bool       `json:"cold_start" db:"cold_start"`
	StartedAt      time.Time  `json:"started_at" db:"started_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty" db:"completed_at"`
}

// EdgeFunctionMetrics represents aggregated function metrics
type EdgeFunctionMetrics struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	FunctionID      uuid.UUID  `json:"function_id" db:"function_id"`
	LocationID      *uuid.UUID `json:"location_id,omitempty" db:"location_id"`
	PeriodStart     time.Time  `json:"period_start" db:"period_start"`
	PeriodEnd       time.Time  `json:"period_end" db:"period_end"`
	InvocationCount int        `json:"invocation_count" db:"invocation_count"`
	SuccessCount    int        `json:"success_count" db:"success_count"`
	ErrorCount      int        `json:"error_count" db:"error_count"`
	TimeoutCount    int        `json:"timeout_count" db:"timeout_count"`
	ColdStartCount  int        `json:"cold_start_count" db:"cold_start_count"`
	AvgDurationMs   float64    `json:"avg_duration_ms" db:"avg_duration_ms"`
	P50DurationMs   float64    `json:"p50_duration_ms" db:"p50_duration_ms"`
	P99DurationMs   float64    `json:"p99_duration_ms" db:"p99_duration_ms"`
	AvgMemoryMb     float64    `json:"avg_memory_mb" db:"avg_memory_mb"`
	TotalBilledMs   int64      `json:"total_billed_ms" db:"total_billed_ms"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
}

// EdgeFunctionSecret represents a secret for an edge function
type EdgeFunctionSecret struct {
	ID             uuid.UUID `json:"id" db:"id"`
	FunctionID     uuid.UUID `json:"function_id" db:"function_id"`
	Name           string    `json:"name" db:"name"`
	EncryptedValue string    `json:"-" db:"encrypted_value"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// EdgeFunctionTest represents a function test
type EdgeFunctionTest struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	FunctionID     uuid.UUID              `json:"function_id" db:"function_id"`
	TestName       string                 `json:"test_name" db:"test_name"`
	InputPayload   map[string]interface{} `json:"input_payload" db:"input_payload"`
	ExpectedOutput map[string]interface{} `json:"expected_output,omitempty" db:"expected_output"`
	ActualOutput   map[string]interface{} `json:"actual_output,omitempty" db:"actual_output"`
	Passed         *bool                  `json:"passed,omitempty" db:"passed"`
	DurationMs     int                    `json:"duration_ms" db:"duration_ms"`
	ErrorMessage   string                 `json:"error_message,omitempty" db:"error_message"`
	ExecutedAt     time.Time              `json:"executed_at" db:"executed_at"`
}

// Request/Response types

// CreateEdgeFunctionRequest is the request to create an edge function
type CreateEdgeFunctionRequest struct {
	Name            string            `json:"name" binding:"required"`
	Description     string            `json:"description"`
	Runtime         string            `json:"runtime"`
	Code            string            `json:"code" binding:"required"`
	EntryPoint      string            `json:"entry_point"`
	TimeoutMs       int               `json:"timeout_ms"`
	MemoryMb        int               `json:"memory_mb"`
	EnvironmentVars map[string]string `json:"environment_vars"`
	Dependencies    []string          `json:"dependencies"`
}

// UpdateEdgeFunctionRequest is the request to update an edge function
type UpdateEdgeFunctionRequest struct {
	Code            string            `json:"code"`
	EntryPoint      string            `json:"entry_point"`
	TimeoutMs       int               `json:"timeout_ms"`
	MemoryMb        int               `json:"memory_mb"`
	EnvironmentVars map[string]string `json:"environment_vars"`
	Dependencies    []string          `json:"dependencies"`
	ChangeLog       string            `json:"change_log"`
}

// DeployEdgeFunctionRequest is the request to deploy an edge function
type DeployEdgeFunctionRequest struct {
	LocationIDs []string `json:"location_ids" binding:"required"`
}

// CreateTriggerRequest is the request to create a function trigger
type CreateTriggerRequest struct {
	TriggerType string                 `json:"trigger_type" binding:"required"`
	EventTypes  []string               `json:"event_types"`
	EndpointIDs []string               `json:"endpoint_ids"`
	Conditions  map[string]interface{} `json:"conditions"`
	Priority    int                    `json:"priority"`
}

// InvokeFunctionRequest is the request to invoke a function
type InvokeFunctionRequest struct {
	Input      map[string]interface{} `json:"input" binding:"required"`
	LocationID string                 `json:"location_id"`
	EventID    string                 `json:"event_id"`
	EndpointID string                 `json:"endpoint_id"`
}

// RunTestRequest is the request to run a function test
type RunTestRequest struct {
	TestName       string                 `json:"test_name" binding:"required"`
	Input          map[string]interface{} `json:"input" binding:"required"`
	ExpectedOutput map[string]interface{} `json:"expected_output"`
}

// FunctionExecutionResult represents the result of a function execution
type FunctionExecutionResult struct {
	Success     bool                   `json:"success"`
	Output      map[string]interface{} `json:"output,omitempty"`
	Error       string                 `json:"error,omitempty"`
	DurationMs  int                    `json:"duration_ms"`
	MemoryUsed  int                    `json:"memory_used_mb"`
	Logs        []string               `json:"logs,omitempty"`
}

// EdgeFunctionDashboard represents the edge functions dashboard
type EdgeFunctionDashboard struct {
	TotalFunctions     int                     `json:"total_functions"`
	ActiveFunctions    int                     `json:"active_functions"`
	TotalDeployments   int                     `json:"total_deployments"`
	TotalInvocations   int64                   `json:"total_invocations"`
	ErrorRate          float64                 `json:"error_rate_percent"`
	AvgDurationMs      float64                 `json:"avg_duration_ms"`
	RecentInvocations  []*EdgeFunctionInvocation `json:"recent_invocations"`
	FunctionsByRuntime map[string]int          `json:"functions_by_runtime"`
	LocationCoverage   map[string]int          `json:"location_coverage"`
}

// EdgeFunctionWithDeployments represents a function with its deployments
type EdgeFunctionWithDeployments struct {
	*EdgeFunction
	Deployments []*EdgeFunctionDeployment `json:"deployments"`
	Triggers    []*EdgeFunctionTrigger    `json:"triggers"`
}
