package monetization

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for monetization
type Handler struct {
	service *Service
}

// NewHandler creates a new handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers HTTP routes
func (h *Handler) RegisterRoutes(r gin.IRouter) {
	monetization := r.Group("/monetization")
	{
		// Plans
		monetization.POST("/plans", h.CreatePlan)
		monetization.GET("/plans", h.ListPlans)
		monetization.GET("/plans/:id", h.GetPlan)
		monetization.PUT("/plans/:id", h.UpdatePlan)

		// Customers
		monetization.POST("/customers", h.CreateCustomer)
		monetization.GET("/customers", h.ListCustomers)
		monetization.GET("/customers/:id", h.GetCustomer)
		monetization.GET("/customers/:id/dashboard", h.GetDashboard)

		// Subscriptions
		monetization.POST("/customers/:id/subscribe", h.Subscribe)
		monetization.DELETE("/subscriptions/:id", h.CancelSubscription)

		// API Keys
		monetization.POST("/customers/:id/api-keys", h.CreateAPIKey)
		monetization.GET("/customers/:id/api-keys", h.ListAPIKeys)
		monetization.DELETE("/api-keys/:id", h.RevokeAPIKey)

		// Usage & Billing
		monetization.GET("/subscriptions/:id/usage", h.GetUsage)
		monetization.POST("/subscriptions/:id/invoice", h.GenerateInvoice)
		monetization.GET("/customers/:id/invoices", h.ListInvoices)

		// Validate API key (for webhook processing)
		monetization.POST("/validate-key", h.ValidateAPIKey)
	}
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// CreatePlan creates a new pricing plan
// @Summary Create a pricing plan
// @Tags monetization
// @Accept json
// @Produce json
// @Param request body Plan true "Plan details"
// @Success 201 {object} Plan
// @Router /monetization/plans [post]
func (h *Handler) CreatePlan(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	var plan Plan
	if err := c.ShouldBindJSON(&plan); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	result, err := h.service.CreatePlan(c.Request.Context(), tenantID, &plan)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// ListPlans lists all plans
// @Summary List pricing plans
// @Tags monetization
// @Produce json
// @Param public query bool false "Only public plans"
// @Success 200 {array} Plan
// @Router /monetization/plans [get]
func (h *Handler) ListPlans(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	publicOnly, _ := strconv.ParseBool(c.Query("public"))

	plans, err := h.service.ListPlans(c.Request.Context(), tenantID, publicOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, plans)
}

// GetPlan gets a specific plan
// @Summary Get a pricing plan
// @Tags monetization
// @Produce json
// @Param id path string true "Plan ID"
// @Success 200 {object} Plan
// @Router /monetization/plans/{id} [get]
func (h *Handler) GetPlan(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	plan, err := h.service.GetPlan(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, plan)
}

// UpdatePlan updates a plan
// @Summary Update a pricing plan
// @Tags monetization
// @Accept json
// @Produce json
// @Param id path string true "Plan ID"
// @Param request body Plan true "Plan details"
// @Success 200 {object} Plan
// @Router /monetization/plans/{id} [put]
func (h *Handler) UpdatePlan(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	var plan Plan
	if err := c.ShouldBindJSON(&plan); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	plan.ID = c.Param("id")
	plan.TenantID = tenantID

	if err := h.service.repo.UpdatePlan(c.Request.Context(), &plan); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, plan)
}

// CreateCustomer creates a new customer
// @Summary Create a customer
// @Tags monetization
// @Accept json
// @Produce json
// @Param request body Customer true "Customer details"
// @Success 201 {object} Customer
// @Router /monetization/customers [post]
func (h *Handler) CreateCustomer(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	var customer Customer
	if err := c.ShouldBindJSON(&customer); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	result, err := h.service.CreateCustomer(c.Request.Context(), tenantID, &customer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// ListCustomers lists all customers
// @Summary List customers
// @Tags monetization
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} object{customers=[]Customer,total=int}
// @Router /monetization/customers [get]
func (h *Handler) ListCustomers(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	customers, total, err := h.service.repo.ListCustomers(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"customers": customers,
		"total":     total,
	})
}

// GetCustomer gets a specific customer
// @Summary Get a customer
// @Tags monetization
// @Produce json
// @Param id path string true "Customer ID"
// @Success 200 {object} Customer
// @Router /monetization/customers/{id} [get]
func (h *Handler) GetCustomer(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	customer, err := h.service.GetCustomer(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, customer)
}

// GetDashboard gets the usage dashboard for a customer
// @Summary Get customer dashboard
// @Tags monetization
// @Produce json
// @Param id path string true "Customer ID"
// @Success 200 {object} UsageDashboard
// @Router /monetization/customers/{id}/dashboard [get]
func (h *Handler) GetDashboard(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	dashboard, err := h.service.GetUsageDashboard(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

// SubscribeRequest represents a subscription request
type SubscribeRequest struct {
	PlanID string `json:"plan_id" binding:"required"`
}

// Subscribe subscribes a customer to a plan
// @Summary Subscribe customer to plan
// @Tags monetization
// @Accept json
// @Produce json
// @Param id path string true "Customer ID"
// @Param request body SubscribeRequest true "Subscription request"
// @Success 201 {object} CustomerSubscription
// @Router /monetization/customers/{id}/subscribe [post]
func (h *Handler) Subscribe(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	var req SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	sub, err := h.service.SubscribeToPlan(c.Request.Context(), tenantID, c.Param("id"), req.PlanID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, sub)
}

// CancelSubscription cancels a subscription
// @Summary Cancel subscription
// @Tags monetization
// @Produce json
// @Param id path string true "Subscription ID"
// @Param immediately query bool false "Cancel immediately"
// @Success 200
// @Router /monetization/subscriptions/{id} [delete]
func (h *Handler) CancelSubscription(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	immediately, _ := strconv.ParseBool(c.Query("immediately"))

	if err := h.service.CancelSubscription(c.Request.Context(), tenantID, c.Param("id"), immediately); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
}

// CreateAPIKeyRequest represents an API key creation request
type CreateAPIKeyRequest struct {
	Name   string   `json:"name" binding:"required"`
	Scopes []string `json:"scopes"`
}

// CreateAPIKey creates an API key for a customer
// @Summary Create API key
// @Tags monetization
// @Accept json
// @Produce json
// @Param id path string true "Customer ID"
// @Param request body CreateAPIKeyRequest true "API key request"
// @Success 201 {object} object{key=APIKey,secret=string}
// @Router /monetization/customers/{id}/api-keys [post]
func (h *Handler) CreateAPIKey(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	key, secret, err := h.service.CreateAPIKey(c.Request.Context(), tenantID, c.Param("id"), req.Name, req.Scopes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"key":    key,
		"secret": secret, // Only shown once
	})
}

// ListAPIKeys lists API keys for a customer
// @Summary List API keys
// @Tags monetization
// @Produce json
// @Param id path string true "Customer ID"
// @Success 200 {array} APIKey
// @Router /monetization/customers/{id}/api-keys [get]
func (h *Handler) ListAPIKeys(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	keys, err := h.service.repo.ListAPIKeys(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, keys)
}

// RevokeAPIKey revokes an API key
// @Summary Revoke API key
// @Tags monetization
// @Produce json
// @Param id path string true "API Key ID"
// @Success 200
// @Router /monetization/api-keys/{id} [delete]
func (h *Handler) RevokeAPIKey(c *gin.Context) {
	if err := h.service.repo.DeleteAPIKey(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "revoked"})
}

// GetUsage gets usage for a subscription
// @Summary Get subscription usage
// @Tags monetization
// @Produce json
// @Param id path string true "Subscription ID"
// @Success 200 {object} UsageRecord
// @Router /monetization/subscriptions/{id}/usage [get]
func (h *Handler) GetUsage(c *gin.Context) {
	sub, err := h.service.repo.GetSubscription(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	usage, err := h.service.repo.GetUsageRecord(c.Request.Context(), sub.ID, sub.CurrentPeriodStart)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, usage)
}

// GenerateInvoice generates an invoice for a subscription
// @Summary Generate invoice
// @Tags monetization
// @Produce json
// @Param id path string true "Subscription ID"
// @Success 201 {object} Invoice
// @Router /monetization/subscriptions/{id}/invoice [post]
func (h *Handler) GenerateInvoice(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	invoice, err := h.service.GenerateInvoice(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, invoice)
}

// ListInvoices lists invoices for a customer
// @Summary List invoices
// @Tags monetization
// @Produce json
// @Param id path string true "Customer ID"
// @Param limit query int false "Limit"
// @Success 200 {array} Invoice
// @Router /monetization/customers/{id}/invoices [get]
func (h *Handler) ListInvoices(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	invoices, err := h.service.repo.ListInvoices(c.Request.Context(), c.Param("id"), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, invoices)
}

// ValidateKeyRequest represents an API key validation request
type ValidateKeyRequest struct {
	APIKey string `json:"api_key" binding:"required"`
}

// ValidateAPIKey validates an API key
// @Summary Validate API key
// @Tags monetization
// @Accept json
// @Produce json
// @Param request body ValidateKeyRequest true "API key to validate"
// @Success 200 {object} APIKey
// @Router /monetization/validate-key [post]
func (h *Handler) ValidateAPIKey(c *gin.Context) {
	var req ValidateKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	key, err := h.service.ValidateAPIKey(c.Request.Context(), req.APIKey)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid API key"})
		return
	}

	c.JSON(http.StatusOK, key)
}
