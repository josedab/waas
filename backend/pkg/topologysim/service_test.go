package topologysim

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateTopology(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	topology, err := svc.CreateTopology("t1", &CreateTopologyRequest{
		Name: "Test Topology",
		Endpoints: []SimEndpoint{
			{ID: "ep-1", Name: "Orders", FailureRate: 0.05, LatencyMean: 100, LatencyStdDev: 20},
			{ID: "ep-2", Name: "Payments", FailureRate: 0.10, LatencyMean: 200, LatencyStdDev: 50},
		},
		Traffic: []TrafficSource{
			{EventType: "order.created", TargetIDs: []string{"ep-1", "ep-2"}, RPS: 100, Duration: "5m"},
		},
	})

	require.NoError(t, err)
	assert.NotEmpty(t, topology.ID)
	assert.Len(t, topology.Endpoints, 2)
}

func TestRunSimulation(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	topology, _ := svc.CreateTopology("t1", &CreateTopologyRequest{
		Name: "Sim Test",
		Endpoints: []SimEndpoint{
			{
				ID: "ep-1", FailureRate: 0.1, LatencyMean: 100, LatencyStdDev: 20,
				RetryPolicy: &SimRetryPolicy{MaxRetries: 3, BackoffBase: 100, BackoffMax: 5000},
			},
		},
		Traffic: []TrafficSource{
			{EventType: "test.event", TargetIDs: []string{"ep-1"}, RPS: 10, Duration: "1s"},
		},
	})

	result, err := svc.RunSimulation(&SimulationConfig{
		TopologyID: topology.ID,
		Duration:   "1s",
		Seed:       42,
	})

	require.NoError(t, err)
	assert.Equal(t, "completed", result.Status)
	assert.Greater(t, result.TotalEvents, int64(0))
	assert.NotNil(t, result.CostEstimate)
}

func TestRunMonteCarloSimulation(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	topology, _ := svc.CreateTopology("t1", &CreateTopologyRequest{
		Name: "Monte Carlo",
		Endpoints: []SimEndpoint{
			{ID: "ep-1", FailureRate: 0.2, LatencyMean: 150, LatencyStdDev: 30},
		},
		Traffic: []TrafficSource{
			{EventType: "test", TargetIDs: []string{"ep-1"}, RPS: 50, Duration: "1s"},
		},
	})

	result, err := svc.RunSimulation(&SimulationConfig{
		TopologyID:     topology.ID,
		Duration:       "1s",
		MonteCarloRuns: 10,
		Seed:           42,
	})

	require.NoError(t, err)
	assert.Equal(t, "completed", result.Status)
}

func TestValidation(t *testing.T) {
	svc := NewService(NewMemoryRepository(), nil)

	_, err := svc.CreateTopology("t1", &CreateTopologyRequest{
		Name: "Bad",
	})
	assert.Error(t, err) // no endpoints

	_, err = svc.CreateTopology("t1", &CreateTopologyRequest{
		Name:      "Bad",
		Endpoints: []SimEndpoint{{ID: "ep-1"}},
	})
	assert.Error(t, err) // no traffic
}

func TestBottleneckDetection(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	topology, _ := svc.CreateTopology("t1", &CreateTopologyRequest{
		Name: "Bottleneck Test",
		Endpoints: []SimEndpoint{
			{ID: "ep-1", FailureRate: 0.0, LatencyMean: 10, MaxConcurrency: 5},
		},
		Traffic: []TrafficSource{
			{EventType: "test", TargetIDs: []string{"ep-1"}, RPS: 100, Duration: "1s"},
		},
		Constraints: &InfraConstraints{
			MaxQueueDepth: 10,
			MaxWorkers:    5,
		},
	})

	result, err := svc.RunSimulation(&SimulationConfig{
		TopologyID: topology.ID,
		Duration:   "1s",
		Seed:       42,
	})

	require.NoError(t, err)
	assert.Equal(t, "completed", result.Status)
}

func TestGenerateFanOutTopology(t *testing.T) {
	svc := NewService(NewMemoryRepository(), nil)

	topology, err := svc.GenerateTopology("t1", &GenerateTopologyRequest{
		Pattern:       PatternFanOut,
		Name:          "Fan-Out Test",
		EndpointCount: 5,
		RPS:           100,
	})
	require.NoError(t, err)
	assert.Len(t, topology.Endpoints, 5)
	assert.Len(t, topology.Traffic, 1)
	assert.Len(t, topology.Traffic[0].TargetIDs, 5)
}

func TestGenerateChainTopology(t *testing.T) {
	svc := NewService(NewMemoryRepository(), nil)

	topology, err := svc.GenerateTopology("t1", &GenerateTopologyRequest{
		Pattern:       PatternChain,
		Name:          "Chain Test",
		EndpointCount: 4,
	})
	require.NoError(t, err)
	assert.Len(t, topology.Endpoints, 4)
	assert.Len(t, topology.Traffic, 3) // 4 endpoints = 3 links
}

func TestGenerateMeshTopology(t *testing.T) {
	svc := NewService(NewMemoryRepository(), nil)

	topology, err := svc.GenerateTopology("t1", &GenerateTopologyRequest{
		Pattern:       PatternMesh,
		Name:          "Mesh Test",
		EndpointCount: 3,
	})
	require.NoError(t, err)
	assert.Len(t, topology.Endpoints, 3)
	assert.Len(t, topology.Traffic, 3) // each endpoint sends to 2 others
}

func TestSimulateFailureCascade(t *testing.T) {
	svc := NewService(NewMemoryRepository(), nil)

	topology, _ := svc.GenerateTopology("t1", &GenerateTopologyRequest{
		Pattern:       PatternChain,
		Name:          "Cascade Test",
		EndpointCount: 4,
	})

	result, err := svc.SimulateFailureCascade(topology.ID, "ep-1")
	require.NoError(t, err)
	assert.Equal(t, "ep-1", result.OriginEndpoint)
	assert.Greater(t, result.AffectedCount, 0)
}

func TestGenerateVisGraph(t *testing.T) {
	svc := NewService(NewMemoryRepository(), nil)

	topology, _ := svc.GenerateTopology("t1", &GenerateTopologyRequest{
		Pattern:       PatternFanOut,
		Name:          "Vis Test",
		EndpointCount: 3,
	})

	graph, err := svc.GenerateVisGraph(topology.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, graph.Nodes)
	assert.NotEmpty(t, graph.Edges)
}
