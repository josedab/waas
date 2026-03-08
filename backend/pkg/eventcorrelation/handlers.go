package eventcorrelation

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP endpoints for event correlation.
type Handler struct {
	service *Service
}

// NewHandler creates a new event correlation handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers event correlation routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/event-correlation")
	{
		g.POST("/rules", h.CreateRule)
		g.GET("/rules", h.ListRules)
		g.GET("/rules/:id", h.GetRule)
		g.DELETE("/rules/:id", h.DeleteRule)

		g.POST("/ingest", h.IngestEvent)
		g.GET("/matches", h.ListMatches)
		g.GET("/stats", h.GetStats)
	}
}

func (h *Handler) CreateRule(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule, err := h.service.CreateRule(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, rule)
}

func (h *Handler) GetRule(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	ruleID := c.Param("id")

	rule, err := h.service.GetRule(c.Request.Context(), tenantID, ruleID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, rule)
}

func (h *Handler) ListRules(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	rules, err := h.service.ListRules(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

func (h *Handler) DeleteRule(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	ruleID := c.Param("id")

	if err := h.service.DeleteRule(c.Request.Context(), tenantID, ruleID); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (h *Handler) IngestEvent(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req IngestEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	composites, err := h.service.IngestEvent(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"composite_events": composites, "matched": len(composites) > 0})
}

func (h *Handler) ListMatches(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	matches, err := h.service.ListMatches(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"matches": matches})
}

func (h *Handler) GetStats(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	stats, err := h.service.GetStats(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}
