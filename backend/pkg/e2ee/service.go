package e2ee

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/utils"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/nacl/box"
)

// ServiceConfig configures the E2EE service.
type ServiceConfig struct {
	KeyRotationInterval time.Duration
	MaxKeysPerEndpoint  int
	GracePeriodDefault  time.Duration
}

// DefaultServiceConfig returns sensible defaults.
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		KeyRotationInterval: 90 * 24 * time.Hour, // 90 days
		MaxKeysPerEndpoint:  10,
		GracePeriodDefault:  24 * time.Hour,
	}
}

// Service implements the E2EE business logic.
type Service struct {
	repo   Repository
	config *ServiceConfig
	logger *utils.Logger
}

// NewService creates a new E2EE service.
func NewService(repo Repository, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}
	return &Service{repo: repo, config: config, logger: utils.NewLogger("e2ee-service")}
}

// GenerateKeyPair creates a new X25519 key pair for an endpoint.
func (s *Service) GenerateKeyPair(tenantID, endpointID string) (*KeyPair, error) {
	if tenantID == "" || endpointID == "" {
		return nil, fmt.Errorf("tenant_id and endpoint_id are required")
	}

	// Determine version
	existing, _ := s.repo.ListKeyPairs(endpointID)
	version := 1
	for _, kp := range existing {
		if kp.Version >= version {
			version = kp.Version + 1
		}
	}

	if len(existing) >= s.config.MaxKeysPerEndpoint {
		return nil, fmt.Errorf("maximum %d keys per endpoint", s.config.MaxKeysPerEndpoint)
	}

	// Generate X25519 key pair
	pubKey, privKey, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	kp := &KeyPair{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		EndpointID: endpointID,
		PublicKey:  base64.StdEncoding.EncodeToString(pubKey[:]),
		PrivateKey: base64.StdEncoding.EncodeToString(privKey[:]),
		Algorithm:  "x25519",
		Status:     "active",
		Version:    version,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(s.config.KeyRotationInterval),
	}

	if err := s.repo.StoreKeyPair(kp); err != nil {
		return nil, fmt.Errorf("failed to store key pair: %w", err)
	}

	s.audit(tenantID, endpointID, "key_generated", version, true, "")
	return kp, nil
}

// GetPublicKey returns the active public key for an endpoint (used by senders).
func (s *Service) GetPublicKey(endpointID string) (string, int, error) {
	kp, err := s.repo.GetActiveKeyPair(endpointID)
	if err != nil {
		return "", 0, err
	}
	return kp.PublicKey, kp.Version, nil
}

// Encrypt encrypts a payload using the receiver's public key.
func (s *Service) Encrypt(endpointID string, plaintext []byte) (*EncryptedPayload, error) {
	kp, err := s.repo.GetActiveKeyPair(endpointID)
	if err != nil {
		return nil, fmt.Errorf("no active key for endpoint: %w", err)
	}

	// Decode receiver's public key
	receiverPubBytes, err := base64.StdEncoding.DecodeString(kp.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("invalid public key: %w", err)
	}
	var receiverPub [32]byte
	copy(receiverPub[:], receiverPubBytes)

	// Generate ephemeral key pair for this message
	ephPub, ephPriv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ephemeral key: %w", err)
	}

	// Compute shared secret
	var sharedKey [32]byte
	curve25519.ScalarMult(&sharedKey, ephPriv, &receiverPub)

	// Generate nonce
	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt using NaCl box (XSalsa20-Poly1305)
	ciphertext := box.SealAfterPrecomputation(nil, plaintext, &nonce, &sharedKey)

	s.audit(kp.TenantID, endpointID, "payload_encrypted", kp.Version, true, "")

	return &EncryptedPayload{
		CiphertextBase64: base64.StdEncoding.EncodeToString(ciphertext),
		NonceBase64:      base64.StdEncoding.EncodeToString(nonce[:]),
		EphemeralPubKey:  base64.StdEncoding.EncodeToString(ephPub[:]),
		KeyVersion:       kp.Version,
		Algorithm:        "x25519-xsalsa20-poly1305",
	}, nil
}

