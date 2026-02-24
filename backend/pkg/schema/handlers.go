package schema

import (
	"github.com/josedab/waas/pkg/httputil"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for schema management
type Handler struct {
	service *Service
}

// NewHandler creates a new schema handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers schema management routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	schemas := router.Group("/schemas")
	{
		schemas.POST("", h.CreateSchema)
		schemas.GET("", h.ListSchemas)
		schemas.GET("/:id", h.GetSchema)
		schemas.PUT("/:id", h.UpdateSchema)
		schemas.DELETE("/:id", h.DeleteSchema)
		
		// Versions
		schemas.POST("/:id/versions", h.CreateVersion)
		schemas.GET("/:id/versions", h.ListVersions)
		
		// Validation
		schemas.POST("/:id/validate", h.ValidatePayload)
		schemas.POST("/validate", h.ValidatePayloadDirect)
		
		// Compatibility
		schemas.POST("/:id/compatibility", h.CheckCompatibility)
	}

	// Endpoint schema assignment
	endpoints := router.Group("/endpoints")
	{
		endpoints.POST("/:id/schema", h.AssignSchema)
		endpoints.GET("/:id/schema", h.GetEndpointSchema)
		endpoints.DELETE("/:id/schema", h.RemoveSchema)
	}
}

// CreateSchema godoc
// @Summary Create a new schema
// @Description Create a new JSON schema for webhook payload validation
// @Tags schemas
// @Accept json
// @Produce json
// @Param request body CreateSchemaRequest true "Schema creation request"
// @Success 201 {object} Schema
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /schemas [post]
func (h *Handler) CreateSchema(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateSchemaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	schema, err := h.service.CreateSchema(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, schema)
}

