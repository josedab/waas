package protocolgw

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for protocol gateway management
type Handler struct {
	service *Service
}

// NewHandler creates a new protocol gateway handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers protocol gateway routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	gw := router.Group("/protocols/gateway")
	{
		gw.POST("/routes", h.CreateRoute)
		gw.GET("/routes", h.ListRoutes)
		gw.GET("/routes/:id", h.GetRoute)
		gw.PUT("/routes/:id", h.UpdateRoute)
		gw.DELETE("/routes/:id", h.DeleteRoute)
		gw.POST("/translate", h.TranslateMessage)
		gw.GET("/routes/:id/stats", h.GetRouteStats)
		gw.GET("/stats", h.GetProtocolStats)
	}
}

// @Summary Create a protocol route
// @Tags ProtocolGateway
// @Accept json
// @Produce json
// @Param body body CreateRouteRequest true "Protocol route configuration"
// @Success 201 {object} ProtocolRoute
// @Router /protocols/gateway/routes [post]
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

// @Summary List protocol routes
// @Tags ProtocolGateway
// @Produce json
// @Success 200 {object} map[string][]ProtocolRoute
// @Router /protocols/gateway/routes [get]
func (h *Handler) ListRoutes(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	routes, err := h.service.ListRoutes(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "LIST_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"routes": routes})
}

// @Summary Get a protocol route
// @Tags ProtocolGateway
// @Produce json
// @Param id path string true "Route ID"
// @Success 200 {object} ProtocolRoute
// @Router /protocols/gateway/routes/{id} [get]
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

// @Summary Update a protocol route
// @Tags ProtocolGateway
// @Accept json
// @Produce json
// @Param id path string true "Route ID"
// @Param body body CreateRouteRequest true "Updated route configuration"
// @Success 200 {object} ProtocolRoute
// @Router /protocols/gateway/routes/{id} [put]
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

// @Summary Delete a protocol route
// @Tags ProtocolGateway
// @Param id path string true "Route ID"
// @Success 204
// @Router /protocols/gateway/routes/{id} [delete]
func (h *Handler) DeleteRoute(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	routeID := c.Param("id")

	if err := h.service.DeleteRoute(c.Request.Context(), tenantID, routeID); err != nil {
		httputil.InternalError(c, "DELETE_FAILED", err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary Translate a message between protocols
// @Tags ProtocolGateway
// @Accept json
// @Produce json
// @Param body body TranslateMessageRequest true "Message to translate"
// @Success 200 {object} TranslationResult
// @Router /protocols/gateway/translate [post]
func (h *Handler) TranslateMessage(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req TranslateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	result, err := h.service.TranslateMessage(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalError(c, "TRANSLATION_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// @Summary Get statistics for a protocol route
// @Tags ProtocolGateway
// @Produce json
// @Param id path string true "Route ID"
// @Success 200 {object} ProtocolStats
// @Router /protocols/gateway/routes/{id}/stats [get]
func (h *Handler) GetRouteStats(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	routeID := c.Param("id")

	stats, err := h.service.GetRouteStats(c.Request.Context(), tenantID, routeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// @Summary Get aggregate protocol statistics
// @Tags ProtocolGateway
// @Produce json
// @Success 200 {object} map[string][]ProtocolStats
// @Router /protocols/gateway/stats [get]
func (h *Handler) GetProtocolStats(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	stats, err := h.service.GetProtocolStats(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "STATS_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}
