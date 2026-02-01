package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"github.com/josedab/waas/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMonitoringIntegration tests the complete monitoring system integration
func TestMonitoringIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	gin.SetMode(gin.TestMode)
	
	// Setup test environment
	logger := utils.NewLogger("integration-test")
	
	// Create test Redis client (use test database)
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // Use test database
	})
	
	// Test Redis connection
	ctx := context.Background()
	err := redisClient.Ping(ctx).Err()
	if err != nil {
		t.Skip("Redis not available for integration test:", err)
	}
	
	// Initialize monitoring components
	healthChecker := NewHealthChecker(nil, redisClient, logger, "integration-test-1.0.0")
	alertManager := NewAlertManager(logger)
	metricsRecorder := NewMetricsRecorder()
	tracer := NewTracer("integration-test-service", logger)
	
	// Setup test notifiers
	testNotifier := &IntegrationTestNotifier{
		alerts:    make([]*Alert, 0),
		callCount: 0,
	}
	logNotifier := NewLogNotifier(logger)
	
	alertManager.AddNotifier(testNotifier)
	alertManager.AddNotifier(logNotifier)
	
	t.Run("Complete Monitoring Workflow", func(t *testing.T) {
		testCompleteMonitoringWorkflow(t, healthChecker, alertManager, metricsRecorder, tracer, testNotifier)
	})
	
	t.Run("Prometheus Metrics Integration", func(t *testing.T) {
		testPrometheusMetricsIntegration(t, metricsRecorder)
	})
	
	t.Run("Alert Escalation and Resolution", func(t *testing.T) {
		testAlertEscalationAndResolution(t, alertManager, testNotifier)
	})
	
	t.Run("Distributed Tracing Flow", func(t *testing.T) {
		testDistributedTracingFlow(t, tracer)
	})
	
	t.Run("Health Check Degradation", func(t *testing.T) {
		testHealthCheckDegradation(t, healthChecker, alertManager, testNotifier)
	})
	
	// Cleanup
	redisClient.Close()
}

