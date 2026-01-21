package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"webhook-platform/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TraceContext represents tracing information for a request
type TraceContext struct {
	TraceID      string            `json:"trace_id"`
	SpanID       string            `json:"span_id"`
	ParentSpanID string            `json:"parent_span_id,omitempty"`
	Operation    string            `json:"operation"`
	Service      string            `json:"service"`
	StartTime    time.Time         `json:"start_time"`
	EndTime      *time.Time        `json:"end_time,omitempty"`
	Duration     *time.Duration    `json:"duration,omitempty"`
	Tags         map[string]string `json:"tags"`
	Logs         []TraceLog        `json:"logs"`
	Status       TraceStatus       `json:"status"`
	Error        *string           `json:"error,omitempty"`
}

// TraceLog represents a log entry within a trace span
type TraceLog struct {
	Timestamp time.Time         `json:"timestamp"`
	Level     string            `json:"level"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields"`
}

// TraceStatus represents the status of a trace span
type TraceStatus string

const (
	TraceStatusOK    TraceStatus = "ok"
	TraceStatusError TraceStatus = "error"
)

// Tracer provides distributed tracing functionality
type Tracer struct {
	serviceName string
	logger      *utils.Logger
	spans       map[string]*TraceContext
}

// NewTracer creates a new tracer instance
func NewTracer(serviceName string, logger *utils.Logger) *Tracer {
	return &Tracer{
		serviceName: serviceName,
		logger:      logger,
		spans:       make(map[string]*TraceContext),
	}
}

// TracingMiddleware creates a Gin middleware for distributed tracing
func (t *Tracer) TracingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract or generate trace ID
		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = uuid.New().String()
		}

		// Extract parent span ID
		parentSpanID := c.GetHeader("X-Parent-Span-ID")

		// Generate new span ID
		spanID := uuid.New().String()

		// Create trace context
		traceCtx := &TraceContext{
			TraceID:      traceID,
			SpanID:       spanID,
			ParentSpanID: parentSpanID,
			Operation:    fmt.Sprintf("%s %s", c.Request.Method, c.FullPath()),
			Service:      t.serviceName,
			StartTime:    time.Now(),
			Tags: map[string]string{
				"http.method":      c.Request.Method,
				"http.url":         c.Request.URL.String(),
				"http.user_agent":  c.Request.UserAgent(),
				"http.remote_addr": c.ClientIP(),
			},
			Logs:   make([]TraceLog, 0),
			Status: TraceStatusOK,
		}

		// Add tenant ID if available
		if tenantID, exists := c.Get("tenant_id"); exists {
			traceCtx.Tags["tenant_id"] = fmt.Sprintf("%v", tenantID)
		}

		// Store trace context in Gin context
		c.Set("trace_context", traceCtx)

		// Set response headers for trace propagation
		c.Header("X-Trace-ID", traceID)
		c.Header("X-Span-ID", spanID)

		// Process request
		c.Next()

		// Finalize trace
		t.finalizeTrace(c, traceCtx)
	}
}

// StartSpan starts a new trace span
func (t *Tracer) StartSpan(ctx context.Context, operation string, tags map[string]string) *TraceContext {
	// Try to get parent trace context
	var parentTraceID, parentSpanID string
	if parentCtx := t.getTraceContextFromContext(ctx); parentCtx != nil {
		parentTraceID = parentCtx.TraceID
		parentSpanID = parentCtx.SpanID
	} else {
		parentTraceID = uuid.New().String()
	}

	spanID := uuid.New().String()

	traceCtx := &TraceContext{
		TraceID:      parentTraceID,
		SpanID:       spanID,
		ParentSpanID: parentSpanID,
		Operation:    operation,
		Service:      t.serviceName,
		StartTime:    time.Now(),
		Tags:         tags,
		Logs:         make([]TraceLog, 0),
		Status:       TraceStatusOK,
	}

	if traceCtx.Tags == nil {
		traceCtx.Tags = make(map[string]string)
	}

	return traceCtx
}

// FinishSpan finishes a trace span
func (t *Tracer) FinishSpan(traceCtx *TraceContext, err error) {
	now := time.Now()
	traceCtx.EndTime = &now
	duration := now.Sub(traceCtx.StartTime)
	traceCtx.Duration = &duration

	if err != nil {
		traceCtx.Status = TraceStatusError
		errMsg := err.Error()
		traceCtx.Error = &errMsg
		traceCtx.Tags["error"] = "true"
	}

	// Log the completed span
	t.logger.InfoWithCorrelation("Trace span completed", traceCtx.TraceID, map[string]interface{}{
		"span_id":       traceCtx.SpanID,
		"parent_span_id": traceCtx.ParentSpanID,
		"operation":     traceCtx.Operation,
		"service":       traceCtx.Service,
		"duration_ms":   duration.Milliseconds(),
		"status":        traceCtx.Status,
		"tags":          traceCtx.Tags,
		"error":         traceCtx.Error,
	})
}

