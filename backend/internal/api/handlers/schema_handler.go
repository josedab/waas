package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/josedab/waas/pkg/schema"
	"github.com/josedab/waas/pkg/utils"

	"github.com/gin-gonic/gin"
)

// SchemaHandler handles schema registry HTTP requests
type SchemaHandler struct {
	service *schema.Service
	logger  *utils.Logger
}

// NewSchemaHandler creates a new schema handler
func NewSchemaHandler(service *schema.Service, logger *utils.Logger) *SchemaHandler {
	return &SchemaHandler{
		service: service,
		logger:  logger,
	}
}

// CreateSchemaRequest represents a schema creation request
type CreateSchemaRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Version     string                 `json:"version" binding:"required"`
	Description string                 `json:"description"`
	JSONSchema  map[string]interface{} `json:"json_schema" binding:"required"`
}

// UpdateSchemaRequest represents a schema update request
type UpdateSchemaRequest struct {
	Description string `json:"description"`
	IsActive    *bool  `json:"is_active"`
	IsDefault   *bool  `json:"is_default"`
}

// CreateVersionRequest represents a new schema version request
type CreateVersionRequest struct {
	Version    string                 `json:"version" binding:"required"`
	JSONSchema map[string]interface{} `json:"json_schema" binding:"required"`
	Changelog  string                 `json:"changelog"`
}

// AssignSchemaRequest represents schema assignment to endpoint
type AssignSchemaRequest struct {
	SchemaID       string `json:"schema_id" binding:"required"`
	SchemaVersion  string `json:"schema_version"`
	ValidationMode string `json:"validation_mode" binding:"required"`
}

// ValidatePayloadRequest represents a payload validation request
type ValidatePayloadRequest struct {
	Payload map[string]interface{} `json:"payload" binding:"required"`
}

// CreateSchema creates a new schema
// @Summary Create a new schema
// @Tags schemas
// @Accept json
// @Produce json
// @Param request body CreateSchemaRequest true "Schema creation request"
// @Success 201 {object} schema.Schema
// @Failure 400 {object} map[string]interface{}
// @Router /schemas [post]
func (h *SchemaHandler) CreateSchema(c *gin.Context) {
	var req CreateSchemaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "INVALID_REQUEST", "Invalid request body")
		return
	}

	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	schemaJSON, err := json.Marshal(req.JSONSchema)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON schema format"})
		return
	}
	createReq := &schema.CreateSchemaRequest{
		Name:        req.Name,
		Version:     req.Version,
		Description: req.Description,
		JSONSchema:  schemaJSON,
	}

	s, err := h.service.CreateSchema(c.Request.Context(), tenantID, createReq)
	if err != nil {
		h.logger.Error("Failed to create schema", map[string]interface{}{"error": err.Error()})
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, s)
}

// GetSchema retrieves a schema by ID
// @Summary Get schema by ID
// @Tags schemas
// @Produce json
// @Param id path string true "Schema ID"
// @Success 200 {object} schema.Schema
// @Failure 404 {object} map[string]interface{}
// @Router /schemas/{id} [get]
func (h *SchemaHandler) GetSchema(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	s, err := h.service.GetSchema(c.Request.Context(), tenantID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schema not found"})
		return
	}

	c.JSON(http.StatusOK, s)
}

// ListSchemas lists all schemas for a tenant
// @Summary List schemas
// @Tags schemas
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {array} schema.Schema
// @Router /schemas [get]
func (h *SchemaHandler) ListSchemas(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit := ParseQueryInt(c, "limit", 50)
	offset := ParseQueryInt(c, "offset", 0)

	schemas, _, err := h.service.ListSchemas(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, schemas)
}

// UpdateSchema updates a schema
// @Summary Update schema
// @Tags schemas
// @Accept json
// @Produce json
// @Param id path string true "Schema ID"
// @Param request body UpdateSchemaRequest true "Update request"
// @Success 200 {object} schema.Schema
// @Router /schemas/{id} [patch]
func (h *SchemaHandler) UpdateSchema(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	var req UpdateSchemaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "INVALID_REQUEST", "Invalid request body")
		return
	}

	updateReq := &schema.UpdateSchemaRequest{
		Description: req.Description,
	}
	if req.IsActive != nil {
		updateReq.IsActive = *req.IsActive
	}
	if req.IsDefault != nil {
		updateReq.IsDefault = *req.IsDefault
	}

	s, err := h.service.UpdateSchema(c.Request.Context(), tenantID, id, updateReq)
	if err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, s)
}

