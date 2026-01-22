package compliancecenter

import (
	"context"
	"errors"
	"time"
)

var (
	ErrFrameworkNotFound   = errors.New("compliance framework not found")
	ErrControlNotFound     = errors.New("control not found")
	ErrReportNotFound      = errors.New("report not found")
	ErrPolicyNotFound      = errors.New("policy not found")
	ErrAssessmentNotFound  = errors.New("assessment not found")
	ErrFrameworkNotEnabled = errors.New("compliance framework not enabled for tenant")
	ErrInvalidDateRange    = errors.New("invalid date range")
)

// Repository defines the interface for compliance data storage
type Repository interface {
	// Templates
	GetTemplate(ctx context.Context, framework ComplianceFramework) (*ComplianceTemplate, error)
	ListTemplates(ctx context.Context) ([]ComplianceTemplate, error)
	SaveTemplate(ctx context.Context, template *ComplianceTemplate) error

	// Tenant Compliance
	GetTenantCompliance(ctx context.Context, tenantID string) (*TenantCompliance, error)
	SaveTenantCompliance(ctx context.Context, compliance *TenantCompliance) error
	UpdateTenantCompliance(ctx context.Context, compliance *TenantCompliance) error

	// Assessments
	CreateAssessment(ctx context.Context, assessment *ControlAssessment) error
	GetAssessment(ctx context.Context, assessmentID string) (*ControlAssessment, error)
	UpdateAssessment(ctx context.Context, assessment *ControlAssessment) error
	ListAssessments(ctx context.Context, tenantID string, framework ComplianceFramework) ([]ControlAssessment, error)
	GetLatestAssessment(ctx context.Context, tenantID, controlID string) (*ControlAssessment, error)
	GetAssessmentsForReview(ctx context.Context, tenantID string, before time.Time) ([]ControlAssessment, error)

	// Reports
	CreateReport(ctx context.Context, report *ComplianceReport) error
	GetReport(ctx context.Context, reportID string) (*ComplianceReport, error)
	ListReports(ctx context.Context, tenantID string, framework *ComplianceFramework, limit int) ([]ComplianceReport, int, error)
	DeleteReport(ctx context.Context, reportID string) error

	// Audit Logs
	CreateAuditLog(ctx context.Context, entry *AuditLogEntry) error
	ListAuditLogs(ctx context.Context, tenantID string, filters *AuditLogFilters) ([]AuditLogEntry, int, error)
	GetAuditLogStats(ctx context.Context, tenantID string, period string) (map[string]interface{}, error)

	// Policy Violations
	CreateViolation(ctx context.Context, violation *PolicyViolation) error
	GetViolation(ctx context.Context, violationID string) (*PolicyViolation, error)
	ListViolations(ctx context.Context, tenantID string, limit int) ([]PolicyViolation, error)
	UpdateViolation(ctx context.Context, violation *PolicyViolation) error
	GetViolationStats(ctx context.Context, tenantID string, period string) (map[string]int, error)

	// Data Retention
	GetRetentionPolicy(ctx context.Context, tenantID, dataType string) (*DataRetentionPolicy, error)
	SaveRetentionPolicy(ctx context.Context, policy *DataRetentionPolicy) error
	ListRetentionPolicies(ctx context.Context, tenantID string) ([]DataRetentionPolicy, error)

	// Custom Policies
	CreatePolicy(ctx context.Context, tenantID string, policy *PolicyTemplate) error
	GetPolicy(ctx context.Context, tenantID, policyID string) (*PolicyTemplate, error)
	ListPolicies(ctx context.Context, tenantID string) ([]PolicyTemplate, error)
	UpdatePolicy(ctx context.Context, tenantID string, policy *PolicyTemplate) error
	DeletePolicy(ctx context.Context, tenantID, policyID string) error
}

// ReportGenerator defines the interface for generating compliance reports
type ReportGenerator interface {
	Generate(ctx context.Context, tenantID string, req *GenerateReportRequest, assessments []ControlAssessment) (*ComplianceReport, error)
	ExportPDF(ctx context.Context, report *ComplianceReport) ([]byte, error)
	ExportCSV(ctx context.Context, report *ComplianceReport) ([]byte, error)
	ExportJSON(ctx context.Context, report *ComplianceReport) ([]byte, error)
}

// PolicyEvaluator defines the interface for evaluating compliance policies
type PolicyEvaluator interface {
	Evaluate(ctx context.Context, policy *PolicyTemplate, resource interface{}) (bool, []PolicyViolation, error)
	ValidateRule(rule *PolicyRule) error
}

// EvidenceCollector defines the interface for collecting compliance evidence
type EvidenceCollector interface {
	CollectAutomated(ctx context.Context, tenantID string, checkType string) ([]Evidence, error)
	ValidateEvidence(evidence *Evidence) error
	StoreEvidence(ctx context.Context, evidence *Evidence) (string, error)
}

// ControlChecker defines the interface for automated control checks
type ControlChecker interface {
	RunCheck(ctx context.Context, tenantID string, check *ControlCheck) (ControlStatus, []Finding, error)
	GetSupportedChecks() []string
}

// Notifier defines the interface for compliance notifications
type Notifier interface {
	NotifyViolation(ctx context.Context, violation *PolicyViolation) error
	NotifyReviewDue(ctx context.Context, assessment *ControlAssessment) error
	NotifyReportReady(ctx context.Context, report *ComplianceReport) error
}

// DataManager defines the interface for compliance data management
type DataManager interface {
	ApplyRetentionPolicy(ctx context.Context, policy *DataRetentionPolicy) error
	ArchiveData(ctx context.Context, tenantID, dataType string, before time.Time) (int64, error)
	DeleteExpiredData(ctx context.Context, tenantID, dataType string) (int64, error)
	ExportTenantData(ctx context.Context, tenantID string) ([]byte, error) // GDPR data export
}
