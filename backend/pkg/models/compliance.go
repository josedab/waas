package models

import (
	"net"
	"time"

	"github.com/google/uuid"
)

// Compliance framework constants
const (
	ComplianceFrameworkSOC2   = "soc2"
	ComplianceFrameworkHIPAA  = "hipaa"
	ComplianceFrameworkGDPR   = "gdpr"
	ComplianceFrameworkPCIDSS = "pci_dss"
	ComplianceFrameworkCCPA   = "ccpa"
)

// PII category constants
const (
	PIICategoryEmail      = "email"
	PIICategoryPhone      = "phone"
	PIICategorySSN        = "ssn"
	PIICategoryCreditCard = "credit_card"
	PIICategoryName       = "name"
	PIICategoryAddress    = "address"
	PIICategoryDOB        = "dob"
	PIICategoryIPAddress  = "ip_address"
)

// Sensitivity level constants
const (
	SensitivityLow      = "low"
	SensitivityMedium   = "medium"
	SensitivityHigh     = "high"
	SensitivityCritical = "critical"
)

// Report status constants
const (
	ReportStatusPending    = "pending"
	ReportStatusGenerating = "generating"
	ReportStatusCompleted  = "completed"
	ReportStatusFailed     = "failed"
)

// Finding severity constants
const (
	FindingSeverityInfo     = "info"
	FindingSeverityLow      = "low"
	FindingSeverityMedium   = "medium"
	FindingSeverityHigh     = "high"
	FindingSeverityCritical = "critical"
)

// Finding status constants
const (
	FindingStatusOpen         = "open"
	FindingStatusAcknowledged = "acknowledged"
	FindingStatusRemediated   = "remediated"
	FindingStatusAccepted     = "accepted"
)

// Data Subject Request types
const (
	DSRTypeAccess       = "access"
	DSRTypeRectification = "rectification"
	DSRTypeErasure      = "erasure"
	DSRTypePortability  = "portability"
	DSRTypeRestriction  = "restriction"
)

// DSR status constants
const (
	DSRStatusPending    = "pending"
	DSRStatusProcessing = "processing"
	DSRStatusCompleted  = "completed"
	DSRStatusRejected   = "rejected"
)

// ComplianceProfile represents a compliance framework configuration
type ComplianceProfile struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	TenantID    uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	Name        string                 `json:"name" db:"name"`
	Framework   string                 `json:"framework" db:"framework"`
	Description string                 `json:"description,omitempty" db:"description"`
	Enabled     bool                   `json:"enabled" db:"enabled"`
	Settings    map[string]interface{} `json:"settings" db:"settings"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
}

// DataRetentionPolicy defines data retention rules
type DataRetentionPolicy struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	TenantID        uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	ProfileID       *uuid.UUID `json:"profile_id,omitempty" db:"profile_id"`
	Name            string     `json:"name" db:"name"`
	Description     string     `json:"description,omitempty" db:"description"`
	DataCategory    string     `json:"data_category" db:"data_category"`
	RetentionDays   int        `json:"retention_days" db:"retention_days"`
	ArchiveEnabled  bool       `json:"archive_enabled" db:"archive_enabled"`
	ArchiveLocation string     `json:"archive_location,omitempty" db:"archive_location"`
	DeletionMethod  string     `json:"deletion_method" db:"deletion_method"`
	Enabled         bool       `json:"enabled" db:"enabled"`
	LastExecution   *time.Time `json:"last_execution,omitempty" db:"last_execution"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

