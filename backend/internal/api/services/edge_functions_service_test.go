package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Mock EdgeFunctionsRepository ---
type MockEdgeFunctionsRepo struct {
	mock.Mock
}

func (m *MockEdgeFunctionsRepo) CreateFunction(ctx context.Context, fn *models.EdgeFunction) error {
	return m.Called(ctx, fn).Error(0)
}
func (m *MockEdgeFunctionsRepo) GetFunction(ctx context.Context, id uuid.UUID) (*models.EdgeFunction, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.EdgeFunction), args.Error(1)
}
func (m *MockEdgeFunctionsRepo) GetFunctionByName(ctx context.Context, tenantID uuid.UUID, name string) (*models.EdgeFunction, error) {
	args := m.Called(ctx, tenantID, name)
	return args.Get(0).(*models.EdgeFunction), args.Error(1)
}
func (m *MockEdgeFunctionsRepo) GetFunctionsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.EdgeFunction, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*models.EdgeFunction), args.Error(1)
}
func (m *MockEdgeFunctionsRepo) GetActiveFunctions(ctx context.Context, tenantID uuid.UUID) ([]*models.EdgeFunction, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*models.EdgeFunction), args.Error(1)
}
func (m *MockEdgeFunctionsRepo) UpdateFunction(ctx context.Context, fn *models.EdgeFunction) error {
	return m.Called(ctx, fn).Error(0)
}
func (m *MockEdgeFunctionsRepo) UpdateFunctionStatus(ctx context.Context, id uuid.UUID, status string) error {
	return m.Called(ctx, id, status).Error(0)
}
func (m *MockEdgeFunctionsRepo) DeleteFunction(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockEdgeFunctionsRepo) CreateVersion(ctx context.Context, version *models.EdgeFunctionVersion) error {
	return m.Called(ctx, version).Error(0)
}
func (m *MockEdgeFunctionsRepo) GetVersions(ctx context.Context, functionID uuid.UUID) ([]*models.EdgeFunctionVersion, error) {
	args := m.Called(ctx, functionID)
	return args.Get(0).([]*models.EdgeFunctionVersion), args.Error(1)
}
func (m *MockEdgeFunctionsRepo) GetVersion(ctx context.Context, functionID uuid.UUID, version int) (*models.EdgeFunctionVersion, error) {
	args := m.Called(ctx, functionID, version)
	return args.Get(0).(*models.EdgeFunctionVersion), args.Error(1)
}
func (m *MockEdgeFunctionsRepo) GetAllLocations(ctx context.Context) ([]*models.EdgeLocation, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*models.EdgeLocation), args.Error(1)
}
func (m *MockEdgeFunctionsRepo) GetLocation(ctx context.Context, id uuid.UUID) (*models.EdgeLocation, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.EdgeLocation), args.Error(1)
}
func (m *MockEdgeFunctionsRepo) GetLocationByCode(ctx context.Context, code string) (*models.EdgeLocation, error) {
	args := m.Called(ctx, code)
	return args.Get(0).(*models.EdgeLocation), args.Error(1)
}
func (m *MockEdgeFunctionsRepo) GetActiveLocations(ctx context.Context) ([]*models.EdgeLocation, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*models.EdgeLocation), args.Error(1)
}
func (m *MockEdgeFunctionsRepo) CreateDeployment(ctx context.Context, deployment *models.EdgeFunctionDeployment) error {
	return m.Called(ctx, deployment).Error(0)
}
func (m *MockEdgeFunctionsRepo) GetDeployment(ctx context.Context, id uuid.UUID) (*models.EdgeFunctionDeployment, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.EdgeFunctionDeployment), args.Error(1)
}
func (m *MockEdgeFunctionsRepo) GetDeploymentsByFunction(ctx context.Context, functionID uuid.UUID) ([]*models.EdgeFunctionDeployment, error) {
	args := m.Called(ctx, functionID)
	return args.Get(0).([]*models.EdgeFunctionDeployment), args.Error(1)
}
func (m *MockEdgeFunctionsRepo) GetActiveDeployment(ctx context.Context, functionID, locationID uuid.UUID) (*models.EdgeFunctionDeployment, error) {
	args := m.Called(ctx, functionID, locationID)
	return args.Get(0).(*models.EdgeFunctionDeployment), args.Error(1)
}
func (m *MockEdgeFunctionsRepo) UpdateDeploymentStatus(ctx context.Context, id uuid.UUID, status string, deploymentURL string) error {
	return m.Called(ctx, id, status, deploymentURL).Error(0)
}
func (m *MockEdgeFunctionsRepo) UpdateDeploymentHealth(ctx context.Context, id uuid.UUID, healthStatus string) error {
	return m.Called(ctx, id, healthStatus).Error(0)
}
func (m *MockEdgeFunctionsRepo) SetDeploymentError(ctx context.Context, id uuid.UUID, errorMsg string) error {
	return m.Called(ctx, id, errorMsg).Error(0)
}
func (m *MockEdgeFunctionsRepo) CreateTrigger(ctx context.Context, trigger *models.EdgeFunctionTrigger) error {
	return m.Called(ctx, trigger).Error(0)
}
func (m *MockEdgeFunctionsRepo) GetTrigger(ctx context.Context, id uuid.UUID) (*models.EdgeFunctionTrigger, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.EdgeFunctionTrigger), args.Error(1)
}
func (m *MockEdgeFunctionsRepo) GetTriggersByFunction(ctx context.Context, functionID uuid.UUID) ([]*models.EdgeFunctionTrigger, error) {
	args := m.Called(ctx, functionID)
	return args.Get(0).([]*models.EdgeFunctionTrigger), args.Error(1)
}
func (m *MockEdgeFunctionsRepo) GetMatchingTriggers(ctx context.Context, tenantID uuid.UUID, triggerType, eventType string, endpointID uuid.UUID) ([]*models.EdgeFunctionTrigger, error) {
	args := m.Called(ctx, tenantID, triggerType, eventType, endpointID)
	return args.Get(0).([]*models.EdgeFunctionTrigger), args.Error(1)
}
func (m *MockEdgeFunctionsRepo) UpdateTrigger(ctx context.Context, trigger *models.EdgeFunctionTrigger) error {
	return m.Called(ctx, trigger).Error(0)
}
func (m *MockEdgeFunctionsRepo) DeleteTrigger(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockEdgeFunctionsRepo) CreateInvocation(ctx context.Context, invocation *models.EdgeFunctionInvocation) error {
	return m.Called(ctx, invocation).Error(0)
}
func (m *MockEdgeFunctionsRepo) GetInvocation(ctx context.Context, id uuid.UUID) (*models.EdgeFunctionInvocation, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.EdgeFunctionInvocation), args.Error(1)
}
func (m *MockEdgeFunctionsRepo) GetInvocationsByFunction(ctx context.Context, functionID uuid.UUID, limit int) ([]*models.EdgeFunctionInvocation, error) {
	args := m.Called(ctx, functionID, limit)
	return args.Get(0).([]*models.EdgeFunctionInvocation), args.Error(1)
}
func (m *MockEdgeFunctionsRepo) GetRecentInvocations(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.EdgeFunctionInvocation, error) {
	args := m.Called(ctx, tenantID, limit)
	return args.Get(0).([]*models.EdgeFunctionInvocation), args.Error(1)
}
func (m *MockEdgeFunctionsRepo) CompleteInvocation(ctx context.Context, id uuid.UUID, status string, durationMs, memoryUsed int, errorMsg string) error {
	return m.Called(ctx, id, status, durationMs, memoryUsed, errorMsg).Error(0)
}
func (m *MockEdgeFunctionsRepo) CreateOrUpdateMetrics(ctx context.Context, metrics *models.EdgeFunctionMetrics) error {
	return m.Called(ctx, metrics).Error(0)
}
func (m *MockEdgeFunctionsRepo) GetMetrics(ctx context.Context, functionID uuid.UUID, since time.Time) ([]*models.EdgeFunctionMetrics, error) {
	args := m.Called(ctx, functionID, since)
	return args.Get(0).([]*models.EdgeFunctionMetrics), args.Error(1)
}
func (m *MockEdgeFunctionsRepo) CreateSecret(ctx context.Context, secret *models.EdgeFunctionSecret) error {
	return m.Called(ctx, secret).Error(0)
}
func (m *MockEdgeFunctionsRepo) GetSecrets(ctx context.Context, functionID uuid.UUID) ([]*models.EdgeFunctionSecret, error) {
	args := m.Called(ctx, functionID)
	return args.Get(0).([]*models.EdgeFunctionSecret), args.Error(1)
}
func (m *MockEdgeFunctionsRepo) DeleteSecret(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockEdgeFunctionsRepo) CreateTest(ctx context.Context, test *models.EdgeFunctionTest) error {
	return m.Called(ctx, test).Error(0)
}
func (m *MockEdgeFunctionsRepo) GetTests(ctx context.Context, functionID uuid.UUID) ([]*models.EdgeFunctionTest, error) {
	args := m.Called(ctx, functionID)
	return args.Get(0).([]*models.EdgeFunctionTest), args.Error(1)
}
func (m *MockEdgeFunctionsRepo) CountFunctions(ctx context.Context, tenantID uuid.UUID) (int, error) {
	args := m.Called(ctx, tenantID)
	return args.Int(0), args.Error(1)
}
func (m *MockEdgeFunctionsRepo) CountActiveFunctions(ctx context.Context, tenantID uuid.UUID) (int, error) {
	args := m.Called(ctx, tenantID)
	return args.Int(0), args.Error(1)
}
func (m *MockEdgeFunctionsRepo) CountDeployments(ctx context.Context, tenantID uuid.UUID) (int, error) {
	args := m.Called(ctx, tenantID)
	return args.Int(0), args.Error(1)
}
func (m *MockEdgeFunctionsRepo) CountInvocations(ctx context.Context, tenantID uuid.UUID, since time.Time) (int64, error) {
	args := m.Called(ctx, tenantID, since)
	return args.Get(0).(int64), args.Error(1)
}
func (m *MockEdgeFunctionsRepo) GetErrorRate(ctx context.Context, tenantID uuid.UUID, since time.Time) (float64, error) {
	args := m.Called(ctx, tenantID, since)
	return args.Get(0).(float64), args.Error(1)
}

