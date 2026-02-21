package eventmesh

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestMeshManager() *MeshManager {
	return NewMeshManager(MeshConfig{
		ListenAddress:     "127.0.0.1",
		GRPCPort:          9090,
		Region:            "us-east-1",
		HeartbeatInterval: 1 * time.Second,
		ElectionTimeout:   3 * time.Second,
	})
}

func TestMeshManager_JoinMesh(t *testing.T) {
	mm := newTestMeshManager()
	ctx := context.Background()

	node, err := mm.JoinMesh(ctx, &JoinMeshRequest{
		Address:  "10.0.0.2",
		GRPCPort: 9090,
		Region:   "us-east-1",
	})

	require.NoError(t, err)
	assert.Equal(t, MeshNodeStateActive, node.State)
	assert.Equal(t, RaftRoleFollower, node.RaftRole)
	assert.NotEmpty(t, node.ID)
}

func TestMeshManager_GetTopology(t *testing.T) {
	mm := newTestMeshManager()
	ctx := context.Background()

	// Add a second node
	_, err := mm.JoinMesh(ctx, &JoinMeshRequest{
		Address:  "10.0.0.2",
		GRPCPort: 9090,
		Region:   "us-east-1",
	})
	require.NoError(t, err)

	topo := mm.GetTopology(ctx)
	assert.Equal(t, 2, len(topo.Nodes))
	assert.True(t, len(topo.Connections) > 0)
	assert.True(t, len(topo.Regions) > 0)
}

func TestMeshManager_RemoveNode(t *testing.T) {
	mm := newTestMeshManager()
	ctx := context.Background()

	node, err := mm.JoinMesh(ctx, &JoinMeshRequest{
		Address:  "10.0.0.2",
		GRPCPort: 9090,
		Region:   "us-east-1",
	})
	require.NoError(t, err)

	err = mm.RemoveNode(ctx, node.ID)
	assert.NoError(t, err)

	topo := mm.GetTopology(ctx)
	assert.Equal(t, 1, len(topo.Nodes)) // Only local node remains
}

func TestMeshManager_RemoveNode_NotFound(t *testing.T) {
	mm := newTestMeshManager()
	ctx := context.Background()

	err := mm.RemoveNode(ctx, "nonexistent")
	assert.Error(t, err)
}

func TestMeshManager_RouteToMesh(t *testing.T) {
	mm := newTestMeshManager()
	ctx := context.Background()

	// Add a target node
	_, err := mm.JoinMesh(ctx, &JoinMeshRequest{
		Address:  "10.0.0.2",
		GRPCPort: 9090,
		Region:   "us-east-1",
	})
	require.NoError(t, err)

	event := &MeshEvent{
		ID:         "evt-1",
		SourceNode: mm.localNode.ID,
		EventType:  "order.created",
		Payload:    []byte(`{"order_id": "123"}`),
		MaxHops:    5,
		CreatedAt:  time.Now(),
	}

	target, err := mm.RouteToMesh(ctx, event)
	require.NoError(t, err)
	assert.NotNil(t, target)
	assert.NotEqual(t, mm.localNode.ID, target.ID)
	assert.Equal(t, 1, event.HopCount)
}

