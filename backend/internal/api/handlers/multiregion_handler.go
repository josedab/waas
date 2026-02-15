package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/josedab/waas/pkg/multiregion"
	"github.com/josedab/waas/pkg/utils"

	"github.com/gin-gonic/gin"
)

// MultiRegionHandler handles multi-region HTTP requests
type MultiRegionHandler struct {
	repo            multiregion.Repository
	router          *multiregion.Router
	failoverManager *multiregion.FailoverManager
	healthChecker   *multiregion.HealthChecker
	logger          *utils.Logger
}

// NewMultiRegionHandler creates a new multi-region handler
func NewMultiRegionHandler(
	repo multiregion.Repository,
	router *multiregion.Router,
	failoverManager *multiregion.FailoverManager,
	healthChecker *multiregion.HealthChecker,
	logger *utils.Logger,
) *MultiRegionHandler {
	return &MultiRegionHandler{
		repo:            repo,
		router:          router,
		failoverManager: failoverManager,
		healthChecker:   healthChecker,
		logger:          logger,
	}
}

// CreateRegionRequest represents region creation request
type CreateRegionRequest struct {
	Name     string                    `json:"name" binding:"required"`
	Code     string                    `json:"code" binding:"required"`
	Endpoint string                    `json:"endpoint" binding:"required"`
	Priority int                       `json:"priority"`
	Metadata multiregion.Metadata      `json:"metadata"`
}

// UpdateRegionRequest represents region update request
type UpdateRegionRequest struct {
	Name      string               `json:"name"`
	Endpoint  string               `json:"endpoint"`
	Priority  *int                 `json:"priority"`
	IsActive  *bool                `json:"is_active"`
	IsPrimary *bool                `json:"is_primary"`
	Metadata  *multiregion.Metadata `json:"metadata"`
}

// TriggerFailoverRequest represents failover trigger request
type TriggerFailoverRequest struct {
	FromRegion string `json:"from_region" binding:"required"`
	ToRegion   string `json:"to_region" binding:"required"`
	Reason     string `json:"reason"`
}

// CreateRoutingPolicyRequest represents routing policy creation
type CreateRoutingPolicyRequest struct {
	PolicyType      string              `json:"policy_type" binding:"required"`
	PrimaryRegion   string              `json:"primary_region" binding:"required"`
	FallbackRegions []string            `json:"fallback_regions"`
	GeoRules        []multiregion.GeoRule `json:"geo_rules"`
	Weights         map[string]int      `json:"weights"`
}

// CreateReplicationConfigRequest represents replication config creation
type CreateReplicationConfigRequest struct {
	SourceRegion   string   `json:"source_region" binding:"required"`
	TargetRegion   string   `json:"target_region" binding:"required"`
	Mode           string   `json:"mode" binding:"required"`
	LagThresholdMs int64    `json:"lag_threshold_ms"`
	RetentionDays  int      `json:"retention_days"`
	Tables         []string `json:"tables"`
}

// ListRegions lists all regions
// @Summary List regions
// @Tags regions
// @Produce json
// @Param active query bool false "Filter active only"
// @Success 200 {array} multiregion.Region
// @Router /regions [get]
func (h *MultiRegionHandler) ListRegions(c *gin.Context) {
	activeOnly := c.Query("active") == "true"

	var regions []*multiregion.Region
	var err error

	if activeOnly {
		regions, err = h.repo.ListActiveRegions(c.Request.Context())
	} else {
		regions, err = h.repo.ListRegions(c.Request.Context())
	}

	if err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, regions)
}

// GetRegion gets a region by ID
// @Summary Get region
// @Tags regions
// @Produce json
// @Param id path string true "Region ID"
// @Success 200 {object} multiregion.Region
// @Router /regions/{id} [get]
func (h *MultiRegionHandler) GetRegion(c *gin.Context) {
	id := c.Param("id")

	region, err := h.repo.GetRegion(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "region not found"})
		return
	}

	c.JSON(http.StatusOK, region)
}

// CreateRegion creates a new region
// @Summary Create region
// @Tags regions
// @Accept json
// @Produce json
// @Param request body CreateRegionRequest true "Region request"
// @Success 201 {object} multiregion.Region
// @Router /regions [post]
func (h *MultiRegionHandler) CreateRegion(c *gin.Context) {
	var req CreateRegionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	region := &multiregion.Region{
		ID:       generateRegionID(),
		Name:     req.Name,
		Code:     req.Code,
		Endpoint: req.Endpoint,
		Priority: req.Priority,
		Metadata: req.Metadata,
		IsActive: true,
	}

	if err := h.repo.CreateRegion(c.Request.Context(), region); err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, region)
}

