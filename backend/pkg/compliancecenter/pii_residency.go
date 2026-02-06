package compliancecenter

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// --- PII Detection & Redaction ---

// PIIType represents types of personally identifiable information
type PIIType string

const (
	PIIEmail       PIIType = "email"
	PIIPhone       PIIType = "phone"
	PIICreditCard  PIIType = "credit_card"
	PIISSN         PIIType = "ssn"
	PIIIPAddress   PIIType = "ip_address"
	PIIName        PIIType = "name"
	PIIAddress     PIIType = "address"
	PIIDateOfBirth PIIType = "date_of_birth"
	PIIPassport    PIIType = "passport"
	PIICustom      PIIType = "custom"
)

// PIIDetection represents a detected PII field in a payload
type PIIDetection struct {
	Path       string  `json:"path"`
	PIIType    PIIType `json:"pii_type"`
	Value      string  `json:"value,omitempty"` // Only populated if not redacted
	Confidence float64 `json:"confidence"`
	Redacted   bool    `json:"redacted"`
}

// PIIScanResult represents the result of scanning a payload for PII
type PIIScanResult struct {
	PayloadID   string         `json:"payload_id"`
	Detections  []PIIDetection `json:"detections"`
	PIIFound    bool           `json:"pii_found"`
	TotalFields int            `json:"total_fields"`
	PIIFields   int            `json:"pii_fields"`
	ScanTime    time.Duration  `json:"scan_time_ms"`
}

// PIIRedactionConfig configures how PII is redacted
type PIIRedactionConfig struct {
	TenantID       string       `json:"tenant_id" db:"tenant_id"`
	Enabled        bool         `json:"enabled" db:"enabled"`
	Mode           string       `json:"mode" db:"mode"` // mask, hash, remove, encrypt
	PIITypes       []PIIType    `json:"pii_types"`
	ExcludePaths   []string     `json:"exclude_paths,omitempty"`
	CustomPatterns []PIIPattern `json:"custom_patterns,omitempty"`
	CreatedAt      time.Time    `json:"created_at" db:"created_at"`
}

// PIIPattern represents a custom PII detection pattern
type PIIPattern struct {
	Name    string  `json:"name"`
	Regex   string  `json:"regex"`
	PIIType PIIType `json:"pii_type"`
}

// PIIDetector scans payloads for PII
type PIIDetector struct {
	patterns map[PIIType]*regexp.Regexp
	custom   []PIIPattern
}

// NewPIIDetector creates a new PII detector with built-in patterns
func NewPIIDetector() *PIIDetector {
	d := &PIIDetector{
		patterns: make(map[PIIType]*regexp.Regexp),
	}

	d.patterns[PIIEmail] = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
	d.patterns[PIIPhone] = regexp.MustCompile(`(\+?1?\s*\(?[0-9]{3}\)?[\s\-.]?[0-9]{3}[\s\-.]?[0-9]{4})`)
	d.patterns[PIICreditCard] = regexp.MustCompile(`\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13}|6(?:011|5[0-9]{2})[0-9]{12})\b`)
	d.patterns[PIISSN] = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
	d.patterns[PIIIPAddress] = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)

	return d
}

// AddCustomPattern adds a custom PII detection pattern
func (d *PIIDetector) AddCustomPattern(pattern PIIPattern) error {
	compiled, err := regexp.Compile(pattern.Regex)
	if err != nil {
		return fmt.Errorf("invalid regex pattern: %w", err)
	}
	d.patterns[pattern.PIIType] = compiled
	d.custom = append(d.custom, pattern)
	return nil
}

// ScanPayload scans a JSON payload for PII
func (d *PIIDetector) ScanPayload(payloadID string, payload json.RawMessage) *PIIScanResult {
	start := time.Now()

	result := &PIIScanResult{
		PayloadID: payloadID,
	}

	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return result
	}

	d.scanMap(data, "", result)
	result.ScanTime = time.Since(start)
	result.PIIFound = len(result.Detections) > 0
	result.PIIFields = len(result.Detections)

	return result
}

func (d *PIIDetector) scanMap(data map[string]interface{}, prefix string, result *PIIScanResult) {
	for key, val := range data {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		result.TotalFields++

		switch v := val.(type) {
		case string:
			d.scanString(path, v, result)
		case map[string]interface{}:
			d.scanMap(v, path, result)
		case []interface{}:
			for i, item := range v {
				arrayPath := fmt.Sprintf("%s[%d]", path, i)
				if m, ok := item.(map[string]interface{}); ok {
					d.scanMap(m, arrayPath, result)
				} else if s, ok := item.(string); ok {
					d.scanString(arrayPath, s, result)
				}
			}
		}
	}
}