// Decrypt decrypts a payload using the receiver's private key.
func (s *Service) Decrypt(endpointID string, encrypted *EncryptedPayload) (*DecryptedPayload, error) {
	kp, err := s.repo.GetKeyPairByVersion(endpointID, encrypted.KeyVersion)
	if err != nil {
		return nil, fmt.Errorf("key not found for version %d: %w", encrypted.KeyVersion, err)
	}

	// Decode private key
	privBytes, err := base64.StdEncoding.DecodeString(kp.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	var privKey [32]byte
	copy(privKey[:], privBytes)

	// Decode ephemeral public key
	ephPubBytes, err := base64.StdEncoding.DecodeString(encrypted.EphemeralPubKey)
	if err != nil {
		return nil, fmt.Errorf("invalid ephemeral public key: %w", err)
	}
	var ephPub [32]byte
	copy(ephPub[:], ephPubBytes)

	// Compute shared secret
	var sharedKey [32]byte
	curve25519.ScalarMult(&sharedKey, &privKey, &ephPub)

	// Decode nonce
	nonceBytes, err := base64.StdEncoding.DecodeString(encrypted.NonceBase64)
	if err != nil {
		return nil, fmt.Errorf("invalid nonce: %w", err)
	}
	var nonce [24]byte
	copy(nonce[:], nonceBytes)

	// Decode ciphertext
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted.CiphertextBase64)
	if err != nil {
		return nil, fmt.Errorf("invalid ciphertext: %w", err)
	}

	// Decrypt
	plaintext, ok := box.OpenAfterPrecomputation(nil, ciphertext, &nonce, &sharedKey)
	if !ok {
		s.audit(kp.TenantID, endpointID, "payload_decrypted", kp.Version, false, "decryption failed")
		return nil, fmt.Errorf("decryption failed: authentication error")
	}

	s.audit(kp.TenantID, endpointID, "payload_decrypted", kp.Version, true, "")

	return &DecryptedPayload{
		Plaintext:  plaintext,
		KeyVersion: kp.Version,
		Verified:   true,
	}, nil
}

// RotateKey generates a new key pair and marks the old one as rotated.
func (s *Service) RotateKey(tenantID string, req *KeyRotationRequest) (*KeyRotationResult, error) {
	if req.EndpointID == "" {
		return nil, fmt.Errorf("endpoint_id is required")
	}

	oldKP, err := s.repo.GetActiveKeyPair(req.EndpointID)
	if err != nil {
		return nil, fmt.Errorf("no active key to rotate: %w", err)
	}

	// Generate new key pair
	newKP, err := s.GenerateKeyPair(tenantID, req.EndpointID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new key: %w", err)
	}

	// Mark old key as rotated
	now := time.Now()
	oldKP.RotatedAt = &now
	if err := s.repo.UpdateKeyPairStatus(oldKP.ID, "rotated"); err != nil {
		return nil, fmt.Errorf("failed to update old key status: %w", err)
	}

	gracePeriod := s.config.GracePeriodDefault
	if req.GracePeriodMins > 0 {
		gracePeriod = time.Duration(req.GracePeriodMins) * time.Minute
	}

	s.audit(tenantID, req.EndpointID, "key_rotated", newKP.Version, true,
		fmt.Sprintf("rotated from v%d to v%d", oldKP.Version, newKP.Version))

	return &KeyRotationResult{
		OldKeyVersion: oldKP.Version,
		NewKeyVersion: newKP.Version,
		NewPublicKey:  newKP.PublicKey,
		GracePeriod:   gracePeriod.String(),
		RotatedAt:     now,
	}, nil
}

// CheckHealth verifies encryption health for an endpoint.
func (s *Service) CheckHealth(endpointID string) (*HealthCheck, error) {
	hc := &HealthCheck{EndpointID: endpointID}

	kp, err := s.repo.GetActiveKeyPair(endpointID)
	if err != nil {
		hc.Status = "critical"
		return hc, nil
	}

	hc.HasActiveKey = true
	hc.KeyVersion = kp.Version
	hc.KeyAge = time.Since(kp.CreatedAt).Round(time.Hour).String()
	hc.NeedsRotation = time.Now().After(kp.ExpiresAt)

	// Round-trip encryption test
	testData := []byte("e2ee-health-check")
	encrypted, err := s.Encrypt(endpointID, testData)
	if err != nil {
		hc.Status = "critical"
		return hc, nil
	}
	decrypted, err := s.Decrypt(endpointID, encrypted)
	if err != nil || string(decrypted.Plaintext) != string(testData) {
		hc.Status = "critical"
		return hc, nil
	}
	hc.EncryptionTest = true

	if hc.NeedsRotation {
		hc.Status = "warning"
	} else {
		hc.Status = "healthy"
	}
	return hc, nil
}

// GetAuditLog returns the audit log for an endpoint.
func (s *Service) GetAuditLog(endpointID string, limit int) ([]*AuditEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListAuditEntries(endpointID, limit)
}

