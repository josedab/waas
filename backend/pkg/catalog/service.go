package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CatalogRepository defines the data access interface for the event catalog
type CatalogRepository interface {
	CreateEventType(ctx context.Context, et *EventType) error
	GetEventType(ctx context.Context, id uuid.UUID) (*EventType, error)
	GetEventTypeBySlug(ctx context.Context, tenantID uuid.UUID, slug string) (*EventType, error)
	GetEventTypeByName(ctx context.Context, tenantID uuid.UUID, name string) (*EventType, error)
	UpdateEventType(ctx context.Context, et *EventType) error
	DeleteEventType(ctx context.Context, id uuid.UUID) error
	SearchEventTypes(ctx context.Context, params *CatalogSearchParams) (*CatalogSearchResult, error)
	CreateEventVersion(ctx context.Context, ev *EventVersion) error
	ListEventVersions(ctx context.Context, eventTypeID uuid.UUID) ([]*EventVersion, error)
	CreateCategory(ctx context.Context, cat *EventCategory) error
	ListCategories(ctx context.Context, tenantID uuid.UUID) ([]*EventCategory, error)
	CreateSubscription(ctx context.Context, sub *EventSubscription) error
	DeleteSubscription(ctx context.Context, endpointID, eventTypeID uuid.UUID) error
	ListEndpointSubscriptions(ctx context.Context, endpointID uuid.UUID) ([]*EventSubscription, error)
	ListEventTypeSubscriptions(ctx context.Context, eventTypeID uuid.UUID) ([]*EventSubscription, error)
	GetSubscriberCount(ctx context.Context, eventTypeID uuid.UUID) (int, error)
	SaveDocumentation(ctx context.Context, doc *EventDocumentation) error
	GetDocumentation(ctx context.Context, eventTypeID uuid.UUID) ([]*EventDocumentation, error)
	GetSchemaValidationConfig(ctx context.Context, tenantID uuid.UUID) (*SchemaValidationConfig, error)
	SaveSchemaValidationConfig(ctx context.Context, config *SchemaValidationConfig) error
}

// Service provides event catalog business logic
type Service struct {
	repo CatalogRepository
}

// NewService creates a new catalog service
func NewService(repo CatalogRepository) *Service {
	return &Service{repo: repo}
}

// CreateEventTypeRequest represents a request to create an event type
type CreateEventTypeRequest struct {
	Name             string          `json:"name" binding:"required"`
	Description      string          `json:"description,omitempty"`
	Category         string          `json:"category,omitempty"`
	SchemaID         *uuid.UUID      `json:"schema_id,omitempty"`
	ExamplePayload   json.RawMessage `json:"example_payload,omitempty"`
	Tags             []string        `json:"tags,omitempty"`
	DocumentationURL string          `json:"documentation_url,omitempty"`
}

// UpdateEventTypeRequest represents a request to update an event type
type UpdateEventTypeRequest struct {
	Name             string          `json:"name,omitempty"`
	Description      string          `json:"description,omitempty"`
	Category         string          `json:"category,omitempty"`
	SchemaID         *uuid.UUID      `json:"schema_id,omitempty"`
	ExamplePayload   json.RawMessage `json:"example_payload,omitempty"`
	Tags             []string        `json:"tags,omitempty"`
	DocumentationURL string          `json:"documentation_url,omitempty"`
}

// DeprecateEventTypeRequest represents a request to deprecate an event type
type DeprecateEventTypeRequest struct {
	Message       string     `json:"message" binding:"required"`
	ReplacementID *uuid.UUID `json:"replacement_id,omitempty"`
}

// PublishVersionRequest represents a request to publish a new version
type PublishVersionRequest struct {
	Version          string     `json:"version" binding:"required"`
	SchemaID         *uuid.UUID `json:"schema_id,omitempty"`
	Changelog        string     `json:"changelog,omitempty"`
	IsBreakingChange bool       `json:"is_breaking_change"`
}

