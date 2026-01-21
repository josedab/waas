package observability

import (
	"time"
)

// TraceContext represents W3C trace context for distributed tracing
type TraceContext struct {
	TraceID      string `json:"trace_id"`
	SpanID       string `json:"span_id"`
	ParentSpanID string `json:"parent_span_id,omitempty"`
	TraceFlags   byte   `json:"trace_flags"`
	TraceState   string `json:"trace_state,omitempty"`
}

// WebhookSpan represents a span in a webhook delivery chain
type WebhookSpan struct {
	ID            string                 `json:"id" db:"id"`
	TraceID       string                 `json:"trace_id" db:"trace_id"`
	SpanID        string                 `json:"span_id" db:"span_id"`
	ParentSpanID  string                 `json:"parent_span_id,omitempty" db:"parent_span_id"`
	TenantID      string                 `json:"tenant_id" db:"tenant_id"`
	WebhookID     string                 `json:"webhook_id,omitempty" db:"webhook_id"`
	EndpointID    string                 `json:"endpoint_id,omitempty" db:"endpoint_id"`
	DeliveryID    string                 `json:"delivery_id,omitempty" db:"delivery_id"`
	OperationName string                 `json:"operation_name" db:"operation_name"`
	ServiceName   string                 `json:"service_name" db:"service_name"`
	Kind          SpanKind               `json:"kind" db:"kind"`
	Status        SpanStatus             `json:"status" db:"status"`
	StatusMessage string                 `json:"status_message,omitempty" db:"status_message"`
	StartTime     time.Time              `json:"start_time" db:"start_time"`
	EndTime       time.Time              `json:"end_time" db:"end_time"`
	DurationMs    int64                  `json:"duration_ms" db:"duration_ms"`
	Attributes    map[string]interface{} `json:"attributes,omitempty" db:"attributes"`
	Events        []SpanEvent            `json:"events,omitempty" db:"events"`
	Links         []SpanLink             `json:"links,omitempty" db:"links"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
}

// SpanKind indicates the role of a span
type SpanKind string

const (
	SpanKindInternal SpanKind = "internal"
	SpanKindServer   SpanKind = "server"
	SpanKindClient   SpanKind = "client"
	SpanKindProducer SpanKind = "producer"
	SpanKindConsumer SpanKind = "consumer"
)

// SpanStatus represents the status of a span
type SpanStatus string

const (
	SpanStatusUnset SpanStatus = "unset"
	SpanStatusOK    SpanStatus = "ok"
	SpanStatusError SpanStatus = "error"
)

// SpanEvent represents a timed event within a span
type SpanEvent struct {
	Name       string                 `json:"name"`
	Timestamp  time.Time              `json:"timestamp"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// SpanLink represents a link to another span
type SpanLink struct {
	TraceID    string                 `json:"trace_id"`
	SpanID     string                 `json:"span_id"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// Trace represents a complete trace with all spans
type Trace struct {
	TraceID    string        `json:"trace_id"`
	TenantID   string        `json:"tenant_id"`
	RootSpan   *WebhookSpan  `json:"root_span"`
	Spans      []WebhookSpan `json:"spans"`
	StartTime  time.Time     `json:"start_time"`
	EndTime    time.Time     `json:"end_time"`
	DurationMs int64         `json:"duration_ms"`
	SpanCount  int           `json:"span_count"`
	ErrorCount int           `json:"error_count"`
	Services   []string      `json:"services"`
}

// TraceTimeline represents a timeline view of a trace
type TraceTimeline struct {
	TraceID   string         `json:"trace_id"`
	StartTime time.Time      `json:"start_time"`
	EndTime   time.Time      `json:"end_time"`
	TotalMs   int64          `json:"total_ms"`
	Spans     []TimelineSpan `json:"spans"`
	Waterfall []WaterfallRow `json:"waterfall"`
	CritPath  []string       `json:"critical_path"`
}

// TimelineSpan represents a span in the timeline
type TimelineSpan struct {
	SpanID        string  `json:"span_id"`
	OperationName string  `json:"operation_name"`
	ServiceName   string  `json:"service_name"`
	StartOffset   int64   `json:"start_offset_ms"`
	DurationMs    int64   `json:"duration_ms"`
	DepthLevel    int     `json:"depth_level"`
	Status        string  `json:"status"`
	PercentOfRoot float64 `json:"percent_of_root"`
}

// WaterfallRow represents a row in waterfall visualization
type WaterfallRow struct {
	SpanID   string         `json:"span_id"`
	Label    string         `json:"label"`
	Bars     []WaterfallBar `json:"bars"`
	Depth    int            `json:"depth"`
	HasError bool           `json:"has_error"`
}

// WaterfallBar represents a bar segment in waterfall
type WaterfallBar struct {
	Type       string  `json:"type"` // queued, active, blocked
	StartPct   float64 `json:"start_pct"`
	WidthPct   float64 `json:"width_pct"`
	DurationMs int64   `json:"duration_ms"`
}

// LatencyBreakdown provides latency analysis
type LatencyBreakdown struct {
	TraceID         string           `json:"trace_id"`
	TotalMs         int64            `json:"total_ms"`
	ByService       map[string]int64 `json:"by_service"`
	ByOperation     map[string]int64 `json:"by_operation"`
	NetworkTime     int64            `json:"network_time_ms"`
	ProcessingTime  int64            `json:"processing_time_ms"`
	QueueTime       int64            `json:"queue_time_ms"`
	Bottleneck      string           `json:"bottleneck"`
	BottleneckPct   float64          `json:"bottleneck_pct"`
	Recommendations []string         `json:"recommendations"`
}

// TraceSearchQuery represents search parameters for traces
type TraceSearchQuery struct {
	TenantID      string            `json:"tenant_id"`
	TraceIDs      []string          `json:"trace_ids,omitempty"`
	WebhookID     string            `json:"webhook_id,omitempty"`
	EndpointID    string            `json:"endpoint_id,omitempty"`
	ServiceName   string            `json:"service_name,omitempty"`
	OperationName string            `json:"operation_name,omitempty"`
	Status        SpanStatus        `json:"status,omitempty"`
	MinDuration   int64             `json:"min_duration_ms,omitempty"`
	MaxDuration   int64             `json:"max_duration_ms,omitempty"`
	StartTime     time.Time         `json:"start_time"`
	EndTime       time.Time         `json:"end_time"`
	Tags          map[string]string `json:"tags,omitempty"`
	Limit         int               `json:"limit"`
	Offset        int               `json:"offset"`
}

// TraceSearchResult represents search results
type TraceSearchResult struct {
	Traces     []TraceSummary `json:"traces"`
	TotalCount int            `json:"total_count"`
	HasMore    bool           `json:"has_more"`
}

// TraceSummary provides a summary of a trace
type TraceSummary struct {
	TraceID       string    `json:"trace_id"`
	RootService   string    `json:"root_service"`
	RootOperation string    `json:"root_operation"`
	StartTime     time.Time `json:"start_time"`
	DurationMs    int64     `json:"duration_ms"`
	SpanCount     int       `json:"span_count"`
	ErrorCount    int       `json:"error_count"`
	Status        string    `json:"status"`
}

// ServiceDependency represents a dependency between services
type ServiceDependency struct {
	Source     string  `json:"source"`
	Target     string  `json:"target"`
	CallCount  int64   `json:"call_count"`
	ErrorCount int64   `json:"error_count"`
	AvgLatency float64 `json:"avg_latency_ms"`
	P99Latency float64 `json:"p99_latency_ms"`
}

// ServiceMap represents the service dependency graph
type ServiceMap struct {
	TenantID     string              `json:"tenant_id"`
	Services     []ServiceNode       `json:"services"`
	Dependencies []ServiceDependency `json:"dependencies"`
	GeneratedAt  time.Time           `json:"generated_at"`
}

// ServiceNode represents a service in the map
type ServiceNode struct {
	Name       string  `json:"name"`
	SpanCount  int64   `json:"span_count"`
	ErrorRate  float64 `json:"error_rate"`
	AvgLatency float64 `json:"avg_latency_ms"`
}

// OTelExportConfig configures OpenTelemetry export
type OTelExportConfig struct {
	ID        string            `json:"id" db:"id"`
	TenantID  string            `json:"tenant_id" db:"tenant_id"`
	Name      string            `json:"name" db:"name"`
	Enabled   bool              `json:"enabled" db:"enabled"`
	Protocol  string            `json:"protocol" db:"protocol"` // grpc, http
	Endpoint  string            `json:"endpoint" db:"endpoint"`
	Headers   map[string]string `json:"headers,omitempty" db:"headers"`
	Sampling  SamplingConfig    `json:"sampling" db:"sampling"`
	BatchSize int               `json:"batch_size" db:"batch_size"`
	Timeout   int               `json:"timeout_seconds" db:"timeout_seconds"`
	CreatedAt time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt time.Time         `json:"updated_at" db:"updated_at"`
}

// SamplingConfig defines trace sampling configuration
type SamplingConfig struct {
	Strategy   string  `json:"strategy"` // always, never, ratio, rate_limiting
	Ratio      float64 `json:"ratio,omitempty"`
	RatePerSec int     `json:"rate_per_second,omitempty"`
}

// CreateSpanRequest represents a request to create a span
type CreateSpanRequest struct {
	TraceID       string                 `json:"trace_id"`
	ParentSpanID  string                 `json:"parent_span_id,omitempty"`
	OperationName string                 `json:"operation_name" binding:"required"`
	ServiceName   string                 `json:"service_name" binding:"required"`
	Kind          SpanKind               `json:"kind"`
	Attributes    map[string]interface{} `json:"attributes,omitempty"`
}

// CreateExportConfigRequest represents a request to create an export config
type CreateExportConfigRequest struct {
	Name      string            `json:"name" binding:"required"`
	Protocol  string            `json:"protocol" binding:"required,oneof=grpc http"`
	Endpoint  string            `json:"endpoint" binding:"required,url"`
	Headers   map[string]string `json:"headers,omitempty"`
	Sampling  SamplingConfig    `json:"sampling"`
	BatchSize int               `json:"batch_size"`
	Timeout   int               `json:"timeout_seconds"`
}

// TraceMetrics provides aggregated trace metrics
type TraceMetrics struct {
	TenantID     string           `json:"tenant_id"`
	Period       string           `json:"period"`
	TotalTraces  int64            `json:"total_traces"`
	TotalSpans   int64            `json:"total_spans"`
	ErrorRate    float64          `json:"error_rate"`
	AvgDuration  float64          `json:"avg_duration_ms"`
	P50Duration  float64          `json:"p50_duration_ms"`
	P95Duration  float64          `json:"p95_duration_ms"`
	P99Duration  float64          `json:"p99_duration_ms"`
	ByService    map[string]int64 `json:"by_service"`
	ByOperation  map[string]int64 `json:"by_operation"`
	LatencyTrend []LatencyPoint   `json:"latency_trend"`
	ErrorTrend   []ErrorPoint     `json:"error_trend"`
}

// LatencyPoint represents a point in latency trend
type LatencyPoint struct {
	Timestamp time.Time `json:"timestamp"`
	AvgMs     float64   `json:"avg_ms"`
	P95Ms     float64   `json:"p95_ms"`
	P99Ms     float64   `json:"p99_ms"`
}

// ErrorPoint represents a point in error trend
type ErrorPoint struct {
	Timestamp  time.Time `json:"timestamp"`
	ErrorCount int64     `json:"error_count"`
	ErrorRate  float64   `json:"error_rate"`
}
