package streaming

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TopicStatus represents the status of a managed topic
type TopicStatus string

const (
	TopicStatusActive   TopicStatus = "active"
	TopicStatusDeleting TopicStatus = "deleting"
	TopicStatusError    TopicStatus = "error"
)

// ManagedTopic represents a streaming topic managed through the admin API
type ManagedTopic struct {
	ID                string            `json:"id"`
	TenantID          string            `json:"tenant_id"`
	BridgeID          string            `json:"bridge_id"`
	Name              string            `json:"name"`
	StreamType        StreamType        `json:"stream_type"`
	Partitions        int               `json:"partitions"`
	ReplicationFactor int               `json:"replication_factor"`
	RetentionMs       int64             `json:"retention_ms"`
	CleanupPolicy     string            `json:"cleanup_policy"` // delete, compact, compact_delete
	Status            TopicStatus       `json:"status"`
	Config            map[string]string `json:"config,omitempty"`
	MessageCount      int64             `json:"message_count"`
	SizeBytes         int64             `json:"size_bytes"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

// CreateTopicRequest represents a request to create a managed topic
type CreateTopicRequest struct {
	Name              string            `json:"name" binding:"required"`
	BridgeID          string            `json:"bridge_id" binding:"required"`
	Partitions        int               `json:"partitions,omitempty"`
	ReplicationFactor int               `json:"replication_factor,omitempty"`
	RetentionMs       int64             `json:"retention_ms,omitempty"`
	CleanupPolicy     string            `json:"cleanup_policy,omitempty"`
	Config            map[string]string `json:"config,omitempty"`
}

// UpdateTopicRequest represents a request to update a topic
type UpdateTopicRequest struct {
	Partitions    *int              `json:"partitions,omitempty"`
	RetentionMs   *int64            `json:"retention_ms,omitempty"`
	CleanupPolicy *string           `json:"cleanup_policy,omitempty"`
	Config        map[string]string `json:"config,omitempty"`
}

// TopicAdminRepository defines storage for topic management
type TopicAdminRepository interface {
	CreateTopic(ctx context.Context, topic *ManagedTopic) error
	GetTopic(ctx context.Context, tenantID, topicID string) (*ManagedTopic, error)
	ListTopics(ctx context.Context, tenantID, bridgeID string) ([]ManagedTopic, error)
	UpdateTopic(ctx context.Context, topic *ManagedTopic) error
	DeleteTopic(ctx context.Context, tenantID, topicID string) error
}

// TopicAdmin provides admin API for topic management
type TopicAdmin struct {
	repo TopicAdminRepository
}

// NewTopicAdmin creates a new topic admin
func NewTopicAdmin(repo TopicAdminRepository) *TopicAdmin {
	return &TopicAdmin{repo: repo}
}

// CreateTopic creates a new managed topic
func (a *TopicAdmin) CreateTopic(ctx context.Context, tenantID string, req *CreateTopicRequest) (*ManagedTopic, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("%w: topic name is required", ErrInvalidConfig)
	}

	partitions := req.Partitions
	if partitions == 0 {
		partitions = 6
	}
	replication := req.ReplicationFactor
	if replication == 0 {
		replication = 3
	}
	retention := req.RetentionMs
	if retention == 0 {
		retention = 7 * 24 * 60 * 60 * 1000 // 7 days
	}
	cleanup := req.CleanupPolicy
	if cleanup == "" {
		cleanup = "delete"
	}

	now := time.Now()
	topic := &ManagedTopic{
		ID:                uuid.New().String(),
		TenantID:          tenantID,
		BridgeID:          req.BridgeID,
		Name:              req.Name,
		StreamType:        StreamTypeKafka,
		Partitions:        partitions,
		ReplicationFactor: replication,
		RetentionMs:       retention,
		CleanupPolicy:     cleanup,
		Status:            TopicStatusActive,
		Config:            req.Config,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := a.repo.CreateTopic(ctx, topic); err != nil {
		return nil, fmt.Errorf("failed to create topic: %w", err)
	}

	return topic, nil
}

// GetTopic retrieves a managed topic
func (a *TopicAdmin) GetTopic(ctx context.Context, tenantID, topicID string) (*ManagedTopic, error) {
	return a.repo.GetTopic(ctx, tenantID, topicID)
}

// ListTopics lists managed topics for a bridge
func (a *TopicAdmin) ListTopics(ctx context.Context, tenantID, bridgeID string) ([]ManagedTopic, error) {
	return a.repo.ListTopics(ctx, tenantID, bridgeID)
}

// UpdateTopic updates a managed topic
func (a *TopicAdmin) UpdateTopic(ctx context.Context, tenantID, topicID string, req *UpdateTopicRequest) (*ManagedTopic, error) {
	topic, err := a.repo.GetTopic(ctx, tenantID, topicID)
	if err != nil {
		return nil, err
	}

	if req.Partitions != nil {
		if *req.Partitions < topic.Partitions {
			return nil, fmt.Errorf("%w: cannot decrease partition count", ErrInvalidConfig)
		}
		topic.Partitions = *req.Partitions
	}
	if req.RetentionMs != nil {
		topic.RetentionMs = *req.RetentionMs
	}
	if req.CleanupPolicy != nil {
		topic.CleanupPolicy = *req.CleanupPolicy
	}
	if req.Config != nil {
		if topic.Config == nil {
			topic.Config = make(map[string]string)
		}
		for k, v := range req.Config {
			topic.Config[k] = v
		}
	}

	topic.UpdatedAt = time.Now()
	if err := a.repo.UpdateTopic(ctx, topic); err != nil {
		return nil, fmt.Errorf("failed to update topic: %w", err)
	}

	return topic, nil
}

// DeleteTopic deletes a managed topic
func (a *TopicAdmin) DeleteTopic(ctx context.Context, tenantID, topicID string) error {
	return a.repo.DeleteTopic(ctx, tenantID, topicID)
}
