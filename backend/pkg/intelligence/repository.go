package intelligence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/josedab/waas/pkg/utils"
)

// Repository defines data access for the intelligence system
type Repository interface {
	SavePrediction(ctx context.Context, p *FailurePrediction) error
	GetPredictions(ctx context.Context, tenantID string, active bool) ([]FailurePrediction, error)
	GetPrediction(ctx context.Context, id string) (*FailurePrediction, error)
	ResolvePrediction(ctx context.Context, id string) error

	SaveAnomaly(ctx context.Context, a *AnomalyDetection) error
	GetAnomalies(ctx context.Context, tenantID string, acknowledged bool) ([]AnomalyDetection, error)
	AcknowledgeAnomaly(ctx context.Context, id string) error

	SaveOptimization(ctx context.Context, o *RetryOptimization) error
	GetOptimizations(ctx context.Context, tenantID string) ([]RetryOptimization, error)
	ApplyOptimization(ctx context.Context, id string) error

	SaveClassification(ctx context.Context, c *EventClassification) error
	GetClassifications(ctx context.Context, tenantID string, limit int) ([]EventClassification, error)

	SaveInsight(ctx context.Context, i *IntelligenceInsight) error
	GetInsights(ctx context.Context, tenantID string) ([]IntelligenceInsight, error)
	DismissInsight(ctx context.Context, id string) error

	SaveHealthScore(ctx context.Context, h *EndpointHealthScore) error
	GetHealthScores(ctx context.Context, tenantID string) ([]EndpointHealthScore, error)
	GetHealthScore(ctx context.Context, tenantID, endpointID string) (*EndpointHealthScore, error)

	GetSummary(ctx context.Context, tenantID string) (*IntelligenceSummary, error)
}

// PostgresRepository implements Repository with PostgreSQL
type PostgresRepository struct {
	db     *sqlx.DB
	logger *utils.Logger
}

func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db, logger: utils.NewLogger("intelligence")}
}

func (r *PostgresRepository) SavePrediction(ctx context.Context, p *FailurePrediction) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	p.CreatedAt = time.Now()
	p.ExpiresAt = time.Now().Add(24 * time.Hour)

	query := `INSERT INTO intel_predictions (id, tenant_id, endpoint_id, webhook_id, prediction_type,
		failure_probability, risk_level, confidence, recommendation, created_at, expires_at, resolved)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`
	_, err := r.db.ExecContext(ctx, query,
		p.ID, p.TenantID, p.EndpointID, p.WebhookID, p.PredictionType,
		p.FailureProbability, p.RiskLevel, p.Confidence, p.Recommendation,
		p.CreatedAt, p.ExpiresAt, false)
	return err
}

func (r *PostgresRepository) GetPredictions(ctx context.Context, tenantID string, active bool) ([]FailurePrediction, error) {
	var predictions []FailurePrediction
	query := `SELECT * FROM intel_predictions WHERE tenant_id = $1`
	if active {
		query += ` AND resolved = false AND expires_at > NOW()`
	}
	query += ` ORDER BY failure_probability DESC LIMIT 50`
	err := r.db.SelectContext(ctx, &predictions, query, tenantID)
	return predictions, err
}

func (r *PostgresRepository) GetPrediction(ctx context.Context, id string) (*FailurePrediction, error) {
	var p FailurePrediction
	err := r.db.GetContext(ctx, &p, `SELECT * FROM intel_predictions WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("prediction not found: %s", id)
	}
	return &p, err
}

func (r *PostgresRepository) ResolvePrediction(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE intel_predictions SET resolved = true WHERE id = $1`, id)
	return err
}

func (r *PostgresRepository) SaveAnomaly(ctx context.Context, a *AnomalyDetection) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	a.DetectedAt = time.Now()
	query := `INSERT INTO intel_anomalies (id, tenant_id, endpoint_id, anomaly_type, description,
		severity, anomaly_score, baseline_value, observed_value, deviation, detected_at, acknowledged)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`
	_, err := r.db.ExecContext(ctx, query,
		a.ID, a.TenantID, a.EndpointID, a.AnomalyType, a.Description,
		a.Severity, a.Score, a.Baseline, a.Observed, a.Deviation,
		a.DetectedAt, false)
	return err
}

