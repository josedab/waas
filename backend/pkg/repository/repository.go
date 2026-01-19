package repository

import "webhook-platform/pkg/database"

// Repositories holds all repository instances
type Repositories struct {
	Tenant          TenantRepository
	WebhookEndpoint WebhookEndpointRepository
	DeliveryAttempt DeliveryAttemptRepository
}

// NewRepositories creates a new instance of all repositories
func NewRepositories(db *database.DB) *Repositories {
	return &Repositories{
		Tenant:          NewTenantRepository(db),
		WebhookEndpoint: NewWebhookEndpointRepository(db),
		DeliveryAttempt: NewDeliveryAttemptRepository(db),
	}
}