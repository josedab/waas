package progressive

import (
	"time"
)

// RolloutStrategy constants define how traffic is shifted.
const (
	StrategyCanary     = "canary"
	StrategyBlueGreen  = "blue_green"
	StrategyPercentage = "percentage"
	StrategyRingBased  = "ring_based"
)

// RolloutStatus constants define the lifecycle of a rollout.
const (
	StatusPending    = "pending"
	StatusActive     = "active"
	StatusPaused     = "paused"
	StatusCompleted  = "completed"
	StatusRolledBack = "rolled_back"
)

// Rollout represents a progressive delivery configuration.
type Rollout struct {
	ID              string          `json:"id" db:"id"`
	TenantID        string          `json:"tenant_id" db:"tenant_id"`
	Name            string          `json:"name" db:"name"`
	EndpointID      string          `json:"endpoint_id" db:"endpoint_id"`
	Strategy        string          `json:"strategy" db:"strategy"`
	Status          string          `json:"status" db:"status"`
	TargetConfig    RolloutConfig   `json:"target_config"`
	BaselineConfig  RolloutConfig   `json:"baseline_config"`
	TrafficSplit    TrafficSplit    `json:"traffic_split"`
	SuccessCriteria SuccessCriteria `json:"success_criteria"`
	Metrics         RolloutMetrics  `json:"metrics"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at" db:"updated_at"`
}

// TrafficSplit defines how traffic is distributed between baseline and target.
type TrafficSplit struct {
	BaselinePercent int `json:"baseline_percent"`
	TargetPercent   int `json:"target_percent"`
}

// RolloutConfig holds the configuration for a rollout variant.
type RolloutConfig struct {
	URL         string            `json:"url"`
	Headers     map[string]string `json:"headers,omitempty"`
	RetryPolicy string            `json:"retry_policy,omitempty"`
	TransformID string            `json:"transform_id,omitempty"`
}

// SuccessCriteria defines when a rollout should be promoted or rolled back.
type SuccessCriteria struct {
	MinSuccessRate   float64       `json:"min_success_rate"`
	MaxLatencyMs     float64       `json:"max_latency_ms"`
	MinSampleSize    int           `json:"min_sample_size"`
	EvaluationWindow time.Duration `json:"evaluation_window"`
}

// RolloutMetrics holds aggregated metrics for both variants.
type RolloutMetrics struct {
	BaselineMetrics VariantMetrics `json:"baseline_metrics"`
	TargetMetrics   VariantMetrics `json:"target_metrics"`
}

// VariantMetrics holds per-variant delivery metrics.
type VariantMetrics struct {
	Requests     int64   `json:"requests"`
	Successes    int64   `json:"successes"`
	Failures     int64   `json:"failures"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	P99LatencyMs float64 `json:"p99_latency_ms"`
	SuccessRate  float64 `json:"success_rate"`
}

// Request DTOs

// CreateRolloutRequest is the payload for creating a new rollout.
type CreateRolloutRequest struct {
	Name            string          `json:"name" binding:"required"`
	EndpointID      string          `json:"endpoint_id" binding:"required"`
	Strategy        string          `json:"strategy" binding:"required"`
	TargetConfig    RolloutConfig   `json:"target_config" binding:"required"`
	BaselineConfig  RolloutConfig   `json:"baseline_config" binding:"required"`
	TrafficSplit    TrafficSplit    `json:"traffic_split"`
	SuccessCriteria SuccessCriteria `json:"success_criteria"`
}

// UpdateTrafficRequest is the payload for adjusting traffic split.
type UpdateTrafficRequest struct {
	BaselinePercent int `json:"baseline_percent" binding:"required"`
	TargetPercent   int `json:"target_percent" binding:"required"`
}
