package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/josedab/waas/internal/api/services"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/repository"
	"github.com/josedab/waas/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockEdgeFunctionsRepo implements repository.EdgeFunctionsRepository for testing
type mockEdgeFunctionsRepo struct {
	mock.Mock
}

// --- Functions ---

func (m *mockEdgeFunctionsRepo) CreateFunction(ctx context.Context, fn *models.EdgeFunction) error {
	args := m.Called(ctx, fn)
	fn.ID = uuid.New()
	fn.CreatedAt = time.Now()
	fn.UpdatedAt = time.Now()
	return args.Error(0)
}

func (m *mockEdgeFunctionsRepo) GetFunction(ctx context.Context, id uuid.UUID) (*models.EdgeFunction, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EdgeFunction), args.Error(1)
}

func (m *mockEdgeFunctionsRepo) GetFunctionByName(ctx context.Context, tenantID uuid.UUID, name string) (*models.EdgeFunction, error) {
	args := m.Called(ctx, tenantID, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EdgeFunction), args.Error(1)
}

func (m *mockEdgeFunctionsRepo) GetFunctionsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.EdgeFunction, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.EdgeFunction), args.Error(1)
}

func (m *mockEdgeFunctionsRepo) GetActiveFunctions(ctx context.Context, tenantID uuid.UUID) ([]*models.EdgeFunction, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.EdgeFunction), args.Error(1)
}

func (m *mockEdgeFunctionsRepo) UpdateFunction(ctx context.Context, fn *models.EdgeFunction) error {
	args := m.Called(ctx, fn)
	return args.Error(0)
}

func (m *mockEdgeFunctionsRepo) UpdateFunctionStatus(ctx context.Context, id uuid.UUID, status string) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *mockEdgeFunctionsRepo) DeleteFunction(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// --- Versions ---

func (m *mockEdgeFunctionsRepo) CreateVersion(ctx context.Context, version *models.EdgeFunctionVersion) error {
	args := m.Called(ctx, version)
	return args.Error(0)
}

func (m *mockEdgeFunctionsRepo) GetVersions(ctx context.Context, functionID uuid.UUID) ([]*models.EdgeFunctionVersion, error) {
	args := m.Called(ctx, functionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.EdgeFunctionVersion), args.Error(1)
}

func (m *mockEdgeFunctionsRepo) GetVersion(ctx context.Context, functionID uuid.UUID, version int) (*models.EdgeFunctionVersion, error) {
	args := m.Called(ctx, functionID, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EdgeFunctionVersion), args.Error(1)
}

// --- Locations ---

func (m *mockEdgeFunctionsRepo) GetAllLocations(ctx context.Context) ([]*models.EdgeLocation, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.EdgeLocation), args.Error(1)
}

func (m *mockEdgeFunctionsRepo) GetLocation(ctx context.Context, id uuid.UUID) (*models.EdgeLocation, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EdgeLocation), args.Error(1)
}

func (m *mockEdgeFunctionsRepo) GetLocationByCode(ctx context.Context, code string) (*models.EdgeLocation, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EdgeLocation), args.Error(1)
}

func (m *mockEdgeFunctionsRepo) GetActiveLocations(ctx context.Context) ([]*models.EdgeLocation, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.EdgeLocation), args.Error(1)
}

// --- Deployments ---

func (m *mockEdgeFunctionsRepo) CreateDeployment(ctx context.Context, deployment *models.EdgeFunctionDeployment) error {
	args := m.Called(ctx, deployment)
	return args.Error(0)
}

func (m *mockEdgeFunctionsRepo) GetDeployment(ctx context.Context, id uuid.UUID) (*models.EdgeFunctionDeployment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EdgeFunctionDeployment), args.Error(1)
}

func (m *mockEdgeFunctionsRepo) GetDeploymentsByFunction(ctx context.Context, functionID uuid.UUID) ([]*models.EdgeFunctionDeployment, error) {
	args := m.Called(ctx, functionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.EdgeFunctionDeployment), args.Error(1)
}

func (m *mockEdgeFunctionsRepo) GetActiveDeployment(ctx context.Context, functionID, locationID uuid.UUID) (*models.EdgeFunctionDeployment, error) {
	args := m.Called(ctx, functionID, locationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EdgeFunctionDeployment), args.Error(1)
}

func (m *mockEdgeFunctionsRepo) UpdateDeploymentStatus(ctx context.Context, id uuid.UUID, status string, deploymentURL string) error {
	args := m.Called(ctx, id, status, deploymentURL)
	return args.Error(0)
}

func (m *mockEdgeFunctionsRepo) UpdateDeploymentHealth(ctx context.Context, id uuid.UUID, healthStatus string) error {
	args := m.Called(ctx, id, healthStatus)
	return args.Error(0)
}

func (m *mockEdgeFunctionsRepo) SetDeploymentError(ctx context.Context, id uuid.UUID, errorMsg string) error {
	args := m.Called(ctx, id, errorMsg)
	return args.Error(0)
}

// --- Triggers ---

func (m *mockEdgeFunctionsRepo) CreateTrigger(ctx context.Context, trigger *models.EdgeFunctionTrigger) error {
	args := m.Called(ctx, trigger)
	return args.Error(0)
}

