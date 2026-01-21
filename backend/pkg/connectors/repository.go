package connectors

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines the interface for connector storage
type Repository interface {
	// Installed connectors
	InstallConnector(ctx context.Context, installed *InstalledConnector) error
	GetInstalledConnector(ctx context.Context, tenantID, id string) (*InstalledConnector, error)
	ListInstalledConnectors(ctx context.Context, tenantID string, limit, offset int) ([]InstalledConnector, int, error)
	UpdateInstalledConnector(ctx context.Context, installed *InstalledConnector) error
	UninstallConnector(ctx context.Context, tenantID, id string) error

	// Execution logging
	LogExecution(ctx context.Context, exec *ConnectorExecution) error
	ListExecutions(ctx context.Context, installedConnectorID string, limit, offset int) ([]ConnectorExecution, int, error)
}

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL connector repository
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// InstallConnector installs a connector for a tenant
func (r *PostgresRepository) InstallConnector(ctx context.Context, installed *InstalledConnector) error {
	if installed.ID == "" {
		installed.ID = uuid.New().String()
	}
	installed.CreatedAt = time.Now()
	installed.UpdatedAt = time.Now()

	query := `
		INSERT INTO installed_connectors (id, tenant_id, connector_id, name, config, is_active, provider_id, endpoint_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.ExecContext(ctx, query,
		installed.ID, installed.TenantID, installed.ConnectorID, installed.Name,
		installed.Config, installed.IsActive, installed.ProviderID, installed.EndpointID,
		installed.CreatedAt, installed.UpdatedAt)
	return err
}

// GetInstalledConnector retrieves an installed connector
func (r *PostgresRepository) GetInstalledConnector(ctx context.Context, tenantID, id string) (*InstalledConnector, error) {
	var installed InstalledConnector
	query := `SELECT * FROM installed_connectors WHERE tenant_id = $1 AND id = $2`
	err := r.db.GetContext(ctx, &installed, query, tenantID, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &installed, err
}

// ListInstalledConnectors lists all installed connectors for a tenant
func (r *PostgresRepository) ListInstalledConnectors(ctx context.Context, tenantID string, limit, offset int) ([]InstalledConnector, int, error) {
	var connectors []InstalledConnector
	var total int

	countQuery := `SELECT COUNT(*) FROM installed_connectors WHERE tenant_id = $1`
	if err := r.db.GetContext(ctx, &total, countQuery, tenantID); err != nil {
		return nil, 0, err
	}

	query := `SELECT * FROM installed_connectors WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	if err := r.db.SelectContext(ctx, &connectors, query, tenantID, limit, offset); err != nil {
		return nil, 0, err
	}

	return connectors, total, nil
}

// UpdateInstalledConnector updates an installed connector
func (r *PostgresRepository) UpdateInstalledConnector(ctx context.Context, installed *InstalledConnector) error {
	installed.UpdatedAt = time.Now()
	query := `
		UPDATE installed_connectors 
		SET name = $1, config = $2, is_active = $3, updated_at = $4
		WHERE id = $5 AND tenant_id = $6
	`
	_, err := r.db.ExecContext(ctx, query,
		installed.Name, installed.Config, installed.IsActive, installed.UpdatedAt,
		installed.ID, installed.TenantID)
	return err
}

// UninstallConnector removes an installed connector
func (r *PostgresRepository) UninstallConnector(ctx context.Context, tenantID, id string) error {
	query := `DELETE FROM installed_connectors WHERE tenant_id = $1 AND id = $2`
	_, err := r.db.ExecContext(ctx, query, tenantID, id)
	return err
}

