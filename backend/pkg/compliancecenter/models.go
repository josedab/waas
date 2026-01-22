// Package compliancecenter provides compliance templates, audit reports, and policy enforcement
package compliancecenter

import (
	"encoding/json"
	"time"
)

// ComplianceFramework represents supported compliance frameworks
type ComplianceFramework string

const (
	FrameworkSOC2   ComplianceFramework = "soc2"
	FrameworkHIPAA  ComplianceFramework = "hipaa"
	FrameworkGDPR   ComplianceFramework = "gdpr"
	FrameworkPCIDSS ComplianceFramework = "pci_dss"
	FrameworkISO27001 ComplianceFramework = "iso27001"
	FrameworkCCPA   ComplianceFramework = "ccpa"
	FrameworkFedRAMP ComplianceFramework = "fedramp"
)

// ControlCategory represents categories of compliance controls
type ControlCategory string

const (
	CategoryAccessControl    ControlCategory = "access_control"
	CategoryDataProtection   ControlCategory = "data_protection"
	CategoryLogging          ControlCategory = "logging_monitoring"
	CategoryIncidentResponse ControlCategory = "incident_response"
	CategoryBusinessContinuity ControlCategory = "business_continuity"
	CategoryRiskManagement   ControlCategory = "risk_management"
	CategoryVendorManagement ControlCategory = "vendor_management"
	CategoryEncryption       ControlCategory = "encryption"
	CategoryNetwork          ControlCategory = "network_security"
)

// ControlStatus represents the compliance status of a control
type ControlStatus string

const (
	StatusCompliant    ControlStatus = "compliant"
	StatusNonCompliant ControlStatus = "non_compliant"
	StatusPartial      ControlStatus = "partial"
	StatusNotAssessed  ControlStatus = "not_assessed"
	StatusNotApplicable ControlStatus = "not_applicable"
)

// PolicyEnforcementMode represents how policies are enforced
type PolicyEnforcementMode string

const (
	EnforcementAudit   PolicyEnforcementMode = "audit"   // Log only
	EnforcementWarn    PolicyEnforcementMode = "warn"    // Warn but allow
	EnforcementEnforce PolicyEnforcementMode = "enforce" // Block non-compliant
)

// ComplianceTemplate represents a compliance framework template
type ComplianceTemplate struct {
	ID          string              `json:"id"`
	Framework   ComplianceFramework `json:"framework"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Version     string              `json:"version"`
	Controls    []Control           `json:"controls"`
	Policies    []PolicyTemplate    `json:"policies"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

// Control represents a compliance control requirement
type Control struct {
	ID          string          `json:"id"`
	Code        string          `json:"code"` // e.g., "CC6.1" for SOC2
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Category    ControlCategory `json:"category"`
	Guidance    string          `json:"guidance,omitempty"`
	Checks      []ControlCheck  `json:"checks"`
	References  []string        `json:"references,omitempty"`
	Priority    string          `json:"priority"` // critical, high, medium, low
}

// ControlCheck represents an automated check for a control
type ControlCheck struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	CheckType   string          `json:"check_type"` // automated, manual, hybrid
	Query       string          `json:"query,omitempty"` // Check query/rule
	Expected    json.RawMessage `json:"expected,omitempty"`
	Remediation string          `json:"remediation,omitempty"`
}

// PolicyTemplate represents a policy to enforce
type PolicyTemplate struct {
	ID            string                `json:"id"`
	Name          string                `json:"name"`
	Description   string                `json:"description"`
	ControlIDs    []string              `json:"control_ids"`
	Rules         []PolicyRule          `json:"rules"`
	DefaultMode   PolicyEnforcementMode `json:"default_mode"`
}

