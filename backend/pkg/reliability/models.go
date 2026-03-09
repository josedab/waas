package reliability

import (
	"time"
)

// ScoreStatus represents the health status derived from the reliability score.
type ScoreStatus string

const (
	ScoreStatusHealthy  ScoreStatus = "healthy"
	ScoreStatusDegraded ScoreStatus = "degraded"
	ScoreStatusCritical ScoreStatus = "critical"
	ScoreStatusUnknown  ScoreStatus = "unknown"
)

// ReliabilityScore represents a computed reliability score for an endpoint.
type ReliabilityScore struct {
	ID                  string      `json:"id" db:"id"`
	TenantID            string      `json:"tenant_id" db:"tenant_id"`
	EndpointID          string      `json:"endpoint_id" db:"endpoint_id"`
	Score               float64     `json:"score" db:"score"`
	Status              ScoreStatus `json:"status" db:"status"`
	SuccessRate         float64     `json:"success_rate" db:"success_rate"`
	LatencyP50Ms        int         `json:"latency_p50_ms" db:"latency_p50_ms"`
	LatencyP95Ms        int         `json:"latency_p95_ms" db:"latency_p95_ms"`
	LatencyP99Ms        int         `json:"latency_p99_ms" db:"latency_p99_ms"`
	ConsecutiveFailures int         `json:"consecutive_failures" db:"consecutive_failures"`
	TotalAttempts       int         `json:"total_attempts" db:"total_attempts"`
	SuccessfulAttempts  int         `json:"successful_attempts" db:"successful_attempts"`
	FailedAttempts      int         `json:"failed_attempts" db:"failed_attempts"`
	WindowStart         time.Time   `json:"window_start" db:"window_start"`
	WindowEnd           time.Time   `json:"window_end" db:"window_end"`
	CreatedAt           time.Time   `json:"created_at" db:"created_at"`
}

// ScoreSnapshot represents an hourly reliability score snapshot for trending.
type ScoreSnapshot struct {
	ID           string    `json:"id" db:"id"`
	TenantID     string    `json:"tenant_id" db:"tenant_id"`
	EndpointID   string    `json:"endpoint_id" db:"endpoint_id"`
	Score        float64   `json:"score" db:"score"`
	SuccessRate  float64   `json:"success_rate" db:"success_rate"`
	LatencyP50Ms int       `json:"latency_p50_ms" db:"latency_p50_ms"`
	LatencyP95Ms int       `json:"latency_p95_ms" db:"latency_p95_ms"`
	LatencyP99Ms int       `json:"latency_p99_ms" db:"latency_p99_ms"`
	SnapshotAt   time.Time `json:"snapshot_at" db:"snapshot_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// SLATarget defines a reliability SLA for an endpoint.
type SLATarget struct {
	ID              string    `json:"id" db:"id"`
	TenantID        string    `json:"tenant_id" db:"tenant_id"`
	EndpointID      string    `json:"endpoint_id" db:"endpoint_id"`
	TargetScore     float64   `json:"target_score" db:"target_score"`
	TargetUptime    float64   `json:"target_uptime" db:"target_uptime"`
	MaxLatencyP95Ms int       `json:"max_latency_p95_ms" db:"max_latency_p95_ms"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// SLAStatus reports compliance with SLA targets.
type SLAStatus struct {
	Target         *SLATarget `json:"target"`
	CurrentScore   float64    `json:"current_score"`
	CurrentUptime  float64    `json:"current_uptime"`
	CurrentP95Ms   int        `json:"current_p95_ms"`
	IsCompliant    bool       `json:"is_compliant"`
	ViolationCount int        `json:"violation_count"`
}

// AlertThreshold defines when to trigger reliability alerts.
type AlertThreshold struct {
	ID           string    `json:"id" db:"id"`
	TenantID     string    `json:"tenant_id" db:"tenant_id"`
	EndpointID   string    `json:"endpoint_id" db:"endpoint_id"`
	MinScore     float64   `json:"min_score" db:"min_score"`
	MaxLatencyMs int       `json:"max_latency_ms" db:"max_latency_ms"`
	MaxFailures  int       `json:"max_failures" db:"max_failures"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// ReliabilityTrend represents the time-series trend data for a period.
type ReliabilityTrend struct {
	EndpointID string          `json:"endpoint_id"`
	Period     string          `json:"period"`
	DataPoints []ScoreSnapshot `json:"data_points"`
}

// ReliabilityReport is the full reliability response for an endpoint.
type ReliabilityReport struct {
	CurrentScore *ReliabilityScore `json:"current_score"`
	Trend        *ReliabilityTrend `json:"trend,omitempty"`
	SLA          *SLAStatus        `json:"sla,omitempty"`
}

// CreateSLARequest represents a request to set an SLA target.
type CreateSLARequest struct {
	TargetScore     float64 `json:"target_score" binding:"required,min=0,max=100"`
	TargetUptime    float64 `json:"target_uptime" binding:"required,min=0,max=100"`
	MaxLatencyP95Ms int     `json:"max_latency_p95_ms" binding:"required,min=1"`
}

// CreateAlertThresholdRequest represents a request to create an alert threshold.
type CreateAlertThresholdRequest struct {
	MinScore     float64 `json:"min_score" binding:"required,min=0,max=100"`
	MaxLatencyMs int     `json:"max_latency_ms" binding:"required,min=1"`
	MaxFailures  int     `json:"max_failures" binding:"required,min=1"`
}