func (r *PostgresRepository) GetAnomalies(ctx context.Context, tenantID string, acknowledged bool) ([]AnomalyDetection, error) {
	var anomalies []AnomalyDetection
	query := `SELECT * FROM intel_anomalies WHERE tenant_id = $1 AND acknowledged = $2 ORDER BY detected_at DESC LIMIT 50`
	err := r.db.SelectContext(ctx, &anomalies, query, tenantID, acknowledged)
	return anomalies, err
}

func (r *PostgresRepository) AcknowledgeAnomaly(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE intel_anomalies SET acknowledged = true WHERE id = $1`, id)
	return err
}

func (r *PostgresRepository) SaveOptimization(ctx context.Context, o *RetryOptimization) error {
	if o.ID == "" {
		o.ID = uuid.New().String()
	}
	o.CreatedAt = time.Now()
	query := `INSERT INTO intel_optimizations (id, tenant_id, endpoint_id, current_max_retries,
		suggested_max_retries, current_backoff, suggested_backoff, estimated_improvement_pct,
		rationale, data_points, created_at, applied)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`
	_, err := r.db.ExecContext(ctx, query,
		o.ID, o.TenantID, o.EndpointID, o.CurrentMaxRetries,
		o.SuggestedRetries, o.CurrentBackoff, o.SuggestedBackoff,
		o.EstimatedImprovement, o.Rationale, o.DataPoints,
		o.CreatedAt, false)
	return err
}

func (r *PostgresRepository) GetOptimizations(ctx context.Context, tenantID string) ([]RetryOptimization, error) {
	var opts []RetryOptimization
	err := r.db.SelectContext(ctx, &opts,
		`SELECT * FROM intel_optimizations WHERE tenant_id = $1 AND applied = false ORDER BY estimated_improvement_pct DESC`, tenantID)
	return opts, err
}

func (r *PostgresRepository) ApplyOptimization(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE intel_optimizations SET applied = true WHERE id = $1`, id)
	return err
}

func (r *PostgresRepository) SaveClassification(ctx context.Context, c *EventClassification) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	c.CreatedAt = time.Now()
	query := `INSERT INTO intel_classifications (id, tenant_id, webhook_id, event_type, category, confidence, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`
	_, err := r.db.ExecContext(ctx, query,
		c.ID, c.TenantID, c.WebhookID, c.EventType, c.Category, c.Confidence, c.CreatedAt)
	return err
}

func (r *PostgresRepository) GetClassifications(ctx context.Context, tenantID string, limit int) ([]EventClassification, error) {
	if limit <= 0 {
		limit = 50
	}
	var classifications []EventClassification
	err := r.db.SelectContext(ctx, &classifications,
		`SELECT * FROM intel_classifications WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2`, tenantID, limit)
	return classifications, err
}

func (r *PostgresRepository) SaveInsight(ctx context.Context, i *IntelligenceInsight) error {
	if i.ID == "" {
		i.ID = uuid.New().String()
	}
	i.CreatedAt = time.Now()
	query := `INSERT INTO intel_insights (id, tenant_id, insight_type, title, description, severity,
		action_url, action_label, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`
	_, err := r.db.ExecContext(ctx, query,
		i.ID, i.TenantID, i.InsightType, i.Title, i.Description,
		i.Severity, i.ActionURL, i.ActionLabel, i.CreatedAt)
	return err
}

func (r *PostgresRepository) GetInsights(ctx context.Context, tenantID string) ([]IntelligenceInsight, error) {
	var insights []IntelligenceInsight
	err := r.db.SelectContext(ctx, &insights,
		`SELECT * FROM intel_insights WHERE tenant_id = $1 AND dismissed_at IS NULL ORDER BY created_at DESC LIMIT 20`, tenantID)
	return insights, err
}