// CreateCategoryRequest represents a request to create a category
type CreateCategoryRequest struct {
	Name        string     `json:"name" binding:"required"`
	Description string     `json:"description,omitempty"`
	Icon        string     `json:"icon,omitempty"`
	Color       string     `json:"color,omitempty"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty"`
}

// SubscribeRequest represents a request to subscribe an endpoint to an event
type SubscribeRequest struct {
	EndpointID       uuid.UUID       `json:"endpoint_id" binding:"required"`
	FilterExpression json.RawMessage `json:"filter_expression,omitempty"`
}

// CreateEventType creates a new event type in the catalog
func (s *Service) CreateEventType(ctx context.Context, tenantID uuid.UUID, req *CreateEventTypeRequest) (*EventType, error) {
	slug := generateSlug(req.Name)

	et := &EventType{
		TenantID:         tenantID,
		Name:             req.Name,
		Slug:             slug,
		Description:      req.Description,
		Category:         req.Category,
		SchemaID:         req.SchemaID,
		Version:          "1.0.0",
		Status:           StatusActive,
		ExamplePayload:   req.ExamplePayload,
		Tags:             req.Tags,
		DocumentationURL: req.DocumentationURL,
	}

	if err := s.repo.CreateEventType(ctx, et); err != nil {
		return nil, err
	}

	// Create initial version record
	ev := &EventVersion{
		EventTypeID:      et.ID,
		Version:          "1.0.0",
		SchemaID:         req.SchemaID,
		Changelog:        "Initial version",
		IsBreakingChange: false,
	}
	if err := s.repo.CreateEventVersion(ctx, ev); err != nil {
		// Log but don't fail — the event type was already created; version can be retried
		_ = err
	}

	return et, nil
}

// GetEventType retrieves an event type by ID
func (s *Service) GetEventType(ctx context.Context, id uuid.UUID) (*EventType, error) {
	et, err := s.repo.GetEventType(ctx, id)
	if err != nil {
		return nil, err
	}

	// Enrich with subscriber count
	count, _ := s.repo.GetSubscriberCount(ctx, id)
	et.Subscribers = count

	// Get versions
	versions, _ := s.repo.ListEventVersions(ctx, id)
	et.Versions = versions

	return et, nil
}

// GetEventTypeBySlug retrieves an event type by slug
func (s *Service) GetEventTypeBySlug(ctx context.Context, tenantID uuid.UUID, slug string) (*EventType, error) {
	return s.repo.GetEventTypeBySlug(ctx, tenantID, slug)
}

// UpdateEventType updates an event type
func (s *Service) UpdateEventType(ctx context.Context, id uuid.UUID, req *UpdateEventTypeRequest) (*EventType, error) {
	et, err := s.repo.GetEventType(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		et.Name = req.Name
	}
	if req.Description != "" {
		et.Description = req.Description
	}
	if req.Category != "" {
		et.Category = req.Category
	}
	if req.SchemaID != nil {
		et.SchemaID = req.SchemaID
	}
	if req.ExamplePayload != nil {
		et.ExamplePayload = req.ExamplePayload
	}
	if req.Tags != nil {
		et.Tags = req.Tags
	}
	if req.DocumentationURL != "" {
		et.DocumentationURL = req.DocumentationURL
	}

	if err := s.repo.UpdateEventType(ctx, et); err != nil {
		return nil, err
	}

	return et, nil
}

// DeleteEventType deletes an event type
func (s *Service) DeleteEventType(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteEventType(ctx, id)
}

// DeprecateEventType marks an event type as deprecated
func (s *Service) DeprecateEventType(ctx context.Context, id uuid.UUID, req *DeprecateEventTypeRequest) (*EventType, error) {
	et, err := s.repo.GetEventType(ctx, id)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	et.Status = StatusDeprecated
	et.DeprecationMessage = req.Message
	et.DeprecatedAt = &now
	et.ReplacementEventID = req.ReplacementID

	if err := s.repo.UpdateEventType(ctx, et); err != nil {
		return nil, err
	}

	return et, nil
}

