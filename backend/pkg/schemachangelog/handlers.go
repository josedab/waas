package schemachangelog

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP endpoints for schema changelog.
type Handler struct {
	service *Service
}

// NewHandler creates a new schema changelog handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers all schema changelog routes.
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	group := router.Group("/schema-changelog")
	{
		group.POST("/schemas", h.RegisterSchema)
		group.GET("/schemas/:event_type", h.GetSchemaVersions)
		group.GET("/changelogs/:event_type", h.GetChangelogs)
		group.GET("/changelog/:id", h.GetChangelog)
		group.POST("/compare", h.CompareVersions)
		group.POST("/migrations", h.CreateMigrationTracking)
		group.GET("/migrations/:changelog_id", h.GetMigrationStatus)
		group.POST("/migrations/:id/acknowledge", h.AcknowledgeMigration)
		group.POST("/migrations/:id/complete", h.CompleteMigration)
	}
}

// RegisterSchema registers a new schema version.
// @Summary Register schema version
// @Tags schema-changelog
// @Accept json
// @Produce json
// @Router /schema-changelog/schemas [post]
func (h *Handler) RegisterSchema(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	var req RegisterSchemaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sv, changelog, err := h.service.RegisterSchema(tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"schema_version": sv, "changelog": changelog})
}

// GetSchemaVersions lists all versions of a schema.
// @Summary Get schema versions
// @Tags schema-changelog
// @Param event_type path string true "Event Type"
// @Produce json
// @Router /schema-changelog/schemas/{event_type} [get]
func (h *Handler) GetSchemaVersions(c *gin.Context) {
	versions, err := h.service.GetSchemaVersions(c.Param("event_type"))
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, versions)
}

// GetChangelogs lists changelogs for an event type.
// @Summary Get changelogs
// @Tags schema-changelog
// @Param event_type path string true "Event Type"
// @Produce json
// @Router /schema-changelog/changelogs/{event_type} [get]
func (h *Handler) GetChangelogs(c *gin.Context) {
	changelogs, err := h.service.GetChangelogs(c.Param("event_type"))
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, changelogs)
}

// GetChangelog retrieves a specific changelog.
// @Summary Get changelog
// @Tags schema-changelog
// @Param id path string true "Changelog ID"
// @Produce json
// @Router /schema-changelog/changelog/{id} [get]
func (h *Handler) GetChangelog(c *gin.Context) {
	entry, err := h.service.GetChangelog(c.Param("id"))
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, entry)
}

// CompareVersions compares two schema versions.
// @Summary Compare schema versions
// @Tags schema-changelog
// @Accept json
// @Produce json
// @Router /schema-changelog/compare [post]
func (h *Handler) CompareVersions(c *gin.Context) {
	var req struct {
		EventType   string `json:"event_type" binding:"required"`
		FromVersion string `json:"from_version" binding:"required"`
		ToVersion   string `json:"to_version" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entry, err := h.service.CompareVersions(req.EventType, req.FromVersion, req.ToVersion)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, entry)
}

// CreateMigrationTracking creates migration tracking for consumers.
// @Summary Create migration tracking
// @Tags schema-changelog
// @Accept json
// @Produce json
// @Router /schema-changelog/migrations [post]
func (h *Handler) CreateMigrationTracking(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	var req struct {
		ChangelogID string   `json:"changelog_id" binding:"required"`
		EndpointIDs []string `json:"endpoint_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	migrations, err := h.service.CreateMigrationTracking(tenantID, req.ChangelogID, req.EndpointIDs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, migrations)
}

// GetMigrationStatus returns migration status for a changelog.
// @Summary Get migration status
// @Tags schema-changelog
// @Param changelog_id path string true "Changelog ID"
// @Produce json
// @Router /schema-changelog/migrations/{changelog_id} [get]
func (h *Handler) GetMigrationStatus(c *gin.Context) {
	migrations, err := h.service.GetMigrationStatus(c.Param("changelog_id"))
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, migrations)
}

// AcknowledgeMigration marks a migration as acknowledged.
// @Summary Acknowledge migration
// @Tags schema-changelog
// @Param id path string true "Migration ID"
// @Produce json
// @Router /schema-changelog/migrations/{id}/acknowledge [post]
func (h *Handler) AcknowledgeMigration(c *gin.Context) {
	m, err := h.service.AcknowledgeMigration(c.Param("id"))
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, m)
}

// CompleteMigration marks a migration as completed.
// @Summary Complete migration
// @Tags schema-changelog
// @Param id path string true "Migration ID"
// @Produce json
// @Router /schema-changelog/migrations/{id}/complete [post]
func (h *Handler) CompleteMigration(c *gin.Context) {
	m, err := h.service.CompleteMigration(c.Param("id"))
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, m)
}
