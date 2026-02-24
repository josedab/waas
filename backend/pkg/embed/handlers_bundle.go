package embed

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"

	"github.com/gin-gonic/gin"
)

// WidgetBundleHandler provides HTTP handlers for widget bundles
type WidgetBundleHandler struct {
	service *WidgetBundleService
}

// NewWidgetBundleHandler creates a new handler
func NewWidgetBundleHandler(service *WidgetBundleService) *WidgetBundleHandler {
	return &WidgetBundleHandler{service: service}
}

// RegisterBundleRoutes registers widget bundle routes
func (h *WidgetBundleHandler) RegisterBundleRoutes(r *gin.RouterGroup) {
	widgets := r.Group("/widgets")
	{
		bundles := widgets.Group("/bundles")
		{
			bundles.POST("", h.CreateBundle)
			bundles.GET("", h.ListBundles)
			bundles.GET("/:bundleId", h.GetBundle)
			bundles.DELETE("/:bundleId", h.DeleteBundle)
		}

		themes := widgets.Group("/themes")
		{
			themes.POST("", h.CreateTheme)
			themes.GET("", h.ListThemes)
			themes.DELETE("/:themeId", h.DeleteTheme)
		}
	}
}

// CreateBundle creates a widget bundle
// @Summary Create widget bundle
// @Tags widgets
// @Accept json
// @Produce json
// @Success 201 {object} WidgetBundle
// @Router /widgets/bundles [post]
func (h *WidgetBundleHandler) CreateBundle(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateWidgetBundleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bundle, err := h.service.CreateBundle(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, bundle)
}

// ListBundles lists widget bundles
// @Summary List widget bundles
// @Tags widgets
// @Produce json
// @Success 200 {array} WidgetBundle
// @Router /widgets/bundles [get]
func (h *WidgetBundleHandler) ListBundles(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	bundles, err := h.service.ListBundles(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, bundles)
}

// GetBundle retrieves a widget bundle
// @Summary Get widget bundle
// @Tags widgets
// @Produce json
// @Param bundleId path string true "Bundle ID"
// @Success 200 {object} WidgetBundle
// @Router /widgets/bundles/{bundleId} [get]
func (h *WidgetBundleHandler) GetBundle(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	bundleID := c.Param("bundleId")
	bundle, err := h.service.GetBundle(c.Request.Context(), tenantID, bundleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "bundle not found"})
		return
	}

	c.JSON(http.StatusOK, bundle)
}

// DeleteBundle deletes a widget bundle
// @Summary Delete widget bundle
// @Tags widgets
// @Param bundleId path string true "Bundle ID"
// @Success 204 "No content"
// @Router /widgets/bundles/{bundleId} [delete]
func (h *WidgetBundleHandler) DeleteBundle(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	bundleID := c.Param("bundleId")
	if err := h.service.DeleteBundle(c.Request.Context(), tenantID, bundleID); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// CreateTheme creates a white-label theme
// @Summary Create white-label theme
// @Tags widgets
// @Accept json
// @Produce json
// @Success 201 {object} WhiteLabelTheme
// @Router /widgets/themes [post]
func (h *WidgetBundleHandler) CreateTheme(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateThemeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	theme, err := h.service.CreateTheme(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, theme)
}

// ListThemes lists white-label themes
// @Summary List themes
// @Tags widgets
// @Produce json
// @Success 200 {array} WhiteLabelTheme
// @Router /widgets/themes [get]
func (h *WidgetBundleHandler) ListThemes(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	themes, err := h.service.ListThemes(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, themes)
}

// DeleteTheme deletes a theme
// @Summary Delete theme
// @Tags widgets
// @Param themeId path string true "Theme ID"
// @Success 204 "No content"
// @Router /widgets/themes/{themeId} [delete]
func (h *WidgetBundleHandler) DeleteTheme(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	themeID := c.Param("themeId")
	if err := h.service.DeleteTheme(c.Request.Context(), tenantID, themeID); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
