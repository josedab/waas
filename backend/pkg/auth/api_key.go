package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	APIKeyPrefix = "wh_"
	APIKeyLength = 32
)

// GenerateAPIKey generates a new API key with the webhook platform prefix
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, APIKeyLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	key := base64.URLEncoding.EncodeToString(bytes)
	// Remove padding and ensure consistent length
	key = strings.TrimRight(key, "=")

	return APIKeyPrefix + key, nil
}

// HashAPIKey creates a bcrypt hash of the API key for secure storage
func HashAPIKey(apiKey string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(apiKey), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash API key: %w", err)
	}
	return string(hash), nil
}

// ValidateAPIKey checks if the provided API key matches the stored hash
func ValidateAPIKey(apiKey, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(apiKey))
	return err == nil
}

// GenerateAPIKeyHash generates both the API key and its hash
func GenerateAPIKeyHash() (apiKey, hash string, err error) {
	apiKey, err = GenerateAPIKey()
	if err != nil {
		return "", "", err
	}

	hash, err = HashAPIKey(apiKey)
	if err != nil {
		return "", "", err
	}

	return apiKey, hash, nil
}

// LookupHash returns a SHA-256 hex digest of the API key for fast DB lookups.
// Unlike bcrypt, SHA-256 is deterministic, so it can be used in WHERE clauses.
func LookupHash(apiKey string) string {
	h := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(h[:])
}

// ExtractTenantFromAPIKey extracts tenant identifier from API key for quick lookups
// This creates a deterministic hash that can be used for database indexing
func ExtractTenantFromAPIKey(apiKey string) string {
	hasher := sha256.New()
	hasher.Write([]byte(apiKey))
	return hex.EncodeToString(hasher.Sum(nil))[:16]
}

// IsValidAPIKeyFormat checks if the API key has the correct format
func IsValidAPIKeyFormat(apiKey string) bool {
	return strings.HasPrefix(apiKey, APIKeyPrefix) && len(apiKey) > len(APIKeyPrefix)
}
