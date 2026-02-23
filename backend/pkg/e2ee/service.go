package e2ee

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
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
}

// NewService creates a new E2EE service.
func NewService(repo Repository, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}
	return &Service{repo: repo, config: config}
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
	_ = s.repo.AppendAuditEntry(entry)
}
