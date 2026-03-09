package reliability

import (
	"context"
)

// Repository defines data access for reliability scoring.
type Repository interface {
	// Score operations
	GetScore(ctx context.Context, tenantID, endpointID string) (*ReliabilityScore, error)
	UpsertScore(ctx context.Context, score *ReliabilityScore) error

	// Snapshot operations
	CreateSnapshot(ctx context.Context, snapshot *ScoreSnapshot) error
	ListSnapshots(ctx context.Context, tenantID, endpointID string, limit int) ([]ScoreSnapshot, error)

	// SLA operations
	GetSLA(ctx context.Context, tenantID, endpointID string) (*SLATarget, error)
	UpsertSLA(ctx context.Context, sla *SLATarget) error
	DeleteSLA(ctx context.Context, tenantID, endpointID string) error

	// Alert threshold operations
	GetAlertThreshold(ctx context.Context, tenantID, endpointID string) (*AlertThreshold, error)
	UpsertAlertThreshold(ctx context.Context, threshold *AlertThreshold) error
	DeleteAlertThreshold(ctx context.Context, tenantID, endpointID string) error

	// Delivery data for scoring
	GetDeliveryStats(ctx context.Context, tenantID, endpointID string, windowHours int) (*DeliveryStats, error)
}

// DeliveryStats holds aggregated delivery data used for score computation.
type DeliveryStats struct {
	TotalAttempts       int
	SuccessfulAttempts  int
	FailedAttempts      int
	ConsecutiveFailures int
	LatencyP50Ms        int
	LatencyP95Ms        int
	LatencyP99Ms        int
}
