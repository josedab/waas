package waf

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	pkgerrors "github.com/josedab/waas/pkg/errors"
)

// Handler provides HTTP handlers for WAF operations
type Handler struct {
	service *Service
}

// NewHandler creates a new WAF handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers WAF routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	waf := r.Group("/waf")
	{
		// Scanning
		waf.POST("/scan", h.ScanPayload)
		waf.GET("/scans", h.ListScans)

		// WAF Rules
		waf.POST("/rules", h.CreateRule)
		waf.GET("/rules", h.ListRules)
		waf.PUT("/rules/:id", h.UpdateRule)
		waf.DELETE("/rules/:id", h.DeleteRule)

		// Quarantine
		waf.GET("/quarantine", h.ListQuarantined)
		waf.POST("/quarantine/:id/review", h.ReviewQuarantine)

		// IP Reputation
		waf.POST("/ip/check", h.CheckIPReputation)
		waf.POST("/ip/report", h.ReportIP)
		waf.GET("/ip/blocklist", h.ListBlockedIPs)

		// Dashboard & Alerts
		waf.GET("/dashboard", h.GetDashboard)
		waf.GET("/alerts", h.GetAlerts)
		waf.POST("/alerts/:id/acknowledge", h.AcknowledgeAlert)

		// Security Scanning & Thresholds
		waf.POST("/scan-endpoint", h.ScanEndpointSecurity)
		waf.POST("/export-report", h.ExportSecurityReport)
		waf.GET("/threshold", h.GetSecurityThreshold)
		waf.PUT("/threshold", h.UpdateSecurityThreshold)
	}
}

// ScanPayload scans a webhook payload for security threats
// @Summary Scan webhook payload
// @Description Scan a webhook payload for XSS, SQL injection, path traversal, and other threats
// @Tags waf
// @Accept json
// @Produce json
// @Param request body ScanPayloadRequest true "Scan request"
// @Success 200 {object} ScanResult
// @Failure 400 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /waf/scan [post]
func (h *Handler) ScanPayload(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		pkgerrors.AbortWithUnauthorized(c)
		return
	}

	var req ScanPayloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	result, err := h.service.ScanPayload(c.Request.Context(), tenantID, &req)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, result)
}

// ListScans lists scan results
// @Summary List scan results
// @Description List webhook payload scan results for the tenant
// @Tags waf
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /waf/scans [get]
func (h *Handler) ListScans(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		pkgerrors.AbortWithUnauthorized(c)
		return
	}

	limit := 20
	offset := 0
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}
	if o, err := strconv.Atoi(c.Query("offset")); err == nil && o >= 0 {
		offset = o
	}

	results, total, err := h.service.ListScanResults(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// CreateRule creates a new WAF rule
// @Summary Create WAF rule
// @Description Create a new custom WAF rule for the tenant
// @Tags waf
// @Accept json
// @Produce json
// @Param request body CreateWAFRuleRequest true "Rule request"
// @Success 201 {object} WAFRule
// @Failure 400 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /waf/rules [post]
func (h *Handler) CreateRule(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		pkgerrors.AbortWithUnauthorized(c)
		return
	}

	var req CreateWAFRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	rule, err := h.service.CreateWAFRule(c.Request.Context(), tenantID, &req)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusCreated, rule)
}

// ListRules lists WAF rules
// @Summary List WAF rules
// @Description List all WAF rules for the tenant
// @Tags waf
// @Produce json
// @Success 200 {array} WAFRule
// @Security ApiKeyAuth
// @Router /waf/rules [get]
func (h *Handler) ListRules(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		pkgerrors.AbortWithUnauthorized(c)
		return
	}

	rules, err := h.service.ListWAFRules(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, rules)
}

// UpdateRule updates a WAF rule
// @Summary Update WAF rule
// @Description Update an existing WAF rule
// @Tags waf
// @Accept json
// @Produce json
// @Param id path string true "Rule ID"
// @Param request body CreateWAFRuleRequest true "Rule update"
// @Success 200 {object} WAFRule
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /waf/rules/{id} [put]
func (h *Handler) UpdateRule(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		pkgerrors.AbortWithUnauthorized(c)
		return
	}

	ruleID := c.Param("id")

	var req CreateWAFRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	rule, err := h.service.UpdateWAFRule(c.Request.Context(), tenantID, ruleID, &req)
	if err != nil {
		if err == ErrWAFRuleNotFound {
			pkgerrors.AbortWithNotFound(c, "rule")
			return
		}
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, rule)
}

