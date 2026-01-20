package protocols

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for protocol management
type Handler struct {
	service *Service
}

// NewHandler creates a new protocol handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers protocol routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	protocols := router.Group("/protocols")
	{
		protocols.GET("", h.ListSupportedProtocols)
		protocols.GET("/configs", h.ListConfigs)
		protocols.POST("/configs", h.CreateConfig)
		protocols.GET("/configs/:id", h.GetConfig)
		protocols.PUT("/configs/:id", h.UpdateConfig)
		protocols.DELETE("/configs/:id", h.DeleteConfig)
		protocols.POST("/configs/:id/test", h.TestConfig)
		protocols.POST("/configs/:id/enable", h.EnableConfig)
		protocols.POST("/configs/:id/disable", h.DisableConfig)
		protocols.GET("/endpoints/:endpointId", h.GetEndpointConfigs)
	}
}

// ListSupportedProtocols godoc
//
//	@Summary		List supported protocols
//	@Description	Get a list of supported webhook delivery protocols
//	@Tags			protocols
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Router			/protocols [get]
func (h *Handler) ListSupportedProtocols(c *gin.Context) {
	protocols := h.service.SupportedProtocols()
	c.JSON(http.StatusOK, gin.H{
		"protocols": protocols,
		"total":     len(protocols),
	})
}

// CreateConfig godoc
//
//	@Summary		Create protocol config
//	@Description	Create a new protocol configuration for an endpoint
//	@Tags			protocols
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateConfigRequest	true	"Config request"
//	@Success		201		{object}	DeliveryConfig
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/protocols/configs [post]
func (h *Handler) CreateConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	config, err := h.service.CreateConfig(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, config)
}

// ListConfigs godoc
//
//	@Summary		List protocol configs
//	@Description	Get a list of protocol configurations
//	@Tags			protocols
//	@Produce		json
//	@Param			protocol	query		string	false	"Filter by protocol"
//	@Param			limit		query		int		false	"Limit"		default(20)
//	@Param			offset		query		int		false	"Offset"	default(0)
//	@Success		200			{object}	map[string]interface{}
//	@Failure		401			{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/protocols/configs [get]
func (h *Handler) ListConfigs(c *gin.Context) {
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

	protocolFilter := c.Query("protocol")

	var configs []*DeliveryConfig
	var total int
	var err error

	if protocolFilter != "" {
		configs, total, err = h.service.ListByProtocol(c.Request.Context(), tenantID, Protocol(protocolFilter), limit, offset)
	} else {
		configs, total, err = h.service.ListConfigs(c.Request.Context(), tenantID, limit, offset)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"configs": configs,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// GetConfig godoc
//
//	@Summary		Get protocol config
//	@Description	Get protocol configuration details
//	@Tags			protocols
//	@Produce		json
//	@Param			id	path		string	true	"Config ID"
//	@Success		200	{object}	DeliveryConfig
//	@Failure		401	{object}	map[string]interface{}
//	@Failure		404	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/protocols/configs/{id} [get]
func (h *Handler) GetConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	configID := c.Param("id")
	config, err := h.service.GetConfig(c.Request.Context(), tenantID, configID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if config == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	c.JSON(http.StatusOK, config)
}

// UpdateConfig godoc
//
//	@Summary		Update protocol config
//	@Description	Update a protocol configuration
//	@Tags			protocols
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Config ID"
//	@Param			request	body		UpdateConfigRequest	true	"Update request"
//	@Success		200		{object}	DeliveryConfig
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/protocols/configs/{id} [put]
func (h *Handler) UpdateConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	configID := c.Param("id")
	var req UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	config, err := h.service.UpdateConfig(c.Request.Context(), tenantID, configID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// DeleteConfig godoc
//
//	@Summary		Delete protocol config
//	@Description	Delete a protocol configuration
//	@Tags			protocols
//	@Produce		json
//	@Param			id	path	string	true	"Config ID"
//	@Success		204	"No content"
//	@Failure		401	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/protocols/configs/{id} [delete]
func (h *Handler) DeleteConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	configID := c.Param("id")
	if err := h.service.DeleteConfig(c.Request.Context(), tenantID, configID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// TestConfig godoc
//
//	@Summary		Test protocol config
//	@Description	Test a protocol configuration by sending a test webhook
//	@Tags			protocols
//	@Produce		json
//	@Param			id	path		string	true	"Config ID"
//	@Success		200	{object}	DeliveryResponse
//	@Failure		400	{object}	map[string]interface{}
//	@Failure		401	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/protocols/configs/{id}/test [post]
func (h *Handler) TestConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	configID := c.Param("id")
	response, err := h.service.TestConfig(c.Request.Context(), tenantID, configID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// EnableConfig godoc
//
//	@Summary		Enable protocol config
//	@Description	Enable a protocol configuration
//	@Tags			protocols
//	@Produce		json
//	@Param			id	path		string	true	"Config ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		401	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/protocols/configs/{id}/enable [post]
func (h *Handler) EnableConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	configID := c.Param("id")
	enabled := true
	_, err := h.service.UpdateConfig(c.Request.Context(), tenantID, configID, &UpdateConfigRequest{
		Enabled: &enabled,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Configuration enabled",
	})
}

// DisableConfig godoc
//
//	@Summary		Disable protocol config
//	@Description	Disable a protocol configuration
//	@Tags			protocols
//	@Produce		json
//	@Param			id	path		string	true	"Config ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		401	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/protocols/configs/{id}/disable [post]
func (h *Handler) DisableConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	configID := c.Param("id")
	enabled := false
	_, err := h.service.UpdateConfig(c.Request.Context(), tenantID, configID, &UpdateConfigRequest{
		Enabled: &enabled,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Configuration disabled",
	})
}

// GetEndpointConfigs godoc
//
//	@Summary		Get endpoint protocol configs
//	@Description	Get all protocol configurations for an endpoint
//	@Tags			protocols
//	@Produce		json
//	@Param			endpointId	path		string	true	"Endpoint ID"
//	@Success		200			{object}	map[string]interface{}
//	@Failure		401			{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/protocols/endpoints/{endpointId} [get]
func (h *Handler) GetEndpointConfigs(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	endpointID := c.Param("endpointId")
	configs, err := h.service.GetEndpointConfigs(c.Request.Context(), tenantID, endpointID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"configs": configs,
		"total":   len(configs),
	})
}
