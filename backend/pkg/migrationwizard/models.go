package migrationwizard

import "time"

// SourcePlatform identifies the platform being migrated from.
type SourcePlatform string

const (
	PlatformSvix        SourcePlatform = "svix"
	PlatformHookdeck    SourcePlatform = "hookdeck"
	PlatformConvoy      SourcePlatform = "convoy"
	PlatformEventBridge SourcePlatform = "aws_eventbridge"
	PlatformCustom      SourcePlatform = "custom"
)

// MigrationStatus tracks the overall state of a migration.
type MigrationStatus string

const (
	MigrationPending    MigrationStatus = "pending"
	MigrationAnalyzing  MigrationStatus = "analyzing"
	MigrationReady      MigrationStatus = "ready"
	MigrationInProgress MigrationStatus = "in_progress"
	MigrationValidating MigrationStatus = "validating"
	MigrationCompleted  MigrationStatus = "completed"
	MigrationFailed     MigrationStatus = "failed"
	MigrationRolledBack MigrationStatus = "rolled_back"
)

// Migration represents a full platform migration job.
type Migration struct {
	ID           string             `json:"id" db:"id"`
	TenantID     string             `json:"tenant_id" db:"tenant_id"`
	Source       SourcePlatform     `json:"source_platform" db:"source_platform"`
	Status       MigrationStatus    `json:"status" db:"status"`
	Config       *MigrationConfig   `json:"config"`
	Analysis     *MigrationAnalysis `json:"analysis,omitempty"`
	Progress     *MigrationProgress `json:"progress"`
	ErrorMessage string             `json:"error_message,omitempty" db:"error_message"`
	StartedAt    *time.Time         `json:"started_at,omitempty" db:"started_at"`
	CompletedAt  *time.Time         `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt    time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at" db:"updated_at"`
}

// MigrationConfig holds configuration for the migration process.
type MigrationConfig struct {
	SourceAPIKey     string `json:"source_api_key"`
	SourceBaseURL    string `json:"source_base_url,omitempty"`
	DualWriteEnabled bool   `json:"dual_write_enabled"`
	DryRun           bool   `json:"dry_run"`
	BatchSize        int    `json:"batch_size"`
	SkipValidation   bool   `json:"skip_validation"`
}

// MigrationAnalysis contains the pre-migration analysis results.
type MigrationAnalysis struct {
	EndpointsFound     int               `json:"endpoints_found"`
	EventTypesFound    int               `json:"event_types_found"`
	CompatibilityScore float64           `json:"compatibility_score"` // 0-100
	Issues             []MigrationIssue  `json:"issues"`
	Mappings           []ResourceMapping `json:"mappings"`
	EstimatedDuration  string            `json:"estimated_duration"`
}

// MigrationIssue describes a potential problem during migration.
type MigrationIssue struct {
	Severity    string `json:"severity"` // warning, error, info
	Resource    string `json:"resource"`
	Description string `json:"description"`
	Resolution  string `json:"resolution,omitempty"`
}

// ResourceMapping maps a source platform resource to its WaaS equivalent.
type ResourceMapping struct {
	SourceType string `json:"source_type"`
	SourceID   string `json:"source_id"`
	SourceName string `json:"source_name"`
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id,omitempty"`
	Status     string `json:"status"` // pending, migrated, skipped, failed
}

// MigrationProgress tracks how far the migration has progressed.
type MigrationProgress struct {
	TotalResources    int             `json:"total_resources"`
	MigratedResources int             `json:"migrated_resources"`
	FailedResources   int             `json:"failed_resources"`
	SkippedResources  int             `json:"skipped_resources"`
	PercentComplete   float64         `json:"percent_complete"`
	CurrentStep       string          `json:"current_step"`
	Steps             []MigrationStep `json:"steps"`
}

// MigrationStep represents a discrete step in the migration process.
type MigrationStep struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // pending, running, completed, failed
	Details string `json:"details,omitempty"`
}

// StartMigrationRequest is the API request to begin a migration.
type StartMigrationRequest struct {
	SourcePlatform SourcePlatform `json:"source_platform" binding:"required"`
	SourceAPIKey   string         `json:"source_api_key" binding:"required"`
	SourceBaseURL  string         `json:"source_base_url,omitempty"`
	DualWrite      bool           `json:"dual_write"`
	DryRun         bool           `json:"dry_run"`
}

// MigrationSummary is a lightweight view for listing migrations.
type MigrationSummary struct {
	ID        string          `json:"id"`
	Source    SourcePlatform  `json:"source_platform"`
	Status    MigrationStatus `json:"status"`
	Progress  float64         `json:"percent_complete"`
	CreatedAt time.Time       `json:"created_at"`
}
