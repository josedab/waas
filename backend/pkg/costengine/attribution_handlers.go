package costengine

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// RecordCostEvent handles recording a cost attribution event.
func (h *Handler) RecordCostEvent(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req RecordCostEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	attr, err := h.service.RecordCostEvent(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, attr)
}

// GetTenantCostSummary handles getting a cost summary.
func (h *Handler) GetTenantCostSummary(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	period := c.DefaultQuery("period", "daily")

	summary, err := h.service.GetTenantCostSummary(c.Request.Context(), tenantID, period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// GenerateChargebackReport handles generating a chargeback report.
func (h *Handler) GenerateChargebackReport(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req GenerateChargebackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	report, err := h.service.GenerateChargebackReport(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, report)
}

// GetCostForecast handles getting a cost forecast.
func (h *Handler) GetCostForecast(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))

	forecast, err := h.service.GetCostForecast(c.Request.Context(), tenantID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, forecast)
}
