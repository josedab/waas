package waf

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// SecuritySuiteHandler provides HTTP handlers for the unified security suite
type SecuritySuiteHandler struct {
	suite *SecuritySuite
}

// NewSecuritySuiteHandler creates a new handler
func NewSecuritySuiteHandler(suite *SecuritySuite) *SecuritySuiteHandler {
	return &SecuritySuiteHandler{suite: suite}
}

// RegisterSecurityRoutes registers security suite routes
func (h *SecuritySuiteHandler) RegisterSecurityRoutes(r *gin.RouterGroup) {
	sec := r.Group("/security")
	{
		sec.GET("/posture", h.GetSecurityPosture)
		sec.POST("/verify", h.VerifyDelivery)
		sec.GET("/audit-logs", h.ListAuditLogs)

		ip := sec.Group("/ip-allowlist")
		{
			ip.GET("", h.ListIPAllowlist)
			ip.POST("", h.AddIPAllowlist)
			ip.DELETE("/:entryId", h.RemoveIPAllowlist)
		}
	}
}

// GetSecurityPosture returns the unified security posture
// @Summary Get security posture
// @Tags security
// @Produce json
// @Success 200 {object} SecurityPosture
// @Router /security/posture [get]
func (h *SecuritySuiteHandler) GetSecurityPosture(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	posture, err := h.suite.GetSecurityPosture(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, posture)
}

// VerifyDelivery performs zero-trust verification
// @Summary Verify delivery (zero-trust)
// @Tags security
// @Accept json
// @Produce json
// @Success 200 {object} ZeroTrustVerification
// @Router /security/verify [post]
func (h *SecuritySuiteHandler) VerifyDelivery(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		EndpointID string `json:"endpoint_id" binding:"required"`
		DeliveryID string `json:"delivery_id" binding:"required"`
		SourceIP   string `json:"source_ip" binding:"required"`
		Payload    []byte `json:"payload,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	verification, err := h.suite.VerifyDelivery(c.Request.Context(), tenantID, req.EndpointID, req.DeliveryID, req.SourceIP, req.Payload)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	status := http.StatusOK
	if !verification.OverallPassed {
		status = http.StatusForbidden
	}
	c.JSON(status, verification)
}

// ListIPAllowlist lists IP allowlist entries
// @Summary List IP allowlist
// @Tags security
// @Produce json
// @Success 200 {array} IPAllowlistEntry
// @Router /security/ip-allowlist [get]
func (h *SecuritySuiteHandler) ListIPAllowlist(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	entries, err := h.suite.ListIPAllowlist(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, entries)
}

// AddIPAllowlist adds an IP to the allowlist
// @Summary Add IP to allowlist
// @Tags security
// @Accept json
// @Produce json
// @Success 201 {object} IPAllowlistEntry
// @Router /security/ip-allowlist [post]
func (h *SecuritySuiteHandler) AddIPAllowlist(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateIPAllowlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entry, err := h.suite.AddIPAllowlist(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, entry)
}

// RemoveIPAllowlist removes an IP from the allowlist
// @Summary Remove IP from allowlist
// @Tags security
// @Param entryId path string true "Entry ID"
// @Success 204 "No content"
// @Router /security/ip-allowlist/{entryId} [delete]
func (h *SecuritySuiteHandler) RemoveIPAllowlist(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	entryID := c.Param("entryId")
	if err := h.suite.RemoveIPAllowlist(c.Request.Context(), tenantID, entryID); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ListAuditLogs lists security audit logs
// @Summary List security audit logs
// @Tags security
// @Produce json
// @Param limit query int false "Limit"
// @Success 200 {array} SecurityAuditLog
// @Router /security/audit-logs [get]
func (h *SecuritySuiteHandler) ListAuditLogs(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit := 50
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}

	logs, err := h.suite.ListAuditLogs(c.Request.Context(), tenantID, limit)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, logs)
}
