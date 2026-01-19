package utils

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWebhookSignatureService(t *testing.T) {
	service := NewWebhookSignatureService(SHA256)
	
	assert.NotNil(t, service)
	assert.NotNil(t, service.generator)
	assert.Equal(t, 0, service.GetSecretCount())
}

func TestNewWebhookSignatureServiceWithSecrets(t *testing.T) {
	secrets := []string{"secret1", "secret2"}
	service := NewWebhookSignatureServiceWithSecrets(SHA256, secrets)
	
	assert.NotNil(t, service)
	assert.Equal(t, 2, service.GetSecretCount())
	assert.Equal(t, "secret1", service.GetPrimarySecret())
}

func TestGenerateWebhookSecret(t *testing.T) {
	service := NewWebhookSignatureService(SHA256)
	
	secret, err := service.GenerateWebhookSecret()
	
	assert.NoError(t, err)
	assert.NotEmpty(t, secret)
	assert.Greater(t, len(secret), 0)

	// Generate another secret and ensure they're different
	secret2, err := service.GenerateWebhookSecret()
	assert.NoError(t, err)
	assert.NotEqual(t, secret, secret2)
}

func TestHashSecret(t *testing.T) {
	service := NewWebhookSignatureService(SHA256)
	secret := "test-secret"
	
	hash := service.HashSecret(secret)
	
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, secret, hash)
	assert.Equal(t, 64, len(hash)) // SHA256 hex string length

	// Same secret should produce same hash
	hash2 := service.HashSecret(secret)
	assert.Equal(t, hash, hash2)

	// Different secret should produce different hash
	hash3 := service.HashSecret("different-secret")
	assert.NotEqual(t, hash, hash3)
}

func TestSignWebhookPayload(t *testing.T) {
	service := NewWebhookSignatureService(SHA256)
	service.AddSecretToEndpoint("test-secret")
	
	payload := []byte("test webhook payload")
	
	signature, err := service.SignWebhookPayload(payload)
	
	assert.NoError(t, err)
	assert.NotEmpty(t, signature)
	assert.Contains(t, signature, "sha256=")
}

func TestSignWebhookPayloadNoSecrets(t *testing.T) {
	service := NewWebhookSignatureService(SHA256)
	
	payload := []byte("test webhook payload")
	
	signature, err := service.SignWebhookPayload(payload)
	
	assert.Error(t, err)
	assert.Empty(t, signature)
	assert.Contains(t, err.Error(), "no secrets configured")
}

