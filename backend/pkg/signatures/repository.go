package signatures

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Repository defines signature data access
type Repository interface {
	// Schemes
	SaveScheme(ctx context.Context, scheme *SignatureScheme) error
	GetScheme(ctx context.Context, tenantID, schemeID string) (*SignatureScheme, error)
	ListSchemes(ctx context.Context, tenantID string) ([]SignatureScheme, error)
	DeleteScheme(ctx context.Context, tenantID, schemeID string) error
	
	// Keys
	SaveKey(ctx context.Context, key *SigningKey) error
	GetKey(ctx context.Context, keyID string) (*SigningKey, error)
	GetPrimaryKey(ctx context.Context, schemeID string) (*SigningKey, error)
	ListKeys(ctx context.Context, schemeID string) ([]SigningKey, error)
	UpdateKeyStatus(ctx context.Context, keyID string, status KeyStatus) error
	UpdateKeyUsage(ctx context.Context, keyID string) error
	
	// Rotations
	SaveRotation(ctx context.Context, rotation *KeyRotation) error
	GetRotation(ctx context.Context, rotationID string) (*KeyRotation, error)
	ListRotations(ctx context.Context, schemeID string) ([]KeyRotation, error)
	GetPendingRotations(ctx context.Context) ([]KeyRotation, error)
	
	// Stats
	GetSchemeStats(ctx context.Context, schemeID string) (*SchemeStats, error)
	IncrementSignCount(ctx context.Context, schemeID string) error
	IncrementVerifyCount(ctx context.Context, schemeID string, success bool) error
}

// PostgresRepository implements Repository with PostgreSQL
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// SaveScheme saves a signature scheme
func (r *PostgresRepository) SaveScheme(ctx context.Context, scheme *SignatureScheme) error {
	configJSON, _ := json.Marshal(scheme.Config)
	keyConfigJSON, _ := json.Marshal(scheme.KeyConfig)

	query := `
		INSERT INTO signature_schemes (
			id, tenant_id, name, description, type, algorithm,
			config, key_config, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			config = EXCLUDED.config,
			key_config = EXCLUDED.key_config,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at`

	_, err := r.db.ExecContext(ctx, query,
		scheme.ID, scheme.TenantID, scheme.Name, scheme.Description,
		scheme.Type, scheme.Algorithm, configJSON, keyConfigJSON,
		scheme.Status, scheme.CreatedAt, scheme.UpdatedAt)

	return err
}