// PIIDetectionPattern defines a pattern for detecting PII
type PIIDetectionPattern struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	TenantID         *uuid.UUID `json:"tenant_id,omitempty" db:"tenant_id"`
	Name             string     `json:"name" db:"name"`
	PatternType      string     `json:"pattern_type" db:"pattern_type"`
	PatternValue     string     `json:"pattern_value" db:"pattern_value"`
	PIICategory      string     `json:"pii_category" db:"pii_category"`
	SensitivityLevel string     `json:"sensitivity_level" db:"sensitivity_level"`
	Enabled          bool       `json:"enabled" db:"enabled"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
}

// PIIDetection represents a detected PII instance
type PIIDetection struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	TenantID         uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	PatternID        *uuid.UUID `json:"pattern_id,omitempty" db:"pattern_id"`
	SourceType       string     `json:"source_type" db:"source_type"`
	SourceID         uuid.UUID  `json:"source_id" db:"source_id"`
	FieldPath        string     `json:"field_path" db:"field_path"`
	PIICategory      string     `json:"pii_category" db:"pii_category"`
	SensitivityLevel string     `json:"sensitivity_level" db:"sensitivity_level"`
	RedactionApplied bool       `json:"redaction_applied" db:"redaction_applied"`
	DetectedAt       time.Time  `json:"detected_at" db:"detected_at"`
}

// ComplianceAuditLog represents an audit log entry
type ComplianceAuditLog struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	TenantID       uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	ActorID        *uuid.UUID             `json:"actor_id,omitempty" db:"actor_id"`
	ActorType      string                 `json:"actor_type" db:"actor_type"`
	Action         string                 `json:"action" db:"action"`
	ResourceType   string                 `json:"resource_type" db:"resource_type"`
	ResourceID     *uuid.UUID             `json:"resource_id,omitempty" db:"resource_id"`
	Details        map[string]interface{} `json:"details" db:"details"`
	IPAddress      net.IP                 `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent      string                 `json:"user_agent,omitempty" db:"user_agent"`
	Timestamp      time.Time              `json:"timestamp" db:"timestamp"`
	RetentionUntil *time.Time             `json:"retention_until,omitempty" db:"retention_until"`
}

