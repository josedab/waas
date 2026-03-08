package dataplane

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP endpoints for data plane management.
type Handler struct {
	service *Service
}

// NewHandler creates a new data plane handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers data plane management routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/data-planes")
	{
		g.POST("", h.ProvisionPlane)
		g.GET("", h.ListPlanes)
		g.GET("/:tenant_id", h.GetPlane)
		g.POST("/:tenant_id/migrate", h.MigratePlane)
		g.GET("/:tenant_id/health", h.GetHealth)
		g.DELETE("/:tenant_id", h.DecommissionPlane)
	}
}

func (h *Handler) ProvisionPlane(c *gin.Context) {
	var req ProvisionPlaneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	plane, err := h.service.ProvisionPlane(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, plane)
}

func (h *Handler) GetPlane(c *gin.Context) {
	tenantID := c.Param("tenant_id")

	plane, err := h.service.GetPlane(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, plane)
}

func (h *Handler) ListPlanes(c *gin.Context) {
	planes, err := h.service.ListPlanes(c.Request.Context())
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data_planes": planes})
}

func (h *Handler) MigratePlane(c *gin.Context) {
	tenantID := c.Param("tenant_id")

	var req MigratePlaneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	plane, err := h.service.MigratePlane(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, plane)
}

func (h *Handler) GetHealth(c *gin.Context) {
	tenantID := c.Param("tenant_id")

	health, err := h.service.GetHealth(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, health)
}

func (h *Handler) DecommissionPlane(c *gin.Context) {
	tenantID := c.Param("tenant_id")

	if err := h.service.DecommissionPlane(c.Request.Context(), tenantID); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
