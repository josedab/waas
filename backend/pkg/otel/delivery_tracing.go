package otel

import (
	"context"
	"fmt"
	"time"
)

// DeliveryTracing provides trace instrumentation for the webhook delivery engine.
// Each delivery operation creates a parent span with child spans for queue,
// HTTP request, and retry operations.

// DeliverySpanName constants for consistent span naming
const (
	SpanHandleDelivery   = "waas.delivery.handle"
	SpanPerformDelivery  = "waas.delivery.perform"
	SpanQueuePublish     = "waas.queue.publish"
	SpanQueueConsume     = "waas.queue.consume"
	SpanQueueDelay       = "waas.queue.delay"
	SpanQueueDLQ         = "waas.queue.dead_letter"
	SpanHTTPRequest      = "waas.delivery.http_request"
	SpanRetrySchedule    = "waas.delivery.retry_schedule"
	SpanSignatureCompute = "waas.delivery.signature"
	SpanTransformApply   = "waas.delivery.transform"
	SpanFilterEvaluate   = "waas.delivery.filter"
	SpanHealthCheck      = "waas.endpoint.health_check"
)

// DeliverySpanAttrs contains standard attributes for delivery trace spans
type DeliverySpanAttrs struct {
	TenantID      string `json:"tenant_id"`
	EndpointID    string `json:"endpoint_id"`
	DeliveryID    string `json:"delivery_id"`
	AttemptNumber int    `json:"attempt_number"`
	EventType     string `json:"event_type"`
	TargetURL     string `json:"target_url"`
	HTTPMethod    string `json:"http_method"`
	HTTPStatus    int    `json:"http_status,omitempty"`
	PayloadSize   int    `json:"payload_size_bytes"`
	QueueName     string `json:"queue_name,omitempty"`
	ErrorType     string `json:"error_type,omitempty"`
}

// StartDeliverySpan creates a parent span for a complete delivery operation
func (t *Tracer) StartDeliverySpan(ctx context.Context, attrs *DeliverySpanAttrs) (context.Context, *SpanData) {
	span := &SpanData{
		TraceID:       generateTraceID(),
		SpanID:        generateSpanID(),
		OperationName: SpanHandleDelivery,
		ServiceName:   t.serviceName,
		StartTime:     time.Now(),
		Attributes: map[string]any{
			"tenant.id":      attrs.TenantID,
			"endpoint.id":    attrs.EndpointID,
			"delivery.id":    attrs.DeliveryID,
			"event.type":     attrs.EventType,
			"attempt.number": attrs.AttemptNumber,
		},
		Status: SpanStatus{Code: 0, Description: "OK"},
	}

	ctx = context.WithValue(ctx, traceContextKey{}, span)
	return ctx, span
}

// StartChildSpan creates a child span linked to the parent delivery span
func (t *Tracer) StartChildSpan(ctx context.Context, name string, attrs map[string]string) (context.Context, *SpanData) {
	parent, _ := ctx.Value(traceContextKey{}).(*SpanData)

	span := &SpanData{
		SpanID:        generateSpanID(),
		OperationName: name,
		ServiceName:   t.serviceName,
		StartTime:     time.Now(),
		Attributes:    toAnyMap(attrs),
		Status:        SpanStatus{Code: 0, Description: "OK"},
	}

	if parent != nil {
		span.TraceID = parent.TraceID
		span.ParentSpanID = parent.SpanID
	} else {
		span.TraceID = generateTraceID()
	}

	ctx = context.WithValue(ctx, traceContextKey{}, span)
	return ctx, span
}

// EndSpan completes a span with optional error
func (t *Tracer) EndSpan(span *SpanData, err error) {
	if span == nil {
		return
	}
	span.EndTime = time.Now()
	span.Duration = span.EndTime.Sub(span.StartTime)

	if err != nil {
		span.Status = SpanStatus{Code: 2, Description: "ERROR"}
		span.Attributes["error"] = err.Error()
		span.Attributes["error.type"] = fmt.Sprintf("%T", err)
	}

	// Send to export pipeline
	select {
	case t.spans <- span:
	default:
		// Channel full, drop span (metric counter would go here)
	}
}

// RecordDeliveryMetric records a delivery-related metric
func (t *Tracer) RecordDeliveryMetric(name string, value float64, attrs map[string]string) {
	metric := &MetricData{
		Name:       name,
		Value:      value,
		Type:       MetricGauge,
		Timestamp:  time.Now(),
		Attributes: attrs,
	}

	select {
	case t.metrics <- metric:
	default:
	}
}

// --- Queue Context Propagation ---

// QueueTraceContext carries trace context through the Redis queue
type QueueTraceContext struct {
	TraceID      string            `json:"trace_id"`
	SpanID       string            `json:"span_id"`
	ParentSpanID string            `json:"parent_span_id,omitempty"`
	Baggage      map[string]string `json:"baggage,omitempty"`
}

// InjectTraceContext extracts trace context from context and serializes for queue
func InjectTraceContext(ctx context.Context) *QueueTraceContext {
	span, ok := ctx.Value(traceContextKey{}).(*SpanData)
	if !ok || span == nil {
		return nil
	}

	return &QueueTraceContext{
		TraceID:      span.TraceID,
		SpanID:       span.SpanID,
		ParentSpanID: span.ParentSpanID,
	}
}

// ExtractTraceContext restores trace context from queue message into context
func ExtractTraceContext(ctx context.Context, tc *QueueTraceContext) context.Context {
	if tc == nil {
		return ctx
	}

	span := &SpanData{
		TraceID:      tc.TraceID,
		SpanID:       tc.SpanID,
		ParentSpanID: tc.ParentSpanID,
	}

	return context.WithValue(ctx, traceContextKey{}, span)
}

type traceContextKey struct{}

// --- Structured Logging with Trace Correlation ---

// LogEntry represents a structured log entry with trace correlation
type LogEntry struct {
	Timestamp  time.Time         `json:"timestamp"`
	Level      string            `json:"level"`
	Message    string            `json:"message"`
	TraceID    string            `json:"trace_id,omitempty"`
	SpanID     string            `json:"span_id,omitempty"`
	Service    string            `json:"service"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// NewLogEntry creates a trace-correlated log entry from context
func NewLogEntry(ctx context.Context, level, message string) *LogEntry {
	entry := &LogEntry{
		Timestamp:  time.Now(),
		Level:      level,
		Message:    message,
		Service:    "waas",
		Attributes: make(map[string]string),
	}

	if span, ok := ctx.Value(traceContextKey{}).(*SpanData); ok && span != nil {
		entry.TraceID = span.TraceID
		entry.SpanID = span.SpanID
	}

	return entry
}

// WithAttr adds an attribute to the log entry
func (e *LogEntry) WithAttr(key, value string) *LogEntry {
	e.Attributes[key] = value
	return e
}

// toAnyMap converts map[string]string to map[string]any
func toAnyMap(m map[string]string) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}
