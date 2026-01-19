package schema

import (
	"encoding/json"
	"time"
)

// Schema represents a webhook payload schema
type Schema struct {
	ID          string          `json:"id" db:"id"`
	TenantID    string          `json:"tenant_id" db:"tenant_id"`
	Name        string          `json:"name" db:"name"`
	Version     string          `json:"version" db:"version"`
	Description string          `json:"description,omitempty" db:"description"`
	JSONSchema  json.RawMessage `json:"json_schema" db:"json_schema"`
	IsActive    bool            `json:"is_active" db:"is_active"`
	IsDefault   bool            `json:"is_default" db:"is_default"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
}

// SchemaVersion represents a specific version of a schema
type SchemaVersion struct {
	ID         string          `json:"id" db:"id"`
	SchemaID   string          `json:"schema_id" db:"schema_id"`
	Version    string          `json:"version" db:"version"`
	JSONSchema json.RawMessage `json:"json_schema" db:"json_schema"`
	Changelog  string          `json:"changelog,omitempty" db:"changelog"`
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`
	CreatedBy  string          `json:"created_by,omitempty" db:"created_by"`
}

// EndpointSchema links an endpoint to a schema
type EndpointSchema struct {
	EndpointID      string    `json:"endpoint_id" db:"endpoint_id"`
	SchemaID        string    `json:"schema_id" db:"schema_id"`
	SchemaVersion   string    `json:"schema_version,omitempty" db:"schema_version"`
	ValidationMode  string    `json:"validation_mode" db:"validation_mode"` // strict, warn, none
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// ValidationMode constants
const (
	ValidationModeStrict = "strict" // Reject invalid payloads
	ValidationModeWarn   = "warn"   // Log warning but allow
	ValidationModeNone   = "none"   // No validation
)

// ValidationResult represents the result of schema validation
type ValidationResult struct {
	Valid       bool              `json:"valid"`
	Errors      []ValidationError `json:"errors,omitempty"`
	SchemaID    string            `json:"schema_id,omitempty"`
	SchemaName  string            `json:"schema_name,omitempty"`
	Version     string            `json:"version,omitempty"`
}

// ValidationError represents a single validation error
type ValidationError struct {
	Path        string `json:"path"`
	Message     string `json:"message"`
	SchemaPath  string `json:"schema_path,omitempty"`
	Type        string `json:"type,omitempty"`
}

// CompatibilityResult represents schema compatibility check result
type CompatibilityResult struct {
	Compatible      bool                  `json:"compatible"`
	BreakingChanges []BreakingChange      `json:"breaking_changes,omitempty"`
	Warnings        []string              `json:"warnings,omitempty"`
}

// BreakingChange represents a breaking schema change
type BreakingChange struct {
	Type        string `json:"type"` // removed_field, type_change, required_added
	Path        string `json:"path"`
	Description string `json:"description"`
	OldValue    string `json:"old_value,omitempty"`
	NewValue    string `json:"new_value,omitempty"`
}

// CreateSchemaRequest represents a request to create a schema
type CreateSchemaRequest struct {
	Name        string          `json:"name" binding:"required,min=1,max=255"`
	Version     string          `json:"version" binding:"required"`
	Description string          `json:"description,omitempty"`
	JSONSchema  json.RawMessage `json:"json_schema" binding:"required"`
	IsDefault   bool            `json:"is_default,omitempty"`
}

// UpdateSchemaRequest represents a request to update a schema
type UpdateSchemaRequest struct {
	Description string `json:"description,omitempty"`
	IsActive    bool   `json:"is_active"`
	IsDefault   bool   `json:"is_default"`
}

// CreateVersionRequest represents a request to create a new schema version
type CreateVersionRequest struct {
	Version    string          `json:"version" binding:"required"`
	JSONSchema json.RawMessage `json:"json_schema" binding:"required"`
	Changelog  string          `json:"changelog,omitempty"`
}

// AssignSchemaRequest represents a request to assign a schema to an endpoint
type AssignSchemaRequest struct {
	SchemaID       string `json:"schema_id" binding:"required"`
	SchemaVersion  string `json:"schema_version,omitempty"` // Empty = latest
	ValidationMode string `json:"validation_mode" binding:"required,oneof=strict warn none"`
}
