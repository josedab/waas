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

// generateWebhookSignature generates HMAC-SHA256 signature for webhook payload.
// The signingKey is the raw secret (stored in the secret_hash DB column) — the
// same value that was returned to the user at endpoint creation. Both the server
// and the SDK verifier use this identical key for HMAC computation.
func (h *WebhookHandler) generateWebhookSignature(payload json.RawMessage, signingKey string) (string, error) {
	signature := utils.GenerateWebhookSignature(payload, signingKey, "sha256")
	return fmt.Sprintf("sha256=%s", signature), nil
}

// generateSecret generates a random secret for webhook signing.
// Returns (secret, storedKey, err) where:
//   - secret: the value shown to the user once at creation time
//   - storedKey: the value persisted in the DB (same as secret, used for HMAC signing)
//
// The secret_hash column stores the raw signing key (not a one-way hash) so that
// the delivery engine can compute HMAC signatures that SDK verifiers can validate.
func (h *WebhookHandler) generateSecret() (secret, storedKey string, err error) {
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate random secret: %w", err)
	}

	secret = hex.EncodeToString(secretBytes)
	// Store the raw secret as the signing key — delivery engine uses this for
	// HMAC computation, and SDK verifiers use the same secret to verify.
	storedKey = secret

	return secret, storedKey, nil
}
