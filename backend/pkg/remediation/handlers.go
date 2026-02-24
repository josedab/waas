package remediation

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for remediation
type Handler struct {
	service *Service
}

// NewHandler creates a new remediation handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers remediation routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	remediation := r.Group("/remediation")
	{
		remediation.GET("/actions", h.ListActions)
		remediation.POST("/actions", h.CreateAction)
		remediation.GET("/actions/pending", h.GetPendingActions)
		remediation.GET("/actions/:id", h.GetAction)
		remediation.POST("/actions/:id/approve", h.ApproveAction)
		remediation.POST("/actions/:id/reject", h.RejectAction)
		remediation.POST("/actions/:id/rollback", h.RollbackAction)
		remediation.GET("/actions/:id/audit", h.GetAuditLogs)

		remediation.GET("/policy", h.GetPolicy)
		remediation.PUT("/policy", h.UpdatePolicy)

		remediation.GET("/metrics", h.GetMetrics)
		remediation.POST("/analyze/:deliveryId", h.AnalyzeAndSuggest)
	}
}

// CreateAction creates a new remediation action
// @Summary Create remediation action
// @Description Create a new remediation action for an endpoint
// @Tags remediation
// @Accept json
// @Produce json
// @Param request body CreateActionRequest true "Action request"
// @Success 201 {object} RemediationAction
// @Failure 400 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /remediation/actions [post]
func (h *Handler) CreateAction(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	action, err := h.service.CreateAction(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, action)
}

// GetAction retrieves a remediation action
// @Summary Get remediation action
// @Description Get details of a remediation action
// @Tags remediation
// @Produce json
// @Param id path string true "Action ID"
// @Success 200 {object} RemediationAction
// @Failure 404 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /remediation/actions/{id} [get]
func (h *Handler) GetAction(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	actionID := c.Param("id")
	action, err := h.service.GetAction(c.Request.Context(), tenantID, actionID)
	if err != nil {
		if err == ErrActionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "action not found"})
			return
		}
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, action)
}

// ListActions lists remediation actions
// @Summary List remediation actions
// @Description List all remediation actions for the tenant
// @Tags remediation
// @Produce json
// @Param endpoint_id query string false "Filter by endpoint ID"
// @Param action_type query string false "Filter by action type"
// @Param status query string false "Filter by status"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Success 200 {object} ListActionsResponse
// @Security ApiKeyAuth
// @Router /remediation/actions [get]
func (h *Handler) ListActions(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	filters := &ActionFilters{}

	if endpointID := c.Query("endpoint_id"); endpointID != "" {
		filters.EndpointID = endpointID
	}
	if actionType := c.Query("action_type"); actionType != "" {
		at := ActionType(actionType)
		filters.ActionType = &at
	}
	if status := c.Query("status"); status != "" {
		s := ActionStatus(status)
		filters.Status = &s
	}
	if page, err := strconv.Atoi(c.Query("page")); err == nil && page > 0 {
		filters.Page = page
	}
	if pageSize, err := strconv.Atoi(c.Query("page_size")); err == nil && pageSize > 0 {
		filters.PageSize = pageSize
	}

	response, err := h.service.ListActions(c.Request.Context(), tenantID, filters)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetPendingActions gets all pending actions
// @Summary Get pending actions
// @Description Get all pending remediation actions awaiting approval
// @Tags remediation
// @Produce json
// @Success 200 {array} RemediationAction
// @Security ApiKeyAuth
// @Router /remediation/actions/pending [get]
func (h *Handler) GetPendingActions(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	actions, err := h.service.GetPendingActions(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, actions)
}

// ApproveAction approves a pending action
// @Summary Approve action
// @Description Approve a pending remediation action
// @Tags remediation
// @Produce json
// @Param id path string true "Action ID"
// @Success 200 {object} RemediationAction
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /remediation/actions/{id}/approve [post]
func (h *Handler) ApproveAction(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	actionID := c.Param("id")

	if userID == "" {
		userID = "api_user"
	}

	action, err := h.service.ApproveAction(c.Request.Context(), tenantID, actionID, userID)
	if err != nil {
		if err == ErrActionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "action not found"})
			return
		}
		if err == ErrInvalidTransition || err == ErrActionExpired {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, action)
}

