package depgraph

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP endpoints for the dependency graph.
type Handler struct {
	service *Service
}

// NewHandler creates a new dependency graph handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers all dependency graph routes.
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	group := router.Group("/dep-graph")
	{
		group.GET("/graph", h.GetGraph)
		group.GET("/impact/:endpoint_id", h.AnalyzeImpact)
		group.POST("/dependencies", h.AddDependency)
		group.POST("/record-delivery", h.RecordDelivery)
	}
}

// GetGraph returns the dependency graph for the tenant.
// @Summary Get dependency graph
// @Tags dep-graph
// @Produce json
// @Success 200 {object} Graph
// @Router /dep-graph/graph [get]
func (h *Handler) GetGraph(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	graph, err := h.service.GetGraph(tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, graph)
}

// AnalyzeImpact computes the blast radius for an endpoint.
// @Summary Analyze impact / blast radius
// @Tags dep-graph
// @Produce json
// @Param endpoint_id path string true "Endpoint ID"
// @Success 200 {object} ImpactAnalysis
// @Router /dep-graph/impact/{endpoint_id} [get]
func (h *Handler) AnalyzeImpact(c *gin.Context) {
	analysis, err := h.service.AnalyzeImpact(c.Param("endpoint_id"))
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, analysis)
}

// AddDependency manually adds a dependency edge.
// @Summary Add dependency
// @Tags dep-graph
// @Accept json
// @Produce json
// @Success 201 {object} Dependency
// @Router /dep-graph/dependencies [post]
func (h *Handler) AddDependency(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	var req struct {
		ProducerID string   `json:"producer_id" binding:"required"`
		ConsumerID string   `json:"consumer_id" binding:"required"`
		EventTypes []string `json:"event_types"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dep, err := h.service.AddDependency(tenantID, req.ProducerID, req.ConsumerID, req.EventTypes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, dep)
}

// RecordDelivery ingests a delivery event to update the graph.
// @Summary Record delivery for graph update
// @Tags dep-graph
// @Accept json
// @Produce json
// @Success 200
// @Router /dep-graph/record-delivery [post]
func (h *Handler) RecordDelivery(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	var req struct {
		ProducerID string  `json:"producer_id" binding:"required"`
		ConsumerID string  `json:"consumer_id" binding:"required"`
		EventType  string  `json:"event_type"`
		Success    bool    `json:"success"`
		LatencyMs  float64 `json:"latency_ms"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.RecordDelivery(tenantID, req.ProducerID, req.ConsumerID, req.EventType, req.Success, req.LatencyMs); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "recorded"})
}
