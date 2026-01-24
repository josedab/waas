package timetravel

import (
	"encoding/json"
	"time"
)

// EventRecord represents a stored webhook event for time-travel replay
type EventRecord struct {
	ID         string          `json:"id" db:"id"`
	TenantID   string          `json:"tenant_id" db:"tenant_id"`
	WebhookID  string          `json:"webhook_id" db:"webhook_id"`
	EndpointID string          `json:"endpoint_id" db:"endpoint_id"`
	EventType  string          `json:"event_type" db:"event_type"`
	Payload    json.RawMessage `json:"payload" db:"payload"`
	Headers    json.RawMessage `json:"headers" db:"headers"`
	Timestamp  time.Time       `json:"timestamp" db:"timestamp"`
	Checksum   string          `json:"checksum" db:"checksum"`
}

// TimeWindow represents a start/end time range
type TimeWindow struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ReplayJob represents an asynchronous replay operation
type ReplayJob struct {
	ID             string          `json:"id" db:"id"`
	TenantID       string          `json:"tenant_id" db:"tenant_id"`
	Status         ReplayJobStatus `json:"status" db:"status"`
	TimeWindow     TimeWindow      `json:"time_window" db:"-"`
	TimeWindowJSON json.RawMessage `json:"-" db:"time_window"`
	Filters        ReplayFilter    `json:"filters" db:"-"`
	FiltersJSON    json.RawMessage `json:"-" db:"filters"`
	TargetEndpoint string          `json:"target_endpoint" db:"target_endpoint"`
	DryRun         bool            `json:"dry_run" db:"dry_run"`
	Progress       int             `json:"progress" db:"progress"`
	Results        ReplayResult    `json:"results" db:"-"`
	ResultsJSON    json.RawMessage `json:"-" db:"results"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
}

// ReplayJobStatus represents the status of a replay job
type ReplayJobStatus string

const (
	ReplayJobStatusPending   ReplayJobStatus = "pending"
	ReplayJobStatusRunning   ReplayJobStatus = "running"
	ReplayJobStatusCompleted ReplayJobStatus = "completed"
	ReplayJobStatusFailed    ReplayJobStatus = "failed"
	ReplayJobStatusCancelled ReplayJobStatus = "cancelled"
)

// ReplayFilter defines criteria for selecting events to replay
type ReplayFilter struct {
	EndpointIDs []string   `json:"endpoint_ids,omitempty"`
	EventTypes  []string   `json:"event_types,omitempty"`
	StatusCodes []int      `json:"status_codes,omitempty"`
	TimeWindow  TimeWindow `json:"time_window,omitempty"`
}

// ReplayResult holds the outcome of a replay job
type ReplayResult struct {
	TotalEvents int   `json:"total_events"`
	Replayed    int   `json:"replayed"`
	Succeeded   int   `json:"succeeded"`
	Failed      int   `json:"failed"`
	Skipped     int   `json:"skipped"`
	DurationMs  int64 `json:"duration_ms"`
}

// PointInTimeSnapshot represents a named snapshot of event state at a given time
type PointInTimeSnapshot struct {
	ID          string    `json:"id" db:"id"`
	TenantID    string    `json:"tenant_id" db:"tenant_id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description,omitempty" db:"description"`
	TimePoint   time.Time `json:"time_point" db:"time_point"`
	EventCount  int       `json:"event_count" db:"event_count"`
	SizeBytes   int64     `json:"size_bytes" db:"size_bytes"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// BlastRadiusAnalysis represents the impact analysis of an endpoint failure
type BlastRadiusAnalysis struct {
	EndpointID        string   `json:"endpoint_id"`
	AffectedEvents    int      `json:"affected_events"`
	AffectedEndpoints []string `json:"affected_endpoints"`
	ImpactScore       float64  `json:"impact_score"`
	Recommendation    string   `json:"recommendation"`
}

// WhatIfScenario represents a what-if analysis comparing original vs modified payloads
type WhatIfScenario struct {
	ID              string          `json:"id" db:"id"`
	TenantID        string          `json:"tenant_id" db:"tenant_id"`
	Name            string          `json:"name" db:"name"`
	Description     string          `json:"description,omitempty" db:"description"`
	ModifiedPayload json.RawMessage `json:"modified_payload" db:"modified_payload"`
	OriginalPayload json.RawMessage `json:"original_payload" db:"original_payload"`
	DiffSummary     string          `json:"diff_summary" db:"diff_summary"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
}

// CreateReplayJobRequest is the request DTO for creating a replay job
type CreateReplayJobRequest struct {
	TimeWindow     TimeWindow   `json:"time_window" binding:"required"`
	Filters        ReplayFilter `json:"filters,omitempty"`
	TargetEndpoint string       `json:"target_endpoint,omitempty"`
	DryRun         bool         `json:"dry_run,omitempty"`
}

// CreateSnapshotRequest is the request DTO for creating a point-in-time snapshot
type CreateSnapshotRequest struct {
	Name        string    `json:"name" binding:"required,min=1,max=255"`
	Description string    `json:"description,omitempty"`
	TimePoint   time.Time `json:"time_point" binding:"required"`
}

// WhatIfRequest is the request DTO for running a what-if scenario
type WhatIfRequest struct {
	Name            string          `json:"name" binding:"required,min=1,max=255"`
	Description     string          `json:"description,omitempty"`
	EndpointID      string          `json:"endpoint_id" binding:"required"`
	OriginalPayload json.RawMessage `json:"original_payload" binding:"required"`
	ModifiedPayload json.RawMessage `json:"modified_payload" binding:"required"`
}