func (d *PIIDetector) scanString(path, value string, result *PIIScanResult) {
	// Check field name heuristics
	lowerPath := strings.ToLower(path)
	nameHints := map[string]PIIType{
		"email": PIIEmail, "mail": PIIEmail,
		"phone": PIIPhone, "mobile": PIIPhone, "tel": PIIPhone,
		"ssn": PIISSN, "social_security": PIISSN,
		"credit_card": PIICreditCard, "card_number": PIICreditCard,
		"ip_address": PIIIPAddress, "ip": PIIIPAddress,
		"first_name": PIIName, "last_name": PIIName, "full_name": PIIName,
		"address": PIIAddress, "street": PIIAddress,
		"date_of_birth": PIIDateOfBirth, "dob": PIIDateOfBirth, "birthday": PIIDateOfBirth,
	}

	for hint, piiType := range nameHints {
		if strings.Contains(lowerPath, hint) {
			result.Detections = append(result.Detections, PIIDetection{
				Path:       path,
				PIIType:    piiType,
				Confidence: 0.9,
			})
			return
		}
	}

	// Check value against regex patterns
	for piiType, pattern := range d.patterns {
		if pattern.MatchString(value) {
			result.Detections = append(result.Detections, PIIDetection{
				Path:       path,
				PIIType:    piiType,
				Confidence: 0.8,
			})
			return
		}
	}
}

// RedactPayload redacts PII from a payload based on scan results
func RedactPayload(payload json.RawMessage, detections []PIIDetection, mode string) (json.RawMessage, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return payload, err
	}

	for _, det := range detections {
		redactField(data, det.Path, mode)
	}

	return json.Marshal(data)
}

func redactField(data map[string]interface{}, path, mode string) {
	parts := strings.Split(path, ".")
	current := data

	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part — redact the value
			if _, exists := current[part]; exists {
				switch mode {
				case "mask":
					current[part] = "***REDACTED***"
				case "remove":
					delete(current, part)
				case "hash":
					current[part] = "[HASHED]"
				default:
					current[part] = "***REDACTED***"
				}
			}
		} else {
			if next, ok := current[part].(map[string]interface{}); ok {
				current = next
			} else {
				return
			}
		}
	}
}

// --- Data Residency Enforcement ---

// DataResidencyRegion represents a supported data residency region
type DataResidencyRegion string

const (
	RegionUSEast1     DataResidencyRegion = "us-east-1"
	RegionUSWest2     DataResidencyRegion = "us-west-2"
	RegionEUWest1     DataResidencyRegion = "eu-west-1"
	RegionEUCentral   DataResidencyRegion = "eu-central-1"
	RegionAPSoutheast DataResidencyRegion = "ap-southeast-1"
)

