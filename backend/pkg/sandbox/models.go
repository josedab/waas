package sandbox

import "time"

// SandboxStatus constants
const (
	StatusActive     = "active"
	StatusExpired    = "expired"
	StatusTerminated = "terminated"
)

// ReplaySessionStatus constants
const (
	ReplayStatusPending  = "pending"
	ReplayStatusReplayed = "replayed"
	ReplayStatusFailed   = "failed"
)

// MaskingStrategy constants
const (
	StrategyRedact   = "redact"
	StrategyHash     = "hash"
	StrategyFake     = "fake"
	StrategyPreserve = "preserve"
)

// SandboxEnvironment represents an isolated replay sandbox
type SandboxEnvironment struct {
	ID           string     `json:"id" db:"id"`
	TenantID     string     `json:"tenant_id" db:"tenant_id"`
	Name         string     `json:"name" db:"name"`
	Description  string     `json:"description" db:"description"`
	Status       string     `json:"status" db:"status"`
	TargetURL    string     `json:"target_url" db:"target_url"`
	MaskingRules string     `json:"masking_rules" db:"masking_rules"`
	TTLMinutes   int        `json:"ttl_minutes" db:"ttl_minutes"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

// ReplaySession represents a single webhook replay attempt
type ReplaySession struct {
	ID                string    `json:"id" db:"id"`
	TenantID          string    `json:"tenant_id" db:"tenant_id"`
	SandboxID         string    `json:"sandbox_id" db:"sandbox_id"`
	SourceEventID     string    `json:"source_event_id" db:"source_event_id"`
	OriginalPayload   string    `json:"original_payload" db:"original_payload"`
	MaskedPayload     string    `json:"masked_payload" db:"masked_payload"`
	ResponseStatus    int       `json:"response_status" db:"response_status"`
	ResponseBody      string    `json:"response_body" db:"response_body"`
	ResponseLatencyMs int64     `json:"response_latency_ms" db:"response_latency_ms"`
	ComparisonResult  string    `json:"comparison_result" db:"comparison_result"`
	Status            string    `json:"status" db:"status"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
}

// MaskingRule defines how a specific field should be masked
type MaskingRule struct {
	FieldPath   string `json:"field_path"`
	Strategy    string `json:"strategy"`
	Replacement string `json:"replacement,omitempty"`
}

// ComparisonReport aggregates replay results for a sandbox
type ComparisonReport struct {
	SandboxID       string             `json:"sandbox_id"`
	TotalReplays    int                `json:"total_replays"`
	Matches         int                `json:"matches"`
	Mismatches      int                `json:"mismatches"`
	Errors          int                `json:"errors"`
	AvgLatencyDelta int64              `json:"avg_latency_delta"`
	Details         []ComparisonDetail `json:"details"`
}

// ComparisonDetail describes the comparison outcome for a single field
type ComparisonDetail struct {
	EventID    string `json:"event_id"`
	FieldPath  string `json:"field_path"`
	Expected   string `json:"expected"`
	Actual     string `json:"actual"`
	IsMismatch bool   `json:"is_mismatch"`
}

// CreateSandboxRequest is the request DTO for creating a sandbox environment
type CreateSandboxRequest struct {
	Name         string        `json:"name" binding:"required,min=1,max=255"`
	Description  string        `json:"description" binding:"max=1024"`
	TargetURL    string        `json:"target_url" binding:"required,url"`
	MaskingRules []MaskingRule `json:"masking_rules"`
	TTLMinutes   int           `json:"ttl_minutes" binding:"required,min=1"`
}

// ReplayRequest is the request DTO for replaying events in a sandbox
type ReplayRequest struct {
	SourceEventIDs []string               `json:"source_event_ids" binding:"required,min=1"`
	ModifyPayload  map[string]interface{} `json:"modify_payload,omitempty"`
}

// FailureType constants
const (
	FailureTimeout    = "timeout"
	Failure500Error   = "500_error"
	FailureRateLimit  = "rate_limit"
	FailureDNSFailure = "dns_failure"
)

// DistributionType constants
const (
	DistributionUniform = "uniform"
	DistributionNormal  = "normal"
)

// StepType constants
const (
	StepSendWebhook    = "send_webhook"
	StepWait           = "wait"
	StepAssertDelivery = "assert_delivery"
)

// ScenarioStatus constants
const (
	ScenarioStatusPending   = "pending"
	ScenarioStatusRunning   = "running"
	ScenarioStatusCompleted = "completed"
	ScenarioStatusFailed    = "failed"
)

// MockEndpointConfig represents a configurable mock endpoint with response behavior
type MockEndpointConfig struct {
	ID              string             `json:"id" db:"id"`
	SandboxID       string             `json:"sandbox_id" db:"sandbox_id"`
	Path            string             `json:"path" db:"path"`
	Method          string             `json:"method" db:"method"`
	ResponseStatus  int                `json:"response_status" db:"response_status"`
	ResponseBody    string             `json:"response_body" db:"response_body"`
	ResponseHeaders map[string]string  `json:"response_headers"`
	Latency         *LatencySimulation `json:"latency,omitempty"`
	FailureRate     float64            `json:"failure_rate" db:"failure_rate"`
	Failures        []FailureScenario  `json:"failures,omitempty"`
	CreatedAt       time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at" db:"updated_at"`
}

