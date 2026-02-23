package piidetection

import "context"

// Repository defines the data access interface for PII detection.
type Repository interface {
	// Policies
	CreatePolicy(ctx context.Context, policy *Policy) error
	GetPolicy(ctx context.Context, tenantID, policyID string) (*Policy, error)
	ListPolicies(ctx context.Context, tenantID string) ([]Policy, error)
	UpdatePolicy(ctx context.Context, policy *Policy) error
	DeletePolicy(ctx context.Context, tenantID, policyID string) error
	GetEnabledPolicies(ctx context.Context, tenantID string) ([]Policy, error)

	// Scan results
	StoreScanResult(ctx context.Context, result *ScanResult) error
	GetScanResult(ctx context.Context, tenantID, resultID string) (*ScanResult, error)
	ListScanResults(ctx context.Context, tenantID string, limit, offset int) ([]ScanResult, error)
	GetDashboardStats(ctx context.Context, tenantID string) (*DashboardStats, error)
}
