package depgraph

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordDeliveryAndGraph(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	err := svc.RecordDelivery("tenant-1", "producer-1", "consumer-1", "order.created", true, 120.0)
	require.NoError(t, err)

	graph, err := svc.GetGraph("tenant-1")
	require.NoError(t, err)
	assert.Len(t, graph.Nodes, 2)
	assert.Len(t, graph.Edges, 1)
	assert.Equal(t, "healthy", graph.Edges[0].HealthStatus)
}

func TestImpactAnalysis(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	// Build a chain: A → B → C → D
	svc.RecordDelivery("t1", "A", "B", "evt", true, 100)
	svc.RecordDelivery("t1", "B", "C", "evt", true, 100)
	svc.RecordDelivery("t1", "C", "D", "evt", true, 100)

	analysis, err := svc.AnalyzeImpact("A")
	require.NoError(t, err)
	assert.Equal(t, 3, analysis.BlastRadius)
	assert.Equal(t, "medium", analysis.RiskLevel)
	assert.Len(t, analysis.DirectConsumers, 1)
	assert.Len(t, analysis.TransitiveClosure, 3)
}

func TestImpactAnalysisCycleProtection(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	// Create a cycle: A → B → C → A
	svc.RecordDelivery("t1", "A", "B", "evt", true, 100)
	svc.RecordDelivery("t1", "B", "C", "evt", true, 100)
	svc.RecordDelivery("t1", "C", "A", "evt", true, 100)

	analysis, err := svc.AnalyzeImpact("A")
	require.NoError(t, err)
	// Should not infinite loop
	assert.Equal(t, 2, analysis.BlastRadius) // B and C (A is already visited)
}

func TestAddDependencyValidation(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	// Self-referencing
	_, err := svc.AddDependency("t1", "A", "A", nil)
	assert.Error(t, err)

	// Missing IDs
	_, err = svc.AddDependency("t1", "", "B", nil)
	assert.Error(t, err)
}

func TestHealthStatusComputation(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	// 100% success rate
	for i := 0; i < 100; i++ {
		svc.RecordDelivery("t1", "P", "C", "evt", true, 50)
	}

	graph, _ := svc.GetGraph("t1")
	assert.Equal(t, "healthy", graph.Edges[0].HealthStatus)
}
