package streaming

import (
	"context"
	"errors"
	"time"
)

var (
	ErrBridgeNotFound      = errors.New("streaming bridge not found")
	ErrBridgeAlreadyExists = errors.New("streaming bridge already exists")
	ErrInvalidConfig       = errors.New("invalid bridge configuration")
	ErrConnectionFailed    = errors.New("failed to connect to streaming platform")
	ErrSchemaValidation    = errors.New("schema validation failed")
	ErrEventNotFound       = errors.New("stream event not found")
)

// Repository defines the interface for streaming bridge storage
type Repository interface {
	// Bridge CRUD
	CreateBridge(ctx context.Context, bridge *StreamingBridge) error
	GetBridge(ctx context.Context, tenantID, bridgeID string) (*StreamingBridge, error)
	GetBridgeByName(ctx context.Context, tenantID, name string) (*StreamingBridge, error)
	UpdateBridge(ctx context.Context, bridge *StreamingBridge) error
	DeleteBridge(ctx context.Context, tenantID, bridgeID string) error
	ListBridges(ctx context.Context, tenantID string, filters *BridgeFilters) ([]StreamingBridge, int, error)

	// Bridge credentials (encrypted storage)
	SaveCredentials(ctx context.Context, bridgeID string, credentials map[string]string) error
	GetCredentials(ctx context.Context, bridgeID string) (map[string]string, error)
	DeleteCredentials(ctx context.Context, bridgeID string) error

	// Event tracking
	SaveEvent(ctx context.Context, event *StreamEvent) error
	GetEvent(ctx context.Context, tenantID, eventID string) (*StreamEvent, error)
	ListEvents(ctx context.Context, tenantID, bridgeID string, filters *EventFilters) ([]StreamEvent, int, error)
	UpdateEventStatus(ctx context.Context, eventID, status, errorMsg string) error

	// Metrics
	SaveMetrics(ctx context.Context, metrics *BridgeMetrics) error
	GetLatestMetrics(ctx context.Context, tenantID, bridgeID string) (*BridgeMetrics, error)
	GetMetricsHistory(ctx context.Context, tenantID, bridgeID string, start, end time.Time) ([]BridgeMetrics, error)
	IncrementEventCounters(ctx context.Context, bridgeID string, eventsIn, eventsOut, eventsFailed int64) error
}

// BridgeFilters contains filter options for listing bridges
type BridgeFilters struct {
	StreamType *StreamType   `json:"stream_type,omitempty"`
	Direction  *Direction    `json:"direction,omitempty"`
	Status     *BridgeStatus `json:"status,omitempty"`
	Search     string        `json:"search,omitempty"`
	Page       int           `json:"page,omitempty"`
	PageSize   int           `json:"page_size,omitempty"`
}

// EventFilters contains filter options for listing events
type EventFilters struct {
	Status    string     `json:"status,omitempty"`
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	Page      int        `json:"page,omitempty"`
	PageSize  int        `json:"page_size,omitempty"`
}

// Producer defines the interface for producing messages to streaming platforms
type Producer interface {
	// Initialize the producer with bridge configuration
	Init(ctx context.Context, bridge *StreamingBridge, credentials map[string]string) error
	// Send a single message
	Send(ctx context.Context, event *StreamEvent) error
	// Send a batch of messages
	SendBatch(ctx context.Context, events []*StreamEvent) error
	// Flush pending messages
	Flush(ctx context.Context) error
	// Close the producer
	Close() error
	// Health check
	Healthy(ctx context.Context) bool
}

// Consumer defines the interface for consuming messages from streaming platforms
type Consumer interface {
	// Initialize the consumer with bridge configuration
	Init(ctx context.Context, bridge *StreamingBridge, credentials map[string]string) error
	// Start consuming messages (blocking)
	Start(ctx context.Context, handler EventHandler) error
	// Pause consumption
	Pause() error
	// Resume consumption
	Resume() error
	// Commit offset/checkpoint
	Commit(ctx context.Context, event *StreamEvent) error
	// Get current lag
	GetLag(ctx context.Context) (int64, error)
	// Close the consumer
	Close() error
	// Health check
	Healthy(ctx context.Context) bool
}

// EventHandler is called for each consumed event
type EventHandler func(ctx context.Context, event *StreamEvent) error

// SchemaRegistry defines the interface for schema registry operations
type SchemaRegistry interface {
	// RegisterSchema registers a new schema
	RegisterSchema(ctx context.Context, subject string, schema string, format SchemaFormat) (int, error)
	// GetSchema retrieves a schema by ID
	GetSchema(ctx context.Context, schemaID int) (string, error)
	// GetLatestSchema retrieves the latest schema for a subject
	GetLatestSchema(ctx context.Context, subject string) (string, int, error)
	// ValidateSchema validates data against a schema
	ValidateSchema(ctx context.Context, schemaID int, data []byte) error
	// Encode encodes data with schema
	Encode(ctx context.Context, schemaID int, data interface{}) ([]byte, error)
	// Decode decodes data with schema
	Decode(ctx context.Context, data []byte) (interface{}, int, error)
	// CheckCompatibility checks if a schema is compatible
	CheckCompatibility(ctx context.Context, subject string, schema string) (bool, error)
	// Close the registry client
	Close() error
}
