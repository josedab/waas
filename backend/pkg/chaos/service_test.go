package chaos

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock Repository ---

type mockChaosRepo struct {
	mu          sync.Mutex
	experiments map[string]*ChaosExperiment
	events      []ChaosEvent
	saveErr     error
	getErr      error
	deleteErr   error
	statsResult map[string]interface{}
}

func newMockChaosRepo() *mockChaosRepo {
	return &mockChaosRepo{
		experiments: make(map[string]*ChaosExperiment),
		statsResult: make(map[string]interface{}),
	}
}

func (m *mockChaosRepo) SaveExperiment(_ context.Context, exp *ChaosExperiment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.saveErr != nil {
		return m.saveErr
	}
	m.experiments[exp.ID] = exp
	return nil
}

func (m *mockChaosRepo) GetExperiment(_ context.Context, tenantID, expID string) (*ChaosExperiment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getErr != nil {
		return nil, m.getErr
	}
	exp, ok := m.experiments[expID]
	if !ok || exp.TenantID != tenantID {
		return nil, fmt.Errorf("experiment not found")
	}
	return exp, nil
}

func (m *mockChaosRepo) ListExperiments(_ context.Context, tenantID string, status *ExperimentStatus, limit, offset int) ([]ChaosExperiment, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []ChaosExperiment
	for _, exp := range m.experiments {
		if exp.TenantID != tenantID {
			continue
		}
		if status != nil && exp.Status != *status {
			continue
		}
		result = append(result, *exp)
	}
	total := len(result)
	if offset >= len(result) {
		return nil, total, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], total, nil
}

func (m *mockChaosRepo) DeleteExperiment(_ context.Context, tenantID, expID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.experiments, expID)
	return nil
}

func (m *mockChaosRepo) SaveEvent(_ context.Context, event *ChaosEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, *event)
	return nil
}

