package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/josedab/waas/pkg/models"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AnalyticsRepository struct {
	db *pgxpool.Pool
}

func NewAnalyticsRepository(db *pgxpool.Pool) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

// RecordDeliveryMetric stores a delivery metric event
func (r *AnalyticsRepository) RecordDeliveryMetric(ctx context.Context, metric *models.DeliveryMetric) error {
	query := `
		INSERT INTO delivery_metrics (tenant_id, endpoint_id, delivery_id, status, http_status, latency_ms, attempt_number, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at`

	err := r.db.QueryRow(ctx, query,
		metric.TenantID,
		metric.EndpointID,
		metric.DeliveryID,
		metric.Status,
		metric.HTTPStatus,
		metric.LatencyMs,
		metric.AttemptNumber,
		metric.ErrorMessage,
	).Scan(&metric.ID, &metric.CreatedAt)

	return err
}

// GetDeliveryMetrics retrieves delivery metrics based on query parameters
func (r *AnalyticsRepository) GetDeliveryMetrics(ctx context.Context, query *models.MetricsQuery) ([]models.DeliveryMetric, error) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	// Build WHERE clause
	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIndex))
	args = append(args, query.TenantID)
	argIndex++

	if len(query.EndpointIDs) > 0 {
		placeholders := make([]string, len(query.EndpointIDs))
		for i, endpointID := range query.EndpointIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, endpointID)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("endpoint_id IN (%s)", strings.Join(placeholders, ",")))
	}

	conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIndex))
	args = append(args, query.StartDate)
	argIndex++

	conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIndex))
	args = append(args, query.EndDate)
	argIndex++

	if len(query.Statuses) > 0 {
		placeholders := make([]string, len(query.Statuses))
		for i, status := range query.Statuses {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, status)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("status IN (%s)", strings.Join(placeholders, ",")))
	}

	sqlQuery := fmt.Sprintf(`
		SELECT id, tenant_id, endpoint_id, delivery_id, status, http_status, latency_ms, attempt_number, error_message, created_at
		FROM delivery_metrics
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`,
		strings.Join(conditions, " AND "),
		argIndex,
		argIndex+1,
	)

	limit := query.Limit
	if limit <= 0 {
		limit = 100
	}
	args = append(args, limit, query.Offset)

	rows, err := r.db.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []models.DeliveryMetric
	for rows.Next() {
		var metric models.DeliveryMetric
		err := rows.Scan(
			&metric.ID,
			&metric.TenantID,
			&metric.EndpointID,
			&metric.DeliveryID,
			&metric.Status,
			&metric.HTTPStatus,
			&metric.LatencyMs,
			&metric.AttemptNumber,
			&metric.ErrorMessage,
			&metric.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, metric)
	}

	return metrics, rows.Err()
}

