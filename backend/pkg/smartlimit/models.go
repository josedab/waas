package smartlimit

import (
	"time"
)

// EndpointBehavior captures endpoint response patterns
type EndpointBehavior struct {
	ID              string           `json:"id" db:"id"`
	TenantID        string           `json:"tenant_id" db:"tenant_id"`
	EndpointID      string           `json:"endpoint_id" db:"endpoint_id"`
	URL             string           `json:"url" db:"url"`
	WindowStart     time.Time        `json:"window_start" db:"window_start"`
	WindowEnd       time.Time        `json:"window_end" db:"window_end"`
	TotalRequests   int64            `json:"total_requests" db:"total_requests"`
	SuccessCount    int64            `json:"success_count" db:"success_count"`
	RateLimitCount  int64            `json:"rate_limit_count" db:"rate_limit_count"`
	TimeoutCount    int64            `json:"timeout_count" db:"timeout_count"`
	ErrorCount      int64            `json:"error_count" db:"error_count"`
	AvgLatencyMs    float64          `json:"avg_latency_ms" db:"avg_latency_ms"`
	P50LatencyMs    float64          `json:"p50_latency_ms" db:"p50_latency_ms"`
	P95LatencyMs    float64          `json:"p95_latency_ms" db:"p95_latency_ms"`
	P99LatencyMs    float64          `json:"p99_latency_ms" db:"p99_latency_ms"`
	MaxLatencyMs    float64          `json:"max_latency_ms" db:"max_latency_ms"`
	AvgResponseSize int64            `json:"avg_response_size" db:"avg_response_size"`
	StatusCodes     map[int]int64    `json:"status_codes" db:"status_codes"`
	HourlyPattern   []float64        `json:"hourly_pattern" db:"hourly_pattern"` // 24 values
	DayOfWeekPattern []float64       `json:"day_of_week_pattern" db:"day_of_week_pattern"` // 7 values
	CreatedAt       time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at" db:"updated_at"`
}

// RateLimitPrediction represents a predicted rate limit
type RateLimitPrediction struct {
	EndpointID        string    `json:"endpoint_id"`
	CurrentRate       float64   `json:"current_rate"`
	PredictedLimit    int       `json:"predicted_limit"`
	RecommendedRate   float64   `json:"recommended_rate"`
	Confidence        float64   `json:"confidence"`
	RiskScore         float64   `json:"risk_score"` // 0-1, chance of hitting rate limit
	PredictedWindow   int       `json:"predicted_window_seconds"`
	BackoffRecommended bool     `json:"backoff_recommended"`
	Reason            string    `json:"reason"`
	ValidUntil        time.Time `json:"valid_until"`
}