// DeleteSchema deletes a schema
// @Summary Delete schema
// @Tags schemas
// @Param id path string true "Schema ID"
// @Success 204
// @Router /schemas/{id} [delete]
func (h *SchemaHandler) DeleteSchema(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	if err := h.service.DeleteSchema(c.Request.Context(), tenantID, id); err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// CreateSchemaVersion creates a new version of a schema
// @Summary Create new schema version
// @Tags schemas
// @Accept json
// @Produce json
// @Param id path string true "Schema ID"
// @Param request body CreateVersionRequest true "Version request"
// @Success 201 {object} schema.SchemaVersion
// @Router /schemas/{id}/versions [post]
func (h *SchemaHandler) CreateSchemaVersion(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	var req CreateVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "INVALID_REQUEST", "Invalid request body")
		return
	}

	schemaJSON, err := json.Marshal(req.JSONSchema)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON schema format"})
		return
	}
	createReq := &schema.CreateVersionRequest{
		Version:    req.Version,
		JSONSchema: schemaJSON,
		Changelog:  req.Changelog,
	}

	version, _, err := h.service.CreateVersion(c.Request.Context(), tenantID, id, createReq)
	if err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, version)
}

// GetSchemaVersions lists versions for a schema
// @Summary List schema versions
// @Tags schemas
// @Produce json
// @Param id path string true "Schema ID"
// @Success 200 {array} schema.SchemaVersion
// @Router /schemas/{id}/versions [get]
func (h *SchemaHandler) GetSchemaVersions(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	versions, err := h.service.ListVersions(c.Request.Context(), tenantID, id)
	if err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, versions)
}

// AssignSchemaToEndpoint assigns a schema to an endpoint
// @Summary Assign schema to endpoint
// @Tags schemas
// @Accept json
// @Produce json
// @Param endpoint_id path string true "Endpoint ID"
// @Param request body AssignSchemaRequest true "Assignment request"
// @Success 200 {object} map[string]interface{}
// @Router /endpoints/{endpoint_id}/schema [post]
func (h *SchemaHandler) AssignSchemaToEndpoint(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	endpointID := c.Param("endpoint_id")

	var req AssignSchemaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "INVALID_REQUEST", "Invalid request body")
		return
	}

	assignReq := &schema.AssignSchemaRequest{
		SchemaID:       req.SchemaID,
		SchemaVersion:  req.SchemaVersion,
		ValidationMode: req.ValidationMode,
	}

	if err := h.service.AssignSchemaToEndpoint(c.Request.Context(), tenantID, endpointID, assignReq); err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "schema assigned"})
}

// RemoveSchemaFromEndpoint removes schema from endpoint
// @Summary Remove schema from endpoint
// @Tags schemas
// @Param endpoint_id path string true "Endpoint ID"
// @Success 204
// @Router /endpoints/{endpoint_id}/schema [delete]
func (h *SchemaHandler) RemoveSchemaFromEndpoint(c *gin.Context) {
	endpointID := c.Param("endpoint_id")

	if err := h.service.RemoveSchemaFromEndpoint(c.Request.Context(), endpointID); err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ValidatePayload validates a payload against a schema
// @Summary Validate payload against schema
// @Tags schemas
// @Accept json
// @Produce json
// @Param id path string true "Schema ID"
// @Param request body ValidatePayloadRequest true "Payload to validate"
// @Success 200 {object} schema.ValidationResult
// @Router /schemas/{id}/validate [post]
func (h *SchemaHandler) ValidatePayload(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	var req ValidatePayloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "INVALID_REQUEST", "Invalid request body")
		return
	}

	payloadBytes, err := json.Marshal(req.Payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload format"})
		return
	}
	result, err := h.service.ValidatePayloadDirect(c.Request.Context(), tenantID, id, payloadBytes)
	if err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// ValidateForEndpoint validates a payload for an endpoint's assigned schema
// @Summary Validate payload for endpoint
// @Tags schemas
// @Accept json
// @Produce json
// @Param endpoint_id path string true "Endpoint ID"
// @Param request body ValidatePayloadRequest true "Payload to validate"
// @Success 200 {object} schema.ValidationResult
// @Router /endpoints/{endpoint_id}/validate [post]
func (h *SchemaHandler) ValidateForEndpoint(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	endpointID := c.Param("endpoint_id")

	var req ValidatePayloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "INVALID_REQUEST", "Invalid request body")
		return
	}

	payloadBytes, err := json.Marshal(req.Payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload format"})
		return
	}
	result, err := h.service.ValidatePayload(c.Request.Context(), tenantID, endpointID, payloadBytes)
	if err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// RegisterSchemaRoutes registers schema routes
func RegisterSchemaRoutes(r *gin.RouterGroup, h *SchemaHandler) {
	schemas := r.Group("/schemas")
	{
		schemas.POST("", h.CreateSchema)
		schemas.GET("", h.ListSchemas)
		schemas.GET("/:id", h.GetSchema)
		schemas.PATCH("/:id", h.UpdateSchema)
		schemas.DELETE("/:id", h.DeleteSchema)
		schemas.POST("/:id/versions", h.CreateSchemaVersion)
		schemas.GET("/:id/versions", h.GetSchemaVersions)
		schemas.POST("/:id/validate", h.ValidatePayload)
	}

	// Endpoint schema assignment routes
	r.POST("/endpoints/:endpoint_id/schema", h.AssignSchemaToEndpoint)
	r.DELETE("/endpoints/:endpoint_id/schema", h.RemoveSchemaFromEndpoint)
	r.POST("/endpoints/:endpoint_id/validate", h.ValidateForEndpoint)
}
