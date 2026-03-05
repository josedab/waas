package ai

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// SaveAnalysis saves a debug analysis
func (r *PostgresRepository) SaveAnalysis(ctx context.Context, analysis *DebugAnalysis) error {
	query := `
		INSERT INTO ai_debug_analyses (
			id, delivery_id, classification, root_cause, explanation,
			suggestions, transform_fix, similar_issues, confidence_score,
			processing_time_ms, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (delivery_id) DO UPDATE SET
			classification = EXCLUDED.classification,
			root_cause = EXCLUDED.root_cause,
			explanation = EXCLUDED.explanation,
			suggestions = EXCLUDED.suggestions,
			transform_fix = EXCLUDED.transform_fix,
			confidence_score = EXCLUDED.confidence_score,
			processing_time_ms = EXCLUDED.processing_time_ms
	`

	classificationJSON, _ := json.Marshal(analysis.Classification)
	suggestionsJSON, _ := json.Marshal(analysis.Suggestions)
	transformFixJSON, _ := json.Marshal(analysis.TransformFix)
	similarJSON, _ := json.Marshal(analysis.SimilarIssues)

	_, err := r.db.ExecContext(ctx, query,
		analysis.ID,
		analysis.DeliveryID,
		classificationJSON,
		analysis.RootCause,
		analysis.Explanation,
		suggestionsJSON,
		transformFixJSON,
		similarJSON,
		analysis.ConfidenceScore,
		analysis.ProcessingTimeMs,
		analysis.CreatedAt,
	)

	return err
}

