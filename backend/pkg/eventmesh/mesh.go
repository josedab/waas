package eventmesh

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MeshNodeState represents the state of a node in the mesh
type MeshNodeState string

const (
	MeshNodeStateActive   MeshNodeState = "active"
	MeshNodeStateDraining MeshNodeState = "draining"
	MeshNodeStateFailed   MeshNodeState = "failed"
	MeshNodeStateJoining  MeshNodeState = "joining"
)

// RaftRole represents a node's role in Raft consensus
type RaftRole string

const (
	RaftRoleLeader    RaftRole = "leader"
	RaftRoleFollower  RaftRole = "follower"
	RaftRoleCandidate RaftRole = "candidate"
)

// MeshNode represents a WaaS instance in the webhook mesh network
type MeshNode struct {
	ID           string        `json:"id"`
	Address      string        `json:"address"`
	GRPCPort     int           `json:"grpc_port"`
	Region       string        `json:"region"`
	Zone         string        `json:"zone,omitempty"`
	State        MeshNodeState `json:"state"`
	RaftRole     RaftRole      `json:"raft_role"`
	RaftTerm     uint64        `json:"raft_term"`
	LastHeartbeat time.Time    `json:"last_heartbeat"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Capacity     *NodeCapacity `json:"capacity"`
	JoinedAt     time.Time     `json:"joined_at"`
}

// NodeCapacity describes the load capacity of a mesh node
type NodeCapacity struct {
	MaxEventsPerSec int     `json:"max_events_per_sec"`
	CurrentLoad     float64 `json:"current_load_pct"`
	QueueDepth      int64   `json:"queue_depth"`
	ActiveConns     int     `json:"active_connections"`
	MemoryUsagePct  float64 `json:"memory_usage_pct"`
}

// MeshTopology represents the full mesh topology
type MeshTopology struct {
	Nodes       []MeshNode     `json:"nodes"`
	Connections []MeshEdge     `json:"connections"`
	LeaderID    string         `json:"leader_id"`
	Term        uint64         `json:"term"`
	Regions     []RegionInfo   `json:"regions"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// MeshEdge represents a connection between two mesh nodes
type MeshEdge struct {
	FromNodeID string  `json:"from_node_id"`
	ToNodeID   string  `json:"to_node_id"`
	LatencyMs  float64 `json:"latency_ms"`
	Bandwidth  int64   `json:"bandwidth_mbps"`
	Status     string  `json:"status"` // connected, degraded, disconnected
}

// RegionInfo describes a region in the mesh
type RegionInfo struct {
	Name       string `json:"name"`
	NodeCount  int    `json:"node_count"`
	LeaderID   string `json:"leader_id,omitempty"`
	HealthPct  float64 `json:"health_pct"`
}

// MeshEvent represents an event routed through the mesh
type MeshEvent struct {
	ID          string            `json:"id"`
	SourceNode  string            `json:"source_node"`
	EventType   string            `json:"event_type"`
	Payload     []byte            `json:"payload"`
	Headers     map[string]string `json:"headers,omitempty"`
	TraceID     string            `json:"trace_id,omitempty"`
	HopCount    int               `json:"hop_count"`
	MaxHops     int               `json:"max_hops"`
	TTLSeconds  int               `json:"ttl_seconds"`
	CreatedAt   time.Time         `json:"created_at"`
}

// FailoverEvent records a failover occurrence
type FailoverEvent struct {
	ID              string    `json:"id"`
	FailedNodeID    string    `json:"failed_node_id"`
	ReplacementNode string    `json:"replacement_node_id"`
	Region          string    `json:"region"`
	Reason          string    `json:"reason"`
	EventsRerouted  int64     `json:"events_rerouted"`
	DowntimeMs      int64     `json:"downtime_ms"`
	OccurredAt      time.Time `json:"occurred_at"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
}

// ReplicationState tracks cross-region replication status
type ReplicationState struct {
	SourceRegion  string    `json:"source_region"`
	TargetRegion  string    `json:"target_region"`
	Status        string    `json:"status"` // synced, lagging, error
	LagMs         int64     `json:"lag_ms"`
	EventsPending int64     `json:"events_pending"`
	LastSyncedAt  time.Time `json:"last_synced_at"`
}

// SplitBrainResolution records a split-brain resolution event
type SplitBrainResolution struct {
	ID            string    `json:"id"`
	DetectedAt    time.Time `json:"detected_at"`
	ResolvedAt    time.Time `json:"resolved_at"`
	PartitionA    []string  `json:"partition_a_nodes"`
	PartitionB    []string  `json:"partition_b_nodes"`
	WinningLeader string    `json:"winning_leader"`
	Strategy      string    `json:"strategy"` // raft_term, node_count, region_priority
	EventsReconciled int64 `json:"events_reconciled"`
}

// MeshConfig holds mesh configuration
type MeshConfig struct {
	NodeID             string `json:"node_id"`
	ListenAddress      string `json:"listen_address"`
	GRPCPort           int    `json:"grpc_port"`
	Region             string `json:"region"`
	Zone               string `json:"zone,omitempty"`
	HeartbeatInterval  time.Duration `json:"heartbeat_interval"`
	ElectionTimeout    time.Duration `json:"election_timeout"`
	MaxHops            int    `json:"max_hops"`
	ReplicationEnabled bool   `json:"replication_enabled"`
	FailoverEnabled    bool   `json:"failover_enabled"`
}

// JoinMeshRequest is the request to join a node to the mesh
type JoinMeshRequest struct {
	Address  string            `json:"address" binding:"required"`
	GRPCPort int               `json:"grpc_port" binding:"required"`
	Region   string            `json:"region" binding:"required"`
	Zone     string            `json:"zone,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// MeshManager manages the webhook mesh network
type MeshManager struct {
	mu              sync.RWMutex
	config          MeshConfig
	localNode       MeshNode
	nodes           map[string]*MeshNode
	edges           map[string]*MeshEdge
	raftTerm        uint64
	raftRole        RaftRole
	leaderID        string
	failoverEvents  []FailoverEvent
	replicationState map[string]*ReplicationState
}

// NewMeshManager creates a new mesh manager
func NewMeshManager(config MeshConfig) *MeshManager {
	if config.HeartbeatInterval == 0 {
		config.HeartbeatInterval = 5 * time.Second
	}
	if config.ElectionTimeout == 0 {
		config.ElectionTimeout = 15 * time.Second
	}
	if config.MaxHops == 0 {
		config.MaxHops = 5
	}
	if config.NodeID == "" {
		config.NodeID = uuid.New().String()
	}

	localNode := MeshNode{
		ID:            config.NodeID,
		Address:       config.ListenAddress,
		GRPCPort:      config.GRPCPort,
		Region:        config.Region,
		Zone:          config.Zone,
		State:         MeshNodeStateActive,
		RaftRole:      RaftRoleFollower,
		LastHeartbeat: time.Now(),
		Capacity: &NodeCapacity{
			MaxEventsPerSec: 10000,
		},
		JoinedAt: time.Now(),
	}

	mm := &MeshManager{
		config:           config,
		localNode:        localNode,
		nodes:            map[string]*MeshNode{config.NodeID: &localNode},
		edges:            make(map[string]*MeshEdge),
		raftRole:         RaftRoleFollower,
		replicationState: make(map[string]*ReplicationState),
	}

	return mm
}

// JoinMesh adds a new node to the mesh network
func (mm *MeshManager) JoinMesh(ctx context.Context, req *JoinMeshRequest) (*MeshNode, error) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	nodeID := uuid.New().String()

	node := &MeshNode{
		ID:            nodeID,
		Address:       req.Address,
		GRPCPort:      req.GRPCPort,
		Region:        req.Region,
		Zone:          req.Zone,
		State:         MeshNodeStateJoining,
		RaftRole:      RaftRoleFollower,
		LastHeartbeat: time.Now(),
		Metadata:      req.Metadata,
		Capacity: &NodeCapacity{
			MaxEventsPerSec: 10000,
		},
		JoinedAt: time.Now(),
	}

	mm.nodes[nodeID] = node

	// Create edges to existing nodes in the same region (full mesh within region)
	for id, existingNode := range mm.nodes {
		if id == nodeID {
			continue
		}
		edgeKey := nodeID + ":" + id
		mm.edges[edgeKey] = &MeshEdge{
			FromNodeID: nodeID,
			ToNodeID:   id,
			LatencyMs:  estimateLatency(node.Region, existingNode.Region),
			Bandwidth:  1000,
			Status:     "connected",
		}
	}

	node.State = MeshNodeStateActive

	// If this is the first node becoming active (single node), elect as leader
	if len(mm.nodes) == 1 {
		node.RaftRole = RaftRoleLeader
		mm.raftRole = RaftRoleLeader
		mm.leaderID = nodeID
		mm.raftTerm = 1
	}

	return node, nil
}

// RemoveNode removes a node from the mesh
func (mm *MeshManager) RemoveNode(ctx context.Context, nodeID string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	node, ok := mm.nodes[nodeID]
	if !ok {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	node.State = MeshNodeStateDraining

	// Remove edges
	for key := range mm.edges {
		edge := mm.edges[key]
		if edge.FromNodeID == nodeID || edge.ToNodeID == nodeID {
			delete(mm.edges, key)
		}
	}

	delete(mm.nodes, nodeID)

	// Trigger leader election if leader was removed
	if mm.leaderID == nodeID {
		mm.electLeader()
	}

	return nil
}

// GetTopology returns the current mesh topology
func (mm *MeshManager) GetTopology(ctx context.Context) *MeshTopology {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	nodes := make([]MeshNode, 0, len(mm.nodes))
	for _, n := range mm.nodes {
		nodes = append(nodes, *n)
	}

	edges := make([]MeshEdge, 0, len(mm.edges))
	for _, e := range mm.edges {
		edges = append(edges, *e)
	}

	regionMap := make(map[string]*RegionInfo)
	for _, n := range mm.nodes {
		ri, ok := regionMap[n.Region]
		if !ok {
			ri = &RegionInfo{Name: n.Region, HealthPct: 100.0}
			regionMap[n.Region] = ri
		}
		ri.NodeCount++
		if n.RaftRole == RaftRoleLeader {
			ri.LeaderID = n.ID
		}
	}

	regions := make([]RegionInfo, 0, len(regionMap))
	for _, ri := range regionMap {
		regions = append(regions, *ri)
	}

	return &MeshTopology{
		Nodes:       nodes,
		Connections: edges,
		LeaderID:    mm.leaderID,
		Term:        mm.raftTerm,
		Regions:     regions,
		UpdatedAt:   time.Now(),
	}
}

// RouteToMesh routes an event through the mesh to the best target node
func (mm *MeshManager) RouteToMesh(ctx context.Context, event *MeshEvent) (*MeshNode, error) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	if event.HopCount >= event.MaxHops {
		return nil, fmt.Errorf("event exceeded max hops (%d)", event.MaxHops)
	}

	if event.MaxHops == 0 {
		event.MaxHops = mm.config.MaxHops
	}

	// Find the best target node (lowest load in appropriate region)
	var bestNode *MeshNode
	var bestScore float64 = 999999

	for _, node := range mm.nodes {
		if node.State != MeshNodeStateActive {
			continue
		}
		if node.ID == event.SourceNode {
			continue
		}

		score := node.Capacity.CurrentLoad
		// Prefer same-region nodes
		if node.Region != mm.localNode.Region {
			score += 50
		}
		if score < bestScore {
			bestScore = score
			bestNode = node
		}
	}

	if bestNode == nil {
		return nil, fmt.Errorf("no available nodes to route event")
	}

	event.HopCount++
	return bestNode, nil
}

// Heartbeat processes a heartbeat from a peer node
func (mm *MeshManager) Heartbeat(ctx context.Context, nodeID string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	node, ok := mm.nodes[nodeID]
	if !ok {
		return fmt.Errorf("unknown node: %s", nodeID)
	}

	node.LastHeartbeat = time.Now()
	return nil
}

// DetectFailures checks for nodes that have missed heartbeats and triggers failover
func (mm *MeshManager) DetectFailures(ctx context.Context) []FailoverEvent {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	threshold := time.Now().Add(-3 * mm.config.HeartbeatInterval)
	var failovers []FailoverEvent

	for _, node := range mm.nodes {
		if node.ID == mm.localNode.ID {
			continue
		}
		if node.State == MeshNodeStateActive && node.LastHeartbeat.Before(threshold) {
			node.State = MeshNodeStateFailed

			// Find replacement node
			replacement := mm.findReplacementNode(node)
			replacementID := ""
			if replacement != nil {
				replacementID = replacement.ID
			}

			failover := FailoverEvent{
				ID:              uuid.New().String(),
				FailedNodeID:    node.ID,
				ReplacementNode: replacementID,
				Region:          node.Region,
				Reason:          "heartbeat_timeout",
				DowntimeMs:      time.Since(node.LastHeartbeat).Milliseconds(),
				OccurredAt:      time.Now(),
			}

			failovers = append(failovers, failover)
			mm.failoverEvents = append(mm.failoverEvents, failover)

			// Trigger leader election if the failed node was the leader
			if mm.leaderID == node.ID {
				mm.electLeader()
			}
		}
	}

	return failovers
}

// ResolveSplitBrain resolves a split-brain scenario using Raft term comparison
func (mm *MeshManager) ResolveSplitBrain(ctx context.Context, partitionA, partitionB []string) (*SplitBrainResolution, error) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if len(partitionA) == 0 || len(partitionB) == 0 {
		return nil, fmt.Errorf("both partitions must have at least one node")
	}

	// Use majority quorum strategy: the larger partition wins
	var winningPartition []string
	var losingPartition []string
	strategy := "node_count"

	if len(partitionA) >= len(partitionB) {
		winningPartition = partitionA
		losingPartition = partitionB
	} else {
		winningPartition = partitionB
		losingPartition = partitionA
	}

	// Elect leader from winning partition
	winnerLeader := winningPartition[0]
	for _, nodeID := range winningPartition {
		if node, ok := mm.nodes[nodeID]; ok {
			if node.RaftRole == RaftRoleLeader {
				winnerLeader = nodeID
				break
			}
		}
	}

	// Demote nodes in losing partition to follower
	for _, nodeID := range losingPartition {
		if node, ok := mm.nodes[nodeID]; ok {
			node.RaftRole = RaftRoleFollower
		}
	}

	mm.raftTerm++
	mm.leaderID = winnerLeader
	if node, ok := mm.nodes[winnerLeader]; ok {
		node.RaftRole = RaftRoleLeader
		node.RaftTerm = mm.raftTerm
	}

	resolution := &SplitBrainResolution{
		ID:               uuid.New().String(),
		DetectedAt:       time.Now().Add(-time.Second),
		ResolvedAt:       time.Now(),
		PartitionA:       partitionA,
		PartitionB:       partitionB,
		WinningLeader:    winnerLeader,
		Strategy:         strategy,
		EventsReconciled: 0,
	}

	return resolution, nil
}

