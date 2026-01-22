package compliancecenter

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for compliance center
type Handler struct {
	service *Service
}

// NewHandler creates a new handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers HTTP routes
func (h *Handler) RegisterRoutes(r gin.IRouter) {
	compliance := r.Group("/compliance")
	{
		// Dashboard
		compliance.GET("/dashboard", h.GetDashboard)

		// Templates
		compliance.GET("/templates", h.ListTemplates)
		compliance.GET("/templates/:framework", h.GetTemplate)

		// Frameworks
		compliance.POST("/frameworks", h.EnableFramework)
		compliance.DELETE("/frameworks/:framework", h.DisableFramework)
		compliance.GET("/settings", h.GetSettings)

		// Assessments
		compliance.POST("/assessments/:framework/:controlId", h.AssessControl)
		compliance.GET("/assessments/:framework", h.GetAssessments)
		compliance.POST("/assessments/:framework/run-checks", h.RunAutomatedChecks)

		// Reports
		compliance.POST("/reports", h.GenerateReport)
		compliance.GET("/reports", h.ListReports)
		compliance.GET("/reports/:id", h.GetReport)
		compliance.GET("/reports/:id/download", h.DownloadReport)

		// Audit Logs
		compliance.GET("/audit-logs", h.GetAuditLogs)

		// Policies
		compliance.POST("/policies", h.CreatePolicy)
		compliance.GET("/policies", h.ListPolicies)
		compliance.GET("/policies/:id", h.GetPolicy)
		compliance.PUT("/policies/:id", h.UpdatePolicy)
		compliance.DELETE("/policies/:id", h.DeletePolicy)

		// Data Management
		compliance.POST("/retention-policies", h.SetRetentionPolicy)
		compliance.GET("/retention-policies", h.ListRetentionPolicies)
		compliance.POST("/export-data", h.ExportData)
	}
}

// GetDashboard retrieves the compliance dashboard
// @Summary Get compliance dashboard
// @Tags compliance
// @Produce json
// @Success 200 {object} ComplianceDashboard
// @Router /compliance/dashboard [get]
func (h *Handler) GetDashboard(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	dashboard, err := h.service.GetDashboard(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

// ListTemplates lists available compliance templates
// @Summary List compliance templates
// @Tags compliance
// @Produce json
// @Success 200 {array} ComplianceTemplate
// @Router /compliance/templates [get]
func (h *Handler) ListTemplates(c *gin.Context) {
	templates := h.service.ListTemplates()
	c.JSON(http.StatusOK, templates)
}

// GetTemplate retrieves a compliance template
// @Summary Get compliance template
// @Tags compliance
// @Produce json
// @Param framework path string true "Framework ID"
// @Success 200 {object} ComplianceTemplate
// @Failure 404 {object} ErrorResponse
// @Router /compliance/templates/{framework} [get]
func (h *Handler) GetTemplate(c *gin.Context) {
	framework := ComplianceFramework(c.Param("framework"))

	template, err := h.service.GetTemplate(framework)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "framework not found"})
		return
	}

	c.JSON(http.StatusOK, template)
}

// EnableFramework enables a compliance framework
// @Summary Enable compliance framework
// @Tags compliance
// @Accept json
// @Produce json
// @Param request body EnableFrameworkRequest true "Enable request"
// @Success 201 {object} TenantCompliance
// @Failure 400 {object} ErrorResponse
// @Router /compliance/frameworks [post]
func (h *Handler) EnableFramework(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	var req EnableFrameworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	compliance, err := h.service.EnableFramework(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, compliance)
}

// DisableFramework disables a compliance framework
// @Summary Disable compliance framework
// @Tags compliance
// @Param framework path string true "Framework ID"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Router /compliance/frameworks/{framework} [delete]
func (h *Handler) DisableFramework(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	framework := ComplianceFramework(c.Param("framework"))

	err := h.service.DisableFramework(c.Request.Context(), tenantID, framework)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetSettings retrieves tenant compliance settings
// @Summary Get compliance settings
// @Tags compliance
// @Produce json
// @Success 200 {object} TenantCompliance
// @Router /compliance/settings [get]
func (h *Handler) GetSettings(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	compliance, err := h.service.GetTenantCompliance(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusOK, &TenantCompliance{TenantID: tenantID})
		return
	}

	c.JSON(http.StatusOK, compliance)
}

