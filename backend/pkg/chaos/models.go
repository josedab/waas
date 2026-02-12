package chaos

import (
	"fmt"
	"time"
)

// ExperimentType defines the type of chaos experiment
type ExperimentType string

const (
	ExperimentLatency     ExperimentType = "latency"
	ExperimentError       ExperimentType = "error"
	ExperimentTimeout     ExperimentType = "timeout"
	ExperimentRateLimit   ExperimentType = "rate_limit"
	ExperimentPacketLoss  ExperimentType = "packet_loss"
	ExperimentCorruption  ExperimentType = "corruption"
	ExperimentBlackhole   ExperimentType = "blackhole"
)

// ExperimentStatus represents the status of an experiment
type ExperimentStatus string

const (
	StatusPending   ExperimentStatus = "pending"
	StatusRunning   ExperimentStatus = "running"
	StatusCompleted ExperimentStatus = "completed"
	StatusFailed    ExperimentStatus = "failed"
	StatusAborted   ExperimentStatus = "aborted"
	StatusScheduled ExperimentStatus = "scheduled"
)

// ChaosExperiment represents a chaos engineering experiment
type ChaosExperiment struct {
	ID            string           `json:"id" db:"id"`
	TenantID      string           `json:"tenant_id" db:"tenant_id"`
	Name          string           `json:"name" db:"name"`
	Description   string           `json:"description,omitempty" db:"description"`
	Type          ExperimentType   `json:"type" db:"type"`
	Status        ExperimentStatus `json:"status" db:"status"`
	TargetConfig  TargetConfig     `json:"target_config" db:"target_config"`
	FaultConfig   FaultConfig      `json:"fault_config" db:"fault_config"`
	Schedule      *ScheduleConfig  `json:"schedule,omitempty" db:"schedule"`
	BlastRadius   BlastRadius      `json:"blast_radius" db:"blast_radius"`
	Duration      int              `json:"duration_seconds" db:"duration_seconds"`
	StartedAt     *time.Time       `json:"started_at,omitempty" db:"started_at"`
	CompletedAt   *time.Time       `json:"completed_at,omitempty" db:"completed_at"`
	Results       *ExperimentResult `json:"results,omitempty" db:"results"`
	CreatedBy     string           `json:"created_by" db:"created_by"`
	CreatedAt     time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at" db:"updated_at"`
}

