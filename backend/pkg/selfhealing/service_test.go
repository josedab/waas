package selfhealing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordFailureAndHealing(t *testing.T) {
	repo := NewMemoryRepository()
	config := DefaultServiceConfig()
	config.FailureThreshold = 3
	svc := NewService(repo, config)

	// First 2 failures should not trigger healing
	for i := 0; i < 2; i++ {
		discovery, err := svc.RecordFailure("t1", "ep-1", "https://old.example.com/hook")
		require.NoError(t, err)
		assert.Nil(t, discovery)
	}

	// 3rd failure triggers healing (will fail to discover since no real server)
	discovery, err := svc.RecordFailure("t1", "ep-1", "https://old.example.com/hook")
	// Discovery will fail since there's no real .well-known endpoint
	assert.Nil(t, discovery)
	assert.Error(t, err)
}

func TestRecordSuccessResetsTracker(t *testing.T) {
	repo := NewMemoryRepository()
	config := DefaultServiceConfig()
	config.FailureThreshold = 5
	svc := NewService(repo, config)

	// Record some failures
	for i := 0; i < 3; i++ {
		svc.RecordFailure("t1", "ep-1", "https://example.com")
	}

	ft, _ := svc.GetFailureStatus("ep-1")
	assert.Equal(t, 3, ft.ConsecutiveFailures)

	// Success resets
	err := svc.RecordSuccess("ep-1")
	require.NoError(t, err)

	ft, _ = svc.GetFailureStatus("ep-1")
	assert.Equal(t, 0, ft.ConsecutiveFailures)
}

func TestGenerateWellKnownSpec(t *testing.T) {
	svc := NewService(NewMemoryRepository(), nil)

	spec := svc.GenerateWellKnownSpec([]WellKnownEndpoint{
		{
			Name:       "orders",
			URL:        "https://api.example.com/webhooks/orders",
			EventTypes: []string{"order.created", "order.updated"},
			Status:     "active",
		},
	})

	assert.Equal(t, "1.0", spec.Version)
	assert.Len(t, spec.Endpoints, 1)
	assert.Equal(t, "orders", spec.Endpoints[0].Name)
}

func TestMigrationEvents(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	events, err := svc.GetMigrationEvents("t1", 10)
	require.NoError(t, err)
	assert.Empty(t, events)
}

func TestDetectDegradedEndpoints(t *testing.T) {
	repo := NewMemoryRepository()
	config := DefaultServiceConfig()
	config.FailureThreshold = 10
	svc := NewService(repo, config)

	// Record some failures for ep-1
	for i := 0; i < 3; i++ {
		svc.RecordFailure("t1", "ep-1", "https://example.com")
	}

	statuses, err := svc.DetectDegradedEndpoints("t1")
	require.NoError(t, err)
	assert.Len(t, statuses, 1)
	assert.Equal(t, "degraded", statuses[0].Status)
}

func TestExecuteRemediation(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	for i := 0; i < 3; i++ {
		svc.RecordFailure("t1", "ep-1", "https://example.com")
	}

	action, err := svc.ExecuteRemediation("t1", "ep-1", "circuit_break")
	require.NoError(t, err)
	assert.Equal(t, "circuit_break", action.ActionType)
	assert.Equal(t, "applied", action.Status)
	assert.True(t, action.Automated)

	_, err = svc.ExecuteRemediation("t1", "ep-1", "invalid")
	assert.Error(t, err)
}

func TestTuneRetryPolicy(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	// Healthy endpoint
	result, err := svc.TuneRetryPolicy("t1", "ep-1")
	require.NoError(t, err)
	assert.Equal(t, 3, result.RecommendedRetries)
	assert.Equal(t, 1.0, result.HistoricalSuccessRate)

	// Degraded endpoint
	for i := 0; i < 12; i++ {
		svc.RecordFailure("t1", "ep-2", "https://example.com")
	}
	result, err = svc.TuneRetryPolicy("t1", "ep-2")
	require.NoError(t, err)
	assert.Greater(t, result.RecommendedRetries, 3)
	assert.Less(t, result.HistoricalSuccessRate, 1.0)
}

func TestAdjustConcurrency(t *testing.T) {
	svc := NewService(NewMemoryRepository(), nil)

	adj, err := svc.AdjustConcurrency("t1", "ep-1", 50)
	require.NoError(t, err)
	assert.Equal(t, 50, adj.NewConcurrency)

	_, err = svc.AdjustConcurrency("t1", "ep-1", 0)
	assert.Error(t, err)
}
