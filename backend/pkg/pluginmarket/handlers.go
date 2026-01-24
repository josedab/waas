package pluginmarket

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	pkgerrors "webhook-platform/pkg/errors"
)

// Handler implements HTTP handlers for the plugin marketplace
type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	plugins := r.Group("/marketplace/plugins")
	{
		plugins.GET("", h.SearchPlugins)
		plugins.GET("/:id", h.GetPlugin)
		plugins.POST("", h.CreatePlugin)
		plugins.PUT("/:id", h.UpdatePlugin)
		plugins.POST("/:id/publish", h.PublishPlugin)
		plugins.POST("/:id/submit-review", h.SubmitForReview)
		plugins.DELETE("/:id", h.DeletePlugin)

		// Versions
		plugins.GET("/:id/versions", h.GetVersions)
		plugins.POST("/:id/versions", h.CreateVersion)

		// Reviews
		plugins.GET("/:id/reviews", h.GetReviews)
		plugins.POST("/:id/reviews", h.CreateReview)

		// Hooks
		plugins.GET("/:id/hooks", h.GetPluginHooks)
		plugins.POST("/:id/hooks", h.RegisterHook)
	}

	installs := r.Group("/marketplace/installations")
	{
		installs.GET("", h.ListInstallations)
		installs.POST("", h.InstallPlugin)
		installs.DELETE("/:pluginId", h.UninstallPlugin)
	}

	r.GET("/marketplace/stats", h.GetMarketplaceStats)
	r.POST("/marketplace/execute-hook", h.ExecuteHook)
}

func (h *Handler) SearchPlugins(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	verified := c.Query("verified")
	featured := c.Query("featured")

	params := &PluginSearchParams{
		Query:    c.Query("q"),
		Type:     PluginType(c.Query("type")),
		Category: c.Query("category"),
		Pricing:  PricingModel(c.Query("pricing")),
		SortBy:   c.DefaultQuery("sort", "installs"),
		Page:     page,
		PageSize: pageSize,
	}
	if verified == "true" {
		v := true
		params.Verified = &v
	}
	if featured == "true" {
		f := true
		params.Featured = &f
	}

	result, err := h.service.SearchPlugins(c.Request.Context(), params)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetPlugin(c *gin.Context) {
	id := c.Param("id")
	plugin, err := h.service.GetPlugin(c.Request.Context(), id)
	if err != nil {
		pkgerrors.AbortWithNotFound(c, "plugin")
		return
	}
	c.JSON(http.StatusOK, plugin)
}

func (h *Handler) CreatePlugin(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreatePluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	plugin, err := h.service.CreatePlugin(c.Request.Context(), tenantID, tenantID, &req)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusCreated, plugin)
}

func (h *Handler) UpdatePlugin(c *gin.Context) {
	id := c.Param("id")
	plugin, err := h.service.GetPlugin(c.Request.Context(), id)
	if err != nil {
		pkgerrors.AbortWithNotFound(c, "plugin")
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	if name, ok := updates["name"].(string); ok {
		plugin.Name = name
	}
	if desc, ok := updates["description"].(string); ok {
		plugin.Description = desc
	}

	if err := h.service.repo.UpdatePlugin(c.Request.Context(), plugin); err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, plugin)
}

func (h *Handler) PublishPlugin(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	plugin, err := h.service.PublishPlugin(c.Request.Context(), id, tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, plugin)
}

func (h *Handler) SubmitForReview(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	plugin, err := h.service.SubmitForReview(c.Request.Context(), id, tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, plugin)
}

func (h *Handler) DeletePlugin(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.repo.DeletePlugin(c.Request.Context(), id); err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "plugin archived"})
}

func (h *Handler) GetVersions(c *gin.Context) {
	id := c.Param("id")
	versions, err := h.service.GetVersions(c.Request.Context(), id)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"versions": versions})
}

func (h *Handler) CreateVersion(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	var req struct {
		Version   string `json:"version" binding:"required"`
		Changelog string `json:"changelog"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	version, err := h.service.CreateVersion(c.Request.Context(), id, tenantID, req.Version, req.Changelog)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusCreated, version)
}

func (h *Handler) GetReviews(c *gin.Context) {
	id := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	reviews, total, err := h.service.GetReviews(c.Request.Context(), id, page, pageSize)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"reviews": reviews, "total": total})
}

func (h *Handler) CreateReview(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	pluginID := c.Param("id")

	var req CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	review, err := h.service.CreateReview(c.Request.Context(), tenantID, pluginID, &req)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusCreated, review)
}

func (h *Handler) GetPluginHooks(c *gin.Context) {
	id := c.Param("id")
	hooks, err := h.service.GetPluginHooks(c.Request.Context(), id)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"hooks": hooks})
}

func (h *Handler) RegisterHook(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		HookPoint HookPoint `json:"hook_point" binding:"required"`
		Priority  int       `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	hook, err := h.service.RegisterHook(c.Request.Context(), id, req.HookPoint, req.Priority)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusCreated, hook)
}

func (h *Handler) ListInstallations(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	installs, err := h.service.ListInstallations(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"installations": installs})
}

func (h *Handler) InstallPlugin(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req InstallPluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	install, err := h.service.InstallPlugin(c.Request.Context(), tenantID, &req)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusCreated, install)
}

func (h *Handler) UninstallPlugin(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	pluginID := c.Param("pluginId")

	if err := h.service.UninstallPlugin(c.Request.Context(), tenantID, pluginID); err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "plugin uninstalled"})
}

func (h *Handler) GetMarketplaceStats(c *gin.Context) {
	stats, err := h.service.GetMarketplaceStats(c.Request.Context())
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *Handler) ExecuteHook(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req struct {
		HookPoint HookPoint      `json:"hook_point" binding:"required"`
		Payload   map[string]any `json:"payload"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	results, err := h.service.ExecuteHook(c.Request.Context(), tenantID, req.HookPoint, req.Payload)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"results": results})
}
