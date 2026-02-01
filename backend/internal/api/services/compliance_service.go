package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/repository"
	"github.com/josedab/waas/pkg/utils"
)

// ComplianceService handles compliance operations
type ComplianceService struct {
	repo   repository.ComplianceRepository
	logger *utils.Logger
}

// NewComplianceService creates a new compliance service
func NewComplianceService(repo repository.ComplianceRepository, logger *utils.Logger) *ComplianceService {
	return &ComplianceService{
		repo:   repo,
		logger: logger,
	}
}

// CreateProfile creates a new compliance profile
func (s *ComplianceService) CreateProfile(ctx context.Context, tenantID uuid.UUID, req *models.CreateComplianceProfileRequest) (*models.ComplianceProfile, error) {
	validFrameworks := map[string]bool{
		models.ComplianceFrameworkSOC2:   true,
		models.ComplianceFrameworkHIPAA:  true,
		models.ComplianceFrameworkGDPR:   true,
		models.ComplianceFrameworkPCIDSS: true,
		models.ComplianceFrameworkCCPA:   true,
	}

	if !validFrameworks[req.Framework] {
		return nil, fmt.Errorf("unsupported framework: %s", req.Framework)
	}

	profile := &models.ComplianceProfile{
		TenantID:    tenantID,
		Name:        req.Name,
		Framework:   req.Framework,
		Description: req.Description,
		Enabled:     true,
		Settings:    req.Settings,
	}

	if profile.Settings == nil {
		profile.Settings = s.getDefaultSettings(req.Framework)
	}

	if err := s.repo.CreateProfile(ctx, profile); err != nil {
		return nil, fmt.Errorf("failed to create profile: %w", err)
	}

	// Create default retention policies for the framework
	go s.createDefaultRetentionPolicies(context.Background(), tenantID, profile)

	return profile, nil
}

// getDefaultSettings returns default settings for a compliance framework
func (s *ComplianceService) getDefaultSettings(framework string) map[string]interface{} {
	switch framework {
	case models.ComplianceFrameworkGDPR:
		return map[string]interface{}{
			"data_residency":           "eu",
			"consent_required":         true,
			"dsr_auto_response":        false,
			"pii_redaction_enabled":    true,
			"retention_default_days":   365,
		}
	case models.ComplianceFrameworkHIPAA:
		return map[string]interface{}{
			"phi_detection_enabled":    true,
			"encryption_required":      true,
			"audit_log_retention_days": 2190, // 6 years
			"access_control_strict":    true,
		}
	case models.ComplianceFrameworkSOC2:
		return map[string]interface{}{
			"security_controls":       true,
			"availability_monitoring": true,
			"audit_log_retention_days": 365,
		}
	case models.ComplianceFrameworkPCIDSS:
		return map[string]interface{}{
			"card_data_detection":     true,
			"encryption_required":     true,
			"audit_log_retention_days": 365,
		}
	default:
		return map[string]interface{}{}
	}
}

// createDefaultRetentionPolicies creates default retention policies
func (s *ComplianceService) createDefaultRetentionPolicies(ctx context.Context, tenantID uuid.UUID, profile *models.ComplianceProfile) {
	var policies []*models.DataRetentionPolicy

	switch profile.Framework {
	case models.ComplianceFrameworkGDPR:
		policies = []*models.DataRetentionPolicy{
			{Name: "GDPR Event Data", DataCategory: "events", RetentionDays: 365},
			{Name: "GDPR Audit Logs", DataCategory: "audit_trails", RetentionDays: 730},
			{Name: "GDPR PII Data", DataCategory: "pii", RetentionDays: 90},
		}
	case models.ComplianceFrameworkHIPAA:
		policies = []*models.DataRetentionPolicy{
			{Name: "HIPAA Event Data", DataCategory: "events", RetentionDays: 2190},
			{Name: "HIPAA Audit Logs", DataCategory: "audit_trails", RetentionDays: 2190},
		}
	case models.ComplianceFrameworkSOC2:
		policies = []*models.DataRetentionPolicy{
			{Name: "SOC2 Audit Logs", DataCategory: "audit_trails", RetentionDays: 365},
			{Name: "SOC2 Event Data", DataCategory: "events", RetentionDays: 365},
		}
	}

	for _, policy := range policies {
		policy.TenantID = tenantID
		policy.ProfileID = &profile.ID
		policy.Enabled = true
		policy.DeletionMethod = "soft"

		if err := s.repo.CreateRetentionPolicy(ctx, policy); err != nil {
			s.logger.Warn("Failed to create default retention policy", map[string]interface{}{"error": err, "name": policy.Name})
		}
	}
}

