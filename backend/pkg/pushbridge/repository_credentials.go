package pushbridge

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// SaveCredentials saves provider credentials
func (r *PostgresRepository) SaveCredentials(ctx context.Context, creds *ProviderCredentials) error {
	if creds.ID == "" {
		creds.ID = uuid.New().String()
	}

	credsJSON, _ := json.Marshal(creds.Credentials)

	query := `
		INSERT INTO push_provider_credentials (
			id, tenant_id, provider, name, credentials, environment,
			is_default, status, last_used_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := r.db.ExecContext(ctx, query,
		creds.ID, creds.TenantID, creds.Provider, creds.Name, credsJSON,
		creds.Environment, creds.IsDefault, creds.Status, creds.LastUsedAt,
		creds.CreatedAt, creds.UpdatedAt)
	return err
}

// GetCredentials retrieves provider credentials by ID
func (r *PostgresRepository) GetCredentials(ctx context.Context, tenantID, credID string) (*ProviderCredentials, error) {
	query := `
		SELECT id, tenant_id, provider, name, credentials, environment,
			   is_default, status, last_used_at, created_at, updated_at
		FROM push_provider_credentials
		WHERE tenant_id = $1 AND id = $2`

	var creds ProviderCredentials
	var credsJSON []byte
	var lastUsedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, tenantID, credID).Scan(
		&creds.ID, &creds.TenantID, &creds.Provider, &creds.Name, &credsJSON,
		&creds.Environment, &creds.IsDefault, &creds.Status, &lastUsedAt,
		&creds.CreatedAt, &creds.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("credentials not found")
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(credsJSON, &creds.Credentials)
	if lastUsedAt.Valid {
		creds.LastUsedAt = &lastUsedAt.Time
	}

	return &creds, nil
}

// ListCredentials lists provider credentials
func (r *PostgresRepository) ListCredentials(ctx context.Context, tenantID string, provider *Platform) ([]*ProviderCredentials, error) {
	query := `
		SELECT id, tenant_id, provider, name, credentials, environment,
			   is_default, status, last_used_at, created_at, updated_at
		FROM push_provider_credentials
		WHERE tenant_id = $1`
	args := []any{tenantID}

	if provider != nil {
		query += " AND provider = $2"
		args = append(args, *provider)
	}

	query += " ORDER BY is_default DESC, created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var credentials []*ProviderCredentials
	for rows.Next() {
		var creds ProviderCredentials
		var credsJSON []byte
		var lastUsedAt sql.NullTime

		err := rows.Scan(
			&creds.ID, &creds.TenantID, &creds.Provider, &creds.Name, &credsJSON,
			&creds.Environment, &creds.IsDefault, &creds.Status, &lastUsedAt,
			&creds.CreatedAt, &creds.UpdatedAt)
		if err != nil {
			continue
		}

		json.Unmarshal(credsJSON, &creds.Credentials)
		if lastUsedAt.Valid {
			creds.LastUsedAt = &lastUsedAt.Time
		}

		credentials = append(credentials, &creds)
	}

	return credentials, nil
}

// UpdateCredentials updates provider credentials
func (r *PostgresRepository) UpdateCredentials(ctx context.Context, creds *ProviderCredentials) error {
	credsJSON, _ := json.Marshal(creds.Credentials)

	query := `
		UPDATE push_provider_credentials SET
			name = $1, credentials = $2, environment = $3,
			is_default = $4, status = $5, last_used_at = $6, updated_at = $7
		WHERE tenant_id = $8 AND id = $9`

	_, err := r.db.ExecContext(ctx, query,
		creds.Name, credsJSON, creds.Environment,
		creds.IsDefault, creds.Status, creds.LastUsedAt, creds.UpdatedAt,
		creds.TenantID, creds.ID)
	return err
}

// DeleteCredentials deletes provider credentials
func (r *PostgresRepository) DeleteCredentials(ctx context.Context, tenantID, credID string) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM push_provider_credentials WHERE tenant_id = $1 AND id = $2",
		tenantID, credID)
	return err
}
