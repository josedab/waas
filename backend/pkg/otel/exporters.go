package otel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// OTLPExporter exports telemetry to an OTLP endpoint
type OTLPExporter struct {
	tracesEndpoint  string
	metricsEndpoint string
	logsEndpoint    string
	headers         map[string]string
	client          *http.Client
	mu              sync.RWMutex
}

// NewOTLPExporter creates a new OTLP exporter
func NewOTLPExporter(endpoint string, headers map[string]string) *OTLPExporter {
	return &OTLPExporter{
		tracesEndpoint:  endpoint + "/v1/traces",
		metricsEndpoint: endpoint + "/v1/metrics",
		logsEndpoint:    endpoint + "/v1/logs",
		headers:         headers,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ExportSpans exports spans to OTLP endpoint
func (e *OTLPExporter) ExportSpans(ctx context.Context, spans []*SpanData) error {
	if len(spans) == 0 {
		return nil
	}

	// Convert to OTLP format
	payload := e.convertSpansToOTLP(spans)
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal spans: %w", err)
	}

	return e.send(ctx, e.tracesEndpoint, data)
}

// ExportMetrics exports metrics to OTLP endpoint
func (e *OTLPExporter) ExportMetrics(ctx context.Context, metrics []*MetricData) error {
	if len(metrics) == 0 {
		return nil
	}

	// Convert to OTLP format
	payload := e.convertMetricsToOTLP(metrics)
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	return e.send(ctx, e.metricsEndpoint, data)
}

// Shutdown shuts down the exporter
func (e *OTLPExporter) Shutdown(ctx context.Context) error {
	e.client.CloseIdleConnections()
	return nil
}

func (e *OTLPExporter) send(ctx context.Context, endpoint string, data []byte) error {
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range e.headers {
		req.Header.Set(k, v)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("OTLP export failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (e *OTLPExporter) convertSpansToOTLP(spans []*SpanData) map[string]any {
	resourceSpans := make([]map[string]any, 0)

	// Group spans by service
	byService := make(map[string][]*SpanData)
	for _, span := range spans {
		byService[span.ServiceName] = append(byService[span.ServiceName], span)
	}

	for serviceName, serviceSpans := range byService {
		scopeSpans := make([]map[string]any, 0, len(serviceSpans))
		for _, span := range serviceSpans {
			otlpSpan := map[string]any{
				"traceId":           span.TraceID,
				"spanId":            span.SpanID,
				"name":              span.OperationName,
				"startTimeUnixNano": span.StartTime.UnixNano(),
				"endTimeUnixNano":   span.EndTime.UnixNano(),
				"status": map[string]any{
					"code":    span.Status.Code,
					"message": span.Status.Description,
				},
			}

			if span.ParentSpanID != "" {
				otlpSpan["parentSpanId"] = span.ParentSpanID
			}

			// Convert attributes
			attrs := make([]map[string]any, 0, len(span.Attributes))
			for k, v := range span.Attributes {
				attrs = append(attrs, map[string]any{
					"key":   k,
					"value": convertValue(v),
				})
			}
			otlpSpan["attributes"] = attrs

			// Convert events
			events := make([]map[string]any, 0, len(span.Events))
			for _, evt := range span.Events {
				evtAttrs := make([]map[string]any, 0, len(evt.Attributes))
				for k, v := range evt.Attributes {
					evtAttrs = append(evtAttrs, map[string]any{
						"key":   k,
						"value": convertValue(v),
					})
				}
				events = append(events, map[string]any{
					"name":         evt.Name,
					"timeUnixNano": evt.Timestamp.UnixNano(),
					"attributes":   evtAttrs,
				})
			}
			otlpSpan["events"] = events

			scopeSpans = append(scopeSpans, map[string]any{
				"span": otlpSpan,
			})
		}

		resourceSpans = append(resourceSpans, map[string]any{
			"resource": map[string]any{
				"attributes": []map[string]any{
					{"key": "service.name", "value": map[string]any{"stringValue": serviceName}},
				},
			},
			"scopeSpans": []map[string]any{
				{
					"scope": map[string]any{
						"name":    "waas-otel",
						"version": "1.0.0",
					},
					"spans": scopeSpans,
				},
			},
		})
	}

	return map[string]any{"resourceSpans": resourceSpans}
}

func (e *OTLPExporter) convertMetricsToOTLP(metrics []*MetricData) map[string]any {
	scopeMetrics := make([]map[string]any, 0, len(metrics))

	for _, metric := range metrics {
		attrs := make([]map[string]any, 0, len(metric.Attributes))
		for k, v := range metric.Attributes {
			attrs = append(attrs, map[string]any{
				"key":   k,
				"value": map[string]any{"stringValue": v},
			})
		}

		var dataPoints []map[string]any
		dataPoint := map[string]any{
			"timeUnixNano": metric.Timestamp.UnixNano(),
			"attributes":   attrs,
		}

		switch metric.Type {
		case MetricCounter:
			dataPoint["asInt"] = int64(metric.Value)
		case MetricGauge, MetricHistogram:
			dataPoint["asDouble"] = metric.Value
		}
		dataPoints = append(dataPoints, dataPoint)

		metricData := map[string]any{
			"name":        metric.Name,
			"description": metric.Description,
			"unit":        metric.Unit,
		}

		switch metric.Type {
		case MetricCounter:
			metricData["sum"] = map[string]any{
				"dataPoints":             dataPoints,
				"aggregationTemporality": 2, // Cumulative
				"isMonotonic":            true,
			}
		case MetricGauge:
			metricData["gauge"] = map[string]any{
				"dataPoints": dataPoints,
			}
		case MetricHistogram:
			metricData["histogram"] = map[string]any{
				"dataPoints":             dataPoints,
				"aggregationTemporality": 2,
			}
		}

		scopeMetrics = append(scopeMetrics, metricData)
	}

	return map[string]any{
		"resourceMetrics": []map[string]any{
			{
				"scopeMetrics": []map[string]any{
					{
						"scope": map[string]any{
							"name":    "waas-otel",
							"version": "1.0.0",
						},
						"metrics": scopeMetrics,
					},
				},
			},
		},
	}
}

func convertValue(v any) map[string]any {
	switch val := v.(type) {
	case string:
		return map[string]any{"stringValue": val}
	case int:
		return map[string]any{"intValue": int64(val)}
	case int64:
		return map[string]any{"intValue": val}
	case float64:
		return map[string]any{"doubleValue": val}
	case bool:
		return map[string]any{"boolValue": val}
	default:
		return map[string]any{"stringValue": fmt.Sprintf("%v", val)}
	}
}

// StdoutExporter exports telemetry to stdout (for debugging)
type StdoutExporter struct {
	mu sync.Mutex
}

// NewStdoutExporter creates a new stdout exporter
func NewStdoutExporter() *StdoutExporter {
	return &StdoutExporter{}
}

// ExportSpans exports spans to stdout
func (e *StdoutExporter) ExportSpans(ctx context.Context, spans []*SpanData) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, span := range spans {
		data, _ := json.MarshalIndent(span, "", "  ")
		fmt.Printf("[SPAN] %s\n", data)
	}
	return nil
}

// ExportMetrics exports metrics to stdout
func (e *StdoutExporter) ExportMetrics(ctx context.Context, metrics []*MetricData) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, metric := range metrics {
		data, _ := json.MarshalIndent(metric, "", "  ")
		fmt.Printf("[METRIC] %s\n", data)
	}
	return nil
}

// Shutdown shuts down the exporter
func (e *StdoutExporter) Shutdown(ctx context.Context) error {
	return nil
}

// PrometheusExporter exposes metrics in Prometheus format
type PrometheusExporter struct {
	mu       sync.RWMutex
	metrics  map[string]*MetricData
	counters map[string]float64
}

// NewPrometheusExporter creates a new Prometheus exporter
func NewPrometheusExporter() *PrometheusExporter {
	return &PrometheusExporter{
		metrics:  make(map[string]*MetricData),
		counters: make(map[string]float64),
	}
}

// ExportSpans is a no-op for Prometheus (Prometheus doesn't support traces)
func (e *PrometheusExporter) ExportSpans(ctx context.Context, spans []*SpanData) error {
	return nil
}

// ExportMetrics stores metrics for Prometheus scraping
func (e *PrometheusExporter) ExportMetrics(ctx context.Context, metrics []*MetricData) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, metric := range metrics {
		key := metric.Name
		for k, v := range metric.Attributes {
			key += "_" + k + "=" + v
		}

		switch metric.Type {
		case MetricCounter:
			e.counters[key] += metric.Value
			updated := *metric
			updated.Value = e.counters[key]
			e.metrics[key] = &updated
		default:
			e.metrics[key] = metric
		}
	}
	return nil
}

