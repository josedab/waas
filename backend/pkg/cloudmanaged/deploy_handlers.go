package cloudmanaged

import (
	"net/http"

	"github.com/gin-gonic/gin"
	pkgerrors "github.com/josedab/waas/pkg/errors"
)

// OneClickDeployRequest represents a one-click deployment request
type OneClickDeployRequest struct {
	Plan   string `json:"plan" binding:"required"`
	Region string `json:"region,omitempty"`
}

// RegisterDeployRoutes registers one-click deployment and control plane routes
func (h *Handler) RegisterDeployRoutes(r *gin.RouterGroup) {
	cloud := r.Group("/cloud")
	{
		cloud.POST("/deploy", h.OneClickDeploy)
		cloud.GET("/deploy/:id/status", h.GetDeploymentStatus)
		cloud.GET("/control-plane", h.GetControlPlaneConfig)
		cloud.GET("/health", h.GetDeploymentHealth)
		cloud.GET("/usage-dashboard", h.GetUsageDashboard)
		cloud.GET("/ops-dashboard", h.GetOperationsDashboard)
		cloud.GET("/sla-report", h.GetSLAReport)
		cloud.GET("/namespace-config", h.GetTenantNamespaceConfig)
	}
}

// OneClickDeploy handles one-click deployment requests
// @Summary One-click deploy a WaaS instance
// @Tags CloudManaged
// @Accept json
// @Produce json
// @Param request body OneClickDeployRequest true "Deployment config"
// @Success 201 {object} OneClickDeployment
// @Router /cloud/deploy [post]
func (h *Handler) OneClickDeploy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req OneClickDeployRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	deployment, err := h.service.OneClickDeploy(c.Request.Context(), tenantID, PlanTier(req.Plan), req.Region)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusCreated, deployment)
}

// GetDeploymentStatus returns the status of an active deployment
// @Summary Get deployment status
// @Tags CloudManaged
// @Produce json
// @Param id path string true "Deployment ID"
// @Success 200 {object} OneClickDeployment
// @Router /cloud/deploy/{id}/status [get]
func (h *Handler) GetDeploymentStatus(c *gin.Context) {
	deploymentID := c.Param("id")

	deployment, err := h.service.GetDeploymentStatus(c.Request.Context(), deploymentID)
	if err != nil {
		pkgerrors.AbortWithNotFound(c, "deployment")
		return
	}

	c.JSON(http.StatusOK, deployment)
}

// GetControlPlaneConfig returns the control plane configuration
// @Summary Get control plane configuration
// @Tags CloudManaged
// @Produce json
// @Success 200 {object} ControlPlaneConfig
// @Router /cloud/control-plane [get]
func (h *Handler) GetControlPlaneConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	config, err := h.service.GetControlPlaneConfig(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, config)
}

// GetDeploymentHealth returns real-time deployment health
// @Summary Get deployment health
// @Tags CloudManaged
// @Produce json
// @Success 200 {object} DeploymentHealth
// @Router /cloud/health [get]
func (h *Handler) GetDeploymentHealth(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	health, err := h.service.GetDeploymentHealth(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, health)
}

// GetUsageDashboard returns the tenant usage dashboard
// @Summary Get usage dashboard
// @Tags CloudManaged
// @Produce json
// @Success 200 {object} UsageDashboard
// @Router /cloud/usage-dashboard [get]
func (h *Handler) GetUsageDashboard(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	dashboard, err := h.service.GetUsageDashboard(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

// GetOperationsDashboard returns the SaaS operations dashboard (admin)
// @Summary Get operations dashboard
// @Tags CloudManaged
// @Produce json
// @Success 200 {object} OperationsDashboard
// @Router /cloud/ops-dashboard [get]
func (h *Handler) GetOperationsDashboard(c *gin.Context) {
	dashboard, err := h.service.GetOperationsDashboard(c.Request.Context())
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

// GetSLAReport returns the SLA compliance report
// @Summary Get SLA report
// @Tags CloudManaged
// @Produce json
// @Success 200 {object} SLAReport
// @Router /cloud/sla-report [get]
func (h *Handler) GetSLAReport(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	report, err := h.service.GetSLAReport(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, report)
}

// GetTenantNamespaceConfig returns K8s namespace configuration
// @Summary Get tenant namespace config
// @Tags CloudManaged
// @Produce json
// @Success 200 {object} TenantNamespaceConfig
// @Router /cloud/namespace-config [get]
func (h *Handler) GetTenantNamespaceConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	config, err := h.service.GetTenantNamespaceConfig(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, config)
}
