package repository

import (
	"context"
	"fmt"
	"time"

	"webhook-platform/pkg/database"
	"webhook-platform/pkg/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// TransformationRepository handles transformation data persistence
type TransformationRepository interface {
	Create(ctx context.Context, t *models.Transformation) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Transformation, error)
	GetByTenantID(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*models.Transformation, error)
	Update(ctx context.Context, t *models.Transformation) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByEndpointID(ctx context.Context, endpointID uuid.UUID) ([]*models.Transformation, error)
	LinkToEndpoint(ctx context.Context, endpointID, transformationID uuid.UUID, priority int) error
	UnlinkFromEndpoint(ctx context.Context, endpointID, transformationID uuid.UUID) error
	CreateLog(ctx context.Context, log *models.TransformationLog) error
	GetLogsByTransformationID(ctx context.Context, transformationID uuid.UUID, limit int) ([]*models.TransformationLog, error)
}

type transformationRepository struct {
	db *database.DB
}

// NewTransformationRepository creates a new transformation repository
func NewTransformationRepository(db *database.DB) TransformationRepository {
	return &transformationRepository{db: db}
}

func (r *transformationRepository) Create(ctx context.Context, t *models.Transformation) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	t.CreatedAt = time.Now().UTC()
	t.UpdatedAt = t.CreatedAt
	t.Version = 1

	query := `
		INSERT INTO transformations (id, tenant_id, name, description, script, enabled, version, 
			timeout_ms, max_memory_mb, allow_http, enable_logging, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := r.db.Pool.Exec(ctx, query,
		t.ID, t.TenantID, t.Name, t.Description, t.Script, t.Enabled, t.Version,
		t.Config.TimeoutMs, t.Config.MaxMemoryMB, t.Config.AllowHTTP, t.Config.EnableLogging,
		t.CreatedAt, t.UpdatedAt,
	)
	return err
}

func (r *transformationRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Transformation, error) {
	query := `
		SELECT id, tenant_id, name, description, script, enabled, version,
			timeout_ms, max_memory_mb, allow_http, enable_logging, created_at, updated_at
		FROM transformations
		WHERE id = $1
	`

	t := &models.Transformation{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.TenantID, &t.Name, &t.Description, &t.Script, &t.Enabled, &t.Version,
		&t.Config.TimeoutMs, &t.Config.MaxMemoryMB, &t.Config.AllowHTTP, &t.Config.EnableLogging,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("transformation not found")
	}
	return t, err
}

func (r *transformationRepository) GetByTenantID(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*models.Transformation, error) {
	query := `
		SELECT id, tenant_id, name, description, script, enabled, version,
			timeout_ms, max_memory_mb, allow_http, enable_logging, created_at, updated_at
		FROM transformations
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Pool.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transformations []*models.Transformation
	for rows.Next() {
		t := &models.Transformation{}
		err := rows.Scan(
			&t.ID, &t.TenantID, &t.Name, &t.Description, &t.Script, &t.Enabled, &t.Version,
			&t.Config.TimeoutMs, &t.Config.MaxMemoryMB, &t.Config.AllowHTTP, &t.Config.EnableLogging,
			&t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		transformations = append(transformations, t)
	}
	return transformations, nil
}

func (r *transformationRepository) Update(ctx context.Context, t *models.Transformation) error {
	t.UpdatedAt = time.Now().UTC()
	t.Version++

	query := `
		UPDATE transformations
		SET name = $2, description = $3, script = $4, enabled = $5, version = $6,
			timeout_ms = $7, max_memory_mb = $8, allow_http = $9, enable_logging = $10, updated_at = $11
		WHERE id = $1
	`

	result, err := r.db.Pool.Exec(ctx, query,
		t.ID, t.Name, t.Description, t.Script, t.Enabled, t.Version,
		t.Config.TimeoutMs, t.Config.MaxMemoryMB, t.Config.AllowHTTP, t.Config.EnableLogging,
		t.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("transformation not found")
	}
	return nil
}

func (r *transformationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM transformations WHERE id = $1`
	result, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("transformation not found")
	}
	return nil
}

func (r *transformationRepository) GetByEndpointID(ctx context.Context, endpointID uuid.UUID) ([]*models.Transformation, error) {
	query := `
		SELECT t.id, t.tenant_id, t.name, t.description, t.script, t.enabled, t.version,
			t.timeout_ms, t.max_memory_mb, t.allow_http, t.enable_logging, t.created_at, t.updated_at
		FROM transformations t
		INNER JOIN endpoint_transformations et ON t.id = et.transformation_id
		WHERE et.endpoint_id = $1 AND et.enabled = true
		ORDER BY et.priority ASC
	`

	rows, err := r.db.Pool.Query(ctx, query, endpointID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transformations []*models.Transformation
	for rows.Next() {
		t := &models.Transformation{}
		err := rows.Scan(
			&t.ID, &t.TenantID, &t.Name, &t.Description, &t.Script, &t.Enabled, &t.Version,
			&t.Config.TimeoutMs, &t.Config.MaxMemoryMB, &t.Config.AllowHTTP, &t.Config.EnableLogging,
			&t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		transformations = append(transformations, t)
	}
	return transformations, nil
}

func (r *transformationRepository) LinkToEndpoint(ctx context.Context, endpointID, transformationID uuid.UUID, priority int) error {
	query := `
		INSERT INTO endpoint_transformations (id, endpoint_id, transformation_id, priority, enabled, created_at)
		VALUES ($1, $2, $3, $4, true, $5)
		ON CONFLICT (endpoint_id, transformation_id) 
		DO UPDATE SET priority = $4, enabled = true
	`

	_, err := r.db.Pool.Exec(ctx, query, uuid.New(), endpointID, transformationID, priority, time.Now().UTC())
	return err
}

func (r *transformationRepository) UnlinkFromEndpoint(ctx context.Context, endpointID, transformationID uuid.UUID) error {
	query := `DELETE FROM endpoint_transformations WHERE endpoint_id = $1 AND transformation_id = $2`
	_, err := r.db.Pool.Exec(ctx, query, endpointID, transformationID)
	return err
}

func (r *transformationRepository) CreateLog(ctx context.Context, log *models.TransformationLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	log.CreatedAt = time.Now().UTC()

	query := `
		INSERT INTO transformation_logs (id, transformation_id, endpoint_id, delivery_id, input_payload, 
			output_payload, success, error_message, execution_time_ms, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.Pool.Exec(ctx, query,
		log.ID, log.TransformationID, log.EndpointID, log.DeliveryID, log.InputPayload,
		log.OutputPayload, log.Success, log.ErrorMessage, log.ExecutionTimeMs, log.CreatedAt,
	)
	return err
}

func (r *transformationRepository) GetLogsByTransformationID(ctx context.Context, transformationID uuid.UUID, limit int) ([]*models.TransformationLog, error) {
	query := `
		SELECT id, transformation_id, delivery_id, input_payload, output_payload, 
			success, error_message, execution_time_ms, created_at
		FROM transformation_logs
		WHERE transformation_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.Pool.Query(ctx, query, transformationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.TransformationLog
	for rows.Next() {
		log := &models.TransformationLog{}
		err := rows.Scan(
			&log.ID, &log.TransformationID, &log.DeliveryID, &log.InputPayload,
			&log.OutputPayload, &log.Success, &log.ErrorMessage, &log.ExecutionTimeMs, &log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, nil
}
