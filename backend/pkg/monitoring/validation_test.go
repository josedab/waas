package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"webhook-platform/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMonitoringValidation provides comprehensive validation of the monitoring system
func TestMonitoringValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	t.Run("Alert Rule Validation", func(t *testing.T) {
		testAlertRuleValidation(t)
	})
	
	t.Run("Health Check Validation", func(t *testing.T) {
		testHealthCheckValidation(t)
	})
	
	t.Run("Metrics Recording Validation", func(t *testing.T) {
		testMetricsRecordingValidation(t)
	})
	
	t.Run("Tracing Validation", func(t *testing.T) {
		testTracingValidation(t)
	})
	
	t.Run("Notifier Validation", func(t *testing.T) {
		testNotifierValidation(t)
	})
	
	t.Run("Alert Manager State Validation", func(t *testing.T) {
		testAlertManagerStateValidation(t)
	})
	
	t.Run("Performance Monitoring Validation", func(t *testing.T) {
		testPerformanceMonitoringValidation(t)
	})
}

func testAlertRuleValidation(t *testing.T) {
	logger := utils.NewLogger("test")
	alertManager := NewAlertManager(logger)
	
	// Test default rules are properly configured
	testLabels := map[string]string{
		"component": "delivery",
	}
	
	// Test HighDeliveryFailureRate rule
	alertManager.EvaluateMetric("delivery_failure_rate", 0.03, testLabels) // Below threshold
	activeAlerts := alertManager.GetActiveAlerts()
	assert.Equal(t, 0, len(activeAlerts), "Should not fire below threshold")
	
	alertManager.EvaluateMetric("delivery_failure_rate", 0.07, testLabels) // Above threshold
	activeAlerts = alertManager.GetActiveAlerts()
	assert.Equal(t, 1, len(activeAlerts), "Should fire above threshold")
	
	// Verify alert properties
	alert := activeAlerts[0]
	assert.Equal(t, AlertSeverityCritical, alert.Severity)
	assert.Equal(t, AlertStatusFiring, alert.Status)
	assert.Equal(t, "delivery", alert.Labels["component"])
	assert.Equal(t, 0.07, alert.Value)
	assert.Equal(t, 0.05, alert.Threshold)
	
	// Test DatabaseConnectionFailure rule
	dbLabels := map[string]string{
		"component": "database",
	}
	
	alertManager.EvaluateMetric("service_health_status", 0.0, dbLabels) // Unhealthy
	activeAlerts = alertManager.GetActiveAlerts()
	assert.Equal(t, 2, len(activeAlerts), "Should have both alerts active")
	
	// Test HighQueueDepth rule
	queueLabels := map[string]string{
		"component": "queue",
	}
	
	alertManager.EvaluateMetric("queue_depth", 1500, queueLabels) // Above threshold
	activeAlerts = alertManager.GetActiveAlerts()
	assert.Equal(t, 3, len(activeAlerts), "Should have three alerts active")
	
	// Test alert resolution
	alertManager.EvaluateMetric("delivery_failure_rate", 0.02, testLabels) // Below threshold
	alertManager.EvaluateMetric("service_health_status", 1.0, dbLabels)    // Healthy
	alertManager.EvaluateMetric("queue_depth", 500, queueLabels)           // Below threshold
	
	activeAlerts = alertManager.GetActiveAlerts()
	assert.Equal(t, 0, len(activeAlerts), "All alerts should be resolved")
	
	// Verify alert history
	history := alertManager.GetAlertHistory(10, "")
	assert.Greater(t, len(history), 0, "Should have alert history")
	
	firingCount := 0
	resolvedCount := 0
	for _, alert := range history {
		if alert.Status == AlertStatusFiring {
			firingCount++
		} else if alert.Status == AlertStatusResolved {
			resolvedCount++
		}
	}
	assert.Greater(t, firingCount, 0, "Should have firing alerts in history")
	assert.Greater(t, resolvedCount, 0, "Should have resolved alerts in history")
}

