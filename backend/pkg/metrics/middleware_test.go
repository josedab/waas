package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestPrometheusMiddleware(t *testing.T) {
	// Setup test router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(PrometheusMiddleware())
	
	router.GET("/test", func(c *gin.Context) {
		c.Set("tenant_id", "test-tenant-123")
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Make test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify metrics were recorded by checking the metric family
	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	assert.NoError(t, err)
	
	// Find our metrics
	var foundRequestsTotal, foundRequestDuration bool
	for _, mf := range metricFamilies {
		if mf.GetName() == "webhook_http_requests_total" {
			foundRequestsTotal = true
			assert.True(t, len(mf.GetMetric()) > 0)
		}
		if mf.GetName() == "webhook_http_request_duration_seconds" {
			foundRequestDuration = true
			assert.True(t, len(mf.GetMetric()) > 0)
		}
	}
	
	assert.True(t, foundRequestsTotal, "Should find webhook_http_requests_total metric")
	assert.True(t, foundRequestDuration, "Should find webhook_http_request_duration_seconds metric")
}

func TestRecordWebhookDelivery(t *testing.T) {
	// Record delivery
	tenantID := "tenant-123"
	endpointID := "endpoint-456"
	status := "success"
	duration := 250 * time.Millisecond

	RecordWebhookDelivery(tenantID, endpointID, status, duration)

	// Verify metrics were recorded by checking the metric families
	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	assert.NoError(t, err)
	
	var foundDeliveriesTotal, foundDeliveryDuration bool
	for _, mf := range metricFamilies {
		if mf.GetName() == "webhook_deliveries_total" {
			foundDeliveriesTotal = true
		}
		if mf.GetName() == "webhook_delivery_duration_seconds" {
			foundDeliveryDuration = true
		}
	}
	
	assert.True(t, foundDeliveriesTotal, "Should find webhook_deliveries_total metric")
	assert.True(t, foundDeliveryDuration, "Should find webhook_delivery_duration_seconds metric")
}

func TestRecordWebhookRetry(t *testing.T) {
	// Record retry
	tenantID := "tenant-123"
	endpointID := "endpoint-456"
	attemptNumber := 2

	RecordWebhookRetry(tenantID, endpointID, attemptNumber)

	// Verify metrics were recorded
	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	assert.NoError(t, err)
	
	var foundRetryAttempts bool
	for _, mf := range metricFamilies {
		if mf.GetName() == "webhook_retry_attempts_total" {
			foundRetryAttempts = true
		}
	}
	
	assert.True(t, foundRetryAttempts, "Should find webhook_retry_attempts_total metric")
}

func TestUpdateQueueSize(t *testing.T) {
	// Update queue size
	queueName := "delivery-queue"
	size := 42.0

	UpdateQueueSize(queueName, size)

	// Verify metrics were recorded
	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	assert.NoError(t, err)
	
	var foundQueueSize bool
	for _, mf := range metricFamilies {
		if mf.GetName() == "webhook_queue_size" {
			foundQueueSize = true
		}
	}
	
	assert.True(t, foundQueueSize, "Should find webhook_queue_size metric")
}

func TestUpdateActiveConnections(t *testing.T) {
	// Update active connections
	count := 15.0

	UpdateActiveConnections(count)

	// Verify metrics were recorded
	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	assert.NoError(t, err)
	
	var foundActiveConnections bool
	for _, mf := range metricFamilies {
		if mf.GetName() == "webhook_active_connections" {
			foundActiveConnections = true
		}
	}
	
	assert.True(t, foundActiveConnections, "Should find webhook_active_connections metric")
}

func TestUpdateDatabaseConnections(t *testing.T) {
	// Update database connections
	active := 5
	idle := 10
	total := 15

	UpdateDatabaseConnections(active, idle, total)

	// Verify metrics were recorded
	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	assert.NoError(t, err)
	
	var foundDatabaseConnections bool
	for _, mf := range metricFamilies {
		if mf.GetName() == "webhook_http_database_connections" {
			foundDatabaseConnections = true
		}
	}
	
	assert.True(t, foundDatabaseConnections, "Should find webhook_http_database_connections metric")
}

func TestGetTenantIDFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		expectedResult string
	}{
		{
			name: "valid tenant ID",
			setupContext: func(c *gin.Context) {
				c.Set("tenant_id", "test-tenant-123")
			},
			expectedResult: "test-tenant-123",
		},
		{
			name: "missing tenant ID",
			setupContext: func(c *gin.Context) {
				// Don't set tenant_id
			},
			expectedResult: "unknown",
		},
		{
			name: "invalid tenant ID type",
			setupContext: func(c *gin.Context) {
				c.Set("tenant_id", 123) // Wrong type
			},
			expectedResult: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			tt.setupContext(c)
			
			result := getTenantIDFromContext(c)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}