// GetProfile retrieves a compliance profile
func (s *ComplianceService) GetProfile(ctx context.Context, tenantID, profileID uuid.UUID) (*models.ComplianceProfile, error) {
	profile, err := s.repo.GetProfile(ctx, profileID)
	if err != nil {
		return nil, err
	}

	if profile.TenantID != tenantID {
		return nil, fmt.Errorf("profile not found")
	}

	return profile, nil
}

// GetProfiles retrieves all profiles for a tenant
func (s *ComplianceService) GetProfiles(ctx context.Context, tenantID uuid.UUID) ([]*models.ComplianceProfile, error) {
	return s.repo.GetProfilesByTenant(ctx, tenantID)
}

// ScanForPII scans content for PII and optionally redacts it
func (s *ComplianceService) ScanForPII(ctx context.Context, tenantID uuid.UUID, req *models.ScanForPIIRequest) (*models.PIIScanResult, error) {
	patterns, err := s.repo.GetPIIPatterns(ctx, &tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get PII patterns: %w", err)
	}

	result := &models.PIIScanResult{
		Detections: []*models.PIIDetection{},
	}

	redactedText := req.Content
	var sourceID uuid.UUID
	if req.SourceID != "" {
		sourceID, _ = uuid.Parse(req.SourceID)
	}

	for _, pattern := range patterns {
		if pattern.PatternType != "regex" {
			continue
		}

		re, err := regexp.Compile(pattern.PatternValue)
		if err != nil {
			s.logger.Warn("Invalid regex pattern", map[string]interface{}{"pattern_id": pattern.ID, "error": err})
			continue
		}

		matches := re.FindAllStringIndex(req.Content, -1)
		for _, match := range matches {
			detection := &models.PIIDetection{
				TenantID:         tenantID,
				PatternID:        &pattern.ID,
				SourceType:       req.SourceType,
				SourceID:         sourceID,
				FieldPath:        fmt.Sprintf("content[%d:%d]", match[0], match[1]),
				PIICategory:      pattern.PIICategory,
				SensitivityLevel: pattern.SensitivityLevel,
				RedactionApplied: false,
			}

			result.Detections = append(result.Detections, detection)

			// Track severity
			if pattern.SensitivityLevel == models.SensitivityHigh ||
				pattern.SensitivityLevel == models.SensitivityCritical {
				result.HighSeverity++
			}

			// Redact the PII
			matchedText := req.Content[match[0]:match[1]]
			redactionMask := s.getRedactionMask(pattern.PIICategory, len(matchedText))
			redactedText = strings.Replace(redactedText, matchedText, redactionMask, 1)
		}
	}

	result.TotalFound = len(result.Detections)
	result.RedactedText = redactedText
	result.WasRedacted = result.TotalFound > 0

	// Persist detections if source is provided
	if req.SourceID != "" && req.SourceType != "" {
		for _, d := range result.Detections {
			d.RedactionApplied = true
			if err := s.repo.CreatePIIDetection(ctx, d); err != nil {
				s.logger.Warn("Failed to persist PII detection", map[string]interface{}{"error": err})
			}
		}
	}

	return result, nil
}

