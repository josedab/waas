package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"webhook-platform/pkg/auth"
	"webhook-platform/pkg/models"
	"webhook-platform/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockTenantRepository for testing
type MockTenantRepository struct {
	mock.Mock
}

func (m *MockTenantRepository) Create(ctx context.Context, tenant *models.Tenant) error {
	args := m.Called(ctx, tenant)
	return args.Error(0)
}

func (m *MockTenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Tenant), args.Error(1)
}

func (m *MockTenantRepository) GetByAPIKeyHash(ctx context.Context, apiKeyHash string) (*models.Tenant, error) {
	args := m.Called(ctx, apiKeyHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Tenant), args.Error(1)
}

func (m *MockTenantRepository) FindByAPIKey(ctx context.Context, apiKey string) (*models.Tenant, error) {
	args := m.Called(ctx, apiKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Tenant), args.Error(1)
}

func (m *MockTenantRepository) Update(ctx context.Context, tenant *models.Tenant) error {
	args := m.Called(ctx, tenant)
	return args.Error(0)
}

func (m *MockTenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTenantRepository) List(ctx context.Context, limit, offset int) ([]*models.Tenant, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Tenant), args.Error(1)
}

func setupTestHandler() (*TenantHandler, *MockTenantRepository) {
	mockRepo := new(MockTenantRepository)
	logger := utils.NewLogger("test")
	handler := NewTenantHandler(mockRepo, logger)
	return handler, mockRepo
}

func setupTestGin() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestTenantHandler_CreateTenant_Success(t *testing.T) {
	handler, mockRepo := setupTestHandler()
	router := setupTestGin()

	// Setup route
	router.POST("/tenants", handler.CreateTenant)

	// Mock repository
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Tenant")).Return(nil)

	// Create request
	reqBody := CreateTenantRequest{
		Name:             "Test Tenant",
		SubscriptionTier: "basic",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/tenants", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusCreated, w.Code)

	var response CreateTenantResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response.Tenant)
	assert.Equal(t, "Test Tenant", response.Tenant.Name)
	assert.Equal(t, "basic", response.Tenant.SubscriptionTier)
	assert.Equal(t, 1000, response.Tenant.RateLimitPerMinute) // Default for basic
	assert.Equal(t, 100000, response.Tenant.MonthlyQuota)     // Default for basic
	assert.NotEmpty(t, response.APIKey)
	assert.True(t, auth.IsValidAPIKeyFormat(response.APIKey))

	mockRepo.AssertExpectations(t)
}

func TestTenantHandler_CreateTenant_InvalidRequest(t *testing.T) {
	handler, _ := setupTestHandler()
	router := setupTestGin()

	// Setup route
	router.POST("/tenants", handler.CreateTenant)

	tests := []struct {
		name     string
		reqBody  interface{}
		expected string
	}{
		{
			name:     "missing name",
			reqBody:  CreateTenantRequest{SubscriptionTier: "basic"},
			expected: "INVALID_REQUEST",
		},
		{
			name:     "invalid subscription tier",
			reqBody:  CreateTenantRequest{Name: "Test", SubscriptionTier: "invalid"},
			expected: "INVALID_REQUEST",
		},
		{
			name:     "empty name",
			reqBody:  CreateTenantRequest{Name: "", SubscriptionTier: "basic"},
			expected: "INVALID_REQUEST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest(http.MethodPost, "/tenants", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), tt.expected)
		})
	}
}

