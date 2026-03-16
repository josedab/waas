package monitoring

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/josedab/waas/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// HealthStatus represents the overall health status
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// ComponentHealth represents the health of a single component
type ComponentHealth struct {
	Status       HealthStatus `json:"status"`
	Message      string       `json:"message,omitempty"`
	LastChecked  time.Time    `json:"last_checked"`
	ResponseTime string       `json:"response_time,omitempty"`
}

// HealthCheckResponse represents the complete health check response
type HealthCheckResponse struct {
	Status     HealthStatus               `json:"status"`
	Timestamp  time.Time                  `json:"timestamp"`
	Version    string                     `json:"version"`
	Components map[string]ComponentHealth `json:"components"`
	Uptime     string                     `json:"uptime"`
}

// HealthChecker provides health check functionality
type HealthChecker struct {
	db          *sql.DB
	redisClient *redis.Client
	logger      *utils.Logger
	startTime   time.Time
	version     string
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(db *sql.DB, redisClient *redis.Client, logger *utils.Logger, version string) *HealthChecker {
	return &HealthChecker{
		db:          db,
		redisClient: redisClient,
		logger:      logger,
		startTime:   time.Now(),
		version:     version,
	}
}

// HealthCheckHandler returns a Gin handler for health checks
func (hc *HealthChecker) HealthCheckHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		response := hc.performHealthCheck(ctx)

		// Set appropriate HTTP status based on overall health
		var httpStatus int
		switch response.Status {
		case HealthStatusHealthy:
			httpStatus = http.StatusOK
		case HealthStatusDegraded:
			httpStatus = http.StatusOK // Still return 200 for degraded
		case HealthStatusUnhealthy:
			httpStatus = http.StatusServiceUnavailable
		default:
			httpStatus = http.StatusInternalServerError
		}

		c.JSON(httpStatus, response)
	}
}

// ReadinessHandler returns a Gin handler for readiness checks
func (hc *HealthChecker) ReadinessHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		dbHealth := hc.checkDatabase(ctx)
		redisHealth := hc.checkRedis(ctx)

		// Both DB and Redis must be healthy for the service to be ready.
		if dbHealth.Status == HealthStatusHealthy && redisHealth.Status != HealthStatusUnhealthy {
			c.JSON(http.StatusOK, gin.H{
				"status":    "ready",
				"timestamp": time.Now(),
			})
		} else {
			reasons := []string{}
			if dbHealth.Status != HealthStatusHealthy {
				reasons = append(reasons, "database: "+dbHealth.Message)
			}
			if redisHealth.Status == HealthStatusUnhealthy {
				reasons = append(reasons, "redis: "+redisHealth.Message)
			}
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":    "not ready",
				"timestamp": time.Now(),
				"reasons":   reasons,
			})
		}
	}
}

// LivenessHandler returns a Gin handler for liveness checks
func (hc *HealthChecker) LivenessHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Simple liveness check - if we can respond, we're alive
		c.JSON(http.StatusOK, gin.H{
			"status":    "alive",
			"timestamp": time.Now(),
			"uptime":    time.Since(hc.startTime).String(),
		})
	}
}

// performHealthCheck performs a comprehensive health check
func (hc *HealthChecker) performHealthCheck(ctx context.Context) HealthCheckResponse {
	components := make(map[string]ComponentHealth)

	// Check database
	components["database"] = hc.checkDatabase(ctx)

	// Check Redis
	components["redis"] = hc.checkRedis(ctx)

	// Check system resources
	components["system"] = hc.checkSystem(ctx)

	// Determine overall status
	overallStatus := hc.determineOverallStatus(components)

	return HealthCheckResponse{
		Status:     overallStatus,
		Timestamp:  time.Now(),
		Version:    hc.version,
		Components: components,
		Uptime:     time.Since(hc.startTime).String(),
	}
}

