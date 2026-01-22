package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAPIKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{
			name: "should generate valid API key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiKey, err := GenerateAPIKey()
			
			require.NoError(t, err)
			assert.NotEmpty(t, apiKey)
			assert.True(t, strings.HasPrefix(apiKey, APIKeyPrefix))
			assert.True(t, len(apiKey) > len(APIKeyPrefix))
		})
	}
}

func TestGenerateAPIKeyUniqueness(t *testing.T) {
	t.Parallel()
	// Generate multiple API keys and ensure they're unique
	keys := make(map[string]bool)
	for i := 0; i < 100; i++ {
		apiKey, err := GenerateAPIKey()
		require.NoError(t, err)
		
		// Ensure uniqueness
		assert.False(t, keys[apiKey], "Generated duplicate API key: %s", apiKey)
		keys[apiKey] = true
	}
}

func TestHashAPIKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		apiKey string
	}{
		{
			name:   "should hash valid API key",
			apiKey: "wh_test_key_123",
		},
		{
			name:   "should hash empty string",
			apiKey: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashAPIKey(tt.apiKey)
			
			require.NoError(t, err)
			assert.NotEmpty(t, hash)
			assert.NotEqual(t, tt.apiKey, hash)
		})
	}
}

func TestValidateAPIKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		apiKey   string
		expected bool
	}{
		{
			name:     "should validate correct API key",
			apiKey:   "wh_test_key_123",
			expected: true,
		},
		{
			name:     "should reject incorrect API key",
			apiKey:   "wh_wrong_key",
			expected: false,
		},
		{
			name:     "should reject empty API key",
			apiKey:   "",
			expected: false,
		},
	}

	// Generate a hash for the test key
	testAPIKey := "wh_test_key_123"
	validHash, err := HashAPIKey(testAPIKey)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hash string
			if tt.apiKey == testAPIKey {
				hash = validHash
			} else {
				// Use a different hash for invalid keys
				hash, _ = HashAPIKey("different_key")
			}
			
			result := ValidateAPIKey(tt.apiKey, hash)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateAPIKeyHash(t *testing.T) {
	t.Parallel()
	apiKey, hash, err := GenerateAPIKeyHash()
	
	require.NoError(t, err)
	assert.NotEmpty(t, apiKey)
	assert.NotEmpty(t, hash)
	assert.True(t, strings.HasPrefix(apiKey, APIKeyPrefix))
	assert.NotEqual(t, apiKey, hash)
	
	// Verify the hash validates the API key
	assert.True(t, ValidateAPIKey(apiKey, hash))
}

func TestExtractTenantFromAPIKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		apiKey string
	}{
		{
			name:   "should extract tenant ID from API key",
			apiKey: "wh_test_key_123",
		},
		{
			name:   "should handle empty API key",
			apiKey: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenantID := ExtractTenantFromAPIKey(tt.apiKey)
			
			assert.NotEmpty(t, tenantID)
			assert.Equal(t, 16, len(tenantID)) // Should be 16 characters (hex)
			
			// Should be deterministic
			tenantID2 := ExtractTenantFromAPIKey(tt.apiKey)
			assert.Equal(t, tenantID, tenantID2)
		})
	}
}

func TestExtractTenantFromAPIKeyUniqueness(t *testing.T) {
	t.Parallel()
	// Different API keys should produce different tenant IDs
	apiKey1 := "wh_key_1"
	apiKey2 := "wh_key_2"
	
	tenantID1 := ExtractTenantFromAPIKey(apiKey1)
	tenantID2 := ExtractTenantFromAPIKey(apiKey2)
	
	assert.NotEqual(t, tenantID1, tenantID2)
}

func TestIsValidAPIKeyFormat(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		apiKey   string
		expected bool
	}{
		{
			name:     "should accept valid API key format",
			apiKey:   "wh_valid_key_123",
			expected: true,
		},
		{
			name:     "should reject API key without prefix",
			apiKey:   "invalid_key_123",
			expected: false,
		},
		{
			name:     "should reject empty API key",
			apiKey:   "",
			expected: false,
		},
		{
			name:     "should reject API key with only prefix",
			apiKey:   "wh_",
			expected: false,
		},
		{
			name:     "should reject API key with wrong prefix",
			apiKey:   "api_key_123",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidAPIKeyFormat(tt.apiKey)
			assert.Equal(t, tt.expected, result)
		})
	}
}