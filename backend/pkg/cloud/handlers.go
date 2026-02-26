package cloud

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP endpoints for the WaaS Cloud managed service
type Handler struct {
	billing *BillingService
	team    *TeamService
	audit   *AuditService
	onboard *OnboardingService
}

// NewHandler creates a new cloud handler
func NewHandler(billing *BillingService, team *TeamService, audit *AuditService, onboard *OnboardingService) *Handler {
	return &Handler{
		billing: billing,
		team:    team,
		audit:   audit,
		onboard: onboard,
	}
}

// RegisterRoutes registers cloud management HTTP routes
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	cloud := rg.Group("/cloud")
	{
		// Plans
		cloud.GET("/plans", h.ListPlans)
		cloud.GET("/plans/:id", h.GetPlan)

		// Subscription management
		cloud.POST("/subscriptions", h.CreateSubscription)
		cloud.GET("/subscriptions", h.GetSubscription)
		cloud.PUT("/subscriptions/plan", h.ChangePlan)
		cloud.POST("/subscriptions/cancel", h.CancelSubscription)

		// Usage & billing
		cloud.GET("/usage", h.GetUsage)
		cloud.GET("/usage/limits", h.GetPlanLimits)
		cloud.GET("/invoices", h.ListInvoices)

		// Team management
		cloud.POST("/team/members", h.InviteMember)
		cloud.GET("/team/members", h.ListMembers)
		cloud.PUT("/team/members/:id/role", h.UpdateMemberRole)
		cloud.DELETE("/team/members/:id", h.RemoveMember)

		// Audit logs
		cloud.GET("/audit-logs", h.ListAuditLogs)

		// Onboarding
		cloud.POST("/onboarding/start", h.StartOnboarding)
		cloud.POST("/onboarding/:session_id/step", h.SubmitOnboardingStep)
		cloud.GET("/onboarding/:session_id", h.GetOnboardingStatus)
	}
}

// ListPlans returns all available subscription plans
// @Summary List available plans
// @Tags Cloud
// @Produce json
// @Success 200 {array} Plan
// @Router /api/v1/cloud/plans [get]
func (h *Handler) ListPlans(c *gin.Context) {
	plans := make([]*Plan, 0)
	for _, p := range AvailablePlans {
		if p.IsActive {
			plans = append(plans, p)
		}
	}
	c.JSON(http.StatusOK, gin.H{"plans": plans})
}

// GetPlan returns a specific plan
// @Summary Get plan details
// @Tags Cloud
// @Produce json
// @Param id path string true "Plan ID"
// @Success 200 {object} Plan
// @Router /api/v1/cloud/plans/{id} [get]
func (h *Handler) GetPlan(c *gin.Context) {
	plan := GetPlanByID(c.Param("id"))
	if plan == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}
	c.JSON(http.StatusOK, plan)
}

type createSubscriptionRequest struct {
	PlanID       string       `json:"plan_id" binding:"required"`
	BillingCycle BillingCycle `json:"billing_cycle" binding:"required"`
}

// CreateSubscription creates a new subscription for the tenant
// @Summary Create subscription
// @Tags Cloud
// @Accept json
// @Produce json
// @Param body body createSubscriptionRequest true "Subscription details"
// @Success 201 {object} Subscription
// @Router /api/v1/cloud/subscriptions [post]
func (h *Handler) CreateSubscription(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req createSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.billing == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "billing service unavailable"})
		return
	}

	sub, err := h.billing.CreateSubscription(c.Request.Context(), tenantID, req.PlanID, req.BillingCycle)
	if err != nil {
		switch err {
		case ErrPlanNotFound:
			c.JSON(http.StatusBadRequest, gin.H{"error": "plan not found"})
		case ErrPaymentRequired:
			c.JSON(http.StatusPaymentRequired, gin.H{"error": "payment required"})
		default:
			httputil.InternalErrorGeneric(c, err)
		}
		return
	}

	c.JSON(http.StatusCreated, sub)
}

// GetSubscription returns the current subscription
// @Summary Get current subscription
// @Tags Cloud
// @Produce json
// @Success 200 {object} Subscription
// @Router /api/v1/cloud/subscriptions [get]
func (h *Handler) GetSubscription(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if h.billing == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "billing service unavailable"})
		return
	}

	sub, err := h.billing.GetSubscription(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active subscription"})
		return
	}

	plan := GetPlanByID(sub.PlanID)
	c.JSON(http.StatusOK, gin.H{
		"subscription": sub,
		"plan":         plan,
	})
}

type changePlanRequest struct {
	PlanID string `json:"plan_id" binding:"required"`
}