// checkDatabase checks database connectivity and performance
func (hc *HealthChecker) checkDatabase(ctx context.Context) ComponentHealth {
	start := time.Now()

	if hc.db == nil {
		return ComponentHealth{
			Status:       HealthStatusUnhealthy,
			Message:      "Database connection not initialized",
			LastChecked:  time.Now(),
			ResponseTime: time.Since(start).String(),
		}
	}

	// Test database connectivity with a simple query
	err := hc.db.PingContext(ctx)
	responseTime := time.Since(start)

	if err != nil {
		hc.logger.Error("Database health check failed", map[string]interface{}{
			"error":         err.Error(),
			"response_time": responseTime.String(),
		})
		return ComponentHealth{
			Status:       HealthStatusUnhealthy,
			Message:      "Database connection failed: " + err.Error(),
			LastChecked:  time.Now(),
			ResponseTime: responseTime.String(),
		}
	}

	// Check if response time is acceptable (< 1 second is healthy, < 5 seconds is degraded)
	status := HealthStatusHealthy
	message := "Database is healthy"

	if responseTime > 5*time.Second {
		status = HealthStatusUnhealthy
		message = "Database response time too slow"
	} else if responseTime > 1*time.Second {
		status = HealthStatusDegraded
		message = "Database response time is slow"
	}

	return ComponentHealth{
		Status:       status,
		Message:      message,
		LastChecked:  time.Now(),
		ResponseTime: responseTime.String(),
	}
}

// checkRedis checks Redis connectivity and performance
func (hc *HealthChecker) checkRedis(ctx context.Context) ComponentHealth {
	start := time.Now()

	if hc.redisClient == nil {
		return ComponentHealth{
			Status:      HealthStatusDegraded, // Redis is not critical for basic functionality
			Message:     "Redis client not initialized",
			LastChecked: time.Now(),
		}
	}

	// Test Redis connectivity with a ping
	err := hc.redisClient.Ping(ctx).Err()
	responseTime := time.Since(start)

	if err != nil {
		hc.logger.Warn("Redis health check failed", map[string]interface{}{
			"error":         err.Error(),
			"response_time": responseTime.String(),
		})
		return ComponentHealth{
			Status:       HealthStatusDegraded, // Redis failure is degraded, not unhealthy
			Message:      "Redis connection failed: " + err.Error(),
			LastChecked:  time.Now(),
			ResponseTime: responseTime.String(),
		}
	}

	// Check if response time is acceptable
	status := HealthStatusHealthy
	message := "Redis is healthy"

	if responseTime > 2*time.Second {
		status = HealthStatusDegraded
		message = "Redis response time is slow"
	}

	return ComponentHealth{
		Status:       status,
		Message:      message,
		LastChecked:  time.Now(),
		ResponseTime: responseTime.String(),
	}
}

// checkSystem checks system-level health indicators
func (hc *HealthChecker) checkSystem(ctx context.Context) ComponentHealth {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Flag degraded if the process is using more than 1 GB of heap.
	const heapThreshold = 1 << 30
	status := HealthStatusHealthy
	message := fmt.Sprintf("goroutines=%d heap_mb=%d", runtime.NumGoroutine(), m.HeapAlloc/(1<<20))

	if m.HeapAlloc > heapThreshold {
		status = HealthStatusDegraded
		message = "high memory usage: " + message
	}

	return ComponentHealth{
		Status:      status,
		Message:     message,
		LastChecked: time.Now(),
	}
}

// determineOverallStatus determines the overall health status based on component statuses
func (hc *HealthChecker) determineOverallStatus(components map[string]ComponentHealth) HealthStatus {
	hasUnhealthy := false
	hasDegraded := false

	for _, component := range components {
		switch component.Status {
		case HealthStatusUnhealthy:
			hasUnhealthy = true
		case HealthStatusDegraded:
			hasDegraded = true
		}
	}

	if hasUnhealthy {
		return HealthStatusUnhealthy
	}
	if hasDegraded {
		return HealthStatusDegraded
	}
	return HealthStatusHealthy
}

// GetHealthStatus returns the current health status (for programmatic access)
func (hc *HealthChecker) GetHealthStatus(ctx context.Context) HealthCheckResponse {
	return hc.performHealthCheck(ctx)
}
