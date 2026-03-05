package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/josedab/waas/pkg/database"
	apperrors "github.com/josedab/waas/pkg/errors"
	"github.com/josedab/waas/pkg/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type deliveryAttemptRepository struct {
	db *database.DB
}

// NewDeliveryAttemptRepository creates a new delivery attempt repository instance
func NewDeliveryAttemptRepository(db *database.DB) DeliveryAttemptRepository {
	return &deliveryAttemptRepository{db: db}
}

func (r *deliveryAttemptRepository) Create(ctx context.Context, attempt *models.DeliveryAttempt) error {
	query := `
		INSERT INTO delivery_attempts (id, endpoint_id, payload_hash, payload_size, status, http_status, 
		                              response_body, error_message, attempt_number, scheduled_at, delivered_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	now := time.Now()
	if attempt.ID == uuid.Nil {
		attempt.ID = uuid.New()
	}
	attempt.CreatedAt = now

	_, err := r.db.Pool.Exec(ctx, query,
		attempt.ID,
		attempt.EndpointID,
		attempt.PayloadHash,
		attempt.PayloadSize,
		attempt.Status,
		attempt.HTTPStatus,
		attempt.ResponseBody,
		attempt.ErrorMessage,
		attempt.AttemptNumber,
		attempt.ScheduledAt,
		attempt.DeliveredAt,
		attempt.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create delivery attempt: %w", err)
	}

	return nil
}

func (r *deliveryAttemptRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.DeliveryAttempt, error) {
	query := `
		SELECT id, endpoint_id, payload_hash, payload_size, status, http_status, response_body, 
		       error_message, attempt_number, scheduled_at, delivered_at, created_at
		FROM delivery_attempts WHERE id = $1`

	var attempt models.DeliveryAttempt
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&attempt.ID,
		&attempt.EndpointID,
		&attempt.PayloadHash,
		&attempt.PayloadSize,
		&attempt.Status,
		&attempt.HTTPStatus,
		&attempt.ResponseBody,
		&attempt.ErrorMessage,
		&attempt.AttemptNumber,
		&attempt.ScheduledAt,
		&attempt.DeliveredAt,
		&attempt.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("delivery attempt: %w", apperrors.ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get delivery attempt: %w", err)
	}

	return &attempt, nil
}

func (r *deliveryAttemptRepository) GetByEndpointID(ctx context.Context, endpointID uuid.UUID, limit, offset int) ([]*models.DeliveryAttempt, error) {
	query := `
		SELECT id, endpoint_id, payload_hash, payload_size, status, http_status, response_body, 
		       error_message, attempt_number, scheduled_at, delivered_at, created_at
		FROM delivery_attempts 
		WHERE endpoint_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	return r.queryAttempts(ctx, query, endpointID, limit, offset)
}

func (r *deliveryAttemptRepository) GetByStatus(ctx context.Context, status string, limit, offset int) ([]*models.DeliveryAttempt, error) {
	query := `
		SELECT id, endpoint_id, payload_hash, payload_size, status, http_status, response_body, 
		       error_message, attempt_number, scheduled_at, delivered_at, created_at
		FROM delivery_attempts 
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	return r.queryAttempts(ctx, query, status, limit, offset)
}

func (r *deliveryAttemptRepository) GetPendingDeliveries(ctx context.Context, limit int) ([]*models.DeliveryAttempt, error) {
	query := `
		SELECT id, endpoint_id, payload_hash, payload_size, status, http_status, response_body, 
		       error_message, attempt_number, scheduled_at, delivered_at, created_at
		FROM delivery_attempts 
		WHERE status IN ('pending', 'retrying') AND scheduled_at <= $1
		ORDER BY scheduled_at ASC
		LIMIT $2`

	rows, err := r.db.Pool.Query(ctx, query, time.Now(), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending deliveries: %w", err)
	}
	defer rows.Close()

	return r.scanAttempts(rows)
}

func (r *deliveryAttemptRepository) Update(ctx context.Context, attempt *models.DeliveryAttempt) error {
	query := `
		UPDATE delivery_attempts 
		SET status = $2, http_status = $3, response_body = $4, error_message = $5, 
		    attempt_number = $6, scheduled_at = $7, delivered_at = $8
		WHERE id = $1`

	result, err := r.db.Pool.Exec(ctx, query,
		attempt.ID,
		attempt.Status,
		attempt.HTTPStatus,
		attempt.ResponseBody,
		attempt.ErrorMessage,
		attempt.AttemptNumber,
		attempt.ScheduledAt,
		attempt.DeliveredAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update delivery attempt: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("delivery attempt: %w", apperrors.ErrNotFound)
	}

	return nil
}

func (r *deliveryAttemptRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `UPDATE delivery_attempts SET status = $2 WHERE id = $1`

	result, err := r.db.Pool.Exec(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("failed to update delivery attempt status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("delivery attempt: %w", apperrors.ErrNotFound)
	}

	return nil
}

func (r *deliveryAttemptRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM delivery_attempts WHERE id = $1`

	result, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete delivery attempt: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("delivery attempt: %w", apperrors.ErrNotFound)
	}

	return nil
}

