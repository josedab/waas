package tracing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// DeliveryStage represents stages in the webhook delivery pipeline
type DeliveryStage string

const (
	StageAPIReceive   DeliveryStage = "api.receive"
	StageValidation   DeliveryStage = "api.validate"
	StageEnqueue      DeliveryStage = "queue.enqueue"
	StageDequeue      DeliveryStage = "queue.dequeue"
	StageTransform    DeliveryStage = "transform.apply"
	StageDelivery     DeliveryStage = "delivery.attempt"
	StageRetry        DeliveryStage = "delivery.retry"
	StageResponse     DeliveryStage = "delivery.response"
	StageDLQ          DeliveryStage = "dlq.enqueue"
	StageComplete     DeliveryStage = "delivery.complete"
)

// DeliveryTrace captures the full trace of a webhook delivery lifecycle
type DeliveryTrace struct {
	TraceID       string              `json:"trace_id"`
	TenantID      string              `json:"tenant_id"`
	WebhookID     string              `json:"webhook_id"`
	EndpointID    string              `json:"endpoint_id"`
	EventType     string              `json:"event_type"`
	Stages        []StageSpan         `json:"stages"`
	TotalDuration int64               `json:"total_duration_ms"`
	Status        string              `json:"status"` // delivered, failed, retrying
	W3CContext    *W3CTraceContext     `json:"w3c_context"`
	CreatedAt     time.Time           `json:"created_at"`
}

// StageSpan represents a span for a specific delivery stage
type StageSpan struct {
	SpanID        string            `json:"span_id"`
	ParentSpanID  string            `json:"parent_span_id,omitempty"`
	Stage         DeliveryStage     `json:"stage"`
	ServiceName   string            `json:"service_name"`
	StartedAt     time.Time         `json:"started_at"`
	EndedAt       time.Time         `json:"ended_at"`
	DurationMs    int64             `json:"duration_ms"`
	Status        string            `json:"status"` // ok, error
	Attributes    map[string]string `json:"attributes,omitempty"`
	Events        []SpanEvent       `json:"events,omitempty"`
	ErrorMessage  string            `json:"error_message,omitempty"`
}

// W3CTraceContext implements the W3C Trace Context specification
type W3CTraceContext struct {
	Version    string `json:"version"`
	TraceID    string `json:"trace_id"`
	ParentID   string `json:"parent_id"`
	TraceFlags string `json:"trace_flags"`
	TraceState string `json:"tracestate,omitempty"`
}

// Traceparent returns the W3C traceparent header value
func (w *W3CTraceContext) Traceparent() string {
	return fmt.Sprintf("%s-%s-%s-%s", w.Version, w.TraceID, w.ParentID, w.TraceFlags)
}

// ParseTraceparent parses a W3C traceparent header
func ParseTraceparent(header string) (*W3CTraceContext, error) {
	if header == "" {
		return nil, fmt.Errorf("empty traceparent header")
	}

	var version, traceID, parentID, flags string
	_, err := fmt.Sscanf(header, "%2s-%32s-%16s-%2s", &version, &traceID, &parentID, &flags)
	if err != nil {
		return nil, fmt.Errorf("invalid traceparent format: %w", err)
	}

	return &W3CTraceContext{
		Version:    version,
		TraceID:    traceID,
		ParentID:   parentID,
		TraceFlags: flags,
	}, nil
}

// NewW3CTraceContext creates a new W3C trace context
func NewW3CTraceContext() (*W3CTraceContext, error) {
	traceIDBytes := make([]byte, 16)
	parentIDBytes := make([]byte, 8)
	if _, err := rand.Read(traceIDBytes); err != nil {
		return nil, fmt.Errorf("failed to generate trace ID: %w", err)
	}
	if _, err := rand.Read(parentIDBytes); err != nil {
		return nil, fmt.Errorf("failed to generate parent ID: %w", err)
	}

	return &W3CTraceContext{
		Version:    "00",
		TraceID:    hex.EncodeToString(traceIDBytes),
		ParentID:   hex.EncodeToString(parentIDBytes),
		TraceFlags: "01", // sampled
	}, nil
}