// Shutdown shuts down the exporter
func (e *PrometheusExporter) Shutdown(ctx context.Context) error {
	return nil
}

// ServeHTTP handles Prometheus scrape requests
func (e *PrometheusExporter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	for _, metric := range e.metrics {
		name := sanitizeMetricName(metric.Name)
		labels := ""
		if len(metric.Attributes) > 0 {
			labelParts := make([]string, 0, len(metric.Attributes))
			for k, v := range metric.Attributes {
				labelParts = append(labelParts, fmt.Sprintf(`%s="%s"`, sanitizeMetricName(k), v))
			}
			labels = "{" + joinStrings(labelParts, ",") + "}"
		}

		var typeStr string
		switch metric.Type {
		case MetricCounter:
			typeStr = "counter"
		case MetricGauge:
			typeStr = "gauge"
		case MetricHistogram:
			typeStr = "histogram"
		}

		fmt.Fprintf(w, "# HELP %s %s\n", name, metric.Description)
		fmt.Fprintf(w, "# TYPE %s %s\n", name, typeStr)
		fmt.Fprintf(w, "%s%s %g\n", name, labels, metric.Value)
	}
}

func sanitizeMetricName(name string) string {
	result := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			result = append(result, c)
		} else {
			result = append(result, '_')
		}
	}
	return string(result)
}

func joinStrings(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += sep + parts[i]
	}
	return result
}
