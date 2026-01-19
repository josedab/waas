package signatures

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"strconv"
	"strings"
	"time"
)

// Service provides signature operations
type Service struct {
	repo   Repository
	config *ServiceConfig
}

// ServiceConfig holds service configuration
type ServiceConfig struct {
	MaxSchemesPerTenant int
	MaxKeysPerScheme    int
	DefaultKeyLength    int
	RotationCheckInterval time.Duration
}

// DefaultServiceConfig returns default configuration
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		MaxSchemesPerTenant:   20,
		MaxKeysPerScheme:      10,
		DefaultKeyLength:      32,
		RotationCheckInterval: time.Hour,
	}
}

// NewService creates a new signature service
func NewService(repo Repository, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}

	return &Service{
		repo:   repo,
		config: config,
	}
}

// CreateScheme creates a new signature scheme
func (s *Service) CreateScheme(ctx context.Context, tenantID string, req *CreateSchemeRequest) (*SignatureScheme, error) {
	existing, err := s.repo.ListSchemes(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if len(existing) >= s.config.MaxSchemesPerTenant {
		return nil, fmt.Errorf("maximum schemes reached: %d", s.config.MaxSchemesPerTenant)
	}

	scheme := &SignatureScheme{
		ID:          GenerateSchemeID(),
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Status:      SchemeActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Set algorithm
	if req.Algorithm != "" {
		scheme.Algorithm = req.Algorithm
	} else {
		scheme.Algorithm = AlgorithmHMACSHA256
	}

	// Set config
	if req.Config != nil {
		scheme.Config = *req.Config
	} else {
		scheme.Config = GetDefaultConfig(req.Type)
	}

	// Set key config
	if req.KeyConfig != nil {
		scheme.KeyConfig = *req.KeyConfig
	} else {
		scheme.KeyConfig = GetDefaultKeyConfig()
	}

	if err := s.repo.SaveScheme(ctx, scheme); err != nil {
		return nil, err
	}

	// Generate initial key
	_, err = s.generateKey(ctx, scheme)
	if err != nil {
		// Delete scheme if key generation fails
		s.repo.DeleteScheme(ctx, tenantID, scheme.ID)
		return nil, fmt.Errorf("failed to generate initial key: %w", err)
	}

	return scheme, nil
}

// GetScheme retrieves a signature scheme
func (s *Service) GetScheme(ctx context.Context, tenantID, schemeID string) (*SignatureScheme, error) {
	return s.repo.GetScheme(ctx, tenantID, schemeID)
}

// ListSchemes lists signature schemes
func (s *Service) ListSchemes(ctx context.Context, tenantID string) ([]SignatureScheme, error) {
	return s.repo.ListSchemes(ctx, tenantID)
}

// UpdateScheme updates a signature scheme
func (s *Service) UpdateScheme(ctx context.Context, tenantID, schemeID string, req *UpdateSchemeRequest) (*SignatureScheme, error) {
	scheme, err := s.repo.GetScheme(ctx, tenantID, schemeID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		scheme.Name = *req.Name
	}
	if req.Description != nil {
		scheme.Description = *req.Description
	}
	if req.Config != nil {
		scheme.Config = *req.Config
	}
	if req.KeyConfig != nil {
		scheme.KeyConfig = *req.KeyConfig
	}

	scheme.UpdatedAt = time.Now()

	if err := s.repo.SaveScheme(ctx, scheme); err != nil {
		return nil, err
	}

	return scheme, nil
}

// DeleteScheme deletes a signature scheme
func (s *Service) DeleteScheme(ctx context.Context, tenantID, schemeID string) error {
	return s.repo.DeleteScheme(ctx, tenantID, schemeID)
}

// Sign signs a payload
func (s *Service) Sign(ctx context.Context, tenantID string, req *SignatureRequest) (*SignatureResult, error) {
	scheme, err := s.repo.GetScheme(ctx, tenantID, req.SchemeID)
	if err != nil {
		return nil, err
	}

	key, err := s.repo.GetPrimaryKey(ctx, scheme.ID)
	if err != nil {
		return nil, fmt.Errorf("no signing key available: %w", err)
	}

	// Prepare timestamp
	timestamp := time.Now()
	if req.Timestamp != nil {
		timestamp = *req.Timestamp
	}

	// Prepare message ID
	messageID := req.MessageID
	if messageID == "" && scheme.Config.IncludeMessageID {
		messageID = GenerateKeyID() // Use UUID for message ID
	}

	// Build signed payload
	signedPayload := s.buildSignedPayload(scheme, req.Payload, timestamp, messageID)

	// Compute signature
	signature, err := s.computeSignature(scheme.Algorithm, key.SecretKey, signedPayload)
	if err != nil {
		return nil, err
	}

	// Format signature
	formattedSig := s.formatSignature(scheme, signature, timestamp)

	// Build headers
	headers := make(map[string]string)
	headers[scheme.Config.SignatureHeader] = formattedSig
	if scheme.Config.IncludeTimestamp && scheme.Config.TimestampHeader != "" {
		headers[scheme.Config.TimestampHeader] = strconv.FormatInt(timestamp.Unix(), 10)
	}
	if scheme.Config.IncludeMessageID && scheme.Config.IDHeader != "" {
		headers[scheme.Config.IDHeader] = messageID
	}

	// Update key usage
	s.repo.UpdateKeyUsage(ctx, key.ID)
	s.repo.IncrementSignCount(ctx, scheme.ID)

	return &SignatureResult{
		Signature:     formattedSig,
		Headers:       headers,
		KeyID:         key.ID,
		KeyVersion:    key.Version,
		Algorithm:     scheme.Algorithm,
		Timestamp:     timestamp,
		MessageID:     messageID,
		SignedPayload: string(signedPayload),
	}, nil
}

// Verify verifies a signature
func (s *Service) Verify(ctx context.Context, tenantID string, req *VerifyRequest) (*VerifyResult, error) {
	scheme, err := s.repo.GetScheme(ctx, tenantID, req.SchemeID)
	if err != nil {
		return &VerifyResult{Valid: false, Error: err.Error(), ErrorCode: "SCHEME_NOT_FOUND"}, nil
	}

	// Get timestamp from request or header
	var timestamp time.Time
	if req.Timestamp != nil {
		timestamp = *req.Timestamp
	} else if req.Headers != nil && scheme.Config.TimestampHeader != "" {
		if ts, ok := req.Headers[scheme.Config.TimestampHeader]; ok {
			tsInt, _ := strconv.ParseInt(ts, 10, 64)
			timestamp = time.Unix(tsInt, 0)
		}
	}

	// Check timestamp tolerance
	if scheme.Config.TimestampToleranceSec > 0 && !timestamp.IsZero() {
		age := time.Since(timestamp)
		if age > time.Duration(scheme.Config.TimestampToleranceSec)*time.Second {
			s.repo.IncrementVerifyCount(ctx, scheme.ID, false)
			return &VerifyResult{
				Valid:        false,
				Error:        "Timestamp too old",
				ErrorCode:    "TIMESTAMP_EXPIRED",
				TimestampAge: age,
				VerifiedAt:   time.Now(),
			}, nil
		}
		if age < -time.Duration(scheme.Config.TimestampToleranceSec)*time.Second {
			s.repo.IncrementVerifyCount(ctx, scheme.ID, false)
			return &VerifyResult{
				Valid:        false,
				Error:        "Timestamp in the future",
				ErrorCode:    "TIMESTAMP_FUTURE",
				TimestampAge: age,
				VerifiedAt:   time.Now(),
			}, nil
		}
	}

	// Get message ID
	messageID := req.MessageID
	if messageID == "" && req.Headers != nil && scheme.Config.IDHeader != "" {
		messageID = req.Headers[scheme.Config.IDHeader]
	}

	// Build signed payload
	signedPayload := s.buildSignedPayload(scheme, req.Payload, timestamp, messageID)

	// Parse the signature
	providedSig := s.parseSignature(scheme, req.Signature)

	// Get all active/rotating keys to try
	keys, err := s.repo.ListKeys(ctx, scheme.ID)
	if err != nil {
		return &VerifyResult{Valid: false, Error: "Failed to get keys", ErrorCode: "KEY_ERROR"}, nil
	}

	// Try each key
	for _, key := range keys {
		if key.Status != KeyActive && key.Status != KeyPrimary && key.Status != KeyRotating {
			continue
		}

		expectedSig, err := s.computeSignature(scheme.Algorithm, key.SecretKey, signedPayload)
		if err != nil {
			continue
		}

		if hmac.Equal([]byte(providedSig), []byte(expectedSig)) {
			s.repo.UpdateKeyUsage(ctx, key.ID)
			s.repo.IncrementVerifyCount(ctx, scheme.ID, true)
			return &VerifyResult{
				Valid:        true,
				KeyID:        key.ID,
				KeyVersion:   key.Version,
				VerifiedAt:   time.Now(),
				TimestampAge: time.Since(timestamp),
			}, nil
		}
	}

	s.repo.IncrementVerifyCount(ctx, scheme.ID, false)
	return &VerifyResult{
		Valid:      false,
		Error:      "Signature verification failed",
		ErrorCode:  "INVALID_SIGNATURE",
		VerifiedAt: time.Now(),
	}, nil
}

func (s *Service) buildSignedPayload(scheme *SignatureScheme, payload []byte, timestamp time.Time, messageID string) []byte {
	template := scheme.Config.SignedPayloadTemplate
	if template == "" {
		return payload
	}

	result := template
	result = strings.ReplaceAll(result, "{body}", string(payload))
	result = strings.ReplaceAll(result, "{timestamp}", strconv.FormatInt(timestamp.Unix(), 10))
	result = strings.ReplaceAll(result, "{id}", messageID)

	return []byte(result)
}

func (s *Service) computeSignature(algorithm SignatureAlgorithm, secret string, payload []byte) (string, error) {
	var h func() hash.Hash
	switch algorithm {
	case AlgorithmHMACSHA256:
		h = sha256.New
	case AlgorithmHMACSHA512:
		h = sha512.New
	default:
		return "", fmt.Errorf("unsupported algorithm: %s", algorithm)
	}

	mac := hmac.New(h, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func (s *Service) formatSignature(scheme *SignatureScheme, signature string, timestamp time.Time) string {
	// Convert encoding if needed
	if scheme.Config.SignatureFormat == "base64" {
		sigBytes, _ := hex.DecodeString(signature)
		signature = base64.StdEncoding.EncodeToString(sigBytes)
	}

	// Add prefix
	if scheme.Config.SignaturePrefix != "" {
		signature = scheme.Config.SignaturePrefix + signature
	}

	// For Stripe-style, include timestamp
	if scheme.Type == TypeStripe {
		return fmt.Sprintf("t=%d,%s", timestamp.Unix(), signature)
	}

	return signature
}

func (s *Service) parseSignature(scheme *SignatureScheme, signature string) string {
	// Handle Stripe-style format: t=timestamp,v1=signature
	if scheme.Type == TypeStripe {
		parts := strings.Split(signature, ",")
		for _, part := range parts {
			if strings.HasPrefix(part, "v1=") {
				signature = strings.TrimPrefix(part, "v1=")
				break
			}
		}
	}

	// Remove prefix
	if scheme.Config.SignaturePrefix != "" {
		signature = strings.TrimPrefix(signature, scheme.Config.SignaturePrefix)
	}

	// Convert from base64 if needed
	if scheme.Config.SignatureFormat == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(signature)
		if err == nil {
			signature = hex.EncodeToString(decoded)
		}
	}

	return signature
}

// RotateKey initiates key rotation
func (s *Service) RotateKey(ctx context.Context, tenantID, schemeID string, req *RotateKeyRequest) (*KeyRotation, error) {
	scheme, err := s.repo.GetScheme(ctx, tenantID, schemeID)
	if err != nil {
		return nil, err
	}

	// Get current primary key
	oldKey, err := s.repo.GetPrimaryKey(ctx, schemeID)
	if err != nil {
		return nil, fmt.Errorf("no active key to rotate: %w", err)
	}

	// Generate new key
	newKey, err := s.generateKey(ctx, scheme)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new key: %w", err)
	}

	// Create rotation record
	rotation := &KeyRotation{
		ID:          GenerateRotationID(),
		SchemeID:    schemeID,
		TenantID:    tenantID,
		OldKeyID:    oldKey.ID,
		NewKeyID:    newKey.ID,
		Status:      RotationScheduledStatus,
		Reason:      req.Reason,
		ScheduledAt: time.Now(),
	}

	if req.ScheduleAt != nil {
		rotation.ScheduledAt = *req.ScheduleAt
	}

	// Set overlap period
	overlapHours := 24
	if req.OverlapHours > 0 {
		overlapHours = req.OverlapHours
	}
	overlapUntil := rotation.ScheduledAt.Add(time.Duration(overlapHours) * time.Hour)
	rotation.OverlapUntil = &overlapUntil

	if req.Immediate {
		// Perform immediate rotation
		now := time.Now()
		rotation.StartedAt = &now
		rotation.Status = RotationInProgress

		// Update old key status
		s.repo.UpdateKeyStatus(ctx, oldKey.ID, KeyRotating)

		// Set new key as primary
		newKey.Status = KeyPrimary
		s.repo.SaveKey(ctx, newKey)

		rotation.Status = RotationCompleted
		rotation.CompletedAt = &now
	}

	if err := s.repo.SaveRotation(ctx, rotation); err != nil {
		return nil, err
	}

	return rotation, nil
}

// GetKeys lists keys for a scheme
func (s *Service) GetKeys(ctx context.Context, tenantID, schemeID string) ([]SigningKey, error) {
	// Verify scheme belongs to tenant
	_, err := s.repo.GetScheme(ctx, tenantID, schemeID)
	if err != nil {
		return nil, err
	}

	return s.repo.ListKeys(ctx, schemeID)
}

// RevokeKey revokes a key
func (s *Service) RevokeKey(ctx context.Context, tenantID, schemeID, keyID string) error {
	// Verify scheme belongs to tenant
	_, err := s.repo.GetScheme(ctx, tenantID, schemeID)
	if err != nil {
		return err
	}

	return s.repo.UpdateKeyStatus(ctx, keyID, KeyRevoked)
}

// GetRotations lists rotations for a scheme
func (s *Service) GetRotations(ctx context.Context, tenantID, schemeID string) ([]KeyRotation, error) {
	// Verify scheme belongs to tenant
	_, err := s.repo.GetScheme(ctx, tenantID, schemeID)
	if err != nil {
		return nil, err
	}

	return s.repo.ListRotations(ctx, schemeID)
}

// GetStats retrieves scheme statistics
func (s *Service) GetStats(ctx context.Context, tenantID, schemeID string) (*SchemeStats, error) {
	// Verify scheme belongs to tenant
	_, err := s.repo.GetScheme(ctx, tenantID, schemeID)
	if err != nil {
		return nil, err
	}

	return s.repo.GetSchemeStats(ctx, schemeID)
}

// GetSupportedSchemes returns supported signature schemes
func (s *Service) GetSupportedSchemes() []SignatureSchemeInfo {
	return GetSupportedSchemes()
}

func (s *Service) generateKey(ctx context.Context, scheme *SignatureScheme) (*SigningKey, error) {
	// Get current version
	keys, _ := s.repo.ListKeys(ctx, scheme.ID)
	version := 1
	for _, k := range keys {
		if k.Version >= version {
			version = k.Version + 1
		}
	}

	key := &SigningKey{
		ID:        GenerateKeyID(),
		SchemeID:  scheme.ID,
		TenantID:  scheme.TenantID,
		Version:   version,
		Algorithm: scheme.Algorithm,
		Status:    KeyPrimary,
		CreatedAt: time.Now(),
	}

	// Generate secret
	keyLength := s.config.DefaultKeyLength
	if scheme.KeyConfig.KeyLength > 0 {
		keyLength = scheme.KeyConfig.KeyLength
	}

	secret := make([]byte, keyLength)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}

	key.SecretKey = base64.StdEncoding.EncodeToString(secret)

	// Generate fingerprint
	h := sha256.Sum256(secret)
	key.Fingerprint = hex.EncodeToString(h[:8])

	// Set expiration if configured
	if scheme.KeyConfig.MaxKeyAge > 0 {
		expiresAt := key.CreatedAt.Add(scheme.KeyConfig.MaxKeyAge)
		key.ExpiresAt = &expiresAt
	}

	// Demote existing primary key
	for _, k := range keys {
		if k.Status == KeyPrimary {
			s.repo.UpdateKeyStatus(ctx, k.ID, KeyActive)
		}
	}

	if err := s.repo.SaveKey(ctx, key); err != nil {
		return nil, err
	}

	return key, nil
}

// ProcessPendingRotations processes scheduled rotations (called by background job)
func (s *Service) ProcessPendingRotations(ctx context.Context) error {
	rotations, err := s.repo.GetPendingRotations(ctx)
	if err != nil {
		return err
	}

	for _, rotation := range rotations {
		switch rotation.Status {
		case RotationScheduledStatus:
			// Start the rotation
			now := time.Now()
			rotation.StartedAt = &now
			rotation.Status = RotationInProgress

			// Update old key to rotating
			s.repo.UpdateKeyStatus(ctx, rotation.OldKeyID, KeyRotating)

			// Set new key as primary
			newKey, err := s.repo.GetKey(ctx, rotation.NewKeyID)
			if err == nil {
				newKey.Status = KeyPrimary
				s.repo.SaveKey(ctx, newKey)
			}

			s.repo.SaveRotation(ctx, &rotation)

		case RotationInProgress:
			// Check if overlap period has passed
			if rotation.OverlapUntil != nil && time.Now().After(*rotation.OverlapUntil) {
				// Complete the rotation
				s.repo.UpdateKeyStatus(ctx, rotation.OldKeyID, KeyExpired)

				now := time.Now()
				rotation.CompletedAt = &now
				rotation.Status = RotationCompleted
				s.repo.SaveRotation(ctx, &rotation)
			}
		}
	}

	return nil
}
