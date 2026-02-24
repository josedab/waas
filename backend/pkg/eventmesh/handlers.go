package eventmesh

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for event mesh routing
type Handler struct {
	service *Service
}

// NewHandler creates a new event mesh handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers event mesh routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	mesh := router.Group("/eventmesh")
	{
		// Routes
		mesh.POST("/routes", h.CreateRoute)
		mesh.GET("/routes", h.ListRoutes)
		mesh.GET("/routes/:id", h.GetRoute)
		mesh.PUT("/routes/:id", h.UpdateRoute)
		mesh.DELETE("/routes/:id", h.DeleteRoute)

		// Routing
		mesh.POST("/route", h.RouteEvent)
		mesh.GET("/stats", h.GetRouteStats)

		// Executions
		mesh.GET("/routes/:id/executions", h.ListExecutions)

		// Dead letter
		mesh.PUT("/routes/:id/dead-letter", h.ConfigureDeadLetter)
		mesh.GET("/routes/:id/dead-letter", h.ListDeadLetterEntries)
		mesh.POST("/dead-letter/:entry_id/redrive", h.RedriveDeadLetter)
	}
}

// @Summary Create a routing rule
// @Tags EventMesh
// @Accept json
// @Produce json
// @Param body body CreateRouteRequest true "Routing rule"
// @Success 201 {object} Route
// @Router /eventmesh/routes [post]
func (h *Handler) CreateRoute(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	route, err := h.service.CreateRoute(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalError(c, "CREATE_FAILED", err)
		return
	}

	c.JSON(http.StatusCreated, route)
}

// @Summary List routing rules
// @Tags EventMesh
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /eventmesh/routes [get]
func (h *Handler) ListRoutes(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	routes, total, err := h.service.ListRoutes(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		httputil.InternalError(c, "LIST_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"routes": routes, "total": total})
}

// @Summary Get a routing rule
// @Tags EventMesh
// @Produce json
// @Param id path string true "Route ID"
// @Success 200 {object} Route
// @Router /eventmesh/routes/{id} [get]
func (h *Handler) GetRoute(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	routeID := c.Param("id")

	route, err := h.service.GetRoute(c.Request.Context(), tenantID, routeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, route)
}

// @Summary Update a routing rule
// @Tags EventMesh
// @Accept json
// @Produce json
// @Param id path string true "Route ID"
// @Param body body CreateRouteRequest true "Updated rule"
// @Success 200 {object} Route
// @Router /eventmesh/routes/{id} [put]
func (h *Handler) UpdateRoute(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	routeID := c.Param("id")

	var req CreateRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	route, err := h.service.UpdateRoute(c.Request.Context(), tenantID, routeID, &req)
	if err != nil {
		httputil.InternalError(c, "UPDATE_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, route)
}

// @Summary Delete a routing rule
// @Tags EventMesh
// @Param id path string true "Route ID"
// @Success 204
// @Router /eventmesh/routes/{id} [delete]
func (h *Handler) DeleteRoute(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	routeID := c.Param("id")

	if err := h.service.DeleteRoute(c.Request.Context(), tenantID, routeID); err != nil {
		httputil.InternalError(c, "DELETE_FAILED", err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary Route an event through the mesh
// @Tags EventMesh
// @Accept json
// @Produce json
// @Param body body RouteEventRequest true "Event to route"
// @Success 200 {object} RouteExecution
// @Router /eventmesh/route [post]
func (h *Handler) RouteEvent(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req RouteEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	exec, err := h.service.RouteEvent(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalError(c, "ROUTE_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, exec)
}

// @Summary Get routing statistics
// @Tags EventMesh
// @Produce json
// @Success 200 {object} RouteStats
// @Router /eventmesh/stats [get]
func (h *Handler) GetRouteStats(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	stats, err := h.service.GetRouteStats(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "STATS_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// @Summary List route executions
// @Tags EventMesh
// @Produce json
// @Param id path string true "Route ID"
// @Success 200 {object} map[string][]RouteExecution
// @Router /eventmesh/routes/{id}/executions [get]
func (h *Handler) ListExecutions(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	routeID := c.Param("id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	execs, err := h.service.ListExecutions(c.Request.Context(), tenantID, routeID, limit, offset)
	if err != nil {
		httputil.InternalError(c, "LIST_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"executions": execs})
}

// @Summary Configure dead letter queue
// @Tags EventMesh
// @Accept json
// @Produce json
// @Param id path string true "Route ID"
// @Param body body ConfigureDeadLetterRequest true "Dead letter config"
// @Success 200 {object} DeadLetterConfig
// @Router /eventmesh/routes/{id}/dead-letter [put]
func (h *Handler) ConfigureDeadLetter(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	routeID := c.Param("id")

	var req ConfigureDeadLetterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	config, err := h.service.ConfigureDeadLetter(c.Request.Context(), tenantID, routeID, &req)
	if err != nil {
		httputil.InternalError(c, "CONFIG_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, config)
}

// @Summary List dead letter entries
// @Tags EventMesh
// @Produce json
// @Param id path string true "Route ID"
// @Success 200 {object} map[string]interface{}
// @Router /eventmesh/routes/{id}/dead-letter [get]
func (h *Handler) ListDeadLetterEntries(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	routeID := c.Param("id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	entries, total, err := h.service.ListDeadLetterEntries(c.Request.Context(), tenantID, routeID, limit, offset)
	if err != nil {
		httputil.InternalError(c, "LIST_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"entries": entries, "total": total})
}

// @Summary Redrive a dead letter entry
// @Tags EventMesh
// @Param entry_id path string true "Entry ID"
// @Success 204
// @Router /eventmesh/dead-letter/{entry_id}/redrive [post]
func (h *Handler) RedriveDeadLetter(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	entryID := c.Param("entry_id")

	if err := h.service.RedriveDeadLetter(c.Request.Context(), tenantID, entryID); err != nil {
		httputil.InternalError(c, "REDRIVE_FAILED", err)
		return
	}

	c.Status(http.StatusNoContent)
}
