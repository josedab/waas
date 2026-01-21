package edge

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Service provides edge function operations
type Service struct {
	repo      Repository
	providers map[RuntimeType]RuntimeProvider
	compiler  CodeCompiler
	router    EdgeRouter
	mu        sync.RWMutex
	config    *ServiceConfig
}

// ServiceConfig holds service configuration
type ServiceConfig struct {
	MaxFunctionsPerTenant int
	MaxCodeSizeKB         int
	DefaultRegions        []EdgeRegion
	EnableAutoScaling     bool
	MetricsRetentionDays  int
}

// DefaultServiceConfig returns default configuration
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		MaxFunctionsPerTenant: 100,
		MaxCodeSizeKB:         1024, // 1MB
		DefaultRegions:        []EdgeRegion{RegionUSEast, RegionEUWest},
		EnableAutoScaling:     true,
		MetricsRetentionDays:  30,
	}
}

// NewService creates a new edge function service
func NewService(repo Repository, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}

	return &Service{
		repo:      repo,
		providers: make(map[RuntimeType]RuntimeProvider),
		config:    config,
	}
}

// RegisterProvider registers a runtime provider
func (s *Service) RegisterProvider(provider RuntimeProvider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.providers[provider.GetType()] = provider
}

// SetCompiler sets the code compiler
func (s *Service) SetCompiler(compiler CodeCompiler) {
	s.compiler = compiler
}

// SetRouter sets the edge router
func (s *Service) SetRouter(router EdgeRouter) {
	s.router = router
}

// CreateFunction creates a new edge function
func (s *Service) CreateFunction(ctx context.Context, tenantID string, req *CreateFunctionRequest) (*EdgeFunction, error) {
	// Validate code size
	if len(req.Code) > s.config.MaxCodeSizeKB*1024 {
		return nil, fmt.Errorf("code size exceeds limit of %dKB", s.config.MaxCodeSizeKB)
	}

	// Check function limit
	existing, _, err := s.repo.ListFunctions(ctx, tenantID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing functions: %w", err)
	}
	if len(existing) >= s.config.MaxFunctionsPerTenant {
		return nil, fmt.Errorf("maximum functions reached: %d", s.config.MaxFunctionsPerTenant)
	}

	// Validate runtime
	s.mu.RLock()
	provider, ok := s.providers[req.Runtime]
	s.mu.RUnlock()
	if !ok {
		// Use local provider as fallback for validation
		provider = NewLocalProvider()
	}

	// Validate code
	if err := provider.ValidateCode(ctx, req.Code); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidCode, err)
	}

	// Set defaults
	regions := req.Regions
	if len(regions) == 0 {
		regions = s.config.DefaultRegions
	}

	config := req.Config
	if config == nil {
		config = DefaultFunctionConfig()
	}

	entryPoint := req.EntryPoint
	if entryPoint == "" {
		entryPoint = "handler"
	}

	now := time.Now()
	fn := &EdgeFunction{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		Runtime:     req.Runtime,
		Status:      FunctionStatusDraft,
		Version:     1,
		Code:        req.Code,
		EntryPoint:  entryPoint,
		Regions:     regions,
		Config:      config,
		Triggers:    req.Triggers,
		Environment: req.Environment,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Compile code if compiler is available
	if s.compiler != nil {
		compiled, err := s.compiler.Compile(ctx, req.Code, req.Runtime)
		if err != nil {
			return nil, fmt.Errorf("compilation failed: %w", err)
		}
		fn.CompiledCode = compiled
	}

	// Save function
	if err := s.repo.CreateFunction(ctx, fn); err != nil {
		return nil, fmt.Errorf("failed to create function: %w", err)
	}

	// Auto-deploy if requested
	if req.AutoDeploy {
		go s.deployFunctionAsync(context.Background(), fn)
	}

	return fn, nil
}

// deployFunctionAsync deploys a function asynchronously
func (s *Service) deployFunctionAsync(ctx context.Context, fn *EdgeFunction) {
	deployment, err := s.DeployFunction(ctx, fn.TenantID, fn.ID)
	if err != nil {
		fn.Status = FunctionStatusFailed
		_ = s.repo.UpdateFunction(ctx, fn)
		return
	}

	if deployment.Status == FunctionStatusActive {
		fn.Status = FunctionStatusActive
		fn.LastDeployedAt = deployment.CompletedAt
		fn.DeploymentURLs = deployment.DeploymentURLs
		_ = s.repo.UpdateFunction(ctx, fn)
	}
}

// GetFunction retrieves an edge function
func (s *Service) GetFunction(ctx context.Context, tenantID, functionID string) (*EdgeFunction, error) {
	return s.repo.GetFunction(ctx, tenantID, functionID)
}

