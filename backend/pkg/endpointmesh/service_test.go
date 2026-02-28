package endpointmesh

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	t.Parallel()
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)
	require.NotNil(t, svc)
	assert.Equal(t, 5, svc.config.FailureThreshold)
	assert.Equal(t, 3, svc.config.RecoveryThreshold)
}

func TestAddNode(t *testing.T) {
	t.Parallel()
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	node, err := svc.AddNode("tenant-1", &CreateMeshNodeRequest{
		EndpointID: "ep-1",
		URL:        "https://example.com/webhook",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, node.ID)
	assert.Equal(t, "tenant-1", node.TenantID)
	assert.Equal(t, StatusHealthy, node.Status)
	assert.Equal(t, 1.0, node.HealthScore)
	assert.Equal(t, CircuitClosed, node.CircuitState.State)
}

func TestRecordHealthCheck_Success(t *testing.T) {
	t.Parallel()
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	node, _ := svc.AddNode("tenant-1", &CreateMeshNodeRequest{
		EndpointID: "ep-1",
		URL:        "https://example.com/webhook",
	})

	hc, err := svc.RecordHealthCheck(node.ID, 200, 42, true, "")
	require.NoError(t, err)
	assert.True(t, hc.Success)
	assert.Equal(t, int64(42), hc.LatencyMs)

	updated, _ := svc.GetNode(node.ID)
	assert.Equal(t, StatusHealthy, updated.Status)
	assert.Equal(t, 0, updated.ConsecutiveFailures)
}

func TestRecordHealthCheck_FailureTriggersCircuitOpen(t *testing.T) {
	t.Parallel()
	repo := NewMemoryRepository()
	config := DefaultMeshConfig()
	config.FailureThreshold = 3
	svc := NewService(repo, config)

	node, _ := svc.AddNode("tenant-1", &CreateMeshNodeRequest{
		EndpointID: "ep-1",
		URL:        "https://example.com/webhook",
	})

	// First 2 failures: degraded
	for i := 0; i < 2; i++ {
		_, err := svc.RecordHealthCheck(node.ID, 500, 100, false, "server error")
		require.NoError(t, err)
	}

	degraded, _ := svc.GetNode(node.ID)
	assert.Equal(t, StatusDegraded, degraded.Status)
	assert.Equal(t, 2, degraded.ConsecutiveFailures)

	// 3rd failure opens circuit
	_, err := svc.RecordHealthCheck(node.ID, 500, 100, false, "server error")
	require.NoError(t, err)

	opened, _ := svc.GetNode(node.ID)
	assert.Equal(t, StatusCircuitOpen, opened.Status)
	assert.Equal(t, CircuitOpen, opened.CircuitState.State)
	assert.Equal(t, 0.0, opened.HealthScore)
}

func TestRouteRequest_HealthyNode(t *testing.T) {
	t.Parallel()
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	node, _ := svc.AddNode("tenant-1", &CreateMeshNodeRequest{
		EndpointID: "ep-1",
		URL:        "https://example.com/webhook",
	})

	routed, err := svc.RouteRequest(node.ID)
	require.NoError(t, err)
	assert.Equal(t, node.ID, routed.ID)
}

func TestRouteRequest_CircuitOpen_UsesFallback(t *testing.T) {
	t.Parallel()
	repo := NewMemoryRepository()
	config := DefaultMeshConfig()
	config.FailureThreshold = 2
	config.CircuitOpenDuration = 10 * time.Minute
	svc := NewService(repo, config)

	primary, _ := svc.AddNode("tenant-1", &CreateMeshNodeRequest{
		EndpointID: "ep-1",
		URL:        "https://primary.example.com/webhook",
	})
	fallback, _ := svc.AddNode("tenant-1", &CreateMeshNodeRequest{
		EndpointID: "ep-2",
		URL:        "https://fallback.example.com/webhook",
	})

	// Set fallback
	err := svc.SetFallback(primary.ID, fallback.ID)
	require.NoError(t, err)

	// Open the circuit on primary
	for i := 0; i < 2; i++ {
		svc.RecordHealthCheck(primary.ID, 500, 100, false, "down")
	}

	// Route should use fallback
	routed, err := svc.RouteRequest(primary.ID)
	require.NoError(t, err)
	assert.Equal(t, fallback.ID, routed.ID)

	// Verify reroute event was recorded
	events, err := svc.ListRerouteEvents("tenant-1", 10)
	require.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, "circuit_open", events[0].Reason)
}

func TestRecoverNode(t *testing.T) {
	t.Parallel()
	repo := NewMemoryRepository()
	config := DefaultMeshConfig()
	config.FailureThreshold = 2
	svc := NewService(repo, config)

	node, _ := svc.AddNode("tenant-1", &CreateMeshNodeRequest{
		EndpointID: "ep-1",
		URL:        "https://example.com/webhook",
	})

	// Open circuit
	for i := 0; i < 2; i++ {
		svc.RecordHealthCheck(node.ID, 500, 100, false, "down")
	}

	opened, _ := svc.GetNode(node.ID)
	assert.Equal(t, StatusCircuitOpen, opened.Status)

	// Recover
	recovered, err := svc.RecoverNode(node.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusRecovering, recovered.Status)
	assert.Equal(t, CircuitHalfOpen, recovered.CircuitState.State)
	assert.Equal(t, 0, recovered.ConsecutiveFailures)
}
