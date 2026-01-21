package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// EncryptionService handles data encryption at rest
type EncryptionService struct {
	gcm cipher.AEAD
}

// NewEncryptionService creates a new encryption service with the provided key
func NewEncryptionService(key []byte) (*EncryptionService, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes long")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return &EncryptionService{gcm: gcm}, nil
}

// Encrypt encrypts plaintext data and returns base64 encoded ciphertext
func (es *EncryptionService) Encrypt(plaintext []byte) (string, error) {
	nonce := make([]byte, es.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := es.gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64 encoded ciphertext and returns plaintext data
func (es *EncryptionService) Decrypt(ciphertextB64 string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	nonceSize := es.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := es.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// EncryptString encrypts a string and returns base64 encoded ciphertext
func (es *EncryptionService) EncryptString(plaintext string) (string, error) {
	return es.Encrypt([]byte(plaintext))
}

// DecryptString decrypts base64 encoded ciphertext and returns plaintext string
func (es *EncryptionService) DecryptString(ciphertextB64 string) (string, error) {
	plaintext, err := es.Decrypt(ciphertextB64)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}