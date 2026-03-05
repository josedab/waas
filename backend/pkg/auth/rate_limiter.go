package auth

import (
	"context"
	"errors"
	"fmt"
	"github.com/josedab/waas/pkg/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimiter enforces per-minute request rate limits per tenant using
// Redis-backed sliding window counters.
type RateLimiter struct {
	redisClient *redis.Client
}

// RateLimitInfo describes the current rate-limit state for a tenant,
// including remaining requests and reset time.
type RateLimitInfo struct {
	Allowed    bool
	Remaining  int
	ResetTime  time.Time
	RetryAfter int64
}

// NewRateLimiter creates a RateLimiter backed by the given Redis client.
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
						"limit":       tenant.RateLimitPerMinute,
						"remaining":   rateLimitInfo.Remaining,
						"reset_time":  rateLimitInfo.ResetTime.Unix(),
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

// CheckRateLimit checks if the tenant has exceeded their rate limit.
//
// Algorithm: fixed-window rate limiting using Redis INCR + EXPIREAT.
// Each 1-minute window is aligned to the clock (time.Truncate(Minute)),
// and the Redis key auto-expires at the end of the window.
//
// Fixed-window was chosen over sliding-window for simplicity and lower Redis
// overhead (single key per tenant vs. sorted set). The trade-off is that a
// burst of requests at a window boundary can briefly allow up to 2× the
// configured limit, but this is acceptable for webhook delivery use-cases
// where exact fairness is less critical than throughput.
//
// Clock skew note: because windows are derived from the application server's
// local clock, servers with significant clock drift may produce inconsistent
// windows. In practice, NTP-synced hosts keep drift well under 1 second.
func (rl *RateLimiter) CheckRateLimit(ctx context.Context, tenant *models.Tenant) (*RateLimitInfo, error) {
	key := fmt.Sprintf("rate_limit:%s", tenant.ID.String())
	window := time.Minute
	limit := tenant.RateLimitPerMinute

	// Align to the start of the current minute to form the fixed window.
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
		Allowed:    allowed,
		Remaining:  remaining,
		ResetTime:  windowEnd,
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
		if errors.Is(err, redis.Nil) {
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
		Allowed:    allowed,
		Remaining:  remaining,
		ResetTime:  windowEnd,
		RetryAfter: retryAfter,
	}, nil
}

// ResetRateLimit resets the rate limit counter for a tenant (admin function)
func (rl *RateLimiter) ResetRateLimit(ctx context.Context, tenantID string) error {
	key := fmt.Sprintf("rate_limit:%s", tenantID)
	return rl.redisClient.Del(ctx, key).Err()
}

// IPRateLimit returns middleware that enforces a per-IP rate limit using Redis.
// This is suitable for unauthenticated endpoints like tenant creation.
func (rl *RateLimiter) IPRateLimit(maxRequests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		key := fmt.Sprintf("rate_limit:ip:%s", ip)

		now := time.Now()
		windowEnd := now.Truncate(window).Add(window)

		pipe := rl.redisClient.Pipeline()
		incrCmd := pipe.Incr(c.Request.Context(), key)
		pipe.ExpireAt(c.Request.Context(), key, windowEnd)
		_, err := pipe.Exec(c.Request.Context())
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

		currentCount := int(incrCmd.Val())
		remaining := maxInt(0, maxRequests-currentCount)

		c.Header("X-RateLimit-Limit", strconv.Itoa(maxRequests))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(windowEnd.Unix(), 10))

		if currentCount > maxRequests {
			retryAfter := int64(windowEnd.Sub(now).Seconds())
			c.Header("Retry-After", strconv.FormatInt(retryAfter, 10))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "Too many requests. Please try again later.",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