// AssessControl records a control assessment
// @Summary Assess a control
// @Tags compliance
// @Accept json
// @Produce json
// @Param framework path string true "Framework ID"
// @Param controlId path string true "Control ID"
// @Param request body AssessControlRequest true "Assessment request"
// @Success 201 {object} ControlAssessment
// @Failure 400 {object} ErrorResponse
// @Router /compliance/assessments/{framework}/{controlId} [post]
func (h *Handler) AssessControl(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	framework := ComplianceFramework(c.Param("framework"))
	controlID := c.Param("controlId")
	userID := c.GetString("user_id")
	if userID == "" {
		userID = "unknown"
	}

	var req AssessControlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	assessment, err := h.service.AssessControl(c.Request.Context(), tenantID, controlID, framework, &req, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, assessment)
}

// GetAssessments retrieves assessments for a framework
// @Summary Get assessments
// @Tags compliance
// @Produce json
// @Param framework path string true "Framework ID"
// @Success 200 {array} ControlAssessment
// @Router /compliance/assessments/{framework} [get]
func (h *Handler) GetAssessments(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	framework := ComplianceFramework(c.Param("framework"))

	assessments, err := h.service.GetAssessments(c.Request.Context(), tenantID, framework)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, assessments)
}

// RunAutomatedChecks runs automated compliance checks
// @Summary Run automated checks
// @Tags compliance
// @Produce json
// @Param framework path string true "Framework ID"
// @Success 200 {array} ControlAssessment
// @Router /compliance/assessments/{framework}/run-checks [post]
func (h *Handler) RunAutomatedChecks(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	framework := ComplianceFramework(c.Param("framework"))

	assessments, err := h.service.RunAutomatedChecks(c.Request.Context(), tenantID, framework)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, assessments)
}

// GenerateReport generates a compliance report
// @Summary Generate compliance report
// @Tags compliance
// @Accept json
// @Produce json
// @Param request body GenerateReportRequest true "Report request"
// @Success 201 {object} ComplianceReport
// @Failure 400 {object} ErrorResponse
// @Router /compliance/reports [post]
func (h *Handler) GenerateReport(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	var req GenerateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	report, err := h.service.GenerateReport(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, report)
}

// ListReports lists compliance reports
// @Summary List compliance reports
// @Tags compliance
// @Produce json
// @Param framework query string false "Filter by framework"
// @Param limit query int false "Limit results"
// @Success 200 {object} ListReportsResponse
// @Router /compliance/reports [get]
func (h *Handler) ListReports(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	var framework *ComplianceFramework
	if f := c.Query("framework"); f != "" {
		fw := ComplianceFramework(f)
		framework = &fw
	}

	limit := 20
	if l, _ := strconv.Atoi(c.Query("limit")); l > 0 && l <= 100 {
		limit = l
	}

	response, err := h.service.ListReports(c.Request.Context(), tenantID, framework, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetReport retrieves a compliance report
// @Summary Get compliance report
// @Tags compliance
// @Produce json
// @Param id path string true "Report ID"
// @Success 200 {object} ComplianceReport
// @Failure 404 {object} ErrorResponse
// @Router /compliance/reports/{id} [get]
func (h *Handler) GetReport(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	reportID := c.Param("id")

	report, err := h.service.GetReport(c.Request.Context(), tenantID, reportID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "report not found"})
		return
	}

	c.JSON(http.StatusOK, report)
}

// DownloadReport downloads a compliance report
// @Summary Download compliance report
// @Tags compliance
// @Produce application/pdf,application/json,text/csv
// @Param id path string true "Report ID"
// @Param format query string false "Format (pdf, json, csv)"
// @Success 200
// @Failure 404 {object} ErrorResponse
// @Router /compliance/reports/{id}/download [get]
func (h *Handler) DownloadReport(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	reportID := c.Param("id")
	format := c.DefaultQuery("format", "pdf")

	report, err := h.service.GetReport(c.Request.Context(), tenantID, reportID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "report not found"})
		return
	}

	// In a real implementation, this would use the generator to export
	switch format {
	case "json":
		c.JSON(http.StatusOK, report)
	case "csv":
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment; filename=compliance-report.csv")
		c.String(http.StatusOK, "Control ID,Status,Score\n")
	default:
		// PDF export would be implemented with a PDF generator
		c.JSON(http.StatusOK, report)
	}
}

