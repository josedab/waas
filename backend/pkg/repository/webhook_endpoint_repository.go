package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/josedab/waas/pkg/database"
	apperrors "github.com/josedab/waas/pkg/errors"
	"github.com/josedab/waas/pkg/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type webhookEndpointRepository struct {
	db *database.DB
}

// NewWebhookEndpointRepository creates a new webhook endpoint repository instance
func NewWebhookEndpointRepository(db *database.DB) WebhookEndpointRepository {
	return &webhookEndpointRepository{db: db}
}

func (r *webhookEndpointRepository) Create(ctx context.Context, endpoint *models.WebhookEndpoint) error {
	query := `
		INSERT INTO webhook_endpoints (id, tenant_id, url, secret_hash, is_active, retry_config, custom_headers, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	now := time.Now()
	if endpoint.ID == uuid.Nil {
		endpoint.ID = uuid.New()
	}
	endpoint.CreatedAt = now
	endpoint.UpdatedAt = now

	// Marshal retry config and custom headers to JSON
	retryConfigJSON, err := json.Marshal(endpoint.RetryConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal retry config: %w", err)
	}

	customHeadersJSON, err := json.Marshal(endpoint.CustomHeaders)
	if err != nil {
		return fmt.Errorf("failed to marshal custom headers: %w", err)
	}

	_, err = r.db.Pool.Exec(ctx, query,
		endpoint.ID,
		endpoint.TenantID,
		endpoint.URL,
		endpoint.SecretHash,
		endpoint.IsActive,
		retryConfigJSON,
		customHeadersJSON,
		endpoint.CreatedAt,
		endpoint.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create webhook endpoint: %w", err)
	}

	return nil
}

func (r *webhookEndpointRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.WebhookEndpoint, error) {
	query := `
		SELECT id, tenant_id, url, secret_hash, is_active, retry_config, custom_headers, created_at, updated_at
		FROM webhook_endpoints WHERE id = $1`

	var endpoint models.WebhookEndpoint
	var retryConfigJSON, customHeadersJSON []byte

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&endpoint.ID,
		&endpoint.TenantID,
		&endpoint.URL,
		&endpoint.SecretHash,
		&endpoint.IsActive,
		&retryConfigJSON,
		&customHeadersJSON,
		&endpoint.CreatedAt,
		&endpoint.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("webhook endpoint: %w", apperrors.ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get webhook endpoint: %w", err)
	}

	// Unmarshal JSON fields
	if err := json.Unmarshal(retryConfigJSON, &endpoint.RetryConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal retry config: %w", err)
	}

	if err := json.Unmarshal(customHeadersJSON, &endpoint.CustomHeaders); err != nil {
		return nil, fmt.Errorf("failed to unmarshal custom headers: %w", err)
	}

	return &endpoint, nil
}

func (r *webhookEndpointRepository) GetByTenantID(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*models.WebhookEndpoint, error) {
	query := `
		SELECT id, tenant_id, url, secret_hash, is_active, retry_config, custom_headers, created_at, updated_at
		FROM webhook_endpoints 
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	return r.queryEndpoints(ctx, query, tenantID, limit, offset)
}

func (r *webhookEndpointRepository) CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM webhook_endpoints WHERE tenant_id = $1`
	var count int
	err := r.db.Pool.QueryRow(ctx, query, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count webhook endpoints: %w", err)
	}
	return count, nil
}

func (r *webhookEndpointRepository) GetActiveByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*models.WebhookEndpoint, error) {
	query := `
		SELECT id, tenant_id, url, secret_hash, is_active, retry_config, custom_headers, created_at, updated_at
		FROM webhook_endpoints 
		WHERE tenant_id = $1 AND is_active = true
		ORDER BY created_at DESC`

	rows, err := r.db.Pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active webhook endpoints: %w", err)
	}
	defer rows.Close()

	return r.scanEndpoints(rows)
}

func (r *webhookEndpointRepository) Update(ctx context.Context, endpoint *models.WebhookEndpoint) error {
	query := `
		UPDATE webhook_endpoints 
		SET url = $2, secret_hash = $3, is_active = $4, retry_config = $5, custom_headers = $6, updated_at = $7
		WHERE id = $1`

	endpoint.UpdatedAt = time.Now()

	// Marshal retry config and custom headers to JSON
	retryConfigJSON, err := json.Marshal(endpoint.RetryConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal retry config: %w", err)
	}

	customHeadersJSON, err := json.Marshal(endpoint.CustomHeaders)
	if err != nil {
		return fmt.Errorf("failed to marshal custom headers: %w", err)
	}

	result, err := r.db.Pool.Exec(ctx, query,
		endpoint.ID,
		endpoint.URL,
		endpoint.SecretHash,
		endpoint.IsActive,
		retryConfigJSON,
		customHeadersJSON,
		endpoint.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update webhook endpoint: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("webhook endpoint: %w", apperrors.ErrNotFound)
	}

	return nil
}

func (r *webhookEndpointRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM webhook_endpoints WHERE id = $1`

	result, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete webhook endpoint: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("webhook endpoint: %w", apperrors.ErrNotFound)
	}

	return nil
}

func (r *webhookEndpointRepository) SetActive(ctx context.Context, id uuid.UUID, active bool) error {
	query := `UPDATE webhook_endpoints SET is_active = $2, updated_at = $3 WHERE id = $1`

	result, err := r.db.Pool.Exec(ctx, query, id, active, time.Now())
	if err != nil {
		return fmt.Errorf("failed to set webhook endpoint active status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("webhook endpoint: %w", apperrors.ErrNotFound)
	}

	return nil
}

func (r *webhookEndpointRepository) UpdateStatus(ctx context.Context, id uuid.UUID, active bool) error {
	return r.SetActive(ctx, id, active)
}

// Helper methods
func (r *webhookEndpointRepository) queryEndpoints(ctx context.Context, query string, args ...interface{}) ([]*models.WebhookEndpoint, error) {
	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query webhook endpoints: %w", err)
	}
	defer rows.Close()

	return r.scanEndpoints(rows)
}

func (r *webhookEndpointRepository) scanEndpoints(rows pgx.Rows) ([]*models.WebhookEndpoint, error) {
	var endpoints []*models.WebhookEndpoint

	for rows.Next() {
		var endpoint models.WebhookEndpoint
		var retryConfigJSON, customHeadersJSON []byte

		err := rows.Scan(
			&endpoint.ID,
			&endpoint.TenantID,
			&endpoint.URL,
			&endpoint.SecretHash,
			&endpoint.IsActive,
			&retryConfigJSON,
			&customHeadersJSON,
			&endpoint.CreatedAt,
			&endpoint.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan webhook endpoint: %w", err)
		}

		// Unmarshal JSON fields
		if err := json.Unmarshal(retryConfigJSON, &endpoint.RetryConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal retry config: %w", err)
		}

		if err := json.Unmarshal(customHeadersJSON, &endpoint.CustomHeaders); err != nil {
			return nil, fmt.Errorf("failed to unmarshal custom headers: %w", err)
		}

		endpoints = append(endpoints, &endpoint)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating webhook endpoint rows: %w", err)
	}

	return endpoints, nil
}