func (r *PostgresRepository) DismissInsight(ctx context.Context, id string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `UPDATE intel_insights SET dismissed_at = $1 WHERE id = $2`, now, id)
	return err
}

func (r *PostgresRepository) SaveHealthScore(ctx context.Context, h *EndpointHealthScore) error {
	h.CalculatedAt = time.Now()
	query := `INSERT INTO intel_health_scores (endpoint_id, tenant_id, overall_score, reliability_score,
		latency_score, throughput_score, error_rate_score, trend_direction, predicted_score_24h, calculated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (endpoint_id) DO UPDATE SET
		overall_score=$3, reliability_score=$4, latency_score=$5, throughput_score=$6,
		error_rate_score=$7, trend_direction=$8, predicted_score_24h=$9, calculated_at=$10`
	_, err := r.db.ExecContext(ctx, query,
		h.EndpointID, h.TenantID, h.OverallScore, h.ReliabilityScore,
		h.LatencyScore, h.ThroughputScore, h.ErrorRateScore,
		h.TrendDirection, h.PredictedScore24h, h.CalculatedAt)
	return err
}

func (r *PostgresRepository) GetHealthScores(ctx context.Context, tenantID string) ([]EndpointHealthScore, error) {
	var scores []EndpointHealthScore
	err := r.db.SelectContext(ctx, &scores,
		`SELECT * FROM intel_health_scores WHERE tenant_id = $1 ORDER BY overall_score ASC`, tenantID)
	return scores, err
}

func (r *PostgresRepository) GetHealthScore(ctx context.Context, tenantID, endpointID string) (*EndpointHealthScore, error) {
	var score EndpointHealthScore
	err := r.db.GetContext(ctx, &score,
		`SELECT * FROM intel_health_scores WHERE tenant_id = $1 AND endpoint_id = $2`, tenantID, endpointID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &score, err
}

func (r *PostgresRepository) GetSummary(ctx context.Context, tenantID string) (*IntelligenceSummary, error) {
	summary := &IntelligenceSummary{}

	if err := r.db.GetContext(ctx, &summary.TotalPredictions,
		`SELECT COUNT(*) FROM intel_predictions WHERE tenant_id = $1 AND resolved = false`, tenantID); err != nil {
		r.logger.Error("failed to query total predictions", map[string]interface{}{"tenant_id": tenantID, "error": err.Error()})
	}
	if err := r.db.GetContext(ctx, &summary.ActiveAnomalies,
		`SELECT COUNT(*) FROM intel_anomalies WHERE tenant_id = $1 AND acknowledged = false`, tenantID); err != nil {
		r.logger.Error("failed to query active anomalies", map[string]interface{}{"tenant_id": tenantID, "error": err.Error()})
	}
	if err := r.db.GetContext(ctx, &summary.PendingOptimizations,
		`SELECT COUNT(*) FROM intel_optimizations WHERE tenant_id = $1 AND applied = false`, tenantID); err != nil {
		r.logger.Error("failed to query pending optimizations", map[string]interface{}{"tenant_id": tenantID, "error": err.Error()})
	}
	if err := r.db.GetContext(ctx, &summary.AvgHealthScore,
		`SELECT COALESCE(AVG(overall_score), 0) FROM intel_health_scores WHERE tenant_id = $1`, tenantID); err != nil {
		r.logger.Error("failed to query avg health score", map[string]interface{}{"tenant_id": tenantID, "error": err.Error()})
	}
	if err := r.db.GetContext(ctx, &summary.AnomaliesDetected7d,
		`SELECT COUNT(*) FROM intel_anomalies WHERE tenant_id = $1 AND detected_at >= NOW() - INTERVAL '7 days'`, tenantID); err != nil {
		r.logger.Error("failed to query 7d anomalies", map[string]interface{}{"tenant_id": tenantID, "error": err.Error()})
	}

	summary.PredictionAccuracy = 0.85 // baseline until enough data
	summary.EstimatedSavings = float64(summary.PendingOptimizations) * 2.5
	return summary, nil
}
