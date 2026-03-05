package mocking

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines the interface for mock storage
type Repository interface {
	CreateMockEndpoint(ctx context.Context, endpoint *MockEndpoint) error
	GetMockEndpoint(ctx context.Context, tenantID, endpointID string) (*MockEndpoint, error)
	ListMockEndpoints(ctx context.Context, tenantID string, limit, offset int) ([]MockEndpoint, int, error)
	UpdateMockEndpoint(ctx context.Context, endpoint *MockEndpoint) error
	DeleteMockEndpoint(ctx context.Context, tenantID, endpointID string) error

	CreateMockDelivery(ctx context.Context, delivery *MockDelivery) error
	ListMockDeliveries(ctx context.Context, tenantID, endpointID string, limit, offset int) ([]MockDelivery, int, error)

	CreateTemplate(ctx context.Context, template *MockTemplate) error
	GetTemplate(ctx context.Context, tenantID, templateID string) (*MockTemplate, error)
	ListTemplates(ctx context.Context, tenantID string, includePublic bool, limit, offset int) ([]MockTemplate, int, error)
	DeleteTemplate(ctx context.Context, tenantID, templateID string) error
}

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// CreateMockEndpoint creates a new mock endpoint
func (r *PostgresRepository) CreateMockEndpoint(ctx context.Context, endpoint *MockEndpoint) error {
	if endpoint.ID == "" {
		endpoint.ID = uuid.New().String()
	}

	templateJSON, err := json.Marshal(endpoint.Template)
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}
	scheduleJSON, err := json.Marshal(endpoint.Schedule)
	if err != nil {
		return fmt.Errorf("failed to marshal schedule: %w", err)
	}
	settingsJSON, err := json.Marshal(endpoint.Settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}
	metadataJSON, err := json.Marshal(endpoint.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO mock_endpoints 
		(id, tenant_id, name, description, url, event_type, template, schedule, settings, metadata, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err = r.db.ExecContext(ctx, query,
		endpoint.ID, endpoint.TenantID, endpoint.Name, endpoint.Description,
		endpoint.URL, endpoint.EventType, templateJSON, scheduleJSON, settingsJSON,
		metadataJSON, endpoint.IsActive, endpoint.CreatedAt, endpoint.UpdatedAt,
	)

	return err
}

// GetMockEndpoint retrieves a mock endpoint by ID
func (r *PostgresRepository) GetMockEndpoint(ctx context.Context, tenantID, endpointID string) (*MockEndpoint, error) {
	query := `
		SELECT id, tenant_id, name, description, url, event_type, template, schedule, settings, metadata, is_active, created_at, updated_at
		FROM mock_endpoints
		WHERE id = $1 AND tenant_id = $2
	`

	var endpoint MockEndpoint
	var templateJSON, scheduleJSON, settingsJSON, metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, endpointID, tenantID).Scan(
		&endpoint.ID, &endpoint.TenantID, &endpoint.Name, &endpoint.Description,
		&endpoint.URL, &endpoint.EventType, &templateJSON, &scheduleJSON, &settingsJSON,
		&metadataJSON, &endpoint.IsActive, &endpoint.CreatedAt, &endpoint.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(templateJSON, &endpoint.Template)
	json.Unmarshal(scheduleJSON, &endpoint.Schedule)
	json.Unmarshal(settingsJSON, &endpoint.Settings)
	json.Unmarshal(metadataJSON, &endpoint.Metadata)

	return &endpoint, nil
}

// ListMockEndpoints lists mock endpoints for a tenant
func (r *PostgresRepository) ListMockEndpoints(ctx context.Context, tenantID string, limit, offset int) ([]MockEndpoint, int, error) {
	countQuery := `SELECT COUNT(*) FROM mock_endpoints WHERE tenant_id = $1`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, tenantID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, tenant_id, name, description, url, event_type, template, schedule, settings, metadata, is_active, created_at, updated_at
		FROM mock_endpoints
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var endpoints []MockEndpoint
	for rows.Next() {
		var endpoint MockEndpoint
		var templateJSON, scheduleJSON, settingsJSON, metadataJSON []byte

		if err := rows.Scan(
			&endpoint.ID, &endpoint.TenantID, &endpoint.Name, &endpoint.Description,
			&endpoint.URL, &endpoint.EventType, &templateJSON, &scheduleJSON, &settingsJSON,
			&metadataJSON, &endpoint.IsActive, &endpoint.CreatedAt, &endpoint.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}

		json.Unmarshal(templateJSON, &endpoint.Template)
		json.Unmarshal(scheduleJSON, &endpoint.Schedule)
		json.Unmarshal(settingsJSON, &endpoint.Settings)
		json.Unmarshal(metadataJSON, &endpoint.Metadata)

		endpoints = append(endpoints, endpoint)
	}

	return endpoints, total, nil
}

// UpdateMockEndpoint updates a mock endpoint
func (r *PostgresRepository) UpdateMockEndpoint(ctx context.Context, endpoint *MockEndpoint) error {
	templateJSON, err := json.Marshal(endpoint.Template)
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}
	scheduleJSON, err := json.Marshal(endpoint.Schedule)
	if err != nil {
		return fmt.Errorf("failed to marshal schedule: %w", err)
	}
	settingsJSON, err := json.Marshal(endpoint.Settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}
	metadataJSON, err := json.Marshal(endpoint.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE mock_endpoints
		SET name = $1, description = $2, url = $3, event_type = $4, template = $5, 
		    schedule = $6, settings = $7, metadata = $8, is_active = $9, updated_at = $10
		WHERE id = $11 AND tenant_id = $12
	`

	_, err = r.db.ExecContext(ctx, query,
		endpoint.Name, endpoint.Description, endpoint.URL, endpoint.EventType,
		templateJSON, scheduleJSON, settingsJSON, metadataJSON,
		endpoint.IsActive, endpoint.UpdatedAt,
		endpoint.ID, endpoint.TenantID,
	)

	return err
}

// DeleteMockEndpoint deletes a mock endpoint
func (r *PostgresRepository) DeleteMockEndpoint(ctx context.Context, tenantID, endpointID string) error {
	query := `DELETE FROM mock_endpoints WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.ExecContext(ctx, query, endpointID, tenantID)
	return err
}

// CreateMockDelivery creates a mock delivery record
func (r *PostgresRepository) CreateMockDelivery(ctx context.Context, delivery *MockDelivery) error {
	if delivery.ID == "" {
		delivery.ID = uuid.New().String()
	}

	payloadJSON, err := json.Marshal(delivery.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	headersJSON, err := json.Marshal(delivery.Headers)
	if err != nil {
		return fmt.Errorf("failed to marshal headers: %w", err)
	}

	query := `
		INSERT INTO mock_deliveries 
		(id, endpoint_id, tenant_id, payload, headers, status, status_code, response_body, error, latency_ms, scheduled_at, sent_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err = r.db.ExecContext(ctx, query,
		delivery.ID, delivery.EndpointID, delivery.TenantID, payloadJSON, headersJSON,
		delivery.Status, delivery.StatusCode, delivery.ResponseBody, delivery.Error,
		delivery.LatencyMs, delivery.ScheduledAt, delivery.SentAt, delivery.CreatedAt,
	)

	return err
}

// ListMockDeliveries lists mock deliveries
func (r *PostgresRepository) ListMockDeliveries(ctx context.Context, tenantID, endpointID string, limit, offset int) ([]MockDelivery, int, error) {
	countQuery := `SELECT COUNT(*) FROM mock_deliveries WHERE tenant_id = $1 AND endpoint_id = $2`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, tenantID, endpointID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, endpoint_id, tenant_id, payload, headers, status, status_code, response_body, error, latency_ms, scheduled_at, sent_at, created_at
		FROM mock_deliveries
		WHERE tenant_id = $1 AND endpoint_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, endpointID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var deliveries []MockDelivery
	for rows.Next() {
		var d MockDelivery
		var payloadJSON, headersJSON []byte
		var scheduledAt, sentAt sql.NullTime

		if err := rows.Scan(
			&d.ID, &d.EndpointID, &d.TenantID, &payloadJSON, &headersJSON,
			&d.Status, &d.StatusCode, &d.ResponseBody, &d.Error,
			&d.LatencyMs, &scheduledAt, &sentAt, &d.CreatedAt,
		); err != nil {
			return nil, 0, err
		}

		json.Unmarshal(payloadJSON, &d.Payload)
		json.Unmarshal(headersJSON, &d.Headers)
		if scheduledAt.Valid {
			d.ScheduledAt = &scheduledAt.Time
		}
		if sentAt.Valid {
			d.SentAt = &sentAt.Time
		}

		deliveries = append(deliveries, d)
	}

	return deliveries, total, nil
}

// CreateTemplate creates a mock template
func (r *PostgresRepository) CreateTemplate(ctx context.Context, template *MockTemplate) error {
	if template.ID == "" {
		template.ID = uuid.New().String()
	}

	templateJSON, err := json.Marshal(template.Template)
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}
	examplesJSON, err := json.Marshal(template.Examples)
	if err != nil {
		return fmt.Errorf("failed to marshal examples: %w", err)
	}

	query := `
		INSERT INTO mock_templates 
		(id, tenant_id, name, description, event_type, category, template, examples, is_public, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err = r.db.ExecContext(ctx, query,
		template.ID, template.TenantID, template.Name, template.Description,
		template.EventType, template.Category, templateJSON, examplesJSON,
		template.IsPublic, template.CreatedAt, template.UpdatedAt,
	)

	return err
}

// GetTemplate retrieves a template by ID
func (r *PostgresRepository) GetTemplate(ctx context.Context, tenantID, templateID string) (*MockTemplate, error) {
	query := `
		SELECT id, tenant_id, name, description, event_type, category, template, examples, is_public, created_at, updated_at
		FROM mock_templates
		WHERE id = $1 AND (tenant_id = $2 OR is_public = true)
	`

	var template MockTemplate
	var templateJSON, examplesJSON []byte

	err := r.db.QueryRowContext(ctx, query, templateID, tenantID).Scan(
		&template.ID, &template.TenantID, &template.Name, &template.Description,
		&template.EventType, &template.Category, &templateJSON, &examplesJSON,
		&template.IsPublic, &template.CreatedAt, &template.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(templateJSON, &template.Template)
	json.Unmarshal(examplesJSON, &template.Examples)

	return &template, nil
}

// ListTemplates lists templates
func (r *PostgresRepository) ListTemplates(ctx context.Context, tenantID string, includePublic bool, limit, offset int) ([]MockTemplate, int, error) {
	var countQuery string
	var countArgs []interface{}

	if includePublic {
		countQuery = `SELECT COUNT(*) FROM mock_templates WHERE tenant_id = $1 OR is_public = true`
		countArgs = []interface{}{tenantID}
	} else {
		countQuery = `SELECT COUNT(*) FROM mock_templates WHERE tenant_id = $1`
		countArgs = []interface{}{tenantID}
	}

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	var query string
	var args []interface{}

	if includePublic {
		query = `
			SELECT id, tenant_id, name, description, event_type, category, template, examples, is_public, created_at, updated_at
			FROM mock_templates
			WHERE tenant_id = $1 OR is_public = true
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{tenantID, limit, offset}
	} else {
		query = `
			SELECT id, tenant_id, name, description, event_type, category, template, examples, is_public, created_at, updated_at
			FROM mock_templates
			WHERE tenant_id = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{tenantID, limit, offset}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var templates []MockTemplate
	for rows.Next() {
		var template MockTemplate
		var templateJSON, examplesJSON []byte

		if err := rows.Scan(
			&template.ID, &template.TenantID, &template.Name, &template.Description,
			&template.EventType, &template.Category, &templateJSON, &examplesJSON,
			&template.IsPublic, &template.CreatedAt, &template.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}

		json.Unmarshal(templateJSON, &template.Template)
		json.Unmarshal(examplesJSON, &template.Examples)

		templates = append(templates, template)
	}

	return templates, total, nil
}

// DeleteTemplate deletes a template
func (r *PostgresRepository) DeleteTemplate(ctx context.Context, tenantID, templateID string) error {
	query := `DELETE FROM mock_templates WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.ExecContext(ctx, query, templateID, tenantID)
	return err
}