// DeliveryTracer manages delivery trace lifecycle
type DeliveryTracer struct {
	service *Service
}

// NewDeliveryTracer creates a new delivery tracer
func NewDeliveryTracer(service *Service) *DeliveryTracer {
	return &DeliveryTracer{service: service}
}

// StartDeliveryTrace begins a new delivery trace
func (dt *DeliveryTracer) StartDeliveryTrace(ctx context.Context, tenantID, webhookID, endpointID, eventType string) (*DeliveryTrace, error) {
	w3cCtx, err := NewW3CTraceContext()
	if err != nil {
		return nil, err
	}

	trace := &DeliveryTrace{
		TraceID:    w3cCtx.TraceID,
		TenantID:   tenantID,
		WebhookID:  webhookID,
		EndpointID: endpointID,
		EventType:  eventType,
		Status:     "active",
		W3CContext: w3cCtx,
		CreatedAt:  time.Now(),
	}

	return trace, nil
}

// RecordStage adds a stage span to a delivery trace
func (dt *DeliveryTracer) RecordStage(trace *DeliveryTrace, stage DeliveryStage, serviceName string, duration time.Duration, attrs map[string]string, errMsg string) *StageSpan {
	spanIDBytes := make([]byte, 8)
	rand.Read(spanIDBytes)

	parentSpanID := ""
	if len(trace.Stages) > 0 {
		parentSpanID = trace.Stages[len(trace.Stages)-1].SpanID
	}

	now := time.Now()
	status := "ok"
	if errMsg != "" {
		status = "error"
	}

	span := StageSpan{
		SpanID:       hex.EncodeToString(spanIDBytes),
		ParentSpanID: parentSpanID,
		Stage:        stage,
		ServiceName:  serviceName,
		StartedAt:    now.Add(-duration),
		EndedAt:      now,
		DurationMs:   duration.Milliseconds(),
		Status:       status,
		Attributes:   attrs,
		ErrorMessage: errMsg,
	}

	trace.Stages = append(trace.Stages, span)
	trace.TotalDuration += span.DurationMs

	return &span
}

// CompleteDeliveryTrace finalizes a delivery trace
func (dt *DeliveryTracer) CompleteDeliveryTrace(ctx context.Context, trace *DeliveryTrace, delivered bool) error {
	if delivered {
		trace.Status = "delivered"
	} else {
		trace.Status = "failed"
	}

	// Calculate total duration from first to last stage
	if len(trace.Stages) > 0 {
		first := trace.Stages[0]
		last := trace.Stages[len(trace.Stages)-1]
		trace.TotalDuration = last.EndedAt.Sub(first.StartedAt).Milliseconds()
	}

	// Record spans in the tracing service
	if dt.service != nil {
		for _, stage := range trace.Stages {
			if _, err := dt.service.RecordSpan(ctx, trace.TenantID, &CreateSpanRequest{
				TraceID:       trace.TraceID,
				SpanID:        stage.SpanID,
				ParentSpanID:  stage.ParentSpanID,
				OperationName: string(stage.Stage),
				ServiceName:   stage.ServiceName,
				SpanKind:      "INTERNAL",
				StatusCode:    stage.Status,
				StatusMessage: stage.ErrorMessage,
				Attributes:    stage.Attributes,
				DurationMs:    stage.DurationMs,
				StartedAt:     stage.StartedAt.Format(time.RFC3339Nano),
			}); err != nil {
				return fmt.Errorf("record span for stage %s: %w", stage.Stage, err)
			}
		}
	}

	return nil
}

// InjectTraceHeaders injects W3C trace context headers into outbound webhook headers
func InjectTraceHeaders(headers map[string]string, w3cCtx *W3CTraceContext) map[string]string {
	if headers == nil {
		headers = make(map[string]string)
	}
	if w3cCtx == nil {
		return headers
	}

	headers["traceparent"] = w3cCtx.Traceparent()
	if w3cCtx.TraceState != "" {
		headers["tracestate"] = w3cCtx.TraceState
	}

	return headers
}

