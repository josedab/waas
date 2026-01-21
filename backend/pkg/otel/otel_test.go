package otel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGenerateTraceID(t *testing.T) {
	traceID := generateTraceID()
	
	if len(traceID) != 32 {
		t.Errorf("expected trace ID length 32, got %d", len(traceID))
	}
	
	// Generate another one and make sure they're different
	traceID2 := generateTraceID()
	if traceID == traceID2 {
		t.Error("expected unique trace IDs")
	}
}

func TestGenerateSpanID(t *testing.T) {
	spanID := generateSpanID()
	
	if len(spanID) != 16 {
		t.Errorf("expected span ID length 16, got %d", len(spanID))
	}
	
	// Generate another one and make sure they're different
	spanID2 := generateSpanID()
	if spanID == spanID2 {
		t.Error("expected unique span IDs")
	}
}

func TestParseTraceparent(t *testing.T) {
	tests := []struct {
		name        string
		traceparent string
		wantTraceID string
		wantSpanID  string
		wantFlags   byte
		wantNil     bool
	}{
		{
			name:        "valid traceparent",
			traceparent: "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01",
			wantTraceID: "0af7651916cd43dd8448eb211c80319c",
			wantSpanID:  "b7ad6b7169203331",
			wantFlags:   1,
		},
		{
			name:        "sampled flag off",
			traceparent: "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-00",
			wantTraceID: "0af7651916cd43dd8448eb211c80319c",
			wantSpanID:  "b7ad6b7169203331",
			wantFlags:   0,
		},
		{
			name:        "invalid format",
			traceparent: "invalid",
			wantNil:     true,
		},
		{
			name:        "too few parts",
			traceparent: "00-0af7651916cd43dd8448eb211c80319c",
			wantNil:     true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := parseTraceparent(tt.traceparent)
			
			if tt.wantNil {
				if tc != nil {
					t.Error("expected nil TraceContext")
				}
				return
			}
			
			if tc == nil {
				t.Fatal("expected non-nil TraceContext")
			}
			
			if tc.TraceID != tt.wantTraceID {
				t.Errorf("expected TraceID %s, got %s", tt.wantTraceID, tc.TraceID)
			}
			if tc.SpanID != tt.wantSpanID {
				t.Errorf("expected SpanID %s, got %s", tt.wantSpanID, tc.SpanID)
			}
			if tc.TraceFlags != tt.wantFlags {
				t.Errorf("expected TraceFlags %d, got %d", tt.wantFlags, tc.TraceFlags)
			}
		})
	}
}

func TestTraceContext(t *testing.T) {
	ctx := context.Background()
	
	// Initially no trace context
	tc := GetTraceContext(ctx)
	if tc != nil {
		t.Error("expected nil trace context initially")
	}
	
	// Add trace context
	newTC := &TraceContext{
		TraceID:    "test-trace-id",
		SpanID:     "test-span-id",
		TraceFlags: 1,
	}
	ctx = WithTraceContext(ctx, newTC)
	
	// Retrieve trace context
	tc = GetTraceContext(ctx)
	if tc == nil {
		t.Fatal("expected non-nil trace context")
	}
	if tc.TraceID != "test-trace-id" {
		t.Errorf("expected TraceID 'test-trace-id', got %s", tc.TraceID)
	}
}

func TestTracer(t *testing.T) {
	config := &Config{
		TenantID:    "test-tenant",
		ServiceName: "test-service",
		Enabled:     true,
		Traces: TracesConfig{
			Enabled:      true,
			SamplingRate: 1.0,
		},
		Metrics: MetricsConfig{
			Enabled:  true,
			Interval: 1,
		},
	}
	
	tracer := NewTracer(config)
	
	if tracer == nil {
		t.Fatal("expected non-nil tracer")
	}
	
	ctx := context.Background()
	
	// Start a span
	ctx, span := tracer.StartSpan(ctx, "test-operation")
	if span == nil {
		t.Fatal("expected non-nil span")
	}
	
	// Check trace context was created
	tc := GetTraceContext(ctx)
	if tc == nil {
		t.Fatal("expected trace context in context")
	}
	
	// Set attributes
	span.SetAttribute("test.key", "test-value")
	span.SetAttribute("test.number", 42)
	
	// Add an event
	span.AddEvent("test-event", map[string]any{"detail": "some detail"})
	
	// End span
	span.End()
}

func TestTracerWithParentSpan(t *testing.T) {
	config := DefaultConfig("test-tenant")
	tracer := NewTracer(config)
	ctx := context.Background()
	
	// Start parent span
	ctx, parentSpan := tracer.StartSpan(ctx, "parent-operation")
	parentTraceID := parentSpan.traceID
	parentSpanID := parentSpan.spanID
	
	// Start child span
	_, childSpan := tracer.StartSpan(ctx, "child-operation")
	
	// Child should have same trace ID but different span ID
	if childSpan.traceID != parentTraceID {
		t.Errorf("expected child trace ID %s, got %s", parentTraceID, childSpan.traceID)
	}
	if childSpan.parentSpanID != parentSpanID {
		t.Errorf("expected parent span ID %s, got %s", parentSpanID, childSpan.parentSpanID)
	}
	if childSpan.spanID == parentSpanID {
		t.Error("expected different span ID from parent")
	}
	
	childSpan.End()
	parentSpan.End()
}

