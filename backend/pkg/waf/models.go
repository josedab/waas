// Package waf provides webhook security scanning, WAF rules, and threat detection
package waf

import (
	"encoding/json"
	"time"
)

// ThreatType represents the type of security threat detected
type ThreatType string

const (
	ThreatTypeXSS               ThreatType = "xss"
	ThreatTypeSQLInjection      ThreatType = "sql_injection"
	ThreatTypePathTraversal     ThreatType = "path_traversal"
	ThreatTypeMaliciousJSON     ThreatType = "malicious_json"
	ThreatTypeOversizedPayload  ThreatType = "oversized_payload"
	ThreatTypeSuspiciousPattern ThreatType = "suspicious_pattern"
)

// ThreatSeverity represents the severity level of a threat
type ThreatSeverity string

const (
	ThreatSeverityInfo     ThreatSeverity = "info"
	ThreatSeverityLow      ThreatSeverity = "low"
	ThreatSeverityMedium   ThreatSeverity = "medium"
	ThreatSeverityHigh     ThreatSeverity = "high"
	ThreatSeverityCritical ThreatSeverity = "critical"
)

// ScanAction represents the action taken after scanning
type ScanAction string

const (
	ScanActionAllow      ScanAction = "allow"
	ScanActionFlag       ScanAction = "flag"
	ScanActionQuarantine ScanAction = "quarantine"
	ScanActionBlock      ScanAction = "block"
)

// WAFRuleType represents the type of WAF rule
type WAFRuleType string

const (
	WAFRuleTypeRegex       WAFRuleType = "regex"
	WAFRuleTypeKeyword     WAFRuleType = "keyword"
	WAFRuleTypeIPList      WAFRuleType = "ip_list"
	WAFRuleTypePayloadSize WAFRuleType = "payload_size"
	WAFRuleTypeRateLimit   WAFRuleType = "rate_limit"
	WAFRuleTypeCustom      WAFRuleType = "custom"
)

// Threat represents a single detected threat
type Threat struct {
	Type           ThreatType     `json:"type"`
	Severity       ThreatSeverity `json:"severity"`
	Description    string         `json:"description"`
	Evidence       string         `json:"evidence,omitempty"`
	Recommendation string         `json:"recommendation,omitempty"`
}

// ScanResult represents the result of scanning a webhook payload
type ScanResult struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	WebhookID  string     `json:"webhook_id"`
	DeliveryID string     `json:"delivery_id,omitempty"`
	Threats    []Threat   `json:"threats"`
	RiskScore  float64    `json:"risk_score"`
	Action     ScanAction `json:"action"`
	ScannedAt  time.Time  `json:"scanned_at"`
	DurationMs int64      `json:"duration_ms"`
}

// WAFRule represents a custom WAF rule for a tenant
type WAFRule struct {
	ID          string      `json:"id"`
	TenantID    string      `json:"tenant_id"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Pattern     string      `json:"pattern"`
	RuleType    WAFRuleType `json:"rule_type"`
	Action      ScanAction  `json:"action"`
	Priority    int         `json:"priority"`
	Enabled     bool        `json:"enabled"`
	HitCount    int64       `json:"hit_count"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// QuarantinedWebhook represents a quarantined webhook delivery
type QuarantinedWebhook struct {
	ID              string          `json:"id"`
	TenantID        string          `json:"tenant_id"`
	WebhookID       string          `json:"webhook_id"`
	Reason          string          `json:"reason"`
	Threats         []Threat        `json:"threats"`
	OriginalPayload json.RawMessage `json:"original_payload,omitempty"`
	QuarantinedAt   time.Time       `json:"quarantined_at"`
	ReviewedAt      *time.Time      `json:"reviewed_at,omitempty"`
	ReviewedBy      string          `json:"reviewed_by,omitempty"`
	Decision        string          `json:"decision,omitempty"` // approve, reject
}

// IPReputation represents the reputation data for an IP address
type IPReputation struct {
	IP          string    `json:"ip"`
	ThreatScore float64   `json:"threat_score"`
	LastSeen    time.Time `json:"last_seen"`
	ReportCount int       `json:"report_count"`
	Categories  []string  `json:"categories,omitempty"`
	Blocked     bool      `json:"blocked"`
}

// SecurityDashboard represents aggregated security metrics
type SecurityDashboard struct {
	TotalScans      int64           `json:"total_scans"`
	ThreatsDetected int64           `json:"threats_detected"`
	ThreatsBlocked  int64           `json:"threats_blocked"`
	QuarantineCount int64           `json:"quarantine_count"`
	TopThreats      []ThreatSummary `json:"top_threats"`
	RecentAlerts    []SecurityAlert `json:"recent_alerts"`
	RiskTrend       []RiskDataPoint `json:"risk_trend"`
}

// ThreatSummary represents a summary of a threat type
type ThreatSummary struct {
	Type  ThreatType `json:"type"`
	Count int64      `json:"count"`
}

// RiskDataPoint represents a risk score at a point in time
type RiskDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Score     float64   `json:"score"`
}

