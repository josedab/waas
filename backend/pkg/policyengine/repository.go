package policyengine

import "context"

// Repository defines the data access interface for the policy engine.
type Repository interface {
	CreatePolicy(ctx context.Context, policy *Policy) error
	GetPolicy(ctx context.Context, tenantID, policyID string) (*Policy, error)
	ListPolicies(ctx context.Context, tenantID string) ([]Policy, error)
	UpdatePolicy(ctx context.Context, policy *Policy) error
	DeletePolicy(ctx context.Context, tenantID, policyID string) error
	GetActivePoliciesByType(ctx context.Context, tenantID, policyType string) ([]Policy, error)

	CreatePolicyVersion(ctx context.Context, version *PolicyVersion) error
	ListPolicyVersions(ctx context.Context, policyID string) ([]PolicyVersion, error)

	StoreEvaluationLog(ctx context.Context, log *EvaluationLog) error
	ListEvaluationLogs(ctx context.Context, tenantID string, limit, offset int) ([]EvaluationLog, error)
}
