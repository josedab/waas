package canary

import "time"

// CanaryDeployment represents a canary release for webhook schema/payload changes
type CanaryDeployment struct {
	ID              string     `json:"id" db:"id"`
	TenantID        string     `json:"tenant_id" db:"tenant_id"`
	Name            string     `json:"name" db:"name"`
	Description     string     `json:"description,omitempty" db:"description"`
	EndpointID      string     `json:"endpoint_id" db:"endpoint_id"`
	EventType       string     `json:"event_type" db:"event_type"`
	TrafficPct      int        `json:"traffic_pct" db:"traffic_pct"`
	Status          string     `json:"status" db:"status"`
	PromotionRule   string     `json:"promotion_rule" db:"promotion_rule"`
	RollbackOnError bool       `json:"rollback_on_error" db:"rollback_on_error"`
	ErrorThreshold  float64    `json:"error_threshold_pct" db:"error_threshold_pct"`
	SoakTimeMins    int        `json:"soak_time_minutes" db:"soak_time_minutes"`
	PromotedAt      *time.Time `json:"promoted_at,omitempty" db:"promoted_at"`
	RolledBackAt    *time.Time `json:"rolled_back_at,omitempty" db:"rolled_back_at"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

const (
	CanaryStatusPending    = "pending"
	CanaryStatusActive     = "active"
	CanaryStatusPromoted   = "promoted"
	CanaryStatusRolledBack = "rolled_back"
	CanaryStatusPaused     = "paused"
)

const (
	PromotionRuleManual    = "manual"
	PromotionRuleAutomatic = "automatic"
)

// CanaryMetrics tracks performance metrics for a canary vs baseline
type CanaryMetrics struct {
	ID               string    `json:"id" db:"id"`
	TenantID         string    `json:"tenant_id" db:"tenant_id"`
	DeploymentID     string    `json:"deployment_id" db:"deployment_id"`
	WindowStart      time.Time `json:"window_start" db:"window_start"`
	WindowEnd        time.Time `json:"window_end" db:"window_end"`
	CanaryRequests   int       `json:"canary_requests" db:"canary_requests"`
	CanaryErrors     int       `json:"canary_errors" db:"canary_errors"`
	CanaryP50Ms      int64     `json:"canary_p50_ms" db:"canary_p50_ms"`
	CanaryP99Ms      int64     `json:"canary_p99_ms" db:"canary_p99_ms"`
	BaselineRequests int       `json:"baseline_requests" db:"baseline_requests"`
	BaselineErrors   int       `json:"baseline_errors" db:"baseline_errors"`
	BaselineP50Ms    int64     `json:"baseline_p50_ms" db:"baseline_p50_ms"`
	BaselineP99Ms    int64     `json:"baseline_p99_ms" db:"baseline_p99_ms"`
}

// CanaryComparison is the analysis result comparing canary to baseline
type CanaryComparison struct {
	DeploymentID       string  `json:"deployment_id"`
	CanaryErrorRate    float64 `json:"canary_error_rate_pct"`
	BaselineErrorRate  float64 `json:"baseline_error_rate_pct"`
	ErrorRateDelta     float64 `json:"error_rate_delta_pct"`
	CanaryAvgLatency   int64   `json:"canary_avg_latency_ms"`
	BaselineAvgLatency int64   `json:"baseline_avg_latency_ms"`
	LatencyDelta       int64   `json:"latency_delta_ms"`
	IsHealthy          bool    `json:"is_healthy"`
	Recommendation     string  `json:"recommendation"`
}

// CreateCanaryRequest is the request DTO for creating a canary deployment
type CreateCanaryRequest struct {
	Name            string  `json:"name" binding:"required,min=1,max=255"`
	Description     string  `json:"description,omitempty"`
	EndpointID      string  `json:"endpoint_id" binding:"required"`
	EventType       string  `json:"event_type" binding:"required"`
	TrafficPct      int     `json:"traffic_pct" binding:"required,min=1,max=100"`
	PromotionRule   string  `json:"promotion_rule" binding:"required,oneof=manual automatic"`
	RollbackOnError bool    `json:"rollback_on_error"`
	ErrorThreshold  float64 `json:"error_threshold_pct" binding:"min=0,max=100"`
	SoakTimeMins    int     `json:"soak_time_minutes" binding:"min=1"`
}

// UpdateTrafficRequest changes the canary traffic percentage
type UpdateTrafficRequest struct {
	TrafficPct int `json:"traffic_pct" binding:"required,min=0,max=100"`
}
