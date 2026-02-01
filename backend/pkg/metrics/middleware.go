package metrics

import (
	"strconv"
	"time"
	"github.com/josedab/waas/pkg/monitoring"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP request metrics
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status_code", "tenant_id"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "webhook_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint", "tenant_id"},
	)

	// Webhook delivery metrics
	webhookDeliveriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_deliveries_total",
			Help: "Total number of webhook deliveries",
		},
		[]string{"status", "tenant_id", "endpoint_id"},
	)

	webhookDeliveryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "webhook_delivery_duration_seconds",
			Help:    "Webhook delivery duration in seconds",
			Buckets: []float64{0.1, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0, 60.0},
		},
		[]string{"tenant_id", "endpoint_id"},
	)

	webhookRetryAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_retry_attempts_total",
			Help: "Total number of webhook retry attempts",
		},
		[]string{"tenant_id", "endpoint_id", "attempt_number"},
	)

	// Queue metrics
	queueSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "webhook_queue_size",
			Help: "Current size of webhook delivery queue",
		},
		[]string{"queue_name"},
	)

	queueProcessingTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "webhook_queue_processing_time_seconds",
			Help:    "Time spent processing queue messages",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"queue_name"},
	)

	// System metrics
	activeConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "webhook_active_connections",
			Help: "Number of active WebSocket connections",
		},
	)

	databaseConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "webhook_http_database_connections",
			Help: "Number of database connections for HTTP middleware",
		},
		[]string{"state"}, // active, idle, total
	)
)

// PrometheusMiddleware creates a Gin middleware for collecting HTTP metrics
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		// Process request
		c.Next()
		
		// Extract tenant ID from context (set by auth middleware)
		tenantID := getTenantIDFromContext(c)
		
		// Record metrics
		duration := time.Since(start).Seconds()
		statusCode := strconv.Itoa(c.Writer.Status())
		
		httpRequestsTotal.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
			statusCode,
			tenantID,
		).Inc()
		
		httpRequestDuration.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
			tenantID,
		).Observe(duration)
	}
}

// EnhancedMetricsMiddleware creates an enhanced middleware that integrates with the monitoring system
func EnhancedMetricsMiddleware(metricsRecorder *monitoring.MetricsRecorder, alertManager *monitoring.AlertManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		// Get request size
		requestSize := c.Request.ContentLength
		if requestSize < 0 {
			requestSize = 0
		}
		
		// Process request
		c.Next()
		
		// Calculate metrics
		duration := time.Since(start)
		statusCode := strconv.Itoa(c.Writer.Status())
		tenantID := getTenantIDFromContext(c)
		
		// Record basic HTTP metrics
		httpRequestsTotal.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
			statusCode,
			tenantID,
		).Inc()
		
		httpRequestDuration.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
			tenantID,
		).Observe(duration.Seconds())
		
		// Record enhanced metrics if recorder is available
		if metricsRecorder != nil {
			metricsRecorder.RecordAPIRequestSize(c.Request.Method, c.FullPath(), requestSize)
			metricsRecorder.RecordAPIResponseSize(c.Request.Method, c.FullPath(), statusCode, int64(c.Writer.Size()))
		}
		
		// Evaluate metrics for alerting if alert manager is available
		if alertManager != nil {
			labels := map[string]string{
				"method":    c.Request.Method,
				"endpoint":  c.FullPath(),
				"tenant_id": tenantID,
			}
			
			// Check for slow requests
			if duration.Seconds() > 5.0 {
				alertManager.EvaluateMetric("http_request_duration", duration.Seconds(), labels)
			}
			
			// Check for error rates
			if c.Writer.Status() >= 500 {
				alertManager.EvaluateMetric("http_error_rate", 1.0, labels)
			}
		}
	}
}

// RecordWebhookDelivery records metrics for webhook delivery attempts
func RecordWebhookDelivery(tenantID, endpointID, status string, duration time.Duration) {
	webhookDeliveriesTotal.WithLabelValues(status, tenantID, endpointID).Inc()
	webhookDeliveryDuration.WithLabelValues(tenantID, endpointID).Observe(duration.Seconds())
}

// RecordWebhookRetry records metrics for webhook retry attempts
func RecordWebhookRetry(tenantID, endpointID string, attemptNumber int) {
	webhookRetryAttempts.WithLabelValues(
		tenantID,
		endpointID,
		strconv.Itoa(attemptNumber),
	).Inc()
}

// UpdateQueueSize updates the queue size metric
func UpdateQueueSize(queueName string, size float64) {
	queueSize.WithLabelValues(queueName).Set(size)
}

// RecordQueueProcessingTime records the time spent processing a queue message
func RecordQueueProcessingTime(queueName string, duration time.Duration) {
	queueProcessingTime.WithLabelValues(queueName).Observe(duration.Seconds())
}

// UpdateActiveConnections updates the active WebSocket connections metric
func UpdateActiveConnections(count float64) {
	activeConnections.Set(count)
}

// UpdateDatabaseConnections updates database connection metrics
func UpdateDatabaseConnections(active, idle, total int) {
	databaseConnections.WithLabelValues("active").Set(float64(active))
	databaseConnections.WithLabelValues("idle").Set(float64(idle))
	databaseConnections.WithLabelValues("total").Set(float64(total))
}

// getTenantIDFromContext extracts tenant ID from Gin context
func getTenantIDFromContext(c *gin.Context) string {
	if tenantID, exists := c.Get("tenant_id"); exists {
		if id, ok := tenantID.(string); ok {
			return id
		}
	}
	return "unknown"
}