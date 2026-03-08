package billing

import (
	"encoding/json"
	"fmt"
	"github.com/josedab/waas/pkg/httputil"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles billing HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new billing handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers billing routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	billing := r.Group("/billing")
	{
		// Pricing Plans
		billing.GET("/plans", h.ListPricingPlans)
		billing.POST("/plans", h.CreatePricingPlan)

		// Subscriptions
		billing.POST("/subscribe", h.Subscribe)
		billing.GET("/subscription", h.GetSubscription)
		billing.PUT("/subscription", h.ChangeSubscription)
		billing.DELETE("/subscription", h.CancelSubscription)

		// Usage
		billing.GET("/usage", h.GetUsageSummary)
		billing.GET("/usage/:resource_type", h.GetUsageByResource)
		billing.GET("/spend", h.GetCurrentSpend)
		billing.GET("/forecast", h.ForecastSpend)

		// Invoices
		billing.GET("/invoices", h.ListInvoices)
		billing.GET("/invoices/:id", h.GetInvoice)

		// Dashboard & Projection
		billing.GET("/dashboard", h.GetDashboard)
		billing.GET("/projection", h.GetProjection)

		// Budgets
		billing.POST("/budgets", h.CreateBudget)
		billing.GET("/budgets", h.ListBudgets)
		billing.GET("/budgets/:id", h.GetBudget)
		billing.PUT("/budgets/:id", h.UpdateBudget)
		billing.DELETE("/budgets/:id", h.DeleteBudget)

		// Alerts
		billing.GET("/alerts", h.ListAlerts)
		billing.POST("/alerts/:id/ack", h.AcknowledgeAlert)
		billing.GET("/alerts/config", h.GetAlertConfig)
		billing.PUT("/alerts/config", h.UpdateAlertConfig)

		// Optimizations
		billing.GET("/optimizations", h.GetOptimizations)
		billing.POST("/optimizations/analyze", h.AnalyzeOptimizations)
		billing.POST("/optimizations/:id/implement", h.ImplementOptimization)
		billing.POST("/optimizations/:id/dismiss", h.DismissOptimization)

		// Anomaly detection
		billing.GET("/anomalies", h.DetectAnomaly)
	}

	// Stripe webhooks (public, no auth)
	r.POST("/billing/webhooks/stripe", h.HandleStripeWebhook)
}

// GetUsageSummary godoc
// @Summary Get usage summary
// @Description Get usage summary for current or specified period
// @Tags Billing
// @Accept json
// @Produce json
// @Param period query string false "Billing period (YYYY-MM)"
// @Success 200 {object} CostUsageSummary
// @Router /billing/usage [get]
func (h *Handler) GetUsageSummary(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	period := c.DefaultQuery("period", "")
	if period == "" {
		period = getCurrentPeriod()
	}

	summary, err := h.service.GetUsageSummary(c.Request.Context(), tenantID, period)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, summary)
}

// GetUsageByResource godoc
// @Summary Get usage by resource type
// @Description Get detailed usage for a specific resource type
// @Tags Billing
// @Accept json
// @Produce json
// @Param resource_type path string true "Resource type"
// @Param period query string false "Billing period (YYYY-MM)"
// @Success 200 {array} CostUsageRecord
// @Router /billing/usage/{resource_type} [get]
func (h *Handler) GetUsageByResource(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	resourceType := c.Param("resource_type")
	period := c.DefaultQuery("period", getCurrentPeriod())

	records, err := h.service.repo.GetUsageByResource(c.Request.Context(), tenantID, resourceType, period)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"records": records})
}

// GetCurrentSpend godoc
// @Summary Get current spend
// @Description Get current period spending total
// @Tags Billing
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /billing/spend [get]
func (h *Handler) GetCurrentSpend(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	spend, err := h.service.GetCurrentSpend(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"current_spend": spend,
		"currency":      h.service.pricing.Currency,
		"period":        getCurrentPeriod(),
	})
}

