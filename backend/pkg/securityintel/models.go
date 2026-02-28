package securityintel

import "time"

// ThreatLevel classifies the severity of a detected threat.
type ThreatLevel string

const (
	ThreatCritical ThreatLevel = "critical"
	ThreatHigh     ThreatLevel = "high"
	ThreatMedium   ThreatLevel = "medium"
	ThreatLow      ThreatLevel = "low"
	ThreatInfo     ThreatLevel = "info"
)

// ThreatType categorizes detected threats.
type ThreatType string

const (
	ThreatSQLInjection    ThreatType = "sql_injection"
	ThreatXSS             ThreatType = "xss"
	ThreatSSRF            ThreatType = "ssrf"
	ThreatPayloadOversize ThreatType = "payload_oversize"
	ThreatRateBurst       ThreatType = "rate_burst"
	ThreatReplayAttack    ThreatType = "replay_attack"
	ThreatInvalidSig      ThreatType = "invalid_signature"
	ThreatSuspiciousIP    ThreatType = "suspicious_ip"
	ThreatDataExfil       ThreatType = "data_exfiltration"
	ThreatAnomalous       ThreatType = "anomalous_pattern"
)

// SecurityEvent represents a detected security event.
type SecurityEvent struct {
	ID          string      `json:"id" db:"id"`
	TenantID    string      `json:"tenant_id" db:"tenant_id"`
	EndpointID  string      `json:"endpoint_id,omitempty" db:"endpoint_id"`
	ThreatType  ThreatType  `json:"threat_type" db:"threat_type"`
	ThreatLevel ThreatLevel `json:"threat_level" db:"threat_level"`
	Description string      `json:"description" db:"description"`
	SourceIP    string      `json:"source_ip,omitempty" db:"source_ip"`
	Payload     string      `json:"payload_snippet,omitempty" db:"payload_snippet"`
	Action      string      `json:"action_taken" db:"action_taken"` // blocked, allowed, flagged
	Resolved    bool        `json:"resolved" db:"resolved"`
	DetectedAt  time.Time   `json:"detected_at" db:"detected_at"`
}

// SecurityPolicy defines automated enforcement rules.
type SecurityPolicy struct {
	ID          string       `json:"id" db:"id"`
	TenantID    string       `json:"tenant_id" db:"tenant_id"`
	Name        string       `json:"name" db:"name"`
	Description string       `json:"description" db:"description"`
	Rules       []PolicyRule `json:"rules"`
	Enabled     bool         `json:"enabled" db:"enabled"`
	CreatedAt   time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at" db:"updated_at"`
}

// PolicyRule defines a single rule within a security policy.
type PolicyRule struct {
	ID        string `json:"id"`
	Type      string `json:"type"`      // block_ip, rate_limit, require_signature, payload_scan
	Condition string `json:"condition"` // Expression evaluated against the request
	Action    string `json:"action"`    // block, allow, flag, quarantine
	Priority  int    `json:"priority"`
}

// PayloadInspection contains the results of a payload security scan.
type PayloadInspection struct {
	DeliveryID   string              `json:"delivery_id"`
	ThreatLevel  ThreatLevel         `json:"overall_threat_level"`
	Findings     []InspectionFinding `json:"findings"`
	ScanDuration string              `json:"scan_duration"`
	Safe         bool                `json:"safe"`
	ScannedAt    time.Time           `json:"scanned_at"`
}

// InspectionFinding represents a single finding from payload inspection.
type InspectionFinding struct {
	Type        ThreatType  `json:"type"`
	Level       ThreatLevel `json:"level"`
	Description string      `json:"description"`
	Location    string      `json:"location"` // JSON path or field reference
	Evidence    string      `json:"evidence,omitempty"`
}

// SecurityDashboard provides an aggregate security overview.
type SecurityDashboard struct {
	TotalEvents     int                 `json:"total_events"`
	CriticalEvents  int                 `json:"critical_events"`
	BlockedRequests int                 `json:"blocked_requests"`
	ThreatsByType   map[ThreatType]int  `json:"threats_by_type"`
	ThreatsByLevel  map[ThreatLevel]int `json:"threats_by_level"`
	ActivePolicies  int                 `json:"active_policies"`
	TopSourceIPs    []IPThreatSummary   `json:"top_source_ips"`
	RecentEvents    []SecurityEvent     `json:"recent_events"`
	Period          string              `json:"period"`
}

// IPThreatSummary summarizes threats from a specific IP.
type IPThreatSummary struct {
	IP         string `json:"ip"`
	EventCount int    `json:"event_count"`
	Blocked    bool   `json:"blocked"`
}

// AnomalyReport describes detected anomalous behavior.
type AnomalyReport struct {
	ID          string      `json:"id"`
	TenantID    string      `json:"tenant_id"`
	Type        string      `json:"anomaly_type"` // volume_spike, pattern_shift, new_source
	Severity    ThreatLevel `json:"severity"`
	Description string      `json:"description"`
	Baseline    float64     `json:"baseline_value"`
	Observed    float64     `json:"observed_value"`
	Deviation   float64     `json:"deviation_percent"`
	DetectedAt  time.Time   `json:"detected_at"`
}

// CreatePolicyRequest is the API request for creating a security policy.
type CreatePolicyRequest struct {
	Name        string       `json:"name" binding:"required"`
	Description string       `json:"description"`
	Rules       []PolicyRule `json:"rules" binding:"required"`
}

// InspectPayloadRequest is the API request for payload inspection.
type InspectPayloadRequest struct {
	Payload     string `json:"payload" binding:"required"`
	ContentType string `json:"content_type"`
	EndpointID  string `json:"endpoint_id,omitempty"`
}