// PublishVersion publishes a new version of an event type
func (s *Service) PublishVersion(ctx context.Context, id uuid.UUID, req *PublishVersionRequest, publisherID *uuid.UUID) (*EventVersion, error) {
	// Validate version format
	if !isValidSemver(req.Version) {
		return nil, fmt.Errorf("invalid version format, must be semver (e.g., 1.2.3)")
	}

	et, err := s.repo.GetEventType(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update the event type's current version
	et.Version = req.Version
	if req.SchemaID != nil {
		et.SchemaID = req.SchemaID
	}

	if err := s.repo.UpdateEventType(ctx, et); err != nil {
		return nil, err
	}

	// Create version record
	ev := &EventVersion{
		EventTypeID:      id,
		Version:          req.Version,
		SchemaID:         req.SchemaID,
		Changelog:        req.Changelog,
		IsBreakingChange: req.IsBreakingChange,
		PublishedBy:      publisherID,
	}

	if err := s.repo.CreateEventVersion(ctx, ev); err != nil {
		return nil, err
	}

	return ev, nil
}

// SearchEventTypes searches the event catalog
func (s *Service) SearchEventTypes(ctx context.Context, params *CatalogSearchParams) (*CatalogSearchResult, error) {
	return s.repo.SearchEventTypes(ctx, params)
}

// ListCategories lists all event categories
func (s *Service) ListCategories(ctx context.Context, tenantID uuid.UUID) ([]*EventCategory, error) {
	categories, err := s.repo.ListCategories(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Build hierarchy
	categoryMap := make(map[uuid.UUID]*EventCategory)
	var roots []*EventCategory

	for _, cat := range categories {
		categoryMap[cat.ID] = cat
	}

	for _, cat := range categories {
		if cat.ParentID == nil {
			roots = append(roots, cat)
		} else if parent, ok := categoryMap[*cat.ParentID]; ok {
			parent.Children = append(parent.Children, cat)
		}
	}

	return roots, nil
}

// CreateCategory creates a new event category
func (s *Service) CreateCategory(ctx context.Context, tenantID uuid.UUID, req *CreateCategoryRequest) (*EventCategory, error) {
	cat := &EventCategory{
		TenantID:    tenantID,
		Name:        req.Name,
		Slug:        generateSlug(req.Name),
		Description: req.Description,
		Icon:        req.Icon,
		Color:       req.Color,
		ParentID:    req.ParentID,
	}

	if err := s.repo.CreateCategory(ctx, cat); err != nil {
		return nil, err
	}

	return cat, nil
}

// Subscribe subscribes an endpoint to an event type
func (s *Service) Subscribe(ctx context.Context, eventTypeID uuid.UUID, req *SubscribeRequest) (*EventSubscription, error) {
	sub := &EventSubscription{
		EndpointID:       req.EndpointID,
		EventTypeID:      eventTypeID,
		FilterExpression: req.FilterExpression,
		IsActive:         true,
	}

	if err := s.repo.CreateSubscription(ctx, sub); err != nil {
		return nil, err
	}

	return sub, nil
}

// Unsubscribe removes an endpoint subscription from an event type
func (s *Service) Unsubscribe(ctx context.Context, eventTypeID, endpointID uuid.UUID) error {
	return s.repo.DeleteSubscription(ctx, endpointID, eventTypeID)
}

// GetEndpointSubscriptions returns all event subscriptions for an endpoint
func (s *Service) GetEndpointSubscriptions(ctx context.Context, endpointID uuid.UUID) ([]*EventSubscription, error) {
	return s.repo.ListEndpointSubscriptions(ctx, endpointID)
}

// GetVersions returns all versions of an event type
func (s *Service) GetVersions(ctx context.Context, eventTypeID uuid.UUID) ([]*EventVersion, error) {
	return s.repo.ListEventVersions(ctx, eventTypeID)
}

// SaveDocumentation saves documentation for an event type
func (s *Service) SaveDocumentation(ctx context.Context, eventTypeID uuid.UUID, section, content string) error {
	doc := &EventDocumentation{
		EventTypeID: eventTypeID,
		ContentType: "markdown",
		Content:     content,
		Section:     section,
	}
	return s.repo.SaveDocumentation(ctx, doc)
}

// GetDocumentation retrieves all documentation for an event type
func (s *Service) GetDocumentation(ctx context.Context, eventTypeID uuid.UUID) ([]*EventDocumentation, error) {
	return s.repo.GetDocumentation(ctx, eventTypeID)
}

// GenerateOpenAPISpec generates an OpenAPI spec for the event catalog
func (s *Service) GenerateOpenAPISpec(ctx context.Context, tenantID uuid.UUID) (json.RawMessage, error) {
	result, err := s.repo.SearchEventTypes(ctx, &CatalogSearchParams{
		TenantID: tenantID,
		Status:   StatusActive,
		Limit:    1000,
	})
	if err != nil {
		return nil, err
	}

	spec := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":       "Webhook Events API",
			"version":     "1.0.0",
			"description": "Event types and schemas for webhooks",
		},
		"webhooks":   make(map[string]interface{}),
		"components": map[string]interface{}{"schemas": make(map[string]interface{})},
	}

	webhooks := spec["webhooks"].(map[string]interface{})
	schemas := spec["components"].(map[string]interface{})["schemas"].(map[string]interface{})

	for _, et := range result.EventTypes {
		webhookDef := map[string]interface{}{
			"post": map[string]interface{}{
				"summary":     et.Name,
				"description": et.Description,
				"operationId": et.Slug,
				"tags":        et.Tags,
				"requestBody": map[string]interface{}{
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"example": et.ExamplePayload,
						},
					},
				},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{"description": "Event received successfully"},
					"400": map[string]interface{}{"description": "Invalid payload"},
				},
			},
		}

		// Attach full JSON schema if available
		if et.Schema != nil && et.Schema.Schema != nil {
			var schemaDef interface{}
			if json.Unmarshal(et.Schema.Schema, &schemaDef) == nil {
				schemaRef := toGoName(et.Slug)
				schemas[schemaRef] = schemaDef
				webhookDef["post"].(map[string]interface{})["requestBody"].(map[string]interface{})["content"].(map[string]interface{})["application/json"].(map[string]interface{})["schema"] = map[string]interface{}{
					"$ref": "#/components/schemas/" + schemaRef,
				}
			}
		}

		webhooks[et.Slug] = webhookDef
	}

	return json.MarshalIndent(spec, "", "  ")
}

