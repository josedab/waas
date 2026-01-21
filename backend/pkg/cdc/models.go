package cdc

import (
	"fmt"
	"time"
)

// ConnectorType defines the type of CDC connector
type ConnectorType string

const (
	ConnectorPostgreSQL ConnectorType = "postgresql"
	ConnectorMySQL      ConnectorType = "mysql"
	ConnectorMongoDB    ConnectorType = "mongodb"
	ConnectorSQLServer  ConnectorType = "sqlserver"
)

// ConnectorStatus represents the status of a connector
type ConnectorStatus string

const (
	StatusCreated    ConnectorStatus = "created"
	StatusStarting   ConnectorStatus = "starting"
	StatusRunning    ConnectorStatus = "running"
	StatusPaused     ConnectorStatus = "paused"
	StatusStopped    ConnectorStatus = "stopped"
	StatusError      ConnectorStatus = "error"
	StatusDeleting   ConnectorStatus = "deleting"
)

// ChangeOperation represents the type of change
type ChangeOperation string

const (
	OpInsert ChangeOperation = "insert"
	OpUpdate ChangeOperation = "update"
	OpDelete ChangeOperation = "delete"
)

// CDCConnector represents a CDC connector configuration
type CDCConnector struct {
	ID                string           `json:"id" db:"id"`
	TenantID          string           `json:"tenant_id" db:"tenant_id"`
	Name              string           `json:"name" db:"name"`
	Type              ConnectorType    `json:"type" db:"type"`
	Status            ConnectorStatus  `json:"status" db:"status"`
	ConnectionConfig  ConnectionConfig `json:"connection_config" db:"connection_config"`
	CaptureConfig     CaptureConfig    `json:"capture_config" db:"capture_config"`
	WebhookConfig     WebhookMapping   `json:"webhook_config" db:"webhook_config"`
	OffsetConfig      OffsetConfig     `json:"offset_config" db:"offset_config"`
	LastOffset        string           `json:"last_offset,omitempty" db:"last_offset"`
	LastEventAt       *time.Time       `json:"last_event_at,omitempty" db:"last_event_at"`
	ErrorMessage      string           `json:"error_message,omitempty" db:"error_message"`
	EventsProcessed   int64            `json:"events_processed" db:"events_processed"`
	EventsFailed      int64            `json:"events_failed" db:"events_failed"`
	CreatedAt         time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at" db:"updated_at"`
}

// ConnectionConfig holds database connection settings
type ConnectionConfig struct {
	Host             string            `json:"host"`
	Port             int               `json:"port"`
	Database         string            `json:"database"`
	Username         string            `json:"username"`
	Password         string            `json:"-"` // Never serialize
	SSLMode          string            `json:"ssl_mode,omitempty"`
	SSLCert          string            `json:"ssl_cert,omitempty"`
	SSLKey           string            `json:"ssl_key,omitempty"`
	SSLRootCert      string            `json:"ssl_root_cert,omitempty"`
	ReplicationSlot  string            `json:"replication_slot,omitempty"` // PostgreSQL
	PublicationName  string            `json:"publication_name,omitempty"` // PostgreSQL
	ServerID         int               `json:"server_id,omitempty"` // MySQL
	ExtraOptions     map[string]string `json:"extra_options,omitempty"`
}

// CaptureConfig defines what changes to capture
type CaptureConfig struct {
	Tables           []TableConfig     `json:"tables"`
	IncludeSchemas   []string          `json:"include_schemas,omitempty"`
	ExcludeSchemas   []string          `json:"exclude_schemas,omitempty"`
	Operations       []ChangeOperation `json:"operations"` // insert, update, delete
	SnapshotMode     string            `json:"snapshot_mode"` // initial, never, when_needed
	IncludeBeforeState bool            `json:"include_before_state"`
}

// TableConfig configures capture for a specific table
type TableConfig struct {
	Schema           string            `json:"schema,omitempty"`
	Table            string            `json:"table"`
	Columns          []string          `json:"columns,omitempty"` // Empty = all
	ExcludeColumns   []string          `json:"exclude_columns,omitempty"`
	PrimaryKey       []string          `json:"primary_key,omitempty"` // Override detection
	Conditions       string            `json:"conditions,omitempty"` // WHERE clause
	Operations       []ChangeOperation `json:"operations,omitempty"` // Override global
}

// WebhookMapping maps CDC events to webhook deliveries
type WebhookMapping struct {
	EndpointID       string            `json:"endpoint_id"`
	EventTypePattern string            `json:"event_type_pattern"` // e.g., "db.{schema}.{table}.{operation}"
	PayloadTemplate  string            `json:"payload_template,omitempty"` // JSON template
	TransformScript  string            `json:"transform_script,omitempty"` // JavaScript
	Headers          map[string]string `json:"headers,omitempty"`
	BatchConfig      *BatchConfig      `json:"batch_config,omitempty"`
}