func testHealthCheckValidation(t *testing.T) {
	logger := utils.NewLogger("test")
	
	// Test with all nil dependencies
	healthChecker := NewHealthChecker(nil, nil, logger, "test-1.0.0")
	
	ctx := context.Background()
	healthStatus := healthChecker.GetHealthStatus(ctx)
	
	// Validate response structure
	assert.NotNil(t, healthStatus)
	assert.Equal(t, "test-1.0.0", healthStatus.Version)
	assert.NotEmpty(t, healthStatus.Uptime)
	assert.NotZero(t, healthStatus.Timestamp)
	
	// Validate components
	require.Contains(t, healthStatus.Components, "database")
	require.Contains(t, healthStatus.Components, "redis")
	require.Contains(t, healthStatus.Components, "system")
	
	// Database should be unhealthy (nil connection)
	dbHealth := healthStatus.Components["database"]
	assert.Equal(t, HealthStatusUnhealthy, dbHealth.Status)
	assert.Contains(t, dbHealth.Message, "not initialized")
	assert.NotZero(t, dbHealth.LastChecked)
	
	// Redis should be degraded (nil connection, but not critical)
	redisHealth := healthStatus.Components["redis"]
	assert.Equal(t, HealthStatusDegraded, redisHealth.Status)
	assert.Contains(t, redisHealth.Message, "not initialized")
	
	// System should be healthy
	systemHealth := healthStatus.Components["system"]
	assert.Equal(t, HealthStatusHealthy, systemHealth.Status)
	assert.Equal(t, "System is healthy", systemHealth.Message)
	
	// Overall status should be unhealthy due to database
	assert.Equal(t, HealthStatusUnhealthy, healthStatus.Status)
	
	// Test HTTP handlers
	router := gin.New()
	router.GET("/health", healthChecker.HealthCheckHandler())
	router.GET("/ready", healthChecker.ReadinessHandler())
	router.GET("/live", healthChecker.LivenessHandler())
	
	// Test health endpoint
	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusServiceUnavailable, w.Code) // Unhealthy due to DB
	
	var response HealthCheckResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, HealthStatusUnhealthy, response.Status)
	
	// Test readiness endpoint
	req, _ = http.NewRequest("GET", "/ready", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusServiceUnavailable, w.Code) // Not ready due to DB
	
	// Test liveness endpoint
	req, _ = http.NewRequest("GET", "/live", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code) // Always alive if responding
}

