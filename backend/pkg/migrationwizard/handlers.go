package migrationwizard

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP endpoints for the migration wizard.
type Handler struct {
	service *Service
}

// NewHandler creates a new migration wizard handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers migration wizard routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/migrations")
	{
		g.POST("", h.StartMigration)
		g.GET("", h.ListMigrations)
		g.GET("/:id", h.GetMigration)
		g.POST("/:id/execute", h.ExecuteMigration)
		g.POST("/:id/rollback", h.RollbackMigration)
		g.DELETE("/:id", h.DeleteMigration)
		g.GET("/compatibility/:platform", h.GetCompatibility)
	}
}

func (h *Handler) StartMigration(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req StartMigrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	m, err := h.service.StartMigration(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, m)
}

func (h *Handler) ListMigrations(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	migrations, err := h.service.ListMigrations(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"migrations": migrations})
}

func (h *Handler) GetMigration(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	m, err := h.service.GetMigration(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, m)
}

func (h *Handler) ExecuteMigration(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	m, err := h.service.ExecuteMigration(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, m)
}

func (h *Handler) RollbackMigration(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	m, err := h.service.RollbackMigration(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, m)
}

func (h *Handler) DeleteMigration(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if err := h.service.DeleteMigration(c.Request.Context(), tenantID, c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) GetCompatibility(c *gin.Context) {
	platform := SourcePlatform(c.Param("platform"))
	info := h.service.GetCompatibility(platform)
	c.JSON(http.StatusOK, info)
}
