package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/repository"
	"github.com/josedab/waas/pkg/utils"
)

// EdgeFunctionsService handles edge function operations
type EdgeFunctionsService struct {
	repo   repository.EdgeFunctionsRepository
	logger *utils.Logger
}

// NewEdgeFunctionsService creates a new edge functions service
func NewEdgeFunctionsService(repo repository.EdgeFunctionsRepository, logger *utils.Logger) *EdgeFunctionsService {
	return &EdgeFunctionsService{
		repo:   repo,
		logger: logger,
	}
}

// CreateFunction creates a new edge function
func (s *EdgeFunctionsService) CreateFunction(ctx context.Context, tenantID uuid.UUID, req *models.CreateEdgeFunctionRequest) (*models.EdgeFunction, error) {
	runtime := req.Runtime
	if runtime == "" {
		runtime = models.RuntimeJavaScript
	}

	validRuntimes := map[string]bool{
		models.RuntimeJavaScript: true,
		models.RuntimeTypeScript: true,
		models.RuntimePython:     true,
	}

	if !validRuntimes[runtime] {
		return nil, fmt.Errorf("invalid runtime: %s", runtime)
	}

	entryPoint := req.EntryPoint
	if entryPoint == "" {
		entryPoint = "handler"
	}

	timeoutMs := req.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = 5000
	}
	if timeoutMs > 30000 {
		timeoutMs = 30000 // Max 30 seconds
	}

	memoryMb := req.MemoryMb
	if memoryMb <= 0 {
		memoryMb = 128
	}
	if memoryMb > 512 {
		memoryMb = 512 // Max 512 MB
	}

	fn := &models.EdgeFunction{
		TenantID:        tenantID,
		Name:            req.Name,
		Description:     req.Description,
		Runtime:         runtime,
		Code:            req.Code,
		EntryPoint:      entryPoint,
		Status:          models.FunctionStatusDraft,
		TimeoutMs:       timeoutMs,
		MemoryMb:        memoryMb,
		EnvironmentVars: req.EnvironmentVars,
		Dependencies:    req.Dependencies,
		Metadata:        make(map[string]interface{}),
	}

	if fn.EnvironmentVars == nil {
		fn.EnvironmentVars = make(map[string]string)
	}

	if err := s.repo.CreateFunction(ctx, fn); err != nil {
		return nil, fmt.Errorf("failed to create function: %w", err)
	}

	// Create initial version
	s.createVersion(ctx, fn, "Initial version", "")

	s.logger.Info("Edge function created", map[string]interface{}{"function_id": fn.ID, "name": fn.Name})

	return fn, nil
}

// GetFunction retrieves a function
func (s *EdgeFunctionsService) GetFunction(ctx context.Context, tenantID, functionID uuid.UUID) (*models.EdgeFunction, error) {
	fn, err := s.repo.GetFunction(ctx, functionID)
	if err != nil {
		return nil, err
	}

	if fn.TenantID != tenantID {
		return nil, fmt.Errorf("function not found")
	}

	return fn, nil
}

// GetFunctions retrieves all functions for a tenant
func (s *EdgeFunctionsService) GetFunctions(ctx context.Context, tenantID uuid.UUID) ([]*models.EdgeFunction, error) {
	return s.repo.GetFunctionsByTenant(ctx, tenantID)
}

// GetFunctionWithDetails retrieves a function with deployments and triggers
func (s *EdgeFunctionsService) GetFunctionWithDetails(ctx context.Context, tenantID, functionID uuid.UUID) (*models.EdgeFunctionWithDeployments, error) {
	fn, err := s.GetFunction(ctx, tenantID, functionID)
	if err != nil {
		return nil, err
	}

	deployments, _ := s.repo.GetDeploymentsByFunction(ctx, functionID)
	triggers, _ := s.repo.GetTriggersByFunction(ctx, functionID)

	return &models.EdgeFunctionWithDeployments{
		EdgeFunction: fn,
		Deployments:  deployments,
		Triggers:     triggers,
	}, nil
}

