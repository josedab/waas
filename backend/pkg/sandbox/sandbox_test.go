package sandbox

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRepository is an in-memory implementation of Repository for testing
type mockRepository struct {
	mu               sync.RWMutex
	sandboxes        map[string]*SandboxEnvironment
	replaySessions   map[string]*ReplaySession
	mockEndpoints    map[string]*MockEndpointConfig
	capturedRequests []CapturedRequest
	testScenarios    map[string]*TestScenario
	scenarioResults  map[string]*ScenarioResult
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		sandboxes:        make(map[string]*SandboxEnvironment),
		replaySessions:   make(map[string]*ReplaySession),
		mockEndpoints:    make(map[string]*MockEndpointConfig),
		capturedRequests: make([]CapturedRequest, 0),
		testScenarios:    make(map[string]*TestScenario),
		scenarioResults:  make(map[string]*ScenarioResult),
	}
}

func (m *mockRepository) CreateSandbox(_ context.Context, sandbox *SandboxEnvironment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sandboxes[sandbox.ID] = sandbox
	return nil
}

func (m *mockRepository) GetSandbox(_ context.Context, _, sandboxID string) (*SandboxEnvironment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.sandboxes[sandboxID]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("sandbox not found")
}

func (m *mockRepository) ListSandboxes(_ context.Context, tenantID string) ([]SandboxEnvironment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []SandboxEnvironment
	for _, s := range m.sandboxes {
		if s.TenantID == tenantID {
			result = append(result, *s)
		}
	}
	return result, nil
}

func (m *mockRepository) UpdateSandbox(_ context.Context, sandbox *SandboxEnvironment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sandboxes[sandbox.ID] = sandbox
	return nil
}

func (m *mockRepository) DeleteSandbox(_ context.Context, _, sandboxID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sandboxes, sandboxID)
	return nil
}

func (m *mockRepository) DeleteExpiredSandboxes(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

func (m *mockRepository) CreateReplaySession(_ context.Context, session *ReplaySession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.replaySessions[session.ID] = session
	return nil
}

func (m *mockRepository) GetReplaySession(_ context.Context, _, sessionID string) (*ReplaySession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.replaySessions[sessionID]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("session not found")
}

func (m *mockRepository) ListReplaySessionsBySandbox(_ context.Context, _, sandboxID string) ([]ReplaySession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []ReplaySession
	for _, s := range m.replaySessions {
		if s.SandboxID == sandboxID {
			result = append(result, *s)
		}
	}
	return result, nil
}

func (m *mockRepository) UpdateReplaySession(_ context.Context, session *ReplaySession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.replaySessions[session.ID] = session
	return nil
}

func (m *mockRepository) GetReplaySessionsForComparison(_ context.Context, _, sandboxID string) ([]ReplaySession, error) {
	return m.ListReplaySessionsBySandbox(context.Background(), "", sandboxID)
}

func (m *mockRepository) CreateMockEndpoint(_ context.Context, endpoint *MockEndpointConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mockEndpoints[endpoint.ID] = endpoint
	return nil
}

func (m *mockRepository) GetMockEndpoint(_ context.Context, sandboxID, endpointID string) (*MockEndpointConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if e, ok := m.mockEndpoints[endpointID]; ok && e.SandboxID == sandboxID {
		return e, nil
	}
	return nil, fmt.Errorf("mock endpoint not found")
}

func (m *mockRepository) ListMockEndpoints(_ context.Context, sandboxID string) ([]MockEndpointConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []MockEndpointConfig
	for _, e := range m.mockEndpoints {
		if e.SandboxID == sandboxID {
			result = append(result, *e)
		}
	}
	return result, nil
}

func (m *mockRepository) UpdateMockEndpoint(_ context.Context, endpoint *MockEndpointConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mockEndpoints[endpoint.ID] = endpoint
	return nil
}

func (m *mockRepository) CreateCapturedRequest(_ context.Context, req *CapturedRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.capturedRequests = append(m.capturedRequests, *req)
	return nil
}

func (m *mockRepository) ListCapturedRequests(_ context.Context, sandboxID string, limit, offset int) ([]CapturedRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []CapturedRequest
	for _, r := range m.capturedRequests {
		if r.SandboxID == sandboxID {
			result = append(result, r)
		}
	}
	if offset >= len(result) {
		return nil, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], nil
}

func (m *mockRepository) CreateTestScenario(_ context.Context, scenario *TestScenario) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.testScenarios[scenario.ID] = scenario
	return nil
}