// AdaptiveRateConfig configures adaptive rate limiting
type AdaptiveRateConfig struct {
	ID               string    `json:"id" db:"id"`
	TenantID         string    `json:"tenant_id" db:"tenant_id"`
	EndpointID       string    `json:"endpoint_id" db:"endpoint_id"`
	Enabled          bool      `json:"enabled" db:"enabled"`
	Mode             RateMode  `json:"mode" db:"mode"`
	BaseRatePerSec   float64   `json:"base_rate_per_sec" db:"base_rate_per_sec"`
	MinRatePerSec    float64   `json:"min_rate_per_sec" db:"min_rate_per_sec"`
	MaxRatePerSec    float64   `json:"max_rate_per_sec" db:"max_rate_per_sec"`
	BurstSize        int       `json:"burst_size" db:"burst_size"`
	RiskThreshold    float64   `json:"risk_threshold" db:"risk_threshold"`
	BackoffFactor    float64   `json:"backoff_factor" db:"backoff_factor"`
	RecoveryFactor   float64   `json:"recovery_factor" db:"recovery_factor"`
	WindowSeconds    int       `json:"window_seconds" db:"window_seconds"`
	LearningEnabled  bool      `json:"learning_enabled" db:"learning_enabled"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// RateMode defines the rate limiting mode
type RateMode string

const (
	RateModeFixed    RateMode = "fixed"
	RateModeAdaptive RateMode = "adaptive"
	RateModeAggressive RateMode = "aggressive"
	RateModeConservative RateMode = "conservative"
)

// RateLimitState tracks current rate limit state for an endpoint
type RateLimitState struct {
	ID               string    `json:"id" db:"id"`
	TenantID         string    `json:"tenant_id" db:"tenant_id"`
	EndpointID       string    `json:"endpoint_id" db:"endpoint_id"`
	CurrentRate      float64   `json:"current_rate" db:"current_rate"`
	AllowedRate      float64   `json:"allowed_rate" db:"allowed_rate"`
	WindowStart      time.Time `json:"window_start" db:"window_start"`
	RequestCount     int64     `json:"request_count" db:"request_count"`
	RateLimitHits    int64     `json:"rate_limit_hits" db:"rate_limit_hits"`
	ConsecutiveOK    int       `json:"consecutive_ok" db:"consecutive_ok"`
	ConsecutiveFail  int       `json:"consecutive_fail" db:"consecutive_fail"`
	LastRateLimitAt  *time.Time `json:"last_rate_limit_at,omitempty" db:"last_rate_limit_at"`
	RetryAfter       *time.Time `json:"retry_after,omitempty" db:"retry_after"`
	Cooldown         bool      `json:"cooldown" db:"cooldown"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// LearningDataPoint represents a data point for ML training
type LearningDataPoint struct {
	ID           string    `json:"id" db:"id"`
	TenantID     string    `json:"tenant_id" db:"tenant_id"`
	EndpointID   string    `json:"endpoint_id" db:"endpoint_id"`
	Timestamp    time.Time `json:"timestamp" db:"timestamp"`
	HourOfDay    int       `json:"hour_of_day" db:"hour_of_day"`
	DayOfWeek    int       `json:"day_of_week" db:"day_of_week"`
	RequestRate  float64   `json:"request_rate" db:"request_rate"`
	SuccessRate  float64   `json:"success_rate" db:"success_rate"`
	AvgLatency   float64   `json:"avg_latency_ms" db:"avg_latency_ms"`
	RateLimited  bool      `json:"rate_limited" db:"rate_limited"`
	ResponseCode int       `json:"response_code" db:"response_code"`
}

// PredictionModel represents a trained prediction model for an endpoint
type PredictionModel struct {
	ID            string    `json:"id" db:"id"`
	TenantID      string    `json:"tenant_id" db:"tenant_id"`
	EndpointID    string    `json:"endpoint_id" db:"endpoint_id"`
	ModelType     string    `json:"model_type" db:"model_type"` // linear, lstm, arima
	Version       int       `json:"version" db:"version"`
	Weights       []float64 `json:"weights" db:"weights"`
	Coefficients  map[string]float64 `json:"coefficients" db:"coefficients"`
	Features      []string  `json:"features" db:"features"`
	Accuracy      float64   `json:"accuracy" db:"accuracy"`
	TrainedAt     time.Time `json:"trained_at" db:"trained_at"`
	DataPointCount int64    `json:"data_point_count" db:"data_point_count"`
	IsActive      bool      `json:"is_active" db:"is_active"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// ThrottleDecision represents a throttling decision
type ThrottleDecision struct {
	EndpointID       string        `json:"endpoint_id"`
	Allowed          bool          `json:"allowed"`
	WaitDuration     time.Duration `json:"wait_duration,omitempty"`
	Reason           string        `json:"reason"`
	CurrentRate      float64       `json:"current_rate"`
	AllowedRate      float64       `json:"allowed_rate"`
	RemainingBurst   int           `json:"remaining_burst"`
	ResetAt          time.Time     `json:"reset_at"`
}

// RateLimitEvent represents a rate limit event for analysis
type RateLimitEvent struct {
	ID           string    `json:"id" db:"id"`
	TenantID     string    `json:"tenant_id" db:"tenant_id"`
	EndpointID   string    `json:"endpoint_id" db:"endpoint_id"`
	DeliveryID   string    `json:"delivery_id" db:"delivery_id"`
	Timestamp    time.Time `json:"timestamp" db:"timestamp"`
	EventType    string    `json:"event_type" db:"event_type"` // hit, near_miss, recovery
	StatusCode   int       `json:"status_code" db:"status_code"`
	RetryAfter   int       `json:"retry_after_seconds" db:"retry_after_seconds"`
	RequestRate  float64   `json:"request_rate" db:"request_rate"`
	Headers      map[string]string `json:"headers,omitempty" db:"headers"`
}

// SmartLimitStats provides statistics for smart rate limiting
type SmartLimitStats struct {
	TenantID           string    `json:"tenant_id"`
	Period             string    `json:"period"`
	TotalEndpoints     int       `json:"total_endpoints"`
	AdaptiveEndpoints  int       `json:"adaptive_endpoints"`
	RateLimitsPrevented int64    `json:"rate_limits_prevented"`
	RateLimitsHit      int64     `json:"rate_limits_hit"`
	AvgRiskScore       float64   `json:"avg_risk_score"`
	TopRiskyEndpoints  []EndpointRisk `json:"top_risky_endpoints"`
	SavingsEstimate    float64   `json:"savings_estimate_percent"`
	GeneratedAt        time.Time `json:"generated_at"`
}

// EndpointRisk represents risk information for an endpoint
type EndpointRisk struct {
	EndpointID     string  `json:"endpoint_id"`
	URL            string  `json:"url"`
	RiskScore      float64 `json:"risk_score"`
	RateLimitCount int64   `json:"rate_limit_count"`
	Recommendation string  `json:"recommendation"`
}

// CreateAdaptiveConfigRequest creates adaptive rate config
type CreateAdaptiveConfigRequest struct {
	EndpointID      string   `json:"endpoint_id" binding:"required"`
	Mode            RateMode `json:"mode"`
	BaseRatePerSec  float64  `json:"base_rate_per_sec"`
	MinRatePerSec   float64  `json:"min_rate_per_sec"`
	MaxRatePerSec   float64  `json:"max_rate_per_sec"`
	BurstSize       int      `json:"burst_size"`
	RiskThreshold   float64  `json:"risk_threshold"`
	LearningEnabled bool     `json:"learning_enabled"`
}

// UpdateAdaptiveConfigRequest updates adaptive rate config
type UpdateAdaptiveConfigRequest struct {
	Enabled         *bool     `json:"enabled,omitempty"`
	Mode            *RateMode `json:"mode,omitempty"`
	BaseRatePerSec  *float64  `json:"base_rate_per_sec,omitempty"`
	MinRatePerSec   *float64  `json:"min_rate_per_sec,omitempty"`
	MaxRatePerSec   *float64  `json:"max_rate_per_sec,omitempty"`
	BurstSize       *int      `json:"burst_size,omitempty"`
	RiskThreshold   *float64  `json:"risk_threshold,omitempty"`
	LearningEnabled *bool     `json:"learning_enabled,omitempty"`
}