// UpdateFunction updates a function
func (s *EdgeFunctionsService) UpdateFunction(ctx context.Context, tenantID, functionID uuid.UUID, req *models.UpdateEdgeFunctionRequest) (*models.EdgeFunction, error) {
	fn, err := s.GetFunction(ctx, tenantID, functionID)
	if err != nil {
		return nil, err
	}

	if req.Code != "" {
		fn.Code = req.Code
	}
	if req.EntryPoint != "" {
		fn.EntryPoint = req.EntryPoint
	}
	if req.TimeoutMs > 0 {
		fn.TimeoutMs = req.TimeoutMs
		if fn.TimeoutMs > 30000 {
			fn.TimeoutMs = 30000
		}
	}
	if req.MemoryMb > 0 {
		fn.MemoryMb = req.MemoryMb
		if fn.MemoryMb > 512 {
			fn.MemoryMb = 512
		}
	}
	if req.EnvironmentVars != nil {
		fn.EnvironmentVars = req.EnvironmentVars
	}
	if req.Dependencies != nil {
		fn.Dependencies = req.Dependencies
	}

	if err := s.repo.UpdateFunction(ctx, fn); err != nil {
		return nil, fmt.Errorf("failed to update function: %w", err)
	}

	// Create new version
	s.createVersion(ctx, fn, req.ChangeLog, "")

	// Reset status to draft if active
	if fn.Status == models.FunctionStatusActive {
		s.repo.UpdateFunctionStatus(ctx, functionID, models.FunctionStatusDraft)
		fn.Status = models.FunctionStatusDraft
	}

	return fn, nil
}

// createVersion creates a new function version
func (s *EdgeFunctionsService) createVersion(ctx context.Context, fn *models.EdgeFunction, changeLog, createdBy string) {
	codeHash := sha256.Sum256([]byte(fn.Code))

	version := &models.EdgeFunctionVersion{
		FunctionID: fn.ID,
		Version:    fn.Version,
		Code:       fn.Code,
		EntryPoint: fn.EntryPoint,
		CodeHash:   hex.EncodeToString(codeHash[:]),
		ChangeLog:  changeLog,
		CreatedBy:  createdBy,
	}

	s.repo.CreateVersion(ctx, version)
}

// DeployFunction deploys a function to edge locations
func (s *EdgeFunctionsService) DeployFunction(ctx context.Context, tenantID, functionID uuid.UUID, req *models.DeployEdgeFunctionRequest) ([]*models.EdgeFunctionDeployment, error) {
	fn, err := s.GetFunction(ctx, tenantID, functionID)
	if err != nil {
		return nil, err
	}

	// Validate code before deployment
	if err := s.validateCode(fn); err != nil {
		return nil, fmt.Errorf("code validation failed: %w", err)
	}

	// Update status to deploying
	s.repo.UpdateFunctionStatus(ctx, functionID, models.FunctionStatusDeploying)

	var deployments []*models.EdgeFunctionDeployment

	for _, locID := range req.LocationIDs {
		locationID, err := uuid.Parse(locID)
		if err != nil {
			continue
		}

		location, err := s.repo.GetLocation(ctx, locationID)
		if err != nil {
			continue
		}

		deployment := &models.EdgeFunctionDeployment{
			FunctionID:   functionID,
			LocationID:   locationID,
			Version:      fn.Version,
			Status:       models.DeploymentStatusPending,
			HealthStatus: models.HealthStatusUnknown,
		}

		if err := s.repo.CreateDeployment(ctx, deployment); err != nil {
			continue
		}

		// Simulate deployment (in production, this would call the edge provider)
		deploymentURL := fmt.Sprintf("https://%s.edge.waas.io/fn/%s", location.Code, fn.Name)
		s.repo.UpdateDeploymentStatus(ctx, deployment.ID, models.DeploymentStatusActive, deploymentURL)
		deployment.Status = models.DeploymentStatusActive
		deployment.DeploymentURL = deploymentURL
		deployment.Location = location

		deployments = append(deployments, deployment)

		s.logger.Info("Function deployed", map[string]interface{}{"function_id": functionID, "location": location.Code})
	}

	if len(deployments) > 0 {
		s.repo.UpdateFunctionStatus(ctx, functionID, models.FunctionStatusActive)
	} else {
		s.repo.UpdateFunctionStatus(ctx, functionID, models.FunctionStatusFailed)
		return nil, fmt.Errorf("deployment failed to all locations")
	}

	return deployments, nil
}

