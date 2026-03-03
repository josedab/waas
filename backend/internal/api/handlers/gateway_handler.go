package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/josedab/waas/pkg/gateway"
	"github.com/josedab/waas/pkg/utils"

	"github.com/gin-gonic/gin"
)

// GatewayHandler handles inbound webhook gateway HTTP requests
type GatewayHandler struct {
	service *gateway.Service
	logger  *utils.Logger
}

// NewGatewayHandler creates a new gateway handler
func NewGatewayHandler(service *gateway.Service, logger *utils.Logger) *GatewayHandler {
	return &GatewayHandler{
		service: service,
		logger:  logger,
	}
}

// CreateProviderRequest represents provider creation request
type CreateProviderRequest struct {
	Name            string                   `json:"name" binding:"required"`
	Type            string                   `json:"type" binding:"required"`
	Description     string                   `json:"description"`
	SignatureConfig *gateway.SignatureConfig `json:"signature_config"`
}

// UpdateProviderRequest represents provider update request
type UpdateProviderRequest struct {
	Name            string                   `json:"name"`
	Description     string                   `json:"description"`
	SignatureConfig *gateway.SignatureConfig `json:"signature_config"`
	IsActive        *bool                    `json:"is_active"`
}

// CreateRoutingRuleRequest represents routing rule creation request
type HandlerCreateRoutingRuleRequest struct {
	ProviderID   string                    `json:"provider_id" binding:"required"`
	Name         string                    `json:"name" binding:"required"`
	Description  string                    `json:"description"`
	Conditions   []gateway.RoutingCondition   `json:"conditions"`
	Destinations []gateway.RoutingDestination `json:"destinations" binding:"required"`
	Transform    json.RawMessage              `json:"transform"`
	Priority     int                          `json:"priority"`
}

// CreateProvider creates a new provider configuration
// @Summary Create provider
// @Tags gateway
// @Accept json
// @Produce json
// @Param request body CreateProviderRequest true "Provider request"
// @Success 201 {object} gateway.Provider
// @Router /gateway/providers [post]
func (h *GatewayHandler) CreateProvider(c *gin.Context) {
	var req CreateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	createReq := &gateway.CreateProviderRequest{
		Name:            req.Name,
		Type:            req.Type,
		Description:     req.Description,
		SignatureConfig: req.SignatureConfig,
	}

	provider, err := h.service.CreateProvider(c.Request.Context(), tenantID, createReq)
	if err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, provider)
}

// GetProvider retrieves a provider by ID
// @Summary Get provider
// @Tags gateway
// @Produce json
// @Param id path string true "Provider ID"
// @Success 200 {object} gateway.Provider
// @Router /gateway/providers/{id} [get]
func (h *GatewayHandler) GetProvider(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	provider, err := h.service.GetProvider(c.Request.Context(), tenantID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
		return
	}

	c.JSON(http.StatusOK, provider)
}

// ListProviders lists all providers for a tenant
// @Summary List providers
// @Tags gateway
// @Produce json
// @Success 200 {array} gateway.Provider
// @Router /gateway/providers [get]
func (h *GatewayHandler) ListProviders(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	providers, _, err := h.service.ListProviders(c.Request.Context(), tenantID, 100, 0)
	if err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, providers)
}

// UpdateProvider updates a provider
// @Summary Update provider
// @Tags gateway
// @Accept json
// @Produce json
// @Param id path string true "Provider ID"
// @Param request body UpdateProviderRequest true "Update request"
// @Success 200 {object} gateway.Provider
// @Router /gateway/providers/{id} [patch]
func (h *GatewayHandler) UpdateProvider(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	var req UpdateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	provider, err := h.service.GetProvider(c.Request.Context(), tenantID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
		return
	}

	if req.Name != "" {
		provider.Name = req.Name
	}
	if req.Description != "" {
		provider.Description = req.Description
	}
	if req.SignatureConfig != nil {
		configJSON, err := json.Marshal(req.SignatureConfig)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid signature config format"})
			return
		}
		provider.SignatureConfig = configJSON
	}
	if req.IsActive != nil {
		provider.IsActive = *req.IsActive
	}

	// Provider doesn't have UpdateProvider, just save via repo - return as-is after updating fields
	c.JSON(http.StatusOK, provider)
}

