package cloudmanaged

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	pkgerrors "webhook-platform/pkg/errors"
)

// Handler handles cloud managed HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new cloud managed handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers cloud managed routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	cloud := r.Group("/cloud")
	{
		// Signup & tenant
		cloud.POST("/signup", h.Signup)
		cloud.GET("/tenant", h.GetTenant)
		cloud.PUT("/tenant", h.UpdateTenant)

		// Plans
		cloud.GET("/plans", h.GetPlans)
		cloud.POST("/upgrade", h.UpgradePlan)
		cloud.POST("/downgrade", h.DowngradePlan)

		// Usage
		cloud.GET("/usage", h.GetUsageSummary)
		cloud.GET("/usage/history", h.GetUsageHistory)

		// Billing
		cloud.GET("/billing", h.GetBillingInfo)
		cloud.PUT("/billing", h.UpdateBillingInfo)

		// Onboarding
		cloud.GET("/onboarding", h.GetOnboardingProgress)
		cloud.POST("/onboarding/:stepId/complete", h.CompleteOnboardingStep)

		// Isolation & Quotas
		cloud.GET("/isolation", h.GetTenantIsolation)
		cloud.GET("/quota", h.GetQuotaStatus)

		// Auto-scaling
		cloud.GET("/autoscale", h.GetAutoScaleConfig)

		// SLA Monitoring
		cloud.GET("/sla", h.GetSLAStatus)

		// Status Page (public-facing)
		cloud.GET("/status", h.GetStatusPage)

		// Regional Deployments
		cloud.GET("/regions", h.GetRegionalDeployments)

		// Tenant lifecycle
		cloud.POST("/suspend", h.SuspendTenant)
		cloud.POST("/reactivate", h.ReactivateTenant)

		// Stripe webhook
		cloud.POST("/stripe-webhook", h.HandleStripeWebhook)

		// Trial management
		cloud.GET("/trial", h.GetTrialStatus)

		// Invoice generation
		cloud.GET("/invoice", h.CalculateInvoice)

		// Admin: list tenants
		cloud.GET("/tenants", h.ListTenants)
	}
}

// Signup handles self-service signup
// @Summary Self-service signup
// @Tags CloudManaged
// @Accept json
// @Produce json
// @Param request body SignupRequest true "Signup details"
// @Success 201 {object} CloudTenant
// @Router /cloud/signup [post]
func (h *Handler) Signup(c *gin.Context) {
	var req SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	tenant, err := h.service.Signup(c.Request.Context(), &req)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusCreated, tenant)
}

// GetTenant retrieves the current tenant
// @Summary Get tenant details
// @Tags CloudManaged
// @Produce json
// @Success 200 {object} CloudTenant
// @Router /cloud/tenant [get]
func (h *Handler) GetTenant(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	tenant, err := h.service.GetTenant(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.AbortWithNotFound(c, "tenant")
		return
	}

	c.JSON(http.StatusOK, tenant)
}

// UpdateTenant updates the current tenant
// @Summary Update tenant details
// @Tags CloudManaged
// @Accept json
// @Produce json
// @Success 200 {object} CloudTenant
// @Router /cloud/tenant [put]
func (h *Handler) UpdateTenant(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	tenant, err := h.service.GetTenant(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.AbortWithNotFound(c, "tenant")
		return
	}

	if err := c.ShouldBindJSON(tenant); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	updated, err := h.service.UpdateTenant(c.Request.Context(), tenant)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, updated)
}

// GetPlans returns all available plans
// @Summary List available plans
// @Tags CloudManaged
// @Produce json
// @Success 200 {array} PlanDefinition
// @Router /cloud/plans [get]
func (h *Handler) GetPlans(c *gin.Context) {
	plans := h.service.GetAvailablePlans(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"plans": plans})
}

// UpgradePlan upgrades the tenant plan
// @Summary Upgrade plan
// @Tags CloudManaged
// @Accept json
// @Produce json
// @Param request body UpgradePlanRequest true "Upgrade details"
// @Success 200 {object} CloudTenant
// @Router /cloud/upgrade [post]
func (h *Handler) UpgradePlan(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req UpgradePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	tenant, err := h.service.UpgradePlan(c.Request.Context(), tenantID, PlanTier(req.Plan))
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, tenant)
}

// DowngradePlan downgrades the tenant plan
// @Summary Downgrade plan
// @Tags CloudManaged
// @Accept json
// @Produce json
// @Param request body UpgradePlanRequest true "Downgrade details"
// @Success 200 {object} CloudTenant
// @Router /cloud/downgrade [post]
func (h *Handler) DowngradePlan(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req UpgradePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	tenant, err := h.service.DowngradePlan(c.Request.Context(), tenantID, PlanTier(req.Plan))
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, tenant)
}