// GetAnalysis retrieves an analysis by ID
func (r *PostgresRepository) GetAnalysis(ctx context.Context, tenantID, analysisID string) (*DebugAnalysis, error) {
	query := `
		SELECT id, delivery_id, classification, root_cause, explanation,
			   suggestions, transform_fix, similar_issues, confidence_score,
			   processing_time_ms, created_at
		FROM ai_debug_analyses
		WHERE id = $1
	`

	var analysis DebugAnalysis
	var classificationJSON, suggestionsJSON, transformFixJSON, similarJSON []byte

	err := r.db.QueryRowContext(ctx, query, analysisID).Scan(
		&analysis.ID,
		&analysis.DeliveryID,
		&classificationJSON,
		&analysis.RootCause,
		&analysis.Explanation,
		&suggestionsJSON,
		&transformFixJSON,
		&similarJSON,
		&analysis.ConfidenceScore,
		&analysis.ProcessingTimeMs,
		&analysis.CreatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(classificationJSON, &analysis.Classification)
	json.Unmarshal(suggestionsJSON, &analysis.Suggestions)
	json.Unmarshal(transformFixJSON, &analysis.TransformFix)
	json.Unmarshal(similarJSON, &analysis.SimilarIssues)

	return &analysis, nil
}

// GetAnalysisByDelivery retrieves analysis by delivery ID
func (r *PostgresRepository) GetAnalysisByDelivery(ctx context.Context, tenantID, deliveryID string) (*DebugAnalysis, error) {
	query := `
		SELECT id, delivery_id, classification, root_cause, explanation,
			   suggestions, transform_fix, similar_issues, confidence_score,
			   processing_time_ms, created_at
		FROM ai_debug_analyses
		WHERE delivery_id = $1
	`

	var analysis DebugAnalysis
	var classificationJSON, suggestionsJSON, transformFixJSON, similarJSON []byte

	err := r.db.QueryRowContext(ctx, query, deliveryID).Scan(
		&analysis.ID,
		&analysis.DeliveryID,
		&classificationJSON,
		&analysis.RootCause,
		&analysis.Explanation,
		&suggestionsJSON,
		&transformFixJSON,
		&similarJSON,
		&analysis.ConfidenceScore,
		&analysis.ProcessingTimeMs,
		&analysis.CreatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(classificationJSON, &analysis.Classification)
	json.Unmarshal(suggestionsJSON, &analysis.Suggestions)
	json.Unmarshal(transformFixJSON, &analysis.TransformFix)
	json.Unmarshal(similarJSON, &analysis.SimilarIssues)

	return &analysis, nil
}

// ListAnalyses lists analyses for a tenant
func (r *PostgresRepository) ListAnalyses(ctx context.Context, tenantID string, limit, offset int) ([]DebugAnalysis, int, error) {
	countQuery := `SELECT COUNT(*) FROM ai_debug_analyses a
		JOIN delivery_attempts d ON a.delivery_id = d.id::text
		JOIN webhook_endpoints e ON d.endpoint_id = e.id
		WHERE e.tenant_id = $1`

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, tenantID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT a.id, a.delivery_id, a.classification, a.root_cause, a.explanation,
			   a.suggestions, a.transform_fix, a.similar_issues, a.confidence_score,
			   a.processing_time_ms, a.created_at
		FROM ai_debug_analyses a
		JOIN delivery_attempts d ON a.delivery_id = d.id::text
		JOIN webhook_endpoints e ON d.endpoint_id = e.id
		WHERE e.tenant_id = $1
		ORDER BY a.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var analyses []DebugAnalysis
	for rows.Next() {
		var analysis DebugAnalysis
		var classificationJSON, suggestionsJSON, transformFixJSON, similarJSON []byte

		if err := rows.Scan(
			&analysis.ID,
			&analysis.DeliveryID,
			&classificationJSON,
			&analysis.RootCause,
			&analysis.Explanation,
			&suggestionsJSON,
			&transformFixJSON,
			&similarJSON,
			&analysis.ConfidenceScore,
			&analysis.ProcessingTimeMs,
			&analysis.CreatedAt,
		); err != nil {
			return nil, 0, err
		}

		json.Unmarshal(classificationJSON, &analysis.Classification)
		json.Unmarshal(suggestionsJSON, &analysis.Suggestions)
		json.Unmarshal(transformFixJSON, &analysis.TransformFix)
		json.Unmarshal(similarJSON, &analysis.SimilarIssues)

		analyses = append(analyses, analysis)
	}

	return analyses, total, nil
}

// SavePattern saves an error pattern
func (r *PostgresRepository) SavePattern(ctx context.Context, pattern *ErrorPattern) error {
	query := `
		INSERT INTO ai_error_patterns (id, tenant_id, pattern, category, frequency, last_seen, resolution, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (tenant_id, pattern) DO UPDATE SET
			frequency = ai_error_patterns.frequency + 1,
			last_seen = EXCLUDED.last_seen
	`

	_, err := r.db.ExecContext(ctx, query,
		pattern.ID,
		pattern.TenantID,
		pattern.Pattern,
		pattern.Category,
		pattern.Frequency,
		pattern.LastSeen,
		pattern.Resolution,
		pattern.Metadata,
		pattern.CreatedAt,
	)

	return err
}

// GetPatterns retrieves error patterns for a tenant
func (r *PostgresRepository) GetPatterns(ctx context.Context, tenantID string, limit int) ([]ErrorPattern, error) {
	query := `
		SELECT id, tenant_id, pattern, category, frequency, last_seen, resolution, metadata, created_at
		FROM ai_error_patterns
		WHERE tenant_id = $1
		ORDER BY frequency DESC, last_seen DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patterns []ErrorPattern
	for rows.Next() {
		var p ErrorPattern
		if err := rows.Scan(
			&p.ID, &p.TenantID, &p.Pattern, &p.Category, &p.Frequency,
			&p.LastSeen, &p.Resolution, &p.Metadata, &p.CreatedAt,
		); err != nil {
			return nil, err
		}
		patterns = append(patterns, p)
	}

	return patterns, nil
}

// IncrementPatternFrequency increments pattern frequency
func (r *PostgresRepository) IncrementPatternFrequency(ctx context.Context, tenantID, patternID string) error {
	query := `UPDATE ai_error_patterns SET frequency = frequency + 1, last_seen = NOW() WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.ExecContext(ctx, query, patternID, tenantID)
	return err
}

// GetDeliveryContext retrieves delivery context for analysis
func (r *PostgresRepository) GetDeliveryContext(ctx context.Context, tenantID, deliveryID string) (*DeliveryContext, error) {
	query := `
		SELECT 
			d.id::text,
			d.endpoint_id::text,
			e.tenant_id::text,
			e.url,
			'POST',
			d.http_status,
			d.error_message,
			d.response_body,
			d.attempt_number,
			COALESCE(EXTRACT(EPOCH FROM (d.delivered_at - d.scheduled_at)) * 1000, 0)::bigint,
			d.created_at
		FROM delivery_attempts d
		JOIN webhook_endpoints e ON d.endpoint_id = e.id
		WHERE d.id::text = $1 AND e.tenant_id::text = $2
	`

	var dc DeliveryContext
	var httpStatus sql.NullInt32
	var errorMsg, responseBody sql.NullString

	err := r.db.QueryRowContext(ctx, query, deliveryID, tenantID).Scan(
		&dc.DeliveryID,
		&dc.EndpointID,
		&dc.TenantID,
		&dc.URL,
		&dc.HTTPMethod,
		&httpStatus,
		&errorMsg,
		&responseBody,
		&dc.AttemptNumber,
		&dc.Latency,
		&dc.Timestamp,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery context: %w", err)
	}

	if httpStatus.Valid {
		status := int(httpStatus.Int32)
		dc.HTTPStatus = &status
	}
	dc.ErrorMessage = errorMsg.String
	dc.ResponseBody = responseBody.String

	return &dc, nil
}

// GetSimilarDeliveries finds deliveries with similar error patterns
func (r *PostgresRepository) GetSimilarDeliveries(ctx context.Context, tenantID string, classification ErrorClassification, limit int) ([]DeliveryContext, error) {
	query := `
		SELECT 
			d.id::text,
			d.endpoint_id::text,
			e.tenant_id::text,
			e.url,
			'POST',
			d.http_status,
			d.error_message,
			d.response_body,
			d.attempt_number,
			0,
			d.created_at
		FROM delivery_attempts d
		JOIN webhook_endpoints e ON d.endpoint_id = e.id
		WHERE e.tenant_id::text = $1
			AND d.status = 'failed'
			AND d.http_status = $2
		ORDER BY d.created_at DESC
		LIMIT $3
	`

	var httpStatus interface{}
	if classification.SubCategory != "" {
		httpStatus = getStatusCodeFromSubCategory(classification.SubCategory)
	} else {
		httpStatus = nil
	}

	rows, err := r.db.QueryContext(ctx, query, tenantID, httpStatus, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deliveries []DeliveryContext
	for rows.Next() {
		var dc DeliveryContext
		var status sql.NullInt32
		var errorMsg, responseBody sql.NullString

		if err := rows.Scan(
			&dc.DeliveryID, &dc.EndpointID, &dc.TenantID, &dc.URL, &dc.HTTPMethod,
			&status, &errorMsg, &responseBody, &dc.AttemptNumber, &dc.Latency, &dc.Timestamp,
		); err != nil {
			continue
		}

		if status.Valid {
			s := int(status.Int32)
			dc.HTTPStatus = &s
		}
		dc.ErrorMessage = errorMsg.String
		dc.ResponseBody = responseBody.String

		deliveries = append(deliveries, dc)
	}

	return deliveries, nil
}

func getStatusCodeFromSubCategory(subCategory string) *int {
	statusMap := map[string]int{
		"unauthorized":    401,
		"forbidden":       403,
		"not_found":       404,
		"throttled":       429,
		"internal":        500,
		"bad_gateway":     502,
		"unavailable":     503,
		"gateway_timeout": 504,
	}

	if status, ok := statusMap[subCategory]; ok {
		return &status
	}
	return nil
}
