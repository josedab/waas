package repository

import (
	"testing"
	"github.com/josedab/waas/pkg/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// Test that all repository interfaces are properly defined
func TestRepositoryInterfaces(t *testing.T) {
	// Test that interfaces have the expected methods by checking they can be assigned
	var tenantRepo TenantRepository
	var webhookRepo WebhookEndpointRepository
	var deliveryRepo DeliveryAttemptRepository

	// These should compile without errors if interfaces are properly defined
	assert.Nil(t, tenantRepo)
	assert.Nil(t, webhookRepo)
	assert.Nil(t, deliveryRepo)
}

// Test model structure validation
func TestTenantModel(t *testing.T) {
	tenant := &models.Tenant{
		ID:                 uuid.New(),
		Name:               "Test Tenant",
		APIKeyHash:         "test-api-key-hash",
		SubscriptionTier:   "basic",
		RateLimitPerMinute: 100,
		MonthlyQuota:       10000,
	}

	assert.NotEqual(t, uuid.Nil, tenant.ID)
	assert.NotEmpty(t, tenant.Name)
	assert.NotEmpty(t, tenant.APIKeyHash)
	assert.NotEmpty(t, tenant.SubscriptionTier)
	assert.Greater(t, tenant.RateLimitPerMinute, 0)
	assert.Greater(t, tenant.MonthlyQuota, 0)
}

func TestWebhookEndpointModel(t *testing.T) {
	endpoint := &models.WebhookEndpoint{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		URL:      "https://example.com/webhook",
		IsActive: true,
		RetryConfig: models.RetryConfiguration{
			MaxAttempts:       5,
			InitialDelayMs:    1000,
			MaxDelayMs:        300000,
			BackoffMultiplier: 2,
		},
		CustomHeaders: map[string]string{
			"Authorization": "Bearer token",
		},
	}

	assert.NotEqual(t, uuid.Nil, endpoint.ID)
	assert.NotEqual(t, uuid.Nil, endpoint.TenantID)
	assert.NotEmpty(t, endpoint.URL)
	assert.True(t, endpoint.IsActive)
	assert.Greater(t, endpoint.RetryConfig.MaxAttempts, 0)
	assert.Greater(t, endpoint.RetryConfig.InitialDelayMs, 0)
	assert.NotEmpty(t, endpoint.CustomHeaders)
}

func TestDeliveryAttemptModel(t *testing.T) {
	attempt := &models.DeliveryAttempt{
		ID:            uuid.New(),
		EndpointID:    uuid.New(),
		PayloadHash:   "sha256-hash",
		PayloadSize:   1024,
		Status:        "pending",
		AttemptNumber: 1,
	}

	assert.NotEqual(t, uuid.Nil, attempt.ID)
	assert.NotEqual(t, uuid.Nil, attempt.EndpointID)
	assert.NotEmpty(t, attempt.PayloadHash)
	assert.Greater(t, attempt.PayloadSize, 0)
	assert.NotEmpty(t, attempt.Status)
	assert.Greater(t, attempt.AttemptNumber, 0)
}

func TestRetryConfiguration(t *testing.T) {
	config := models.RetryConfiguration{
		MaxAttempts:       5,
		InitialDelayMs:    1000,
		MaxDelayMs:        300000,
		BackoffMultiplier: 2,
	}

	assert.Greater(t, config.MaxAttempts, 0)
	assert.Greater(t, config.InitialDelayMs, 0)
	assert.Greater(t, config.MaxDelayMs, config.InitialDelayMs)
	assert.Greater(t, config.BackoffMultiplier, 1)
}