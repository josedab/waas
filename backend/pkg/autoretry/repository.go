package autoretry

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/josedab/waas/pkg/database"
)

// Repository handles auto-retry data persistence
type Repository struct {
	db *database.DB
}

// NewRepository creates a new auto-retry repository
func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

// SaveDeliveryFeatures stores extracted features for training
func (r *Repository) SaveDeliveryFeatures(ctx context.Context, f *DeliveryFeatures) error {
	query := `
		INSERT INTO delivery_features (
			delivery_id, endpoint_id,
			endpoint_success_rate_1h, endpoint_success_rate_24h,
			endpoint_avg_response_time_ms, endpoint_error_rate_1h,
			endpoint_last_success_minutes, hour_of_day, day_of_week,
			is_weekend, is_business_hours, payload_size_bytes,
			has_large_payload, attempt_number, time_since_first_attempt_seconds,
			previous_error_code, consecutive_failures
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id, created_at`

	return r.db.Pool.QueryRow(ctx, query,
		f.DeliveryID, f.EndpointID,
		f.EndpointSuccessRate1h, f.EndpointSuccessRate24h,
		f.EndpointAvgResponseTimeMs, f.EndpointErrorRate1h,
		f.EndpointLastSuccessMin, f.HourOfDay, f.DayOfWeek,
		f.IsWeekend, f.IsBusinessHours, f.PayloadSizeBytes,
		f.HasLargePayload, f.AttemptNumber, f.TimeSinceFirstAttemptSec,
		f.PreviousErrorCode, f.ConsecutiveFailures,
	).Scan(&f.ID, &f.CreatedAt)
}

// UpdateDeliveryOutcome updates features with actual delivery outcome
func (r *Repository) UpdateDeliveryOutcome(ctx context.Context, deliveryID string, success bool, responseTimeMs, statusCode int) error {
	query := `
		UPDATE delivery_features
		SET was_successful = $2, response_time_ms = $3, http_status_code = $4
		WHERE delivery_id = $1`

	_, err := r.db.Pool.Exec(ctx, query, deliveryID, success, responseTimeMs, statusCode)
	return err
}

