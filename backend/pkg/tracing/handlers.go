package tracing

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for distributed tracing
type Handler struct {
	service *Service
}

// NewHandler creates a new tracing handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers distributed tracing routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	traces := router.Group("/traces")
	{
		traces.GET("", h.SearchTraces)
		traces.GET("/stats", h.GetTraceStats)
		traces.GET("/:traceID", h.GetTrace)
		traces.GET("/:traceID/waterfall", h.GetSpanWaterfall)
		traces.POST("/:traceID/complete", h.CompleteTrace)

		// Span ingestion
		traces.POST("/spans", h.RecordSpan)

		// Propagation config
		traces.GET("/propagation/config", h.GetPropagationConfig)
		traces.PUT("/propagation/config", h.UpdatePropagationConfig)

		// Context generation
		traces.POST("/context/generate", h.GenerateTraceContext)
	}
}

// @Summary Search traces
// @Tags Tracing
// @Produce json
// @Param service_name query string false "Filter by service"
// @Param status query string false "Filter by status"
// @Param has_errors query bool false "Filter by error presence"
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} map[string]interface{}
// @Router /traces [get]
func (h *Handler) SearchTraces(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var filter TraceSearchRequest
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	traces, total, err := h.service.SearchTraces(c.Request.Context(), tenantID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "SEARCH_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"traces": traces, "total": total})
}

// @Summary Get a trace
// @Tags Tracing
// @Produce json
// @Param traceID path string true "Trace ID"
// @Success 200 {object} Trace
// @Router /traces/{traceID} [get]
func (h *Handler) GetTrace(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	traceID := c.Param("traceID")

	trace, err := h.service.GetTrace(c.Request.Context(), tenantID, traceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, trace)
}

// @Summary Get span waterfall for a trace
// @Tags Tracing
// @Produce json
// @Param traceID path string true "Trace ID"
// @Success 200 {object} SpanWaterfall
// @Router /traces/{traceID}/waterfall [get]
func (h *Handler) GetSpanWaterfall(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	traceID := c.Param("traceID")

	waterfall, err := h.service.GetSpanWaterfall(c.Request.Context(), tenantID, traceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "WATERFALL_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, waterfall)
}

// @Summary Record a span
// @Tags Tracing
// @Accept json
// @Produce json
// @Param body body CreateSpanRequest true "Span data"
// @Success 201 {object} Span
// @Router /traces/spans [post]
func (h *Handler) RecordSpan(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req CreateSpanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	span, err := h.service.RecordSpan(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "RECORD_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusCreated, span)
}

// @Summary Complete a trace
// @Tags Tracing
// @Produce json
// @Param traceID path string true "Trace ID"
// @Success 200 {object} Trace
// @Router /traces/{traceID}/complete [post]
func (h *Handler) CompleteTrace(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	traceID := c.Param("traceID")

	trace, err := h.service.CompleteTrace(c.Request.Context(), tenantID, traceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "COMPLETE_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, trace)
}

// @Summary Get trace statistics
// @Tags Tracing
// @Produce json
// @Param start_time query string false "Start time (RFC3339)"
// @Param end_time query string false "End time (RFC3339)"
// @Success 200 {object} TraceStats
// @Router /traces/stats [get]
func (h *Handler) GetTraceStats(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	startTime := c.DefaultQuery("start_time", "")
	endTime := c.DefaultQuery("end_time", "")

	stats, err := h.service.GetTraceStats(c.Request.Context(), tenantID, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "STATS_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// @Summary Get propagation configuration
// @Tags Tracing
// @Produce json
// @Success 200 {object} PropagationConfig
// @Router /traces/propagation/config [get]
func (h *Handler) GetPropagationConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	config, err := h.service.GetPropagationConfig(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "CONFIG_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, config)
}

// @Summary Update propagation configuration
// @Tags Tracing
// @Accept json
// @Produce json
// @Param body body UpdatePropagationConfigRequest true "Propagation config"
// @Success 200 {object} PropagationConfig
// @Router /traces/propagation/config [put]
func (h *Handler) UpdatePropagationConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req UpdatePropagationConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	config, err := h.service.UpdatePropagationConfig(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "UPDATE_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, config)
}

// @Summary Generate a new trace context
// @Tags Tracing
// @Produce json
// @Success 201 {object} TraceContext
// @Router /traces/context/generate [post]
func (h *Handler) GenerateTraceContext(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	traceCtx, err := h.service.GenerateTraceContext(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "GENERATE_FAILED", "message": err.Error()}})
		return
	}

	if traceCtx == nil {
		c.JSON(http.StatusOK, gin.H{"message": "tracing is disabled for this tenant"})
		return
	}

	traceparent := "00-" + traceCtx.TraceID + "-" + traceCtx.SpanID + "-" + traceCtx.TraceFlags
	c.JSON(http.StatusCreated, gin.H{
		"trace_context": traceCtx,
		"traceparent":   traceparent,
	})
}

// Ensure strconv import is used
var _ = strconv.Itoa