// ListSchemas godoc
// @Summary List schemas
// @Description Get a list of schemas for the tenant
// @Tags schemas
// @Produce json
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /schemas [get]
func (h *Handler) ListSchemas(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := c.Query("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	schemas, total, err := h.service.ListSchemas(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"schemas": schemas,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// GetSchema godoc
// @Summary Get schema details
// @Description Get details of a specific schema
// @Tags schemas
// @Produce json
// @Param id path string true "Schema ID"
// @Success 200 {object} Schema
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /schemas/{id} [get]
func (h *Handler) GetSchema(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	schemaID := c.Param("id")
	schema, err := h.service.GetSchema(c.Request.Context(), tenantID, schemaID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	if schema == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schema not found"})
		return
	}

	c.JSON(http.StatusOK, schema)
}

// UpdateSchema godoc
// @Summary Update schema
// @Description Update an existing schema
// @Tags schemas
// @Accept json
// @Produce json
// @Param id path string true "Schema ID"
// @Param request body UpdateSchemaRequest true "Schema update request"
// @Success 200 {object} Schema
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /schemas/{id} [put]
func (h *Handler) UpdateSchema(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	schemaID := c.Param("id")
	var req UpdateSchemaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	schema, err := h.service.UpdateSchema(c.Request.Context(), tenantID, schemaID, &req)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, schema)
}

// DeleteSchema godoc
// @Summary Delete schema
// @Description Delete a schema
// @Tags schemas
// @Produce json
// @Param id path string true "Schema ID"
// @Success 204 "No content"
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /schemas/{id} [delete]
func (h *Handler) DeleteSchema(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	schemaID := c.Param("id")
	if err := h.service.DeleteSchema(c.Request.Context(), tenantID, schemaID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// CreateVersion godoc
// @Summary Create schema version
// @Description Create a new version of a schema with compatibility check
// @Tags schemas
// @Accept json
// @Produce json
// @Param id path string true "Schema ID"
// @Param request body CreateVersionRequest true "Version creation request"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /schemas/{id}/versions [post]
func (h *Handler) CreateVersion(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	schemaID := c.Param("id")
	var req CreateVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	version, compatibility, err := h.service.CreateVersion(c.Request.Context(), tenantID, schemaID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"version":       version,
		"compatibility": compatibility,
	})
}

// ListVersions godoc
// @Summary List schema versions
// @Description Get all versions of a schema
// @Tags schemas
// @Produce json
// @Param id path string true "Schema ID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /schemas/{id}/versions [get]
func (h *Handler) ListVersions(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	schemaID := c.Param("id")
	versions, err := h.service.ListVersions(c.Request.Context(), tenantID, schemaID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"versions": versions})
}

// ValidatePayload godoc
// @Summary Validate payload against schema
// @Description Validate a payload against a specific schema
// @Tags schemas
// @Accept json
// @Produce json
// @Param id path string true "Schema ID"
// @Param payload body interface{} true "Payload to validate"
// @Success 200 {object} ValidationResult
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /schemas/{id}/validate [post]
func (h *Handler) ValidatePayload(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	schemaID := c.Param("id")
	
	payload, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read payload"})
		return
	}

	result, err := h.service.ValidatePayloadDirect(c.Request.Context(), tenantID, schemaID, payload)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ValidatePayloadDirect validates a payload without requiring a schema ID
type ValidateDirectRequest struct {
	Schema  interface{} `json:"schema" binding:"required"`
	Payload interface{} `json:"payload" binding:"required"`
}

// ValidatePayloadDirect godoc
// @Summary Validate payload against inline schema
// @Description Validate a payload against an inline JSON schema (no storage)
// @Tags schemas
// @Accept json
// @Produce json
// @Param request body ValidateDirectRequest true "Validation request"
// @Success 200 {object} ValidationResult
// @Failure 400 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /schemas/validate [post]
func (h *Handler) ValidatePayloadDirect(c *gin.Context) {
	var req ValidateDirectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// This is a direct validation without storage
	validator := NewValidator(nil)
	
	schemaBytes, _ := c.GetRawData()
	payloadBytes, _ := c.GetRawData()
	
	result := validator.ValidatePayloadDirect(payloadBytes, schemaBytes)
	c.JSON(http.StatusOK, result)
}

// CheckCompatibility godoc
// @Summary Check schema compatibility
// @Description Check if a new schema is compatible with the current version
// @Tags schemas
// @Accept json
// @Produce json
// @Param id path string true "Schema ID"
// @Param schema body interface{} true "New schema to check"
// @Success 200 {object} CompatibilityResult
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /schemas/{id}/compatibility [post]
func (h *Handler) CheckCompatibility(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	schemaID := c.Param("id")
	
	newSchema, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read schema"})
		return
	}

	// Get current schema
	schema, err := h.service.GetSchema(c.Request.Context(), tenantID, schemaID)
	if err != nil || schema == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schema not found"})
		return
	}

	validator := NewValidator(nil)
	result, err := validator.CheckCompatibility(schema.JSONSchema, newSchema)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// AssignSchema godoc
// @Summary Assign schema to endpoint
// @Description Assign a schema to a webhook endpoint for validation
// @Tags schemas
// @Accept json
// @Produce json
// @Param id path string true "Endpoint ID"
// @Param request body AssignSchemaRequest true "Schema assignment request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /endpoints/{id}/schema [post]
func (h *Handler) AssignSchema(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	endpointID := c.Param("id")
	var req AssignSchemaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if err := h.service.AssignSchemaToEndpoint(c.Request.Context(), tenantID, endpointID, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "schema assigned successfully"})
}

// GetEndpointSchema godoc
// @Summary Get endpoint schema assignment
// @Description Get the schema assigned to an endpoint
// @Tags schemas
// @Produce json
// @Param id path string true "Endpoint ID"
// @Success 200 {object} EndpointSchema
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /endpoints/{id}/schema [get]
func (h *Handler) GetEndpointSchema(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	endpointID := c.Param("id")
	assignment, err := h.service.GetEndpointSchema(c.Request.Context(), endpointID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	if assignment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no schema assigned"})
		return
	}

	c.JSON(http.StatusOK, assignment)
}

// RemoveSchema godoc
// @Summary Remove schema from endpoint
// @Description Remove schema assignment from an endpoint
// @Tags schemas
// @Produce json
// @Param id path string true "Endpoint ID"
// @Success 204 "No content"
// @Failure 401 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /endpoints/{id}/schema [delete]
func (h *Handler) RemoveSchema(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	endpointID := c.Param("id")
	if err := h.service.RemoveSchemaFromEndpoint(c.Request.Context(), endpointID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
