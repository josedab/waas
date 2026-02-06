package replay

import (
	"encoding/json"
	"time"
)

// ReplayRequest represents a request to replay a delivery
type ReplayRequest struct {
	DeliveryID    string `json:"delivery_id" binding:"required"`
	EndpointID    string `json:"endpoint_id,omitempty"`    // Override original endpoint
	ModifyPayload bool   `json:"modify_payload,omitempty"` // Allow payload modification
	Payload       []byte `json:"payload,omitempty"`        // Modified payload if allowed
}

// ReplayResult represents the result of a replay operation
type ReplayResult struct {
	OriginalDeliveryID string    `json:"original_delivery_id"`
	NewDeliveryID      string    `json:"new_delivery_id"`
	EndpointID         string    `json:"endpoint_id"`
	Status             string    `json:"status"`
	ReplayedAt         time.Time `json:"replayed_at"`
}

// BulkReplayRequest represents a request to replay multiple deliveries
type BulkReplayRequest struct {
	DeliveryIDs []string  `json:"delivery_ids,omitempty"` // Specific delivery IDs
	EndpointID  string    `json:"endpoint_id,omitempty"`  // Filter by endpoint
	Status      string    `json:"status,omitempty"`       // Filter by status (failed, delivered)
	StartTime   time.Time `json:"start_time,omitempty"`   // Filter by time range
	EndTime     time.Time `json:"end_time,omitempty"`
	Limit       int       `json:"limit,omitempty"`      // Max deliveries to replay
	DryRun      bool      `json:"dry_run,omitempty"`    // Preview without replaying
	RateLimit   int       `json:"rate_limit,omitempty"` // Max replays per second
}

// BulkReplayResult represents the result of a bulk replay operation
type BulkReplayResult struct {
	TotalFound    int            `json:"total_found"`
	TotalReplayed int            `json:"total_replayed"`
	TotalFailed   int            `json:"total_failed"`
	TotalSkipped  int            `json:"total_skipped"`
	Results       []ReplayResult `json:"results,omitempty"`
	Errors        []ReplayError  `json:"errors,omitempty"`
	DryRun        bool           `json:"dry_run"`
}

// ReplayError represents an error during replay
type ReplayError struct {
	DeliveryID string `json:"delivery_id"`
	Error      string `json:"error"`
}

// Snapshot represents a point-in-time snapshot of delivery state
type Snapshot struct {
	ID          string          `json:"id" db:"id"`
	TenantID    string          `json:"tenant_id" db:"tenant_id"`
	Name        string          `json:"name" db:"name"`
	Description string          `json:"description,omitempty" db:"description"`
	Filters     json.RawMessage `json:"filters" db:"filters"`
	DeliveryIDs []string        `json:"delivery_ids" db:"-"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	ExpiresAt   time.Time       `json:"expires_at,omitempty" db:"expires_at"`
}

// SnapshotFilters represents the filters used to create a snapshot
type SnapshotFilters struct {
	EndpointID string    `json:"endpoint_id,omitempty"`
	Status     string    `json:"status,omitempty"`
	StartTime  time.Time `json:"start_time,omitempty"`
	EndTime    time.Time `json:"end_time,omitempty"`
}

// CreateSnapshotRequest represents a request to create a snapshot
type CreateSnapshotRequest struct {
	Name        string          `json:"name" binding:"required,min=1,max=255"`
	Description string          `json:"description,omitempty"`
	Filters     SnapshotFilters `json:"filters" binding:"required"`
	TTLDays     int             `json:"ttl_days,omitempty"` // 0 = no expiration
}

// ReplayFromSnapshotRequest represents a request to replay from a snapshot
type ReplayFromSnapshotRequest struct {
	SnapshotID string `json:"snapshot_id" binding:"required"`
	Limit      int    `json:"limit,omitempty"`
	DryRun     bool   `json:"dry_run,omitempty"`
}

// DeliveryArchive represents an archived delivery with full payload
type DeliveryArchive struct {
	ID             string            `json:"id" db:"id"`
	TenantID       string            `json:"tenant_id" db:"tenant_id"`
	EndpointID     string            `json:"endpoint_id" db:"endpoint_id"`
	EndpointURL    string            `json:"endpoint_url" db:"endpoint_url"`
	Payload        json.RawMessage   `json:"payload" db:"payload"`
	Headers        map[string]string `json:"headers" db:"-"`
	HeadersJSON    json.RawMessage   `json:"-" db:"headers"`
	Status         string            `json:"status" db:"status"`
	AttemptCount   int               `json:"attempt_count" db:"attempt_count"`
	LastHTTPStatus int               `json:"last_http_status,omitempty" db:"last_http_status"`
	LastError      string            `json:"last_error,omitempty" db:"last_error"`
	CreatedAt      time.Time         `json:"created_at" db:"created_at"`
	CompletedAt    *time.Time        `json:"completed_at,omitempty" db:"completed_at"`
}

// TimeRange represents a time range for queries
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// WhatIfRequest represents a what-if scenario simulation request
type WhatIfRequest struct {
	DeliveryID      string            `json:"delivery_id" binding:"required"`
	ModifiedPayload json.RawMessage   `json:"modified_payload,omitempty"`
	TargetEndpoints []string          `json:"target_endpoints,omitempty"`
	ModifiedHeaders map[string]string `json:"modified_headers,omitempty"`
}

// WhatIfResult represents the result of a what-if simulation
type WhatIfResult struct {
	OriginalDeliveryID string            `json:"original_delivery_id"`
	Original           *WhatIfDelivery   `json:"original"`
	Simulated          *WhatIfDelivery   `json:"simulated"`
	PayloadDiff        []PayloadDiffItem `json:"payload_diff,omitempty"`
	EndpointChanges    []string          `json:"endpoint_changes,omitempty"`
	Analysis           string            `json:"analysis"`
}

// WhatIfDelivery represents delivery details in a what-if scenario
type WhatIfDelivery struct {
	EndpointID  string            `json:"endpoint_id"`
	EndpointURL string            `json:"endpoint_url"`
	Payload     json.RawMessage   `json:"payload"`
	Headers     map[string]string `json:"headers"`
	PayloadSize int               `json:"payload_size"`
}

// PayloadDiffItem represents a diff between original and modified payloads
type PayloadDiffItem struct {
	Path     string      `json:"path"`
	Type     string      `json:"type"` // added, removed, changed
	OldValue interface{} `json:"old_value,omitempty"`
	NewValue interface{} `json:"new_value,omitempty"`
}
