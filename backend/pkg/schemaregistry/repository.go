package schemaregistry

import "context"

// Repository defines the data access interface for schema registry
type Repository interface {
	// Schema CRUD
	CreateSchema(ctx context.Context, schema *SchemaDefinition) error
	GetSchema(ctx context.Context, tenantID, schemaID string) (*SchemaDefinition, error)
	UpdateSchema(ctx context.Context, schema *SchemaDefinition) error
	DeleteSchema(ctx context.Context, tenantID, schemaID string) error

	// Subject operations
	GetSchemaBySubject(ctx context.Context, tenantID, subject string) (*SchemaDefinition, error)
	ListSchemas(ctx context.Context, tenantID string, limit, offset int) ([]SchemaDefinition, error)
	SearchSchemas(ctx context.Context, tenantID, query string) ([]SchemaDefinition, error)

	// Version operations
	ListVersions(ctx context.Context, tenantID, subject string) ([]SchemaVersion, error)
	GetLatestVersion(ctx context.Context, tenantID, subject string) (*SchemaDefinition, error)
	CreateVersion(ctx context.Context, version *SchemaVersion) error

	// Stats
	GetStats(ctx context.Context, tenantID string) (*SchemaStats, error)
}