// SecurityAlert represents a security alert
type SecurityAlert struct {
	ID           string         `json:"id"`
	TenantID     string         `json:"tenant_id"`
	AlertType    string         `json:"alert_type"`
	Severity     ThreatSeverity `json:"severity"`
	Title        string         `json:"title"`
	Description  string         `json:"description"`
	Acknowledged bool           `json:"acknowledged"`
	CreatedAt    time.Time      `json:"created_at"`
}

// CreateWAFRuleRequest represents a request to create a WAF rule
type CreateWAFRuleRequest struct {
	Name        string      `json:"name" binding:"required"`
	Description string      `json:"description,omitempty"`
	Pattern     string      `json:"pattern" binding:"required"`
	RuleType    WAFRuleType `json:"rule_type" binding:"required"`
	Action      ScanAction  `json:"action" binding:"required"`
	Priority    int         `json:"priority,omitempty"`
	Enabled     bool        `json:"enabled"`
}

// ScanPayloadRequest represents a request to scan a webhook payload
type ScanPayloadRequest struct {
	WebhookID  string            `json:"webhook_id" binding:"required"`
	DeliveryID string            `json:"delivery_id,omitempty"`
	Payload    json.RawMessage   `json:"payload" binding:"required"`
	SourceIP   string            `json:"source_ip,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
}

// ReviewQuarantineRequest represents a request to review a quarantined webhook
type ReviewQuarantineRequest struct {
	Decision string `json:"decision" binding:"required"` // approve, reject
	Reason   string `json:"reason,omitempty"`
}

// ReportIPRequest represents a request to report a malicious IP
type ReportIPRequest struct {
	IP         string   `json:"ip" binding:"required"`
	Categories []string `json:"categories,omitempty"`
	Reason     string   `json:"reason,omitempty"`
}

// SecurityScanResult represents a full endpoint security scan
type SecurityScanResult struct {
	EndpointID       string            `json:"endpoint_id"`
	URL              string            `json:"url"`
	OverallScore     string            `json:"overall_score"` // A-F
	NumericScore     int               `json:"numeric_score"` // 0-100
	TLSInfo          *TLSInfo          `json:"tls_info,omitempty"`
	SecurityHeaders  []HeaderCheck     `json:"security_headers"`
	ResponseTimeMs   int               `json:"response_time_ms"`
	Findings         []SecurityFinding `json:"findings"`
	ComplianceChecks []ComplianceCheck `json:"compliance_checks"`
	ScannedAt        time.Time         `json:"scanned_at"`
}

// TLSInfo holds TLS certificate and cipher info
type TLSInfo struct {
	Version      string    `json:"version"`
	CipherSuite  string    `json:"cipher_suite"`
	CertIssuer   string    `json:"cert_issuer"`
	CertExpiry   time.Time `json:"cert_expiry"`
	CertValid    bool      `json:"cert_valid"`
	CertDaysLeft int       `json:"cert_days_left"`
	IsHTTPS      bool      `json:"is_https"`
	SupportsHSTS bool      `json:"supports_hsts"`
}

// HeaderCheck represents a security header check result
type HeaderCheck struct {
	Header   string `json:"header"`
	Present  bool   `json:"present"`
	Value    string `json:"value,omitempty"`
	Expected string `json:"expected,omitempty"`
	Severity string `json:"severity"` // critical, high, medium, low, info
}

// SecurityFinding represents a specific security finding
type SecurityFinding struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	Severity       string `json:"severity"`
	Category       string `json:"category"`
	Recommendation string `json:"recommendation"`
}

// ComplianceCheck represents a compliance framework check
type ComplianceCheck struct {
	Framework   string `json:"framework"` // SOC2, HIPAA
	ControlID   string `json:"control_id"`
	ControlName string `json:"control_name"`
	Status      string `json:"status"` // pass, fail, warning, not_applicable
	Details     string `json:"details,omitempty"`
}

// SecurityThreshold defines when to auto-disable endpoints
type SecurityThreshold struct {
	TenantID       string `json:"tenant_id" db:"tenant_id"`
	MinScore       int    `json:"min_score" db:"min_score"`
	AutoDisable    bool   `json:"auto_disable" db:"auto_disable"`
	AlertOnDegrade bool   `json:"alert_on_degrade" db:"alert_on_degrade"`
}

// GetSeverityScore returns a numeric score for severity ranking
func GetSeverityScore(severity ThreatSeverity) int {
	switch severity {
	case ThreatSeverityCritical:
		return 5
	case ThreatSeverityHigh:
		return 4
	case ThreatSeverityMedium:
		return 3
	case ThreatSeverityLow:
		return 2
	case ThreatSeverityInfo:
		return 1
	default:
		return 0
	}
}
