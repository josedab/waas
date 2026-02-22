package dlq

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AnalyzeRootCause handles root-cause analysis for a DLQ entry.
func (h *Handler) AnalyzeRootCause(c *gin.Context) {
	entryID := c.Param("id")

	analysis, err := h.service.AnalyzeRootCause(c.Request.Context(), entryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, analysis)
}

// GetSmartRetryRecommendation returns a retry recommendation for a DLQ entry.
func (h *Handler) GetSmartRetryRecommendation(c *gin.Context) {
	entryID := c.Param("id")

	rec, err := h.service.GetSmartRetryRecommendation(c.Request.Context(), entryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, rec)
}

// AnalyzeEndpointHealth returns health analysis for an endpoint.
func (h *Handler) AnalyzeEndpointHealth(c *gin.Context) {
	tenantID := h.getTenantID(c)
	endpointID := c.Param("endpoint_id")

	health, err := h.service.AnalyzeEndpointHealth(c.Request.Context(), tenantID, endpointID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, health)
}