// validateCode validates function code
func (s *EdgeFunctionsService) validateCode(fn *models.EdgeFunction) error {
	switch fn.Runtime {
	case models.RuntimeJavaScript, models.RuntimeTypeScript:
		return s.validateJavaScript(fn.Code, fn.EntryPoint)
	case models.RuntimePython:
		// Python validation would require a Python runtime
		return nil
	}
	return fmt.Errorf("unsupported runtime: %s", fn.Runtime)
}

// validateJavaScript validates JavaScript code
func (s *EdgeFunctionsService) validateJavaScript(code, entryPoint string) error {
	vm := goja.New()

	// Compile the code
	_, err := vm.RunString(code)
	if err != nil {
		return fmt.Errorf("syntax error: %w", err)
	}

	// Check if entry point exists
	handler := vm.Get(entryPoint)
	if handler == nil || handler.ExportType() == nil {
		return fmt.Errorf("entry point '%s' not found", entryPoint)
	}

	return nil
}

// InvokeFunction invokes a function
func (s *EdgeFunctionsService) InvokeFunction(ctx context.Context, tenantID, functionID uuid.UUID, req *models.InvokeFunctionRequest) (*models.FunctionExecutionResult, error) {
	fn, err := s.GetFunction(ctx, tenantID, functionID)
	if err != nil {
		return nil, err
	}

	if fn.Status != models.FunctionStatusActive {
		return nil, fmt.Errorf("function is not active")
	}

	// Create invocation record
	inputBytes, _ := json.Marshal(req.Input)
	invocation := &models.EdgeFunctionInvocation{
		FunctionID:     functionID,
		TenantID:       tenantID,
		Status:         "running",
		InputSizeBytes: len(inputBytes),
		ColdStart:      true, // For simplicity, assume cold start
	}

	if req.EventID != "" {
		eventID, _ := uuid.Parse(req.EventID)
		invocation.EventID = &eventID
	}

	if req.EndpointID != "" {
		endpointID, _ := uuid.Parse(req.EndpointID)
		invocation.EndpointID = &endpointID
	}

	s.repo.CreateInvocation(ctx, invocation)

	// Execute the function
	startTime := time.Now()
	result := s.executeFunction(fn, req.Input)
	duration := int(time.Since(startTime).Milliseconds())

	// Update invocation
	status := models.InvocationStatusSuccess
	errorMsg := ""
	if !result.Success {
		status = models.InvocationStatusError
		errorMsg = result.Error
	}
	s.repo.CompleteInvocation(ctx, invocation.ID, status, duration, result.MemoryUsed, errorMsg)

	result.DurationMs = duration

	s.logger.Info("Function invoked", map[string]interface{}{"function_id": functionID, "duration_ms": duration, "success": result.Success})

	return result, nil
}

// executeFunction executes function code
func (s *EdgeFunctionsService) executeFunction(fn *models.EdgeFunction, input map[string]interface{}) *models.FunctionExecutionResult {
	result := &models.FunctionExecutionResult{
		Success:    false,
		MemoryUsed: 10, // Placeholder
		Logs:       []string{},
	}

	switch fn.Runtime {
	case models.RuntimeJavaScript, models.RuntimeTypeScript:
		return s.executeJavaScript(fn, input)
	case models.RuntimePython:
		result.Error = "Python runtime not yet implemented"
		return result
	default:
		result.Error = "Unsupported runtime"
		return result
	}
}