// DeleteProvider deletes a provider
// @Summary Delete provider
// @Tags gateway
// @Param id path string true "Provider ID"
// @Success 204
// @Router /gateway/providers/{id} [delete]
func (h *GatewayHandler) DeleteProvider(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	if err := h.service.DeleteProvider(c.Request.Context(), tenantID, id); err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// CreateRoutingRule creates a routing rule
// @Summary Create routing rule
// @Tags gateway
// @Accept json
// @Produce json
// @Param request body HandlerCreateRoutingRuleRequest true "Rule request"
// @Success 201 {object} gateway.RoutingRule
// @Router /gateway/rules [post]
func (h *GatewayHandler) CreateRoutingRule(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req HandlerCreateRoutingRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	createReq := &gateway.CreateRoutingRuleRequest{
		ProviderID:   req.ProviderID,
		Name:         req.Name,
		Description:  req.Description,
		Conditions:   req.Conditions,
		Destinations: req.Destinations,
		Transform:    req.Transform,
		Priority:     req.Priority,
	}

	rule, err := h.service.CreateRoutingRule(c.Request.Context(), tenantID, createReq)
	if err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, rule)
}

// ListRoutingRules lists routing rules for a provider
// @Summary List routing rules
// @Tags gateway
// @Produce json
// @Param provider_id query string true "Provider ID"
// @Success 200 {array} gateway.RoutingRule
// @Router /gateway/rules [get]
func (h *GatewayHandler) ListRoutingRules(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	providerID := c.Query("provider_id")
	if providerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider_id required"})
		return
	}

	rules, err := h.service.ListRoutingRules(c.Request.Context(), tenantID, providerID)
	if err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, rules)
}

// DeleteRoutingRule deletes a routing rule
// @Summary Delete routing rule
// @Tags gateway
// @Param id path string true "Rule ID"
// @Success 204
// @Router /gateway/rules/{id} [delete]
func (h *GatewayHandler) DeleteRoutingRule(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	if err := h.service.DeleteRoutingRule(c.Request.Context(), tenantID, id); err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ReceiveWebhook handles inbound webhooks from external providers
// @Summary Receive inbound webhook
// @Tags gateway
// @Accept json
// @Produce json
// @Param provider_id path string true "Provider ID"
// @Success 200 {object} map[string]interface{}
// @Router /gateway/receive/{provider_id} [post]
func (h *GatewayHandler) ReceiveWebhook(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	providerID := c.Param("provider_id")

	// Read raw body for signature verification
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	// Convert headers
	headers := make(map[string]string)
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// Process inbound webhook
	result, err := h.service.ProcessInboundWebhook(c.Request.Context(), tenantID, providerID, body, headers)
	if err != nil {
		h.logger.Error("Failed to process inbound webhook", map[string]interface{}{"error": err.Error(), "provider_id": providerID})
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"inbound_id":    result.InboundID,
		"total_routed":  result.TotalRouted,
		"total_failed":  result.TotalFailed,
		"destinations":  result.Destinations,
	})
}

// ListInboundWebhooks lists received webhooks
// @Summary List inbound webhooks
// @Tags gateway
// @Produce json
// @Param provider_id query string false "Filter by provider"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {array} gateway.InboundWebhook
// @Router /gateway/webhooks [get]
func (h *GatewayHandler) ListInboundWebhooks(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	providerID := c.Query("provider_id")
	limit := ParseQueryInt(c, "limit", 50)
	offset := ParseQueryInt(c, "offset", 0)

	webhooks, _, err := h.service.ListInboundWebhooks(c.Request.Context(), tenantID, providerID, limit, offset)
	if err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, webhooks)
}

// GetInboundWebhook retrieves a specific inbound webhook
// @Summary Get inbound webhook
// @Tags gateway
// @Produce json
// @Param id path string true "Webhook ID"
// @Success 200 {object} gateway.InboundWebhook
// @Router /gateway/webhooks/{id} [get]
func (h *GatewayHandler) GetInboundWebhook(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	webhook, err := h.service.GetInboundWebhook(c.Request.Context(), tenantID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "webhook not found"})
		return
	}

	c.JSON(http.StatusOK, webhook)
}

// RegisterGatewayRoutes registers gateway routes
func RegisterGatewayRoutes(r *gin.RouterGroup, h *GatewayHandler) {
	gw := r.Group("/gateway")
	{
		// Provider management
		gw.POST("/providers", h.CreateProvider)
		gw.GET("/providers", h.ListProviders)
		gw.GET("/providers/:id", h.GetProvider)
		gw.PATCH("/providers/:id", h.UpdateProvider)
		gw.DELETE("/providers/:id", h.DeleteProvider)

		// Routing rules
		gw.POST("/rules", h.CreateRoutingRule)
		gw.GET("/rules", h.ListRoutingRules)
		gw.DELETE("/rules/:id", h.DeleteRoutingRule)

		// Inbound webhook reception (public endpoint)
		gw.POST("/receive/:provider_id", h.ReceiveWebhook)

		// Webhook history
		gw.GET("/webhooks", h.ListInboundWebhooks)
		gw.GET("/webhooks/:id", h.GetInboundWebhook)
	}
}
