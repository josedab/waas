package autoremediation

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for auto-remediation
type Handler struct {
	service *Service
}

// NewHandler creates a new auto-remediation handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers auto-remediation routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	remediation := router.Group("/remediation")
	{
		// Patterns
		remediation.GET("/patterns", h.ListPatterns)
		remediation.GET("/patterns/:id", h.GetPattern)
		remediation.POST("/patterns/analyze", h.AnalyzeFailures)

		// Recommendations
		remediation.GET("/recommendations", h.GetRecommendations)

		// Rules
		remediation.POST("/rules", h.CreateRule)
		remediation.GET("/rules", h.ListRules)

		// Actions
		remediation.POST("/actions/apply", h.ApplyRemediation)
		remediation.POST("/actions/:id/revert", h.RevertAction)
		remediation.GET("/actions", h.ListActions)

		// Predictions
		remediation.GET("/predictions", h.PredictEndpointHealth)
	}
}

// @Summary List failure patterns
// @Tags AutoRemediation
// @Produce json
// @Success 200 {object} map[string][]FailurePattern
// @Router /remediation/patterns [get]
func (h *Handler) ListPatterns(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	patterns, err := h.service.GetPatterns(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "LIST_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"patterns": patterns})
}

// @Summary Get a failure pattern
// @Tags AutoRemediation
// @Produce json
// @Param id path string true "Pattern ID"
// @Success 200 {object} FailurePattern
// @Router /remediation/patterns/{id} [get]
func (h *Handler) GetPattern(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	patternID := c.Param("id")

	pattern, err := h.service.GetPattern(c.Request.Context(), tenantID, patternID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, pattern)
}

// @Summary Analyze failures and detect patterns
// @Tags AutoRemediation
// @Produce json
// @Success 200 {object} map[string][]FailurePattern
// @Router /remediation/patterns/analyze [post]
func (h *Handler) AnalyzeFailures(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	patterns, err := h.service.AnalyzeFailures(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "ANALYZE_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"patterns": patterns, "count": len(patterns)})
}

// @Summary Get remediation recommendations
// @Tags AutoRemediation
// @Produce json
// @Success 200 {object} map[string][]Recommendation
// @Router /remediation/recommendations [get]
func (h *Handler) GetRecommendations(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	recommendations, err := h.service.GetRecommendations(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "RECOMMENDATIONS_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"recommendations": recommendations})
}

// @Summary Create a remediation rule
// @Tags AutoRemediation
// @Accept json
// @Produce json
// @Param body body CreateRuleRequest true "Remediation rule configuration"
// @Success 201 {object} RemediationRule
// @Router /remediation/rules [post]
func (h *Handler) CreateRule(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	rule, err := h.service.CreateRule(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalError(c, "CREATE_FAILED", err)
		return
	}

	c.JSON(http.StatusCreated, rule)
}

// @Summary List remediation rules
// @Tags AutoRemediation
// @Produce json
// @Success 200 {object} map[string][]RemediationRule
// @Router /remediation/rules [get]
func (h *Handler) ListRules(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	rules, err := h.service.ListRules(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "LIST_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

// @Summary Apply a remediation action
// @Tags AutoRemediation
// @Accept json
// @Produce json
// @Param body body ApplyActionRequest true "Action to apply"
// @Success 201 {object} RemediationAction
// @Router /remediation/actions/apply [post]
func (h *Handler) ApplyRemediation(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req ApplyActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	action, err := h.service.ApplyRemediation(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalError(c, "APPLY_FAILED", err)
		return
	}

	c.JSON(http.StatusCreated, action)
}

// @Summary Revert a remediation action
// @Tags AutoRemediation
// @Produce json
// @Param id path string true "Action ID"
// @Success 200 {object} RemediationAction
// @Router /remediation/actions/{id}/revert [post]
func (h *Handler) RevertAction(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	actionID := c.Param("id")

	action, err := h.service.RevertAction(c.Request.Context(), tenantID, actionID)
	if err != nil {
		httputil.InternalError(c, "REVERT_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, action)
}

// @Summary List remediation actions
// @Tags AutoRemediation
// @Produce json
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} map[string][]RemediationAction
// @Router /remediation/actions [get]
func (h *Handler) ListActions(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	actions, err := h.service.ListActions(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		httputil.InternalError(c, "LIST_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"actions": actions})
}

// @Summary Predict endpoint health
// @Tags AutoRemediation
// @Produce json
// @Success 200 {object} map[string][]HealthPrediction
// @Router /remediation/predictions [get]
func (h *Handler) PredictEndpointHealth(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	predictions, err := h.service.PredictEndpointHealth(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "PREDICTION_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"predictions": predictions})
}
