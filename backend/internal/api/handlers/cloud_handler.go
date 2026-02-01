package handlers

import (
	"net/http"
	"strconv"

	"github.com/josedab/waas/pkg/cloud"
	"github.com/josedab/waas/pkg/utils"

	"github.com/gin-gonic/gin"
)

// CloudHandler handles cloud billing and team management HTTP requests
type CloudHandler struct {
	billingService *cloud.BillingService
	teamService    *cloud.TeamService
	auditService   *cloud.AuditService
	repo           cloud.Repository
	logger         *utils.Logger
}

// NewCloudHandler creates a new cloud handler
func NewCloudHandler(
	billingService *cloud.BillingService,
	teamService *cloud.TeamService,
	auditService *cloud.AuditService,
	repo cloud.Repository,
	logger *utils.Logger,
) *CloudHandler {
	return &CloudHandler{
		billingService: billingService,
		teamService:    teamService,
		auditService:   auditService,
		repo:           repo,
		logger:         logger,
	}
}

// CreateSubscriptionRequest represents subscription creation
type CreateSubscriptionRequest struct {
	PlanID       string `json:"plan_id" binding:"required"`
	BillingCycle string `json:"billing_cycle" binding:"required"`
}

// ChangePlanRequest represents plan change request
type ChangePlanRequest struct {
	PlanID string `json:"plan_id" binding:"required"`
}

// InviteMemberRequest represents team member invitation
type InviteMemberRequest struct {
	Email string `json:"email" binding:"required"`
	Name  string `json:"name" binding:"required"`
	Role  string `json:"role" binding:"required"`
}

// UpdateMemberRequest represents team member update
type UpdateMemberRequest struct {
	Role string `json:"role" binding:"required"`
}

// UpdateCustomerRequest represents customer info update
type UpdateCustomerRequest struct {
	Email        string `json:"email"`
	Name         string `json:"name"`
	Company      string `json:"company"`
	AddressLine1 string `json:"address_line1"`
	AddressLine2 string `json:"address_line2"`
	City         string `json:"city"`
	State        string `json:"state"`
	PostalCode   string `json:"postal_code"`
	Country      string `json:"country"`
	TaxID        string `json:"tax_id"`
}

// GetPlans lists available plans
// @Summary List available plans
// @Tags billing
// @Produce json
// @Success 200 {array} cloud.Plan
// @Router /billing/plans [get]
func (h *CloudHandler) GetPlans(c *gin.Context) {
	plans := cloud.GetAllPlans()
	c.JSON(http.StatusOK, plans)
}

// GetPlan gets a specific plan
// @Summary Get plan details
// @Tags billing
// @Produce json
// @Param id path string true "Plan ID"
// @Success 200 {object} cloud.Plan
// @Router /billing/plans/{id} [get]
func (h *CloudHandler) GetPlan(c *gin.Context) {
	id := c.Param("id")

	plan := cloud.GetPlanByID(id)
	if plan == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}

	c.JSON(http.StatusOK, plan)
}

// GetSubscription gets current subscription
// @Summary Get current subscription
// @Tags billing
// @Produce json
// @Success 200 {object} cloud.Subscription
// @Router /billing/subscription [get]
func (h *CloudHandler) GetSubscription(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	sub, err := h.billingService.GetSubscription(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}

	c.JSON(http.StatusOK, sub)
}

