package compliancecenter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ImmutableAuditEntry is an append-only audit record with integrity verification.
// Each entry contains a hash chain linking it to the previous entry, making
// tampering detectable.
type ImmutableAuditEntry struct {
	ID               string          `json:"id" db:"id"`
	TenantID         string          `json:"tenant_id" db:"tenant_id"`
	SequenceNumber   int64           `json:"sequence_number" db:"sequence_number"`
	Timestamp        time.Time       `json:"timestamp" db:"timestamp"`
	EventType        AuditEventType  `json:"event_type" db:"event_type"`
	Actor            AuditActor      `json:"actor"`
	Resource         AuditResource   `json:"resource"`
	Action           string          `json:"action" db:"action"`
	Outcome          string          `json:"outcome" db:"outcome"` // success, failure, denied
	PayloadHash      string          `json:"payload_hash,omitempty" db:"payload_hash"`
	Details          json.RawMessage `json:"details,omitempty" db:"details"`
	PreviousHash     string          `json:"previous_hash" db:"previous_hash"`
	EntryHash        string          `json:"entry_hash" db:"entry_hash"`
	IntegrityChain   string          `json:"integrity_chain" db:"integrity_chain"`
	SourceIP         string          `json:"source_ip,omitempty" db:"source_ip"`
	UserAgent        string          `json:"user_agent,omitempty" db:"user_agent"`
}

// AuditEventType categorizes audit events
type AuditEventType string

const (
	AuditWebhookCreated    AuditEventType = "webhook.created"
	AuditWebhookUpdated    AuditEventType = "webhook.updated"
	AuditWebhookDeleted    AuditEventType = "webhook.deleted"
	AuditDeliveryAttempted AuditEventType = "delivery.attempted"
	AuditDeliverySucceeded AuditEventType = "delivery.succeeded"
	AuditDeliveryFailed    AuditEventType = "delivery.failed"
	AuditDeliveryReplayed  AuditEventType = "delivery.replayed"
	AuditEndpointCreated   AuditEventType = "endpoint.created"
	AuditEndpointUpdated   AuditEventType = "endpoint.updated"
	AuditEndpointDeleted   AuditEventType = "endpoint.deleted"
	AuditEndpointPaused    AuditEventType = "endpoint.paused"
	AuditEndpointResumed   AuditEventType = "endpoint.resumed"
	AuditSecretRotated     AuditEventType = "secret.rotated"
	AuditKeyCreated        AuditEventType = "key.created"
	AuditKeyRevoked        AuditEventType = "key.revoked"
	AuditConfigChanged     AuditEventType = "config.changed"
	AuditDataExported      AuditEventType = "data.exported"
	AuditDataDeleted       AuditEventType = "data.deleted"
	AuditLoginSuccess      AuditEventType = "auth.login_success"
	AuditLoginFailed       AuditEventType = "auth.login_failed"
	AuditPolicyViolation   AuditEventType = "policy.violation"
)