// ChangePlan changes the subscription plan
// @Summary Change subscription plan
// @Tags Cloud
// @Accept json
// @Produce json
// @Param body body changePlanRequest true "New plan"
// @Success 200 {object} Subscription
// @Router /api/v1/cloud/subscriptions/plan [put]
func (h *Handler) ChangePlan(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req changePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.billing == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "billing service unavailable"})
		return
	}

	sub, err := h.billing.ChangePlan(c.Request.Context(), tenantID, req.PlanID)
	if err != nil {
		switch err {
		case ErrPlanNotFound:
			c.JSON(http.StatusBadRequest, gin.H{"error": "plan not found"})
		case ErrSubscriptionNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		case ErrUsageLimitExceeded:
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "usage limit exceeded"})
		case ErrInvalidPlanUpgrade:
			c.JSON(http.StatusConflict, gin.H{"error": "invalid plan upgrade"})
		case ErrPaymentRequired:
			c.JSON(http.StatusPaymentRequired, gin.H{"error": "payment required"})
		default:
			httputil.InternalErrorGeneric(c, err)
		}
		return
	}

	c.JSON(http.StatusOK, sub)
}

type cancelRequest struct {
	Immediate bool `json:"immediate"`
}

// CancelSubscription cancels the current subscription
// @Summary Cancel subscription
// @Tags Cloud
// @Accept json
// @Produce json
// @Success 200 {object} Subscription
// @Router /api/v1/cloud/subscriptions/cancel [post]
func (h *Handler) CancelSubscription(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req cancelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	if h.billing == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "billing service unavailable"})
		return
	}

	sub, err := h.billing.CancelSubscription(c.Request.Context(), tenantID, req.Immediate)
	if err != nil {
		switch err {
		case ErrPlanNotFound:
			c.JSON(http.StatusBadRequest, gin.H{"error": "plan not found"})
		case ErrSubscriptionNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		case ErrUsageLimitExceeded:
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "usage limit exceeded"})
		case ErrInvalidPlanUpgrade:
			c.JSON(http.StatusConflict, gin.H{"error": "invalid plan upgrade"})
		case ErrPaymentRequired:
			c.JSON(http.StatusPaymentRequired, gin.H{"error": "payment required"})
		default:
			httputil.InternalErrorGeneric(c, err)
		}
		return
	}

	c.JSON(http.StatusOK, sub)
}

// GetUsage returns current usage for the tenant
// @Summary Get current usage
// @Tags Cloud
// @Produce json
// @Success 200 {object} UsageRecord
// @Router /api/v1/cloud/usage [get]
func (h *Handler) GetUsage(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if h.billing == nil || h.billing.usageTracker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "usage tracking unavailable"})
		return
	}

	usage, err := h.billing.usageTracker.GetCurrentUsage(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"tenant_id": tenantID,
			"period":    time.Now().Format("2006-01"),
			"usage":     UsageRecord{},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tenant_id": tenantID,
		"period":    time.Now().Format("2006-01"),
		"usage":     usage,
	})
}

// GetPlanLimits returns plan limits for the tenant
// @Summary Get plan limits
// @Tags Cloud
// @Produce json
// @Success 200 {object} PlanLimits
// @Router /api/v1/cloud/usage/limits [get]
func (h *Handler) GetPlanLimits(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if h.billing == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "billing service unavailable"})
		return
	}

	limits, err := h.billing.GetPlanLimits(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"limits": limits})
}

// ListInvoices lists invoices for the tenant
// @Summary List invoices
// @Tags Cloud
// @Produce json
// @Param limit query int false "Limit" default(20)
// @Success 200 {array} Invoice
// @Router /api/v1/cloud/invoices [get]
func (h *Handler) ListInvoices(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if h.billing == nil || h.billing.repo == nil {
		c.JSON(http.StatusOK, gin.H{"invoices": []Invoice{}})
		return
	}

	invoices, err := h.billing.repo.ListInvoices(c.Request.Context(), tenantID, limit)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"invoices": invoices})
}

type inviteMemberRequest struct {
	Email string   `json:"email" binding:"required"`
	Name  string   `json:"name" binding:"required"`
	Role  TeamRole `json:"role" binding:"required"`
}

// InviteMember invites a new team member
// @Summary Invite team member
// @Tags Cloud
// @Accept json
// @Produce json
// @Success 201 {object} TeamMember
// @Router /api/v1/cloud/team/members [post]
func (h *Handler) InviteMember(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req inviteMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.team == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "team service unavailable"})
		return
	}

	member, err := h.team.InviteMember(c.Request.Context(), tenantID, "system", req.Email, req.Name, req.Role)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, member)
}

// ListMembers lists team members
// @Summary List team members
// @Tags Cloud
// @Produce json
// @Success 200 {array} TeamMember
// @Router /api/v1/cloud/team/members [get]
func (h *Handler) ListMembers(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if h.team == nil {
		c.JSON(http.StatusOK, gin.H{"members": []TeamMember{}})
		return
	}

	members, err := h.team.ListMembers(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"members": members})
}

type updateRoleRequest struct {
	Role TeamRole `json:"role" binding:"required"`
}

