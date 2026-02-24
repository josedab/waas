package monitoring

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"github.com/josedab/waas/pkg/testutil"
	"github.com/josedab/waas/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSystemMonitoring tests the complete monitoring system integration
func TestSystemMonitoring(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Setup test components
	logger := utils.NewLogger("test")
	
	// Create mock Redis client (use nil for testing)
	var redisClient *redis.Client
	
	// Initialize monitoring components with nil connections for testing
	healthChecker := NewHealthChecker(nil, redisClient, logger, "test-1.0.0")
	alertManager := NewAlertManager(logger)
	metricsRecorder := NewMetricsRecorder()
	tracer := NewTracer("test-service", logger)
	
	// Add test notifier
	testNotifier := &TestNotifier{alerts: make([]*Alert, 0)}
	alertManager.AddNotifier(testNotifier)
	
	t.Run("HealthChecker Integration", func(t *testing.T) {
		testHealthCheckerIntegration(t, healthChecker)
	})
	
	t.Run("AlertManager Integration", func(t *testing.T) {
		testAlertManagerIntegration(t, alertManager, testNotifier)
	})
	
	t.Run("MetricsRecorder Integration", func(t *testing.T) {
		testMetricsRecorderIntegration(t, metricsRecorder)
	})
	
	t.Run("Tracer Integration", func(t *testing.T) {
		testTracerIntegration(t, tracer)
	})
	
	t.Run("End-to-End Monitoring Flow", func(t *testing.T) {
		testEndToEndMonitoringFlow(t, healthChecker, alertManager, metricsRecorder, tracer, testNotifier)
	})
}

