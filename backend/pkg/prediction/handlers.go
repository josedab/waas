package prediction

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for prediction service
type Handler struct {
	service *Service
}

// NewHandler creates a new handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers HTTP routes
func (h *Handler) RegisterRoutes(r gin.IRouter) {
	prediction := r.Group("/prediction")
	{
		// Dashboard
		prediction.GET("/dashboard", h.GetDashboard)

		// Endpoint Health
		prediction.GET("/health", h.ListEndpointHealth)
		prediction.GET("/health/:endpointId", h.GetEndpointHealth)
		prediction.POST("/health/:endpointId/calculate", h.CalculateHealth)

		// Predictions
		prediction.POST("/predict", h.Predict)
		prediction.GET("/predictions", h.ListPredictions)
		prediction.GET("/predictions/:id", h.GetPrediction)
		prediction.POST("/predictions/:id/outcome", h.RecordOutcome)

		// Alerts
		prediction.GET("/alerts", h.ListAlerts)
		prediction.GET("/alerts/:id", h.GetAlert)
		prediction.PUT("/alerts/:id", h.UpdateAlert)

		// Alert Rules
		prediction.POST("/rules", h.CreateAlertRule)
		prediction.GET("/rules", h.ListAlertRules)
		prediction.GET("/rules/:id", h.GetAlertRule)
		prediction.PUT("/rules/:id", h.UpdateAlertRule)
		prediction.DELETE("/rules/:id", h.DeleteAlertRule)

		// Notifications
		prediction.POST("/notifications", h.SaveNotificationConfig)
		prediction.GET("/notifications", h.ListNotificationConfigs)
		prediction.POST("/notifications/test", h.TestNotificationConfig)

		// Metrics
		prediction.POST("/metrics", h.RecordMetrics)
		prediction.POST("/failures", h.RecordFailure)
	}
}

// GetDashboard retrieves prediction dashboard
// @Summary Get prediction dashboard
// @Tags prediction
// @Produce json
// @Success 200 {object} DashboardStats
// @Router /prediction/dashboard [get]
func (h *Handler) GetDashboard(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	dashboard, err := h.service.GetDashboard(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

// ListEndpointHealth lists endpoint health
// @Summary List endpoint health
// @Tags prediction
// @Produce json
// @Success 200 {array} EndpointHealth
// @Router /prediction/health [get]
func (h *Handler) ListEndpointHealth(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	health, err := h.service.ListEndpointHealth(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, health)
}

// GetEndpointHealth retrieves health for an endpoint
// @Summary Get endpoint health
// @Tags prediction
// @Produce json
// @Param endpointId path string true "Endpoint ID"
// @Success 200 {object} EndpointHealth
// @Failure 404 {object} ErrorResponse
// @Router /prediction/health/{endpointId} [get]
func (h *Handler) GetEndpointHealth(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	endpointID := c.Param("endpointId")

	health, err := h.service.GetEndpointHealth(c.Request.Context(), tenantID, endpointID)
	if err != nil {
		if err == ErrEndpointNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "endpoint not found"})
			return
		}
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, health)
}

// CalculateHealth calculates health for an endpoint
// @Summary Calculate endpoint health
// @Tags prediction
// @Produce json
// @Param endpointId path string true "Endpoint ID"
// @Success 200 {object} EndpointHealth
// @Router /prediction/health/{endpointId}/calculate [post]
func (h *Handler) CalculateHealth(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	endpointID := c.Param("endpointId")

	health, err := h.service.CalculateEndpointHealth(c.Request.Context(), tenantID, endpointID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, health)
}

// Predict makes predictions for an endpoint
// @Summary Make predictions
// @Tags prediction
// @Accept json
// @Produce json
// @Param request body PredictRequest true "Prediction request"
// @Success 200 {array} Prediction
// @Failure 400 {object} ErrorResponse
// @Router /prediction/predict [post]
func (h *Handler) Predict(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	var req PredictRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	predictions, err := h.service.Predict(c.Request.Context(), tenantID, &req)
	if err != nil {
		if err == ErrInsufficientData {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "insufficient data for prediction"})
			return
		}
		if err == ErrModelNotReady {
			c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "prediction model not ready"})
			return
		}
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, predictions)
}