// ForecastSpend godoc
// @Summary Forecast spend
// @Description Forecast spending for upcoming days
// @Tags Billing
// @Produce json
// @Param days query int false "Days to forecast" default(30)
// @Success 200 {object} SpendForecast
// @Router /billing/forecast [get]
func (h *Handler) ForecastSpend(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	days := 30
	if d, err := parseIntParam(c, "days"); err == nil && d > 0 {
		days = d
	}

	forecast, err := h.service.ForecastSpend(c.Request.Context(), tenantID, days)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, forecast)
}

// CreateBudget godoc
// @Summary Create budget
// @Description Create a new budget with alerts
// @Tags Billing
// @Accept json
// @Produce json
// @Param request body CreateBudgetRequest true "Budget configuration"
// @Success 201 {object} BudgetConfig
// @Router /billing/budgets [post]
func (h *Handler) CreateBudget(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req CreateBudgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	budget, err := h.service.CreateBudget(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, budget)
}

// ListBudgets godoc
// @Summary List budgets
// @Description List all budgets for tenant
// @Tags Billing
// @Produce json
// @Success 200 {array} BudgetConfig
// @Router /billing/budgets [get]
func (h *Handler) ListBudgets(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	budgets, err := h.service.ListBudgets(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"budgets": budgets})
}

// GetBudget godoc
// @Summary Get budget
// @Description Get budget by ID
// @Tags Billing
// @Produce json
// @Param id path string true "Budget ID"
// @Success 200 {object} BudgetConfig
// @Router /billing/budgets/{id} [get]
func (h *Handler) GetBudget(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	budgetID := c.Param("id")

	budget, err := h.service.GetBudget(c.Request.Context(), tenantID, budgetID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, budget)
}

// UpdateBudget godoc
// @Summary Update budget
// @Description Update an existing budget
// @Tags Billing
// @Accept json
// @Produce json
// @Param id path string true "Budget ID"
// @Param request body UpdateBudgetRequest true "Budget updates"
// @Success 200 {object} BudgetConfig
// @Router /billing/budgets/{id} [put]
func (h *Handler) UpdateBudget(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	budgetID := c.Param("id")

	var req UpdateBudgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	budget, err := h.service.UpdateBudget(c.Request.Context(), tenantID, budgetID, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, budget)
}

// DeleteBudget godoc
// @Summary Delete budget
// @Description Delete a budget
// @Tags Billing
// @Param id path string true "Budget ID"
// @Success 204
// @Router /billing/budgets/{id} [delete]
func (h *Handler) DeleteBudget(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	budgetID := c.Param("id")

	if err := h.service.DeleteBudget(c.Request.Context(), tenantID, budgetID); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ListAlerts godoc
// @Summary List alerts
// @Description List billing alerts
// @Tags Billing
// @Produce json
// @Param status query string false "Filter by status"
// @Success 200 {array} BillingAlert
// @Router /billing/alerts [get]
func (h *Handler) ListAlerts(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var status *AlertStatus
	if s := c.Query("status"); s != "" {
		st := AlertStatus(s)
		status = &st
	}

	alerts, err := h.service.GetAlerts(c.Request.Context(), tenantID, status)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"alerts": alerts})
}

// AcknowledgeAlert godoc
// @Summary Acknowledge alert
// @Description Acknowledge a billing alert
// @Tags Billing
// @Param id path string true "Alert ID"
// @Success 200 {object} map[string]interface{}
// @Router /billing/alerts/{id}/ack [post]
func (h *Handler) AcknowledgeAlert(c *gin.Context) {
	alertID := c.Param("id")
	userID := c.GetString("user_id")

	if err := h.service.AcknowledgeAlert(c.Request.Context(), alertID, userID); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "acknowledged"})
}

// GetAlertConfig godoc
// @Summary Get alert configuration
// @Description Get alert notification configuration
// @Tags Billing
// @Produce json
// @Success 200 {object} AlertConfig
// @Router /billing/alerts/config [get]
func (h *Handler) GetAlertConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	config, err := h.service.GetAlertConfig(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, config)
}

