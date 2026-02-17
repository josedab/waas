package compliancecenter

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Service provides compliance center operations
type Service struct {
	repo        Repository
	generator   ReportGenerator
	evaluator   PolicyEvaluator
	collector   EvidenceCollector
	checker     ControlChecker
	notifier    Notifier
	dataManager DataManager
	templates   map[ComplianceFramework]*ComplianceTemplate
	mu          sync.RWMutex
	config      *ServiceConfig
}

// ServiceConfig holds service configuration
type ServiceConfig struct {
	DefaultRetentionDays    int
	EnableAutoAssessment    bool
	AssessmentIntervalDays  int
	EnablePolicyEnforcement bool
	MaxReportsPerTenant     int
	ReportRetentionDays     int
}

// DefaultServiceConfig returns default configuration
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		DefaultRetentionDays:    365,
		EnableAutoAssessment:    true,
		AssessmentIntervalDays:  90,
		EnablePolicyEnforcement: true,
		MaxReportsPerTenant:     100,
		ReportRetentionDays:     730, // 2 years
	}
}

// NewService creates a new compliance center service
func NewService(repo Repository, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}

	s := &Service{
		repo:      repo,
		templates: make(map[ComplianceFramework]*ComplianceTemplate),
		config:    config,
	}

	// Load built-in templates
	for _, t := range GetBuiltInTemplates() {
		template := t // Create copy
		s.templates[t.Framework] = &template
	}

	return s
}

// SetGenerator sets the report generator
func (s *Service) SetGenerator(generator ReportGenerator) {
	s.generator = generator
}

// SetEvaluator sets the policy evaluator
func (s *Service) SetEvaluator(evaluator PolicyEvaluator) {
	s.evaluator = evaluator
}

// SetCollector sets the evidence collector
func (s *Service) SetCollector(collector EvidenceCollector) {
	s.collector = collector
}

// SetChecker sets the control checker
func (s *Service) SetChecker(checker ControlChecker) {
	s.checker = checker
}

// SetNotifier sets the notifier
func (s *Service) SetNotifier(notifier Notifier) {
	s.notifier = notifier
}

// SetDataManager sets the data manager
func (s *Service) SetDataManager(manager DataManager) {
	s.dataManager = manager
}

// GetTemplate retrieves a compliance template
func (s *Service) GetTemplate(framework ComplianceFramework) (*ComplianceTemplate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	template, ok := s.templates[framework]
	if !ok {
		return nil, ErrFrameworkNotFound
	}
	return template, nil
}

// ListTemplates lists all available compliance templates
func (s *Service) ListTemplates() []ComplianceTemplate {
	s.mu.RLock()
	defer s.mu.RUnlock()

	templates := make([]ComplianceTemplate, 0, len(s.templates))
	for _, t := range s.templates {
		templates = append(templates, *t)
	}
	return templates
}

// EnableFramework enables a compliance framework for a tenant
func (s *Service) EnableFramework(ctx context.Context, tenantID string, req *EnableFrameworkRequest) (*TenantCompliance, error) {
	// Validate framework exists
	if _, err := s.GetTemplate(req.Framework); err != nil {
		return nil, err
	}

	// Get or create tenant compliance
	compliance, err := s.repo.GetTenantCompliance(ctx, tenantID)
	if err != nil {
		// Create new compliance record
		now := time.Now()
		compliance = &TenantCompliance{
			ID:                 uuid.New().String(),
			TenantID:           tenantID,
			Frameworks:         []ComplianceFramework{},
			EnabledPolicies:    []string{},
			EnforcementMode:    EnforcementAudit,
			DataResidency:      []string{},
			RetentionDays:      s.config.DefaultRetentionDays,
			EncryptionRequired: true,
			AuditLogEnabled:    true,
			CreatedAt:          now,
			UpdatedAt:          now,
		}
	}

	// Check if framework already enabled
	for _, f := range compliance.Frameworks {
		if f == req.Framework {
			return compliance, nil // Already enabled
		}
	}

	// Add framework
	compliance.Frameworks = append(compliance.Frameworks, req.Framework)

	// Apply settings
	if req.EnforcementMode != "" {
		compliance.EnforcementMode = req.EnforcementMode
	}
	if len(req.DataResidency) > 0 {
		compliance.DataResidency = req.DataResidency
	}
	if req.Settings != nil {
		compliance.Settings = req.Settings
	}

	compliance.UpdatedAt = time.Now()

	// Save
	if err := s.repo.SaveTenantCompliance(ctx, compliance); err != nil {
		return nil, fmt.Errorf("failed to save compliance settings: %w", err)
	}

	// Enable default policies for framework
	// error intentionally ignored: framework was already validated above; empty template is safe
	template, _ := s.GetTemplate(req.Framework)
	for _, policy := range template.Policies {
		compliance.EnabledPolicies = append(compliance.EnabledPolicies, policy.ID)
	}
	// best-effort: persist enabled policies; compliance object was already created
	_ = s.repo.UpdateTenantCompliance(ctx, compliance)

	// Create initial assessments for all controls
	go s.createInitialAssessments(ctx, tenantID, req.Framework)

	return compliance, nil
}