// executeJavaScript executes JavaScript code
func (s *EdgeFunctionsService) executeJavaScript(fn *models.EdgeFunction, input map[string]interface{}) *models.FunctionExecutionResult {
	result := &models.FunctionExecutionResult{
		Success:    false,
		MemoryUsed: 10,
		Logs:       []string{},
	}

	vm := goja.New()

	// Set up console.log
	logs := []string{}
	console := vm.NewObject()
	console.Set("log", func(call goja.FunctionCall) goja.Value {
		args := make([]interface{}, len(call.Arguments))
		for i, arg := range call.Arguments {
			args[i] = arg.Export()
		}
		logMsg := fmt.Sprint(args...)
		logs = append(logs, logMsg)
		return goja.Undefined()
	})
	vm.Set("console", console)

	// Set environment variables
	envObj := vm.NewObject()
	for k, v := range fn.EnvironmentVars {
		envObj.Set(k, v)
	}
	vm.Set("env", envObj)

	// Run the code
	_, err := vm.RunString(fn.Code)
	if err != nil {
		result.Error = fmt.Sprintf("execution error: %v", err)
		result.Logs = logs
		return result
	}

	// Get the handler function
	handler := vm.Get(fn.EntryPoint)
	if handler == nil {
		result.Error = fmt.Sprintf("entry point '%s' not found", fn.EntryPoint)
		result.Logs = logs
		return result
	}

	callable, ok := goja.AssertFunction(handler)
	if !ok {
		result.Error = fmt.Sprintf("entry point '%s' is not a function", fn.EntryPoint)
		result.Logs = logs
		return result
	}

	// Call the handler with input
	inputValue := vm.ToValue(input)
	returnValue, err := callable(goja.Undefined(), inputValue)
	if err != nil {
		result.Error = fmt.Sprintf("handler error: %v", err)
		result.Logs = logs
		return result
	}

	// Export the result
	if returnValue != nil && returnValue != goja.Undefined() {
		exported := returnValue.Export()
		if outputMap, ok := exported.(map[string]interface{}); ok {
			result.Output = outputMap
		} else {
			result.Output = map[string]interface{}{"result": exported}
		}
	}

	result.Success = true
	result.Logs = logs

	return result
}

// CreateTrigger creates a function trigger
func (s *EdgeFunctionsService) CreateTrigger(ctx context.Context, tenantID, functionID uuid.UUID, req *models.CreateTriggerRequest) (*models.EdgeFunctionTrigger, error) {
	_, err := s.GetFunction(ctx, tenantID, functionID)
	if err != nil {
		return nil, err
	}

	validTriggerTypes := map[string]bool{
		models.TriggerPreSend:      true,
		models.TriggerPostReceive:  true,
		models.TriggerTransform:    true,
		models.TriggerAuthenticate: true,
		models.TriggerEnrich:       true,
	}

	if !validTriggerTypes[req.TriggerType] {
		return nil, fmt.Errorf("invalid trigger_type: %s", req.TriggerType)
	}

	var endpointIDs []uuid.UUID
	for _, ep := range req.EndpointIDs {
		id, err := uuid.Parse(ep)
		if err == nil {
			endpointIDs = append(endpointIDs, id)
		}
	}

	trigger := &models.EdgeFunctionTrigger{
		FunctionID:  functionID,
		TriggerType: req.TriggerType,
		EventTypes:  req.EventTypes,
		EndpointIDs: endpointIDs,
		Conditions:  req.Conditions,
		Priority:    req.Priority,
		Enabled:     true,
	}

	if trigger.Conditions == nil {
		trigger.Conditions = make(map[string]interface{})
	}

	if err := s.repo.CreateTrigger(ctx, trigger); err != nil {
		return nil, fmt.Errorf("failed to create trigger: %w", err)
	}

	return trigger, nil
}

