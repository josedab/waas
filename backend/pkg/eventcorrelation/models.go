package eventcorrelation

import (
	"encoding/json"
	"time"
)

// Correlation state status constants
const (
	StatePending    = "pending"
	StateCorrelated = "correlated"
	StateExpired    = "expired"
)

// CorrelationRule defines a correlation pattern between two event types.
type CorrelationRule struct {
	ID             string    `json:"id" db:"id"`
	TenantID       string    `json:"tenant_id" db:"tenant_id"`
	Name           string    `json:"name" db:"name"`
	Description    string    `json:"description" db:"description"`
	TriggerEvent   string    `json:"trigger_event" db:"trigger_event"`
	FollowEvent    string    `json:"follow_event" db:"follow_event"`
	TimeWindowSec  int       `json:"time_window_sec" db:"time_window_sec"`
	MatchFields    []string  `json:"match_fields"`
	CompositeEvent string    `json:"composite_event" db:"composite_event"`
	IsEnabled      bool      `json:"is_enabled" db:"is_enabled"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// CorrelationState holds the pending state for a trigger event awaiting its follow event.
type CorrelationState struct {
	ID             string          `json:"id" db:"id"`
	RuleID         string          `json:"rule_id" db:"rule_id"`
	TenantID       string          `json:"tenant_id" db:"tenant_id"`
	TriggerEventID string          `json:"trigger_event_id" db:"trigger_event_id"`
	MatchKey       string          `json:"match_key" db:"match_key"`
	Payload        json.RawMessage `json:"payload" db:"payload"`
	Status         string          `json:"status" db:"status"`
	ExpiresAt      time.Time       `json:"expires_at" db:"expires_at"`
	CorrelatedAt   *time.Time      `json:"correlated_at,omitempty" db:"correlated_at"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
}

// CorrelationMatch records a successful correlation between two events.
type CorrelationMatch struct {
	ID               string    `json:"id" db:"id"`
	RuleID           string    `json:"rule_id" db:"rule_id"`
	TenantID         string    `json:"tenant_id" db:"tenant_id"`
	TriggerEventID   string    `json:"trigger_event_id" db:"trigger_event_id"`
	FollowEventID    string    `json:"follow_event_id" db:"follow_event_id"`
	MatchKey         string    `json:"match_key" db:"match_key"`
	CompositeEventID string    `json:"composite_event_id" db:"composite_event_id"`
	MatchedAt        time.Time `json:"matched_at" db:"matched_at"`
}

// CompositeEvent is the derived event emitted when a correlation matches.
type CompositeEvent struct {
	ID             string          `json:"id"`
	TenantID       string          `json:"tenant_id"`
	EventType      string          `json:"event_type"`
	TriggerPayload json.RawMessage `json:"trigger_payload"`
	FollowPayload  json.RawMessage `json:"follow_payload"`
	RuleID         string          `json:"rule_id"`
	CorrelatedAt   time.Time       `json:"correlated_at"`
}

// CorrelationStats holds aggregated correlation metrics.
type CorrelationStats struct {
	ActiveRules    int   `json:"active_rules"`
	PendingStates  int64 `json:"pending_states"`
	TotalMatches   int64 `json:"total_matches"`
	ExpiredStates  int64 `json:"expired_states"`
	AvgMatchTimeMs int64 `json:"avg_match_time_ms"`
}

// Request DTOs

type CreateRuleRequest struct {
	Name           string   `json:"name" binding:"required"`
	Description    string   `json:"description"`
	TriggerEvent   string   `json:"trigger_event" binding:"required"`
	FollowEvent    string   `json:"follow_event" binding:"required"`
	TimeWindowSec  int      `json:"time_window_sec"`
	MatchFields    []string `json:"match_fields"`
	CompositeEvent string   `json:"composite_event" binding:"required"`
}

type IngestEventRequest struct {
	EventID   string          `json:"event_id" binding:"required"`
	EventType string          `json:"event_type" binding:"required"`
	Payload   json.RawMessage `json:"payload" binding:"required"`
}
