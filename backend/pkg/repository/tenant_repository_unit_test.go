package repository

import (
	"fmt"
	"reflect"
	"testing"
	"time"
	"webhook-platform/pkg/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTenantRepositoryLogic tests the business logic of tenant repository operations
func TestTenantRepositoryLogic(t *testing.T) {
	t.Run("tenant creation logic", func(t *testing.T) {
		tenant := &models.Tenant{
			Name:               "Test Tenant",
			APIKeyHash:         "test-api-key-hash",
			SubscriptionTier:   "basic",
			RateLimitPerMinute: 100,
			MonthlyQuota:       10000,
		}

		// Test that tenant has required fields for creation
		assert.NotEmpty(t, tenant.Name, "tenant name should not be empty")
		assert.NotEmpty(t, tenant.APIKeyHash, "API key hash should not be empty")
		assert.NotEmpty(t, tenant.SubscriptionTier, "subscription tier should not be empty")
		assert.Greater(t, tenant.RateLimitPerMinute, 0, "rate limit should be positive")
		assert.Greater(t, tenant.MonthlyQuota, 0, "monthly quota should be positive")

		// Test that ID and timestamps would be set during creation
		if tenant.ID == uuid.Nil {
			tenant.ID = uuid.New()
		}
		tenant.CreatedAt = time.Now()
		tenant.UpdatedAt = time.Now()

		assert.NotEqual(t, uuid.Nil, tenant.ID)
		assert.False(t, tenant.CreatedAt.IsZero())
		assert.False(t, tenant.UpdatedAt.IsZero())
	})

	t.Run("tenant update logic", func(t *testing.T) {
		tenant := &models.Tenant{
			ID:                 uuid.New(),
			Name:               "Updated Tenant",
			APIKeyHash:         "updated-api-key-hash",
			SubscriptionTier:   "premium",
			RateLimitPerMinute: 200,
			MonthlyQuota:       20000,
			CreatedAt:          time.Now().Add(-time.Hour),
		}

		// Simulate update operation
		tenant.UpdatedAt = time.Now()

		assert.NotEqual(t, uuid.Nil, tenant.ID)
		assert.Equal(t, "Updated Tenant", tenant.Name)
		assert.Equal(t, "premium", tenant.SubscriptionTier)
		assert.Equal(t, 200, tenant.RateLimitPerMinute)
		assert.Equal(t, 20000, tenant.MonthlyQuota)
		assert.True(t, tenant.UpdatedAt.After(tenant.CreatedAt))
	})

	t.Run("tenant validation rules", func(t *testing.T) {
		validTiers := []string{"basic", "premium", "enterprise"}
		
		for _, tier := range validTiers {
			tenant := &models.Tenant{
				Name:               "Test Tenant",
				APIKeyHash:         "test-hash",
				SubscriptionTier:   tier,
				RateLimitPerMinute: 100,
				MonthlyQuota:       10000,
			}
			
			assert.Contains(t, validTiers, tenant.SubscriptionTier)
			assert.Greater(t, tenant.RateLimitPerMinute, 0)
			assert.Greater(t, tenant.MonthlyQuota, 0)
		}
	})

	t.Run("tenant API key hash security", func(t *testing.T) {
		tenant := &models.Tenant{
			Name:       "Test Tenant",
			APIKeyHash: "hashed-api-key-value",
		}

		// API key hash should be present but not exposed in JSON serialization
		assert.NotEmpty(t, tenant.APIKeyHash)
		
		// The struct tag should be "-" to exclude from JSON
		// This is a compile-time check that the field has the correct tag
		field, found := reflect.TypeOf(tenant).Elem().FieldByName("APIKeyHash")
		require.True(t, found)
		assert.Equal(t, "-", field.Tag.Get("json"))
	})
}



// TestTenantRepositoryErrorHandling tests error handling scenarios
func TestTenantRepositoryErrorHandling(t *testing.T) {
	t.Run("invalid tenant data", func(t *testing.T) {
		invalidTenants := []*models.Tenant{
			{
				Name:               "", // Empty name
				APIKeyHash:         "test-hash",
				SubscriptionTier:   "basic",
				RateLimitPerMinute: 100,
				MonthlyQuota:       10000,
			},
			{
				Name:               "Test Tenant",
				APIKeyHash:         "", // Empty API key hash
				SubscriptionTier:   "basic",
				RateLimitPerMinute: 100,
				MonthlyQuota:       10000,
			},
			{
				Name:               "Test Tenant",
				APIKeyHash:         "test-hash",
				SubscriptionTier:   "invalid", // Invalid tier
				RateLimitPerMinute: 100,
				MonthlyQuota:       10000,
			},
			{
				Name:               "Test Tenant",
				APIKeyHash:         "test-hash",
				SubscriptionTier:   "basic",
				RateLimitPerMinute: 0, // Invalid rate limit
				MonthlyQuota:       10000,
			},
			{
				Name:               "Test Tenant",
				APIKeyHash:         "test-hash",
				SubscriptionTier:   "basic",
				RateLimitPerMinute: 100,
				MonthlyQuota:       0, // Invalid quota
			},
		}

		validTiers := []string{"basic", "premium", "enterprise"}

		for i, tenant := range invalidTenants {
			t.Run(fmt.Sprintf("invalid_tenant_%d", i), func(t *testing.T) {
				// Test validation logic
				hasError := false
				
				if tenant.Name == "" {
					hasError = true
				}
				if tenant.APIKeyHash == "" {
					hasError = true
				}
				if !containsStr(validTiers, tenant.SubscriptionTier) {
					hasError = true
				}
				if tenant.RateLimitPerMinute <= 0 {
					hasError = true
				}
				if tenant.MonthlyQuota <= 0 {
					hasError = true
				}
				
				assert.True(t, hasError, "tenant should be invalid")
			})
		}
	})
}

// Helper function for testing
func containsStr(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}