func testCompleteMonitoringWorkflow(t *testing.T, healthChecker *HealthChecker, alertManager *AlertManager, metricsRecorder *MetricsRecorder, tracer *Tracer, testNotifier *IntegrationTestNotifier) {
	ctx := context.Background()
	
	// Step 1: Start distributed trace for a webhook delivery
	deliverySpan := tracer.TraceWebhookDelivery(ctx, "endpoint-integration-test", "tenant-integration-test")
	
	// Step 2: Record initial metrics
	metricsRecorder.RecordWebhookDeliveryLatency("tenant-integration-test", "endpoint-integration-test", "pending", 0)
	metricsRecorder.RecordQueueDepth("delivery-queue", "normal", 10)
	
	// Step 3: Simulate processing with database operations
	dbSpan := tracer.TraceDatabaseQuery(ctx, "SELECT", "webhook_endpoints")
	time.Sleep(50 * time.Millisecond) // Simulate query time
	tracer.FinishSpan(dbSpan, nil)
	
	metricsRecorder.RecordDatabaseQuery("SELECT", "webhook_endpoints", 50*time.Millisecond)
	
	// Step 4: Simulate queue processing
	queueSpan := tracer.TraceQueueOperation(ctx, "consume", "delivery-queue")
	time.Sleep(25 * time.Millisecond) // Simulate processing time
	tracer.FinishSpan(queueSpan, nil)
	
	metricsRecorder.RecordQueueProcessingLatency("delivery-queue", 25*time.Millisecond)
	metricsRecorder.RecordQueueThroughput("delivery-queue", "processed")
	
	// Step 5: Simulate webhook delivery failure (should trigger alert)
	tracer.AddLog(deliverySpan, "error", "Webhook delivery failed", map[string]string{
		"error":  "connection_timeout",
		"status": "408",
	})
	
	metricsRecorder.RecordWebhookDeliveryLatency("tenant-integration-test", "endpoint-integration-test", "failed", 5*time.Second)
	metricsRecorder.RecordWebhookDeliveryError("tenant-integration-test", "endpoint-integration-test", "timeout", "408")
	
	// Step 6: Trigger alert evaluation
	alertManager.EvaluateMetric("webhook_delivery_failure_rate", 0.1, map[string]string{
		"tenant_id":   "tenant-integration-test",
		"endpoint_id": "endpoint-integration-test",
		"component":   "delivery",
	})
	
	// Step 7: Finish trace with error
	tracer.FinishSpan(deliverySpan, assert.AnError)
	
	// Step 8: Check health status
	healthStatus := healthChecker.GetHealthStatus(ctx)
	
	// Wait for async processing
	time.Sleep(200 * time.Millisecond)
	
	// Verify complete workflow
	assert.Equal(t, TraceStatusError, deliverySpan.Status)
	assert.NotNil(t, deliverySpan.Error)
	assert.NotNil(t, deliverySpan.EndTime)
	assert.Greater(t, len(deliverySpan.Logs), 0)
	
	// Verify health check
	assert.NotNil(t, healthStatus)
	assert.Equal(t, "integration-test-1.0.0", healthStatus.Version)
	assert.Contains(t, healthStatus.Components, "redis")
	
	// Verify alerts were triggered
	activeAlerts := alertManager.GetActiveAlerts()
	assert.Greater(t, len(activeAlerts), 0, "Expected active alerts")
	
	// Verify notifications were sent
	assert.Greater(t, testNotifier.callCount, 0, "Expected alert notifications")
	assert.Greater(t, len(testNotifier.alerts), 0, "Expected alert records")
	
	// Verify alert contains correct information
	if len(testNotifier.alerts) > 0 {
		alert := testNotifier.alerts[len(testNotifier.alerts)-1]
		assert.Equal(t, AlertStatusFiring, alert.Status)
		assert.Contains(t, alert.Labels, "tenant_id")
		assert.Equal(t, "tenant-integration-test", alert.Labels["tenant_id"])
	}
}

