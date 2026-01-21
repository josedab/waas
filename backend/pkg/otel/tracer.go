package otel

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Tracer provides tracing functionality
type Tracer struct {
	config      *Config
	serviceName string
	attributes  map[string]string
	spans       chan *SpanData
	metrics     chan *MetricData
	exporters   []Exporter
	mu          sync.RWMutex
	running     bool
	stopCh      chan struct{}
}

// Exporter interface for exporting telemetry data
type Exporter interface {
	ExportSpans(ctx context.Context, spans []*SpanData) error
	ExportMetrics(ctx context.Context, metrics []*MetricData) error
	Shutdown(ctx context.Context) error
}

// NewTracer creates a new tracer instance
func NewTracer(config *Config) *Tracer {
	serviceName := config.ServiceName
	if serviceName == "" {
		serviceName = "waas-webhook-service"
	}

	return &Tracer{
		config:      config,
		serviceName: serviceName,
		attributes:  config.Attributes,
		spans:       make(chan *SpanData, 1000),
		metrics:     make(chan *MetricData, 1000),
		exporters:   []Exporter{},
		stopCh:      make(chan struct{}),
	}
}

// AddExporter adds an exporter to the tracer
func (t *Tracer) AddExporter(e Exporter) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.exporters = append(t.exporters, e)
}

// Start begins the tracer background processing
func (t *Tracer) Start(ctx context.Context) error {
	t.mu.Lock()
	if t.running {
		t.mu.Unlock()
		return nil
	}
	t.running = true
	t.mu.Unlock()

	go t.processSpans(ctx)
	go t.processMetrics(ctx)

	return nil
}

// Stop gracefully stops the tracer
func (t *Tracer) Stop(ctx context.Context) error {
	t.mu.Lock()
	if !t.running {
		t.mu.Unlock()
		return nil
	}
	t.running = false
	close(t.stopCh)
	t.mu.Unlock()

	// Shutdown exporters
	for _, e := range t.exporters {
		if err := e.Shutdown(ctx); err != nil {
			// Log but continue shutting down others
			fmt.Printf("error shutting down exporter: %v\n", err)
		}
	}

	return nil
}

// StartSpan starts a new span
func (t *Tracer) StartSpan(ctx context.Context, operationName string, opts ...SpanOption) (context.Context, *Span) {
	span := &Span{
		tracer:        t,
		traceID:       generateTraceID(),
		spanID:        generateSpanID(),
		operationName: operationName,
		serviceName:   t.serviceName,
		startTime:     time.Now(),
		attributes:    make(map[string]any),
		events:        []SpanEvent{},
		links:         []SpanLink{},
	}

	// Check for parent context
	if tc := GetTraceContext(ctx); tc != nil {
		span.traceID = tc.TraceID
		span.parentSpanID = tc.SpanID
	}

	// Apply options
	for _, opt := range opts {
		opt(span)
	}

	// Add default attributes
	for k, v := range t.attributes {
		if _, exists := span.attributes[k]; !exists {
			span.attributes[k] = v
		}
	}

	// Create new trace context
	tc := &TraceContext{
		TraceID:    span.traceID,
		SpanID:     span.spanID,
		TraceFlags: 1, // sampled
	}

	return WithTraceContext(ctx, tc), span
}

// RecordMetric records a metric value
func (t *Tracer) RecordMetric(name string, value float64, metricType MetricType, attrs map[string]string) {
	metric := &MetricData{
		Name:       name,
		Type:       metricType,
		Value:      value,
		Timestamp:  time.Now(),
		Attributes: attrs,
	}

	select {
	case t.metrics <- metric:
	default:
		// Channel full, drop metric
	}
}

// IncrementCounter increments a counter metric
func (t *Tracer) IncrementCounter(name string, attrs map[string]string) {
	t.RecordMetric(name, 1, MetricCounter, attrs)
}

// RecordGauge records a gauge metric
func (t *Tracer) RecordGauge(name string, value float64, attrs map[string]string) {
	t.RecordMetric(name, value, MetricGauge, attrs)
}

// RecordHistogram records a histogram metric
func (t *Tracer) RecordHistogram(name string, value float64, attrs map[string]string) {
	t.RecordMetric(name, value, MetricHistogram, attrs)
}

func (t *Tracer) processSpans(ctx context.Context) {
	batch := make([]*SpanData, 0, 100)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-t.stopCh:
			// Flush remaining spans
			if len(batch) > 0 {
				t.exportSpans(ctx, batch)
			}
			return
		case span := <-t.spans:
			batch = append(batch, span)
			if len(batch) >= 100 {
				t.exportSpans(ctx, batch)
				batch = make([]*SpanData, 0, 100)
			}
		case <-ticker.C:
			if len(batch) > 0 {
				t.exportSpans(ctx, batch)
				batch = make([]*SpanData, 0, 100)
			}
		}
	}
}

func (t *Tracer) processMetrics(ctx context.Context) {
	batch := make([]*MetricData, 0, 100)
	interval := 30 * time.Second
	if t.config.Metrics.Interval > 0 {
		interval = time.Duration(t.config.Metrics.Interval) * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-t.stopCh:
			// Flush remaining metrics
			if len(batch) > 0 {
				t.exportMetrics(ctx, batch)
			}
			return
		case metric := <-t.metrics:
			batch = append(batch, metric)
		case <-ticker.C:
			if len(batch) > 0 {
				t.exportMetrics(ctx, batch)
				batch = make([]*MetricData, 0, 100)
			}
		}
	}
}

func (t *Tracer) exportSpans(ctx context.Context, spans []*SpanData) {
	for _, e := range t.exporters {
		if err := e.ExportSpans(ctx, spans); err != nil {
			fmt.Printf("error exporting spans: %v\n", err)
		}
	}
}