// ComplianceReport represents a compliance report
type ComplianceReport struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	TenantID    uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	ProfileID   *uuid.UUID             `json:"profile_id,omitempty" db:"profile_id"`
	ReportType  string                 `json:"report_type" db:"report_type"`
	Title       string                 `json:"title" db:"title"`
	Description string                 `json:"description,omitempty" db:"description"`
	Status      string                 `json:"status" db:"status"`
	PeriodStart *time.Time             `json:"period_start,omitempty" db:"period_start"`
	PeriodEnd   *time.Time             `json:"period_end,omitempty" db:"period_end"`
	ReportData  map[string]interface{} `json:"report_data" db:"report_data"`
	ArtifactURL string                 `json:"artifact_url,omitempty" db:"artifact_url"`
	GeneratedBy *uuid.UUID             `json:"generated_by,omitempty" db:"generated_by"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
}

// ComplianceFinding represents a finding from a compliance report
type ComplianceFinding struct {
	ID                  uuid.UUID  `json:"id" db:"id"`
	ReportID            uuid.UUID  `json:"report_id" db:"report_id"`
	TenantID            uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	Severity            string     `json:"severity" db:"severity"`
	Category            string     `json:"category" db:"category"`
	Title               string     `json:"title" db:"title"`
	Description         string     `json:"description,omitempty" db:"description"`
	Recommendation      string     `json:"recommendation,omitempty" db:"recommendation"`
	Status              string     `json:"status" db:"status"`
	RemediationDeadline *time.Time `json:"remediation_deadline,omitempty" db:"remediation_deadline"`
	RemediatedAt        *time.Time `json:"remediated_at,omitempty" db:"remediated_at"`
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
}

// DataSubjectRequest represents a GDPR data subject request
type DataSubjectRequest struct {
	ID               uuid.UUID              `json:"id" db:"id"`
	TenantID         uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	RequestType      string                 `json:"request_type" db:"request_type"`
	DataSubjectID    string                 `json:"data_subject_id" db:"data_subject_id"`
	DataSubjectEmail string                 `json:"data_subject_email,omitempty" db:"data_subject_email"`
	Status           string                 `json:"status" db:"status"`
	RequestDetails   map[string]interface{} `json:"request_details" db:"request_details"`
	ResponseData     map[string]interface{} `json:"response_data,omitempty" db:"response_data"`
	Deadline         *time.Time             `json:"deadline,omitempty" db:"deadline"`
	CompletedAt      *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
}

// Request models

// CreateComplianceProfileRequest represents a request to create a compliance profile
type CreateComplianceProfileRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Framework   string                 `json:"framework" binding:"required"`
	Description string                 `json:"description"`
	Settings    map[string]interface{} `json:"settings"`
}

// CreateRetentionPolicyRequest represents a request to create a retention policy
type CreateRetentionPolicyRequest struct {
	ProfileID       string `json:"profile_id"`
	Name            string `json:"name" binding:"required"`
	Description     string `json:"description"`
	DataCategory    string `json:"data_category" binding:"required"`
	RetentionDays   int    `json:"retention_days" binding:"required"`
	ArchiveEnabled  bool   `json:"archive_enabled"`
	ArchiveLocation string `json:"archive_location"`
	DeletionMethod  string `json:"deletion_method"`
}

// CreatePIIPatternRequest represents a request to create a PII detection pattern
type CreatePIIPatternRequest struct {
	Name             string `json:"name" binding:"required"`
	PatternType      string `json:"pattern_type" binding:"required"`
	PatternValue     string `json:"pattern_value" binding:"required"`
	PIICategory      string `json:"pii_category" binding:"required"`
	SensitivityLevel string `json:"sensitivity_level"`
}

// GenerateReportRequest represents a request to generate a compliance report
type GenerateReportRequest struct {
	ProfileID   string     `json:"profile_id"`
	ReportType  string     `json:"report_type" binding:"required"`
	Title       string     `json:"title" binding:"required"`
	Description string     `json:"description"`
	PeriodStart *time.Time `json:"period_start"`
	PeriodEnd   *time.Time `json:"period_end"`
}

// CreateDSRRequest represents a request to create a data subject request
type CreateDSRRequest struct {
	RequestType      string                 `json:"request_type" binding:"required"`
	DataSubjectID    string                 `json:"data_subject_id" binding:"required"`
	DataSubjectEmail string                 `json:"data_subject_email"`
	RequestDetails   map[string]interface{} `json:"request_details"`
}

// ScanForPIIRequest represents a request to scan content for PII
type ScanForPIIRequest struct {
	Content    string `json:"content" binding:"required"`
	SourceType string `json:"source_type"`
	SourceID   string `json:"source_id"`
}

// PIIScanResult represents the result of a PII scan
type PIIScanResult struct {
	Detections    []*PIIDetection `json:"detections"`
	RedactedText  string          `json:"redacted_text,omitempty"`
	TotalFound    int             `json:"total_found"`
	HighSeverity  int             `json:"high_severity"`
	WasRedacted   bool            `json:"was_redacted"`
}

// ComplianceDashboard represents the compliance dashboard data
type ComplianceDashboard struct {
	ActiveProfiles     int                      `json:"active_profiles"`
	OpenFindings       int                      `json:"open_findings"`
	CriticalFindings   int                      `json:"critical_findings"`
	PendingDSRs        int                      `json:"pending_dsrs"`
	PIIDetectionsToday int                      `json:"pii_detections_today"`
	RecentReports      []*ComplianceReport      `json:"recent_reports"`
	FindingsByCategory map[string]int           `json:"findings_by_category"`
	ComplianceScore    float64                  `json:"compliance_score"`
}

// AuditLogQuery represents a query for audit logs
type AuditLogQuery struct {
	TenantID     uuid.UUID  `json:"tenant_id"`
	ActorID      *uuid.UUID `json:"actor_id,omitempty"`
	Action       string     `json:"action,omitempty"`
	ResourceType string     `json:"resource_type,omitempty"`
	StartTime    *time.Time `json:"start_time,omitempty"`
	EndTime      *time.Time `json:"end_time,omitempty"`
	Limit        int        `json:"limit"`
	Offset       int        `json:"offset"`
}