func TestSpanStatus(t *testing.T) {
	config := DefaultConfig("test-tenant")
	tracer := NewTracer(config)
	ctx := context.Background()
	
	// Test OK status
	_, span := tracer.StartSpan(ctx, "test-operation")
	span.SetStatus(0, "OK")
	
	if span.status.Code != 0 {
		t.Errorf("expected status code 0, got %d", span.status.Code)
	}
	if span.status.Description != "OK" {
		t.Errorf("expected status description 'OK', got %s", span.status.Description)
	}
	span.End()
	
	// Test error
	_, errSpan := tracer.StartSpan(ctx, "error-operation")
	errSpan.SetError(context.DeadlineExceeded)
	
	if errSpan.status.Code != 2 {
		t.Errorf("expected status code 2 for error, got %d", errSpan.status.Code)
	}
	errSpan.End()
}

func TestMetrics(t *testing.T) {
	config := DefaultConfig("test-tenant")
	tracer := NewTracer(config)
	
	// Record various metrics
	tracer.IncrementCounter("test.counter", map[string]string{"env": "test"})
	tracer.RecordGauge("test.gauge", 42.5, map[string]string{"env": "test"})
	tracer.RecordHistogram("test.histogram", 100.0, map[string]string{"env": "test"})
}

func TestStdoutExporter(t *testing.T) {
	exporter := NewStdoutExporter()
	ctx := context.Background()
	
	// Test span export
	spans := []*SpanData{
		{
			TraceID:       "test-trace",
			SpanID:        "test-span",
			OperationName: "test-op",
			ServiceName:   "test-service",
			StartTime:     time.Now().Add(-time.Second),
			EndTime:       time.Now(),
			Duration:      time.Second,
			Status:        SpanStatus{Code: 0, Description: "OK"},
			Attributes:    map[string]any{"key": "value"},
		},
	}
	
	err := exporter.ExportSpans(ctx, spans)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	
	// Test metrics export
	metrics := []*MetricData{
		{
			Name:       "test.metric",
			Type:       MetricCounter,
			Value:      1,
			Timestamp:  time.Now(),
			Attributes: map[string]string{"env": "test"},
		},
	}
	
	err = exporter.ExportMetrics(ctx, metrics)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	
	err = exporter.Shutdown(ctx)
	if err != nil {
		t.Errorf("unexpected error during shutdown: %v", err)
	}
}

func TestPrometheusExporter(t *testing.T) {
	exporter := NewPrometheusExporter()
	ctx := context.Background()
	
	// Export some metrics
	metrics := []*MetricData{
		{
			Name:        "http_requests_total",
			Description: "Total HTTP requests",
			Type:        MetricCounter,
			Value:       100,
			Timestamp:   time.Now(),
			Attributes:  map[string]string{"method": "GET", "status": "200"},
		},
		{
			Name:        "http_request_duration_seconds",
			Description: "HTTP request duration",
			Type:        MetricHistogram,
			Value:       0.5,
			Timestamp:   time.Now(),
			Attributes:  map[string]string{"method": "POST"},
		},
	}
	
	err := exporter.ExportMetrics(ctx, metrics)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	
	// Test HTTP handler
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()
	
	exporter.ServeHTTP(rr, req)
	
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
	
	// Check content type
	contentType := rr.Header().Get("Content-Type")
	if contentType != "text/plain; version=0.0.4" {
		t.Errorf("expected Prometheus content type, got %s", contentType)
	}
	
	// Body should contain our metric
	body := rr.Body.String()
	if body == "" {
		t.Error("expected non-empty response body")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig("test-tenant")
	
	if config.TenantID != "test-tenant" {
		t.Errorf("expected tenant ID 'test-tenant', got %s", config.TenantID)
	}
	if config.ServiceName != "waas-webhook-service" {
		t.Errorf("expected service name 'waas-webhook-service', got %s", config.ServiceName)
	}
	if config.Traces.SamplingRate != 1.0 {
		t.Errorf("expected sampling rate 1.0, got %f", config.Traces.SamplingRate)
	}
}

func TestSupportedProtocols(t *testing.T) {
	exporters := []ExporterType{
		ExporterOTLP,
		ExporterOTLPHTTP,
		ExporterPrometheus,
		ExporterJaeger,
		ExporterZipkin,
		ExporterStdout,
	}
	
	for _, exp := range exporters {
		if exp == "" {
			t.Error("exporter type should not be empty")
		}
	}
}

func TestStandardSpanNames(t *testing.T) {
	names := StandardSpanNames
	
	if names.DeliveryProcess == "" {
		t.Error("DeliveryProcess span name should not be empty")
	}
	if names.DeliveryAttempt == "" {
		t.Error("DeliveryAttempt span name should not be empty")
	}
	if names.PayloadTransform == "" {
		t.Error("PayloadTransform span name should not be empty")
	}
}

func TestStandardMetricNames(t *testing.T) {
	names := StandardMetricNames
	
	if names.DeliveriesTotal == "" {
		t.Error("DeliveriesTotal metric name should not be empty")
	}
	if names.DeliveryDuration == "" {
		t.Error("DeliveryDuration metric name should not be empty")
	}
}
