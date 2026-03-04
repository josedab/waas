package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/josedab/waas/internal/api/services"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/utils"
)

// EdgeFunctionsHandler handles edge functions HTTP endpoints
type EdgeFunctionsHandler struct {
	service *services.EdgeFunctionsService
	logger  *utils.Logger
}

// NewEdgeFunctionsHandler creates a new edge functions handler
func NewEdgeFunctionsHandler(service *services.EdgeFunctionsService, logger *utils.Logger) *EdgeFunctionsHandler {
	return &EdgeFunctionsHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers all edge functions routes
func (h *EdgeFunctionsHandler) RegisterRoutes(rg *gin.RouterGroup) {
	functions := rg.Group("/edge-functions")
	{
		// Functions
		functions.POST("", h.CreateFunction)
		functions.GET("", h.GetFunctions)
		functions.GET("/:id", h.GetFunction)
		functions.PUT("/:id", h.UpdateFunction)
		functions.DELETE("/:id", h.DeleteFunction)

		// Deployment
		functions.POST("/:id/deploy", h.DeployFunction)
		functions.GET("/:id/deployments", h.GetDeployments)

		// Invocation
		functions.POST("/:id/invoke", h.InvokeFunction)
		functions.GET("/:id/invocations", h.GetInvocations)

		// Triggers
		functions.POST("/:id/triggers", h.CreateTrigger)
		functions.GET("/:id/triggers", h.GetTriggers)

		// Versions
		functions.GET("/:id/versions", h.GetVersions)
		functions.POST("/:id/rollback", h.RollbackFunction)

		// Testing
		functions.POST("/:id/test", h.RunTest)

		// Locations
		functions.GET("/locations", h.GetLocations)

		// Dashboard
		functions.GET("/dashboard", h.GetDashboard)
	}
}

// CreateFunction creates a new edge function
func (h *EdgeFunctionsHandler) CreateFunction(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	var req models.CreateEdgeFunctionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fn, err := h.service.CreateFunction(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, fn)
}

// GetFunctions retrieves all functions for a tenant
func (h *EdgeFunctionsHandler) GetFunctions(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	functions, err := h.service.GetFunctions(c.Request.Context(), tenantID)
	if err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, functions)
}

// GetFunction retrieves a function by ID
func (h *EdgeFunctionsHandler) GetFunction(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	functionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid function id"})
		return
	}

	// Check if details are requested
	if c.Query("details") == "true" {
		fn, err := h.service.GetFunctionWithDetails(c.Request.Context(), tenantID, functionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, fn)
		return
	}

	fn, err := h.service.GetFunction(c.Request.Context(), tenantID, functionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, fn)
}

// UpdateFunction updates a function
func (h *EdgeFunctionsHandler) UpdateFunction(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	functionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid function id"})
		return
	}

	var req models.UpdateEdgeFunctionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fn, err := h.service.UpdateFunction(c.Request.Context(), tenantID, functionID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, fn)
}

// DeleteFunction deletes a function
func (h *EdgeFunctionsHandler) DeleteFunction(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	functionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid function id"})
		return
	}

	if err := h.service.DeleteFunction(c.Request.Context(), tenantID, functionID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// DeployFunction deploys a function to edge locations
func (h *EdgeFunctionsHandler) DeployFunction(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	functionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid function id"})
		return
	}

	var req models.DeployEdgeFunctionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deployments, err := h.service.DeployFunction(c.Request.Context(), tenantID, functionID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":      "deployed",
		"deployments": deployments,
	})
}

// GetDeployments retrieves deployments for a function
func (h *EdgeFunctionsHandler) GetDeployments(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	functionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid function id"})
		return
	}

	deployments, err := h.service.GetDeployments(c.Request.Context(), tenantID, functionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, deployments)
}

// InvokeFunction invokes a function
func (h *EdgeFunctionsHandler) InvokeFunction(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	functionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid function id"})
		return
	}

	var req models.InvokeFunctionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.InvokeFunction(c.Request.Context(), tenantID, functionID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetInvocations retrieves invocations for a function
func (h *EdgeFunctionsHandler) GetInvocations(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	functionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid function id"})
		return
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > 100 {
		limit = 100
	}

	invocations, err := h.service.GetInvocations(c.Request.Context(), tenantID, functionID, limit)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, invocations)
}

// CreateTrigger creates a function trigger
func (h *EdgeFunctionsHandler) CreateTrigger(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	functionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid function id"})
		return
	}

	var req models.CreateTriggerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	trigger, err := h.service.CreateTrigger(c.Request.Context(), tenantID, functionID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, trigger)
}

// GetTriggers retrieves triggers for a function
func (h *EdgeFunctionsHandler) GetTriggers(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	functionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid function id"})
		return
	}

	triggers, err := h.service.GetTriggers(c.Request.Context(), tenantID, functionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, triggers)
}

// GetVersions retrieves versions for a function
func (h *EdgeFunctionsHandler) GetVersions(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	functionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid function id"})
		return
	}

	versions, err := h.service.GetVersions(c.Request.Context(), tenantID, functionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, versions)
}

// RollbackFunction rolls back to a previous version
func (h *EdgeFunctionsHandler) RollbackFunction(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	functionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid function id"})
		return
	}

	var req struct {
		Version int `json:"version" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fn, err := h.service.RollbackFunction(c.Request.Context(), tenantID, functionID, req.Version)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, fn)
}

// RunTest runs a function test
func (h *EdgeFunctionsHandler) RunTest(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	functionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid function id"})
		return
	}

	var req models.RunTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	test, err := h.service.RunTest(c.Request.Context(), tenantID, functionID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, test)
}

// GetLocations retrieves all edge locations
func (h *EdgeFunctionsHandler) GetLocations(c *gin.Context) {
	locations, err := h.service.GetActiveLocations(c.Request.Context())
	if err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, locations)
}

// GetDashboard retrieves the edge functions dashboard
func (h *EdgeFunctionsHandler) GetDashboard(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	dashboard, err := h.service.GetDashboard(c.Request.Context(), tenantID)
	if err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, dashboard)
}
