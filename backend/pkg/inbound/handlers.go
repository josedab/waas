package inbound

import (
	"github.com/josedab/waas/pkg/httputil"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler handles inbound webhook HTTP endpoints
type Handler struct {
	service *Service
}

// NewHandler creates a new inbound handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers protected inbound management routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	inbound := r.Group("/inbound")
	{
		inbound.POST("/sources", h.CreateSource)
		inbound.GET("/sources", h.ListSources)
		inbound.GET("/sources/:id", h.GetSource)
		inbound.PUT("/sources/:id", h.UpdateSource)
		inbound.DELETE("/sources/:id", h.DeleteSource)
		inbound.GET("/sources/:id/events", h.GetSourceEvents)
		inbound.POST("/sources/:id/events/:eventId/replay", h.ReplayEvent)
		// DLQ
		inbound.GET("/sources/:id/dlq", h.GetDLQ)
		inbound.POST("/sources/:id/dlq/:entryId/replay", h.ReplayDLQEntry)
		// Provider health
		inbound.GET("/sources/:id/health", h.GetProviderHealth)
		// Rate limiting
		inbound.GET("/sources/:id/rate-limit", h.GetRateLimitConfig)
		// Stats
		inbound.GET("/sources/:id/stats", h.GetInboundStats)
		// Content routing
		inbound.POST("/sources/:id/routes", h.CreateContentRoute)
		// Transform rules
		inbound.POST("/sources/:id/transforms", h.CreateTransformRule)
	}
}

// RegisterPublicRoutes registers the public webhook receiver endpoint (no auth)
func (h *Handler) RegisterPublicRoutes(r *gin.RouterGroup) {
	r.POST("/inbound/receive/:source_id", h.ReceiveWebhook)
}

// CreateSource godoc
// @Summary Create inbound source
// @Description Register a new inbound webhook source
// @Tags inbound
// @Accept json
// @Produce json
// @Param request body CreateSourceRequest true "Inbound source request"
// @Success 201 {object} InboundSource
// @Failure 400 {object} map[string]interface{}
// @Router /inbound/sources [post]
func (h *Handler) CreateSource(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req CreateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	source, err := h.service.CreateSource(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalError(c, "CREATE_FAILED", err)
		return
	}

	c.JSON(http.StatusCreated, source)
}

