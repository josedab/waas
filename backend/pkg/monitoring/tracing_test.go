package monitoring

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"webhook-platform/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestTracer_TracingMiddleware(t *testing.T) {
	logger := utils.NewLogger("test")
	tracer := NewTracer("test-service", logger)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(tracer.TracingMiddleware())
	router.GET("/test", func(c *gin.Context) {
		// Check that trace context is set
		traceCtx, exists := c.Get("trace_context")
		assert.True(t, exists)
		assert.NotNil(t, traceCtx)

		tc := traceCtx.(*TraceContext)
		assert.NotEmpty(t, tc.TraceID)
		assert.NotEmpty(t, tc.SpanID)
		assert.Equal(t, "GET /test", tc.Operation)
		assert.Equal(t, "test-service", tc.Service)

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Test without existing trace headers
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-Trace-ID"))
	assert.NotEmpty(t, w.Header().Get("X-Span-ID"))
}

func TestTracer_TracingMiddleware_WithExistingTrace(t *testing.T) {
	logger := utils.NewLogger("test")
	tracer := NewTracer("test-service", logger)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(tracer.TracingMiddleware())
	router.GET("/test", func(c *gin.Context) {
		traceCtx, exists := c.Get("trace_context")
		assert.True(t, exists)

		tc := traceCtx.(*TraceContext)
		assert.Equal(t, "existing-trace-id", tc.TraceID)
		assert.Equal(t, "existing-parent-span", tc.ParentSpanID)
		assert.NotEmpty(t, tc.SpanID)

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Test with existing trace headers
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Trace-ID", "existing-trace-id")
	req.Header.Set("X-Parent-Span-ID", "existing-parent-span")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "existing-trace-id", w.Header().Get("X-Trace-ID"))
}

func TestTracer_StartSpan(t *testing.T) {
	logger := utils.NewLogger("test")
	tracer := NewTracer("test-service", logger)

	ctx := context.Background()
	tags := map[string]string{
		"component": "test",
		"operation": "test-op",
	}

	span := tracer.StartSpan(ctx, "test.operation", tags)

	assert.NotEmpty(t, span.TraceID)
	assert.NotEmpty(t, span.SpanID)
	assert.Equal(t, "test.operation", span.Operation)
	assert.Equal(t, "test-service", span.Service)
	assert.Equal(t, tags, span.Tags)
	assert.Equal(t, TraceStatusOK, span.Status)
	assert.False(t, span.StartTime.IsZero())
	assert.Nil(t, span.EndTime)
	assert.Empty(t, span.Logs)
}

func TestTracer_FinishSpan(t *testing.T) {
	logger := utils.NewLogger("test")
	tracer := NewTracer("test-service", logger)

	span := &TraceContext{
		TraceID:   "test-trace",
		SpanID:    "test-span",
		Operation: "test.operation",
		Service:   "test-service",
		StartTime: time.Now().Add(-100 * time.Millisecond),
		Tags:      make(map[string]string),
		Status:    TraceStatusOK,
	}

	// Test successful completion
	tracer.FinishSpan(span, nil)

	assert.NotNil(t, span.EndTime)
	assert.NotNil(t, span.Duration)
	assert.Equal(t, TraceStatusOK, span.Status)
	assert.Nil(t, span.Error)
	assert.Greater(t, span.Duration.Milliseconds(), int64(0))

	// Test with error
	span2 := &TraceContext{
		TraceID:   "test-trace-2",
		SpanID:    "test-span-2",
		Operation: "test.operation",
		Service:   "test-service",
		StartTime: time.Now().Add(-50 * time.Millisecond),
		Tags:      make(map[string]string),
		Status:    TraceStatusOK,
	}

	testError := assert.AnError
	tracer.FinishSpan(span2, testError)

	assert.NotNil(t, span2.EndTime)
	assert.NotNil(t, span2.Duration)
	assert.Equal(t, TraceStatusError, span2.Status)
	assert.NotNil(t, span2.Error)
	assert.Equal(t, testError.Error(), *span2.Error)
	assert.Equal(t, "true", span2.Tags["error"])
}

func TestTracer_AddLog(t *testing.T) {
	logger := utils.NewLogger("test")
	tracer := NewTracer("test-service", logger)

	span := &TraceContext{
		TraceID: "test-trace",
		SpanID:  "test-span",
		Logs:    make([]TraceLog, 0),
	}

	fields := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	tracer.AddLog(span, "info", "test message", fields)

	assert.Len(t, span.Logs, 1)
	log := span.Logs[0]
	assert.Equal(t, "info", log.Level)
	assert.Equal(t, "test message", log.Message)
	assert.Equal(t, fields, log.Fields)
	assert.False(t, log.Timestamp.IsZero())
}

func TestTracer_AddTag(t *testing.T) {
	logger := utils.NewLogger("test")
	tracer := NewTracer("test-service", logger)

	span := &TraceContext{
		TraceID: "test-trace",
		SpanID:  "test-span",
		Tags:    nil,
	}

	tracer.AddTag(span, "key1", "value1")
	assert.NotNil(t, span.Tags)
	assert.Equal(t, "value1", span.Tags["key1"])

	tracer.AddTag(span, "key2", "value2")
	assert.Equal(t, "value2", span.Tags["key2"])
	assert.Len(t, span.Tags, 2)
}

func TestTracer_InjectHeaders(t *testing.T) {
	logger := utils.NewLogger("test")
	tracer := NewTracer("test-service", logger)

	span := &TraceContext{
		TraceID: "test-trace-id",
		SpanID:  "test-span-id",
	}

	req := httptest.NewRequest("GET", "/test", nil)
	tracer.InjectHeaders(span, req)

	assert.Equal(t, "test-trace-id", req.Header.Get("X-Trace-ID"))
	assert.Equal(t, "test-span-id", req.Header.Get("X-Parent-Span-ID"))
}

func TestTracer_ExtractTraceContext(t *testing.T) {
	logger := utils.NewLogger("test")
	tracer := NewTracer("test-service", logger)

	// Test with headers
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Trace-ID", "extracted-trace-id")
	req.Header.Set("X-Parent-Span-ID", "extracted-parent-span")

	traceCtx := tracer.ExtractTraceContext(req)
	assert.NotNil(t, traceCtx)
	assert.Equal(t, "extracted-trace-id", traceCtx.TraceID)
	assert.Equal(t, "extracted-parent-span", traceCtx.ParentSpanID)

	// Test without headers
	req2 := httptest.NewRequest("GET", "/test", nil)
	traceCtx2 := tracer.ExtractTraceContext(req2)
	assert.Nil(t, traceCtx2)
}

func TestTracer_SpecializedSpanCreators(t *testing.T) {
	logger := utils.NewLogger("test")
	tracer := NewTracer("test-service", logger)
	ctx := context.Background()

	// Test webhook delivery span
	webhookSpan := tracer.TraceWebhookDelivery(ctx, "endpoint-123", "tenant-456")
	assert.Equal(t, "webhook.delivery", webhookSpan.Operation)
	assert.Equal(t, "endpoint-123", webhookSpan.Tags["endpoint_id"])
	assert.Equal(t, "tenant-456", webhookSpan.Tags["tenant_id"])
	assert.Equal(t, "delivery_engine", webhookSpan.Tags["component"])

	// Test API request span
	apiSpan := tracer.TraceAPIRequest(ctx, "POST", "/webhooks/send")
	assert.Equal(t, "api.POST", apiSpan.Operation)
	assert.Equal(t, "POST", apiSpan.Tags["http.method"])
	assert.Equal(t, "/webhooks/send", apiSpan.Tags["http.endpoint"])
	assert.Equal(t, "api_service", apiSpan.Tags["component"])

	// Test database query span
	dbSpan := tracer.TraceDatabaseQuery(ctx, "SELECT", "webhooks")
	assert.Equal(t, "db.SELECT", dbSpan.Operation)
	assert.Equal(t, "SELECT", dbSpan.Tags["db.operation"])
	assert.Equal(t, "webhooks", dbSpan.Tags["db.table"])
	assert.Equal(t, "database", dbSpan.Tags["component"])

	// Test queue operation span
	queueSpan := tracer.TraceQueueOperation(ctx, "publish", "webhook-delivery")
	assert.Equal(t, "queue.publish", queueSpan.Operation)
	assert.Equal(t, "publish", queueSpan.Tags["queue.operation"])
	assert.Equal(t, "webhook-delivery", queueSpan.Tags["queue.name"])
	assert.Equal(t, "message_queue", queueSpan.Tags["component"])
}