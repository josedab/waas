package intelligence

import (
	"net/http"

	"github.com/gin-gonic/gin"
	pkgerrors "github.com/josedab/waas/pkg/errors"
)

// RegisterMLRoutes registers ML model and auto-remediation routes
func (h *Handler) RegisterMLRoutes(r *gin.RouterGroup) {
	ml := r.Group("/intelligence/ml")
	{
		// ML Model training and metrics
		ml.POST("/train", h.TrainModel)
		ml.GET("/metrics", h.GetModelMetrics)
		ml.POST("/predict-gb", h.PredictWithGradientBoosting)

		// Root-cause analysis
		ml.POST("/analyze-failures", h.AnalyzeFailures)
		ml.POST("/cluster-failures", h.ClusterFailures)

		// Predictive SLA alerting
		ml.POST("/predict-sla-breach", h.PredictSLABreach)

		// Auto-remediation
		ml.POST("/remediate", h.ExecuteRemediation)
		ml.GET("/remediations", h.GetRemediations)

		// Adaptive retry
		ml.POST("/adaptive-retry/record", h.RecordDeliveryOutcome)
		ml.GET("/adaptive-retry/:endpoint_id", h.GetRetryProfile)

		// Failure reporting
		ml.POST("/failure-report", h.GenerateFailureReport)

		// Auto-adjust retry policy
		ml.POST("/auto-adjust-retry", h.AutoAdjustRetryPolicy)
	}
}

func (h *Handler) TrainModel(c *gin.Context) {
	var req struct {
		Samples []TrainingSample `json:"samples" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	model := NewGradientBoostingModel(10, 0.1)
	if err := model.Train(req.Samples); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "model trained successfully",
		"metrics": model.GetMetrics(),
	})
}

func (h *Handler) GetModelMetrics(c *gin.Context) {
	model := NewGradientBoostingModel(10, 0.1)
	c.JSON(http.StatusOK, model.GetMetrics())
}

func (h *Handler) PredictWithGradientBoosting(c *gin.Context) {
	var req struct {
		Features FeatureVector `json:"features" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	model := NewGradientBoostingModel(10, 0.1)
	prob := model.Predict(&req.Features)

	c.JSON(http.StatusOK, gin.H{
		"failure_probability": prob,
		"model":               "gradient_boosting",
	})
}

func (h *Handler) AnalyzeFailures(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req struct {
		EndpointID string        `json:"endpoint_id" binding:"required"`
		Logs       []DeliveryLog `json:"logs" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	analysis, err := h.service.AnalyzeFailures(c.Request.Context(), tenantID, req.EndpointID, req.Logs)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, analysis)
}

func (h *Handler) ClusterFailures(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req struct {
		Logs []DeliveryLog `json:"logs" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	clusters, err := h.service.ClusterFailures(c.Request.Context(), tenantID, req.Logs)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"clusters": clusters})
}

func (h *Handler) PredictSLABreach(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req struct {
		EndpointID string         `json:"endpoint_id" binding:"required"`
		Features   *FeatureVector `json:"features" binding:"required"`
		SLATarget  float64        `json:"sla_target" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	alert, err := h.service.PredictSLABreach(c.Request.Context(), tenantID, req.EndpointID, req.Features, req.SLATarget)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	if alert == nil {
		c.JSON(http.StatusOK, gin.H{"message": "no SLA breach predicted", "alert": nil})
		return
	}
	c.JSON(http.StatusOK, alert)
}

func (h *Handler) ExecuteRemediation(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req struct {
		EndpointID string `json:"endpoint_id" binding:"required"`
		ActionType string `json:"action_type" binding:"required"`
		Trigger    string `json:"trigger"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	remediator := NewAutoRemediator()
	action, err := remediator.ExecuteRemediation(c.Request.Context(), tenantID, req.EndpointID, req.ActionType, req.Trigger)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, action)
}

func (h *Handler) GetRemediations(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	remediator := NewAutoRemediator()
	actions := remediator.GetActions(tenantID)
	c.JSON(http.StatusOK, gin.H{"actions": actions})
}

func (h *Handler) RecordDeliveryOutcome(c *gin.Context) {
	var req struct {
		EndpointID string `json:"endpoint_id" binding:"required"`
		AttemptNum int    `json:"attempt_num" binding:"required"`
		Success    bool   `json:"success"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	tuner := NewAdaptiveRetryTuner()
	tuner.RecordDeliveryOutcome(req.EndpointID, req.AttemptNum, req.Success)
	c.JSON(http.StatusOK, gin.H{"message": "outcome recorded"})
}

func (h *Handler) GetRetryProfile(c *gin.Context) {
	endpointID := c.Param("endpoint_id")
	tuner := NewAdaptiveRetryTuner()
	profile := tuner.GetRetryProfile(endpointID)
	c.JSON(http.StatusOK, profile)
}

func (h *Handler) GenerateFailureReport(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req struct {
		Logs   []DeliveryLog `json:"logs"`
		Format string        `json:"format"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	if req.Format == "" {
		req.Format = "plain"
	}

	report, err := h.service.GenerateFailureReport(c.Request.Context(), tenantID, req.Logs, req.Format)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, report)
}

func (h *Handler) AutoAdjustRetryPolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req struct {
		EndpointID       string        `json:"endpoint_id" binding:"required"`
		Logs             []DeliveryLog `json:"logs" binding:"required"`
		CurrentMaxRetries int          `json:"current_max_retries"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	adjustment, err := h.service.AutoAdjustRetryPolicy(c.Request.Context(), tenantID, req.EndpointID, req.Logs, req.CurrentMaxRetries)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	if adjustment == nil {
		c.JSON(http.StatusOK, gin.H{"message": "no adjustment needed"})
		return
	}
	c.JSON(http.StatusOK, adjustment)
}