func (t *Tracer) exportMetrics(ctx context.Context, metrics []*MetricData) {
	for _, e := range t.exporters {
		if err := e.ExportMetrics(ctx, metrics); err != nil {
			fmt.Printf("error exporting metrics: %v\n", err)
		}
	}
}

// Span represents an individual span
type Span struct {
	tracer        *Tracer
	traceID       string
	spanID        string
	parentSpanID  string
	operationName string
	serviceName   string
	startTime     time.Time
	endTime       time.Time
	status        SpanStatus
	attributes    map[string]any
	events        []SpanEvent
	links         []SpanLink
}

// SetAttribute sets an attribute on the span
func (s *Span) SetAttribute(key string, value any) {
	s.attributes[key] = value
}

// SetStatus sets the span status
func (s *Span) SetStatus(code int, description string) {
	s.status = SpanStatus{Code: code, Description: description}
}

// SetError marks the span as an error
func (s *Span) SetError(err error) {
	s.status = SpanStatus{Code: 2, Description: err.Error()} // Error status code
	s.attributes["error"] = true
	s.attributes["error.message"] = err.Error()
}

// AddEvent adds an event to the span
func (s *Span) AddEvent(name string, attrs map[string]any) {
	s.events = append(s.events, SpanEvent{
		Name:       name,
		Timestamp:  time.Now(),
		Attributes: attrs,
	})
}

// AddLink adds a link to another span
func (s *Span) AddLink(traceID, spanID string, attrs map[string]any) {
	s.links = append(s.links, SpanLink{
		TraceID:    traceID,
		SpanID:     spanID,
		Attributes: attrs,
	})
}

// End ends the span and records it
func (s *Span) End() {
	s.endTime = time.Now()

	spanData := &SpanData{
		TraceID:       s.traceID,
		SpanID:        s.spanID,
		ParentSpanID:  s.parentSpanID,
		OperationName: s.operationName,
		ServiceName:   s.serviceName,
		StartTime:     s.startTime,
		EndTime:       s.endTime,
		Duration:      s.endTime.Sub(s.startTime),
		Status:        s.status,
		Attributes:    s.attributes,
		Events:        s.events,
		Links:         s.links,
	}

	select {
	case s.tracer.spans <- spanData:
	default:
		// Channel full, drop span
	}
}

// SpanOption is a function that configures a span
type SpanOption func(*Span)

// WithParent sets the parent span
func WithParent(parentSpan *Span) SpanOption {
	return func(s *Span) {
		if parentSpan != nil {
			s.traceID = parentSpan.traceID
			s.parentSpanID = parentSpan.spanID
		}
	}
}

// WithAttributes sets initial attributes
func WithAttributes(attrs map[string]any) SpanOption {
	return func(s *Span) {
		for k, v := range attrs {
			s.attributes[k] = v
		}
	}
}

// WithLinks sets initial links
func WithLinks(links []SpanLink) SpanOption {
	return func(s *Span) {
		s.links = append(s.links, links...)
	}
}

// Middleware provides HTTP middleware for tracing
func (t *Tracer) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract trace context from headers
			ctx := t.ExtractContext(r.Context(), r.Header)

			// Start span
			ctx, span := t.StartSpan(ctx, fmt.Sprintf("%s %s", r.Method, r.URL.Path))
			defer span.End()

			// Set HTTP attributes
			span.SetAttribute("http.method", r.Method)
			span.SetAttribute("http.url", r.URL.String())
			span.SetAttribute("http.host", r.Host)
			span.SetAttribute("http.user_agent", r.UserAgent())

			// Wrap response writer to capture status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Inject context into response headers
			t.InjectContext(ctx, w.Header())

			// Call next handler
			next.ServeHTTP(rw, r.WithContext(ctx))

			// Record response attributes
			span.SetAttribute("http.status_code", rw.statusCode)
			if rw.statusCode >= 400 {
				span.SetStatus(2, fmt.Sprintf("HTTP %d", rw.statusCode))
			}
		})
	}
}

// ExtractContext extracts trace context from HTTP headers
func (t *Tracer) ExtractContext(ctx context.Context, headers http.Header) context.Context {
	// W3C Trace Context
	if traceparent := headers.Get("traceparent"); traceparent != "" {
		tc := parseTraceparent(traceparent)
		if tc != nil {
			tc.TraceState = headers.Get("tracestate")
			return WithTraceContext(ctx, tc)
		}
	}

	// B3 headers
	if traceID := headers.Get("X-B3-TraceId"); traceID != "" {
		tc := &TraceContext{
			TraceID: traceID,
			SpanID:  headers.Get("X-B3-SpanId"),
		}
		return WithTraceContext(ctx, tc)
	}

	return ctx
}

// InjectContext injects trace context into HTTP headers
func (t *Tracer) InjectContext(ctx context.Context, headers http.Header) {
	tc := GetTraceContext(ctx)
	if tc == nil {
		return
	}

	// W3C Trace Context
	headers.Set("traceparent", fmt.Sprintf("00-%s-%s-%02x", tc.TraceID, tc.SpanID, tc.TraceFlags))
	if tc.TraceState != "" {
		headers.Set("tracestate", tc.TraceState)
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func parseTraceparent(traceparent string) *TraceContext {
	parts := strings.Split(traceparent, "-")
	if len(parts) != 4 {
		return nil
	}

	flags := byte(0)
	if len(parts[3]) >= 2 {
		b, _ := hex.DecodeString(parts[3][:2])
		if len(b) > 0 {
			flags = b[0]
		}
	}

	return &TraceContext{
		TraceID:    parts[1],
		SpanID:     parts[2],
		TraceFlags: flags,
	}
}

func generateTraceID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateSpanID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
