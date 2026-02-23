package experiment

import (
	"time"
)

// Experiment status constants
const (
	StatusDraft     = "draft"
	StatusRunning   = "running"
	StatusPaused    = "paused"
	StatusCompleted = "completed"
	StatusCancelled = "cancelled"
)

// Experiment represents an A/B test configuration.
type Experiment struct {
	ID              string          `json:"id" db:"id"`
	TenantID        string          `json:"tenant_id" db:"tenant_id"`
	Name            string          `json:"name" db:"name"`
	Description     string          `json:"description" db:"description"`
	Status          string          `json:"status" db:"status"`
	EventType       string          `json:"event_type" db:"event_type"`
	Variants        []Variant       `json:"variants"`
	SuccessCriteria SuccessCriteria `json:"success_criteria"`
	WinnerVariant   string          `json:"winner_variant,omitempty" db:"winner_variant"`
	StartedAt       *time.Time      `json:"started_at,omitempty" db:"started_at"`
	EndedAt         *time.Time      `json:"ended_at,omitempty" db:"ended_at"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at" db:"updated_at"`
}

// Variant represents a single experiment arm.
type Variant struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	TrafficPercent int                    `json:"traffic_percent"`
	IsControl      bool                   `json:"is_control"`
	Config         map[string]interface{} `json:"config,omitempty"`
}

// SuccessCriteria defines when an experiment should conclude.
type SuccessCriteria struct {
	MinSampleSize    int     `json:"min_sample_size"`
	ConfidenceLevel  float64 `json:"confidence_level"`
	MetricName       string  `json:"metric_name"`
	AutoPromote      bool    `json:"auto_promote"`
	MaxDurationHours int     `json:"max_duration_hours"`
}

// Assignment records which variant a webhook was assigned to.
type Assignment struct {
	ID           string    `json:"id" db:"id"`
	ExperimentID string    `json:"experiment_id" db:"experiment_id"`
	WebhookID    string    `json:"webhook_id" db:"webhook_id"`
	VariantID    string    `json:"variant_id" db:"variant_id"`
	AssignedAt   time.Time `json:"assigned_at" db:"assigned_at"`
}

// VariantMetrics holds aggregated metrics per variant.
type VariantMetrics struct {
	ID            string    `json:"id" db:"id"`
	ExperimentID  string    `json:"experiment_id" db:"experiment_id"`
	VariantID     string    `json:"variant_id" db:"variant_id"`
	TotalRequests int64     `json:"total_requests" db:"total_requests"`
	SuccessCount  int64     `json:"success_count" db:"success_count"`
	FailureCount  int64     `json:"failure_count" db:"failure_count"`
	AvgLatencyMs  float64   `json:"avg_latency_ms" db:"avg_latency_ms"`
	P99LatencyMs  float64   `json:"p99_latency_ms" db:"p99_latency_ms"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// ExperimentResults holds analysis results with statistical significance.
type ExperimentResults struct {
	ExperimentID    string          `json:"experiment_id"`
	Status          string          `json:"status"`
	Variants        []VariantResult `json:"variants"`
	IsSignificant   bool            `json:"is_significant"`
	WinnerVariant   string          `json:"winner_variant,omitempty"`
	ConfidenceLevel float64         `json:"confidence_level"`
}

// VariantResult holds per-variant results with success rate.
type VariantResult struct {
	VariantID    string  `json:"variant_id"`
	VariantName  string  `json:"variant_name"`
	SuccessRate  float64 `json:"success_rate"`
	SampleSize   int64   `json:"sample_size"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
}

// Request DTOs

type CreateExperimentRequest struct {
	Name            string          `json:"name" binding:"required"`
	Description     string          `json:"description"`
	EventType       string          `json:"event_type" binding:"required"`
	Variants        []Variant       `json:"variants" binding:"required"`
	SuccessCriteria SuccessCriteria `json:"success_criteria"`
}

type RecordOutcomeRequest struct {
	ExperimentID string  `json:"experiment_id" binding:"required"`
	WebhookID    string  `json:"webhook_id" binding:"required"`
	Success      bool    `json:"success"`
	LatencyMs    float64 `json:"latency_ms"`
}
