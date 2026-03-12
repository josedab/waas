package schemaregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/httputil"
)

// EventCatalogEntry represents a cataloged event type with schema binding.
type EventCatalogEntry struct {
	ID          string    `json:"id" db:"id"`
	TenantID    string    `json:"tenant_id" db:"tenant_id"`
	EventType   string    `json:"event_type" db:"event_type"`
	Description string    `json:"description,omitempty" db:"description"`
	SchemaID    string    `json:"schema_id,omitempty" db:"schema_id"`
	Version     string    `json:"version" db:"version"`
	Status      string    `json:"status" db:"status"`
	Owner       string    `json:"owner,omitempty" db:"owner"`
	Tags        []string  `json:"tags,omitempty"`
	Examples    []string  `json:"examples,omitempty"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// CreateCatalogEntryRequest is the request DTO for creating a catalog entry.
type CreateCatalogEntryRequest struct {
	EventType   string   `json:"event_type" binding:"required,min=1,max=255"`
	Description string   `json:"description,omitempty"`
	SchemaID    string   `json:"schema_id,omitempty"`
	Version     string   `json:"version,omitempty"`
	Owner       string   `json:"owner,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Examples    []string `json:"examples,omitempty"`
}

// ValidationResult represents the result of validating a payload against a schema.
type ValidationResult struct {
	IsValid     bool      `json:"is_valid"`
	Errors      []string  `json:"errors,omitempty"`
	SchemaID    string    `json:"schema_id"`
	EventType   string    `json:"event_type"`
	ValidatedAt time.Time `json:"validated_at"`
}

// ValidatePayloadRequest is the request to validate a payload.
type ValidatePayloadRequest struct {
	EventType string          `json:"event_type" binding:"required"`
	Payload   json.RawMessage `json:"payload" binding:"required"`
}

// BreakingChangeReport provides detailed breaking change analysis.
type BreakingChangeReport struct {
	Subject         string   `json:"subject"`
	OldVersion      int      `json:"old_version"`
	NewVersion      int      `json:"new_version"`
	IsBreaking      bool     `json:"is_breaking"`
	RemovedFields   []string `json:"removed_fields,omitempty"`
	TypeChanges     []string `json:"type_changes,omitempty"`
	NewRequired     []string `json:"new_required_fields,omitempty"`
	Recommendations []string `json:"recommendations,omitempty"`
}