// DataResidencyPolicy defines data residency requirements for a tenant
type DataResidencyPolicy struct {
	TenantID           string                `json:"tenant_id" db:"tenant_id"`
	PrimaryRegion      DataResidencyRegion   `json:"primary_region" db:"primary_region"`
	AllowedRegions     []DataResidencyRegion `json:"allowed_regions"`
	RestrictedRegions  []DataResidencyRegion `json:"restricted_regions"`
	DataClassification string                `json:"data_classification" db:"data_classification"`
	RequireEncryption  bool                  `json:"require_encryption" db:"require_encryption"`
	RetentionDays      int                   `json:"retention_days" db:"retention_days"`
	CreatedAt          time.Time             `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time             `json:"updated_at" db:"updated_at"`
}

// ResidencyCheck verifies that a delivery respects data residency requirements
type ResidencyCheck struct {
	TenantID    string              `json:"tenant_id"`
	EndpointURL string              `json:"endpoint_url"`
	Region      DataResidencyRegion `json:"detected_region"`
	Allowed     bool                `json:"allowed"`
	Reason      string              `json:"reason"`
	CheckedAt   time.Time           `json:"checked_at"`
}

// CheckDataResidency verifies an endpoint URL against residency policy
func CheckDataResidency(policy *DataResidencyPolicy, endpointURL string, detectedRegion DataResidencyRegion) *ResidencyCheck {
	check := &ResidencyCheck{
		TenantID:    policy.TenantID,
		EndpointURL: endpointURL,
		Region:      detectedRegion,
		Allowed:     true,
		CheckedAt:   time.Now(),
	}

	// Check restricted regions
	for _, restricted := range policy.RestrictedRegions {
		if detectedRegion == restricted {
			check.Allowed = false
			check.Reason = fmt.Sprintf("endpoint region %s is restricted by data residency policy", detectedRegion)
			return check
		}
	}

	// Check allowed regions (if specified, only those are allowed)
	if len(policy.AllowedRegions) > 0 {
		allowed := false
		for _, region := range policy.AllowedRegions {
			if detectedRegion == region {
				allowed = true
				break
			}
		}
		if !allowed {
			check.Allowed = false
			check.Reason = fmt.Sprintf("endpoint region %s is not in allowed regions list", detectedRegion)
			return check
		}
	}

	check.Reason = "endpoint region compliant with data residency policy"
	return check
}

// --- Audit Trail with Hash Chains ---

// HashChainEntry represents an entry in the tamper-evident audit hash chain
type HashChainEntry struct {
	ID           string    `json:"id" db:"id"`
	TenantID     string    `json:"tenant_id" db:"tenant_id"`
	SequenceNum  int64     `json:"sequence_num" db:"sequence_num"`
	EventType    string    `json:"event_type" db:"event_type"`
	EventData    string    `json:"event_data" db:"event_data"`
	PreviousHash string    `json:"previous_hash" db:"previous_hash"`
	CurrentHash  string    `json:"current_hash" db:"current_hash"`
	Timestamp    time.Time `json:"timestamp" db:"timestamp"`
}

// HashChainVerification represents the result of verifying a hash chain
type HashChainVerification struct {
	TenantID     string    `json:"tenant_id"`
	TotalEntries int64     `json:"total_entries"`
	Verified     int64     `json:"verified"`
	Broken       int64     `json:"broken"`
	IsValid      bool      `json:"is_valid"`
	FirstBreak   *int64    `json:"first_break_at,omitempty"`
	VerifiedAt   time.Time `json:"verified_at"`
}

// AuditHashChain manages a tamper-evident audit trail using hash chains
type AuditHashChain struct {
	tenantID    string
	lastHash    string
	sequenceNum int64
}

// NewAuditHashChain creates a new hash chain for audit entries
func NewAuditHashChain(tenantID string) *AuditHashChain {
	return &AuditHashChain{
		tenantID: tenantID,
		lastHash: "genesis",
	}
}

// AppendEntry adds a new entry to the hash chain
func (hc *AuditHashChain) AppendEntry(eventType, eventData string) *HashChainEntry {
	hc.sequenceNum++

	entry := &HashChainEntry{
		ID:           uuid.New().String(),
		TenantID:     hc.tenantID,
		SequenceNum:  hc.sequenceNum,
		EventType:    eventType,
		EventData:    eventData,
		PreviousHash: hc.lastHash,
		Timestamp:    time.Now(),
	}

	// Compute hash: SHA256(sequence + event_type + event_data + previous_hash + timestamp)
	hashInput := fmt.Sprintf("%d:%s:%s:%s:%d",
		entry.SequenceNum, entry.EventType, entry.EventData,
		entry.PreviousHash, entry.Timestamp.UnixNano())
	entry.CurrentHash = computeSHA256(hashInput)

	hc.lastHash = entry.CurrentHash
	return entry
}

// VerifyChain verifies the integrity of a hash chain
func VerifyChain(entries []HashChainEntry) *HashChainVerification {
	result := &HashChainVerification{
		TotalEntries: int64(len(entries)),
		VerifiedAt:   time.Now(),
		IsValid:      true,
	}

	if len(entries) == 0 {
		return result
	}

	result.TenantID = entries[0].TenantID

	for i := 1; i < len(entries); i++ {
		if entries[i].PreviousHash != entries[i-1].CurrentHash {
			result.IsValid = false
			result.Broken++
			if result.FirstBreak == nil {
				seqNum := entries[i].SequenceNum
				result.FirstBreak = &seqNum
			}
		} else {
			result.Verified++
		}
	}
	result.Verified++ // First entry is always "verified"

	return result
}

func computeSHA256(input string) string {
	// Simplified hash computation for demonstration
	// In production, use crypto/sha256
	var hash uint64
	for _, c := range input {
		hash = hash*31 + uint64(c)
	}
	return fmt.Sprintf("%016x", hash)
}