// UpdateRegion updates a region
// @Summary Update region
// @Tags regions
// @Accept json
// @Produce json
// @Param id path string true "Region ID"
// @Param request body UpdateRegionRequest true "Update request"
// @Success 200 {object} multiregion.Region
// @Router /regions/{id} [patch]
func (h *MultiRegionHandler) UpdateRegion(c *gin.Context) {
	id := c.Param("id")

	var req UpdateRegionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	region, err := h.repo.GetRegion(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "region not found"})
		return
	}

	if req.Name != "" {
		region.Name = req.Name
	}
	if req.Endpoint != "" {
		region.Endpoint = req.Endpoint
	}
	if req.Priority != nil {
		region.Priority = *req.Priority
	}
	if req.IsActive != nil {
		region.IsActive = *req.IsActive
	}
	if req.IsPrimary != nil {
		region.IsPrimary = *req.IsPrimary
	}
	if req.Metadata != nil {
		region.Metadata = *req.Metadata
	}

	if err := h.repo.UpdateRegion(c.Request.Context(), region); err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, region)
}

// DeleteRegion deletes a region
// @Summary Delete region
// @Tags regions
// @Param id path string true "Region ID"
// @Success 204
// @Router /regions/{id} [delete]
func (h *MultiRegionHandler) DeleteRegion(c *gin.Context) {
	id := c.Param("id")

	if err := h.repo.DeleteRegion(c.Request.Context(), id); err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// GetRegionHealth gets health for all regions
// @Summary Get region health
// @Tags regions
// @Produce json
// @Success 200 {object} map[string]multiregion.RegionHealth
// @Router /regions/health [get]
func (h *MultiRegionHandler) GetRegionHealth(c *gin.Context) {
	health := h.healthChecker.GetAllHealth()
	c.JSON(http.StatusOK, health)
}

// GetSingleRegionHealth gets health for a specific region
// @Summary Get single region health
// @Tags regions
// @Produce json
// @Param id path string true "Region ID"
// @Success 200 {object} multiregion.RegionHealth
// @Router /regions/{id}/health [get]
func (h *MultiRegionHandler) GetSingleRegionHealth(c *gin.Context) {
	id := c.Param("id")

	health := h.healthChecker.GetHealth(id)
	if health == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "health data not found"})
		return
	}

	c.JSON(http.StatusOK, health)
}

// TriggerFailover triggers a manual failover
// @Summary Trigger failover
// @Tags failover
// @Accept json
// @Produce json
// @Param request body TriggerFailoverRequest true "Failover request"
// @Success 201 {object} multiregion.FailoverEvent
// @Router /failover [post]
func (h *MultiRegionHandler) TriggerFailover(c *gin.Context) {
	var req TriggerFailoverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	event, err := h.failoverManager.TriggerFailover(c.Request.Context(), req.FromRegion, req.ToRegion, req.Reason)
	if err != nil {
		h.logger.Error("Failed to trigger failover", map[string]interface{}{"error": err.Error()})
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, event)
}

// ListFailoverEvents lists failover events
// @Summary List failover events
// @Tags failover
// @Produce json
// @Param limit query int false "Limit"
// @Success 200 {array} multiregion.FailoverEvent
// @Router /failover/events [get]
func (h *MultiRegionHandler) ListFailoverEvents(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	events, err := h.repo.ListFailoverEvents(c.Request.Context(), limit)
	if err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, events)
}

// GetFailoverEvent gets a specific failover event
// @Summary Get failover event
// @Tags failover
// @Produce json
// @Param id path string true "Event ID"
// @Success 200 {object} multiregion.FailoverEvent
// @Router /failover/events/{id} [get]
func (h *MultiRegionHandler) GetFailoverEvent(c *gin.Context) {
	id := c.Param("id")

	event, err := h.repo.GetFailoverEvent(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}

	c.JSON(http.StatusOK, event)
}

// GetRoutingPolicy gets the routing policy for a tenant
// @Summary Get routing policy
// @Tags routing
// @Produce json
// @Success 200 {object} multiregion.RoutingPolicy
// @Router /routing/policy [get]
func (h *MultiRegionHandler) GetRoutingPolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	policy, err := h.repo.GetRoutingPolicy(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}

	c.JSON(http.StatusOK, policy)
}

// CreateRoutingPolicy creates a routing policy
// @Summary Create routing policy
// @Tags routing
// @Accept json
// @Produce json
// @Param request body CreateRoutingPolicyRequest true "Policy request"
// @Success 201 {object} multiregion.RoutingPolicy
// @Router /routing/policy [post]
func (h *MultiRegionHandler) CreateRoutingPolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateRoutingPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	policy := &multiregion.RoutingPolicy{
		ID:              generateRegionID(),
		TenantID:        tenantID,
		PolicyType:      multiregion.RoutingType(req.PolicyType),
		PrimaryRegion:   req.PrimaryRegion,
		FallbackRegions: req.FallbackRegions,
		GeoRules:        req.GeoRules,
		Weights:         req.Weights,
		Enabled:         true,
	}

	if err := h.repo.CreateRoutingPolicy(c.Request.Context(), policy); err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, policy)
}