func (m *mockRepository) GetTestScenario(_ context.Context, scenarioID string) (*TestScenario, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.testScenarios[scenarioID]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("scenario not found")
}

func (m *mockRepository) ListTestScenarios(_ context.Context, tenantID string) ([]TestScenario, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []TestScenario
	for _, s := range m.testScenarios {
		if s.TenantID == tenantID {
			result = append(result, *s)
		}
	}
	return result, nil
}

func (m *mockRepository) UpdateTestScenario(_ context.Context, scenario *TestScenario) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.testScenarios[scenario.ID] = scenario
	return nil
}

func (m *mockRepository) CreateScenarioResult(_ context.Context, result *ScenarioResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scenarioResults[result.ScenarioID] = result
	return nil
}

func (m *mockRepository) GetScenarioResult(_ context.Context, scenarioID string) (*ScenarioResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if r, ok := m.scenarioResults[scenarioID]; ok {
		return r, nil
	}
	return nil, fmt.Errorf("scenario result not found")
}

func (m *mockRepository) UpdateScenarioResult(_ context.Context, result *ScenarioResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scenarioResults[result.ScenarioID] = result
	return nil
}

// --- Tests ---

func TestHandleMockRequest_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	sandboxID := "sandbox-1"
	endpoint := &MockEndpointConfig{
		ID:             "endpoint-1",
		SandboxID:      sandboxID,
		Path:           "/webhook",
		Method:         "POST",
		ResponseStatus: 200,
		ResponseBody:   `{"status":"ok"}`,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	require.NoError(t, repo.CreateMockEndpoint(ctx, endpoint))

	captured, statusCode, respBody, err := svc.HandleMockRequest(
		ctx, sandboxID, "endpoint-1", "POST", "/webhook",
		map[string]string{"Content-Type": "application/json"},
		`{"event":"test"}`,
	)

	require.NoError(t, err)
	assert.Equal(t, 200, statusCode)
	assert.Equal(t, `{"status":"ok"}`, respBody)
	assert.Equal(t, sandboxID, captured.SandboxID)
	assert.Equal(t, "endpoint-1", captured.EndpointID)
	assert.Equal(t, "POST", captured.Method)
	assert.Equal(t, `{"event":"test"}`, captured.RequestBody)
	assert.False(t, captured.FailureInjected)
}