// BatchConfig configures event batching
type BatchConfig struct {
	Enabled       bool          `json:"enabled"`
	MaxSize       int           `json:"max_size"`
	MaxWaitMs     int           `json:"max_wait_ms"`
	GroupByTable  bool          `json:"group_by_table"`
}

// OffsetConfig configures offset tracking
type OffsetConfig struct {
	StorageType      string `json:"storage_type"` // database, file, kafka
	FlushIntervalMs  int    `json:"flush_interval_ms"`
	CommitOnSuccess  bool   `json:"commit_on_success"`
}

// ChangeEvent represents a captured change event
type ChangeEvent struct {
	ID              string                 `json:"id"`
	ConnectorID     string                 `json:"connector_id"`
	TenantID        string                 `json:"tenant_id"`
	Timestamp       time.Time              `json:"timestamp"`
	Operation       ChangeOperation        `json:"operation"`
	Source          EventSource            `json:"source"`
	Before          map[string]interface{} `json:"before,omitempty"`
	After           map[string]interface{} `json:"after,omitempty"`
	Key             map[string]interface{} `json:"key"`
	Offset          string                 `json:"offset"`
	TransactionID   string                 `json:"transaction_id,omitempty"`
	EventType       string                 `json:"event_type"`
}

// EventSource identifies the source of a change
type EventSource struct {
	Connector  string `json:"connector"`
	Database   string `json:"database"`
	Schema     string `json:"schema"`
	Table      string `json:"table"`
	ServerTime int64  `json:"server_time"`
	TxID       int64  `json:"tx_id,omitempty"`
	LSN        string `json:"lsn,omitempty"` // PostgreSQL
	BinlogPos  string `json:"binlog_pos,omitempty"` // MySQL
}

// CDCMetrics holds connector metrics
type CDCMetrics struct {
	ConnectorID      string    `json:"connector_id"`
	TenantID         string    `json:"tenant_id"`
	Status           string    `json:"status"`
	EventsPerSecond  float64   `json:"events_per_second"`
	TotalEvents      int64     `json:"total_events"`
	FailedEvents     int64     `json:"failed_events"`
	LagMs            int64     `json:"lag_ms"`
	LastEventTime    time.Time `json:"last_event_time"`
	BytesProcessed   int64     `json:"bytes_processed"`
	TablesMonitored  int       `json:"tables_monitored"`
	PendingEvents    int64     `json:"pending_events"`
	ReplicationSlot  string    `json:"replication_slot,omitempty"`
}

// ConnectorHealth represents connector health status
type ConnectorHealth struct {
	ConnectorID     string    `json:"connector_id"`
	Healthy         bool      `json:"healthy"`
	LastCheck       time.Time `json:"last_check"`
	ConnectionOK    bool      `json:"connection_ok"`
	ReplicationOK   bool      `json:"replication_ok"`
	LagAcceptable   bool      `json:"lag_acceptable"`
	ErrorCount      int       `json:"error_count"`
	Warnings        []string  `json:"warnings,omitempty"`
}

