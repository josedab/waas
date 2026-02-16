package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"strings"
	"time"
)

// SignatureAlgorithm represents supported HMAC algorithms
type SignatureAlgorithm string

const (
	SHA256 SignatureAlgorithm = "sha256"
	SHA512 SignatureAlgorithm = "sha512"
)

// SignatureConfig holds configuration for signature generation
type SignatureConfig struct {
	Algorithm SignatureAlgorithm `json:"algorithm"`
	Secrets   []string           `json:"secrets"` // Multiple secrets for rotation support
}

// SignatureGenerator handles webhook signature generation and verification
type SignatureGenerator struct {
	config SignatureConfig
}

// NewSignatureGenerator creates a new signature generator with the given configuration
func NewSignatureGenerator(config SignatureConfig) *SignatureGenerator {
	return &SignatureGenerator{
		config: config,
	}
}

// GenerateSignature creates an HMAC signature for the given payload using the primary secret
func (sg *SignatureGenerator) GenerateSignature(payload []byte) (string, error) {
	if len(sg.config.Secrets) == 0 {
		return "", fmt.Errorf("no secrets configured for signature generation")
	}

	// Use the first secret as the primary secret for generation
	primarySecret := sg.config.Secrets[0]

	hasher, err := sg.getHasher(sg.config.Algorithm)
	if err != nil {
		return "", fmt.Errorf("failed to get hasher: %w", err)
	}

	mac := hmac.New(hasher, []byte(primarySecret))
	mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))

	return fmt.Sprintf("%s=%s", sg.config.Algorithm, signature), nil
}

// VerifySignature verifies a signature against the payload using any of the configured secrets
func (sg *SignatureGenerator) VerifySignature(payload []byte, signature string) (bool, error) {
	if len(sg.config.Secrets) == 0 {
		return false, fmt.Errorf("no secrets configured for signature verification")
	}

	// Parse the signature to extract algorithm and hash
	parts := strings.SplitN(signature, "=", 2)
	if len(parts) != 2 {
		return false, fmt.Errorf("invalid signature format, expected 'algorithm=hash'")
	}

	algorithm := SignatureAlgorithm(parts[0])
	expectedHash := parts[1]

	// Verify the algorithm matches our configuration
	if algorithm != sg.config.Algorithm {
		return false, fmt.Errorf("signature algorithm %s does not match configured algorithm %s", algorithm, sg.config.Algorithm)
	}

	hasher, err := sg.getHasher(algorithm)
	if err != nil {
		return false, fmt.Errorf("failed to get hasher: %w", err)
	}

	// Try verification with each configured secret (supports rotation)
	for _, secret := range sg.config.Secrets {
		mac := hmac.New(hasher, []byte(secret))
		mac.Write(payload)
		computedHash := hex.EncodeToString(mac.Sum(nil))

		if hmac.Equal([]byte(expectedHash), []byte(computedHash)) {
			return true, nil
		}
	}

	return false, nil
}

// GenerateSignatureWithTimestamp creates a signature that includes timestamp for replay protection
func (sg *SignatureGenerator) GenerateSignatureWithTimestamp(payload []byte, timestamp time.Time) (string, error) {
	if len(sg.config.Secrets) == 0 {
		return "", fmt.Errorf("no secrets configured for signature generation")
	}

	// Create payload with timestamp
	timestampStr := fmt.Sprintf("%d", timestamp.Unix())
	signedPayload := append([]byte(timestampStr+"."), payload...)

	primarySecret := sg.config.Secrets[0]

	hasher, err := sg.getHasher(sg.config.Algorithm)
	if err != nil {
		return "", fmt.Errorf("failed to get hasher: %w", err)
	}

	mac := hmac.New(hasher, []byte(primarySecret))
	mac.Write(signedPayload)
	signature := hex.EncodeToString(mac.Sum(nil))

	return fmt.Sprintf("%s=%s", sg.config.Algorithm, signature), nil
}

// VerifySignatureWithTimestamp verifies a signature with timestamp validation
func (sg *SignatureGenerator) VerifySignatureWithTimestamp(payload []byte, signature string, timestamp time.Time, toleranceSeconds int) (bool, error) {
	if len(sg.config.Secrets) == 0 {
		return false, fmt.Errorf("no secrets configured for signature verification")
	}

	// Check timestamp tolerance to prevent replay attacks
	now := time.Now()
	if now.Sub(timestamp).Seconds() > float64(toleranceSeconds) {
		return false, fmt.Errorf("timestamp is too old, potential replay attack")
	}

	// Parse the signature
	parts := strings.SplitN(signature, "=", 2)
	if len(parts) != 2 {
		return false, fmt.Errorf("invalid signature format, expected 'algorithm=hash'")
	}

	algorithm := SignatureAlgorithm(parts[0])
	expectedHash := parts[1]

	if algorithm != sg.config.Algorithm {
		return false, fmt.Errorf("signature algorithm %s does not match configured algorithm %s", algorithm, sg.config.Algorithm)
	}

	hasher, err := sg.getHasher(algorithm)
	if err != nil {
		return false, fmt.Errorf("failed to get hasher: %w", err)
	}

	// Create payload with timestamp
	timestampStr := fmt.Sprintf("%d", timestamp.Unix())
	signedPayload := append([]byte(timestampStr+"."), payload...)

	// Try verification with each configured secret
	for _, secret := range sg.config.Secrets {
		mac := hmac.New(hasher, []byte(secret))
		mac.Write(signedPayload)
		computedHash := hex.EncodeToString(mac.Sum(nil))

		if hmac.Equal([]byte(expectedHash), []byte(computedHash)) {
			return true, nil
		}
	}

	return false, nil
}

// AddSecret adds a new secret to the configuration for rotation support
func (sg *SignatureGenerator) AddSecret(secret string) {
	sg.config.Secrets = append(sg.config.Secrets, secret)
}

// RemoveSecret removes a secret from the configuration
func (sg *SignatureGenerator) RemoveSecret(secret string) {
	for i, s := range sg.config.Secrets {
		if hmac.Equal([]byte(s), []byte(secret)) {
			sg.config.Secrets = append(sg.config.Secrets[:i], sg.config.Secrets[i+1:]...)
			break
		}
	}
}

// RotateSecrets replaces all secrets with new ones
func (sg *SignatureGenerator) RotateSecrets(newSecrets []string) {
	sg.config.Secrets = newSecrets
}

// GetPrimarySecret returns the primary secret (first in the list)
func (sg *SignatureGenerator) GetPrimarySecret() string {
	if len(sg.config.Secrets) == 0 {
		return ""
	}
	return sg.config.Secrets[0]
}

// GetSecretCount returns the number of configured secrets
func (sg *SignatureGenerator) GetSecretCount() int {
	return len(sg.config.Secrets)
}

// getHasher returns the appropriate hash function for the algorithm
func (sg *SignatureGenerator) getHasher(algorithm SignatureAlgorithm) (func() hash.Hash, error) {
	switch algorithm {
	case SHA256:
		return sha256.New, nil
	case SHA512:
		return sha512.New, nil
	default:
		return nil, fmt.Errorf("unsupported signature algorithm: %s", algorithm)
	}
}

// ValidateAlgorithm checks if the given algorithm is supported
func ValidateAlgorithm(algorithm SignatureAlgorithm) bool {
	return algorithm == SHA256 || algorithm == SHA512
}

// GenerateSecret creates a cryptographically secure secret for webhook signing
func GenerateSecret() (string, error) {
	return GenerateRandomString(32)
}
