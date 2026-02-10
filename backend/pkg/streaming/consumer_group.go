package streaming

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ConsumerGroupStatus represents the status of a consumer group
type ConsumerGroupStatus string

const (
	ConsumerGroupStatusActive      ConsumerGroupStatus = "active"
	ConsumerGroupStatusRebalancing ConsumerGroupStatus = "rebalancing"
	ConsumerGroupStatusStopped     ConsumerGroupStatus = "stopped"
	ConsumerGroupStatusError       ConsumerGroupStatus = "error"
)

// ConsumerGroup represents a partitioned consumer group for parallel consumption
type ConsumerGroup struct {
	ID          string              `json:"id"`
	TenantID    string              `json:"tenant_id"`
	BridgeID    string              `json:"bridge_id"`
	Name        string              `json:"name"`
	Status      ConsumerGroupStatus `json:"status"`
	Partitions  int                 `json:"partitions"`
	Members     []ConsumerMember    `json:"members"`
	Strategy    string              `json:"strategy"` // round_robin, range, sticky
	MaxMembers  int                 `json:"max_members"`
	SessionTTL  time.Duration       `json:"session_ttl"`
	HeartbeatMs int                 `json:"heartbeat_ms"`
	Offsets     map[int]int64       `json:"offsets,omitempty"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

// ConsumerMember represents a member of a consumer group
type ConsumerMember struct {
	ID              string     `json:"id"`
	GroupID         string     `json:"group_id"`
	ClientID        string     `json:"client_id"`
	Host            string     `json:"host"`
	AssignedParts   []int      `json:"assigned_partitions"`
	LastHeartbeat   time.Time  `json:"last_heartbeat"`
	JoinedAt        time.Time  `json:"joined_at"`
	ProcessedCount  int64      `json:"processed_count"`
	LastProcessedAt *time.Time `json:"last_processed_at,omitempty"`
}

// CreateConsumerGroupRequest represents a request to create a consumer group
type CreateConsumerGroupRequest struct {
	Name        string `json:"name" binding:"required"`
	BridgeID    string `json:"bridge_id" binding:"required"`
	Partitions  int    `json:"partitions" binding:"required,min=1,max=256"`
	Strategy    string `json:"strategy,omitempty"`
	MaxMembers  int    `json:"max_members,omitempty"`
	SessionTTL  int    `json:"session_ttl_seconds,omitempty"`
	HeartbeatMs int    `json:"heartbeat_ms,omitempty"`
}

// ConsumerGroupManager manages partitioned consumer groups
type ConsumerGroupManager struct {
	repo   ConsumerGroupRepository
	groups map[string]*ConsumerGroup
	mu     sync.RWMutex
}

// ConsumerGroupRepository defines storage for consumer groups
type ConsumerGroupRepository interface {
	CreateConsumerGroup(ctx context.Context, group *ConsumerGroup) error
	GetConsumerGroup(ctx context.Context, tenantID, groupID string) (*ConsumerGroup, error)
	ListConsumerGroups(ctx context.Context, tenantID, bridgeID string) ([]ConsumerGroup, error)
	UpdateConsumerGroup(ctx context.Context, group *ConsumerGroup) error
	DeleteConsumerGroup(ctx context.Context, tenantID, groupID string) error
	AddMember(ctx context.Context, groupID string, member *ConsumerMember) error
	RemoveMember(ctx context.Context, groupID, memberID string) error
	UpdateMemberHeartbeat(ctx context.Context, groupID, memberID string) error
	CommitOffset(ctx context.Context, groupID string, partition int, offset int64) error
	GetOffsets(ctx context.Context, groupID string) (map[int]int64, error)
}

// NewConsumerGroupManager creates a new consumer group manager
func NewConsumerGroupManager(repo ConsumerGroupRepository) *ConsumerGroupManager {
	return &ConsumerGroupManager{
		repo:   repo,
		groups: make(map[string]*ConsumerGroup),
	}
}

// CreateGroup creates a new consumer group
func (m *ConsumerGroupManager) CreateGroup(ctx context.Context, tenantID string, req *CreateConsumerGroupRequest) (*ConsumerGroup, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("%w: consumer group name is required", ErrInvalidConfig)
	}
	if req.Partitions < 1 || req.Partitions > 256 {
		return nil, fmt.Errorf("%w: partitions must be between 1 and 256", ErrInvalidConfig)
	}

	strategy := req.Strategy
	if strategy == "" {
		strategy = "round_robin"
	}
	maxMembers := req.MaxMembers
	if maxMembers == 0 {
		maxMembers = req.Partitions
	}
	sessionTTL := time.Duration(req.SessionTTL) * time.Second
	if sessionTTL == 0 {
		sessionTTL = 30 * time.Second
	}
	heartbeatMs := req.HeartbeatMs
	if heartbeatMs == 0 {
		heartbeatMs = 3000
	}

	now := time.Now()
	group := &ConsumerGroup{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		BridgeID:    req.BridgeID,
		Name:        req.Name,
		Status:      ConsumerGroupStatusActive,
		Partitions:  req.Partitions,
		Members:     []ConsumerMember{},
		Strategy:    strategy,
		MaxMembers:  maxMembers,
		SessionTTL:  sessionTTL,
		HeartbeatMs: heartbeatMs,
		Offsets:     make(map[int]int64),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := m.repo.CreateConsumerGroup(ctx, group); err != nil {
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}

	m.mu.Lock()
	m.groups[group.ID] = group
	m.mu.Unlock()

	return group, nil
}

// JoinGroup adds a member to a consumer group and triggers rebalancing
func (m *ConsumerGroupManager) JoinGroup(ctx context.Context, tenantID, groupID, clientID, host string) (*ConsumerMember, error) {
	group, err := m.repo.GetConsumerGroup(ctx, tenantID, groupID)
	if err != nil {
		return nil, fmt.Errorf("consumer group not found: %w", err)
	}

	if len(group.Members) >= group.MaxMembers {
		return nil, fmt.Errorf("consumer group is full: max %d members", group.MaxMembers)
	}

	now := time.Now()
	member := &ConsumerMember{
		ID:            uuid.New().String(),
		GroupID:       groupID,
		ClientID:      clientID,
		Host:          host,
		JoinedAt:      now,
		LastHeartbeat: now,
	}

	if err := m.repo.AddMember(ctx, groupID, member); err != nil {
		return nil, fmt.Errorf("failed to add member: %w", err)
	}

	// Trigger partition rebalance
	group.Members = append(group.Members, *member)
	m.rebalancePartitions(group)
	group.Status = ConsumerGroupStatusActive
	group.UpdatedAt = now
	_ = m.repo.UpdateConsumerGroup(ctx, group)

	return member, nil
}

// rebalancePartitions reassigns partitions across members
func (m *ConsumerGroupManager) rebalancePartitions(group *ConsumerGroup) {
	if len(group.Members) == 0 {
		return
	}

	// Clear existing assignments
	for i := range group.Members {
		group.Members[i].AssignedParts = nil
	}

	switch group.Strategy {
	case "range":
		partsPerMember := group.Partitions / len(group.Members)
		extra := group.Partitions % len(group.Members)
		partition := 0
		for i := range group.Members {
			count := partsPerMember
			if i < extra {
				count++
			}
			for j := 0; j < count; j++ {
				group.Members[i].AssignedParts = append(group.Members[i].AssignedParts, partition)
				partition++
			}
		}
	default: // round_robin
		for p := 0; p < group.Partitions; p++ {
			idx := p % len(group.Members)
			group.Members[idx].AssignedParts = append(group.Members[idx].AssignedParts, p)
		}
	}
}

// LeaveGroup removes a member from a consumer group
func (m *ConsumerGroupManager) LeaveGroup(ctx context.Context, tenantID, groupID, memberID string) error {
	if err := m.repo.RemoveMember(ctx, groupID, memberID); err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}

	group, err := m.repo.GetConsumerGroup(ctx, tenantID, groupID)
	if err != nil {
		return err
	}

	m.rebalancePartitions(group)
	group.UpdatedAt = time.Now()
	return m.repo.UpdateConsumerGroup(ctx, group)
}

// CommitOffset commits an offset for a partition
func (m *ConsumerGroupManager) CommitOffset(ctx context.Context, groupID string, partition int, offset int64) error {
	return m.repo.CommitOffset(ctx, groupID, partition, offset)
}

// GetGroup retrieves a consumer group
func (m *ConsumerGroupManager) GetGroup(ctx context.Context, tenantID, groupID string) (*ConsumerGroup, error) {
	return m.repo.GetConsumerGroup(ctx, tenantID, groupID)
}

// ListGroups lists consumer groups for a bridge
func (m *ConsumerGroupManager) ListGroups(ctx context.Context, tenantID, bridgeID string) ([]ConsumerGroup, error) {
	return m.repo.ListConsumerGroups(ctx, tenantID, bridgeID)
}

// DeleteGroup deletes a consumer group
func (m *ConsumerGroupManager) DeleteGroup(ctx context.Context, tenantID, groupID string) error {
	m.mu.Lock()
	delete(m.groups, groupID)
	m.mu.Unlock()
	return m.repo.DeleteConsumerGroup(ctx, tenantID, groupID)
}
