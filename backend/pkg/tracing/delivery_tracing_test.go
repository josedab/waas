package tracing

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewW3CTraceContext(t *testing.T) {
	ctx, err := NewW3CTraceContext()
	require.NoError(t, err)
	assert.Equal(t, "00", ctx.Version)
	assert.Len(t, ctx.TraceID, 32)
	assert.Len(t, ctx.ParentID, 16)
	assert.Equal(t, "01", ctx.TraceFlags)
}

func TestW3CTraceContext_Traceparent(t *testing.T) {
	ctx := &W3CTraceContext{
		Version:    "00",
		TraceID:    "0af7651916cd43dd8448eb211c80319c",
		ParentID:   "b7ad6b7169203331",
		TraceFlags: "01",
	}
	assert.Equal(t, "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01", ctx.Traceparent())
}

func TestParseTraceparent(t *testing.T) {
	ctx, err := ParseTraceparent("00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01")
	require.NoError(t, err)
	assert.Equal(t, "00", ctx.Version)
	assert.Equal(t, "0af7651916cd43dd8448eb211c80319c", ctx.TraceID)
	assert.Equal(t, "b7ad6b7169203331", ctx.ParentID)
	assert.Equal(t, "01", ctx.TraceFlags)
}

func TestParseTraceparent_Empty(t *testing.T) {
	_, err := ParseTraceparent("")
	assert.Error(t, err)
}

func TestInjectTraceHeaders(t *testing.T) {
	w3cCtx := &W3CTraceContext{
		Version:    "00",
		TraceID:    "abcdef1234567890abcdef1234567890",
		ParentID:   "1234567890abcdef",
		TraceFlags: "01",
		TraceState: "vendor1=value1",
	}

	headers := InjectTraceHeaders(nil, w3cCtx)
	assert.Contains(t, headers["traceparent"], "abcdef1234567890abcdef1234567890")
	assert.Equal(t, "vendor1=value1", headers["tracestate"])
}

func TestInjectTraceHeaders_NilContext(t *testing.T) {
	headers := InjectTraceHeaders(map[string]string{"existing": "header"}, nil)
	assert.Equal(t, "header", headers["existing"])
	assert.Empty(t, headers["traceparent"])
}

func TestExtractTraceHeaders(t *testing.T) {
	headers := map[string]string{
		"traceparent": "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01",
		"tracestate":  "vendor1=value1",
	}

	ctx := ExtractTraceHeaders(headers)
	require.NotNil(t, ctx)
	assert.Equal(t, "0af7651916cd43dd8448eb211c80319c", ctx.TraceID)
	assert.Equal(t, "vendor1=value1", ctx.TraceState)
}

func TestExtractTraceHeaders_Missing(t *testing.T) {
	ctx := ExtractTraceHeaders(map[string]string{})
	assert.Nil(t, ctx)
}

func TestDeliveryTracer_StartAndComplete(t *testing.T) {
	tracer := NewDeliveryTracer(nil)
	ctx := context.Background()

	trace, err := tracer.StartDeliveryTrace(ctx, "tenant-1", "wh-1", "ep-1", "order.created")
	require.NoError(t, err)
	assert.NotEmpty(t, trace.TraceID)
	assert.NotNil(t, trace.W3CContext)
	assert.Equal(t, "active", trace.Status)

	// Record stages
	tracer.RecordStage(trace, StageAPIReceive, "api-service", 5*time.Millisecond, map[string]string{"http.method": "POST"}, "")
	tracer.RecordStage(trace, StageValidation, "api-service", 2*time.Millisecond, nil, "")
	tracer.RecordStage(trace, StageEnqueue, "api-service", 3*time.Millisecond, map[string]string{"queue": "deliveries"}, "")
	tracer.RecordStage(trace, StageDequeue, "delivery-engine", 1*time.Millisecond, nil, "")
	tracer.RecordStage(trace, StageDelivery, "delivery-engine", 120*time.Millisecond, map[string]string{"http.status": "200"}, "")
	tracer.RecordStage(trace, StageResponse, "delivery-engine", 1*time.Millisecond, nil, "")

	assert.Equal(t, 6, len(trace.Stages))
	assert.True(t, trace.TotalDuration > 0)

	// Complete trace
	err = tracer.CompleteDeliveryTrace(ctx, trace, true)
	require.NoError(t, err)
	assert.Equal(t, "delivered", trace.Status)
}

