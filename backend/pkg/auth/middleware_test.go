package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"webhook-platform/pkg/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)



func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestAuthMiddleware_RequireAuth_Success(t *testing.T) {
	// Setup
	mockRepo := new(MockTenantRepository)
	middleware := NewAuthMiddleware(mockRepo)
	router := setupTestRouter()

	// Create test tenant
	tenant := &models.Tenant{
		ID:                 uuid.New(),
		Name:               "Test Tenant",
		APIKeyHash:         "hash123",
		SubscriptionTier:   "basic",
		RateLimitPerMinute: 1000,
		MonthlyQuota:       100000,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// Mock repository response
	mockRepo.On("FindByAPIKey", mock.Anything, "wh_valid_key").Return(tenant, nil)

	// Setup route with middleware
	router.Use(middleware.RequireAuth())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer wh_valid_key")
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestAuthMiddleware_RequireAuth_MissingHeader(t *testing.T) {
	// Setup
	mockRepo := new(MockTenantRepository)
	middleware := NewAuthMiddleware(mockRepo)
	router := setupTestRouter()

	// Setup route with middleware
	router.Use(middleware.RequireAuth())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Create request without Authorization header
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "MISSING_AUTH_HEADER")
}

func TestAuthMiddleware_RequireAuth_InvalidFormat(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{
			name:   "missing Bearer prefix",
			header: "wh_valid_key",
		},
		{
			name:   "wrong prefix",
			header: "Basic wh_valid_key",
		},
		{
			name:   "empty Bearer",
			header: "Bearer ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := new(MockTenantRepository)
			middleware := NewAuthMiddleware(mockRepo)
			router := setupTestRouter()

			// Setup route with middleware
			router.Use(middleware.RequireAuth())
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			// Create request with invalid header
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", tt.header)
			w := httptest.NewRecorder()

			// Execute request
			router.ServeHTTP(w, req)

			// Assertions
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

func TestAuthMiddleware_RequireAuth_InvalidAPIKey(t *testing.T) {
	// Setup
	mockRepo := new(MockTenantRepository)
	middleware := NewAuthMiddleware(mockRepo)
	router := setupTestRouter()

	// Mock repository to return error (tenant not found)
	mockRepo.On("FindByAPIKey", mock.Anything, "wh_invalid_key").Return(nil, assert.AnError)

	// Setup route with middleware
	router.Use(middleware.RequireAuth())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Create request with invalid API key
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer wh_invalid_key")
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "INVALID_API_KEY")
	mockRepo.AssertExpectations(t)
}

func TestAuthMiddleware_RequireAuth_InvalidAPIKeyFormat(t *testing.T) {
	// Setup
	mockRepo := new(MockTenantRepository)
	middleware := NewAuthMiddleware(mockRepo)
	router := setupTestRouter()

	// Setup route with middleware
	router.Use(middleware.RequireAuth())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Create request with invalid API key format
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid_format")
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "INVALID_API_KEY_FORMAT")
}

func TestGetTenantFromContext(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	// Test case 1: No tenant in context
	tenant, exists := GetTenantFromContext(c)
	assert.False(t, exists)
	assert.Nil(t, tenant)

	// Test case 2: Valid tenant in context
	testTenant := &models.Tenant{
		ID:   uuid.New(),
		Name: "Test Tenant",
	}
	c.Set(TenantKey, testTenant)

	tenant, exists = GetTenantFromContext(c)
	assert.True(t, exists)
	assert.Equal(t, testTenant, tenant)

	// Test case 3: Invalid type in context
	c.Set(TenantKey, "invalid_type")
	tenant, exists = GetTenantFromContext(c)
	assert.False(t, exists)
	assert.Nil(t, tenant)
}

func TestGetTenantIDFromContext(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	// Test case 1: No tenant ID in context
	tenantID, exists := GetTenantIDFromContext(c)
	assert.False(t, exists)
	assert.Equal(t, uuid.Nil, tenantID)

	// Test case 2: Valid tenant ID in context
	testTenantID := uuid.New()
	c.Set(TenantIDKey, testTenantID.String())

	tenantID, exists = GetTenantIDFromContext(c)
	assert.True(t, exists)
	assert.Equal(t, testTenantID, tenantID)

	// Test case 3: Invalid UUID format in context
	c.Set(TenantIDKey, "invalid-uuid")
	tenantID, exists = GetTenantIDFromContext(c)
	assert.False(t, exists)
	assert.Equal(t, uuid.Nil, tenantID)

	// Test case 4: Invalid type in context
	c.Set(TenantIDKey, 123)
	tenantID, exists = GetTenantIDFromContext(c)
	assert.False(t, exists)
	assert.Equal(t, uuid.Nil, tenantID)
}