// CreateSubscription creates a new subscription
// @Summary Create subscription
// @Tags billing
// @Accept json
// @Produce json
// @Param request body CreateSubscriptionRequest true "Subscription request"
// @Success 201 {object} cloud.Subscription
// @Router /billing/subscription [post]
func (h *CloudHandler) CreateSubscription(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sub, err := h.billingService.CreateSubscription(c.Request.Context(), tenantID, req.PlanID, cloud.BillingCycle(req.BillingCycle))
	if err != nil {
		h.logger.Error("Failed to create subscription", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, sub)
}

// ChangePlan changes the subscription plan
// @Summary Change plan
// @Tags billing
// @Accept json
// @Produce json
// @Param request body ChangePlanRequest true "Plan change request"
// @Success 200 {object} cloud.Subscription
// @Router /billing/subscription/plan [patch]
func (h *CloudHandler) ChangePlan(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req ChangePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sub, err := h.billingService.ChangePlan(c.Request.Context(), tenantID, req.PlanID)
	if err != nil {
		h.logger.Error("Failed to change plan", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sub)
}

// CancelSubscription cancels the subscription
// @Summary Cancel subscription
// @Tags billing
// @Produce json
// @Param immediate query bool false "Cancel immediately"
// @Success 200 {object} cloud.Subscription
// @Router /billing/subscription [delete]
func (h *CloudHandler) CancelSubscription(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	immediate := c.Query("immediate") == "true"

	sub, err := h.billingService.CancelSubscription(c.Request.Context(), tenantID, immediate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sub)
}

// GetUsage gets current usage
// @Summary Get current usage
// @Tags billing
// @Produce json
// @Param period query string false "Period (YYYY-MM)"
// @Success 200 {object} cloud.UsageRecord
// @Router /billing/usage [get]
func (h *CloudHandler) GetUsage(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	period := c.Query("period")
	if period == "" {
		// Current month
		usage, err := h.billingService.GetSubscription(c.Request.Context(), tenantID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "usage not found"})
			return
		}
		c.JSON(http.StatusOK, usage)
		return
	}

	usage, err := h.repo.GetUsage(c.Request.Context(), tenantID, period)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "usage not found"})
		return
	}

	c.JSON(http.StatusOK, usage)
}

// ListUsageHistory lists usage history
// @Summary List usage history
// @Tags billing
// @Produce json
// @Param limit query int false "Limit"
// @Success 200 {array} cloud.UsageRecord
// @Router /billing/usage/history [get]
func (h *CloudHandler) ListUsageHistory(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "12"))

	history, err := h.repo.ListUsage(c.Request.Context(), tenantID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, history)
}

// ListInvoices lists invoices
// @Summary List invoices
// @Tags billing
// @Produce json
// @Param limit query int false "Limit"
// @Success 200 {array} cloud.Invoice
// @Router /billing/invoices [get]
func (h *CloudHandler) ListInvoices(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "24"))

	invoices, err := h.repo.ListInvoices(c.Request.Context(), tenantID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, invoices)
}

// GetInvoice gets an invoice
// @Summary Get invoice
// @Tags billing
// @Produce json
// @Param id path string true "Invoice ID"
// @Success 200 {object} cloud.Invoice
// @Router /billing/invoices/{id} [get]
func (h *CloudHandler) GetInvoice(c *gin.Context) {
	id := c.Param("id")

	invoice, err := h.repo.GetInvoice(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
		return
	}

	c.JSON(http.StatusOK, invoice)
}

// GetCustomer gets customer billing info
// @Summary Get customer info
// @Tags billing
// @Produce json
// @Success 200 {object} cloud.Customer
// @Router /billing/customer [get]
func (h *CloudHandler) GetCustomer(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	customer, err := h.repo.GetCustomer(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "customer not found"})
		return
	}

	c.JSON(http.StatusOK, customer)
}

// UpdateCustomer updates customer billing info
// @Summary Update customer info
// @Tags billing
// @Accept json
// @Produce json
// @Param request body UpdateCustomerRequest true "Customer update"
// @Success 200 {object} cloud.Customer
// @Router /billing/customer [patch]
func (h *CloudHandler) UpdateCustomer(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req UpdateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	customer, err := h.repo.GetCustomer(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "customer not found"})
		return
	}

	if req.Email != "" {
		customer.Email = req.Email
	}
	if req.Name != "" {
		customer.Name = req.Name
	}
	if req.Company != "" {
		customer.Company = req.Company
	}
	if req.AddressLine1 != "" {
		customer.AddressLine1 = req.AddressLine1
	}
	if req.AddressLine2 != "" {
		customer.AddressLine2 = req.AddressLine2
	}
	if req.City != "" {
		customer.City = req.City
	}
	if req.State != "" {
		customer.State = req.State
	}
	if req.PostalCode != "" {
		customer.PostalCode = req.PostalCode
	}
	if req.Country != "" {
		customer.Country = req.Country
	}
	if req.TaxID != "" {
		customer.TaxID = req.TaxID
	}

	if err := h.repo.UpdateCustomer(c.Request.Context(), customer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, customer)
}

// ListPaymentMethods lists payment methods
// @Summary List payment methods
// @Tags billing
// @Produce json
// @Success 200 {array} cloud.PaymentMethod
// @Router /billing/payment-methods [get]
func (h *CloudHandler) ListPaymentMethods(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	methods, err := h.repo.ListPaymentMethods(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, methods)
}