func TestHandleMockRequest_ConfiguredFailure(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	sandboxID := "sandbox-2"
	endpoint := &MockEndpointConfig{
		ID:             "endpoint-2",
		SandboxID:      sandboxID,
		Path:           "/webhook",
		Method:         "POST",
		ResponseStatus: 200,
		ResponseBody:   `{"status":"ok"}`,
		FailureRate:    1.0, // Always fail
		Failures: []FailureScenario{
			{Name: "server_error", Type: Failure500Error, Probability: 1.0},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, repo.CreateMockEndpoint(ctx, endpoint))

	captured, statusCode, _, err := svc.HandleMockRequest(
		ctx, sandboxID, "endpoint-2", "POST", "/webhook", nil, `{}`,
	)

	require.NoError(t, err)
	assert.Equal(t, 500, statusCode)
	assert.True(t, captured.FailureInjected)
	assert.NotEmpty(t, captured.FailureType)
}

func TestFailureInjection_Simulation(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	endpoint := &MockEndpointConfig{
		FailureRate: 1.0,
		Failures: []FailureScenario{
			{Name: "timeout", Type: FailureTimeout, Probability: 1.0},
		},
	}

	injected, failType := svc.checkFailureInjection(endpoint)
	assert.True(t, injected)
	assert.Equal(t, FailureTimeout, failType)

	endpoint.FailureRate = 0.0
	endpoint.Failures = nil
	injected, _ = svc.checkFailureInjection(endpoint)
	assert.False(t, injected)
}

func TestSimulateLatency_UniformDistribution(t *testing.T) {
	config := LatencySimulation{
		MinMs:            100,
		MaxMs:            500,
		DistributionType: DistributionUniform,
	}

	for i := 0; i < 100; i++ {
		latency := SimulateLatency(config)
		assert.GreaterOrEqual(t, latency, 100)
		assert.LessOrEqual(t, latency, 500)
	}
}

func TestSimulateLatency_NormalDistribution(t *testing.T) {
	config := LatencySimulation{
		MinMs:            100,
		MaxMs:            500,
		DistributionType: DistributionNormal,
	}

	for i := 0; i < 100; i++ {
		latency := SimulateLatency(config)
		assert.GreaterOrEqual(t, latency, 100)
		assert.LessOrEqual(t, latency, 500)
	}
}

func TestSimulateLatency_EqualMinMax(t *testing.T) {
	config := LatencySimulation{
		MinMs:            200,
		MaxMs:            200,
		DistributionType: DistributionUniform,
	}

	latency := SimulateLatency(config)
	assert.Equal(t, 200, latency)
}

func TestRequestCaptureAndRetrieval(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	sandboxID := "sandbox-3"
	endpoint := &MockEndpointConfig{
		ID:             "endpoint-3",
		SandboxID:      sandboxID,
		Path:           "/webhook",
		Method:         "POST",
		ResponseStatus: 200,
		ResponseBody:   `{"status":"ok"}`,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	require.NoError(t, repo.CreateMockEndpoint(ctx, endpoint))

	// Send multiple requests
	for i := 0; i < 5; i++ {
		_, _, _, err := svc.HandleMockRequest(
			ctx, sandboxID, "endpoint-3", "POST", "/webhook", nil,
			fmt.Sprintf(`{"event":"test_%d"}`, i),
		)
		require.NoError(t, err)
	}

	// Retrieve captured requests
	requests, err := svc.GetCapturedRequests(ctx, sandboxID, 10, 0)
	require.NoError(t, err)
	assert.Len(t, requests, 5)

	// Test pagination
	requests, err = svc.GetCapturedRequests(ctx, sandboxID, 2, 0)
	require.NoError(t, err)
	assert.Len(t, requests, 2)

	requests, err = svc.GetCapturedRequests(ctx, sandboxID, 10, 3)
	require.NoError(t, err)
	assert.Len(t, requests, 2)
}

func TestCreateTestScenario(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	req := &CreateTestScenarioRequest{
		SandboxID:   "sandbox-4",
		Name:        "Basic webhook test",
		Description: "Tests basic webhook delivery",
		Steps: []TestStep{
			{Type: StepSendWebhook, EndpointID: "ep-1", Payload: `{"event":"test"}`, ExpectedStatus: 200},
			{Type: StepWait, WaitMs: 10},
			{Type: StepAssertDelivery},
		},
	}

	scenario, err := svc.CreateTestScenario(ctx, "tenant-1", req)
	require.NoError(t, err)
	assert.NotEmpty(t, scenario.ID)
	assert.Equal(t, "Basic webhook test", scenario.Name)
	assert.Equal(t, ScenarioStatusPending, scenario.Status)
	assert.Len(t, scenario.Steps, 3)

	for i, step := range scenario.Steps {
		assert.NotEmpty(t, step.ID)
		assert.Equal(t, scenario.ID, step.ScenarioID)
		assert.Equal(t, i+1, step.Order)
	}
}

func TestRunTestScenario_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	sandboxID := "sandbox-5"

	// Create a mock endpoint
	endpoint := &MockEndpointConfig{
		ID:             "ep-run-1",
		SandboxID:      sandboxID,
		Path:           "/webhook",
		Method:         "POST",
		ResponseStatus: 200,
		ResponseBody:   `{"status":"ok"}`,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	require.NoError(t, repo.CreateMockEndpoint(ctx, endpoint))

	// Create a test scenario
	scenarioReq := &CreateTestScenarioRequest{
		SandboxID:   sandboxID,
		Name:        "Run test",
		Description: "Test scenario execution",
		Steps: []TestStep{
			{Type: StepSendWebhook, EndpointID: "ep-run-1", Payload: `{"event":"test"}`, ExpectedStatus: 200},
			{Type: StepWait, WaitMs: 1},
			{Type: StepAssertDelivery},
		},
	}

	scenario, err := svc.CreateTestScenario(ctx, "tenant-1", scenarioReq)
	require.NoError(t, err)

	// Run the scenario
	result, err := svc.RunTestScenario(ctx, scenario.ID)
	require.NoError(t, err)
	assert.Equal(t, ScenarioStatusCompleted, result.Status)
	assert.Equal(t, 3, result.TotalSteps)
	assert.Equal(t, 3, result.PassedSteps)
	assert.Equal(t, 0, result.FailedSteps)
	assert.Len(t, result.Steps, 3)

	// Verify the scenario status was updated
	updated, err := repo.GetTestScenario(ctx, scenario.ID)
	require.NoError(t, err)
	assert.Equal(t, ScenarioStatusCompleted, updated.Status)

	// Verify results are retrievable
	retrieved, err := svc.GetScenarioResults(ctx, scenario.ID)
	require.NoError(t, err)
	assert.Equal(t, ScenarioStatusCompleted, retrieved.Status)
}

func TestRunTestScenario_FailedStep(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	sandboxID := "sandbox-6"

	endpoint := &MockEndpointConfig{
		ID:             "ep-fail-1",
		SandboxID:      sandboxID,
		Path:           "/webhook",
		Method:         "POST",
		ResponseStatus: 500,
		ResponseBody:   `{"error":"server error"}`,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	require.NoError(t, repo.CreateMockEndpoint(ctx, endpoint))

	scenarioReq := &CreateTestScenarioRequest{
		SandboxID: sandboxID,
		Name:      "Failure test",
		Steps: []TestStep{
			{Type: StepSendWebhook, EndpointID: "ep-fail-1", Payload: `{}`, ExpectedStatus: 200},
		},
	}

	scenario, err := svc.CreateTestScenario(ctx, "tenant-1", scenarioReq)
	require.NoError(t, err)

	result, err := svc.RunTestScenario(ctx, scenario.ID)
	require.NoError(t, err)
	assert.Equal(t, ScenarioStatusFailed, result.Status)
	assert.Equal(t, 1, result.FailedSteps)
	assert.Contains(t, result.Steps[0].ErrorMessage, "expected status 200, got 500")
}

func TestInjectChaos(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	sandboxID := "sandbox-7"

	endpoint := &MockEndpointConfig{
		ID:             "ep-chaos-1",
		SandboxID:      sandboxID,
		Path:           "/webhook",
		Method:         "POST",
		ResponseStatus: 200,
		ResponseBody:   `{"status":"ok"}`,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	require.NoError(t, repo.CreateMockEndpoint(ctx, endpoint))

	err := svc.InjectChaos(ctx, sandboxID, FailureTimeout, 0.5)
	require.NoError(t, err)

	// Verify the endpoint was updated with the chaos config
	updated, err := repo.GetMockEndpoint(ctx, sandboxID, "ep-chaos-1")
	require.NoError(t, err)
	require.Len(t, updated.Failures, 1)
	assert.Equal(t, FailureTimeout, updated.Failures[0].Type)
	assert.Equal(t, 0.5, updated.Failures[0].Probability)
}

func TestGetCapturedRequests_DefaultLimits(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	// Test with invalid limit and offset
	requests, err := svc.GetCapturedRequests(ctx, "sandbox-empty", -1, -1)
	require.NoError(t, err)
	assert.Empty(t, requests)
}
