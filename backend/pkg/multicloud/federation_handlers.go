package multicloud

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// FederationHandler provides HTTP handlers for multi-cloud federation
type FederationHandler struct {
	service *FederationService
}

// NewFederationHandler creates a new handler
func NewFederationHandler(service *FederationService) *FederationHandler {
	return &FederationHandler{service: service}
}

// RegisterFederationRoutes registers HTTP routes for federation
func (h *FederationHandler) RegisterFederationRoutes(r gin.IRouter) {
	fed := r.Group("/federation")
	{
		// Clusters
		fed.POST("/clusters", h.RegisterCluster)
		fed.GET("/clusters", h.ListClusters)
		fed.GET("/clusters/:id", h.GetCluster)
		fed.PUT("/clusters/:id", h.UpdateCluster)
		fed.DELETE("/clusters/:id", h.DeleteCluster)
		fed.POST("/clusters/health", h.CheckClusterHealth)

		// Routes
		fed.POST("/routes", h.CreateRoute)
		fed.GET("/routes", h.ListRoutes)
		fed.GET("/routes/:id", h.GetRoute)
		fed.PUT("/routes/:id", h.UpdateRoute)
		fed.DELETE("/routes/:id", h.DeleteRoute)

		// Failover
		fed.POST("/failover", h.InitiateFailover)
		fed.GET("/failover/history", h.GetFailoverHistory)

		// Metrics
		fed.GET("/metrics", h.GetCloudMetrics)

		// Forward (for testing)
		fed.POST("/forward", h.ForwardRequest)
	}
}

// FederationErrorResponse represents an error response
type FederationErrorResponse struct {
	Error string `json:"error"`
}

// RegisterCluster registers a new cluster
// @Summary Register a federation cluster
// @Tags federation
// @Accept json
// @Produce json
// @Param request body FederationCluster true "Cluster details"
// @Success 201 {object} FederationCluster
// @Router /federation/clusters [post]
func (h *FederationHandler) RegisterCluster(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	var cluster FederationCluster
	if err := c.ShouldBindJSON(&cluster); err != nil {
		c.JSON(http.StatusBadRequest, FederationErrorResponse{Error: err.Error()})
		return
	}

	result, err := h.service.RegisterCluster(c.Request.Context(), tenantID, &cluster)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, result)
}

// ListClusters lists all clusters
// @Summary List federation clusters
// @Tags federation
// @Produce json
// @Param provider query string false "Filter by provider"
// @Success 200 {array} FederationCluster
// @Router /federation/clusters [get]
func (h *FederationHandler) ListClusters(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	var provider *Provider
	if p := c.Query("provider"); p != "" {
		cp := Provider(p)
		provider = &cp
	}

	clusters, err := h.service.ListClusters(c.Request.Context(), tenantID, provider)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, clusters)
}

// GetCluster gets a specific cluster
// @Summary Get a federation cluster
// @Tags federation
// @Produce json
// @Param id path string true "Cluster ID"
// @Success 200 {object} FederationCluster
// @Router /federation/clusters/{id} [get]
func (h *FederationHandler) GetCluster(c *gin.Context) {
	cluster, err := h.service.GetCluster(c.Request.Context(), c.Param("id"))
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, cluster)
}

// UpdateCluster updates a cluster
// @Summary Update a federation cluster
// @Tags federation
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID"
// @Param request body FederationCluster true "Cluster details"
// @Success 200 {object} FederationCluster
// @Router /federation/clusters/{id} [put]
func (h *FederationHandler) UpdateCluster(c *gin.Context) {
	var cluster FederationCluster
	if err := c.ShouldBindJSON(&cluster); err != nil {
		c.JSON(http.StatusBadRequest, FederationErrorResponse{Error: err.Error()})
		return
	}

	cluster.ID = c.Param("id")

	if err := h.service.UpdateCluster(c.Request.Context(), &cluster); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, cluster)
}

// DeleteCluster deletes a cluster
// @Summary Delete a federation cluster
// @Tags federation
// @Produce json
// @Param id path string true "Cluster ID"
// @Success 200
// @Router /federation/clusters/{id} [delete]
func (h *FederationHandler) DeleteCluster(c *gin.Context) {
	if err := h.service.repo.DeleteCluster(c.Request.Context(), c.Param("id")); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// CheckClusterHealth checks cluster health
// @Summary Check all cluster health
// @Tags federation
// @Produce json
// @Success 200 {array} FederationCluster
// @Router /federation/clusters/health [post]
func (h *FederationHandler) CheckClusterHealth(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	clusters, err := h.service.CheckClusterHealth(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, clusters)
}

// CreateRoute creates a routing rule
// @Summary Create a federation route
// @Tags federation
// @Accept json
// @Produce json
// @Param request body FederationRoute true "Route details"
// @Success 201 {object} FederationRoute
// @Router /federation/routes [post]
func (h *FederationHandler) CreateRoute(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	var route FederationRoute
	if err := c.ShouldBindJSON(&route); err != nil {
		c.JSON(http.StatusBadRequest, FederationErrorResponse{Error: err.Error()})
		return
	}

	result, err := h.service.CreateFederationRoute(c.Request.Context(), tenantID, &route)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, result)
}

