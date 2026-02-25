package pushbridge

import (
	"context"
	"database/sql"
	"fmt"
)

// SaveProvider saves a provider config
func (r *PostgresRepository) SaveProvider(ctx context.Context, provider *PushProviderConfig) error {
	query := `
		INSERT INTO push_providers (id, tenant_id, provider, name, enabled, config, credentials, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (tenant_id, provider) DO UPDATE SET
			name = EXCLUDED.name,
			enabled = EXCLUDED.enabled,
			config = EXCLUDED.config,
			credentials = EXCLUDED.credentials,
			updated_at = EXCLUDED.updated_at`

	_, err := r.db.ExecContext(ctx, query,
		provider.ID, provider.TenantID, provider.Provider, provider.Name,
		provider.Enabled, provider.Config, provider.Credentials,
		provider.CreatedAt, provider.UpdatedAt)
	return err
}

// GetProvider retrieves a provider
func (r *PostgresRepository) GetProvider(ctx context.Context, tenantID string, providerType ProviderType) (*PushProviderConfig, error) {
	query := `
		SELECT id, tenant_id, provider, name, enabled, config, credentials, created_at, updated_at
		FROM push_providers
		WHERE tenant_id = $1 AND provider = $2`

	var provider PushProviderConfig
	err := r.db.QueryRowContext(ctx, query, tenantID, providerType).Scan(
		&provider.ID, &provider.TenantID, &provider.Provider, &provider.Name,
		&provider.Enabled, &provider.Config, &provider.Credentials,
		&provider.CreatedAt, &provider.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("provider not found")
	}
	return &provider, err
}

// ListProviders lists providers
func (r *PostgresRepository) ListProviders(ctx context.Context, tenantID string) ([]PushProviderConfig, error) {
	query := `
		SELECT id, tenant_id, provider, name, enabled, config, created_at, updated_at
		FROM push_providers
		WHERE tenant_id = $1`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []PushProviderConfig
	for rows.Next() {
		var p PushProviderConfig
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Provider, &p.Name, &p.Enabled, &p.Config, &p.CreatedAt, &p.UpdatedAt); err != nil {
			continue
		}
		providers = append(providers, p)
	}

	return providers, nil
}

// DeleteProvider deletes a provider
func (r *PostgresRepository) DeleteProvider(ctx context.Context, tenantID string, providerType ProviderType) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM push_providers WHERE tenant_id = $1 AND provider = $2",
		tenantID, providerType)
	return err
}
