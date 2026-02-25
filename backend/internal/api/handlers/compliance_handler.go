package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/josedab/waas/internal/api/services"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/utils"
)

// ComplianceHandler handles compliance endpoints
type ComplianceHandler struct {
	service *services.ComplianceService
	logger  *utils.Logger
}

// NewComplianceHandler creates a new compliance handler
func NewComplianceHandler(service *services.ComplianceService, logger *utils.Logger) *ComplianceHandler {
	return &ComplianceHandler{
		service: service,
		logger:  logger,
	}
}

// CreateProfile creates a new compliance profile
// @Summary Create compliance profile
// @Tags Compliance
// @Accept json
// @Produce json
// @Param request body models.CreateComplianceProfileRequest true "Profile config"
// @Success 201 {object} models.ComplianceProfile
// @Router /compliance/profiles [post]
func (h *ComplianceHandler) CreateProfile(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	var req models.CreateComplianceProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	profile, err := h.service.CreateProfile(c.Request.Context(), tenantID, &req)
	if err != nil {
		h.logger.Error("Failed to create compliance profile", map[string]interface{}{"error": err.Error()})
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, profile)
}

// GetProfiles retrieves all compliance profiles for the tenant
// @Summary List compliance profiles
// @Tags Compliance
// @Produce json
// @Success 200 {array} models.ComplianceProfile
// @Router /compliance/profiles [get]
func (h *ComplianceHandler) GetProfiles(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	profiles, err := h.service.GetProfiles(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Error("Failed to get compliance profiles", map[string]interface{}{"error": err.Error()})
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"profiles": profiles})
}

// GetProfile retrieves a specific compliance profile
// @Summary Get compliance profile
// @Tags Compliance
// @Produce json
// @Param profile_id path string true "Profile ID"
// @Success 200 {object} models.ComplianceProfile
// @Router /compliance/profiles/{profile_id} [get]
func (h *ComplianceHandler) GetProfile(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	profileID, err := uuid.Parse(c.Param("profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid profile_id"})
		return
	}

	profile, err := h.service.GetProfile(c.Request.Context(), tenantID, profileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// ScanForPII scans content for PII
// @Summary Scan content for PII
// @Tags Compliance
// @Accept json
// @Produce json
// @Param request body models.ScanForPIIRequest true "Content to scan"
// @Success 200 {object} models.PIIScanResult
// @Router /compliance/pii/scan [post]
func (h *ComplianceHandler) ScanForPII(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	var req models.ScanForPIIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.ScanForPII(c.Request.Context(), tenantID, &req)
	if err != nil {
		h.logger.Error("Failed to scan for PII", map[string]interface{}{"error": err.Error()})
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// GenerateReport generates a compliance report
// @Summary Generate compliance report
// @Tags Compliance
// @Accept json
// @Produce json
// @Param request body models.GenerateReportRequest true "Report request"
// @Success 202 {object} models.ComplianceReport
// @Router /compliance/reports [post]
func (h *ComplianceHandler) GenerateReport(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	var req models.GenerateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	report, err := h.service.GenerateReport(c.Request.Context(), tenantID, &req)
	if err != nil {
		h.logger.Error("Failed to generate report", map[string]interface{}{"error": err.Error()})
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusAccepted, report)
}

// GetReports retrieves compliance reports for the tenant
// @Summary List compliance reports
// @Tags Compliance
// @Produce json
// @Success 200 {array} models.ComplianceReport
// @Router /compliance/reports [get]
func (h *ComplianceHandler) GetReports(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	reports, err := h.service.GetReports(c.Request.Context(), tenantID, 20)
	if err != nil {
		h.logger.Error("Failed to get reports", map[string]interface{}{"error": err.Error()})
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"reports": reports})
}

// GetReport retrieves a specific compliance report
// @Summary Get compliance report
// @Tags Compliance
// @Produce json
// @Param report_id path string true "Report ID"
// @Success 200 {object} models.ComplianceReport
// @Router /compliance/reports/{report_id} [get]
func (h *ComplianceHandler) GetReport(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	reportID, err := uuid.Parse(c.Param("report_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid report_id"})
		return
	}

	report, err := h.service.GetReport(c.Request.Context(), tenantID, reportID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "report not found"})
		return
	}

	c.JSON(http.StatusOK, report)
}

// GetFindings retrieves findings for a report
// @Summary Get report findings
// @Tags Compliance
// @Produce json
// @Param report_id path string true "Report ID"
// @Success 200 {array} models.ComplianceFinding
// @Router /compliance/reports/{report_id}/findings [get]
func (h *ComplianceHandler) GetFindings(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	reportID, err := uuid.Parse(c.Param("report_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid report_id"})
		return
	}

	findings, err := h.service.GetFindings(c.Request.Context(), tenantID, reportID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"findings": findings})
}

// CreateDSR creates a data subject request
// @Summary Create data subject request
// @Tags Compliance
// @Accept json
// @Produce json
// @Param request body models.CreateDSRRequest true "DSR request"
// @Success 201 {object} models.DataSubjectRequest
// @Router /compliance/dsr [post]
func (h *ComplianceHandler) CreateDSR(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	var req models.CreateDSRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dsr, err := h.service.CreateDSR(c.Request.Context(), tenantID, &req)
	if err != nil {
		h.logger.Error("Failed to create DSR", map[string]interface{}{"error": err.Error()})
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, dsr)
}