// Helper functions

func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", ".")
	re := regexp.MustCompile(`[^a-z0-9.]`)
	slug = re.ReplaceAllString(slug, "")
	return slug
}

func isValidSemver(version string) bool {
	re := regexp.MustCompile(`^\d+\.\d+\.\d+(-[a-zA-Z0-9.]+)?$`)
	return re.MatchString(version)
}

// ValidatePayloadByEventType validates a webhook payload against the registered schema for a named event type
func (s *Service) ValidatePayloadByEventType(ctx context.Context, tenantID uuid.UUID, eventType string, payload json.RawMessage) (*ValidationResult, error) {
	// Look up the event type by name
	et, err := s.repo.GetEventTypeByName(ctx, tenantID, eventType)
	if err != nil {
		// If no schema registered, validation passes (opt-in)
		return &ValidationResult{Valid: true, EventType: eventType, Message: "no schema registered, skipping validation"}, nil
	}

	// Get schema bytes
	var schemaBytes json.RawMessage
	if et.Schema != nil && et.Schema.Schema != nil {
		schemaBytes = et.Schema.Schema
	} else if et.ExamplePayload != nil {
		schemaBytes = et.ExamplePayload
	} else {
		return &ValidationResult{Valid: true, EventType: eventType, Message: "no schema defined"}, nil
	}

	// Parse the schema
	var schemaDef map[string]interface{}
	if err := json.Unmarshal(schemaBytes, &schemaDef); err != nil {
		return &ValidationResult{Valid: false, EventType: eventType, Message: "invalid schema definition"}, nil
	}

	// Parse the payload
	var payloadData map[string]interface{}
	if err := json.Unmarshal(payload, &payloadData); err != nil {
		return &ValidationResult{Valid: false, EventType: eventType, Message: "payload is not valid JSON"}, nil
	}

	// Validate required fields and types from schema
	_, issues := validateAgainstSchema(payloadData, schemaDef)
	if len(issues) > 0 {
		return &ValidationResult{
			Valid:     false,
			EventType: eventType,
			Errors:    issues,
			Message:   fmt.Sprintf("validation failed: %d error(s)", len(issues)),
		}, nil
	}

	return &ValidationResult{Valid: true, EventType: eventType, Message: "payload is valid"}, nil
}

