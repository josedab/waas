package security

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPayloadEncryptor_Roundtrip(t *testing.T) {
	t.Parallel()
	key := "01234567890123456789012345678901" // 32 bytes
	enc, err := NewPayloadEncryptor(key)
	require.NoError(t, err)

	plaintext := []byte(`{"event":"order.created","data":{"id":123,"amount":99.99}}`)
	ciphertext, err := enc.Encrypt(plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, string(plaintext), ciphertext)

	decrypted, err := enc.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestPayloadEncryptor_EmptyPayload(t *testing.T) {
	t.Parallel()
	key := "01234567890123456789012345678901"
	enc, err := NewPayloadEncryptor(key)
	require.NoError(t, err)

	ciphertext, err := enc.Encrypt([]byte{})
	require.NoError(t, err)
	assert.NotEmpty(t, ciphertext)

	decrypted, err := enc.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Empty(t, decrypted)
}

func TestPayloadEncryptor_LargePayload(t *testing.T) {
	t.Parallel()
	key := "01234567890123456789012345678901"
	enc, err := NewPayloadEncryptor(key)
	require.NoError(t, err)

	// 1MB payload
	large := []byte(strings.Repeat("A", 1024*1024))
	ciphertext, err := enc.Encrypt(large)
	require.NoError(t, err)

	decrypted, err := enc.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, large, decrypted)
}

func TestPayloadEncryptor_InvalidKeySize_Short(t *testing.T) {
	t.Parallel()
	_, err := NewPayloadEncryptor("too-short")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "32 bytes")
}

func TestPayloadEncryptor_InvalidKeySize_Long(t *testing.T) {
	t.Parallel()
	_, err := NewPayloadEncryptor(strings.Repeat("A", 64))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "32 bytes")
}

func TestPayloadEncryptor_InvalidKeySize_Empty(t *testing.T) {
	t.Parallel()
	_, err := NewPayloadEncryptor("")
	assert.Error(t, err)
}

func TestPayloadEncryptor_DecryptInvalidBase64(t *testing.T) {
	t.Parallel()
	key := "01234567890123456789012345678901"
	enc, err := NewPayloadEncryptor(key)
	require.NoError(t, err)

	_, err = enc.Decrypt("not-valid-base64!!!")
	assert.Error(t, err)
}

func TestPayloadEncryptor_DecryptTamperedCiphertext(t *testing.T) {
	t.Parallel()
	key := "01234567890123456789012345678901"
	enc, err := NewPayloadEncryptor(key)
	require.NoError(t, err)

	ciphertext, err := enc.Encrypt([]byte("secret data"))
	require.NoError(t, err)

	// Tamper with the ciphertext
	tampered := ciphertext[:len(ciphertext)-2] + "XX"
	_, err = enc.Decrypt(tampered)
	assert.Error(t, err)
}

func TestPayloadEncryptor_UniqueNonces(t *testing.T) {
	t.Parallel()
	key := "01234567890123456789012345678901"
	enc, err := NewPayloadEncryptor(key)
	require.NoError(t, err)

	plaintext := []byte("same plaintext")
	ct1, err := enc.Encrypt(plaintext)
	require.NoError(t, err)
	ct2, err := enc.Encrypt(plaintext)
	require.NoError(t, err)

	assert.NotEqual(t, ct1, ct2, "encrypting same plaintext twice should produce different ciphertexts")
}

func TestGenerateEncryptionKey(t *testing.T) {
	t.Parallel()
	key, err := GenerateEncryptionKey()
	require.NoError(t, err)
	assert.NotEmpty(t, key)

	// Keys should be unique
	key2, err := GenerateEncryptionKey()
	require.NoError(t, err)
	assert.NotEqual(t, key, key2)
}
