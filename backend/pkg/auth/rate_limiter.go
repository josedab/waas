package auth

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"
	"webhook-platform/pkg/models"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	redisClient *redis.Client
}

type RateLimitInfo struct {
	Allowed   bool
	Remaining int
	ResetTime time.Time
	RetryAfter int64
}

func NewRateLimiter(redisClient *redis.Client) *RateLimiter {
	return &RateLimiter{
		redisClient: redisClient,
	}
}

// RateLimit middleware enforces rate limits based on tenant configuration
func (rl *RateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenant, exists := GetTenantFromContext(c)
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":    "MISSING_TENANT_CONTEXT",
					"message": "Tenant context not found",
				},
			})
			c.Abort()
			return
		}

		rateLimitInfo, err := rl.CheckRateLimit(c.Request.Context(), tenant)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":    "RATE_LIMIT_ERROR",
					"message": "Failed to check rate limit",
				},
			})
			c.Abort()
			return
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(tenant.RateLimitPerMinute))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(rateLimitInfo.Remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(rateLimitInfo.ResetTime.Unix(), 10))

		if !rateLimitInfo.Allowed {
			c.Header("Retry-After", strconv.FormatInt(rateLimitInfo.RetryAfter, 10))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "Rate limit exceeded. Please try again later.",
					"details": gin.H{
						"limit":      tenant.RateLimitPerMinute,
						"remaining":  rateLimitInfo.Remaining,
						"reset_time": rateLimitInfo.ResetTime.Unix(),
						"retry_after": rateLimitInfo.RetryAfter,
					},
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// CheckRateLimit checks if the tenant has exceeded their rate limit
func (rl *RateLimiter) CheckRateLimit(ctx context.Context, tenant *models.Tenant) (*RateLimitInfo, error) {
	key := fmt.Sprintf("rate_limit:%s", tenant.ID.String())
	window := time.Minute
	limit := tenant.RateLimitPerMinute

	now := time.Now()
	windowStart := now.Truncate(window)
	windowEnd := windowStart.Add(window)

	// Use Redis pipeline for atomic operations
	pipe := rl.redisClient.Pipeline()
	
	// Increment counter for current window
	incrCmd := pipe.Incr(ctx, key)
	
	// Set expiration if this is the first request in the window
	pipe.ExpireAt(ctx, key, windowEnd)
	
	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute rate limit pipeline: %w", err)
	}

	currentCount := int(incrCmd.Val())
	remaining := maxInt(0, limit-currentCount)
	allowed := currentCount <= limit
	
	retryAfter := int64(0)
	if !allowed {
		retryAfter = int64(windowEnd.Sub(now).Seconds())
	}

	return &RateLimitInfo{
		Allowed:   allowed,
		Remaining: remaining,
		ResetTime: windowEnd,
		RetryAfter: retryAfter,
	}, nil
}

// GetRateLimitStatus returns current rate limit status for a tenant
func (rl *RateLimiter) GetRateLimitStatus(ctx context.Context, tenant *models.Tenant) (*RateLimitInfo, error) {
	key := fmt.Sprintf("rate_limit:%s", tenant.ID.String())
	window := time.Minute
	limit := tenant.RateLimitPerMinute

	now := time.Now()
	windowStart := now.Truncate(window)
	windowEnd := windowStart.Add(window)

	// Get current count without incrementing
	currentCount, err := rl.redisClient.Get(ctx, key).Int()
	if err != nil {
		if err == redis.Nil {
			currentCount = 0
		} else {
			return nil, fmt.Errorf("failed to get rate limit count: %w", err)
		}
	}

	remaining := maxInt(0, limit-currentCount)
	allowed := currentCount < limit

	retryAfter := int64(0)
	if !allowed {
		retryAfter = int64(windowEnd.Sub(now).Seconds())
	}

	return &RateLimitInfo{
		Allowed:   allowed,
		Remaining: remaining,
		ResetTime: windowEnd,
		RetryAfter: retryAfter,
	}, nil
}

// ResetRateLimit resets the rate limit counter for a tenant (admin function)
func (rl *RateLimiter) ResetRateLimit(ctx context.Context, tenantID string) error {
	key := fmt.Sprintf("rate_limit:%s", tenantID)
	return rl.redisClient.Del(ctx, key).Err()
}

