package fanout

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Topic represents a fan-out topic that events can be published to
type Topic struct {
	ID             uuid.UUID `json:"id" db:"id"`
	TenantID       uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Name           string    `json:"name" db:"name"`
	Description    string    `json:"description" db:"description"`
	Status         string    `json:"status" db:"status"` // active, paused, archived
	MaxSubscribers int       `json:"max_subscribers" db:"max_subscribers"`
	RetentionDays  int       `json:"retention_days" db:"retention_days"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// Subscription links an endpoint to a topic with optional filtering
type Subscription struct {
	ID         uuid.UUID `json:"id" db:"id"`
	TopicID    uuid.UUID `json:"topic_id" db:"topic_id"`
	TenantID   uuid.UUID `json:"tenant_id" db:"tenant_id"`
	EndpointID uuid.UUID `json:"endpoint_id" db:"endpoint_id"`
	FilterExpr string    `json:"filter_expression" db:"filter_expression"` // JSONPath filter
	Active     bool      `json:"active" db:"active"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// TopicEvent represents an event published to a topic
type TopicEvent struct {
	ID          uuid.UUID       `json:"id" db:"id"`
	TopicID     uuid.UUID       `json:"topic_id" db:"topic_id"`
	TenantID    uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	EventType   string          `json:"event_type" db:"event_type"`
	Payload     json.RawMessage `json:"payload" db:"payload"`
	Metadata    json.RawMessage `json:"metadata" db:"metadata"`
	FanOutCount int             `json:"fan_out_count" db:"fan_out_count"`
	Status      string          `json:"status" db:"status"` // published, fan_out_complete, partial_failure
	PublishedAt time.Time       `json:"published_at" db:"published_at"`
}

// FanOutResult contains the results of a fan-out operation
type FanOutResult struct {
	EventID      uuid.UUID        `json:"event_id"`
	TotalTargets int              `json:"total_targets"`
	Delivered    int              `json:"delivered"`
	Failed       int              `json:"failed"`
	Filtered     int              `json:"filtered"` // Didn't match filter
	Results      []DeliveryTarget `json:"results"`
}

// DeliveryTarget represents the result of delivering to a single subscriber
type DeliveryTarget struct {
	SubscriptionID uuid.UUID `json:"subscription_id"`
	EndpointID     uuid.UUID `json:"endpoint_id"`
	Status         string    `json:"status"` // delivered, queued, filtered, failed
	Error          string    `json:"error,omitempty"`
}

// Topic status constants
const (
	TopicStatusActive   = "active"
	TopicStatusPaused   = "paused"
	TopicStatusArchived = "archived"
)

// Event status constants
const (
	EventStatusPublished      = "published"
	EventStatusFanOutComplete = "fan_out_complete"
	EventStatusPartialFailure = "partial_failure"
)

// Delivery target status constants
const (
	TargetStatusDelivered = "delivered"
	TargetStatusQueued    = "queued"
	TargetStatusFiltered  = "filtered"
	TargetStatusFailed    = "failed"
)

// CreateTopicRequest represents the request to create a topic
type CreateTopicRequest struct {
	Name           string `json:"name" binding:"required"`
	Description    string `json:"description"`
	MaxSubscribers int    `json:"max_subscribers"`
	RetentionDays  int    `json:"retention_days"`
}

// UpdateTopicRequest represents the request to update a topic
type UpdateTopicRequest struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	Status         string `json:"status"`
	MaxSubscribers int    `json:"max_subscribers"`
	RetentionDays  int    `json:"retention_days"`
}

// SubscribeRequest represents the request to subscribe an endpoint to a topic
type SubscribeRequest struct {
	EndpointID       string `json:"endpoint_id" binding:"required"`
	FilterExpression string `json:"filter_expression"`
}

// PublishRequest represents the request to publish an event to a topic
type PublishRequest struct {
	EventType string          `json:"event_type" binding:"required"`
	Payload   json.RawMessage `json:"payload" binding:"required"`
	Metadata  json.RawMessage `json:"metadata"`
}
