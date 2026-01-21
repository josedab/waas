package security

import (
	"context"
	"crypto/rand"
	"testing"
	"time"
	"webhook-platform/pkg/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockSecretRepository is a mock implementation of SecretRepository
type MockSecretRepository struct {
	mock.Mock
}

func (m *MockSecretRepository) CreateSecret(ctx context.Context, secret *repository.SecretVersion) error {
	args := m.Called(ctx, secret)
	return args.Error(0)
}

func (m *MockSecretRepository) GetActiveSecrets(ctx context.Context, tenantID uuid.UUID, secretID string) ([]*repository.SecretVersion, error) {
	args := m.Called(ctx, tenantID, secretID)
	return args.Get(0).([]*repository.SecretVersion), args.Error(1)
}

func (m *MockSecretRepository) GetSecretByVersion(ctx context.Context, tenantID uuid.UUID, secretID string, version int) (*repository.SecretVersion, error) {
	args := m.Called(ctx, tenantID, secretID, version)
	return args.Get(0).(*repository.SecretVersion), args.Error(1)
}

func (m *MockSecretRepository) UpdateSecretStatus(ctx context.Context, id uuid.UUID, isActive bool) error {
	args := m.Called(ctx, id, isActive)
	return args.Error(0)
}

func (m *MockSecretRepository) DeleteExpiredSecrets(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func setupSecretManager(t *testing.T) (*SecretManager, *MockSecretRepository) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	encryption, err := NewEncryptionService(key)
	require.NoError(t, err)

	mockRepo := &MockSecretRepository{}
	manager := NewSecretManager(encryption, mockRepo)

	return manager, mockRepo
}

func TestGenerateSecret(t *testing.T) {
	manager, mockRepo := setupSecretManager(t)
	ctx := context.Background()
	tenantID := uuid.New()
	secretID := "webhook-secret"

	// Mock repository calls
	mockRepo.On("GetActiveSecrets", ctx, tenantID, secretID).Return([]*repository.SecretVersion{}, nil)
	mockRepo.On("CreateSecret", ctx, mock.Anything).Return(nil)

	secret, secretValue, err := manager.GenerateSecret(ctx, tenantID, secretID, nil)

	require.NoError(t, err)
	assert.NotNil(t, secret)
	assert.NotEmpty(t, secretValue)
	assert.Equal(t, tenantID, secret.TenantID)
	assert.Equal(t, secretID, secret.SecretID)
	assert.Equal(t, 1, secret.Version)
	assert.True(t, secret.IsActive)
	assert.Len(t, secretValue, 64) // 32 bytes = 64 hex characters

	mockRepo.AssertExpectations(t)
}

func TestGenerateSecretWithExistingVersions(t *testing.T) {
	manager, mockRepo := setupSecretManager(t)
	ctx := context.Background()
	tenantID := uuid.New()
	secretID := "webhook-secret"

	// Mock existing secrets
	existingSecrets := []*repository.SecretVersion{
		{Version: 1, IsActive: true},
		{Version: 2, IsActive: true},
	}

	mockRepo.On("GetActiveSecrets", ctx, tenantID, secretID).Return(existingSecrets, nil)
	mockRepo.On("CreateSecret", ctx, mock.Anything).Return(nil)

	secret, _, err := manager.GenerateSecret(ctx, tenantID, secretID, nil)

	require.NoError(t, err)
	assert.Equal(t, 3, secret.Version) // Should be next version

	mockRepo.AssertExpectations(t)
}

func TestGetActiveSecrets(t *testing.T) {
	manager, mockRepo := setupSecretManager(t)
	ctx := context.Background()
	tenantID := uuid.New()
	secretID := "webhook-secret"

	// Create encrypted secret values
	secret1Value := "secret1"
	secret2Value := "secret2"
	encrypted1, err := manager.encryption.EncryptString(secret1Value)
	require.NoError(t, err)
	encrypted2, err := manager.encryption.EncryptString(secret2Value)
	require.NoError(t, err)

	// Mock active secrets
	activeSecrets := []*repository.SecretVersion{
		{
			ID:       uuid.New(),
			Value:    encrypted1,
			IsActive: true,
		},
		{
			ID:       uuid.New(),
			Value:    encrypted2,
			IsActive: true,
		},
	}

	mockRepo.On("GetActiveSecrets", ctx, tenantID, secretID).Return(activeSecrets, nil)

	secrets, err := manager.GetActiveSecrets(ctx, tenantID, secretID)

	require.NoError(t, err)
	assert.Len(t, secrets, 2)
	assert.Contains(t, secrets, secret1Value)
	assert.Contains(t, secrets, secret2Value)

	mockRepo.AssertExpectations(t)
}

func TestGetActiveSecretsWithExpired(t *testing.T) {
	manager, mockRepo := setupSecretManager(t)
	ctx := context.Background()
	tenantID := uuid.New()
	secretID := "webhook-secret"

	// Create encrypted secret values
	activeSecretValue := "active-secret"
	expiredSecretValue := "expired-secret"
	encryptedActive, err := manager.encryption.EncryptString(activeSecretValue)
	require.NoError(t, err)
	encryptedExpired, err := manager.encryption.EncryptString(expiredSecretValue)
	require.NoError(t, err)

	expiredTime := time.Now().Add(-1 * time.Hour)

	// Mock secrets with one expired
	activeSecrets := []*repository.SecretVersion{
		{
			ID:        uuid.New(),
			Value:     encryptedActive,
			IsActive:  true,
			ExpiresAt: nil,
		},
		{
			ID:        uuid.New(),
			Value:     encryptedExpired,
			IsActive:  true,
			ExpiresAt: &expiredTime,
		},
	}

	mockRepo.On("GetActiveSecrets", ctx, tenantID, secretID).Return(activeSecrets, nil)

	secrets, err := manager.GetActiveSecrets(ctx, tenantID, secretID)

	require.NoError(t, err)
	assert.Len(t, secrets, 1)
	assert.Contains(t, secrets, activeSecretValue)
	assert.NotContains(t, secrets, expiredSecretValue)

	mockRepo.AssertExpectations(t)
}

func TestRotateSecret(t *testing.T) {
	manager, mockRepo := setupSecretManager(t)
	ctx := context.Background()
	tenantID := uuid.New()
	secretID := "webhook-secret"
	gracePeriod := 24 * time.Hour

	// Mock existing secrets
	existingSecrets := []*repository.SecretVersion{
		{ID: uuid.New(), Version: 1, IsActive: true, ExpiresAt: nil},
	}

	mockRepo.On("GetActiveSecrets", ctx, tenantID, secretID).Return(existingSecrets, nil).Twice()
	mockRepo.On("CreateSecret", ctx, mock.Anything).Return(nil)

	newSecret, secretValue, err := manager.RotateSecret(ctx, tenantID, secretID, gracePeriod)

	require.NoError(t, err)
	assert.NotNil(t, newSecret)
	assert.NotEmpty(t, secretValue)
	assert.Equal(t, 2, newSecret.Version)
	assert.NotNil(t, newSecret.ExpiresAt)
	assert.True(t, newSecret.ExpiresAt.After(time.Now()))

	mockRepo.AssertExpectations(t)
}

func TestValidateSecret(t *testing.T) {
	manager, mockRepo := setupSecretManager(t)
	ctx := context.Background()
	tenantID := uuid.New()
	secretID := "webhook-secret"

	validSecret := "valid-secret"
	invalidSecret := "invalid-secret"

	// Create encrypted secret value
	encrypted, err := manager.encryption.EncryptString(validSecret)
	require.NoError(t, err)

	activeSecrets := []*repository.SecretVersion{
		{
			ID:       uuid.New(),
			Value:    encrypted,
			IsActive: true,
		},
	}

	mockRepo.On("GetActiveSecrets", ctx, tenantID, secretID).Return(activeSecrets, nil).Twice()

	// Test valid secret
	isValid, err := manager.ValidateSecret(ctx, tenantID, secretID, validSecret)
	require.NoError(t, err)
	assert.True(t, isValid)

	// Test invalid secret
	isValid, err = manager.ValidateSecret(ctx, tenantID, secretID, invalidSecret)
	require.NoError(t, err)
	assert.False(t, isValid)

	mockRepo.AssertExpectations(t)
}

func TestDeactivateSecret(t *testing.T) {
	manager, mockRepo := setupSecretManager(t)
	ctx := context.Background()
	secretID := uuid.New()

	mockRepo.On("UpdateSecretStatus", ctx, secretID, false).Return(nil)

	err := manager.DeactivateSecret(ctx, secretID)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestCleanupExpiredSecrets(t *testing.T) {
	manager, mockRepo := setupSecretManager(t)
	ctx := context.Background()

	mockRepo.On("DeleteExpiredSecrets", ctx).Return(nil)

	err := manager.CleanupExpiredSecrets(ctx)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}