// TargetConfig defines what to target with chaos
type TargetConfig struct {
	EndpointIDs   []string          `json:"endpoint_ids,omitempty"`
	URLPatterns   []string          `json:"url_patterns,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
	Percentage    float64           `json:"percentage"` // % of traffic affected
	Environment   string            `json:"environment,omitempty"` // Only in specific env
}

// FaultConfig defines the fault to inject
type FaultConfig struct {
	// Latency injection
	LatencyMs      int     `json:"latency_ms,omitempty"`
	LatencyJitter  int     `json:"latency_jitter_ms,omitempty"` // Random variance
	
	// Error injection
	ErrorCode      int     `json:"error_code,omitempty"`
	ErrorMessage   string  `json:"error_message,omitempty"`
	ErrorRate      float64 `json:"error_rate,omitempty"` // 0-1
	
	// Timeout
	TimeoutMs      int     `json:"timeout_ms,omitempty"`
	
	// Rate limiting
	RateLimitCode  int     `json:"rate_limit_code,omitempty"`
	RateLimitAfter int     `json:"rate_limit_after,omitempty"` // Requests before limiting
	RetryAfterSec  int     `json:"retry_after_seconds,omitempty"`
	
	// Packet loss / network issues
	PacketLossRate float64 `json:"packet_loss_rate,omitempty"`
	
	// Payload corruption
	CorruptPayload  bool    `json:"corrupt_payload,omitempty"`
	CorruptHeaders  bool    `json:"corrupt_headers,omitempty"`
	CorruptionRate  float64 `json:"corruption_rate,omitempty"`
}

// ScheduleConfig defines when to run experiments
type ScheduleConfig struct {
	Type        string    `json:"type"` // once, recurring, cron
	StartTime   time.Time `json:"start_time,omitempty"`
	EndTime     time.Time `json:"end_time,omitempty"`
	CronExpr    string    `json:"cron_expression,omitempty"`
	Timezone    string    `json:"timezone,omitempty"`
	MaxRuns     int       `json:"max_runs,omitempty"`
	CurrentRuns int       `json:"current_runs,omitempty"`
}

// BlastRadius controls the scope of chaos
type BlastRadius struct {
	MaxAffectedEndpoints   int     `json:"max_affected_endpoints"`
	MaxAffectedDeliveries  int64   `json:"max_affected_deliveries"`
	MaxErrorRate           float64 `json:"max_error_rate"` // Stop if error rate exceeds
	RequireApproval        bool    `json:"require_approval"`
	AutoRollbackThreshold  float64 `json:"auto_rollback_threshold"`
}

// ExperimentResult holds the results of an experiment
type ExperimentResult struct {
	TotalDeliveries     int64             `json:"total_deliveries"`
	AffectedDeliveries  int64             `json:"affected_deliveries"`
	SuccessfulRecovery  int64             `json:"successful_recovery"`
	FailedRecovery      int64             `json:"failed_recovery"`
	AvgRecoveryTimeMs   float64           `json:"avg_recovery_time_ms"`
	P95RecoveryTimeMs   float64           `json:"p95_recovery_time_ms"`
	ResilienceScore     float64           `json:"resilience_score"` // 0-100
	ByEndpoint          []EndpointResult  `json:"by_endpoint"`
	Observations        []string          `json:"observations"`
	Recommendations     []string          `json:"recommendations"`
}

// EndpointResult holds per-endpoint results
type EndpointResult struct {
	EndpointID       string  `json:"endpoint_id"`
	URL              string  `json:"url"`
	Affected         int64   `json:"affected_deliveries"`
	Recovered        int64   `json:"recovered"`
	FailedPermanent  int64   `json:"failed_permanent"`
	RecoveryRate     float64 `json:"recovery_rate"`
	AvgRecoveryTime  float64 `json:"avg_recovery_time_ms"`
	ResilienceScore  float64 `json:"resilience_score"`
}

// ChaosEvent represents an event during chaos experiment
type ChaosEvent struct {
	ID            string    `json:"id" db:"id"`
	ExperimentID  string    `json:"experiment_id" db:"experiment_id"`
	TenantID      string    `json:"tenant_id" db:"tenant_id"`
	EndpointID    string    `json:"endpoint_id" db:"endpoint_id"`
	DeliveryID    string    `json:"delivery_id" db:"delivery_id"`
	EventType     string    `json:"event_type" db:"event_type"`
	InjectedFault string    `json:"injected_fault" db:"injected_fault"`
	OriginalState string    `json:"original_state,omitempty" db:"original_state"`
	InjectedState string    `json:"injected_state,omitempty" db:"injected_state"`
	Recovered     bool      `json:"recovered" db:"recovered"`
	RecoveryTime  int64     `json:"recovery_time_ms,omitempty" db:"recovery_time_ms"`
	Timestamp     time.Time `json:"timestamp" db:"timestamp"`
}

// ResilienceReport provides a resilience assessment
type ResilienceReport struct {
	TenantID        string            `json:"tenant_id"`
	GeneratedAt     time.Time         `json:"generated_at"`
	Period          string            `json:"period"`
	OverallScore    float64           `json:"overall_score"` // 0-100
	Grade           string            `json:"grade"` // A, B, C, D, F
	ByCategory      []CategoryScore   `json:"by_category"`
	ByEndpoint      []EndpointScore   `json:"by_endpoint"`
	Strengths       []string          `json:"strengths"`
	Weaknesses      []string          `json:"weaknesses"`
	Recommendations []Recommendation  `json:"recommendations"`
	ExperimentCount int               `json:"experiment_count"`
}

// CategoryScore scores a resilience category
type CategoryScore struct {
	Category    string  `json:"category"`
	Score       float64 `json:"score"`
	MaxScore    float64 `json:"max_score"`
	Description string  `json:"description"`
}

// EndpointScore scores endpoint resilience
type EndpointScore struct {
	EndpointID  string  `json:"endpoint_id"`
	URL         string  `json:"url"`
	Score       float64 `json:"score"`
	Experiments int     `json:"experiments_run"`
	LastTested  time.Time `json:"last_tested"`
}

// Recommendation provides improvement suggestions
type Recommendation struct {
	Priority    string `json:"priority"` // high, medium, low
	Category    string `json:"category"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
}

