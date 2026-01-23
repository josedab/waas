package debugger

import "time"

// DeliveryTrace captures the full lifecycle of a webhook delivery at each stage
type DeliveryTrace struct {
	ID           string       `json:"id" db:"id"`
	TenantID     string       `json:"tenant_id" db:"tenant_id"`
	DeliveryID   string       `json:"delivery_id" db:"delivery_id"`
	EndpointID   string       `json:"endpoint_id" db:"endpoint_id"`
	Stages       []TraceStage `json:"stages"`
	StagesJSON   string       `json:"-" db:"stages"`
	TotalDurMs   int          `json:"total_duration_ms" db:"total_duration_ms"`
	FinalStatus  string       `json:"final_status" db:"final_status"`
	CreatedAt    time.Time    `json:"created_at" db:"created_at"`
}

// TraceStage represents a single stage in the delivery pipeline
type TraceStage struct {
	Name       string            `json:"name"`
	Status     string            `json:"status"`
	DurationMs int               `json:"duration_ms"`
	Input      string            `json:"input,omitempty"`
	Output     string            `json:"output,omitempty"`
	Error      string            `json:"error,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Timestamp  time.Time         `json:"timestamp"`
}

// StageNames for the delivery pipeline
const (
	StageReceived     = "received"
	StageValidation   = "validation"
	StageTransform    = "pre_transform"
	StagePostTransform = "post_transform"
	StageDelivery     = "delivery"
	StageResponse     = "response"
	StageRetry        = "retry"
)

// PayloadDiff represents the diff between two payloads
type PayloadDiff struct {
	DeliveryID string       `json:"delivery_id"`
	Diffs      []FieldDiff  `json:"diffs"`
	Identical  bool         `json:"identical"`
}

// FieldDiff represents a single field difference
type FieldDiff struct {
	Path     string `json:"path"`
	Type     string `json:"type"` // added, removed, changed
	OldValue string `json:"old_value,omitempty"`
	NewValue string `json:"new_value,omitempty"`
}

// ReplayWithModRequest is the request DTO for replay with modifications
type ReplayWithModRequest struct {
	DeliveryID      string            `json:"delivery_id" binding:"required"`
	PayloadOverride string            `json:"payload_override,omitempty"`
	HeaderOverride  map[string]string `json:"header_override,omitempty"`
	EndpointOverride string           `json:"endpoint_override,omitempty"`
}

// BulkReplayRequest is the request DTO for bulk replaying deliveries
type BulkReplayRequest struct {
	DeliveryIDs    []string `json:"delivery_ids,omitempty"`
	EndpointID     string   `json:"endpoint_id,omitempty"`
	StatusFilter   string   `json:"status_filter,omitempty"`
	Since          string   `json:"since,omitempty"`
	Until          string   `json:"until,omitempty"`
	DryRun         bool     `json:"dry_run"`
	RatePerSecond  int      `json:"rate_per_second,omitempty"`
}

// BulkReplayResult is the response for a bulk replay operation
type BulkReplayResult struct {
	TotalFound    int      `json:"total_found"`
	TotalReplayed int      `json:"total_replayed"`
	TotalFailed   int      `json:"total_failed"`
	DryRun        bool     `json:"dry_run"`
	DeliveryIDs   []string `json:"new_delivery_ids,omitempty"`
}

// DebugSession represents an active debugging session
type DebugSession struct {
	ID          string    `json:"id" db:"id"`
	TenantID    string    `json:"tenant_id" db:"tenant_id"`
	DeliveryID  string    `json:"delivery_id" db:"delivery_id"`
	CurrentStep int       `json:"current_step" db:"current_step"`
	Breakpoints []string  `json:"breakpoints"`
	BreakJSON   string    `json:"-" db:"breakpoints"`
	Status      string    `json:"status" db:"status"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	ExpiresAt   time.Time `json:"expires_at" db:"expires_at"`
}

// DebugSessionStatus constants
const (
	DebugStatusActive  = "active"
	DebugStatusPaused  = "paused"
	DebugStatusExpired = "expired"
)

// CreateDebugSessionRequest is the request DTO for starting a debug session
type CreateDebugSessionRequest struct {
	DeliveryID  string   `json:"delivery_id" binding:"required"`
	Breakpoints []string `json:"breakpoints,omitempty"`
}