// UpdateRoutingPolicy updates a routing policy
// @Summary Update routing policy
// @Tags routing
// @Accept json
// @Produce json
// @Param request body CreateRoutingPolicyRequest true "Policy update"
// @Success 200 {object} multiregion.RoutingPolicy
// @Router /routing/policy [patch]
func (h *MultiRegionHandler) UpdateRoutingPolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateRoutingPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	policy, err := h.repo.GetRoutingPolicy(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}

	policy.PolicyType = multiregion.RoutingType(req.PolicyType)
	policy.PrimaryRegion = req.PrimaryRegion
	policy.FallbackRegions = req.FallbackRegions
	policy.GeoRules = req.GeoRules
	policy.Weights = req.Weights

	if err := h.repo.UpdateRoutingPolicy(c.Request.Context(), policy); err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, policy)
}

// DeleteRoutingPolicy deletes a routing policy
// @Summary Delete routing policy
// @Tags routing
// @Success 204
// @Router /routing/policy [delete]
func (h *MultiRegionHandler) DeleteRoutingPolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.repo.DeleteRoutingPolicy(c.Request.Context(), tenantID); err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ListReplicationConfigs lists replication configurations
// @Summary List replication configs
// @Tags replication
// @Produce json
// @Success 200 {array} multiregion.ReplicationConfig
// @Router /replication/configs [get]
func (h *MultiRegionHandler) ListReplicationConfigs(c *gin.Context) {
	configs, err := h.repo.ListReplicationConfigs(c.Request.Context())
	if err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, configs)
}

// CreateReplicationConfig creates a replication configuration
// @Summary Create replication config
// @Tags replication
// @Accept json
// @Produce json
// @Param request body CreateReplicationConfigRequest true "Replication config"
// @Success 201 {object} multiregion.ReplicationConfig
// @Router /replication/configs [post]
func (h *MultiRegionHandler) CreateReplicationConfig(c *gin.Context) {
	var req CreateReplicationConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config := &multiregion.ReplicationConfig{
		ID:             generateRegionID(),
		SourceRegion:   req.SourceRegion,
		TargetRegion:   req.TargetRegion,
		Mode:           multiregion.ReplicationMode(req.Mode),
		Enabled:        true,
		LagThresholdMs: req.LagThresholdMs,
		RetentionDays:  req.RetentionDays,
		Tables:         req.Tables,
	}

	if config.LagThresholdMs == 0 {
		config.LagThresholdMs = 1000
	}
	if config.RetentionDays == 0 {
		config.RetentionDays = 30
	}

	if err := h.repo.CreateReplicationConfig(c.Request.Context(), config); err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, config)
}

// GetReplicationConfig gets a replication configuration
// @Summary Get replication config
// @Tags replication
// @Produce json
// @Param id path string true "Config ID"
// @Success 200 {object} multiregion.ReplicationConfig
// @Router /replication/configs/{id} [get]
func (h *MultiRegionHandler) GetReplicationConfig(c *gin.Context) {
	id := c.Param("id")

	config, err := h.repo.GetReplicationConfig(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	c.JSON(http.StatusOK, config)
}

func generateRegionID() string {
	return "reg-" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

// RegisterMultiRegionRoutes registers multi-region routes
func RegisterMultiRegionRoutes(r *gin.RouterGroup, h *MultiRegionHandler) {
	// Region management
	regions := r.Group("/regions")
	{
		regions.GET("", h.ListRegions)
		regions.POST("", h.CreateRegion)
		regions.GET("/health", h.GetRegionHealth)
		regions.GET("/:id", h.GetRegion)
		regions.PATCH("/:id", h.UpdateRegion)
		regions.DELETE("/:id", h.DeleteRegion)
		regions.GET("/:id/health", h.GetSingleRegionHealth)
	}

	// Failover management
	failover := r.Group("/failover")
	{
		failover.POST("", h.TriggerFailover)
		failover.GET("/events", h.ListFailoverEvents)
		failover.GET("/events/:id", h.GetFailoverEvent)
	}

	// Routing policies
	routing := r.Group("/routing")
	{
		routing.GET("/policy", h.GetRoutingPolicy)
		routing.POST("/policy", h.CreateRoutingPolicy)
		routing.PATCH("/policy", h.UpdateRoutingPolicy)
		routing.DELETE("/policy", h.DeleteRoutingPolicy)
	}

	// Replication
	replication := r.Group("/replication")
	{
		replication.GET("/configs", h.ListReplicationConfigs)
		replication.POST("/configs", h.CreateReplicationConfig)
		replication.GET("/configs/:id", h.GetReplicationConfig)
	}
}