// UpdateAlertConfig godoc
// @Summary Update alert configuration
// @Description Update alert notification settings
// @Tags Billing
// @Accept json
// @Produce json
// @Param request body UpdateAlertConfigRequest true "Alert config"
// @Success 200 {object} AlertConfig
// @Router /billing/alerts/config [put]
func (h *Handler) UpdateAlertConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req UpdateAlertConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := h.service.UpdateAlertConfig(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, config)
}

// GetOptimizations godoc
// @Summary Get optimizations
// @Description Get cost optimization recommendations
// @Tags Billing
// @Produce json
// @Success 200 {array} CostOptimization
// @Router /billing/optimizations [get]
func (h *Handler) GetOptimizations(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	opts, err := h.service.GetOptimizations(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"optimizations": opts})
}

// AnalyzeOptimizations godoc
// @Summary Analyze optimizations
// @Description Analyze usage and find optimization opportunities
// @Tags Billing
// @Produce json
// @Success 200 {array} CostOptimization
// @Router /billing/optimizations/analyze [post]
func (h *Handler) AnalyzeOptimizations(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	opts, err := h.service.AnalyzeOptimizations(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"optimizations": opts})
}

// ImplementOptimization godoc
// @Summary Implement optimization
// @Description Mark optimization as implemented
// @Tags Billing
// @Param id path string true "Optimization ID"
// @Success 200 {object} map[string]interface{}
// @Router /billing/optimizations/{id}/implement [post]
func (h *Handler) ImplementOptimization(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	optID := c.Param("id")

	if err := h.service.ImplementOptimization(c.Request.Context(), tenantID, optID); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "implemented"})
}

// DismissOptimization godoc
// @Summary Dismiss optimization
// @Description Dismiss an optimization recommendation
// @Tags Billing
// @Param id path string true "Optimization ID"
// @Success 200 {object} map[string]interface{}
// @Router /billing/optimizations/{id}/dismiss [post]
func (h *Handler) DismissOptimization(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	optID := c.Param("id")

	if err := h.service.DismissOptimization(c.Request.Context(), tenantID, optID); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "dismissed"})
}

// ListInvoices godoc
// @Summary List invoices
// @Description List all invoices for tenant
// @Tags Billing
// @Produce json
// @Success 200 {array} Invoice
// @Router /billing/invoices [get]
func (h *Handler) ListInvoices(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	invoices, err := h.service.GetInvoices(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"invoices": invoices})
}

// GetInvoice godoc
// @Summary Get invoice
// @Description Get invoice by ID
// @Tags Billing
// @Produce json
// @Param id path string true "Invoice ID"
// @Success 200 {object} Invoice
// @Router /billing/invoices/{id} [get]
func (h *Handler) GetInvoice(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	invoiceID := c.Param("id")

	invoice, err := h.service.GetInvoice(c.Request.Context(), tenantID, invoiceID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, invoice)
}

// DetectAnomaly godoc
// @Summary Detect spending anomaly
// @Description Check for anomalous spending patterns
// @Tags Billing
// @Produce json
// @Success 200 {object} SpendAnomaly
// @Router /billing/anomalies [get]
func (h *Handler) DetectAnomaly(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	anomaly, err := h.service.DetectSpendAnomaly(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	if anomaly == nil {
		c.JSON(http.StatusOK, gin.H{"anomaly_detected": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"anomaly_detected": true,
		"anomaly":          anomaly,
	})
}

// Helpers

func getCurrentPeriod() string {
	return getCurrentTime().Format("2006-01")
}

func getCurrentTime() time.Time {
	return time.Now()
}

func parseIntParam(c *gin.Context, name string) (int, error) {
	val := c.Query(name)
	if val == "" {
		return 0, nil
	}
	var i int
	_, err := fmt.Sscanf(val, "%d", &i)
	return i, err
}

// --- Subscription Billing Handlers ---

// ListPricingPlans lists available pricing plans
// @Summary List pricing plans
// @Tags Billing
// @Produce json
// @Success 200 {array} PricingPlan
// @Router /billing/plans [get]
func (h *Handler) ListPricingPlans(c *gin.Context) {
	plans, err := h.service.ListPricingPlans(c.Request.Context())
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"plans": plans})
}

