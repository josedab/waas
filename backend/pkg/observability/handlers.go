package observability

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler handles observability HTTP endpoints
type Handler struct {
	service *Service
}

// NewHandler creates a new observability handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers observability routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	traces := r.Group("/traces")
	{
		traces.GET("", h.SearchTraces)
		traces.GET("/:traceId", h.GetTrace)
		traces.GET("/:traceId/timeline", h.GetTraceTimeline)
		traces.GET("/:traceId/latency", h.GetLatencyBreakdown)
	}

	r.GET("/metrics", h.GetTraceMetrics)
	r.GET("/service-map", h.GetServiceMap)

	exports := r.Group("/exports")
	{
		exports.GET("", h.ListExportConfigs)
		exports.POST("", h.CreateExportConfig)
		exports.DELETE("/:configId", h.DeleteExportConfig)
	}
}

// SearchTraces godoc
// @Summary Search traces
// @Description Search for traces with filtering options
// @Tags observability
// @Accept json
// @Produce json
// @Param webhook_id query string false "Filter by webhook ID"
// @Param endpoint_id query string false "Filter by endpoint ID"
// @Param service query string false "Filter by service name"
// @Param min_duration query int false "Minimum duration in ms"
// @Param max_duration query int false "Maximum duration in ms"
// @Param start query string false "Start time (RFC3339)"
// @Param end query string false "End time (RFC3339)"
// @Param limit query int false "Limit results" default(20)
// @Param offset query int false "Offset for pagination"
// @Success 200 {object} TraceSearchResult
// @Router /observability/traces [get]
func (h *Handler) SearchTraces(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	
	query := &TraceSearchQuery{
		TenantID:      tenantID,
		WebhookID:     c.Query("webhook_id"),
		EndpointID:    c.Query("endpoint_id"),
		ServiceName:   c.Query("service"),
		OperationName: c.Query("operation"),
		Limit:         20,
	}

	if minDur := c.Query("min_duration"); minDur != "" {
		var dur int64
		if _, err := parseIntQuery(minDur, &dur); err == nil {
			query.MinDuration = dur
		}
	}

	if maxDur := c.Query("max_duration"); maxDur != "" {
		var dur int64
		if _, err := parseIntQuery(maxDur, &dur); err == nil {
			query.MaxDuration = dur
		}
	}

	if start := c.Query("start"); start != "" {
		if t, err := time.Parse(time.RFC3339, start); err == nil {
			query.StartTime = t
		}
	}
	if query.StartTime.IsZero() {
		query.StartTime = time.Now().Add(-24 * time.Hour)
	}

	if end := c.Query("end"); end != "" {
		if t, err := time.Parse(time.RFC3339, end); err == nil {
			query.EndTime = t
		}
	}
	if query.EndTime.IsZero() {
		query.EndTime = time.Now()
	}

	if limit := c.Query("limit"); limit != "" {
		var l int
		if _, err := parseIntQuery(limit, &l); err == nil {
			query.Limit = l
		}
	}

	if offset := c.Query("offset"); offset != "" {
		var o int
		if _, err := parseIntQuery(offset, &o); err == nil {
			query.Offset = o
		}
	}

	result, err := h.service.SearchTraces(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetTrace godoc
// @Summary Get trace details
// @Description Get a complete trace with all spans
// @Tags observability
// @Accept json
// @Produce json
// @Param traceId path string true "Trace ID"
// @Success 200 {object} Trace
// @Router /observability/traces/{traceId} [get]
func (h *Handler) GetTrace(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	traceID := c.Param("traceId")

	trace, err := h.service.GetTrace(c.Request.Context(), tenantID, traceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Trace not found"})
		return
	}

	c.JSON(http.StatusOK, trace)
}

// GetTraceTimeline godoc
// @Summary Get trace timeline
// @Description Get a timeline visualization of a trace
// @Tags observability
// @Accept json
// @Produce json
// @Param traceId path string true "Trace ID"
// @Success 200 {object} TraceTimeline
// @Router /observability/traces/{traceId}/timeline [get]
func (h *Handler) GetTraceTimeline(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	traceID := c.Param("traceId")

	timeline, err := h.service.GetTraceTimeline(c.Request.Context(), tenantID, traceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Trace not found"})
		return
	}

	c.JSON(http.StatusOK, timeline)
}

// GetLatencyBreakdown godoc
// @Summary Get latency breakdown
// @Description Get latency analysis for a trace
// @Tags observability
// @Accept json
// @Produce json
// @Param traceId path string true "Trace ID"
// @Success 200 {object} LatencyBreakdown
// @Router /observability/traces/{traceId}/latency [get]
func (h *Handler) GetLatencyBreakdown(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	traceID := c.Param("traceId")

	breakdown, err := h.service.GetLatencyBreakdown(c.Request.Context(), tenantID, traceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Trace not found"})
		return
	}

	c.JSON(http.StatusOK, breakdown)
}

// GetTraceMetrics godoc
// @Summary Get trace metrics
// @Description Get aggregated trace metrics
// @Tags observability
// @Accept json
// @Produce json
// @Param start query string false "Start time (RFC3339)"
// @Param end query string false "End time (RFC3339)"
// @Success 200 {object} TraceMetrics
// @Router /observability/metrics [get]
func (h *Handler) GetTraceMetrics(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()

	if s := c.Query("start"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			start = t
		}
	}
	if e := c.Query("end"); e != "" {
		if t, err := time.Parse(time.RFC3339, e); err == nil {
			end = t
		}
	}

	metrics, err := h.service.GetTraceMetrics(c.Request.Context(), tenantID, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// GetServiceMap godoc
// @Summary Get service map
// @Description Get service dependency graph
// @Tags observability
// @Accept json
// @Produce json
// @Param start query string false "Start time (RFC3339)"
// @Param end query string false "End time (RFC3339)"
// @Success 200 {object} ServiceMap
// @Router /observability/service-map [get]
func (h *Handler) GetServiceMap(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()

	if s := c.Query("start"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			start = t
		}
	}
	if e := c.Query("end"); e != "" {
		if t, err := time.Parse(time.RFC3339, e); err == nil {
			end = t
		}
	}

	serviceMap, err := h.service.GetServiceMap(c.Request.Context(), tenantID, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, serviceMap)
}

// ListExportConfigs godoc
// @Summary List export configurations
// @Description List all OTel export configurations
// @Tags observability
// @Accept json
// @Produce json
// @Success 200 {array} OTelExportConfig
// @Router /observability/exports [get]
func (h *Handler) ListExportConfigs(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	configs, err := h.service.ListExportConfigs(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, configs)
}

// CreateExportConfig godoc
// @Summary Create export configuration
// @Description Create a new OTel export configuration
// @Tags observability
// @Accept json
// @Produce json
// @Param request body CreateExportConfigRequest true "Export config request"
// @Success 201 {object} OTelExportConfig
// @Router /observability/exports [post]
func (h *Handler) CreateExportConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req CreateExportConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := h.service.CreateExportConfig(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, config)
}

// DeleteExportConfig godoc
// @Summary Delete export configuration
// @Description Delete an OTel export configuration
// @Tags observability
// @Accept json
// @Produce json
// @Param configId path string true "Config ID"
// @Success 204
// @Router /observability/exports/{configId} [delete]
func (h *Handler) DeleteExportConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	configID := c.Param("configId")

	if err := h.service.DeleteExportConfig(c.Request.Context(), tenantID, configID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func parseIntQuery[T int | int64](s string, out *T) (T, error) {
	var val int64
	_, err := time.ParseDuration(s + "ms")
	if err != nil {
		for _, c := range s {
			if c < '0' || c > '9' {
				return 0, err
			}
			val = val*10 + int64(c-'0')
		}
	}
	*out = T(val)
	return *out, nil
}