func (m *mockEdgeFunctionsRepo) GetTrigger(ctx context.Context, id uuid.UUID) (*models.EdgeFunctionTrigger, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EdgeFunctionTrigger), args.Error(1)
}

func (m *mockEdgeFunctionsRepo) GetTriggersByFunction(ctx context.Context, functionID uuid.UUID) ([]*models.EdgeFunctionTrigger, error) {
	args := m.Called(ctx, functionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.EdgeFunctionTrigger), args.Error(1)
}

func (m *mockEdgeFunctionsRepo) GetMatchingTriggers(ctx context.Context, tenantID uuid.UUID, triggerType, eventType string, endpointID uuid.UUID) ([]*models.EdgeFunctionTrigger, error) {
	args := m.Called(ctx, tenantID, triggerType, eventType, endpointID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.EdgeFunctionTrigger), args.Error(1)
}

func (m *mockEdgeFunctionsRepo) UpdateTrigger(ctx context.Context, trigger *models.EdgeFunctionTrigger) error {
	args := m.Called(ctx, trigger)
	return args.Error(0)
}

func (m *mockEdgeFunctionsRepo) DeleteTrigger(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// --- Invocations ---

func (m *mockEdgeFunctionsRepo) CreateInvocation(ctx context.Context, invocation *models.EdgeFunctionInvocation) error {
	args := m.Called(ctx, invocation)
	return args.Error(0)
}

func (m *mockEdgeFunctionsRepo) GetInvocation(ctx context.Context, id uuid.UUID) (*models.EdgeFunctionInvocation, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EdgeFunctionInvocation), args.Error(1)
}

func (m *mockEdgeFunctionsRepo) GetInvocationsByFunction(ctx context.Context, functionID uuid.UUID, limit int) ([]*models.EdgeFunctionInvocation, error) {
	args := m.Called(ctx, functionID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.EdgeFunctionInvocation), args.Error(1)
}

func (m *mockEdgeFunctionsRepo) GetRecentInvocations(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.EdgeFunctionInvocation, error) {
	args := m.Called(ctx, tenantID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.EdgeFunctionInvocation), args.Error(1)
}

func (m *mockEdgeFunctionsRepo) CompleteInvocation(ctx context.Context, id uuid.UUID, status string, durationMs, memoryUsed int, errorMsg string) error {
	args := m.Called(ctx, id, status, durationMs, memoryUsed, errorMsg)
	return args.Error(0)
}

// --- Metrics ---

func (m *mockEdgeFunctionsRepo) CreateOrUpdateMetrics(ctx context.Context, metrics *models.EdgeFunctionMetrics) error {
	args := m.Called(ctx, metrics)
	return args.Error(0)
}

func (m *mockEdgeFunctionsRepo) GetMetrics(ctx context.Context, functionID uuid.UUID, since time.Time) ([]*models.EdgeFunctionMetrics, error) {
	args := m.Called(ctx, functionID, since)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.EdgeFunctionMetrics), args.Error(1)
}

// --- Secrets ---

func (m *mockEdgeFunctionsRepo) CreateSecret(ctx context.Context, secret *models.EdgeFunctionSecret) error {
	args := m.Called(ctx, secret)
	return args.Error(0)
}

func (m *mockEdgeFunctionsRepo) GetSecrets(ctx context.Context, functionID uuid.UUID) ([]*models.EdgeFunctionSecret, error) {
	args := m.Called(ctx, functionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.EdgeFunctionSecret), args.Error(1)
}