// UpsertHourlyMetric creates or updates hourly aggregated metrics
func (r *AnalyticsRepository) UpsertHourlyMetric(ctx context.Context, metric *models.HourlyMetric) error {
	query := `
		INSERT INTO hourly_metrics (
			tenant_id, endpoint_id, hour_timestamp, total_deliveries, successful_deliveries, 
			failed_deliveries, retrying_deliveries, avg_latency_ms, p95_latency_ms, p99_latency_ms, total_retries
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (tenant_id, COALESCE(endpoint_id, '00000000-0000-0000-0000-000000000000'::UUID), hour_timestamp)
		DO UPDATE SET
			total_deliveries = EXCLUDED.total_deliveries,
			successful_deliveries = EXCLUDED.successful_deliveries,
			failed_deliveries = EXCLUDED.failed_deliveries,
			retrying_deliveries = EXCLUDED.retrying_deliveries,
			avg_latency_ms = EXCLUDED.avg_latency_ms,
			p95_latency_ms = EXCLUDED.p95_latency_ms,
			p99_latency_ms = EXCLUDED.p99_latency_ms,
			total_retries = EXCLUDED.total_retries,
			updated_at = NOW()
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRow(ctx, query,
		metric.TenantID,
		metric.EndpointID,
		metric.HourTimestamp,
		metric.TotalDeliveries,
		metric.SuccessfulDeliveries,
		metric.FailedDeliveries,
		metric.RetryingDeliveries,
		metric.AvgLatencyMs,
		metric.P95LatencyMs,
		metric.P99LatencyMs,
		metric.TotalRetries,
	).Scan(&metric.ID, &metric.CreatedAt, &metric.UpdatedAt)

	return err
}

// GetHourlyMetrics retrieves hourly aggregated metrics
func (r *AnalyticsRepository) GetHourlyMetrics(ctx context.Context, query *models.MetricsQuery) ([]models.HourlyMetric, error) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIndex))
	args = append(args, query.TenantID)
	argIndex++

	if len(query.EndpointIDs) > 0 {
		placeholders := make([]string, len(query.EndpointIDs))
		for i, endpointID := range query.EndpointIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, endpointID)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("endpoint_id IN (%s)", strings.Join(placeholders, ",")))
	}

	conditions = append(conditions, fmt.Sprintf("hour_timestamp >= $%d", argIndex))
	args = append(args, query.StartDate)
	argIndex++

	conditions = append(conditions, fmt.Sprintf("hour_timestamp <= $%d", argIndex))
	args = append(args, query.EndDate)
	argIndex++

	sqlQuery := fmt.Sprintf(`
		SELECT id, tenant_id, endpoint_id, hour_timestamp, total_deliveries, successful_deliveries,
			   failed_deliveries, retrying_deliveries, avg_latency_ms, p95_latency_ms, p99_latency_ms,
			   total_retries, created_at, updated_at
		FROM hourly_metrics
		WHERE %s
		ORDER BY hour_timestamp DESC
		LIMIT $%d OFFSET $%d`,
		strings.Join(conditions, " AND "),
		argIndex,
		argIndex+1,
	)

	limit := query.Limit
	if limit <= 0 {
		limit = 168 // Default to 1 week of hourly data
	}
	args = append(args, limit, query.Offset)

	rows, err := r.db.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []models.HourlyMetric
	for rows.Next() {
		var metric models.HourlyMetric
		err := rows.Scan(
			&metric.ID,
			&metric.TenantID,
			&metric.EndpointID,
			&metric.HourTimestamp,
			&metric.TotalDeliveries,
			&metric.SuccessfulDeliveries,
			&metric.FailedDeliveries,
			&metric.RetryingDeliveries,
			&metric.AvgLatencyMs,
			&metric.P95LatencyMs,
			&metric.P99LatencyMs,
			&metric.TotalRetries,
			&metric.CreatedAt,
			&metric.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, metric)
	}

	return metrics, rows.Err()
}

// RecordRealtimeMetric stores a real-time metric for WebSocket updates
func (r *AnalyticsRepository) RecordRealtimeMetric(ctx context.Context, metric *models.RealtimeMetric) error {
	query := `
		INSERT INTO realtime_metrics (tenant_id, metric_type, metric_value, metadata)
		VALUES ($1, $2, $3, $4)
		RETURNING id, timestamp`

	err := r.db.QueryRow(ctx, query,
		metric.TenantID,
		metric.MetricType,
		metric.MetricValue,
		metric.Metadata,
	).Scan(&metric.ID, &metric.Timestamp)

	return err
}

// GetRealtimeMetrics retrieves recent real-time metrics
func (r *AnalyticsRepository) GetRealtimeMetrics(ctx context.Context, tenantID uuid.UUID, metricType string, since time.Time) ([]models.RealtimeMetric, error) {
	query := `
		SELECT id, tenant_id, metric_type, metric_value, timestamp, metadata
		FROM realtime_metrics
		WHERE tenant_id = $1 AND metric_type = $2 AND timestamp >= $3
		ORDER BY timestamp DESC
		LIMIT 100`

	rows, err := r.db.Query(ctx, query, tenantID, metricType, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []models.RealtimeMetric
	for rows.Next() {
		var metric models.RealtimeMetric
		err := rows.Scan(
			&metric.ID,
			&metric.TenantID,
			&metric.MetricType,
			&metric.MetricValue,
			&metric.Timestamp,
			&metric.Metadata,
		)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, metric)
	}

	return metrics, rows.Err()
}

// GetDashboardMetrics retrieves summary metrics for the dashboard
func (r *AnalyticsRepository) GetDashboardMetrics(ctx context.Context, tenantID uuid.UUID, timeWindow time.Duration) (*models.DashboardMetrics, error) {
	since := time.Now().Add(-timeWindow)

	// Get delivery rate (deliveries per minute)
	var deliveryRate float64
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(COUNT(*)::float / EXTRACT(EPOCH FROM (MAX(created_at) - MIN(created_at))) * 60, 0)
		FROM delivery_metrics
		WHERE tenant_id = $1 AND created_at >= $2`,
		tenantID, since,
	).Scan(&deliveryRate)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// Get success rate
	var successRate float64
	err = r.db.QueryRow(ctx, `
		SELECT CASE 
			WHEN COUNT(*) = 0 THEN 0
			ELSE COUNT(*) FILTER (WHERE status = 'success')::float / COUNT(*) * 100
		END
		FROM delivery_metrics
		WHERE tenant_id = $1 AND created_at >= $2`,
		tenantID, since,
	).Scan(&successRate)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// Get average latency
	var avgLatency float64
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(AVG(latency_ms), 0)
		FROM delivery_metrics
		WHERE tenant_id = $1 AND created_at >= $2 AND status = 'success'`,
		tenantID, since,
	).Scan(&avgLatency)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// Get active endpoints count
	var activeEndpoints int
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(DISTINCT endpoint_id)
		FROM delivery_metrics
		WHERE tenant_id = $1 AND created_at >= $2`,
		tenantID, since,
	).Scan(&activeEndpoints)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	return &models.DashboardMetrics{
		DeliveryRate:    deliveryRate,
		SuccessRate:     successRate,
		AvgLatency:      avgLatency,
		ActiveEndpoints: activeEndpoints,
		QueueSize:       0, // This would be populated from queue metrics
		RecentAlerts:    []models.AlertHistory{},
		TopEndpoints:    []models.EndpointMetrics{},
	}, nil
}