func TestMeshManager_RouteToMesh_MaxHopsExceeded(t *testing.T) {
	mm := newTestMeshManager()
	ctx := context.Background()

	event := &MeshEvent{
		ID:         "evt-1",
		SourceNode: mm.localNode.ID,
		EventType:  "order.created",
		HopCount:   5,
		MaxHops:    5,
	}

	_, err := mm.RouteToMesh(ctx, event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max hops")
}

func TestMeshManager_Heartbeat(t *testing.T) {
	mm := newTestMeshManager()
	ctx := context.Background()

	node, _ := mm.JoinMesh(ctx, &JoinMeshRequest{
		Address: "10.0.0.2", GRPCPort: 9090, Region: "us-east-1",
	})

	err := mm.Heartbeat(ctx, node.ID)
	assert.NoError(t, err)

	err = mm.Heartbeat(ctx, "unknown")
	assert.Error(t, err)
}

func TestMeshManager_DetectFailures(t *testing.T) {
	mm := NewMeshManager(MeshConfig{
		ListenAddress:     "127.0.0.1",
		GRPCPort:          9090,
		Region:            "us-east-1",
		HeartbeatInterval: 1 * time.Millisecond, // Very short for testing
	})
	ctx := context.Background()

	node, _ := mm.JoinMesh(ctx, &JoinMeshRequest{
		Address: "10.0.0.2", GRPCPort: 9090, Region: "us-east-1",
	})

	// Set heartbeat to the past to simulate failure
	mm.mu.Lock()
	mm.nodes[node.ID].LastHeartbeat = time.Now().Add(-10 * time.Second)
	mm.mu.Unlock()

	failovers := mm.DetectFailures(ctx)
	assert.True(t, len(failovers) > 0)
	assert.Equal(t, node.ID, failovers[0].FailedNodeID)
	assert.Equal(t, "heartbeat_timeout", failovers[0].Reason)
}

func TestMeshManager_ResolveSplitBrain(t *testing.T) {
	mm := newTestMeshManager()
	ctx := context.Background()

	nodeA, _ := mm.JoinMesh(ctx, &JoinMeshRequest{
		Address: "10.0.0.2", GRPCPort: 9090, Region: "us-east-1",
	})
	nodeB, _ := mm.JoinMesh(ctx, &JoinMeshRequest{
		Address: "10.0.0.3", GRPCPort: 9090, Region: "us-east-1",
	})
	nodeC, _ := mm.JoinMesh(ctx, &JoinMeshRequest{
		Address: "10.0.0.4", GRPCPort: 9090, Region: "us-west-2",
	})

	resolution, err := mm.ResolveSplitBrain(ctx,
		[]string{mm.localNode.ID, nodeA.ID, nodeB.ID},
		[]string{nodeC.ID},
	)

	require.NoError(t, err)
	assert.Equal(t, "node_count", resolution.Strategy)
	assert.NotEmpty(t, resolution.WinningLeader)
}

func TestMeshManager_ResolveSplitBrain_EmptyPartition(t *testing.T) {
	mm := newTestMeshManager()
	ctx := context.Background()

	_, err := mm.ResolveSplitBrain(ctx, []string{}, []string{"node-1"})
	assert.Error(t, err)
}

func TestMeshManager_CrossRegionTopology(t *testing.T) {
	mm := newTestMeshManager()
	ctx := context.Background()

	// Add nodes in different regions
	_, _ = mm.JoinMesh(ctx, &JoinMeshRequest{
		Address: "10.0.1.1", GRPCPort: 9090, Region: "eu-west-1",
	})
	_, _ = mm.JoinMesh(ctx, &JoinMeshRequest{
		Address: "10.0.2.1", GRPCPort: 9090, Region: "ap-southeast-1",
	})

	topo := mm.GetTopology(ctx)
	assert.Equal(t, 3, len(topo.Nodes))

	// Should have 3 regions
	regionNames := make(map[string]bool)
	for _, r := range topo.Regions {
		regionNames[r.Name] = true
	}
	assert.True(t, regionNames["us-east-1"])
	assert.True(t, regionNames["eu-west-1"])
	assert.True(t, regionNames["ap-southeast-1"])
}

func TestMeshManager_GetReplicationState(t *testing.T) {
	mm := newTestMeshManager()
	ctx := context.Background()

	_, _ = mm.JoinMesh(ctx, &JoinMeshRequest{
		Address: "10.0.1.1", GRPCPort: 9090, Region: "eu-west-1",
	})

	states := mm.GetReplicationState(ctx)
	assert.True(t, len(states) > 0)
	assert.Equal(t, "synced", states[0].Status)
}

func TestMeshManager_LeaderElection(t *testing.T) {
	mm := newTestMeshManager()
	ctx := context.Background()

	// Add two more nodes
	nodeA, _ := mm.JoinMesh(ctx, &JoinMeshRequest{
		Address: "10.0.0.2", GRPCPort: 9090, Region: "us-east-1",
	})
	nodeB, _ := mm.JoinMesh(ctx, &JoinMeshRequest{
		Address: "10.0.0.3", GRPCPort: 9090, Region: "us-east-1",
	})

	// Set local node as leader explicitly
	mm.mu.Lock()
	mm.leaderID = mm.localNode.ID
	mm.nodes[mm.localNode.ID].RaftRole = RaftRoleLeader
	mm.mu.Unlock()

	// Remove leader to trigger re-election
	_ = mm.RemoveNode(ctx, mm.localNode.ID)

	newTopo := mm.GetTopology(ctx)
	assert.NotEmpty(t, newTopo.LeaderID)
	// Leader should be one of the remaining nodes
	assert.True(t, newTopo.LeaderID == nodeA.ID || newTopo.LeaderID == nodeB.ID)
}

func TestGenerateMeshToken(t *testing.T) {
	token, err := generateMeshToken()
	require.NoError(t, err)
	assert.True(t, len(token) > 10)
	assert.Contains(t, token, "mesh_")
}