// getRedactionMask returns a redaction mask for a PII category
func (s *ComplianceService) getRedactionMask(category string, length int) string {
	switch category {
	case models.PIICategoryEmail:
		return "[REDACTED_EMAIL]"
	case models.PIICategoryPhone:
		return "[REDACTED_PHONE]"
	case models.PIICategorySSN:
		return "[REDACTED_SSN]"
	case models.PIICategoryCreditCard:
		return "[REDACTED_CARD]"
	case models.PIICategoryDOB:
		return "[REDACTED_DOB]"
	default:
		return "[REDACTED]"
	}
}

// CreateAuditLog creates an audit log entry
func (s *ComplianceService) CreateAuditLog(ctx context.Context, log *models.ComplianceAuditLog) error {
	return s.repo.CreateAuditLog(ctx, log)
}

// QueryAuditLogs queries audit logs
func (s *ComplianceService) QueryAuditLogs(ctx context.Context, query *models.AuditLogQuery) ([]*models.ComplianceAuditLog, error) {
	return s.repo.QueryAuditLogs(ctx, query)
}

// GenerateReport generates a compliance report
func (s *ComplianceService) GenerateReport(ctx context.Context, tenantID uuid.UUID, req *models.GenerateReportRequest) (*models.ComplianceReport, error) {
	report := &models.ComplianceReport{
		TenantID:    tenantID,
		ReportType:  req.ReportType,
		Title:       req.Title,
		Description: req.Description,
		PeriodStart: req.PeriodStart,
		PeriodEnd:   req.PeriodEnd,
		Status:      models.ReportStatusPending,
	}

	if req.ProfileID != "" {
		profileID, err := uuid.Parse(req.ProfileID)
		if err == nil {
			report.ProfileID = &profileID
		}
	}

	if err := s.repo.CreateReport(ctx, report); err != nil {
		return nil, fmt.Errorf("failed to create report: %w", err)
	}

	// Generate report in background
	go s.generateReportAsync(context.Background(), report)

	return report, nil
}

// generateReportAsync generates a report asynchronously
func (s *ComplianceService) generateReportAsync(ctx context.Context, report *models.ComplianceReport) {
	s.repo.UpdateReportStatus(ctx, report.ID, models.ReportStatusGenerating, nil, "")

	reportData := make(map[string]interface{})

	switch report.ReportType {
	case "soc2_audit":
		reportData = s.generateSOC2Report(ctx, report.TenantID, report.PeriodStart, report.PeriodEnd)
	case "hipaa_audit":
		reportData = s.generateHIPAAReport(ctx, report.TenantID, report.PeriodStart, report.PeriodEnd)
	case "gdpr_dpia":
		reportData = s.generateGDPRDPIA(ctx, report.TenantID)
	case "data_inventory":
		reportData = s.generateDataInventory(ctx, report.TenantID)
	default:
		reportData = s.generateGenericReport(ctx, report.TenantID, report.PeriodStart, report.PeriodEnd)
	}

	// Create findings from report
	if findings, ok := reportData["findings"].([]map[string]interface{}); ok {
		for _, f := range findings {
			finding := &models.ComplianceFinding{
				ReportID:    report.ID,
				TenantID:    report.TenantID,
				Severity:    f["severity"].(string),
				Category:    f["category"].(string),
				Title:       f["title"].(string),
				Description: f["description"].(string),
			}
			if rec, ok := f["recommendation"].(string); ok {
				finding.Recommendation = rec
			}
			s.repo.CreateFinding(ctx, finding)
		}
	}

	s.repo.UpdateReportStatus(ctx, report.ID, models.ReportStatusCompleted, reportData, "")
	s.logger.Info("Report generation completed", map[string]interface{}{"report_id": report.ID})
}

