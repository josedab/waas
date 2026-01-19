package repository

import (
	"context"
	"webhook-platform/pkg/database"
	"webhook-platform/pkg/models"

	"github.com/google/uuid"
)

// ExampleUsage demonstrates how to use the repositories in a service
func ExampleUsage(db *database.DB) error {
	ctx := context.Background()
	
	// Initialize all repositories
	repos := NewRepositories(db)
	
	// Example: Create a new tenant
	tenant := &models.Tenant{
		Name:               "Example Company",
		APIKeyHash:         "hashed-api-key-123",
		SubscriptionTier:   "premium",
		RateLimitPerMinute: 1000,
		MonthlyQuota:       100000,
	}
	
	if err := repos.Tenant.Create(ctx, tenant); err != nil {
		return err
	}
	
	// Example: Create a webhook endpoint for the tenant
	endpoint := &models.WebhookEndpoint{
		TenantID:   tenant.ID,
		URL:        "https://api.example.com/webhooks",
		SecretHash: "hashed-secret-456",
		IsActive:   true,
		RetryConfig: models.RetryConfiguration{
			MaxAttempts:       5,
			InitialDelayMs:    1000,
			MaxDelayMs:        300000,
			BackoffMultiplier: 2,
		},
		CustomHeaders: map[string]string{
			"Authorization": "Bearer token",
			"Content-Type":  "application/json",
		},
	}
	
	if err := repos.WebhookEndpoint.Create(ctx, endpoint); err != nil {
		return err
	}
	
	// Example: Create a delivery attempt
	attempt := &models.DeliveryAttempt{
		EndpointID:    endpoint.ID,
		PayloadHash:   "sha256-payload-hash",
		PayloadSize:   2048,
		Status:        "pending",
		AttemptNumber: 1,
	}
	
	if err := repos.DeliveryAttempt.Create(ctx, attempt); err != nil {
		return err
	}
	
	// Example: Query operations
	
	// Get tenant by API key
	retrievedTenant, err := repos.Tenant.GetByAPIKeyHash(ctx, tenant.APIKeyHash)
	if err != nil {
		return err
	}
	_ = retrievedTenant
	
	// Get active endpoints for tenant
	activeEndpoints, err := repos.WebhookEndpoint.GetActiveByTenantID(ctx, tenant.ID)
	if err != nil {
		return err
	}
	_ = activeEndpoints
	
	// Get pending deliveries
	pendingDeliveries, err := repos.DeliveryAttempt.GetPendingDeliveries(ctx, 100)
	if err != nil {
		return err
	}
	_ = pendingDeliveries
	
	return nil
}

// ServiceIntegrationExample shows how repositories would be integrated into a service
type ServiceIntegrationExample struct {
	repos *Repositories
}

func NewServiceIntegrationExample(db *database.DB) *ServiceIntegrationExample {
	return &ServiceIntegrationExample{
		repos: NewRepositories(db),
	}
}

func (s *ServiceIntegrationExample) CreateTenant(ctx context.Context, name, apiKeyHash, tier string, rateLimit, quota int) (*models.Tenant, error) {
	tenant := &models.Tenant{
		Name:               name,
		APIKeyHash:         apiKeyHash,
		SubscriptionTier:   tier,
		RateLimitPerMinute: rateLimit,
		MonthlyQuota:       quota,
	}
	
	if err := s.repos.Tenant.Create(ctx, tenant); err != nil {
		return nil, err
	}
	
	return tenant, nil
}

func (s *ServiceIntegrationExample) CreateWebhookEndpoint(ctx context.Context, tenantID uuid.UUID, url, secretHash string) (*models.WebhookEndpoint, error) {
	endpoint := &models.WebhookEndpoint{
		TenantID:   tenantID,
		URL:        url,
		SecretHash: secretHash,
		IsActive:   true,
		RetryConfig: models.RetryConfiguration{
			MaxAttempts:       5,
			InitialDelayMs:    1000,
			MaxDelayMs:        300000,
			BackoffMultiplier: 2,
		},
		CustomHeaders: make(map[string]string),
	}
	
	if err := s.repos.WebhookEndpoint.Create(ctx, endpoint); err != nil {
		return nil, err
	}
	
	return endpoint, nil
}

func (s *ServiceIntegrationExample) GetTenantEndpoints(ctx context.Context, tenantID uuid.UUID) ([]*models.WebhookEndpoint, error) {
	return s.repos.WebhookEndpoint.GetActiveByTenantID(ctx, tenantID)
}