package topologysim

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP endpoints for topology simulation.
type Handler struct {
	service *Service
}

// NewHandler creates a new topology simulation handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers all topology simulation routes.
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	group := router.Group("/topology-sim")
	{
		group.POST("/topologies", h.CreateTopology)
		group.GET("/topologies", h.ListTopologies)
		group.GET("/topologies/:id", h.GetTopology)
		group.DELETE("/topologies/:id", h.DeleteTopology)
		group.POST("/simulate", h.RunSimulation)
		group.GET("/results/:id", h.GetResult)
		group.GET("/results/topology/:topology_id", h.ListResults)
	}
}

// CreateTopology creates a new topology.
// @Summary Create topology
// @Tags topology-sim
// @Accept json
// @Produce json
// @Success 201 {object} Topology
// @Router /topology-sim/topologies [post]
func (h *Handler) CreateTopology(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	var req CreateTopologyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	topology, err := h.service.CreateTopology(tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, topology)
}

// ListTopologies lists all topologies.
// @Summary List topologies
// @Tags topology-sim
// @Produce json
// @Success 200 {array} Topology
// @Router /topology-sim/topologies [get]
func (h *Handler) ListTopologies(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	topologies, err := h.service.ListTopologies(tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, topologies)
}

// GetTopology retrieves a topology.
// @Summary Get topology
// @Tags topology-sim
// @Param id path string true "Topology ID"
// @Produce json
// @Success 200 {object} Topology
// @Router /topology-sim/topologies/{id} [get]
func (h *Handler) GetTopology(c *gin.Context) {
	topology, err := h.service.GetTopology(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, topology)
}

// DeleteTopology deletes a topology.
// @Summary Delete topology
// @Tags topology-sim
// @Param id path string true "Topology ID"
// @Success 204
// @Router /topology-sim/topologies/{id} [delete]
func (h *Handler) DeleteTopology(c *gin.Context) {
	if err := h.service.repo.DeleteTopology(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// RunSimulation starts a simulation.
// @Summary Run simulation
// @Tags topology-sim
// @Accept json
// @Produce json
// @Success 200 {object} SimulationResult
// @Router /topology-sim/simulate [post]
func (h *Handler) RunSimulation(c *gin.Context) {
	var cfg SimulationConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.RunSimulation(&cfg)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// GetResult retrieves a simulation result.
// @Summary Get simulation result
// @Tags topology-sim
// @Param id path string true "Result ID"
// @Produce json
// @Success 200 {object} SimulationResult
// @Router /topology-sim/results/{id} [get]
func (h *Handler) GetResult(c *gin.Context) {
	result, err := h.service.GetResult(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// ListResults lists all results for a topology.
// @Summary List simulation results
// @Tags topology-sim
// @Param topology_id path string true "Topology ID"
// @Produce json
// @Success 200 {array} SimulationResult
// @Router /topology-sim/results/topology/{topology_id} [get]
func (h *Handler) ListResults(c *gin.Context) {
	results, err := h.service.ListResults(c.Param("topology_id"))
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, results)
}
