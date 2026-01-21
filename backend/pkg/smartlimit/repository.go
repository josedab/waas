package smartlimit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Repository defines the interface for smart limit storage
type Repository interface {
	// Behavior operations
	SaveBehavior(ctx context.Context, behavior *EndpointBehavior) error
	GetBehavior(ctx context.Context, tenantID, endpointID string) (*EndpointBehavior, error)
	GetRecentBehaviors(ctx context.Context, tenantID, endpointID string, limit int) ([]EndpointBehavior, error)

	// Config operations
	SaveConfig(ctx context.Context, config *AdaptiveRateConfig) error
	GetConfig(ctx context.Context, tenantID, endpointID string) (*AdaptiveRateConfig, error)
	ListConfigs(ctx context.Context, tenantID string) ([]AdaptiveRateConfig, error)
	DeleteConfig(ctx context.Context, tenantID, endpointID string) error

	// State operations
	SaveState(ctx context.Context, state *RateLimitState) error
	GetState(ctx context.Context, tenantID, endpointID string) (*RateLimitState, error)

	// Learning data operations
	SaveLearningData(ctx context.Context, data *LearningDataPoint) error
	GetLearningData(ctx context.Context, tenantID, endpointID string, start, end time.Time) ([]LearningDataPoint, error)

	// Model operations
	SaveModel(ctx context.Context, model *PredictionModel) error
	GetActiveModel(ctx context.Context, tenantID, endpointID string) (*PredictionModel, error)

	// Event operations
	SaveEvent(ctx context.Context, event *RateLimitEvent) error
	GetEvents(ctx context.Context, tenantID, endpointID string, start, end time.Time) ([]RateLimitEvent, error)

	// Stats
	GetStats(ctx context.Context, tenantID string, start, end time.Time) (*SmartLimitStats, error)
}

