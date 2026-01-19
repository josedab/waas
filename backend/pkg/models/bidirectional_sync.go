package models

import (
	"time"

	"github.com/google/uuid"
)

// Sync mode constants
const (
	SyncModeRequestResponse    = "request_response"
	SyncModeEventAcknowledgment = "event_acknowledgment"
	SyncModeStateSync          = "state_sync"
)

// Sync transaction state constants
const (
	SyncStatesPending          = "pending"
	SyncStatesAwaitingResponse = "awaiting_response"
	SyncStatesCompleted        = "completed"
	SyncStatesTimeout          = "timeout"
	SyncStatesFailed           = "failed"
)

// Sync status constants for state records
const (
	SyncStatusSynced      = "synced"
	SyncStatusPendingPush = "pending_push"
	SyncStatusPendingPull = "pending_pull"
	SyncStatusConflict    = "conflict"
)

// Acknowledgment type constants
const (
	AckTypeReceived  = "received"
	AckTypeProcessed = "processed"
	AckTypeRejected  = "rejected"
)

// Conflict resolution strategy constants
const (
	ConflictStrategyLocalWins  = "local_wins"
	ConflictStrategyRemoteWins = "remote_wins"
	ConflictStrategyMerge      = "merge"
	ConflictStrategyManual     = "manual"
)

// WebhookSyncConfig represents a bi-directional sync configuration
type WebhookSyncConfig struct {
	ID                 uuid.UUID              `json:"id" db:"id"`
	TenantID           uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	Name               string                 `json:"name" db:"name"`
	Description        string                 `json:"description,omitempty" db:"description"`
	OutboundEndpointID *uuid.UUID             `json:"outbound_endpoint_id,omitempty" db:"outbound_endpoint_id"`
	InboundEventType   string                 `json:"inbound_event_type,omitempty" db:"inbound_event_type"`
	SyncMode           string                 `json:"sync_mode" db:"sync_mode"`
	TimeoutSeconds     int                    `json:"timeout_seconds" db:"timeout_seconds"`
	RetryOnTimeout     bool                   `json:"retry_on_timeout" db:"retry_on_timeout"`
	MaxRetries         int                    `json:"max_retries" db:"max_retries"`
	CorrelationConfig  map[string]interface{} `json:"correlation_config" db:"correlation_config"`
	Enabled            bool                   `json:"enabled" db:"enabled"`
	CreatedAt          time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at" db:"updated_at"`
}

// SyncTransaction represents a request-response pair
type SyncTransaction struct {
	ID                 uuid.UUID              `json:"id" db:"id"`
	TenantID           uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	ConfigID           uuid.UUID              `json:"config_id" db:"config_id"`
	CorrelationID      string                 `json:"correlation_id" db:"correlation_id"`
	OutboundEventID    *uuid.UUID             `json:"outbound_event_id,omitempty" db:"outbound_event_id"`
	InboundEventID     *uuid.UUID             `json:"inbound_event_id,omitempty" db:"inbound_event_id"`
	State              string                 `json:"state" db:"state"`
	RequestPayload     map[string]interface{} `json:"request_payload,omitempty" db:"request_payload"`
	ResponsePayload    map[string]interface{} `json:"response_payload,omitempty" db:"response_payload"`
	RequestSentAt      time.Time              `json:"request_sent_at" db:"request_sent_at"`
	ResponseReceivedAt *time.Time             `json:"response_received_at,omitempty" db:"response_received_at"`
	TimeoutAt          *time.Time             `json:"timeout_at,omitempty" db:"timeout_at"`
	RetryCount         int                    `json:"retry_count" db:"retry_count"`
	ErrorMessage       string                 `json:"error_message,omitempty" db:"error_message"`
	Metadata           map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt          time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at" db:"updated_at"`
}

// SyncStateRecord represents a state synchronization record
type SyncStateRecord struct {
	ID                 uuid.UUID              `json:"id" db:"id"`
	TenantID           uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	ConfigID           uuid.UUID              `json:"config_id" db:"config_id"`
	ResourceType       string                 `json:"resource_type" db:"resource_type"`
	ResourceID         string                 `json:"resource_id" db:"resource_id"`
	LocalState         map[string]interface{} `json:"local_state" db:"local_state"`
	RemoteState        map[string]interface{} `json:"remote_state,omitempty" db:"remote_state"`
	LastLocalUpdate    time.Time              `json:"last_local_update" db:"last_local_update"`
	LastRemoteUpdate   *time.Time             `json:"last_remote_update,omitempty" db:"last_remote_update"`
	SyncStatus         string                 `json:"sync_status" db:"sync_status"`
	ConflictData       map[string]interface{} `json:"conflict_data,omitempty" db:"conflict_data"`
	ConflictResolvedAt *time.Time             `json:"conflict_resolved_at,omitempty" db:"conflict_resolved_at"`
	CreatedAt          time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at" db:"updated_at"`
}

