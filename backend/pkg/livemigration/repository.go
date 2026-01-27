package livemigration

import "context"

// Repository defines the data access interface for live migration management
type Repository interface {
	// Jobs
	CreateJob(ctx context.Context, job *MigrationJob) error
	GetJob(ctx context.Context, tenantID, jobID string) (*MigrationJob, error)
	ListJobs(ctx context.Context, tenantID string) ([]MigrationJob, error)
	UpdateJob(ctx context.Context, job *MigrationJob) error
	DeleteJob(ctx context.Context, tenantID, jobID string) error

	// Endpoints
	CreateEndpoint(ctx context.Context, endpoint *MigrationEndpoint) error
	GetEndpoint(ctx context.Context, tenantID, endpointID string) (*MigrationEndpoint, error)
	ListEndpointsByJob(ctx context.Context, tenantID, jobID string) ([]MigrationEndpoint, error)
	UpdateEndpoint(ctx context.Context, endpoint *MigrationEndpoint) error
	DeleteEndpointsByJob(ctx context.Context, tenantID, jobID string) error

	// Parallel delivery results
	CreateParallelResult(ctx context.Context, result *ParallelDeliveryResult) error
	ListParallelResultsByJob(ctx context.Context, tenantID, jobID string) ([]ParallelDeliveryResult, error)

	// Stats and readiness
	GetMigrationStats(ctx context.Context, tenantID, jobID string) (*MigrationStats, error)
	GetCutoverReadiness(ctx context.Context, tenantID, jobID string) (totalEndpoints, readyCount, failedCount int, parallelSuccessRate float64, err error)

	// Checkpoints
	CreateCheckpoint(ctx context.Context, checkpoint *MigrationCheckpoint) error
	GetCheckpoint(ctx context.Context, migrationID string) (*MigrationCheckpoint, error)
	UpdateCheckpoint(ctx context.Context, checkpoint *MigrationCheckpoint) error
}