func (m *mockChaosRepo) GetEvents(_ context.Context, tenantID, expID string, limit int) ([]ChaosEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []ChaosEvent
	for _, e := range m.events {
		if e.TenantID == tenantID && e.ExperimentID == expID {
			result = append(result, e)
		}
	}
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (m *mockChaosRepo) GetEventsByDelivery(_ context.Context, tenantID, deliveryID string) ([]ChaosEvent, error) {
	return nil, nil
}

func (m *mockChaosRepo) GetExperimentStats(_ context.Context, tenantID string, start, end time.Time) (map[string]interface{}, error) {
	return m.statsResult, nil
}

// --- Constructor Tests ---

func TestNewService(t *testing.T) {
	svc := NewService(nil, nil)
	if svc == nil {
		t.Fatal("NewService returned nil")
	}
}

func TestNewService_DefaultConfig(t *testing.T) {
	svc := NewService(nil, nil)
	assert.Equal(t, 5, svc.config.MaxConcurrentExperiments)
	assert.True(t, svc.config.SafetyChecksEnabled)
}

func TestNewService_CustomConfig(t *testing.T) {
	cfg := &ServiceConfig{MaxConcurrentExperiments: 2}
	svc := NewService(nil, cfg)
	assert.Equal(t, 2, svc.config.MaxConcurrentExperiments)
}

// --- CreateExperiment Tests ---

func TestCreateExperiment_ValidConfig(t *testing.T) {
	repo := newMockChaosRepo()
	svc := NewService(repo, nil)
	defer svc.Close()

	req := &CreateExperimentRequest{
		Name:     "test-latency",
		Type:     ExperimentLatency,
		Duration: 60,
		TargetConfig: TargetConfig{
			EndpointIDs: []string{"ep-1"},
			Percentage:  50,
		},
		FaultConfig: FaultConfig{LatencyMs: 500},
		BlastRadius: BlastRadius{MaxAffectedEndpoints: 5, MaxAffectedDeliveries: 100},
	}

	exp, err := svc.CreateExperiment(context.Background(), "tenant-1", "user-1", req)
	require.NoError(t, err)
	assert.NotEmpty(t, exp.ID)
	assert.Equal(t, StatusPending, exp.Status)
	assert.Equal(t, "tenant-1", exp.TenantID)
	assert.Equal(t, int64(100), exp.BlastRadius.MaxAffectedDeliveries)
}

func TestCreateExperiment_DefaultBlastRadius(t *testing.T) {
	repo := newMockChaosRepo()
	svc := NewService(repo, nil)
	defer svc.Close()

	req := &CreateExperimentRequest{
		Name:     "test",
		Type:     ExperimentError,
		Duration: 30,
	}

	exp, err := svc.CreateExperiment(context.Background(), "tenant-1", "user-1", req)
	require.NoError(t, err)
	assert.Equal(t, svc.config.DefaultBlastRadius.MaxAffectedEndpoints, exp.BlastRadius.MaxAffectedEndpoints)
}

func TestCreateExperiment_Scheduled(t *testing.T) {
	repo := newMockChaosRepo()
	svc := NewService(repo, nil)
	defer svc.Close()

	future := time.Now().Add(1 * time.Hour)
	req := &CreateExperimentRequest{
		Name:     "scheduled",
		Type:     ExperimentLatency,
		Duration: 60,
		Schedule: &ScheduleConfig{StartTime: future},
		BlastRadius: BlastRadius{MaxAffectedEndpoints: 1, MaxAffectedDeliveries: 10},
	}

	exp, err := svc.CreateExperiment(context.Background(), "tenant-1", "user-1", req)
	require.NoError(t, err)
	assert.Equal(t, StatusScheduled, exp.Status)
}

func TestCreateExperiment_RepoError(t *testing.T) {
	repo := newMockChaosRepo()
	repo.saveErr = fmt.Errorf("db error")
	svc := NewService(repo, nil)
	defer svc.Close()

	req := &CreateExperimentRequest{
		Name:     "fail",
		Type:     ExperimentError,
		Duration: 10,
		BlastRadius: BlastRadius{MaxAffectedEndpoints: 1},
	}

	_, err := svc.CreateExperiment(context.Background(), "t1", "u1", req)
	assert.Error(t, err)
}

// --- StartExperiment Tests ---

func TestStartExperiment_Valid(t *testing.T) {
	repo := newMockChaosRepo()
	svc := NewService(repo, nil)
	defer svc.Close()

	req := &CreateExperimentRequest{
		Name:        "start-test",
		Type:        ExperimentLatency,
		Duration:    1, // 1 second
		FaultConfig: FaultConfig{LatencyMs: 100},
		BlastRadius: BlastRadius{MaxAffectedEndpoints: 1, MaxAffectedDeliveries: 10},
	}
	exp, _ := svc.CreateExperiment(context.Background(), "tenant-1", "user-1", req)

	started, err := svc.StartExperiment(context.Background(), "tenant-1", exp.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusRunning, started.Status)
	assert.NotNil(t, started.StartedAt)
}

func TestStartExperiment_AlreadyRunning(t *testing.T) {
	repo := newMockChaosRepo()
	svc := NewService(repo, nil)
	defer svc.Close()

	req := &CreateExperimentRequest{
		Name:        "run-test",
		Type:        ExperimentLatency,
		Duration:    300,
		FaultConfig: FaultConfig{LatencyMs: 100},
		BlastRadius: BlastRadius{MaxAffectedEndpoints: 1, MaxAffectedDeliveries: 10},
	}
	exp, _ := svc.CreateExperiment(context.Background(), "tenant-1", "user-1", req)
	svc.StartExperiment(context.Background(), "tenant-1", exp.ID)

	_, err := svc.StartExperiment(context.Background(), "tenant-1", exp.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be started")
}

func TestStartExperiment_ConcurrentLimit(t *testing.T) {
	repo := newMockChaosRepo()
	cfg := &ServiceConfig{
		MaxConcurrentExperiments: 1,
		DefaultBlastRadius:      BlastRadius{MaxAffectedEndpoints: 1, MaxAffectedDeliveries: 10},
	}
	svc := NewService(repo, cfg)
	defer svc.Close()

	// Start first experiment
	req1 := &CreateExperimentRequest{Name: "exp1", Type: ExperimentLatency, Duration: 300, BlastRadius: BlastRadius{MaxAffectedEndpoints: 1, MaxAffectedDeliveries: 10}}
	exp1, _ := svc.CreateExperiment(context.Background(), "tenant-1", "user-1", req1)
	_, err := svc.StartExperiment(context.Background(), "tenant-1", exp1.ID)
	require.NoError(t, err)

	// Second should fail
	req2 := &CreateExperimentRequest{Name: "exp2", Type: ExperimentError, Duration: 300, BlastRadius: BlastRadius{MaxAffectedEndpoints: 1, MaxAffectedDeliveries: 10}}
	exp2, _ := svc.CreateExperiment(context.Background(), "tenant-1", "user-1", req2)
	_, err = svc.StartExperiment(context.Background(), "tenant-1", exp2.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maximum concurrent")
}

// --- StopExperiment Tests ---

func TestStopExperiment_Running(t *testing.T) {
	repo := newMockChaosRepo()
	svc := NewService(repo, nil)
	defer svc.Close()

	req := &CreateExperimentRequest{Name: "stop-test", Type: ExperimentLatency, Duration: 300, BlastRadius: BlastRadius{MaxAffectedEndpoints: 1, MaxAffectedDeliveries: 10}}
	exp, _ := svc.CreateExperiment(context.Background(), "tenant-1", "user-1", req)
	svc.StartExperiment(context.Background(), "tenant-1", exp.ID)

	stopped, err := svc.StopExperiment(context.Background(), "tenant-1", exp.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusAborted, stopped.Status)
	assert.NotNil(t, stopped.CompletedAt)
}

func TestStopExperiment_NotRunning(t *testing.T) {
	repo := newMockChaosRepo()
	svc := NewService(repo, nil)
	defer svc.Close()

	req := &CreateExperimentRequest{Name: "stop-pending", Type: ExperimentLatency, Duration: 60, BlastRadius: BlastRadius{MaxAffectedEndpoints: 1, MaxAffectedDeliveries: 10}}
	exp, _ := svc.CreateExperiment(context.Background(), "tenant-1", "user-1", req)

	_, err := svc.StopExperiment(context.Background(), "tenant-1", exp.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

// --- ShouldInjectFault Tests ---

func TestShouldInjectFault_MatchingEndpoint(t *testing.T) {
	repo := newMockChaosRepo()
	svc := NewService(repo, nil)
	defer svc.Close()

	req := &CreateExperimentRequest{
		Name:     "inject-test",
		Type:     ExperimentLatency,
		Duration: 300,
		TargetConfig: TargetConfig{
			EndpointIDs: []string{"ep-1"},
			Percentage:  100,
		},
		FaultConfig: FaultConfig{LatencyMs: 500},
		BlastRadius: BlastRadius{MaxAffectedEndpoints: 10, MaxAffectedDeliveries: 1000},
	}
	exp, _ := svc.CreateExperiment(context.Background(), "tenant-1", "user-1", req)
	svc.StartExperiment(context.Background(), "tenant-1", exp.ID)
	time.Sleep(10 * time.Millisecond) // Let agent start

	injection, err := svc.ShouldInjectFault(context.Background(), "tenant-1", "ep-1", "del-1")
	require.NoError(t, err)
	assert.NotNil(t, injection)
	assert.Equal(t, ExperimentLatency, injection.Type)
}

func TestShouldInjectFault_NonMatchingEndpoint(t *testing.T) {
	repo := newMockChaosRepo()
	svc := NewService(repo, nil)
	defer svc.Close()

	req := &CreateExperimentRequest{
		Name:     "no-inject",
		Type:     ExperimentLatency,
		Duration: 300,
		TargetConfig: TargetConfig{
			EndpointIDs: []string{"ep-1"},
			Percentage:  100,
		},
		FaultConfig: FaultConfig{LatencyMs: 500},
		BlastRadius: BlastRadius{MaxAffectedEndpoints: 10, MaxAffectedDeliveries: 1000},
	}
	exp, _ := svc.CreateExperiment(context.Background(), "tenant-1", "user-1", req)
	svc.StartExperiment(context.Background(), "tenant-1", exp.ID)
	time.Sleep(10 * time.Millisecond)

	injection, err := svc.ShouldInjectFault(context.Background(), "tenant-1", "ep-other", "del-1")
	require.NoError(t, err)
	assert.Nil(t, injection)
}

func TestShouldInjectFault_DifferentTenant(t *testing.T) {
	repo := newMockChaosRepo()
	svc := NewService(repo, nil)
	defer svc.Close()

	req := &CreateExperimentRequest{
		Name:     "tenant-check",
		Type:     ExperimentLatency,
		Duration: 300,
		TargetConfig: TargetConfig{Percentage: 100},
		FaultConfig:  FaultConfig{LatencyMs: 500},
		BlastRadius:  BlastRadius{MaxAffectedEndpoints: 10, MaxAffectedDeliveries: 1000},
	}
	exp, _ := svc.CreateExperiment(context.Background(), "tenant-1", "user-1", req)
	svc.StartExperiment(context.Background(), "tenant-1", exp.ID)
	time.Sleep(10 * time.Millisecond)

	injection, err := svc.ShouldInjectFault(context.Background(), "tenant-other", "ep-1", "del-1")
	require.NoError(t, err)
	assert.Nil(t, injection)
}

// --- Agent.getFaultInjection Tests ---

func TestAgent_GetFaultInjection_Latency(t *testing.T) {
	repo := newMockChaosRepo()
	exp := &ChaosExperiment{
		ID: "exp-1", TenantID: "t1", Type: ExperimentLatency,
		FaultConfig: FaultConfig{LatencyMs: 500},
	}
	agent := NewAgent(exp, repo)
	agent.running = true

	inj := agent.getFaultInjection("ep-1", "del-1")
	assert.Equal(t, ExperimentLatency, inj.Type)
	assert.Equal(t, 500, inj.LatencyMs)
	assert.False(t, inj.ShouldDrop)
}

func TestAgent_GetFaultInjection_Error(t *testing.T) {
	repo := newMockChaosRepo()
	exp := &ChaosExperiment{
		ID: "exp-1", TenantID: "t1", Type: ExperimentError,
		FaultConfig: FaultConfig{ErrorRate: 1.0, ErrorCode: 500, ErrorMessage: "test"},
	}
	agent := NewAgent(exp, repo)
	agent.running = true

	inj := agent.getFaultInjection("ep-1", "del-1")
	assert.Equal(t, ExperimentError, inj.Type)
	assert.Equal(t, 500, inj.ErrorCode)
}

func TestAgent_GetFaultInjection_Timeout(t *testing.T) {
	repo := newMockChaosRepo()
	exp := &ChaosExperiment{
		ID: "exp-1", TenantID: "t1", Type: ExperimentTimeout,
		FaultConfig: FaultConfig{TimeoutMs: 30000},
	}
	agent := NewAgent(exp, repo)
	agent.running = true

	inj := agent.getFaultInjection("ep-1", "del-1")
	assert.Equal(t, ExperimentTimeout, inj.Type)
	assert.Equal(t, 30000, inj.LatencyMs)
}

func TestAgent_GetFaultInjection_RateLimit(t *testing.T) {
	repo := newMockChaosRepo()
	exp := &ChaosExperiment{
		ID: "exp-1", TenantID: "t1", Type: ExperimentRateLimit,
		FaultConfig: FaultConfig{RateLimitCode: 429, RetryAfterSec: 30},
	}
	agent := NewAgent(exp, repo)
	agent.running = true

	inj := agent.getFaultInjection("ep-1", "del-1")
	assert.Equal(t, ExperimentRateLimit, inj.Type)
	assert.Equal(t, 429, inj.ErrorCode)
	assert.Contains(t, inj.ErrorMessage, "30 seconds")
}

func TestAgent_GetFaultInjection_Blackhole(t *testing.T) {
	repo := newMockChaosRepo()
	exp := &ChaosExperiment{
		ID: "exp-1", TenantID: "t1", Type: ExperimentBlackhole,
	}
	agent := NewAgent(exp, repo)
	agent.running = true

	inj := agent.getFaultInjection("ep-1", "del-1")
	assert.Equal(t, ExperimentBlackhole, inj.Type)
	assert.True(t, inj.ShouldDrop)
}

// --- Agent.shouldAffect Tests ---

func TestAgent_ShouldAffect_NotRunning(t *testing.T) {
	exp := &ChaosExperiment{
		TargetConfig: TargetConfig{Percentage: 100},
		BlastRadius:  BlastRadius{MaxAffectedDeliveries: 1000},
	}
	agent := NewAgent(exp, nil)
	// running = false by default
	assert.False(t, agent.shouldAffect("ep-1"))
}

func TestAgent_ShouldAffect_BlastRadiusExceeded(t *testing.T) {
	exp := &ChaosExperiment{
		TargetConfig: TargetConfig{Percentage: 100},
		BlastRadius:  BlastRadius{MaxAffectedDeliveries: 0},
	}
	agent := NewAgent(exp, nil)
	agent.running = true
	assert.False(t, agent.shouldAffect("ep-1"))
}

func TestAgent_ShouldAffect_TargetedEndpoint(t *testing.T) {
	exp := &ChaosExperiment{
		TargetConfig: TargetConfig{
			EndpointIDs: []string{"ep-1", "ep-2"},
			Percentage:  100,
		},
		BlastRadius: BlastRadius{MaxAffectedDeliveries: 1000},
	}
	agent := NewAgent(exp, nil)
	agent.running = true

	assert.True(t, agent.shouldAffect("ep-1"))
	assert.False(t, agent.shouldAffect("ep-3"))
}

func TestAgent_ShouldAffect_AllEndpoints(t *testing.T) {
	exp := &ChaosExperiment{
		TargetConfig: TargetConfig{Percentage: 100},
		BlastRadius:  BlastRadius{MaxAffectedDeliveries: 1000},
	}
	agent := NewAgent(exp, nil)
	agent.running = true
	assert.True(t, agent.shouldAffect("any-endpoint"))
}

// --- Agent.Run / Context Cancellation ---

func TestAgent_Run_ContextCancellation(t *testing.T) {
	exp := &ChaosExperiment{Duration: 300}
	agent := NewAgent(exp, nil)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		agent.Run(ctx)
		close(done)
	}()

	time.Sleep(10 * time.Millisecond)
	assert.True(t, agent.running)

	cancel()
	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("agent.Run did not stop on context cancellation")
	}
	assert.False(t, agent.running)
}

func TestAgent_Run_Stop(t *testing.T) {
	exp := &ChaosExperiment{Duration: 300}
	agent := NewAgent(exp, nil)

	done := make(chan struct{})
	go func() {
		agent.Run(context.Background())
		close(done)
	}()

	time.Sleep(10 * time.Millisecond)
	agent.Stop()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("agent.Run did not stop when Stop() called")
	}
}

// --- RecordFaultResult ---

func TestRecordFaultResult(t *testing.T) {
	repo := newMockChaosRepo()
	svc := NewService(repo, nil)
	defer svc.Close()

	err := svc.RecordFaultResult(context.Background(), "t1", "exp-1", "ep-1", "del-1", true, 250)
	require.NoError(t, err)
	assert.Len(t, repo.events, 1)
	assert.Equal(t, "fault_result", repo.events[0].EventType)
	assert.True(t, repo.events[0].Recovered)
	assert.Equal(t, int64(250), repo.events[0].RecoveryTime)
}

// --- GetResilienceReport Tests ---

func TestGetResilienceReport_EmptyExperiments(t *testing.T) {
	repo := newMockChaosRepo()
	svc := NewService(repo, nil)
	defer svc.Close()

	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	report, err := svc.GetResilienceReport(context.Background(), "tenant-1", start, end)
	require.NoError(t, err)
	assert.Equal(t, 0.0, report.OverallScore)
	assert.Equal(t, "F", report.Grade)
	assert.Equal(t, 0, report.ExperimentCount)
}

func TestGetResilienceReport_WithExperiments(t *testing.T) {
	repo := newMockChaosRepo()
	svc := NewService(repo, nil)
	defer svc.Close()

	now := time.Now()
	completedAt := now.Add(-1 * time.Hour)
	repo.experiments["exp-1"] = &ChaosExperiment{
		ID: "exp-1", TenantID: "tenant-1", Status: StatusCompleted,
		CompletedAt: &completedAt,
		Results: &ExperimentResult{
			ResilienceScore: 85.0,
			ByEndpoint: []EndpointResult{
				{EndpointID: "ep-1", ResilienceScore: 90.0},
			},
		},
	}

	start := now.Add(-24 * time.Hour)
	end := now
	report, err := svc.GetResilienceReport(context.Background(), "tenant-1", start, end)
	require.NoError(t, err)
	assert.Equal(t, 85.0, report.OverallScore)
	assert.Equal(t, "B", report.Grade)
}

// --- scoreToGrade Tests ---

func TestScoreToGrade(t *testing.T) {
	tests := []struct {
		score float64
		grade string
	}{
		{95, "A"}, {90, "A"}, {85, "B"}, {80, "B"},
		{75, "C"}, {70, "C"}, {65, "D"}, {60, "D"},
		{50, "F"}, {0, "F"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.grade, scoreToGrade(tt.score), "score %.0f", tt.score)
	}
}

// --- calculateResults Tests ---

func TestCalculateResults_WithRecovery(t *testing.T) {
	events := []ChaosEvent{
		{EndpointID: "ep-1", InjectedFault: "latency", EventType: "fault_result", Recovered: true, RecoveryTime: 100},
		{EndpointID: "ep-1", InjectedFault: "latency", EventType: "fault_result", Recovered: true, RecoveryTime: 200},
		{EndpointID: "ep-1", InjectedFault: "latency", EventType: "fault_result", Recovered: false},
	}

	result := calculateResults(events)
	assert.Equal(t, int64(3), result.TotalDeliveries)
	assert.Equal(t, int64(3), result.AffectedDeliveries)
	assert.Equal(t, int64(2), result.SuccessfulRecovery)
	assert.Equal(t, int64(1), result.FailedRecovery)
	assert.InDelta(t, 150.0, result.AvgRecoveryTimeMs, 0.01)
	assert.InDelta(t, 66.67, result.ResilienceScore, 0.1)
}

func TestCalculateResults_Empty(t *testing.T) {
	result := calculateResults(nil)
	assert.Equal(t, int64(0), result.TotalDeliveries)
	assert.Equal(t, 0.0, result.ResilienceScore)
}

func TestCalculateResults_AllRecovered(t *testing.T) {
	events := []ChaosEvent{
		{EndpointID: "ep-1", InjectedFault: "latency", Recovered: true, RecoveryTime: 50},
		{EndpointID: "ep-1", InjectedFault: "latency", Recovered: true, RecoveryTime: 100},
	}
	result := calculateResults(events)
	assert.Equal(t, 100.0, result.ResilienceScore)
	assert.Contains(t, result.Observations[0], "Excellent")
}

// --- DeleteExperiment Tests ---

func TestDeleteExperiment_Running(t *testing.T) {
	repo := newMockChaosRepo()
	svc := NewService(repo, nil)
	defer svc.Close()

	req := &CreateExperimentRequest{Name: "del-run", Type: ExperimentLatency, Duration: 300, BlastRadius: BlastRadius{MaxAffectedEndpoints: 1, MaxAffectedDeliveries: 10}}
	exp, _ := svc.CreateExperiment(context.Background(), "t1", "u1", req)
	svc.StartExperiment(context.Background(), "t1", exp.ID)

	err := svc.DeleteExperiment(context.Background(), "t1", exp.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete running")
}

func TestDeleteExperiment_Pending(t *testing.T) {
	repo := newMockChaosRepo()
	svc := NewService(repo, nil)
	defer svc.Close()

	req := &CreateExperimentRequest{Name: "del-ok", Type: ExperimentLatency, Duration: 60, BlastRadius: BlastRadius{MaxAffectedEndpoints: 1, MaxAffectedDeliveries: 10}}
	exp, _ := svc.CreateExperiment(context.Background(), "t1", "u1", req)

	err := svc.DeleteExperiment(context.Background(), "t1", exp.ID)
	assert.NoError(t, err)
}

// --- ListExperiments Tests ---

func TestListExperiments_DefaultLimit(t *testing.T) {
	repo := newMockChaosRepo()
	svc := NewService(repo, nil)
	defer svc.Close()

	exps, total, err := svc.ListExperiments(context.Background(), "t1", nil, 0, 0)
	require.NoError(t, err)
	assert.Empty(t, exps)
	assert.Equal(t, 0, total)
}

func TestListExperiments_LimitCap(t *testing.T) {
	repo := newMockChaosRepo()
	svc := NewService(repo, nil)
	defer svc.Close()

	// Limit > 100 should be capped
	_, _, err := svc.ListExperiments(context.Background(), "t1", nil, 200, 0)
	assert.NoError(t, err)
}
