package endpointmesh

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/josedab/waas/pkg/httputil"
)

// Handler provides HTTP endpoints for the endpoint mesh.
type Handler struct {
	service *Service
}

// NewHandler creates a new endpoint mesh handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers all endpoint mesh routes.
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	g := router.Group("/endpoint-mesh")
	{
		g.POST("/nodes", h.AddNode)
		g.GET("/nodes", h.ListNodes)
		g.GET("/nodes/:id", h.GetNode)
		g.DELETE("/nodes/:id", h.DeleteNode)
		g.POST("/nodes/:id/fallback", h.SetFallback)
		g.POST("/nodes/:id/health-check", h.RecordHealthCheck)
		g.GET("/topology", h.GetTopology)
		g.GET("/reroute-events", h.ListRerouteEvents)
		g.POST("/nodes/:id/recover", h.RecoverNode)
	}
}

func (h *Handler) AddNode(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateMeshNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	node, err := h.service.AddNode(tenantID, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, node)
}

func (h *Handler) GetNode(c *gin.Context) {
	node, err := h.service.GetNode(c.Param("id"))
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, node)
}

func (h *Handler) ListNodes(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	nodes, err := h.service.ListNodes(tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"nodes": nodes})
}

func (h *Handler) DeleteNode(c *gin.Context) {
	if err := h.service.RemoveNode(c.Param("id")); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (h *Handler) SetFallback(c *gin.Context) {
	nodeID := c.Param("id")
	var req SetFallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.SetFallback(nodeID, req.FallbackNodeID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "fallback_set"})
}

func (h *Handler) RecordHealthCheck(c *gin.Context) {
	nodeID := c.Param("id")
	var req struct {
		StatusCode int    `json:"status_code"`
		LatencyMs  int64  `json:"latency_ms"`
		Success    bool   `json:"success"`
		Error      string `json:"error,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hc, err := h.service.RecordHealthCheck(nodeID, req.StatusCode, req.LatencyMs, req.Success, req.Error)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, hc)
}

func (h *Handler) GetTopology(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	topology, err := h.service.GetTopology(tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, topology)
}

func (h *Handler) ListRerouteEvents(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	events, err := h.service.ListRerouteEvents(tenantID, limit)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"events": events})
}

func (h *Handler) RecoverNode(c *gin.Context) {
	node, err := h.service.RecoverNode(c.Param("id"))
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, node)
}
