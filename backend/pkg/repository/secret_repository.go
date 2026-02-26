package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// SecretVersion represents a versioned secret
type SecretVersion struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	TenantID  uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	SecretID  string     `json:"secret_id" db:"secret_id"`
	Version   int        `json:"version" db:"version"`
	Value     string     `json:"-" db:"encrypted_value"` // Encrypted secret value
	IsActive  bool       `json:"is_active" db:"is_active"`
	ExpiresAt *time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

// SecretRepository defines the interface for secret storage
type SecretRepository interface {
	CreateSecret(ctx context.Context, secret *SecretVersion) error
	GetActiveSecrets(ctx context.Context, tenantID uuid.UUID, secretID string) ([]*SecretVersion, error)
	GetSecretByVersion(ctx context.Context, tenantID uuid.UUID, secretID string, version int) (*SecretVersion, error)
	UpdateSecretStatus(ctx context.Context, id uuid.UUID, isActive bool) error
	DeleteExpiredSecrets(ctx context.Context) error
}

type secretRepository struct {
	db *sqlx.DB
}

// NewSecretRepository creates a new secret repository
func NewSecretRepository(db *sqlx.DB) SecretRepository {
	return &secretRepository{db: db}
}

// CreateSecret stores a new secret version
func (r *secretRepository) CreateSecret(ctx context.Context, secret *SecretVersion) error {
	query := `
		INSERT INTO secret_versions (id, tenant_id, secret_id, version, encrypted_value, is_active, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.db.ExecContext(ctx, query,
		secret.ID,
		secret.TenantID,
		secret.SecretID,
		secret.Version,
		secret.Value,
		secret.IsActive,
		secret.ExpiresAt,
		secret.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}

	return nil
}

// GetActiveSecrets retrieves all active secrets for a given secret ID
func (r *secretRepository) GetActiveSecrets(ctx context.Context, tenantID uuid.UUID, secretID string) ([]*SecretVersion, error) {
	query := `
		SELECT id, tenant_id, secret_id, version, encrypted_value, is_active, expires_at, created_at
		FROM secret_versions
		WHERE tenant_id = $1 AND secret_id = $2 AND is_active = true
		ORDER BY version DESC`

	var secrets []*SecretVersion
	err := r.db.SelectContext(ctx, &secrets, query, tenantID, secretID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active secrets: %w", err)
	}

	return secrets, nil
}

// GetSecretByVersion retrieves a specific secret version
func (r *secretRepository) GetSecretByVersion(ctx context.Context, tenantID uuid.UUID, secretID string, version int) (*SecretVersion, error) {
	query := `
		SELECT id, tenant_id, secret_id, version, encrypted_value, is_active, expires_at, created_at
		FROM secret_versions
		WHERE tenant_id = $1 AND secret_id = $2 AND version = $3`

	var secret SecretVersion
	err := r.db.GetContext(ctx, &secret, query, tenantID, secretID, version)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("secret version not found")
		}
		return nil, fmt.Errorf("failed to get secret by version: %w", err)
	}

	return &secret, nil
}

// UpdateSecretStatus updates the active status of a secret
func (r *secretRepository) UpdateSecretStatus(ctx context.Context, id uuid.UUID, isActive bool) error {
	query := `UPDATE secret_versions SET is_active = $1 WHERE id = $2`

	result, err := r.db.ExecContext(ctx, query, isActive, id)
	if err != nil {
		return fmt.Errorf("failed to update secret status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("secret not found")
	}

	return nil
}

// DeleteExpiredSecrets removes expired secrets from storage
func (r *secretRepository) DeleteExpiredSecrets(ctx context.Context) error {
	query := `DELETE FROM secret_versions WHERE expires_at IS NOT NULL AND expires_at < $1`

	result, err := r.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete expired secrets: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		log.Printf("secret_repository: deleted %d expired secrets", rowsAffected)
	}

	return nil
}