// ListRoutes lists all routes
// @Summary List federation routes
// @Tags federation
// @Produce json
// @Success 200 {array} FederationRoute
// @Router /federation/routes [get]
func (h *FederationHandler) ListRoutes(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	routes, err := h.service.ListFederationRoutes(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, routes)
}

// GetRoute gets a specific route
// @Summary Get a federation route
// @Tags federation
// @Produce json
// @Param id path string true "Route ID"
// @Success 200 {object} FederationRoute
// @Router /federation/routes/{id} [get]
func (h *FederationHandler) GetRoute(c *gin.Context) {
	route, err := h.service.GetFederationRoute(c.Request.Context(), c.Param("id"))
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, route)
}

// UpdateRoute updates a route
// @Summary Update a federation route
// @Tags federation
// @Accept json
// @Produce json
// @Param id path string true "Route ID"
// @Param request body FederationRoute true "Route details"
// @Success 200 {object} FederationRoute
// @Router /federation/routes/{id} [put]
func (h *FederationHandler) UpdateRoute(c *gin.Context) {
	var route FederationRoute
	if err := c.ShouldBindJSON(&route); err != nil {
		c.JSON(http.StatusBadRequest, FederationErrorResponse{Error: err.Error()})
		return
	}

	route.ID = c.Param("id")

	if err := h.service.repo.UpdateFederationRoute(c.Request.Context(), &route); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, route)
}

// DeleteRoute deletes a route
// @Summary Delete a federation route
// @Tags federation
// @Produce json
// @Param id path string true "Route ID"
// @Success 200
// @Router /federation/routes/{id} [delete]
func (h *FederationHandler) DeleteRoute(c *gin.Context) {
	if err := h.service.repo.DeleteFederationRoute(c.Request.Context(), c.Param("id")); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// FailoverRequest represents a failover request
type FailoverRequest struct {
	RouteID       string `json:"route_id" binding:"required"`
	FromClusterID string `json:"from_cluster_id" binding:"required"`
	ToClusterID   string `json:"to_cluster_id" binding:"required"`
	Reason        string `json:"reason"`
}

// InitiateFailover initiates a manual failover
// @Summary Initiate failover
// @Tags federation
// @Accept json
// @Produce json
// @Param request body FailoverRequest true "Failover request"
// @Success 200 {object} FailoverEvent
// @Router /federation/failover [post]
func (h *FederationHandler) InitiateFailover(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	var req FailoverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, FederationErrorResponse{Error: err.Error()})
		return
	}

	initiatedBy := c.GetString("user_id")
	if initiatedBy == "" {
		initiatedBy = "system"
	}

	event, err := h.service.InitiateFailover(c.Request.Context(), tenantID, req.RouteID, req.FromClusterID, req.ToClusterID, req.Reason, initiatedBy)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, event)
}

// GetFailoverHistory gets failover history
// @Summary Get failover history
// @Tags federation
// @Produce json
// @Param days query int false "Number of days to look back"
// @Success 200 {array} FailoverEvent
// @Router /federation/failover/history [get]
func (h *FederationHandler) GetFailoverHistory(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	since := time.Now().AddDate(0, 0, -days)

	events, err := h.service.repo.ListFailoverEvents(c.Request.Context(), tenantID, since)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, events)
}

// GetCloudMetrics gets cross-cloud metrics
// @Summary Get cloud metrics
// @Tags federation
// @Produce json
// @Success 200 {object} CloudMetrics
// @Router /federation/metrics [get]
func (h *FederationHandler) GetCloudMetrics(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	metrics, err := h.service.GetCloudMetrics(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// ForwardRequest forwards a test request
// @Summary Forward a request through federation
// @Tags federation
// @Accept json
// @Produce json
// @Param request body ForwardRequest true "Forward request"
// @Success 200 {object} ForwardResponse
// @Router /federation/forward [post]
func (h *FederationHandler) ForwardRequest(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	var req ForwardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, FederationErrorResponse{Error: err.Error()})
		return
	}

	resp, err := h.service.RouteRequest(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
