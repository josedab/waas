package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/josedab/waas/internal/api/services"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/utils"
)

// SelfHealingHandler handles self-healing endpoint intelligence endpoints
type SelfHealingHandler struct {
	service *services.SelfHealingService
	logger  *utils.Logger
}

// NewSelfHealingHandler creates a new self-healing handler
func NewSelfHealingHandler(service *services.SelfHealingService, logger *utils.Logger) *SelfHealingHandler {
	return &SelfHealingHandler{
		service: service,
		logger:  logger,
	}
}

// GetDashboard retrieves the self-healing dashboard
// @Summary Get self-healing dashboard
// @Tags Self-Healing
// @Produce json
// @Success 200 {object} models.SelfHealingDashboard
// @Router /self-healing/dashboard [get]
func (h *SelfHealingHandler) GetDashboard(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	dashboard, err := h.service.GetDashboard(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Error("Failed to get dashboard", map[string]interface{}{"error": err.Error()})
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

// PredictHealth generates a health prediction for an endpoint
// @Summary Predict endpoint health
// @Tags Self-Healing
// @Accept json
// @Produce json
// @Param endpoint_id path string true "Endpoint ID"
// @Param request body models.MLFeatureVector true "Feature vector"
// @Success 200 {object} models.EndpointHealthPrediction
// @Router /self-healing/endpoints/{endpoint_id}/predict [post]
func (h *SelfHealingHandler) PredictHealth(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	endpointID, err := uuid.Parse(c.Param("endpoint_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid endpoint_id"})
		return
	}

	var features models.MLFeatureVector
	if err := c.ShouldBindJSON(&features); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	features.EndpointID = endpointID

	prediction, err := h.service.PredictEndpointHealth(c.Request.Context(), tenantID, endpointID, &features)
	if err != nil {
		h.logger.Error("Failed to predict health", map[string]interface{}{"error": err.Error()})
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, prediction)
}

// GetEndpointAnalysis retrieves comprehensive health analysis for an endpoint
// @Summary Get endpoint health analysis
// @Tags Self-Healing
// @Produce json
// @Param endpoint_id path string true "Endpoint ID"
// @Success 200 {object} models.EndpointHealthAnalysis
// @Router /self-healing/endpoints/{endpoint_id}/analysis [get]
func (h *SelfHealingHandler) GetEndpointAnalysis(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	endpointID, err := uuid.Parse(c.Param("endpoint_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid endpoint_id"})
		return
	}

	analysis, err := h.service.GetEndpointHealthAnalysis(c.Request.Context(), tenantID, endpointID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, analysis)
}

// GetPredictions retrieves recent health predictions
// @Summary List health predictions
// @Tags Self-Healing
// @Produce json
// @Success 200 {array} models.EndpointHealthPrediction
// @Router /self-healing/predictions [get]
func (h *SelfHealingHandler) GetPredictions(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	predictions, err := h.service.GetPredictions(c.Request.Context(), tenantID, 20)
	if err != nil {
		h.logger.Error("Failed to get predictions", map[string]interface{}{"error": err.Error()})
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"predictions": predictions})
}

// CreateRemediationRule creates a new remediation rule
// @Summary Create remediation rule
// @Tags Self-Healing
// @Accept json
// @Produce json
// @Param request body models.CreateRemediationRuleRequest true "Remediation rule"
// @Success 201 {object} models.AutoRemediationRule
// @Router /self-healing/rules [post]
func (h *SelfHealingHandler) CreateRemediationRule(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	var req models.CreateRemediationRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule, err := h.service.CreateRemediationRule(c.Request.Context(), tenantID, &req)
	if err != nil {
		h.logger.Error("Failed to create rule", map[string]interface{}{"error": err.Error()})
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, rule)
}

