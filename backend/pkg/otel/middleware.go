package otel

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// TracingMiddleware returns a Gin middleware that adds trace context to requests
func TracingMiddleware(logger *StructuredLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Extract or generate trace ID
		traceID := c.GetHeader("X-Trace-Id")
		if traceID == "" {
			traceID = c.GetHeader("traceparent")
		}
		if traceID == "" {
			traceID = generateTraceID()
		}

		spanID := generateSpanID()

		// Set trace context in request context
		tc := &TraceContext{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: 1,
		}
		ctx := WithTraceContext(c.Request.Context(), tc)
		c.Request = c.Request.WithContext(ctx)

		// Set response headers
		c.Header("X-Trace-Id", traceID)
		c.Header("X-Span-Id", spanID)

		c.Next()

		// Log the request with trace correlation
		duration := time.Since(start)
		statusCode := c.Writer.Status()

		attrs := map[string]interface{}{
			"http.method":      c.Request.Method,
			"http.url":         c.Request.URL.Path,
			"http.status_code": statusCode,
			"http.duration_ms": duration.Milliseconds(),
			"http.client_ip":   c.ClientIP(),
		}

		level := LogLevelInfo
		if statusCode >= 500 {
			level = LogLevelError
		} else if statusCode >= 400 {
			level = LogLevelWarn
		}

		logger.Log(ctx, level, "HTTP "+c.Request.Method+" "+c.Request.URL.Path+" "+strconv.Itoa(statusCode), attrs)
	}
}