// LogExecution logs a connector execution
func (r *PostgresRepository) LogExecution(ctx context.Context, exec *ConnectorExecution) error {
	if exec.ID == "" {
		exec.ID = uuid.New().String()
	}
	exec.CreatedAt = time.Now()

	query := `
		INSERT INTO connector_executions (id, installed_connector_id, event_type, input_payload, output_payload, status, error, duration_ms, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, query,
		exec.ID, exec.InstalledConnectorID, exec.EventType, exec.InputPayload,
		exec.OutputPayload, exec.Status, exec.Error, exec.Duration, exec.CreatedAt)
	return err
}

// ListExecutions lists connector executions
func (r *PostgresRepository) ListExecutions(ctx context.Context, installedConnectorID string, limit, offset int) ([]ConnectorExecution, int, error) {
	var executions []ConnectorExecution
	var total int

	countQuery := `SELECT COUNT(*) FROM connector_executions WHERE installed_connector_id = $1`
	if err := r.db.GetContext(ctx, &total, countQuery, installedConnectorID); err != nil {
		return nil, 0, err
	}

	query := `SELECT * FROM connector_executions WHERE installed_connector_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	if err := r.db.SelectContext(ctx, &executions, query, installedConnectorID, limit, offset); err != nil {
		return nil, 0, err
	}

	return executions, total, nil
}

// InMemoryRepository implements Repository for testing
type InMemoryRepository struct {
	connectors []InstalledConnector
	executions []ConnectorExecution
}

// NewInMemoryRepository creates a new in-memory repository
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		connectors: make([]InstalledConnector, 0),
		executions: make([]ConnectorExecution, 0),
	}
}

func (r *InMemoryRepository) InstallConnector(ctx context.Context, installed *InstalledConnector) error {
	if installed.ID == "" {
		installed.ID = uuid.New().String()
	}
	installed.CreatedAt = time.Now()
	installed.UpdatedAt = time.Now()
	r.connectors = append(r.connectors, *installed)
	return nil
}

func (r *InMemoryRepository) GetInstalledConnector(ctx context.Context, tenantID, id string) (*InstalledConnector, error) {
	for _, c := range r.connectors {
		if c.TenantID == tenantID && c.ID == id {
			return &c, nil
		}
	}
	return nil, nil
}

func (r *InMemoryRepository) ListInstalledConnectors(ctx context.Context, tenantID string, limit, offset int) ([]InstalledConnector, int, error) {
	var result []InstalledConnector
	for _, c := range r.connectors {
		if c.TenantID == tenantID {
			result = append(result, c)
		}
	}
	total := len(result)
	if offset < len(result) {
		end := offset + limit
		if end > len(result) {
			end = len(result)
		}
		result = result[offset:end]
	} else {
		result = []InstalledConnector{}
	}
	return result, total, nil
}

func (r *InMemoryRepository) UpdateInstalledConnector(ctx context.Context, installed *InstalledConnector) error {
	for i, c := range r.connectors {
		if c.TenantID == installed.TenantID && c.ID == installed.ID {
			installed.UpdatedAt = time.Now()
			r.connectors[i] = *installed
			return nil
		}
	}
	return nil
}

func (r *InMemoryRepository) UninstallConnector(ctx context.Context, tenantID, id string) error {
	for i, c := range r.connectors {
		if c.TenantID == tenantID && c.ID == id {
			r.connectors = append(r.connectors[:i], r.connectors[i+1:]...)
			return nil
		}
	}
	return nil
}

func (r *InMemoryRepository) LogExecution(ctx context.Context, exec *ConnectorExecution) error {
	if exec.ID == "" {
		exec.ID = uuid.New().String()
	}
	exec.CreatedAt = time.Now()
	r.executions = append(r.executions, *exec)
	return nil
}

func (r *InMemoryRepository) ListExecutions(ctx context.Context, installedConnectorID string, limit, offset int) ([]ConnectorExecution, int, error) {
	var result []ConnectorExecution
	for _, e := range r.executions {
		if e.InstalledConnectorID == installedConnectorID {
			result = append(result, e)
		}
	}
	total := len(result)
	if offset < len(result) {
		end := offset + limit
		if end > len(result) {
			end = len(result)
		}
		result = result[offset:end]
	} else {
		result = []ConnectorExecution{}
	}
	return result, total, nil
}

var _ Repository = (*InMemoryRepository)(nil) // Compile-time check

// Define JSON as needed
var _ = json.Marshal
