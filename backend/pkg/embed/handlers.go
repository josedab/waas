package embed

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for embeddable analytics
type Handler struct {
	service *Service
}

// NewHandler creates a new embed handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers embed routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	// Admin routes (require tenant auth)
	tokens := router.Group("/embed/tokens")
	{
		tokens.POST("", h.CreateToken)
		tokens.GET("", h.ListTokens)
		tokens.GET("/:id", h.GetToken)
		tokens.PUT("/:id", h.UpdateToken)
		tokens.DELETE("/:id", h.DeleteToken)
		tokens.POST("/:id/rotate", h.RotateToken)
	}

	// Public embed routes (authenticated by embed token)
	embed := router.Group("/embed/v1")
	embed.Use(h.embedTokenAuth())
	{
		embed.GET("/config", h.GetConfig)
		embed.GET("/stats", h.GetStats)
		embed.GET("/activity", h.GetActivity)
		embed.GET("/chart/:type", h.GetChart)
		embed.GET("/errors", h.GetErrors)
		embed.GET("/components", h.GetComponents)
	}
}

// embedTokenAuth middleware validates embed tokens
func (h *Handler) embedTokenAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenValue := c.GetHeader("X-Embed-Token")
		if tokenValue == "" {
			tokenValue = c.Query("token")
		}

		if tokenValue == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing embed token"})
			return
		}

		origin := c.GetHeader("Origin")
		token, err := h.service.ValidateToken(c.Request.Context(), tokenValue, origin)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		// Set token info in context
		c.Set("embed_token", token)
		c.Set("tenant_id", token.TenantID)

		// Record session
		h.service.RecordSession(
			c.Request.Context(),
			token.ID, token.TenantID,
			origin, c.GetHeader("User-Agent"), c.ClientIP(),
		)

		c.Next()
	}
}

// CreateToken godoc
//
//	@Summary		Create embed token
//	@Description	Create a new embeddable analytics token
//	@Tags			embed
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateTokenRequest	true	"Token creation request"
//	@Success		201		{object}	EmbedToken
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/embed/tokens [post]
func (h *Handler) CreateToken(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	token, err := h.service.CreateToken(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, token)
}

// ListTokens godoc
//
//	@Summary		List embed tokens
//	@Description	Get a list of embed tokens for the tenant
//	@Tags			embed
//	@Produce		json
//	@Param			limit	query		int	false	"Limit"		default(20)
//	@Param			offset	query		int	false	"Offset"	default(0)
//	@Success		200		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/embed/tokens [get]
func (h *Handler) ListTokens(c *gin.Context) {
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

	tokens, total, err := h.service.ListTokens(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tokens": tokens,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetToken godoc
//
//	@Summary		Get embed token
//	@Description	Get details of a specific embed token
//	@Tags			embed
//	@Produce		json
//	@Param			id	path		string	true	"Token ID"
//	@Success		200	{object}	EmbedToken
//	@Failure		401	{object}	map[string]interface{}
//	@Failure		404	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/embed/tokens/{id} [get]
func (h *Handler) GetToken(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	tokenID := c.Param("id")
	token, err := h.service.GetToken(c.Request.Context(), tenantID, tokenID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if token == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "token not found"})
		return
	}

	c.JSON(http.StatusOK, token)
}

// UpdateToken godoc
//
//	@Summary		Update embed token
//	@Description	Update an embed token
//	@Tags			embed
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Token ID"
//	@Param			request	body		UpdateTokenRequest	true	"Update request"
//	@Success		200		{object}	EmbedToken
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/embed/tokens/{id} [put]
func (h *Handler) UpdateToken(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	tokenID := c.Param("id")
	var req UpdateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	token, err := h.service.UpdateToken(c.Request.Context(), tenantID, tokenID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, token)
}

