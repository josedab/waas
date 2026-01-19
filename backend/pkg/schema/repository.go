package schema

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines the interface for schema storage
type Repository interface {
	// Schema CRUD
	CreateSchema(ctx context.Context, schema *Schema) error
	GetSchema(ctx context.Context, tenantID, schemaID string) (*Schema, error)
	GetSchemaByName(ctx context.Context, tenantID, name string) (*Schema, error)
	ListSchemas(ctx context.Context, tenantID string, limit, offset int) ([]Schema, int, error)
	UpdateSchema(ctx context.Context, schema *Schema) error
	DeleteSchema(ctx context.Context, tenantID, schemaID string) error

	// Version management
	CreateVersion(ctx context.Context, version *SchemaVersion) error
	GetVersion(ctx context.Context, schemaID, version string) (*SchemaVersion, error)
	GetLatestVersion(ctx context.Context, schemaID string) (*SchemaVersion, error)
	ListVersions(ctx context.Context, schemaID string) ([]SchemaVersion, error)

	// Endpoint schema assignment
	AssignSchemaToEndpoint(ctx context.Context, assignment *EndpointSchema) error
	GetEndpointSchema(ctx context.Context, endpointID string) (*EndpointSchema, error)
	RemoveSchemaFromEndpoint(ctx context.Context, endpointID string) error
	ListEndpointsWithSchema(ctx context.Context, schemaID string) ([]string, error)
}

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL schema repository
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// CreateSchema creates a new schema
func (r *PostgresRepository) CreateSchema(ctx context.Context, schema *Schema) error {
	if schema.ID == "" {
		schema.ID = uuid.New().String()
	}
	schema.CreatedAt = time.Now()
	schema.UpdatedAt = time.Now()

	query := `
		INSERT INTO schemas (id, tenant_id, name, version, description, json_schema, is_active, is_default, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.ExecContext(ctx, query,
		schema.ID, schema.TenantID, schema.Name, schema.Version, schema.Description,
		schema.JSONSchema, schema.IsActive, schema.IsDefault, schema.CreatedAt, schema.UpdatedAt)
	return err
}

// GetSchema retrieves a schema by ID
func (r *PostgresRepository) GetSchema(ctx context.Context, tenantID, schemaID string) (*Schema, error) {
	var schema Schema
	query := `SELECT * FROM schemas WHERE tenant_id = $1 AND id = $2`
	err := r.db.GetContext(ctx, &schema, query, tenantID, schemaID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &schema, err
}

// GetSchemaByName retrieves a schema by name
func (r *PostgresRepository) GetSchemaByName(ctx context.Context, tenantID, name string) (*Schema, error) {
	var schema Schema
	query := `SELECT * FROM schemas WHERE tenant_id = $1 AND name = $2`
	err := r.db.GetContext(ctx, &schema, query, tenantID, name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &schema, err
}

// ListSchemas lists all schemas for a tenant
func (r *PostgresRepository) ListSchemas(ctx context.Context, tenantID string, limit, offset int) ([]Schema, int, error) {
	var schemas []Schema
	var total int

	countQuery := `SELECT COUNT(*) FROM schemas WHERE tenant_id = $1`
	if err := r.db.GetContext(ctx, &total, countQuery, tenantID); err != nil {
		return nil, 0, err
	}

	query := `SELECT * FROM schemas WHERE tenant_id = $1 ORDER BY name ASC LIMIT $2 OFFSET $3`
	if err := r.db.SelectContext(ctx, &schemas, query, tenantID, limit, offset); err != nil {
		return nil, 0, err
	}

	return schemas, total, nil
}

// UpdateSchema updates a schema
func (r *PostgresRepository) UpdateSchema(ctx context.Context, schema *Schema) error {
	schema.UpdatedAt = time.Now()
	query := `
		UPDATE schemas 
		SET description = $1, is_active = $2, is_default = $3, updated_at = $4
		WHERE id = $5 AND tenant_id = $6
	`
	_, err := r.db.ExecContext(ctx, query,
		schema.Description, schema.IsActive, schema.IsDefault, schema.UpdatedAt,
		schema.ID, schema.TenantID)
	return err
}

// DeleteSchema deletes a schema
func (r *PostgresRepository) DeleteSchema(ctx context.Context, tenantID, schemaID string) error {
	query := `DELETE FROM schemas WHERE tenant_id = $1 AND id = $2`
	_, err := r.db.ExecContext(ctx, query, tenantID, schemaID)
	return err
}

// CreateVersion creates a new schema version
func (r *PostgresRepository) CreateVersion(ctx context.Context, version *SchemaVersion) error {
	if version.ID == "" {
		version.ID = uuid.New().String()
	}
	version.CreatedAt = time.Now()

	query := `
		INSERT INTO schema_versions (id, schema_id, version, json_schema, changelog, created_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		version.ID, version.SchemaID, version.Version, version.JSONSchema,
		version.Changelog, version.CreatedAt, version.CreatedBy)
	
	if err == nil {
		// Update the main schema version
		updateQuery := `UPDATE schemas SET version = $1, json_schema = $2, updated_at = $3 WHERE id = $4`
		_, err = r.db.ExecContext(ctx, updateQuery, version.Version, version.JSONSchema, time.Now(), version.SchemaID)
	}
	
	return err
}