// createInitialAssessments creates initial assessments for a framework
func (s *Service) createInitialAssessments(ctx context.Context, tenantID string, framework ComplianceFramework) {
	template, err := s.GetTemplate(framework)
	if err != nil {
		return
	}

	now := time.Now()
	for _, control := range template.Controls {
		assessment := &ControlAssessment{
			ID:         uuid.New().String(),
			TenantID:   tenantID,
			ControlID:  control.ID,
			Framework:  framework,
			Status:     StatusNotAssessed,
			Score:      0,
			AssessedBy: "system",
			AssessedAt: now,
		}
		// best-effort: persist initial assessment; partial setup is acceptable
		_ = s.repo.CreateAssessment(ctx, assessment)
	}
}

// DisableFramework disables a compliance framework for a tenant
func (s *Service) DisableFramework(ctx context.Context, tenantID string, framework ComplianceFramework) error {
	compliance, err := s.repo.GetTenantCompliance(ctx, tenantID)
	if err != nil {
		return ErrFrameworkNotEnabled
	}

	// Remove framework
	newFrameworks := make([]ComplianceFramework, 0)
	for _, f := range compliance.Frameworks {
		if f != framework {
			newFrameworks = append(newFrameworks, f)
		}
	}

	if len(newFrameworks) == len(compliance.Frameworks) {
		return ErrFrameworkNotEnabled
	}

	compliance.Frameworks = newFrameworks
	compliance.UpdatedAt = time.Now()

	return s.repo.UpdateTenantCompliance(ctx, compliance)
}

// GetTenantCompliance retrieves tenant compliance settings
func (s *Service) GetTenantCompliance(ctx context.Context, tenantID string) (*TenantCompliance, error) {
	return s.repo.GetTenantCompliance(ctx, tenantID)
}

// AssessControl records an assessment for a control
func (s *Service) AssessControl(ctx context.Context, tenantID, controlID string, framework ComplianceFramework, req *AssessControlRequest, assessedBy string) (*ControlAssessment, error) {
	// Verify framework is enabled
	compliance, err := s.repo.GetTenantCompliance(ctx, tenantID)
	if err != nil {
		return nil, ErrFrameworkNotEnabled
	}

	enabled := false
	for _, f := range compliance.Frameworks {
		if f == framework {
			enabled = true
			break
		}
	}
	if !enabled {
		return nil, ErrFrameworkNotEnabled
	}

	// Calculate score based on status
	score := 0
	switch req.Status {
	case StatusCompliant:
		score = 100
	case StatusPartial:
		score = 50
	case StatusNonCompliant:
		score = 0
	case StatusNotApplicable:
		score = 100 // N/A doesn't affect score negatively
	}

	now := time.Now()
	nextReview := now.AddDate(0, 0, s.config.AssessmentIntervalDays)

	assessment := &ControlAssessment{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		ControlID:    controlID,
		Framework:    framework,
		Status:       req.Status,
		Score:        score,
		Evidence:     req.Evidence,
		AssessedBy:   assessedBy,
		AssessedAt:   now,
		NextReviewAt: &nextReview,
		Notes:        req.Notes,
	}

	if err := s.repo.CreateAssessment(ctx, assessment); err != nil {
		return nil, fmt.Errorf("failed to create assessment: %w", err)
	}

	return assessment, nil
}

// GetAssessments retrieves all assessments for a framework
func (s *Service) GetAssessments(ctx context.Context, tenantID string, framework ComplianceFramework) ([]ControlAssessment, error) {
	return s.repo.ListAssessments(ctx, tenantID, framework)
}