// DeletePaymentMethod deletes a payment method
// @Summary Delete payment method
// @Tags billing
// @Param id path string true "Payment method ID"
// @Success 204
// @Router /billing/payment-methods/{id} [delete]
func (h *CloudHandler) DeletePaymentMethod(c *gin.Context) {
	id := c.Param("id")

	if err := h.repo.DeletePaymentMethod(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// SetDefaultPaymentMethod sets the default payment method
// @Summary Set default payment method
// @Tags billing
// @Param id path string true "Payment method ID"
// @Success 200 {object} map[string]interface{}
// @Router /billing/payment-methods/{id}/default [post]
func (h *CloudHandler) SetDefaultPaymentMethod(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	id := c.Param("id")

	if err := h.repo.SetDefaultPaymentMethod(c.Request.Context(), tenantID, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "default payment method updated"})
}

// Team Management

// ListTeamMembers lists team members
// @Summary List team members
// @Tags team
// @Produce json
// @Success 200 {array} cloud.TeamMember
// @Router /team/members [get]
func (h *CloudHandler) ListTeamMembers(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	members, err := h.teamService.ListMembers(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, members)
}

// InviteMember invites a new team member
// @Summary Invite team member
// @Tags team
// @Accept json
// @Produce json
// @Param request body InviteMemberRequest true "Invitation request"
// @Success 201 {object} cloud.TeamMember
// @Router /team/members [post]
func (h *CloudHandler) InviteMember(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req InviteMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	member, err := h.teamService.InviteMember(c.Request.Context(), tenantID, userID, req.Email, req.Name, cloud.TeamRole(req.Role))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, member)
}

// UpdateMember updates a team member's role
// @Summary Update team member
// @Tags team
// @Accept json
// @Produce json
// @Param id path string true "Member ID"
// @Param request body UpdateMemberRequest true "Update request"
// @Success 200 {object} cloud.TeamMember
// @Router /team/members/{id} [patch]
func (h *CloudHandler) UpdateMember(c *gin.Context) {
	id := c.Param("id")

	var req UpdateMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.teamService.UpdateMemberRole(c.Request.Context(), id, cloud.TeamRole(req.Role)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	member, _ := h.repo.GetTeamMember(c.Request.Context(), id)
	c.JSON(http.StatusOK, member)
}

// RemoveMember removes a team member
// @Summary Remove team member
// @Tags team
// @Param id path string true "Member ID"
// @Success 204
// @Router /team/members/{id} [delete]
func (h *CloudHandler) RemoveMember(c *gin.Context) {
	id := c.Param("id")

	if err := h.teamService.RemoveMember(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// Audit Logs

// ListAuditLogs lists audit logs
// @Summary List audit logs
// @Tags audit
// @Produce json
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Success 200 {array} cloud.AuditLog
// @Router /audit/logs [get]
func (h *CloudHandler) ListAuditLogs(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	logs, err := h.auditService.ListLogs(c.Request.Context(), tenantID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, logs)
}

// RegisterCloudRoutes registers cloud billing and team routes
func RegisterCloudRoutes(r *gin.RouterGroup, h *CloudHandler) {
	// Billing routes
	billing := r.Group("/billing")
	{
		billing.GET("/plans", h.GetPlans)
		billing.GET("/plans/:id", h.GetPlan)
		billing.GET("/subscription", h.GetSubscription)
		billing.POST("/subscription", h.CreateSubscription)
		billing.PATCH("/subscription/plan", h.ChangePlan)
		billing.DELETE("/subscription", h.CancelSubscription)
		billing.GET("/usage", h.GetUsage)
		billing.GET("/usage/history", h.ListUsageHistory)
		billing.GET("/invoices", h.ListInvoices)
		billing.GET("/invoices/:id", h.GetInvoice)
		billing.GET("/customer", h.GetCustomer)
		billing.PATCH("/customer", h.UpdateCustomer)
		billing.GET("/payment-methods", h.ListPaymentMethods)
		billing.DELETE("/payment-methods/:id", h.DeletePaymentMethod)
		billing.POST("/payment-methods/:id/default", h.SetDefaultPaymentMethod)
	}

	// Team routes
	team := r.Group("/team")
	{
		team.GET("/members", h.ListTeamMembers)
		team.POST("/members", h.InviteMember)
		team.PATCH("/members/:id", h.UpdateMember)
		team.DELETE("/members/:id", h.RemoveMember)
	}

	// Audit routes
	audit := r.Group("/audit")
	{
		audit.GET("/logs", h.ListAuditLogs)
	}
}