// GetUsageSummary returns usage summary for the current period
// @Summary Get usage summary
// @Tags CloudManaged
// @Produce json
// @Param period query string false "Period (YYYY-MM)"
// @Success 200 {object} UsageSummary
// @Router /cloud/usage [get]
func (h *Handler) GetUsageSummary(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	period := c.DefaultQuery("period", time.Now().Format("2006-01"))

	summary, err := h.service.GetUsageSummary(c.Request.Context(), tenantID, period)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	withinQuota, _ := h.service.CheckQuota(c.Request.Context(), tenantID)
	c.JSON(http.StatusOK, gin.H{
		"usage":        summary,
		"within_quota": withinQuota,
	})
}

// GetUsageHistory returns recent usage history
// @Summary Get usage history
// @Tags CloudManaged
// @Produce json
// @Param limit query int false "Limit" default(100)
// @Success 200 {array} UsageMeter
// @Router /cloud/usage/history [get]
func (h *Handler) GetUsageHistory(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	limit := 100

	summary, err := h.service.repo.GetUsageHistory(c.Request.Context(), tenantID, limit)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"history": summary})
}

// GetBillingInfo returns billing information
// @Summary Get billing info
// @Tags CloudManaged
// @Produce json
// @Success 200 {object} BillingInfo
// @Router /cloud/billing [get]
func (h *Handler) GetBillingInfo(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	info, err := h.service.GetBillingInfo(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.AbortWithNotFound(c, "billing info")
		return
	}

	c.JSON(http.StatusOK, info)
}

// UpdateBillingInfo updates billing information
// @Summary Update billing info
// @Tags CloudManaged
// @Accept json
// @Produce json
// @Param request body UpdateBillingRequest true "Billing update"
// @Success 200 {object} BillingInfo
// @Router /cloud/billing [put]
func (h *Handler) UpdateBillingInfo(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req UpdateBillingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	info, err := h.service.UpdateBillingInfo(c.Request.Context(), tenantID, &req)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, info)
}

// GetOnboardingProgress returns onboarding progress
// @Summary Get onboarding progress
// @Tags CloudManaged
// @Produce json
// @Success 200 {object} OnboardingProgress
// @Router /cloud/onboarding [get]
func (h *Handler) GetOnboardingProgress(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	progress, err := h.service.GetOnboardingProgress(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.AbortWithNotFound(c, "onboarding progress")
		return
	}

	c.JSON(http.StatusOK, progress)
}

// CompleteOnboardingStep marks an onboarding step as completed
// @Summary Complete onboarding step
// @Tags CloudManaged
// @Produce json
// @Param stepId path string true "Step ID"
// @Success 200 {object} OnboardingProgress
// @Router /cloud/onboarding/{stepId}/complete [post]
func (h *Handler) CompleteOnboardingStep(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	stepID := c.Param("stepId")

	progress, err := h.service.CompleteOnboardingStep(c.Request.Context(), tenantID, stepID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, progress)
}

func (h *Handler) GetTenantIsolation(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	isolation, err := h.service.GetTenantIsolation(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, isolation)
}

func (h *Handler) GetQuotaStatus(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	status, err := h.service.GetQuotaStatus(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, status)
}

func (h *Handler) GetAutoScaleConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	config, err := h.service.GetAutoScaleConfig(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, config)
}

func (h *Handler) GetSLAStatus(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	sla, err := h.service.GetSLAStatus(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, sla)
}

func (h *Handler) GetStatusPage(c *gin.Context) {
	page := h.service.GetStatusPage(c.Request.Context())
	c.JSON(http.StatusOK, page)
}

func (h *Handler) GetRegionalDeployments(c *gin.Context) {
	regions := h.service.GetRegionalDeployments(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"regions": regions})
}

func (h *Handler) SuspendTenant(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	tenant, err := h.service.SuspendTenant(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, tenant)
}

func (h *Handler) ReactivateTenant(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	tenant, err := h.service.ReactivateTenant(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, tenant)
}

// HandleStripeWebhook processes incoming Stripe webhook events
func (h *Handler) HandleStripeWebhook(c *gin.Context) {
	var event StripeWebhookEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event"})
		return
	}

	if err := h.service.HandleStripeWebhook(c.Request.Context(), &event); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}

// GetTrialStatus returns trial status
// @Summary Get trial status
// @Tags CloudManaged
// @Produce json
// @Success 200 {object} TrialStatus
// @Router /cloud/trial [get]
func (h *Handler) GetTrialStatus(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	status, err := h.service.GetTrialStatus(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, status)
}

// CalculateInvoice returns the current period invoice
// @Summary Calculate invoice
// @Tags CloudManaged
// @Produce json
// @Success 200 {object} Invoice
// @Router /cloud/invoice [get]
func (h *Handler) CalculateInvoice(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	invoice, err := h.service.CalculateInvoice(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, invoice)
}

// ListTenants returns paginated cloud tenants
// @Summary List cloud tenants (admin)
// @Tags CloudManaged
// @Produce json
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} CloudTenant
// @Router /cloud/tenants [get]
func (h *Handler) ListTenants(c *gin.Context) {
	limit := 50
	offset := 0
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := c.Query("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}
	tenants, err := h.service.ListTenants(c.Request.Context(), limit, offset)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"tenants": tenants, "limit": limit, "offset": offset})
}