// GetReplicationState returns the current replication state between regions
func (mm *MeshManager) GetReplicationState(ctx context.Context) []ReplicationState {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	// Build replication state from region pairs
	regions := make(map[string]bool)
	for _, n := range mm.nodes {
		regions[n.Region] = true
	}

	var states []ReplicationState
	regionList := make([]string, 0, len(regions))
	for r := range regions {
		regionList = append(regionList, r)
	}

	for i := 0; i < len(regionList); i++ {
		for j := i + 1; j < len(regionList); j++ {
			key := regionList[i] + ":" + regionList[j]
			state, ok := mm.replicationState[key]
			if !ok {
				state = &ReplicationState{
					SourceRegion: regionList[i],
					TargetRegion: regionList[j],
					Status:       "synced",
					LagMs:        0,
					LastSyncedAt: time.Now(),
				}
			}
			states = append(states, *state)
		}
	}

	return states
}

// GetFailoverEvents returns recent failover events
func (mm *MeshManager) GetFailoverEvents(ctx context.Context) []FailoverEvent {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	result := make([]FailoverEvent, len(mm.failoverEvents))
	copy(result, mm.failoverEvents)
	return result
}

// electLeader performs Raft-style leader election among active nodes
func (mm *MeshManager) electLeader() {
	mm.raftTerm++

	var bestNode *MeshNode
	for _, node := range mm.nodes {
		if node.State != MeshNodeStateActive {
			continue
		}
		if bestNode == nil || node.JoinedAt.Before(bestNode.JoinedAt) {
			bestNode = node
		}
	}

	if bestNode != nil {
		bestNode.RaftRole = RaftRoleLeader
		bestNode.RaftTerm = mm.raftTerm
		mm.leaderID = bestNode.ID
	}
}

// findReplacementNode finds the best replacement for a failed node
func (mm *MeshManager) findReplacementNode(failed *MeshNode) *MeshNode {
	var best *MeshNode
	for _, node := range mm.nodes {
		if node.State != MeshNodeStateActive {
			continue
		}
		if node.ID == failed.ID {
			continue
		}
		// Prefer same region
		if best == nil || (node.Region == failed.Region && best.Region != failed.Region) {
			best = node
		}
	}
	return best
}

func estimateLatency(regionA, regionB string) float64 {
	if regionA == regionB {
		return 1.0
	}
	// Cross-region latency estimation
	return 50.0
}

// generateMeshToken creates a secure token for mesh node authentication
func generateMeshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "mesh_" + hex.EncodeToString(b), nil
}
