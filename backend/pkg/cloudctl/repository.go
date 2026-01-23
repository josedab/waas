package cloudctl

import "context"

// Repository defines the data access interface for the cloud control plane
type Repository interface {
	// Tenants
	CreateTenant(ctx context.Context, tenant *CloudTenant) error
	GetTenant(ctx context.Context, tenantID string) (*CloudTenant, error)
	ListTenants(ctx context.Context, status string, limit, offset int) ([]CloudTenant, int, error)
	UpdateTenant(ctx context.Context, tenant *CloudTenant) error
	DeleteTenant(ctx context.Context, tenantID string) error

	// Usage
	RecordUsage(ctx context.Context, metrics *UsageMetrics) error
	GetUsage(ctx context.Context, tenantID, period string) (*UsageMetrics, error)
	GetUsageHistory(ctx context.Context, tenantID string, limit int) ([]UsageMetrics, error)

	// Scaling
	GetScalingConfig(ctx context.Context, tenantID string) (*ScalingConfig, error)
	UpsertScalingConfig(ctx context.Context, config *ScalingConfig) error

	// Dashboard
	GetDashboard(ctx context.Context) (*CloudDashboard, error)
}
