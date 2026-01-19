package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"webhook-platform/pkg/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t *testing.T) *redis.Client {
	// Use Redis test database (DB 15)
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15,
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available for testing")
	}

	// Clear test database
	client.FlushDB(ctx)

	return client
}

func createTestTenant() *models.Tenant {
	return &models.Tenant{
		ID:                 uuid.New(),
		Name:               "Test Tenant",
		APIKeyHash:         "hash123",
		SubscriptionTier:   "basic",
		RateLimitPerMinute: 5, // Low limit for testing
		MonthlyQuota:       100000,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
}

func TestRateLimiter_CheckRateLimit_Success(t *testing.T) {
	redisClient := setupTestRedis(t)
	defer redisClient.Close()

	rateLimiter := NewRateLimiter(redisClient)
	tenant := createTestTenant()
	ctx := context.Background()

	// First request should be allowed
	info, err := rateLimiter.CheckRateLimit(ctx, tenant)
	require.NoError(t, err)
	assert.True(t, info.Allowed)
	assert.Equal(t, 4, info.Remaining) // 5 - 1 = 4
	assert.Equal(t, int64(0), info.RetryAfter)
}

func TestRateLimiter_CheckRateLimit_ExceedsLimit(t *testing.T) {
	redisClient := setupTestRedis(t)
	defer redisClient.Close()

	rateLimiter := NewRateLimiter(redisClient)
	tenant := createTestTenant()
	ctx := context.Background()

	// Make requests up to the limit
	for i := 0; i < tenant.RateLimitPerMinute; i++ {
		info, err := rateLimiter.CheckRateLimit(ctx, tenant)
		require.NoError(t, err)
		assert.True(t, info.Allowed)
	}

	// Next request should be denied
	info, err := rateLimiter.CheckRateLimit(ctx, tenant)
	require.NoError(t, err)
	assert.False(t, info.Allowed)
	assert.Equal(t, 0, info.Remaining)
	assert.Greater(t, info.RetryAfter, int64(0))
}

func TestRateLimiter_GetRateLimitStatus(t *testing.T) {
	redisClient := setupTestRedis(t)
	defer redisClient.Close()

	rateLimiter := NewRateLimiter(redisClient)
	tenant := createTestTenant()
	ctx := context.Background()

	// Initial status should show full limit available
	info, err := rateLimiter.GetRateLimitStatus(ctx, tenant)
	require.NoError(t, err)
	assert.True(t, info.Allowed)
	assert.Equal(t, tenant.RateLimitPerMinute, info.Remaining)

	// Make a request
	_, err = rateLimiter.CheckRateLimit(ctx, tenant)
	require.NoError(t, err)

	// Status should reflect the consumed request
	info, err = rateLimiter.GetRateLimitStatus(ctx, tenant)
	require.NoError(t, err)
	assert.True(t, info.Allowed)
	assert.Equal(t, tenant.RateLimitPerMinute-1, info.Remaining)
}

func TestRateLimiter_ResetRateLimit(t *testing.T) {
	redisClient := setupTestRedis(t)
	defer redisClient.Close()

	rateLimiter := NewRateLimiter(redisClient)
	tenant := createTestTenant()
	ctx := context.Background()

	// Consume all requests
	for i := 0; i < tenant.RateLimitPerMinute+1; i++ {
		rateLimiter.CheckRateLimit(ctx, tenant)
	}

	// Verify limit is exceeded
	info, err := rateLimiter.GetRateLimitStatus(ctx, tenant)
	require.NoError(t, err)
	assert.False(t, info.Allowed)

	// Reset rate limit
	err = rateLimiter.ResetRateLimit(ctx, tenant.ID.String())
	require.NoError(t, err)

	// Verify limit is reset
	info, err = rateLimiter.GetRateLimitStatus(ctx, tenant)
	require.NoError(t, err)
	assert.True(t, info.Allowed)
	assert.Equal(t, tenant.RateLimitPerMinute, info.Remaining)
}

func TestRateLimiter_Middleware_Success(t *testing.T) {
	redisClient := setupTestRedis(t)
	defer redisClient.Close()

	rateLimiter := NewRateLimiter(redisClient)
	router := setupTestRouter()

	// Setup middleware
	router.Use(func(c *gin.Context) {
		// Mock tenant in context
		tenant := createTestTenant()
		c.Set(TenantKey, tenant)
		c.Next()
	})
	router.Use(rateLimiter.RateLimit())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "5", w.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "4", w.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))
}

func TestRateLimiter_Middleware_ExceedsLimit(t *testing.T) {
	redisClient := setupTestRedis(t)
	defer redisClient.Close()

	rateLimiter := NewRateLimiter(redisClient)
	router := setupTestRouter()
	tenant := createTestTenant()

	// Setup middleware
	router.Use(func(c *gin.Context) {
		c.Set(TenantKey, tenant)
		c.Next()
	})
	router.Use(rateLimiter.RateLimit())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Consume all requests
	for i := 0; i < tenant.RateLimitPerMinute; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// Next request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Contains(t, w.Body.String(), "RATE_LIMIT_EXCEEDED")
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
}

func TestRateLimiter_Middleware_MissingTenant(t *testing.T) {
	redisClient := setupTestRedis(t)
	defer redisClient.Close()

	rateLimiter := NewRateLimiter(redisClient)
	router := setupTestRouter()

	// Setup middleware without setting tenant context
	router.Use(rateLimiter.RateLimit())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "MISSING_TENANT_CONTEXT")
}

func TestRateLimiter_DifferentTenants(t *testing.T) {
	redisClient := setupTestRedis(t)
	defer redisClient.Close()

	rateLimiter := NewRateLimiter(redisClient)
	ctx := context.Background()

	tenant1 := createTestTenant()
	tenant2 := createTestTenant()

	// Consume all requests for tenant1
	for i := 0; i < tenant1.RateLimitPerMinute+1; i++ {
		rateLimiter.CheckRateLimit(ctx, tenant1)
	}

	// Verify tenant1 is rate limited
	info1, err := rateLimiter.GetRateLimitStatus(ctx, tenant1)
	require.NoError(t, err)
	assert.False(t, info1.Allowed)

	// Verify tenant2 is not affected
	info2, err := rateLimiter.GetRateLimitStatus(ctx, tenant2)
	require.NoError(t, err)
	assert.True(t, info2.Allowed)
	assert.Equal(t, tenant2.RateLimitPerMinute, info2.Remaining)
}

func TestRateLimiter_WindowReset(t *testing.T) {
	redisClient := setupTestRedis(t)
	defer redisClient.Close()

	rateLimiter := NewRateLimiter(redisClient)
	tenant := createTestTenant()
	ctx := context.Background()

	// Make one request
	info, err := rateLimiter.CheckRateLimit(ctx, tenant)
	require.NoError(t, err)
	assert.True(t, info.Allowed)
	assert.Equal(t, tenant.RateLimitPerMinute-1, info.Remaining)

	// Wait for the key to expire (simulate window reset)
	// In a real test, you might want to use a shorter window or mock time
	key := "rate_limit:" + tenant.ID.String()
	redisClient.Del(ctx, key) // Simulate window reset

	// Next request should have full limit available
	info, err = rateLimiter.CheckRateLimit(ctx, tenant)
	require.NoError(t, err)
	assert.True(t, info.Allowed)
	assert.Equal(t, tenant.RateLimitPerMinute-1, info.Remaining)
}