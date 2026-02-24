package costing

import (
	"github.com/josedab/waas/pkg/httputil"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for cost attribution
type Handler struct {
	service *Service
}

// NewHandler creates a new costing handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers costing routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	costs := router.Group("/costs")
	{
		costs.GET("/report", h.GetCostReport)
		costs.GET("/forecast", h.GetForecast)
		costs.GET("/usage", h.GetUsageStats)
		costs.GET("/allocations/:resourceType/:resourceId", h.GetAllocation)
		costs.GET("/top-endpoints", h.GetTopEndpoints)
	}

	budgets := router.Group("/budgets")
	{
		budgets.POST("", h.CreateBudget)
		budgets.GET("", h.ListBudgets)
		budgets.GET("/:id", h.GetBudget)
		budgets.PUT("/:id", h.UpdateBudget)
		budgets.DELETE("/:id", h.DeleteBudget)
		budgets.POST("/check-alerts", h.CheckAlerts)
	}
}

// GetCostReport godoc
//
//	@Summary		Get cost report
//	@Description	Get a cost report for the specified period
//	@Tags			costs
//	@Produce		json
//	@Param			period	query		string	false	"Period (YYYY-MM)"	default(current month)
//	@Success		200		{object}	CostReport
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/costs/report [get]
func (h *Handler) GetCostReport(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	period := c.DefaultQuery("period", time.Now().Format("2006-01"))

	report, err := h.service.GetCostReport(c.Request.Context(), tenantID, period)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, report)
}

// GetForecast godoc
//
//	@Summary		Get cost forecast
//	@Description	Get a cost forecast for the current period
//	@Tags			costs
//	@Produce		json
//	@Success		200	{object}	CostForecast
//	@Failure		401	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/costs/forecast [get]
func (h *Handler) GetForecast(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	forecast, err := h.service.GetForecast(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, forecast)
}

// GetUsageStats godoc
//
//	@Summary		Get usage statistics
//	@Description	Get usage statistics for a time range
//	@Tags			costs
//	@Produce		json
//	@Param			start	query		string	false	"Start date (YYYY-MM-DD)"
//	@Param			end		query		string	false	"End date (YYYY-MM-DD)"
//	@Success		200		{object}	UsageSummary
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/costs/usage [get]
func (h *Handler) GetUsageStats(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	startStr := c.DefaultQuery("start", time.Now().AddDate(0, -1, 0).Format("2006-01-02"))
	endStr := c.DefaultQuery("end", time.Now().Format("2006-01-02"))

	startDate, _ := time.Parse("2006-01-02", startStr)
	endDate, _ := time.Parse("2006-01-02", endStr)

	stats, err := h.service.GetUsageStats(c.Request.Context(), tenantID, startDate, endDate)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetAllocation godoc
//
//	@Summary		Get cost allocation
//	@Description	Get cost allocation for a specific resource
//	@Tags			costs
//	@Produce		json
//	@Param			resourceType	path		string	true	"Resource type (endpoint, customer)"
//	@Param			resourceId		path		string	true	"Resource ID"
//	@Param			period			query		string	false	"Period (YYYY-MM)"	default(current month)
//	@Success		200				{object}	CostAllocation
//	@Failure		401				{object}	map[string]interface{}
//	@Failure		404				{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/costs/allocations/{resourceType}/{resourceId} [get]
func (h *Handler) GetAllocation(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	resourceType := c.Param("resourceType")
	resourceID := c.Param("resourceId")
	period := c.DefaultQuery("period", time.Now().Format("2006-01"))

	allocation, err := h.service.GetCostAllocation(c.Request.Context(), tenantID, period, resourceType, resourceID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	if allocation == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "allocation not found"})
		return
	}

	c.JSON(http.StatusOK, allocation)
}

// GetTopEndpoints godoc
//
//	@Summary		Get top endpoints by cost
//	@Description	Get the top endpoints by cost for a period
//	@Tags			costs
//	@Produce		json
//	@Param			period	query		string	false	"Period (YYYY-MM)"	default(current month)
//	@Param			limit	query		int		false	"Limit"				default(10)
//	@Success		200		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/costs/top-endpoints [get]
func (h *Handler) GetTopEndpoints(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	period := c.DefaultQuery("period", time.Now().Format("2006-01"))
	limit := 10
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	endpoints, err := h.service.GetTopEndpoints(c.Request.Context(), tenantID, period, limit)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"endpoints": endpoints})
}

// CreateBudget godoc
//
//	@Summary		Create budget
//	@Description	Create a new budget
//	@Tags			budgets
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateBudgetRequest	true	"Budget request"
//	@Success		201		{object}	Budget
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/budgets [post]
func (h *Handler) CreateBudget(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateBudgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	budget, err := h.service.CreateBudget(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, budget)
}

// ListBudgets godoc
//
//	@Summary		List budgets
//	@Description	Get a list of budgets
//	@Tags			budgets
//	@Produce		json
//	@Param			limit	query		int	false	"Limit"		default(20)
//	@Param			offset	query		int	false	"Offset"	default(0)
//	@Success		200		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/budgets [get]
func (h *Handler) ListBudgets(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := c.Query("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	budgets, total, err := h.service.ListBudgets(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"budgets": budgets,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// GetBudget godoc
//
//	@Summary		Get budget
//	@Description	Get budget details
//	@Tags			budgets
//	@Produce		json
//	@Param			id	path		string	true	"Budget ID"
//	@Success		200	{object}	Budget
//	@Failure		401	{object}	map[string]interface{}
//	@Failure		404	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/budgets/{id} [get]
func (h *Handler) GetBudget(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	budgetID := c.Param("id")
	budget, err := h.service.GetBudget(c.Request.Context(), tenantID, budgetID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	if budget == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "budget not found"})
		return
	}

	c.JSON(http.StatusOK, budget)
}

// UpdateBudget godoc
//
//	@Summary		Update budget
//	@Description	Update a budget
//	@Tags			budgets
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Budget ID"
//	@Param			request	body		UpdateBudgetRequest	true	"Update request"
//	@Success		200		{object}	Budget
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/budgets/{id} [put]
func (h *Handler) UpdateBudget(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	budgetID := c.Param("id")
	var req UpdateBudgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	budget, err := h.service.UpdateBudget(c.Request.Context(), tenantID, budgetID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, budget)
}

// DeleteBudget godoc
//
//	@Summary		Delete budget
//	@Description	Delete a budget
//	@Tags			budgets
//	@Produce		json
//	@Param			id	path	string	true	"Budget ID"
//	@Success		204	"No content"
//	@Failure		401	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/budgets/{id} [delete]
func (h *Handler) DeleteBudget(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	budgetID := c.Param("id")
	if err := h.service.DeleteBudget(c.Request.Context(), tenantID, budgetID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// CheckAlerts godoc
//
//	@Summary		Check budget alerts
//	@Description	Check and return budgets that have exceeded alert thresholds
//	@Tags			budgets
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Failure		401	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/budgets/check-alerts [post]
func (h *Handler) CheckAlerts(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	alertedBudgets, err := h.service.CheckBudgetAlerts(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"alerted_budgets": alertedBudgets,
		"count":           len(alertedBudgets),
	})
}