// PolicyRule represents a specific policy rule
type PolicyRule struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Condition   string          `json:"condition"` // CEL expression
	Action      string          `json:"action"`    // allow, deny, log
	Message     string          `json:"message"`
	Severity    string          `json:"severity"` // critical, high, medium, low
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// TenantCompliance represents a tenant's compliance configuration
type TenantCompliance struct {
	ID                string                 `json:"id"`
	TenantID          string                 `json:"tenant_id"`
	Frameworks        []ComplianceFramework  `json:"frameworks"`
	EnabledPolicies   []string               `json:"enabled_policies"`
	EnforcementMode   PolicyEnforcementMode  `json:"enforcement_mode"`
	DataResidency     []string               `json:"data_residency"` // Allowed regions
	RetentionDays     int                    `json:"retention_days"`
	EncryptionRequired bool                  `json:"encryption_required"`
	AuditLogEnabled   bool                   `json:"audit_log_enabled"`
	Settings          map[string]interface{} `json:"settings,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

// ControlAssessment represents an assessment of a control
type ControlAssessment struct {
	ID           string        `json:"id"`
	TenantID     string        `json:"tenant_id"`
	ControlID    string        `json:"control_id"`
	Framework    ComplianceFramework `json:"framework"`
	Status       ControlStatus `json:"status"`
	Score        int           `json:"score"` // 0-100
	Evidence     []Evidence    `json:"evidence,omitempty"`
	Findings     []Finding     `json:"findings,omitempty"`
	AssessedBy   string        `json:"assessed_by"`
	AssessedAt   time.Time     `json:"assessed_at"`
	NextReviewAt *time.Time    `json:"next_review_at,omitempty"`
	Notes        string        `json:"notes,omitempty"`
}

// Evidence represents compliance evidence
type Evidence struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // screenshot, log, config, document
	Description string    `json:"description"`
	URL         string    `json:"url,omitempty"`
	Content     string    `json:"content,omitempty"`
	CollectedAt time.Time `json:"collected_at"`
	CollectedBy string    `json:"collected_by"`
}

// Finding represents a compliance finding
type Finding struct {
	ID          string    `json:"id"`
	Severity    string    `json:"severity"` // critical, high, medium, low
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Remediation string    `json:"remediation,omitempty"`
	Status      string    `json:"status"` // open, in_progress, resolved, accepted
	DueDate     *time.Time `json:"due_date,omitempty"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	ResolvedBy  string    `json:"resolved_by,omitempty"`
}

// ComplianceReport represents a generated compliance report
type ComplianceReport struct {
	ID              string              `json:"id"`
	TenantID        string              `json:"tenant_id"`
	Framework       ComplianceFramework `json:"framework"`
	ReportType      string              `json:"report_type"` // full, summary, executive
	Title           string              `json:"title"`
	Period          ReportPeriod        `json:"period"`
	Summary         ReportSummary       `json:"summary"`
	Sections        []ReportSection     `json:"sections"`
	GeneratedAt     time.Time           `json:"generated_at"`
	GeneratedBy     string              `json:"generated_by"`
	ExpiresAt       *time.Time          `json:"expires_at,omitempty"`
	DownloadURL     string              `json:"download_url,omitempty"`
	Format          string              `json:"format"` // pdf, json, csv
}

// ReportPeriod represents the time period for a report
type ReportPeriod struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

// ReportSummary provides high-level compliance metrics
type ReportSummary struct {
	OverallScore       int            `json:"overall_score"` // 0-100
	TotalControls      int            `json:"total_controls"`
	CompliantControls  int            `json:"compliant_controls"`
	NonCompliantControls int          `json:"non_compliant_controls"`
	PartialControls    int            `json:"partial_controls"`
	NotAssessedControls int           `json:"not_assessed_controls"`
	CriticalFindings   int            `json:"critical_findings"`
	HighFindings       int            `json:"high_findings"`
	MediumFindings     int            `json:"medium_findings"`
	LowFindings        int            `json:"low_findings"`
	CategoryScores     map[string]int `json:"category_scores"`
}

// ReportSection represents a section in a compliance report
type ReportSection struct {
	Title       string              `json:"title"`
	Category    ControlCategory     `json:"category,omitempty"`
	Description string              `json:"description,omitempty"`
	Score       int                 `json:"score,omitempty"`
	Assessments []ControlAssessment `json:"assessments,omitempty"`
	Charts      []ChartData         `json:"charts,omitempty"`
}

// ChartData represents data for report visualizations
type ChartData struct {
	Type   string            `json:"type"` // pie, bar, line, gauge
	Title  string            `json:"title"`
	Data   []ChartDataPoint  `json:"data"`
}

// ChartDataPoint represents a single data point
type ChartDataPoint struct {
	Label string      `json:"label"`
	Value interface{} `json:"value"`
	Color string      `json:"color,omitempty"`
}

// AuditLogEntry represents an audit log entry
type AuditLogEntry struct {
	ID           string          `json:"id"`
	TenantID     string          `json:"tenant_id"`
	Timestamp    time.Time       `json:"timestamp"`
	Actor        string          `json:"actor"`
	ActorType    string          `json:"actor_type"` // user, api_key, system
	Action       string          `json:"action"`
	Resource     string          `json:"resource"`
	ResourceID   string          `json:"resource_id,omitempty"`
	Details      json.RawMessage `json:"details,omitempty"`
	IPAddress    string          `json:"ip_address,omitempty"`
	UserAgent    string          `json:"user_agent,omitempty"`
	Result       string          `json:"result"` // success, failure
	ErrorMessage string          `json:"error_message,omitempty"`
}

