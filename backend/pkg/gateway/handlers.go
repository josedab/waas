package gateway

import (
	"fmt"
	"github.com/josedab/waas/pkg/httputil"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for the inbound webhook gateway
type Handler struct {
	service *Service
	baseURL string
}

// NewHandler creates a new gateway handler
func NewHandler(service *Service, baseURL string) *Handler {
	return &Handler{service: service, baseURL: baseURL}
}

// RegisterRoutes registers gateway routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	providers := router.Group("/gateway/providers")
	{
		providers.POST("", h.CreateProvider)
		providers.GET("", h.ListProviders)
		providers.GET("/:id", h.GetProvider)
		providers.DELETE("/:id", h.DeleteProvider)
		providers.GET("/:id/url", h.GetIngestURL)
	}

	rules := router.Group("/gateway/rules")
	{
		rules.POST("", h.CreateRoutingRule)
		rules.GET("/:id", h.GetRoutingRule)
		rules.PUT("/:id", h.UpdateRoutingRule)
		rules.DELETE("/:id", h.DeleteRoutingRule)
	}

	router.GET("/gateway/providers/:id/rules", h.ListRoutingRules)
	router.GET("/gateway/webhooks", h.ListInboundWebhooks)
	router.GET("/gateway/webhooks/:id", h.GetInboundWebhook)

	// Ingest endpoint — receives webhooks from external providers
	router.POST("/gateway/ingest/:provider_id", h.IngestWebhook)
}

// CreateProvider godoc
// @Summary Create a webhook provider
// @Tags gateway
// @Accept json
// @Produce json
// @Param request body CreateProviderRequest true "Provider creation request"
// @Success 201 {object} Provider
// @Router /gateway/providers [post]
func (h *Handler) CreateProvider(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	provider, err := h.service.CreateProvider(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, provider)
}

// ListProviders godoc
// @Summary List webhook providers
// @Tags gateway
// @Produce json
// @Router /gateway/providers [get]
func (h *Handler) ListProviders(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit, offset := 20, 0
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := c.Query("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	providers, total, err := h.service.ListProviders(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"providers": providers,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}

// GetProvider godoc
// @Summary Get a webhook provider
// @Tags gateway
// @Produce json
// @Param id path string true "Provider ID"
// @Router /gateway/providers/{id} [get]
func (h *Handler) GetProvider(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	provider, err := h.service.GetProvider(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	if provider == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
		return
	}

	c.JSON(http.StatusOK, provider)
}

// DeleteProvider godoc
// @Summary Delete a webhook provider
// @Tags gateway
// @Param id path string true "Provider ID"
// @Router /gateway/providers/{id} [delete]
func (h *Handler) DeleteProvider(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.service.DeleteProvider(c.Request.Context(), tenantID, c.Param("id")); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// GetIngestURL godoc
// @Summary Get the ingest URL for a provider
// @Tags gateway
// @Produce json
// @Param id path string true "Provider ID"
// @Router /gateway/providers/{id}/url [get]
func (h *Handler) GetIngestURL(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	providerID := c.Param("id")
	url := h.service.GetProviderEndpointURL(h.baseURL, tenantID, providerID)

	c.JSON(http.StatusOK, gin.H{
		"ingest_url":  url,
		"provider_id": providerID,
		"tenant_id":   tenantID,
	})
}

// CreateRoutingRule godoc
// @Summary Create a routing rule
// @Tags gateway
// @Accept json
// @Produce json
// @Param request body CreateRoutingRuleRequest true "Routing rule creation request"
// @Router /gateway/rules [post]
func (h *Handler) CreateRoutingRule(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateRoutingRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	rule, err := h.service.CreateRoutingRule(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, rule)
}

// GetRoutingRule godoc
// @Summary Get a routing rule
// @Tags gateway
// @Produce json
// @Param id path string true "Rule ID"
// @Router /gateway/rules/{id} [get]
func (h *Handler) GetRoutingRule(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	rule, err := h.service.GetRoutingRule(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	if rule == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "routing rule not found"})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// UpdateRoutingRule godoc
// @Summary Update a routing rule
// @Tags gateway
// @Accept json
// @Produce json
// @Param id path string true "Rule ID"
// @Param request body UpdateRoutingRuleRequest true "Rule update request"
// @Router /gateway/rules/{id} [put]
func (h *Handler) UpdateRoutingRule(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req UpdateRoutingRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	rule, err := h.service.UpdateRoutingRule(c.Request.Context(), tenantID, c.Param("id"), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// DeleteRoutingRule godoc
// @Summary Delete a routing rule
// @Tags gateway
// @Param id path string true "Rule ID"
// @Router /gateway/rules/{id} [delete]
func (h *Handler) DeleteRoutingRule(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.service.DeleteRoutingRule(c.Request.Context(), tenantID, c.Param("id")); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ListRoutingRules godoc
// @Summary List routing rules for a provider
// @Tags gateway
// @Produce json
// @Param id path string true "Provider ID"
// @Router /gateway/providers/{id}/rules [get]
func (h *Handler) ListRoutingRules(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	rules, err := h.service.ListRoutingRules(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

// IngestWebhook godoc
// @Summary Receive a webhook from an external provider
// @Tags gateway
// @Accept json
// @Produce json
// @Param provider_id path string true "Provider ID"
// @Router /gateway/ingest/{provider_id} [post]
func (h *Handler) IngestWebhook(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	providerID := c.Param("provider_id")

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	headers := make(map[string]string)
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	result, err := h.service.ProcessInboundWebhook(c.Request.Context(), tenantID, providerID, body, headers)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// ListInboundWebhooks godoc
// @Summary List received webhooks
// @Tags gateway
// @Produce json
// @Router /gateway/webhooks [get]
func (h *Handler) ListInboundWebhooks(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	providerID := c.Query("provider_id")
	limit, offset := 20, 0
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := c.Query("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	webhooks, total, err := h.service.ListInboundWebhooks(c.Request.Context(), tenantID, providerID, limit, offset)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"webhooks": webhooks,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// GetInboundWebhook godoc
// @Summary Get a specific received webhook
// @Tags gateway
// @Produce json
// @Param id path string true "Webhook ID"
// @Router /gateway/webhooks/{id} [get]
func (h *Handler) GetInboundWebhook(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	webhook, err := h.service.GetInboundWebhook(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	if webhook == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "webhook not found"})
		return
	}

	c.JSON(http.StatusOK, webhook)
}
