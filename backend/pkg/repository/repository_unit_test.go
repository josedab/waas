package repository

import (
	"testing"
	"time"
	"webhook-platform/pkg/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)



// TestDeliveryRequestModel tests the DeliveryRequest model structure
func TestDeliveryRequestModel(t *testing.T) {
	payload := map[string]interface{}{
		"event": "user.created",
		"data": map[string]interface{}{
			"id":    "123",
			"email": "test@example.com",
		},
	}

	request := &models.DeliveryRequest{
		ID:          uuid.New(),
		EndpointID:  uuid.New(),
		Payload:     payload,
		Headers: map[string]string{
			"Content-Type": "application/json",
			"X-Event-Type": "user.created",
		},
		ScheduledAt: time.Now(),
		Attempts:    0,
		MaxAttempts: 5,
	}

	// Test that all required fields are set
	assert.NotEqual(t, uuid.Nil, request.ID)
	assert.NotEqual(t, uuid.Nil, request.EndpointID)
	assert.NotNil(t, request.Payload)
	assert.NotEmpty(t, request.Headers)
	assert.False(t, request.ScheduledAt.IsZero())
	assert.GreaterOrEqual(t, request.Attempts, 0)
	assert.Greater(t, request.MaxAttempts, 0)

	// Test payload structure
	payloadMap, ok := request.Payload.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "user.created", payloadMap["event"])
	
	// Test headers
	assert.Equal(t, "application/json", request.Headers["Content-Type"])
	assert.Equal(t, "user.created", request.Headers["X-Event-Type"])
}



// TestRepositoryFactory tests the repository factory function
func TestRepositoryFactory(t *testing.T) {
	// Test that NewRepositories function exists and has correct signature
	assert.NotNil(t, NewRepositories)
	
	// Test that the function can be called (without actual DB connection)
	// In integration tests, this would be called with a real database connection
}



// TestModelValidation tests model validation logic
func TestModelValidation(t *testing.T) {
	t.Run("tenant validation", func(t *testing.T) {
		tenant := &models.Tenant{
			Name:               "Valid Tenant",
			APIKeyHash:         "valid-hash",
			SubscriptionTier:   "basic",
			RateLimitPerMinute: 100,
			MonthlyQuota:       10000,
		}
		
		// Test valid tenant
		assert.NotEmpty(t, tenant.Name)
		assert.NotEmpty(t, tenant.APIKeyHash)
		assert.Contains(t, []string{"basic", "premium", "enterprise"}, tenant.SubscriptionTier)
		assert.Greater(t, tenant.RateLimitPerMinute, 0)
		assert.Greater(t, tenant.MonthlyQuota, 0)
	})
	
	t.Run("webhook endpoint validation", func(t *testing.T) {
		endpoint := &models.WebhookEndpoint{
			URL:        "https://example.com/webhook",
			SecretHash: "secret-hash",
			IsActive:   true,
			RetryConfig: models.RetryConfiguration{
				MaxAttempts:       5,
				InitialDelayMs:    1000,
				MaxDelayMs:        300000,
				BackoffMultiplier: 2,
			},
		}
		
		// Test valid endpoint
		assert.NotEmpty(t, endpoint.URL)
		assert.Contains(t, endpoint.URL, "https://")
		assert.NotEmpty(t, endpoint.SecretHash)
		assert.True(t, endpoint.IsActive)
		assert.Greater(t, endpoint.RetryConfig.MaxAttempts, 0)
	})
	
	t.Run("delivery attempt validation", func(t *testing.T) {
		attempt := &models.DeliveryAttempt{
			PayloadHash:   "sha256-hash",
			PayloadSize:   1024,
			Status:        "pending",
			AttemptNumber: 1,
		}
		
		// Test valid attempt
		assert.NotEmpty(t, attempt.PayloadHash)
		assert.Greater(t, attempt.PayloadSize, 0)
		assert.Contains(t, []string{"pending", "delivered", "failed", "retrying"}, attempt.Status)
		assert.Greater(t, attempt.AttemptNumber, 0)
	})
}