// UpdateFunction updates an edge function
func (s *Service) UpdateFunction(ctx context.Context, tenantID, functionID string, req *UpdateFunctionRequest) (*EdgeFunction, error) {
	fn, err := s.repo.GetFunction(ctx, tenantID, functionID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.Name != nil {
		fn.Name = *req.Name
	}
	if req.Description != nil {
		fn.Description = *req.Description
	}
	if req.Code != nil {
		if len(*req.Code) > s.config.MaxCodeSizeKB*1024 {
			return nil, fmt.Errorf("code size exceeds limit of %dKB", s.config.MaxCodeSizeKB)
		}
		fn.Code = *req.Code
		fn.Version++

		// Recompile
		if s.compiler != nil {
			compiled, err := s.compiler.Compile(ctx, *req.Code, fn.Runtime)
			if err != nil {
				return nil, fmt.Errorf("compilation failed: %w", err)
			}
			fn.CompiledCode = compiled
		}
	}
	if req.EntryPoint != nil {
		fn.EntryPoint = *req.EntryPoint
	}
	if len(req.Regions) > 0 {
		fn.Regions = req.Regions
	}
	if req.Config != nil {
		fn.Config = req.Config
	}
	if req.Triggers != nil {
		fn.Triggers = req.Triggers
	}
	if req.Environment != nil {
		fn.Environment = req.Environment
	}
	if req.Status != nil {
		fn.Status = *req.Status
	}

	fn.UpdatedAt = time.Now()

	if err := s.repo.UpdateFunction(ctx, fn); err != nil {
		return nil, err
	}

	return fn, nil
}

// DeleteFunction deletes an edge function
func (s *Service) DeleteFunction(ctx context.Context, tenantID, functionID string) error {
	fn, err := s.repo.GetFunction(ctx, tenantID, functionID)
	if err != nil {
		return err
	}

	// Undeploy from all providers
	s.mu.RLock()
	provider, ok := s.providers[fn.Runtime]
	s.mu.RUnlock()
	if ok && fn.Status == FunctionStatusActive {
		_ = provider.Undeploy(ctx, fn)
	}

	return s.repo.DeleteFunction(ctx, tenantID, functionID)
}

// ListFunctions lists edge functions
func (s *Service) ListFunctions(ctx context.Context, tenantID string, filters *FunctionFilters) (*ListFunctionsResponse, error) {
	functions, total, err := s.repo.ListFunctions(ctx, tenantID, filters)
	if err != nil {
		return nil, err
	}

	page := 1
	pageSize := 20
	if filters != nil {
		if filters.Page > 0 {
			page = filters.Page
		}
		if filters.PageSize > 0 {
			pageSize = filters.PageSize
		}
	}

	return &ListFunctionsResponse{
		Functions:  functions,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
	}, nil
}

// DeployFunction deploys a function to edge locations
func (s *Service) DeployFunction(ctx context.Context, tenantID, functionID string) (*FunctionDeployment, error) {
	fn, err := s.repo.GetFunction(ctx, tenantID, functionID)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	provider, ok := s.providers[fn.Runtime]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrRuntimeNotSupported, fn.Runtime)
	}

	// Update status to deploying
	fn.Status = FunctionStatusDeploying
	_ = s.repo.UpdateFunction(ctx, fn)

	// Create deployment record
	deployment := &FunctionDeployment{
		ID:         uuid.New().String(),
		FunctionID: fn.ID,
		TenantID:   fn.TenantID,
		Version:    fn.Version,
		Status:     FunctionStatusDeploying,
		Runtime:    fn.Runtime,
		Regions:    fn.Regions,
		StartedAt:  time.Now(),
	}
	_ = s.repo.CreateDeployment(ctx, deployment)

	// Deploy
	result, err := provider.Deploy(ctx, fn)
	if err != nil {
		deployment.Status = FunctionStatusFailed
		deployment.ErrorMessage = err.Error()
		completedAt := time.Now()
		deployment.CompletedAt = &completedAt
		deployment.Duration = completedAt.Sub(deployment.StartedAt).Milliseconds()
		_ = s.repo.UpdateDeployment(ctx, deployment)

		fn.Status = FunctionStatusFailed
		_ = s.repo.UpdateFunction(ctx, fn)

		return deployment, err
	}

	// Update deployment
	deployment.Status = FunctionStatusActive
	deployment.DeploymentURLs = result.DeploymentURLs
	completedAt := time.Now()
	deployment.CompletedAt = &completedAt
	deployment.Duration = completedAt.Sub(deployment.StartedAt).Milliseconds()
	_ = s.repo.UpdateDeployment(ctx, deployment)

	// Update function
	fn.Status = FunctionStatusActive
	fn.LastDeployedAt = &completedAt
	fn.DeploymentURLs = result.DeploymentURLs
	_ = s.repo.UpdateFunction(ctx, fn)

	// Update router
	if s.router != nil {
		_ = s.router.UpdateRoutes(ctx, tenantID)
	}

	return deployment, nil
}

