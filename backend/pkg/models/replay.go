package models

import (
	"time"

	"github.com/google/uuid"
)

// Replay job statuses
const (
	ReplayStatusPending    = "pending"
	ReplayStatusRunning    = "running"
	ReplayStatusPaused     = "paused"
	ReplayStatusCompleted  = "completed"
	ReplayStatusFailed     = "failed"
	ReplayStatusCancelled  = "cancelled"
)

// Replay event statuses
const (
	ReplayEventPending   = "pending"
	ReplayEventSuccess   = "success"
	ReplayEventFailed    = "failed"
	ReplayEventSkipped   = "skipped"
)

// EventArchive represents a stored historical event
type EventArchive struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	TenantID    uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	EndpointID  *uuid.UUID             `json:"endpoint_id,omitempty" db:"endpoint_id"`
	EventType   string                 `json:"event_type" db:"event_type"`
	Payload     map[string]interface{} `json:"payload" db:"payload"`
	PayloadHash string                 `json:"payload_hash" db:"payload_hash"`
	Headers     map[string]string      `json:"headers" db:"headers"`
	SourceIP    string                 `json:"source_ip" db:"source_ip"`
	ReceivedAt  time.Time              `json:"received_at" db:"received_at"`
	Metadata    map[string]interface{} `json:"metadata" db:"metadata"`
}

// ReplayJob represents a replay operation
type ReplayJob struct {
	ID               uuid.UUID              `json:"id" db:"id"`
	TenantID         uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	Name             string                 `json:"name" db:"name"`
	Description      string                 `json:"description" db:"description"`
	Status           string                 `json:"status" db:"status"`
	FilterCriteria   ReplayFilterCriteria   `json:"filter_criteria" db:"filter_criteria"`
	TimeRangeStart   time.Time              `json:"time_range_start" db:"time_range_start"`
	TimeRangeEnd     time.Time              `json:"time_range_end" db:"time_range_end"`
	TargetEndpointID *uuid.UUID             `json:"target_endpoint_id,omitempty" db:"target_endpoint_id"`
	TransformationID *uuid.UUID             `json:"transformation_id,omitempty" db:"transformation_id"`
	Options          ReplayOptions          `json:"options" db:"options"`
	TotalEvents      int                    `json:"total_events" db:"total_events"`
	ProcessedEvents  int                    `json:"processed_events" db:"processed_events"`
	SuccessfulEvents int                    `json:"successful_events" db:"successful_events"`
	FailedEvents     int                    `json:"failed_events" db:"failed_events"`
	StartedAt        *time.Time             `json:"started_at,omitempty" db:"started_at"`
	CompletedAt      *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
	ErrorMessage     string                 `json:"error_message,omitempty" db:"error_message"`
	CreatedBy        *uuid.UUID             `json:"created_by,omitempty" db:"created_by"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
}

// ReplayFilterCriteria defines filters for selecting events to replay
type ReplayFilterCriteria struct {
	EventTypes  []string               `json:"event_types,omitempty"`
	EndpointIDs []uuid.UUID            `json:"endpoint_ids,omitempty"`
	PayloadFilters map[string]interface{} `json:"payload_filters,omitempty"`
	ExcludeHashes []string              `json:"exclude_hashes,omitempty"`
}

// ReplayOptions defines replay behavior options
type ReplayOptions struct {
	RateLimit           int    `json:"rate_limit,omitempty"`             // Events per second
	PreserveTimestamps  bool   `json:"preserve_timestamps,omitempty"`
	DryRun              bool   `json:"dry_run,omitempty"`
	StopOnError         bool   `json:"stop_on_error,omitempty"`
	RetryFailedEvents   bool   `json:"retry_failed_events,omitempty"`
	BatchSize           int    `json:"batch_size,omitempty"`
	TransformationCode  string `json:"transformation_code,omitempty"`
}

// ReplayJobEvent represents a single event being replayed
type ReplayJobEvent struct {
	ID                 uuid.UUID              `json:"id" db:"id"`
	ReplayJobID        uuid.UUID              `json:"replay_job_id" db:"replay_job_id"`
	EventArchiveID     uuid.UUID              `json:"event_archive_id" db:"event_archive_id"`
	Status             string                 `json:"status" db:"status"`
	OriginalPayload    map[string]interface{} `json:"original_payload" db:"original_payload"`
	TransformedPayload map[string]interface{} `json:"transformed_payload,omitempty" db:"transformed_payload"`
	DeliveryAttemptID  *uuid.UUID             `json:"delivery_attempt_id,omitempty" db:"delivery_attempt_id"`
	ErrorMessage       string                 `json:"error_message,omitempty" db:"error_message"`
	ProcessedAt        *time.Time             `json:"processed_at,omitempty" db:"processed_at"`
	CreatedAt          time.Time              `json:"created_at" db:"created_at"`
}

// ReplaySnapshot represents a point-in-time snapshot
type ReplaySnapshot struct {
	ID              uuid.UUID              `json:"id" db:"id"`
	TenantID        uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	Name            string                 `json:"name" db:"name"`
	Description     string                 `json:"description" db:"description"`
	SnapshotTime    time.Time              `json:"snapshot_time" db:"snapshot_time"`
	FilterCriteria  ReplayFilterCriteria   `json:"filter_criteria" db:"filter_criteria"`
	EventCount      int                    `json:"event_count" db:"event_count"`
	SizeBytes       int64                  `json:"size_bytes" db:"size_bytes"`
	StorageLocation string                 `json:"storage_location" db:"storage_location"`
	ExpiresAt       *time.Time             `json:"expires_at,omitempty" db:"expires_at"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
}