// GetTriggers retrieves triggers for a function
func (s *EdgeFunctionsService) GetTriggers(ctx context.Context, tenantID, functionID uuid.UUID) ([]*models.EdgeFunctionTrigger, error) {
	_, err := s.GetFunction(ctx, tenantID, functionID)
	if err != nil {
		return nil, err
	}

	return s.repo.GetTriggersByFunction(ctx, functionID)
}

// GetLocations retrieves all edge locations
func (s *EdgeFunctionsService) GetLocations(ctx context.Context) ([]*models.EdgeLocation, error) {
	return s.repo.GetAllLocations(ctx)
}

// GetActiveLocations retrieves active edge locations
func (s *EdgeFunctionsService) GetActiveLocations(ctx context.Context) ([]*models.EdgeLocation, error) {
	return s.repo.GetActiveLocations(ctx)
}

// RunTest runs a function test
func (s *EdgeFunctionsService) RunTest(ctx context.Context, tenantID, functionID uuid.UUID, req *models.RunTestRequest) (*models.EdgeFunctionTest, error) {
	fn, err := s.GetFunction(ctx, tenantID, functionID)
	if err != nil {
		return nil, err
	}

	// Execute the function with test input
	startTime := time.Now()
	result := s.executeFunction(fn, req.Input)
	duration := int(time.Since(startTime).Milliseconds())

	test := &models.EdgeFunctionTest{
		FunctionID:     functionID,
		TestName:       req.TestName,
		InputPayload:   req.Input,
		ExpectedOutput: req.ExpectedOutput,
		ActualOutput:   result.Output,
		DurationMs:     duration,
	}

	if !result.Success {
		test.ErrorMessage = result.Error
		passed := false
		test.Passed = &passed
	} else {
		// Compare outputs if expected is provided
		if req.ExpectedOutput != nil {
			passed := s.compareOutputs(result.Output, req.ExpectedOutput)
			test.Passed = &passed
		} else {
			passed := true
			test.Passed = &passed
		}
	}

	if err := s.repo.CreateTest(ctx, test); err != nil {
		return nil, fmt.Errorf("failed to save test: %w", err)
	}

	return test, nil
}

// compareOutputs compares actual and expected outputs
func (s *EdgeFunctionsService) compareOutputs(actual, expected map[string]interface{}) bool {
	actualJSON, _ := json.Marshal(actual)
	expectedJSON, _ := json.Marshal(expected)
	return string(actualJSON) == string(expectedJSON)
}

// GetVersions retrieves function versions
func (s *EdgeFunctionsService) GetVersions(ctx context.Context, tenantID, functionID uuid.UUID) ([]*models.EdgeFunctionVersion, error) {
	_, err := s.GetFunction(ctx, tenantID, functionID)
	if err != nil {
		return nil, err
	}

	return s.repo.GetVersions(ctx, functionID)
}

// RollbackFunction rolls back to a previous version
func (s *EdgeFunctionsService) RollbackFunction(ctx context.Context, tenantID, functionID uuid.UUID, targetVersion int) (*models.EdgeFunction, error) {
	fn, err := s.GetFunction(ctx, tenantID, functionID)
	if err != nil {
		return nil, err
	}

	version, err := s.repo.GetVersion(ctx, functionID, targetVersion)
	if err != nil {
		return nil, fmt.Errorf("version %d not found", targetVersion)
	}

	fn.Code = version.Code
	fn.EntryPoint = version.EntryPoint

	if err := s.repo.UpdateFunction(ctx, fn); err != nil {
		return nil, err
	}

	// Create rollback version
	s.createVersion(ctx, fn, fmt.Sprintf("Rollback to version %d", targetVersion), "")

	// Re-deploy if was active
	if fn.Status == models.FunctionStatusActive {
		s.repo.UpdateFunctionStatus(ctx, functionID, models.FunctionStatusDraft)
		fn.Status = models.FunctionStatusDraft
	}

	s.logger.Info("Function rolled back", map[string]interface{}{"function_id": functionID, "to_version": targetVersion})

	return fn, nil
}

