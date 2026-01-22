package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateRandomString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		length int
	}{
		{"Generate 16 byte string", 16},
		{"Generate 32 byte string", 32},
		{"Generate 64 byte string", 64},
		{"Generate 1 byte string", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateRandomString(tt.length)
			
			assert.NoError(t, err)
			assert.NotEmpty(t, result)
			// Hex encoding doubles the length
			assert.Equal(t, tt.length*2, len(result))
		})
	}
}

func TestGenerateRandomStringUniqueness(t *testing.T) {
	t.Parallel()
	// Generate multiple random strings and ensure they're all different
	strings := make(map[string]bool)
	iterations := 100
	length := 32

	for i := 0; i < iterations; i++ {
		result, err := GenerateRandomString(length)
		assert.NoError(t, err)
		
		// Ensure we haven't seen this string before
		assert.False(t, strings[result], "Generated duplicate random string: %s", result)
		strings[result] = true
	}

	assert.Equal(t, iterations, len(strings))
}

func TestGenerateRandomBytes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		length int
	}{
		{"Generate 16 bytes", 16},
		{"Generate 32 bytes", 32},
		{"Generate 64 bytes", 64},
		{"Generate 1 byte", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateRandomBytes(tt.length)
			
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.length, len(result))
		})
	}
}

func TestGenerateRandomBytesUniqueness(t *testing.T) {
	t.Parallel()
	// Generate multiple random byte arrays and ensure they're all different
	byteArrays := make(map[string]bool)
	iterations := 100
	length := 32

	for i := 0; i < iterations; i++ {
		result, err := GenerateRandomBytes(length)
		assert.NoError(t, err)
		
		// Convert to string for comparison
		resultStr := string(result)
		
		// Ensure we haven't seen this byte array before
		assert.False(t, byteArrays[resultStr], "Generated duplicate random bytes")
		byteArrays[resultStr] = true
	}

	assert.Equal(t, iterations, len(byteArrays))
}

func TestGenerateRandomStringZeroLength(t *testing.T) {
	t.Parallel()
	result, err := GenerateRandomString(0)
	
	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestGenerateRandomBytesZeroLength(t *testing.T) {
	t.Parallel()
	result, err := GenerateRandomBytes(0)
	
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, len(result))
}

func TestHashPayload(t *testing.T) {
	t.Parallel()
	payload1 := []byte(`{"event": "test", "data": {"id": 123}}`)
	payload2 := []byte(`{"event": "test", "data": {"id": 456}}`)
	
	hash1 := HashPayload(payload1)
	hash2 := HashPayload(payload2)
	
	// Hashes should be different for different payloads
	assert.NotEqual(t, hash1, hash2)
	
	// Hash should be consistent for same payload
	hash1Again := HashPayload(payload1)
	assert.Equal(t, hash1, hash1Again)
	
	// Hash should be 64 characters (32 bytes in hex)
	assert.Len(t, hash1, 64)
	assert.Len(t, hash2, 64)
	
	// Test empty payload
	emptyHash := HashPayload([]byte{})
	assert.Len(t, emptyHash, 64)
	assert.NotEmpty(t, emptyHash)
}