// ValidatePayload validates a JSON payload against the event type's schema
func (s *Service) ValidatePayload(ctx context.Context, eventTypeID uuid.UUID, payload json.RawMessage) (bool, []string) {
	et, err := s.repo.GetEventType(ctx, eventTypeID)
	if err != nil {
		return false, []string{fmt.Sprintf("event type not found: %v", err)}
	}

	var schemaBytes json.RawMessage
	if et.Schema != nil && et.Schema.Schema != nil {
		schemaBytes = et.Schema.Schema
	} else if et.ExamplePayload != nil {
		// Use example payload structure for validation
		schemaBytes = et.ExamplePayload
	} else {
		return true, nil // No schema to validate against
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		return false, []string{"invalid schema definition"}
	}

	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return false, []string{"invalid JSON payload"}
	}

	return validateAgainstSchema(data, schema)
}

// CheckCompatibility checks schema compatibility between versions
func (s *Service) CheckCompatibility(ctx context.Context, eventTypeID uuid.UUID, newSchema json.RawMessage, mode string) (bool, []string, error) {
	et, err := s.repo.GetEventType(ctx, eventTypeID)
	if err != nil {
		return false, nil, fmt.Errorf("event type not found: %w", err)
	}

	var oldSchema json.RawMessage
	if et.Schema != nil && et.Schema.Schema != nil {
		oldSchema = et.Schema.Schema
	} else {
		return true, nil, nil // No existing schema, always compatible
	}

	checker := NewCompatibilityChecker()
	switch mode {
	case "backward":
		compatible, issues := checker.CheckBackwardCompatibility(oldSchema, newSchema)
		return compatible, issues, nil
	case "forward":
		compatible, issues := checker.CheckForwardCompatibility(oldSchema, newSchema)
		return compatible, issues, nil
	case "full":
		compatible, issues := checker.CheckFullCompatibility(oldSchema, newSchema)
		return compatible, issues, nil
	default:
		return true, nil, nil
	}
}

// GenerateSDKTypes generates typed code for the given event type and language
func (s *Service) GenerateSDKTypes(ctx context.Context, eventTypeID uuid.UUID, language string) (string, error) {
	et, err := s.repo.GetEventType(ctx, eventTypeID)
	if err != nil {
		return "", fmt.Errorf("event type not found: %w", err)
	}

	switch language {
	case LangGo:
		return GenerateGoTypes(et)
	case LangPython:
		return GeneratePythonTypes(et)
	case LangTypeScript:
		return GenerateTypeScriptTypes(et)
	case LangJava:
		return GenerateJavaTypes(et)
	case LangRuby:
		return GenerateRubyTypes(et)
	case LangPHP:
		return GeneratePHPTypes(et)
	case LangCSharp:
		return GenerateCSharpTypes(et)
	default:
		return "", fmt.Errorf("unsupported language: %s", language)
	}
}