// GetDeployments retrieves deployments for a function
func (s *EdgeFunctionsService) GetDeployments(ctx context.Context, tenantID, functionID uuid.UUID) ([]*models.EdgeFunctionDeployment, error) {
	_, err := s.GetFunction(ctx, tenantID, functionID)
	if err != nil {
		return nil, err
	}

	return s.repo.GetDeploymentsByFunction(ctx, functionID)
}

// GetInvocations retrieves invocations for a function
func (s *EdgeFunctionsService) GetInvocations(ctx context.Context, tenantID, functionID uuid.UUID, limit int) ([]*models.EdgeFunctionInvocation, error) {
	_, err := s.GetFunction(ctx, tenantID, functionID)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 50
	}

	return s.repo.GetInvocationsByFunction(ctx, functionID, limit)
}

// GetDashboard retrieves the edge functions dashboard
func (s *EdgeFunctionsService) GetDashboard(ctx context.Context, tenantID uuid.UUID) (*models.EdgeFunctionDashboard, error) {
	dashboard := &models.EdgeFunctionDashboard{
		FunctionsByRuntime: make(map[string]int),
		LocationCoverage:   make(map[string]int),
	}

	var err error

	dashboard.TotalFunctions, err = s.repo.CountFunctions(ctx, tenantID)
	if err != nil {
		s.logger.Warn("Failed to count functions for dashboard", map[string]interface{}{"tenant_id": tenantID, "error": err.Error()})
	}
	dashboard.ActiveFunctions, err = s.repo.CountActiveFunctions(ctx, tenantID)
	if err != nil {
		s.logger.Warn("Failed to count active functions for dashboard", map[string]interface{}{"tenant_id": tenantID, "error": err.Error()})
	}
	dashboard.TotalDeployments, err = s.repo.CountDeployments(ctx, tenantID)
	if err != nil {
		s.logger.Warn("Failed to count deployments for dashboard", map[string]interface{}{"tenant_id": tenantID, "error": err.Error()})
	}

	since := time.Now().Add(-24 * time.Hour)
	dashboard.TotalInvocations, err = s.repo.CountInvocations(ctx, tenantID, since)
	if err != nil {
		s.logger.Warn("Failed to count invocations for dashboard", map[string]interface{}{"tenant_id": tenantID, "error": err.Error()})
	}
	dashboard.ErrorRate, err = s.repo.GetErrorRate(ctx, tenantID, since)
	if err != nil {
		s.logger.Warn("Failed to get error rate for dashboard", map[string]interface{}{"tenant_id": tenantID, "error": err.Error()})
	}

	recentInvocations, err := s.repo.GetRecentInvocations(ctx, tenantID, 10)
	if err != nil {
		s.logger.Warn("Failed to get recent invocations for dashboard", map[string]interface{}{"tenant_id": tenantID, "error": err.Error()})
	}
	dashboard.RecentInvocations = recentInvocations

	// Calculate avg duration
	if len(recentInvocations) > 0 {
		totalDuration := 0
		for _, inv := range recentInvocations {
			totalDuration += inv.DurationMs
		}
		dashboard.AvgDurationMs = float64(totalDuration) / float64(len(recentInvocations))
	}

	// Get runtime distribution
	functions, err := s.repo.GetFunctionsByTenant(ctx, tenantID)
	if err != nil {
		s.logger.Warn("Failed to get functions for dashboard", map[string]interface{}{"tenant_id": tenantID, "error": err.Error()})
	}
	for _, fn := range functions {
		dashboard.FunctionsByRuntime[fn.Runtime]++
	}

	return dashboard, nil
}

// DeleteFunction deletes a function
func (s *EdgeFunctionsService) DeleteFunction(ctx context.Context, tenantID, functionID uuid.UUID) error {
	fn, err := s.GetFunction(ctx, tenantID, functionID)
	if err != nil {
		return err
	}

	if fn.Status == models.FunctionStatusActive {
		return fmt.Errorf("cannot delete active function, deprecate first")
	}

	return s.repo.DeleteFunction(ctx, functionID)
}
