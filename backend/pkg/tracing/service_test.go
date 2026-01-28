package tracing

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// --- In-memory repository for testing ---

type memoryRepository struct {
	traces      map[string]*Trace // keyed by traceID
	spans       map[string]*Span  // keyed by span ID
	configs     map[string]*PropagationConfig
	stats       map[string]*TraceStats
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{
		traces:  make(map[string]*Trace),
		spans:   make(map[string]*Span),
		configs: make(map[string]*PropagationConfig),
		stats:   make(map[string]*TraceStats),
	}
}

func (r *memoryRepository) CreateTrace(_ context.Context, t *Trace) error {
	r.traces[t.TraceID] = t
	return nil
}
func (r *memoryRepository) GetTrace(_ context.Context, tenantID, traceID string) (*Trace, error) {
	t, ok := r.traces[traceID]
	if !ok || t.TenantID != tenantID {
		return nil, fmt.Errorf("trace not found")
	}
	return t, nil
}
func (r *memoryRepository) ListTraces(_ context.Context, tenantID string, filter TraceSearchRequest) ([]Trace, int, error) {
	var out []Trace
	for _, t := range r.traces {
		if t.TenantID != tenantID {
			continue
		}
		if filter.ServiceName != "" && t.ServiceName != filter.ServiceName {
			continue
		}
		if filter.Status != "" && t.Status != filter.Status {
			continue
		}
		out = append(out, *t)
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, len(out), nil
}
func (r *memoryRepository) UpdateTrace(_ context.Context, t *Trace) error {
	r.traces[t.TraceID] = t
	return nil
}
func (r *memoryRepository) DeleteTrace(_ context.Context, tenantID, traceID string) error {
	delete(r.traces, traceID)
	return nil
}
func (r *memoryRepository) CreateSpan(_ context.Context, s *Span) error {
	r.spans[s.ID] = s
	return nil
}
func (r *memoryRepository) GetSpan(_ context.Context, tenantID, spanID string) (*Span, error) {
	s, ok := r.spans[spanID]
	if !ok || s.TenantID != tenantID {
		return nil, fmt.Errorf("span not found")
	}
	return s, nil
}
func (r *memoryRepository) ListSpansByTrace(_ context.Context, tenantID, traceID string) ([]Span, error) {
	var out []Span
	for _, s := range r.spans {
		if s.TenantID == tenantID && s.TraceID == traceID {
			out = append(out, *s)
		}
	}
	return out, nil
}
func (r *memoryRepository) DeleteSpansByTrace(_ context.Context, tenantID, traceID string) error {
	for id, s := range r.spans {
		if s.TenantID == tenantID && s.TraceID == traceID {
			delete(r.spans, id)
		}
	}
	return nil
}
func (r *memoryRepository) GetPropagationConfig(_ context.Context, tenantID string) (*PropagationConfig, error) {
	c, ok := r.configs[tenantID]
	if !ok {
		return nil, fmt.Errorf("config not found")
	}
	return c, nil
}
func (r *memoryRepository) UpsertPropagationConfig(_ context.Context, config *PropagationConfig) error {
	r.configs[config.TenantID] = config
	return nil
}
func (r *memoryRepository) GetTraceStats(_ context.Context, tenantID, startTime, endTime string) (*TraceStats, error) {
	s, ok := r.stats[tenantID]
	if !ok {
		return &TraceStats{}, nil
	}
	return s, nil
}

// --- Tests ---

func newTestService() *Service {
	return NewService(newMemoryRepository())
}

func TestRecordSpan_CreatesTrace(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	span, err := svc.RecordSpan(ctx, "tenant-1", &CreateSpanRequest{
		TraceID:       "trace-001",
		SpanID:        "span-001",
		OperationName: "webhook.deliver",
		ServiceName:   "delivery-engine",
		SpanKind:      SpanKindProducer,
		StatusCode:    "OK",
		DurationMs:    150,
		StartedAt:     time.Now().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("RecordSpan: %v", err)
	}
	if span.ID == "" {
		t.Fatal("expected non-empty span ID")
	}
	if span.TraceID != "trace-001" {
		t.Fatalf("expected trace-001, got %s", span.TraceID)
	}

	// Verify trace was created
	trace, err := svc.GetTrace(ctx, "tenant-1", "trace-001")
	if err != nil {
		t.Fatalf("GetTrace: %v", err)
	}
	if trace.SpanCount != 1 {
		t.Fatalf("expected span count 1, got %d", trace.SpanCount)
	}
	if trace.Status != TraceStatusActive {
		t.Fatalf("expected active status, got %s", trace.Status)
	}
}

func TestRecordSpan_UpdatesExistingTrace(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	now := time.Now()

	// First span
	_, err := svc.RecordSpan(ctx, "tenant-1", &CreateSpanRequest{
		TraceID:       "trace-002",
		SpanID:        "span-001",
		OperationName: "webhook.deliver",
		ServiceName:   "delivery-engine",
		SpanKind:      SpanKindProducer,
		DurationMs:    100,
		StartedAt:     now.Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("RecordSpan 1: %v", err)
	}

	// Second span
	_, err = svc.RecordSpan(ctx, "tenant-1", &CreateSpanRequest{
		TraceID:       "trace-002",
		SpanID:        "span-002",
		ParentSpanID:  "span-001",
		OperationName: "http.request",
		ServiceName:   "delivery-engine",
		SpanKind:      SpanKindClient,
		DurationMs:    50,
		StartedAt:     now.Add(10 * time.Millisecond).Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("RecordSpan 2: %v", err)
	}

	trace, err := svc.GetTrace(ctx, "tenant-1", "trace-002")
	if err != nil {
		t.Fatalf("GetTrace: %v", err)
	}
	if trace.SpanCount != 2 {
		t.Fatalf("expected span count 2, got %d", trace.SpanCount)
	}
}

func TestRecordSpan_ErrorPropagation(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, err := svc.RecordSpan(ctx, "tenant-1", &CreateSpanRequest{
		TraceID:       "trace-err",
		SpanID:        "span-001",
		OperationName: "webhook.deliver",
		ServiceName:   "delivery-engine",
		SpanKind:      SpanKindProducer,
		StatusCode:    "ERROR",
		StatusMessage: "connection timeout",
		DurationMs:    5000,
		StartedAt:     time.Now().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("RecordSpan: %v", err)
	}

	trace, err := svc.GetTrace(ctx, "tenant-1", "trace-err")
	if err != nil {
		t.Fatalf("GetTrace: %v", err)
	}
	if !trace.HasErrors {
		t.Fatal("expected trace to have errors")
	}
}

func TestRecordSpan_DefaultStatusCode(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	span, err := svc.RecordSpan(ctx, "tenant-1", &CreateSpanRequest{
		TraceID:       "trace-default",
		SpanID:        "span-001",
		OperationName: "op",
		ServiceName:   "svc",
		SpanKind:      SpanKindInternal,
		DurationMs:    10,
		StartedAt:     time.Now().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("RecordSpan: %v", err)
	}
	if span.StatusCode != "OK" {
		t.Fatalf("expected default status OK, got %s", span.StatusCode)
	}
}

func TestRecordSpan_InvalidTimestamp(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, err := svc.RecordSpan(ctx, "tenant-1", &CreateSpanRequest{
		TraceID:       "trace-bad",
		SpanID:        "span-001",
		OperationName: "op",
		ServiceName:   "svc",
		SpanKind:      SpanKindInternal,
		DurationMs:    10,
		StartedAt:     "not-a-timestamp",
	})
	if err == nil {
		t.Fatal("expected error for invalid timestamp")
	}
}

func TestSearchTraces(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	// Create traces via spans
	for i := 0; i < 3; i++ {
		_, err := svc.RecordSpan(ctx, "tenant-1", &CreateSpanRequest{
			TraceID:       fmt.Sprintf("trace-search-%d", i),
			SpanID:        fmt.Sprintf("span-%d", i),
			OperationName: "op",
			ServiceName:   "delivery-engine",
			SpanKind:      SpanKindProducer,
			DurationMs:    int64(100 * (i + 1)),
			StartedAt:     time.Now().Format(time.RFC3339),
		})
		if err != nil {
			t.Fatalf("RecordSpan %d: %v", i, err)
		}
	}

	traces, total, err := svc.SearchTraces(ctx, "tenant-1", TraceSearchRequest{
		ServiceName: "delivery-engine",
	})
	if err != nil {
		t.Fatalf("SearchTraces: %v", err)
	}
	if total == 0 || len(traces) == 0 {
		t.Fatal("expected traces to be returned")
	}
}

func TestSearchTraces_DefaultLimit(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, _, err := svc.SearchTraces(ctx, "tenant-1", TraceSearchRequest{Limit: 0})
	if err != nil {
		t.Fatalf("SearchTraces with zero limit: %v", err)
	}
}

func TestCompleteTrace(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, err := svc.RecordSpan(ctx, "tenant-1", &CreateSpanRequest{
		TraceID:       "trace-complete",
		SpanID:        "span-001",
		OperationName: "op",
		ServiceName:   "svc",
		SpanKind:      SpanKindProducer,
		DurationMs:    100,
		StartedAt:     time.Now().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("RecordSpan: %v", err)
	}

	trace, err := svc.CompleteTrace(ctx, "tenant-1", "trace-complete")
	if err != nil {
		t.Fatalf("CompleteTrace: %v", err)
	}
	if trace.Status != TraceStatusCompleted {
		t.Fatalf("expected completed, got %s", trace.Status)
	}
}

func TestCompleteTrace_WithErrors(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, err := svc.RecordSpan(ctx, "tenant-1", &CreateSpanRequest{
		TraceID:       "trace-err-complete",
		SpanID:        "span-001",
		OperationName: "op",
		ServiceName:   "svc",
		SpanKind:      SpanKindProducer,
		StatusCode:    "ERROR",
		DurationMs:    100,
		StartedAt:     time.Now().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("RecordSpan: %v", err)
	}

	trace, err := svc.CompleteTrace(ctx, "tenant-1", "trace-err-complete")
	if err != nil {
		t.Fatalf("CompleteTrace: %v", err)
	}
	if trace.Status != TraceStatusError {
		t.Fatalf("expected error status, got %s", trace.Status)
	}
}

func TestGenerateTraceContext(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	traceCtx, err := svc.GenerateTraceContext(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("GenerateTraceContext: %v", err)
	}
	if traceCtx == nil {
		t.Fatal("expected non-nil trace context")
	}
	if len(traceCtx.TraceID) != 32 { // 16 bytes = 32 hex chars
		t.Fatalf("expected 32-char trace ID, got %d", len(traceCtx.TraceID))
	}
	if len(traceCtx.SpanID) != 16 { // 8 bytes = 16 hex chars
		t.Fatalf("expected 16-char span ID, got %d", len(traceCtx.SpanID))
	}
	if traceCtx.TraceFlags != "01" {
		t.Fatalf("expected TraceFlags 01, got %s", traceCtx.TraceFlags)
	}
}

func TestGenerateTraceContext_DisabledTenant(t *testing.T) {
	repo := newMemoryRepository()
	repo.configs["tenant-disabled"] = &PropagationConfig{
		TenantID: "tenant-disabled",
		IsActive: false,
	}
	svc := NewService(repo)
	ctx := context.Background()

	traceCtx, err := svc.GenerateTraceContext(ctx, "tenant-disabled")
	if err != nil {
		t.Fatalf("GenerateTraceContext: %v", err)
	}
	if traceCtx != nil {
		t.Fatal("expected nil trace context for disabled tenant")
	}
}

func TestGetPropagationConfig_Defaults(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	config, err := svc.GetPropagationConfig(ctx, "tenant-no-config")
	if err != nil {
		t.Fatalf("GetPropagationConfig: %v", err)
	}
	if !config.IsActive {
		t.Fatal("expected default config to be active")
	}
	if config.HeaderPrefix != "traceparent" {
		t.Fatalf("expected default header prefix 'traceparent', got %s", config.HeaderPrefix)
	}
	if config.SamplingRate != 1.0 {
		t.Fatalf("expected default sampling rate 1.0, got %f", config.SamplingRate)
	}
}

func TestUpdatePropagationConfig(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	config, err := svc.UpdatePropagationConfig(ctx, "tenant-1", &UpdatePropagationConfigRequest{
		InjectHeaders: true,
		InjectPayload: true,
		HeaderPrefix:  "x-trace",
		PayloadField:  "_tracing",
		SamplingRate:  0.5,
		IsActive:      true,
	})
	if err != nil {
		t.Fatalf("UpdatePropagationConfig: %v", err)
	}
	if config.HeaderPrefix != "x-trace" {
		t.Fatalf("expected x-trace, got %s", config.HeaderPrefix)
	}
	if config.SamplingRate != 0.5 {
		t.Fatalf("expected 0.5, got %f", config.SamplingRate)
	}
}

func TestUpdatePropagationConfig_Defaults(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	config, err := svc.UpdatePropagationConfig(ctx, "tenant-1", &UpdatePropagationConfigRequest{
		IsActive: true,
	})
	if err != nil {
		t.Fatalf("UpdatePropagationConfig: %v", err)
	}
	if config.HeaderPrefix != "traceparent" {
		t.Fatalf("expected default traceparent, got %s", config.HeaderPrefix)
	}
	if config.PayloadField != "_trace" {
		t.Fatalf("expected default _trace, got %s", config.PayloadField)
	}
	if config.SamplingRate != 1.0 {
		t.Fatalf("expected default 1.0 sampling rate, got %f", config.SamplingRate)
	}
}

func TestGetSpanWaterfall(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	now := time.Now()

	// Root span
	_, err := svc.RecordSpan(ctx, "tenant-1", &CreateSpanRequest{
		TraceID:       "trace-wf",
		SpanID:        "root-span",
		OperationName: "webhook.process",
		ServiceName:   "api",
		SpanKind:      SpanKindServer,
		DurationMs:    200,
		StartedAt:     now.Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("RecordSpan root: %v", err)
	}

	// Child span
	_, err = svc.RecordSpan(ctx, "tenant-1", &CreateSpanRequest{
		TraceID:       "trace-wf",
		SpanID:        "child-span",
		ParentSpanID:  "root-span",
		OperationName: "http.deliver",
		ServiceName:   "delivery",
		SpanKind:      SpanKindClient,
		DurationMs:    100,
		StartedAt:     now.Add(10 * time.Millisecond).Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("RecordSpan child: %v", err)
	}

	wf, err := svc.GetSpanWaterfall(ctx, "tenant-1", "trace-wf")
	if err != nil {
		t.Fatalf("GetSpanWaterfall: %v", err)
	}
	if wf.TotalSpans != 2 {
		t.Fatalf("expected 2 spans, got %d", wf.TotalSpans)
	}
	if wf.RootSpan == nil {
		t.Fatal("expected non-nil root span")
	}
	if len(wf.ServiceMap) != 2 {
		t.Fatalf("expected 2 services in map, got %d", len(wf.ServiceMap))
	}
}

func TestGetSpanWaterfall_NoSpans(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, err := svc.GetSpanWaterfall(ctx, "tenant-1", "nonexistent-trace")
	if err == nil {
		t.Fatal("expected error for nonexistent trace")
	}
}