// GetDSRs retrieves data subject requests for the tenant
// @Summary List data subject requests
// @Tags Compliance
// @Produce json
// @Param status query string false "Filter by status"
// @Success 200 {array} models.DataSubjectRequest
// @Router /compliance/dsr [get]
func (h *ComplianceHandler) GetDSRs(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}
	status := c.Query("status")

	dsrs, err := h.service.GetDSRs(c.Request.Context(), tenantID, status)
	if err != nil {
		h.logger.Error("Failed to get DSRs", map[string]interface{}{"error": err.Error()})
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"requests": dsrs})
}

// GetDSR retrieves a specific data subject request
// @Summary Get data subject request
// @Tags Compliance
// @Produce json
// @Param dsr_id path string true "DSR ID"
// @Success 200 {object} models.DataSubjectRequest
// @Router /compliance/dsr/{dsr_id} [get]
func (h *ComplianceHandler) GetDSR(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	dsrID, err := uuid.Parse(c.Param("dsr_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid dsr_id"})
		return
	}

	dsr, err := h.service.GetDSR(c.Request.Context(), tenantID, dsrID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "DSR not found"})
		return
	}

	c.JSON(http.StatusOK, dsr)
}

// ProcessDSR processes a data subject request
// @Summary Process data subject request
// @Tags Compliance
// @Produce json
// @Param dsr_id path string true "DSR ID"
// @Success 200 {object} map[string]interface{}
// @Router /compliance/dsr/{dsr_id}/process [post]
func (h *ComplianceHandler) ProcessDSR(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	dsrID, err := uuid.Parse(c.Param("dsr_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid dsr_id"})
		return
	}

	if err := h.service.ProcessDSR(c.Request.Context(), tenantID, dsrID); err != nil {
		h.logger.Error("Failed to process DSR", map[string]interface{}{"error": err.Error()})
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "DSR processed successfully"})
}

// GetAuditLogs retrieves audit logs
// @Summary Get audit logs
// @Tags Compliance
// @Produce json
// @Param action query string false "Filter by action"
// @Param resource_type query string false "Filter by resource type"
// @Param limit query int false "Limit results"
// @Success 200 {array} models.ComplianceAuditLog
// @Router /compliance/audit-logs [get]
func (h *ComplianceHandler) GetAuditLogs(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	query := &models.AuditLogQuery{
		TenantID:     tenantID,
		Action:       c.Query("action"),
		ResourceType: c.Query("resource_type"),
		Limit:        100,
	}

	// Validate enum-style parameters contain only safe characters
	validParam := func(s string) bool {
		for _, r := range s {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
				return false
			}
		}
		return true
	}
	if query.Action != "" && !validParam(query.Action) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action parameter"})
		return
	}
	if query.ResourceType != "" && !validParam(query.ResourceType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resource_type parameter"})
		return
	}

	logs, err := h.service.QueryAuditLogs(c.Request.Context(), query)
	if err != nil {
		h.logger.Error("Failed to get audit logs", map[string]interface{}{"error": err.Error()})
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

// GetDashboard retrieves the compliance dashboard
// @Summary Get compliance dashboard
// @Tags Compliance
// @Produce json
// @Success 200 {object} models.ComplianceDashboard
// @Router /compliance/dashboard [get]
func (h *ComplianceHandler) GetDashboard(c *gin.Context) {
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

// CreateRetentionPolicy creates a data retention policy
// @Summary Create retention policy
// @Tags Compliance
// @Accept json
// @Produce json
// @Param request body models.CreateRetentionPolicyRequest true "Retention policy"
// @Success 201 {object} models.DataRetentionPolicy
// @Router /compliance/retention-policies [post]
func (h *ComplianceHandler) CreateRetentionPolicy(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	var req models.CreateRetentionPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	policy, err := h.service.CreateRetentionPolicy(c.Request.Context(), tenantID, &req)
	if err != nil {
		h.logger.Error("Failed to create retention policy", map[string]interface{}{"error": err.Error()})
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, policy)
}

// GetRetentionPolicies retrieves retention policies for the tenant
// @Summary List retention policies
// @Tags Compliance
// @Produce json
// @Success 200 {array} models.DataRetentionPolicy
// @Router /compliance/retention-policies [get]
func (h *ComplianceHandler) GetRetentionPolicies(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	policies, err := h.service.GetRetentionPolicies(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Error("Failed to get retention policies", map[string]interface{}{"error": err.Error()})
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"policies": policies})
}

// GetSupportedFrameworks returns supported compliance frameworks
// @Summary Get supported frameworks
// @Tags Compliance
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /compliance/frameworks [get]
func (h *ComplianceHandler) GetSupportedFrameworks(c *gin.Context) {
	frameworks := []map[string]interface{}{
		{"code": "soc2", "name": "SOC 2", "description": "Service Organization Control 2"},
		{"code": "hipaa", "name": "HIPAA", "description": "Health Insurance Portability and Accountability Act"},
		{"code": "gdpr", "name": "GDPR", "description": "General Data Protection Regulation"},
		{"code": "pci_dss", "name": "PCI DSS", "description": "Payment Card Industry Data Security Standard"},
		{"code": "ccpa", "name": "CCPA", "description": "California Consumer Privacy Act"},
	}

	c.JSON(http.StatusOK, gin.H{"frameworks": frameworks})
}
