package faas

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/utils"
)

var (
	ErrFunctionNotFound = errors.New("function not found")
	ErrRuntimeTimeout   = errors.New("function execution timed out")
	ErrCodeRequired     = errors.New("function code is required")
	ErrInvalidRuntime   = errors.New("unsupported runtime")
)

// ServiceConfig holds configuration for the FaaS service.
type ServiceConfig struct {
	MaxFunctionsPerTenant int
	DefaultTimeoutMs      int
	DefaultMemoryMB       int
	MaxCodeSizeBytes      int
	MaxExecutionTimeMs    int
}

// DefaultServiceConfig returns sensible defaults.
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		MaxFunctionsPerTenant: 100,
		DefaultTimeoutMs:      5000,
		DefaultMemoryMB:       128,
		MaxCodeSizeBytes:      512 * 1024,
		MaxExecutionTimeMs:    30000,
	}
}

// Service provides FaaS operations.
type Service struct {
	repo   Repository
	config *ServiceConfig
	logger *utils.Logger
}

// NewService creates a new FaaS service.
func NewService(repo Repository, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}
	if repo == nil {
		repo = NewMemoryRepository()
	}
	return &Service{repo: repo, config: config, logger: utils.NewLogger("faas")}
}

// CreateFunction creates and deploys a new serverless function.
func (s *Service) CreateFunction(ctx context.Context, tenantID string, req *CreateFunctionRequest) (*Function, error) {
	if req.Code == "" {
		return nil, ErrCodeRequired
	}
	if !isValidRuntime(req.Runtime) {
		return nil, ErrInvalidRuntime
	}
	if len(req.Code) > s.config.MaxCodeSizeBytes {
		return nil, fmt.Errorf("code exceeds maximum size of %d bytes", s.config.MaxCodeSizeBytes)
	}

	timeoutMs := req.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = s.config.DefaultTimeoutMs
	}
	memoryMB := req.MemoryMB
	if memoryMB <= 0 {
		memoryMB = s.config.DefaultMemoryMB
	}
	entryPoint := req.EntryPoint
	if entryPoint == "" {
		entryPoint = "transform"
	}

	fn := &Function{
		ID:            uuid.New().String(),
		TenantID:      tenantID,
		Name:          req.Name,
		Description:   req.Description,
		Runtime:       req.Runtime,
		Status:        FunctionReady,
		Code:          req.Code,
		Version:       1,
		EntryPoint:    entryPoint,
		Timeout:       time.Duration(timeoutMs) * time.Millisecond,
		MemoryLimitMB: memoryMB,
		EnvVars:       req.EnvVars,
		EndpointIDs:   req.EndpointIDs,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, fn); err != nil {
		return nil, fmt.Errorf("create function: %w", err)
	}
	return fn, nil
}

// GetFunction retrieves a function by ID.
func (s *Service) GetFunction(ctx context.Context, tenantID, id string) (*Function, error) {
	return s.repo.Get(ctx, tenantID, id)
}

// ListFunctions returns functions for a tenant.
func (s *Service) ListFunctions(ctx context.Context, tenantID string, limit, offset int) ([]Function, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.repo.List(ctx, tenantID, limit, offset)
}

// UpdateFunction updates a function's code and configuration.
func (s *Service) UpdateFunction(ctx context.Context, tenantID, id string, req *CreateFunctionRequest) (*Function, error) {
	fn, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if req.Code != "" {
		fn.Code = req.Code
		fn.Version++
	}
	if req.Name != "" {
		fn.Name = req.Name
	}
	if req.Description != "" {
		fn.Description = req.Description
	}
	if req.TimeoutMs > 0 {
		fn.Timeout = time.Duration(req.TimeoutMs) * time.Millisecond
	}
	fn.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, fn); err != nil {
		return nil, err
	}
	return fn, nil
}

// DeleteFunction removes a function.
func (s *Service) DeleteFunction(ctx context.Context, tenantID, id string) error {
	return s.repo.Delete(ctx, tenantID, id)
}

