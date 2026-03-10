package e2ee

import "time"

// KeyPair represents an X25519 key pair for an endpoint.
type KeyPair struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	EndpointID string     `json:"endpoint_id"`
	PublicKey  string     `json:"public_key"` // base64-encoded
	PrivateKey string     `json:"-"`          // never exposed via API
	Algorithm  string     `json:"algorithm"`  // x25519
	Status     string     `json:"status"`     // active, rotated, revoked
	Version    int        `json:"version"`
	CreatedAt  time.Time  `json:"created_at"`
	RotatedAt  *time.Time `json:"rotated_at,omitempty"`
	ExpiresAt  time.Time  `json:"expires_at"`
}

// EncryptedPayload wraps an encrypted webhook payload.
type EncryptedPayload struct {
	CiphertextBase64 string `json:"ciphertext"`
	NonceBase64      string `json:"nonce"`
	EphemeralPubKey  string `json:"ephemeral_public_key"`
	KeyVersion       int    `json:"key_version"`
	Algorithm        string `json:"algorithm"`
}

// DecryptedPayload is the result of a successful decryption.
type DecryptedPayload struct {
	Plaintext  []byte `json:"plaintext"`
	KeyVersion int    `json:"key_version"`
	Verified   bool   `json:"verified"`
}

// KeyRotationRequest configures a key rotation.
type KeyRotationRequest struct {
	EndpointID      string `json:"endpoint_id" binding:"required"`
	GracePeriodMins int    `json:"grace_period_mins"` // how long old key stays active
}

// KeyRotationResult describes the outcome of a rotation.
type KeyRotationResult struct {
	OldKeyVersion int       `json:"old_key_version"`
	NewKeyVersion int       `json:"new_key_version"`
	NewPublicKey  string    `json:"new_public_key"`
	GracePeriod   string    `json:"grace_period"`
	RotatedAt     time.Time `json:"rotated_at"`
}

// AuditEntry records a cryptographic operation for compliance.
type AuditEntry struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	EndpointID string    `json:"endpoint_id"`
	Operation  string    `json:"operation"` // key_generated, key_rotated, payload_encrypted, payload_decrypted, key_revoked
	KeyVersion int       `json:"key_version"`
	Success    bool      `json:"success"`
	Details    string    `json:"details,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// HealthCheck verifies encryption health for an endpoint.
type HealthCheck struct {
	EndpointID     string `json:"endpoint_id"`
	HasActiveKey   bool   `json:"has_active_key"`
	KeyVersion     int    `json:"key_version"`
	KeyAge         string `json:"key_age"`
	NeedsRotation  bool   `json:"needs_rotation"`
	EncryptionTest bool   `json:"encryption_test"` // round-trip test passed
	Status         string `json:"status"`          // healthy, warning, critical
}

// RegisterKeyRequest is used during endpoint registration.
type RegisterKeyRequest struct {
	EndpointID string `json:"endpoint_id" binding:"required"`
}

// EnvelopeEncryptedPayload uses AES-256-GCM envelope encryption with X25519 key exchange.
type EnvelopeEncryptedPayload struct {
	EncryptedDEK    string `json:"encrypted_dek"` // DEK encrypted with shared key (base64)
	DEKNonce        string `json:"dek_nonce"`     // Nonce used for DEK encryption (base64)
	Ciphertext      string `json:"ciphertext"`    // Payload encrypted with DEK (base64)
	PayloadNonce    string `json:"payload_nonce"` // Nonce used for payload encryption (base64)
	EphemeralPubKey string `json:"ephemeral_public_key"`
	KeyVersion      int    `json:"key_version"`
	Algorithm       string `json:"algorithm"` // x25519-aes256gcm
}

// KeyEscrowConfig configures key escrow for compliance scenarios.
type KeyEscrowConfig struct {
	Enabled         bool   `json:"enabled"`
	EscrowKeyID     string `json:"escrow_key_id,omitempty"`
	RequireDualAuth bool   `json:"require_dual_auth,omitempty"`
}

// BatchEncryptRequest allows encrypting to multiple endpoints at once.
type BatchEncryptRequest struct {
	EndpointIDs []string `json:"endpoint_ids" binding:"required"`
	Plaintext   []byte   `json:"plaintext" binding:"required"`
}

// BatchEncryptResult captures per-endpoint encryption results.
type BatchEncryptResult struct {
	Results map[string]*EncryptedPayload `json:"results"`
	Errors  map[string]string            `json:"errors,omitempty"`
}

// KeyMetrics aggregates encryption metrics for observability.
type KeyMetrics struct {
	TenantID            string `json:"tenant_id"`
	TotalKeys           int    `json:"total_keys"`
	ActiveKeys          int    `json:"active_keys"`
	RotatedKeys         int    `json:"rotated_keys"`
	RevokedKeys         int    `json:"revoked_keys"`
	EncryptionOps       int64  `json:"encryption_ops"`
	DecryptionOps       int64  `json:"decryption_ops"`
	KeysNeedingRotation int    `json:"keys_needing_rotation"`
}