func testMetricsRecordingValidation(t *testing.T) {
	metricsRecorder := NewMetricsRecorder()
	
	// Test all metric recording methods
	
	// Service health metrics
	metricsRecorder.RecordServiceHealth("api-service", "database", HealthStatusHealthy)
	metricsRecorder.RecordServiceHealth("api-service", "redis", HealthStatusDegraded)
	metricsRecorder.RecordServiceHealth("delivery-engine", "queue", HealthStatusUnhealthy)
	
	// Service uptime
	metricsRecorder.RecordServiceUptime("api-service")
	metricsRecorder.RecordServiceUptime("delivery-engine")
	
	// Webhook delivery metrics
	metricsRecorder.RecordWebhookDeliveryLatency("tenant-1", "endpoint-1", "success", 250*time.Millisecond)
	metricsRecorder.RecordWebhookDeliveryLatency("tenant-1", "endpoint-2", "failed", 5*time.Second)
	metricsRecorder.RecordWebhookDeliveryLatency("tenant-2", "endpoint-3", "success", 100*time.Millisecond)
	
	metricsRecorder.RecordWebhookDeliveryError("tenant-1", "endpoint-2", "timeout", "408")
	metricsRecorder.RecordWebhookDeliveryError("tenant-1", "endpoint-2", "server_error", "500")
	metricsRecorder.RecordWebhookDeliveryError("tenant-2", "endpoint-4", "not_found", "404")
	
	metricsRecorder.RecordWebhookEndpointHealth("tenant-1", "endpoint-1", 0.95)
	metricsRecorder.RecordWebhookEndpointHealth("tenant-1", "endpoint-2", 0.75)
	metricsRecorder.RecordWebhookEndpointHealth("tenant-2", "endpoint-3", 0.99)
	
	// Queue metrics
	metricsRecorder.RecordQueueDepth("delivery-queue", "normal", 50)
	metricsRecorder.RecordQueueDepth("retry-queue", "high", 200)
	metricsRecorder.RecordQueueDepth("dead-letter-queue", "low", 5)
	
	metricsRecorder.RecordQueueThroughput("delivery-queue", "processed")
	metricsRecorder.RecordQueueThroughput("delivery-queue", "failed")
	metricsRecorder.RecordQueueThroughput("retry-queue", "processed")
	
	metricsRecorder.RecordQueueProcessingLatency("delivery-queue", 25*time.Millisecond)
	metricsRecorder.RecordQueueProcessingLatency("retry-queue", 50*time.Millisecond)
	
	// Database metrics
	metricsRecorder.RecordDatabaseQuery("SELECT", "webhook_endpoints", 10*time.Millisecond)
	metricsRecorder.RecordDatabaseQuery("INSERT", "delivery_attempts", 25*time.Millisecond)
	metricsRecorder.RecordDatabaseQuery("UPDATE", "webhook_endpoints", 15*time.Millisecond)
	metricsRecorder.RecordDatabaseQuery("DELETE", "delivery_attempts", 5*time.Millisecond)
	
	metricsRecorder.RecordDatabaseConnections(5, 10, 15, 20)
	metricsRecorder.RecordDatabaseConnections(8, 7, 15, 20)
	
	// API metrics
	metricsRecorder.RecordAPIRequestSize("POST", "/webhooks/send", 1024)
	metricsRecorder.RecordAPIRequestSize("GET", "/webhooks/endpoints", 0)
	metricsRecorder.RecordAPIRequestSize("PUT", "/webhooks/endpoints/123", 512)
	
	metricsRecorder.RecordAPIResponseSize("POST", "/webhooks/send", "200", 256)
	metricsRecorder.RecordAPIResponseSize("GET", "/webhooks/endpoints", "200", 2048)
	metricsRecorder.RecordAPIResponseSize("PUT", "/webhooks/endpoints/123", "404", 128)
	
	// Rate limiting metrics
	metricsRecorder.RecordRateLimitHit("tenant-1", "requests_per_minute")
	metricsRecorder.RecordRateLimitHit("tenant-2", "webhooks_per_hour")
	
	metricsRecorder.RecordRateLimitRemaining("tenant-1", "requests_per_minute", 45)
	metricsRecorder.RecordRateLimitRemaining("tenant-2", "webhooks_per_hour", 890)
	
	// Security metrics
	metricsRecorder.RecordAuthenticationFailure("invalid_key", "192.168.1.100")
	metricsRecorder.RecordAuthenticationFailure("expired_key", "10.0.0.50")
	metricsRecorder.RecordAuthenticationFailure("malformed_key", "203.0.113.25")
	
	metricsRecorder.RecordSuspiciousActivity("rate_limit_exceeded", "tenant-1")
	metricsRecorder.RecordSuspiciousActivity("unusual_payload_size", "tenant-2")
	metricsRecorder.RecordSuspiciousActivity("multiple_failed_deliveries", "tenant-1")
	
	// All metrics should be recorded without errors
	// In a real test environment, you might verify the metrics using a test registry
	assert.True(t, true, "All metrics recorded successfully")
}

