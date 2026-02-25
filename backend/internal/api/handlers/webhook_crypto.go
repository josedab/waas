package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/josedab/waas/pkg/utils"
)

// calculatePayloadHash calculates SHA256 hash of the payload
func (h *WebhookHandler) calculatePayloadHash(payload json.RawMessage) string {
	hasher := sha256.New()
	hasher.Write(payload)
	return hex.EncodeToString(hasher.Sum(nil))
}

// generateWebhookSignature generates HMAC signature for webhook payload
func (h *WebhookHandler) generateWebhookSignature(payload json.RawMessage, secretHash string) (string, error) {
	// For now, we'll use the secret hash directly as the signing key
	// In a production system, you'd want to retrieve the actual secret
	// This is a simplified implementation for the webhook sending API
	signature := utils.GenerateWebhookSignature(payload, secretHash, "sha256")
	return fmt.Sprintf("sha256=%s", signature), nil
}

// generateSecret generates a random secret and its hash for webhook signing
func (h *WebhookHandler) generateSecret() (secret, hash string, err error) {
	// Generate 32 random bytes
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate random secret: %w", err)
	}

	// Convert to hex string
	secret = hex.EncodeToString(secretBytes)

	// Create hash for storage
	hasher := sha256.New()
	hasher.Write([]byte(secret))
	hash = hex.EncodeToString(hasher.Sum(nil))

	return secret, hash, nil
}