// --- Edge Functions Service Tests ---

func TestEdgeFunctionsService_CreateFunction_ValidJS(t *testing.T) {
	t.Parallel()
	repo := &MockEdgeFunctionsRepo{}
	logger := utils.NewLogger("test")
	svc := NewEdgeFunctionsService(repo, logger)

	tenantID := uuid.New()
	req := &models.CreateEdgeFunctionRequest{
		Name:    "transform-payload",
		Code:    `function handler(input) { return { event: input.type }; }`,
		Runtime: models.RuntimeJavaScript,
	}

	repo.On("CreateFunction", mock.Anything, mock.AnythingOfType("*models.EdgeFunction")).Return(nil)
	repo.On("CreateVersion", mock.Anything, mock.AnythingOfType("*models.EdgeFunctionVersion")).Return(nil).Maybe()

	fn, err := svc.CreateFunction(context.Background(), tenantID, req)
	require.NoError(t, err)
	assert.Equal(t, tenantID, fn.TenantID)
	assert.Equal(t, models.FunctionStatusDraft, fn.Status)
	assert.Equal(t, models.RuntimeJavaScript, fn.Runtime)
	assert.Equal(t, "handler", fn.EntryPoint)
}

func TestEdgeFunctionsService_CreateFunction_InvalidRuntime(t *testing.T) {
	t.Parallel()
	repo := &MockEdgeFunctionsRepo{}
	logger := utils.NewLogger("test")
	svc := NewEdgeFunctionsService(repo, logger)

	_, err := svc.CreateFunction(context.Background(), uuid.New(), &models.CreateEdgeFunctionRequest{
		Name:    "test-fn",
		Code:    "some code",
		Runtime: "rust",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid runtime")
}

func TestEdgeFunctionsService_GetFunction_TenantMismatch(t *testing.T) {
	t.Parallel()
	repo := &MockEdgeFunctionsRepo{}
	logger := utils.NewLogger("test")
	svc := NewEdgeFunctionsService(repo, logger)

	functionID := uuid.New()
	otherTenantID := uuid.New()
	requestingTenantID := uuid.New()

	repo.On("GetFunction", mock.Anything, functionID).Return(&models.EdgeFunction{
		TenantID: otherTenantID,
	}, nil)

	_, err := svc.GetFunction(context.Background(), requestingTenantID, functionID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "function not found")
}

func TestEdgeFunctionsService_InvokeFunction_NotActive(t *testing.T) {
	t.Parallel()
	repo := &MockEdgeFunctionsRepo{}
	logger := utils.NewLogger("test")
	svc := NewEdgeFunctionsService(repo, logger)

	tenantID := uuid.New()
	functionID := uuid.New()

	repo.On("GetFunction", mock.Anything, functionID).Return(&models.EdgeFunction{
		TenantID: tenantID,
		Status:   models.FunctionStatusDraft,
	}, nil)

	_, err := svc.InvokeFunction(context.Background(), tenantID, functionID, &models.InvokeFunctionRequest{
		Input: map[string]interface{}{"key": "value"},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "function is not active")
}

func TestEdgeFunctionsService_InvokeFunction_ValidJSHandler(t *testing.T) {
	t.Parallel()
	repo := &MockEdgeFunctionsRepo{}
	logger := utils.NewLogger("test")
	svc := NewEdgeFunctionsService(repo, logger)

	tenantID := uuid.New()
	functionID := uuid.New()

	repo.On("GetFunction", mock.Anything, functionID).Return(&models.EdgeFunction{
		ID:              functionID,
		TenantID:        tenantID,
		Status:          models.FunctionStatusActive,
		Runtime:         models.RuntimeJavaScript,
		Code:            `function handler(input) { return { greeting: "hello " + input.name }; }`,
		EntryPoint:      "handler",
		TimeoutMs:       5000,
		EnvironmentVars: map[string]string{},
	}, nil)
	repo.On("CreateInvocation", mock.Anything, mock.AnythingOfType("*models.EdgeFunctionInvocation")).Return(nil).Maybe()
	repo.On("CompleteInvocation", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	result, err := svc.InvokeFunction(context.Background(), tenantID, functionID, &models.InvokeFunctionRequest{
		Input: map[string]interface{}{"name": "world"},
	})
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "hello world", result.Output["greeting"])
}

func TestEdgeFunctionsService_DeleteFunction_WhenActive(t *testing.T) {
	t.Parallel()
	repo := &MockEdgeFunctionsRepo{}
	logger := utils.NewLogger("test")
	svc := NewEdgeFunctionsService(repo, logger)

	tenantID := uuid.New()
	functionID := uuid.New()

	repo.On("GetFunction", mock.Anything, functionID).Return(&models.EdgeFunction{
		TenantID: tenantID,
		Status:   models.FunctionStatusActive,
	}, nil)

	err := svc.DeleteFunction(context.Background(), tenantID, functionID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete active")
}

func TestEdgeFunctionsService_CreateFunction_RepoError(t *testing.T) {
	t.Parallel()
	repo := &MockEdgeFunctionsRepo{}
	logger := utils.NewLogger("test")
	svc := NewEdgeFunctionsService(repo, logger)

	repo.On("CreateFunction", mock.Anything, mock.Anything).Return(fmt.Errorf("db error"))

	_, err := svc.CreateFunction(context.Background(), uuid.New(), &models.CreateEdgeFunctionRequest{
		Name:    "fail-fn",
		Code:    "code",
		Runtime: models.RuntimeJavaScript,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create function")
}

// --- UpdateFunction Tests ---

func TestEdgeFunctionsService_UpdateFunction_ValidUpdate(t *testing.T) {
	t.Parallel()
	repo := &MockEdgeFunctionsRepo{}
	logger := utils.NewLogger("test")
	svc := NewEdgeFunctionsService(repo, logger)

	tenantID := uuid.New()
	functionID := uuid.New()

	repo.On("GetFunction", mock.Anything, functionID).Return(&models.EdgeFunction{
		ID:              functionID,
		TenantID:        tenantID,
		Code:            "old code",
		EntryPoint:      "handler",
		Status:          models.FunctionStatusDraft,
		Runtime:         models.RuntimeJavaScript,
		EnvironmentVars: map[string]string{},
	}, nil)
	repo.On("UpdateFunction", mock.Anything, mock.AnythingOfType("*models.EdgeFunction")).Return(nil)
	repo.On("CreateVersion", mock.Anything, mock.AnythingOfType("*models.EdgeFunctionVersion")).Return(nil).Maybe()
	repo.On("UpdateFunctionStatus", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	fn, err := svc.UpdateFunction(context.Background(), tenantID, functionID, &models.UpdateEdgeFunctionRequest{
		Code:      "new code",
		ChangeLog: "updated code",
	})
	require.NoError(t, err)
	assert.Equal(t, "new code", fn.Code)
}

func TestEdgeFunctionsService_UpdateFunction_TenantMismatch(t *testing.T) {
	t.Parallel()
	repo := &MockEdgeFunctionsRepo{}
	logger := utils.NewLogger("test")
	svc := NewEdgeFunctionsService(repo, logger)

	functionID := uuid.New()
	ownerTenantID := uuid.New()
	otherTenantID := uuid.New()

	repo.On("GetFunction", mock.Anything, functionID).Return(&models.EdgeFunction{
		TenantID: ownerTenantID,
	}, nil)

	_, err := svc.UpdateFunction(context.Background(), otherTenantID, functionID, &models.UpdateEdgeFunctionRequest{
		Code: "hacked",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "function not found")
}

// --- GetFunctionWithDetails Tests ---

func TestEdgeFunctionsService_GetFunctionWithDetails(t *testing.T) {
	t.Parallel()
	repo := &MockEdgeFunctionsRepo{}
	logger := utils.NewLogger("test")
	svc := NewEdgeFunctionsService(repo, logger)

	tenantID := uuid.New()
	functionID := uuid.New()

	repo.On("GetFunction", mock.Anything, functionID).Return(&models.EdgeFunction{
		ID:       functionID,
		TenantID: tenantID,
		Name:     "my-fn",
	}, nil)
	repo.On("GetDeploymentsByFunction", mock.Anything, functionID).Return([]*models.EdgeFunctionDeployment{
		{ID: uuid.New(), FunctionID: functionID, Status: "active"},
	}, nil)
	repo.On("GetTriggersByFunction", mock.Anything, functionID).Return([]*models.EdgeFunctionTrigger{
		{ID: uuid.New(), FunctionID: functionID, TriggerType: models.TriggerPreSend},
	}, nil)

	result, err := svc.GetFunctionWithDetails(context.Background(), tenantID, functionID)
	require.NoError(t, err)
	assert.Equal(t, "my-fn", result.EdgeFunction.Name)
	assert.Len(t, result.Deployments, 1)
	assert.Len(t, result.Triggers, 1)
}

// --- CreateTrigger Tests ---

func TestEdgeFunctionsService_CreateTrigger_ValidPreSend(t *testing.T) {
	t.Parallel()
	repo := &MockEdgeFunctionsRepo{}
	logger := utils.NewLogger("test")
	svc := NewEdgeFunctionsService(repo, logger)

	tenantID := uuid.New()
	functionID := uuid.New()

	repo.On("GetFunction", mock.Anything, functionID).Return(&models.EdgeFunction{
		ID:       functionID,
		TenantID: tenantID,
	}, nil)
	repo.On("CreateTrigger", mock.Anything, mock.AnythingOfType("*models.EdgeFunctionTrigger")).Return(nil)

	trigger, err := svc.CreateTrigger(context.Background(), tenantID, functionID, &models.CreateTriggerRequest{
		TriggerType: models.TriggerPreSend,
		EventTypes:  []string{"message.created"},
		Priority:    10,
	})
	require.NoError(t, err)
	assert.Equal(t, models.TriggerPreSend, trigger.TriggerType)
	assert.True(t, trigger.Enabled)
	assert.Equal(t, 10, trigger.Priority)
}

func TestEdgeFunctionsService_CreateTrigger_InvalidType(t *testing.T) {
	t.Parallel()
	repo := &MockEdgeFunctionsRepo{}
	logger := utils.NewLogger("test")
	svc := NewEdgeFunctionsService(repo, logger)

	tenantID := uuid.New()
	functionID := uuid.New()

	repo.On("GetFunction", mock.Anything, functionID).Return(&models.EdgeFunction{
		ID:       functionID,
		TenantID: tenantID,
	}, nil)

	_, err := svc.CreateTrigger(context.Background(), tenantID, functionID, &models.CreateTriggerRequest{
		TriggerType: "invalid_type",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid trigger_type")
}

// --- GetTriggers Tests ---

func TestEdgeFunctionsService_GetTriggers(t *testing.T) {
	t.Parallel()
	repo := &MockEdgeFunctionsRepo{}
	logger := utils.NewLogger("test")
	svc := NewEdgeFunctionsService(repo, logger)

	tenantID := uuid.New()
	functionID := uuid.New()

	repo.On("GetFunction", mock.Anything, functionID).Return(&models.EdgeFunction{
		ID:       functionID,
		TenantID: tenantID,
	}, nil)
	repo.On("GetTriggersByFunction", mock.Anything, functionID).Return([]*models.EdgeFunctionTrigger{
		{ID: uuid.New(), FunctionID: functionID, TriggerType: models.TriggerPreSend},
		{ID: uuid.New(), FunctionID: functionID, TriggerType: models.TriggerPostReceive},
	}, nil)

	triggers, err := svc.GetTriggers(context.Background(), tenantID, functionID)
	require.NoError(t, err)
	assert.Len(t, triggers, 2)
}

// --- RunTest Tests ---

func TestEdgeFunctionsService_RunTest_ExpectedOutput(t *testing.T) {
	t.Parallel()
	repo := &MockEdgeFunctionsRepo{}
	logger := utils.NewLogger("test")
	svc := NewEdgeFunctionsService(repo, logger)

	tenantID := uuid.New()
	functionID := uuid.New()

	repo.On("GetFunction", mock.Anything, functionID).Return(&models.EdgeFunction{
		ID:              functionID,
		TenantID:        tenantID,
		Status:          models.FunctionStatusDraft,
		Runtime:         models.RuntimeJavaScript,
		Code:            `function handler(input) { return { doubled: input.val * 2 }; }`,
		EntryPoint:      "handler",
		TimeoutMs:       5000,
		EnvironmentVars: map[string]string{},
	}, nil)
	repo.On("CreateTest", mock.Anything, mock.AnythingOfType("*models.EdgeFunctionTest")).Return(nil)

	test, err := svc.RunTest(context.Background(), tenantID, functionID, &models.RunTestRequest{
		TestName:       "double-test",
		Input:          map[string]interface{}{"val": 5},
		ExpectedOutput: map[string]interface{}{"doubled": float64(10)},
	})
	require.NoError(t, err)
	assert.NotNil(t, test.Passed)
	assert.True(t, *test.Passed)
	assert.Equal(t, "double-test", test.TestName)
}

func TestEdgeFunctionsService_RunTest_ErrorInCode(t *testing.T) {
	t.Parallel()
	repo := &MockEdgeFunctionsRepo{}
	logger := utils.NewLogger("test")
	svc := NewEdgeFunctionsService(repo, logger)

	tenantID := uuid.New()
	functionID := uuid.New()

	repo.On("GetFunction", mock.Anything, functionID).Return(&models.EdgeFunction{
		ID:              functionID,
		TenantID:        tenantID,
		Status:          models.FunctionStatusDraft,
		Runtime:         models.RuntimeJavaScript,
		Code:            `function handler(input) { throw new Error("boom"); }`,
		EntryPoint:      "handler",
		TimeoutMs:       5000,
		EnvironmentVars: map[string]string{},
	}, nil)
	repo.On("CreateTest", mock.Anything, mock.AnythingOfType("*models.EdgeFunctionTest")).Return(nil)

	test, err := svc.RunTest(context.Background(), tenantID, functionID, &models.RunTestRequest{
		TestName: "error-test",
		Input:    map[string]interface{}{"key": "value"},
	})
	require.NoError(t, err)
	assert.NotNil(t, test.Passed)
	assert.False(t, *test.Passed)
	assert.NotEmpty(t, test.ErrorMessage)
}

// --- RollbackFunction Tests ---

func TestEdgeFunctionsService_RollbackFunction(t *testing.T) {
	t.Parallel()
	repo := &MockEdgeFunctionsRepo{}
	logger := utils.NewLogger("test")
	svc := NewEdgeFunctionsService(repo, logger)

	tenantID := uuid.New()
	functionID := uuid.New()

	repo.On("GetFunction", mock.Anything, functionID).Return(&models.EdgeFunction{
		ID:              functionID,
		TenantID:        tenantID,
		Code:            "current code",
		EntryPoint:      "handler",
		Status:          models.FunctionStatusActive,
		Runtime:         models.RuntimeJavaScript,
		EnvironmentVars: map[string]string{},
	}, nil)
	repo.On("GetVersion", mock.Anything, functionID, 1).Return(&models.EdgeFunctionVersion{
		FunctionID: functionID,
		Version:    1,
		Code:       "old code v1",
		EntryPoint: "handler",
	}, nil)
	repo.On("UpdateFunction", mock.Anything, mock.AnythingOfType("*models.EdgeFunction")).Return(nil)
	repo.On("CreateVersion", mock.Anything, mock.AnythingOfType("*models.EdgeFunctionVersion")).Return(nil).Maybe()
	repo.On("UpdateFunctionStatus", mock.Anything, functionID, models.FunctionStatusDraft).Return(nil).Maybe()

	fn, err := svc.RollbackFunction(context.Background(), tenantID, functionID, 1)
	require.NoError(t, err)
	assert.Equal(t, "old code v1", fn.Code)
	assert.Equal(t, models.FunctionStatusDraft, fn.Status)
}

// --- GetVersions Tests ---

func TestEdgeFunctionsService_GetVersions(t *testing.T) {
	t.Parallel()
	repo := &MockEdgeFunctionsRepo{}
	logger := utils.NewLogger("test")
	svc := NewEdgeFunctionsService(repo, logger)

	tenantID := uuid.New()
	functionID := uuid.New()

	repo.On("GetFunction", mock.Anything, functionID).Return(&models.EdgeFunction{
		ID:       functionID,
		TenantID: tenantID,
	}, nil)
	repo.On("GetVersions", mock.Anything, functionID).Return([]*models.EdgeFunctionVersion{
		{FunctionID: functionID, Version: 1},
		{FunctionID: functionID, Version: 2},
	}, nil)

	versions, err := svc.GetVersions(context.Background(), tenantID, functionID)
	require.NoError(t, err)
	assert.Len(t, versions, 2)
}

// --- GetDeployments Tests ---

func TestEdgeFunctionsService_GetDeployments(t *testing.T) {
	t.Parallel()
	repo := &MockEdgeFunctionsRepo{}
	logger := utils.NewLogger("test")
	svc := NewEdgeFunctionsService(repo, logger)

	tenantID := uuid.New()
	functionID := uuid.New()

	repo.On("GetFunction", mock.Anything, functionID).Return(&models.EdgeFunction{
		ID:       functionID,
		TenantID: tenantID,
	}, nil)
	repo.On("GetDeploymentsByFunction", mock.Anything, functionID).Return([]*models.EdgeFunctionDeployment{
		{ID: uuid.New(), FunctionID: functionID, Status: "active"},
		{ID: uuid.New(), FunctionID: functionID, Status: "pending"},
	}, nil)

	deployments, err := svc.GetDeployments(context.Background(), tenantID, functionID)
	require.NoError(t, err)
	assert.Len(t, deployments, 2)
}

// --- GetInvocations Tests ---

func TestEdgeFunctionsService_GetInvocations(t *testing.T) {
	t.Parallel()
	repo := &MockEdgeFunctionsRepo{}
	logger := utils.NewLogger("test")
	svc := NewEdgeFunctionsService(repo, logger)

	tenantID := uuid.New()
	functionID := uuid.New()

	repo.On("GetFunction", mock.Anything, functionID).Return(&models.EdgeFunction{
		ID:       functionID,
		TenantID: tenantID,
	}, nil)
	repo.On("GetInvocationsByFunction", mock.Anything, functionID, 10).Return([]*models.EdgeFunctionInvocation{
		{ID: uuid.New(), FunctionID: functionID, Status: "success"},
	}, nil)

	invocations, err := svc.GetInvocations(context.Background(), tenantID, functionID, 10)
	require.NoError(t, err)
	assert.Len(t, invocations, 1)
}

func TestEdgeFunctionsService_GetInvocations_DefaultLimit(t *testing.T) {
	t.Parallel()
	repo := &MockEdgeFunctionsRepo{}
	logger := utils.NewLogger("test")
	svc := NewEdgeFunctionsService(repo, logger)

	tenantID := uuid.New()
	functionID := uuid.New()

	repo.On("GetFunction", mock.Anything, functionID).Return(&models.EdgeFunction{
		ID:       functionID,
		TenantID: tenantID,
	}, nil)
	repo.On("GetInvocationsByFunction", mock.Anything, functionID, 50).Return([]*models.EdgeFunctionInvocation{}, nil)

	invocations, err := svc.GetInvocations(context.Background(), tenantID, functionID, 0)
	require.NoError(t, err)
	assert.Empty(t, invocations)
}

// --- GetDashboard Tests ---

func TestEdgeFunctionsService_GetDashboard(t *testing.T) {
	t.Parallel()
	repo := &MockEdgeFunctionsRepo{}
	logger := utils.NewLogger("test")
	svc := NewEdgeFunctionsService(repo, logger)

	tenantID := uuid.New()

	repo.On("CountFunctions", mock.Anything, tenantID).Return(5, nil)
	repo.On("CountActiveFunctions", mock.Anything, tenantID).Return(3, nil)
	repo.On("CountDeployments", mock.Anything, tenantID).Return(8, nil)
	repo.On("CountInvocations", mock.Anything, tenantID, mock.AnythingOfType("time.Time")).Return(int64(100), nil)
	repo.On("GetErrorRate", mock.Anything, tenantID, mock.AnythingOfType("time.Time")).Return(2.5, nil)
	repo.On("GetRecentInvocations", mock.Anything, tenantID, 10).Return([]*models.EdgeFunctionInvocation{
		{ID: uuid.New(), DurationMs: 20},
		{ID: uuid.New(), DurationMs: 40},
	}, nil)
	repo.On("GetFunctionsByTenant", mock.Anything, tenantID).Return([]*models.EdgeFunction{
		{Runtime: models.RuntimeJavaScript},
		{Runtime: models.RuntimeJavaScript},
		{Runtime: models.RuntimeTypeScript},
	}, nil)

	dashboard, err := svc.GetDashboard(context.Background(), tenantID)
	require.NoError(t, err)
	assert.Equal(t, 5, dashboard.TotalFunctions)
	assert.Equal(t, 3, dashboard.ActiveFunctions)
	assert.Equal(t, 8, dashboard.TotalDeployments)
	assert.Equal(t, int64(100), dashboard.TotalInvocations)
	assert.Equal(t, 2.5, dashboard.ErrorRate)
	assert.Equal(t, 30.0, dashboard.AvgDurationMs)
	assert.Equal(t, 2, dashboard.FunctionsByRuntime[models.RuntimeJavaScript])
	assert.Equal(t, 1, dashboard.FunctionsByRuntime[models.RuntimeTypeScript])
}

// --- GetLocations / GetActiveLocations Tests ---

func TestEdgeFunctionsService_GetLocations(t *testing.T) {
	t.Parallel()
	repo := &MockEdgeFunctionsRepo{}
	logger := utils.NewLogger("test")
	svc := NewEdgeFunctionsService(repo, logger)

	repo.On("GetAllLocations", mock.Anything).Return([]*models.EdgeLocation{
		{ID: uuid.New(), Name: "US East", Code: "us-east-1", Region: "us-east"},
		{ID: uuid.New(), Name: "EU West", Code: "eu-west-1", Region: "eu-west"},
	}, nil)

	locations, err := svc.GetLocations(context.Background())
	require.NoError(t, err)
	assert.Len(t, locations, 2)
	assert.Equal(t, "us-east-1", locations[0].Code)
}

func TestEdgeFunctionsService_GetActiveLocations(t *testing.T) {
	t.Parallel()
	repo := &MockEdgeFunctionsRepo{}
	logger := utils.NewLogger("test")
	svc := NewEdgeFunctionsService(repo, logger)

	repo.On("GetActiveLocations", mock.Anything).Return([]*models.EdgeLocation{
		{ID: uuid.New(), Name: "US East", Code: "us-east-1", Status: "active"},
	}, nil)

	locations, err := svc.GetActiveLocations(context.Background())
	require.NoError(t, err)
	assert.Len(t, locations, 1)
	assert.Equal(t, "active", locations[0].Status)
}