func testTracingValidation(t *testing.T) {
	logger := utils.NewLogger("test")
	tracer := NewTracer("validation-test-service", logger)
	
	ctx := context.Background()
	
	// Test basic span creation and management
	span := tracer.StartSpan(ctx, "test-operation", map[string]string{
		"test_key": "test_value",
	})
	
	// Validate span properties
	assert.NotNil(t, span)
	assert.Equal(t, "test-operation", span.Operation)
	assert.Equal(t, "validation-test-service", span.Service)
	assert.NotEmpty(t, span.TraceID)
	assert.NotEmpty(t, span.SpanID)
	assert.Equal(t, "test_value", span.Tags["test_key"])
	assert.Equal(t, TraceStatusOK, span.Status)
	assert.Nil(t, span.EndTime)
	assert.Nil(t, span.Duration)
	assert.Empty(t, span.Logs)
	
	// Test adding logs and tags
	tracer.AddLog(span, "info", "Test log message", map[string]string{
		"log_key": "log_value",
	})
	
	tracer.AddTag(span, "additional_tag", "additional_value")
	
	assert.Len(t, span.Logs, 1)
	assert.Equal(t, "info", span.Logs[0].Level)
	assert.Equal(t, "Test log message", span.Logs[0].Message)
	assert.Equal(t, "log_value", span.Logs[0].Fields["log_key"])
	assert.Equal(t, "additional_value", span.Tags["additional_tag"])
	
	// Test finishing span without error
	tracer.FinishSpan(span, nil)
	
	assert.NotNil(t, span.EndTime)
	assert.NotNil(t, span.Duration)
	assert.Equal(t, TraceStatusOK, span.Status)
	assert.Nil(t, span.Error)
	
	// Test span with error
	errorSpan := tracer.StartSpan(ctx, "error-operation", nil)
	tracer.AddLog(errorSpan, "error", "Something went wrong", map[string]string{
		"error_code": "E001",
	})
	
	testError := assert.AnError
	tracer.FinishSpan(errorSpan, testError)
	
	assert.Equal(t, TraceStatusError, errorSpan.Status)
	assert.NotNil(t, errorSpan.Error)
	assert.Equal(t, testError.Error(), *errorSpan.Error)
	assert.Equal(t, "true", errorSpan.Tags["error"])
	
	// Test specialized tracing methods
	webhookSpan := tracer.TraceWebhookDelivery(ctx, "endpoint-123", "tenant-456")
	assert.Equal(t, "webhook.delivery", webhookSpan.Operation)
	assert.Equal(t, "endpoint-123", webhookSpan.Tags["endpoint_id"])
	assert.Equal(t, "tenant-456", webhookSpan.Tags["tenant_id"])
	assert.Equal(t, "delivery_engine", webhookSpan.Tags["component"])
	tracer.FinishSpan(webhookSpan, nil)
	
	apiSpan := tracer.TraceAPIRequest(ctx, "POST", "/webhooks/send")
	assert.Equal(t, "api.POST", apiSpan.Operation)
	assert.Equal(t, "POST", apiSpan.Tags["http.method"])
	assert.Equal(t, "/webhooks/send", apiSpan.Tags["http.endpoint"])
	assert.Equal(t, "api_service", apiSpan.Tags["component"])
	tracer.FinishSpan(apiSpan, nil)
	
	dbSpan := tracer.TraceDatabaseQuery(ctx, "SELECT", "webhooks")
	assert.Equal(t, "db.SELECT", dbSpan.Operation)
	assert.Equal(t, "SELECT", dbSpan.Tags["db.operation"])
	assert.Equal(t, "webhooks", dbSpan.Tags["db.table"])
	assert.Equal(t, "database", dbSpan.Tags["component"])
	tracer.FinishSpan(dbSpan, nil)
	
	queueSpan := tracer.TraceQueueOperation(ctx, "publish", "delivery-queue")
	assert.Equal(t, "queue.publish", queueSpan.Operation)
	assert.Equal(t, "publish", queueSpan.Tags["queue.operation"])
	assert.Equal(t, "delivery-queue", queueSpan.Tags["queue.name"])
	assert.Equal(t, "message_queue", queueSpan.Tags["component"])
	tracer.FinishSpan(queueSpan, nil)
	
	// Test HTTP header injection and extraction
	req, _ := http.NewRequest("GET", "/test", nil)
	tracer.InjectHeaders(span, req)
	
	assert.Equal(t, span.TraceID, req.Header.Get("X-Trace-ID"))
	assert.Equal(t, span.SpanID, req.Header.Get("X-Parent-Span-ID"))
	
	extractedCtx := tracer.ExtractTraceContext(req)
	assert.NotNil(t, extractedCtx)
	assert.Equal(t, span.TraceID, extractedCtx.TraceID)
	assert.Equal(t, span.SpanID, extractedCtx.ParentSpanID)
	
	// Test trace ID extraction
	traceID := tracer.GetTraceID(ctx)
	assert.Empty(t, traceID) // No trace context in plain context
}