// PostgresRepository implements Repository for PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) SaveBehavior(ctx context.Context, behavior *EndpointBehavior) error {
	statusCodesJSON, _ := json.Marshal(behavior.StatusCodes)
	hourlyJSON, _ := json.Marshal(behavior.HourlyPattern)
	dayOfWeekJSON, _ := json.Marshal(behavior.DayOfWeekPattern)

	query := `
		INSERT INTO endpoint_behaviors (
			id, tenant_id, endpoint_id, url, window_start, window_end,
			total_requests, success_count, rate_limit_count, timeout_count, error_count,
			avg_latency_ms, p50_latency_ms, p95_latency_ms, p99_latency_ms, max_latency_ms,
			avg_response_size, status_codes, hourly_pattern, day_of_week_pattern,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
		ON CONFLICT (id) DO UPDATE SET
			total_requests = EXCLUDED.total_requests,
			success_count = EXCLUDED.success_count,
			rate_limit_count = EXCLUDED.rate_limit_count,
			timeout_count = EXCLUDED.timeout_count,
			error_count = EXCLUDED.error_count,
			avg_latency_ms = EXCLUDED.avg_latency_ms,
			p50_latency_ms = EXCLUDED.p50_latency_ms,
			p95_latency_ms = EXCLUDED.p95_latency_ms,
			p99_latency_ms = EXCLUDED.p99_latency_ms,
			max_latency_ms = EXCLUDED.max_latency_ms,
			avg_response_size = EXCLUDED.avg_response_size,
			status_codes = EXCLUDED.status_codes,
			hourly_pattern = EXCLUDED.hourly_pattern,
			day_of_week_pattern = EXCLUDED.day_of_week_pattern,
			updated_at = EXCLUDED.updated_at`

	_, err := r.db.ExecContext(ctx, query,
		behavior.ID, behavior.TenantID, behavior.EndpointID, behavior.URL,
		behavior.WindowStart, behavior.WindowEnd,
		behavior.TotalRequests, behavior.SuccessCount, behavior.RateLimitCount,
		behavior.TimeoutCount, behavior.ErrorCount,
		behavior.AvgLatencyMs, behavior.P50LatencyMs, behavior.P95LatencyMs,
		behavior.P99LatencyMs, behavior.MaxLatencyMs, behavior.AvgResponseSize,
		statusCodesJSON, hourlyJSON, dayOfWeekJSON,
		behavior.CreatedAt, behavior.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetBehavior(ctx context.Context, tenantID, endpointID string) (*EndpointBehavior, error) {
	query := `
		SELECT id, tenant_id, endpoint_id, url, window_start, window_end,
			total_requests, success_count, rate_limit_count, timeout_count, error_count,
			avg_latency_ms, p50_latency_ms, p95_latency_ms, p99_latency_ms, max_latency_ms,
			avg_response_size, status_codes, hourly_pattern, day_of_week_pattern,
			created_at, updated_at
		FROM endpoint_behaviors
		WHERE tenant_id = $1 AND endpoint_id = $2
		ORDER BY window_end DESC
		LIMIT 1`

	var behavior EndpointBehavior
	var statusCodesJSON, hourlyJSON, dayOfWeekJSON []byte

	err := r.db.QueryRowContext(ctx, query, tenantID, endpointID).Scan(
		&behavior.ID, &behavior.TenantID, &behavior.EndpointID, &behavior.URL,
		&behavior.WindowStart, &behavior.WindowEnd,
		&behavior.TotalRequests, &behavior.SuccessCount, &behavior.RateLimitCount,
		&behavior.TimeoutCount, &behavior.ErrorCount,
		&behavior.AvgLatencyMs, &behavior.P50LatencyMs, &behavior.P95LatencyMs,
		&behavior.P99LatencyMs, &behavior.MaxLatencyMs, &behavior.AvgResponseSize,
		&statusCodesJSON, &hourlyJSON, &dayOfWeekJSON,
		&behavior.CreatedAt, &behavior.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(statusCodesJSON, &behavior.StatusCodes)
	json.Unmarshal(hourlyJSON, &behavior.HourlyPattern)
	json.Unmarshal(dayOfWeekJSON, &behavior.DayOfWeekPattern)

	return &behavior, nil
}

func (r *PostgresRepository) GetRecentBehaviors(ctx context.Context, tenantID, endpointID string, limit int) ([]EndpointBehavior, error) {
	query := `
		SELECT id, tenant_id, endpoint_id, url, window_start, window_end,
			total_requests, success_count, rate_limit_count, timeout_count, error_count,
			avg_latency_ms, p50_latency_ms, p95_latency_ms, p99_latency_ms, max_latency_ms,
			avg_response_size, status_codes, hourly_pattern, day_of_week_pattern,
			created_at, updated_at
		FROM endpoint_behaviors
		WHERE tenant_id = $1 AND endpoint_id = $2
		ORDER BY window_end DESC
		LIMIT $3`

	rows, err := r.db.QueryContext(ctx, query, tenantID, endpointID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var behaviors []EndpointBehavior
	for rows.Next() {
		var behavior EndpointBehavior
		var statusCodesJSON, hourlyJSON, dayOfWeekJSON []byte

		err := rows.Scan(
			&behavior.ID, &behavior.TenantID, &behavior.EndpointID, &behavior.URL,
			&behavior.WindowStart, &behavior.WindowEnd,
			&behavior.TotalRequests, &behavior.SuccessCount, &behavior.RateLimitCount,
			&behavior.TimeoutCount, &behavior.ErrorCount,
			&behavior.AvgLatencyMs, &behavior.P50LatencyMs, &behavior.P95LatencyMs,
			&behavior.P99LatencyMs, &behavior.MaxLatencyMs, &behavior.AvgResponseSize,
			&statusCodesJSON, &hourlyJSON, &dayOfWeekJSON,
			&behavior.CreatedAt, &behavior.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		json.Unmarshal(statusCodesJSON, &behavior.StatusCodes)
		json.Unmarshal(hourlyJSON, &behavior.HourlyPattern)
		json.Unmarshal(dayOfWeekJSON, &behavior.DayOfWeekPattern)

		behaviors = append(behaviors, behavior)
	}

	return behaviors, nil
}

func (r *PostgresRepository) SaveConfig(ctx context.Context, config *AdaptiveRateConfig) error {
	query := `
		INSERT INTO adaptive_rate_configs (
			id, tenant_id, endpoint_id, enabled, mode, base_rate_per_sec,
			min_rate_per_sec, max_rate_per_sec, burst_size, risk_threshold,
			backoff_factor, recovery_factor, window_seconds, learning_enabled,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (tenant_id, endpoint_id) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			mode = EXCLUDED.mode,
			base_rate_per_sec = EXCLUDED.base_rate_per_sec,
			min_rate_per_sec = EXCLUDED.min_rate_per_sec,
			max_rate_per_sec = EXCLUDED.max_rate_per_sec,
			burst_size = EXCLUDED.burst_size,
			risk_threshold = EXCLUDED.risk_threshold,
			backoff_factor = EXCLUDED.backoff_factor,
			recovery_factor = EXCLUDED.recovery_factor,
			window_seconds = EXCLUDED.window_seconds,
			learning_enabled = EXCLUDED.learning_enabled,
			updated_at = EXCLUDED.updated_at`

	_, err := r.db.ExecContext(ctx, query,
		config.ID, config.TenantID, config.EndpointID, config.Enabled, config.Mode,
		config.BaseRatePerSec, config.MinRatePerSec, config.MaxRatePerSec,
		config.BurstSize, config.RiskThreshold, config.BackoffFactor,
		config.RecoveryFactor, config.WindowSeconds, config.LearningEnabled,
		config.CreatedAt, config.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetConfig(ctx context.Context, tenantID, endpointID string) (*AdaptiveRateConfig, error) {
	query := `
		SELECT id, tenant_id, endpoint_id, enabled, mode, base_rate_per_sec,
			min_rate_per_sec, max_rate_per_sec, burst_size, risk_threshold,
			backoff_factor, recovery_factor, window_seconds, learning_enabled,
			created_at, updated_at
		FROM adaptive_rate_configs
		WHERE tenant_id = $1 AND endpoint_id = $2`

	var config AdaptiveRateConfig
	err := r.db.QueryRowContext(ctx, query, tenantID, endpointID).Scan(
		&config.ID, &config.TenantID, &config.EndpointID, &config.Enabled, &config.Mode,
		&config.BaseRatePerSec, &config.MinRatePerSec, &config.MaxRatePerSec,
		&config.BurstSize, &config.RiskThreshold, &config.BackoffFactor,
		&config.RecoveryFactor, &config.WindowSeconds, &config.LearningEnabled,
		&config.CreatedAt, &config.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func (r *PostgresRepository) ListConfigs(ctx context.Context, tenantID string) ([]AdaptiveRateConfig, error) {
	query := `
		SELECT id, tenant_id, endpoint_id, enabled, mode, base_rate_per_sec,
			min_rate_per_sec, max_rate_per_sec, burst_size, risk_threshold,
			backoff_factor, recovery_factor, window_seconds, learning_enabled,
			created_at, updated_at
		FROM adaptive_rate_configs
		WHERE tenant_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []AdaptiveRateConfig
	for rows.Next() {
		var config AdaptiveRateConfig
		err := rows.Scan(
			&config.ID, &config.TenantID, &config.EndpointID, &config.Enabled, &config.Mode,
			&config.BaseRatePerSec, &config.MinRatePerSec, &config.MaxRatePerSec,
			&config.BurstSize, &config.RiskThreshold, &config.BackoffFactor,
			&config.RecoveryFactor, &config.WindowSeconds, &config.LearningEnabled,
			&config.CreatedAt, &config.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}

	return configs, nil
}

func (r *PostgresRepository) DeleteConfig(ctx context.Context, tenantID, endpointID string) error {
	query := `DELETE FROM adaptive_rate_configs WHERE tenant_id = $1 AND endpoint_id = $2`
	_, err := r.db.ExecContext(ctx, query, tenantID, endpointID)
	return err
}

func (r *PostgresRepository) SaveState(ctx context.Context, state *RateLimitState) error {
	query := `
		INSERT INTO rate_limit_states (
			id, tenant_id, endpoint_id, current_rate, allowed_rate, window_start,
			request_count, rate_limit_hits, consecutive_ok, consecutive_fail,
			last_rate_limit_at, retry_after, cooldown, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (tenant_id, endpoint_id) DO UPDATE SET
			current_rate = EXCLUDED.current_rate,
			allowed_rate = EXCLUDED.allowed_rate,
			window_start = EXCLUDED.window_start,
			request_count = EXCLUDED.request_count,
			rate_limit_hits = EXCLUDED.rate_limit_hits,
			consecutive_ok = EXCLUDED.consecutive_ok,
			consecutive_fail = EXCLUDED.consecutive_fail,
			last_rate_limit_at = EXCLUDED.last_rate_limit_at,
			retry_after = EXCLUDED.retry_after,
			cooldown = EXCLUDED.cooldown,
			updated_at = EXCLUDED.updated_at`

	_, err := r.db.ExecContext(ctx, query,
		state.ID, state.TenantID, state.EndpointID, state.CurrentRate, state.AllowedRate,
		state.WindowStart, state.RequestCount, state.RateLimitHits,
		state.ConsecutiveOK, state.ConsecutiveFail, state.LastRateLimitAt,
		state.RetryAfter, state.Cooldown, state.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetState(ctx context.Context, tenantID, endpointID string) (*RateLimitState, error) {
	query := `
		SELECT id, tenant_id, endpoint_id, current_rate, allowed_rate, window_start,
			request_count, rate_limit_hits, consecutive_ok, consecutive_fail,
			last_rate_limit_at, retry_after, cooldown, updated_at
		FROM rate_limit_states
		WHERE tenant_id = $1 AND endpoint_id = $2`

	var state RateLimitState
	err := r.db.QueryRowContext(ctx, query, tenantID, endpointID).Scan(
		&state.ID, &state.TenantID, &state.EndpointID, &state.CurrentRate, &state.AllowedRate,
		&state.WindowStart, &state.RequestCount, &state.RateLimitHits,
		&state.ConsecutiveOK, &state.ConsecutiveFail, &state.LastRateLimitAt,
		&state.RetryAfter, &state.Cooldown, &state.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &state, nil
}

func (r *PostgresRepository) SaveLearningData(ctx context.Context, data *LearningDataPoint) error {
	query := `
		INSERT INTO rate_limit_learning_data (
			id, tenant_id, endpoint_id, timestamp, hour_of_day, day_of_week,
			request_rate, success_rate, avg_latency_ms, rate_limited, response_code
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := r.db.ExecContext(ctx, query,
		data.ID, data.TenantID, data.EndpointID, data.Timestamp,
		data.HourOfDay, data.DayOfWeek, data.RequestRate, data.SuccessRate,
		data.AvgLatency, data.RateLimited, data.ResponseCode,
	)
	return err
}

func (r *PostgresRepository) GetLearningData(ctx context.Context, tenantID, endpointID string, start, end time.Time) ([]LearningDataPoint, error) {
	query := `
		SELECT id, tenant_id, endpoint_id, timestamp, hour_of_day, day_of_week,
			request_rate, success_rate, avg_latency_ms, rate_limited, response_code
		FROM rate_limit_learning_data
		WHERE tenant_id = $1 AND endpoint_id = $2 AND timestamp >= $3 AND timestamp <= $4
		ORDER BY timestamp ASC`

	rows, err := r.db.QueryContext(ctx, query, tenantID, endpointID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []LearningDataPoint
	for rows.Next() {
		var point LearningDataPoint
		err := rows.Scan(
			&point.ID, &point.TenantID, &point.EndpointID, &point.Timestamp,
			&point.HourOfDay, &point.DayOfWeek, &point.RequestRate, &point.SuccessRate,
			&point.AvgLatency, &point.RateLimited, &point.ResponseCode,
		)
		if err != nil {
			return nil, err
		}
		data = append(data, point)
	}

	return data, nil
}

func (r *PostgresRepository) SaveModel(ctx context.Context, model *PredictionModel) error {
	weightsJSON, _ := json.Marshal(model.Weights)
	coefficientsJSON, _ := json.Marshal(model.Coefficients)
	featuresJSON, _ := json.Marshal(model.Features)

	query := `
		INSERT INTO prediction_models (
			id, tenant_id, endpoint_id, model_type, version, weights,
			coefficients, features, accuracy, trained_at, data_point_count,
			is_active, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	_, err := r.db.ExecContext(ctx, query,
		model.ID, model.TenantID, model.EndpointID, model.ModelType,
		model.Version, weightsJSON, coefficientsJSON, featuresJSON,
		model.Accuracy, model.TrainedAt, model.DataPointCount,
		model.IsActive, model.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) GetActiveModel(ctx context.Context, tenantID, endpointID string) (*PredictionModel, error) {
	query := `
		SELECT id, tenant_id, endpoint_id, model_type, version, weights,
			coefficients, features, accuracy, trained_at, data_point_count,
			is_active, created_at
		FROM prediction_models
		WHERE tenant_id = $1 AND endpoint_id = $2 AND is_active = true
		ORDER BY version DESC
		LIMIT 1`

	var model PredictionModel
	var weightsJSON, coefficientsJSON, featuresJSON []byte

	err := r.db.QueryRowContext(ctx, query, tenantID, endpointID).Scan(
		&model.ID, &model.TenantID, &model.EndpointID, &model.ModelType,
		&model.Version, &weightsJSON, &coefficientsJSON, &featuresJSON,
		&model.Accuracy, &model.TrainedAt, &model.DataPointCount,
		&model.IsActive, &model.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(weightsJSON, &model.Weights)
	json.Unmarshal(coefficientsJSON, &model.Coefficients)
	json.Unmarshal(featuresJSON, &model.Features)

	return &model, nil
}

func (r *PostgresRepository) SaveEvent(ctx context.Context, event *RateLimitEvent) error {
	headersJSON, _ := json.Marshal(event.Headers)

	query := `
		INSERT INTO rate_limit_events (
			id, tenant_id, endpoint_id, delivery_id, timestamp, event_type,
			status_code, retry_after_seconds, request_rate, headers
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.db.ExecContext(ctx, query,
		event.ID, event.TenantID, event.EndpointID, event.DeliveryID,
		event.Timestamp, event.EventType, event.StatusCode,
		event.RetryAfter, event.RequestRate, headersJSON,
	)
	return err
}

func (r *PostgresRepository) GetEvents(ctx context.Context, tenantID, endpointID string, start, end time.Time) ([]RateLimitEvent, error) {
	query := `
		SELECT id, tenant_id, endpoint_id, delivery_id, timestamp, event_type,
			status_code, retry_after_seconds, request_rate, headers
		FROM rate_limit_events
		WHERE tenant_id = $1 AND endpoint_id = $2 AND timestamp >= $3 AND timestamp <= $4
		ORDER BY timestamp DESC`

	rows, err := r.db.QueryContext(ctx, query, tenantID, endpointID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []RateLimitEvent
	for rows.Next() {
		var event RateLimitEvent
		var headersJSON []byte

		err := rows.Scan(
			&event.ID, &event.TenantID, &event.EndpointID, &event.DeliveryID,
			&event.Timestamp, &event.EventType, &event.StatusCode,
			&event.RetryAfter, &event.RequestRate, &headersJSON,
		)
		if err != nil {
			return nil, err
		}

		json.Unmarshal(headersJSON, &event.Headers)
		events = append(events, event)
	}

	return events, nil
}

func (r *PostgresRepository) GetStats(ctx context.Context, tenantID string, start, end time.Time) (*SmartLimitStats, error) {
	stats := &SmartLimitStats{
		TenantID:    tenantID,
		Period:      fmt.Sprintf("%s to %s", start.Format(time.RFC3339), end.Format(time.RFC3339)),
		GeneratedAt: time.Now(),
	}

	// Get total and adaptive endpoints
	configQuery := `
		SELECT COUNT(*) as total,
			SUM(CASE WHEN enabled AND mode = 'adaptive' THEN 1 ELSE 0 END) as adaptive
		FROM adaptive_rate_configs
		WHERE tenant_id = $1`

	r.db.QueryRowContext(ctx, configQuery, tenantID).Scan(&stats.TotalEndpoints, &stats.AdaptiveEndpoints)

	// Get rate limit events
	eventQuery := `
		SELECT 
			SUM(CASE WHEN event_type = 'near_miss' THEN 1 ELSE 0 END) as prevented,
			SUM(CASE WHEN event_type = 'hit' THEN 1 ELSE 0 END) as hit
		FROM rate_limit_events
		WHERE tenant_id = $1 AND timestamp >= $2 AND timestamp <= $3`

	r.db.QueryRowContext(ctx, eventQuery, tenantID, start, end).Scan(
		&stats.RateLimitsPrevented, &stats.RateLimitsHit,
	)

	// Calculate savings estimate
	total := stats.RateLimitsPrevented + stats.RateLimitsHit
	if total > 0 {
		stats.SavingsEstimate = float64(stats.RateLimitsPrevented) / float64(total) * 100
	}

	return stats, nil
}
