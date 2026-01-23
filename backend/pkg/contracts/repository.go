package contracts

import "context"

// Repository defines the data access interface for contract testing
type Repository interface {
	// Contracts
	CreateContract(ctx context.Context, contract *Contract) error
	GetContract(ctx context.Context, tenantID, contractID string) (*Contract, error)
	ListContracts(ctx context.Context, tenantID string, limit, offset int) ([]Contract, int, error)
	UpdateContract(ctx context.Context, contract *Contract) error
	DeleteContract(ctx context.Context, tenantID, contractID string) error
	GetContractByEventType(ctx context.Context, tenantID, eventType string) (*Contract, error)

	// Test results
	SaveTestResult(ctx context.Context, result *ContractTestResult) error
	GetTestResult(ctx context.Context, tenantID, resultID string) (*ContractTestResult, error)
	ListTestResults(ctx context.Context, tenantID, contractID string, limit, offset int) ([]ContractTestResult, error)
	GetContractStatus(ctx context.Context, tenantID string) (*ContractStatus, error)
}