// RunAutomatedChecks runs automated control checks
func (s *Service) RunAutomatedChecks(ctx context.Context, tenantID string, framework ComplianceFramework) ([]ControlAssessment, error) {
	if s.checker == nil {
		return nil, fmt.Errorf("control checker not configured")
	}

	template, err := s.GetTemplate(framework)
	if err != nil {
		return nil, err
	}

	var assessments []ControlAssessment
	now := time.Now()

	for _, control := range template.Controls {
		for _, check := range control.Checks {
			if check.CheckType != "automated" {
				continue
			}

			status, findings, err := s.checker.RunCheck(ctx, tenantID, &check)
			if err != nil {
				continue
			}

			score := 0
			if status == StatusCompliant {
				score = 100
			} else if status == StatusPartial {
				score = 50
			}

			assessment := &ControlAssessment{
				ID:         uuid.New().String(),
				TenantID:   tenantID,
				ControlID:  control.ID,
				Framework:  framework,
				Status:     status,
				Score:      score,
				Findings:   findings,
				AssessedBy: "automated",
				AssessedAt: now,
			}

			if err := s.repo.CreateAssessment(ctx, assessment); err == nil {
				assessments = append(assessments, *assessment)
			}
		}
	}

	return assessments, nil
}

// GenerateReport generates a compliance report
func (s *Service) GenerateReport(ctx context.Context, tenantID string, req *GenerateReportRequest) (*ComplianceReport, error) {
	// Validate framework
	template, err := s.GetTemplate(req.Framework)
	if err != nil {
		return nil, err
	}

	// Get assessments
	assessments, err := s.repo.ListAssessments(ctx, tenantID, req.Framework)
	if err != nil {
		return nil, fmt.Errorf("failed to get assessments: %w", err)
	}

	// Set default period if not specified
	period := req.Period
	if period == nil {
		end := time.Now()
		start := end.AddDate(0, 0, -90) // Last 90 days
		period = &ReportPeriod{StartDate: start, EndDate: end}
	}

	// Set default report type
	reportType := req.ReportType
	if reportType == "" {
		reportType = "full"
	}

	// Calculate summary
	summary := s.calculateSummary(assessments, template.Controls)

	// Build sections
	sections := s.buildReportSections(assessments, template.Controls, req.Sections)

	now := time.Now()
	expiresAt := now.AddDate(0, 0, s.config.ReportRetentionDays)

	report := &ComplianceReport{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Framework:   req.Framework,
		ReportType:  reportType,
		Title:       fmt.Sprintf("%s Compliance Report", template.Name),
		Period:      *period,
		Summary:     summary,
		Sections:    sections,
		GeneratedAt: now,
		GeneratedBy: "system",
		ExpiresAt:   &expiresAt,
		Format:      req.Format,
	}

	// Use custom generator if available
	if s.generator != nil {
		generated, err := s.generator.Generate(ctx, tenantID, req, assessments)
		if err == nil {
			report = generated
		}
	}

	// Save report
	if err := s.repo.CreateReport(ctx, report); err != nil {
		return nil, fmt.Errorf("failed to save report: %w", err)
	}

	// Notify if configured
	if s.notifier != nil {
		go s.notifier.NotifyReportReady(ctx, report)
	}

	return report, nil
}

// calculateSummary calculates report summary metrics
func (s *Service) calculateSummary(assessments []ControlAssessment, controls []Control) ReportSummary {
	summary := ReportSummary{
		TotalControls:  len(controls),
		CategoryScores: make(map[string]int),
	}

	// Create assessment map
	assessmentMap := make(map[string]*ControlAssessment)
	for i := range assessments {
		assessmentMap[assessments[i].ControlID] = &assessments[i]
	}

	totalScore := 0
	assessedCount := 0
	categoryScores := make(map[string][]int)

	for _, control := range controls {
		assessment, ok := assessmentMap[control.ID]
		if !ok {
			summary.NotAssessedControls++
			continue
		}

		switch assessment.Status {
		case StatusCompliant:
			summary.CompliantControls++
		case StatusNonCompliant:
			summary.NonCompliantControls++
		case StatusPartial:
			summary.PartialControls++
		case StatusNotAssessed:
			summary.NotAssessedControls++
		}

		totalScore += assessment.Score
		assessedCount++

		// Track category scores
		cat := string(control.Category)
		categoryScores[cat] = append(categoryScores[cat], assessment.Score)

		// Count findings by severity
		for _, finding := range assessment.Findings {
			switch finding.Severity {
			case "critical":
				summary.CriticalFindings++
			case "high":
				summary.HighFindings++
			case "medium":
				summary.MediumFindings++
			case "low":
				summary.LowFindings++
			}
		}
	}

	// Calculate overall score
	if assessedCount > 0 {
		summary.OverallScore = totalScore / assessedCount
	}

	// Calculate category averages
	for cat, scores := range categoryScores {
		if len(scores) > 0 {
			sum := 0
			for _, s := range scores {
				sum += s
			}
			summary.CategoryScores[cat] = sum / len(scores)
		}
	}

	return summary
}

