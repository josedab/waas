package schemaregistry

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service provides schema registry functionality
type Service struct {
	repo Repository
}

// NewService creates a new schema registry service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// RegisterSchema validates and registers a new schema version
func (s *Service) RegisterSchema(ctx context.Context, tenantID string, req *RegisterSchemaRequest) (*SchemaDefinition, error) {
	if err := s.validateSchemaContent(req.SchemaFormat, req.SchemaContent); err != nil {
		return nil, fmt.Errorf("schema validation failed: %w", err)
	}

	fingerprint := s.computeFingerprint(req.SchemaContent)

	compatMode := req.CompatibilityMode
	if compatMode == "" {
		compatMode = CompatibilityBackward
	}

	version := 1
	existing, err := s.repo.GetLatestVersion(ctx, tenantID, req.Subject)
	if err == nil && existing != nil {
		// Check compatibility with previous version
		if existing.CompatibilityMode != CompatibilityNone {
			result := s.checkCompatibility(existing.SchemaContent, req.SchemaContent, existing.CompatibilityMode, existing.Version)
			if !result.IsCompatible {
				return nil, fmt.Errorf("schema is not compatible with version %d: %v", existing.Version, result.BreakingChanges)
			}
		}

		// Mark previous version as not latest
		existing.IsLatest = false
		existing.UpdatedAt = time.Now()
		if err := s.repo.UpdateSchema(ctx, existing); err != nil {
			return nil, fmt.Errorf("failed to update previous version: %w", err)
		}

		version = existing.Version + 1
	}

	schema := &SchemaDefinition{
		ID:                uuid.New().String(),
		TenantID:          tenantID,
		Subject:           req.Subject,
		Version:           version,
		SchemaFormat:      req.SchemaFormat,
		SchemaContent:     req.SchemaContent,
		Fingerprint:       fingerprint,
		Description:       req.Description,
		IsLatest:          true,
		CompatibilityMode: compatMode,
		Status:            SchemaStatusActive,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := s.repo.CreateSchema(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	// Create version record
	schemaVersion := &SchemaVersion{
		ID:            uuid.New().String(),
		SchemaID:      schema.ID,
		Version:       version,
		SchemaContent: req.SchemaContent,
		ChangeLog:     req.Description,
		CreatedAt:     time.Now(),
	}
	if err := s.repo.CreateVersion(ctx, schemaVersion); err != nil {
		return nil, fmt.Errorf("failed to create schema version: %w", err)
	}

	return schema, nil
}

// GetSchema retrieves a schema by ID
func (s *Service) GetSchema(ctx context.Context, tenantID, schemaID string) (*SchemaDefinition, error) {
	return s.repo.GetSchema(ctx, tenantID, schemaID)
}

// GetSchemaBySubject retrieves the latest schema for a subject
func (s *Service) GetSchemaBySubject(ctx context.Context, tenantID, subject string) (*SchemaDefinition, error) {
	return s.repo.GetSchemaBySubject(ctx, tenantID, subject)
}

// ListSchemas lists schemas with pagination
func (s *Service) ListSchemas(ctx context.Context, tenantID string, limit, offset int) ([]SchemaDefinition, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListSchemas(ctx, tenantID, limit, offset)
}

// ListVersions lists all versions of a schema subject
func (s *Service) ListVersions(ctx context.Context, tenantID, subject string) ([]SchemaVersion, error) {
	return s.repo.ListVersions(ctx, tenantID, subject)
}

// CheckCompatibility validates a new schema against the latest version
func (s *Service) CheckCompatibility(ctx context.Context, tenantID string, req *CheckCompatibilityRequest) (*CompatibilityResult, error) {
	if err := s.validateSchemaContent(req.SchemaFormat, req.SchemaContent); err != nil {
		return nil, fmt.Errorf("schema validation failed: %w", err)
	}

	existing, err := s.repo.GetLatestVersion(ctx, tenantID, req.Subject)
	if err != nil {
		return nil, fmt.Errorf("no existing schema found for subject %s: %w", req.Subject, err)
	}

	result := s.checkCompatibility(existing.SchemaContent, req.SchemaContent, existing.CompatibilityMode, existing.Version)
	return &result, nil
}

// DeprecateSchema marks a schema as deprecated
func (s *Service) DeprecateSchema(ctx context.Context, tenantID, schemaID string) (*SchemaDefinition, error) {
	schema, err := s.repo.GetSchema(ctx, tenantID, schemaID)
	if err != nil {
		return nil, fmt.Errorf("schema not found: %w", err)
	}

	schema.Status = SchemaStatusDeprecated
	schema.UpdatedAt = time.Now()

	if err := s.repo.UpdateSchema(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to deprecate schema: %w", err)
	}

	return schema, nil
}

// GetStats retrieves schema registry statistics
func (s *Service) GetStats(ctx context.Context, tenantID string) (*SchemaStats, error) {
	return s.repo.GetStats(ctx, tenantID)
}

func (s *Service) validateSchemaContent(format, content string) error {
	switch format {
	case SchemaFormatJSONSchema:
		return s.validateJSONSchema(content)
	case SchemaFormatAvro:
		return s.validateAvroSchema(content)
	case SchemaFormatProtobuf:
		// Basic protobuf validation: ensure non-empty content
		if len(content) == 0 {
			return fmt.Errorf("protobuf schema content cannot be empty")
		}
		return nil
	default:
		return fmt.Errorf("unsupported schema format: %s", format)
	}
}

func (s *Service) validateJSONSchema(content string) error {
	var schema map[string]interface{}
	if err := json.Unmarshal([]byte(content), &schema); err != nil {
		return fmt.Errorf("invalid JSON schema: %w", err)
	}
	return nil
}

func (s *Service) validateAvroSchema(content string) error {
	var schema map[string]interface{}
	if err := json.Unmarshal([]byte(content), &schema); err != nil {
		return fmt.Errorf("invalid Avro schema: %w", err)
	}
	if _, ok := schema["type"]; !ok {
		return fmt.Errorf("avro schema must have a 'type' field")
	}
	return nil
}

func (s *Service) computeFingerprint(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}

func (s *Service) checkCompatibility(oldContent, newContent, mode string, oldVersion int) CompatibilityResult {
	result := CompatibilityResult{
		IsCompatible: true,
		OldVersion:   oldVersion,
		NewVersion:   oldVersion + 1,
	}

	switch mode {
	case CompatibilityBackward:
		s.checkBackwardCompatibility(oldContent, newContent, &result)
	case CompatibilityForward:
		s.checkForwardCompatibility(oldContent, newContent, &result)
	case CompatibilityFull:
		s.checkBackwardCompatibility(oldContent, newContent, &result)
		s.checkForwardCompatibility(oldContent, newContent, &result)
	case CompatibilityNone:
		// No compatibility checks needed
	}

	return result
}

func (s *Service) checkBackwardCompatibility(oldContent, newContent string, result *CompatibilityResult) {
	var oldSchema, newSchema map[string]interface{}
	if err := json.Unmarshal([]byte(oldContent), &oldSchema); err != nil {
		return
	}
	if err := json.Unmarshal([]byte(newContent), &newSchema); err != nil {
		return
	}

	// Check for removed fields (breaking in backward compatibility)
	oldProps, oldOk := oldSchema["properties"].(map[string]interface{})
	newProps, newOk := newSchema["properties"].(map[string]interface{})

	if oldOk && newOk {
		for field := range oldProps {
			if _, exists := newProps[field]; !exists {
				result.IsCompatible = false
				result.BreakingChanges = append(result.BreakingChanges, fmt.Sprintf("field '%s' was removed", field))
			}
		}
	}

	// Check for new required fields (breaking in backward compatibility)
	oldRequired := extractStringSlice(oldSchema["required"])
	newRequired := extractStringSlice(newSchema["required"])

	for _, field := range newRequired {
		if !contains(oldRequired, field) {
			if _, existedBefore := oldProps[field]; !existedBefore {
				result.IsCompatible = false
				result.BreakingChanges = append(result.BreakingChanges, fmt.Sprintf("new required field '%s' added", field))
			}
		}
	}
}

func (s *Service) checkForwardCompatibility(oldContent, newContent string, result *CompatibilityResult) {
	var oldSchema, newSchema map[string]interface{}
	if err := json.Unmarshal([]byte(oldContent), &oldSchema); err != nil {
		return
	}
	if err := json.Unmarshal([]byte(newContent), &newSchema); err != nil {
		return
	}

	// Check for added required fields without defaults (breaking in forward compatibility)
	newProps, newOk := newSchema["properties"].(map[string]interface{})
	oldProps, oldOk := oldSchema["properties"].(map[string]interface{})

	if newOk && oldOk {
		for field := range oldProps {
			if _, exists := newProps[field]; !exists {
				result.Warnings = append(result.Warnings, fmt.Sprintf("field '%s' removed in new version", field))
			}
		}
	}

	// Check for type changes
	if newOk && oldOk {
		for field, oldDef := range oldProps {
			newDef, exists := newProps[field]
			if !exists {
				continue
			}
			oldType := extractType(oldDef)
			newType := extractType(newDef)
			if oldType != "" && newType != "" && oldType != newType {
				result.IsCompatible = false
				result.BreakingChanges = append(result.BreakingChanges, fmt.Sprintf("field '%s' type changed from '%s' to '%s'", field, oldType, newType))
			}
		}
	}
}

func extractStringSlice(v interface{}) []string {
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	var result []string
	for _, item := range arr {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func extractType(fieldDef interface{}) string {
	m, ok := fieldDef.(map[string]interface{})
	if !ok {
		return ""
	}
	t, ok := m["type"].(string)
	if !ok {
		return ""
	}
	return t
}
