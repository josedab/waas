package handlers

import (
	"errors"
	apperrors "github.com/josedab/waas/pkg/errors"
	"github.com/josedab/waas/pkg/httputil"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/monitoring"
	"github.com/josedab/waas/pkg/repository"
	"github.com/josedab/waas/pkg/utils"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MonitoringHandler handles delivery history, health status, and alert endpoints.
type MonitoringHandler struct {
	deliveryAttemptRepo repository.DeliveryAttemptRepository
	webhookRepo         repository.WebhookEndpointRepository
	logger              *utils.Logger
	healthChecker       *monitoring.HealthChecker
	alertManager        *monitoring.AlertManager
	metricsRecorder     *monitoring.MetricsRecorder
}

// DeliveryHistoryRequest holds query parameters for filtering delivery history.
type DeliveryHistoryRequest struct {
	EndpointIDs []string  `form:"endpoint_ids" json:"endpoint_ids,omitempty"`
	Statuses    []string  `form:"statuses" json:"statuses,omitempty"`
	StartDate   time.Time `form:"start_date" json:"start_date,omitempty" time_format:"2006-01-02T15:04:05Z07:00"`
	EndDate     time.Time `form:"end_date" json:"end_date,omitempty" time_format:"2006-01-02T15:04:05Z07:00"`
	Limit       int       `form:"limit" json:"limit,omitempty"`
	Offset      int       `form:"offset" json:"offset,omitempty"`
}

// DeliveryHistoryResponse is the paginated response for delivery history queries.
type DeliveryHistoryResponse struct {
	Deliveries []DeliveryAttemptResponse `json:"deliveries"`
	Pagination PaginationResponse        `json:"pagination"`
}

// DeliveryAttemptResponse represents a single delivery attempt in API responses.
type DeliveryAttemptResponse struct {
	ID            uuid.UUID  `json:"id"`
	EndpointID    uuid.UUID  `json:"endpoint_id"`
	PayloadHash   string     `json:"payload_hash"`
	PayloadSize   int        `json:"payload_size"`
	Status        string     `json:"status"`
	HTTPStatus    *int       `json:"http_status"`
	ResponseBody  *string    `json:"response_body,omitempty"`
	ErrorMessage  *string    `json:"error_message,omitempty"`
	AttemptNumber int        `json:"attempt_number"`
	ScheduledAt   time.Time  `json:"scheduled_at"`
	DeliveredAt   *time.Time `json:"delivered_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

// DeliveryDetailResponse provides full delivery details including all retry attempts.
type DeliveryDetailResponse struct {
	DeliveryID uuid.UUID                 `json:"delivery_id"`
	Attempts   []DeliveryAttemptResponse `json:"attempts"`
	Summary    DeliverySummary           `json:"summary"`
}

// DeliverySummary aggregates delivery status including attempt counts and final error.
type DeliverySummary struct {
	TotalAttempts   int        `json:"total_attempts"`
	Status          string     `json:"status"`
	FirstAttemptAt  time.Time  `json:"first_attempt_at"`
	LastAttemptAt   *time.Time `json:"last_attempt_at"`
	NextRetryAt     *time.Time `json:"next_retry_at,omitempty"`
	FinalHTTPStatus *int       `json:"final_http_status,omitempty"`
	FinalErrorMsg   *string    `json:"final_error_message,omitempty"`
}

// PaginationResponse holds standard pagination metadata for list endpoints.
type PaginationResponse struct {
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	Total   int  `json:"total"`
	HasMore bool `json:"has_more"`
}

func NewMonitoringHandler(deliveryAttemptRepo repository.DeliveryAttemptRepository, webhookRepo repository.WebhookEndpointRepository, logger *utils.Logger, healthChecker *monitoring.HealthChecker, alertManager *monitoring.AlertManager, metricsRecorder *monitoring.MetricsRecorder) *MonitoringHandler {
	return &MonitoringHandler{
		deliveryAttemptRepo: deliveryAttemptRepo,
		webhookRepo:         webhookRepo,
		logger:              logger,
		healthChecker:       healthChecker,
		alertManager:        alertManager,
		metricsRecorder:     metricsRecorder,
	}
}

// GetDeliveryHistory retrieves webhook delivery history with filtering
// @Summary Get delivery history
// @Description Get paginated webhook delivery history with optional filtering by endpoint, status, and date range
// @Tags monitoring
// @Accept json
// @Produce json
// @Param endpoint_ids query []string false "Filter by endpoint IDs (comma-separated)"
// @Param statuses query []string false "Filter by delivery statuses (comma-separated)"
// @Param start_date query string false "Filter deliveries after this date (RFC3339 format)"
// @Param end_date query string false "Filter deliveries before this date (RFC3339 format)"
// @Param limit query int false "Number of results to return (max 100)" default(50)
// @Param offset query int false "Number of results to skip" default(0)
// @Success 200 {object} DeliveryHistoryResponse "Delivery history with pagination"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid or missing API key"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security ApiKeyAuth
// @Router /webhooks/deliveries [get]
func (h *MonitoringHandler) GetDeliveryHistory(c *gin.Context) {
	correlationID := c.GetHeader("X-Correlation-ID")
	if correlationID == "" {
		correlationID = uuid.New().String()
	}

	// Get tenant from context
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	// Parse request parameters
	var req DeliveryHistoryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.WarnWithCorrelation("Invalid delivery history request parameters", correlationID, map[string]interface{}{
			"error":      err.Error(),
			"tenant_id":  tenantID.String(),
			"request_id": c.GetHeader("X-Request-ID"),
		})
		BadRequest(c, "INVALID_REQUEST", "Invalid request parameters")
		return
	}

	// Set default pagination
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 50
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	// Parse endpoint IDs
	var endpointIDs []uuid.UUID
	for _, idStr := range req.EndpointIDs {
		if id, err := uuid.Parse(idStr); err == nil {
			endpointIDs = append(endpointIDs, id)
		} else {
			h.logger.WarnWithCorrelation("Invalid endpoint ID in request", correlationID, map[string]interface{}{
				"invalid_id": idStr,
				"tenant_id":  tenantID.String(),
				"request_id": c.GetHeader("X-Request-ID"),
			})
		}
	}

	// Build filters
	filters := repository.DeliveryHistoryFilters{
		EndpointIDs: endpointIDs,
		Statuses:    req.Statuses,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
	}

	h.logger.InfoWithCorrelation("Fetching delivery history", correlationID, map[string]interface{}{
		"tenant_id":  tenantID.String(),
		"filters":    filters,
		"limit":      req.Limit,
		"offset":     req.Offset,
		"request_id": c.GetHeader("X-Request-ID"),
	})

	// Get delivery history from repository
	attempts, totalCount, err := h.deliveryAttemptRepo.GetDeliveryHistoryWithFilters(
		c.Request.Context(),
		tenantID,
		filters,
		req.Limit,
		req.Offset,
	)
	if err != nil {
		h.logger.ErrorWithCorrelation("Failed to get delivery history", correlationID, map[string]interface{}{
			"error":      err.Error(),
			"tenant_id":  tenantID.String(),
			"request_id": c.GetHeader("X-Request-ID"),
		})
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "DATABASE_ERROR", Message: "Failed to retrieve delivery history"})
		return
	}

	// Convert to response format
	deliveries := make([]DeliveryAttemptResponse, len(attempts))
	for i, attempt := range attempts {
		deliveries[i] = DeliveryAttemptResponse{
			ID:            attempt.ID,
			EndpointID:    attempt.EndpointID,
			PayloadHash:   attempt.PayloadHash,
			PayloadSize:   attempt.PayloadSize,
			Status:        attempt.Status,
			HTTPStatus:    attempt.HTTPStatus,
			ResponseBody:  h.truncateResponseBody(attempt.ResponseBody),
			ErrorMessage:  attempt.ErrorMessage,
			AttemptNumber: attempt.AttemptNumber,
			ScheduledAt:   attempt.ScheduledAt,
			DeliveredAt:   attempt.DeliveredAt,
			CreatedAt:     attempt.CreatedAt,
		}
	}

	response := DeliveryHistoryResponse{
		Deliveries: deliveries,
		Pagination: PaginationResponse{
			Limit:   req.Limit,
			Offset:  req.Offset,
			Total:   totalCount,
			HasMore: req.Offset+req.Limit < totalCount,
		},
	}

	h.logger.InfoWithCorrelation("Delivery history retrieved successfully", correlationID, map[string]interface{}{
		"tenant_id":     tenantID.String(),
		"results_count": len(deliveries),
		"total_count":   totalCount,
		"request_id":    c.GetHeader("X-Request-ID"),
	})

	c.JSON(http.StatusOK, response)
}

// GetDeliveryDetails retrieves detailed information about a specific delivery
// @Summary Get delivery details
// @Description Get detailed information about a specific webhook delivery including all attempts and timeline
// @Tags monitoring
// @Accept json
// @Produce json
// @Param id path string true "Delivery ID" format(uuid)
// @Success 200 {object} DeliveryDetailResponse "Detailed delivery information"
// @Failure 400 {object} map[string]interface{} "Invalid delivery ID format"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid or missing API key"
// @Failure 404 {object} map[string]interface{} "Delivery not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security ApiKeyAuth
// @Router /webhooks/deliveries/{id} [get]
func (h *MonitoringHandler) GetDeliveryDetails(c *gin.Context) {
	correlationID := c.GetHeader("X-Correlation-ID")
	if correlationID == "" {
		correlationID = uuid.New().String()
	}

	// Parse delivery ID
	deliveryIDStr := c.Param("id")
	deliveryID, err := uuid.Parse(deliveryIDStr)
	if err != nil {
		h.logger.WarnWithCorrelation("Invalid delivery ID format", correlationID, map[string]interface{}{
			"delivery_id": deliveryIDStr,
			"error":       err.Error(),
			"request_id":  c.GetHeader("X-Request-ID"),
		})
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_ID", Message: "Invalid delivery ID format"})
		return
	}

	// Get tenant from context
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	h.logger.InfoWithCorrelation("Fetching delivery details", correlationID, map[string]interface{}{
		"delivery_id": deliveryID.String(),
		"tenant_id":   tenantID.String(),
		"request_id":  c.GetHeader("X-Request-ID"),
	})

	// Get all attempts for this delivery
	attempts, err := h.deliveryAttemptRepo.GetDeliveryAttemptsByDeliveryID(
		c.Request.Context(),
		deliveryID,
		tenantID,
	)
	if err != nil {
		h.logger.ErrorWithCorrelation("Failed to get delivery attempts", correlationID, map[string]interface{}{
			"error":       err.Error(),
			"delivery_id": deliveryID.String(),
			"tenant_id":   tenantID.String(),
			"request_id":  c.GetHeader("X-Request-ID"),
		})
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "DATABASE_ERROR", Message: "Failed to retrieve delivery details"})
		return
	}

	if len(attempts) == 0 {
		h.logger.WarnWithCorrelation("Delivery not found", correlationID, map[string]interface{}{
			"delivery_id": deliveryID.String(),
			"tenant_id":   tenantID.String(),
			"request_id":  c.GetHeader("X-Request-ID"),
		})
		c.JSON(http.StatusNotFound, ErrorResponse{Code: "DELIVERY_NOT_FOUND", Message: "Delivery not found"})
		return
	}

	// Convert to response format
	attemptResponses := make([]DeliveryAttemptResponse, len(attempts))
	for i, attempt := range attempts {
		attemptResponses[i] = DeliveryAttemptResponse{
			ID:            attempt.ID,
			EndpointID:    attempt.EndpointID,
			PayloadHash:   attempt.PayloadHash,
			PayloadSize:   attempt.PayloadSize,
			Status:        attempt.Status,
			HTTPStatus:    attempt.HTTPStatus,
			ResponseBody:  attempt.ResponseBody, // Full response body for details view
			ErrorMessage:  attempt.ErrorMessage,
			AttemptNumber: attempt.AttemptNumber,
			ScheduledAt:   attempt.ScheduledAt,
			DeliveredAt:   attempt.DeliveredAt,
			CreatedAt:     attempt.CreatedAt,
		}
	}

	// Build summary
	summary := h.buildDeliverySummary(attempts)

	response := DeliveryDetailResponse{
		DeliveryID: deliveryID,
		Attempts:   attemptResponses,
		Summary:    summary,
	}

	h.logger.InfoWithCorrelation("Delivery details retrieved successfully", correlationID, map[string]interface{}{
		"delivery_id":    deliveryID.String(),
		"tenant_id":      tenantID.String(),
		"attempts_count": len(attempts),
		"final_status":   summary.Status,
		"request_id":     c.GetHeader("X-Request-ID"),
	})

	c.JSON(http.StatusOK, response)
}

// GetEndpointDeliveryHistory handles GET /webhooks/endpoints/:id/deliveries
func (h *MonitoringHandler) GetEndpointDeliveryHistory(c *gin.Context) {
	correlationID := c.GetHeader("X-Correlation-ID")
	if correlationID == "" {
		correlationID = uuid.New().String()
	}

	// Parse endpoint ID
	endpointIDStr := c.Param("id")
	endpointID, err := uuid.Parse(endpointIDStr)
	if err != nil {
		h.logger.WarnWithCorrelation("Invalid endpoint ID format", correlationID, map[string]interface{}{
			"endpoint_id": endpointIDStr,
			"error":       err.Error(),
			"request_id":  c.GetHeader("X-Request-ID"),
		})
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_ID", Message: "Invalid endpoint ID format"})
		return
	}

	// Get tenant from context
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	// Verify endpoint ownership
	endpoint, err := h.webhookRepo.GetByID(c.Request.Context(), endpointID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			h.logger.WarnWithCorrelation("Endpoint not found", correlationID, map[string]interface{}{
				"endpoint_id": endpointID.String(),
				"tenant_id":   tenantID.String(),
				"request_id":  c.GetHeader("X-Request-ID"),
			})
			c.JSON(http.StatusNotFound, ErrorResponse{Code: "ENDPOINT_NOT_FOUND", Message: "Webhook endpoint not found"})
			return
		}

		h.logger.ErrorWithCorrelation("Failed to get webhook endpoint", correlationID, map[string]interface{}{
			"error":       err.Error(),
			"endpoint_id": endpointID.String(),
			"request_id":  c.GetHeader("X-Request-ID"),
		})
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "DATABASE_ERROR", Message: "Failed to retrieve webhook endpoint"})
		return
	}

	// Verify tenant ownership
	if endpoint.TenantID != tenantID {
		h.logger.WarnWithCorrelation("Access denied to endpoint", correlationID, map[string]interface{}{
			"endpoint_id":       endpointID.String(),
			"tenant_id":         tenantID.String(),
			"endpoint_owner_id": endpoint.TenantID.String(),
			"request_id":        c.GetHeader("X-Request-ID"),
		})
		c.JSON(http.StatusForbidden, ErrorResponse{Code: "FORBIDDEN", Message: "Access denied to this webhook endpoint"})
		return
	}

	// Parse pagination and filters
	p := httputil.ParsePagination(c)
	var statuses []string

	if statusStr := c.Query("status"); statusStr != "" {
		statuses = strings.Split(statusStr, ",")
	}

	h.logger.InfoWithCorrelation("Fetching endpoint delivery history", correlationID, map[string]interface{}{
		"endpoint_id": endpointID.String(),
		"tenant_id":   tenantID.String(),
		"limit":       p.Limit,
		"offset":      p.Offset,
		"statuses":    statuses,
		"request_id":  c.GetHeader("X-Request-ID"),
	})

	// Get delivery history
	attempts, err := h.deliveryAttemptRepo.GetDeliveryHistory(
		c.Request.Context(),
		endpointID,
		statuses,
		p.Limit,
		p.Offset,
	)
	if err != nil {
		h.logger.ErrorWithCorrelation("Failed to get endpoint delivery history", correlationID, map[string]interface{}{
			"error":       err.Error(),
			"endpoint_id": endpointID.String(),
			"tenant_id":   tenantID.String(),
			"request_id":  c.GetHeader("X-Request-ID"),
		})
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "DATABASE_ERROR", Message: "Failed to retrieve delivery history"})
		return
	}

	// Convert to response format
	deliveries := make([]DeliveryAttemptResponse, len(attempts))
	for i, attempt := range attempts {
		deliveries[i] = DeliveryAttemptResponse{
			ID:            attempt.ID,
			EndpointID:    attempt.EndpointID,
			PayloadHash:   attempt.PayloadHash,
			PayloadSize:   attempt.PayloadSize,
			Status:        attempt.Status,
			HTTPStatus:    attempt.HTTPStatus,
			ResponseBody:  h.truncateResponseBody(attempt.ResponseBody),
			ErrorMessage:  attempt.ErrorMessage,
			AttemptNumber: attempt.AttemptNumber,
			ScheduledAt:   attempt.ScheduledAt,
			DeliveredAt:   attempt.DeliveredAt,
			CreatedAt:     attempt.CreatedAt,
		}
	}

	response := gin.H{
		"deliveries": deliveries,
		"pagination": httputil.PaginationMeta{
			Limit:   p.Limit,
			Offset:  p.Offset,
			Total:   -1, // total unknown without a count query
			HasMore: len(deliveries) == p.Limit,
		},
	}

	h.logger.InfoWithCorrelation("Endpoint delivery history retrieved successfully", correlationID, map[string]interface{}{
		"endpoint_id":   endpointID.String(),
		"tenant_id":     tenantID.String(),
		"results_count": len(deliveries),
		"request_id":    c.GetHeader("X-Request-ID"),
	})

	c.JSON(http.StatusOK, response)
}

// Helper methods

func (h *MonitoringHandler) truncateResponseBody(responseBody *string) *string {
	if responseBody == nil {
		return nil
	}

	const maxLength = 1000
	if len(*responseBody) <= maxLength {
		return responseBody
	}

	truncated := (*responseBody)[:maxLength] + "... [truncated]"
	return &truncated
}

func (h *MonitoringHandler) buildDeliverySummary(attempts []*models.DeliveryAttempt) DeliverySummary {
	if len(attempts) == 0 {
		return DeliverySummary{}
	}

	// Sort attempts by attempt number to ensure correct order
	// (assuming they come sorted from the database)
	firstAttempt := attempts[0]
	lastAttempt := attempts[len(attempts)-1]

	summary := DeliverySummary{
		TotalAttempts:  len(attempts),
		Status:         lastAttempt.Status,
		FirstAttemptAt: firstAttempt.CreatedAt,
		LastAttemptAt:  &lastAttempt.CreatedAt,
	}

	// Set final status information
	if lastAttempt.HTTPStatus != nil {
		summary.FinalHTTPStatus = lastAttempt.HTTPStatus
	}
	if lastAttempt.ErrorMessage != nil {
		summary.FinalErrorMsg = lastAttempt.ErrorMessage
	}

	// Calculate next retry time if still retrying
	if lastAttempt.Status == "retrying" || lastAttempt.Status == "pending" {
		// This would typically be calculated based on retry configuration
		// For now, we'll use the scheduled_at time from the last attempt
		summary.NextRetryAt = &lastAttempt.ScheduledAt
	}

	return summary
}

// GetHealthStatus returns the current health status
// @Summary Get service health status
// @Description Get comprehensive health status of all service components
// @Tags monitoring
// @Accept json
// @Produce json
// @Success 200 {object} monitoring.HealthCheckResponse "Service health status"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /health [get]
func (h *MonitoringHandler) GetHealthStatus(c *gin.Context) {
	if h.healthChecker == nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "HEALTH_CHECKER_NOT_AVAILABLE", Message: "Health checker not initialized"})
		return
	}

	h.healthChecker.HealthCheckHandler()(c)
}

// GetReadinessStatus returns the readiness status
// @Summary Get service readiness status
// @Description Check if the service is ready to accept requests
// @Tags monitoring
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Service is ready"
// @Failure 503 {object} map[string]interface{} "Service is not ready"
// @Router /ready [get]
func (h *MonitoringHandler) GetReadinessStatus(c *gin.Context) {
	if h.healthChecker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"reason": "Health checker not initialized",
		})
		return
	}

	h.healthChecker.ReadinessHandler()(c)
}

// GetLivenessStatus returns the liveness status
// @Summary Get service liveness status
// @Description Check if the service is alive and responding
// @Tags monitoring
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Service is alive"
// @Router /live [get]
func (h *MonitoringHandler) GetLivenessStatus(c *gin.Context) {
	if h.healthChecker == nil {
		c.JSON(http.StatusOK, gin.H{
			"status":    "alive",
			"timestamp": time.Now(),
		})
		return
	}

	h.healthChecker.LivenessHandler()(c)
}

// GetActiveAlerts returns currently active alerts
// @Summary Get active alerts
// @Description Get all currently active monitoring alerts
// @Tags monitoring
// @Accept json
// @Produce json
// @Success 200 {object} []monitoring.Alert "List of active alerts"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /alerts/active [get]
func (h *MonitoringHandler) GetActiveAlerts(c *gin.Context) {
	if h.alertManager == nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "ALERT_MANAGER_NOT_AVAILABLE", Message: "Alert manager not initialized"})
		return
	}

	alerts := h.alertManager.GetActiveAlerts()
	c.JSON(http.StatusOK, gin.H{
		"alerts": alerts,
		"count":  len(alerts),
	})
}

// GetAlertHistory returns alert history
// @Summary Get alert history
// @Description Get historical alerts with optional filtering
// @Tags monitoring
// @Accept json
// @Produce json
// @Param limit query int false "Number of alerts to return" default(50)
// @Param severity query string false "Filter by severity (critical, warning, info)"
// @Success 200 {object} []monitoring.Alert "List of historical alerts"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /alerts/history [get]
func (h *MonitoringHandler) GetAlertHistory(c *gin.Context) {
	if h.alertManager == nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "ALERT_MANAGER_NOT_AVAILABLE", Message: "Alert manager not initialized"})
		return
	}

	// Parse query parameters
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 1000 {
			limit = parsedLimit
		}
	}

	severity := monitoring.AlertSeverity(c.Query("severity"))
	if severity != "" && severity != monitoring.AlertSeverityCritical &&
		severity != monitoring.AlertSeverityWarning && severity != monitoring.AlertSeverityInfo {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_SEVERITY", Message: "Invalid severity. Must be one of: critical, warning, info"})
		return
	}

	alerts := h.alertManager.GetAlertHistory(limit, severity)
	c.JSON(http.StatusOK, gin.H{
		"alerts": alerts,
		"count":  len(alerts),
		"limit":  limit,
	})
}
