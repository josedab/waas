package experiment

import "context"

// Repository defines the data access interface for experiments.
type Repository interface {
	CreateExperiment(ctx context.Context, exp *Experiment) error
	GetExperiment(ctx context.Context, tenantID, expID string) (*Experiment, error)
	ListExperiments(ctx context.Context, tenantID string) ([]Experiment, error)
	UpdateExperiment(ctx context.Context, exp *Experiment) error
	DeleteExperiment(ctx context.Context, tenantID, expID string) error

	GetAssignment(ctx context.Context, experimentID, webhookID string) (*Assignment, error)
	CreateAssignment(ctx context.Context, assignment *Assignment) error

	GetVariantMetrics(ctx context.Context, experimentID string) ([]VariantMetrics, error)
	UpsertVariantMetrics(ctx context.Context, metrics *VariantMetrics) error
}
