package sla

import "context"

// Repository defines the data access interface for SLA management
type Repository interface {
	// Targets
	CreateTarget(ctx context.Context, target *Target) error
	GetTarget(ctx context.Context, tenantID, targetID string) (*Target, error)
	ListTargets(ctx context.Context, tenantID string) ([]Target, error)
	UpdateTarget(ctx context.Context, target *Target) error
	DeleteTarget(ctx context.Context, tenantID, targetID string) error

	// Compliance metrics
	GetDeliveryStats(ctx context.Context, tenantID, endpointID string, windowMinutes int) (total, success, failed int, avgLatencyMs float64, p50Ms, p99Ms int, err error)

	// Breaches
	CreateBreach(ctx context.Context, breach *Breach) error
	ListActiveBreaches(ctx context.Context, tenantID string) ([]Breach, error)
	ListBreachHistory(ctx context.Context, tenantID string, limit, offset int) ([]Breach, error)
	ResolveBreach(ctx context.Context, tenantID, breachID string) error

	// Alert config
	GetAlertConfig(ctx context.Context, tenantID, targetID string) (*AlertConfig, error)
	UpsertAlertConfig(ctx context.Context, config *AlertConfig) error
}
