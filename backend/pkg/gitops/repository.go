package gitops

import "context"

// Repository defines the data access interface for GitOps configuration management
type Repository interface {
	// Manifests
	CreateManifest(ctx context.Context, manifest *ConfigManifest) error
	GetManifest(ctx context.Context, tenantID, manifestID string) (*ConfigManifest, error)
	ListManifests(ctx context.Context, tenantID string) ([]ConfigManifest, error)
	UpdateManifest(ctx context.Context, manifest *ConfigManifest) error
	DeleteManifest(ctx context.Context, tenantID, manifestID string) error

	// Resources
	CreateResource(ctx context.Context, resource *ConfigResource) error
	GetResource(ctx context.Context, resourceID string) (*ConfigResource, error)
	ListResourcesByManifest(ctx context.Context, manifestID string) ([]ConfigResource, error)
	UpdateResource(ctx context.Context, resource *ConfigResource) error
	DeleteResource(ctx context.Context, resourceID string) error

	// Drift reports
	CreateDriftReport(ctx context.Context, report *DriftReport) error
	GetDriftReport(ctx context.Context, tenantID, reportID string) (*DriftReport, error)
	ListDriftReports(ctx context.Context, tenantID string) ([]DriftReport, error)
	UpdateDriftReport(ctx context.Context, report *DriftReport) error
	DeleteDriftReport(ctx context.Context, tenantID, reportID string) error

	// State queries
	GetCurrentState(ctx context.Context, tenantID, resourceType string) ([]ConfigResource, error)
}