// ListSources godoc
// @Summary List inbound sources
// @Description List inbound webhook sources for the tenant
// @Tags inbound
// @Produce json
// @Param limit query int false "Limit results"
// @Param offset query int false "Offset for pagination"
// @Success 200 {object} map[string]interface{}
// @Router /inbound/sources [get]
func (h *Handler) ListSources(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}

	sources, total, err := h.service.ListSources(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		httputil.InternalError(c, "LIST_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sources": sources,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// GetSource godoc
// @Summary Get inbound source
// @Description Get an inbound source by ID
// @Tags inbound
// @Produce json
// @Param id path string true "Source ID"
// @Success 200 {object} InboundSource
// @Failure 404 {object} map[string]interface{}
// @Router /inbound/sources/{id} [get]
func (h *Handler) GetSource(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	sourceID := c.Param("id")

	source, err := h.service.GetSource(c.Request.Context(), tenantID, sourceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, source)
}

// UpdateSource godoc
// @Summary Update inbound source
// @Description Update an inbound source configuration
// @Tags inbound
// @Accept json
// @Produce json
// @Param id path string true "Source ID"
// @Param request body UpdateSourceRequest true "Update source request"
// @Success 200 {object} InboundSource
// @Failure 400 {object} map[string]interface{}
// @Router /inbound/sources/{id} [put]
func (h *Handler) UpdateSource(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	sourceID := c.Param("id")

	var req UpdateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	source, err := h.service.UpdateSource(c.Request.Context(), tenantID, sourceID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "UPDATE_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, source)
}

// DeleteSource godoc
// @Summary Delete inbound source
// @Description Delete an inbound source
// @Tags inbound
// @Produce json
// @Param id path string true "Source ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /inbound/sources/{id} [delete]
func (h *Handler) DeleteSource(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	sourceID := c.Param("id")

	if err := h.service.DeleteSource(c.Request.Context(), tenantID, sourceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "DELETE_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "source deleted"})
}

// GetSourceEvents godoc
// @Summary Get source events
// @Description Get event history for an inbound source
// @Tags inbound
// @Produce json
// @Param id path string true "Source ID"
// @Param status query string false "Filter by status"
// @Param limit query int false "Limit results"
// @Param offset query int false "Offset for pagination"
// @Success 200 {object} map[string]interface{}
// @Router /inbound/sources/{id}/events [get]
func (h *Handler) GetSourceEvents(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	sourceID := c.Param("id")
	status := c.Query("status")

	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}

	events, total, err := h.service.GetSourceEvents(c.Request.Context(), tenantID, sourceID, status, limit, offset)
	if err != nil {
		httputil.InternalError(c, "LIST_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// ReplayEvent godoc
// @Summary Replay inbound event
// @Description Re-process an existing inbound event
// @Tags inbound
// @Produce json
// @Param id path string true "Source ID"
// @Param eventId path string true "Event ID"
// @Success 200 {object} InboundEvent
// @Failure 400 {object} map[string]interface{}
// @Router /inbound/sources/{id}/events/{eventId}/replay [post]
func (h *Handler) ReplayEvent(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	eventID := c.Param("eventId")

	event, err := h.service.ReplayInboundEvent(c.Request.Context(), tenantID, eventID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "REPLAY_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, event)
}

// ReceiveWebhook godoc
// @Summary Receive inbound webhook
// @Description Public endpoint for receiving webhooks from external providers
// @Tags inbound
// @Accept json
// @Produce json
// @Param source_id path string true "Source ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /inbound/receive/{source_id} [post]
func (h *Handler) ReceiveWebhook(c *gin.Context) {
	sourceID := c.Param("source_id")

	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "READ_FAILED", "message": "failed to read request body"}})
		return
	}

	headers := make(map[string][]string)
	for k, v := range c.Request.Header {
		headers[k] = v
	}

	event, err := h.service.ProcessInboundWebhook(c.Request.Context(), sourceID, payload, headers)
	if err != nil {
		status := http.StatusBadRequest
		if event == nil {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": gin.H{"code": "WEBHOOK_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"event_id":        event.ID,
		"status":          event.Status,
		"signature_valid": event.SignatureValid,
	})
}

// GetDLQ returns DLQ entries for a source
func (h *Handler) GetDLQ(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}

	entries, err := h.service.GetDLQEntries(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		httputil.InternalError(c, "DLQ_FAILED", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"entries": entries, "total": len(entries)})
}

// ReplayDLQEntry replays a failed event from the DLQ
func (h *Handler) ReplayDLQEntry(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	entryID := c.Param("entryId")

	event, err := h.service.ReplayDLQEntry(c.Request.Context(), tenantID, entryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, event)
}

// GetProviderHealth returns health status for a source
func (h *Handler) GetProviderHealth(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	sourceID := c.Param("id")

	health, err := h.service.GetProviderHealth(c.Request.Context(), tenantID, sourceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, health)
}

// GetRateLimitConfig returns rate limit configuration
func (h *Handler) GetRateLimitConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	sourceID := c.Param("id")

	config, err := h.service.GetRateLimitConfig(c.Request.Context(), tenantID, sourceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, config)
}

// GetInboundStats returns statistics for a source
func (h *Handler) GetInboundStats(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	sourceID := c.Param("id")

	stats, err := h.service.GetInboundStats(c.Request.Context(), tenantID, sourceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// CreateContentRoute creates a content-based routing rule
func (h *Handler) CreateContentRoute(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	sourceID := c.Param("id")

	var req CreateContentRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	route, err := h.service.CreateContentRoute(c.Request.Context(), tenantID, sourceID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "CREATE_FAILED", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusCreated, route)
}

// CreateTransformRule creates a payload transformation rule
func (h *Handler) CreateTransformRule(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	sourceID := c.Param("id")

	var req CreateTransformRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	rule, err := h.service.CreateTransformRule(c.Request.Context(), tenantID, sourceID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "CREATE_FAILED", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusCreated, rule)
}
