package tfprovider

import "context"

// Repository defines the data access interface for IaC resource management
type Repository interface {
	SaveResource(ctx context.Context, resource *ManagedResource) error
	GetResource(ctx context.Context, tenantID string, resourceType ResourceType, resourceID string) (*ManagedResource, error)
	ListResources(ctx context.Context, tenantID string, resourceType ResourceType) ([]ManagedResource, error)
	UpdateResource(ctx context.Context, resource *ManagedResource) error
	DeleteResource(ctx context.Context, tenantID string, resourceType ResourceType, resourceID string) error
}