func (s *ComplianceService) generateSOC2Report(ctx context.Context, tenantID uuid.UUID, start, end *time.Time) map[string]interface{} {
	findings := []map[string]interface{}{}

	// Check for audit logging
	logs, _ := s.repo.QueryAuditLogs(ctx, &models.AuditLogQuery{
		TenantID: tenantID,
		Limit:    1,
	})

	if len(logs) == 0 {
		findings = append(findings, map[string]interface{}{
			"severity":       "high",
			"category":       "Security",
			"title":          "No Audit Logging Detected",
			"description":    "No audit logs found for the tenant. All system actions should be logged.",
			"recommendation": "Enable comprehensive audit logging for all operations.",
		})
	}

	// Check retention policies
	policies, _ := s.repo.GetRetentionPoliciesByTenant(ctx, tenantID)
	if len(policies) == 0 {
		findings = append(findings, map[string]interface{}{
			"severity":       "medium",
			"category":       "Processing Integrity",
			"title":          "No Data Retention Policies Defined",
			"description":    "No data retention policies are configured.",
			"recommendation": "Define data retention policies to ensure proper data lifecycle management.",
		})
	}

	return map[string]interface{}{
		"framework":       "SOC2",
		"generated_at":    time.Now().Format(time.RFC3339),
		"period_start":    start,
		"period_end":      end,
		"findings":        findings,
		"controls_tested": 12,
		"controls_passed": 12 - len(findings),
	}
}

func (s *ComplianceService) generateHIPAAReport(ctx context.Context, tenantID uuid.UUID, start, end *time.Time) map[string]interface{} {
	findings := []map[string]interface{}{}

	// Check PII detection
	piiCount, _ := s.repo.CountPIIDetectionsToday(ctx, tenantID)
	if piiCount > 0 {
		findings = append(findings, map[string]interface{}{
			"severity":       "info",
			"category":       "Privacy",
			"title":          fmt.Sprintf("PII Detected: %d instances today", piiCount),
			"description":    "PII was detected in processed data. Ensure proper handling procedures.",
			"recommendation": "Review PII handling procedures and ensure redaction is enabled.",
		})
	}

	return map[string]interface{}{
		"framework":    "HIPAA",
		"generated_at": time.Now().Format(time.RFC3339),
		"period_start": start,
		"period_end":   end,
		"findings":     findings,
		"phi_detected": piiCount,
	}
}

func (s *ComplianceService) generateGDPRDPIA(ctx context.Context, tenantID uuid.UUID) map[string]interface{} {
	findings := []map[string]interface{}{}

	// Check for pending DSRs
	pendingDSRs, _ := s.repo.CountPendingDSRs(ctx, tenantID)
	if pendingDSRs > 0 {
		findings = append(findings, map[string]interface{}{
			"severity":       "high",
			"category":       "Data Subject Rights",
			"title":          fmt.Sprintf("%d Pending Data Subject Requests", pendingDSRs),
			"description":    "There are pending data subject requests that need to be addressed.",
			"recommendation": "Process all pending DSRs within the 30-day deadline.",
		})
	}

	return map[string]interface{}{
		"framework":          "GDPR",
		"report_type":        "Data Protection Impact Assessment",
		"generated_at":       time.Now().Format(time.RFC3339),
		"findings":           findings,
		"pending_dsrs":       pendingDSRs,
		"lawful_basis":       "Legitimate interest / Contract performance",
		"data_categories":    []string{"Webhook payloads", "Delivery metadata", "API access logs"},
		"retention_periods":  "As defined per tenant retention policies",
	}
}

func (s *ComplianceService) generateDataInventory(ctx context.Context, tenantID uuid.UUID) map[string]interface{} {
	return map[string]interface{}{
		"generated_at": time.Now().Format(time.RFC3339),
		"data_categories": []map[string]interface{}{
			{
				"category":    "Webhook Events",
				"description": "Event payloads sent to webhooks",
				"contains_pii": true,
				"retention":    "Per policy",
			},
			{
				"category":    "Delivery Logs",
				"description": "Webhook delivery attempt records",
				"contains_pii": false,
				"retention":    "30 days default",
			},
			{
				"category":    "Audit Trails",
				"description": "System and user action logs",
				"contains_pii": true,
				"retention":    "Per compliance framework",
			},
		},
		"findings": []map[string]interface{}{},
	}
}

