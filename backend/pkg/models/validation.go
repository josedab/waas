package models

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

// ValidateTenant validates a tenant model before database operations
func ValidateTenant(tenant *Tenant) error {
	if tenant == nil {
		return fmt.Errorf("tenant cannot be nil")
	}

	if strings.TrimSpace(tenant.Name) == "" {
		return fmt.Errorf("tenant name cannot be empty")
	}

	if strings.TrimSpace(tenant.APIKeyHash) == "" {
		return fmt.Errorf("API key hash cannot be empty")
	}

	validTiers := []string{"free", "basic", "premium", "pro", "enterprise"}
	if !contains(validTiers, tenant.SubscriptionTier) {
		return fmt.Errorf("invalid subscription tier: %s, must be one of %v", tenant.SubscriptionTier, validTiers)
	}

	if tenant.RateLimitPerMinute <= 0 {
		return fmt.Errorf("rate limit per minute must be positive, got: %d", tenant.RateLimitPerMinute)
	}

	if tenant.MonthlyQuota <= 0 {
		return fmt.Errorf("monthly quota must be positive, got: %d", tenant.MonthlyQuota)
	}

	return nil
}

// ValidateWebhookEndpoint validates a webhook endpoint model before database operations
func ValidateWebhookEndpoint(endpoint *WebhookEndpoint) error {
	if endpoint == nil {
		return fmt.Errorf("webhook endpoint cannot be nil")
	}

	if endpoint.TenantID == uuid.Nil {
		return fmt.Errorf("tenant ID cannot be nil")
	}

	if strings.TrimSpace(endpoint.URL) == "" {
		return fmt.Errorf("webhook URL cannot be empty")
	}

	// Validate URL format
	parsedURL, err := url.Parse(endpoint.URL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	if parsedURL.Scheme != "https" {
		return fmt.Errorf("webhook URL must use HTTPS, got: %s", parsedURL.Scheme)
	}

	if strings.TrimSpace(endpoint.SecretHash) == "" {
		return fmt.Errorf("secret hash cannot be empty")
	}

	// Validate retry configuration
	if err := ValidateRetryConfiguration(&endpoint.RetryConfig); err != nil {
		return fmt.Errorf("invalid retry configuration: %w", err)
	}

	return nil
}

// ValidateRetryConfiguration validates a retry configuration
func ValidateRetryConfiguration(config *RetryConfiguration) error {
	if config == nil {
		return fmt.Errorf("retry configuration cannot be nil")
	}

	if config.MaxAttempts <= 0 {
		return fmt.Errorf("max attempts must be positive, got: %d", config.MaxAttempts)
	}

	if config.InitialDelayMs <= 0 {
		return fmt.Errorf("initial delay must be positive, got: %d", config.InitialDelayMs)
	}

	if config.MaxDelayMs < config.InitialDelayMs {
		return fmt.Errorf("max delay (%d) must be greater than or equal to initial delay (%d)", config.MaxDelayMs, config.InitialDelayMs)
	}

	if config.BackoffMultiplier <= 1 {
		return fmt.Errorf("backoff multiplier must be greater than 1, got: %d", config.BackoffMultiplier)
	}

	return nil
}

// ValidateDeliveryAttempt validates a delivery attempt model before database operations
func ValidateDeliveryAttempt(attempt *DeliveryAttempt) error {
	if attempt == nil {
		return fmt.Errorf("delivery attempt cannot be nil")
	}

	if attempt.EndpointID == uuid.Nil {
		return fmt.Errorf("endpoint ID cannot be nil")
	}

	if strings.TrimSpace(attempt.PayloadHash) == "" {
		return fmt.Errorf("payload hash cannot be empty")
	}

	// Validate payload hash format (should be sha256-<hex>)
	if !strings.HasPrefix(attempt.PayloadHash, "sha256-") || len(attempt.PayloadHash) != 71 {
		return fmt.Errorf("invalid payload hash format, expected sha256-<64 hex chars>, got: %s", attempt.PayloadHash)
	}

	if attempt.PayloadSize <= 0 {
		return fmt.Errorf("payload size must be positive, got: %d", attempt.PayloadSize)
	}

	validStatuses := []string{"pending", "processing", "delivered", "failed", "retrying"}
	if !contains(validStatuses, attempt.Status) {
		return fmt.Errorf("invalid status: %s, must be one of %v", attempt.Status, validStatuses)
	}

	if attempt.AttemptNumber <= 0 {
		return fmt.Errorf("attempt number must be positive, got: %d", attempt.AttemptNumber)
	}

	return nil
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
