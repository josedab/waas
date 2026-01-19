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
	"github.com/stretchr/testify/mock"
)



func TestQuotaMiddleware_EnforceQuota_WithinLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := &MockQuotaRepository{}
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	middleware := NewQuotaMiddleware(mockRepo, redisClient)

	tenant := &models.Tenant{
		ID:           uuid.New(),
		Name:         "Test Tenant",
		MonthlyQuota: 1000,
		SubscriptionTier: "starter",
	}

	usage := &models.QuotaUsage{
		ID:           uuid.New(),
		TenantID:     tenant.ID,
		RequestCount: 500,
		SuccessCount: 450,
		FailureCount: 50,
		OverageCount: 0,
	}

	mockRepo.On("GetOrCreateQuotaUsage", mock.Anything, tenant.ID, mock.Anything).Return(usage, nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(TenantKey, tenant)
		c.Next()
	})
	router.Use(middleware.EnforceQuota())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "1000", w.Header().Get("X-Quota-Limit"))
	assert.Equal(t, "500", w.Header().Get("X-Quota-Remaining"))
	assert.Equal(t, "500", w.Header().Get("X-Quota-Used"))

	mockRepo.AssertExpectations(t)
}

func TestQuotaMiddleware_EnforceQuota_ExceedsLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := &MockQuotaRepository{}
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	middleware := NewQuotaMiddleware(mockRepo, redisClient)

	tenant := &models.Tenant{
		ID:           uuid.New(),
		Name:         "Test Tenant",
		MonthlyQuota: 1000,
		SubscriptionTier: "starter",
	}

	usage := &models.QuotaUsage{
		ID:           uuid.New(),
		TenantID:     tenant.ID,
		RequestCount: 1200,
		SuccessCount: 1100,
		FailureCount: 100,
		OverageCount: 1500, // Exceeds burst allowance (1000)
	}

	mockRepo.On("GetOrCreateQuotaUsage", mock.Anything, tenant.ID, mock.Anything).Return(usage, nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(TenantKey, tenant)
		c.Next()
	})
	router.Use(middleware.EnforceQuota())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusPaymentRequired, w.Code)
	assert.Contains(t, w.Body.String(), "QUOTA_EXCEEDED")

	mockRepo.AssertExpectations(t)
}

func TestQuotaMiddleware_EnforceQuota_BurstAllowance(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := &MockQuotaRepository{}
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	middleware := NewQuotaMiddleware(mockRepo, redisClient)

	tenant := &models.Tenant{
		ID:           uuid.New(),
		Name:         "Test Tenant",
		MonthlyQuota: 1000,
		SubscriptionTier: "starter", // Has 1000 burst allowance
	}

	usage := &models.QuotaUsage{
		ID:           uuid.New(),
		TenantID:     tenant.ID,
		RequestCount: 1050,
		SuccessCount: 1000,
		FailureCount: 50,
		OverageCount: 50, // Within burst allowance
	}

	mockRepo.On("GetOrCreateQuotaUsage", mock.Anything, tenant.ID, mock.Anything).Return(usage, nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(TenantKey, tenant)
		c.Next()
	})
	router.Use(middleware.EnforceQuota())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "true", w.Header().Get("X-Quota-Overage"))
	assert.Equal(t, "950", w.Header().Get("X-Quota-Overage-Remaining"))

	mockRepo.AssertExpectations(t)
}

func TestQuotaMiddleware_TrackUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := &MockQuotaRepository{}
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	middleware := NewQuotaMiddleware(mockRepo, redisClient)

	tenant := &models.Tenant{
		ID:           uuid.New(),
		Name:         "Test Tenant",
		MonthlyQuota: 1000,
	}

	mockRepo.On("IncrementUsage", mock.Anything, tenant.ID, true).Return(nil)
	mockRepo.On("GetQuotaUsageByTenant", mock.Anything, tenant.ID, mock.Anything).Return(&models.QuotaUsage{
		RequestCount: 500,
	}, nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(TenantKey, tenant)
		c.Next()
	})
	router.Use(middleware.TrackUsage())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Give some time for async processing
	time.Sleep(100 * time.Millisecond)

	mockRepo.AssertExpectations(t)
}

func TestQuotaMiddleware_CheckQuota(t *testing.T) {
	mockRepo := &MockQuotaRepository{}
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	middleware := NewQuotaMiddleware(mockRepo, redisClient)

	tenant := &models.Tenant{
		ID:           uuid.New(),
		MonthlyQuota: 1000,
	}

	usage := &models.QuotaUsage{
		RequestCount: 750,
		SuccessCount: 700,
		FailureCount: 50,
		OverageCount: 0,
	}

	mockRepo.On("GetOrCreateQuotaUsage", mock.Anything, tenant.ID, mock.Anything).Return(usage, nil)

	quotaInfo, err := middleware.CheckQuota(context.Background(), tenant)

	assert.NoError(t, err)
	assert.NotNil(t, quotaInfo)
	assert.True(t, quotaInfo.Allowed)
	assert.Equal(t, 750, quotaInfo.CurrentUsage)
	assert.Equal(t, 1000, quotaInfo.MonthlyQuota)
	assert.Equal(t, 250, quotaInfo.Remaining)
	assert.Equal(t, 75.0, quotaInfo.UsagePercent)
	assert.False(t, quotaInfo.IsOverage)

	mockRepo.AssertExpectations(t)
}