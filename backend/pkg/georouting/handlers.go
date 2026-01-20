package georouting

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for geo-routing
type Handler struct {
	service *Service
}

// NewHandler creates a new geo-routing handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers geo-routing routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	geo := router.Group("/geo")
	{
		geo.GET("/regions", h.GetRegions)
		geo.GET("/regions/health", h.GetRegionHealth)
	}

	routing := router.Group("/endpoints")
	{
		routing.POST("/:id/routing", h.CreateRouting)
		routing.GET("/:id/routing", h.GetRouting)
		routing.PUT("/:id/routing", h.UpdateRouting)
		routing.DELETE("/:id/routing", h.DeleteRouting)
		routing.POST("/:id/route", h.RouteDelivery)
	}

	stats := router.Group("/routing")
	{
		stats.GET("/stats", h.GetStats)
	}
}

// GetRegions godoc
//
//	@Summary		Get available regions
//	@Description	Get all available geographic regions with their info
//	@Tags			geo-routing
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/geo/regions [get]
func (h *Handler) GetRegions(c *gin.Context) {
	regions := h.service.GetRegions()
	c.JSON(http.StatusOK, gin.H{"regions": regions})
}

// GetRegionHealth godoc
//
//	@Summary		Get region health
//	@Description	Get health status for all regions
//	@Tags			geo-routing
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/geo/regions/health [get]
func (h *Handler) GetRegionHealth(c *gin.Context) {
	health, err := h.service.GetRegionHealth(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"health": health})
}

// CreateRouting godoc
//
//	@Summary		Create endpoint routing
//	@Description	Configure geographic routing for an endpoint
//	@Tags			geo-routing
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Endpoint ID"
//	@Param			request	body		CreateRoutingRequest	true	"Routing configuration"
//	@Success		201		{object}	EndpointRouting
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/endpoints/{id}/routing [post]
func (h *Handler) CreateRouting(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	endpointID := c.Param("id")
	var req CreateRoutingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	req.EndpointID = endpointID
	routing, err := h.service.CreateEndpointRouting(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, routing)
}

// GetRouting godoc
//
//	@Summary		Get endpoint routing
//	@Description	Get routing configuration for an endpoint
//	@Tags			geo-routing
//	@Produce		json
//	@Param			id	path		string	true	"Endpoint ID"
//	@Success		200	{object}	EndpointRouting
//	@Failure		401	{object}	map[string]interface{}
//	@Failure		404	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/endpoints/{id}/routing [get]
func (h *Handler) GetRouting(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	endpointID := c.Param("id")
	routing, err := h.service.GetEndpointRouting(c.Request.Context(), tenantID, endpointID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if routing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "routing configuration not found"})
		return
	}

	c.JSON(http.StatusOK, routing)
}

// UpdateRouting godoc
//
//	@Summary		Update endpoint routing
//	@Description	Update routing configuration for an endpoint
//	@Tags			geo-routing
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Endpoint ID"
//	@Param			request	body		UpdateRoutingRequest	true	"Routing update"
//	@Success		200		{object}	EndpointRouting
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/endpoints/{id}/routing [put]
func (h *Handler) UpdateRouting(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	endpointID := c.Param("id")
	var req UpdateRoutingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	routing, err := h.service.UpdateEndpointRouting(c.Request.Context(), tenantID, endpointID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, routing)
}

// DeleteRouting godoc
//
//	@Summary		Delete endpoint routing
//	@Description	Delete routing configuration for an endpoint
//	@Tags			geo-routing
//	@Produce		json
//	@Param			id	path	string	true	"Endpoint ID"
//	@Success		204	"No content"
//	@Failure		401	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/endpoints/{id}/routing [delete]
func (h *Handler) DeleteRouting(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	endpointID := c.Param("id")
	if err := h.service.DeleteEndpointRouting(c.Request.Context(), tenantID, endpointID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// RouteDelivery godoc
//
//	@Summary		Get routing decision
//	@Description	Determine the best region for a webhook delivery
//	@Tags			geo-routing
//	@Produce		json
//	@Param			id	path		string	true	"Endpoint ID"
//	@Success		200	{object}	RoutingDecision
//	@Failure		400	{object}	map[string]interface{}
//	@Failure		401	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/endpoints/{id}/route [post]
func (h *Handler) RouteDelivery(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	endpointID := c.Param("id")
	clientIP := c.ClientIP()

	decision, err := h.service.RouteDelivery(c.Request.Context(), tenantID, endpointID, clientIP)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, decision)
}

// GetStats godoc
//
//	@Summary		Get routing statistics
//	@Description	Get routing statistics for the tenant
//	@Tags			geo-routing
//	@Produce		json
//	@Param			period	query		string	false	"Time period (hour, day, week, month)"	default(day)
//	@Success		200		{object}	RoutingStats
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/routing/stats [get]
func (h *Handler) GetStats(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	period := c.DefaultQuery("period", "day")
	if period != "hour" && period != "day" && period != "week" && period != "month" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid period: %s", period)})
		return
	}

	stats, err := h.service.GetRoutingStats(c.Request.Context(), tenantID, period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}
