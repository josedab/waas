// Package eventsourcing provides event sourcing capabilities with replay
package eventsourcing

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrConcurrencyConflict = errors.New("concurrency conflict: stream version mismatch")
	ErrStreamNotFound      = errors.New("stream not found")
	ErrCheckpointNotFound  = errors.New("checkpoint not found")
)

// Event represents a domain event in the event store
type Event struct {
	SequenceID    int64                  `json:"sequence_id"`
	EventID       string                 `json:"event_id"`
	TenantID      string                 `json:"tenant_id"`
	StreamID      string                 `json:"stream_id"`
	StreamType    string                 `json:"stream_type"`
	EventType     string                 `json:"event_type"`
	Payload       json.RawMessage        `json:"payload"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Version       int                    `json:"version"`
	CorrelationID *string                `json:"correlation_id,omitempty"`
	CausationID   *string                `json:"causation_id,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
}

// EventInput represents input for appending an event
type EventInput struct {
	StreamID      string
	StreamType    string
	EventType     string
	Payload       interface{}
	Metadata      map[string]interface{}
	CorrelationID *string
	CausationID   *string
}

// Checkpoint represents a consumer's position in the event stream
type Checkpoint struct {
	ID              string                 `json:"id"`
	TenantID        string                 `json:"tenant_id"`
	ConsumerGroup   string                 `json:"consumer_group"`
	StreamID        *string                `json:"stream_id,omitempty"`
	LastSequenceID  int64                  `json:"last_sequence_id"`
	LastProcessedAt *time.Time             `json:"last_processed_at,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// Snapshot represents a point-in-time state snapshot
type Snapshot struct {
	ID         string          `json:"id"`
	TenantID   string          `json:"tenant_id"`
	StreamID   string          `json:"stream_id"`
	StreamType string          `json:"stream_type"`
	Version    int             `json:"version"`
	State      json.RawMessage `json:"state"`
	CreatedAt  time.Time       `json:"created_at"`
}

// Projection represents a read model projection
type Projection struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description,omitempty"`
	EventTypes     []string  `json:"event_types"`
	HandlerCode    string    `json:"handler_code,omitempty"`
	OutputSchema   any       `json:"output_schema,omitempty"`
	Status         string    `json:"status"`
	LastSequenceID int64     `json:"last_sequence_id"`
	LastError      string    `json:"last_error,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ReplayJob represents a replay operation
type ReplayJob struct {
	ID                string     `json:"id"`
	TenantID          string     `json:"tenant_id"`
	StreamID          *string    `json:"stream_id,omitempty"`
	StreamType        *string    `json:"stream_type,omitempty"`
	EventTypes        []string   `json:"event_types,omitempty"`
	FromSequenceID    *int64     `json:"from_sequence_id,omitempty"`
	ToSequenceID      *int64     `json:"to_sequence_id,omitempty"`
	TargetType        string     `json:"target_type"`
	TargetID          string     `json:"target_id"`
	Status            string     `json:"status"`
	CurrentSequenceID *int64     `json:"current_sequence_id,omitempty"`
	EventsProcessed   int        `json:"events_processed"`
	EventsTotal       *int       `json:"events_total,omitempty"`
	ErrorMessage      string     `json:"error_message,omitempty"`
	StartedAt         *time.Time `json:"started_at,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

// EventStore provides event sourcing operations
type EventStore interface {
	// Append adds events to a stream with optimistic concurrency
	Append(ctx context.Context, tenantID string, expectedVersion int, events ...EventInput) ([]Event, error)

	// Read reads events from a stream
	Read(ctx context.Context, tenantID, streamID string, fromVersion int, limit int) ([]Event, error)

	// ReadAll reads all events from a position
	ReadAll(ctx context.Context, tenantID string, fromSequence int64, limit int) ([]Event, error)

	// GetStreamVersion gets the current version of a stream
	GetStreamVersion(ctx context.Context, tenantID, streamID string) (int, error)
}

// CheckpointStore manages consumer checkpoints
type CheckpointStore interface {
	// GetCheckpoint retrieves a checkpoint
	GetCheckpoint(ctx context.Context, tenantID, consumerGroup string, streamID *string) (*Checkpoint, error)

	// SaveCheckpoint saves a checkpoint
	SaveCheckpoint(ctx context.Context, checkpoint *Checkpoint) error
}

// SnapshotStore manages state snapshots
type SnapshotStore interface {
	// GetSnapshot retrieves the latest snapshot
	GetSnapshot(ctx context.Context, tenantID, streamID string) (*Snapshot, error)

	// SaveSnapshot saves a snapshot
	SaveSnapshot(ctx context.Context, snapshot *Snapshot) error
}

// Aggregate is the base interface for event-sourced aggregates
type Aggregate interface {
	Apply(event Event)
	GetID() string
	GetVersion() int
}

// BaseAggregate provides common aggregate functionality
type BaseAggregate struct {
	ID      string
	Version int
	Changes []EventInput
}

func (a *BaseAggregate) GetID() string  { return a.ID }
func (a *BaseAggregate) GetVersion() int { return a.Version }

func (a *BaseAggregate) RecordChange(eventType string, payload interface{}) {
	a.Changes = append(a.Changes, EventInput{
		EventType: eventType,
		Payload:   payload,
	})
	a.Version++
}

func (a *BaseAggregate) GetUncommittedChanges() []EventInput {
	return a.Changes
}

func (a *BaseAggregate) ClearChanges() {
	a.Changes = nil
}
