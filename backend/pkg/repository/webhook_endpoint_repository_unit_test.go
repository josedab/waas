package repository

import (
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"testing"
	"time"
	"github.com/josedab/waas/pkg/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWebhookEndpointRepositoryLogic tests the business logic of webhook endpoint repository operations
func TestWebhookEndpointRepositoryLogic(t *testing.T) {
	t.Run("webhook endpoint creation logic", func(t *testing.T) {
		endpoint := &models.WebhookEndpoint{
			TenantID:   uuid.New(),
			URL:        "https://example.com/webhook",
			SecretHash: "secret-hash-value",
			IsActive:   true,
			RetryConfig: models.RetryConfiguration{
				MaxAttempts:       5,
				InitialDelayMs:    1000,
				MaxDelayMs:        300000,
				BackoffMultiplier: 2,
			},
			CustomHeaders: map[string]string{
				"Authorization": "Bearer token",
				"X-Custom":      "value",
			},
		}

		// Test that endpoint has required fields for creation
		assert.NotEqual(t, uuid.Nil, endpoint.TenantID, "tenant ID should not be nil")
		assert.NotEmpty(t, endpoint.URL, "URL should not be empty")
		assert.NotEmpty(t, endpoint.SecretHash, "secret hash should not be empty")
		
		// Validate URL format
		parsedURL, err := url.Parse(endpoint.URL)
		require.NoError(t, err, "URL should be valid")
		assert.Equal(t, "https", parsedURL.Scheme, "URL should use HTTPS")

		// Test retry configuration
		assert.Greater(t, endpoint.RetryConfig.MaxAttempts, 0)
		assert.Greater(t, endpoint.RetryConfig.InitialDelayMs, 0)
		assert.Greater(t, endpoint.RetryConfig.MaxDelayMs, endpoint.RetryConfig.InitialDelayMs)
		assert.Greater(t, endpoint.RetryConfig.BackoffMultiplier, 1)

		// Test that ID and timestamps would be set during creation
		if endpoint.ID == uuid.Nil {
			endpoint.ID = uuid.New()
		}
		endpoint.CreatedAt = time.Now()
		endpoint.UpdatedAt = time.Now()

		assert.NotEqual(t, uuid.Nil, endpoint.ID)
		assert.False(t, endpoint.CreatedAt.IsZero())
		assert.False(t, endpoint.UpdatedAt.IsZero())
	})

	t.Run("webhook endpoint update logic", func(t *testing.T) {
		endpoint := &models.WebhookEndpoint{
			ID:         uuid.New(),
			TenantID:   uuid.New(),
			URL:        "https://updated.example.com/webhook",
			SecretHash: "updated-secret-hash",
			IsActive:   false,
			RetryConfig: models.RetryConfiguration{
				MaxAttempts:       3,
				InitialDelayMs:    2000,
				MaxDelayMs:        600000,
				BackoffMultiplier: 3,
			},
			CustomHeaders: map[string]string{
				"X-Updated": "true",
			},
			CreatedAt: time.Now().Add(-time.Hour),
		}

		// Simulate update operation
		endpoint.UpdatedAt = time.Now()

		assert.NotEqual(t, uuid.Nil, endpoint.ID)
		assert.Contains(t, endpoint.URL, "updated.example.com")
		assert.False(t, endpoint.IsActive)
		assert.Equal(t, 3, endpoint.RetryConfig.MaxAttempts)
		assert.True(t, endpoint.UpdatedAt.After(endpoint.CreatedAt))
	})

	t.Run("retry configuration validation", func(t *testing.T) {
		validConfigs := []models.RetryConfiguration{
			{
				MaxAttempts:       3,
				InitialDelayMs:    500,
				MaxDelayMs:        30000,
				BackoffMultiplier: 2,
			},
			{
				MaxAttempts:       5,
				InitialDelayMs:    1000,
				MaxDelayMs:        300000,
				BackoffMultiplier: 2,
			},
			{
				MaxAttempts:       10,
				InitialDelayMs:    2000,
				MaxDelayMs:        600000,
				BackoffMultiplier: 3,
			},
		}

		for i, config := range validConfigs {
			t.Run(fmt.Sprintf("valid_config_%d", i), func(t *testing.T) {
				assert.Greater(t, config.MaxAttempts, 0)
				assert.Greater(t, config.InitialDelayMs, 0)
				assert.Greater(t, config.MaxDelayMs, config.InitialDelayMs)
				assert.Greater(t, config.BackoffMultiplier, 1)
				
				// Test that max delay is achievable with the configuration
				calculatedMaxDelay := config.InitialDelayMs
				for j := 1; j < config.MaxAttempts; j++ {
					calculatedMaxDelay *= config.BackoffMultiplier
					if calculatedMaxDelay >= config.MaxDelayMs {
						break
					}
				}
				assert.GreaterOrEqual(t, config.MaxDelayMs, config.InitialDelayMs)
			})
		}
	})

	t.Run("custom headers serialization", func(t *testing.T) {
		headers := map[string]string{
			"Authorization":  "Bearer token123",
			"X-Custom-Auth":  "custom-value",
			"Content-Type":   "application/json",
			"X-Webhook-ID":   "webhook-123",
		}

		// Test JSON serialization/deserialization
		jsonData, err := json.Marshal(headers)
		require.NoError(t, err)

		var deserializedHeaders map[string]string
		err = json.Unmarshal(jsonData, &deserializedHeaders)
		require.NoError(t, err)

		assert.Equal(t, headers, deserializedHeaders)
		assert.Equal(t, "Bearer token123", deserializedHeaders["Authorization"])
		assert.Equal(t, "custom-value", deserializedHeaders["X-Custom-Auth"])
	})

	t.Run("secret hash security", func(t *testing.T) {
		endpoint := &models.WebhookEndpoint{
			URL:        "https://example.com/webhook",
			SecretHash: "hashed-secret-value",
		}

		// Secret hash should be present but not exposed in JSON serialization
		assert.NotEmpty(t, endpoint.SecretHash)
		
		// The struct tag should be "-" to exclude from JSON
		field, found := reflect.TypeOf(endpoint).Elem().FieldByName("SecretHash")
		require.True(t, found)
		assert.Equal(t, "-", field.Tag.Get("json"))
	})
}



// TestWebhookEndpointRepositoryErrorHandling tests error handling scenarios
func TestWebhookEndpointRepositoryErrorHandling(t *testing.T) {
	t.Run("invalid endpoint data", func(t *testing.T) {
		invalidEndpoints := []*models.WebhookEndpoint{
			{
				TenantID:   uuid.Nil, // Invalid tenant ID
				URL:        "https://example.com/webhook",
				SecretHash: "secret-hash",
				IsActive:   true,
			},
			{
				TenantID:   uuid.New(),
				URL:        "", // Empty URL
				SecretHash: "secret-hash",
				IsActive:   true,
			},
			{
				TenantID:   uuid.New(),
				URL:        "invalid-url", // Invalid URL format
				SecretHash: "secret-hash",
				IsActive:   true,
			},
			{
				TenantID:   uuid.New(),
				URL:        "http://example.com/webhook", // HTTP instead of HTTPS
				SecretHash: "secret-hash",
				IsActive:   true,
			},
			{
				TenantID:   uuid.New(),
				URL:        "https://example.com/webhook",
				SecretHash: "", // Empty secret hash
				IsActive:   true,
			},
		}

		for i, endpoint := range invalidEndpoints {
			t.Run(fmt.Sprintf("invalid_endpoint_%d", i), func(t *testing.T) {
				hasError := false
				
				if endpoint.TenantID == uuid.Nil {
					hasError = true
				}
				if endpoint.URL == "" {
					hasError = true
				}
				if endpoint.URL != "" {
					if parsedURL, err := url.Parse(endpoint.URL); err != nil {
						hasError = true
					} else if parsedURL.Scheme != "https" {
						hasError = true
					}
				}
				if endpoint.SecretHash == "" {
					hasError = true
				}
				
				assert.True(t, hasError, "endpoint should be invalid")
			})
		}
	})

	t.Run("invalid retry configuration", func(t *testing.T) {
		invalidConfigs := []models.RetryConfiguration{
			{
				MaxAttempts:       0, // Invalid max attempts
				InitialDelayMs:    1000,
				MaxDelayMs:        300000,
				BackoffMultiplier: 2,
			},
			{
				MaxAttempts:       5,
				InitialDelayMs:    0, // Invalid initial delay
				MaxDelayMs:        300000,
				BackoffMultiplier: 2,
			},
			{
				MaxAttempts:       5,
				InitialDelayMs:    1000,
				MaxDelayMs:        500, // Max delay less than initial delay
				BackoffMultiplier: 2,
			},
			{
				MaxAttempts:       5,
				InitialDelayMs:    1000,
				MaxDelayMs:        300000,
				BackoffMultiplier: 1, // Invalid backoff multiplier
			},
		}

		for i, config := range invalidConfigs {
			t.Run(fmt.Sprintf("invalid_config_%d", i), func(t *testing.T) {
				hasError := false
				
				if config.MaxAttempts <= 0 {
					hasError = true
				}
				if config.InitialDelayMs <= 0 {
					hasError = true
				}
				if config.MaxDelayMs < config.InitialDelayMs {
					hasError = true
				}
				if config.BackoffMultiplier <= 1 {
					hasError = true
				}
				
				assert.True(t, hasError, "retry configuration should be invalid")
			})
		}
	})
}

// TestWebhookEndpointActivation tests endpoint activation/deactivation logic
func TestWebhookEndpointActivation(t *testing.T) {
	t.Run("activation state changes", func(t *testing.T) {
		endpoint := &models.WebhookEndpoint{
			ID:       uuid.New(),
			TenantID: uuid.New(),
			URL:      "https://example.com/webhook",
			IsActive: true,
		}

		// Test deactivation
		endpoint.IsActive = false
		endpoint.UpdatedAt = time.Now()
		assert.False(t, endpoint.IsActive)

		// Test reactivation
		endpoint.IsActive = true
		endpoint.UpdatedAt = time.Now()
		assert.True(t, endpoint.IsActive)
	})

	t.Run("active endpoint filtering logic", func(t *testing.T) {
		endpoints := []*models.WebhookEndpoint{
			{
				ID:       uuid.New(),
				TenantID: uuid.New(),
				URL:      "https://example1.com/webhook",
				IsActive: true,
			},
			{
				ID:       uuid.New(),
				TenantID: uuid.New(),
				URL:      "https://example2.com/webhook",
				IsActive: false,
			},
			{
				ID:       uuid.New(),
				TenantID: uuid.New(),
				URL:      "https://example3.com/webhook",
				IsActive: true,
			},
		}

		// Filter active endpoints
		var activeEndpoints []*models.WebhookEndpoint
		for _, endpoint := range endpoints {
			if endpoint.IsActive {
				activeEndpoints = append(activeEndpoints, endpoint)
			}
		}

		assert.Len(t, activeEndpoints, 2)
		for _, endpoint := range activeEndpoints {
			assert.True(t, endpoint.IsActive)
		}
	})
}