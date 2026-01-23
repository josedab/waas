package tracing

import "time"

// Trace represents a distributed trace across webhook delivery chains
type Trace struct {
	ID          string    `json:"id" db:"id"`
	TenantID    string    `json:"tenant_id" db:"tenant_id"`
	TraceID     string    `json:"trace_id" db:"trace_id"`
	RootSpanID  string    `json:"root_span_id" db:"root_span_id"`
	ServiceName string    `json:"service_name" db:"service_name"`
	OperationName string  `json:"operation_name" db:"operation_name"`
	Status      string    `json:"status" db:"status"`
	SpanCount   int       `json:"span_count" db:"span_count"`
	DurationMs  int64     `json:"duration_ms" db:"duration_ms"`
	HasErrors   bool      `json:"has_errors" db:"has_errors"`
	StartedAt   time.Time `json:"started_at" db:"started_at"`
	EndedAt     time.Time `json:"ended_at" db:"ended_at"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// Trace status constants
const (
	TraceStatusActive    = "active"
	TraceStatusCompleted = "completed"
	TraceStatusError     = "error"
)

// Span represents a single span within a trace
type Span struct {
	ID            string            `json:"id" db:"id"`
	TenantID      string            `json:"tenant_id" db:"tenant_id"`
	TraceID       string            `json:"trace_id" db:"trace_id"`
	SpanID        string            `json:"span_id" db:"span_id"`
	ParentSpanID  string            `json:"parent_span_id,omitempty" db:"parent_span_id"`
	OperationName string            `json:"operation_name" db:"operation_name"`
	ServiceName   string            `json:"service_name" db:"service_name"`
	SpanKind      string            `json:"span_kind" db:"span_kind"`
	StatusCode    string            `json:"status_code" db:"status_code"`
	StatusMessage string            `json:"status_message,omitempty" db:"status_message"`
	Attributes    map[string]string `json:"attributes,omitempty"`
	Events        []SpanEvent       `json:"events,omitempty"`
	DurationMs    int64             `json:"duration_ms" db:"duration_ms"`
	StartedAt     time.Time         `json:"started_at" db:"started_at"`
	EndedAt       time.Time         `json:"ended_at" db:"ended_at"`
}

// SpanKind constants (W3C)
const (
	SpanKindClient   = "CLIENT"
	SpanKindServer   = "SERVER"
	SpanKindProducer = "PRODUCER"
	SpanKindConsumer = "CONSUMER"
	SpanKindInternal = "INTERNAL"
)

// SpanEvent represents a time-stamped annotation on a span
type SpanEvent struct {
	Name       string            `json:"name"`
	Timestamp  time.Time         `json:"timestamp"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// TraceContext holds W3C TraceContext propagation headers
type TraceContext struct {
	TraceID    string `json:"traceparent_trace_id"`
	SpanID     string `json:"traceparent_span_id"`
	TraceFlags string `json:"traceparent_flags"`
	TraceState string `json:"tracestate,omitempty"`
}

// PropagationConfig controls how trace context is injected into webhook payloads
type PropagationConfig struct {
	ID              string `json:"id" db:"id"`
	TenantID        string `json:"tenant_id" db:"tenant_id"`
	InjectHeaders   bool   `json:"inject_headers" db:"inject_headers"`
	InjectPayload   bool   `json:"inject_payload" db:"inject_payload"`
	HeaderPrefix    string `json:"header_prefix" db:"header_prefix"`
	PayloadField    string `json:"payload_field" db:"payload_field"`
	SamplingRate    float64 `json:"sampling_rate" db:"sampling_rate"`
	IsActive        bool   `json:"is_active" db:"is_active"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// SpanWaterfall is a hierarchical view of spans for visualization
type SpanWaterfall struct {
	TraceID     string          `json:"trace_id"`
	RootSpan    *SpanNode       `json:"root_span"`
	TotalSpans  int             `json:"total_spans"`
	DurationMs  int64           `json:"total_duration_ms"`
	ServiceMap  map[string]int  `json:"service_map"`
}

// SpanNode is a tree node in the span waterfall
type SpanNode struct {
	Span     Span       `json:"span"`
	Children []SpanNode `json:"children,omitempty"`
}

// TraceSearchRequest is the request DTO for searching traces
type TraceSearchRequest struct {
	ServiceName   string `form:"service_name"`
	OperationName string `form:"operation_name"`
	Status        string `form:"status"`
	MinDurationMs int64  `form:"min_duration_ms"`
	MaxDurationMs int64  `form:"max_duration_ms"`
	HasErrors     *bool  `form:"has_errors"`
	StartTime     string `form:"start_time"`
	EndTime       string `form:"end_time"`
	Limit         int    `form:"limit"`
	Offset        int    `form:"offset"`
}

// CreateSpanRequest is the request DTO for recording a span
type CreateSpanRequest struct {
	TraceID       string            `json:"trace_id" binding:"required"`
	SpanID        string            `json:"span_id" binding:"required"`
	ParentSpanID  string            `json:"parent_span_id,omitempty"`
	OperationName string            `json:"operation_name" binding:"required"`
	ServiceName   string            `json:"service_name" binding:"required"`
	SpanKind      string            `json:"span_kind" binding:"required"`
	StatusCode    string            `json:"status_code"`
	StatusMessage string            `json:"status_message,omitempty"`
	Attributes    map[string]string `json:"attributes,omitempty"`
	DurationMs    int64             `json:"duration_ms" binding:"required,min=0"`
	StartedAt     string            `json:"started_at" binding:"required"`
}

// UpdatePropagationConfigRequest is the request DTO for configuring trace propagation
type UpdatePropagationConfigRequest struct {
	InjectHeaders bool    `json:"inject_headers"`
	InjectPayload bool    `json:"inject_payload"`
	HeaderPrefix  string  `json:"header_prefix"`
	PayloadField  string  `json:"payload_field"`
	SamplingRate  float64 `json:"sampling_rate" binding:"min=0,max=1"`
	IsActive      bool    `json:"is_active"`
}

// TraceStats aggregates trace statistics for a tenant
type TraceStats struct {
	TotalTraces      int64              `json:"total_traces"`
	ErrorTraces      int64              `json:"error_traces"`
	AvgDurationMs    float64            `json:"avg_duration_ms"`
	P50DurationMs    int64              `json:"p50_duration_ms"`
	P99DurationMs    int64              `json:"p99_duration_ms"`
	ServiceBreakdown map[string]int64   `json:"service_breakdown"`
	ErrorRate        float64            `json:"error_rate_pct"`
}