// RegisterCatalogEntry adds an event type to the catalog.
func (s *Service) RegisterCatalogEntry(ctx context.Context, tenantID string, req *CreateCatalogEntryRequest) (*EventCatalogEntry, error) {
	if req.EventType == "" {
		return nil, fmt.Errorf("event_type is required")
	}

	version := req.Version
	if version == "" {
		version = "1.0.0"
	}

	entry := &EventCatalogEntry{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		EventType:   req.EventType,
		Description: req.Description,
		SchemaID:    req.SchemaID,
		Version:     version,
		Status:      SchemaStatusActive,
		Owner:       req.Owner,
		Tags:        req.Tags,
		Examples:    req.Examples,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return entry, nil
}

// ValidatePayload validates a payload against its registered schema.
func (s *Service) ValidatePayload(ctx context.Context, tenantID string, req *ValidatePayloadRequest) (*ValidationResult, error) {
	schema, err := s.repo.GetSchemaBySubject(ctx, tenantID, req.EventType)
	if err != nil {
		return &ValidationResult{
			IsValid:     true, // No schema = pass-through
			EventType:   req.EventType,
			ValidatedAt: time.Now(),
		}, nil
	}

	result := &ValidationResult{
		IsValid:     true,
		SchemaID:    schema.ID,
		EventType:   req.EventType,
		ValidatedAt: time.Now(),
	}

	if schema.SchemaFormat == SchemaFormatJSONSchema {
		errors := s.validatePayloadAgainstJSONSchema(schema.SchemaContent, req.Payload)
		if len(errors) > 0 {
			result.IsValid = false
			result.Errors = errors
		}
	}

	return result, nil
}

// DetectBreakingChanges provides detailed breaking change analysis between versions.
func (s *Service) DetectBreakingChanges(ctx context.Context, tenantID, subject string, newContent string) (*BreakingChangeReport, error) {
	existing, err := s.repo.GetLatestVersion(ctx, tenantID, subject)
	if err != nil {
		return nil, fmt.Errorf("no existing schema for subject %s: %w", subject, err)
	}

	report := &BreakingChangeReport{
		Subject:    subject,
		OldVersion: existing.Version,
		NewVersion: existing.Version + 1,
	}

	var oldSchema, newSchema map[string]interface{}
	if err := json.Unmarshal([]byte(existing.SchemaContent), &oldSchema); err != nil {
		return nil, fmt.Errorf("invalid old schema: %w", err)
	}
	if err := json.Unmarshal([]byte(newContent), &newSchema); err != nil {
		return nil, fmt.Errorf("invalid new schema: %w", err)
	}

	oldProps, _ := oldSchema["properties"].(map[string]interface{})
	newProps, _ := newSchema["properties"].(map[string]interface{})

	// Detect removed fields
	for field := range oldProps {
		if _, exists := newProps[field]; !exists {
			report.RemovedFields = append(report.RemovedFields, field)
			report.IsBreaking = true
		}
	}

	// Detect type changes
	for field, oldDef := range oldProps {
		newDef, exists := newProps[field]
		if !exists {
			continue
		}
		oldType := extractType(oldDef)
		newType := extractType(newDef)
		if oldType != "" && newType != "" && oldType != newType {
			report.TypeChanges = append(report.TypeChanges, fmt.Sprintf("%s: %s → %s", field, oldType, newType))
			report.IsBreaking = true
		}
	}

	// Detect new required fields
	oldRequired := extractStringSlice(oldSchema["required"])
	newRequired := extractStringSlice(newSchema["required"])
	for _, field := range newRequired {
		if !contains(oldRequired, field) {
			report.NewRequired = append(report.NewRequired, field)
			report.IsBreaking = true
		}
	}

	// Generate recommendations
	if report.IsBreaking {
		report.Recommendations = append(report.Recommendations, "Consider using a new event type version instead of modifying the existing schema")
		if len(report.RemovedFields) > 0 {
			report.Recommendations = append(report.Recommendations, "Mark removed fields as deprecated before removing them")
		}
		if len(report.NewRequired) > 0 {
			report.Recommendations = append(report.Recommendations, "Add default values for new required fields to maintain backward compatibility")
		}
	}

	return report, nil
}

func (s *Service) validatePayloadAgainstJSONSchema(schemaContent string, payload json.RawMessage) []string {
	var schema map[string]interface{}
	if err := json.Unmarshal([]byte(schemaContent), &schema); err != nil {
		return []string{"invalid schema: " + err.Error()}
	}

	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return []string{"invalid payload JSON: " + err.Error()}
	}

	var errors []string

	// Check required fields
	required := extractStringSlice(schema["required"])
	for _, field := range required {
		if _, exists := data[field]; !exists {
			errors = append(errors, fmt.Sprintf("missing required field: %s", field))
		}
	}

	// Check field types
	props, _ := schema["properties"].(map[string]interface{})
	for field, value := range data {
		propDef, exists := props[field]
		if !exists {
			continue
		}
		expectedType := extractType(propDef)
		if expectedType == "" {
			continue
		}
		actualType := jsonType(value)
		if actualType != expectedType {
			errors = append(errors, fmt.Sprintf("field '%s': expected type '%s', got '%s'", field, expectedType, actualType))
		}
	}

	return errors
}

func jsonType(v interface{}) string {
	switch v.(type) {
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

// RegisterCatalogRoutes registers event catalog API routes.
func (h *Handler) RegisterCatalogRoutes(router *gin.RouterGroup) {
	catalog := router.Group("/schemas/catalog")
	{
		catalog.POST("", h.CreateCatalogEntry)
		catalog.POST("/validate", h.ValidatePayload)
		catalog.POST("/breaking-changes", h.DetectBreakingChanges)
	}
}

func (h *Handler) CreateCatalogEntry(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateCatalogEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httputil.APIErrorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}

	entry, err := h.service.RegisterCatalogEntry(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, entry)
}

func (h *Handler) ValidatePayload(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req ValidatePayloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httputil.APIErrorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}

	result, err := h.service.ValidatePayload(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// DetectBreakingChanges handles POST /schemas/catalog/breaking-changes
func (h *Handler) DetectBreakingChanges(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req struct {
		Subject       string `json:"subject" binding:"required"`
		SchemaContent string `json:"schema_content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httputil.APIErrorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}

	report, err := h.service.DetectBreakingChanges(c.Request.Context(), tenantID, req.Subject, req.SchemaContent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, report)
}
