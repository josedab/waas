package obscodepipeline

import "context"

// Repository defines the data access interface for observability pipelines.
type Repository interface {
	CreatePipeline(ctx context.Context, p *ObservabilityPipeline) error
	GetPipeline(ctx context.Context, tenantID, pipelineID string) (*ObservabilityPipeline, error)
	ListPipelines(ctx context.Context, tenantID string) ([]ObservabilityPipeline, error)
	UpdatePipeline(ctx context.Context, p *ObservabilityPipeline) error
	DeletePipeline(ctx context.Context, tenantID, pipelineID string) error

	SaveExecution(ctx context.Context, exec *PipelineExecution) error
	ListExecutions(ctx context.Context, pipelineID string, limit, offset int) ([]PipelineExecution, error)

	SaveAlertEvent(ctx context.Context, alert *AlertEvent) error
	ListAlertEvents(ctx context.Context, tenantID string, limit, offset int) ([]AlertEvent, error)
	GetActiveAlerts(ctx context.Context, tenantID string) ([]AlertEvent, error)

	GetPipelineStats(ctx context.Context, pipelineID string) (*PipelineStats, error)

	SaveReconcileResult(ctx context.Context, result *ReconcileResult) error
	GetLatestReconcileResult(ctx context.Context, tenantID string) (*ReconcileResult, error)
}