// GetMetricsSummary calculates summary statistics for a query
func (r *AnalyticsRepository) GetMetricsSummary(ctx context.Context, query *models.MetricsQuery) (*models.MetricsSummary, error) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIndex))
	args = append(args, query.TenantID)
	argIndex++

	if len(query.EndpointIDs) > 0 {
		placeholders := make([]string, len(query.EndpointIDs))
		for i, endpointID := range query.EndpointIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, endpointID)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("endpoint_id IN (%s)", strings.Join(placeholders, ",")))
	}

	conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIndex))
	args = append(args, query.StartDate)
	argIndex++

	conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIndex))
	args = append(args, query.EndDate)
	argIndex++

	sqlQuery := fmt.Sprintf(`
		SELECT 
			COUNT(*) as total_deliveries,
			COUNT(*) FILTER (WHERE status = 'success') as successful_deliveries,
			COUNT(*) FILTER (WHERE status = 'failed') as failed_deliveries,
			CASE 
				WHEN COUNT(*) = 0 THEN 0
				ELSE COUNT(*) FILTER (WHERE status = 'success')::float / COUNT(*) * 100
			END as success_rate,
			COALESCE(AVG(latency_ms) FILTER (WHERE status = 'success'), 0) as avg_latency_ms,
			COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY latency_ms) FILTER (WHERE status = 'success'), 0) as p95_latency_ms,
			COALESCE(PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY latency_ms) FILTER (WHERE status = 'success'), 0) as p99_latency_ms
		FROM delivery_metrics
		WHERE %s`,
		strings.Join(conditions, " AND "),
	)

	var summary models.MetricsSummary
	err := r.db.QueryRow(ctx, sqlQuery, args...).Scan(
		&summary.TotalDeliveries,
		&summary.SuccessfulDeliveries,
		&summary.FailedDeliveries,
		&summary.SuccessRate,
		&summary.AvgLatencyMs,
		&summary.P95LatencyMs,
		&summary.P99LatencyMs,
	)

	return &summary, err
}

// CleanupOldMetrics removes old metrics data to manage storage
func (r *AnalyticsRepository) CleanupOldMetrics(ctx context.Context, retentionDays int) error {
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

	// Clean up old delivery metrics (keep raw data for shorter period)
	_, err := r.db.Exec(ctx, `
		DELETE FROM delivery_metrics 
		WHERE created_at < $1`,
		cutoffDate,
	)
	if err != nil {
		return err
	}

	// Clean up old real-time metrics (keep only 24 hours)
	_, err = r.db.Exec(ctx, `
		DELETE FROM realtime_metrics 
		WHERE timestamp < NOW() - INTERVAL '24 hours'`,
	)

	return err
}
