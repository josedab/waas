package observability

import (
	"net/http"

	"github.com/josedab/waas/pkg/httputil"

	"github.com/gin-gonic/gin"
)

// SuiteHandler provides HTTP handlers for the observability suite
type SuiteHandler struct {
	suite *ObservabilitySuite
}

// NewSuiteHandler creates a new observability suite handler
func NewSuiteHandler(suite *ObservabilitySuite) *SuiteHandler {
	return &SuiteHandler{suite: suite}
}

// RegisterSuiteRoutes registers the unified observability suite routes
func (h *SuiteHandler) RegisterSuiteRoutes(router *gin.RouterGroup) {
	obs := router.Group("/observability-suite")
	{
		// Unified dashboard
		obs.GET("/dashboard", h.GetDashboard)

		// End-to-end delivery traces
		obs.GET("/traces/:delivery_id", h.GetDeliveryTrace)

		// Cost attribution
		obs.GET("/cost/:delivery_id", h.GetCostAttribution)

		// Smart alerting
		obs.POST("/alerts", h.CreateSmartAlert)
		obs.GET("/alerts", h.ListSmartAlerts)
	}
}

// @Summary Get unified observability dashboard
// @Tags ObservabilitySuite
// @Produce json
// @Param period query string false "Time period (1h, 24h, 7d, 30d)"
// @Success 200 {object} ObservabilityDashboard
// @Router /observability-suite/dashboard [get]
func (h *SuiteHandler) GetDashboard(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	period := c.DefaultQuery("period", "24h")

	dashboard, err := h.suite.GetDashboard(c.Request.Context(), tenantID, period)
	if err != nil {
		httputil.InternalError(c, "DASHBOARD_ERROR", err)
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

// @Summary Get end-to-end delivery trace
// @Tags ObservabilitySuite
// @Produce json
// @Param delivery_id path string true "Delivery ID"
// @Success 200 {object} DeliveryTrace
// @Router /observability-suite/traces/{delivery_id} [get]
func (h *SuiteHandler) GetDeliveryTrace(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	deliveryID := c.Param("delivery_id")

	trace, err := h.suite.GetDeliveryTrace(c.Request.Context(), tenantID, deliveryID)
	if err != nil {
		httputil.InternalError(c, "TRACE_NOT_FOUND", err)
		return
	}

	c.JSON(http.StatusOK, trace)
}

// @Summary Get cost attribution for a delivery
// @Tags ObservabilitySuite
// @Produce json
// @Param delivery_id path string true "Delivery ID"
// @Success 200 {object} CostAttribution
// @Router /observability-suite/cost/{delivery_id} [get]
func (h *SuiteHandler) GetCostAttribution(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	deliveryID := c.Param("delivery_id")

	cost := h.suite.CalculateCostAttribution(c.Request.Context(), tenantID, deliveryID, 250, 2048)
	c.JSON(http.StatusOK, cost)
}

// @Summary Create a smart alert rule
// @Tags ObservabilitySuite
// @Accept json
// @Produce json
// @Param body body CreateSmartAlertRequest true "Smart alert config"
// @Success 201 {object} SmartAlert
// @Router /observability-suite/alerts [post]
func (h *SuiteHandler) CreateSmartAlert(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateSmartAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httputil.APIErrorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}

	alert, err := h.suite.CreateSmartAlert(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, httputil.APIErrorResponse{Code: "CREATE_FAILED", Message: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, alert)
}

// @Summary List smart alert rules
// @Tags ObservabilitySuite
// @Produce json
// @Success 200
// @Router /observability-suite/alerts [get]
func (h *SuiteHandler) ListSmartAlerts(c *gin.Context) {
	// Placeholder - would query from repository
	c.JSON(http.StatusOK, gin.H{"alerts": []interface{}{}, "total": 0})
}
