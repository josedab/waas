package analyticsembed

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for embeddable analytics
type Handler struct {
	service *Service
}

// NewHandler creates a new embeddable analytics handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers embeddable analytics routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	embed := router.Group("/analytics/embed")
	{
		// Widgets
		embed.POST("/widgets", h.CreateWidget)
		embed.GET("/widgets", h.ListWidgets)
		embed.GET("/widgets/:id", h.GetWidget)
		embed.PUT("/widgets/:id", h.UpdateWidget)
		embed.DELETE("/widgets/:id", h.DeleteWidget)
		embed.GET("/widgets/:id/data", h.GetWidgetData)
		embed.GET("/widgets/:id/snippet", h.GetEmbedSnippet)

		// Tokens
		embed.POST("/tokens", h.GenerateEmbedToken)
		embed.POST("/tokens/validate", h.ValidateEmbedToken)

		// Theme
		embed.GET("/theme", h.GetTheme)
		embed.PUT("/theme", h.UpdateTheme)
	}
}

// @Summary Create an analytics widget
// @Tags AnalyticsEmbed
// @Accept json
// @Produce json
// @Param body body CreateWidgetRequest true "Widget configuration"
// @Success 201 {object} WidgetConfig
// @Router /analytics/embed/widgets [post]
func (h *Handler) CreateWidget(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateWidgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	widget, err := h.service.CreateWidget(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalError(c, "CREATE_FAILED", err)
		return
	}

	c.JSON(http.StatusCreated, widget)
}

// @Summary List analytics widgets
// @Tags AnalyticsEmbed
// @Produce json
// @Success 200 {object} map[string][]WidgetConfig
// @Router /analytics/embed/widgets [get]
func (h *Handler) ListWidgets(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	widgets, err := h.service.ListWidgets(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "LIST_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"widgets": widgets})
}

// @Summary Get an analytics widget
// @Tags AnalyticsEmbed
// @Produce json
// @Param id path string true "Widget ID"
// @Success 200 {object} WidgetConfig
// @Router /analytics/embed/widgets/{id} [get]
func (h *Handler) GetWidget(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	widgetID := c.Param("id")

	widget, err := h.service.GetWidget(c.Request.Context(), tenantID, widgetID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, widget)
}

// @Summary Update an analytics widget
// @Tags AnalyticsEmbed
// @Accept json
// @Produce json
// @Param id path string true "Widget ID"
// @Param body body CreateWidgetRequest true "Updated widget configuration"
// @Success 200 {object} WidgetConfig
// @Router /analytics/embed/widgets/{id} [put]
func (h *Handler) UpdateWidget(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	widgetID := c.Param("id")

	var req CreateWidgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	widget, err := h.service.UpdateWidget(c.Request.Context(), tenantID, widgetID, &req)
	if err != nil {
		httputil.InternalError(c, "UPDATE_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, widget)
}

// @Summary Delete an analytics widget
// @Tags AnalyticsEmbed
// @Param id path string true "Widget ID"
// @Success 204
// @Router /analytics/embed/widgets/{id} [delete]
func (h *Handler) DeleteWidget(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	widgetID := c.Param("id")

	if err := h.service.DeleteWidget(c.Request.Context(), tenantID, widgetID); err != nil {
		httputil.InternalError(c, "DELETE_FAILED", err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary Get widget data
// @Tags AnalyticsEmbed
// @Produce json
// @Param id path string true "Widget ID"
// @Success 200 {object} WidgetData
// @Router /analytics/embed/widgets/{id}/data [get]
func (h *Handler) GetWidgetData(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	widgetID := c.Param("id")

	data, err := h.service.GetWidgetData(c.Request.Context(), tenantID, widgetID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, data)
}

// @Summary Get embed snippet for a widget
// @Tags AnalyticsEmbed
// @Produce json
// @Param id path string true "Widget ID"
// @Success 200 {object} EmbedSnippet
// @Router /analytics/embed/widgets/{id}/snippet [get]
func (h *Handler) GetEmbedSnippet(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	widgetID := c.Param("id")

	snippet, err := h.service.GetEmbedSnippet(c.Request.Context(), tenantID, widgetID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, snippet)
}

// @Summary Generate an embed token
// @Tags AnalyticsEmbed
// @Accept json
// @Produce json
// @Param body body CreateEmbedTokenRequest true "Token configuration"
// @Success 201 {object} EmbedToken
// @Router /analytics/embed/tokens [post]
func (h *Handler) GenerateEmbedToken(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateEmbedTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	token, err := h.service.GenerateEmbedToken(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalError(c, "TOKEN_GENERATION_FAILED", err)
		return
	}

	c.JSON(http.StatusCreated, token)
}

// @Summary Validate an embed token
// @Tags AnalyticsEmbed
// @Accept json
// @Produce json
// @Success 200 {object} EmbedToken
// @Router /analytics/embed/tokens/validate [post]
func (h *Handler) ValidateEmbedToken(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
		Scope string `json:"scope"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	token, err := h.service.ValidateEmbedToken(c.Request.Context(), req.Token, req.Scope)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "INVALID_TOKEN", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true, "token": token})
}

// @Summary Get theme configuration
// @Tags AnalyticsEmbed
// @Produce json
// @Success 200 {object} ThemeConfig
// @Router /analytics/embed/theme [get]
func (h *Handler) GetTheme(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	theme, err := h.service.GetTheme(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "THEME_FETCH_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, theme)
}

// @Summary Update theme configuration
// @Tags AnalyticsEmbed
// @Accept json
// @Produce json
// @Param body body UpdateThemeRequest true "Theme configuration"
// @Success 200 {object} ThemeConfig
// @Router /analytics/embed/theme [put]
func (h *Handler) UpdateTheme(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req UpdateThemeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	theme, err := h.service.UpdateTheme(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalError(c, "UPDATE_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, theme)
}
