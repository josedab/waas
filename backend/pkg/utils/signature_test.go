package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSignatureGenerator(t *testing.T) {
	config := SignatureConfig{
		Algorithm: SHA256,
		Secrets:   []string{"secret1", "secret2"},
	}

	generator := NewSignatureGenerator(config)
	
	assert.NotNil(t, generator)
	assert.Equal(t, config.Algorithm, generator.config.Algorithm)
	assert.Equal(t, config.Secrets, generator.config.Secrets)
}

func TestGenerateSignature(t *testing.T) {
	tests := []struct {
		name      string
		algorithm SignatureAlgorithm
		secrets   []string
		payload   []byte
		wantErr   bool
	}{
		{
			name:      "SHA256 signature generation",
			algorithm: SHA256,
			secrets:   []string{"test-secret"},
			payload:   []byte("test payload"),
			wantErr:   false,
		},
		{
			name:      "SHA512 signature generation",
			algorithm: SHA512,
			secrets:   []string{"test-secret"},
			payload:   []byte("test payload"),
			wantErr:   false,
		},
		{
			name:      "No secrets configured",
			algorithm: SHA256,
			secrets:   []string{},
			payload:   []byte("test payload"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := SignatureConfig{
				Algorithm: tt.algorithm,
				Secrets:   tt.secrets,
			}
			generator := NewSignatureGenerator(config)

			signature, err := generator.GenerateSignature(tt.payload)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, signature)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, signature)
				assert.Contains(t, signature, string(tt.algorithm)+"=")
			}
		})
	}
}

func TestVerifySignature(t *testing.T) {
	config := SignatureConfig{
		Algorithm: SHA256,
		Secrets:   []string{"secret1", "secret2"},
	}
	generator := NewSignatureGenerator(config)
	payload := []byte("test payload")

	// Generate a signature
	signature, err := generator.GenerateSignature(payload)
	require.NoError(t, err)

	tests := []struct {
		name      string
		payload   []byte
		signature string
		want      bool
		wantErr   bool
	}{
		{
			name:      "Valid signature verification",
			payload:   payload,
			signature: signature,
			want:      true,
			wantErr:   false,
		},
		{
			name:      "Invalid signature",
			payload:   payload,
			signature: "sha256=invalid",
			want:      false,
			wantErr:   false,
		},
		{
			name:      "Wrong payload",
			payload:   []byte("different payload"),
			signature: signature,
			want:      false,
			wantErr:   false,
		},
		{
			name:      "Invalid signature format",
			payload:   payload,
			signature: "invalid-format",
			want:      false,
			wantErr:   true,
		},
		{
			name:      "Wrong algorithm",
			payload:   payload,
			signature: "sha512=" + signature[7:], // Replace sha256 with sha512
			want:      false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := generator.VerifySignature(tt.payload, tt.signature)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, valid)
			}
		})
	}
}

func TestVerifySignatureWithMultipleSecrets(t *testing.T) {
	// Test signature verification with multiple secrets (rotation scenario)
	config1 := SignatureConfig{
		Algorithm: SHA256,
		Secrets:   []string{"old-secret"},
	}
	generator1 := NewSignatureGenerator(config1)
	payload := []byte("test payload")

	// Generate signature with old secret
	oldSignature, err := generator1.GenerateSignature(payload)
	require.NoError(t, err)

	// Create new generator with both old and new secrets
	config2 := SignatureConfig{
		Algorithm: SHA256,
		Secrets:   []string{"new-secret", "old-secret"},
	}
	generator2 := NewSignatureGenerator(config2)

	// Should be able to verify signature created with old secret
	valid, err := generator2.VerifySignature(payload, oldSignature)
	assert.NoError(t, err)
	assert.True(t, valid)

	// Generate new signature with new secret (primary)
	newSignature, err := generator2.GenerateSignature(payload)
	require.NoError(t, err)

	// Should be able to verify new signature
	valid, err = generator2.VerifySignature(payload, newSignature)
	assert.NoError(t, err)
	assert.True(t, valid)

	// Old and new signatures should be different
	assert.NotEqual(t, oldSignature, newSignature)
}

func TestGenerateSignatureWithTimestamp(t *testing.T) {
	config := SignatureConfig{
		Algorithm: SHA256,
		Secrets:   []string{"test-secret"},
	}
	generator := NewSignatureGenerator(config)
	payload := []byte("test payload")
	timestamp := time.Now()

	signature, err := generator.GenerateSignatureWithTimestamp(payload, timestamp)
	
	assert.NoError(t, err)
	assert.NotEmpty(t, signature)
	assert.Contains(t, signature, "sha256=")
}

