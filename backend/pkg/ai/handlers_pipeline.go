package ai

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// PipelineHandler provides HTTP handlers for the unified prediction pipeline
type PipelineHandler struct {
	pipeline *PredictionPipeline
}

// NewPipelineHandler creates a new pipeline handler
func NewPipelineHandler(pipeline *PredictionPipeline) *PipelineHandler {
	return &PipelineHandler{pipeline: pipeline}
}

// RegisterPipelineRoutes registers prediction pipeline routes
func (h *PipelineHandler) RegisterPipelineRoutes(r *gin.RouterGroup) {
	ai := r.Group("/intelligence")
	{
		ai.GET("/dashboard", h.GetDashboard)
		ai.GET("/predictions", h.ListPredictions)
		ai.POST("/predictions/:id/acknowledge", h.AcknowledgePrediction)
		ai.POST("/evaluate/:endpointId", h.EvaluateEndpoint)
		ai.POST("/remediation/:actionId/execute", h.ExecuteRemediation)
		ai.GET("/reports", h.ListReports)
		ai.GET("/reports/:reportId", h.GetReport)
		ai.POST("/reports/generate", h.GenerateReport)
	}
}

// GetDashboard retrieves the unified intelligence dashboard
// @Summary Get intelligence dashboard
// @Tags ai
// @Produce json
// @Success 200 {object} IntelligenceDashboard
// @Router /intelligence/dashboard [get]
func (h *PipelineHandler) GetDashboard(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	dashboard, err := h.pipeline.GetDashboard(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

// ListPredictions lists active predictions
// @Summary List active predictions
// @Tags ai
// @Produce json
// @Success 200 {array} UnifiedPrediction
// @Router /intelligence/predictions [get]
func (h *PipelineHandler) ListPredictions(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	predictions, err := h.pipeline.ListPredictions(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, predictions)
}

// AcknowledgePrediction acknowledges a prediction
// @Summary Acknowledge prediction
// @Tags ai
// @Param id path string true "Prediction ID"
// @Success 204 "No content"
// @Router /intelligence/predictions/{id}/acknowledge [post]
func (h *PipelineHandler) AcknowledgePrediction(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	predictionID := c.Param("id")
	if err := h.pipeline.AcknowledgePrediction(c.Request.Context(), tenantID, predictionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// EvaluateEndpoint runs prediction evaluation for an endpoint
// @Summary Evaluate endpoint for predictions
// @Tags ai
// @Accept json
// @Produce json
// @Param endpointId path string true "Endpoint ID"
// @Success 200 {object} UnifiedPrediction
// @Router /intelligence/evaluate/{endpointId} [post]
func (h *PipelineHandler) EvaluateEndpoint(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	endpointID := c.Param("endpointId")
	var req struct {
		Failures []DeliveryContext `json:"failures" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	prediction, err := h.pipeline.EvaluateEndpoint(c.Request.Context(), tenantID, endpointID, req.Failures)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if prediction == nil {
		c.JSON(http.StatusOK, gin.H{"message": "no prediction generated"})
		return
	}

	c.JSON(http.StatusOK, prediction)
}

// ExecuteRemediation executes a remediation action
// @Summary Execute remediation action
// @Tags ai
// @Param actionId path string true "Action ID"
// @Success 202 {object} map[string]interface{}
// @Router /intelligence/remediation/{actionId}/execute [post]
func (h *PipelineHandler) ExecuteRemediation(c *gin.Context) {
	actionID := c.Param("actionId")
	if err := h.pipeline.ExecuteRemediation(c.Request.Context(), actionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"status": "executing"})
}

// ListReports lists root-cause analysis reports
// @Summary List root-cause reports
// @Tags ai
// @Produce json
// @Success 200 {array} RootCauseReport
// @Router /intelligence/reports [get]
func (h *PipelineHandler) ListReports(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit := 10
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}

	reports, err := h.pipeline.repo.ListRootCauseReports(c.Request.Context(), tenantID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, reports)
}

// GetReport retrieves a specific root-cause report
// @Summary Get root-cause report
// @Tags ai
// @Produce json
// @Param reportId path string true "Report ID"
// @Success 200 {object} RootCauseReport
// @Router /intelligence/reports/{reportId} [get]
func (h *PipelineHandler) GetReport(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	reportID := c.Param("reportId")
	report, err := h.pipeline.repo.GetRootCauseReport(c.Request.Context(), tenantID, reportID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "report not found"})
		return
	}

	c.JSON(http.StatusOK, report)
}

// GenerateReport generates a root-cause analysis report
// @Summary Generate root-cause report
// @Tags ai
// @Accept json
// @Produce json
// @Success 201 {object} RootCauseReport
// @Router /intelligence/reports/generate [post]
func (h *PipelineHandler) GenerateReport(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		EndpointID string            `json:"endpoint_id" binding:"required"`
		Failures   []DeliveryContext `json:"failures" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	report, err := h.pipeline.GenerateRootCauseReport(c.Request.Context(), tenantID, req.EndpointID, req.Failures)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, report)
}