// ListSubscribers returns all active subscriptions for an event type
func (s *Service) ListSubscribers(ctx context.Context, eventTypeID uuid.UUID) ([]*EventSubscription, error) {
	return s.repo.ListEventTypeSubscriptions(ctx, eventTypeID)
}

// validateAgainstSchema performs basic JSON Schema validation
func validateAgainstSchema(data map[string]interface{}, schema map[string]interface{}) (bool, []string) {
	var issues []string

	// Check required fields
	if required, ok := schema["required"].([]interface{}); ok {
		for _, r := range required {
			field, _ := r.(string)
			if _, exists := data[field]; !exists {
				issues = append(issues, fmt.Sprintf("missing required field '%s'", field))
			}
		}
	}

	// Check property types
	props, _ := schema["properties"].(map[string]interface{})
	for field, val := range data {
		propDef, exists := props[field]
		if !exists {
			continue
		}
		propMap, ok := propDef.(map[string]interface{})
		if !ok {
			continue
		}
		expectedType, _ := propMap["type"].(string)
		if expectedType == "" {
			continue
		}
		actualType := inferGoJSONType(val)
		if !isTypeCompatible(expectedType, actualType) {
			issues = append(issues, fmt.Sprintf("field '%s' expected type '%s', got '%s'", field, expectedType, actualType))
		}
	}

	return len(issues) == 0, issues
}

func inferGoJSONType(val interface{}) string {
	switch val.(type) {
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	case nil:
		return "null"
	default:
		return "unknown"
	}
}

func isTypeCompatible(expected, actual string) bool {
	if expected == actual {
		return true
	}
	if expected == "integer" && actual == "number" {
		return true
	}
	if expected == "number" && actual == "integer" {
		return true
	}
	return false
}

// ValidatePayloadWithMode validates a payload with a specific validation mode
func (s *Service) ValidatePayloadWithMode(ctx context.Context, eventTypeID uuid.UUID, payload json.RawMessage, mode ValidationMode) *ValidationResult {
	result := &ValidationResult{
		Mode: string(mode),
	}

	if mode == ValidationModeNone {
		result.Valid = true
		return result
	}

	valid, issues := s.ValidatePayload(ctx, eventTypeID, payload)
	result.Valid = valid

	if mode == ValidationModeWarn {
		result.Warnings = issues
		result.Valid = true // Warn mode always passes
	} else {
		result.Issues = issues
	}

	return result
}

// GenerateChangelog generates a changelog for an event type from its versions
func (s *Service) GenerateChangelog(ctx context.Context, eventTypeID uuid.UUID) (*EventChangelog, error) {
	et, err := s.repo.GetEventType(ctx, eventTypeID)
	if err != nil {
		return nil, fmt.Errorf("event type not found: %w", err)
	}

	versions, err := s.repo.ListEventVersions(ctx, eventTypeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get versions: %w", err)
	}

	changelog := &EventChangelog{
		EventTypeID: eventTypeID,
		EventName:   et.Name,
	}

	for _, v := range versions {
		entry := ChangelogEntry{
			Version:  v.Version,
			Date:     v.PublishedAt,
			Breaking: v.IsBreakingChange,
		}

		if v.Changelog != "" {
			entry.Changes = strings.Split(v.Changelog, "\n")
		} else {
			entry.Changes = []string{"Version " + v.Version + " published"}
		}

		changelog.Entries = append(changelog.Entries, entry)
	}

	return changelog, nil
}

