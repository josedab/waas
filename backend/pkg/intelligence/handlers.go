package intelligence

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	pkgerrors "github.com/josedab/waas/pkg/errors"
)

// Handler implements HTTP handlers for webhook intelligence
type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	intel := r.Group("/intelligence")
	{
		intel.GET("/dashboard", h.GetDashboard)

		intel.POST("/predict", h.PredictFailure)
		intel.GET("/predictions", h.GetPredictions)
		intel.POST("/predictions/:id/resolve", h.ResolvePrediction)

		intel.POST("/detect-anomalies", h.DetectAnomalies)
		intel.GET("/anomalies", h.GetAnomalies)
		intel.POST("/anomalies/:id/acknowledge", h.AcknowledgeAnomaly)

		intel.POST("/optimize-retry", h.OptimizeRetry)
		intel.GET("/optimizations", h.GetOptimizations)
		intel.POST("/optimizations/:id/apply", h.ApplyOptimization)

		intel.POST("/classify", h.ClassifyEvent)
		intel.GET("/classifications", h.GetClassifications)

		intel.POST("/health-score", h.CalculateHealthScore)
		intel.GET("/health-scores", h.GetHealthScores)

		intel.GET("/insights", h.GetInsights)
		intel.POST("/insights/:id/dismiss", h.DismissInsight)
	}
}

func (h *Handler) GetDashboard(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	dashboard, err := h.service.GetDashboard(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, dashboard)
}

func (h *Handler) PredictFailure(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req struct {
		EndpointID string         `json:"endpoint_id" binding:"required"`
		Features   *FeatureVector `json:"features" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	prediction, err := h.service.PredictFailure(c.Request.Context(), tenantID, req.EndpointID, req.Features)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, prediction)
}

func (h *Handler) GetPredictions(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	predictions, err := h.service.GetPredictions(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"predictions": predictions})
}

func (h *Handler) ResolvePrediction(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.repo.ResolvePrediction(c.Request.Context(), id); err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "prediction resolved"})
}

func (h *Handler) DetectAnomalies(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req struct {
		EndpointID string         `json:"endpoint_id" binding:"required"`
		Features   *FeatureVector `json:"features" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	anomalies, err := h.service.DetectAnomalies(c.Request.Context(), tenantID, req.EndpointID, req.Features)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"anomalies": anomalies})
}

func (h *Handler) GetAnomalies(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	anomalies, err := h.service.GetAnomalies(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"anomalies": anomalies})
}

func (h *Handler) AcknowledgeAnomaly(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.AcknowledgeAnomaly(c.Request.Context(), id); err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "anomaly acknowledged"})
}

func (h *Handler) OptimizeRetry(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req struct {
		EndpointID     string         `json:"endpoint_id" binding:"required"`
		CurrentRetries int            `json:"current_retries"`
		CurrentBackoff string         `json:"current_backoff"`
		Features       *FeatureVector `json:"features" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	opt, err := h.service.OptimizeRetry(c.Request.Context(), tenantID, req.EndpointID, req.CurrentRetries, req.CurrentBackoff, req.Features)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, opt)
}

func (h *Handler) GetOptimizations(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	opts, err := h.service.GetOptimizations(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"optimizations": opts})
}

func (h *Handler) ApplyOptimization(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.ApplyOptimization(c.Request.Context(), id); err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "optimization applied"})
}

func (h *Handler) ClassifyEvent(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req struct {
		WebhookID string         `json:"webhook_id" binding:"required"`
		EventType string         `json:"event_type" binding:"required"`
		Payload   map[string]any `json:"payload"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	classification, err := h.service.ClassifyEvent(c.Request.Context(), tenantID, req.WebhookID, req.EventType, req.Payload)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, classification)
}

func (h *Handler) GetClassifications(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	classifications, err := h.service.repo.GetClassifications(c.Request.Context(), tenantID, limit)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"classifications": classifications})
}

func (h *Handler) CalculateHealthScore(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req struct {
		EndpointID string         `json:"endpoint_id" binding:"required"`
		Features   *FeatureVector `json:"features" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	score, err := h.service.CalculateHealthScore(c.Request.Context(), tenantID, req.EndpointID, req.Features)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, score)
}

func (h *Handler) GetHealthScores(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	scores, err := h.service.GetHealthScores(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"health_scores": scores})
}

func (h *Handler) GetInsights(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	insights, err := h.service.repo.GetInsights(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"insights": insights})
}

func (h *Handler) DismissInsight(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DismissInsight(c.Request.Context(), id); err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "insight dismissed"})
}