// InvokeFunction invokes an edge function
func (s *Service) InvokeFunction(ctx context.Context, tenantID, functionID string, req *InvokeFunctionRequest) (*InvokeFunctionResponse, error) {
	fn, err := s.repo.GetFunction(ctx, tenantID, functionID)
	if err != nil {
		return nil, err
	}

	if fn.Status != FunctionStatusActive {
		return nil, fmt.Errorf("function is not active: %s", fn.Status)
	}

	s.mu.RLock()
	provider, ok := s.providers[fn.Runtime]
	s.mu.RUnlock()
	if !ok {
		// Fall back to local provider
		provider = NewLocalProvider()
	}

	// Determine region
	region := req.Region
	if region == "" {
		region = fn.Regions[0]
	}

	// Check if region is available
	available := false
	for _, r := range fn.Regions {
		if r == region {
			available = true
			break
		}
	}
	if !available {
		return nil, fmt.Errorf("%w: %s", ErrRegionNotAvailable, region)
	}

	// Invoke
	startTime := time.Now()
	response, err := provider.Invoke(ctx, fn, req.Input, region)
	duration := time.Since(startTime).Milliseconds()

	// Record invocation
	invocation := &FunctionInvocation{
		ID:         uuid.New().String(),
		FunctionID: fn.ID,
		TenantID:   fn.TenantID,
		Region:     region,
		TriggerType: TriggerTypeHTTPRoute,
		Input:      req.Input,
		DurationMs: duration,
		InvokedAt:  startTime,
	}

	if err != nil {
		invocation.Status = "error"
		invocation.ErrorMessage = err.Error()
		_ = s.repo.CreateInvocation(ctx, invocation)
		_ = s.repo.IncrementCounters(ctx, fn.ID, 1, 1)
		return nil, err
	}

	invocation.Status = "success"
	invocation.Output = response.Output
	invocation.CacheHit = response.CacheHit
	_ = s.repo.CreateInvocation(ctx, invocation)
	_ = s.repo.IncrementCounters(ctx, fn.ID, 1, 0)

	// Update last invoked
	now := time.Now()
	fn.LastInvokedAt = &now
	fn.InvocationCount++
	_ = s.repo.UpdateFunction(ctx, fn)

	response.InvocationID = invocation.ID
	response.DurationMs = duration

	return response, nil
}

// GetLogs retrieves function logs
func (s *Service) GetLogs(ctx context.Context, tenantID, functionID string, since time.Time, limit int) ([]LogEntry, error) {
	fn, err := s.repo.GetFunction(ctx, tenantID, functionID)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	provider, ok := s.providers[fn.Runtime]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrRuntimeNotSupported, fn.Runtime)
	}

	return provider.GetLogs(ctx, fn, since, limit)
}

// GetMetrics retrieves function metrics
func (s *Service) GetMetrics(ctx context.Context, tenantID, functionID, period string) (*FunctionMetrics, error) {
	_, err := s.repo.GetFunction(ctx, tenantID, functionID)
	if err != nil {
		return nil, err
	}

	return s.repo.GetMetrics(ctx, functionID, period)
}

// GetDeployments retrieves function deployments
func (s *Service) GetDeployments(ctx context.Context, tenantID, functionID string, limit int) ([]FunctionDeployment, error) {
	_, err := s.repo.GetFunction(ctx, tenantID, functionID)
	if err != nil {
		return nil, err
	}

	return s.repo.ListDeployments(ctx, functionID, limit)
}

// LocalProvider implements a local runtime provider for testing
type LocalProvider struct{}

// NewLocalProvider creates a new local provider
func NewLocalProvider() *LocalProvider {
	return &LocalProvider{}
}

// GetType returns the runtime type
func (p *LocalProvider) GetType() RuntimeType {
	return RuntimeLocal
}

// Deploy deploys locally (no-op)
func (p *LocalProvider) Deploy(ctx context.Context, fn *EdgeFunction) (*FunctionDeployment, error) {
	now := time.Now()
	return &FunctionDeployment{
		Status:      FunctionStatusActive,
		CompletedAt: &now,
		DeploymentURLs: map[EdgeRegion]string{
			RegionUSEast: fmt.Sprintf("http://localhost:8080/edge/%s", fn.ID),
		},
	}, nil
}

// Undeploy undeploys locally (no-op)
func (p *LocalProvider) Undeploy(ctx context.Context, fn *EdgeFunction) error {
	return nil
}

// Invoke executes locally
func (p *LocalProvider) Invoke(ctx context.Context, fn *EdgeFunction, input []byte, region EdgeRegion) (*InvokeFunctionResponse, error) {
	// Simple echo for local testing
	return &InvokeFunctionResponse{
		Output:     json.RawMessage(input),
		Status:     "success",
		DurationMs: 1,
		Region:     region,
		CacheHit:   false,
	}, nil
}

// GetLogs returns empty logs for local
func (p *LocalProvider) GetLogs(ctx context.Context, fn *EdgeFunction, since time.Time, limit int) ([]LogEntry, error) {
	return []LogEntry{}, nil
}

// ValidateCode validates code syntax
func (p *LocalProvider) ValidateCode(ctx context.Context, code string) error {
	if len(code) == 0 {
		return fmt.Errorf("code cannot be empty")
	}
	return nil
}

// GetSupportedRegions returns supported regions
func (p *LocalProvider) GetSupportedRegions() []EdgeRegion {
	return AllEdgeRegions()
}

// HealthCheck returns healthy
func (p *LocalProvider) HealthCheck(ctx context.Context) error {
	return nil
}