func (r *deliveryAttemptRepository) GetDeliveryHistory(ctx context.Context, endpointID uuid.UUID, statuses []string, limit, offset int) ([]*models.DeliveryAttempt, error) {
	query := `
		SELECT id, endpoint_id, payload_hash, payload_size, status, http_status, response_body, 
		       error_message, attempt_number, scheduled_at, delivered_at, created_at
		FROM delivery_attempts 
		WHERE endpoint_id = $1`

	args := []interface{}{endpointID}
	argIndex := 2

	if len(statuses) > 0 {
		query += fmt.Sprintf(" AND status = ANY($%d)", argIndex)
		args = append(args, statuses)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery history: %w", err)
	}
	defer rows.Close()

	return r.scanAttempts(rows)
}

func (r *deliveryAttemptRepository) GetDeliveryHistoryWithFilters(ctx context.Context, tenantID uuid.UUID, filters DeliveryHistoryFilters, limit, offset int) ([]*models.DeliveryAttempt, int, error) {
	// Build the base query with JOIN to get tenant information
	baseQuery := `
		FROM delivery_attempts da
		JOIN webhook_endpoints we ON da.endpoint_id = we.id
		WHERE we.tenant_id = $1`

	args := []interface{}{tenantID}
	argIndex := 2

	// Add filters
	if len(filters.EndpointIDs) > 0 {
		baseQuery += fmt.Sprintf(" AND da.endpoint_id = ANY($%d)", argIndex)
		args = append(args, filters.EndpointIDs)
		argIndex++
	}

	if len(filters.Statuses) > 0 {
		baseQuery += fmt.Sprintf(" AND da.status = ANY($%d)", argIndex)
		args = append(args, filters.Statuses)
		argIndex++
	}

	if !filters.StartDate.IsZero() {
		baseQuery += fmt.Sprintf(" AND da.created_at >= $%d", argIndex)
		args = append(args, filters.StartDate)
		argIndex++
	}

	if !filters.EndDate.IsZero() {
		baseQuery += fmt.Sprintf(" AND da.created_at <= $%d", argIndex)
		args = append(args, filters.EndDate)
		argIndex++
	}

	// Get total count
	countQuery := "SELECT COUNT(*) " + baseQuery
	var totalCount int
	err := r.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get delivery history count: %w", err)
	}

	// Get the actual records
	selectQuery := `
		SELECT da.id, da.endpoint_id, da.payload_hash, da.payload_size, da.status, da.http_status, 
		       da.response_body, da.error_message, da.attempt_number, da.scheduled_at, da.delivered_at, da.created_at
		` + baseQuery + fmt.Sprintf(" ORDER BY da.created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := r.db.Pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get delivery history: %w", err)
	}
	defer rows.Close()

	attempts, err := r.scanAttempts(rows)
	if err != nil {
		return nil, 0, err
	}

	return attempts, totalCount, nil
}

func (r *deliveryAttemptRepository) GetDeliveryAttemptsByDeliveryID(ctx context.Context, deliveryID uuid.UUID, tenantID uuid.UUID) ([]*models.DeliveryAttempt, error) {
	query := `
		SELECT da.id, da.endpoint_id, da.payload_hash, da.payload_size, da.status, da.http_status, 
		       da.response_body, da.error_message, da.attempt_number, da.scheduled_at, da.delivered_at, da.created_at
		FROM delivery_attempts da
		JOIN webhook_endpoints we ON da.endpoint_id = we.id
		WHERE da.id = $1 AND we.tenant_id = $2
		ORDER BY da.attempt_number ASC`

	rows, err := r.db.Pool.Query(ctx, query, deliveryID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery attempts by delivery ID: %w", err)
	}
	defer rows.Close()

	return r.scanAttempts(rows)
}

// Helper methods
func (r *deliveryAttemptRepository) queryAttempts(ctx context.Context, query string, args ...interface{}) ([]*models.DeliveryAttempt, error) {
	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query delivery attempts: %w", err)
	}
	defer rows.Close()

	return r.scanAttempts(rows)
}

func (r *deliveryAttemptRepository) scanAttempts(rows pgx.Rows) ([]*models.DeliveryAttempt, error) {
	var attempts []*models.DeliveryAttempt

	for rows.Next() {
		var attempt models.DeliveryAttempt
		err := rows.Scan(
			&attempt.ID,
			&attempt.EndpointID,
			&attempt.PayloadHash,
			&attempt.PayloadSize,
			&attempt.Status,
			&attempt.HTTPStatus,
			&attempt.ResponseBody,
			&attempt.ErrorMessage,
			&attempt.AttemptNumber,
			&attempt.ScheduledAt,
			&attempt.DeliveredAt,
			&attempt.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan delivery attempt: %w", err)
		}
		attempts = append(attempts, &attempt)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating delivery attempt rows: %w", err)
	}

	return attempts, nil
}
