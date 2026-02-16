package security

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"github.com/josedab/waas/pkg/repository"
	"time"

	"github.com/google/uuid"
)

// SecretManager handles secure secret storage and rotation
type SecretManager struct {
	encryption *EncryptionService
	repository repository.SecretRepository
}

// NewSecretManager creates a new secret manager
func NewSecretManager(encryption *EncryptionService, repo repository.SecretRepository) *SecretManager {
	return &SecretManager{
		encryption: encryption,
		repository: repo,
	}
}

// GenerateSecret creates a new cryptographically secure secret
func (sm *SecretManager) GenerateSecret(ctx context.Context, tenantID uuid.UUID, secretID string, expiresAt *time.Time) (*repository.SecretVersion, string, error) {
	// Generate a new secret value
	secretBytes := make([]byte, 32) // 256-bit secret
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, "", fmt.Errorf("failed to generate secret: %w", err)
	}
	secretValue := hex.EncodeToString(secretBytes)

	// Get the next version number
	activeSecrets, err := sm.repository.GetActiveSecrets(ctx, tenantID, secretID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get active secrets: %w", err)
	}

	nextVersion := 1
	for _, secret := range activeSecrets {
		if secret.Version >= nextVersion {
			nextVersion = secret.Version + 1
		}
	}

	// Encrypt the secret value
	encryptedValue, err := sm.encryption.EncryptString(secretValue)
	if err != nil {
		return nil, "", fmt.Errorf("failed to encrypt secret: %w", err)
	}

	// Create the secret version
	secret := &repository.SecretVersion{
		ID:        uuid.New(),
		TenantID:  tenantID,
		SecretID:  secretID,
		Version:   nextVersion,
		Value:     encryptedValue,
		IsActive:  true,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	if err := sm.repository.CreateSecret(ctx, secret); err != nil {
		return nil, "", fmt.Errorf("failed to store secret: %w", err)
	}

	return secret, secretValue, nil
}

// GetActiveSecrets retrieves all active secrets for a given secret ID
func (sm *SecretManager) GetActiveSecrets(ctx context.Context, tenantID uuid.UUID, secretID string) ([]string, error) {
	secrets, err := sm.repository.GetActiveSecrets(ctx, tenantID, secretID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active secrets: %w", err)
	}

	var decryptedSecrets []string
	for _, secret := range secrets {
		// Skip expired secrets
		if secret.ExpiresAt != nil && secret.ExpiresAt.Before(time.Now()) {
			continue
		}

		decryptedValue, err := sm.encryption.DecryptString(secret.Value)
		if err != nil {
			// Log error but continue with other secrets
			continue
		}
		decryptedSecrets = append(decryptedSecrets, decryptedValue)
	}

	return decryptedSecrets, nil
}

// RotateSecret creates a new secret version and optionally deactivates old ones
func (sm *SecretManager) RotateSecret(ctx context.Context, tenantID uuid.UUID, secretID string, gracePeriod time.Duration) (*repository.SecretVersion, string, error) {
	// Generate new secret
	expiresAt := time.Now().Add(gracePeriod)
	newSecret, secretValue, err := sm.GenerateSecret(ctx, tenantID, secretID, &expiresAt)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate new secret: %w", err)
	}

	// Get existing active secrets
	activeSecrets, err := sm.repository.GetActiveSecrets(ctx, tenantID, secretID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get active secrets: %w", err)
	}

	// Set expiration time for old secrets (grace period)
	for _, secret := range activeSecrets {
		if secret.ID != newSecret.ID && secret.ExpiresAt == nil {
			// Update the secret to expire after grace period
			secret.ExpiresAt = &expiresAt
		}
	}

	return newSecret, secretValue, nil
}

// DeactivateSecret marks a secret version as inactive
func (sm *SecretManager) DeactivateSecret(ctx context.Context, secretID uuid.UUID) error {
	return sm.repository.UpdateSecretStatus(ctx, secretID, false)
}

// CleanupExpiredSecrets removes expired secrets from storage
func (sm *SecretManager) CleanupExpiredSecrets(ctx context.Context) error {
	return sm.repository.DeleteExpiredSecrets(ctx)
}

// ValidateSecret checks if a provided secret matches any active secret
func (sm *SecretManager) ValidateSecret(ctx context.Context, tenantID uuid.UUID, secretID string, providedSecret string) (bool, error) {
	activeSecrets, err := sm.GetActiveSecrets(ctx, tenantID, secretID)
	if err != nil {
		return false, fmt.Errorf("failed to get active secrets: %w", err)
	}

	for _, secret := range activeSecrets {
		if subtle.ConstantTimeCompare([]byte(secret), []byte(providedSecret)) == 1 {
			return true, nil
		}
	}

	return false, nil
}
