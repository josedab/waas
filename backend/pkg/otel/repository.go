package otel

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx"
)

// Repository handles OTEL config persistence
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new OTEL repository
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// configRow represents a database row for OTEL config
type configRow struct {
	ID          string    `db:"id"`
	TenantID    string    `db:"tenant_id"`
	Name        string    `db:"name"`
	ServiceName string    `db:"service_name"`
	Enabled     bool      `db:"enabled"`
	Traces      []byte    `db:"traces"`
	Metrics     []byte    `db:"metrics"`
	Logs        []byte    `db:"logs"`
	Attributes  []byte    `db:"attributes"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

func (r *configRow) toConfig() (*Config, error) {
	config := &Config{
		ID:          r.ID,
		TenantID:    r.TenantID,
		Name:        r.Name,
		ServiceName: r.ServiceName,
		Enabled:     r.Enabled,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}

	if len(r.Traces) > 0 {
		if err := json.Unmarshal(r.Traces, &config.Traces); err != nil {
			return nil, err
		}
	}
	if len(r.Metrics) > 0 {
		if err := json.Unmarshal(r.Metrics, &config.Metrics); err != nil {
			return nil, err
		}
	}
	if len(r.Logs) > 0 {
		if err := json.Unmarshal(r.Logs, &config.Logs); err != nil {
			return nil, err
		}
	}
	if len(r.Attributes) > 0 {
		if err := json.Unmarshal(r.Attributes, &config.Attributes); err != nil {
			return nil, err
		}
	}

	return config, nil
}

// Create creates a new OTEL config
func (r *Repository) Create(ctx context.Context, config *Config) error {
	tracesJSON, _ := json.Marshal(config.Traces)
	metricsJSON, _ := json.Marshal(config.Metrics)
	logsJSON, _ := json.Marshal(config.Logs)
	attrsJSON, _ := json.Marshal(config.Attributes)

	query := `
		INSERT INTO otel_configs (id, tenant_id, name, service_name, enabled, traces, metrics, logs, attributes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.ExecContext(ctx, query,
		config.ID,
		config.TenantID,
		config.Name,
		config.ServiceName,
		config.Enabled,
		tracesJSON,
		metricsJSON,
		logsJSON,
		attrsJSON,
		config.CreatedAt,
		config.UpdatedAt,
	)

	return err
}

// GetByID retrieves an OTEL config by ID
func (r *Repository) GetByID(ctx context.Context, tenantID, configID string) (*Config, error) {
	query := `
		SELECT id, tenant_id, name, service_name, enabled, traces, metrics, logs, attributes, created_at, updated_at
		FROM otel_configs
		WHERE id = $1 AND tenant_id = $2
	`

	var row configRow
	err := r.db.GetContext(ctx, &row, query, configID, tenantID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return row.toConfig()
}

// GetDefault retrieves the default/active OTEL config for a tenant
func (r *Repository) GetDefault(ctx context.Context, tenantID string) (*Config, error) {
	query := `
		SELECT id, tenant_id, name, service_name, enabled, traces, metrics, logs, attributes, created_at, updated_at
		FROM otel_configs
		WHERE tenant_id = $1 AND enabled = true
		ORDER BY created_at DESC
		LIMIT 1
	`

	var row configRow
	err := r.db.GetContext(ctx, &row, query, tenantID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return row.toConfig()
}

// List retrieves OTEL configs for a tenant
func (r *Repository) List(ctx context.Context, tenantID string, limit, offset int) ([]*Config, int, error) {
	countQuery := `SELECT COUNT(*) FROM otel_configs WHERE tenant_id = $1`
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, tenantID); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, tenant_id, name, service_name, enabled, traces, metrics, logs, attributes, created_at, updated_at
		FROM otel_configs
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var rows []configRow
	if err := r.db.SelectContext(ctx, &rows, query, tenantID, limit, offset); err != nil {
		return nil, 0, err
	}

	configs := make([]*Config, 0, len(rows))
	for _, row := range rows {
		config, err := row.toConfig()
		if err != nil {
			return nil, 0, err
		}
		configs = append(configs, config)
	}

	return configs, total, nil
}

// Update updates an OTEL config
func (r *Repository) Update(ctx context.Context, config *Config) error {
	tracesJSON, _ := json.Marshal(config.Traces)
	metricsJSON, _ := json.Marshal(config.Metrics)
	logsJSON, _ := json.Marshal(config.Logs)
	attrsJSON, _ := json.Marshal(config.Attributes)

	query := `
		UPDATE otel_configs
		SET name = $3, service_name = $4, enabled = $5, traces = $6, metrics = $7, logs = $8, attributes = $9, updated_at = $10
		WHERE id = $1 AND tenant_id = $2
	`

	_, err := r.db.ExecContext(ctx, query,
		config.ID,
		config.TenantID,
		config.Name,
		config.ServiceName,
		config.Enabled,
		tracesJSON,
		metricsJSON,
		logsJSON,
		attrsJSON,
		time.Now(),
	)

	return err
}

// Delete deletes an OTEL config
func (r *Repository) Delete(ctx context.Context, tenantID, configID string) error {
	query := `DELETE FROM otel_configs WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.ExecContext(ctx, query, configID, tenantID)
	return err
}

// SetEnabled enables or disables an OTEL config
func (r *Repository) SetEnabled(ctx context.Context, tenantID, configID string, enabled bool) error {
	query := `
		UPDATE otel_configs
		SET enabled = $3, updated_at = $4
		WHERE id = $1 AND tenant_id = $2
	`
	_, err := r.db.ExecContext(ctx, query, configID, tenantID, enabled, time.Now())
	return err
}
