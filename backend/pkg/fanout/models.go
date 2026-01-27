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

// RoutingRule defines a routing rule with conditions and actions
type RoutingRule struct {
	ID          uuid.UUID       `json:"id" db:"id"`
	TenantID    uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	TopicID     uuid.UUID       `json:"topic_id" db:"topic_id"`
	Name        string          `json:"name" db:"name"`
	Description string          `json:"description,omitempty" db:"description"`
	Version     int             `json:"version" db:"version"`
	Conditions  []RuleCondition `json:"conditions" db:"conditions"`
	Actions     []RuleAction    `json:"actions" db:"actions"`
	Priority    int             `json:"priority" db:"priority"`
	Enabled     bool            `json:"enabled" db:"enabled"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
}

// RuleCondition defines a matching condition
type RuleCondition struct {
	Type       string `json:"type"`       // jsonpath, header, regex, event_type
	Expression string `json:"expression"` // JSONPath expr, header name, regex pattern
	Operator   string `json:"operator"`   // equals, contains, matches, gt, lt, exists
	Value      string `json:"value"`
}

// RuleAction defines what to do when conditions match
type RuleAction struct {
	Type          string            `json:"type"` // route, transform, filter, delay
	DestinationID string            `json:"destination_id,omitempty"`
	Transform     string            `json:"transform,omitempty"` // JS transform expression
	DelaySeconds  int               `json:"delay_seconds,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	RetryPolicy   *RetryPolicy      `json:"retry_policy,omitempty"`
}

// RetryPolicy defines retry behavior per target
type RetryPolicy struct {
	MaxRetries        int `json:"max_retries"`
	InitialDelayMs    int `json:"initial_delay_ms"`
	MaxDelayMs        int `json:"max_delay_ms"`
	BackoffMultiplier int `json:"backoff_multiplier"`
}

// RuleVersion tracks version history for rollback
type RuleVersion struct {
	ID         uuid.UUID       `json:"id"`
	RuleID     uuid.UUID       `json:"rule_id"`
	Version    int             `json:"version"`
	Conditions []RuleCondition `json:"conditions"`
	Actions    []RuleAction    `json:"actions"`
	CreatedAt  time.Time       `json:"created_at"`
	CreatedBy  string          `json:"created_by,omitempty"`
}

// RuleTestRequest represents a request to test a rule
type RuleTestRequest struct {
	Payload   json.RawMessage   `json:"payload" binding:"required"`
	Headers   map[string]string `json:"headers,omitempty"`
	EventType string            `json:"event_type,omitempty"`
}

// RuleTestResult represents the result of testing a rule
type RuleTestResult struct {
	RuleID     uuid.UUID         `json:"rule_id"`
	RuleName   string            `json:"rule_name"`
	Matched    bool              `json:"matched"`
	Conditions []ConditionResult `json:"conditions"`
	Actions    []RuleAction      `json:"triggered_actions,omitempty"`
}

// ConditionResult represents the evaluation result of a single condition
type ConditionResult struct {
	Condition   RuleCondition `json:"condition"`
	Matched     bool          `json:"matched"`
	ActualValue string        `json:"actual_value,omitempty"`
}

// CreateRuleRequest is the request to create a routing rule
type CreateRuleRequest struct {
	Name        string          `json:"name" binding:"required"`
	Description string          `json:"description,omitempty"`
	Conditions  []RuleCondition `json:"conditions" binding:"required"`
	Actions     []RuleAction    `json:"actions" binding:"required"`
	Priority    int             `json:"priority"`
}

// RollbackRuleRequest is the request to rollback a routing rule to a specific version
type RollbackRuleRequest struct {
	Version int `json:"version" binding:"required"`
}

// FanOutDeliveryRequest represents the request to fan-out deliver an event
type FanOutDeliveryRequest struct {
	Payload json.RawMessage   `json:"payload" binding:"required"`
	Headers map[string]string `json:"headers,omitempty"`
}

// FanOutDeliveryResult captures the results of parallel delivery
type FanOutDeliveryResult struct {
	EventID       uuid.UUID              `json:"event_id"`
	TopicID       uuid.UUID              `json:"topic_id"`
	TotalTargets  int                    `json:"total_targets"`
	Succeeded     int                    `json:"succeeded"`
	Failed        int                    `json:"failed"`
	Pending       int                    `json:"pending"`
	TargetResults []TargetDeliveryResult `json:"target_results"`
}

// TargetDeliveryResult represents the result of delivering to a single target
type TargetDeliveryResult struct {
	SubscriptionID uuid.UUID    `json:"subscription_id"`
	EndpointURL    string       `json:"endpoint_url"`
	Status         string       `json:"status"` // delivered, failed, pending, filtered
	RetryPolicy    *RetryPolicy `json:"retry_policy,omitempty"`
	ErrorMessage   string       `json:"error_message,omitempty"`
	DurationMs     int          `json:"duration_ms"`
}

// Delivery status constants for FanOutDelivery
const (
	DeliveryStatusDelivered = "delivered"
	DeliveryStatusFailed    = "failed"
	DeliveryStatusPending   = "pending"
	DeliveryStatusFiltered  = "filtered"
)
