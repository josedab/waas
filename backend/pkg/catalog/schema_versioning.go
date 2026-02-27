package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SchemaVersioningService manages schema CRUD with semantic versioning
type SchemaVersioningService struct {
	repo      CatalogRepository
	compat    *CompatibilityChecker
	validator *PublishTimeValidator
	docGen    *AutoDocGenerator
}

// NewSchemaVersioningService creates a new schema versioning service
func NewSchemaVersioningService(repo CatalogRepository) *SchemaVersioningService {
	return &SchemaVersioningService{
		repo:      repo,
		compat:    NewCompatibilityChecker(),
		validator: NewPublishTimeValidator(),
		docGen:    NewAutoDocGenerator(),
	}
}

// SemanticVersion represents a parsed semver
type SemanticVersion struct {
	Major int    `json:"major"`
	Minor int    `json:"minor"`
	Patch int    `json:"patch"`
	Label string `json:"label,omitempty"` // e.g., "beta.1"
}

// String returns the string representation of the version
func (v SemanticVersion) String() string {
	base := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Label != "" {
		return base + "-" + v.Label
	}
	return base
}

// ParseSemanticVersion parses a semver string
func ParseSemanticVersion(s string) (*SemanticVersion, error) {
	re := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:-(.+))?$`)
	matches := re.FindStringSubmatch(s)
	if matches == nil {
		return nil, fmt.Errorf("invalid semantic version: %s", s)
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])

	return &SemanticVersion{
		Major: major,
		Minor: minor,
		Patch: patch,
		Label: matches[4],
	}, nil
}

// CompareVersions returns -1, 0, or 1
func CompareVersions(a, b *SemanticVersion) int {
	if a.Major != b.Major {
		if a.Major < b.Major {
			return -1
		}
		return 1
	}
	if a.Minor != b.Minor {
		if a.Minor < b.Minor {
			return -1
		}
		return 1
	}
	if a.Patch != b.Patch {
		if a.Patch < b.Patch {
			return -1
		}
		return 1
	}
	return 0
}

// PublishVersionRequest contains the data for publishing a new schema version
type PublishNewVersionRequest struct {
	EventTypeID      uuid.UUID       `json:"event_type_id" binding:"required"`
	Version          string          `json:"version" binding:"required"`
	Schema           json.RawMessage `json:"schema,omitempty"`
	Changelog        string          `json:"changelog,omitempty"`
	IsBreakingChange bool            `json:"is_breaking_change"`
}

// PublishVersionResult contains the result of a version publish
type PublishVersionResult struct {
	Version              *EventVersion `json:"version"`
	CompatibleBackward   bool          `json:"compatible_backward"`
	CompatibleForward    bool          `json:"compatible_forward"`
	CompatIssues         []string      `json:"compat_issues,omitempty"`
	AutoDetectedBreaking bool          `json:"auto_detected_breaking"`
	Documentation        string        `json:"documentation,omitempty"`
}

// PublishVersion publishes a new version of an event type with schema validation
func (s *SchemaVersioningService) PublishVersion(ctx context.Context, tenantID uuid.UUID, req *PublishNewVersionRequest) (*PublishVersionResult, error) {
	// Parse and validate version
	newVer, err := ParseSemanticVersion(req.Version)
	if err != nil {
		return nil, fmt.Errorf("invalid version format: %w", err)
	}

	// Get current event type
	eventType, err := s.repo.GetEventType(ctx, req.EventTypeID)
	if err != nil {
		return nil, fmt.Errorf("event type not found: %w", err)
	}

	// Verify version is newer
	if eventType.Version != "" {
		currentVer, err := ParseSemanticVersion(eventType.Version)
		if err == nil && CompareVersions(newVer, currentVer) <= 0 {
			return nil, fmt.Errorf("new version %s must be greater than current %s", req.Version, eventType.Version)
		}
	}

	result := &PublishVersionResult{}

	// Check schema compatibility with previous version
	if req.Schema != nil && eventType.Schema != nil && eventType.Schema.Schema != nil {
		backCompat, backIssues := s.compat.CheckBackwardCompatibility(eventType.Schema.Schema, req.Schema)
		fwdCompat, fwdIssues := s.compat.CheckForwardCompatibility(eventType.Schema.Schema, req.Schema)

		result.CompatibleBackward = backCompat
		result.CompatibleForward = fwdCompat

		var allIssues []string
		allIssues = append(allIssues, backIssues...)
		allIssues = append(allIssues, fwdIssues...)
		result.CompatIssues = deduplicateStrings(allIssues)

		// Auto-detect breaking changes
		if !backCompat {
			result.AutoDetectedBreaking = true
			if !req.IsBreakingChange {
				req.IsBreakingChange = true
			}
		}

		// Breaking changes should bump major version
		if req.IsBreakingChange && newVer.Major == 0 {
			currentVer, _ := ParseSemanticVersion(eventType.Version)
			if currentVer != nil && newVer.Major <= currentVer.Major {
				// Warning only - don't block
			}
		}
	}

	// Create the version record
	version := &EventVersion{
		ID:               uuid.New(),
		EventTypeID:      req.EventTypeID,
		Version:          req.Version,
		SchemaID:         eventType.SchemaID,
		Changelog:        req.Changelog,
		IsBreakingChange: req.IsBreakingChange,
		PublishedAt:      time.Now(),
	}

	if err := s.repo.CreateEventVersion(ctx, version); err != nil {
		return nil, fmt.Errorf("failed to create version: %w", err)
	}

	// Update event type version
	eventType.Version = req.Version
	if req.Schema != nil {
		eventType.Schema = &EventSchema{Schema: req.Schema}
	}
	eventType.UpdatedAt = time.Now()

	if err := s.repo.UpdateEventType(ctx, eventType); err != nil {
		return nil, fmt.Errorf("failed to update event type: %w", err)
	}

	result.Version = version

	// Auto-generate documentation
	result.Documentation = s.docGen.GenerateMarkdown(eventType)

	return result, nil
}

// ValidateAndPublish validates a payload and publishes an event
func (s *SchemaVersioningService) ValidateAndPublish(ctx context.Context, tenantID uuid.UUID, eventTypeSlug string, payload json.RawMessage) (*ValidationResult, error) {
	eventType, err := s.repo.GetEventTypeBySlug(ctx, tenantID, eventTypeSlug)
	if err != nil {
		return nil, fmt.Errorf("event type not found: %w", err)
	}

	if eventType.Schema != nil {
		s.validator.RegisterSchema(eventTypeSlug, eventType.Schema)
	}

	config, err := s.repo.GetSchemaValidationConfig(ctx, tenantID)
	if err == nil && config != nil {
		s.validator.SetConfig(tenantID, config)
	}

	result := s.validator.Validate(tenantID, eventTypeSlug, payload)
	return result, nil
}

// GenerateSDKTypes generates type definitions for SDK codegen
func (s *SchemaVersioningService) GenerateSDKTypes(ctx context.Context, tenantID uuid.UUID, language string) (string, error) {
	params := &CatalogSearchParams{
		TenantID: tenantID,
		Status:   StatusActive,
		Limit:    1000,
	}
	searchResult, err := s.repo.SearchEventTypes(ctx, params)
	if err != nil {
		return "", fmt.Errorf("failed to search event types: %w", err)
	}

	switch strings.ToLower(language) {
	case "typescript":
		return generateTypeScriptTypes(searchResult.EventTypes), nil
	case "go":
		return generateGoTypes(searchResult.EventTypes), nil
	case "python":
		return generatePythonTypes(searchResult.EventTypes), nil
	default:
		return "", fmt.Errorf("unsupported language: %s (supported: typescript, go, python)", language)
	}
}

func generateTypeScriptTypes(events []*EventType) string {
	var sb strings.Builder
	sb.WriteString("// Auto-generated by WaaS Event Catalog\n")
	sb.WriteString("// Do not edit manually\n\n")

	for _, et := range events {
		typeName := toPascalCase(et.Slug)
		sb.WriteString(fmt.Sprintf("export interface %sEvent {\n", typeName))
		if et.Schema != nil && len(et.Schema.Properties) > 0 {
			for _, prop := range et.Schema.Properties {
				tsType := toTypeScriptType(prop.Type)
				optional := ""
				if !prop.Required {
					optional = "?"
				}
				sb.WriteString(fmt.Sprintf("  %s%s: %s;\n", prop.Name, optional, tsType))
			}
		}
		sb.WriteString("}\n\n")
	}

	// Generate event type union
	sb.WriteString("export type WebhookEvent =\n")
	for i, et := range events {
		typeName := toPascalCase(et.Slug)
		if i < len(events)-1 {
			sb.WriteString(fmt.Sprintf("  | %sEvent\n", typeName))
		} else {
			sb.WriteString(fmt.Sprintf("  | %sEvent;\n", typeName))
		}
	}

	return sb.String()
}

func generateGoTypes(events []*EventType) string {
	var sb strings.Builder
	sb.WriteString("// Code generated by WaaS Event Catalog. DO NOT EDIT.\n")
	sb.WriteString("package webhooks\n\n")

	for _, et := range events {
		typeName := toPascalCase(et.Slug)
		sb.WriteString(fmt.Sprintf("// %sEvent represents the %s webhook event.\n", typeName, et.Name))
		sb.WriteString(fmt.Sprintf("type %sEvent struct {\n", typeName))
		if et.Schema != nil && len(et.Schema.Properties) > 0 {
			for _, prop := range et.Schema.Properties {
				goType := toGoType(prop.Type)
				tag := fmt.Sprintf("`json:\"%s", prop.Name)
				if !prop.Required {
					tag += ",omitempty"
				}
				tag += "\"`"
				sb.WriteString(fmt.Sprintf("\t%s %s %s\n", toPascalCase(prop.Name), goType, tag))
			}
		}
		sb.WriteString("}\n\n")
	}

	return sb.String()
}

