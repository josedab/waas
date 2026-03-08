package versioning

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler handles versioning HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new versioning handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers versioning routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	versions := r.Group("/versions")
	{
		// Versions
		versions.POST("", h.CreateVersion)
		versions.GET("/webhook/:webhook_id", h.ListVersions)
		versions.GET("/:id", h.GetVersion)
		versions.POST("/:id/publish", h.PublishVersion)
		versions.POST("/:id/deprecate", h.DeprecateVersion)
		versions.POST("/:id/sunset", h.SunsetVersion)
		versions.GET("/:id/metrics", h.GetVersionMetrics)

		// Compatibility
		versions.GET("/compare", h.CompareVersions)
		versions.GET("/compatibility", h.CheckCompatibility)

		// Migrations
		versions.POST("/migrations", h.StartMigration)
		versions.GET("/migrations/:id", h.GetMigration)
		versions.GET("/migrations/webhook/:webhook_id", h.ListMigrations)

		// Schemas
		versions.POST("/schemas", h.CreateSchema)
		versions.GET("/schemas", h.ListSchemas)
		versions.GET("/schemas/:id", h.GetSchema)

		// Policy
		versions.GET("/policy", h.GetPolicy)
		versions.PUT("/policy", h.UpdatePolicy)

		// Negotiation
		versions.POST("/negotiate", h.NegotiateVersion)
	}
}

// CreateVersion godoc
// @Summary Create version
// @Description Create a new webhook version
// @Tags Versioning
// @Accept json
// @Produce json
// @Param request body CreateVersionRequest true "Version configuration"
// @Success 201 {object} Version
// @Router /versions [post]
func (h *Handler) CreateVersion(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req CreateVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	version, err := h.service.CreateVersion(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, version)
}

// ListVersions godoc
// @Summary List versions
// @Description List all versions for a webhook
// @Tags Versioning
// @Produce json
// @Param webhook_id path string true "Webhook ID"
// @Success 200 {array} Version
// @Router /versions/webhook/{webhook_id} [get]
func (h *Handler) ListVersions(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	webhookID := c.Param("webhook_id")

	versions, err := h.service.ListVersions(c.Request.Context(), tenantID, webhookID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"versions": versions})
}

// GetVersion godoc
// @Summary Get version
// @Description Get version by ID
// @Tags Versioning
// @Produce json
// @Param id path string true "Version ID"
// @Success 200 {object} Version
// @Router /versions/{id} [get]
func (h *Handler) GetVersion(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	versionID := c.Param("id")

	version, err := h.service.GetVersion(c.Request.Context(), tenantID, versionID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, version)
}

// PublishVersion godoc
// @Summary Publish version
// @Description Publish a draft version
// @Tags Versioning
// @Param id path string true "Version ID"
// @Success 200 {object} Version
// @Router /versions/{id}/publish [post]
func (h *Handler) PublishVersion(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	versionID := c.Param("id")

	version, err := h.service.PublishVersion(c.Request.Context(), tenantID, versionID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, version)
}

// DeprecateVersion godoc
// @Summary Deprecate version
// @Description Mark a version as deprecated
// @Tags Versioning
// @Accept json
// @Produce json
// @Param id path string true "Version ID"
// @Param request body DeprecateRequest true "Deprecation options"
// @Success 200 {object} Version
// @Router /versions/{id}/deprecate [post]
func (h *Handler) DeprecateVersion(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	versionID := c.Param("id")

	var req DeprecateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	version, err := h.service.DeprecateVersion(c.Request.Context(), tenantID, versionID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, version)
}

// SunsetVersion godoc
// @Summary Sunset version
// @Description Mark a version as sunset (no longer available)
// @Tags Versioning
// @Param id path string true "Version ID"
// @Success 200 {object} Version
// @Router /versions/{id}/sunset [post]
func (h *Handler) SunsetVersion(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	versionID := c.Param("id")

	version, err := h.service.SunsetVersion(c.Request.Context(), tenantID, versionID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, version)
}

