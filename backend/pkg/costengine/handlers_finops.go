package costengine

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"

	"github.com/gin-gonic/gin"
)

// FinOpsHandler provides HTTP handlers for the FinOps engine
type FinOpsHandler struct {
	service *FinOpsService
}

// NewFinOpsHandler creates a new handler
func NewFinOpsHandler(service *FinOpsService) *FinOpsHandler {
	return &FinOpsHandler{service: service}
}

// RegisterFinOpsRoutes registers FinOps routes
func (h *FinOpsHandler) RegisterFinOpsRoutes(r *gin.RouterGroup) {
	finops := r.Group("/finops")
	{
		finops.GET("/dashboard", h.GetDashboard)
		finops.POST("/budget", h.CreateBudget)
		finops.GET("/budget", h.GetBudget)
		finops.DELETE("/budget", h.DeleteBudget)
		finops.POST("/record", h.RecordDeliveryCost)
	}
}

// GetDashboard retrieves the FinOps dashboard
// @Summary Get FinOps dashboard
// @Tags finops
// @Produce json
// @Success 200 {object} FinOpsDashboard
// @Router /finops/dashboard [get]
func (h *FinOpsHandler) GetDashboard(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	dashboard, err := h.service.GetDashboard(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

// CreateBudget creates a budget
// @Summary Create budget
// @Tags finops
// @Accept json
// @Produce json
// @Success 201 {object} Budget
// @Router /finops/budget [post]
func (h *FinOpsHandler) CreateBudget(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateFinOpsBudgetRequest
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

// GetBudget retrieves a budget
// @Summary Get budget
// @Tags finops
// @Produce json
// @Success 200 {object} Budget
// @Router /finops/budget [get]
func (h *FinOpsHandler) GetBudget(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	budget, err := h.service.GetBudget(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "budget not found"})
		return
	}

	c.JSON(http.StatusOK, budget)
}

// DeleteBudget deletes a budget
// @Summary Delete budget
// @Tags finops
// @Success 204 "No content"
// @Router /finops/budget [delete]
func (h *FinOpsHandler) DeleteBudget(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.service.DeleteBudget(c.Request.Context(), tenantID); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// RecordDeliveryCost records a delivery cost
// @Summary Record delivery cost
// @Tags finops
// @Accept json
// @Produce json
// @Success 201 {object} CostAttribution
// @Router /finops/record [post]
func (h *FinOpsHandler) RecordDeliveryCost(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		EndpointID string `json:"endpoint_id" binding:"required"`
		EventType  string `json:"event_type" binding:"required"`
		Region     string `json:"region,omitempty"`
		BytesOut   int64  `json:"bytes_out" binding:"required"`
		RetryCount int    `json:"retry_count"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	attr, err := h.service.RecordDeliveryCost(c.Request.Context(), tenantID, req.EndpointID, req.EventType, req.Region, req.BytesOut, req.RetryCount)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, attr)
}