// UpdateMemberRole updates a member's role
// @Summary Update member role
// @Tags Cloud
// @Accept json
// @Produce json
// @Param id path string true "Member ID"
// @Success 200 {object} map[string]string
// @Router /api/v1/cloud/team/members/{id}/role [put]
func (h *Handler) UpdateMemberRole(c *gin.Context) {
	var req updateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.team == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "team service unavailable"})
		return
	}

	if err := h.team.UpdateMemberRole(c.Request.Context(), c.Param("id"), req.Role); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

// RemoveMember removes a team member
// @Summary Remove team member
// @Tags Cloud
// @Param id path string true "Member ID"
// @Success 204
// @Router /api/v1/cloud/team/members/{id} [delete]
func (h *Handler) RemoveMember(c *gin.Context) {
	if h.team == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "team service unavailable"})
		return
	}

	if err := h.team.RemoveMember(c.Request.Context(), c.Param("id")); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ListAuditLogs lists audit log entries
// @Summary List audit logs
// @Tags Cloud
// @Produce json
// @Param page query int false "Page" default(1)
// @Param page_size query int false "Page size" default(50)
// @Success 200 {array} AuditLog
// @Router /api/v1/cloud/audit-logs [get]
func (h *Handler) ListAuditLogs(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	if h.audit == nil {
		c.JSON(http.StatusOK, gin.H{"logs": []AuditLog{}, "page": page})
		return
	}

	logs, err := h.audit.ListLogs(c.Request.Context(), tenantID, page, pageSize)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs, "page": page, "page_size": pageSize})
}

// StartOnboarding begins the self-service onboarding flow
// @Summary Start onboarding
// @Tags Cloud
// @Accept json
// @Produce json
// @Success 201 {object} OnboardingSession
// @Router /api/v1/cloud/onboarding/start [post]
func (h *Handler) StartOnboarding(c *gin.Context) {
	if h.onboard == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "onboarding service unavailable"})
		return
	}

	var req struct {
		Email          string            `json:"email" binding:"required"`
		ReferralSource string            `json:"referral_source"`
		UTMParams      map[string]string `json:"utm_params"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, token, err := h.onboard.StartOnboarding(
		c.Request.Context(),
		req.Email,
		c.ClientIP(),
		c.GetHeader("User-Agent"),
		req.ReferralSource,
		req.UTMParams,
	)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"session":            session,
		"verification_token": token,
	})
}

type onboardingStepRequest struct {
	Step string                 `json:"step" binding:"required"`
	Data map[string]interface{} `json:"data" binding:"required"`
}

// SubmitOnboardingStep submits a step in the onboarding flow
// @Summary Submit onboarding step
// @Tags Cloud
// @Accept json
// @Produce json
// @Param session_id path string true "Session ID"
// @Success 200 {object} OnboardingSession
// @Router /api/v1/cloud/onboarding/{session_id}/step [post]
func (h *Handler) SubmitOnboardingStep(c *gin.Context) {
	if h.onboard == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "onboarding service unavailable"})
		return
	}

	var req onboardingStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sessionID := c.Param("session_id")
	ctx := c.Request.Context()
	var result interface{}
	var err error

	switch req.Step {
	case "verify_email":
		token, _ := req.Data["token"].(string)
		result, err = h.onboard.VerifyEmail(ctx, sessionID, token)
	case "organization":
		name, _ := req.Data["name"].(string)
		result, _, err = h.onboard.SetupOrganization(ctx, sessionID, name)
	case "plan":
		planID, _ := req.Data["plan_id"].(string)
		result, err = h.onboard.SelectPlan(ctx, sessionID, planID)
	case "payment":
		paymentMethodID, _ := req.Data["payment_method_id"].(string)
		result, err = h.onboard.CompletePayment(ctx, sessionID, paymentMethodID)
	case "complete":
		result, err = h.onboard.CompleteOnboarding(ctx, sessionID)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown step: " + req.Step})
		return
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetOnboardingStatus gets the current onboarding status
// @Summary Get onboarding status
// @Tags Cloud
// @Produce json
// @Param session_id path string true "Session ID"
// @Success 200 {object} OnboardingSession
// @Router /api/v1/cloud/onboarding/{session_id} [get]
func (h *Handler) GetOnboardingStatus(c *gin.Context) {
	if h.onboard == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "onboarding service unavailable"})
		return
	}

	session, err := h.onboard.GetSession(c.Request.Context(), c.Param("session_id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	c.JSON(http.StatusOK, session)
}

// status maps a cloud service error to an HTTP status code.
func status(err error) int {
	switch err {
	case ErrPlanNotFound:
		return http.StatusBadRequest
	case ErrSubscriptionNotFound:
		return http.StatusNotFound
	case ErrUsageLimitExceeded:
		return http.StatusTooManyRequests
	case ErrInvalidPlanUpgrade:
		return http.StatusConflict
	case ErrPaymentRequired:
		return http.StatusPaymentRequired
	default:
		return http.StatusInternalServerError
	}
}