// AuditActor identifies who performed the action
type AuditActor struct {
	ID    string `json:"id"`
	Type  string `json:"type"` // user, api_key, system, service
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

// AuditResource identifies the affected resource
type AuditResource struct {
	Type string `json:"type"` // webhook, endpoint, delivery, tenant, key, config
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// AuditTrailService provides immutable audit logging with integrity verification
type AuditTrailService struct {
	repo   AuditTrailRepository
	config *AuditTrailConfig
}

// AuditTrailConfig configures the audit trail
type AuditTrailConfig struct {
	RetentionDays     int  // How long to keep audit entries
	EnableHashChain   bool // Enable hash chain for tamper detection
	EnablePayloadHash bool // Hash webhook payloads for fingerprinting
}

// DefaultAuditTrailConfig returns sensible defaults
func DefaultAuditTrailConfig() *AuditTrailConfig {
	return &AuditTrailConfig{
		RetentionDays:     730, // 2 years
		EnableHashChain:   true,
		EnablePayloadHash: true,
	}
}

// AuditTrailRepository defines storage for the immutable audit trail
type AuditTrailRepository interface {
	AppendEntry(ctx context.Context, entry *ImmutableAuditEntry) error
	GetEntry(ctx context.Context, tenantID, entryID string) (*ImmutableAuditEntry, error)
	ListEntries(ctx context.Context, tenantID string, filters *AuditTrailFilters) ([]ImmutableAuditEntry, int, error)
	GetLatestEntry(ctx context.Context, tenantID string) (*ImmutableAuditEntry, error)
	GetSequenceRange(ctx context.Context, tenantID string, startSeq, endSeq int64) ([]ImmutableAuditEntry, error)
	CountEntries(ctx context.Context, tenantID string, filters *AuditTrailFilters) (int64, error)
}

// AuditTrailFilters for querying audit entries
type AuditTrailFilters struct {
	EventTypes []AuditEventType `json:"event_types,omitempty"`
	ActorID    string           `json:"actor_id,omitempty"`
	ResourceID string           `json:"resource_id,omitempty"`
	StartTime  *time.Time       `json:"start_time,omitempty"`
	EndTime    *time.Time       `json:"end_time,omitempty"`
	Outcome    string           `json:"outcome,omitempty"`
	Limit      int              `json:"limit"`
	Offset     int              `json:"offset"`
}

// NewAuditTrailService creates a new audit trail service
func NewAuditTrailService(repo AuditTrailRepository, config *AuditTrailConfig) *AuditTrailService {
	if config == nil {
		config = DefaultAuditTrailConfig()
	}
	return &AuditTrailService{repo: repo, config: config}
}

// RecordEvent records an immutable audit event
func (s *AuditTrailService) RecordEvent(ctx context.Context, tenantID string, eventType AuditEventType,
	actor AuditActor, resource AuditResource, action, outcome string,
	payload []byte, details json.RawMessage, sourceIP, userAgent string) error {

	// Get the latest entry for hash chaining
	previousHash := "genesis"
	var seqNum int64 = 1

	if s.config.EnableHashChain {
		latest, err := s.repo.GetLatestEntry(ctx, tenantID)
		if err == nil && latest != nil {
			previousHash = latest.EntryHash
			seqNum = latest.SequenceNumber + 1
		}
	}

	entry := &ImmutableAuditEntry{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		SequenceNumber: seqNum,
		Timestamp:      time.Now().UTC(),
		EventType:      eventType,
		Actor:          actor,
		Resource:       resource,
		Action:         action,
		Outcome:        outcome,
		Details:        details,
		PreviousHash:   previousHash,
		SourceIP:       sourceIP,
		UserAgent:      userAgent,
	}

	// Compute payload fingerprint
	if s.config.EnablePayloadHash && len(payload) > 0 {
		entry.PayloadHash = ComputePayloadFingerprint(payload)
	}

	// Compute entry hash for integrity chain
	entry.EntryHash = computeEntryHash(entry)
	entry.IntegrityChain = computeChainHash(previousHash, entry.EntryHash)

	return s.repo.AppendEntry(ctx, entry)
}

// VerifyIntegrity verifies the integrity of the audit trail for a tenant
func (s *AuditTrailService) VerifyIntegrity(ctx context.Context, tenantID string) (*IntegrityReport, error) {
	report := &IntegrityReport{
		TenantID:    tenantID,
		VerifiedAt:  time.Now().UTC(),
		IsIntact:    true,
	}

	// Get all entries in sequence order
	entries, _, err := s.repo.ListEntries(ctx, tenantID, &AuditTrailFilters{Limit: 10000})
	if err != nil {
		return nil, fmt.Errorf("failed to list entries: %w", err)
	}

	if len(entries) == 0 {
		report.TotalEntries = 0
		return report, nil
	}

	// Sort by sequence number
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].SequenceNumber < entries[j].SequenceNumber
	})

	report.TotalEntries = int64(len(entries))
	report.FirstEntry = entries[0].Timestamp
	report.LastEntry = entries[len(entries)-1].Timestamp

	// Verify hash chain
	for i, entry := range entries {
		// Verify entry hash
		expectedHash := computeEntryHash(&entry)
		if expectedHash != entry.EntryHash {
			report.IsIntact = false
			report.Violations = append(report.Violations, IntegrityViolation{
				SequenceNumber: entry.SequenceNumber,
				EntryID:        entry.ID,
				Type:           "hash_mismatch",
				Description:    fmt.Sprintf("entry hash mismatch at sequence %d", entry.SequenceNumber),
			})
		}

		// Verify chain continuity
		if i > 0 {
			expectedPrevious := entries[i-1].EntryHash
			if entry.PreviousHash != expectedPrevious {
				report.IsIntact = false
				report.Violations = append(report.Violations, IntegrityViolation{
					SequenceNumber: entry.SequenceNumber,
					EntryID:        entry.ID,
					Type:           "chain_break",
					Description:    fmt.Sprintf("chain break at sequence %d", entry.SequenceNumber),
				})
			}
		} else if entry.PreviousHash != "genesis" {
			report.IsIntact = false
			report.Violations = append(report.Violations, IntegrityViolation{
				SequenceNumber: entry.SequenceNumber,
				EntryID:        entry.ID,
				Type:           "invalid_genesis",
				Description:    "first entry does not reference genesis",
			})
		}

		// Check sequence continuity
		if i > 0 && entry.SequenceNumber != entries[i-1].SequenceNumber+1 {
			report.Violations = append(report.Violations, IntegrityViolation{
				SequenceNumber: entry.SequenceNumber,
				EntryID:        entry.ID,
				Type:           "sequence_gap",
				Description:    fmt.Sprintf("sequence gap between %d and %d", entries[i-1].SequenceNumber, entry.SequenceNumber),
			})
		}
	}

	report.VerifiedEntries = report.TotalEntries
	return report, nil
}