func (m *mockEdgeFunctionsRepo) DeleteSecret(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// --- Tests ---

func (m *mockEdgeFunctionsRepo) CreateTest(ctx context.Context, test *models.EdgeFunctionTest) error {
	args := m.Called(ctx, test)
	return args.Error(0)
}

func (m *mockEdgeFunctionsRepo) GetTests(ctx context.Context, functionID uuid.UUID) ([]*models.EdgeFunctionTest, error) {
	args := m.Called(ctx, functionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.EdgeFunctionTest), args.Error(1)
}

// --- Dashboard ---

func (m *mockEdgeFunctionsRepo) CountFunctions(ctx context.Context, tenantID uuid.UUID) (int, error) {
	args := m.Called(ctx, tenantID)
	return args.Int(0), args.Error(1)
}

func (m *mockEdgeFunctionsRepo) CountActiveFunctions(ctx context.Context, tenantID uuid.UUID) (int, error) {
	args := m.Called(ctx, tenantID)
	return args.Int(0), args.Error(1)
}

func (m *mockEdgeFunctionsRepo) CountDeployments(ctx context.Context, tenantID uuid.UUID) (int, error) {
	args := m.Called(ctx, tenantID)
	return args.Int(0), args.Error(1)
}

func (m *mockEdgeFunctionsRepo) CountInvocations(ctx context.Context, tenantID uuid.UUID, since time.Time) (int64, error) {
	args := m.Called(ctx, tenantID, since)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockEdgeFunctionsRepo) GetErrorRate(ctx context.Context, tenantID uuid.UUID, since time.Time) (float64, error) {
	args := m.Called(ctx, tenantID, since)
	return args.Get(0).(float64), args.Error(1)
}

// Compile-time check
var _ repository.EdgeFunctionsRepository = (*mockEdgeFunctionsRepo)(nil)

// setupEdgeFunctionsTestRouter creates a test router with mock repo
func setupEdgeFunctionsTestRouter(repo *mockEdgeFunctionsRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	logger := utils.NewLogger("test")
	svc := services.NewEdgeFunctionsService(repo, logger)
	handler := NewEdgeFunctionsHandler(svc, logger)

	router := gin.New()
	api := router.Group("/api/v1")
	api.Use(func(c *gin.Context) {
		if tid := c.GetHeader("X-Tenant-ID"); tid != "" {
			parsed, err := uuid.Parse(tid)
			if err == nil {
				c.Set("tenant_id", parsed)
			}
		}
		c.Next()
	})
	handler.RegisterRoutes(api)
	return router
}

// --- GetLocations tests ---

func TestGetLocations_Success(t *testing.T) {
	repo := new(mockEdgeFunctionsRepo)
	locations := []*models.EdgeLocation{
		{ID: uuid.New(), Name: "US East", Code: "us-east-1", Region: "us-east", Status: "active"},
		{ID: uuid.New(), Name: "EU West", Code: "eu-west-1", Region: "eu-west", Status: "active"},
	}
	repo.On("GetActiveLocations", mock.Anything).Return(locations, nil)

	router := setupEdgeFunctionsTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/edge-functions/locations", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "us-east-1")
	assert.Contains(t, w.Body.String(), "eu-west-1")
	repo.AssertExpectations(t)
}

// --- GetDashboard tests ---

func TestEdgeFunctionsGetDashboard_Success(t *testing.T) {
	repo := new(mockEdgeFunctionsRepo)
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	repo.On("CountFunctions", mock.Anything, tenantID).Return(5, nil)
	repo.On("CountActiveFunctions", mock.Anything, tenantID).Return(3, nil)
	repo.On("CountDeployments", mock.Anything, tenantID).Return(8, nil)
	repo.On("CountInvocations", mock.Anything, tenantID, mock.Anything).Return(int64(100), nil)
	repo.On("GetErrorRate", mock.Anything, tenantID, mock.Anything).Return(float64(2.5), nil)
	repo.On("GetRecentInvocations", mock.Anything, tenantID, 10).Return([]*models.EdgeFunctionInvocation{}, nil)
	repo.On("GetFunctionsByTenant", mock.Anything, tenantID).Return([]*models.EdgeFunction{}, nil)

	router := setupEdgeFunctionsTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/edge-functions/dashboard", nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"total_functions":5`)
	assert.Contains(t, w.Body.String(), `"active_functions":3`)
	repo.AssertExpectations(t)
}

func TestEdgeFunctionsGetDashboard_MissingTenantID(t *testing.T) {
	repo := new(mockEdgeFunctionsRepo)
	router := setupEdgeFunctionsTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/edge-functions/dashboard", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- GetFunctions tests ---

func TestGetFunctions_Success(t *testing.T) {
	repo := new(mockEdgeFunctionsRepo)
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	functions := []*models.EdgeFunction{
		{ID: uuid.New(), TenantID: tenantID, Name: "fn-1", Runtime: "javascript", Status: "active"},
		{ID: uuid.New(), TenantID: tenantID, Name: "fn-2", Runtime: "python", Status: "draft"},
	}
	repo.On("GetFunctionsByTenant", mock.Anything, tenantID).Return(functions, nil)

	router := setupEdgeFunctionsTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/edge-functions", nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "fn-1")
	assert.Contains(t, w.Body.String(), "fn-2")
	repo.AssertExpectations(t)
}

func TestGetFunctions_MissingTenantID(t *testing.T) {
	repo := new(mockEdgeFunctionsRepo)
	router := setupEdgeFunctionsTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/edge-functions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- CreateFunction tests ---

func TestCreateFunction_Success(t *testing.T) {
	repo := new(mockEdgeFunctionsRepo)
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	repo.On("CreateFunction", mock.Anything, mock.AnythingOfType("*models.EdgeFunction")).Return(nil)
	repo.On("CreateVersion", mock.Anything, mock.AnythingOfType("*models.EdgeFunctionVersion")).Return(nil)

	router := setupEdgeFunctionsTestRouter(repo)

	body := `{"name":"my-function","code":"function handler(event) { return {ok: true}; }"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/edge-functions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID.String())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "my-function")
	repo.AssertExpectations(t)
}

func TestCreateFunction_MissingTenantID(t *testing.T) {
	repo := new(mockEdgeFunctionsRepo)
	router := setupEdgeFunctionsTestRouter(repo)

	body := `{"name":"my-function","code":"function handler(event) { return {ok: true}; }"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/edge-functions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateFunction_InvalidJSON(t *testing.T) {
	repo := new(mockEdgeFunctionsRepo)
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	router := setupEdgeFunctionsTestRouter(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/edge-functions", bytes.NewBufferString(`{invalid`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID.String())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