// AddLog adds a log entry to a trace span
func (t *Tracer) AddLog(traceCtx *TraceContext, level, message string, fields map[string]string) {
	log := TraceLog{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Fields:    fields,
	}
	traceCtx.Logs = append(traceCtx.Logs, log)
}

// AddTag adds a tag to a trace span
func (t *Tracer) AddTag(traceCtx *TraceContext, key, value string) {
	if traceCtx.Tags == nil {
		traceCtx.Tags = make(map[string]string)
	}
	traceCtx.Tags[key] = value
}

// InjectHeaders injects tracing headers into HTTP request
func (t *Tracer) InjectHeaders(traceCtx *TraceContext, req *http.Request) {
	req.Header.Set("X-Trace-ID", traceCtx.TraceID)
	req.Header.Set("X-Parent-Span-ID", traceCtx.SpanID)
}

// ExtractTraceContext extracts trace context from HTTP headers
func (t *Tracer) ExtractTraceContext(req *http.Request) *TraceContext {
	traceID := req.Header.Get("X-Trace-ID")
	parentSpanID := req.Header.Get("X-Parent-Span-ID")

	if traceID == "" {
		return nil
	}

	return &TraceContext{
		TraceID:      traceID,
		ParentSpanID: parentSpanID,
	}
}

// TraceWebhookDelivery creates a trace span for webhook delivery
func (t *Tracer) TraceWebhookDelivery(ctx context.Context, endpointID, tenantID string) *TraceContext {
	return t.StartSpan(ctx, "webhook.delivery", map[string]string{
		"endpoint_id": endpointID,
		"tenant_id":   tenantID,
		"component":   "delivery_engine",
	})
}

// TraceAPIRequest creates a trace span for API requests
func (t *Tracer) TraceAPIRequest(ctx context.Context, method, endpoint string) *TraceContext {
	return t.StartSpan(ctx, fmt.Sprintf("api.%s", method), map[string]string{
		"http.method":   method,
		"http.endpoint": endpoint,
		"component":     "api_service",
	})
}

// TraceDatabaseQuery creates a trace span for database queries
func (t *Tracer) TraceDatabaseQuery(ctx context.Context, operation, table string) *TraceContext {
	return t.StartSpan(ctx, fmt.Sprintf("db.%s", operation), map[string]string{
		"db.operation": operation,
		"db.table":     table,
		"component":    "database",
	})
}

// TraceQueueOperation creates a trace span for queue operations
func (t *Tracer) TraceQueueOperation(ctx context.Context, operation, queueName string) *TraceContext {
	return t.StartSpan(ctx, fmt.Sprintf("queue.%s", operation), map[string]string{
		"queue.operation": operation,
		"queue.name":      queueName,
		"component":       "message_queue",
	})
}

// GetTraceID extracts trace ID from context
func (t *Tracer) GetTraceID(ctx context.Context) string {
	if traceCtx := t.getTraceContextFromContext(ctx); traceCtx != nil {
		return traceCtx.TraceID
	}
	return ""
}

// finalizeTrace finalizes the trace for an HTTP request
func (t *Tracer) finalizeTrace(c *gin.Context, traceCtx *TraceContext) {
	now := time.Now()
	traceCtx.EndTime = &now
	duration := now.Sub(traceCtx.StartTime)
	traceCtx.Duration = &duration

	// Add response information
	traceCtx.Tags["http.status_code"] = fmt.Sprintf("%d", c.Writer.Status())
	traceCtx.Tags["http.response_size"] = fmt.Sprintf("%d", c.Writer.Size())

	// Check for errors
	if c.Writer.Status() >= 400 {
		traceCtx.Status = TraceStatusError
		traceCtx.Tags["error"] = "true"
		
		// Try to get error from context
		if errors := c.Errors; len(errors) > 0 {
			errMsg := errors.Last().Error()
			traceCtx.Error = &errMsg
		}
	}

	// Log the completed trace
	t.logger.InfoWithCorrelation("HTTP request trace completed", traceCtx.TraceID, map[string]interface{}{
		"span_id":         traceCtx.SpanID,
		"operation":       traceCtx.Operation,
		"duration_ms":     duration.Milliseconds(),
		"status_code":     c.Writer.Status(),
		"response_size":   c.Writer.Size(),
		"status":          traceCtx.Status,
		"error":           traceCtx.Error,
	})
}

// getTraceContextFromContext extracts trace context from Go context
func (t *Tracer) getTraceContextFromContext(ctx context.Context) *TraceContext {
	if ginCtx, ok := ctx.(*gin.Context); ok {
		if traceCtx, exists := ginCtx.Get("trace_context"); exists {
			if tc, ok := traceCtx.(*TraceContext); ok {
				return tc
			}
		}
	}
	return nil
}

// ContextWithTrace adds trace context to Go context
func (t *Tracer) ContextWithTrace(ctx context.Context, traceCtx *TraceContext) context.Context {
	if ginCtx, ok := ctx.(*gin.Context); ok {
		ginCtx.Set("trace_context", traceCtx)
		return ginCtx
	}
	// For non-Gin contexts, you might use context.WithValue
	return context.WithValue(ctx, "trace_context", traceCtx)
}