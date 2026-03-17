package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestValidateTenant(t *testing.T) {
	t.Parallel()
	t.Run("valid tenant", func(t *testing.T) {
		tenant := &Tenant{
			Name:               "Valid Tenant",
			APIKeyHash:         "valid-hash",
			SubscriptionTier:   "basic",
			RateLimitPerMinute: 100,
			MonthlyQuota:       10000,
		}

		err := ValidateTenant(tenant)
		assert.NoError(t, err)
	})

	t.Run("nil tenant", func(t *testing.T) {
		err := ValidateTenant(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("empty name", func(t *testing.T) {
		tenant := &Tenant{
			Name:               "",
			APIKeyHash:         "valid-hash",
			SubscriptionTier:   "basic",
			RateLimitPerMinute: 100,
			MonthlyQuota:       10000,
		}

		err := ValidateTenant(tenant)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name cannot be empty")
	})

	t.Run("empty API key hash", func(t *testing.T) {
		tenant := &Tenant{
			Name:               "Valid Tenant",
			APIKeyHash:         "",
			SubscriptionTier:   "basic",
			RateLimitPerMinute: 100,
			MonthlyQuota:       10000,
		}

		err := ValidateTenant(tenant)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "API key hash cannot be empty")
	})

	t.Run("invalid subscription tier", func(t *testing.T) {
		tenant := &Tenant{
			Name:               "Valid Tenant",
			APIKeyHash:         "valid-hash",
			SubscriptionTier:   "invalid",
			RateLimitPerMinute: 100,
			MonthlyQuota:       10000,
		}

		err := ValidateTenant(tenant)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid subscription tier")
	})

	t.Run("invalid rate limit", func(t *testing.T) {
		tenant := &Tenant{
			Name:               "Valid Tenant",
			APIKeyHash:         "valid-hash",
			SubscriptionTier:   "basic",
			RateLimitPerMinute: 0,
			MonthlyQuota:       10000,
		}

		err := ValidateTenant(tenant)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rate limit per minute must be positive")
	})

	t.Run("invalid monthly quota", func(t *testing.T) {
		tenant := &Tenant{
			Name:               "Valid Tenant",
			APIKeyHash:         "valid-hash",
			SubscriptionTier:   "basic",
			RateLimitPerMinute: 100,
			MonthlyQuota:       0,
		}

		err := ValidateTenant(tenant)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "monthly quota must be positive")
	})
}

func TestValidateWebhookEndpoint(t *testing.T) {
	t.Parallel()
	t.Run("valid endpoint", func(t *testing.T) {
		endpoint := &WebhookEndpoint{
			TenantID:   uuid.New(),
			URL:        "https://example.com/webhook",
			SecretHash: "secret-hash",
			IsActive:   true,
			RetryConfig: RetryConfiguration{
				MaxAttempts:       5,
				InitialDelayMs:    1000,
				MaxDelayMs:        300000,
				BackoffMultiplier: 2,
			},
		}

		err := ValidateWebhookEndpoint(endpoint)
		assert.NoError(t, err)
	})

	t.Run("nil endpoint", func(t *testing.T) {
		err := ValidateWebhookEndpoint(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("nil tenant ID", func(t *testing.T) {
		endpoint := &WebhookEndpoint{
			TenantID:   uuid.Nil,
			URL:        "https://example.com/webhook",
			SecretHash: "secret-hash",
		}

		err := ValidateWebhookEndpoint(endpoint)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tenant ID cannot be nil")
	})

	t.Run("empty URL", func(t *testing.T) {
		endpoint := &WebhookEndpoint{
			TenantID:   uuid.New(),
			URL:        "",
			SecretHash: "secret-hash",
		}

		err := ValidateWebhookEndpoint(endpoint)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "URL cannot be empty")
	})

	t.Run("invalid URL format", func(t *testing.T) {
		endpoint := &WebhookEndpoint{
			TenantID:   uuid.New(),
			URL:        "invalid-url",
			SecretHash: "secret-hash",
		}

		err := ValidateWebhookEndpoint(endpoint)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must use HTTPS")
	})

	t.Run("non-HTTPS URL", func(t *testing.T) {
		endpoint := &WebhookEndpoint{
			TenantID:   uuid.New(),
			URL:        "http://example.com/webhook",
			SecretHash: "secret-hash",
		}

		err := ValidateWebhookEndpoint(endpoint)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must use HTTPS")
	})

	t.Run("empty secret hash", func(t *testing.T) {
		endpoint := &WebhookEndpoint{
			TenantID:   uuid.New(),
			URL:        "https://example.com/webhook",
			SecretHash: "",
		}

		err := ValidateWebhookEndpoint(endpoint)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "secret hash cannot be empty")
	})
}

func TestValidateRetryConfiguration(t *testing.T) {
	t.Parallel()
	t.Run("valid configuration", func(t *testing.T) {
		config := &RetryConfiguration{
			MaxAttempts:       5,
			InitialDelayMs:    1000,
			MaxDelayMs:        300000,
			BackoffMultiplier: 2,
		}

		err := ValidateRetryConfiguration(config)
		assert.NoError(t, err)
	})

	t.Run("nil configuration", func(t *testing.T) {
		err := ValidateRetryConfiguration(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("invalid max attempts", func(t *testing.T) {
		config := &RetryConfiguration{
			MaxAttempts:       0,
			InitialDelayMs:    1000,
			MaxDelayMs:        300000,
			BackoffMultiplier: 2,
		}

		err := ValidateRetryConfiguration(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max attempts must be positive")
	})

	t.Run("invalid initial delay", func(t *testing.T) {
		config := &RetryConfiguration{
			MaxAttempts:       5,
			InitialDelayMs:    0,
			MaxDelayMs:        300000,
			BackoffMultiplier: 2,
		}

		err := ValidateRetryConfiguration(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "initial delay must be positive")
	})

	t.Run("max delay less than initial delay", func(t *testing.T) {
		config := &RetryConfiguration{
			MaxAttempts:       5,
			InitialDelayMs:    1000,
			MaxDelayMs:        500,
			BackoffMultiplier: 2,
		}

		err := ValidateRetryConfiguration(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max delay")
		assert.Contains(t, err.Error(), "must be greater than or equal to initial delay")
	})

	t.Run("invalid backoff multiplier", func(t *testing.T) {
		config := &RetryConfiguration{
			MaxAttempts:       5,
			InitialDelayMs:    1000,
			MaxDelayMs:        300000,
			BackoffMultiplier: 1,
		}

		err := ValidateRetryConfiguration(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "backoff multiplier must be greater than 1")
	})
}

func TestValidateDeliveryAttempt(t *testing.T) {
	t.Parallel()
	t.Run("valid attempt", func(t *testing.T) {
		attempt := &DeliveryAttempt{
			EndpointID:    uuid.New(),
			PayloadHash:   "sha256-abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			PayloadSize:   1024,
			Status:        "pending",
			AttemptNumber: 1,
		}

		err := ValidateDeliveryAttempt(attempt)
		assert.NoError(t, err)
	})

	t.Run("nil attempt", func(t *testing.T) {
		err := ValidateDeliveryAttempt(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("nil endpoint ID", func(t *testing.T) {
		attempt := &DeliveryAttempt{
			EndpointID:    uuid.Nil,
			PayloadHash:   "sha256-abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			PayloadSize:   1024,
			Status:        "pending",
			AttemptNumber: 1,
		}

		err := ValidateDeliveryAttempt(attempt)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "endpoint ID cannot be nil")
	})

	t.Run("empty payload hash", func(t *testing.T) {
		attempt := &DeliveryAttempt{
			EndpointID:    uuid.New(),
			PayloadHash:   "",
			PayloadSize:   1024,
			Status:        "pending",
			AttemptNumber: 1,
		}

		err := ValidateDeliveryAttempt(attempt)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "payload hash cannot be empty")
	})

	t.Run("invalid payload hash format", func(t *testing.T) {
		attempt := &DeliveryAttempt{
			EndpointID:    uuid.New(),
			PayloadHash:   "invalid-hash",
			PayloadSize:   1024,
			Status:        "pending",
			AttemptNumber: 1,
		}

		err := ValidateDeliveryAttempt(attempt)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid payload hash format")
	})

	t.Run("invalid payload size", func(t *testing.T) {
		attempt := &DeliveryAttempt{
			EndpointID:    uuid.New(),
			PayloadHash:   "sha256-abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			PayloadSize:   0,
			Status:        "pending",
			AttemptNumber: 1,
		}

		err := ValidateDeliveryAttempt(attempt)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "payload size must be positive")
	})

	t.Run("invalid status", func(t *testing.T) {
		attempt := &DeliveryAttempt{
			EndpointID:    uuid.New(),
			PayloadHash:   "sha256-abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			PayloadSize:   1024,
			Status:        "invalid",
			AttemptNumber: 1,
		}

		err := ValidateDeliveryAttempt(attempt)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status")
	})

	t.Run("invalid attempt number", func(t *testing.T) {
		attempt := &DeliveryAttempt{
			EndpointID:    uuid.New(),
			PayloadHash:   "sha256-abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			PayloadSize:   1024,
			Status:        "pending",
			AttemptNumber: 0,
		}

		err := ValidateDeliveryAttempt(attempt)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "attempt number must be positive")
	})
}

func TestContainsHelper(t *testing.T) {
	t.Parallel()
	t.Run("contains existing item", func(t *testing.T) {
		slice := []string{"a", "b", "c"}
		assert.True(t, contains(slice, "b"))
	})

	t.Run("does not contain missing item", func(t *testing.T) {
		slice := []string{"a", "b", "c"}
		assert.False(t, contains(slice, "d"))
	})

	t.Run("empty slice", func(t *testing.T) {
		slice := []string{}
		assert.False(t, contains(slice, "a"))
	})
}

func TestValidateTenant_AllTiers(t *testing.T) {
	t.Parallel()
	validTiers := []string{"free", "basic", "premium", "pro", "enterprise"}
	for _, tier := range validTiers {
		t.Run("valid_tier_"+tier, func(t *testing.T) {
			tenant := &Tenant{
				Name:               "Test",
				APIKeyHash:         "hash",
				SubscriptionTier:   tier,
				RateLimitPerMinute: 10,
				MonthlyQuota:       100,
			}
			assert.NoError(t, ValidateTenant(tenant))
		})
	}

	invalidTiers := []string{"", "gold", "unlimited", "FREE", "BASIC"}
	for _, tier := range invalidTiers {
		t.Run("invalid_tier_"+tier, func(t *testing.T) {
			tenant := &Tenant{
				Name:               "Test",
				APIKeyHash:         "hash",
				SubscriptionTier:   tier,
				RateLimitPerMinute: 10,
				MonthlyQuota:       100,
			}
			assert.Error(t, ValidateTenant(tenant))
		})
	}
}

func TestValidateRetryConfiguration_Bounds(t *testing.T) {
	t.Parallel()

	t.Run("initial_delay_exceeds_max_delay", func(t *testing.T) {
		cfg := &RetryConfiguration{
			MaxAttempts:       3,
			InitialDelayMs:    5000,
			MaxDelayMs:        1000,
			BackoffMultiplier: 2,
		}
		err := ValidateRetryConfiguration(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max delay")
	})

	t.Run("equal_delays_valid", func(t *testing.T) {
		cfg := &RetryConfiguration{
			MaxAttempts:       3,
			InitialDelayMs:    1000,
			MaxDelayMs:        1000,
			BackoffMultiplier: 2,
		}
		assert.NoError(t, ValidateRetryConfiguration(cfg))
	})

	t.Run("backoff_multiplier_must_exceed_one", func(t *testing.T) {
		cfg := &RetryConfiguration{
			MaxAttempts:       3,
			InitialDelayMs:    100,
			MaxDelayMs:        10000,
			BackoffMultiplier: 1,
		}
		assert.Error(t, ValidateRetryConfiguration(cfg))
	})
}

func TestValidateDeliveryAttempt_HashFormat(t *testing.T) {
	t.Parallel()

	t.Run("valid_hash", func(t *testing.T) {
		a := &DeliveryAttempt{
			EndpointID:    uuid.New(),
			PayloadHash:   "sha256-" + "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
			PayloadSize:   42,
			Status:        "pending",
			AttemptNumber: 1,
		}
		assert.NoError(t, ValidateDeliveryAttempt(a))
	})

	t.Run("wrong_prefix", func(t *testing.T) {
		a := &DeliveryAttempt{
			EndpointID:    uuid.New(),
			PayloadHash:   "md5-abc123",
			PayloadSize:   42,
			Status:        "pending",
			AttemptNumber: 1,
		}
		assert.Error(t, ValidateDeliveryAttempt(a))
	})

	t.Run("invalid_status", func(t *testing.T) {
		a := &DeliveryAttempt{
			EndpointID:    uuid.New(),
			PayloadHash:   "sha256-" + "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
			PayloadSize:   42,
			Status:        "cancelled",
			AttemptNumber: 1,
		}
		assert.Error(t, ValidateDeliveryAttempt(a))
	})
}
