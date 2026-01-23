package sandbox

import "context"

// Repository defines the data access interface for sandbox management
type Repository interface {
	// Sandbox environments
	CreateSandbox(ctx context.Context, sandbox *SandboxEnvironment) error
	GetSandbox(ctx context.Context, tenantID, sandboxID string) (*SandboxEnvironment, error)
	ListSandboxes(ctx context.Context, tenantID string) ([]SandboxEnvironment, error)
	UpdateSandbox(ctx context.Context, sandbox *SandboxEnvironment) error
	DeleteSandbox(ctx context.Context, tenantID, sandboxID string) error
	DeleteExpiredSandboxes(ctx context.Context, tenantID string) (int64, error)

	// Replay sessions
	CreateReplaySession(ctx context.Context, session *ReplaySession) error
	GetReplaySession(ctx context.Context, tenantID, sessionID string) (*ReplaySession, error)
	ListReplaySessionsBySandbox(ctx context.Context, tenantID, sandboxID string) ([]ReplaySession, error)
	UpdateReplaySession(ctx context.Context, session *ReplaySession) error

	// Comparison data
	GetReplaySessionsForComparison(ctx context.Context, tenantID, sandboxID string) ([]ReplaySession, error)

	// Mock endpoints
	CreateMockEndpoint(ctx context.Context, endpoint *MockEndpointConfig) error
	GetMockEndpoint(ctx context.Context, sandboxID, endpointID string) (*MockEndpointConfig, error)
	ListMockEndpoints(ctx context.Context, sandboxID string) ([]MockEndpointConfig, error)
	UpdateMockEndpoint(ctx context.Context, endpoint *MockEndpointConfig) error

	// Captured requests
	CreateCapturedRequest(ctx context.Context, req *CapturedRequest) error
	ListCapturedRequests(ctx context.Context, sandboxID string, limit, offset int) ([]CapturedRequest, error)

	// Test scenarios
	CreateTestScenario(ctx context.Context, scenario *TestScenario) error
	GetTestScenario(ctx context.Context, scenarioID string) (*TestScenario, error)
	ListTestScenarios(ctx context.Context, tenantID string) ([]TestScenario, error)
	UpdateTestScenario(ctx context.Context, scenario *TestScenario) error

	// Scenario results
	CreateScenarioResult(ctx context.Context, result *ScenarioResult) error
	GetScenarioResult(ctx context.Context, scenarioID string) (*ScenarioResult, error)
	UpdateScenarioResult(ctx context.Context, result *ScenarioResult) error
}
