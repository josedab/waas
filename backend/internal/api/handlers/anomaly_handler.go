package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"webhook-platform/pkg/anomaly"
	"webhook-platform/pkg/utils"

	"github.com/gin-gonic/gin"
)

// AnomalyHandler handles anomaly detection HTTP requests
type AnomalyHandler struct {
	service *anomaly.Service
	logger  *utils.Logger
}

// NewAnomalyHandler creates a new anomaly handler
func NewAnomalyHandler(service *anomaly.Service, logger *utils.Logger) *AnomalyHandler {
	return &AnomalyHandler{
		service: service,
		logger:  logger,
	}
}

// UpdateConfigRequest represents detection config update
type UpdateConfigRequest struct {
	Enabled           *bool    `json:"enabled"`
	Sensitivity       *float64 `json:"sensitivity"`
	MinSamples        *int     `json:"min_samples"`
	CooldownMinutes   *int     `json:"cooldown_minutes"`
	CriticalThreshold *float64 `json:"critical_threshold"`
	WarningThreshold  *float64 `json:"warning_threshold"`
}

// CreateAlertConfigRequest represents alert config creation
type CreateAlertConfigRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Channel     string                 `json:"channel" binding:"required"`
	Config      map[string]interface{} `json:"config" binding:"required"`
	MinSeverity string                 `json:"min_severity"`
}

// ListAnomalies lists detected anomalies
// @Summary List anomalies
// @Tags anomaly
// @Produce json
// @Param status query string false "Filter by status (open, acknowledged, resolved)"
// @Param severity query string false "Filter by severity"
// @Param endpoint_id query string false "Filter by endpoint"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {array} anomaly.Anomaly
// @Router /anomalies [get]
func (h *AnomalyHandler) ListAnomalies(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	status := c.Query("status")
	severity := c.Query("severity")
	endpointID := c.Query("endpoint_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	anomalies, err := h.service.ListAnomalies(c.Request.Context(), tenantID, status, severity, endpointID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, anomalies)
}

// GetAnomaly gets anomaly details
// @Summary Get anomaly
// @Tags anomaly
// @Produce json
// @Param id path string true "Anomaly ID"
// @Success 200 {object} anomaly.Anomaly
// @Router /anomalies/{id} [get]
func (h *AnomalyHandler) GetAnomaly(c *gin.Context) {
	id := c.Param("id")

	a, err := h.service.GetAnomaly(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "anomaly not found"})
		return
	}

	c.JSON(http.StatusOK, a)
}

// AcknowledgeAnomaly acknowledges an anomaly
// @Summary Acknowledge anomaly
// @Tags anomaly
// @Param id path string true "Anomaly ID"
// @Success 200 {object} anomaly.Anomaly
// @Router /anomalies/{id}/acknowledge [post]
func (h *AnomalyHandler) AcknowledgeAnomaly(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	a, err := h.service.AcknowledgeAnomaly(c.Request.Context(), tenantID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, a)
}

// ResolveAnomaly resolves an anomaly
// @Summary Resolve anomaly
// @Tags anomaly
// @Param id path string true "Anomaly ID"
// @Success 200 {object} anomaly.Anomaly
// @Router /anomalies/{id}/resolve [post]
func (h *AnomalyHandler) ResolveAnomaly(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	a, err := h.service.ResolveAnomaly(c.Request.Context(), tenantID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, a)
}

// GetBaselines gets current baselines
// @Summary Get baselines
// @Tags anomaly
// @Produce json
// @Param endpoint_id query string false "Filter by endpoint"
// @Success 200 {array} anomaly.Baseline
// @Router /anomalies/baselines [get]
func (h *AnomalyHandler) GetBaselines(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	endpointID := c.Query("endpoint_id")

	baselines, err := h.service.GetBaselines(c.Request.Context(), tenantID, endpointID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, baselines)
}

// RecalculateBaselines triggers baseline recalculation
// @Summary Recalculate baselines
// @Tags anomaly
// @Produce json
// @Param endpoint_id query string false "Specific endpoint"
// @Success 200 {object} map[string]interface{}
// @Router /anomalies/baselines/recalculate [post]
func (h *AnomalyHandler) RecalculateBaselines(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	endpointID := c.Query("endpoint_id")

	if err := h.service.RecalculateBaselines(c.Request.Context(), tenantID, endpointID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "baseline recalculation triggered",
	})
}