// ExtractTraceHeaders extracts W3C trace context from incoming headers
func ExtractTraceHeaders(headers map[string]string) *W3CTraceContext {
	traceparent := headers["traceparent"]
	if traceparent == "" {
		traceparent = headers["Traceparent"]
	}
	if traceparent == "" {
		return nil
	}

	ctx, err := ParseTraceparent(traceparent)
	if err != nil {
		return nil
	}

	if ts, ok := headers["tracestate"]; ok {
		ctx.TraceState = ts
	}

	return ctx
}

// CorrelationContext builds cross-system correlation metadata
type CorrelationContext struct {
	TraceID     string            `json:"trace_id"`
	SpanID      string            `json:"span_id"`
	WebhookID   string            `json:"webhook_id"`
	EndpointID  string            `json:"endpoint_id"`
	EventType   string            `json:"event_type"`
	DeliveryID  string            `json:"delivery_id,omitempty"`
	TenantID    string            `json:"tenant_id"`
	Timestamp   time.Time         `json:"timestamp"`
	Baggage     map[string]string `json:"baggage,omitempty"`
}

// NewCorrelationContext creates a correlation context from a delivery trace
func NewCorrelationContext(trace *DeliveryTrace) *CorrelationContext {
	spanID := ""
	if trace.W3CContext != nil {
		spanID = trace.W3CContext.ParentID
	}

	return &CorrelationContext{
		TraceID:    trace.TraceID,
		SpanID:     spanID,
		WebhookID:  trace.WebhookID,
		EndpointID: trace.EndpointID,
		EventType:  trace.EventType,
		TenantID:   trace.TenantID,
		Timestamp:  time.Now(),
	}
}

// ToHeaders converts correlation context to HTTP headers
func (cc *CorrelationContext) ToHeaders() map[string]string {
	headers := map[string]string{
		"X-WaaS-Trace-ID":    cc.TraceID,
		"X-WaaS-Span-ID":     cc.SpanID,
		"X-WaaS-Webhook-ID":  cc.WebhookID,
		"X-WaaS-Endpoint-ID": cc.EndpointID,
		"X-WaaS-Event-Type":  cc.EventType,
	}

	if cc.DeliveryID != "" {
		headers["X-WaaS-Delivery-ID"] = cc.DeliveryID
	}

	// Add baggage
	for k, v := range cc.Baggage {
		headers["baggage"] += fmt.Sprintf("%s=%s,", k, v)
	}

	return headers
}

// WaterfallNode represents a node in the trace waterfall visualization
type WaterfallNode struct {
	SpanID       string          `json:"span_id"`
	Stage        string          `json:"stage"`
	Service      string          `json:"service"`
	StartOffset  int64           `json:"start_offset_ms"`
	DurationMs   int64           `json:"duration_ms"`
	Status       string          `json:"status"`
	ErrorMessage string          `json:"error_message,omitempty"`
	Depth        int             `json:"depth"`
	Children     []WaterfallNode `json:"children,omitempty"`
}

// BuildWaterfall creates a waterfall visualization from a delivery trace
func BuildWaterfall(trace *DeliveryTrace) []WaterfallNode {
	if len(trace.Stages) == 0 {
		return nil
	}

	baseTime := trace.Stages[0].StartedAt
	nodes := make([]WaterfallNode, 0, len(trace.Stages))

	spanDepth := make(map[string]int)
	for _, stage := range trace.Stages {
		depth := 0
		if stage.ParentSpanID != "" {
			if parentDepth, ok := spanDepth[stage.ParentSpanID]; ok {
				depth = parentDepth + 1
			}
		}
		spanDepth[stage.SpanID] = depth

		nodes = append(nodes, WaterfallNode{
			SpanID:       stage.SpanID,
			Stage:        string(stage.Stage),
			Service:      stage.ServiceName,
			StartOffset:  stage.StartedAt.Sub(baseTime).Milliseconds(),
			DurationMs:   stage.DurationMs,
			Status:       stage.Status,
			ErrorMessage: stage.ErrorMessage,
			Depth:        depth,
		})
	}

	return nodes
}
