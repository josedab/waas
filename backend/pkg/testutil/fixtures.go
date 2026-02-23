package testutil

import (
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/models"
)

// SampleTenant returns a test tenant with sensible defaults.
// Fields can be overridden after creation.
func SampleTenant() *models.Tenant {
	return &models.Tenant{
		ID:                 uuid.New(),
		Name:               "test-tenant",
		APIKeyHash:         "hash_test_api_key",
		SubscriptionTier:   "pro",
		RateLimitPerMinute: 1000,
		MonthlyQuota:       100000,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
}

// SampleEndpoint returns a test webhook endpoint with sensible defaults.
func SampleEndpoint(tenantID uuid.UUID) *models.WebhookEndpoint {
	return &models.WebhookEndpoint{
		ID:         uuid.New(),
		TenantID:   tenantID,
		URL:        "https://example.com/webhook",
		SecretHash: "hash_test_secret",
		IsActive:   true,
		RetryConfig: models.RetryConfiguration{
			MaxAttempts:       5,
			InitialDelayMs:    1000,
			MaxDelayMs:        60000,
			BackoffMultiplier: 2,
		},
		CustomHeaders: map[string]string{
			"X-Custom": "test",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// SampleDeliveryAttempt returns a test delivery attempt with sensible defaults.
func SampleDeliveryAttempt(endpointID uuid.UUID) *models.DeliveryAttempt {
	status := 200
	body := `{"ok": true}`
	return &models.DeliveryAttempt{
		ID:            uuid.New(),
		EndpointID:    endpointID,
		PayloadHash:   "sha256_test_hash",
		PayloadSize:   256,
		Status:        "delivered",
		HTTPStatus:    &status,
		ResponseBody:  &body,
		AttemptNumber: 1,
		ScheduledAt:   time.Now().Add(-time.Minute),
		CreatedAt:     time.Now(),
	}
}
