package edge

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/utils"
)

// WasmRuntimeStatus represents the status of a Wasm runtime instance
type WasmRuntimeStatus string

const (
	WasmRuntimeReady   WasmRuntimeStatus = "ready"
	WasmRuntimeBusy    WasmRuntimeStatus = "busy"
	WasmRuntimeError   WasmRuntimeStatus = "error"
	WasmRuntimeStopped WasmRuntimeStatus = "stopped"
)

// TransformType defines types of edge transforms
type TransformType string

const (
	TransformTypePreDelivery  TransformType = "pre_delivery"
	TransformTypePostDelivery TransformType = "post_delivery"
	TransformTypeRouting      TransformType = "routing"
	TransformTypeEnrichment   TransformType = "enrichment"
	TransformTypeValidation   TransformType = "validation"
	TransformTypeFilter       TransformType = "filter"
)

// WasmFunction represents a user-defined Wasm/V8 edge function
type WasmFunction struct {
	ID           string            `json:"id"`
	TenantID     string            `json:"tenant_id"`
	Name         string            `json:"name"`
	Description  string            `json:"description,omitempty"`
	Language     string            `json:"language"` // javascript, typescript, rust, go
	SourceCode   string            `json:"source_code"`
	CompiledWasm []byte            `json:"-"` // Never serialize
	WasmSize     int64             `json:"wasm_size_bytes"`
	Transform    TransformType     `json:"transform_type"`
	EntryPoint   string            `json:"entry_point"`
	Timeout      time.Duration     `json:"timeout"`
	MemoryLimit  int64             `json:"memory_limit_mb"`
	EnvVars      map[string]string `json:"env_vars,omitempty"`
	Status       FunctionStatus    `json:"status"`
	Version      int               `json:"version"`
	Metrics      *FunctionMetrics  `json:"metrics,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// CreateWasmFunctionRequest represents a request to create a Wasm function
type CreateWasmFunctionRequest struct {
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description,omitempty"`
	Language    string            `json:"language" binding:"required"`
	SourceCode  string            `json:"source_code" binding:"required"`
	Transform   TransformType     `json:"transform_type" binding:"required"`
	EntryPoint  string            `json:"entry_point,omitempty"`
	TimeoutMs   int               `json:"timeout_ms,omitempty"`
	MemoryLimit int64             `json:"memory_limit_mb,omitempty"`
	EnvVars     map[string]string `json:"env_vars,omitempty"`
}