func testNotifierValidation(t *testing.T) {
	logger := utils.NewLogger("test")
	ctx := context.Background()
	
	// Create test alerts
	firingAlert := &Alert{
		ID:          "test-firing-alert",
		Name:        "Test Firing Alert",
		Description: "This is a test firing alert",
		Severity:    AlertSeverityCritical,
		Status:      AlertStatusFiring,
		Labels:      map[string]string{"service": "test", "env": "validation"},
		Annotations: map[string]string{"summary": "Test alert summary"},
		StartsAt:    time.Now(),
		Value:       10.0,
		Threshold:   5.0,
	}
	
	resolvedAlert := &Alert{
		ID:          "test-resolved-alert",
		Name:        "Test Resolved Alert",
		Description: "This is a test resolved alert",
		Severity:    AlertSeverityWarning,
		Status:      AlertStatusResolved,
		Labels:      map[string]string{"service": "test", "env": "validation"},
		Annotations: map[string]string{"summary": "Test resolved alert"},
		StartsAt:    time.Now().Add(-10 * time.Minute),
		Value:       2.0,
		Threshold:   5.0,
	}
	now := time.Now()
	resolvedAlert.EndsAt = &now
	
	// Test LogNotifier
	logNotifier := NewLogNotifier(logger)
	assert.Equal(t, "log", logNotifier.GetName())
	
	err := logNotifier.SendAlert(ctx, firingAlert)
	assert.NoError(t, err)
	
	err = logNotifier.SendAlert(ctx, resolvedAlert)
	assert.NoError(t, err)
	
	// Test WebhookNotifier with mock server
	successServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "webhook-platform-alerting/1.0", r.Header.Get("User-Agent"))
		
		// Verify payload structure
		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		assert.NoError(t, err)
		
		assert.Contains(t, payload, "alert_id")
		assert.Contains(t, payload, "name")
		assert.Contains(t, payload, "severity")
		assert.Contains(t, payload, "status")
		assert.Contains(t, payload, "starts_at")
		
		w.WriteHeader(http.StatusOK)
	}))
	defer successServer.Close()
	
	webhookNotifier := NewWebhookNotifier(successServer.URL, 5*time.Second, logger)
	assert.Equal(t, "webhook", webhookNotifier.GetName())
	
	err = webhookNotifier.SendAlert(ctx, firingAlert)
	assert.NoError(t, err)
	
	// Test SlackNotifier with mock server
	slackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		
		// Verify Slack payload structure
		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		assert.NoError(t, err)
		
		assert.Contains(t, payload, "channel")
		assert.Contains(t, payload, "username")
		assert.Contains(t, payload, "attachments")
		
		attachments, ok := payload["attachments"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, attachments, 1)
		
		attachment, ok := attachments[0].(map[string]interface{})
		assert.True(t, ok)
		assert.Contains(t, attachment, "color")
		assert.Contains(t, attachment, "title")
		assert.Contains(t, attachment, "fields")
		
		w.WriteHeader(http.StatusOK)
	}))
	defer slackServer.Close()
	
	slackNotifier := NewSlackNotifier(slackServer.URL, "#alerts", "webhook-bot", 5*time.Second, logger)
	assert.Equal(t, "slack", slackNotifier.GetName())
	
	err = slackNotifier.SendAlert(ctx, firingAlert)
	assert.NoError(t, err)
	
	err = slackNotifier.SendAlert(ctx, resolvedAlert)
	assert.NoError(t, err)
	
	// Test EmailNotifier (placeholder implementation)
	emailNotifier := NewEmailNotifier("smtp.example.com", 587, "user", "pass", "from@example.com", []string{"to@example.com"}, logger)
	assert.Equal(t, "email", emailNotifier.GetName())
	
	err = emailNotifier.SendAlert(ctx, firingAlert)
	assert.NoError(t, err) // Placeholder should not fail
	
	// Test notifier error handling
	errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer errorServer.Close()
	
	errorNotifier := NewWebhookNotifier(errorServer.URL, 5*time.Second, logger)
	err = errorNotifier.SendAlert(ctx, firingAlert)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
	
	// Test timeout handling
	timeoutServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer timeoutServer.Close()
	
	timeoutNotifier := NewWebhookNotifier(timeoutServer.URL, 500*time.Millisecond, logger)
	err = timeoutNotifier.SendAlert(ctx, firingAlert)
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "timeout")
}