// ListPredictions lists predictions
// @Summary List predictions
// @Tags prediction
// @Produce json
// @Param endpoint_id query string false "Filter by endpoint ID"
// @Param type query string false "Filter by prediction type"
// @Param min_probability query number false "Minimum probability"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Success 200 {object} ListPredictionsResponse
// @Router /prediction/predictions [get]
func (h *Handler) ListPredictions(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	filters := &PredictionFilters{
		Page:     1,
		PageSize: 20,
	}

	if endpointID := c.Query("endpoint_id"); endpointID != "" {
		filters.EndpointID = endpointID
	}
	if pType := c.Query("type"); pType != "" {
		t := PredictionType(pType)
		filters.Type = &t
	}
	if minProb := c.Query("min_probability"); minProb != "" {
		if p, err := strconv.ParseFloat(minProb, 64); err == nil {
			filters.MinProbability = p
		}
	}
	if page, _ := strconv.Atoi(c.Query("page")); page > 0 {
		filters.Page = page
	}
	if pageSize, _ := strconv.Atoi(c.Query("page_size")); pageSize > 0 && pageSize <= 100 {
		filters.PageSize = pageSize
	}

	response, err := h.service.ListPredictions(c.Request.Context(), tenantID, filters)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetPrediction retrieves a prediction
// @Summary Get prediction
// @Tags prediction
// @Produce json
// @Param id path string true "Prediction ID"
// @Success 200 {object} Prediction
// @Failure 404 {object} ErrorResponse
// @Router /prediction/predictions/{id} [get]
func (h *Handler) GetPrediction(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	predictionID := c.Param("id")

	prediction, err := h.service.GetPrediction(c.Request.Context(), tenantID, predictionID)
	if err != nil {
		if err == ErrPredictionNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "prediction not found"})
			return
		}
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, prediction)
}

// RecordOutcome records a prediction outcome
// @Summary Record prediction outcome
// @Tags prediction
// @Accept json
// @Produce json
// @Param id path string true "Prediction ID"
// @Param request body PredictionOutcome true "Outcome"
// @Success 200
// @Failure 400 {object} ErrorResponse
// @Router /prediction/predictions/{id}/outcome [post]
func (h *Handler) RecordOutcome(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	predictionID := c.Param("id")

	var outcome PredictionOutcome
	if err := c.ShouldBindJSON(&outcome); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	err := h.service.RecordPredictionOutcome(c.Request.Context(), tenantID, predictionID, &outcome)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// ListAlerts lists alerts
// @Summary List alerts
// @Tags prediction
// @Produce json
// @Param endpoint_id query string false "Filter by endpoint ID"
// @Param type query string false "Filter by type"
// @Param severity query string false "Filter by severity"
// @Param status query string false "Filter by status"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Success 200 {object} ListAlertsResponse
// @Router /prediction/alerts [get]
func (h *Handler) ListAlerts(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	filters := &AlertFilters{
		Page:     1,
		PageSize: 20,
	}

	if endpointID := c.Query("endpoint_id"); endpointID != "" {
		filters.EndpointID = endpointID
	}
	if pType := c.Query("type"); pType != "" {
		t := PredictionType(pType)
		filters.Type = &t
	}
	if severity := c.Query("severity"); severity != "" {
		s := AlertSeverity(severity)
		filters.Severity = &s
	}
	if status := c.Query("status"); status != "" {
		s := AlertStatus(status)
		filters.Status = &s
	}
	if page, _ := strconv.Atoi(c.Query("page")); page > 0 {
		filters.Page = page
	}
	if pageSize, _ := strconv.Atoi(c.Query("page_size")); pageSize > 0 && pageSize <= 100 {
		filters.PageSize = pageSize
	}

	response, err := h.service.ListAlerts(c.Request.Context(), tenantID, filters)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetAlert retrieves an alert
// @Summary Get alert
// @Tags prediction
// @Produce json
// @Param id path string true "Alert ID"
// @Success 200 {object} Alert
// @Failure 404 {object} ErrorResponse
// @Router /prediction/alerts/{id} [get]
func (h *Handler) GetAlert(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	alertID := c.Param("id")

	alert, err := h.service.GetAlert(c.Request.Context(), tenantID, alertID)
	if err != nil {
		if err == ErrAlertNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "alert not found"})
			return
		}
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, alert)
}

// UpdateAlert updates an alert
// @Summary Update alert
// @Tags prediction
// @Accept json
// @Produce json
// @Param id path string true "Alert ID"
// @Param request body UpdateAlertRequest true "Update request"
// @Success 200 {object} Alert
// @Failure 400 {object} ErrorResponse
// @Router /prediction/alerts/{id} [put]
func (h *Handler) UpdateAlert(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	alertID := c.Param("id")
	userID := c.GetString("user_id")
	if userID == "" {
		userID = "unknown"
	}

	var req UpdateAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	alert, err := h.service.UpdateAlert(c.Request.Context(), tenantID, alertID, userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, alert)
}