func (s *Service) audit(tenantID, endpointID, operation string, version int, success bool, details string) {
	entry := &AuditEntry{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		EndpointID: endpointID,
		Operation:  operation,
		KeyVersion: version,
		Success:    success,
		Details:    details,
		Timestamp:  time.Now(),
	}
	if err := s.repo.AppendAuditEntry(entry); err != nil {
		s.logger.Error("failed to append audit entry", map[string]interface{}{"error": err.Error(), "tenant_id": tenantID, "endpoint_id": endpointID})
	}
}

// EnvelopeEncrypt performs AES-256-GCM envelope encryption using X25519 key exchange.
// A random Data Encryption Key (DEK) encrypts the payload; the DEK is then encrypted
// with the shared secret derived from X25519 key exchange.
func (s *Service) EnvelopeEncrypt(endpointID string, plaintext []byte) (*EnvelopeEncryptedPayload, error) {
	kp, err := s.repo.GetActiveKeyPair(endpointID)
	if err != nil {
		return nil, fmt.Errorf("no active key for endpoint: %w", err)
	}

	receiverPubBytes, err := base64.StdEncoding.DecodeString(kp.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("invalid public key: %w", err)
	}
	var receiverPub [32]byte
	copy(receiverPub[:], receiverPubBytes)

	// Generate ephemeral X25519 key pair
	ephPub, ephPriv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ephemeral key: %w", err)
	}

	// Derive shared secret via X25519
	var sharedSecret [32]byte
	curve25519.ScalarMult(&sharedSecret, ephPriv, &receiverPub)

	// Generate random DEK (32 bytes for AES-256)
	dek := make([]byte, 32)
	if _, err := rand.Read(dek); err != nil {
		return nil, fmt.Errorf("failed to generate DEK: %w", err)
	}

	// Encrypt payload with DEK using AES-256-GCM
	payloadBlock, err := aes.NewCipher(dek)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}
	payloadGCM, err := cipher.NewGCM(payloadBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	payloadNonce := make([]byte, payloadGCM.NonceSize())
	if _, err := rand.Read(payloadNonce); err != nil {
		return nil, fmt.Errorf("failed to generate payload nonce: %w", err)
	}
	ciphertext := payloadGCM.Seal(nil, payloadNonce, plaintext, nil)

	// Encrypt DEK with shared secret using AES-256-GCM
	dekBlock, err := aes.NewCipher(sharedSecret[:])
	if err != nil {
		return nil, fmt.Errorf("failed to create DEK cipher: %w", err)
	}
	dekGCM, err := cipher.NewGCM(dekBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to create DEK GCM: %w", err)
	}
	dekNonce := make([]byte, dekGCM.NonceSize())
	if _, err := rand.Read(dekNonce); err != nil {
		return nil, fmt.Errorf("failed to generate DEK nonce: %w", err)
	}
	encryptedDEK := dekGCM.Seal(nil, dekNonce, dek, nil)

	s.audit(kp.TenantID, endpointID, "payload_encrypted", kp.Version, true, "aes256gcm-envelope")

	return &EnvelopeEncryptedPayload{
		EncryptedDEK:    base64.StdEncoding.EncodeToString(encryptedDEK),
		DEKNonce:        base64.StdEncoding.EncodeToString(dekNonce),
		Ciphertext:      base64.StdEncoding.EncodeToString(ciphertext),
		PayloadNonce:    base64.StdEncoding.EncodeToString(payloadNonce),
		EphemeralPubKey: base64.StdEncoding.EncodeToString(ephPub[:]),
		KeyVersion:      kp.Version,
		Algorithm:       "x25519-aes256gcm",
	}, nil
}