// GetScheme retrieves a signature scheme
func (r *PostgresRepository) GetScheme(ctx context.Context, tenantID, schemeID string) (*SignatureScheme, error) {
	query := `
		SELECT id, tenant_id, name, description, type, algorithm,
			   config, key_config, status, created_at, updated_at
		FROM signature_schemes
		WHERE tenant_id = $1 AND id = $2`

	var scheme SignatureScheme
	var configJSON, keyConfigJSON []byte
	var description sql.NullString

	err := r.db.QueryRowContext(ctx, query, tenantID, schemeID).Scan(
		&scheme.ID, &scheme.TenantID, &scheme.Name, &description,
		&scheme.Type, &scheme.Algorithm, &configJSON, &keyConfigJSON,
		&scheme.Status, &scheme.CreatedAt, &scheme.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("signature scheme not found")
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(configJSON, &scheme.Config)
	json.Unmarshal(keyConfigJSON, &scheme.KeyConfig)
	if description.Valid {
		scheme.Description = description.String
	}

	return &scheme, nil
}

// ListSchemes lists signature schemes
func (r *PostgresRepository) ListSchemes(ctx context.Context, tenantID string) ([]SignatureScheme, error) {
	query := `
		SELECT id, tenant_id, name, description, type, algorithm,
			   config, key_config, status, created_at, updated_at
		FROM signature_schemes
		WHERE tenant_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schemes []SignatureScheme
	for rows.Next() {
		var scheme SignatureScheme
		var configJSON, keyConfigJSON []byte
		var description sql.NullString

		err := rows.Scan(
			&scheme.ID, &scheme.TenantID, &scheme.Name, &description,
			&scheme.Type, &scheme.Algorithm, &configJSON, &keyConfigJSON,
			&scheme.Status, &scheme.CreatedAt, &scheme.UpdatedAt)
		if err != nil {
			continue
		}

		json.Unmarshal(configJSON, &scheme.Config)
		json.Unmarshal(keyConfigJSON, &scheme.KeyConfig)
		if description.Valid {
			scheme.Description = description.String
		}

		schemes = append(schemes, scheme)
	}

	return schemes, nil
}

// DeleteScheme deletes a signature scheme
func (r *PostgresRepository) DeleteScheme(ctx context.Context, tenantID, schemeID string) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM signature_schemes WHERE tenant_id = $1 AND id = $2",
		tenantID, schemeID)
	return err
}

// SaveKey saves a signing key
func (r *PostgresRepository) SaveKey(ctx context.Context, key *SigningKey) error {
	query := `
		INSERT INTO signing_keys (
			id, scheme_id, tenant_id, version, algorithm, status,
			secret_key, secret_hash, public_key, private_key, fingerprint,
			created_at, expires_at, revoked_at, last_used_at, usage_count
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			revoked_at = EXCLUDED.revoked_at,
			last_used_at = EXCLUDED.last_used_at,
			usage_count = EXCLUDED.usage_count`

	_, err := r.db.ExecContext(ctx, query,
		key.ID, key.SchemeID, key.TenantID, key.Version, key.Algorithm, key.Status,
		key.SecretKey, key.SecretHash, key.PublicKey, key.PrivateKey, key.Fingerprint,
		key.CreatedAt, key.ExpiresAt, key.RevokedAt, key.LastUsedAt, key.UsageCount)

	return err
}

// GetKey retrieves a signing key
func (r *PostgresRepository) GetKey(ctx context.Context, keyID string) (*SigningKey, error) {
	query := `
		SELECT id, scheme_id, tenant_id, version, algorithm, status,
			   secret_key, secret_hash, public_key, private_key, fingerprint,
			   created_at, expires_at, revoked_at, last_used_at, usage_count
		FROM signing_keys
		WHERE id = $1`

	var key SigningKey
	var secretKey, secretHash, publicKey, privateKey sql.NullString
	var expiresAt, revokedAt, lastUsedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, keyID).Scan(
		&key.ID, &key.SchemeID, &key.TenantID, &key.Version, &key.Algorithm, &key.Status,
		&secretKey, &secretHash, &publicKey, &privateKey, &key.Fingerprint,
		&key.CreatedAt, &expiresAt, &revokedAt, &lastUsedAt, &key.UsageCount)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("signing key not found")
	}
	if err != nil {
		return nil, err
	}

	if secretKey.Valid {
		key.SecretKey = secretKey.String
	}
	if secretHash.Valid {
		key.SecretHash = secretHash.String
	}
	if publicKey.Valid {
		key.PublicKey = publicKey.String
	}
	if privateKey.Valid {
		key.PrivateKey = privateKey.String
	}
	if expiresAt.Valid {
		key.ExpiresAt = &expiresAt.Time
	}
	if revokedAt.Valid {
		key.RevokedAt = &revokedAt.Time
	}
	if lastUsedAt.Valid {
		key.LastUsedAt = &lastUsedAt.Time
	}

	return &key, nil
}

// GetPrimaryKey gets the primary signing key for a scheme
func (r *PostgresRepository) GetPrimaryKey(ctx context.Context, schemeID string) (*SigningKey, error) {
	query := `
		SELECT id, scheme_id, tenant_id, version, algorithm, status,
			   secret_key, secret_hash, public_key, private_key, fingerprint,
			   created_at, expires_at, revoked_at, last_used_at, usage_count
		FROM signing_keys
		WHERE scheme_id = $1 AND status IN ('primary', 'active')
		ORDER BY CASE WHEN status = 'primary' THEN 0 ELSE 1 END, version DESC
		LIMIT 1`

	var key SigningKey
	var secretKey, secretHash, publicKey, privateKey sql.NullString
	var expiresAt, revokedAt, lastUsedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, schemeID).Scan(
		&key.ID, &key.SchemeID, &key.TenantID, &key.Version, &key.Algorithm, &key.Status,
		&secretKey, &secretHash, &publicKey, &privateKey, &key.Fingerprint,
		&key.CreatedAt, &expiresAt, &revokedAt, &lastUsedAt, &key.UsageCount)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no active signing key found")
	}
	if err != nil {
		return nil, err
	}

	if secretKey.Valid {
		key.SecretKey = secretKey.String
	}
	if secretHash.Valid {
		key.SecretHash = secretHash.String
	}
	if publicKey.Valid {
		key.PublicKey = publicKey.String
	}
	if privateKey.Valid {
		key.PrivateKey = privateKey.String
	}
	if expiresAt.Valid {
		key.ExpiresAt = &expiresAt.Time
	}
	if revokedAt.Valid {
		key.RevokedAt = &revokedAt.Time
	}
	if lastUsedAt.Valid {
		key.LastUsedAt = &lastUsedAt.Time
	}

	return &key, nil
}

// ListKeys lists all keys for a scheme
func (r *PostgresRepository) ListKeys(ctx context.Context, schemeID string) ([]SigningKey, error) {
	query := `
		SELECT id, scheme_id, tenant_id, version, algorithm, status,
			   fingerprint, created_at, expires_at, revoked_at, last_used_at, usage_count
		FROM signing_keys
		WHERE scheme_id = $1
		ORDER BY version DESC`

	rows, err := r.db.QueryContext(ctx, query, schemeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []SigningKey
	for rows.Next() {
		var key SigningKey
		var expiresAt, revokedAt, lastUsedAt sql.NullTime

		err := rows.Scan(
			&key.ID, &key.SchemeID, &key.TenantID, &key.Version, &key.Algorithm, &key.Status,
			&key.Fingerprint, &key.CreatedAt, &expiresAt, &revokedAt, &lastUsedAt, &key.UsageCount)
		if err != nil {
			continue
		}

		if expiresAt.Valid {
			key.ExpiresAt = &expiresAt.Time
		}
		if revokedAt.Valid {
			key.RevokedAt = &revokedAt.Time
		}
		if lastUsedAt.Valid {
			key.LastUsedAt = &lastUsedAt.Time
		}

		keys = append(keys, key)
	}

	return keys, nil
}

// UpdateKeyStatus updates key status
func (r *PostgresRepository) UpdateKeyStatus(ctx context.Context, keyID string, status KeyStatus) error {
	var revokedAt *time.Time
	if status == KeyRevoked {
		now := time.Now()
		revokedAt = &now
	}

	_, err := r.db.ExecContext(ctx,
		"UPDATE signing_keys SET status = $1, revoked_at = $2 WHERE id = $3",
		status, revokedAt, keyID)
	return err
}

// UpdateKeyUsage updates key last used time and count
func (r *PostgresRepository) UpdateKeyUsage(ctx context.Context, keyID string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE signing_keys SET last_used_at = NOW(), usage_count = usage_count + 1 WHERE id = $1",
		keyID)
	return err
}

// SaveRotation saves a key rotation
func (r *PostgresRepository) SaveRotation(ctx context.Context, rotation *KeyRotation) error {
	query := `
		INSERT INTO key_rotations (
			id, scheme_id, tenant_id, old_key_id, new_key_id, status,
			reason, scheduled_at, started_at, completed_at, overlap_until, error
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			started_at = EXCLUDED.started_at,
			completed_at = EXCLUDED.completed_at,
			error = EXCLUDED.error`

	_, err := r.db.ExecContext(ctx, query,
		rotation.ID, rotation.SchemeID, rotation.TenantID,
		rotation.OldKeyID, rotation.NewKeyID, rotation.Status,
		rotation.Reason, rotation.ScheduledAt, rotation.StartedAt,
		rotation.CompletedAt, rotation.OverlapUntil, rotation.Error)

	return err
}

// GetRotation retrieves a rotation
func (r *PostgresRepository) GetRotation(ctx context.Context, rotationID string) (*KeyRotation, error) {
	query := `
		SELECT id, scheme_id, tenant_id, old_key_id, new_key_id, status,
			   reason, scheduled_at, started_at, completed_at, overlap_until, error
		FROM key_rotations
		WHERE id = $1`

	var rotation KeyRotation
	var reason, errMsg sql.NullString
	var startedAt, completedAt, overlapUntil sql.NullTime

	err := r.db.QueryRowContext(ctx, query, rotationID).Scan(
		&rotation.ID, &rotation.SchemeID, &rotation.TenantID,
		&rotation.OldKeyID, &rotation.NewKeyID, &rotation.Status,
		&reason, &rotation.ScheduledAt, &startedAt, &completedAt, &overlapUntil, &errMsg)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("rotation not found")
	}
	if err != nil {
		return nil, err
	}

	if reason.Valid {
		rotation.Reason = reason.String
	}
	if startedAt.Valid {
		rotation.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		rotation.CompletedAt = &completedAt.Time
	}
	if overlapUntil.Valid {
		rotation.OverlapUntil = &overlapUntil.Time
	}
	if errMsg.Valid {
		rotation.Error = errMsg.String
	}

	return &rotation, nil
}

// ListRotations lists rotations for a scheme
func (r *PostgresRepository) ListRotations(ctx context.Context, schemeID string) ([]KeyRotation, error) {
	query := `
		SELECT id, scheme_id, tenant_id, old_key_id, new_key_id, status,
			   reason, scheduled_at, started_at, completed_at, overlap_until, error
		FROM key_rotations
		WHERE scheme_id = $1
		ORDER BY scheduled_at DESC`

	rows, err := r.db.QueryContext(ctx, query, schemeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rotations []KeyRotation
	for rows.Next() {
		var rotation KeyRotation
		var reason, errMsg sql.NullString
		var startedAt, completedAt, overlapUntil sql.NullTime

		err := rows.Scan(
			&rotation.ID, &rotation.SchemeID, &rotation.TenantID,
			&rotation.OldKeyID, &rotation.NewKeyID, &rotation.Status,
			&reason, &rotation.ScheduledAt, &startedAt, &completedAt, &overlapUntil, &errMsg)
		if err != nil {
			continue
		}

		if reason.Valid {
			rotation.Reason = reason.String
		}
		if startedAt.Valid {
			rotation.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			rotation.CompletedAt = &completedAt.Time
		}
		if overlapUntil.Valid {
			rotation.OverlapUntil = &overlapUntil.Time
		}
		if errMsg.Valid {
			rotation.Error = errMsg.String
		}

		rotations = append(rotations, rotation)
	}

	return rotations, nil
}

// GetPendingRotations gets rotations that need processing
func (r *PostgresRepository) GetPendingRotations(ctx context.Context) ([]KeyRotation, error) {
	query := `
		SELECT id, scheme_id, tenant_id, old_key_id, new_key_id, status,
			   reason, scheduled_at, started_at, completed_at, overlap_until, error
		FROM key_rotations
		WHERE status IN ('scheduled', 'in_progress')
		  AND scheduled_at <= NOW()
		ORDER BY scheduled_at ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rotations []KeyRotation
	for rows.Next() {
		var rotation KeyRotation
		var reason, errMsg sql.NullString
		var startedAt, completedAt, overlapUntil sql.NullTime

		err := rows.Scan(
			&rotation.ID, &rotation.SchemeID, &rotation.TenantID,
			&rotation.OldKeyID, &rotation.NewKeyID, &rotation.Status,
			&reason, &rotation.ScheduledAt, &startedAt, &completedAt, &overlapUntil, &errMsg)
		if err != nil {
			continue
		}

		if reason.Valid {
			rotation.Reason = reason.String
		}
		if startedAt.Valid {
			rotation.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			rotation.CompletedAt = &completedAt.Time
		}
		if overlapUntil.Valid {
			rotation.OverlapUntil = &overlapUntil.Time
		}
		if errMsg.Valid {
			rotation.Error = errMsg.String
		}

		rotations = append(rotations, rotation)
	}

	return rotations, nil
}

// GetSchemeStats retrieves scheme statistics
func (r *PostgresRepository) GetSchemeStats(ctx context.Context, schemeID string) (*SchemeStats, error) {
	query := `
		SELECT 
			COALESCE(SUM(total_signed), 0) as total_signed,
			COALESCE(SUM(total_verified), 0) as total_verified,
			COALESCE(SUM(total_failed), 0) as total_failed,
			MAX(last_signed_at) as last_signed,
			MAX(last_verified_at) as last_verified
		FROM signature_stats
		WHERE scheme_id = $1`

	stats := &SchemeStats{SchemeID: schemeID}
	var lastSigned, lastVerified sql.NullTime

	err := r.db.QueryRowContext(ctx, query, schemeID).Scan(
		&stats.TotalSigned, &stats.TotalVerified, &stats.TotalFailed,
		&lastSigned, &lastVerified)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if lastSigned.Valid {
		stats.LastSignedAt = &lastSigned.Time
	}
	if lastVerified.Valid {
		stats.LastVerifiedAt = &lastVerified.Time
	}

	// Count active keys
	r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM signing_keys WHERE scheme_id = $1 AND status IN ('active', 'primary')",
		schemeID).Scan(&stats.ActiveKeys)

	return stats, nil
}

// IncrementSignCount increments signature count
func (r *PostgresRepository) IncrementSignCount(ctx context.Context, schemeID string) error {
	query := `
		INSERT INTO signature_stats (scheme_id, total_signed, last_signed_at)
		VALUES ($1, 1, NOW())
		ON CONFLICT (scheme_id) DO UPDATE SET
			total_signed = signature_stats.total_signed + 1,
			last_signed_at = NOW()`
	_, err := r.db.ExecContext(ctx, query, schemeID)
	return err
}

// IncrementVerifyCount increments verification count
func (r *PostgresRepository) IncrementVerifyCount(ctx context.Context, schemeID string, success bool) error {
	var query string
	if success {
		query = `
			INSERT INTO signature_stats (scheme_id, total_verified, last_verified_at)
			VALUES ($1, 1, NOW())
			ON CONFLICT (scheme_id) DO UPDATE SET
				total_verified = signature_stats.total_verified + 1,
				last_verified_at = NOW()`
	} else {
		query = `
			INSERT INTO signature_stats (scheme_id, total_failed)
			VALUES ($1, 1)
			ON CONFLICT (scheme_id) DO UPDATE SET
				total_failed = signature_stats.total_failed + 1`
	}
	_, err := r.db.ExecContext(ctx, query, schemeID)
	return err
}

// GenerateSchemeID generates a new scheme ID
func GenerateSchemeID() string {
	return uuid.New().String()
}

// GenerateKeyID generates a new key ID
func GenerateKeyID() string {
	return uuid.New().String()
}

// GenerateRotationID generates a new rotation ID
func GenerateRotationID() string {
	return uuid.New().String()
}
