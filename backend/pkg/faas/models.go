package faas

import "time"

// RuntimeType identifies the function execution runtime.
type RuntimeType string

const (
	RuntimeJavaScript RuntimeType = "javascript"
	RuntimeTypeScript RuntimeType = "typescript"
	RuntimeWASM       RuntimeType = "wasm"
)

// FunctionStatus tracks the lifecycle of a deployed function.
type FunctionStatus string

const (
	FunctionDraft    FunctionStatus = "draft"
	FunctionBuilding FunctionStatus = "building"
	FunctionReady    FunctionStatus = "ready"
	FunctionRunning  FunctionStatus = "running"
	FunctionFailed   FunctionStatus = "failed"
	FunctionDisabled FunctionStatus = "disabled"
)

// Function represents a serverless transformation function.
type Function struct {
	ID            string            `json:"id" db:"id"`
	TenantID      string            `json:"tenant_id" db:"tenant_id"`
	Name          string            `json:"name" db:"name"`
	Description   string            `json:"description" db:"description"`
	Runtime       RuntimeType       `json:"runtime" db:"runtime"`
	Status        FunctionStatus    `json:"status" db:"status"`
	Code          string            `json:"code" db:"code"`
	Version       int               `json:"version" db:"version"`
	EntryPoint    string            `json:"entry_point" db:"entry_point"`
	Timeout       time.Duration     `json:"timeout_ms" db:"timeout_ms"`
	MemoryLimitMB int               `json:"memory_limit_mb" db:"memory_limit_mb"`
	EnvVars       map[string]string `json:"env_vars,omitempty"`
	EndpointIDs   []string          `json:"endpoint_ids,omitempty"`
	Invocations   int64             `json:"invocations" db:"invocations"`
	AvgDuration   time.Duration     `json:"avg_duration_ms" db:"avg_duration_ms"`
	LastError     string            `json:"last_error,omitempty" db:"last_error"`
	CreatedAt     time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at" db:"updated_at"`
}

// FunctionExecution represents a single invocation of a function.
type FunctionExecution struct {
	ID         string        `json:"id"`
	FunctionID string        `json:"function_id"`
	TenantID   string        `json:"tenant_id"`
	Input      string        `json:"input"`
	Output     string        `json:"output"`
	Duration   time.Duration `json:"duration_ms"`
	MemoryUsed int           `json:"memory_used_mb"`
	Success    bool          `json:"success"`
	Error      string        `json:"error,omitempty"`
	LogOutput  string        `json:"log_output,omitempty"`
	ExecutedAt time.Time     `json:"executed_at"`
}

// FunctionMetrics holds aggregate metrics for a function.
type FunctionMetrics struct {
	FunctionID       string     `json:"function_id"`
	TotalInvocations int64      `json:"total_invocations"`
	SuccessRate      float64    `json:"success_rate"`
	AvgDurationMs    float64    `json:"avg_duration_ms"`
	P99DurationMs    float64    `json:"p99_duration_ms"`
	ErrorCount       int64      `json:"error_count"`
	LastInvokedAt    *time.Time `json:"last_invoked_at,omitempty"`
}

// CreateFunctionRequest is the API request for creating a function.
type CreateFunctionRequest struct {
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description"`
	Runtime     RuntimeType       `json:"runtime" binding:"required"`
	Code        string            `json:"code" binding:"required"`
	EntryPoint  string            `json:"entry_point"`
	TimeoutMs   int               `json:"timeout_ms"`
	MemoryMB    int               `json:"memory_limit_mb"`
	EnvVars     map[string]string `json:"env_vars,omitempty"`
	EndpointIDs []string          `json:"endpoint_ids,omitempty"`
}

// InvokeFunctionRequest is the API request for invoking a function.
type InvokeFunctionRequest struct {
	Payload     string            `json:"payload" binding:"required"`
	ContentType string            `json:"content_type"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// InvokeFunctionResponse is the API response from a function invocation.
type InvokeFunctionResponse struct {
	Output    string        `json:"output"`
	Duration  time.Duration `json:"duration_ms"`
	Success   bool          `json:"success"`
	Error     string        `json:"error,omitempty"`
	LogOutput string        `json:"log_output,omitempty"`
}

// FunctionTemplate provides pre-built function code.
type FunctionTemplate struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Runtime     RuntimeType `json:"runtime"`
	Code        string      `json:"code"`
	Category    string      `json:"category"`
}
