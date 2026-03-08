package sla

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for SLA management
type Handler struct {
	service *Service
}

// NewHandler creates a new SLA handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers SLA routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	sla := router.Group("/sla")
	{
		// Targets
		sla.POST("/targets", h.CreateTarget)
		sla.GET("/targets", h.ListTargets)
		sla.GET("/targets/:id", h.GetTarget)
		sla.PUT("/targets/:id", h.UpdateTarget)
		sla.DELETE("/targets/:id", h.DeleteTarget)

		// Compliance
		sla.GET("/targets/:id/compliance", h.GetTargetCompliance)
		sla.GET("/dashboard", h.GetDashboard)

		// Breaches
		sla.GET("/breaches", h.ListBreaches)
		sla.GET("/breaches/active", h.ListActiveBreaches)
		sla.POST("/check", h.CheckBreaches)

		// Alerts
		sla.GET("/targets/:id/alerts", h.GetAlertConfig)
		sla.PUT("/targets/:id/alerts", h.UpdateAlertConfig)
	}
}

// @Summary Create an SLA target
// @Tags SLA
// @Accept json
// @Produce json
// @Param body body CreateTargetRequest true "SLA target configuration"
// @Success 201 {object} Target
// @Router /sla/targets [post]
func (h *Handler) CreateTarget(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateTargetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	target, err := h.service.CreateTarget(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalError(c, "CREATE_FAILED", err)
		return
	}

	c.JSON(http.StatusCreated, target)
}

// @Summary List SLA targets
// @Tags SLA
// @Produce json
// @Success 200 {object} map[string][]Target
// @Router /sla/targets [get]
func (h *Handler) ListTargets(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	targets, err := h.service.ListTargets(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "LIST_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"targets": targets})
}

// @Summary Get an SLA target
// @Tags SLA
// @Produce json
// @Param id path string true "Target ID"
// @Success 200 {object} Target
// @Router /sla/targets/{id} [get]
func (h *Handler) GetTarget(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	targetID := c.Param("id")

	target, err := h.service.GetTarget(c.Request.Context(), tenantID, targetID)
	if err != nil {
		httputil.InternalError(c, "NOT_FOUND", err)
		return
	}

	c.JSON(http.StatusOK, target)
}

// @Summary Update an SLA target
// @Tags SLA
// @Accept json
// @Produce json
// @Param id path string true "Target ID"
// @Param body body CreateTargetRequest true "Updated target configuration"
// @Success 200 {object} Target
// @Router /sla/targets/{id} [put]
func (h *Handler) UpdateTarget(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	targetID := c.Param("id")

	var req CreateTargetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	target, err := h.service.UpdateTarget(c.Request.Context(), tenantID, targetID, &req)
	if err != nil {
		httputil.InternalError(c, "UPDATE_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, target)
}

// @Summary Delete an SLA target
// @Tags SLA
// @Param id path string true "Target ID"
// @Success 204
// @Router /sla/targets/{id} [delete]
func (h *Handler) DeleteTarget(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	targetID := c.Param("id")

	if err := h.service.DeleteTarget(c.Request.Context(), tenantID, targetID); err != nil {
		httputil.InternalError(c, "DELETE_FAILED", err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary Get compliance status for a target
// @Tags SLA
// @Produce json
// @Param id path string true "Target ID"
// @Success 200 {object} ComplianceStatus
// @Router /sla/targets/{id}/compliance [get]
func (h *Handler) GetTargetCompliance(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	targetID := c.Param("id")

	target, err := h.service.GetTarget(c.Request.Context(), tenantID, targetID)
	if err != nil {
		httputil.InternalError(c, "NOT_FOUND", err)
		return
	}

	status, err := h.service.GetComplianceStatus(c.Request.Context(), tenantID, target)
	if err != nil {
		httputil.InternalError(c, "COMPLIANCE_CHECK_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, status)
}

// @Summary Get SLA dashboard
// @Tags SLA
// @Produce json
// @Success 200 {object} Dashboard
// @Router /sla/dashboard [get]
func (h *Handler) GetDashboard(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	dashboard, err := h.service.GetDashboard(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "DASHBOARD_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

// @Summary List breach history
// @Tags SLA
// @Produce json
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} map[string][]Breach
// @Router /sla/breaches [get]
func (h *Handler) ListBreaches(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	breaches, err := h.service.GetBreachHistory(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		httputil.InternalError(c, "LIST_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"breaches": breaches})
}

// @Summary List active breaches
// @Tags SLA
// @Produce json
// @Success 200 {object} map[string][]Breach
// @Router /sla/breaches/active [get]
func (h *Handler) ListActiveBreaches(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	breaches, err := h.service.repo.ListActiveBreaches(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "LIST_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"breaches": breaches})
}

// @Summary Check for SLA breaches
// @Tags SLA
// @Produce json
// @Success 200 {object} map[string][]Breach
// @Router /sla/check [post]
func (h *Handler) CheckBreaches(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	breaches, err := h.service.CheckAndRecordBreaches(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "CHECK_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"new_breaches": breaches, "count": len(breaches)})
}

// @Summary Get alert configuration for a target
// @Tags SLA
// @Produce json
// @Param id path string true "Target ID"
// @Success 200 {object} AlertConfig
// @Router /sla/targets/{id}/alerts [get]
func (h *Handler) GetAlertConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	targetID := c.Param("id")

	config, err := h.service.GetAlertConfig(c.Request.Context(), tenantID, targetID)
	if err != nil {
		httputil.InternalError(c, "NOT_FOUND", err)
		return
	}

	c.JSON(http.StatusOK, config)
}

// @Summary Update alert configuration for a target
// @Tags SLA
// @Accept json
// @Produce json
// @Param id path string true "Target ID"
// @Param body body UpdateAlertConfigRequest true "Alert configuration"
// @Success 200 {object} AlertConfig
// @Router /sla/targets/{id}/alerts [put]
func (h *Handler) UpdateAlertConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	targetID := c.Param("id")

	var req UpdateAlertConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	config, err := h.service.UpdateAlertConfig(c.Request.Context(), tenantID, targetID, &req)
	if err != nil {
		httputil.InternalError(c, "UPDATE_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, config)
}