func TestTenantHandler_CreateTenant_RepositoryError(t *testing.T) {
	handler, mockRepo := setupTestHandler()
	router := setupTestGin()

	// Setup route
	router.POST("/tenants", handler.CreateTenant)

	// Mock repository to return error
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Tenant")).Return(assert.AnError)

	// Create request
	reqBody := CreateTenantRequest{
		Name:             "Test Tenant",
		SubscriptionTier: "basic",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/tenants", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "TENANT_CREATION_FAILED")
	mockRepo.AssertExpectations(t)
}

func TestTenantHandler_GetTenant_Success(t *testing.T) {
	handler, _ := setupTestHandler()
	router := setupTestGin()

	// Create test tenant
	tenant := &models.Tenant{
		ID:                 uuid.New(),
		Name:               "Test Tenant",
		SubscriptionTier:   "basic",
		RateLimitPerMinute: 1000,
		MonthlyQuota:       100000,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// Setup middleware to inject tenant context
	router.Use(func(c *gin.Context) {
		c.Set(auth.TenantKey, tenant)
		c.Next()
	})
	router.GET("/tenant", handler.GetTenant)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/tenant", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), tenant.Name)
	assert.Contains(t, w.Body.String(), tenant.ID.String())
}

func TestTenantHandler_GetTenant_MissingContext(t *testing.T) {
	handler, _ := setupTestHandler()
	router := setupTestGin()

	// Setup route without tenant context
	router.GET("/tenant", handler.GetTenant)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/tenant", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "MISSING_TENANT_CONTEXT")
}

func TestTenantHandler_UpdateTenant_Success(t *testing.T) {
	handler, mockRepo := setupTestHandler()
	router := setupTestGin()

	// Create test tenant
	tenant := &models.Tenant{
		ID:                 uuid.New(),
		Name:               "Test Tenant",
		SubscriptionTier:   "basic",
		RateLimitPerMinute: 1000,
		MonthlyQuota:       100000,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// Mock repository
	mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Tenant")).Return(nil)

	// Setup middleware to inject tenant context
	router.Use(func(c *gin.Context) {
		c.Set(auth.TenantKey, tenant)
		c.Next()
	})
	router.PUT("/tenant", handler.UpdateTenant)

	// Create request
	reqBody := UpdateTenantRequest{
		Name: "Updated Tenant",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/tenant", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Updated Tenant")
	mockRepo.AssertExpectations(t)
}

func TestTenantHandler_RegenerateAPIKey_Success(t *testing.T) {
	handler, mockRepo := setupTestHandler()
	router := setupTestGin()

	// Create test tenant
	tenant := &models.Tenant{
		ID:                 uuid.New(),
		Name:               "Test Tenant",
		APIKeyHash:         "old_hash",
		SubscriptionTier:   "basic",
		RateLimitPerMinute: 1000,
		MonthlyQuota:       100000,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// Mock repository
	mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Tenant")).Return(nil)

	// Setup middleware to inject tenant context
	router.Use(func(c *gin.Context) {
		c.Set(auth.TenantKey, tenant)
		c.Next()
	})
	router.POST("/tenant/regenerate-key", handler.RegenerateAPIKey)

	// Create request
	req := httptest.NewRequest(http.MethodPost, "/tenant/regenerate-key", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	apiKey, exists := response["api_key"].(string)
	assert.True(t, exists)
	assert.True(t, auth.IsValidAPIKeyFormat(apiKey))

	mockRepo.AssertExpectations(t)
}

func TestTenantHandler_ListTenants_Success(t *testing.T) {
	handler, mockRepo := setupTestHandler()
	router := setupTestGin()

	// Create test tenants
	tenants := []*models.Tenant{
		{
			ID:               uuid.New(),
			Name:             "Tenant 1",
			SubscriptionTier: "basic",
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
		{
			ID:               uuid.New(),
			Name:             "Tenant 2",
			SubscriptionTier: "premium",
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
	}

	// Mock repository
	mockRepo.On("List", mock.Anything, 50, 0).Return(tenants, nil)

	// Setup route
	router.GET("/admin/tenants", handler.ListTenants)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/admin/tenants", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	tenantsResponse, exists := response["tenants"].([]interface{})
	assert.True(t, exists)
	assert.Len(t, tenantsResponse, 2)

	mockRepo.AssertExpectations(t)
}

func TestGetDefaultRateLimit(t *testing.T) {
	tests := []struct {
		tier     string
		expected int
	}{
		{"free", 100},
		{"basic", 1000},
		{"premium", 5000},
		{"enterprise", 10000},
		{"unknown", 100},
	}

	for _, tt := range tests {
		t.Run(tt.tier, func(t *testing.T) {
			result := getDefaultRateLimit(tt.tier)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetDefaultMonthlyQuota(t *testing.T) {
	tests := []struct {
		tier     string
		expected int
	}{
		{"free", 10000},
		{"basic", 100000},
		{"premium", 1000000},
		{"enterprise", 10000000},
		{"unknown", 10000},
	}

	for _, tt := range tests {
		t.Run(tt.tier, func(t *testing.T) {
			result := getDefaultMonthlyQuota(tt.tier)
			assert.Equal(t, tt.expected, result)
		})
	}
}