// DeleteRule deletes a WAF rule
// @Summary Delete WAF rule
// @Description Delete a WAF rule
// @Tags waf
// @Param id path string true "Rule ID"
// @Success 204
// @Failure 404 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /waf/rules/{id} [delete]
func (h *Handler) DeleteRule(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		pkgerrors.AbortWithUnauthorized(c)
		return
	}

	ruleID := c.Param("id")

	if err := h.service.DeleteWAFRule(c.Request.Context(), tenantID, ruleID); err != nil {
		if err == ErrWAFRuleNotFound {
			pkgerrors.AbortWithNotFound(c, "rule")
			return
		}
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.Status(http.StatusNoContent)
}

// ListQuarantined lists quarantined webhooks
// @Summary List quarantined webhooks
// @Description List all quarantined webhook deliveries
// @Tags waf
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /waf/quarantine [get]
func (h *Handler) ListQuarantined(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		pkgerrors.AbortWithUnauthorized(c)
		return
	}

	limit := 20
	offset := 0
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}
	if o, err := strconv.Atoi(c.Query("offset")); err == nil && o >= 0 {
		offset = o
	}

	items, total, err := h.service.ListQuarantined(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"quarantined": items,
		"total":       total,
		"limit":       limit,
		"offset":      offset,
	})
}

// ReviewQuarantine reviews a quarantined webhook
// @Summary Review quarantined webhook
// @Description Approve or reject a quarantined webhook delivery
// @Tags waf
// @Accept json
// @Produce json
// @Param id path string true "Quarantine ID"
// @Param request body ReviewQuarantineRequest true "Review request"
// @Success 200 {object} QuarantinedWebhook
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /waf/quarantine/{id}/review [post]
func (h *Handler) ReviewQuarantine(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")
	if tenantID == "" {
		pkgerrors.AbortWithUnauthorized(c)
		return
	}

	quarantineID := c.Param("id")

	var req ReviewQuarantineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	if userID == "" {
		userID = "api_user"
	}

	quarantine, err := h.service.ReviewQuarantine(c.Request.Context(), tenantID, quarantineID, userID, &req)
	if err != nil {
		if err == ErrQuarantineNotFound {
			pkgerrors.AbortWithNotFound(c, "quarantined webhook")
			return
		}
		if err == ErrAlreadyReviewed {
			pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
			return
		}
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, quarantine)
}