func TestDeliveryTracer_FailedDelivery(t *testing.T) {
	tracer := NewDeliveryTracer(nil)
	ctx := context.Background()

	trace, _ := tracer.StartDeliveryTrace(ctx, "tenant-1", "wh-2", "ep-2", "payment.failed")

	tracer.RecordStage(trace, StageAPIReceive, "api-service", 5*time.Millisecond, nil, "")
	tracer.RecordStage(trace, StageEnqueue, "api-service", 3*time.Millisecond, nil, "")
	tracer.RecordStage(trace, StageDelivery, "delivery-engine", 5000*time.Millisecond, nil, "connection timeout")
	tracer.RecordStage(trace, StageDLQ, "delivery-engine", 2*time.Millisecond, nil, "")

	_ = tracer.CompleteDeliveryTrace(ctx, trace, false)
	assert.Equal(t, "failed", trace.Status)

	// Check error span
	deliveryStage := trace.Stages[2]
	assert.Equal(t, "error", deliveryStage.Status)
	assert.Equal(t, "connection timeout", deliveryStage.ErrorMessage)
}

func TestStageSpan_ParentChaining(t *testing.T) {
	tracer := NewDeliveryTracer(nil)
	ctx := context.Background()

	trace, _ := tracer.StartDeliveryTrace(ctx, "t1", "w1", "e1", "test")

	s1 := tracer.RecordStage(trace, StageAPIReceive, "api", 1*time.Millisecond, nil, "")
	s2 := tracer.RecordStage(trace, StageEnqueue, "api", 1*time.Millisecond, nil, "")

	assert.Empty(t, s1.ParentSpanID) // First span has no parent
	assert.Equal(t, s1.SpanID, s2.ParentSpanID) // Second span chains to first
}

func TestCorrelationContext(t *testing.T) {
	trace := &DeliveryTrace{
		TraceID:    "abc123",
		TenantID:   "tenant-1",
		WebhookID:  "wh-1",
		EndpointID: "ep-1",
		EventType:  "order.created",
		W3CContext: &W3CTraceContext{
			ParentID: "span-456",
		},
	}

	cc := NewCorrelationContext(trace)
	assert.Equal(t, "abc123", cc.TraceID)
	assert.Equal(t, "span-456", cc.SpanID)

	headers := cc.ToHeaders()
	assert.Equal(t, "abc123", headers["X-WaaS-Trace-ID"])
	assert.Equal(t, "wh-1", headers["X-WaaS-Webhook-ID"])
}

func TestBuildWaterfall(t *testing.T) {
	tracer := NewDeliveryTracer(nil)
	ctx := context.Background()

	trace, _ := tracer.StartDeliveryTrace(ctx, "t1", "w1", "e1", "test")
	tracer.RecordStage(trace, StageAPIReceive, "api", 5*time.Millisecond, nil, "")
	tracer.RecordStage(trace, StageEnqueue, "api", 3*time.Millisecond, nil, "")
	tracer.RecordStage(trace, StageDelivery, "delivery", 100*time.Millisecond, nil, "")

	nodes := BuildWaterfall(trace)
	assert.Equal(t, 3, len(nodes))
	assert.Equal(t, "api.receive", nodes[0].Stage)
	assert.Equal(t, 0, nodes[0].Depth)
	// Verify waterfall nodes have proper structure
	assert.True(t, nodes[0].DurationMs >= 0)
	assert.True(t, nodes[2].DurationMs >= 0)
}

func TestBuildWaterfall_Empty(t *testing.T) {
	trace := &DeliveryTrace{}
	nodes := BuildWaterfall(trace)
	assert.Nil(t, nodes)
}
