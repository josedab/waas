package georouting

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
		geo.POST("/regions", h.CreateGeoRegion)
		geo.GET("/regions/health", h.GetRegionHealth)
		geo.GET("/regions/:name/health", h.GetRegionHealthByName)
		geo.POST("/policies", h.CreateGeoRoutingPolicy)
		geo.GET("/policies", h.ListGeoRoutingPolicies)
		geo.PUT("/policies/:id", h.UpdateGeoRoutingPolicy)
		geo.POST("/endpoints/:id/region", h.ConfigureEndpointRegion)
		geo.POST("/simulate", h.SimulateRouting)
		geo.GET("/routing-history", h.GetRoutingHistory)
		geo.GET("/dashboard", h.GetDashboard)
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

// CreateGeoRegion godoc
//
//	@Summary		Create a geo region
//	@Description	Register a new geographic region (admin)
//	@Tags			geo-routing
//	@Accept			json
//	@Produce		json
//	@Param			request	body		GeoRegion	true	"Region"
//	@Success		201		{object}	GeoRegion
//	@Failure		400		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/geo/regions [post]
func (h *Handler) CreateGeoRegion(c *gin.Context) {
	var region GeoRegion
	if err := c.ShouldBindJSON(&region); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if err := h.service.CreateGeoRegion(c.Request.Context(), &region); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, region)
}

// GetRegionHealthByName godoc
//
//	@Summary		Get region health by name
//	@Description	Get health status for a specific region
//	@Tags			geo-routing
//	@Produce		json
//	@Param			name	path		string	true	"Region name"
//	@Success		200		{object}	GeoRegionHealth
//	@Failure		404		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/geo/regions/{name}/health [get]
func (h *Handler) GetRegionHealthByName(c *gin.Context) {
	name := c.Param("name")
	health, err := h.service.GetRouter().healthTracker.CheckRegionHealth(c.Request.Context(), name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, health)
}

// CreateGeoRoutingPolicy godoc
//
//	@Summary		Create routing policy
//	@Description	Create a geo-routing policy for the tenant
//	@Tags			geo-routing
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateGeoRoutingPolicyRequest	true	"Policy"
//	@Success		201		{object}	GeoRoutingPolicy
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/geo/policies [post]
func (h *Handler) CreateGeoRoutingPolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateGeoRoutingPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}

	policy := &GeoRoutingPolicy{
		Name:             req.Name,
		Strategy:         req.Strategy,
		DataResidencyReq: req.DataResidency,
		PreferredRegions: req.PreferredRegions,
		FailoverOrder:    req.FailoverOrder,
		Weights:          req.Weights,
	}

	if err := h.service.CreateGeoRoutingPolicy(c.Request.Context(), tid, policy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, policy)
}

// ListGeoRoutingPolicies godoc
//
//	@Summary		List routing policies
//	@Description	Get routing policies for the tenant
//	@Tags			geo-routing
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Failure		401	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/geo/policies [get]
func (h *Handler) ListGeoRoutingPolicies(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}

	policies, err := h.service.ListGeoRoutingPolicies(c.Request.Context(), tid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"policies": policies})
}

// UpdateGeoRoutingPolicy godoc
//
//	@Summary		Update routing policy
//	@Description	Update a geo-routing policy
//	@Tags			geo-routing
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"Policy ID"
//	@Param			request	body		CreateGeoRoutingPolicyRequest	true	"Policy update"
//	@Success		200		{object}	GeoRoutingPolicy
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/geo/policies/{id} [put]
func (h *Handler) UpdateGeoRoutingPolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	policyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy ID"})
		return
	}

	var req CreateGeoRoutingPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	existing, err := h.service.repo.GetGeoRoutingPolicyByID(c.Request.Context(), policyID)
	if err != nil || existing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}

	existing.Name = req.Name
	existing.Strategy = req.Strategy
	existing.DataResidencyReq = req.DataResidency
	existing.PreferredRegions = req.PreferredRegions
	existing.FailoverOrder = req.FailoverOrder
	existing.Weights = req.Weights

	if err := h.service.UpdateGeoRoutingPolicy(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, existing)
}

// ConfigureEndpointRegion godoc
//
//	@Summary		Configure endpoint region
//	@Description	Set region preferences for an endpoint
//	@Tags			geo-routing
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"Endpoint ID"
//	@Param			request	body		ConfigureEndpointRegionRequest	true	"Region config"
//	@Success		200		{object}	EndpointRegionConfig
//	@Failure		400		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/geo/endpoints/{id}/region [post]
func (h *Handler) ConfigureEndpointRegion(c *gin.Context) {
	endpointID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid endpoint ID"})
		return
	}

	var req ConfigureEndpointRegionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	config := &EndpointRegionConfig{
		PrimaryRegion:   req.PrimaryRegion,
		FailoverRegions: req.FailoverRegions,
		DataResidencyRq: req.DataResidency,
	}

	if err := h.service.ConfigureEndpointRegion(c.Request.Context(), endpointID, config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// SimulateRouting godoc
//
//	@Summary		Simulate routing
//	@Description	Simulate a routing decision without actually sending
//	@Tags			geo-routing
//	@Accept			json
//	@Produce		json
//	@Param			request	body		SimulateRoutingRequest	true	"Simulation request"
//	@Success		200		{object}	GeoRoutingDecision
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/geo/simulate [post]
func (h *Handler) SimulateRouting(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}

	var req SimulateRoutingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	decision, err := h.service.SimulateRouting(c.Request.Context(), tid, req.SourceIP)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, decision)
}

// GetRoutingHistory godoc
//
//	@Summary		Get routing history
//	@Description	Get recent routing decision history
//	@Tags			geo-routing
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Failure		401	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/geo/routing-history [get]
func (h *Handler) GetRoutingHistory(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}

	history, err := h.service.GetRoutingHistory(c.Request.Context(), tid, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"history": history})
}

// GetDashboard godoc
//
//	@Summary		Get geo-routing dashboard
//	@Description	Get dashboard data with latency map and load distribution
//	@Tags			geo-routing
//	@Produce		json
//	@Success		200	{object}	GeoDashboardData
//	@Security		ApiKeyAuth
//	@Router			/geo/dashboard [get]
func (h *Handler) GetDashboard(c *gin.Context) {
	dashboard, err := h.service.GetGeoDashboard(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dashboard)
}