// GetAuditLogs retrieves audit logs
// @Summary Get audit logs
// @Tags compliance
// @Produce json
// @Param actor query string false "Filter by actor"
// @Param action query string false "Filter by action"
// @Param resource query string false "Filter by resource"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Success 200 {object} ListAuditLogsResponse
// @Router /compliance/audit-logs [get]
func (h *Handler) GetAuditLogs(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	filters := &AuditLogFilters{
		Page:     1,
		PageSize: 50,
	}

	if actor := c.Query("actor"); actor != "" {
		filters.Actor = actor
	}
	if action := c.Query("action"); action != "" {
		filters.Action = action
	}
	if resource := c.Query("resource"); resource != "" {
		filters.Resource = resource
	}
	if result := c.Query("result"); result != "" {
		filters.Result = result
	}
	if page, _ := strconv.Atoi(c.Query("page")); page > 0 {
		filters.Page = page
	}
	if pageSize, _ := strconv.Atoi(c.Query("page_size")); pageSize > 0 && pageSize <= 100 {
		filters.PageSize = pageSize
	}

	response, err := h.service.GetAuditLogs(c.Request.Context(), tenantID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// CreatePolicy creates a custom policy
// @Summary Create custom policy
// @Tags compliance
// @Accept json
// @Produce json
// @Param request body CreatePolicyRequest true "Policy request"
// @Success 201 {object} PolicyTemplate
// @Failure 400 {object} ErrorResponse
// @Router /compliance/policies [post]
func (h *Handler) CreatePolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	var req CreatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	policy := &PolicyTemplate{
		Name:        req.Name,
		Description: req.Description,
		ControlIDs:  req.ControlIDs,
		Rules:       req.Rules,
		DefaultMode: req.EnforcementMode,
	}

	err := h.service.repo.CreatePolicy(c.Request.Context(), tenantID, policy)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, policy)
}

// ListPolicies lists custom policies
// @Summary List custom policies
// @Tags compliance
// @Produce json
// @Success 200 {array} PolicyTemplate
// @Router /compliance/policies [get]
func (h *Handler) ListPolicies(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	policies, err := h.service.repo.ListPolicies(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, policies)
}

// GetPolicy retrieves a custom policy
// @Summary Get custom policy
// @Tags compliance
// @Produce json
// @Param id path string true "Policy ID"
// @Success 200 {object} PolicyTemplate
// @Failure 404 {object} ErrorResponse
// @Router /compliance/policies/{id} [get]
func (h *Handler) GetPolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	policyID := c.Param("id")

	policy, err := h.service.repo.GetPolicy(c.Request.Context(), tenantID, policyID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "policy not found"})
		return
	}

	c.JSON(http.StatusOK, policy)
}

// UpdatePolicy updates a custom policy
// @Summary Update custom policy
// @Tags compliance
// @Accept json
// @Produce json
// @Param id path string true "Policy ID"
// @Param request body PolicyTemplate true "Policy update"
// @Success 200 {object} PolicyTemplate
// @Failure 400 {object} ErrorResponse
// @Router /compliance/policies/{id} [put]
func (h *Handler) UpdatePolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	policyID := c.Param("id")

	var policy PolicyTemplate
	if err := c.ShouldBindJSON(&policy); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	policy.ID = policyID
	err := h.service.repo.UpdatePolicy(c.Request.Context(), tenantID, &policy)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, policy)
}

// DeletePolicy deletes a custom policy
// @Summary Delete custom policy
// @Tags compliance
// @Param id path string true "Policy ID"
// @Success 204
// @Failure 404 {object} ErrorResponse
// @Router /compliance/policies/{id} [delete]
func (h *Handler) DeletePolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	policyID := c.Param("id")

	err := h.service.repo.DeletePolicy(c.Request.Context(), tenantID, policyID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "policy not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

// SetRetentionPolicy sets a data retention policy
// @Summary Set retention policy
// @Tags compliance
// @Accept json
// @Produce json
// @Param request body DataRetentionPolicy true "Retention policy"
// @Success 201 {object} DataRetentionPolicy
// @Failure 400 {object} ErrorResponse
// @Router /compliance/retention-policies [post]
func (h *Handler) SetRetentionPolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	var policy DataRetentionPolicy
	if err := c.ShouldBindJSON(&policy); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	err := h.service.SetRetentionPolicy(c.Request.Context(), tenantID, &policy)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, policy)
}

// ListRetentionPolicies lists data retention policies
// @Summary List retention policies
// @Tags compliance
// @Produce json
// @Success 200 {array} DataRetentionPolicy
// @Router /compliance/retention-policies [get]
func (h *Handler) ListRetentionPolicies(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	policies, err := h.service.repo.ListRetentionPolicies(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, policies)
}

// ExportData exports tenant data (GDPR compliance)
// @Summary Export tenant data
// @Tags compliance
// @Produce application/json
// @Success 200
// @Router /compliance/export-data [post]
func (h *Handler) ExportData(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	data, err := h.service.ExportTenantData(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=tenant-data-export.json")
	c.Data(http.StatusOK, "application/json", data)
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}