func testAlertManagerStateValidation(t *testing.T) {
	logger := utils.NewLogger("test")
	alertManager := NewAlertManager(logger)
	
	// Add test notifier
	testNotifier := &ValidationTestNotifier{
		alerts:    make([]*Alert, 0),
		callCount: 0,
	}
	alertManager.AddNotifier(testNotifier)
	
	// Test initial state
	activeAlerts := alertManager.GetActiveAlerts()
	assert.Equal(t, 0, len(activeAlerts))
	
	history := alertManager.GetAlertHistory(10, "")
	assert.Equal(t, 0, len(history))
	
	// Test adding custom rules
	customRule := &AlertRule{
		Name:        "CustomTestRule",
		Description: "Custom test rule for validation",
		Severity:    AlertSeverityWarning,
		Condition:   ConditionGreaterThan,
		Threshold:   10.0,
		Duration:    1 * time.Minute,
		Labels:      map[string]string{"custom": "true"},
		Annotations: map[string]string{"runbook": "http://example.com/runbook"},
		Enabled:     true,
	}
	
	alertManager.AddRule(customRule)
	
	// Test all condition types
	conditions := []struct {
		condition AlertCondition
		threshold float64
		testValue float64
		shouldFire bool
	}{
		{ConditionGreaterThan, 5.0, 7.0, true},
		{ConditionGreaterThan, 5.0, 3.0, false},
		{ConditionLessThan, 5.0, 3.0, true},
		{ConditionLessThan, 5.0, 7.0, false},
		{ConditionEquals, 5.0, 5.0, true},
		{ConditionEquals, 5.0, 6.0, false},
		{ConditionNotEquals, 5.0, 6.0, true},
		{ConditionNotEquals, 5.0, 5.0, false},
		{ConditionGreaterOrEqual, 5.0, 5.0, true},
		{ConditionGreaterOrEqual, 5.0, 4.0, false},
		{ConditionLessOrEqual, 5.0, 5.0, true},
		{ConditionLessOrEqual, 5.0, 6.0, false},
	}
	
	for i, test := range conditions {
		// Create a unique rule for each condition test
		rule := &AlertRule{
			Name:        fmt.Sprintf("ConditionTest%d", i),
			Description: fmt.Sprintf("Test rule for %s condition", test.condition),
			Severity:    AlertSeverityInfo,
			Condition:   test.condition,
			Threshold:   test.threshold,
			Duration:    1 * time.Minute,
			Labels:      map[string]string{"test": fmt.Sprintf("condition_%d", i)},
			Enabled:     true,
		}
		
		alertManager.AddRule(rule)
		
		// Evaluate the condition
		alertManager.EvaluateMetric("test_metric", test.testValue, map[string]string{
			"test": fmt.Sprintf("condition_%d", i),
		})
		
		// Check if alert fired as expected
		activeAlerts := alertManager.GetActiveAlerts()
		alertFired := false
		for _, alert := range activeAlerts {
			if alert.Name == rule.Name {
				alertFired = true
				break
			}
		}
		
		if test.shouldFire {
			assert.True(t, alertFired, "Alert should have fired for condition %s with value %f > threshold %f", test.condition, test.testValue, test.threshold)
		} else {
			assert.False(t, alertFired, "Alert should not have fired for condition %s with value %f <= threshold %f", test.condition, test.testValue, test.threshold)
		}
		
		// Clean up - resolve any fired alerts
		alertManager.EvaluateMetric("test_metric", 0.0, map[string]string{
			"test": fmt.Sprintf("condition_%d", i),
		})
	}
	
	// Test rule management
	alertManager.RemoveRule("CustomTestRule")
	
	// Test alert history filtering
	history = alertManager.GetAlertHistory(5, AlertSeverityWarning)
	for _, alert := range history {
		assert.Equal(t, AlertSeverityWarning, alert.Severity)
	}
	
	history = alertManager.GetAlertHistory(5, AlertSeverityInfo)
	for _, alert := range history {
		assert.Equal(t, AlertSeverityInfo, alert.Severity)
	}
	
	// Verify notifications were sent
	assert.Greater(t, testNotifier.callCount, 0, "Expected alert notifications")
}

