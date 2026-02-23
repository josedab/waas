package eventlineage

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP endpoints for event lineage.
type Handler struct {
	service *Service
}

// NewHandler creates a new event lineage handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers event lineage routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/event-lineage")
	{
		g.POST("/record", h.RecordLineage)
		g.GET("/graph/:event_id", h.GetLineageGraph)
		g.GET("/entries", h.ListEntries)
		g.GET("/stats", h.GetStats)
	}
}

func (h *Handler) RecordLineage(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req RecordLineageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entry, err := h.service.RecordLineage(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, entry)
}

func (h *Handler) GetLineageGraph(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	eventID := c.Param("event_id")

	graph, err := h.service.GetLineageGraph(c.Request.Context(), tenantID, eventID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, graph)
}

func (h *Handler) ListEntries(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	entries, err := h.service.ListEntries(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"entries": entries})
}

func (h *Handler) GetStats(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	stats, err := h.service.GetStats(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}