// CreateExperimentRequest creates a new experiment
type CreateExperimentRequest struct {
	Name         string         `json:"name" binding:"required"`
	Description  string         `json:"description,omitempty"`
	Type         ExperimentType `json:"type" binding:"required"`
	TargetConfig TargetConfig   `json:"target_config" binding:"required"`
	FaultConfig  FaultConfig    `json:"fault_config" binding:"required"`
	Schedule     *ScheduleConfig `json:"schedule,omitempty"`
	BlastRadius  BlastRadius    `json:"blast_radius"`
	Duration     int            `json:"duration_seconds" binding:"required,min=1,max=3600"`
}

// UpdateExperimentRequest updates an experiment
type UpdateExperimentRequest struct {
	Name         *string        `json:"name,omitempty"`
	Description  *string        `json:"description,omitempty"`
	TargetConfig *TargetConfig  `json:"target_config,omitempty"`
	FaultConfig  *FaultConfig   `json:"fault_config,omitempty"`
	Schedule     *ScheduleConfig `json:"schedule,omitempty"`
	BlastRadius  *BlastRadius   `json:"blast_radius,omitempty"`
	Duration     *int           `json:"duration_seconds,omitempty"`
}

// ExperimentTemplate provides pre-built experiment templates
type ExperimentTemplate struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Category    string         `json:"category"`
	Type        ExperimentType `json:"type"`
	FaultConfig FaultConfig    `json:"fault_config"`
	BlastRadius BlastRadius    `json:"blast_radius"`
	Duration    int            `json:"duration_seconds"`
}