func testPrometheusMetricsIntegration(t *testing.T, metricsRecorder *MetricsRecorder) {
	// Create a test HTTP server to serve Prometheus metrics
	router := gin.New()
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	
	server := httptest.NewServer(router)
	defer server.Close()
	
	// Record various metrics
	metricsRecorder.RecordServiceHealth("test-service", "database", HealthStatusHealthy)
	metricsRecorder.RecordServiceHealth("test-service", "redis", HealthStatusDegraded)
	metricsRecorder.RecordWebhookDeliveryLatency("tenant-metrics-test", "endpoint-1", "success", 250*time.Millisecond)
	metricsRecorder.RecordWebhookDeliveryError("tenant-metrics-test", "endpoint-2", "timeout", "408")
	metricsRecorder.RecordQueueDepth("test-queue", "high", 150)
	metricsRecorder.RecordDatabaseConnections(5, 10, 15, 20)
	
	// Fetch metrics from Prometheus endpoint
	resp, err := http.Get(server.URL + "/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	// Read response body to verify metrics are present
	body := make([]byte, 4096)
	n, _ := resp.Body.Read(body)
	metricsContent := string(body[:n])
	
	// Verify key metrics are present in the output
	assert.Contains(t, metricsContent, "webhook_service_health_status")
	assert.Contains(t, metricsContent, "webhook_delivery_latency_seconds")
	assert.Contains(t, metricsContent, "webhook_delivery_errors_total")
	assert.Contains(t, metricsContent, "webhook_queue_depth")
	assert.Contains(t, metricsContent, "webhook_database_connections")
}

func testAlertEscalationAndResolution(t *testing.T, alertManager *AlertManager, testNotifier *IntegrationTestNotifier) {
	// Reset test notifier
	testNotifier.alerts = make([]*Alert, 0)
	testNotifier.callCount = 0
	
	testLabels := map[string]string{
		"service":   "test-escalation",
		"component": "delivery",
	}
	
	// Step 1: Trigger initial alert
	alertManager.EvaluateMetric("test_escalation_metric", 10.0, testLabels)
	time.Sleep(50 * time.Millisecond)
	
	initialAlertCount := len(testNotifier.alerts)
	assert.Greater(t, initialAlertCount, 0, "Expected initial alert")
	
	// Step 2: Continue high values (should not create duplicate alerts)
	alertManager.EvaluateMetric("test_escalation_metric", 12.0, testLabels)
	alertManager.EvaluateMetric("test_escalation_metric", 15.0, testLabels)
	time.Sleep(50 * time.Millisecond)
	
	// Should not have created new alerts for same condition
	assert.Equal(t, initialAlertCount, len(testNotifier.alerts), "Should not create duplicate alerts")
	
	// Step 3: Resolve the alert
	alertManager.EvaluateMetric("test_escalation_metric", 2.0, testLabels)
	time.Sleep(50 * time.Millisecond)
	
	// Should have sent resolution notification
	assert.Greater(t, len(testNotifier.alerts), initialAlertCount, "Expected resolution notification")
	
	// Verify resolution
	activeAlerts := alertManager.GetActiveAlerts()
	hasActiveAlert := false
	for _, alert := range activeAlerts {
		if alert.Labels["service"] == "test-escalation" {
			hasActiveAlert = true
			break
		}
	}
	assert.False(t, hasActiveAlert, "Alert should be resolved")
	
	// Verify alert history contains both firing and resolved alerts
	history := alertManager.GetAlertHistory(10, "")
	firingCount := 0
	resolvedCount := 0
	for _, alert := range history {
		if alert.Labels["service"] == "test-escalation" {
			if alert.Status == AlertStatusFiring {
				firingCount++
			} else if alert.Status == AlertStatusResolved {
				resolvedCount++
			}
		}
	}
	assert.Greater(t, firingCount, 0, "Expected firing alerts in history")
	assert.Greater(t, resolvedCount, 0, "Expected resolved alerts in history")
}

func testDistributedTracingFlow(t *testing.T, tracer *Tracer) {
	ctx := context.Background()
	
	// Simulate a complex distributed operation
	
	// Step 1: API request comes in
	apiSpan := tracer.TraceAPIRequest(ctx, "POST", "/webhooks/send")
	tracer.AddTag(apiSpan, "tenant_id", "tenant-tracing-test")
	tracer.AddTag(apiSpan, "request_size", "1024")
	
	// Step 2: Database lookup
	dbSpan := tracer.TraceDatabaseQuery(ctx, "SELECT", "webhook_endpoints")
	tracer.AddLog(dbSpan, "info", "Looking up webhook endpoints", map[string]string{
		"tenant_id": "tenant-tracing-test",
		"count":     "3",
	})
	time.Sleep(10 * time.Millisecond)
	tracer.FinishSpan(dbSpan, nil)
	
	// Step 3: Queue multiple webhook deliveries
	for i := 0; i < 3; i++ {
		queueSpan := tracer.TraceQueueOperation(ctx, "publish", "delivery-queue")
		tracer.AddTag(queueSpan, "endpoint_id", fmt.Sprintf("endpoint-%d", i+1))
		time.Sleep(5 * time.Millisecond)
		tracer.FinishSpan(queueSpan, nil)
	}
	
	// Step 4: Process webhook deliveries
	for i := 0; i < 3; i++ {
		deliverySpan := tracer.TraceWebhookDelivery(ctx, fmt.Sprintf("endpoint-%d", i+1), "tenant-tracing-test")
		
		// Simulate HTTP request
		tracer.AddLog(deliverySpan, "info", "Sending HTTP request", map[string]string{
			"url":    fmt.Sprintf("https://example.com/webhook-%d", i+1),
			"method": "POST",
		})
		
		time.Sleep(time.Duration(50+i*25) * time.Millisecond) // Variable latency
		
		if i == 1 {
			// Simulate one failure
			tracer.AddLog(deliverySpan, "error", "HTTP request failed", map[string]string{
				"error":       "connection_timeout",
				"status_code": "408",
			})
			tracer.FinishSpan(deliverySpan, assert.AnError)
		} else {
			tracer.AddLog(deliverySpan, "info", "HTTP request successful", map[string]string{
				"status_code": "200",
				"response_time": fmt.Sprintf("%dms", 50+i*25),
			})
			tracer.FinishSpan(deliverySpan, nil)
		}
	}
	
	// Step 5: Finish API request
	tracer.AddTag(apiSpan, "deliveries_sent", "3")
	tracer.AddTag(apiSpan, "deliveries_failed", "1")
	tracer.FinishSpan(apiSpan, nil)
	
	// Verify trace structure
	assert.Equal(t, TraceStatusOK, apiSpan.Status)
	assert.NotNil(t, apiSpan.EndTime)
	assert.NotNil(t, apiSpan.Duration)
	assert.Equal(t, "tenant-tracing-test", apiSpan.Tags["tenant_id"])
	assert.Equal(t, "3", apiSpan.Tags["deliveries_sent"])
	
	// All spans should share the same trace ID
	traceID := apiSpan.TraceID
	assert.NotEmpty(t, traceID)
	
	// Verify trace propagation would work
	req, _ := http.NewRequest("GET", "/test", nil)
	tracer.InjectHeaders(apiSpan, req)
	
	assert.Equal(t, traceID, req.Header.Get("X-Trace-ID"))
	assert.Equal(t, apiSpan.SpanID, req.Header.Get("X-Parent-Span-ID"))
	
	// Verify extraction
	extractedCtx := tracer.ExtractTraceContext(req)
	assert.NotNil(t, extractedCtx)
	assert.Equal(t, traceID, extractedCtx.TraceID)
	assert.Equal(t, apiSpan.SpanID, extractedCtx.ParentSpanID)
}

func testHealthCheckDegradation(t *testing.T, healthChecker *HealthChecker, alertManager *AlertManager, testNotifier *IntegrationTestNotifier) {
	ctx := context.Background()
	
	// Reset test notifier
	testNotifier.alerts = make([]*Alert, 0)
	testNotifier.callCount = 0
	
	// Get initial health status
	initialHealth := healthChecker.GetHealthStatus(ctx)
	assert.NotNil(t, initialHealth)
	
	// Redis should be healthy (we have a real connection)
	redisHealth := initialHealth.Components["redis"]
	assert.Equal(t, HealthStatusHealthy, redisHealth.Status)
	
	// Database should be unhealthy (nil connection)
	dbHealth := initialHealth.Components["database"]
	assert.Equal(t, HealthStatusUnhealthy, dbHealth.Status)
	
	// Overall status should be unhealthy due to database
	assert.Equal(t, HealthStatusUnhealthy, initialHealth.Status)
	
	// Simulate health degradation by evaluating health metrics
	alertManager.EvaluateMetric("service_health_status", 0.0, map[string]string{
		"service":   "integration-test",
		"component": "database",
	})
	
	time.Sleep(50 * time.Millisecond)
	
	// Should have triggered database health alert
	activeAlerts := alertManager.GetActiveAlerts()
	hasDbAlert := false
	for _, alert := range activeAlerts {
		if alert.Labels["component"] == "database" {
			hasDbAlert = true
			break
		}
	}
	assert.True(t, hasDbAlert, "Expected database health alert")
	
	// Verify health check response structure
	assert.Contains(t, initialHealth.Components, "database")
	assert.Contains(t, initialHealth.Components, "redis")
	assert.Contains(t, initialHealth.Components, "system")
	assert.NotEmpty(t, initialHealth.Version)
	assert.NotEmpty(t, initialHealth.Uptime)
	assert.NotZero(t, initialHealth.Timestamp)
}

// IntegrationTestNotifier captures alerts for testing
type IntegrationTestNotifier struct {
	alerts    []*Alert
	callCount int
}

func (itn *IntegrationTestNotifier) SendAlert(ctx context.Context, alert *Alert) error {
	itn.callCount++
	itn.alerts = append(itn.alerts, alert)
	return nil
}

func (itn *IntegrationTestNotifier) GetName() string {
	return "integration-test"
}

// TestPrometheusMetricsCollection tests that Prometheus metrics are properly collected
func TestPrometheusMetricsCollection(t *testing.T) {
	// Create a custom registry for testing
	registry := prometheus.NewRegistry()
	
	// Create test metrics
	testCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_counter_total",
			Help: "Test counter for integration testing",
		},
		[]string{"label1", "label2"},
	)
	
	testHistogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "test_histogram_seconds",
			Help:    "Test histogram for integration testing",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)
	
	testGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "test_gauge_value",
			Help: "Test gauge for integration testing",
		},
		[]string{"instance"},
	)
	
	// Register metrics
	registry.MustRegister(testCounter, testHistogram, testGauge)
	
	// Record some test data
	testCounter.WithLabelValues("value1", "value2").Inc()
	testCounter.WithLabelValues("value1", "value2").Add(5)
	testHistogram.WithLabelValues("test_op").Observe(0.5)
	testHistogram.WithLabelValues("test_op").Observe(1.2)
	testGauge.WithLabelValues("instance1").Set(42.0)
	
	// Create HTTP handler with custom registry
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	
	// Test metrics endpoint
	req, _ := http.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	metricsOutput := w.Body.String()
	
	// Verify metrics are present
	assert.Contains(t, metricsOutput, "test_counter_total")
	assert.Contains(t, metricsOutput, "test_histogram_seconds")
	assert.Contains(t, metricsOutput, "test_gauge_value")
	
	// Verify metric values
	assert.Contains(t, metricsOutput, `test_counter_total{label1="value1",label2="value2"} 6`)
	assert.Contains(t, metricsOutput, `test_gauge_value{instance="instance1"} 42`)
	
	// Verify histogram buckets are present
	assert.Contains(t, metricsOutput, "test_histogram_seconds_bucket")
	assert.Contains(t, metricsOutput, "test_histogram_seconds_count")
	assert.Contains(t, metricsOutput, "test_histogram_seconds_sum")
}

