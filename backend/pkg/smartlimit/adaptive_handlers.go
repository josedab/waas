package smartlimit

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CreateAdaptiveConfig handles creating an adaptive rate limit config.
func (h *Handler) CreateAdaptiveConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateRecvAdaptiveConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := h.service.CreateAdaptiveConfig(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, config)
}

// GetAdaptiveConfig handles getting an adaptive rate limit config.
func (h *Handler) GetAdaptiveConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	endpointID := c.Param("endpointId")

	config, err := h.service.GetAdaptiveConfig(c.Request.Context(), tenantID, endpointID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// RecordDeliveryResult handles recording a delivery result for adaptive rate adjustment.
func (h *Handler) RecordRecvDeliveryResult(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req RecvDeliveryResultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	health, err := h.service.RecordRecvDeliveryResult(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, health)
}

// GetReceiverHealth handles getting the receiver health status.
func (h *Handler) GetReceiverHealth(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	endpointID := c.Param("endpointId")

	health, err := h.service.GetReceiverHealth(c.Request.Context(), tenantID, endpointID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, health)
}

// GetAdaptiveStats handles getting adaptive rate limiting stats.
func (h *Handler) GetAdaptiveStats(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	endpointID := c.Param("endpointId")

	stats, err := h.service.GetAdaptiveStats(c.Request.Context(), tenantID, endpointID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}