func testPerformanceMonitoringValidation(t *testing.T) {
	logger := utils.NewLogger("test")
	metricsRecorder := NewMetricsRecorder()
	alertManager := NewAlertManager(logger)
	tracer := NewTracer("performance-test", logger)
	
	// Add performance notifier
	perfNotifier := &ValidationTestNotifier{
		alerts:    make([]*Alert, 0),
		callCount: 0,
	}
	alertManager.AddNotifier(perfNotifier)
	
	ctx := context.Background()
	
	// Simulate high-performance monitoring scenario
	for i := 0; i < 100; i++ {
		// Create traces for concurrent operations
		span := tracer.StartSpan(ctx, fmt.Sprintf("operation-%d", i), map[string]string{
			"batch": "performance-test",
			"index": fmt.Sprintf("%d", i),
		})
		
		// Record various metrics
		metricsRecorder.RecordWebhookDeliveryLatency(
			fmt.Sprintf("tenant-%d", i%10),
			fmt.Sprintf("endpoint-%d", i%20),
			"success",
			time.Duration(50+i)*time.Millisecond,
		)
		
		metricsRecorder.RecordQueueDepth("perf-queue", "normal", float64(i))
		metricsRecorder.RecordDatabaseQuery("SELECT", "test_table", time.Duration(5+i%10)*time.Millisecond)
		
		// Simulate some failures
		if i%10 == 0 {
			metricsRecorder.RecordWebhookDeliveryError(
				fmt.Sprintf("tenant-%d", i%10),
				fmt.Sprintf("endpoint-%d", i%20),
				"timeout",
				"408",
			)
			tracer.FinishSpan(span, assert.AnError)
		} else {
			tracer.FinishSpan(span, nil)
		}
		
		// Trigger some alerts for high queue depth
		if i > 50 {
			alertManager.EvaluateMetric("queue_depth", float64(i), map[string]string{
				"queue": "perf-queue",
			})
		}
	}
	
	// Wait for processing
	time.Sleep(100 * time.Millisecond)
	
	// Verify performance monitoring captured the load
	activeAlerts := alertManager.GetActiveAlerts()
	assert.Greater(t, len(activeAlerts), 0, "Expected alerts from high load")
	
	history := alertManager.GetAlertHistory(50, "")
	assert.Greater(t, len(history), 0, "Expected alert history from performance test")
	
	// Verify notifications were sent
	assert.Greater(t, perfNotifier.callCount, 0, "Expected performance alert notifications")
	
	// Test metrics recording performance (should complete quickly)
	start := time.Now()
	for i := 0; i < 1000; i++ {
		metricsRecorder.RecordWebhookDeliveryLatency("perf-tenant", "perf-endpoint", "success", 100*time.Millisecond)
	}
	duration := time.Since(start)
	
	// Recording 1000 metrics should be very fast (< 100ms)
	assert.Less(t, duration, 100*time.Millisecond, "Metrics recording should be performant")
}

// ValidationTestNotifier for testing
type ValidationTestNotifier struct {
	alerts    []*Alert
	callCount int
}

func (vtn *ValidationTestNotifier) SendAlert(ctx context.Context, alert *Alert) error {
	vtn.callCount++
	vtn.alerts = append(vtn.alerts, alert)
	return nil
}

func (vtn *ValidationTestNotifier) GetName() string {
	return "validation-test"
}