func TestVerifySignatureWithTimestamp(t *testing.T) {
	config := SignatureConfig{
		Algorithm: SHA256,
		Secrets:   []string{"test-secret"},
	}
	generator := NewSignatureGenerator(config)
	payload := []byte("test payload")
	timestamp := time.Now()

	signature, err := generator.GenerateSignatureWithTimestamp(payload, timestamp)
	require.NoError(t, err)

	tests := []struct {
		name              string
		payload           []byte
		signature         string
		timestamp         time.Time
		toleranceSeconds  int
		want              bool
		wantErr           bool
	}{
		{
			name:             "Valid timestamp signature",
			payload:          payload,
			signature:        signature,
			timestamp:        timestamp,
			toleranceSeconds: 300, // 5 minutes
			want:             true,
			wantErr:          false,
		},
		{
			name:             "Expired timestamp",
			payload:          payload,
			signature:        signature,
			timestamp:        time.Now().Add(-10 * time.Minute), // 10 minutes ago
			toleranceSeconds: 300, // 5 minutes tolerance
			want:             false,
			wantErr:          true,
		},
		{
			name:             "Invalid signature with timestamp",
			payload:          payload,
			signature:        "sha256=invalid",
			timestamp:        timestamp,
			toleranceSeconds: 300,
			want:             false,
			wantErr:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := generator.VerifySignatureWithTimestamp(tt.payload, tt.signature, tt.timestamp, tt.toleranceSeconds)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, valid)
			}
		})
	}
}

func TestSecretManagement(t *testing.T) {
	config := SignatureConfig{
		Algorithm: SHA256,
		Secrets:   []string{"secret1"},
	}
	generator := NewSignatureGenerator(config)

	// Test adding secret
	generator.AddSecret("secret2")
	assert.Equal(t, 2, generator.GetSecretCount())

	// Test getting primary secret
	assert.Equal(t, "secret1", generator.GetPrimarySecret())

	// Test removing secret
	generator.RemoveSecret("secret1")
	assert.Equal(t, 1, generator.GetSecretCount())
	assert.Equal(t, "secret2", generator.GetPrimarySecret())

	// Test rotating secrets
	newSecrets := []string{"new1", "new2", "new3"}
	generator.RotateSecrets(newSecrets)
	assert.Equal(t, 3, generator.GetSecretCount())
	assert.Equal(t, "new1", generator.GetPrimarySecret())
}

func TestValidateAlgorithm(t *testing.T) {
	tests := []struct {
		algorithm SignatureAlgorithm
		want      bool
	}{
		{SHA256, true},
		{SHA512, true},
		{"md5", false},
		{"sha1", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.algorithm), func(t *testing.T) {
			result := ValidateAlgorithm(tt.algorithm)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestGenerateSecret(t *testing.T) {
	secret, err := GenerateSecret()
	
	assert.NoError(t, err)
	assert.NotEmpty(t, secret)
	assert.Greater(t, len(secret), 0)

	// Generate another secret and ensure they're different
	secret2, err := GenerateSecret()
	assert.NoError(t, err)
	assert.NotEqual(t, secret, secret2)
}

func TestSignatureConsistency(t *testing.T) {
	// Test that the same payload and secret always generate the same signature
	config := SignatureConfig{
		Algorithm: SHA256,
		Secrets:   []string{"consistent-secret"},
	}
	generator := NewSignatureGenerator(config)
	payload := []byte("consistent payload")

	signature1, err := generator.GenerateSignature(payload)
	require.NoError(t, err)

	signature2, err := generator.GenerateSignature(payload)
	require.NoError(t, err)

	assert.Equal(t, signature1, signature2)
}

func TestDifferentAlgorithmsProduceDifferentSignatures(t *testing.T) {
	payload := []byte("test payload")
	secret := "test-secret"

	config256 := SignatureConfig{
		Algorithm: SHA256,
		Secrets:   []string{secret},
	}
	generator256 := NewSignatureGenerator(config256)

	config512 := SignatureConfig{
		Algorithm: SHA512,
		Secrets:   []string{secret},
	}
	generator512 := NewSignatureGenerator(config512)

	signature256, err := generator256.GenerateSignature(payload)
	require.NoError(t, err)

	signature512, err := generator512.GenerateSignature(payload)
	require.NoError(t, err)

	assert.NotEqual(t, signature256, signature512)
	assert.Contains(t, signature256, "sha256=")
	assert.Contains(t, signature512, "sha512=")
}

func TestEmptyPayloadSignature(t *testing.T) {
	config := SignatureConfig{
		Algorithm: SHA256,
		Secrets:   []string{"test-secret"},
	}
	generator := NewSignatureGenerator(config)

	signature, err := generator.GenerateSignature([]byte{})
	assert.NoError(t, err)
	assert.NotEmpty(t, signature)

	valid, err := generator.VerifySignature([]byte{}, signature)
	assert.NoError(t, err)
	assert.True(t, valid)
}