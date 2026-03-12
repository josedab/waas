package costengine

import (
	"net/http"

	"github.com/josedab/waas/pkg/httputil"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for cost attribution
type Handler struct {
	service *Service
}

// NewHandler creates a new cost engine handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers cost engine routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	costs := router.Group("/costs")
	{
		// Cost models
		costs.POST("/models", h.CreateCostModel)
		costs.GET("/models", h.ListCostModels)
		costs.GET("/models/:id", h.GetCostModel)
		costs.PUT("/models/:id", h.UpdateCostModel)

		// Delivery costs
		costs.POST("/record", h.RecordDeliveryCost)

		// Reports
		costs.POST("/report", h.GenerateReport)

		// Budgets
		costs.POST("/budgets", h.CreateBudget)
		costs.GET("/budgets", h.ListBudgets)
		costs.GET("/budgets/:id", h.GetBudget)
		costs.PUT("/budgets/:id", h.UpdateBudget)
		costs.GET("/budgets/alerts", h.CheckBudgetAlerts)

		// Anomalies
		costs.GET("/anomalies", h.DetectAnomalies)

		// Spend
		costs.GET("/spend/current", h.GetCurrentSpend)

		// Cost Attribution
		costs.POST("/attribution/record", h.RecordCostEvent)
		costs.GET("/attribution/summary", h.GetTenantCostSummary)
		costs.POST("/attribution/chargeback", h.GenerateChargebackReport)
		costs.GET("/attribution/forecast", h.GetCostForecast)
	}
}

// @Summary Create a cost model
// @Tags CostEngine
// @Accept json
// @Produce json
// @Param body body CreateCostModelRequest true "Cost model configuration"
// @Success 201 {object} CostModel
// @Router /costs/models [post]
func (h *Handler) CreateCostModel(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateCostModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httputil.APIErrorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}

	model, err := h.service.CreateCostModel(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalError(c, "CREATE_FAILED", err)
		return
	}

	c.JSON(http.StatusCreated, model)
}

// @Summary List cost models
// @Tags CostEngine
// @Produce json
// @Success 200 {object} map[string][]CostModel
// @Router /costs/models [get]
func (h *Handler) ListCostModels(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	models, err := h.service.ListCostModels(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "LIST_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"models": models})
}

// @Summary Get a cost model
// @Tags CostEngine
// @Produce json
// @Param id path string true "Cost Model ID"
// @Success 200 {object} CostModel
// @Router /costs/models/{id} [get]
func (h *Handler) GetCostModel(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	modelID := c.Param("id")

	model, err := h.service.GetCostModel(c.Request.Context(), tenantID, modelID)
	if err != nil {
		httputil.InternalError(c, "NOT_FOUND", err)
		return
	}

	c.JSON(http.StatusOK, model)
}

// @Summary Update a cost model
// @Tags CostEngine
// @Accept json
// @Produce json
// @Param id path string true "Cost Model ID"
// @Param body body CreateCostModelRequest true "Updated cost model configuration"
// @Success 200 {object} CostModel
// @Router /costs/models/{id} [put]
func (h *Handler) UpdateCostModel(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	modelID := c.Param("id")

	var req CreateCostModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httputil.APIErrorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}

	model, err := h.service.UpdateCostModel(c.Request.Context(), tenantID, modelID, &req)
	if err != nil {
		httputil.InternalError(c, "UPDATE_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, model)
}

// @Summary Record a delivery cost
// @Tags CostEngine
// @Accept json
// @Produce json
// @Param body body RecordDeliveryCostRequest true "Delivery cost details"
// @Success 201 {object} DeliveryCost
// @Router /costs/record [post]
func (h *Handler) RecordDeliveryCost(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req RecordDeliveryCostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httputil.APIErrorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}

	cost, err := h.service.RecordDeliveryCost(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalError(c, "RECORD_FAILED", err)
		return
	}

	c.JSON(http.StatusCreated, cost)
}

// @Summary Generate a cost report
// @Tags CostEngine
// @Accept json
// @Produce json
// @Param body body GenerateReportRequest true "Report period"
// @Success 200 {object} CostReport
// @Router /costs/report [post]
func (h *Handler) GenerateReport(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req GenerateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httputil.APIErrorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}

	report, err := h.service.GenerateReport(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalError(c, "REPORT_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, report)
}

// @Summary Create a cost budget
// @Tags CostEngine
// @Accept json
// @Produce json
// @Param body body CreateBudgetRequest true "Budget configuration"
// @Success 201 {object} CostBudget
// @Router /costs/budgets [post]
func (h *Handler) CreateBudget(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateBudgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httputil.APIErrorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}

	budget, err := h.service.CreateBudget(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalError(c, "CREATE_FAILED", err)
		return
	}

	c.JSON(http.StatusCreated, budget)
}

// @Summary List cost budgets
// @Tags CostEngine
// @Produce json
// @Success 200 {object} map[string][]CostBudget
// @Router /costs/budgets [get]
func (h *Handler) ListBudgets(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	budgets, err := h.service.ListBudgets(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "LIST_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"budgets": budgets})
}

// @Summary Get a cost budget
// @Tags CostEngine
// @Produce json
// @Param id path string true "Budget ID"
// @Success 200 {object} CostBudget
// @Router /costs/budgets/{id} [get]
func (h *Handler) GetBudget(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	budgetID := c.Param("id")

	budget, err := h.service.GetBudget(c.Request.Context(), tenantID, budgetID)
	if err != nil {
		httputil.InternalError(c, "NOT_FOUND", err)
		return
	}

	c.JSON(http.StatusOK, budget)
}

// @Summary Update a cost budget
// @Tags CostEngine
// @Accept json
// @Produce json
// @Param id path string true "Budget ID"
// @Param body body CreateBudgetRequest true "Updated budget configuration"
// @Success 200 {object} CostBudget
// @Router /costs/budgets/{id} [put]
func (h *Handler) UpdateBudget(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	budgetID := c.Param("id")

	var req CreateBudgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httputil.APIErrorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}

	budget, err := h.service.UpdateBudget(c.Request.Context(), tenantID, budgetID, &req)
	if err != nil {
		httputil.InternalError(c, "UPDATE_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, budget)
}

// @Summary Check budget alerts
// @Tags CostEngine
// @Produce json
// @Success 200 {object} map[string][]CostBudget
// @Router /costs/budgets/alerts [get]
func (h *Handler) CheckBudgetAlerts(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	alerts, err := h.service.CheckBudgetAlerts(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "CHECK_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"alerts": alerts, "count": len(alerts)})
}

// @Summary Detect cost anomalies
// @Tags CostEngine
// @Produce json
// @Success 200 {object} map[string][]CostAnomaly
// @Router /costs/anomalies [get]
func (h *Handler) DetectAnomalies(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	anomalies, err := h.service.DetectAnomalies(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "DETECTION_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"anomalies": anomalies, "count": len(anomalies)})
}

// @Summary Get current spend
// @Tags CostEngine
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /costs/spend/current [get]
func (h *Handler) GetCurrentSpend(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	spend, err := h.service.GetCurrentSpend(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "SPEND_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"tenant_id": tenantID, "current_spend": spend, "period": "current_month"})
}