// PolicyViolation represents a policy violation
type PolicyViolation struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	PolicyID     string    `json:"policy_id"`
	PolicyName   string    `json:"policy_name"`
	RuleID       string    `json:"rule_id"`
	Severity     string    `json:"severity"`
	Resource     string    `json:"resource"`
	ResourceID   string    `json:"resource_id"`
	Description  string    `json:"description"`
	Action       string    `json:"action"` // blocked, warned, logged
	Details      json.RawMessage `json:"details,omitempty"`
	OccurredAt   time.Time `json:"occurred_at"`
	ResolvedAt   *time.Time `json:"resolved_at,omitempty"`
}

// DataRetentionPolicy represents data retention settings
type DataRetentionPolicy struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	DataType        string    `json:"data_type"` // webhook_payload, audit_log, etc.
	RetentionDays   int       `json:"retention_days"`
	ArchiveEnabled  bool      `json:"archive_enabled"`
	ArchiveLocation string    `json:"archive_location,omitempty"`
	DeleteAfterArchive bool   `json:"delete_after_archive"`
	Active          bool      `json:"active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Request/Response types

// EnableFrameworkRequest represents a request to enable a compliance framework
type EnableFrameworkRequest struct {
	Framework       ComplianceFramework   `json:"framework" binding:"required"`
	EnforcementMode PolicyEnforcementMode `json:"enforcement_mode"`
	DataResidency   []string              `json:"data_residency,omitempty"`
	Settings        map[string]interface{} `json:"settings,omitempty"`
}

// GenerateReportRequest represents a request to generate a compliance report
type GenerateReportRequest struct {
	Framework  ComplianceFramework `json:"framework" binding:"required"`
	ReportType string              `json:"report_type"` // full, summary, executive
	Period     *ReportPeriod       `json:"period,omitempty"`
	Format     string              `json:"format"` // pdf, json, csv
	Sections   []string            `json:"sections,omitempty"` // Specific sections to include
}

// AssessControlRequest represents a request to assess a control
type AssessControlRequest struct {
	Status   ControlStatus `json:"status" binding:"required"`
	Evidence []Evidence    `json:"evidence,omitempty"`
	Notes    string        `json:"notes,omitempty"`
}

// CreatePolicyRequest represents a request to create a custom policy
type CreatePolicyRequest struct {
	Name          string                `json:"name" binding:"required"`
	Description   string                `json:"description"`
	ControlIDs    []string              `json:"control_ids,omitempty"`
	Rules         []PolicyRule          `json:"rules" binding:"required"`
	EnforcementMode PolicyEnforcementMode `json:"enforcement_mode"`
}

// AuditLogFilters for querying audit logs
type AuditLogFilters struct {
	Actor      string    `json:"actor,omitempty"`
	Action     string    `json:"action,omitempty"`
	Resource   string    `json:"resource,omitempty"`
	StartTime  time.Time `json:"start_time,omitempty"`
	EndTime    time.Time `json:"end_time,omitempty"`
	Result     string    `json:"result,omitempty"`
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
}

// ListReportsResponse represents paginated reports
type ListReportsResponse struct {
	Reports    []ComplianceReport `json:"reports"`
	Total      int                `json:"total"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	TotalPages int                `json:"total_pages"`
}

// ListAuditLogsResponse represents paginated audit logs
type ListAuditLogsResponse struct {
	Logs       []AuditLogEntry `json:"logs"`
	Total      int             `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages"`
}

// ComplianceDashboard represents the compliance dashboard data
type ComplianceDashboard struct {
	OverallScore       int                       `json:"overall_score"`
	FrameworkScores    map[string]int            `json:"framework_scores"`
	ActiveFrameworks   []ComplianceFramework     `json:"active_frameworks"`
	RecentViolations   []PolicyViolation         `json:"recent_violations"`
	UpcomingReviews    []ControlAssessment       `json:"upcoming_reviews"`
	OpenFindings       int                       `json:"open_findings"`
	CriticalFindings   int                       `json:"critical_findings"`
	LastAssessment     *time.Time                `json:"last_assessment,omitempty"`
	NextScheduledReport *time.Time               `json:"next_scheduled_report,omitempty"`
}
