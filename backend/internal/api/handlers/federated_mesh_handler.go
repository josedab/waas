package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/josedab/waas/internal/api/services"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/utils"
)

// FederatedMeshHandler handles federated mesh HTTP endpoints
type FederatedMeshHandler struct {
	service *services.FederatedMeshService
	logger  *utils.Logger
}

// NewFederatedMeshHandler creates a new federated mesh handler
func NewFederatedMeshHandler(service *services.FederatedMeshService, logger *utils.Logger) *FederatedMeshHandler {
	return &FederatedMeshHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers all federated mesh routes
func (h *FederatedMeshHandler) RegisterRoutes(rg *gin.RouterGroup) {
	mesh := rg.Group("/mesh")
	{
		// Regions
		mesh.GET("/regions", h.GetAllRegions)
		mesh.GET("/regions/active", h.GetActiveRegions)
		mesh.GET("/regions/:id", h.GetRegion)
		mesh.GET("/regions/metrics", h.GetRegionsWithMetrics)

		// Tenant Region Configuration
		mesh.POST("/tenant-region", h.SetupTenantRegion)
		mesh.GET("/tenant-region", h.GetTenantRegion)

		// Geo Routing Rules
		mesh.POST("/routing-rules", h.CreateRoutingRule)
		mesh.GET("/routing-rules", h.GetRoutingRules)
		mesh.POST("/route-event", h.RouteEvent)

		// Replication Streams
		mesh.POST("/replication-streams", h.CreateReplicationStream)
		mesh.GET("/replication-streams", h.GetReplicationStreams)

		// Failover
		mesh.POST("/failover", h.InitiateFailover)

		// Compliance
		mesh.POST("/compliance/check", h.CheckDataResidencyCompliance)

		// Dashboard
		mesh.GET("/dashboard", h.GetDashboard)
	}
}

// GetAllRegions retrieves all regions
func (h *FederatedMeshHandler) GetAllRegions(c *gin.Context) {
	regions, err := h.service.GetAllRegions(c.Request.Context())
	if err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, regions)
}

// GetActiveRegions retrieves all active regions
func (h *FederatedMeshHandler) GetActiveRegions(c *gin.Context) {
	regions, err := h.service.GetActiveRegions(c.Request.Context())
	if err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, regions)
}

// GetRegion retrieves a region by ID
func (h *FederatedMeshHandler) GetRegion(c *gin.Context) {
	regionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		// Try by code
		region, err := h.service.GetRegionByCode(c.Request.Context(), c.Param("id"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "region not found"})
			return
		}
		c.JSON(http.StatusOK, region)
		return
	}

	region, err := h.service.GetRegion(c.Request.Context(), regionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "region not found"})
		return
	}

	c.JSON(http.StatusOK, region)
}

// GetRegionsWithMetrics retrieves regions with health metrics
func (h *FederatedMeshHandler) GetRegionsWithMetrics(c *gin.Context) {
	regions, err := h.service.GetRegionsWithMetrics(c.Request.Context())
	if err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, regions)
}

// SetupTenantRegion sets up tenant region configuration
func (h *FederatedMeshHandler) SetupTenantRegion(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found"})
		return
	}

	var req models.CreateTenantRegionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantRegion, err := h.service.SetupTenantRegion(c.Request.Context(), tenantID.(uuid.UUID), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, tenantRegion)
}

// GetTenantRegion retrieves tenant region configuration
func (h *FederatedMeshHandler) GetTenantRegion(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found"})
		return
	}

	tenantRegion, err := h.service.GetTenantRegion(c.Request.Context(), tenantID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant region not configured"})
		return
	}

	c.JSON(http.StatusOK, tenantRegion)
}

// CreateRoutingRule creates a geo-routing rule
func (h *FederatedMeshHandler) CreateRoutingRule(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found"})
		return
	}

	var req models.CreateGeoRoutingRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule, err := h.service.CreateRoutingRule(c.Request.Context(), tenantID.(uuid.UUID), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, rule)
}

// GetRoutingRules retrieves routing rules for a tenant
func (h *FederatedMeshHandler) GetRoutingRules(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found"})
		return
	}

	rules, err := h.service.GetRoutingRules(c.Request.Context(), tenantID.(uuid.UUID))
	if err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, rules)
}

// RouteEvent routes an event to the optimal region
func (h *FederatedMeshHandler) RouteEvent(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found"})
		return
	}

	var req models.RouteEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	decision, err := h.service.RouteEvent(c.Request.Context(), tenantID.(uuid.UUID), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, decision)
}

// CreateReplicationStream creates a replication stream
func (h *FederatedMeshHandler) CreateReplicationStream(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found"})
		return
	}

	var req models.CreateReplicationStreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	stream, err := h.service.CreateReplicationStream(c.Request.Context(), tenantID.(uuid.UUID), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, stream)
}

// GetReplicationStreams retrieves replication streams for a tenant
func (h *FederatedMeshHandler) GetReplicationStreams(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found"})
		return
	}

	streams, err := h.service.GetReplicationStreams(c.Request.Context(), tenantID.(uuid.UUID))
	if err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, streams)
}

// InitiateFailover initiates a region failover
func (h *FederatedMeshHandler) InitiateFailover(c *gin.Context) {
	var req models.InitiateFailoverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	event, err := h.service.InitiateFailover(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, event)
}

// CheckDataResidencyCompliance checks data residency compliance
func (h *FederatedMeshHandler) CheckDataResidencyCompliance(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found"})
		return
	}

	var req struct {
		SourceRegionID string `json:"source_region_id" binding:"required"`
		TargetRegionID string `json:"target_region_id" binding:"required"`
		DataType       string `json:"data_type"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sourceRegionID, err := uuid.Parse(req.SourceRegionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source_region_id"})
		return
	}

	targetRegionID, err := uuid.Parse(req.TargetRegionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid target_region_id"})
		return
	}

	audit, err := h.service.CheckDataResidencyCompliance(c.Request.Context(), tenantID.(uuid.UUID), sourceRegionID, targetRegionID, req.DataType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if audit == nil {
		c.JSON(http.StatusOK, gin.H{"status": "no_restrictions"})
		return
	}

	c.JSON(http.StatusOK, audit)
}

// GetDashboard retrieves the federated mesh dashboard
func (h *FederatedMeshHandler) GetDashboard(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found"})
		return
	}

	dashboard, err := h.service.GetMeshDashboard(c.Request.Context(), tenantID.(uuid.UUID))
	if err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, dashboard)
}
