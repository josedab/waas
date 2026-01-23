package schemaregistry

import "time"

// SchemaFormat constants
const (
	SchemaFormatJSONSchema = "json_schema"
	SchemaFormatAvro       = "avro"
	SchemaFormatProtobuf   = "protobuf"
)

// CompatibilityMode constants
const (
	CompatibilityBackward = "backward"
	CompatibilityForward  = "forward"
	CompatibilityFull     = "full"
	CompatibilityNone     = "none"
)

// SchemaStatus constants
const (
	SchemaStatusActive     = "active"
	SchemaStatusDeprecated = "deprecated"
)

// SchemaDefinition represents a registered event schema
type SchemaDefinition struct {
	ID                string    `json:"id" db:"id"`
	TenantID          string    `json:"tenant_id" db:"tenant_id"`
	Subject           string    `json:"subject" db:"subject"`
	Version           int       `json:"version" db:"version"`
	SchemaFormat      string    `json:"schema_format" db:"schema_format"`
	SchemaContent     string    `json:"schema_content" db:"schema_content"`
	Fingerprint       string    `json:"fingerprint" db:"fingerprint"`
	Description       string    `json:"description,omitempty" db:"description"`
	IsLatest          bool      `json:"is_latest" db:"is_latest"`
	CompatibilityMode string    `json:"compatibility_mode" db:"compatibility_mode"`
	Status            string    `json:"status" db:"status"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// CompatibilityResult represents the outcome of a compatibility check
type CompatibilityResult struct {
	IsCompatible    bool     `json:"is_compatible"`
	BreakingChanges []string `json:"breaking_changes,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
	OldVersion      int      `json:"old_version"`
	NewVersion      int      `json:"new_version"`
}

// SchemaVersion represents a specific version of a schema
type SchemaVersion struct {
	ID            string    `json:"id" db:"id"`
	SchemaID      string    `json:"schema_id" db:"schema_id"`
	Version       int       `json:"version" db:"version"`
	SchemaContent string    `json:"schema_content" db:"schema_content"`
	ChangeLog     string    `json:"change_log,omitempty" db:"change_log"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// RegisterSchemaRequest is the request DTO for registering a schema
type RegisterSchemaRequest struct {
	Subject           string `json:"subject" binding:"required,min=1,max=255"`
	SchemaFormat      string `json:"schema_format" binding:"required,oneof=json_schema avro protobuf"`
	SchemaContent     string `json:"schema_content" binding:"required"`
	Description       string `json:"description,omitempty"`
	CompatibilityMode string `json:"compatibility_mode,omitempty" binding:"omitempty,oneof=backward forward full none"`
}

// CheckCompatibilityRequest is the request DTO for checking compatibility
type CheckCompatibilityRequest struct {
	Subject       string `json:"subject" binding:"required,min=1,max=255"`
	SchemaContent string `json:"schema_content" binding:"required"`
	SchemaFormat  string `json:"schema_format" binding:"required,oneof=json_schema avro protobuf"`
}

// SchemaStats aggregates schema registry statistics
type SchemaStats struct {
	TotalSchemas    int            `json:"total_schemas"`
	TotalVersions   int            `json:"total_versions"`
	FormatBreakdown map[string]int `json:"format_breakdown"`
}