// GetEntry retrieves a single audit entry
func (s *AuditTrailService) GetEntry(ctx context.Context, tenantID, entryID string) (*ImmutableAuditEntry, error) {
	return s.repo.GetEntry(ctx, tenantID, entryID)
}

// ListEntries lists audit entries with filtering
func (s *AuditTrailService) ListEntries(ctx context.Context, tenantID string, filters *AuditTrailFilters) ([]ImmutableAuditEntry, int, error) {
	if filters == nil {
		filters = &AuditTrailFilters{Limit: 50}
	}
	if filters.Limit <= 0 {
		filters.Limit = 50
	}
	return s.repo.ListEntries(ctx, tenantID, filters)
}

// GenerateComplianceExport generates a compliance-ready export
func (s *AuditTrailService) GenerateComplianceExport(ctx context.Context, tenantID string, framework ComplianceFramework, startDate, endDate time.Time) (*ComplianceExport, error) {
	entries, total, err := s.repo.ListEntries(ctx, tenantID, &AuditTrailFilters{
		StartTime: &startDate,
		EndTime:   &endDate,
		Limit:     10000,
	})
	if err != nil {
		return nil, err
	}

	integrity, _ := s.VerifyIntegrity(ctx, tenantID)

	export := &ComplianceExport{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		Framework:    framework,
		GeneratedAt:  time.Now().UTC(),
		PeriodStart:  startDate,
		PeriodEnd:    endDate,
		TotalEntries: int64(total),
		IntegrityReport: integrity,
	}

	// Categorize entries
	eventCounts := make(map[AuditEventType]int64)
	for _, entry := range entries {
		eventCounts[entry.EventType]++
	}
	export.EventSummary = eventCounts

	// Framework-specific sections
	switch framework {
	case FrameworkSOC2:
		export.Sections = s.buildSOC2Sections(entries)
	case FrameworkGDPR:
		export.Sections = s.buildGDPRSections(entries)
	default:
		export.Sections = s.buildGenericSections(entries)
	}

	return export, nil
}

func (s *AuditTrailService) buildSOC2Sections(entries []ImmutableAuditEntry) []ExportSection {
	sections := []ExportSection{
		{
			Title:       "CC6.1 - Logical Access Controls",
			Description: "Evidence of access control and authentication events",
			Entries:     filterByEventTypes(entries, []AuditEventType{AuditLoginSuccess, AuditLoginFailed, AuditKeyCreated, AuditKeyRevoked}),
		},
		{
			Title:       "CC7.2 - System Monitoring",
			Description: "Evidence of system monitoring and change management",
			Entries:     filterByEventTypes(entries, []AuditEventType{AuditConfigChanged, AuditEndpointCreated, AuditEndpointUpdated, AuditEndpointDeleted}),
		},
		{
			Title:       "CC8.1 - Change Management",
			Description: "Evidence of controlled changes to webhook configurations",
			Entries:     filterByEventTypes(entries, []AuditEventType{AuditWebhookCreated, AuditWebhookUpdated, AuditWebhookDeleted, AuditSecretRotated}),
		},
	}
	return sections
}