func generatePythonTypes(events []*EventType) string {
	var sb strings.Builder
	sb.WriteString("# Auto-generated by WaaS Event Catalog\n")
	sb.WriteString("# Do not edit manually\n\n")
	sb.WriteString("from dataclasses import dataclass\n")
	sb.WriteString("from typing import Optional\n\n")

	for _, et := range events {
		className := toPascalCase(et.Slug)
		sb.WriteString("@dataclass\n")
		sb.WriteString(fmt.Sprintf("class %sEvent:\n", className))
		sb.WriteString(fmt.Sprintf("    \"\"\"Represents the %s webhook event.\"\"\"\n", et.Name))
		if et.Schema != nil && len(et.Schema.Properties) > 0 {
			for _, prop := range et.Schema.Properties {
				pyType := toPythonType(prop.Type)
				if !prop.Required {
					pyType = fmt.Sprintf("Optional[%s]", pyType)
					sb.WriteString(fmt.Sprintf("    %s: %s = None\n", prop.Name, pyType))
				} else {
					sb.WriteString(fmt.Sprintf("    %s: %s\n", prop.Name, pyType))
				}
			}
		} else {
			sb.WriteString("    pass\n")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func toPascalCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == '.'
	})
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

func toTypeScriptType(t string) string {
	switch t {
	case "string":
		return "string"
	case "integer", "number":
		return "number"
	case "boolean":
		return "boolean"
	case "array":
		return "any[]"
	case "object":
		return "Record<string, any>"
	default:
		return "any"
	}
}

func toGoType(t string) string {
	switch t {
	case "string":
		return "string"
	case "integer":
		return "int64"
	case "number":
		return "float64"
	case "boolean":
		return "bool"
	case "array":
		return "[]interface{}"
	case "object":
		return "map[string]interface{}"
	default:
		return "interface{}"
	}
}

func toPythonType(t string) string {
	switch t {
	case "string":
		return "str"
	case "integer":
		return "int"
	case "number":
		return "float"
	case "boolean":
		return "bool"
	case "array":
		return "list"
	case "object":
		return "dict"
	default:
		return "object"
	}
}

func deduplicateStrings(input []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range input {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