// GetBreakingChangeNotifications returns breaking change notifications for a tenant
func (s *Service) GetBreakingChangeNotifications(ctx context.Context, tenantID uuid.UUID) ([]BreakingChangeNotification, error) {
	result, err := s.repo.SearchEventTypes(ctx, &CatalogSearchParams{
		TenantID: tenantID,
		Limit:    1000,
	})
	if err != nil {
		return nil, err
	}

	var notifications []BreakingChangeNotification
	for _, et := range result.EventTypes {
		versions, _ := s.repo.ListEventVersions(ctx, et.ID)
		subscriberCount, _ := s.repo.GetSubscriberCount(ctx, et.ID)

		for _, v := range versions {
			if v.IsBreakingChange {
				notifications = append(notifications, BreakingChangeNotification{
					ID:            v.ID,
					EventTypeID:   et.ID,
					EventName:     et.Name,
					NewVersion:    v.Version,
					Description:   v.Changelog,
					AffectedCount: subscriberCount,
					NotifiedAt:    v.PublishedAt,
				})
			}
		}
	}

	return notifications, nil
}

// GenerateDocPortal generates a documentation portal page for an event type
func (s *Service) GenerateDocPortal(ctx context.Context, eventTypeID uuid.UUID) (*DocPortalPage, error) {
	et, err := s.GetEventType(ctx, eventTypeID)
	if err != nil {
		return nil, err
	}

	docs, _ := s.GetDocumentation(ctx, eventTypeID)
	versions, _ := s.GetVersions(ctx, eventTypeID)
	changelog, _ := s.GenerateChangelog(ctx, eventTypeID)
	subscriberCount, _ := s.repo.GetSubscriberCount(ctx, eventTypeID)

	// Generate example code for supported languages
	exampleCode := make(map[string]string)
	for _, lang := range []string{LangGo, LangPython, LangTypeScript, LangJava, LangRuby, LangPHP, LangCSharp} {
		code, err := s.GenerateSDKTypes(ctx, eventTypeID, lang)
		if err == nil {
			exampleCode[lang] = code
		}
	}

	return &DocPortalPage{
		EventType:       et,
		Documentation:   docs,
		Schema:          et.Schema,
		Versions:        versions,
		ExampleCode:     exampleCode,
		Changelog:       changelog,
		SubscriberCount: subscriberCount,
	}, nil
}

// GetValidationConfig returns the schema validation configuration for a tenant
func (s *Service) GetValidationConfig(ctx context.Context, tenantID uuid.UUID) *SchemaValidationConfig {
	if s.repo != nil {
		if config, err := s.repo.GetSchemaValidationConfig(ctx, tenantID); err == nil {
			return config
		}
	}
	return &SchemaValidationConfig{
		TenantID:        tenantID,
		Mode:            ValidationModeWarn,
		RejectUnknown:   false,
		CoerceTypes:     true,
		MaxPayloadBytes: 1024 * 1024, // 1MB default
	}
}

// UpdateValidationConfig updates the schema validation configuration for a tenant
func (s *Service) UpdateValidationConfig(ctx context.Context, config *SchemaValidationConfig) error {
	return s.repo.SaveSchemaValidationConfig(ctx, config)
}

// ValidateForDelivery validates a payload for the delivery pipeline
func (s *Service) ValidateForDelivery(ctx context.Context, tenantID uuid.UUID, eventType string, payload json.RawMessage) *ValidationResult {
	result := &ValidationResult{}

	// 1. Look up event type by slug
	et, err := s.repo.GetEventTypeBySlug(ctx, tenantID, eventType)
	if err != nil {
		// Unknown event type — pass through
		result.Valid = true
		result.Mode = string(ValidationModeNone)
		return result
	}

	// 2. Get tenant validation config
	config := s.GetValidationConfig(ctx, tenantID)
	result.Mode = string(config.Mode)

	// 3. If mode is "none", return valid
	if config.Mode == ValidationModeNone {
		result.Valid = true
		return result
	}

	// 4. Validate payload against schema
	valid, issues := s.ValidatePayload(ctx, et.ID, payload)

	// 5. Return result with mode-appropriate handling
	if config.Mode == ValidationModeWarn {
		result.Valid = true
		result.Warnings = issues
	} else {
		result.Valid = valid
		result.Issues = issues
	}

	return result
}