// CreatePricingPlan creates a new pricing plan (admin)
// @Summary Create pricing plan
// @Tags Billing
// @Accept json
// @Produce json
// @Param request body PricingPlan true "Plan details"
// @Success 201 {object} PricingPlan
// @Router /billing/plans [post]
func (h *Handler) CreatePricingPlan(c *gin.Context) {
	var plan PricingPlan
	if err := c.ShouldBindJSON(&plan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.CreatePricingPlan(c.Request.Context(), &plan)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusCreated, result)
}

// SubscribeRequest represents a subscription request
type SubscribeRequest struct {
	PlanID string `json:"plan_id" binding:"required"`
}

// Subscribe subscribes a tenant to a plan
// @Summary Subscribe to plan
// @Tags Billing
// @Accept json
// @Produce json
// @Param request body SubscribeRequest true "Subscription request"
// @Success 201 {object} Subscription
// @Router /billing/subscribe [post]
func (h *Handler) Subscribe(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	planID, err := uuid.Parse(req.PlanID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan_id"})
		return
	}

	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}
	sub, err := h.service.CreateSubscriptionForTenant(c.Request.Context(), tid, planID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusCreated, sub)
}

// GetSubscription gets the current subscription
// @Summary Get subscription
// @Tags Billing
// @Produce json
// @Success 200 {object} Subscription
// @Router /billing/subscription [get]
func (h *Handler) GetSubscription(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}

	sub, err := h.service.GetSubscriptionForTenant(c.Request.Context(), tid)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, sub)
}

// ChangeSubscription changes the subscription plan
// @Summary Change subscription plan
// @Tags Billing
// @Accept json
// @Produce json
// @Param request body SubscribeRequest true "New plan"
// @Success 200 {object} Subscription
// @Router /billing/subscription [put]
func (h *Handler) ChangeSubscription(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	planID, err := uuid.Parse(req.PlanID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan_id"})
		return
	}

	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}
	sub, err := h.service.ChangeSubscription(c.Request.Context(), tid, planID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, sub)
}

// CancelSubscription cancels the subscription
// @Summary Cancel subscription
// @Tags Billing
// @Produce json
// @Success 200
// @Router /billing/subscription [delete]
func (h *Handler) CancelSubscription(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}

	if err := h.service.CancelSubscriptionForTenant(c.Request.Context(), tid); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
}

// GetDashboard returns billing dashboard data
// @Summary Get billing dashboard
// @Tags Billing
// @Produce json
// @Success 200 {object} BillingDashboard
// @Router /billing/dashboard [get]
func (h *Handler) GetDashboard(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}

	dashboard, err := h.service.GetBillingDashboard(c.Request.Context(), tid)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, dashboard)
}

// GetProjection returns cost projection
// @Summary Get cost projection
// @Tags Billing
// @Produce json
// @Param days query int false "Days to project" default(30)
// @Success 200 {object} UsageSummary
// @Router /billing/projection [get]
func (h *Handler) GetProjection(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}
	days := 30
	if d, err := strconv.Atoi(c.Query("days")); err == nil && d > 0 {
		days = d
	}

	projection, err := h.service.ProjectCost(c.Request.Context(), tid, days)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, projection)
}

// HandleStripeWebhook processes Stripe webhook events (public, no auth)
// @Summary Handle Stripe webhook
// @Tags Billing
// @Accept json
// @Produce json
// @Success 200
// @Router /billing/webhooks/stripe [post]
func (h *Handler) HandleStripeWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read body"})
		return
	}

	var event struct {
		Type string                 `json:"type"`
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	if err := h.service.HandleStripeWebhook(c.Request.Context(), event.Type, event.Data); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"received": true})
}

// Ensure imports are used
var (
	_ = strconv.Atoi
	_ = json.Unmarshal
	_ = io.ReadAll
	_ = time.Now
	_ = uuid.New
)
