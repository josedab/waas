package catalog

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RecordTraffic handles recording traffic samples for auto-discovery.
func (h *Handler) RecordTraffic(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req RecordTrafficRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	count, err := h.service.RecordTraffic(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"discovered": count})
}

// ListDiscoveries handles listing all discovered event types.
func (h *Handler) ListDiscoveries(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	discoveries, err := h.service.ListDiscoveries(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"discoveries": discoveries})
}

// GetDiscoverySummary handles getting the discovery summary.
func (h *Handler) GetDiscoverySummary(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	summary, err := h.service.GetDiscoverySummary(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, summary)
}

// InferSchema handles schema inference from traffic samples.
func (h *Handler) InferSchema(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	eventType := c.Param("event_type")

	inference, err := h.service.InferSchema(c.Request.Context(), tenantID, eventType)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, inference)
}

// PromoteDiscovery handles promoting a discovered event type to the catalog.
func (h *Handler) PromoteDiscovery(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req PromoteDiscoveryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Ensure tenantID is a valid UUID for the catalog service
	if _, err := uuid.Parse(tenantID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id format"})
		return
	}

	et, err := h.service.PromoteDiscovery(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, et)
}

// IgnoreDiscovery handles marking a discovered event type as ignored.
func (h *Handler) IgnoreDiscovery(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	discoveryID := c.Param("id")

	if err := h.service.IgnoreDiscovery(c.Request.Context(), tenantID, discoveryID); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ignored"})
}