// GetRemediationRules retrieves remediation rules
// @Summary List remediation rules
// @Tags Self-Healing
// @Produce json
// @Success 200 {array} models.AutoRemediationRule
// @Router /self-healing/rules [get]
func (h *SelfHealingHandler) GetRemediationRules(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	rules, err := h.service.GetRemediationRules(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Error("Failed to get rules", map[string]interface{}{"error": err.Error()})
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

// TriggerRemediation triggers a manual remediation
// @Summary Trigger remediation manually
// @Tags Self-Healing
// @Accept json
// @Produce json
// @Param request body models.TriggerRemediationRequest true "Remediation request"
// @Success 200 {object} models.RemediationAction
// @Router /self-healing/remediate [post]
func (h *SelfHealingHandler) TriggerRemediation(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	var req models.TriggerRemediationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	action, err := h.service.TriggerManualRemediation(c.Request.Context(), tenantID, &req)
	if err != nil {
		h.logger.Error("Failed to trigger remediation", map[string]interface{}{"error": err.Error()})
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, action)
}

// GetRemediationActions retrieves recent remediation actions
// @Summary List remediation actions
// @Tags Self-Healing
// @Produce json
// @Success 200 {array} models.RemediationAction
// @Router /self-healing/actions [get]
func (h *SelfHealingHandler) GetRemediationActions(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	actions, err := h.service.GetRemediationActions(c.Request.Context(), tenantID, 20)
	if err != nil {
		h.logger.Error("Failed to get actions", map[string]interface{}{"error": err.Error()})
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"actions": actions})
}

// UpdateCircuitBreaker updates circuit breaker configuration
// @Summary Update circuit breaker
// @Tags Self-Healing
// @Accept json
// @Produce json
// @Param request body models.UpdateCircuitBreakerRequest true "Circuit breaker config"
// @Success 200 {object} models.EndpointCircuitBreaker
// @Router /self-healing/circuit-breakers [put]
func (h *SelfHealingHandler) UpdateCircuitBreaker(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	var req models.UpdateCircuitBreakerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cb, err := h.service.UpdateCircuitBreaker(c.Request.Context(), tenantID, &req)
	if err != nil {
		h.logger.Error("Failed to update circuit breaker", map[string]interface{}{"error": err.Error()})
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, cb)
}

// GetSuggestions retrieves pending optimization suggestions
// @Summary List optimization suggestions
// @Tags Self-Healing
// @Produce json
// @Success 200 {array} models.EndpointOptimizationSuggestion
// @Router /self-healing/suggestions [get]
func (h *SelfHealingHandler) GetSuggestions(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	suggestions, err := h.service.GetSuggestions(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Error("Failed to get suggestions", map[string]interface{}{"error": err.Error()})
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"suggestions": suggestions})
}

// ApplySuggestion applies an optimization suggestion
// @Summary Apply optimization suggestion
// @Tags Self-Healing
// @Produce json
// @Param suggestion_id path string true "Suggestion ID"
// @Success 200 {object} map[string]interface{}
// @Router /self-healing/suggestions/{suggestion_id}/apply [post]
func (h *SelfHealingHandler) ApplySuggestion(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	suggestionID, err := uuid.Parse(c.Param("suggestion_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid suggestion_id"})
		return
	}

	if err := h.service.ApplySuggestion(c.Request.Context(), tenantID, suggestionID); err != nil {
		h.logger.Error("Failed to apply suggestion", map[string]interface{}{"error": err.Error()})
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Suggestion applied successfully"})
}

// GetSupportedActionTypes returns supported remediation action types
// @Summary Get supported action types
// @Tags Self-Healing
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /self-healing/action-types [get]
func (h *SelfHealingHandler) GetSupportedActionTypes(c *gin.Context) {
	actionTypes := []map[string]interface{}{
		{"code": "disable_endpoint", "name": "Disable Endpoint", "description": "Temporarily disable the endpoint"},
		{"code": "adjust_retry", "name": "Adjust Retry Config", "description": "Modify retry timing and attempts"},
		{"code": "notify", "name": "Send Notification", "description": "Alert team about potential issues"},
		{"code": "circuit_break", "name": "Open Circuit Breaker", "description": "Stop requests to failing endpoint"},
		{"code": "rate_limit", "name": "Apply Rate Limit", "description": "Reduce request rate to endpoint"},
	}

	c.JSON(http.StatusOK, gin.H{"action_types": actionTypes})
}
