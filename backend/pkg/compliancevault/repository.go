package compliancevault

import "context"

// Repository defines the data access interface for the compliance vault.
type Repository interface {
	// Vault entries
	StoreEntry(ctx context.Context, entry *VaultEntry) error
	GetEntry(ctx context.Context, tenantID, entryID string) (*VaultEntry, error)
	ListEntries(ctx context.Context, tenantID string, limit, offset int) ([]VaultEntry, error)
	DeleteEntry(ctx context.Context, tenantID, entryID string) error
	DeleteEntriesBySubject(ctx context.Context, tenantID, subjectID string) (int, error)
	GetExpiredEntries(ctx context.Context, limit int) ([]VaultEntry, error)

	// Retention policies
	CreateRetentionPolicy(ctx context.Context, policy *RetentionPolicy) error
	GetRetentionPolicy(ctx context.Context, tenantID, policyID string) (*RetentionPolicy, error)
	ListRetentionPolicies(ctx context.Context, tenantID string) ([]RetentionPolicy, error)
	UpdateRetentionPolicy(ctx context.Context, policy *RetentionPolicy) error
	DeleteRetentionPolicy(ctx context.Context, tenantID, policyID string) error

	// Audit trail
	RecordAudit(ctx context.Context, entry *AuditTrailEntry) error
	ListAuditTrail(ctx context.Context, tenantID string, limit, offset int) ([]AuditTrailEntry, error)
	GetAuditTrailForEntry(ctx context.Context, entryID string) ([]AuditTrailEntry, error)

	// Encryption keys
	StoreKey(ctx context.Context, key *EncryptionKey) error
	GetActiveKey(ctx context.Context, tenantID string) (*EncryptionKey, error)
	RotateKey(ctx context.Context, tenantID string) (*EncryptionKey, error)

	// Erasure requests
	CreateErasureRequest(ctx context.Context, req *ErasureRequest) error
	GetErasureRequest(ctx context.Context, tenantID, requestID string) (*ErasureRequest, error)
	ListErasureRequests(ctx context.Context, tenantID string) ([]ErasureRequest, error)
	UpdateErasureRequest(ctx context.Context, req *ErasureRequest) error

	// Stats
	GetVaultStats(ctx context.Context, tenantID string) (*VaultStats, error)
}

// Encryptor provides encryption/decryption operations.
type Encryptor interface {
	Encrypt(plaintext []byte, keyID string) (ciphertext []byte, err error)
	Decrypt(ciphertext []byte, keyID string) (plaintext []byte, err error)
}