// SyncAcknowledgment represents an acknowledgment record
type SyncAcknowledgment struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	TenantID       uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	ConfigID       uuid.UUID              `json:"config_id" db:"config_id"`
	EventID        uuid.UUID              `json:"event_id" db:"event_id"`
	CorrelationID  string                 `json:"correlation_id" db:"correlation_id"`
	AckType        string                 `json:"ack_type" db:"ack_type"`
	AckPayload     map[string]interface{} `json:"ack_payload,omitempty" db:"ack_payload"`
	SentAt         time.Time              `json:"sent_at" db:"sent_at"`
	AcknowledgedAt *time.Time             `json:"acknowledged_at,omitempty" db:"acknowledged_at"`
	TimeoutAt      *time.Time             `json:"timeout_at,omitempty" db:"timeout_at"`
	RetryCount     int                    `json:"retry_count" db:"retry_count"`
	Status         string                 `json:"status" db:"status"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
}

// SyncConflictHistory represents conflict resolution history
type SyncConflictHistory struct {
	ID                 uuid.UUID              `json:"id" db:"id"`
	TenantID           uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	StateRecordID      uuid.UUID              `json:"state_record_id" db:"state_record_id"`
	LocalState         map[string]interface{} `json:"local_state" db:"local_state"`
	RemoteState        map[string]interface{} `json:"remote_state" db:"remote_state"`
	ResolutionStrategy string                 `json:"resolution_strategy" db:"resolution_strategy"`
	ResolvedState      map[string]interface{} `json:"resolved_state,omitempty" db:"resolved_state"`
	ResolvedBy         *uuid.UUID             `json:"resolved_by,omitempty" db:"resolved_by"`
	ResolvedAt         time.Time              `json:"resolved_at" db:"resolved_at"`
}

// Request models

// CreateSyncConfigRequest represents a request to create a sync config
type CreateSyncConfigRequest struct {
	Name               string                 `json:"name" binding:"required"`
	Description        string                 `json:"description"`
	OutboundEndpointID string                 `json:"outbound_endpoint_id"`
	InboundEventType   string                 `json:"inbound_event_type"`
	SyncMode           string                 `json:"sync_mode" binding:"required"`
	TimeoutSeconds     int                    `json:"timeout_seconds"`
	RetryOnTimeout     bool                   `json:"retry_on_timeout"`
	MaxRetries         int                    `json:"max_retries"`
	CorrelationConfig  map[string]interface{} `json:"correlation_config"`
}

// SendSyncRequestRequest represents a request to send a sync request
type SendSyncRequestRequest struct {
	ConfigID string                 `json:"config_id" binding:"required"`
	Payload  map[string]interface{} `json:"payload" binding:"required"`
	Metadata map[string]interface{} `json:"metadata"`
}

// ReceiveSyncResponseRequest represents a received sync response
type ReceiveSyncResponseRequest struct {
	CorrelationID string                 `json:"correlation_id" binding:"required"`
	Payload       map[string]interface{} `json:"payload" binding:"required"`
}

// SendAcknowledgmentRequest represents a request to send an acknowledgment
type SendAcknowledgmentRequest struct {
	ConfigID      string                 `json:"config_id" binding:"required"`
	EventID       string                 `json:"event_id" binding:"required"`
	AckType       string                 `json:"ack_type" binding:"required"`
	AckPayload    map[string]interface{} `json:"ack_payload"`
}

// UpdateStateRequest represents a request to update local state
type UpdateStateRequest struct {
	ConfigID     string                 `json:"config_id" binding:"required"`
	ResourceType string                 `json:"resource_type" binding:"required"`
	ResourceID   string                 `json:"resource_id" binding:"required"`
	State        map[string]interface{} `json:"state" binding:"required"`
}

// ResolveConflictRequest represents a request to resolve a conflict
type ResolveConflictRequest struct {
	StateRecordID      string                 `json:"state_record_id" binding:"required"`
	ResolutionStrategy string                 `json:"resolution_strategy" binding:"required"`
	ResolvedState      map[string]interface{} `json:"resolved_state"`
}

// SyncDashboard represents the sync dashboard data
type SyncDashboard struct {
	ActiveConfigs      int                  `json:"active_configs"`
	PendingTransactions int                 `json:"pending_transactions"`
	CompletedToday     int                  `json:"completed_today"`
	TimeoutsToday      int                  `json:"timeouts_today"`
	ActiveConflicts    int                  `json:"active_conflicts"`
	PendingAcks        int                  `json:"pending_acks"`
	RecentTransactions []*SyncTransaction   `json:"recent_transactions"`
	RecentConflicts    []*SyncStateRecord   `json:"recent_conflicts"`
}

// TransactionStats represents statistics for sync transactions
type TransactionStats struct {
	TotalTransactions   int     `json:"total_transactions"`
	CompletedCount      int     `json:"completed_count"`
	PendingCount        int     `json:"pending_count"`
	TimeoutCount        int     `json:"timeout_count"`
	FailedCount         int     `json:"failed_count"`
	AvgResponseTimeMs   int64   `json:"avg_response_time_ms"`
	SuccessRate         float64 `json:"success_rate"`
}
