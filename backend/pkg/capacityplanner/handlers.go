package capacityplanner

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/josedab/waas/pkg/httputil"
)

// Handler provides HTTP handlers for capacity planning.
type Handler struct {
	service *Service
}

// NewHandler creates a new capacity planner handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers capacity planner routes.
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	capacity := router.Group("/capacity")
	{
		capacity.POST("/reports", h.GenerateReport)
		capacity.GET("/reports/:id", h.GetReport)
		capacity.GET("/usage", h.GetCurrentUsage)
		capacity.GET("/traffic-history", h.GetTrafficHistory)
		capacity.GET("/projections", h.GetProjections)
		capacity.GET("/recommendations", h.GetRecommendations)
		capacity.GET("/alerts", h.ListAlerts)
		capacity.POST("/alerts/:id/acknowledge", h.AcknowledgeAlert)
		capacity.POST("/alert-thresholds", h.SetAlertThreshold)
	}
}

// @Summary Generate a capacity report
// @Tags CapacityPlanner
// @Accept json
// @Produce json
// @Param body body GenerateReportRequest true "Report period"
// @Success 201 {object} CapacityReport
// @Router /capacity/reports [post]
func (h *Handler) GenerateReport(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req GenerateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httputil.APIErrorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}

	report, err := h.service.GenerateReport(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalError(c, "REPORT_GENERATION_FAILED", err)
		return
	}

	c.JSON(http.StatusCreated, report)
}

// @Summary Get a capacity report
// @Tags CapacityPlanner
// @Produce json
// @Param id path string true "Report ID"
// @Success 200 {object} CapacityReport
// @Router /capacity/reports/{id} [get]
func (h *Handler) GetReport(c *gin.Context) {
	reportID := c.Param("id")

	report, err := h.service.GetReport(c.Request.Context(), reportID)
	if err != nil {
		httputil.InternalError(c, "NOT_FOUND", err)
		return
	}

	c.JSON(http.StatusOK, report)
}

// @Summary Get current usage metrics
// @Tags CapacityPlanner
// @Produce json
// @Success 200 {object} UsageMetrics
// @Router /capacity/usage [get]
func (h *Handler) GetCurrentUsage(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	usage, err := h.service.GetCurrentUsage(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "USAGE_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, usage)
}

// @Summary Get traffic history
// @Tags CapacityPlanner
// @Produce json
// @Success 200 {object} map[string][]TrafficSnapshot
// @Router /capacity/traffic-history [get]
func (h *Handler) GetTrafficHistory(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	tr := TimeRange{
		Start: time.Now().AddDate(0, 0, -30),
		End:   time.Now(),
	}

	snapshots, err := h.service.GetTrafficHistory(c.Request.Context(), tenantID, tr)
	if err != nil {
		httputil.InternalError(c, "TRAFFIC_HISTORY_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"snapshots": snapshots})
}

// @Summary Get growth projections
// @Tags CapacityPlanner
// @Produce json
// @Success 200 {object} map[string][]GrowthProjection
// @Router /capacity/projections [get]
func (h *Handler) GetProjections(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	projections, err := h.service.GetProjections(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "PROJECTIONS_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"projections": projections})
}

// @Summary Get scaling recommendations
// @Tags CapacityPlanner
// @Produce json
// @Success 200 {object} map[string][]ScalingRecommendation
// @Router /capacity/recommendations [get]
func (h *Handler) GetRecommendations(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	recommendations, err := h.service.GetRecommendations(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "RECOMMENDATIONS_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"recommendations": recommendations})
}

// @Summary List capacity alerts
// @Tags CapacityPlanner
// @Produce json
// @Success 200 {object} map[string][]CapacityAlert
// @Router /capacity/alerts [get]
func (h *Handler) ListAlerts(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	alerts, err := h.service.ListAlerts(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "LIST_ALERTS_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"alerts": alerts, "count": len(alerts)})
}

// @Summary Acknowledge a capacity alert
// @Tags CapacityPlanner
// @Produce json
// @Param id path string true "Alert ID"
// @Success 200 {object} map[string]string
// @Router /capacity/alerts/{id}/acknowledge [post]
func (h *Handler) AcknowledgeAlert(c *gin.Context) {
	alertID := c.Param("id")

	if err := h.service.AcknowledgeAlert(c.Request.Context(), alertID); err != nil {
		httputil.InternalError(c, "NOT_FOUND", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "acknowledged"})
}

// @Summary Set alert threshold
// @Tags CapacityPlanner
// @Accept json
// @Produce json
// @Param body body SetAlertThresholdRequest true "Threshold configuration"
// @Success 200 {object} map[string]string
// @Router /capacity/alert-thresholds [post]
func (h *Handler) SetAlertThreshold(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req SetAlertThresholdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httputil.APIErrorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}

	if err := h.service.SetAlertThreshold(c.Request.Context(), tenantID, &req); err != nil {
		httputil.InternalError(c, "SET_THRESHOLD_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "threshold_set", "resource": req.Resource, "threshold": req.ThresholdValue})
}
