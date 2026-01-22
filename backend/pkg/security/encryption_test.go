package security

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEncryptionService(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		keyLen  int
		wantErr bool
	}{
		{
			name:    "valid 32-byte key",
			keyLen:  32,
			wantErr: false,
		},
		{
			name:    "invalid 16-byte key",
			keyLen:  16,
			wantErr: true,
		},
		{
			name:    "invalid 64-byte key",
			keyLen:  64,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keyLen)
			_, err := rand.Read(key)
			require.NoError(t, err)

			service, err := NewEncryptionService(key)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, service)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)
			}
		})
	}
}

func TestEncryptDecrypt(t *testing.T) {
	t.Parallel()
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	service, err := NewEncryptionService(key)
	require.NoError(t, err)

	tests := []struct {
		name      string
		plaintext string
	}{
		{
			name:      "simple string",
			plaintext: "hello world",
		},
		{
			name:      "empty string",
			plaintext: "",
		},
		{
			name:      "json payload",
			plaintext: `{"event": "user.created", "data": {"id": 123, "email": "test@example.com"}}`,
		},
		{
			name:      "large payload",
			plaintext: string(make([]byte, 10000)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test string encryption/decryption
			encrypted, err := service.EncryptString(tt.plaintext)
			require.NoError(t, err)
			assert.NotEmpty(t, encrypted)
			assert.NotEqual(t, tt.plaintext, encrypted)

			decrypted, err := service.DecryptString(encrypted)
			require.NoError(t, err)
			assert.Equal(t, tt.plaintext, decrypted)

			// Test byte encryption/decryption
			encryptedBytes, err := service.Encrypt([]byte(tt.plaintext))
			require.NoError(t, err)
			assert.NotEmpty(t, encryptedBytes)

			decryptedBytes, err := service.Decrypt(encryptedBytes)
			require.NoError(t, err)
			if len(tt.plaintext) == 0 {
				assert.Empty(t, decryptedBytes)
			} else {
				assert.Equal(t, []byte(tt.plaintext), decryptedBytes)
			}
		})
	}
}

func TestEncryptionDeterminism(t *testing.T) {
	t.Parallel()
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	service, err := NewEncryptionService(key)
	require.NoError(t, err)

	plaintext := "test data"

	// Encrypt the same data multiple times
	encrypted1, err := service.EncryptString(plaintext)
	require.NoError(t, err)

	encrypted2, err := service.EncryptString(plaintext)
	require.NoError(t, err)

	// Results should be different due to random nonces
	assert.NotEqual(t, encrypted1, encrypted2)

	// But both should decrypt to the same plaintext
	decrypted1, err := service.DecryptString(encrypted1)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted1)

	decrypted2, err := service.DecryptString(encrypted2)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted2)
}

func TestDecryptInvalidData(t *testing.T) {
	t.Parallel()
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	service, err := NewEncryptionService(key)
	require.NoError(t, err)

	tests := []struct {
		name        string
		ciphertext  string
		expectError bool
	}{
		{
			name:        "invalid base64",
			ciphertext:  "invalid-base64!",
			expectError: true,
		},
		{
			name:        "too short ciphertext",
			ciphertext:  "dGVzdA==", // "test" in base64, too short for nonce
			expectError: true,
		},
		{
			name:        "corrupted ciphertext",
			ciphertext:  "YWJjZGVmZ2hpams=", // valid base64 but invalid ciphertext
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.DecryptString(tt.ciphertext)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}