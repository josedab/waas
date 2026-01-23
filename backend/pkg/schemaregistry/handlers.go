package schemaregistry

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for schema registry
type Handler struct {
	service *Service
}

// NewHandler creates a new schema registry handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers schema registry routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	schemas := router.Group("/schemas/registry")
	{
		schemas.POST("", h.RegisterSchema)
		schemas.GET("", h.ListSchemas)
		schemas.GET("/stats", h.GetStats)
		schemas.GET("/subjects/:subject", h.GetSchemaBySubject)
		schemas.GET("/subjects/:subject/versions", h.ListVersions)
		schemas.GET("/:id", h.GetSchema)
		schemas.POST("/compatibility/check", h.CheckCompatibility)
		schemas.POST("/:id/deprecate", h.DeprecateSchema)
	}
}

// @Summary Register a new schema
// @Tags SchemaRegistry
// @Accept json
// @Produce json
// @Param body body RegisterSchemaRequest true "Schema registration request"
// @Success 201 {object} SchemaDefinition
// @Router /schemas/registry [post]
func (h *Handler) RegisterSchema(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req RegisterSchemaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	schema, err := h.service.RegisterSchema(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "REGISTER_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusCreated, schema)
}

// @Summary List schemas
// @Tags SchemaRegistry
// @Produce json
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} map[string][]SchemaDefinition
// @Router /schemas/registry [get]
func (h *Handler) ListSchemas(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	schemas, err := h.service.ListSchemas(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "LIST_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"schemas": schemas})
}

// @Summary Get schema registry stats
// @Tags SchemaRegistry
// @Produce json
// @Success 200 {object} SchemaStats
// @Router /schemas/registry/stats [get]
func (h *Handler) GetStats(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	stats, err := h.service.GetStats(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "STATS_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// @Summary Get schema by subject
// @Tags SchemaRegistry
// @Produce json
// @Param subject path string true "Schema subject"
// @Success 200 {object} SchemaDefinition
// @Router /schemas/registry/subjects/{subject} [get]
func (h *Handler) GetSchemaBySubject(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	subject := c.Param("subject")

	schema, err := h.service.GetSchemaBySubject(c.Request.Context(), tenantID, subject)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, schema)
}

// @Summary List schema versions
// @Tags SchemaRegistry
// @Produce json
// @Param subject path string true "Schema subject"
// @Success 200 {object} map[string][]SchemaVersion
// @Router /schemas/registry/subjects/{subject}/versions [get]
func (h *Handler) ListVersions(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	subject := c.Param("subject")

	versions, err := h.service.ListVersions(c.Request.Context(), tenantID, subject)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "LIST_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"versions": versions})
}

// @Summary Get schema by ID
// @Tags SchemaRegistry
// @Produce json
// @Param id path string true "Schema ID"
// @Success 200 {object} SchemaDefinition
// @Router /schemas/registry/{id} [get]
func (h *Handler) GetSchema(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	schemaID := c.Param("id")

	schema, err := h.service.GetSchema(c.Request.Context(), tenantID, schemaID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, schema)
}

// @Summary Check schema compatibility
// @Tags SchemaRegistry
// @Accept json
// @Produce json
// @Param body body CheckCompatibilityRequest true "Compatibility check request"
// @Success 200 {object} CompatibilityResult
// @Router /schemas/registry/compatibility/check [post]
func (h *Handler) CheckCompatibility(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CheckCompatibilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	result, err := h.service.CheckCompatibility(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "COMPATIBILITY_CHECK_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, result)
}

// @Summary Deprecate a schema
// @Tags SchemaRegistry
// @Produce json
// @Param id path string true "Schema ID"
// @Success 200 {object} SchemaDefinition
// @Router /schemas/registry/{id}/deprecate [post]
func (h *Handler) DeprecateSchema(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	schemaID := c.Param("id")

	schema, err := h.service.DeprecateSchema(c.Request.Context(), tenantID, schemaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "DEPRECATE_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, schema)
}