// CreateAlertRule creates an alert rule
// @Summary Create alert rule
// @Tags prediction
// @Accept json
// @Produce json
// @Param request body CreateAlertRuleRequest true "Rule request"
// @Success 201 {object} AlertRule
// @Failure 400 {object} ErrorResponse
// @Router /prediction/rules [post]
func (h *Handler) CreateAlertRule(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	var req CreateAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	rule, err := h.service.CreateAlertRule(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, rule)
}

// ListAlertRules lists alert rules
// @Summary List alert rules
// @Tags prediction
// @Produce json
// @Success 200 {array} AlertRule
// @Router /prediction/rules [get]
func (h *Handler) ListAlertRules(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	rules, err := h.service.ListAlertRules(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, rules)
}

// GetAlertRule retrieves an alert rule
// @Summary Get alert rule
// @Tags prediction
// @Produce json
// @Param id path string true "Rule ID"
// @Success 200 {object} AlertRule
// @Failure 404 {object} ErrorResponse
// @Router /prediction/rules/{id} [get]
func (h *Handler) GetAlertRule(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	ruleID := c.Param("id")

	rule, err := h.service.GetAlertRule(c.Request.Context(), tenantID, ruleID)
	if err != nil {
		if err == ErrAlertRuleNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "rule not found"})
			return
		}
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, rule)
}

// UpdateAlertRule updates an alert rule
// @Summary Update alert rule
// @Tags prediction
// @Accept json
// @Produce json
// @Param id path string true "Rule ID"
// @Param request body AlertRule true "Rule update"
// @Success 200 {object} AlertRule
// @Failure 400 {object} ErrorResponse
// @Router /prediction/rules/{id} [put]
func (h *Handler) UpdateAlertRule(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	ruleID := c.Param("id")

	var rule AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	rule.ID = ruleID
	rule.TenantID = tenantID

	err := h.service.UpdateAlertRule(c.Request.Context(), tenantID, &rule)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// DeleteAlertRule deletes an alert rule
// @Summary Delete alert rule
// @Tags prediction
// @Param id path string true "Rule ID"
// @Success 204
// @Failure 404 {object} ErrorResponse
// @Router /prediction/rules/{id} [delete]
func (h *Handler) DeleteAlertRule(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	ruleID := c.Param("id")

	err := h.service.DeleteAlertRule(c.Request.Context(), tenantID, ruleID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "rule not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

// SaveNotificationConfig saves a notification configuration
// @Summary Save notification config
// @Tags prediction
// @Accept json
// @Produce json
// @Param request body NotificationConfig true "Notification config"
// @Success 201 {object} NotificationConfig
// @Failure 400 {object} ErrorResponse
// @Router /prediction/notifications [post]
func (h *Handler) SaveNotificationConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	var config NotificationConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	err := h.service.SaveNotificationConfig(c.Request.Context(), tenantID, &config)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, config)
}

// ListNotificationConfigs lists notification configurations
// @Summary List notification configs
// @Tags prediction
// @Produce json
// @Success 200 {array} NotificationConfig
// @Router /prediction/notifications [get]
func (h *Handler) ListNotificationConfigs(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	configs, err := h.service.ListNotificationConfigs(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, configs)
}

// TestNotificationConfig tests a notification configuration
// @Summary Test notification config
// @Tags prediction
// @Accept json
// @Produce json
// @Param request body NotificationConfig true "Notification config"
// @Success 200
// @Failure 400 {object} ErrorResponse
// @Router /prediction/notifications/test [post]
func (h *Handler) TestNotificationConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	var config NotificationConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	err := h.service.TestNotificationConfig(c.Request.Context(), tenantID, &config)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Test notification sent"})
}

// RecordMetrics records endpoint metrics
// @Summary Record metrics
// @Tags prediction
// @Accept json
// @Produce json
// @Param request body MetricDataPoint true "Metric data"
// @Success 201
// @Failure 400 {object} ErrorResponse
// @Router /prediction/metrics [post]
func (h *Handler) RecordMetrics(c *gin.Context) {
	var dataPoint MetricDataPoint
	if err := c.ShouldBindJSON(&dataPoint); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	err := h.service.RecordMetrics(c.Request.Context(), &dataPoint)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}

// RecordFailure records a failure event
// @Summary Record failure event
// @Tags prediction
// @Accept json
// @Produce json
// @Param request body FailureEvent true "Failure event"
// @Success 201
// @Failure 400 {object} ErrorResponse
// @Router /prediction/failures [post]
func (h *Handler) RecordFailure(c *gin.Context) {
	var failure FailureEvent
	if err := c.ShouldBindJSON(&failure); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	err := h.service.RecordFailure(c.Request.Context(), &failure)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}
