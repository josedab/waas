package monitoring

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"webhook-platform/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)



func TestHealthChecker_HealthCheckHandler(t *testing.T) {
	t.Parallel()
	logger := utils.NewLogger("test")
	hc := NewHealthChecker(nil, nil, logger, "test")

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/health", hc.HealthCheckHandler())

	// Make request
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusServiceUnavailable, w.Code) // Should be unhealthy without real DB

	// Parse response and check health status
	var response HealthCheckResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "test", response.Version)
	assert.NotEmpty(t, response.Uptime)
	assert.Contains(t, response.Components, "system")
	assert.Contains(t, response.Components, "database")
}

func TestHealthChecker_ReadinessHandler(t *testing.T) {
	t.Parallel()
	logger := utils.NewLogger("test")
	hc := NewHealthChecker(nil, nil, logger, "test")

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/ready", hc.ReadinessHandler())

	// Make request
	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions - without real DB, readiness should be not ready
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestHealthChecker_LivenessHandler(t *testing.T) {
	t.Parallel()
	logger := utils.NewLogger("test")
	hc := NewHealthChecker(nil, nil, logger, "test")

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/live", hc.LivenessHandler())

	// Make request
	req := httptest.NewRequest("GET", "/live", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "alive", response["status"])
	assert.NotEmpty(t, response["uptime"])
}

func TestHealthChecker_checkDatabase(t *testing.T) {
	t.Parallel()
	logger := utils.NewLogger("test")
	hc := NewHealthChecker(nil, nil, logger, "test")

	ctx := context.Background()
	health := hc.checkDatabase(ctx)

	// Without real DB, it should be unhealthy
	assert.Equal(t, HealthStatusUnhealthy, health.Status)
	assert.NotEmpty(t, health.ResponseTime)
	assert.False(t, health.LastChecked.IsZero())
}

func TestHealthChecker_determineOverallStatus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		components map[string]ComponentHealth
		expected   HealthStatus
	}{
		{
			name: "all healthy",
			components: map[string]ComponentHealth{
				"db":    {Status: HealthStatusHealthy},
				"redis": {Status: HealthStatusHealthy},
			},
			expected: HealthStatusHealthy,
		},
		{
			name: "one degraded",
			components: map[string]ComponentHealth{
				"db":    {Status: HealthStatusHealthy},
				"redis": {Status: HealthStatusDegraded},
			},
			expected: HealthStatusDegraded,
		},
		{
			name: "one unhealthy",
			components: map[string]ComponentHealth{
				"db":    {Status: HealthStatusUnhealthy},
				"redis": {Status: HealthStatusHealthy},
			},
			expected: HealthStatusUnhealthy,
		},
		{
			name: "mixed statuses",
			components: map[string]ComponentHealth{
				"db":     {Status: HealthStatusUnhealthy},
				"redis":  {Status: HealthStatusDegraded},
				"system": {Status: HealthStatusHealthy},
			},
			expected: HealthStatusUnhealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hc := &HealthChecker{}
			result := hc.determineOverallStatus(tt.components)
			assert.Equal(t, tt.expected, result)
		})
	}
}