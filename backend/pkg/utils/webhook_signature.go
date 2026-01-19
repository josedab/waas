package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// WebhookSignatureService provides webhook-specific signature operations
type WebhookSignatureService struct {
	generator *SignatureGenerator
}

// NewWebhookSignatureService creates a new webhook signature service
func NewWebhookSignatureService(algorithm SignatureAlgorithm) *WebhookSignatureService {
	config := SignatureConfig{
		Algorithm: algorithm,
		Secrets:   []string{},
	}
	
	return &WebhookSignatureService{
		generator: NewSignatureGenerator(config),
	}
}

// NewWebhookSignatureServiceWithSecrets creates a new webhook signature service with predefined secrets
func NewWebhookSignatureServiceWithSecrets(algorithm SignatureAlgorithm, secrets []string) *WebhookSignatureService {
	config := SignatureConfig{
		Algorithm: algorithm,
		Secrets:   secrets,
	}
	
	return &WebhookSignatureService{
		generator: NewSignatureGenerator(config),
	}
}

// GenerateWebhookSecret creates a new secret for webhook signing
func (wss *WebhookSignatureService) GenerateWebhookSecret() (string, error) {
	return GenerateSecret()
}

// HashSecret creates a hash of the secret for storage (never store raw secrets)
func (wss *WebhookSignatureService) HashSecret(secret string) string {
	hash := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(hash[:])
}

// AddSecretToEndpoint adds a secret to the signature generator
func (wss *WebhookSignatureService) AddSecretToEndpoint(secret string) {
	wss.generator.AddSecret(secret)
}

// SignWebhookPayload generates a signature for a webhook payload
func (wss *WebhookSignatureService) SignWebhookPayload(payload []byte) (string, error) {
	return wss.generator.GenerateSignature(payload)
}

// SignWebhookPayloadWithTimestamp generates a signature with timestamp for replay protection
func (wss *WebhookSignatureService) SignWebhookPayloadWithTimestamp(payload []byte, timestamp time.Time) (string, error) {
	return wss.generator.GenerateSignatureWithTimestamp(payload, timestamp)
}

// VerifyWebhookSignature verifies a webhook signature
func (wss *WebhookSignatureService) VerifyWebhookSignature(payload []byte, signature string) (bool, error) {
	return wss.generator.VerifySignature(payload, signature)
}

// VerifyWebhookSignatureWithTimestamp verifies a webhook signature with timestamp validation
func (wss *WebhookSignatureService) VerifyWebhookSignatureWithTimestamp(payload []byte, signature string, timestamp time.Time, toleranceSeconds int) (bool, error) {
	return wss.generator.VerifySignatureWithTimestamp(payload, signature, timestamp, toleranceSeconds)
}

// RotateEndpointSecrets rotates all secrets for an endpoint
func (wss *WebhookSignatureService) RotateEndpointSecrets(newSecrets []string) {
	wss.generator.RotateSecrets(newSecrets)
}

// GetSignatureHeaders returns the standard webhook signature headers
func (wss *WebhookSignatureService) GetSignatureHeaders(payload []byte) (map[string]string, error) {
	signature, err := wss.SignWebhookPayload(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to generate signature: %w", err)
	}

	timestamp := time.Now()
	timestampSignature, err := wss.SignWebhookPayloadWithTimestamp(payload, timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to generate timestamp signature: %w", err)
	}

	headers := map[string]string{
		"X-Webhook-Signature":           signature,
		"X-Webhook-Signature-Timestamp": timestampSignature,
		"X-Webhook-Timestamp":           fmt.Sprintf("%d", timestamp.Unix()),
	}

	return headers, nil
}

// ValidateWebhookHeaders validates incoming webhook signature headers
func (wss *WebhookSignatureService) ValidateWebhookHeaders(payload []byte, headers map[string]string, toleranceSeconds int) error {
	signature, hasSignature := headers["X-Webhook-Signature"]
	timestampSignature, hasTimestampSignature := headers["X-Webhook-Signature-Timestamp"]
	timestampStr, hasTimestamp := headers["X-Webhook-Timestamp"]

	// Basic signature validation
	if hasSignature {
		valid, err := wss.VerifyWebhookSignature(payload, signature)
		if err != nil {
			return fmt.Errorf("signature verification failed: %w", err)
		}
		if !valid {
			return fmt.Errorf("invalid webhook signature")
		}
	}

	// Timestamp signature validation (if available)
	if hasTimestampSignature && hasTimestamp {
		timestamp, err := parseTimestamp(timestampStr)
		if err != nil {
			return fmt.Errorf("invalid timestamp format: %w", err)
		}

		valid, err := wss.VerifyWebhookSignatureWithTimestamp(payload, timestampSignature, timestamp, toleranceSeconds)
		if err != nil {
			return fmt.Errorf("timestamp signature verification failed: %w", err)
		}
		if !valid {
			return fmt.Errorf("invalid webhook timestamp signature")
		}
	}

	return nil
}

// parseTimestamp parses a Unix timestamp string
func parseTimestamp(timestampStr string) (time.Time, error) {
	var timestamp int64
	_, err := fmt.Sscanf(timestampStr, "%d", &timestamp)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse timestamp: %w", err)
	}
	return time.Unix(timestamp, 0), nil
}

// GetSecretCount returns the number of secrets configured
func (wss *WebhookSignatureService) GetSecretCount() int {
	return wss.generator.GetSecretCount()
}

// GetPrimarySecret returns the primary secret (for generation)
func (wss *WebhookSignatureService) GetPrimarySecret() string {
	return wss.generator.GetPrimarySecret()
}

// GenerateWebhookSignature is a simple utility function to generate HMAC signatures
func GenerateWebhookSignature(payload []byte, secret string, algorithm string) string {
	var alg SignatureAlgorithm
	switch algorithm {
	case "sha256":
		alg = SHA256
	case "sha512":
		alg = SHA512
	default:
		alg = SHA256
	}
	
	config := SignatureConfig{
		Algorithm: alg,
		Secrets:   []string{secret},
	}
	
	generator := NewSignatureGenerator(config)
	signature, _ := generator.GenerateSignature(payload)
	return signature
}