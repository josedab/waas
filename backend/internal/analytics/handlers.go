package analytics

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handlers contains the HTTP handlers for analytics endpoints
type Handlers struct {
	analyticsRepo repository.AnalyticsRepositoryInterface
}

// NewHandlers creates a new analytics handlers instance
func NewHandlers(analyticsRepo repository.AnalyticsRepositoryInterface) *Handlers {
	return &Handlers{
		analyticsRepo: analyticsRepo,
	}
}

// RegisterRoutes registers all analytics routes
func (h *Handlers) RegisterRoutes(router *gin.Engine) {
	analytics := router.Group("/analytics")
	{
		analytics.GET("/dashboard", h.GetDashboard)
		analytics.GET("/metrics", h.GetMetrics)
		analytics.GET("/metrics/summary", h.GetMetricsSummary)
		analytics.GET("/metrics/hourly", h.GetHourlyMetrics)
		analytics.GET("/metrics/realtime", h.GetRealtimeMetrics)
		analytics.GET("/endpoints/:endpoint_id/metrics", h.GetEndpointMetrics)
	}
}

// GetDashboard returns dashboard metrics for a tenant
func (h *Handlers) GetDashboard(c *gin.Context) {
	tenantID, err := h.getTenantIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant"})
		return
	}

	// Parse time window parameter (default to 24 hours)
	timeWindowStr := c.DefaultQuery("window", "24h")
	timeWindow, err := time.ParseDuration(timeWindowStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid time window format"})
		return
	}

	dashboard, err := h.analyticsRepo.GetDashboardMetrics(c.Request.Context(), tenantID, timeWindow)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get dashboard metrics"})
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

// GetMetrics returns delivery metrics based on query parameters
func (h *Handlers) GetMetrics(c *gin.Context) {
	tenantID, err := h.getTenantIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant"})
		return
	}

	query, err := h.parseMetricsQuery(c, tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	metrics, err := h.analyticsRepo.GetDeliveryMetrics(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get metrics"})
		return
	}

	// Get summary for the same query
	summary, err := h.analyticsRepo.GetMetricsSummary(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get metrics summary"})
		return
	}

	response := &models.MetricsResponse{
		Data:       make([]interface{}, len(metrics)),
		TotalCount: len(metrics),
		Summary:    *summary,
	}

	for i, metric := range metrics {
		response.Data[i] = metric
	}

	c.JSON(http.StatusOK, response)
}

// GetMetricsSummary returns aggregated summary statistics
func (h *Handlers) GetMetricsSummary(c *gin.Context) {
	tenantID, err := h.getTenantIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant"})
		return
	}

	query, err := h.parseMetricsQuery(c, tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	summary, err := h.analyticsRepo.GetMetricsSummary(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get metrics summary"})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// GetHourlyMetrics returns hourly aggregated metrics
func (h *Handlers) GetHourlyMetrics(c *gin.Context) {
	tenantID, err := h.getTenantIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant"})
		return
	}

	query, err := h.parseMetricsQuery(c, tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	metrics, err := h.analyticsRepo.GetHourlyMetrics(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get hourly metrics"})
		return
	}

	response := &models.MetricsResponse{
		Data:       make([]interface{}, len(metrics)),
		TotalCount: len(metrics),
	}

	for i, metric := range metrics {
		response.Data[i] = metric
	}

	c.JSON(http.StatusOK, response)
}

// GetRealtimeMetrics returns recent real-time metrics
func (h *Handlers) GetRealtimeMetrics(c *gin.Context) {
	tenantID, err := h.getTenantIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant"})
		return
	}

	metricType := c.Query("type")
	if metricType == "" {
		metricType = "delivery_rate"
	}

	// Get metrics from the last 5 minutes
	since := time.Now().Add(-5 * time.Minute)
	
	metrics, err := h.analyticsRepo.GetRealtimeMetrics(c.Request.Context(), tenantID, metricType, since)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get real-time metrics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"metrics": metrics,
		"type":    metricType,
		"since":   since,
	})
}

// GetEndpointMetrics returns metrics for a specific endpoint
func (h *Handlers) GetEndpointMetrics(c *gin.Context) {
	tenantID, err := h.getTenantIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant"})
		return
	}

	endpointIDStr := c.Param("endpoint_id")
	endpointID, err := uuid.Parse(endpointIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid endpoint ID"})
		return
	}

	query, err := h.parseMetricsQuery(c, tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Filter by specific endpoint
	query.EndpointIDs = []uuid.UUID{endpointID}

	metrics, err := h.analyticsRepo.GetDeliveryMetrics(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get endpoint metrics"})
		return
	}

	summary, err := h.analyticsRepo.GetMetricsSummary(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get endpoint summary"})
		return
	}

	response := &models.MetricsResponse{
		Data:       make([]interface{}, len(metrics)),
		TotalCount: len(metrics),
		Summary:    *summary,
	}

	for i, metric := range metrics {
		response.Data[i] = metric
	}

	c.JSON(http.StatusOK, response)
}

// Helper functions

func (h *Handlers) getTenantIDFromContext(c *gin.Context) (uuid.UUID, error) {
	tenantIDInterface, exists := c.Get("tenant_id")
	if !exists {
		return uuid.Nil, fmt.Errorf("tenant_id not found in context")
	}

	tenantIDStr, ok := tenantIDInterface.(string)
	if !ok {
		return uuid.Nil, fmt.Errorf("tenant_id is not a string")
	}

	return uuid.Parse(tenantIDStr)
}

func (h *Handlers) parseMetricsQuery(c *gin.Context, tenantID uuid.UUID) (*models.MetricsQuery, error) {
	query := &models.MetricsQuery{
		TenantID: tenantID,
	}

	// Parse start date (required)
	startDateStr := c.Query("start_date")
	if startDateStr == "" {
		// Default to 24 hours ago
		query.StartDate = time.Now().Add(-24 * time.Hour)
	} else {
		startDate, err := time.Parse(time.RFC3339, startDateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid start_date format: %w", err)
		}
		query.StartDate = startDate
	}

	// Parse end date (optional, defaults to now)
	endDateStr := c.Query("end_date")
	if endDateStr == "" {
		query.EndDate = time.Now()
	} else {
		endDate, err := time.Parse(time.RFC3339, endDateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid end_date format: %w", err)
		}
		query.EndDate = endDate
	}

	// Parse endpoint IDs (optional)
	endpointIDsStr := c.QueryArray("endpoint_ids")
	if len(endpointIDsStr) > 0 {
		for _, idStr := range endpointIDsStr {
			id, err := uuid.Parse(idStr)
			if err != nil {
				return nil, fmt.Errorf("invalid endpoint_id format: %w", err)
			}
			query.EndpointIDs = append(query.EndpointIDs, id)
		}
	}

	// Parse statuses (optional)
	query.Statuses = c.QueryArray("statuses")

	// Parse group by (optional)
	query.GroupBy = c.DefaultQuery("group_by", "hour")

	// Parse pagination
	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return nil, fmt.Errorf("invalid limit format: %w", err)
	}
	query.Limit = limit

	offsetStr := c.DefaultQuery("offset", "0")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		return nil, fmt.Errorf("invalid offset format: %w", err)
	}
	query.Offset = offset

	return query, nil
}