// GetVersionMetrics godoc
// @Summary Get version metrics
// @Description Get usage metrics for a version
// @Tags Versioning
// @Produce json
// @Param id path string true "Version ID"
// @Success 200 {object} VersionMetrics
// @Router /versions/{id}/metrics [get]
func (h *Handler) GetVersionMetrics(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	versionID := c.Param("id")

	metrics, err := h.service.GetVersionMetrics(c.Request.Context(), tenantID, versionID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// CompareVersions godoc
// @Summary Compare versions
// @Description Compare two versions
// @Tags Versioning
// @Produce json
// @Param source query string true "Source version ID"
// @Param target query string true "Target version ID"
// @Success 200 {object} VersionComparison
// @Router /versions/compare [get]
func (h *Handler) CompareVersions(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	sourceID := c.Query("source")
	targetID := c.Query("target")

	if sourceID == "" || targetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source and target required"})
		return
	}

	comparison, err := h.service.CompareVersions(c.Request.Context(), tenantID, sourceID, targetID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, comparison)
}

// CheckCompatibility godoc
// @Summary Check compatibility
// @Description Check schema compatibility between versions
// @Tags Versioning
// @Produce json
// @Param source query string true "Source version ID"
// @Param target query string true "Target version ID"
// @Success 200 {object} CompatibilityResult
// @Router /versions/compatibility [get]
func (h *Handler) CheckCompatibility(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	sourceID := c.Query("source")
	targetID := c.Query("target")

	if sourceID == "" || targetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source and target required"})
		return
	}

	result, err := h.service.CheckCompatibility(c.Request.Context(), tenantID, sourceID, targetID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// StartMigration godoc
// @Summary Start migration
// @Description Start a version migration
// @Tags Versioning
// @Accept json
// @Produce json
// @Param request body StartMigrationRequest true "Migration configuration"
// @Success 201 {object} Migration
// @Router /versions/migrations [post]
func (h *Handler) StartMigration(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req StartMigrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	migration, err := h.service.StartMigration(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, migration)
}

// GetMigration godoc
// @Summary Get migration
// @Description Get migration by ID
// @Tags Versioning
// @Produce json
// @Param id path string true "Migration ID"
// @Success 200 {object} Migration
// @Router /versions/migrations/{id} [get]
func (h *Handler) GetMigration(c *gin.Context) {
	migID := c.Param("id")

	migration, err := h.service.GetMigration(c.Request.Context(), migID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, migration)
}

// ListMigrations godoc
// @Summary List migrations
// @Description List migrations for a webhook
// @Tags Versioning
// @Produce json
// @Param webhook_id path string true "Webhook ID"
// @Success 200 {array} Migration
// @Router /versions/migrations/webhook/{webhook_id} [get]
func (h *Handler) ListMigrations(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	webhookID := c.Param("webhook_id")

	migrations, err := h.service.ListMigrations(c.Request.Context(), tenantID, webhookID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"migrations": migrations})
}

// CreateSchema godoc
// @Summary Create schema
// @Description Create a new version schema
// @Tags Versioning
// @Accept json
// @Produce json
// @Param request body CreateSchemaRequest true "Schema definition"
// @Success 201 {object} VersionSchema
// @Router /versions/schemas [post]
func (h *Handler) CreateSchema(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req CreateSchemaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	schema, err := h.service.CreateSchema(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, schema)
}

// ListSchemas godoc
// @Summary List schemas
// @Description List all schemas
// @Tags Versioning
// @Produce json
// @Success 200 {array} VersionSchema
// @Router /versions/schemas [get]
func (h *Handler) ListSchemas(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	schemas, err := h.service.ListSchemas(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"schemas": schemas})
}

// GetSchema godoc
// @Summary Get schema
// @Description Get schema by ID
// @Tags Versioning
// @Produce json
// @Param id path string true "Schema ID"
// @Success 200 {object} VersionSchema
// @Router /versions/schemas/{id} [get]
func (h *Handler) GetSchema(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	schemaID := c.Param("id")

	schema, err := h.service.GetSchema(c.Request.Context(), tenantID, schemaID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, schema)
}

// GetPolicy godoc
// @Summary Get policy
// @Description Get versioning policy
// @Tags Versioning
// @Produce json
// @Success 200 {object} VersionPolicy
// @Router /versions/policy [get]
func (h *Handler) GetPolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	policy, err := h.service.GetPolicy(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, policy)
}

// UpdatePolicy godoc
// @Summary Update policy
// @Description Update versioning policy
// @Tags Versioning
// @Accept json
// @Produce json
// @Param request body UpdatePolicyRequest true "Policy updates"
// @Success 200 {object} VersionPolicy
// @Router /versions/policy [put]
func (h *Handler) UpdatePolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req UpdatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	policy, err := h.service.UpdatePolicy(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, policy)
}

// NegotiateVersion godoc
// @Summary Negotiate version
// @Description Negotiate webhook version from headers
// @Tags Versioning
// @Accept json
// @Produce json
// @Param webhook_id query string true "Webhook ID"
// @Param accept header string false "Version accept header"
// @Success 200 {object} Version
// @Router /versions/negotiate [post]
func (h *Handler) NegotiateVersion(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	webhookID := c.Query("webhook_id")
	acceptHeader := c.GetHeader("X-API-Version")
	if acceptHeader == "" {
		acceptHeader = c.GetHeader("Accept-Version")
	}

	if webhookID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "webhook_id required"})
		return
	}

	version, err := h.service.NegotiateVersion(c.Request.Context(), tenantID, webhookID, acceptHeader)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, version)
}
