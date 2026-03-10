package e2ee

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateKeyPair(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	kp, err := svc.GenerateKeyPair("tenant-1", "ep-1")
	require.NoError(t, err)
	assert.NotEmpty(t, kp.ID)
	assert.NotEmpty(t, kp.PublicKey)
	assert.Equal(t, "x25519", kp.Algorithm)
	assert.Equal(t, "active", kp.Status)
	assert.Equal(t, 1, kp.Version)
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	_, err := svc.GenerateKeyPair("tenant-1", "ep-1")
	require.NoError(t, err)

	plaintext := []byte(`{"event": "order.created", "data": {"id": 123}}`)

	encrypted, err := svc.Encrypt("ep-1", plaintext)
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted.CiphertextBase64)
	assert.NotEmpty(t, encrypted.NonceBase64)
	assert.NotEmpty(t, encrypted.EphemeralPubKey)

	decrypted, err := svc.Decrypt("ep-1", encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted.Plaintext)
	assert.True(t, decrypted.Verified)
}

func TestKeyRotation(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	_, err := svc.GenerateKeyPair("tenant-1", "ep-1")
	require.NoError(t, err)

	result, err := svc.RotateKey("tenant-1", &KeyRotationRequest{EndpointID: "ep-1"})
	require.NoError(t, err)
	assert.Equal(t, 1, result.OldKeyVersion)
	assert.Equal(t, 2, result.NewKeyVersion)
	assert.NotEmpty(t, result.NewPublicKey)

	// Old key should still decrypt (grace period)
	plaintext := []byte("test message")
	encrypted, _ := svc.Encrypt("ep-1", plaintext)
	assert.Equal(t, 2, encrypted.KeyVersion)

	decrypted, err := svc.Decrypt("ep-1", encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted.Plaintext)
}

func TestDecryptWithOldKey(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	_, _ = svc.GenerateKeyPair("tenant-1", "ep-1")

	// Encrypt with v1
	plaintext := []byte("secret data")
	encrypted, _ := svc.Encrypt("ep-1", plaintext)
	assert.Equal(t, 1, encrypted.KeyVersion)

	// Rotate to v2
	svc.RotateKey("tenant-1", &KeyRotationRequest{EndpointID: "ep-1"})

	// Should still decrypt v1 message
	decrypted, err := svc.Decrypt("ep-1", encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted.Plaintext)
}

func TestHealthCheck(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	// No key = critical
	hc, err := svc.CheckHealth("ep-1")
	require.NoError(t, err)
	assert.Equal(t, "critical", hc.Status)

	// Generate key = healthy
	svc.GenerateKeyPair("tenant-1", "ep-1")
	hc, err = svc.CheckHealth("ep-1")
	require.NoError(t, err)
	assert.Equal(t, "healthy", hc.Status)
	assert.True(t, hc.EncryptionTest)
}

func TestAuditLog(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	svc.GenerateKeyPair("tenant-1", "ep-1")
	plaintext := []byte("test")
	encrypted, _ := svc.Encrypt("ep-1", plaintext)
	svc.Decrypt("ep-1", encrypted)

	entries, err := svc.GetAuditLog("ep-1", 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(entries), 3) // generated, encrypted, decrypted
}

func TestValidation(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	_, err := svc.GenerateKeyPair("", "ep-1")
	assert.Error(t, err)

	_, err = svc.GenerateKeyPair("tenant-1", "")
	assert.Error(t, err)
}

func TestEnvelopeEncryptDecrypt(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	_, err := svc.GenerateKeyPair("tenant-1", "ep-1")
	require.NoError(t, err)

	plaintext := []byte(`{"event": "order.created", "amount": 99.99}`)

	encrypted, err := svc.EnvelopeEncrypt("ep-1", plaintext)
	require.NoError(t, err)
	assert.Equal(t, "x25519-aes256gcm", encrypted.Algorithm)
	assert.NotEmpty(t, encrypted.EncryptedDEK)
	assert.NotEmpty(t, encrypted.Ciphertext)
	assert.NotEmpty(t, encrypted.PayloadNonce)
	assert.NotEmpty(t, encrypted.DEKNonce)

	decrypted, err := svc.EnvelopeDecrypt("ep-1", encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted.Plaintext)
	assert.True(t, decrypted.Verified)
}

func TestEnvelopeEncryptWithRotatedKey(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	svc.GenerateKeyPair("tenant-1", "ep-1")

	// Encrypt with v1
	plaintext := []byte("secret data v1")
	encV1, err := svc.EnvelopeEncrypt("ep-1", plaintext)
	require.NoError(t, err)
	assert.Equal(t, 1, encV1.KeyVersion)

	// Rotate key
	svc.RotateKey("tenant-1", &KeyRotationRequest{EndpointID: "ep-1"})

	// Decrypt v1 message with v1 key
	decrypted, err := svc.EnvelopeDecrypt("ep-1", encV1)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted.Plaintext)

	// New encryption uses v2
	encV2, err := svc.EnvelopeEncrypt("ep-1", []byte("new data"))
	require.NoError(t, err)
	assert.Equal(t, 2, encV2.KeyVersion)
}

func TestBatchEncrypt(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	svc.GenerateKeyPair("tenant-1", "ep-1")
	svc.GenerateKeyPair("tenant-1", "ep-2")

	result, err := svc.BatchEncrypt(&BatchEncryptRequest{
		EndpointIDs: []string{"ep-1", "ep-2", "ep-nonexistent"},
		Plaintext:   []byte("batch test"),
	})
	require.NoError(t, err)
	assert.Len(t, result.Results, 2)
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors, "ep-nonexistent")
}

func TestRevokeKey(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	svc.GenerateKeyPair("tenant-1", "ep-1")

	err := svc.RevokeKey("tenant-1", "ep-1", 1)
	require.NoError(t, err)

	// Should fail to encrypt since no active key
	_, err = svc.Encrypt("ep-1", []byte("test"))
	assert.Error(t, err)
}

func TestGetKeyMetrics(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	svc.GenerateKeyPair("tenant-1", "ep-1")
	svc.Encrypt("ep-1", []byte("test"))

	metrics, err := svc.GetKeyMetrics("tenant-1")
	require.NoError(t, err)
	assert.Equal(t, "tenant-1", metrics.TenantID)
	assert.Equal(t, int64(1), metrics.EncryptionOps)
}