// GetVersion retrieves a specific version of a schema
func (r *PostgresRepository) GetVersion(ctx context.Context, schemaID, version string) (*SchemaVersion, error) {
	var v SchemaVersion
	query := `SELECT * FROM schema_versions WHERE schema_id = $1 AND version = $2`
	err := r.db.GetContext(ctx, &v, query, schemaID, version)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &v, err
}

// GetLatestVersion retrieves the latest version of a schema
func (r *PostgresRepository) GetLatestVersion(ctx context.Context, schemaID string) (*SchemaVersion, error) {
	var v SchemaVersion
	query := `SELECT * FROM schema_versions WHERE schema_id = $1 ORDER BY created_at DESC LIMIT 1`
	err := r.db.GetContext(ctx, &v, query, schemaID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &v, err
}

// ListVersions lists all versions of a schema
func (r *PostgresRepository) ListVersions(ctx context.Context, schemaID string) ([]SchemaVersion, error) {
	var versions []SchemaVersion
	query := `SELECT * FROM schema_versions WHERE schema_id = $1 ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &versions, query, schemaID)
	return versions, err
}

// AssignSchemaToEndpoint assigns a schema to an endpoint
func (r *PostgresRepository) AssignSchemaToEndpoint(ctx context.Context, assignment *EndpointSchema) error {
	assignment.CreatedAt = time.Now()
	query := `
		INSERT INTO endpoint_schemas (endpoint_id, schema_id, schema_version, validation_mode, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (endpoint_id) DO UPDATE SET
			schema_id = EXCLUDED.schema_id,
			schema_version = EXCLUDED.schema_version,
			validation_mode = EXCLUDED.validation_mode
	`
	_, err := r.db.ExecContext(ctx, query,
		assignment.EndpointID, assignment.SchemaID, assignment.SchemaVersion,
		assignment.ValidationMode, assignment.CreatedAt)
	return err
}

// GetEndpointSchema retrieves the schema assignment for an endpoint
func (r *PostgresRepository) GetEndpointSchema(ctx context.Context, endpointID string) (*EndpointSchema, error) {
	var assignment EndpointSchema
	query := `SELECT * FROM endpoint_schemas WHERE endpoint_id = $1`
	err := r.db.GetContext(ctx, &assignment, query, endpointID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &assignment, err
}

// RemoveSchemaFromEndpoint removes schema assignment from an endpoint
func (r *PostgresRepository) RemoveSchemaFromEndpoint(ctx context.Context, endpointID string) error {
	query := `DELETE FROM endpoint_schemas WHERE endpoint_id = $1`
	_, err := r.db.ExecContext(ctx, query, endpointID)
	return err
}

// ListEndpointsWithSchema lists all endpoints using a specific schema
func (r *PostgresRepository) ListEndpointsWithSchema(ctx context.Context, schemaID string) ([]string, error) {
	var endpoints []string
	query := `SELECT endpoint_id FROM endpoint_schemas WHERE schema_id = $1`
	err := r.db.SelectContext(ctx, &endpoints, query, schemaID)
	return endpoints, err
}
