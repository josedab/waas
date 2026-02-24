package portal

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for the embeddable portal
type Handler struct {
	service *Service
}

// NewHandler creates a new portal handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers portal routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	p := router.Group("/portal")
	{
		// Portal configuration (API key authenticated)
		p.POST("/configs", h.CreatePortal)
		p.GET("/configs", h.ListPortals)
		p.GET("/configs/:id", h.GetPortal)
		p.PUT("/configs/:id", h.UpdatePortal)
		p.DELETE("/configs/:id", h.DeletePortal)

		// Tenant-level portal config
		p.GET("/config", h.GetPortalConfig)
		p.PUT("/config", h.UpdatePortalConfig)

		// Embed tokens
		p.POST("/tokens", h.CreateToken)
		p.GET("/configs/:id/tokens", h.ListTokens)
		p.DELETE("/tokens/:token_id", h.RevokeToken)

		// Embed snippets
		p.GET("/configs/:id/snippet", h.GetEmbedSnippet)
		p.GET("/embed-code", h.GetEmbedCode)

		// Sessions
		p.GET("/sessions", h.ListSessions)

		// Portal-token-authenticated endpoints
		portalAuth := p.Group("")
		portalAuth.Use(h.PortalTokenMiddleware())
		{
			portalAuth.GET("/endpoints", RequireScope(ScopeEndpointsRead), h.GetPortalEndpoints)
			portalAuth.GET("/deliveries", RequireScope(ScopeDeliveriesRead), h.GetPortalDeliveries)
			portalAuth.POST("/deliveries/:id/retry", RequireScope(ScopeDeliveriesRetry), h.RetryDelivery)
			portalAuth.GET("/stats", RequireScope(ScopeAnalyticsRead), h.GetPortalStats)
		}
	}
}

// @Summary Create a portal configuration
// @Tags Portal
// @Accept json
// @Produce json
// @Param body body CreatePortalRequest true "Portal config"
// @Success 201 {object} PortalConfig
// @Router /portal/configs [post]
func (h *Handler) CreatePortal(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreatePortalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	config, err := h.service.CreatePortal(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalError(c, "CREATE_FAILED", err)
		return
	}

	c.JSON(http.StatusCreated, config)
}

