package cloudctl

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for the cloud control plane
type Handler struct {
	service *Service
}

// NewHandler creates a new cloud control plane handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers cloud control plane routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	cloud := router.Group("/cloud")
	{
		// Tenant provisioning
		cloud.POST("/tenants", h.ProvisionTenant)
		cloud.GET("/tenants", h.ListTenants)
		cloud.GET("/tenants/:id", h.GetTenant)
		cloud.PUT("/tenants/:id/plan", h.UpdatePlan)
		cloud.POST("/tenants/:id/suspend", h.SuspendTenant)
		cloud.POST("/tenants/:id/reactivate", h.ReactivateTenant)
		cloud.DELETE("/tenants/:id", h.DeleteTenant)

		// Usage
		cloud.GET("/tenants/:id/usage", h.GetUsage)
		cloud.GET("/tenants/:id/usage/history", h.GetUsageHistory)

		// Scaling
		cloud.GET("/tenants/:id/scaling", h.GetScalingConfig)
		cloud.PUT("/tenants/:id/scaling", h.UpdateScalingConfig)

		// Platform
		cloud.GET("/dashboard", h.GetDashboard)
		cloud.GET("/regions", h.GetRegions)
		cloud.GET("/plans", h.GetPlans)
	}
}

// @Summary Provision a new cloud tenant
// @Tags Cloud
// @Accept json
// @Produce json
// @Param body body ProvisionRequest true "Tenant provisioning request"
// @Success 201 {object} CloudTenant
// @Router /cloud/tenants [post]
func (h *Handler) ProvisionTenant(c *gin.Context) {
	var req ProvisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	tenant, err := h.service.ProvisionTenant(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "PROVISION_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusCreated, tenant)
}

// @Summary List cloud tenants
// @Tags Cloud
// @Produce json
// @Param status query string false "Filter by status"
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} map[string]interface{}
// @Router /cloud/tenants [get]
func (h *Handler) ListTenants(c *gin.Context) {
	status := c.Query("status")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	tenants, total, err := h.service.ListTenants(c.Request.Context(), status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "LIST_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tenants": tenants, "total": total})
}

// @Summary Get a cloud tenant
// @Tags Cloud
// @Produce json
// @Param id path string true "Tenant ID"
// @Success 200 {object} CloudTenant
// @Router /cloud/tenants/{id} [get]
func (h *Handler) GetTenant(c *gin.Context) {
	tenantID := c.Param("id")

	tenant, err := h.service.GetTenant(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, tenant)
}

// @Summary Update tenant plan
// @Tags Cloud
// @Accept json
// @Produce json
// @Param id path string true "Tenant ID"
// @Param body body UpdatePlanRequest true "New plan"
// @Success 200 {object} CloudTenant
// @Router /cloud/tenants/{id}/plan [put]
func (h *Handler) UpdatePlan(c *gin.Context) {
	tenantID := c.Param("id")

	var req UpdatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	tenant, err := h.service.UpdatePlan(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "UPDATE_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, tenant)
}

// @Summary Suspend a tenant
// @Tags Cloud
// @Param id path string true "Tenant ID"
// @Success 204
// @Router /cloud/tenants/{id}/suspend [post]
func (h *Handler) SuspendTenant(c *gin.Context) {
	tenantID := c.Param("id")

	if err := h.service.SuspendTenant(c.Request.Context(), tenantID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "SUSPEND_FAILED", "message": err.Error()}})
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary Reactivate a tenant
// @Tags Cloud
// @Param id path string true "Tenant ID"
// @Success 204
// @Router /cloud/tenants/{id}/reactivate [post]
func (h *Handler) ReactivateTenant(c *gin.Context) {
	tenantID := c.Param("id")

	if err := h.service.ReactivateTenant(c.Request.Context(), tenantID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "REACTIVATE_FAILED", "message": err.Error()}})
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary Delete a tenant
// @Tags Cloud
// @Param id path string true "Tenant ID"
// @Success 204
// @Router /cloud/tenants/{id} [delete]
func (h *Handler) DeleteTenant(c *gin.Context) {
	tenantID := c.Param("id")

	if err := h.service.DeleteTenant(c.Request.Context(), tenantID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "DELETE_FAILED", "message": err.Error()}})
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary Get usage metrics
// @Tags Cloud
// @Produce json
// @Param id path string true "Tenant ID"
// @Param period query string false "Period (YYYY-MM)"
// @Success 200 {object} UsageMetrics
// @Router /cloud/tenants/{id}/usage [get]
func (h *Handler) GetUsage(c *gin.Context) {
	tenantID := c.Param("id")
	period := c.Query("period")

	usage, err := h.service.GetUsage(c.Request.Context(), tenantID, period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "USAGE_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, usage)
}

// @Summary Get usage history
// @Tags Cloud
// @Produce json
// @Param id path string true "Tenant ID"
// @Param limit query int false "Months of history" default(12)
// @Success 200 {object} map[string][]UsageMetrics
// @Router /cloud/tenants/{id}/usage/history [get]
func (h *Handler) GetUsageHistory(c *gin.Context) {
	tenantID := c.Param("id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "12"))

	history, err := h.service.GetUsageHistory(c.Request.Context(), tenantID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "HISTORY_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"usage_history": history})
}

// @Summary Get scaling configuration
// @Tags Cloud
// @Produce json
// @Param id path string true "Tenant ID"
// @Success 200 {object} ScalingConfig
// @Router /cloud/tenants/{id}/scaling [get]
func (h *Handler) GetScalingConfig(c *gin.Context) {
	tenantID := c.Param("id")

	config, err := h.service.GetScalingConfig(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, config)
}

// @Summary Update scaling configuration
// @Tags Cloud
// @Accept json
// @Produce json
// @Param id path string true "Tenant ID"
// @Param body body UpdateScalingRequest true "Scaling config"
// @Success 200 {object} ScalingConfig
// @Router /cloud/tenants/{id}/scaling [put]
func (h *Handler) UpdateScalingConfig(c *gin.Context) {
	tenantID := c.Param("id")

	var req UpdateScalingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	config, err := h.service.UpdateScalingConfig(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "UPDATE_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, config)
}

// @Summary Get cloud platform dashboard
// @Tags Cloud
// @Produce json
// @Success 200 {object} CloudDashboard
// @Router /cloud/dashboard [get]
func (h *Handler) GetDashboard(c *gin.Context) {
	dashboard, err := h.service.GetDashboard(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "DASHBOARD_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

// @Summary Get available regions
// @Tags Cloud
// @Produce json
// @Success 200 {object} map[string][]string
// @Router /cloud/regions [get]
func (h *Handler) GetRegions(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"regions": h.service.GetAvailableRegions()})
}

// @Summary Get available plans
// @Tags Cloud
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /cloud/plans [get]
func (h *Handler) GetPlans(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"plans": h.service.GetAvailablePlans()})
}