// DeleteToken godoc
//
//	@Summary		Delete embed token
//	@Description	Delete an embed token
//	@Tags			embed
//	@Produce		json
//	@Param			id	path	string	true	"Token ID"
//	@Success		204	"No content"
//	@Failure		401	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/embed/tokens/{id} [delete]
func (h *Handler) DeleteToken(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	tokenID := c.Param("id")
	if err := h.service.DeleteToken(c.Request.Context(), tenantID, tokenID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// RotateToken godoc
//
//	@Summary		Rotate embed token
//	@Description	Generate a new token value
//	@Tags			embed
//	@Produce		json
//	@Param			id	path		string	true	"Token ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		401	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/embed/tokens/{id}/rotate [post]
func (h *Handler) RotateToken(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	tokenID := c.Param("id")
	token, newValue, err := h.service.RotateToken(c.Request.Context(), tenantID, tokenID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":     token,
		"new_value": newValue,
		"message":   "Token rotated successfully. Update your embed code with the new token.",
	})
}

// GetConfig godoc
//
//	@Summary		Get embed configuration
//	@Description	Get configuration for the embed token
//	@Tags			embed-public
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Security		EmbedToken
//	@Router			/embed/v1/config [get]
func (h *Handler) GetConfig(c *gin.Context) {
	token := c.MustGet("embed_token").(*EmbedToken)

	c.JSON(http.StatusOK, gin.H{
		"permissions":     token.Permissions,
		"scopes":          token.Scopes,
		"theme":           token.Theme,
		"allowed_origins": token.AllowedOrigins,
	})
}

// GetStats godoc
//
//	@Summary		Get delivery statistics
//	@Description	Get delivery statistics for embedded view
//	@Tags			embed-public
//	@Produce		json
//	@Success		200	{object}	DeliveryStats
//	@Security		EmbedToken
//	@Router			/embed/v1/stats [get]
func (h *Handler) GetStats(c *gin.Context) {
	token := c.MustGet("embed_token").(*EmbedToken)

	if !h.service.HasPermission(token, PermissionReadMetrics) {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	stats, err := h.service.GetDeliveryStats(c.Request.Context(), token.TenantID, token.Scopes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetActivity godoc
//
//	@Summary		Get activity feed
//	@Description	Get activity feed for embedded view
//	@Tags			embed-public
//	@Produce		json
//	@Param			limit	query		int	false	"Limit"	default(20)
//	@Success		200		{object}	map[string]interface{}
//	@Security		EmbedToken
//	@Router			/embed/v1/activity [get]
func (h *Handler) GetActivity(c *gin.Context) {
	token := c.MustGet("embed_token").(*EmbedToken)

	if !h.service.HasPermission(token, PermissionReadActivity) {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	items, err := h.service.GetActivityFeed(c.Request.Context(), token.TenantID, token.Scopes, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

// GetChart godoc
//
//	@Summary		Get chart data
//	@Description	Get chart data for embedded view
//	@Tags			embed-public
//	@Produce		json
//	@Param			type	path		string	true	"Chart type"
//	@Success		200		{object}	ChartData
//	@Security		EmbedToken
//	@Router			/embed/v1/chart/{type} [get]
func (h *Handler) GetChart(c *gin.Context) {
	token := c.MustGet("embed_token").(*EmbedToken)

	if !h.service.HasPermission(token, PermissionReadMetrics) {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	chartType := c.Param("type")
	data, err := h.service.GetChartData(c.Request.Context(), token.TenantID, chartType, token.Scopes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

// GetErrors godoc
//
//	@Summary		Get error summary
//	@Description	Get error summary for embedded view
//	@Tags			embed-public
//	@Produce		json
//	@Success		200	{object}	ErrorSummary
//	@Security		EmbedToken
//	@Router			/embed/v1/errors [get]
func (h *Handler) GetErrors(c *gin.Context) {
	token := c.MustGet("embed_token").(*EmbedToken)

	if !h.service.HasPermission(token, PermissionReadErrors) {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	summary, err := h.service.GetErrorSummary(c.Request.Context(), token.TenantID, token.Scopes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// GetComponents godoc
//
//	@Summary		Get available components
//	@Description	Get available embed components
//	@Tags			embed-public
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Security		EmbedToken
//	@Router			/embed/v1/components [get]
func (h *Handler) GetComponents(c *gin.Context) {
	components := h.service.GetAvailableComponents()
	c.JSON(http.StatusOK, gin.H{"components": components})
}
