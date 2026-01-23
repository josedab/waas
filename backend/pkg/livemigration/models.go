package livemigration

import "time"

// Platform constants
const (
	PlatformSvix     = "svix"
	PlatformConvoy   = "convoy"
	PlatformHookdeck = "hookdeck"
	PlatformCustom   = "custom"
)

// Job status constants
const (
	JobStatusPending         = "pending"
	JobStatusDiscovering     = "discovering"
	JobStatusImporting       = "importing"
	JobStatusValidating      = "validating"
	JobStatusParallelRunning = "parallel_running"
	JobStatusCuttingOver     = "cutting_over"
	JobStatusCompleted       = "completed"
	JobStatusFailed          = "failed"
	JobStatusRolledBack      = "rolled_back"
)

// Endpoint status constants
const (
	EndpointStatusPending   = "pending"
	EndpointStatusImported  = "imported"
	EndpointStatusValidated = "validated"
	EndpointStatusActive    = "active"
	EndpointStatusFailed    = "failed"
)

// Recommendation constants
const (
	RecommendationProceed = "proceed"
	RecommendationWait    = "wait"
	RecommendationAbort   = "abort"
)

// Risk level constants
const (
	RiskLevelLow    = "low"
	RiskLevelMedium = "medium"
	RiskLevelHigh   = "high"
)

// MigrationJob represents a webhook migration job from a source platform
type MigrationJob struct {
	ID             string     `json:"id" db:"id"`
	TenantID       string     `json:"tenant_id" db:"tenant_id"`
	Name           string     `json:"name" db:"name"`
	SourcePlatform string     `json:"source_platform" db:"source_platform"`
	SourceConfig   string     `json:"source_config" db:"source_config"`
	Status         string     `json:"status" db:"status"`
	EndpointCount  int        `json:"endpoint_count" db:"endpoint_count"`
	MigratedCount  int        `json:"migrated_count" db:"migrated_count"`
	FailedCount    int        `json:"failed_count" db:"failed_count"`
	StartedAt      *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// MigrationEndpoint represents an endpoint being migrated
type MigrationEndpoint struct {
	ID            string    `json:"id" db:"id"`
	TenantID      string    `json:"tenant_id" db:"tenant_id"`
	JobID         string    `json:"job_id" db:"job_id"`
	SourceID      string    `json:"source_id" db:"source_id"`
	SourceURL     string    `json:"source_url" db:"source_url"`
	DestinationID string    `json:"destination_id,omitempty" db:"destination_id"`
	Status        string    `json:"status" db:"status"`
	MappingConfig string    `json:"mapping_config,omitempty" db:"mapping_config"`
	ErrorMessage  string    `json:"error_message,omitempty" db:"error_message"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// ParallelDeliveryResult captures the result of a dual-write comparison
type ParallelDeliveryResult struct {
	ID             string    `json:"id" db:"id"`
	TenantID       string    `json:"tenant_id" db:"tenant_id"`
	JobID          string    `json:"job_id" db:"job_id"`
	EndpointID     string    `json:"endpoint_id" db:"endpoint_id"`
	EventID        string    `json:"event_id" db:"event_id"`
	SourceStatus   int       `json:"source_status" db:"source_status"`
	DestStatus     int       `json:"dest_status" db:"dest_status"`
	SourceLatencyMs int64    `json:"source_latency_ms" db:"source_latency_ms"`
	DestLatencyMs  int64     `json:"dest_latency_ms" db:"dest_latency_ms"`
	ResponseMatch  bool      `json:"response_match" db:"response_match"`
	Discrepancy    string    `json:"discrepancy,omitempty" db:"discrepancy"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// CutoverPlan provides a recommendation for cutting over to the new platform
type CutoverPlan struct {
	JobID              string        `json:"job_id"`
	TotalEndpoints     int           `json:"total_endpoints"`
	ReadyCount         int           `json:"ready_count"`
	FailedCount        int           `json:"failed_count"`
	ParallelSuccessRate float64      `json:"parallel_success_rate"`
	Recommendation     string        `json:"recommendation"`
	RiskLevel          string        `json:"risk_level"`
	Steps              []CutoverStep `json:"steps"`
}

// CutoverStep represents a step in the cutover plan
type CutoverStep struct {
	Order       int    `json:"order"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// MigrationStats aggregates statistics for a migration job
type MigrationStats struct {
	JobID              string  `json:"job_id"`
	SourcePlatform     string  `json:"source_platform"`
	TotalEndpoints     int     `json:"total_endpoints"`
	ImportedCount      int     `json:"imported_count"`
	ValidatedCount     int     `json:"validated_count"`
	ParallelDeliveries int64   `json:"parallel_deliveries"`
	MatchRate          float64 `json:"match_rate"`
	AvgLatencyDelta    int64   `json:"avg_latency_delta"`
}

// CreateMigrationRequest is the request DTO for creating a migration job
type CreateMigrationRequest struct {
	Name           string `json:"name" binding:"required,min=1,max=255"`
	SourcePlatform string `json:"source_platform" binding:"required"`
	SourceConfig   string `json:"source_config" binding:"required"`
}

// ImportEndpointsRequest is the request DTO for importing endpoints
type ImportEndpointsRequest struct {
	JobID string `json:"job_id" binding:"required"`
}

// StartParallelRequest is the request DTO for starting parallel delivery
type StartParallelRequest struct {
	JobID           string  `json:"job_id" binding:"required"`
	DurationMinutes int     `json:"duration_minutes" binding:"required,min=1"`
	SampleRate      float64 `json:"sample_rate" binding:"required,min=0,max=1"`
}
