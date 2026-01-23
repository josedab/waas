package canary

import "context"

// Repository defines the data access interface for canary deployments
type Repository interface {
	CreateDeployment(ctx context.Context, deployment *CanaryDeployment) error
	GetDeployment(ctx context.Context, tenantID, deploymentID string) (*CanaryDeployment, error)
	ListDeployments(ctx context.Context, tenantID string, status string, limit, offset int) ([]CanaryDeployment, int, error)
	UpdateDeployment(ctx context.Context, deployment *CanaryDeployment) error
	DeleteDeployment(ctx context.Context, tenantID, deploymentID string) error

	SaveMetrics(ctx context.Context, metrics *CanaryMetrics) error
	GetMetrics(ctx context.Context, tenantID, deploymentID string, limit int) ([]CanaryMetrics, error)
	GetLatestMetrics(ctx context.Context, tenantID, deploymentID string) (*CanaryMetrics, error)
}