// buildReportSections builds report sections
func (s *Service) buildReportSections(assessments []ControlAssessment, controls []Control, requestedSections []string) []ReportSection {
	// Create assessment map
	assessmentMap := make(map[string]*ControlAssessment)
	for i := range assessments {
		assessmentMap[assessments[i].ControlID] = &assessments[i]
	}

	// Group controls by category
	categoryControls := make(map[ControlCategory][]Control)
	for _, control := range controls {
		categoryControls[control.Category] = append(categoryControls[control.Category], control)
	}

	var sections []ReportSection

	for category, catControls := range categoryControls {
		// Filter if sections requested
		if len(requestedSections) > 0 {
			found := false
			for _, s := range requestedSections {
				if s == string(category) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		var sectionAssessments []ControlAssessment
		totalScore := 0
		count := 0

		for _, control := range catControls {
			if assessment, ok := assessmentMap[control.ID]; ok {
				sectionAssessments = append(sectionAssessments, *assessment)
				totalScore += assessment.Score
				count++
			}
		}

		avgScore := 0
		if count > 0 {
			avgScore = totalScore / count
		}

		section := ReportSection{
			Title:       formatCategoryName(category),
			Category:    category,
			Description: getCategoryDescription(category),
			Score:       avgScore,
			Assessments: sectionAssessments,
		}

		sections = append(sections, section)
	}

	return sections
}

// GetReport retrieves a compliance report
func (s *Service) GetReport(ctx context.Context, tenantID, reportID string) (*ComplianceReport, error) {
	report, err := s.repo.GetReport(ctx, reportID)
	if err != nil {
		return nil, err
	}
	if report.TenantID != tenantID {
		return nil, ErrReportNotFound
	}
	return report, nil
}

// ListReports lists compliance reports
func (s *Service) ListReports(ctx context.Context, tenantID string, framework *ComplianceFramework, limit int) (*ListReportsResponse, error) {
	if limit <= 0 {
		limit = 20
	}

	reports, total, err := s.repo.ListReports(ctx, tenantID, framework, limit)
	if err != nil {
		return nil, err
	}

	return &ListReportsResponse{
		Reports:    reports,
		Total:      total,
		Page:       1,
		PageSize:   limit,
		TotalPages: (total + limit - 1) / limit,
	}, nil
}

// LogAuditEvent logs an audit event
func (s *Service) LogAuditEvent(ctx context.Context, entry *AuditLogEntry) error {
	entry.ID = uuid.New().String()
	entry.Timestamp = time.Now()
	return s.repo.CreateAuditLog(ctx, entry)
}

// GetAuditLogs retrieves audit logs
func (s *Service) GetAuditLogs(ctx context.Context, tenantID string, filters *AuditLogFilters) (*ListAuditLogsResponse, error) {
	if filters == nil {
		filters = &AuditLogFilters{Page: 1, PageSize: 50}
	}
	if filters.Page <= 0 {
		filters.Page = 1
	}
	if filters.PageSize <= 0 || filters.PageSize > 100 {
		filters.PageSize = 50
	}

	logs, total, err := s.repo.ListAuditLogs(ctx, tenantID, filters)
	if err != nil {
		return nil, err
	}

	return &ListAuditLogsResponse{
		Logs:       logs,
		Total:      total,
		Page:       filters.Page,
		PageSize:   filters.PageSize,
		TotalPages: (total + filters.PageSize - 1) / filters.PageSize,
	}, nil
}

// EvaluatePolicy evaluates a policy against a resource
func (s *Service) EvaluatePolicy(ctx context.Context, tenantID, policyID string, resource interface{}) (bool, []PolicyViolation, error) {
	if s.evaluator == nil {
		return true, nil, nil // Pass if no evaluator configured
	}

	policy, err := s.repo.GetPolicy(ctx, tenantID, policyID)
	if err != nil {
		return true, nil, err
	}

	pass, violations, err := s.evaluator.Evaluate(ctx, policy, resource)
	if err != nil {
		return false, nil, err
	}

	// Record violations
	for i := range violations {
		violations[i].TenantID = tenantID
		violations[i].PolicyID = policyID
		violations[i].PolicyName = policy.Name
		// best-effort: persist violation record; policy evaluation result is still returned
		_ = s.repo.CreateViolation(ctx, &violations[i])
	}

	return pass, violations, nil
}

// GetDashboard retrieves the compliance dashboard data
func (s *Service) GetDashboard(ctx context.Context, tenantID string) (*ComplianceDashboard, error) {
	compliance, err := s.repo.GetTenantCompliance(ctx, tenantID)
	if err != nil {
		return &ComplianceDashboard{}, nil // Return empty dashboard
	}

	dashboard := &ComplianceDashboard{
		ActiveFrameworks: compliance.Frameworks,
		FrameworkScores:  make(map[string]int),
	}

	totalScore := 0
	frameworkCount := 0

	for _, framework := range compliance.Frameworks {
		assessments, err := s.repo.ListAssessments(ctx, tenantID, framework)
		if err != nil {
			continue
		}

		if len(assessments) == 0 {
			continue
		}

		sum := 0
		for _, a := range assessments {
			sum += a.Score
		}
		avgScore := sum / len(assessments)
		dashboard.FrameworkScores[string(framework)] = avgScore
		totalScore += avgScore
		frameworkCount++
	}

	if frameworkCount > 0 {
		dashboard.OverallScore = totalScore / frameworkCount
	}

	// Get recent violations (error ignored: nil results in empty dashboard section)
	violations, _ := s.repo.ListViolations(ctx, tenantID, 5)
	dashboard.RecentViolations = violations

	// Count open findings
	for _, framework := range compliance.Frameworks {
		// error ignored: nil assessments safely skips the inner loops
		assessments, _ := s.repo.ListAssessments(ctx, tenantID, framework)
		for _, a := range assessments {
			for _, f := range a.Findings {
				if f.Status == "open" {
					dashboard.OpenFindings++
					if f.Severity == "critical" {
						dashboard.CriticalFindings++
					}
				}
			}
		}
	}

	// Get upcoming reviews (error ignored: nil results in empty dashboard section)
	reviewDate := time.Now().AddDate(0, 0, 30) // Next 30 days
	upcoming, _ := s.repo.GetAssessmentsForReview(ctx, tenantID, reviewDate)
	dashboard.UpcomingReviews = upcoming

	return dashboard, nil
}

// SetRetentionPolicy sets a data retention policy
func (s *Service) SetRetentionPolicy(ctx context.Context, tenantID string, policy *DataRetentionPolicy) error {
	policy.ID = uuid.New().String()
	policy.TenantID = tenantID
	policy.Active = true
	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()
	return s.repo.SaveRetentionPolicy(ctx, policy)
}

// ExportTenantData exports all tenant data (GDPR compliance)
func (s *Service) ExportTenantData(ctx context.Context, tenantID string) ([]byte, error) {
	if s.dataManager == nil {
		return nil, fmt.Errorf("data manager not configured")
	}
	return s.dataManager.ExportTenantData(ctx, tenantID)
}

// Helper functions

func formatCategoryName(category ControlCategory) string {
	names := map[ControlCategory]string{
		CategoryAccessControl:      "Access Control",
		CategoryDataProtection:     "Data Protection",
		CategoryLogging:            "Logging & Monitoring",
		CategoryIncidentResponse:   "Incident Response",
		CategoryBusinessContinuity: "Business Continuity",
		CategoryRiskManagement:     "Risk Management",
		CategoryVendorManagement:   "Vendor Management",
		CategoryEncryption:         "Encryption",
		CategoryNetwork:            "Network Security",
	}
	if name, ok := names[category]; ok {
		return name
	}
	return string(category)
}

func getCategoryDescription(category ControlCategory) string {
	descriptions := map[ControlCategory]string{
		CategoryAccessControl:      "Controls related to user authentication, authorization, and access management",
		CategoryDataProtection:     "Controls for protecting data confidentiality and integrity",
		CategoryLogging:            "Controls for audit logging, monitoring, and alerting",
		CategoryIncidentResponse:   "Controls for detecting, responding to, and recovering from security incidents",
		CategoryBusinessContinuity: "Controls ensuring business operations can continue during disruptions",
		CategoryRiskManagement:     "Controls for identifying, assessing, and mitigating risks",
		CategoryVendorManagement:   "Controls for managing third-party vendor relationships",
		CategoryEncryption:         "Controls for encryption of data at rest and in transit",
		CategoryNetwork:            "Controls for network security and segmentation",
	}
	if desc, ok := descriptions[category]; ok {
		return desc
	}
	return ""
}

// MarshalJSON implements custom JSON marshaling for ComplianceDashboard
func (d *ComplianceDashboard) MarshalJSON() ([]byte, error) {
	type Alias ComplianceDashboard
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(d),
	})
}
