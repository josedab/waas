package edge

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CreateEdgeDispatchConfig handles creating an edge dispatch configuration.
func (h *Handler) CreateEdgeDispatchConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateEdgeDispatchConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := h.service.CreateEdgeDispatchConfig(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, config)
}

// GetEdgeDispatchConfig handles getting the edge dispatch config.
func (h *Handler) GetEdgeDispatchConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	config, err := h.service.GetEdgeDispatchConfig(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// DispatchWebhookEdge handles dispatching a webhook via the edge network.
func (h *Handler) DispatchWebhookEdge(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req DispatchWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.DispatchWebhook(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// RecordEdgeDelivery handles recording an edge delivery result.
func (h *Handler) RecordEdgeDelivery(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req RecordEdgeDeliveryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.RecordEdgeDelivery(c.Request.Context(), tenantID, &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "recorded"})
}

// GetEdgeNetworkOverview handles getting the edge network overview.
func (h *Handler) GetEdgeNetworkOverview(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	overview, err := h.service.GetEdgeNetworkOverview(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, overview)
}