// WasmInvokeRequest represents a request to invoke a Wasm function
type WasmInvokeRequest struct {
	Input   json.RawMessage        `json:"input" binding:"required"`
	Headers map[string]string      `json:"headers,omitempty"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// WasmRuntimeRepository defines storage for Wasm runtime
type WasmRuntimeRepository interface {
	CreateFunction(ctx context.Context, fn *WasmFunction) error
	GetFunction(ctx context.Context, tenantID, functionID string) (*WasmFunction, error)
	ListFunctions(ctx context.Context, tenantID string) ([]WasmFunction, error)
	UpdateFunction(ctx context.Context, fn *WasmFunction) error
	DeleteFunction(ctx context.Context, tenantID, functionID string) error
	SaveInvocation(ctx context.Context, invocation *FunctionInvocation) error
	ListInvocations(ctx context.Context, tenantID, functionID string, limit int) ([]FunctionInvocation, error)
	UpdateMetrics(ctx context.Context, functionID string, metrics *FunctionMetrics) error
	ListTemplates(ctx context.Context) ([]FunctionTemplate, error)
}

// WasmRuntime provides edge function execution capabilities
type WasmRuntime struct {
	repo    WasmRuntimeRepository
	sandbox *Sandbox
	mu      sync.RWMutex
	logger  *utils.Logger
}

// Sandbox provides isolated execution for edge functions
type Sandbox struct {
	MaxMemoryMB  int64
	MaxTimeoutMs int
	MaxCodeSize  int64
}

// DefaultSandbox returns default sandbox limits
func DefaultSandbox() *Sandbox {
	return &Sandbox{
		MaxMemoryMB:  128,
		MaxTimeoutMs: 5000,
		MaxCodeSize:  1024 * 1024, // 1MB
	}
}

// NewWasmRuntime creates a new Wasm runtime
func NewWasmRuntime(repo WasmRuntimeRepository, sandbox *Sandbox) *WasmRuntime {
	if sandbox == nil {
		sandbox = DefaultSandbox()
	}
	return &WasmRuntime{
		repo:    repo,
		sandbox: sandbox,
		logger:  utils.NewLogger("edge-wasm-runtime"),
	}
}

// CreateFunction creates a new edge function
func (r *WasmRuntime) CreateFunction(ctx context.Context, tenantID string, req *CreateWasmFunctionRequest) (*WasmFunction, error) {
	if err := r.validateFunction(req); err != nil {
		return nil, err
	}

	timeout := time.Duration(req.TimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 100 * time.Millisecond
	}
	memLimit := req.MemoryLimit
	if memLimit == 0 {
		memLimit = 64
	}
	entryPoint := req.EntryPoint
	if entryPoint == "" {
		entryPoint = "handler"
	}

	now := time.Now()
	fn := &WasmFunction{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		Language:    req.Language,
		SourceCode:  req.SourceCode,
		Transform:   req.Transform,
		EntryPoint:  entryPoint,
		Timeout:     timeout,
		MemoryLimit: memLimit,
		EnvVars:     req.EnvVars,
		Status:      FunctionStatusActive,
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := r.repo.CreateFunction(ctx, fn); err != nil {
		return nil, fmt.Errorf("failed to create function: %w", err)
	}

	return fn, nil
}

// InvokeFunction executes an edge function
func (r *WasmRuntime) InvokeFunction(ctx context.Context, tenantID, functionID string, req *WasmInvokeRequest) (*FunctionInvocation, error) {
	fn, err := r.repo.GetFunction(ctx, tenantID, functionID)
	if err != nil {
		return nil, fmt.Errorf("function not found: %w", err)
	}

	if fn.Status != FunctionStatusActive {
		return nil, fmt.Errorf("function is not active: %s", fn.Status)
	}

	start := time.Now()
	invocation := &FunctionInvocation{
		ID:         uuid.New().String(),
		FunctionID: functionID,
		TenantID:   tenantID,
		Input:      req.Input,
		InvokedAt:  start,
	}

	// Execute in sandbox (simplified - real implementation uses wazero)
	output, execErr := r.executeInSandbox(ctx, fn, req.Input)
	invocation.DurationMs = time.Since(start).Milliseconds()

	if execErr != nil {
		invocation.Status = "error"
		invocation.ErrorMessage = execErr.Error()
	} else {
		invocation.Status = "success"
		invocation.Output = output
	}

	if err := r.repo.SaveInvocation(ctx, invocation); err != nil {
		r.logger.Error("failed to save invocation", map[string]interface{}{"error": err.Error(), "function_id": invocation.FunctionID})
	}

	return invocation, execErr
}

// executeInSandbox runs function code in an isolated sandbox
func (r *WasmRuntime) executeInSandbox(ctx context.Context, fn *WasmFunction, input json.RawMessage) (json.RawMessage, error) {
	// In production, this would use wazero or V8 isolate
	// For now, return input as passthrough to demonstrate the pipeline
	switch fn.Transform {
	case TransformTypeFilter:
		// Filter: return input if conditions pass, nil otherwise
		return input, nil
	case TransformTypeEnrichment:
		// Enrichment: add metadata to payload
		var data map[string]interface{}
		if err := json.Unmarshal(input, &data); err != nil {
			return input, nil
		}
		data["_enriched"] = true
		data["_enriched_at"] = time.Now().UTC().Format(time.RFC3339)
		data["_function_id"] = fn.ID
		enriched, err := json.Marshal(data)
		if err != nil {
			return input, nil
		}
		return enriched, nil
	case TransformTypeValidation:
		// Validation: check if payload is valid JSON
		var js json.RawMessage
		if err := json.Unmarshal(input, &js); err != nil {
			return nil, fmt.Errorf("validation failed: invalid JSON")
		}
		return input, nil
	default:
		// Pre/post delivery: passthrough
		return input, nil
	}
}

// GetFunction retrieves an edge function
func (r *WasmRuntime) GetFunction(ctx context.Context, tenantID, functionID string) (*WasmFunction, error) {
	return r.repo.GetFunction(ctx, tenantID, functionID)
}

// ListFunctions lists edge functions
func (r *WasmRuntime) ListFunctions(ctx context.Context, tenantID string) ([]WasmFunction, error) {
	return r.repo.ListFunctions(ctx, tenantID)
}

// DeleteFunction deletes an edge function
func (r *WasmRuntime) DeleteFunction(ctx context.Context, tenantID, functionID string) error {
	return r.repo.DeleteFunction(ctx, tenantID, functionID)
}

// ListTemplates returns available function templates
func (r *WasmRuntime) ListTemplates(ctx context.Context) ([]FunctionTemplate, error) {
	templates, err := r.repo.ListTemplates(ctx)
	if err != nil {
		return defaultTemplates(), nil
	}
	return templates, nil
}

// ListInvocations lists recent function invocations
func (r *WasmRuntime) ListInvocations(ctx context.Context, tenantID, functionID string, limit int) ([]FunctionInvocation, error) {
	if limit <= 0 {
		limit = 50
	}
	return r.repo.ListInvocations(ctx, tenantID, functionID, limit)
}

func (r *WasmRuntime) validateFunction(req *CreateWasmFunctionRequest) error {
	validLanguages := map[string]bool{
		"javascript": true, "typescript": true, "rust": true, "go": true,
	}
	if !validLanguages[req.Language] {
		return fmt.Errorf("unsupported language: %s", req.Language)
	}

	validTransforms := map[TransformType]bool{
		TransformTypePreDelivery:  true,
		TransformTypePostDelivery: true,
		TransformTypeRouting:      true,
		TransformTypeEnrichment:   true,
		TransformTypeValidation:   true,
		TransformTypeFilter:       true,
	}
	if !validTransforms[req.Transform] {
		return fmt.Errorf("invalid transform type: %s", req.Transform)
	}

	if int64(len(req.SourceCode)) > r.sandbox.MaxCodeSize {
		return fmt.Errorf("source code exceeds maximum size: %d bytes", r.sandbox.MaxCodeSize)
	}

	return nil
}

func defaultTemplates() []FunctionTemplate {
	return []FunctionTemplate{
		{
			ID:          "tpl-enrich-timestamp",
			Name:        "Add Timestamp",
			Description: "Enriches payload with processing timestamp",
			Code:        `export function handler(event) { event.processed_at = new Date().toISOString(); return event; }`,
			Category:    "enrichment",
			EntryPoint:  "handler",
			Runtime:     RuntimeLocal,
		},
		{
			ID:          "tpl-filter-status",
			Name:        "Filter by Status",
			Description: "Filters events based on status field",
			Code:        `export function handler(event) { if (event.status === 'active') return event; return null; }`,
			Category:    "filtering",
			EntryPoint:  "handler",
			Runtime:     RuntimeLocal,
		},
		{
			ID:          "tpl-validate-schema",
			Name:        "JSON Schema Validator",
			Description: "Validates payload against a JSON schema",
			Code:        `export function handler(event) { if (!event.id || !event.type) throw new Error('Missing required fields'); return event; }`,
			Category:    "validation",
			EntryPoint:  "handler",
			Runtime:     RuntimeLocal,
		},
		{
			ID:          "tpl-transform-payload",
			Name:        "Payload Transformer",
			Description: "Transforms webhook payload structure",
			Code:        `export function handler(event) { return { data: event, meta: { transformed: true } }; }`,
			Category:    "transform",
			EntryPoint:  "handler",
			Runtime:     RuntimeLocal,
		},
	}
}