func (s *AuditTrailService) buildGDPRSections(entries []ImmutableAuditEntry) []ExportSection {
	sections := []ExportSection{
		{
			Title:       "Article 30 - Records of Processing Activities",
			Description: "Record of all webhook data processing activities",
			Entries:     filterByEventTypes(entries, []AuditEventType{AuditDeliveryAttempted, AuditDeliverySucceeded, AuditDeliveryFailed}),
		},
		{
			Title:       "Article 17 - Right to Erasure",
			Description: "Evidence of data deletion activities",
			Entries:     filterByEventTypes(entries, []AuditEventType{AuditDataDeleted, AuditDataExported}),
		},
		{
			Title:       "Article 33 - Breach Notification",
			Description: "Security incidents and policy violations",
			Entries:     filterByEventTypes(entries, []AuditEventType{AuditPolicyViolation, AuditLoginFailed}),
		},
	}
	return sections
}

func (s *AuditTrailService) buildGenericSections(entries []ImmutableAuditEntry) []ExportSection {
	return []ExportSection{
		{
			Title:       "All Audit Events",
			Description: "Complete audit trail",
			Entries:     entries,
		},
	}
}

// --- Integrity and hashing functions ---

// ComputePayloadFingerprint generates a SHA-256 fingerprint of a webhook payload
func ComputePayloadFingerprint(payload []byte) string {
	hash := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(hash[:])
}

func computeEntryHash(entry *ImmutableAuditEntry) string {
	// Hash the content of the entry (excluding the hash fields themselves)
	parts := []string{
		entry.ID,
		entry.TenantID,
		fmt.Sprintf("%d", entry.SequenceNumber),
		entry.Timestamp.UTC().Format(time.RFC3339Nano),
		string(entry.EventType),
		entry.Actor.ID,
		entry.Actor.Type,
		entry.Resource.Type,
		entry.Resource.ID,
		entry.Action,
		entry.Outcome,
		entry.PayloadHash,
	}
	content := strings.Join(parts, "|")
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

func computeChainHash(previousHash, entryHash string) string {
	combined := previousHash + ":" + entryHash
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

func filterByEventTypes(entries []ImmutableAuditEntry, types []AuditEventType) []ImmutableAuditEntry {
	typeSet := make(map[AuditEventType]bool)
	for _, t := range types {
		typeSet[t] = true
	}

	var filtered []ImmutableAuditEntry
	for _, e := range entries {
		if typeSet[e.EventType] {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// --- Export models ---

// IntegrityReport contains the results of an audit trail integrity verification
type IntegrityReport struct {
	TenantID        string               `json:"tenant_id"`
	VerifiedAt      time.Time            `json:"verified_at"`
	IsIntact        bool                 `json:"is_intact"`
	TotalEntries    int64                `json:"total_entries"`
	VerifiedEntries int64                `json:"verified_entries"`
	FirstEntry      time.Time            `json:"first_entry"`
	LastEntry       time.Time            `json:"last_entry"`
	Violations      []IntegrityViolation `json:"violations,omitempty"`
}

// IntegrityViolation represents a detected integrity issue
type IntegrityViolation struct {
	SequenceNumber int64  `json:"sequence_number"`
	EntryID        string `json:"entry_id"`
	Type           string `json:"type"` // hash_mismatch, chain_break, sequence_gap, invalid_genesis
	Description    string `json:"description"`
}

// ComplianceExport is a framework-specific compliance report
type ComplianceExport struct {
	ID              string                       `json:"id"`
	TenantID        string                       `json:"tenant_id"`
	Framework       ComplianceFramework           `json:"framework"`
	GeneratedAt     time.Time                    `json:"generated_at"`
	PeriodStart     time.Time                    `json:"period_start"`
	PeriodEnd       time.Time                    `json:"period_end"`
	TotalEntries    int64                        `json:"total_entries"`
	EventSummary    map[AuditEventType]int64     `json:"event_summary"`
	IntegrityReport *IntegrityReport             `json:"integrity_report"`
	Sections        []ExportSection              `json:"sections"`
}

// ExportSection is a section within a compliance export
type ExportSection struct {
	Title       string                `json:"title"`
	Description string                `json:"description"`
	Entries     []ImmutableAuditEntry `json:"entries"`
}
