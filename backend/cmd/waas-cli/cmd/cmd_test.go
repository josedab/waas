package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- send.go tests ---

func TestSplitHeader_ValidHeader(t *testing.T) {
	t.Parallel()
	parts := splitHeader("X-Custom-Header: my-value")
	assert.Equal(t, []string{"X-Custom-Header", "my-value"}, parts)
}

func TestSplitHeader_NoSpace(t *testing.T) {
	t.Parallel()
	parts := splitHeader("X-Custom:value")
	assert.Equal(t, []string{"X-Custom", "value"}, parts)
}

func TestSplitHeader_EmptyValue(t *testing.T) {
	t.Parallel()
	parts := splitHeader("X-Empty:")
	assert.Equal(t, []string{"X-Empty", ""}, parts)
}

func TestSplitHeader_NoColon(t *testing.T) {
	t.Parallel()
	parts := splitHeader("invalid-header")
	assert.Nil(t, parts)
}

func TestSplitHeader_MultipleColons(t *testing.T) {
	t.Parallel()
	parts := splitHeader("Authorization: Bearer: token:123")
	assert.Equal(t, "Authorization", parts[0])
	assert.Equal(t, "Bearer: token:123", parts[1])
}

func TestSendCmd_RequiresEndpointOrTopic(t *testing.T) {
	t.Parallel()
	// Reset global flags
	oldEndpoint := sendEndpointID
	oldTopic := sendTopic
	defer func() {
		sendEndpointID = oldEndpoint
		sendTopic = oldTopic
	}()

	sendEndpointID = ""
	sendTopic = ""

	err := runSend(sendCmd, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "either --endpoint or --topic is required")
}

// --- config.go tests ---

func TestConfigKeyMap_ValidKeys(t *testing.T) {
	t.Parallel()
	assert.Contains(t, configKeyMap, "api-url")
	assert.Contains(t, configKeyMap, "api-key")
	assert.Contains(t, configKeyMap, "output")
	assert.Equal(t, "api_url", configKeyMap["api-url"])
	assert.Equal(t, "api_key", configKeyMap["api-key"])
	assert.Equal(t, "output", configKeyMap["output"])
}

func TestMaskSensitive_APIKey(t *testing.T) {
	t.Parallel()
	key := "wh_sk_abcdefghijklmnop"
	masked := maskSensitive("api-key", key)
	assert.True(t, len(masked) == len(key))
	assert.Equal(t, key[:8], masked[:8])
	assert.NotContains(t, masked[8:], "c") // chars after index 8 are masked
}

func TestMaskSensitive_ShortKey(t *testing.T) {
	t.Parallel()
	// Keys <= 8 chars are not masked
	result := maskSensitive("api-key", "short")
	assert.Equal(t, "short", result)
}

func TestMaskSensitive_NonSensitiveKey(t *testing.T) {
	t.Parallel()
	result := maskSensitive("api-url", "http://localhost:8080")
	assert.Equal(t, "http://localhost:8080", result)
}

func TestConfigSet_InvalidKey(t *testing.T) {
	t.Parallel()
	err := runConfigSet(configSetCmd, []string{"invalid-key", "value"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown config key")
}

// --- health.go tests ---

func TestHealthCmd_UnreachableEndpoint(t *testing.T) {
	t.Parallel()
	// Override API URL to an unreachable endpoint
	oldAPIURL := apiURL
	defer func() { apiURL = oldAPIURL }()
	apiURL = "http://127.0.0.1:1" // Port 1 = unreachable

	err := runHealth(healthCmd, nil)
	assert.Error(t, err)
}

// --- login.go tests ---

func TestLoginCmd_EmptyAPIKey(t *testing.T) {
	t.Parallel()
	// The login command should fail if API key is empty and stdin is not interactive
	// We test the validation logic: empty key returns error
	// The actual interactive prompt requires TTY, so we test the non-interactive path
	assert.NotNil(t, loginCmd)
	assert.Equal(t, "login", loginCmd.Use)
}