// SavePrediction stores a model prediction
func (r *Repository) SavePrediction(ctx context.Context, p *RetryPrediction) error {
	featureJSON, err := json.Marshal(p.FeatureVector)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO retry_predictions (
			delivery_id, endpoint_id, predicted_success_probability,
			recommended_delay_seconds, confidence_score, model_version, feature_vector
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`

	return r.db.Pool.QueryRow(ctx, query,
		p.DeliveryID, p.EndpointID, p.PredictedSuccessProbability,
		p.RecommendedDelaySec, p.ConfidenceScore, p.ModelVersion, featureJSON,
	).Scan(&p.ID, &p.CreatedAt)
}

// UpdatePredictionOutcome updates prediction with actual outcome
func (r *Repository) UpdatePredictionOutcome(ctx context.Context, predictionID string, success bool, delayUsed int) error {
	query := `
		UPDATE retry_predictions
		SET actual_success = $2, actual_delay_used_seconds = $3, evaluated_at = NOW()
		WHERE id = $1`

	_, err := r.db.Pool.Exec(ctx, query, predictionID, success, delayUsed)
	return err
}

// GetEndpointStats retrieves endpoint statistics for feature extraction
func (r *Repository) GetEndpointStats(ctx context.Context, endpointID string) (*EndpointStats, error) {
	stats := &EndpointStats{EndpointID: endpointID}

	// Get success rate in last 1 hour
	query1h := `
		SELECT 
			COUNT(CASE WHEN was_successful = true THEN 1 END)::float / NULLIF(COUNT(*), 0),
			COUNT(CASE WHEN was_successful = false THEN 1 END)::float / NULLIF(COUNT(*), 0),
			AVG(response_time_ms)::int
		FROM delivery_features
		WHERE endpoint_id = $1 AND created_at > NOW() - INTERVAL '1 hour'`

	err := r.db.Pool.QueryRow(ctx, query1h, endpointID).Scan(
		&stats.SuccessRate1h,
		&stats.ErrorRate1h,
		&stats.AvgResponseTimeMs,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	// Get success rate in last 24 hours
	query24h := `
		SELECT COUNT(CASE WHEN was_successful = true THEN 1 END)::float / NULLIF(COUNT(*), 0)
		FROM delivery_features
		WHERE endpoint_id = $1 AND created_at > NOW() - INTERVAL '24 hours'`

	err = r.db.Pool.QueryRow(ctx, query24h, endpointID).Scan(&stats.SuccessRate24h)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	// Get last success time
	queryLastSuccess := `
		SELECT EXTRACT(EPOCH FROM (NOW() - MAX(created_at)))::int / 60
		FROM delivery_features
		WHERE endpoint_id = $1 AND was_successful = true`

	err = r.db.Pool.QueryRow(ctx, queryLastSuccess, endpointID).Scan(&stats.LastSuccessMinutes)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	// Get consecutive failures
	queryConsecFailures := `
		WITH recent_deliveries AS (
			SELECT was_successful
			FROM delivery_features
			WHERE endpoint_id = $1
			ORDER BY created_at DESC
			LIMIT 100
		)
		SELECT COUNT(*)
		FROM (
			SELECT was_successful, ROW_NUMBER() OVER () as rn
			FROM recent_deliveries
		) sub
		WHERE NOT was_successful AND rn <= (
			SELECT COALESCE(MIN(rn) - 1, COUNT(*))
			FROM (
				SELECT was_successful, ROW_NUMBER() OVER () as rn
				FROM recent_deliveries
			) inner_sub
			WHERE was_successful = true
		)`

	err = r.db.Pool.QueryRow(ctx, queryConsecFailures, endpointID).Scan(&stats.ConsecutiveFailures)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	return stats, nil
}

// EndpointStats contains calculated endpoint statistics
type EndpointStats struct {
	EndpointID          string
	SuccessRate1h       *float64
	SuccessRate24h      *float64
	ErrorRate1h         *float64
	AvgResponseTimeMs   *int
	LastSuccessMinutes  *int
	ConsecutiveFailures int
}

// GetModelMetrics retrieves model performance metrics
func (r *Repository) GetModelMetrics(ctx context.Context, modelVersion string) ([]ModelMetric, error) {
	query := `
		SELECT id, model_version, metric_name, metric_value, sample_size, 
			period_start, period_end, metadata, created_at
		FROM model_metrics
		WHERE model_version = $1
		ORDER BY created_at DESC`

	rows, err := r.db.Pool.Query(ctx, query, modelVersion)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []ModelMetric
	for rows.Next() {
		var m ModelMetric
		var metadataJSON []byte
		err := rows.Scan(
			&m.ID, &m.ModelVersion, &m.MetricName, &m.MetricValue,
			&m.SampleSize, &m.PeriodStart, &m.PeriodEnd, &metadataJSON, &m.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &m.Metadata)
		}
		metrics = append(metrics, m)
	}

	return metrics, nil
}

// ModelMetric represents a model performance metric
type ModelMetric struct {
	ID           string                 `json:"id"`
	ModelVersion string                 `json:"model_version"`
	MetricName   string                 `json:"metric_name"`
	MetricValue  float64                `json:"metric_value"`
	SampleSize   *int                   `json:"sample_size,omitempty"`
	PeriodStart  *time.Time             `json:"period_start,omitempty"`
	PeriodEnd    *time.Time             `json:"period_end,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

// SaveModelMetric saves a model metric
func (r *Repository) SaveModelMetric(ctx context.Context, m *ModelMetric) error {
	metadataJSON, err := json.Marshal(m.Metadata)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO model_metrics (model_version, metric_name, metric_value, sample_size, period_start, period_end, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`

	return r.db.Pool.QueryRow(ctx, query,
		m.ModelVersion, m.MetricName, m.MetricValue,
		m.SampleSize, m.PeriodStart, m.PeriodEnd, metadataJSON,
	).Scan(&m.ID, &m.CreatedAt)
}

// CreateExperiment creates a new A/B test experiment
func (r *Repository) CreateExperiment(ctx context.Context, e *Experiment) error {
	controlJSON, _ := json.Marshal(e.ControlStrategy)
	treatmentJSON, _ := json.Marshal(e.TreatmentStrategy)

	query := `
		INSERT INTO retry_experiments (name, description, control_strategy, treatment_strategy, traffic_split)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, status, created_at, updated_at`

	return r.db.Pool.QueryRow(ctx, query,
		e.Name, e.Description, controlJSON, treatmentJSON, e.TrafficSplit,
	).Scan(&e.ID, &e.Status, &e.CreatedAt, &e.UpdatedAt)
}

// GetExperiment retrieves an experiment by ID
func (r *Repository) GetExperiment(ctx context.Context, id string) (*Experiment, error) {
	var e Experiment
	var controlJSON, treatmentJSON []byte

	query := `
		SELECT id, name, description, status, control_strategy, treatment_strategy,
			traffic_split, start_date, end_date, control_sample_size, treatment_sample_size,
			control_success_rate, treatment_success_rate, created_at, updated_at
		FROM retry_experiments
		WHERE id = $1`

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&e.ID, &e.Name, &e.Description, &e.Status,
		&controlJSON, &treatmentJSON, &e.TrafficSplit,
		&e.StartDate, &e.EndDate, &e.ControlSampleSize, &e.TreatmentSampleSize,
		&e.ControlSuccessRate, &e.TreatmentSuccessRate, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(controlJSON, &e.ControlStrategy)
	json.Unmarshal(treatmentJSON, &e.TreatmentStrategy)

	return &e, nil
}

// ListActiveExperiments returns all running experiments
func (r *Repository) ListActiveExperiments(ctx context.Context) ([]Experiment, error) {
	query := `
		SELECT id, name, description, status, control_strategy, treatment_strategy,
			traffic_split, start_date, end_date, control_sample_size, treatment_sample_size,
			control_success_rate, treatment_success_rate, created_at, updated_at
		FROM retry_experiments
		WHERE status = 'running'`

	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var experiments []Experiment
	for rows.Next() {
		var e Experiment
		var controlJSON, treatmentJSON []byte
		err := rows.Scan(
			&e.ID, &e.Name, &e.Description, &e.Status,
			&controlJSON, &treatmentJSON, &e.TrafficSplit,
			&e.StartDate, &e.EndDate, &e.ControlSampleSize, &e.TreatmentSampleSize,
			&e.ControlSuccessRate, &e.TreatmentSuccessRate, &e.CreatedAt, &e.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		json.Unmarshal(controlJSON, &e.ControlStrategy)
		json.Unmarshal(treatmentJSON, &e.TreatmentStrategy)
		experiments = append(experiments, e)
	}

	return experiments, nil
}
