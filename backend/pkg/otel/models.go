package otel

import (
	"context"
	"time"
)

// Config represents OpenTelemetry configuration
type Config struct {
	ID          string            `json:"id" db:"id"`
	TenantID    string            `json:"tenant_id" db:"tenant_id"`
	Name        string            `json:"name" db:"name"`
	ServiceName string            `json:"service_name" db:"service_name"`
	Enabled     bool              `json:"enabled" db:"enabled"`
	Traces      TracesConfig      `json:"traces" db:"traces"`
	Metrics     MetricsConfig     `json:"metrics" db:"metrics"`
	Logs        LogsConfig        `json:"logs" db:"logs"`
	Attributes  map[string]string `json:"attributes" db:"attributes"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" db:"updated_at"`
}

// TracesConfig represents trace exporter configuration
type TracesConfig struct {
	Enabled      bool              `json:"enabled"`
	Endpoint     string            `json:"endpoint"`
	Exporter     ExporterType      `json:"exporter"`
	SamplingRate float64           `json:"sampling_rate"`
	Propagators  []string          `json:"propagators"`
	Headers      map[string]string `json:"headers,omitempty"`
}

// MetricsConfig represents metrics exporter configuration
type MetricsConfig struct {
	Enabled    bool              `json:"enabled"`
	Endpoint   string            `json:"endpoint"`
	Exporter   ExporterType      `json:"exporter"`
	Interval   int               `json:"interval"` // seconds
	Histograms []string          `json:"histograms,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
}

// LogsConfig represents logs exporter configuration
type LogsConfig struct {
	Enabled  bool              `json:"enabled"`
	Endpoint string            `json:"endpoint"`
	Exporter ExporterType      `json:"exporter"`
	LogLevel string            `json:"log_level"`
	Headers  map[string]string `json:"headers,omitempty"`
}

// ExporterType represents the type of OTEL exporter
type ExporterType string

const (
	ExporterOTLP       ExporterType = "otlp"
	ExporterOTLPHTTP   ExporterType = "otlp-http"
	ExporterPrometheus ExporterType = "prometheus"
	ExporterJaeger     ExporterType = "jaeger"
	ExporterZipkin     ExporterType = "zipkin"
	ExporterStdout     ExporterType = "stdout"
)

// SpanData represents a recorded span
type SpanData struct {
	TraceID       string         `json:"trace_id"`
	SpanID        string         `json:"span_id"`
	ParentSpanID  string         `json:"parent_span_id,omitempty"`
	OperationName string         `json:"operation_name"`
	ServiceName   string         `json:"service_name"`
	StartTime     time.Time      `json:"start_time"`
	EndTime       time.Time      `json:"end_time"`
	Duration      time.Duration  `json:"duration"`
	Status        SpanStatus     `json:"status"`
	Attributes    map[string]any `json:"attributes,omitempty"`
	Events        []SpanEvent    `json:"events,omitempty"`
	Links         []SpanLink     `json:"links,omitempty"`
}

// SpanStatus represents the status of a span
type SpanStatus struct {
	Code        int    `json:"code"`
	Description string `json:"description,omitempty"`
}

// SpanEvent represents an event within a span
type SpanEvent struct {
	Name       string         `json:"name"`
	Timestamp  time.Time      `json:"timestamp"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

// SpanLink represents a link to another span
type SpanLink struct {
	TraceID    string         `json:"trace_id"`
	SpanID     string         `json:"span_id"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

// MetricData represents a recorded metric
type MetricData struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Unit        string            `json:"unit,omitempty"`
	Type        MetricType        `json:"type"`
	Value       float64           `json:"value"`
	Timestamp   time.Time         `json:"timestamp"`
	Attributes  map[string]string `json:"attributes,omitempty"`
}

// MetricType represents the type of metric
type MetricType string

const (
	MetricCounter   MetricType = "counter"
	MetricGauge     MetricType = "gauge"
	MetricHistogram MetricType = "histogram"
)

// DeliverySpanAttributes are attributes added to delivery spans
type DeliverySpanAttributes struct {
	WebhookID        string `json:"webhook.id"`
	EndpointID       string `json:"webhook.endpoint_id"`
	EndpointURL      string `json:"webhook.endpoint_url"`
	DeliveryID       string `json:"webhook.delivery_id"`
	DeliveryAttempt  int    `json:"webhook.delivery_attempt"`
	PayloadSizeBytes int64  `json:"webhook.payload_size_bytes"`
	ResponseStatus   int    `json:"http.response_status_code"`
	ErrorType        string `json:"error.type,omitempty"`
	ErrorMessage     string `json:"error.message,omitempty"`
}

// StandardSpanNames define standard operation names for consistency
var StandardSpanNames = struct {
	DeliveryProcess   string
	DeliveryAttempt   string
	PayloadTransform  string
	SignatureGenerate string
	RetrySchedule     string
	EndpointHealth    string
}{
	DeliveryProcess:   "webhook.delivery.process",
	DeliveryAttempt:   "webhook.delivery.attempt",
	PayloadTransform:  "webhook.payload.transform",
	SignatureGenerate: "webhook.signature.generate",
	RetrySchedule:     "webhook.retry.schedule",
	EndpointHealth:    "webhook.endpoint.health",
}

// StandardMetricNames define standard metric names
var StandardMetricNames = struct {
	DeliveriesTotal      string
	DeliveriesSuccessful string
	DeliveriesFailed     string
	DeliveryDuration     string
	PayloadSize          string
	RetryCount           string
	EndpointsActive      string
	QueueDepth           string
}{
	DeliveriesTotal:      "webhook.deliveries.total",
	DeliveriesSuccessful: "webhook.deliveries.successful",
	DeliveriesFailed:     "webhook.deliveries.failed",
	DeliveryDuration:     "webhook.delivery.duration",
	PayloadSize:          "webhook.payload.size",
	RetryCount:           "webhook.retry.count",
	EndpointsActive:      "webhook.endpoints.active",
	QueueDepth:           "webhook.queue.depth",
}

// CreateConfigRequest represents a request to create OTEL config
type CreateConfigRequest struct {
	Name        string            `json:"name" binding:"required"`
	ServiceName string            `json:"service_name"`
	Enabled     bool              `json:"enabled"`
	Traces      TracesConfig      `json:"traces"`
	Metrics     MetricsConfig     `json:"metrics"`
	Logs        LogsConfig        `json:"logs"`
	Attributes  map[string]string `json:"attributes"`
}

// UpdateConfigRequest represents a request to update OTEL config
type UpdateConfigRequest struct {
	Name        *string           `json:"name,omitempty"`
	ServiceName *string           `json:"service_name,omitempty"`
	Enabled     *bool             `json:"enabled,omitempty"`
	Traces      *TracesConfig     `json:"traces,omitempty"`
	Metrics     *MetricsConfig    `json:"metrics,omitempty"`
	Logs        *LogsConfig       `json:"logs,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
}

// TraceContext represents distributed trace context
type TraceContext struct {
	TraceID    string
	SpanID     string
	TraceFlags byte
	TraceState string
}

// TraceContextKey is the key for trace context in Go context
type TraceContextKey struct{}

// GetTraceContext extracts trace context from Go context
func GetTraceContext(ctx context.Context) *TraceContext {
	if tc, ok := ctx.Value(TraceContextKey{}).(*TraceContext); ok {
		return tc
	}
	return nil
}

// WithTraceContext adds trace context to Go context
func WithTraceContext(ctx context.Context, tc *TraceContext) context.Context {
	return context.WithValue(ctx, TraceContextKey{}, tc)
}

// PropagatorType represents a context propagator type
type PropagatorType string

const (
	PropagatorW3CTraceContext PropagatorType = "tracecontext"
	PropagatorW3CBaggage      PropagatorType = "baggage"
	PropagatorB3              PropagatorType = "b3"
	PropagatorB3Multi         PropagatorType = "b3multi"
	PropagatorJaeger          PropagatorType = "jaeger"
)

// DefaultConfig returns a default OTEL configuration
func DefaultConfig(tenantID string) *Config {
	return &Config{
		TenantID:    tenantID,
		Name:        "Default OTEL Config",
		ServiceName: "waas-webhook-service",
		Enabled:     false,
		Traces: TracesConfig{
			Enabled:      true,
			Exporter:     ExporterOTLP,
			SamplingRate: 1.0,
			Propagators:  []string{string(PropagatorW3CTraceContext), string(PropagatorW3CBaggage)},
		},
		Metrics: MetricsConfig{
			Enabled:  true,
			Exporter: ExporterPrometheus,
			Interval: 30,
			Histograms: []string{
				StandardMetricNames.DeliveryDuration,
				StandardMetricNames.PayloadSize,
			},
		},
		Logs: LogsConfig{
			Enabled:  false,
			Exporter: ExporterOTLP,
			LogLevel: "info",
		},
		Attributes: map[string]string{
			"service.version":        "1.0.0",
			"deployment.environment": "production",
		},
	}
}