// OffsetState tracks the current offset state
type OffsetState struct {
	ID            string    `json:"id" db:"id"`
	ConnectorID   string    `json:"connector_id" db:"connector_id"`
	TenantID      string    `json:"tenant_id" db:"tenant_id"`
	Offset        string    `json:"offset" db:"offset"`
	TransactionID string    `json:"transaction_id,omitempty" db:"transaction_id"`
	Committed     bool      `json:"committed" db:"committed"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// CreateConnectorRequest creates a new CDC connector
type CreateConnectorRequest struct {
	Name             string           `json:"name" binding:"required"`
	Type             ConnectorType    `json:"type" binding:"required"`
	ConnectionConfig ConnectionConfig `json:"connection_config" binding:"required"`
	CaptureConfig    CaptureConfig    `json:"capture_config" binding:"required"`
	WebhookConfig    WebhookMapping   `json:"webhook_config" binding:"required"`
	OffsetConfig     *OffsetConfig    `json:"offset_config,omitempty"`
}

// UpdateConnectorRequest updates a CDC connector
type UpdateConnectorRequest struct {
	Name             *string           `json:"name,omitempty"`
	ConnectionConfig *ConnectionConfig `json:"connection_config,omitempty"`
	CaptureConfig    *CaptureConfig    `json:"capture_config,omitempty"`
	WebhookConfig    *WebhookMapping   `json:"webhook_config,omitempty"`
}

// TestConnectionRequest tests database connectivity
type TestConnectionRequest struct {
	Type             ConnectorType    `json:"type" binding:"required"`
	ConnectionConfig ConnectionConfig `json:"connection_config" binding:"required"`
}

// TestConnectionResult holds test connection results
type TestConnectionResult struct {
	Success          bool     `json:"success"`
	Message          string   `json:"message"`
	Version          string   `json:"version,omitempty"`
	ReplicationOK    bool     `json:"replication_ok"`
	AvailableTables  []string `json:"available_tables,omitempty"`
	Warnings         []string `json:"warnings,omitempty"`
}

// SchemaInfo provides table schema information
type SchemaInfo struct {
	Schema   string       `json:"schema"`
	Table    string       `json:"table"`
	Columns  []ColumnInfo `json:"columns"`
	KeyColumns []string   `json:"key_columns"`
}

// ColumnInfo describes a table column
type ColumnInfo struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Nullable   bool   `json:"nullable"`
	PrimaryKey bool   `json:"primary_key"`
	Default    string `json:"default,omitempty"`
}

// EventHistory records processed events
type EventHistory struct {
	ID            string          `json:"id" db:"id"`
	ConnectorID   string          `json:"connector_id" db:"connector_id"`
	TenantID      string          `json:"tenant_id" db:"tenant_id"`
	EventID       string          `json:"event_id" db:"event_id"`
	Operation     ChangeOperation `json:"operation" db:"operation"`
	TableName     string          `json:"table_name" db:"table_name"`
	WebhookID     string          `json:"webhook_id,omitempty" db:"webhook_id"`
	DeliveryID    string          `json:"delivery_id,omitempty" db:"delivery_id"`
	Status        string          `json:"status" db:"status"`
	ErrorMessage  string          `json:"error_message,omitempty" db:"error_message"`
	Offset        string          `json:"offset" db:"offset"`
	ProcessedAt   time.Time       `json:"processed_at" db:"processed_at"`
}

// GetSupportedConnectors returns supported connector types
func GetSupportedConnectors() []ConnectorTypeInfo {
	return []ConnectorTypeInfo{
		{
			Type:        ConnectorPostgreSQL,
			DisplayName: "PostgreSQL",
			Description: "Capture changes from PostgreSQL using logical replication",
			MinVersion:  "10.0",
			Features:    []string{"logical_replication", "transaction_support", "schema_filtering"},
		},
		{
			Type:        ConnectorMySQL,
			DisplayName: "MySQL",
			Description: "Capture changes from MySQL using binlog",
			MinVersion:  "5.7",
			Features:    []string{"binlog", "transaction_support", "gtid"},
		},
		{
			Type:        ConnectorMongoDB,
			DisplayName: "MongoDB",
			Description: "Capture changes from MongoDB using change streams",
			MinVersion:  "4.0",
			Features:    []string{"change_streams", "resume_token", "document_filtering"},
		},
		{
			Type:        ConnectorSQLServer,
			DisplayName: "SQL Server",
			Description: "Capture changes from SQL Server using CDC",
			MinVersion:  "2016",
			Features:    []string{"cdc", "transaction_support"},
		},
	}
}

// ConnectorTypeInfo provides information about a connector type
type ConnectorTypeInfo struct {
	Type        ConnectorType `json:"type"`
	DisplayName string        `json:"display_name"`
	Description string        `json:"description"`
	MinVersion  string        `json:"min_version"`
	Features    []string      `json:"features"`
}

// DatabaseType is an alias for ConnectorType used in tests
type DatabaseType = ConnectorType

// Additional constants for test compatibility
const (
	DBPostgreSQL DatabaseType = ConnectorPostgreSQL
	DBMySQL      DatabaseType = ConnectorMySQL
	DBMongoDB    DatabaseType = ConnectorMongoDB
	DBSQLServer  DatabaseType = ConnectorSQLServer
)

// Additional status constants for test compatibility
const (
	ConnectorPending ConnectorStatus = StatusCreated
	ConnectorRunning ConnectorStatus = StatusRunning
	ConnectorPaused  ConnectorStatus = StatusPaused
	ConnectorFailed  ConnectorStatus = StatusError
	ConnectorStopped ConnectorStatus = StatusStopped
)

// ChangeType is an alias for ChangeOperation used in tests
type ChangeType = ChangeOperation

// Additional constants for test compatibility
const (
	ChangeInsert ChangeType = OpInsert
	ChangeUpdate ChangeType = OpUpdate
	ChangeDelete ChangeType = OpDelete
)

// Connector is an alias for CDCConnector used in tests
type Connector = CDCConnector

// EventMapping represents event-to-webhook mapping used in tests
type EventMapping struct {
	ID          string             `json:"id"`
	ConnectorID string             `json:"connector_id"`
	TenantID    string             `json:"tenant_id"`
	Table       string             `json:"table"`
	EventType   string             `json:"event_type"`
	Conditions  []MappingCondition `json:"conditions"`
	Transform   string             `json:"transform"`
	WebhookID   string             `json:"webhook_id"`
	Enabled     bool               `json:"enabled"`
}

// MappingCondition defines a condition for event mapping
type MappingCondition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    interface{} `json:"value"`
}

// Validate validates the connection configuration
func (c *ConnectionConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("host is required")
	}
	if c.Port <= 0 {
		return fmt.Errorf("port must be positive")
	}
	if c.Database == "" {
		return fmt.Errorf("database is required")
	}
	return nil
}