// RejectAction rejects a pending action
// @Summary Reject action
// @Description Reject a pending remediation action
// @Tags remediation
// @Accept json
// @Produce json
// @Param id path string true "Action ID"
// @Param request body object{reason=string} false "Rejection reason"
// @Success 200 {object} RemediationAction
// @Failure 400 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /remediation/actions/{id}/reject [post]
func (h *Handler) RejectAction(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	actionID := c.Param("id")

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	if userID == "" {
		userID = "api_user"
	}

	action, err := h.service.RejectAction(c.Request.Context(), tenantID, actionID, userID, req.Reason)
	if err != nil {
		if err == ErrActionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "action not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, action)
}

// RollbackAction rolls back a completed action
// @Summary Rollback action
// @Description Rollback a completed remediation action to previous state
// @Tags remediation
// @Produce json
// @Param id path string true "Action ID"
// @Success 200 {object} RemediationAction
// @Failure 400 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /remediation/actions/{id}/rollback [post]
func (h *Handler) RollbackAction(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	actionID := c.Param("id")

	if userID == "" {
		userID = "api_user"
	}

	action, err := h.service.RollbackAction(c.Request.Context(), tenantID, actionID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, action)
}

// GetAuditLogs retrieves audit logs for an action
// @Summary Get audit logs
// @Description Get audit logs for a remediation action
// @Tags remediation
// @Produce json
// @Param id path string true "Action ID"
// @Success 200 {array} ActionAuditLog
// @Security ApiKeyAuth
// @Router /remediation/actions/{id}/audit [get]
func (h *Handler) GetAuditLogs(c *gin.Context) {
	actionID := c.Param("id")

	logs, err := h.service.GetAuditLogs(c.Request.Context(), actionID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, logs)
}

// GetPolicy retrieves the remediation policy
// @Summary Get remediation policy
// @Description Get the remediation policy for the tenant
// @Tags remediation
// @Produce json
// @Success 200 {object} RemediationPolicy
// @Security ApiKeyAuth
// @Router /remediation/policy [get]
func (h *Handler) GetPolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	policy, err := h.service.GetPolicy(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, policy)
}

// UpdatePolicy updates the remediation policy
// @Summary Update remediation policy
// @Description Update the remediation policy for the tenant
// @Tags remediation
// @Accept json
// @Produce json
// @Param request body RemediationPolicy true "Policy update"
// @Success 200 {object} RemediationPolicy
// @Security ApiKeyAuth
// @Router /remediation/policy [put]
func (h *Handler) UpdatePolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var policy RemediationPolicy
	if err := c.ShouldBindJSON(&policy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdatePolicy(c.Request.Context(), tenantID, &policy); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, policy)
}

// GetMetrics retrieves remediation metrics
// @Summary Get remediation metrics
// @Description Get remediation metrics for the tenant
// @Tags remediation
// @Produce json
// @Param period query string false "Period (daily, weekly, monthly)"
// @Success 200 {object} RemediationMetrics
// @Security ApiKeyAuth
// @Router /remediation/metrics [get]
func (h *Handler) GetMetrics(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	period := c.DefaultQuery("period", "daily")

	metrics, err := h.service.GetMetrics(c.Request.Context(), tenantID, period)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// AnalyzeAndSuggest analyzes a delivery failure and suggests remediation
// @Summary Analyze and suggest remediation
// @Description Analyze a delivery failure and get remediation suggestions
// @Tags remediation
// @Produce json
// @Param deliveryId path string true "Delivery ID"
// @Success 200 {object} AnalysisWithRemediation
// @Security ApiKeyAuth
// @Router /remediation/analyze/{deliveryId} [post]
func (h *Handler) AnalyzeAndSuggest(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	deliveryID := c.Param("deliveryId")

	analysis, err := h.service.AnalyzeAndSuggest(c.Request.Context(), tenantID, deliveryID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, analysis)
}
