package compliancevault

import (
	"encoding/json"
	"time"
)

// Compliance framework constants
const (
	FrameworkGDPR   = "gdpr"
	FrameworkSOC2   = "soc2"
	FrameworkHIPAA  = "hipaa"
	FrameworkPCIDSS = "pci_dss"
)

// Retention action constants
const (
	RetentionActionArchive   = "archive"
	RetentionActionDelete    = "delete"
	RetentionActionAnonymize = "anonymize"
)

// Encryption algorithm constants
const (
	EncryptionAES256GCM = "aes-256-gcm"
	EncryptionChaCha20  = "chacha20-poly1305"
)

// Audit action constants
const (
	AuditActionCreate    = "create"
	AuditActionRead      = "read"
	AuditActionUpdate    = "update"
	AuditActionDelete    = "delete"
	AuditActionExport    = "export"
	AuditActionDecrypt   = "decrypt"
	AuditActionErasure   = "erasure"
	AuditActionRetention = "retention_applied"
)

// VaultEntry represents an encrypted webhook payload stored in the vault.
type VaultEntry struct {
	ID               string          `json:"id" db:"id"`
	TenantID         string          `json:"tenant_id" db:"tenant_id"`
	WebhookID        string          `json:"webhook_id" db:"webhook_id"`
	EndpointID       string          `json:"endpoint_id" db:"endpoint_id"`
	EventType        string          `json:"event_type" db:"event_type"`
	EncryptedPayload []byte          `json:"-" db:"encrypted_payload"`
	PayloadHash      string          `json:"payload_hash" db:"payload_hash"`
	EncryptionAlgo   string          `json:"encryption_algo" db:"encryption_algo"`
	KeyID            string          `json:"key_id" db:"key_id"`
	Metadata         json.RawMessage `json:"metadata,omitempty" db:"metadata"`
	ContentType      string          `json:"content_type" db:"content_type"`
	SizeBytes        int64           `json:"size_bytes" db:"size_bytes"`
	RetainUntil      *time.Time      `json:"retain_until,omitempty" db:"retain_until"`
	Frameworks       []string        `json:"frameworks,omitempty"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
	ExpiresAt        *time.Time      `json:"expires_at,omitempty" db:"expires_at"`
}

// RetentionPolicy defines data retention rules per tenant.
type RetentionPolicy struct {
	ID              string    `json:"id" db:"id"`
	TenantID        string    `json:"tenant_id" db:"tenant_id"`
	Name            string    `json:"name" db:"name"`
	Framework       string    `json:"framework" db:"framework"`
	RetentionDays   int       `json:"retention_days" db:"retention_days"`
	Action          string    `json:"action" db:"action"`
	EventTypeFilter string    `json:"event_type_filter,omitempty" db:"event_type_filter"`
	IsActive        bool      `json:"is_active" db:"is_active"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// AuditTrailEntry records every access to vault data for compliance evidence.
type AuditTrailEntry struct {
	ID        string            `json:"id" db:"id"`
	TenantID  string            `json:"tenant_id" db:"tenant_id"`
	EntryID   string            `json:"entry_id,omitempty" db:"entry_id"`
	ActorID   string            `json:"actor_id" db:"actor_id"`
	ActorType string            `json:"actor_type" db:"actor_type"`
	Action    string            `json:"action" db:"action"`
	Resource  string            `json:"resource" db:"resource"`
	Details   map[string]string `json:"details,omitempty"`
	IPAddress string            `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent string            `json:"user_agent,omitempty" db:"user_agent"`
	CreatedAt time.Time         `json:"created_at" db:"created_at"`
}

// EncryptionKey represents a key used for vault encryption.
type EncryptionKey struct {
	ID        string     `json:"id" db:"id"`
	TenantID  string     `json:"tenant_id" db:"tenant_id"`
	Algorithm string     `json:"algorithm" db:"algorithm"`
	Version   int        `json:"version" db:"version"`
	IsActive  bool       `json:"is_active" db:"is_active"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	RotatedAt *time.Time `json:"rotated_at,omitempty" db:"rotated_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty" db:"expires_at"`
}

// ErasureRequest tracks GDPR right-to-erasure requests.
type ErasureRequest struct {
	ID            string     `json:"id" db:"id"`
	TenantID      string     `json:"tenant_id" db:"tenant_id"`
	SubjectID     string     `json:"subject_id" db:"subject_id"`
	SubjectType   string     `json:"subject_type" db:"subject_type"`
	Reason        string     `json:"reason" db:"reason"`
	Status        string     `json:"status" db:"status"`
	EntriesFound  int        `json:"entries_found" db:"entries_found"`
	EntriesErased int        `json:"entries_erased" db:"entries_erased"`
	RequestedAt   time.Time  `json:"requested_at" db:"requested_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty" db:"completed_at"`
}

// ComplianceReport is a generated compliance status report.
type ComplianceReport struct {
	ID             string              `json:"id" db:"id"`
	TenantID       string              `json:"tenant_id" db:"tenant_id"`
	Framework      string              `json:"framework" db:"framework"`
	Status         string              `json:"status" db:"status"`
	Score          float64             `json:"score" db:"score"`
	TotalControls  int                 `json:"total_controls" db:"total_controls"`
	PassedControls int                 `json:"passed_controls" db:"passed_controls"`
	FailedControls int                 `json:"failed_controls" db:"failed_controls"`
	Findings       []ComplianceFinding `json:"findings,omitempty"`
	GeneratedAt    time.Time           `json:"generated_at" db:"generated_at"`
}

// ComplianceFinding describes a specific compliance finding.
type ComplianceFinding struct {
	Control     string `json:"control"`
	Status      string `json:"status"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Remediation string `json:"remediation,omitempty"`
}

// VaultStats aggregates vault metrics for a tenant.
type VaultStats struct {
	TotalEntries     int64      `json:"total_entries"`
	TotalSizeBytes   int64      `json:"total_size_bytes"`
	EncryptedEntries int64      `json:"encrypted_entries"`
	ExpiredEntries   int64      `json:"expired_entries"`
	ActivePolicies   int        `json:"active_policies"`
	PendingErasures  int        `json:"pending_erasures"`
	LastAuditAt      *time.Time `json:"last_audit_at,omitempty"`
}

// Request DTOs

type StorePayloadRequest struct {
	WebhookID   string          `json:"webhook_id" binding:"required"`
	EndpointID  string          `json:"endpoint_id" binding:"required"`
	EventType   string          `json:"event_type" binding:"required"`
	Payload     json.RawMessage `json:"payload" binding:"required"`
	ContentType string          `json:"content_type,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}

type CreateRetentionPolicyRequest struct {
	Name            string `json:"name" binding:"required"`
	Framework       string `json:"framework" binding:"required"`
	RetentionDays   int    `json:"retention_days" binding:"required,min=1"`
	Action          string `json:"action" binding:"required"`
	EventTypeFilter string `json:"event_type_filter,omitempty"`
}

type CreateErasureRequest struct {
	SubjectID   string `json:"subject_id" binding:"required"`
	SubjectType string `json:"subject_type" binding:"required"`
	Reason      string `json:"reason" binding:"required"`
}

type GenerateReportRequest struct {
	Framework string `json:"framework" binding:"required"`
}