// FailureScenario represents a named failure scenario with a probability
type FailureScenario struct {
	Name        string  `json:"name" db:"name"`
	Type        string  `json:"type" db:"type"`
	Probability float64 `json:"probability" db:"probability"`
	StatusCode  int     `json:"status_code,omitempty" db:"status_code"`
	Message     string  `json:"message,omitempty" db:"message"`
}

// LatencySimulation configures simulated latency behavior
type LatencySimulation struct {
	MinMs            int    `json:"min_ms" db:"min_ms"`
	MaxMs            int    `json:"max_ms" db:"max_ms"`
	DistributionType string `json:"distribution_type" db:"distribution_type"`
}

// CapturedRequest stores captured request/response data for inspection
type CapturedRequest struct {
	ID             string            `json:"id" db:"id"`
	SandboxID      string            `json:"sandbox_id" db:"sandbox_id"`
	EndpointID     string            `json:"endpoint_id" db:"endpoint_id"`
	Method         string            `json:"method" db:"method"`
	URL            string            `json:"url" db:"url"`
	RequestHeaders map[string]string `json:"request_headers"`
	RequestBody    string            `json:"request_body" db:"request_body"`
	ResponseStatus int               `json:"response_status" db:"response_status"`
	ResponseBody   string            `json:"response_body" db:"response_body"`
	LatencyMs      int64             `json:"latency_ms" db:"latency_ms"`
	FailureInjected bool            `json:"failure_injected" db:"failure_injected"`
	FailureType     string          `json:"failure_type,omitempty" db:"failure_type"`
	CapturedAt     time.Time         `json:"captured_at" db:"captured_at"`
}

// TestScenario represents a multi-step test scenario with expected outcomes
type TestScenario struct {
	ID          string     `json:"id" db:"id"`
	TenantID    string     `json:"tenant_id" db:"tenant_id"`
	SandboxID   string     `json:"sandbox_id" db:"sandbox_id"`
	Name        string     `json:"name" db:"name"`
	Description string     `json:"description" db:"description"`
	Steps       []TestStep `json:"steps"`
	Status      string     `json:"status" db:"status"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// TestStep represents an individual step within a test scenario
type TestStep struct {
	ID              string                 `json:"id" db:"id"`
	ScenarioID      string                 `json:"scenario_id" db:"scenario_id"`
	Order           int                    `json:"order" db:"order"`
	Type            string                 `json:"type" db:"type"`
	EndpointID      string                 `json:"endpoint_id,omitempty" db:"endpoint_id"`
	Payload         string                 `json:"payload,omitempty" db:"payload"`
	WaitMs          int                    `json:"wait_ms,omitempty" db:"wait_ms"`
	ExpectedStatus  int                    `json:"expected_status,omitempty" db:"expected_status"`
	ExpectedBody    string                 `json:"expected_body,omitempty" db:"expected_body"`
	Assertions      map[string]interface{} `json:"assertions,omitempty"`
}

// ScenarioResult holds the overall result of a test scenario execution
type ScenarioResult struct {
	ID          string       `json:"id" db:"id"`
	ScenarioID  string       `json:"scenario_id" db:"scenario_id"`
	Status      string       `json:"status" db:"status"`
	TotalSteps  int          `json:"total_steps" db:"total_steps"`
	PassedSteps int          `json:"passed_steps" db:"passed_steps"`
	FailedSteps int          `json:"failed_steps" db:"failed_steps"`
	Steps       []StepResult `json:"steps"`
	StartedAt   time.Time    `json:"started_at" db:"started_at"`
	CompletedAt time.Time    `json:"completed_at" db:"completed_at"`
}

// StepResult holds the result of an individual test step
type StepResult struct {
	ID             string    `json:"id" db:"id"`
	ScenarioID     string    `json:"scenario_id" db:"scenario_id"`
	StepID         string    `json:"step_id" db:"step_id"`
	StepOrder      int       `json:"step_order" db:"step_order"`
	Status         string    `json:"status" db:"status"`
	ActualStatus   int       `json:"actual_status,omitempty" db:"actual_status"`
	ActualBody     string    `json:"actual_body,omitempty" db:"actual_body"`
	ErrorMessage   string    `json:"error_message,omitempty" db:"error_message"`
	LatencyMs      int64     `json:"latency_ms" db:"latency_ms"`
	ExecutedAt     time.Time `json:"executed_at" db:"executed_at"`
}

// CreateMockEndpointRequest is the request DTO for creating a mock endpoint
type CreateMockEndpointRequest struct {
	Path            string             `json:"path" binding:"required"`
	Method          string             `json:"method" binding:"required"`
	ResponseStatus  int                `json:"response_status" binding:"required"`
	ResponseBody    string             `json:"response_body"`
	ResponseHeaders map[string]string  `json:"response_headers"`
	Latency         *LatencySimulation `json:"latency,omitempty"`
	FailureRate     float64            `json:"failure_rate"`
	Failures        []FailureScenario  `json:"failures,omitempty"`
}

// InjectChaosRequest is the request DTO for injecting chaos into a sandbox
type InjectChaosRequest struct {
	FailureType string  `json:"failure_type" binding:"required"`
	Probability float64 `json:"probability" binding:"required,min=0,max=1"`
}

// CreateTestScenarioRequest is the request DTO for creating a test scenario
type CreateTestScenarioRequest struct {
	SandboxID   string     `json:"sandbox_id" binding:"required"`
	Name        string     `json:"name" binding:"required,min=1,max=255"`
	Description string     `json:"description" binding:"max=1024"`
	Steps       []TestStep `json:"steps" binding:"required,min=1"`
}
