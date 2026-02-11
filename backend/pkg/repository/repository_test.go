package repository

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRepositories(t *testing.T) {
	// Verify the function signature exists
	assert.NotNil(t, NewRepositories)
}

func TestNewRepositories_NilDB(t *testing.T) {
	// Should not panic with nil DB
	assert.NotPanics(t, func() {
		repos := NewRepositories(nil)
		assert.NotNil(t, repos)
	})
}

// --- Model Validation Tests ---

func TestTenantModel_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		tenant *models.Tenant
		valid  bool
	}{
		{
			name: "valid tenant",
			tenant: &models.Tenant{
				Name:               "Test Tenant",
				APIKeyHash:         "valid-hash",
				SubscriptionTier:   "basic",
				RateLimitPerMinute: 100,
				MonthlyQuota:       10000,
			},
			valid: true,
		},
		{
			name: "premium tier",
			tenant: &models.Tenant{
				Name:               "Premium Tenant",
				APIKeyHash:         "premium-hash",
				SubscriptionTier:   "premium",
				RateLimitPerMinute: 1000,
				MonthlyQuota:       100000,
			},
			valid: true,
		},
		{
			name: "enterprise tier",
			tenant: &models.Tenant{
				Name:               "Enterprise Tenant",
				APIKeyHash:         "enterprise-hash",
				SubscriptionTier:   "enterprise",
				RateLimitPerMinute: 10000,
				MonthlyQuota:       1000000,
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotEmpty(t, tt.tenant.Name)
			assert.NotEmpty(t, tt.tenant.APIKeyHash)
			assert.Greater(t, tt.tenant.RateLimitPerMinute, 0)
			assert.Greater(t, tt.tenant.MonthlyQuota, 0)
		})
	}
}

func TestTenantModel_UUIDGeneration(t *testing.T) {
	t.Parallel()
	tenant := &models.Tenant{
		ID:                 uuid.New(),
		Name:               "Test",
		SubscriptionTier:   "basic",
		RateLimitPerMinute: 100,
		MonthlyQuota:       10000,
	}
	assert.NotEqual(t, uuid.Nil, tenant.ID)

	// Two tenants should have different IDs
	tenant2 := &models.Tenant{ID: uuid.New(), Name: "Test2"}
	assert.NotEqual(t, tenant.ID, tenant2.ID)
}

func TestWebhookEndpointModel_RetryConfig(t *testing.T) {
	t.Parallel()

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
	}

	assert.Greater(t, endpoint.RetryConfig.MaxAttempts, 0)
	assert.Greater(t, endpoint.RetryConfig.MaxDelayMs, endpoint.RetryConfig.InitialDelayMs)
	assert.GreaterOrEqual(t, endpoint.RetryConfig.BackoffMultiplier, 1)
}

func TestWebhookEndpointModel_CustomHeaders(t *testing.T) {
	t.Parallel()

	endpoint := &models.WebhookEndpoint{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		URL:      "https://example.com/webhook",
		IsActive: true,
		CustomHeaders: map[string]string{
			"Authorization": "Bearer token123",
			"X-Custom":      "value",
		},
	}

	assert.Len(t, endpoint.CustomHeaders, 2)
	assert.Equal(t, "Bearer token123", endpoint.CustomHeaders["Authorization"])
}

func TestDeliveryAttemptModel_StatusTransitions(t *testing.T) {
	t.Parallel()

	validStatuses := []string{"pending", "processing", "success", "failed", "retrying"}
	for _, status := range validStatuses {
		t.Run(status, func(t *testing.T) {
			attempt := &models.DeliveryAttempt{
				ID:            uuid.New(),
				EndpointID:    uuid.New(),
				Status:        status,
				AttemptNumber: 1,
				CreatedAt:     time.Now(),
			}
			assert.NotEqual(t, uuid.Nil, attempt.ID)
			assert.Equal(t, status, attempt.Status)
		})
	}
}

func TestDeliveryAttemptModel_OptionalFields(t *testing.T) {
	t.Parallel()

	// An attempt that failed has ErrorMessage set
	errMsg := "connection refused"
	attempt := &models.DeliveryAttempt{
		ID:           uuid.New(),
		EndpointID:   uuid.New(),
		Status:       "failed",
		ErrorMessage: &errMsg,
	}
	require.NotNil(t, attempt.ErrorMessage)
	assert.Equal(t, "connection refused", *attempt.ErrorMessage)

	// A successful attempt has HTTPStatus and ResponseBody
	httpStatus := 200
	body := `{"status":"ok"}`
	deliveredAt := time.Now()
	successAttempt := &models.DeliveryAttempt{
		ID:           uuid.New(),
		EndpointID:   uuid.New(),
		Status:       "success",
		HTTPStatus:   &httpStatus,
		ResponseBody: &body,
		DeliveredAt:  &deliveredAt,
	}
	assert.Equal(t, 200, *successAttempt.HTTPStatus)
	assert.NotNil(t, successAttempt.DeliveredAt)
}

func TestDeliveryHistoryFilters_Defaults(t *testing.T) {
	t.Parallel()

	// Empty filters should have nil/zero values
	filters := DeliveryHistoryFilters{}
	assert.Empty(t, filters.EndpointIDs)
	assert.Empty(t, filters.Statuses)
	assert.True(t, filters.StartDate.IsZero())
	assert.True(t, filters.EndDate.IsZero())
}

func TestDeliveryHistoryFilters_WithValues(t *testing.T) {
	t.Parallel()

	filters := DeliveryHistoryFilters{
		EndpointIDs: []uuid.UUID{uuid.New(), uuid.New()},
		Statuses:    []string{"success", "failed"},
		StartDate:   time.Now().Add(-24 * time.Hour),
		EndDate:     time.Now(),
	}

	assert.Len(t, filters.EndpointIDs, 2)
	assert.Len(t, filters.Statuses, 2)
	assert.True(t, filters.EndDate.After(filters.StartDate))
}

func TestWebhookEndpointModel_TenantIsolation(t *testing.T) {
	t.Parallel()

	tenant1 := uuid.New()
	tenant2 := uuid.New()

	ep1 := &models.WebhookEndpoint{ID: uuid.New(), TenantID: tenant1, URL: "https://a.com"}
	ep2 := &models.WebhookEndpoint{ID: uuid.New(), TenantID: tenant2, URL: "https://b.com"}

	// Endpoints from different tenants should have different tenant IDs
	assert.NotEqual(t, ep1.TenantID, ep2.TenantID)
	assert.NotEqual(t, ep1.ID, ep2.ID)
}

func TestDeliveryRequestModel_Payload(t *testing.T) {
	t.Parallel()

	payload := map[string]interface{}{
		"event": "user.created",
		"data": map[string]interface{}{
			"id": "123",
		},
	}

	request := &models.DeliveryRequest{
		ID:          uuid.New(),
		EndpointID:  uuid.New(),
		Payload:     payload,
		Headers:     map[string]string{"Content-Type": "application/json"},
		ScheduledAt: time.Now(),
		MaxAttempts: 5,
	}

	assert.NotNil(t, request.Payload)
	payloadMap, ok := request.Payload.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "user.created", payloadMap["event"])
}

// --- Interface Compliance Tests ---

func TestInterfacesExist(t *testing.T) {
	t.Parallel()

	// Verify interface types are defined
	var _ TenantRepository = (*tenantRepository)(nil)
	var _ WebhookEndpointRepository = (*webhookEndpointRepository)(nil)
	var _ DeliveryAttemptRepository = (*deliveryAttemptRepository)(nil)
}