// EnvelopeDecrypt decrypts an AES-256-GCM envelope-encrypted payload.
func (s *Service) EnvelopeDecrypt(endpointID string, encrypted *EnvelopeEncryptedPayload) (*DecryptedPayload, error) {
	kp, err := s.repo.GetKeyPairByVersion(endpointID, encrypted.KeyVersion)
	if err != nil {
		return nil, fmt.Errorf("key not found for version %d: %w", encrypted.KeyVersion, err)
	}

	privBytes, err := base64.StdEncoding.DecodeString(kp.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	var privKey [32]byte
	copy(privKey[:], privBytes)

	ephPubBytes, err := base64.StdEncoding.DecodeString(encrypted.EphemeralPubKey)
	if err != nil {
		return nil, fmt.Errorf("invalid ephemeral public key: %w", err)
	}
	var ephPub [32]byte
	copy(ephPub[:], ephPubBytes)

	// Derive shared secret
	var sharedSecret [32]byte
	curve25519.ScalarMult(&sharedSecret, &privKey, &ephPub)

	// Decrypt DEK
	encDEK, err := base64.StdEncoding.DecodeString(encrypted.EncryptedDEK)
	if err != nil {
		return nil, fmt.Errorf("invalid encrypted DEK: %w", err)
	}
	dekNonce, err := base64.StdEncoding.DecodeString(encrypted.DEKNonce)
	if err != nil {
		return nil, fmt.Errorf("invalid DEK nonce: %w", err)
	}
	dekBlock, err := aes.NewCipher(sharedSecret[:])
	if err != nil {
		return nil, fmt.Errorf("failed to create DEK cipher: %w", err)
	}
	dekGCM, err := cipher.NewGCM(dekBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to create DEK GCM: %w", err)
	}
	dek, err := dekGCM.Open(nil, dekNonce, encDEK, nil)
	if err != nil {
		s.audit(kp.TenantID, endpointID, "payload_decrypted", kp.Version, false, "DEK decryption failed")
		return nil, fmt.Errorf("DEK decryption failed: %w", err)
	}

	// Decrypt payload
	ciphertextBytes, err := base64.StdEncoding.DecodeString(encrypted.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("invalid ciphertext: %w", err)
	}
	payloadNonce, err := base64.StdEncoding.DecodeString(encrypted.PayloadNonce)
	if err != nil {
		return nil, fmt.Errorf("invalid payload nonce: %w", err)
	}
	payloadBlock, err := aes.NewCipher(dek)
	if err != nil {
		return nil, fmt.Errorf("failed to create payload cipher: %w", err)
	}
	payloadGCM, err := cipher.NewGCM(payloadBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to create payload GCM: %w", err)
	}
	plaintext, err := payloadGCM.Open(nil, payloadNonce, ciphertextBytes, nil)
	if err != nil {
		s.audit(kp.TenantID, endpointID, "payload_decrypted", kp.Version, false, "payload decryption failed")
		return nil, fmt.Errorf("payload decryption failed: %w", err)
	}

	s.audit(kp.TenantID, endpointID, "payload_decrypted", kp.Version, true, "aes256gcm-envelope")

	return &DecryptedPayload{
		Plaintext:  plaintext,
		KeyVersion: kp.Version,
		Verified:   true,
	}, nil
}

// BatchEncrypt encrypts a payload for multiple endpoints simultaneously.
func (s *Service) BatchEncrypt(req *BatchEncryptRequest) (*BatchEncryptResult, error) {
	if len(req.EndpointIDs) == 0 {
		return nil, fmt.Errorf("at least one endpoint_id is required")
	}
	if len(req.Plaintext) == 0 {
		return nil, fmt.Errorf("plaintext is required")
	}

	result := &BatchEncryptResult{
		Results: make(map[string]*EncryptedPayload),
		Errors:  make(map[string]string),
	}

	for _, epID := range req.EndpointIDs {
		encrypted, err := s.Encrypt(epID, req.Plaintext)
		if err != nil {
			result.Errors[epID] = err.Error()
		} else {
			result.Results[epID] = encrypted
		}
	}

	return result, nil
}

// RevokeKey revokes a key, preventing further use.
func (s *Service) RevokeKey(tenantID, endpointID string, version int) error {
	kp, err := s.repo.GetKeyPairByVersion(endpointID, version)
	if err != nil {
		return fmt.Errorf("key not found: %w", err)
	}

	if err := s.repo.UpdateKeyPairStatus(kp.ID, "revoked"); err != nil {
		return fmt.Errorf("failed to revoke key: %w", err)
	}

	s.audit(tenantID, endpointID, "key_revoked", version, true, "")
	return nil
}

// GetKeyMetrics returns encryption metrics for a tenant.
func (s *Service) GetKeyMetrics(tenantID string) (*KeyMetrics, error) {
	metrics := &KeyMetrics{TenantID: tenantID}

	// Scan all endpoints for this tenant
	allKeys, _ := s.repo.ListKeyPairs("")
	endpoints := make(map[string]bool)
	for _, kp := range allKeys {
		if kp.TenantID != tenantID {
			continue
		}
		endpoints[kp.EndpointID] = true
		metrics.TotalKeys++
		switch kp.Status {
		case "active":
			metrics.ActiveKeys++
		case "rotated":
			metrics.RotatedKeys++
		case "revoked":
			metrics.RevokedKeys++
		}
	}

	for epID := range endpoints {
		entries, _ := s.repo.ListAuditEntries(epID, 10000)
		for _, e := range entries {
			if e.Operation == "payload_encrypted" {
				metrics.EncryptionOps++
			}
			if e.Operation == "payload_decrypted" {
				metrics.DecryptionOps++
			}
		}
	}

	return metrics, nil
}
