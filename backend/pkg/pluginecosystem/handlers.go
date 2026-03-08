package pluginecosystem

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/josedab/waas/pkg/httputil"
)

// Handler provides HTTP endpoints for the plugin ecosystem.
type Handler struct {
	service *Service
}

// NewHandler creates a new plugin ecosystem handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers plugin ecosystem routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/plugin-ecosystem")
	{
		g.POST("/plugins", h.PublishPlugin)
		g.GET("/plugins", h.SearchPlugins)
		g.GET("/plugins/:id", h.GetPlugin)
		g.POST("/plugins/:id/approve", h.ApprovePlugin)
		g.POST("/plugins/:id/install", h.InstallPlugin)
		g.DELETE("/plugins/:id/uninstall", h.UninstallPlugin)
		g.GET("/plugins/:id/reviews", h.ListReviews)
		g.POST("/plugins/:id/reviews", h.AddReview)
		g.GET("/installations", h.ListInstallations)
	}
}

func (h *Handler) PublishPlugin(c *gin.Context) {
	developerID := c.GetString("tenant_id")
	var req PublishPluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	plugin, err := h.service.PublishPlugin(c.Request.Context(), developerID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, plugin)
}

func (h *Handler) SearchPlugins(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	req := &SearchPluginsRequest{
		Query:   c.Query("q"),
		Type:    PluginType(c.Query("type")),
		Pricing: PricingModel(c.Query("pricing")),
		SortBy:  c.DefaultQuery("sort", "installs"),
		Limit:   limit,
		Offset:  offset,
	}
	plugins, err := h.service.SearchPlugins(c.Request.Context(), req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"plugins": plugins})
}

func (h *Handler) GetPlugin(c *gin.Context) {
	plugin, err := h.service.GetPlugin(c.Request.Context(), c.Param("id"))
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, plugin)
}

func (h *Handler) ApprovePlugin(c *gin.Context) {
	if err := h.service.ApprovePlugin(c.Request.Context(), c.Param("id")); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "approved"})
}

func (h *Handler) InstallPlugin(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	req := &InstallPluginRequest{PluginID: c.Param("id")}
	if c.Request.ContentLength > 0 {
		c.ShouldBindJSON(req)
	}
	inst, err := h.service.InstallPlugin(c.Request.Context(), tenantID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, inst)
}

func (h *Handler) UninstallPlugin(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if err := h.service.UninstallPlugin(c.Request.Context(), tenantID, c.Param("id")); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListInstallations(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	installations, err := h.service.ListInstallations(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"installations": installations})
}

func (h *Handler) AddReview(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req struct {
		Rating  int    `json:"rating" binding:"required"`
		Comment string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	review, err := h.service.AddReview(c.Request.Context(), tenantID, c.Param("id"), req.Rating, req.Comment)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, review)
}

func (h *Handler) ListReviews(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	reviews, err := h.service.ListReviews(c.Request.Context(), c.Param("id"), limit)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"reviews": reviews})
}