// CheckIPReputation checks IP reputation
// @Summary Check IP reputation
// @Description Check the reputation and threat score of an IP address
// @Tags waf
// @Accept json
// @Produce json
// @Param request body object{ip=string} true "IP check request"
// @Success 200 {object} IPReputation
// @Failure 400 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /waf/ip/check [post]
func (h *Handler) CheckIPReputation(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		pkgerrors.AbortWithUnauthorized(c)
		return
	}

	var req struct {
		IP string `json:"ip" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	reputation, err := h.service.CheckIPReputation(c.Request.Context(), req.IP)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, reputation)
}

// ReportIP reports a malicious IP
// @Summary Report malicious IP
// @Description Report an IP address as malicious
// @Tags waf
// @Accept json
// @Produce json
// @Param request body ReportIPRequest true "Report request"
// @Success 200 {object} IPReputation
// @Failure 400 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /waf/ip/report [post]
func (h *Handler) ReportIP(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		pkgerrors.AbortWithUnauthorized(c)
		return
	}

	var req ReportIPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	reputation, err := h.service.ReportIP(c.Request.Context(), &req)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, reputation)
}

// ListBlockedIPs lists blocked IPs
// @Summary List blocked IPs
// @Description List all blocked IP addresses
// @Tags waf
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /waf/ip/blocklist [get]
func (h *Handler) ListBlockedIPs(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		pkgerrors.AbortWithUnauthorized(c)
		return
	}

	limit := 50
	offset := 0
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}
	if o, err := strconv.Atoi(c.Query("offset")); err == nil && o >= 0 {
		offset = o
	}

	ips, total, err := h.service.ListBlockedIPs(c.Request.Context(), limit, offset)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"blocked_ips": ips,
		"total":       total,
		"limit":       limit,
		"offset":      offset,
	})
}

// GetDashboard returns security dashboard
// @Summary Get security dashboard
// @Description Get aggregated security metrics and dashboard data
// @Tags waf
// @Produce json
// @Success 200 {object} SecurityDashboard
// @Security ApiKeyAuth
// @Router /waf/dashboard [get]
func (h *Handler) GetDashboard(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		pkgerrors.AbortWithUnauthorized(c)
		return
	}

	dashboard, err := h.service.GetSecurityDashboard(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

// GetAlerts lists security alerts
// @Summary Get security alerts
// @Description List security alerts for the tenant
// @Tags waf
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /waf/alerts [get]
func (h *Handler) GetAlerts(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		pkgerrors.AbortWithUnauthorized(c)
		return
	}

	limit := 20
	offset := 0
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}
	if o, err := strconv.Atoi(c.Query("offset")); err == nil && o >= 0 {
		offset = o
	}

	alerts, total, err := h.service.GetSecurityAlerts(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"alerts": alerts,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// AcknowledgeAlert acknowledges a security alert
// @Summary Acknowledge alert
// @Description Acknowledge a security alert
// @Tags waf
// @Produce json
// @Param id path string true "Alert ID"
// @Success 200 {object} SecurityAlert
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /waf/alerts/{id}/acknowledge [post]
func (h *Handler) AcknowledgeAlert(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		pkgerrors.AbortWithUnauthorized(c)
		return
	}

	alertID := c.Param("id")

	alert, err := h.service.AcknowledgeAlert(c.Request.Context(), tenantID, alertID)
	if err != nil {
		if err == ErrAlertNotFound {
			pkgerrors.AbortWithNotFound(c, "alert")
			return
		}
		if err == ErrAlreadyAcknowledged {
			pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
			return
		}
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, alert)
}

// ScanEndpointSecurity scans an endpoint for security issues
// @Summary Scan endpoint security
// @Description Perform a full security scan of a webhook endpoint including TLS, headers, and compliance
// @Tags waf
// @Accept json
// @Produce json
// @Param request body object{url=string} true "Endpoint URL to scan"
// @Success 200 {object} SecurityScanResult
// @Failure 400 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /waf/scan-endpoint [post]
func (h *Handler) ScanEndpointSecurity(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		pkgerrors.AbortWithUnauthorized(c)
		return
	}

	var req struct {
		URL string `json:"url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	result, err := h.service.ScanEndpointSecurity(c.Request.Context(), tenantID, req.URL)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExportSecurityReport exports a security report
// @Summary Export security report
// @Description Export a security scan result in JSON or CSV format
// @Tags waf
// @Accept json
// @Produce json
// @Param request body object true "Export request with scan result and format"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /waf/export-report [post]
func (h *Handler) ExportSecurityReport(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		pkgerrors.AbortWithUnauthorized(c)
		return
	}

	var req struct {
		Result SecurityScanResult `json:"result" binding:"required"`
		Format string             `json:"format" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	data, err := h.service.ExportSecurityReport(c.Request.Context(), &req.Result, req.Format)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"format": req.Format,
		"data":   string(data),
	})
}

// GetSecurityThreshold retrieves security threshold settings
// @Summary Get security threshold
// @Description Get the security threshold configuration for the tenant
// @Tags waf
// @Produce json
// @Success 200 {object} SecurityThreshold
// @Security ApiKeyAuth
// @Router /waf/threshold [get]
func (h *Handler) GetSecurityThreshold(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		pkgerrors.AbortWithUnauthorized(c)
		return
	}

	threshold, err := h.service.GetSecurityThreshold(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, threshold)
}

// UpdateSecurityThreshold updates security threshold settings
// @Summary Update security threshold
// @Description Update the security threshold configuration for the tenant
// @Tags waf
// @Accept json
// @Produce json
// @Param request body SecurityThreshold true "Threshold settings"
// @Success 200 {object} SecurityThreshold
// @Failure 400 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /waf/threshold [put]
func (h *Handler) UpdateSecurityThreshold(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		pkgerrors.AbortWithUnauthorized(c)
		return
	}

	var threshold SecurityThreshold
	if err := c.ShouldBindJSON(&threshold); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	threshold.TenantID = tenantID

	if err := h.service.UpdateSecurityThreshold(c.Request.Context(), &threshold); err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, threshold)
}