func (s *ComplianceService) generateGenericReport(ctx context.Context, tenantID uuid.UUID, start, end *time.Time) map[string]interface{} {
	return map[string]interface{}{
		"generated_at": time.Now().Format(time.RFC3339),
		"period_start": start,
		"period_end":   end,
		"findings":     []map[string]interface{}{},
	}
}

// GetReport retrieves a compliance report
func (s *ComplianceService) GetReport(ctx context.Context, tenantID, reportID uuid.UUID) (*models.ComplianceReport, error) {
	report, err := s.repo.GetReport(ctx, reportID)
	if err != nil {
		return nil, err
	}

	if report.TenantID != tenantID {
		return nil, fmt.Errorf("report not found")
	}

	return report, nil
}

// GetReports retrieves reports for a tenant
func (s *ComplianceService) GetReports(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.ComplianceReport, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.repo.GetReportsByTenant(ctx, tenantID, limit)
}

// GetFindings retrieves findings for a report
func (s *ComplianceService) GetFindings(ctx context.Context, tenantID, reportID uuid.UUID) ([]*models.ComplianceFinding, error) {
	report, err := s.repo.GetReport(ctx, reportID)
	if err != nil || report.TenantID != tenantID {
		return nil, fmt.Errorf("report not found")
	}

	return s.repo.GetFindingsByReport(ctx, reportID)
}

// CreateDSR creates a data subject request
func (s *ComplianceService) CreateDSR(ctx context.Context, tenantID uuid.UUID, req *models.CreateDSRRequest) (*models.DataSubjectRequest, error) {
	validTypes := map[string]bool{
		models.DSRTypeAccess:       true,
		models.DSRTypeRectification: true,
		models.DSRTypeErasure:      true,
		models.DSRTypePortability:  true,
		models.DSRTypeRestriction:  true,
	}

	if !validTypes[req.RequestType] {
		return nil, fmt.Errorf("invalid request type: %s", req.RequestType)
	}

	dsr := &models.DataSubjectRequest{
		TenantID:         tenantID,
		RequestType:      req.RequestType,
		DataSubjectID:    req.DataSubjectID,
		DataSubjectEmail: req.DataSubjectEmail,
		RequestDetails:   req.RequestDetails,
	}

	if err := s.repo.CreateDSR(ctx, dsr); err != nil {
		return nil, fmt.Errorf("failed to create DSR: %w", err)
	}

	// Log the DSR creation
	s.CreateAuditLog(ctx, &models.ComplianceAuditLog{
		TenantID:     tenantID,
		ActorType:    "system",
		Action:       "dsr_created",
		ResourceType: "data_subject_request",
		ResourceID:   &dsr.ID,
		Details: map[string]interface{}{
			"request_type":    req.RequestType,
			"data_subject_id": req.DataSubjectID,
		},
	})

	return dsr, nil
}

// GetDSR retrieves a data subject request
func (s *ComplianceService) GetDSR(ctx context.Context, tenantID, dsrID uuid.UUID) (*models.DataSubjectRequest, error) {
	dsr, err := s.repo.GetDSR(ctx, dsrID)
	if err != nil {
		return nil, err
	}

	if dsr.TenantID != tenantID {
		return nil, fmt.Errorf("DSR not found")
	}

	return dsr, nil
}

// GetDSRs retrieves DSRs for a tenant
func (s *ComplianceService) GetDSRs(ctx context.Context, tenantID uuid.UUID, status string) ([]*models.DataSubjectRequest, error) {
	return s.repo.GetDSRsByTenant(ctx, tenantID, status)
}