// GetTemplates returns predefined chaos experiment templates
func GetTemplates() []ExperimentTemplate {
	return []ExperimentTemplate{
		{
			ID:          "latency-spike",
			Name:        "Latency Spike",
			Description: "Inject 500ms latency to test timeout handling",
			Category:    "latency",
			Type:        ExperimentLatency,
			FaultConfig: FaultConfig{
				LatencyMs:     500,
				LatencyJitter: 100,
			},
			BlastRadius: BlastRadius{
				MaxAffectedEndpoints:  5,
				MaxAffectedDeliveries: 100,
				AutoRollbackThreshold: 0.5,
			},
			Duration: 300,
		},
		{
			ID:          "server-error",
			Name:        "Server Error (500)",
			Description: "Simulate server errors to test retry logic",
			Category:    "errors",
			Type:        ExperimentError,
			FaultConfig: FaultConfig{
				ErrorCode:    500,
				ErrorMessage: "Internal Server Error (Chaos Injection)",
				ErrorRate:    0.3,
			},
			BlastRadius: BlastRadius{
				MaxAffectedEndpoints:  3,
				MaxAffectedDeliveries: 50,
				AutoRollbackThreshold: 0.3,
			},
			Duration: 180,
		},
		{
			ID:          "rate-limit",
			Name:        "Rate Limit (429)",
			Description: "Simulate rate limiting to test backoff",
			Category:    "rate-limiting",
			Type:        ExperimentRateLimit,
			FaultConfig: FaultConfig{
				RateLimitCode:  429,
				RateLimitAfter: 5,
				RetryAfterSec:  30,
			},
			BlastRadius: BlastRadius{
				MaxAffectedEndpoints:  3,
				MaxAffectedDeliveries: 100,
				AutoRollbackThreshold: 0.4,
			},
			Duration: 300,
		},
		{
			ID:          "timeout",
			Name:        "Request Timeout",
			Description: "Simulate timeouts to test timeout handling",
			Category:    "timeout",
			Type:        ExperimentTimeout,
			FaultConfig: FaultConfig{
				TimeoutMs: 30000,
			},
			BlastRadius: BlastRadius{
				MaxAffectedEndpoints:  2,
				MaxAffectedDeliveries: 30,
				AutoRollbackThreshold: 0.3,
			},
			Duration: 120,
		},
		{
			ID:          "network-flaky",
			Name:        "Flaky Network",
			Description: "Simulate packet loss for unreliable network",
			Category:    "network",
			Type:        ExperimentPacketLoss,
			FaultConfig: FaultConfig{
				PacketLossRate: 0.1,
			},
			BlastRadius: BlastRadius{
				MaxAffectedEndpoints:  5,
				MaxAffectedDeliveries: 100,
				AutoRollbackThreshold: 0.4,
			},
			Duration: 300,
		},
		{
			ID:          "blackhole",
			Name:        "Blackhole",
			Description: "Complete endpoint unavailability",
			Category:    "availability",
			Type:        ExperimentBlackhole,
			FaultConfig: FaultConfig{
				ErrorCode:    0, // No response
				ErrorRate:    1.0,
			},
			BlastRadius: BlastRadius{
				MaxAffectedEndpoints:  1,
				MaxAffectedDeliveries: 20,
				AutoRollbackThreshold: 0.2,
				RequireApproval:       true,
			},
			Duration: 60,
		},
	}
}

// FaultType defines types of faults for testing
type FaultType string

const (
	FaultLatency    FaultType = "latency"
	FaultError      FaultType = "error"
	FaultTimeout    FaultType = "timeout"
	FaultRateLimit  FaultType = "rate_limit"
	FaultPacketLoss FaultType = "packet_loss"
	FaultBlackhole  FaultType = "blackhole"
)

// Experiment is an alias for chaos experiment used in tests
type Experiment struct {
	ID          string       `json:"id"`
	TenantID    string       `json:"tenant_id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	TargetType  string       `json:"target_type"`
	TargetIDs   []string     `json:"target_ids"`
	FaultConfig FaultConfig  `json:"fault_config"`
	Schedule    *Schedule    `json:"schedule,omitempty"`
	Status      ExperimentStatus `json:"status"`
	CreatedAt   time.Time    `json:"created_at"`
}

// Schedule defines experiment scheduling
type Schedule struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// ResilienceScore represents endpoint resilience metrics
type ResilienceScore struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenant_id"`
	WebhookID        string    `json:"webhook_id"`
	OverallScore     float64   `json:"overall_score"`
	LatencyScore     float64   `json:"latency_score"`
	ErrorScore       float64   `json:"error_score"`
	RecoveryScore    float64   `json:"recovery_score"`
	ExperimentCount  int       `json:"experiment_count"`
	LastExperimentID string    `json:"last_experiment_id"`
	CalculatedAt     time.Time `json:"calculated_at"`
}

// Validate validates the fault configuration
func (f *FaultConfig) Validate() error {
	// For test compatibility, validate percentage-like fields
	if f.ErrorRate < 0 || f.ErrorRate > 100 {
		return fmt.Errorf("error rate must be between 0 and 100")
	}
	if f.PacketLossRate < 0 || f.PacketLossRate > 100 {
		return fmt.Errorf("packet loss rate must be between 0 and 100")
	}
	return nil
}