// @Summary List portal configurations
// @Tags Portal
// @Produce json
// @Success 200 {object} map[string][]PortalConfig
// @Router /portal/configs [get]
func (h *Handler) ListPortals(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	configs, err := h.service.ListPortals(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "LIST_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"portals": configs})
}

// @Summary Get a portal configuration
// @Tags Portal
// @Produce json
// @Param id path string true "Portal ID"
// @Success 200 {object} PortalConfig
// @Router /portal/configs/{id} [get]
func (h *Handler) GetPortal(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	portalID := c.Param("id")

	config, err := h.service.GetPortal(c.Request.Context(), tenantID, portalID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, config)
}

// @Summary Update a portal configuration
// @Tags Portal
// @Accept json
// @Produce json
// @Param id path string true "Portal ID"
// @Param body body CreatePortalRequest true "Updated config"
// @Success 200 {object} PortalConfig
// @Router /portal/configs/{id} [put]
func (h *Handler) UpdatePortal(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	portalID := c.Param("id")

	var req CreatePortalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	config, err := h.service.UpdatePortal(c.Request.Context(), tenantID, portalID, &req)
	if err != nil {
		httputil.InternalError(c, "UPDATE_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, config)
}

// @Summary Delete a portal configuration
// @Tags Portal
// @Param id path string true "Portal ID"
// @Success 204
// @Router /portal/configs/{id} [delete]
func (h *Handler) DeletePortal(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	portalID := c.Param("id")

	if err := h.service.DeletePortal(c.Request.Context(), tenantID, portalID); err != nil {
		httputil.InternalError(c, "DELETE_FAILED", err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary Create an embed token
// @Tags Portal
// @Accept json
// @Produce json
// @Param body body CreateTokenRequest true "Token request"
// @Success 201 {object} EmbedToken
// @Router /portal/tokens [post]
func (h *Handler) CreateToken(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	token, err := h.service.CreateEmbedToken(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalError(c, "TOKEN_FAILED", err)
		return
	}

	c.JSON(http.StatusCreated, token)
}

// @Summary List embed tokens for a portal
// @Tags Portal
// @Produce json
// @Param id path string true "Portal ID"
// @Success 200 {object} map[string][]EmbedToken
// @Router /portal/configs/{id}/tokens [get]
func (h *Handler) ListTokens(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	portalID := c.Param("id")

	tokens, err := h.service.ListTokens(c.Request.Context(), tenantID, portalID)
	if err != nil {
		httputil.InternalError(c, "LIST_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"tokens": tokens})
}

// @Summary Revoke an embed token
// @Tags Portal
// @Param token_id path string true "Token ID"
// @Success 204
// @Router /portal/tokens/{token_id} [delete]
func (h *Handler) RevokeToken(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	tokenID := c.Param("token_id")

	if err := h.service.RevokeToken(c.Request.Context(), tenantID, tokenID); err != nil {
		httputil.InternalError(c, "REVOKE_FAILED", err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary Get embed code snippets
// @Tags Portal
// @Produce json
// @Param id path string true "Portal ID"
// @Success 200 {object} EmbedSnippet
// @Router /portal/configs/{id}/snippet [get]
func (h *Handler) GetEmbedSnippet(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	portalID := c.Param("id")

	apiURL := c.Request.Host
	if c.Request.TLS != nil {
		apiURL = "https://" + apiURL
	} else {
		apiURL = "http://" + apiURL
	}

	snippet, err := h.service.GetEmbedSnippet(c.Request.Context(), tenantID, portalID, apiURL)
	if err != nil {
		httputil.InternalError(c, "SNIPPET_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, snippet)
}

// @Summary List active portal sessions
// @Tags Portal
// @Produce json
// @Success 200 {object} map[string][]PortalSession
// @Router /portal/sessions [get]
func (h *Handler) ListSessions(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	sessions, err := h.service.ListSessions(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "LIST_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"sessions": sessions})
}

// @Summary Get portal configuration for tenant
// @Tags Portal
// @Produce json
// @Success 200 {object} PortalConfig
// @Router /portal/config [get]
func (h *Handler) GetPortalConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	config, err := h.service.GetPortalConfig(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, config)
}

// @Summary Update portal configuration for tenant
// @Tags Portal
// @Accept json
// @Produce json
// @Success 200 {object} PortalConfig
// @Router /portal/config [put]
func (h *Handler) UpdatePortalConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req UpdatePortalConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	config, err := h.service.UpdatePortalConfig(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalError(c, "UPDATE_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, config)
}

// @Summary Generate embed code snippet
// @Tags Portal
// @Produce json
// @Param portal_id query string true "Portal ID"
// @Param format query string false "Format: html, react, iframe"
// @Success 200 {object} map[string]string
// @Router /portal/embed-code [get]
func (h *Handler) GetEmbedCode(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	portalID := c.Query("portal_id")
	format := c.DefaultQuery("format", "html")

	if portalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": "portal_id query parameter is required"}})
		return
	}

	apiURL := c.Request.Host
	if c.Request.TLS != nil {
		apiURL = "https://" + apiURL
	} else {
		apiURL = "http://" + apiURL
	}

	code, err := h.service.GenerateEmbedSnippetForFormat(c.Request.Context(), tenantID, portalID, format, apiURL)
	if err != nil {
		httputil.InternalError(c, "SNIPPET_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"format": format, "code": code})
}

// @Summary List endpoints (portal-token authenticated)
// @Tags Portal
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} map[string]interface{}
// @Router /portal/endpoints [get]
func (h *Handler) GetPortalEndpoints(c *gin.Context) {
	tenantID := c.GetString(PortalTenantIDKey)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	endpoints, total, err := h.service.GetPortalEndpoints(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		httputil.InternalError(c, "LIST_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"endpoints": endpoints, "total": total})
}

// @Summary List deliveries (portal-token authenticated)
// @Tags Portal
// @Produce json
// @Param endpoint_id query string false "Endpoint ID filter"
// @Param status query string false "Status filter"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} map[string]interface{}
// @Router /portal/deliveries [get]
func (h *Handler) GetPortalDeliveries(c *gin.Context) {
	tenantID := c.GetString(PortalTenantIDKey)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	filter := DeliveryFilter{
		EndpointID: c.Query("endpoint_id"),
		Status:     c.Query("status"),
		EventType:  c.Query("event_type"),
	}

	deliveries, total, err := h.service.GetPortalDeliveries(c.Request.Context(), tenantID, filter, limit, offset)
	if err != nil {
		httputil.InternalError(c, "LIST_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"deliveries": deliveries, "total": total})
}

// @Summary Retry a failed delivery (portal-token authenticated)
// @Tags Portal
// @Param id path string true "Delivery ID"
// @Success 200 {object} map[string]string
// @Router /portal/deliveries/{id}/retry [post]
func (h *Handler) RetryDelivery(c *gin.Context) {
	tenantID := c.GetString(PortalTenantIDKey)
	deliveryID := c.Param("id")

	if err := h.service.RetryPortalDelivery(c.Request.Context(), tenantID, deliveryID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "RETRY_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "delivery retry initiated", "delivery_id": deliveryID})
}

// @Summary Get portal usage statistics (portal-token authenticated)
// @Tags Portal
// @Produce json
// @Success 200 {object} PortalStats
// @Router /portal/stats [get]
func (h *Handler) GetPortalStats(c *gin.Context) {
	tenantID := c.GetString(PortalTenantIDKey)

	stats, err := h.service.GetPortalStats(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "STATS_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, stats)
}