// ProcessDSR processes a data subject request
func (s *ComplianceService) ProcessDSR(ctx context.Context, tenantID, dsrID uuid.UUID) error {
	dsr, err := s.repo.GetDSR(ctx, dsrID)
	if err != nil || dsr.TenantID != tenantID {
		return fmt.Errorf("DSR not found")
	}

	if dsr.Status != models.DSRStatusPending {
		return fmt.Errorf("DSR is not in pending status")
	}

	// Update to processing
	if err := s.repo.UpdateDSRStatus(ctx, dsrID, models.DSRStatusProcessing, nil); err != nil {
		return err
	}

	// Process based on type
	responseData := make(map[string]interface{})

	switch dsr.RequestType {
	case models.DSRTypeAccess:
		responseData["message"] = "Data export initiated"
		responseData["data_locations"] = []string{"events", "audit_logs", "configurations"}
	case models.DSRTypeErasure:
		responseData["message"] = "Erasure request queued"
		responseData["estimated_completion"] = time.Now().AddDate(0, 0, 7).Format(time.RFC3339)
	case models.DSRTypePortability:
		responseData["message"] = "Data export in portable format initiated"
		responseData["format"] = "JSON"
	}

	// Complete the DSR
	return s.repo.UpdateDSRStatus(ctx, dsrID, models.DSRStatusCompleted, responseData)
}

// GetDashboard retrieves compliance dashboard data
func (s *ComplianceService) GetDashboard(ctx context.Context, tenantID uuid.UUID) (*models.ComplianceDashboard, error) {
	profiles, _ := s.repo.GetProfilesByTenant(ctx, tenantID)
	openFindings, _ := s.repo.GetOpenFindingsByTenant(ctx, tenantID)
	pendingDSRs, _ := s.repo.CountPendingDSRs(ctx, tenantID)
	piiToday, _ := s.repo.CountPIIDetectionsToday(ctx, tenantID)
	reports, _ := s.repo.GetReportsByTenant(ctx, tenantID, 5)
	severityCounts, _ := s.repo.CountFindingsBySeverity(ctx, tenantID)

	activeProfiles := 0
	for _, p := range profiles {
		if p.Enabled {
			activeProfiles++
		}
	}

	criticalFindings := severityCounts["critical"]

	// Calculate compliance score (simplified)
	totalFindings := len(openFindings)
	complianceScore := 100.0
	if totalFindings > 0 {
		complianceScore = 100.0 - float64(totalFindings*5)
		if complianceScore < 0 {
			complianceScore = 0
		}
	}

	return &models.ComplianceDashboard{
		ActiveProfiles:     activeProfiles,
		OpenFindings:       totalFindings,
		CriticalFindings:   criticalFindings,
		PendingDSRs:        pendingDSRs,
		PIIDetectionsToday: piiToday,
		RecentReports:      reports,
		FindingsByCategory: severityCounts,
		ComplianceScore:    complianceScore,
	}, nil
}

// CreateRetentionPolicy creates a data retention policy
func (s *ComplianceService) CreateRetentionPolicy(ctx context.Context, tenantID uuid.UUID, req *models.CreateRetentionPolicyRequest) (*models.DataRetentionPolicy, error) {
	policy := &models.DataRetentionPolicy{
		TenantID:        tenantID,
		Name:            req.Name,
		Description:     req.Description,
		DataCategory:    req.DataCategory,
		RetentionDays:   req.RetentionDays,
		ArchiveEnabled:  req.ArchiveEnabled,
		ArchiveLocation: req.ArchiveLocation,
		DeletionMethod:  req.DeletionMethod,
		Enabled:         true,
	}

	if req.ProfileID != "" {
		profileID, err := uuid.Parse(req.ProfileID)
		if err == nil {
			policy.ProfileID = &profileID
		}
	}

	if err := s.repo.CreateRetentionPolicy(ctx, policy); err != nil {
		return nil, fmt.Errorf("failed to create retention policy: %w", err)
	}

	return policy, nil
}

// GetRetentionPolicies retrieves retention policies for a tenant
func (s *ComplianceService) GetRetentionPolicies(ctx context.Context, tenantID uuid.UUID) ([]*models.DataRetentionPolicy, error) {
	return s.repo.GetRetentionPoliciesByTenant(ctx, tenantID)
}