// ReplayComparison represents a comparison between two replay runs
type ReplayComparison struct {
	ID              uuid.UUID              `json:"id" db:"id"`
	TenantID        uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	Name            string                 `json:"name" db:"name"`
	Description     string                 `json:"description" db:"description"`
	OriginalJobID   *uuid.UUID             `json:"original_job_id,omitempty" db:"original_job_id"`
	ComparisonJobID *uuid.UUID             `json:"comparison_job_id,omitempty" db:"comparison_job_id"`
	Status          string                 `json:"status" db:"status"`
	TotalEvents     int                    `json:"total_events" db:"total_events"`
	MatchingEvents  int                    `json:"matching_events" db:"matching_events"`
	DifferingEvents int                    `json:"differing_events" db:"differing_events"`
	DiffReport      map[string]interface{} `json:"diff_report" db:"diff_report"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
}

// Request types

type CreateReplayJobRequest struct {
	Name             string               `json:"name" binding:"required"`
	Description      string               `json:"description"`
	TimeRangeStart   time.Time            `json:"time_range_start" binding:"required"`
	TimeRangeEnd     time.Time            `json:"time_range_end" binding:"required"`
	FilterCriteria   ReplayFilterCriteria `json:"filter_criteria"`
	TargetEndpointID string               `json:"target_endpoint_id"`
	Options          ReplayOptions        `json:"options"`
}

type ReplayJobProgress struct {
	JobID            uuid.UUID `json:"job_id"`
	Status           string    `json:"status"`
	TotalEvents      int       `json:"total_events"`
	ProcessedEvents  int       `json:"processed_events"`
	SuccessfulEvents int       `json:"successful_events"`
	FailedEvents     int       `json:"failed_events"`
	ProgressPercent  float64   `json:"progress_percent"`
	EstimatedTimeRemaining string `json:"estimated_time_remaining,omitempty"`
}

type EventSearchRequest struct {
	TimeRangeStart time.Time            `json:"time_range_start"`
	TimeRangeEnd   time.Time            `json:"time_range_end"`
	EventTypes     []string             `json:"event_types"`
	EndpointIDs    []string             `json:"endpoint_ids"`
	PayloadQuery   string               `json:"payload_query"`
	Limit          int                  `json:"limit"`
	Offset         int                  `json:"offset"`
}

type CreateSnapshotRequest struct {
	Name           string               `json:"name" binding:"required"`
	Description    string               `json:"description"`
	SnapshotTime   time.Time            `json:"snapshot_time" binding:"required"`
	FilterCriteria ReplayFilterCriteria `json:"filter_criteria"`
	ExpiresInDays  int                  `json:"expires_in_days"`
}
