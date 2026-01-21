package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Repository defines the interface for observability storage
type Repository interface {
	// Span operations
	SaveSpan(ctx context.Context, span *WebhookSpan) error
	GetSpan(ctx context.Context, tenantID, spanID string) (*WebhookSpan, error)
	GetSpansByTrace(ctx context.Context, tenantID, traceID string) ([]WebhookSpan, error)
	
	// Trace operations
	GetTrace(ctx context.Context, tenantID, traceID string) (*Trace, error)
	SearchTraces(ctx context.Context, query *TraceSearchQuery) (*TraceSearchResult, error)
	
	// Export config operations
	SaveExportConfig(ctx context.Context, config *OTelExportConfig) error
	GetExportConfig(ctx context.Context, tenantID, configID string) (*OTelExportConfig, error)
	ListExportConfigs(ctx context.Context, tenantID string) ([]OTelExportConfig, error)
	DeleteExportConfig(ctx context.Context, tenantID, configID string) error
	
	// Metrics
	GetTraceMetrics(ctx context.Context, tenantID string, start, end time.Time) (*TraceMetrics, error)
	GetServiceMap(ctx context.Context, tenantID string, start, end time.Time) (*ServiceMap, error)
}

// PostgresRepository implements Repository for PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) SaveSpan(ctx context.Context, span *WebhookSpan) error {
	attributesJSON, _ := json.Marshal(span.Attributes)
	eventsJSON, _ := json.Marshal(span.Events)
	linksJSON, _ := json.Marshal(span.Links)

	query := `
		INSERT INTO webhook_spans (
			id, trace_id, span_id, parent_span_id, tenant_id, webhook_id, endpoint_id,
			delivery_id, operation_name, service_name, kind, status, status_message,
			start_time, end_time, duration_ms, attributes, events, links, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
		) ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			status_message = EXCLUDED.status_message,
			end_time = EXCLUDED.end_time,
			duration_ms = EXCLUDED.duration_ms,
			attributes = EXCLUDED.attributes,
			events = EXCLUDED.events`

	_, err := r.db.ExecContext(ctx, query,
		span.ID, span.TraceID, span.SpanID, span.ParentSpanID, span.TenantID,
		span.WebhookID, span.EndpointID, span.DeliveryID, span.OperationName,
		span.ServiceName, span.Kind, span.Status, span.StatusMessage,
		span.StartTime, span.EndTime, span.DurationMs, attributesJSON,
		eventsJSON, linksJSON, span.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) GetSpan(ctx context.Context, tenantID, spanID string) (*WebhookSpan, error) {
	query := `
		SELECT id, trace_id, span_id, parent_span_id, tenant_id, webhook_id, endpoint_id,
			delivery_id, operation_name, service_name, kind, status, status_message,
			start_time, end_time, duration_ms, attributes, events, links, created_at
		FROM webhook_spans
		WHERE tenant_id = $1 AND span_id = $2`

	var span WebhookSpan
	var attributesJSON, eventsJSON, linksJSON []byte

	err := r.db.QueryRowContext(ctx, query, tenantID, spanID).Scan(
		&span.ID, &span.TraceID, &span.SpanID, &span.ParentSpanID, &span.TenantID,
		&span.WebhookID, &span.EndpointID, &span.DeliveryID, &span.OperationName,
		&span.ServiceName, &span.Kind, &span.Status, &span.StatusMessage,
		&span.StartTime, &span.EndTime, &span.DurationMs, &attributesJSON,
		&eventsJSON, &linksJSON, &span.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(attributesJSON, &span.Attributes)
	json.Unmarshal(eventsJSON, &span.Events)
	json.Unmarshal(linksJSON, &span.Links)

	return &span, nil
}

func (r *PostgresRepository) GetSpansByTrace(ctx context.Context, tenantID, traceID string) ([]WebhookSpan, error) {
	query := `
		SELECT id, trace_id, span_id, parent_span_id, tenant_id, webhook_id, endpoint_id,
			delivery_id, operation_name, service_name, kind, status, status_message,
			start_time, end_time, duration_ms, attributes, events, links, created_at
		FROM webhook_spans
		WHERE tenant_id = $1 AND trace_id = $2
		ORDER BY start_time ASC`

	rows, err := r.db.QueryContext(ctx, query, tenantID, traceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var spans []WebhookSpan
	for rows.Next() {
		var span WebhookSpan
		var attributesJSON, eventsJSON, linksJSON []byte

		err := rows.Scan(
			&span.ID, &span.TraceID, &span.SpanID, &span.ParentSpanID, &span.TenantID,
			&span.WebhookID, &span.EndpointID, &span.DeliveryID, &span.OperationName,
			&span.ServiceName, &span.Kind, &span.Status, &span.StatusMessage,
			&span.StartTime, &span.EndTime, &span.DurationMs, &attributesJSON,
			&eventsJSON, &linksJSON, &span.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		json.Unmarshal(attributesJSON, &span.Attributes)
		json.Unmarshal(eventsJSON, &span.Events)
		json.Unmarshal(linksJSON, &span.Links)

		spans = append(spans, span)
	}

	return spans, nil
}

func (r *PostgresRepository) GetTrace(ctx context.Context, tenantID, traceID string) (*Trace, error) {
	spans, err := r.GetSpansByTrace(ctx, tenantID, traceID)
	if err != nil {
		return nil, err
	}

	if len(spans) == 0 {
		return nil, fmt.Errorf("trace not found")
	}

	trace := &Trace{
		TraceID:   traceID,
		TenantID:  tenantID,
		Spans:     spans,
		SpanCount: len(spans),
	}

	serviceSet := make(map[string]bool)
	for i := range spans {
		span := &spans[i]
		serviceSet[span.ServiceName] = true

		if span.Status == SpanStatusError {
			trace.ErrorCount++
		}

		if span.ParentSpanID == "" {
			trace.RootSpan = span
		}

		if trace.StartTime.IsZero() || span.StartTime.Before(trace.StartTime) {
			trace.StartTime = span.StartTime
		}
		if span.EndTime.After(trace.EndTime) {
			trace.EndTime = span.EndTime
		}
	}

	for svc := range serviceSet {
		trace.Services = append(trace.Services, svc)
	}

	trace.DurationMs = trace.EndTime.Sub(trace.StartTime).Milliseconds()

	return trace, nil
}

func (r *PostgresRepository) SearchTraces(ctx context.Context, query *TraceSearchQuery) (*TraceSearchResult, error) {
	baseQuery := `
		WITH trace_summaries AS (
			SELECT DISTINCT ON (trace_id)
				trace_id,
				service_name as root_service,
				operation_name as root_operation,
				start_time,
				duration_ms,
				status
			FROM webhook_spans
			WHERE tenant_id = $1
				AND parent_span_id = ''
				AND start_time >= $2
				AND start_time <= $3`

	args := []interface{}{query.TenantID, query.StartTime, query.EndTime}
	argIdx := 4

	if query.ServiceName != "" {
		baseQuery += fmt.Sprintf(" AND service_name = $%d", argIdx)
		args = append(args, query.ServiceName)
		argIdx++
	}

	if query.MinDuration > 0 {
		baseQuery += fmt.Sprintf(" AND duration_ms >= $%d", argIdx)
		args = append(args, query.MinDuration)
		argIdx++
	}

	if query.MaxDuration > 0 {
		baseQuery += fmt.Sprintf(" AND duration_ms <= $%d", argIdx)
		args = append(args, query.MaxDuration)
		argIdx++
	}

	baseQuery += `
			ORDER BY trace_id, start_time
		)
		SELECT trace_id, root_service, root_operation, start_time, duration_ms,
			(SELECT COUNT(*) FROM webhook_spans ws WHERE ws.trace_id = ts.trace_id AND ws.tenant_id = $1) as span_count,
			(SELECT COUNT(*) FROM webhook_spans ws WHERE ws.trace_id = ts.trace_id AND ws.tenant_id = $1 AND ws.status = 'error') as error_count,
			status
		FROM trace_summaries ts
		ORDER BY start_time DESC
		LIMIT $` + fmt.Sprintf("%d", argIdx) + ` OFFSET $` + fmt.Sprintf("%d", argIdx+1)

	args = append(args, query.Limit, query.Offset)

	rows, err := r.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var traces []TraceSummary
	for rows.Next() {
		var ts TraceSummary
		err := rows.Scan(&ts.TraceID, &ts.RootService, &ts.RootOperation,
			&ts.StartTime, &ts.DurationMs, &ts.SpanCount, &ts.ErrorCount, &ts.Status)
		if err != nil {
			return nil, err
		}
		traces = append(traces, ts)
	}

	// Get total count
	countQuery := `
		SELECT COUNT(DISTINCT trace_id)
		FROM webhook_spans
		WHERE tenant_id = $1 AND start_time >= $2 AND start_time <= $3`
	
	var totalCount int
	r.db.QueryRowContext(ctx, countQuery, query.TenantID, query.StartTime, query.EndTime).Scan(&totalCount)

	return &TraceSearchResult{
		Traces:     traces,
		TotalCount: totalCount,
		HasMore:    query.Offset+len(traces) < totalCount,
	}, nil
}

func (r *PostgresRepository) SaveExportConfig(ctx context.Context, config *OTelExportConfig) error {
	headersJSON, _ := json.Marshal(config.Headers)
	samplingJSON, _ := json.Marshal(config.Sampling)

	query := `
		INSERT INTO otel_export_configs (
			id, tenant_id, name, enabled, protocol, endpoint, headers,
			sampling, batch_size, timeout_seconds, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			enabled = EXCLUDED.enabled,
			protocol = EXCLUDED.protocol,
			endpoint = EXCLUDED.endpoint,
			headers = EXCLUDED.headers,
			sampling = EXCLUDED.sampling,
			batch_size = EXCLUDED.batch_size,
			timeout_seconds = EXCLUDED.timeout_seconds,
			updated_at = EXCLUDED.updated_at`

	_, err := r.db.ExecContext(ctx, query,
		config.ID, config.TenantID, config.Name, config.Enabled, config.Protocol,
		config.Endpoint, headersJSON, samplingJSON, config.BatchSize,
		config.Timeout, config.CreatedAt, config.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetExportConfig(ctx context.Context, tenantID, configID string) (*OTelExportConfig, error) {
	query := `
		SELECT id, tenant_id, name, enabled, protocol, endpoint, headers,
			sampling, batch_size, timeout_seconds, created_at, updated_at
		FROM otel_export_configs
		WHERE tenant_id = $1 AND id = $2`

	var config OTelExportConfig
	var headersJSON, samplingJSON []byte

	err := r.db.QueryRowContext(ctx, query, tenantID, configID).Scan(
		&config.ID, &config.TenantID, &config.Name, &config.Enabled,
		&config.Protocol, &config.Endpoint, &headersJSON, &samplingJSON,
		&config.BatchSize, &config.Timeout, &config.CreatedAt, &config.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(headersJSON, &config.Headers)
	json.Unmarshal(samplingJSON, &config.Sampling)

	return &config, nil
}

func (r *PostgresRepository) ListExportConfigs(ctx context.Context, tenantID string) ([]OTelExportConfig, error) {
	query := `
		SELECT id, tenant_id, name, enabled, protocol, endpoint, headers,
			sampling, batch_size, timeout_seconds, created_at, updated_at
		FROM otel_export_configs
		WHERE tenant_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []OTelExportConfig
	for rows.Next() {
		var config OTelExportConfig
		var headersJSON, samplingJSON []byte

		err := rows.Scan(
			&config.ID, &config.TenantID, &config.Name, &config.Enabled,
			&config.Protocol, &config.Endpoint, &headersJSON, &samplingJSON,
			&config.BatchSize, &config.Timeout, &config.CreatedAt, &config.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		json.Unmarshal(headersJSON, &config.Headers)
		json.Unmarshal(samplingJSON, &config.Sampling)

		configs = append(configs, config)
	}

	return configs, nil
}

func (r *PostgresRepository) DeleteExportConfig(ctx context.Context, tenantID, configID string) error {
	query := `DELETE FROM otel_export_configs WHERE tenant_id = $1 AND id = $2`
	_, err := r.db.ExecContext(ctx, query, tenantID, configID)
	return err
}

func (r *PostgresRepository) GetTraceMetrics(ctx context.Context, tenantID string, start, end time.Time) (*TraceMetrics, error) {
	metrics := &TraceMetrics{
		TenantID:    tenantID,
		Period:      fmt.Sprintf("%s to %s", start.Format(time.RFC3339), end.Format(time.RFC3339)),
		ByService:   make(map[string]int64),
		ByOperation: make(map[string]int64),
	}

	// Get aggregate metrics
	query := `
		SELECT 
			COUNT(DISTINCT trace_id) as total_traces,
			COUNT(*) as total_spans,
			AVG(duration_ms) as avg_duration,
			PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY duration_ms) as p50,
			PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_ms) as p95,
			PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY duration_ms) as p99,
			SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0) as error_rate
		FROM webhook_spans
		WHERE tenant_id = $1 AND start_time >= $2 AND start_time <= $3`

	err := r.db.QueryRowContext(ctx, query, tenantID, start, end).Scan(
		&metrics.TotalTraces, &metrics.TotalSpans, &metrics.AvgDuration,
		&metrics.P50Duration, &metrics.P95Duration, &metrics.P99Duration,
		&metrics.ErrorRate,
	)
	if err != nil {
		return nil, err
	}

	// Get by service
	serviceQuery := `
		SELECT service_name, COUNT(*) as count
		FROM webhook_spans
		WHERE tenant_id = $1 AND start_time >= $2 AND start_time <= $3
		GROUP BY service_name`

	rows, err := r.db.QueryContext(ctx, serviceQuery, tenantID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var count int64
		if err := rows.Scan(&name, &count); err != nil {
			continue
		}
		metrics.ByService[name] = count
	}

	return metrics, nil
}

func (r *PostgresRepository) GetServiceMap(ctx context.Context, tenantID string, start, end time.Time) (*ServiceMap, error) {
	serviceMap := &ServiceMap{
		TenantID:    tenantID,
		GeneratedAt: time.Now(),
	}

	// Get services
	serviceQuery := `
		SELECT service_name, COUNT(*) as span_count,
			AVG(duration_ms) as avg_latency,
			SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0) as error_rate
		FROM webhook_spans
		WHERE tenant_id = $1 AND start_time >= $2 AND start_time <= $3
		GROUP BY service_name`

	rows, err := r.db.QueryContext(ctx, serviceQuery, tenantID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var node ServiceNode
		if err := rows.Scan(&node.Name, &node.SpanCount, &node.AvgLatency, &node.ErrorRate); err != nil {
			continue
		}
		serviceMap.Services = append(serviceMap.Services, node)
	}

	// Get dependencies from parent-child relationships
	depQuery := `
		SELECT p.service_name as source, c.service_name as target,
			COUNT(*) as call_count,
			SUM(CASE WHEN c.status = 'error' THEN 1 ELSE 0 END) as error_count,
			AVG(c.duration_ms) as avg_latency,
			PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY c.duration_ms) as p99_latency
		FROM webhook_spans p
		JOIN webhook_spans c ON c.parent_span_id = p.span_id AND c.tenant_id = p.tenant_id
		WHERE p.tenant_id = $1 AND p.start_time >= $2 AND p.start_time <= $3
			AND p.service_name != c.service_name
		GROUP BY p.service_name, c.service_name`

	depRows, err := r.db.QueryContext(ctx, depQuery, tenantID, start, end)
	if err != nil {
		return nil, err
	}
	defer depRows.Close()

	for depRows.Next() {
		var dep ServiceDependency
		if err := depRows.Scan(&dep.Source, &dep.Target, &dep.CallCount,
			&dep.ErrorCount, &dep.AvgLatency, &dep.P99Latency); err != nil {
			continue
		}
		serviceMap.Dependencies = append(serviceMap.Dependencies, dep)
	}

	return serviceMap, nil
}
