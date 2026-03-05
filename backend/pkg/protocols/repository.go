package protocols

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
)

// Repository handles protocol config persistence
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new protocol repository
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// configRow represents a database row
type configRow struct {
	ID         string    `db:"id"`
	TenantID   string    `db:"tenant_id"`
	EndpointID string    `db:"endpoint_id"`
	Protocol   string    `db:"protocol"`
	Target     string    `db:"target"`
	Options    []byte    `db:"options"`
	Headers    []byte    `db:"headers"`
	TLS        []byte    `db:"tls"`
	Auth       []byte    `db:"auth"`
	Timeout    int       `db:"timeout"`
	Retries    int       `db:"retries"`
	Enabled    bool      `db:"enabled"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}

func (r *configRow) toConfig() (*DeliveryConfig, error) {
	config := &DeliveryConfig{
		ID:         r.ID,
		TenantID:   r.TenantID,
		EndpointID: r.EndpointID,
		Protocol:   Protocol(r.Protocol),
		Target:     r.Target,
		Timeout:    r.Timeout,
		Retries:    r.Retries,
		Enabled:    r.Enabled,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}

	if len(r.Options) > 0 {
		if err := json.Unmarshal(r.Options, &config.Options); err != nil {
			return nil, err
		}
	}
	if len(r.Headers) > 0 {
		if err := json.Unmarshal(r.Headers, &config.Headers); err != nil {
			return nil, err
		}
	}
	if len(r.TLS) > 0 {
		if err := json.Unmarshal(r.TLS, &config.TLS); err != nil {
			return nil, err
		}
	}
	if len(r.Auth) > 0 {
		if err := json.Unmarshal(r.Auth, &config.Auth); err != nil {
			return nil, err
		}
	}

	return config, nil
}

// Create creates a new protocol config
func (r *Repository) Create(ctx context.Context, config *DeliveryConfig) error {
	optionsJSON, _ := json.Marshal(config.Options)
	headersJSON, _ := json.Marshal(config.Headers)
	tlsJSON, _ := json.Marshal(config.TLS)
	authJSON, _ := json.Marshal(config.Auth)

	query := `
		INSERT INTO protocol_configs (id, tenant_id, endpoint_id, protocol, target, options, headers, tls, auth, timeout, retries, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err := r.db.ExecContext(ctx, query,
		config.ID,
		config.TenantID,
		config.EndpointID,
		string(config.Protocol),
		config.Target,
		optionsJSON,
		headersJSON,
		tlsJSON,
		authJSON,
		config.Timeout,
		config.Retries,
		config.Enabled,
		config.CreatedAt,
		config.UpdatedAt,
	)

	return err
}

// GetByID retrieves a protocol config by ID
func (r *Repository) GetByID(ctx context.Context, tenantID, configID string) (*DeliveryConfig, error) {
	query := `
		SELECT id, tenant_id, endpoint_id, protocol, target, options, headers, tls, auth, timeout, retries, enabled, created_at, updated_at
		FROM protocol_configs
		WHERE id = $1 AND tenant_id = $2
	`

	var row configRow
	err := r.db.GetContext(ctx, &row, query, configID, tenantID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return row.toConfig()
}

// GetByEndpoint retrieves protocol configs for an endpoint
func (r *Repository) GetByEndpoint(ctx context.Context, tenantID, endpointID string) ([]*DeliveryConfig, error) {
	query := `
		SELECT id, tenant_id, endpoint_id, protocol, target, options, headers, tls, auth, timeout, retries, enabled, created_at, updated_at
		FROM protocol_configs
		WHERE tenant_id = $1 AND endpoint_id = $2 AND enabled = true
		ORDER BY created_at ASC
	`

	var rows []configRow
	if err := r.db.SelectContext(ctx, &rows, query, tenantID, endpointID); err != nil {
		return nil, err
	}

	configs := make([]*DeliveryConfig, 0, len(rows))
	for _, row := range rows {
		config, err := row.toConfig()
		if err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}

	return configs, nil
}

// List retrieves protocol configs for a tenant
func (r *Repository) List(ctx context.Context, tenantID string, limit, offset int) ([]*DeliveryConfig, int, error) {
	countQuery := `SELECT COUNT(*) FROM protocol_configs WHERE tenant_id = $1`
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, tenantID); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, tenant_id, endpoint_id, protocol, target, options, headers, tls, auth, timeout, retries, enabled, created_at, updated_at
		FROM protocol_configs
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var rows []configRow
	if err := r.db.SelectContext(ctx, &rows, query, tenantID, limit, offset); err != nil {
		return nil, 0, err
	}

	configs := make([]*DeliveryConfig, 0, len(rows))
	for _, row := range rows {
		config, err := row.toConfig()
		if err != nil {
			return nil, 0, err
		}
		configs = append(configs, config)
	}

	return configs, total, nil
}

// ListByProtocol retrieves protocol configs for a tenant by protocol
func (r *Repository) ListByProtocol(ctx context.Context, tenantID string, protocol Protocol, limit, offset int) ([]*DeliveryConfig, int, error) {
	countQuery := `SELECT COUNT(*) FROM protocol_configs WHERE tenant_id = $1 AND protocol = $2`
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, tenantID, string(protocol)); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, tenant_id, endpoint_id, protocol, target, options, headers, tls, auth, timeout, retries, enabled, created_at, updated_at
		FROM protocol_configs
		WHERE tenant_id = $1 AND protocol = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	var rows []configRow
	if err := r.db.SelectContext(ctx, &rows, query, tenantID, string(protocol), limit, offset); err != nil {
		return nil, 0, err
	}

	configs := make([]*DeliveryConfig, 0, len(rows))
	for _, row := range rows {
		config, err := row.toConfig()
		if err != nil {
			return nil, 0, err
		}
		configs = append(configs, config)
	}

	return configs, total, nil
}

// Update updates a protocol config
func (r *Repository) Update(ctx context.Context, config *DeliveryConfig) error {
	optionsJSON, _ := json.Marshal(config.Options)
	headersJSON, _ := json.Marshal(config.Headers)
	tlsJSON, _ := json.Marshal(config.TLS)
	authJSON, _ := json.Marshal(config.Auth)

	query := `
		UPDATE protocol_configs
		SET protocol = $3, target = $4, options = $5, headers = $6, tls = $7, auth = $8, timeout = $9, retries = $10, enabled = $11, updated_at = $12
		WHERE id = $1 AND tenant_id = $2
	`

	_, err := r.db.ExecContext(ctx, query,
		config.ID,
		config.TenantID,
		string(config.Protocol),
		config.Target,
		optionsJSON,
		headersJSON,
		tlsJSON,
		authJSON,
		config.Timeout,
		config.Retries,
		config.Enabled,
		time.Now(),
	)

	return err
}

// Delete deletes a protocol config
func (r *Repository) Delete(ctx context.Context, tenantID, configID string) error {
	query := `DELETE FROM protocol_configs WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.ExecContext(ctx, query, configID, tenantID)
	return err
}

// SetEnabled enables or disables a protocol config
func (r *Repository) SetEnabled(ctx context.Context, tenantID, configID string, enabled bool) error {
	query := `
		UPDATE protocol_configs
		SET enabled = $3, updated_at = $4
		WHERE id = $1 AND tenant_id = $2
	`
	_, err := r.db.ExecContext(ctx, query, configID, tenantID, enabled, time.Now())
	return err
}
