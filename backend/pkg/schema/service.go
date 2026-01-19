package schema

import (
	"context"
	"fmt"
)

// Service provides schema management functionality
type Service struct {
	repo      Repository
	validator *Validator
}

// NewService creates a new schema service
func NewService(repo Repository) *Service {
	return &Service{
		repo:      repo,
		validator: NewValidator(repo),
	}
}

// CreateSchema creates a new schema
func (s *Service) CreateSchema(ctx context.Context, tenantID string, req *CreateSchemaRequest) (*Schema, error) {
	// Validate the JSON schema itself
	result := s.validator.ValidatePayloadDirect(req.JSONSchema, []byte(`{"type": "object"}`))
	if !result.Valid {
		return nil, fmt.Errorf("invalid JSON schema format")
	}

	// Check for duplicate name
	existing, err := s.repo.GetSchemaByName(ctx, tenantID, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing schema: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("schema with name '%s' already exists", req.Name)
	}

	schema := &Schema{
		TenantID:    tenantID,
		Name:        req.Name,
		Version:     req.Version,
		Description: req.Description,
		JSONSchema:  req.JSONSchema,
		IsActive:    true,
		IsDefault:   req.IsDefault,
	}

	if err := s.repo.CreateSchema(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	// Create initial version
	version := &SchemaVersion{
		SchemaID:   schema.ID,
		Version:    schema.Version,
		JSONSchema: schema.JSONSchema,
		Changelog:  "Initial version",
	}
	if err := s.repo.CreateVersion(ctx, version); err != nil {
		// Non-fatal, schema is created
	}

	return schema, nil
}

// GetSchema retrieves a schema by ID
func (s *Service) GetSchema(ctx context.Context, tenantID, schemaID string) (*Schema, error) {
	return s.repo.GetSchema(ctx, tenantID, schemaID)
}

// ListSchemas lists all schemas for a tenant
func (s *Service) ListSchemas(ctx context.Context, tenantID string, limit, offset int) ([]Schema, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListSchemas(ctx, tenantID, limit, offset)
}

// UpdateSchema updates a schema
func (s *Service) UpdateSchema(ctx context.Context, tenantID, schemaID string, req *UpdateSchemaRequest) (*Schema, error) {
	schema, err := s.repo.GetSchema(ctx, tenantID, schemaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}
	if schema == nil {
		return nil, fmt.Errorf("schema not found")
	}

	schema.Description = req.Description
	schema.IsActive = req.IsActive
	schema.IsDefault = req.IsDefault

	if err := s.repo.UpdateSchema(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to update schema: %w", err)
	}

	return schema, nil
}

// DeleteSchema deletes a schema
func (s *Service) DeleteSchema(ctx context.Context, tenantID, schemaID string) error {
	// Check if schema is in use
	endpoints, err := s.repo.ListEndpointsWithSchema(ctx, schemaID)
	if err != nil {
		return fmt.Errorf("failed to check schema usage: %w", err)
	}
	if len(endpoints) > 0 {
		return fmt.Errorf("schema is assigned to %d endpoints, remove assignments first", len(endpoints))
	}

	return s.repo.DeleteSchema(ctx, tenantID, schemaID)
}

// CreateVersion creates a new version of a schema
func (s *Service) CreateVersion(ctx context.Context, tenantID, schemaID string, req *CreateVersionRequest) (*SchemaVersion, *CompatibilityResult, error) {
	schema, err := s.repo.GetSchema(ctx, tenantID, schemaID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get schema: %w", err)
	}
	if schema == nil {
		return nil, nil, fmt.Errorf("schema not found")
	}

	// Check for duplicate version
	existing, err := s.repo.GetVersion(ctx, schemaID, req.Version)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check existing version: %w", err)
	}
	if existing != nil {
		return nil, nil, fmt.Errorf("version '%s' already exists", req.Version)
	}

	// Check compatibility with previous version
	compatibility, err := s.validator.CheckCompatibility(schema.JSONSchema, req.JSONSchema)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check compatibility: %w", err)
	}

	version := &SchemaVersion{
		SchemaID:   schemaID,
		Version:    req.Version,
		JSONSchema: req.JSONSchema,
		Changelog:  req.Changelog,
	}

	if err := s.repo.CreateVersion(ctx, version); err != nil {
		return nil, nil, fmt.Errorf("failed to create version: %w", err)
	}

	return version, compatibility, nil
}

// ListVersions lists all versions of a schema
func (s *Service) ListVersions(ctx context.Context, tenantID, schemaID string) ([]SchemaVersion, error) {
	schema, err := s.repo.GetSchema(ctx, tenantID, schemaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}
	if schema == nil {
		return nil, fmt.Errorf("schema not found")
	}

	return s.repo.ListVersions(ctx, schemaID)
}

// AssignSchemaToEndpoint assigns a schema to an endpoint
func (s *Service) AssignSchemaToEndpoint(ctx context.Context, tenantID, endpointID string, req *AssignSchemaRequest) error {
	// Validate schema exists
	schema, err := s.repo.GetSchema(ctx, tenantID, req.SchemaID)
	if err != nil {
		return fmt.Errorf("failed to get schema: %w", err)
	}
	if schema == nil {
		return fmt.Errorf("schema not found")
	}

	// Validate version if specified
	if req.SchemaVersion != "" {
		version, err := s.repo.GetVersion(ctx, req.SchemaID, req.SchemaVersion)
		if err != nil {
			return fmt.Errorf("failed to get version: %w", err)
		}
		if version == nil {
			return fmt.Errorf("schema version '%s' not found", req.SchemaVersion)
		}
	}

	assignment := &EndpointSchema{
		EndpointID:     endpointID,
		SchemaID:       req.SchemaID,
		SchemaVersion:  req.SchemaVersion,
		ValidationMode: req.ValidationMode,
	}

	return s.repo.AssignSchemaToEndpoint(ctx, assignment)
}

// RemoveSchemaFromEndpoint removes schema assignment from an endpoint
func (s *Service) RemoveSchemaFromEndpoint(ctx context.Context, endpointID string) error {
	return s.repo.RemoveSchemaFromEndpoint(ctx, endpointID)
}

// GetEndpointSchema gets the schema assignment for an endpoint
func (s *Service) GetEndpointSchema(ctx context.Context, endpointID string) (*EndpointSchema, error) {
	return s.repo.GetEndpointSchema(ctx, endpointID)
}

// ValidatePayload validates a payload against the schema assigned to an endpoint
func (s *Service) ValidatePayload(ctx context.Context, tenantID, endpointID string, payload []byte) (*ValidationResult, error) {
	return s.validator.Validate(ctx, tenantID, endpointID, payload)
}

// ValidatePayloadDirect validates a payload directly against a schema
func (s *Service) ValidatePayloadDirect(ctx context.Context, tenantID, schemaID string, payload []byte) (*ValidationResult, error) {
	schema, err := s.repo.GetSchema(ctx, tenantID, schemaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}
	if schema == nil {
		return nil, fmt.Errorf("schema not found")
	}

	result := s.validator.ValidatePayloadDirect(payload, schema.JSONSchema)
	result.SchemaID = schema.ID
	result.SchemaName = schema.Name
	result.Version = schema.Version

	return result, nil
}