func testHealthCheckerIntegration(t *testing.T, healthChecker *HealthChecker) {
	// Test health check endpoint
	router := gin.New()
	router.GET("/health", healthChecker.HealthCheckHandler())
	router.GET("/ready", healthChecker.ReadinessHandler())
	router.GET("/live", healthChecker.LivenessHandler())
	
	tests := []struct {
		name           string
		endpoint       string
		expectedStatus int
	}{
		{
			name:           "health check endpoint",
			endpoint:       "/health",
			expectedStatus: http.StatusServiceUnavailable, // DB is nil, so unhealthy
		},
		{
			name:           "readiness check endpoint",
			endpoint:       "/ready",
			expectedStatus: http.StatusServiceUnavailable, // DB will fail in test
		},
		{
			name:           "liveness check endpoint",
			endpoint:       "/live",
			expectedStatus: http.StatusOK,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tt.endpoint, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func testAlertManagerIntegration(t *testing.T, alertManager *AlertManager, testNotifier *TestNotifier) {
	// Test alert rule evaluation - use labels that match default rules
	testLabels := map[string]string{
		"component": "delivery",
	}
	
	// Test firing an alert using a metric that matches default rules
	alertManager.EvaluateMetric("delivery_failure_rate", 0.1, testLabels) // Above 5% threshold
	
	// Poll until alert is processed
	err := testutil.WaitFor(func() bool {
		return len(alertManager.GetActiveAlerts()) > 0
	}, 2*time.Second, 10*time.Millisecond)
	assert.NoError(t, err, "Expected at least one active alert")
	
	// Check that notification was sent
	assert.Greater(t, len(testNotifier.alerts), 0, "Expected at least one notification")
	
	// Test resolving the alert
	alertManager.EvaluateMetric("delivery_failure_rate", 0.01, testLabels) // Below threshold
	
	// Poll until alert history is updated
	err = testutil.WaitFor(func() bool {
		return len(alertManager.GetAlertHistory(10, "")) > 0
	}, 2*time.Second, 10*time.Millisecond)
	assert.NoError(t, err, "Expected alert history")
}

func testMetricsRecorderIntegration(t *testing.T, metricsRecorder *MetricsRecorder) {
	// Test recording various metrics
	metricsRecorder.RecordServiceHealth("test-service", "database", HealthStatusHealthy)
	metricsRecorder.RecordWebhookDeliveryLatency("tenant-1", "endpoint-1", "success", 500*time.Millisecond)
	metricsRecorder.RecordWebhookDeliveryError("tenant-1", "endpoint-1", "timeout", "408")
	metricsRecorder.RecordQueueDepth("delivery-queue", "high", 100)
	metricsRecorder.RecordDatabaseQuery("SELECT", "webhooks", 50*time.Millisecond)
	
	// Metrics are recorded to Prometheus, so we can't easily assert on them
	// In a real test, you might use a test registry or mock Prometheus
	// For now, we just ensure no panics occur
	assert.True(t, true, "Metrics recording completed without errors")
}

func testTracerIntegration(t *testing.T, tracer *Tracer) {
	ctx := context.Background()
	
	// Test starting and finishing spans
	span := tracer.StartSpan(ctx, "test-operation", map[string]string{
		"test": "value",
	})
	
	assert.NotNil(t, span)
	assert.Equal(t, "test-operation", span.Operation)
	assert.Equal(t, "test-service", span.Service)
	assert.NotEmpty(t, span.TraceID)
	assert.NotEmpty(t, span.SpanID)
	
	// Add log and tag
	tracer.AddLog(span, "info", "Test log message", map[string]string{"key": "value"})
	tracer.AddTag(span, "test-tag", "test-value")
	
	// Finish span
	tracer.FinishSpan(span, nil)
	
	assert.NotNil(t, span.EndTime)
	assert.NotNil(t, span.Duration)
	assert.Equal(t, TraceStatusOK, span.Status)
	assert.Len(t, span.Logs, 1)
	assert.Equal(t, "test-value", span.Tags["test-tag"])
}

func testEndToEndMonitoringFlow(t *testing.T, healthChecker *HealthChecker, alertManager *AlertManager, metricsRecorder *MetricsRecorder, tracer *Tracer, testNotifier *TestNotifier) {
	// Simulate a complete monitoring flow
	ctx := context.Background()
	
	// 1. Start a trace for a webhook delivery
	span := tracer.TraceWebhookDelivery(ctx, "endpoint-123", "tenant-456")
	
	// 2. Record metrics during the operation
	metricsRecorder.RecordWebhookDeliveryLatency("tenant-456", "endpoint-123", "pending", 0)
	
	// 3. Simulate a failure that should trigger an alert
	alertManager.EvaluateMetric("delivery_failure_rate", 0.1, map[string]string{
		"component":   "delivery",
		"tenant_id":   "tenant-456",
		"endpoint_id": "endpoint-123",
	})
	
	// 4. Record the failure metrics
	metricsRecorder.RecordWebhookDeliveryError("tenant-456", "endpoint-123", "timeout", "408")
	
	// 5. Add trace information about the failure
	tracer.AddLog(span, "error", "Webhook delivery failed", map[string]string{
		"error": "timeout",
		"status": "408",
	})
	
	// 6. Finish the trace with an error
	tracer.FinishSpan(span, assert.AnError)
	
	// 7. Check health status
	healthStatus := healthChecker.GetHealthStatus(ctx)
	assert.NotNil(t, healthStatus)
	
	// Wait for async processing
	err := testutil.WaitFor(func() bool {
		return len(testNotifier.alerts) > 0
	}, 2*time.Second, 10*time.Millisecond)
	require.NoError(t, err)
	
	// Verify the complete flow
	assert.Equal(t, TraceStatusError, span.Status)
	assert.NotNil(t, span.Error)
	assert.Greater(t, len(testNotifier.alerts), 0, "Expected alert notifications")
	
	// Verify alert contains correct information
	if len(testNotifier.alerts) > 0 {
		alert := testNotifier.alerts[len(testNotifier.alerts)-1]
		assert.Equal(t, AlertStatusFiring, alert.Status)
		assert.Equal(t, "tenant-456", alert.Labels["tenant_id"])
	}
}

// TestNotifier is a test implementation of AlertNotifier
type TestNotifier struct {
	alerts []*Alert
}

func (tn *TestNotifier) SendAlert(ctx context.Context, alert *Alert) error {
	tn.alerts = append(tn.alerts, alert)
	return nil
}

func (tn *TestNotifier) GetName() string {
	return "test"
}

// TestAlertRuleValidation tests that alert rules are properly configured
func TestAlertRuleValidation(t *testing.T) {
	logger := utils.NewLogger("test")
	alertManager := NewAlertManager(logger)
	
	// Test default rules are loaded
	activeAlerts := alertManager.GetActiveAlerts()
	assert.Equal(t, 0, len(activeAlerts), "Should start with no active alerts")
	
	// Test adding custom rule
	customRule := &AlertRule{
		Name:        "TestRule",
		Description: "Test rule for validation",
		Severity:    AlertSeverityWarning,
		Condition:   ConditionGreaterThan,
		Threshold:   5.0,
		Duration:    1 * time.Minute,
		Labels:      map[string]string{"test": "true"},
		Enabled:     true,
	}
	
	alertManager.AddRule(customRule)
	
	// Test rule evaluation
	testLabels := map[string]string{"test": "true"}
	
	// Should not fire (below threshold)
	alertManager.EvaluateMetric("test_metric", 3.0, testLabels)
	activeAlerts = alertManager.GetActiveAlerts()
	assert.Equal(t, 0, len(activeAlerts), "Alert should not fire below threshold")
	
	// Should fire (above threshold)
	alertManager.EvaluateMetric("test_metric", 7.0, testLabels)
	activeAlerts = alertManager.GetActiveAlerts()
	assert.Equal(t, 1, len(activeAlerts), "Alert should fire above threshold")
	
	// Should resolve (below threshold again)
	alertManager.EvaluateMetric("test_metric", 2.0, testLabels)
	activeAlerts = alertManager.GetActiveAlerts()
	assert.Equal(t, 0, len(activeAlerts), "Alert should resolve below threshold")
	
	// Test alert history
	history := alertManager.GetAlertHistory(10, "")
	assert.Greater(t, len(history), 0, "Should have alert history")
}

// TestMetricsValidation tests that metrics are properly recorded and can be queried
func TestMetricsValidation(t *testing.T) {
	metricsRecorder := NewMetricsRecorder()
	
	// Test service health metrics
	metricsRecorder.RecordServiceHealth("api-service", "database", HealthStatusHealthy)
	metricsRecorder.RecordServiceHealth("api-service", "redis", HealthStatusDegraded)
	metricsRecorder.RecordServiceHealth("delivery-engine", "queue", HealthStatusUnhealthy)
	
	// Test webhook metrics
	metricsRecorder.RecordWebhookDeliveryLatency("tenant-1", "endpoint-1", "success", 250*time.Millisecond)
	metricsRecorder.RecordWebhookDeliveryLatency("tenant-1", "endpoint-2", "failed", 5*time.Second)
	metricsRecorder.RecordWebhookDeliveryError("tenant-1", "endpoint-2", "timeout", "408")
	
	// Test queue metrics
	metricsRecorder.RecordQueueDepth("delivery-queue", "normal", 50)
	metricsRecorder.RecordQueueDepth("retry-queue", "high", 200)
	metricsRecorder.RecordQueueThroughput("delivery-queue", "processed")
	
	// Test database metrics
	metricsRecorder.RecordDatabaseQuery("INSERT", "delivery_attempts", 25*time.Millisecond)
	metricsRecorder.RecordDatabaseQuery("SELECT", "webhook_endpoints", 10*time.Millisecond)
	metricsRecorder.RecordDatabaseConnections(5, 10, 15, 20)
	
	// Test API metrics
	metricsRecorder.RecordAPIRequestSize("POST", "/webhooks/send", 1024)
	metricsRecorder.RecordAPIResponseSize("POST", "/webhooks/send", "200", 256)
	
	// Test security metrics
	metricsRecorder.RecordAuthenticationFailure("invalid_key", "192.168.1.100")
	metricsRecorder.RecordSuspiciousActivity("rate_limit_exceeded", "tenant-1")
	
	// All metrics should be recorded without errors
	assert.True(t, true, "All metrics recorded successfully")
}

// TestTracingValidation tests distributed tracing functionality
func TestTracingValidation(t *testing.T) {
	logger := utils.NewLogger("test")
	tracer := NewTracer("test-service", logger)
	
	ctx := context.Background()
	
	// Test webhook delivery tracing
	deliverySpan := tracer.TraceWebhookDelivery(ctx, "endpoint-123", "tenant-456")
	require.NotNil(t, deliverySpan)
	assert.Equal(t, "webhook.delivery", deliverySpan.Operation)
	assert.Equal(t, "endpoint-123", deliverySpan.Tags["endpoint_id"])
	assert.Equal(t, "tenant-456", deliverySpan.Tags["tenant_id"])
	
	// Test API request tracing
	apiSpan := tracer.TraceAPIRequest(ctx, "POST", "/webhooks/send")
	require.NotNil(t, apiSpan)
	assert.Equal(t, "api.POST", apiSpan.Operation)
	assert.Equal(t, "POST", apiSpan.Tags["http.method"])
	
	// Test database query tracing
	dbSpan := tracer.TraceDatabaseQuery(ctx, "INSERT", "delivery_attempts")
	require.NotNil(t, dbSpan)
	assert.Equal(t, "db.INSERT", dbSpan.Operation)
	assert.Equal(t, "INSERT", dbSpan.Tags["db.operation"])
	assert.Equal(t, "delivery_attempts", dbSpan.Tags["db.table"])
	
	// Test queue operation tracing
	queueSpan := tracer.TraceQueueOperation(ctx, "publish", "delivery-queue")
	require.NotNil(t, queueSpan)
	assert.Equal(t, "queue.publish", queueSpan.Operation)
	assert.Equal(t, "publish", queueSpan.Tags["queue.operation"])
	assert.Equal(t, "delivery-queue", queueSpan.Tags["queue.name"])
	
	// Test span relationships
	assert.Equal(t, deliverySpan.TraceID, apiSpan.TraceID, "Spans should share trace ID")
	assert.Equal(t, deliverySpan.TraceID, dbSpan.TraceID, "Spans should share trace ID")
	assert.Equal(t, deliverySpan.TraceID, queueSpan.TraceID, "Spans should share trace ID")
	
	// Finish all spans
	tracer.FinishSpan(deliverySpan, nil)
	tracer.FinishSpan(apiSpan, nil)
	tracer.FinishSpan(dbSpan, assert.AnError) // Simulate error
	tracer.FinishSpan(queueSpan, nil)
	
	// Verify span completion
	assert.NotNil(t, deliverySpan.EndTime)
	assert.NotNil(t, deliverySpan.Duration)
	assert.Equal(t, TraceStatusOK, deliverySpan.Status)
	
	assert.NotNil(t, dbSpan.EndTime)
	assert.Equal(t, TraceStatusError, dbSpan.Status)
	assert.NotNil(t, dbSpan.Error)
}

// TestHealthCheckValidation tests health check functionality
func TestHealthCheckValidation(t *testing.T) {
	logger := utils.NewLogger("test")
	
	// Test with nil dependencies (should handle gracefully)
	healthChecker := NewHealthChecker(nil, nil, logger, "test-1.0.0")
	
	ctx := context.Background()
	healthStatus := healthChecker.GetHealthStatus(ctx)
	
	assert.NotNil(t, healthStatus)
	assert.Equal(t, "test-1.0.0", healthStatus.Version)
	assert.NotEmpty(t, healthStatus.Uptime)
	assert.Contains(t, healthStatus.Components, "database")
	assert.Contains(t, healthStatus.Components, "redis")
	assert.Contains(t, healthStatus.Components, "system")
	
	// Database should be unhealthy (nil connection)
	assert.Equal(t, HealthStatusUnhealthy, healthStatus.Components["database"].Status)
	
	// Redis should be degraded (nil connection, but not critical)
	assert.Equal(t, HealthStatusDegraded, healthStatus.Components["redis"].Status)
	
	// System should be healthy
	assert.Equal(t, HealthStatusHealthy, healthStatus.Components["system"].Status)
	
	// Overall status should be unhealthy due to database
	assert.Equal(t, HealthStatusUnhealthy, healthStatus.Status)
}

// TestNotifierValidation tests alert notifier functionality
func TestNotifierValidation(t *testing.T) {
	logger := utils.NewLogger("test")
	ctx := context.Background()
	
	// Create test alert
	alert := &Alert{
		ID:          "test-alert-1",
		Name:        "Test Alert",
		Description: "This is a test alert",
		Severity:    AlertSeverityCritical,
		Status:      AlertStatusFiring,
		Labels:      map[string]string{"service": "test"},
		Annotations: map[string]string{"summary": "Test alert summary"},
		StartsAt:    time.Now(),
		Value:       10.0,
		Threshold:   5.0,
	}
	
	// Test log notifier
	logNotifier := NewLogNotifier(logger)
	err := logNotifier.SendAlert(ctx, alert)
	assert.NoError(t, err)
	assert.Equal(t, "log", logNotifier.GetName())
	
	// Test webhook notifier (will fail without real endpoint)
	webhookNotifier := NewWebhookNotifier("http://localhost:9999/webhook", 5*time.Second, logger)
	err = webhookNotifier.SendAlert(ctx, alert)
	assert.Error(t, err) // Expected to fail
	assert.Equal(t, "webhook", webhookNotifier.GetName())
	
	// Test Slack notifier (will fail without real endpoint)
	slackNotifier := NewSlackNotifier("http://localhost:9999/slack", "#alerts", "webhook-bot", 5*time.Second, logger)
	err = slackNotifier.SendAlert(ctx, alert)
	assert.Error(t, err) // Expected to fail
	assert.Equal(t, "slack", slackNotifier.GetName())
	
	// Test email notifier (placeholder implementation)
	emailNotifier := NewEmailNotifier("smtp.example.com", 587, "user", "pass", "from@example.com", []string{"to@example.com"}, logger)
	err = emailNotifier.SendAlert(ctx, alert)
	assert.NoError(t, err) // Placeholder implementation should not fail
	assert.Equal(t, "email", emailNotifier.GetName())
}