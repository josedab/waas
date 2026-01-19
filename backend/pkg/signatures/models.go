package signatures

import (
	"time"
)

// SignatureScheme represents a webhook signature scheme
type SignatureScheme struct {
	ID          string            `json:"id"`
	TenantID    string            `json:"tenant_id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Type        SignatureType     `json:"type"`
	Algorithm   SignatureAlgorithm `json:"algorithm"`
	Config      SignatureConfig   `json:"config"`
	KeyConfig   KeyConfiguration  `json:"key_config"`
	Status      SchemeStatus      `json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// SignatureType defines the signature scheme type
type SignatureType string

const (
	// Standard Webhooks spec (standardwebhooks.com)
	TypeStandardWebhooks SignatureType = "standard_webhooks"
	// Stripe-style signatures
	TypeStripe SignatureType = "stripe"
	// GitHub-style signatures
	TypeGitHub SignatureType = "github"
	// Slack-style signatures
	TypeSlack SignatureType = "slack"
	// Custom HMAC signatures
	TypeCustomHMAC SignatureType = "custom_hmac"
	// AWS SNS signatures
	TypeAWSSNS SignatureType = "aws_sns"
	// Twilio signatures
	TypeTwilio SignatureType = "twilio"
)

// SignatureAlgorithm defines supported algorithms
type SignatureAlgorithm string

const (
	AlgorithmHMACSHA256 SignatureAlgorithm = "hmac-sha256"
	AlgorithmHMACSHA512 SignatureAlgorithm = "hmac-sha512"
	AlgorithmHMACSHA1   SignatureAlgorithm = "hmac-sha1"
	AlgorithmED25519    SignatureAlgorithm = "ed25519"
	AlgorithmRSASHA256  SignatureAlgorithm = "rsa-sha256"
)

// SchemeStatus represents scheme lifecycle status
type SchemeStatus string

const (
	SchemeActive     SchemeStatus = "active"
	SchemeDeprecated SchemeStatus = "deprecated"
	SchemeDisabled   SchemeStatus = "disabled"
)

// SignatureConfig holds signature generation configuration
type SignatureConfig struct {
	// Header configuration
	SignatureHeader   string `json:"signature_header"`    // e.g., "X-Signature", "Webhook-Signature"
	TimestampHeader   string `json:"timestamp_header"`    // e.g., "X-Timestamp", "Webhook-Timestamp"
	IDHeader          string `json:"id_header"`           // e.g., "X-Webhook-ID", "Webhook-Id"
	
	// Signature format
	SignaturePrefix   string `json:"signature_prefix"`    // e.g., "sha256=", "v1="
	SignatureFormat   string `json:"signature_format"`    // hex, base64
	IncludeTimestamp  bool   `json:"include_timestamp"`
	IncludeMessageID  bool   `json:"include_message_id"`
	
	// Payload signing
	PayloadEncoding   string `json:"payload_encoding"`    // raw, base64
	SignedPayloadTemplate string `json:"signed_payload_template"` // e.g., "{id}.{timestamp}.{body}"
	
	// Tolerance settings
	TimestampToleranceSec int `json:"timestamp_tolerance_sec"`
	
	// Multi-signature support (like Stripe v1,v0)
	VersionPrefix     string   `json:"version_prefix"`     // e.g., "v1"
	SupportedVersions []string `json:"supported_versions"` // e.g., ["v1", "v0"]
}

// KeyConfiguration holds key management settings
type KeyConfiguration struct {
	KeyType           KeyType       `json:"key_type"`
	RotationPolicy    RotationPolicy `json:"rotation_policy"`
	MinKeyAge         time.Duration `json:"min_key_age"`
	MaxKeyAge         time.Duration `json:"max_key_age"`
	OverlapPeriod     time.Duration `json:"overlap_period"`
	AutoRotate        bool          `json:"auto_rotate"`
	NotifyOnRotation  bool          `json:"notify_on_rotation"`
	KeyLength         int           `json:"key_length"`
}

// KeyType defines key types
type KeyType string

const (
	KeyTypeSymmetric  KeyType = "symmetric"
	KeyTypeAsymmetric KeyType = "asymmetric"
)

// RotationPolicy defines key rotation strategies
type RotationPolicy string

const (
	RotationManual   RotationPolicy = "manual"
	RotationScheduled RotationPolicy = "scheduled"
	RotationOnDemand RotationPolicy = "on_demand"
)

// SigningKey represents a signing key
type SigningKey struct {
	ID          string    `json:"id"`
	SchemeID    string    `json:"scheme_id"`
	TenantID    string    `json:"tenant_id"`
	Version     int       `json:"version"`
	Algorithm   SignatureAlgorithm `json:"algorithm"`
	Status      KeyStatus `json:"status"`
	
	// For symmetric keys (HMAC)
	SecretKey   string    `json:"secret_key,omitempty"` // Encrypted
	SecretHash  string    `json:"secret_hash,omitempty"`
	
	// For asymmetric keys
	PublicKey   string    `json:"public_key,omitempty"`
	PrivateKey  string    `json:"private_key,omitempty"` // Encrypted
	
	// Key metadata
	Fingerprint string    `json:"fingerprint"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	UsageCount  int64     `json:"usage_count"`
}

// KeyStatus defines key lifecycle status
type KeyStatus string

const (
	KeyPending  KeyStatus = "pending"   // Created but not yet active
	KeyActive   KeyStatus = "active"    // Currently used for signing
	KeyPrimary  KeyStatus = "primary"   // Primary signing key
	KeyRotating KeyStatus = "rotating"  // Being phased out
	KeyExpired  KeyStatus = "expired"   // Past expiration
	KeyRevoked  KeyStatus = "revoked"   // Manually revoked
)

// SignatureRequest represents a request to sign a payload
type SignatureRequest struct {
	SchemeID  string            `json:"scheme_id"`
	Payload   []byte            `json:"payload"`
	MessageID string            `json:"message_id,omitempty"`
	Timestamp *time.Time        `json:"timestamp,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// SignatureResult represents the signed result
type SignatureResult struct {
	Signature     string            `json:"signature"`
	Headers       map[string]string `json:"headers"`
	KeyID         string            `json:"key_id"`
	KeyVersion    int               `json:"key_version"`
	Algorithm     SignatureAlgorithm `json:"algorithm"`
	Timestamp     time.Time         `json:"timestamp"`
	MessageID     string            `json:"message_id"`
	SignedPayload string            `json:"signed_payload,omitempty"`
}

// VerifyRequest represents a signature verification request
type VerifyRequest struct {
	SchemeID    string            `json:"scheme_id"`
	Payload     []byte            `json:"payload"`
	Signature   string            `json:"signature"`
	Timestamp   *time.Time        `json:"timestamp,omitempty"`
	MessageID   string            `json:"message_id,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// VerifyResult represents verification result
type VerifyResult struct {
	Valid        bool      `json:"valid"`
	KeyID        string    `json:"key_id,omitempty"`
	KeyVersion   int       `json:"key_version,omitempty"`
	VerifiedAt   time.Time `json:"verified_at"`
	Error        string    `json:"error,omitempty"`
	ErrorCode    string    `json:"error_code,omitempty"`
	TimestampAge time.Duration `json:"timestamp_age,omitempty"`
}

// KeyRotation represents a key rotation event
type KeyRotation struct {
	ID           string    `json:"id"`
	SchemeID     string    `json:"scheme_id"`
	TenantID     string    `json:"tenant_id"`
	OldKeyID     string    `json:"old_key_id"`
	NewKeyID     string    `json:"new_key_id"`
	Status       RotationStatus `json:"status"`
	Reason       string    `json:"reason,omitempty"`
	ScheduledAt  time.Time `json:"scheduled_at"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	OverlapUntil *time.Time `json:"overlap_until,omitempty"`
	Error        string    `json:"error,omitempty"`
}

// RotationStatus defines rotation lifecycle
type RotationStatus string

const (
	RotationScheduledStatus RotationStatus = "scheduled"
	RotationInProgress      RotationStatus = "in_progress"
	RotationCompleted       RotationStatus = "completed"
	RotationFailed          RotationStatus = "failed"
	RotationCancelled       RotationStatus = "cancelled"
)

// CreateSchemeRequest represents scheme creation request
type CreateSchemeRequest struct {
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description,omitempty"`
	Type        SignatureType     `json:"type" binding:"required"`
	Algorithm   SignatureAlgorithm `json:"algorithm,omitempty"`
	Config      *SignatureConfig  `json:"config,omitempty"`
	KeyConfig   *KeyConfiguration `json:"key_config,omitempty"`
}

// UpdateSchemeRequest represents scheme update request
type UpdateSchemeRequest struct {
	Name        *string           `json:"name,omitempty"`
	Description *string           `json:"description,omitempty"`
	Config      *SignatureConfig  `json:"config,omitempty"`
	KeyConfig   *KeyConfiguration `json:"key_config,omitempty"`
}

// RotateKeyRequest represents key rotation request
type RotateKeyRequest struct {
	Reason      string     `json:"reason,omitempty"`
	ScheduleAt  *time.Time `json:"schedule_at,omitempty"`
	Immediate   bool       `json:"immediate,omitempty"`
	OverlapHours int       `json:"overlap_hours,omitempty"`
}

// MigrationRequest represents signature scheme migration
type MigrationRequest struct {
	FromSchemeID string `json:"from_scheme_id" binding:"required"`
	ToSchemeID   string `json:"to_scheme_id" binding:"required"`
	WebhookIDs   []string `json:"webhook_ids,omitempty"`
	DryRun       bool   `json:"dry_run,omitempty"`
}

// MigrationResult represents migration result
type MigrationResult struct {
	Success        bool     `json:"success"`
	MigratedCount  int      `json:"migrated_count"`
	FailedCount    int      `json:"failed_count"`
	FailedWebhooks []string `json:"failed_webhooks,omitempty"`
	Errors         []string `json:"errors,omitempty"`
}

// SchemeStats represents scheme statistics
type SchemeStats struct {
	SchemeID       string    `json:"scheme_id"`
	TotalSigned    int64     `json:"total_signed"`
	TotalVerified  int64     `json:"total_verified"`
	TotalFailed    int64     `json:"total_failed"`
	ActiveKeys     int       `json:"active_keys"`
	LastSignedAt   *time.Time `json:"last_signed_at,omitempty"`
	LastVerifiedAt *time.Time `json:"last_verified_at,omitempty"`
	AvgSignTimeMs  float64   `json:"avg_sign_time_ms"`
}

// GetDefaultConfig returns default signature configuration for a type
func GetDefaultConfig(sigType SignatureType) SignatureConfig {
	switch sigType {
	case TypeStandardWebhooks:
		return SignatureConfig{
			SignatureHeader:       "Webhook-Signature",
			TimestampHeader:       "Webhook-Timestamp",
			IDHeader:              "Webhook-Id",
			SignaturePrefix:       "v1,",
			SignatureFormat:       "base64",
			IncludeTimestamp:      true,
			IncludeMessageID:      true,
			PayloadEncoding:       "raw",
			SignedPayloadTemplate: "{id}.{timestamp}.{body}",
			TimestampToleranceSec: 300,
			VersionPrefix:         "v1",
			SupportedVersions:     []string{"v1"},
		}
	case TypeStripe:
		return SignatureConfig{
			SignatureHeader:       "Stripe-Signature",
			SignaturePrefix:       "v1=",
			SignatureFormat:       "hex",
			IncludeTimestamp:      true,
			PayloadEncoding:       "raw",
			SignedPayloadTemplate: "{timestamp}.{body}",
			TimestampToleranceSec: 300,
			VersionPrefix:         "v1",
			SupportedVersions:     []string{"v1", "v0"},
		}
	case TypeGitHub:
		return SignatureConfig{
			SignatureHeader:       "X-Hub-Signature-256",
			SignaturePrefix:       "sha256=",
			SignatureFormat:       "hex",
			IncludeTimestamp:      false,
			PayloadEncoding:       "raw",
			SignedPayloadTemplate: "{body}",
			TimestampToleranceSec: 0,
		}
	case TypeSlack:
		return SignatureConfig{
			SignatureHeader:       "X-Slack-Signature",
			TimestampHeader:       "X-Slack-Request-Timestamp",
			SignaturePrefix:       "v0=",
			SignatureFormat:       "hex",
			IncludeTimestamp:      true,
			PayloadEncoding:       "raw",
			SignedPayloadTemplate: "v0:{timestamp}:{body}",
			TimestampToleranceSec: 300,
			VersionPrefix:         "v0",
		}
	default:
		return SignatureConfig{
			SignatureHeader:       "X-Webhook-Signature",
			SignatureFormat:       "hex",
			IncludeTimestamp:      true,
			PayloadEncoding:       "raw",
			SignedPayloadTemplate: "{timestamp}.{body}",
			TimestampToleranceSec: 300,
		}
	}
}

// GetDefaultKeyConfig returns default key configuration
func GetDefaultKeyConfig() KeyConfiguration {
	return KeyConfiguration{
		KeyType:          KeyTypeSymmetric,
		RotationPolicy:   RotationManual,
		MinKeyAge:        24 * time.Hour,
		MaxKeyAge:        90 * 24 * time.Hour,
		OverlapPeriod:    24 * time.Hour,
		AutoRotate:       false,
		NotifyOnRotation: true,
		KeyLength:        32,
	}
}

// GetSupportedSchemes returns supported signature schemes
func GetSupportedSchemes() []SignatureSchemeInfo {
	return []SignatureSchemeInfo{
		{Type: TypeStandardWebhooks, Name: "Standard Webhooks", Description: "Industry standard signature format (standardwebhooks.com)", DefaultAlgorithm: AlgorithmHMACSHA256},
		{Type: TypeStripe, Name: "Stripe-style", Description: "Stripe webhook signature format", DefaultAlgorithm: AlgorithmHMACSHA256},
		{Type: TypeGitHub, Name: "GitHub-style", Description: "GitHub webhook signature format", DefaultAlgorithm: AlgorithmHMACSHA256},
		{Type: TypeSlack, Name: "Slack-style", Description: "Slack webhook signature format", DefaultAlgorithm: AlgorithmHMACSHA256},
		{Type: TypeCustomHMAC, Name: "Custom HMAC", Description: "Custom HMAC-based signatures", DefaultAlgorithm: AlgorithmHMACSHA256},
		{Type: TypeAWSSNS, Name: "AWS SNS", Description: "AWS SNS message signatures", DefaultAlgorithm: AlgorithmRSASHA256},
	}
}

// SignatureSchemeInfo describes a signature scheme type
type SignatureSchemeInfo struct {
	Type             SignatureType      `json:"type"`
	Name             string             `json:"name"`
	Description      string             `json:"description"`
	DefaultAlgorithm SignatureAlgorithm `json:"default_algorithm"`
}
