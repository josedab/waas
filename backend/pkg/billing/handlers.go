package billing

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
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
		// Usage
		billing.GET("/usage", h.GetUsageSummary)
		billing.GET("/usage/:resource_type", h.GetUsageByResource)
		billing.GET("/spend", h.GetCurrentSpend)
		billing.GET("/forecast", h.ForecastSpend)

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

		// Invoices
		billing.GET("/invoices", h.ListInvoices)
		billing.GET("/invoices/:id", h.GetInvoice)

		// Anomaly detection
		billing.GET("/anomalies", h.DetectAnomaly)
	}
}

// GetUsageSummary godoc
// @Summary Get usage summary
// @Description Get usage summary for current or specified period
// @Tags Billing
// @Accept json
// @Produce json
// @Param period query string false "Billing period (YYYY-MM)"
// @Success 200 {object} UsageSummary
// @Router /billing/usage [get]
func (h *Handler) GetUsageSummary(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	period := c.DefaultQuery("period", "")
	if period == "" {
		period = getCurrentPeriod()
	}

	summary, err := h.service.GetUsageSummary(c.Request.Context(), tenantID, period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
// @Success 200 {array} UsageRecord
// @Router /billing/usage/{resource_type} [get]
func (h *Handler) GetUsageByResource(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	resourceType := c.Param("resource_type")
	period := c.DefaultQuery("period", getCurrentPeriod())

	records, err := h.service.repo.GetUsageByResource(c.Request.Context(), tenantID, resourceType, period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