// TestAlertNotifierReliability tests alert notifier reliability and error handling
func TestAlertNotifierReliability(t *testing.T) {
	logger := utils.NewLogger("test")
	ctx := context.Background()
	
	// Create test alert
	alert := &Alert{
		ID:          "reliability-test-1",
		Name:        "Reliability Test Alert",
		Description: "Testing notifier reliability",
		Severity:    AlertSeverityWarning,
		Status:      AlertStatusFiring,
		Labels:      map[string]string{"test": "reliability"},
		StartsAt:    time.Now(),
		Value:       10.0,
		Threshold:   5.0,
	}
	
	// Test successful notifiers
	logNotifier := NewLogNotifier(logger)
	err := logNotifier.SendAlert(ctx, alert)
	assert.NoError(t, err)
	
	// Test failing webhook notifier (invalid URL)
	webhookNotifier := NewWebhookNotifier("http://invalid-url-that-does-not-exist.local/webhook", 1*time.Second, logger)
	err = webhookNotifier.SendAlert(ctx, alert)
	assert.Error(t, err)
	
	// Test webhook notifier with timeout
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Longer than timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()
	
	timeoutNotifier := NewWebhookNotifier(slowServer.URL, 500*time.Millisecond, logger)
	err = timeoutNotifier.SendAlert(ctx, alert)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
	
	// Test webhook notifier with error response
	errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer errorServer.Close()
	
	errorNotifier := NewWebhookNotifier(errorServer.URL, 5*time.Second, logger)
	err = errorNotifier.SendAlert(ctx, alert)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
	
	// Test successful webhook notifier
	successServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
	}))
	defer successServer.Close()
	
	successNotifier := NewWebhookNotifier(successServer.URL, 5*time.Second, logger)
	err = successNotifier.SendAlert(ctx, alert)
	assert.NoError(t, err)
}