func TestVerifyWebhookSignature(t *testing.T) {
	service := NewWebhookSignatureService(SHA256)
	service.AddSecretToEndpoint("test-secret")
	
	payload := []byte("test webhook payload")
	
	signature, err := service.SignWebhookPayload(payload)
	require.NoError(t, err)
	
	valid, err := service.VerifyWebhookSignature(payload, signature)
	
	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestVerifyWebhookSignatureInvalid(t *testing.T) {
	service := NewWebhookSignatureService(SHA256)
	service.AddSecretToEndpoint("test-secret")
	
	payload := []byte("test webhook payload")
	invalidSignature := "sha256=invalid"
	
	valid, err := service.VerifyWebhookSignature(payload, invalidSignature)
	
	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestSignWebhookPayloadWithTimestamp(t *testing.T) {
	service := NewWebhookSignatureService(SHA256)
	service.AddSecretToEndpoint("test-secret")
	
	payload := []byte("test webhook payload")
	timestamp := time.Now()
	
	signature, err := service.SignWebhookPayloadWithTimestamp(payload, timestamp)
	
	assert.NoError(t, err)
	assert.NotEmpty(t, signature)
	assert.Contains(t, signature, "sha256=")
}

func TestVerifyWebhookSignatureWithTimestamp(t *testing.T) {
	service := NewWebhookSignatureService(SHA256)
	service.AddSecretToEndpoint("test-secret")
	
	payload := []byte("test webhook payload")
	timestamp := time.Now()
	
	signature, err := service.SignWebhookPayloadWithTimestamp(payload, timestamp)
	require.NoError(t, err)
	
	valid, err := service.VerifyWebhookSignatureWithTimestamp(payload, signature, timestamp, 300)
	
	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestRotateEndpointSecrets(t *testing.T) {
	service := NewWebhookSignatureService(SHA256)
	service.AddSecretToEndpoint("old-secret")
	
	assert.Equal(t, 1, service.GetSecretCount())
	assert.Equal(t, "old-secret", service.GetPrimarySecret())
	
	newSecrets := []string{"new-secret1", "new-secret2"}
	service.RotateEndpointSecrets(newSecrets)
	
	assert.Equal(t, 2, service.GetSecretCount())
	assert.Equal(t, "new-secret1", service.GetPrimarySecret())
}

func TestGetSignatureHeaders(t *testing.T) {
	service := NewWebhookSignatureService(SHA256)
	service.AddSecretToEndpoint("test-secret")
	
	payload := []byte("test webhook payload")
	
	headers, err := service.GetSignatureHeaders(payload)
	
	assert.NoError(t, err)
	assert.NotNil(t, headers)
	
	// Check required headers are present
	assert.Contains(t, headers, "X-Webhook-Signature")
	assert.Contains(t, headers, "X-Webhook-Signature-Timestamp")
	assert.Contains(t, headers, "X-Webhook-Timestamp")
	
	// Check signature format
	assert.Contains(t, headers["X-Webhook-Signature"], "sha256=")
	assert.Contains(t, headers["X-Webhook-Signature-Timestamp"], "sha256=")
	
	// Check timestamp is numeric
	timestamp := headers["X-Webhook-Timestamp"]
	assert.NotEmpty(t, timestamp)
	
	var ts int64
	_, err = fmt.Sscanf(timestamp, "%d", &ts)
	assert.NoError(t, err)
}

func TestGetSignatureHeadersNoSecrets(t *testing.T) {
	service := NewWebhookSignatureService(SHA256)
	
	payload := []byte("test webhook payload")
	
	headers, err := service.GetSignatureHeaders(payload)
	
	assert.Error(t, err)
	assert.Nil(t, headers)
	assert.Contains(t, err.Error(), "failed to generate signature")
}

func TestValidateWebhookHeaders(t *testing.T) {
	service := NewWebhookSignatureService(SHA256)
	service.AddSecretToEndpoint("test-secret")
	
	payload := []byte("test webhook payload")
	
	// Generate valid headers
	headers, err := service.GetSignatureHeaders(payload)
	require.NoError(t, err)
	
	// Validate the headers
	err = service.ValidateWebhookHeaders(payload, headers, 300)
	assert.NoError(t, err)
}

func TestValidateWebhookHeadersInvalidSignature(t *testing.T) {
	service := NewWebhookSignatureService(SHA256)
	service.AddSecretToEndpoint("test-secret")
	
	payload := []byte("test webhook payload")
	
	headers := map[string]string{
		"X-Webhook-Signature": "sha256=invalid",
	}
	
	err := service.ValidateWebhookHeaders(payload, headers, 300)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid webhook signature")
}

func TestValidateWebhookHeadersInvalidTimestamp(t *testing.T) {
	service := NewWebhookSignatureService(SHA256)
	service.AddSecretToEndpoint("test-secret")
	
	payload := []byte("test webhook payload")
	
	headers := map[string]string{
		"X-Webhook-Timestamp": "invalid-timestamp",
		"X-Webhook-Signature-Timestamp": "sha256=somehash",
	}
	
	err := service.ValidateWebhookHeaders(payload, headers, 300)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid timestamp format")
}

func TestValidateWebhookHeadersExpiredTimestamp(t *testing.T) {
	service := NewWebhookSignatureService(SHA256)
	service.AddSecretToEndpoint("test-secret")
	
	payload := []byte("test webhook payload")
	
	// Create timestamp that's too old
	oldTimestamp := time.Now().Add(-10 * time.Minute)
	signature, err := service.SignWebhookPayloadWithTimestamp(payload, oldTimestamp)
	require.NoError(t, err)
	
	headers := map[string]string{
		"X-Webhook-Timestamp":           fmt.Sprintf("%d", oldTimestamp.Unix()),
		"X-Webhook-Signature-Timestamp": signature,
	}
	
	err = service.ValidateWebhookHeaders(payload, headers, 300) // 5 minutes tolerance
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timestamp signature verification failed")
}

func TestValidateWebhookHeadersNoHeaders(t *testing.T) {
	service := NewWebhookSignatureService(SHA256)
	service.AddSecretToEndpoint("test-secret")
	
	payload := []byte("test webhook payload")
	headers := map[string]string{}
	
	// Should not error when no headers are provided (optional validation)
	err := service.ValidateWebhookHeaders(payload, headers, 300)
	assert.NoError(t, err)
}

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		name        string
		timestampStr string
		wantErr     bool
	}{
		{
			name:        "Valid timestamp",
			timestampStr: "1640995200", // 2022-01-01 00:00:00 UTC
			wantErr:     false,
		},
		{
			name:        "Invalid timestamp format",
			timestampStr: "not-a-number",
			wantErr:     true,
		},
		{
			name:        "Empty timestamp",
			timestampStr: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timestamp, err := parseTimestamp(tt.timestampStr)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.True(t, timestamp.IsZero())
			} else {
				assert.NoError(t, err)
				assert.False(t, timestamp.IsZero())
			}
		})
	}
}

func TestWebhookSignatureServiceIntegration(t *testing.T) {
	// Test complete workflow: generate secret, sign payload, verify signature
	service := NewWebhookSignatureService(SHA256)
	
	// Generate a secret
	secret, err := service.GenerateWebhookSecret()
	require.NoError(t, err)
	
	// Add secret to service
	service.AddSecretToEndpoint(secret)
	
	// Create payload
	payload := []byte(`{"event": "user.created", "data": {"id": 123, "email": "test@example.com"}}`)
	
	// Generate signature headers
	headers, err := service.GetSignatureHeaders(payload)
	require.NoError(t, err)
	
	// Validate the headers
	err = service.ValidateWebhookHeaders(payload, headers, 300)
	assert.NoError(t, err)
	
	// Test with wrong payload should fail
	wrongPayload := []byte(`{"event": "user.deleted", "data": {"id": 123}}`)
	err = service.ValidateWebhookHeaders(wrongPayload, headers, 300)
	assert.Error(t, err)
}