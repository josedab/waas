package edgenetwork

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP endpoints for the edge delivery network.
type Handler struct {
	service *Service
}

// NewHandler creates a new edge network handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers edge network routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/edge-network")
	{
		g.POST("/nodes", h.RegisterNode)
		g.GET("/nodes", h.ListNodes)
		g.GET("/nodes/:id", h.GetNode)
		g.DELETE("/nodes/:id", h.RemoveNode)
		g.POST("/resolve", h.ResolveRoute)
		g.GET("/topology", h.GetTopology)
		g.GET("/metrics", h.GetNetworkMetrics)
		g.POST("/routing-rules", h.CreateRoutingRule)
		g.GET("/routing-rules", h.ListRoutingRules)
		g.DELETE("/routing-rules/:id", h.DeleteRoutingRule)
	}
}

func (h *Handler) RegisterNode(c *gin.Context) {
	var req CreateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	node, err := h.service.RegisterNode(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, node)
}

func (h *Handler) ListNodes(c *gin.Context) {
	var region *Region
	if r := c.Query("region"); r != "" {
		reg := Region(r)
		region = &reg
	}
	nodes, err := h.service.ListNodes(c.Request.Context(), region)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"nodes": nodes})
}

func (h *Handler) GetNode(c *gin.Context) {
	node, err := h.service.GetNode(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, node)
}

func (h *Handler) RemoveNode(c *gin.Context) {
	if err := h.service.RemoveNode(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ResolveRoute(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req struct {
		TargetURL    string `json:"target_url" binding:"required"`
		SourceRegion Region `json:"source_region"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	route, err := h.service.ResolveRoute(c.Request.Context(), tenantID, req.TargetURL, req.SourceRegion)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, route)
}

func (h *Handler) GetTopology(c *gin.Context) {
	topology, err := h.service.GetTopology(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, topology)
}

func (h *Handler) GetNetworkMetrics(c *gin.Context) {
	metrics, err := h.service.GetMetrics(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, metrics)
}

func (h *Handler) CreateRoutingRule(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateRoutingRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rule, err := h.service.CreateRoutingRule(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, rule)
}

func (h *Handler) ListRoutingRules(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	rules, err := h.service.ListRoutingRules(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

func (h *Handler) DeleteRoutingRule(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if err := h.service.DeleteRoutingRule(c.Request.Context(), tenantID, c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