// InvokeFunction executes a function with the given payload.
func (s *Service) InvokeFunction(ctx context.Context, tenantID, id string, req *InvokeFunctionRequest) (*InvokeFunctionResponse, error) {
	fn, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if fn.Status != FunctionReady && fn.Status != FunctionRunning {
		return nil, fmt.Errorf("function is not in a runnable state: %s", fn.Status)
	}

	start := time.Now()
	var output string
	var execErr error

	switch fn.Runtime {
	case RuntimeJavaScript, RuntimeTypeScript:
		output, execErr = s.executeJavaScript(fn, req.Payload)
	default:
		return nil, fmt.Errorf("runtime %s execution not yet implemented", fn.Runtime)
	}

	duration := time.Since(start)
	execution := &FunctionExecution{
		ID:         uuid.New().String(),
		FunctionID: fn.ID,
		TenantID:   tenantID,
		Input:      req.Payload,
		Output:     output,
		Duration:   duration,
		Success:    execErr == nil,
		ExecutedAt: time.Now().UTC(),
	}
	if execErr != nil {
		execution.Error = execErr.Error()
	}
	s.repo.RecordExecution(ctx, execution)

	// Update function stats
	fn.Invocations++
	fn.UpdatedAt = time.Now().UTC()
	if execErr != nil {
		fn.LastError = execErr.Error()
	}
	s.repo.Update(ctx, fn)

	resp := &InvokeFunctionResponse{
		Output:   output,
		Duration: duration,
		Success:  execErr == nil,
	}
	if execErr != nil {
		resp.Error = execErr.Error()
	}
	return resp, nil
}

// GetMetrics returns metrics for a function.
func (s *Service) GetMetrics(ctx context.Context, tenantID, functionID string) (*FunctionMetrics, error) {
	fn, err := s.repo.Get(ctx, tenantID, functionID)
	if err != nil {
		return nil, err
	}

	execs, _ := s.repo.ListExecutions(ctx, functionID, 1000)
	successCount := 0
	var totalDuration time.Duration
	for _, e := range execs {
		if e.Success {
			successCount++
		}
		totalDuration += e.Duration
	}

	metrics := &FunctionMetrics{
		FunctionID:       fn.ID,
		TotalInvocations: fn.Invocations,
		ErrorCount:       fn.Invocations - int64(successCount),
	}
	if len(execs) > 0 {
		metrics.AvgDurationMs = float64(totalDuration.Milliseconds()) / float64(len(execs))
		metrics.SuccessRate = float64(successCount) / float64(len(execs)) * 100
	}

	return metrics, nil
}

// ListTemplates returns available function templates.
func (s *Service) ListTemplates() []FunctionTemplate {
	return []FunctionTemplate{
		{
			ID:          "tpl-transform-json",
			Name:        "JSON Field Mapper",
			Description: "Map and rename fields in a JSON webhook payload",
			Runtime:     RuntimeJavaScript,
			Category:    "transformation",
			Code:        "function transform(payload) {\n  return { mapped: payload };\n}",
		},
		{
			ID:          "tpl-filter-events",
			Name:        "Event Type Filter",
			Description: "Filter webhooks by event type before delivery",
			Runtime:     RuntimeJavaScript,
			Category:    "filter",
			Code:        "function transform(payload) {\n  if (payload.type === 'user.created') return payload;\n  return null;\n}",
		},
		{
			ID:          "tpl-enrich-payload",
			Name:        "Payload Enrichment",
			Description: "Add computed fields and metadata to webhook payloads",
			Runtime:     RuntimeJavaScript,
			Category:    "enrichment",
			Code:        "function transform(payload) {\n  payload.processed_at = new Date().toISOString();\n  return payload;\n}",
		},
	}
}

func (s *Service) executeJavaScript(fn *Function, payload string) (string, error) {
	vm := goja.New()

	// Set up sandbox with payload
	vm.Set("__payload", payload)

	// Inject console.log capture
	var logOutput string
	vm.Set("console", map[string]interface{}{
		"log": func(args ...interface{}) {
			for _, a := range args {
				logOutput += fmt.Sprintf("%v ", a)
			}
			logOutput += "\n"
		},
	})

	// Execute user code + call entry point
	script := fn.Code + fmt.Sprintf("\n\nJSON.stringify(%s(JSON.parse(__payload)));", fn.EntryPoint)

	val, err := vm.RunString(script)
	if err != nil {
		return "", fmt.Errorf("execution error: %w", err)
	}

	result := val.String()
	if result == "undefined" || result == "null" {
		return "{}", nil
	}
	return result, nil
}

func isValidRuntime(r RuntimeType) bool {
	switch r {
	case RuntimeJavaScript, RuntimeTypeScript, RuntimeWASM:
		return true
	default:
		return false
	}
}
