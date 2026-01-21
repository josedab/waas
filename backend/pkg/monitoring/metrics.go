package monitoring

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Custom metrics for webhook platform monitoring
var (
	// Service health metrics
	serviceHealthStatus = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "webhook_service_health_status",
			Help: "Health status of service components (1=healthy, 0.5=degraded, 0=unhealthy)",
		},
		[]string{"service", "component"},
	)

	serviceUptime = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "webhook_service_uptime_seconds",
			Help: "Service uptime in seconds",
		},
		[]string{"service"},
	)

	// Enhanced webhook delivery metrics
	webhookDeliveryLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "webhook_delivery_latency_seconds",
			Help:    "Webhook delivery latency distribution",
			Buckets: []float64{0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0, 60.0, 120.0},
		},
		[]string{"tenant_id", "endpoint_id", "status"},
	)

	webhookDeliveryErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_delivery_errors_total",
			Help: "Total number of webhook delivery errors by type",
		},
		[]string{"tenant_id", "endpoint_id", "error_type", "http_status"},
	)

	webhookEndpointHealth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "webhook_endpoint_health_score",
			Help: "Health score of webhook endpoints (0-1, based on success rate)",
		},
		[]string{"tenant_id", "endpoint_id"},
	)

	// Queue performance metrics
	queueDepth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "webhook_queue_depth",
			Help: "Current depth of webhook delivery queues",
		},
		[]string{"queue_name", "priority"},
	)

	queueThroughput = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_queue_messages_processed_total",
			Help: "Total number of queue messages processed",
		},
		[]string{"queue_name", "status"},
	)

	queueProcessingLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "webhook_queue_processing_latency_seconds",
			Help:    "Time spent processing queue messages",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"queue_name"},
	)

	// Database performance metrics
	databaseQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "webhook_database_query_duration_seconds",
			Help:    "Database query execution time",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
		},
		[]string{"operation", "table"},
	)

	databaseConnectionPool = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "webhook_database_connections",
			Help: "Database connection pool statistics",
		},
		[]string{"state"}, // active, idle, total, max
	)

	// API performance metrics
	apiRequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "webhook_api_request_size_bytes",
			Help:    "Size of API requests in bytes",
			Buckets: []float64{100, 1000, 10000, 100000, 1000000, 10000000},
		},
		[]string{"method", "endpoint"},
	)

	apiResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "webhook_api_response_size_bytes",
			Help:    "Size of API responses in bytes",
			Buckets: []float64{100, 1000, 10000, 100000, 1000000},
		},
		[]string{"method", "endpoint", "status_code"},
	)

	// Rate limiting metrics
	rateLimitHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_rate_limit_hits_total",
			Help: "Total number of rate limit hits",
		},
		[]string{"tenant_id", "limit_type"},
	)

	rateLimitRemaining = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "webhook_rate_limit_remaining",
			Help: "Remaining rate limit quota",
		},
		[]string{"tenant_id", "limit_type"},
	)

	// Security metrics
	authenticationFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_authentication_failures_total",
			Help: "Total number of authentication failures",
		},
		[]string{"reason", "source_ip"},
	)

	suspiciousActivity = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_suspicious_activity_total",
			Help: "Total number of suspicious activities detected",
		},
		[]string{"type", "tenant_id"},
	)
)

// MetricsRecorder provides methods to record various metrics
type MetricsRecorder struct {
	startTime time.Time
}

// NewMetricsRecorder creates a new metrics recorder
func NewMetricsRecorder() *MetricsRecorder {
	return &MetricsRecorder{
		startTime: time.Now(),
	}
}

// RecordServiceHealth records the health status of a service component
func (mr *MetricsRecorder) RecordServiceHealth(service, component string, status HealthStatus) {
	var value float64
	switch status {
	case HealthStatusHealthy:
		value = 1.0
	case HealthStatusDegraded:
		value = 0.5
	case HealthStatusUnhealthy:
		value = 0.0
	}
	serviceHealthStatus.WithLabelValues(service, component).Set(value)
}