// GetDetectionConfig gets detection configuration
// @Summary Get detection config
// @Tags anomaly
// @Produce json
// @Param endpoint_id query string false "Endpoint ID"
// @Param metric_type query string false "Metric type"
// @Success 200 {object} anomaly.DetectionConfig
// @Router /anomalies/config [get]
func (h *AnomalyHandler) GetDetectionConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	endpointID := c.Query("endpoint_id")
	metricType := c.Query("metric_type")

	config, err := h.service.GetDetectionConfig(c.Request.Context(), tenantID, endpointID, metricType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// UpdateDetectionConfig updates detection configuration
// @Summary Update detection config
// @Tags anomaly
// @Accept json
// @Produce json
// @Param endpoint_id query string false "Endpoint ID"
// @Param metric_type query string false "Metric type"
// @Param request body UpdateConfigRequest true "Config updates"
// @Success 200 {object} anomaly.DetectionConfig
// @Router /anomalies/config [patch]
func (h *AnomalyHandler) UpdateDetectionConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	endpointID := c.Query("endpoint_id")
	metricType := c.Query("metric_type")

	config, err := h.service.GetDetectionConfig(c.Request.Context(), tenantID, endpointID, metricType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if req.Enabled != nil {
		config.Enabled = *req.Enabled
	}
	if req.Sensitivity != nil {
		config.Sensitivity = *req.Sensitivity
	}
	if req.MinSamples != nil {
		config.MinSamples = *req.MinSamples
	}
	if req.CooldownMinutes != nil {
		config.CooldownMinutes = *req.CooldownMinutes
	}
	if req.CriticalThreshold != nil {
		config.CriticalThreshold = *req.CriticalThreshold
	}
	if req.WarningThreshold != nil {
		config.WarningThreshold = *req.WarningThreshold
	}

	if err := h.service.UpdateDetectionConfig(c.Request.Context(), config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// ListAlertConfigs lists alert configurations
// @Summary List alert configs
// @Tags anomaly
// @Produce json
// @Success 200 {array} anomaly.AlertConfig
// @Router /anomalies/alerts/config [get]
func (h *AnomalyHandler) ListAlertConfigs(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	configs, err := h.service.ListAlertConfigs(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, configs)
}

// CreateAlertConfig creates an alert configuration
// @Summary Create alert config
// @Tags anomaly
// @Accept json
// @Produce json
// @Param request body CreateAlertConfigRequest true "Alert config"
// @Success 201 {object} anomaly.AlertConfig
// @Router /anomalies/alerts/config [post]
func (h *AnomalyHandler) CreateAlertConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateAlertConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert config map to JSON string
	configJSON, _ := json.Marshal(req.Config)

	alertReq := &anomaly.CreateAlertConfigRequest{
		Name:        req.Name,
		Channel:     req.Channel,
		Config:      string(configJSON),
		MinSeverity: anomaly.Severity(req.MinSeverity),
	}

	config, err := h.service.CreateAlertConfig(c.Request.Context(), tenantID, alertReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, config)
}

// DeleteAlertConfig deletes an alert configuration
// @Summary Delete alert config
// @Tags anomaly
// @Param id path string true "Alert config ID"
// @Success 204
// @Router /anomalies/alerts/config/{id} [delete]
func (h *AnomalyHandler) DeleteAlertConfig(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.DeleteAlertConfig(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// ListAlerts lists sent alerts
// @Summary List alerts
// @Tags anomaly
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {array} anomaly.Alert
// @Router /anomalies/alerts [get]
func (h *AnomalyHandler) ListAlerts(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	alerts, err := h.service.ListAlerts(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, alerts)
}

// GetAnomalyStats gets anomaly statistics
// @Summary Get anomaly stats
// @Tags anomaly
// @Produce json
// @Param period query string false "Period (7d, 30d, 90d)"
// @Success 200 {object} map[string]interface{}
// @Router /anomalies/stats [get]
func (h *AnomalyHandler) GetAnomalyStats(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	_ = c.DefaultQuery("period", "7d")

	// In production, calculate from database
	stats := gin.H{
		"total_anomalies":    45,
		"open":               5,
		"acknowledged":       8,
		"resolved":           32,
		"critical":           3,
		"warning":            12,
		"info":               30,
		"alerts_sent":        67,
		"avg_resolution_time": "4h 23m",
	}

	c.JSON(http.StatusOK, stats)
}

// RegisterAnomalyRoutes registers anomaly routes
func RegisterAnomalyRoutes(r *gin.RouterGroup, h *AnomalyHandler) {
	an := r.Group("/anomalies")
	{
		// Anomaly management
		an.GET("", h.ListAnomalies)
		an.GET("/stats", h.GetAnomalyStats)
		an.GET("/:id", h.GetAnomaly)
		an.POST("/:id/acknowledge", h.AcknowledgeAnomaly)
		an.POST("/:id/resolve", h.ResolveAnomaly)

		// Baselines
		an.GET("/baselines", h.GetBaselines)
		an.POST("/baselines/recalculate", h.RecalculateBaselines)

		// Detection config
		an.GET("/config", h.GetDetectionConfig)
		an.PATCH("/config", h.UpdateDetectionConfig)

		// Alert configs
		an.GET("/alerts/config", h.ListAlertConfigs)
		an.POST("/alerts/config", h.CreateAlertConfig)
		an.DELETE("/alerts/config/:id", h.DeleteAlertConfig)

		// Alert history
		an.GET("/alerts", h.ListAlerts)
	}
}