// RecordServiceUptime records the uptime of a service
func (mr *MetricsRecorder) RecordServiceUptime(service string) {
	uptime := time.Since(mr.startTime).Seconds()
	serviceUptime.WithLabelValues(service).Set(uptime)
}

// RecordWebhookDeliveryLatency records webhook delivery latency
func (mr *MetricsRecorder) RecordWebhookDeliveryLatency(tenantID, endpointID, status string, duration time.Duration) {
	webhookDeliveryLatency.WithLabelValues(tenantID, endpointID, status).Observe(duration.Seconds())
}

// RecordWebhookDeliveryError records webhook delivery errors
func (mr *MetricsRecorder) RecordWebhookDeliveryError(tenantID, endpointID, errorType, httpStatus string) {
	webhookDeliveryErrors.WithLabelValues(tenantID, endpointID, errorType, httpStatus).Inc()
}

// RecordWebhookEndpointHealth records the health score of a webhook endpoint
func (mr *MetricsRecorder) RecordWebhookEndpointHealth(tenantID, endpointID string, healthScore float64) {
	webhookEndpointHealth.WithLabelValues(tenantID, endpointID).Set(healthScore)
}

// RecordQueueDepth records the current depth of a queue
func (mr *MetricsRecorder) RecordQueueDepth(queueName, priority string, depth float64) {
	queueDepth.WithLabelValues(queueName, priority).Set(depth)
}

// RecordQueueThroughput records queue message processing
func (mr *MetricsRecorder) RecordQueueThroughput(queueName, status string) {
	queueThroughput.WithLabelValues(queueName, status).Inc()
}

// RecordQueueProcessingLatency records queue processing latency
func (mr *MetricsRecorder) RecordQueueProcessingLatency(queueName string, duration time.Duration) {
	queueProcessingLatency.WithLabelValues(queueName).Observe(duration.Seconds())
}

// RecordDatabaseQuery records database query performance
func (mr *MetricsRecorder) RecordDatabaseQuery(operation, table string, duration time.Duration) {
	databaseQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

// RecordDatabaseConnections records database connection pool statistics
func (mr *MetricsRecorder) RecordDatabaseConnections(active, idle, total, max int) {
	databaseConnectionPool.WithLabelValues("active").Set(float64(active))
	databaseConnectionPool.WithLabelValues("idle").Set(float64(idle))
	databaseConnectionPool.WithLabelValues("total").Set(float64(total))
	databaseConnectionPool.WithLabelValues("max").Set(float64(max))
}

// RecordAPIRequestSize records the size of API requests
func (mr *MetricsRecorder) RecordAPIRequestSize(method, endpoint string, size int64) {
	apiRequestSize.WithLabelValues(method, endpoint).Observe(float64(size))
}

// RecordAPIResponseSize records the size of API responses
func (mr *MetricsRecorder) RecordAPIResponseSize(method, endpoint, statusCode string, size int64) {
	apiResponseSize.WithLabelValues(method, endpoint, statusCode).Observe(float64(size))
}

// RecordRateLimitHit records a rate limit hit
func (mr *MetricsRecorder) RecordRateLimitHit(tenantID, limitType string) {
	rateLimitHits.WithLabelValues(tenantID, limitType).Inc()
}

// RecordRateLimitRemaining records remaining rate limit quota
func (mr *MetricsRecorder) RecordRateLimitRemaining(tenantID, limitType string, remaining float64) {
	rateLimitRemaining.WithLabelValues(tenantID, limitType).Set(remaining)
}

// RecordAuthenticationFailure records authentication failures
func (mr *MetricsRecorder) RecordAuthenticationFailure(reason, sourceIP string) {
	authenticationFailures.WithLabelValues(reason, sourceIP).Inc()
}

// RecordSuspiciousActivity records suspicious activities
func (mr *MetricsRecorder) RecordSuspiciousActivity(activityType, tenantID string) {
	suspiciousActivity.WithLabelValues(activityType, tenantID).Inc()
}