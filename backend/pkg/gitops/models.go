package gitops

import "time"

// ManifestStatus constants
const (
	ManifestStatusPending    = "pending"
	ManifestStatusApplied    = "applied"
	ManifestStatusFailed     = "failed"
	ManifestStatusRolledBack = "rolled_back"
)

// ResourceStatus constants
const (
	ResourceStatusPending = "pending"
	ResourceStatusApplied = "applied"
	ResourceStatusFailed  = "failed"
)

// ResourceType constants
const (
	ResourceTypeEndpoint  = "endpoint"
	ResourceTypeTransform = "transform"
	ResourceTypeContract  = "contract"
	ResourceTypeRoute     = "route"
	ResourceTypeAlert     = "alert"
)

// ResourceAction constants
const (
	ActionCreate   = "create"
	ActionUpdate   = "update"
	ActionDelete   = "delete"
	ActionNoChange = "no_change"
)

// DriftStatus constants
const (
	DriftStatusDetected = "detected"
	DriftStatusResolved = "resolved"
	DriftStatusIgnored  = "ignored"
)

// DriftSeverity constants
const (
	DriftSeverityInfo     = "info"
	DriftSeverityWarning  = "warning"
	DriftSeverityCritical = "critical"
)

// ConfigManifest represents a configuration manifest for GitOps management
type ConfigManifest struct {
	ID        string     `json:"id" db:"id"`
	TenantID  string     `json:"tenant_id" db:"tenant_id"`
	Name      string     `json:"name" db:"name"`
	Version   int        `json:"version" db:"version"`
	Content   string     `json:"content" db:"content"`
	Checksum  string     `json:"checksum" db:"checksum"`
	Status    string     `json:"status" db:"status"`
	AppliedAt *time.Time `json:"applied_at,omitempty" db:"applied_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
}

// ConfigResource represents a resource managed by a configuration manifest
type ConfigResource struct {
	ID            string `json:"id" db:"id"`
	ManifestID    string `json:"manifest_id" db:"manifest_id"`
	ResourceType  string `json:"resource_type" db:"resource_type"`
	ResourceID    string `json:"resource_id" db:"resource_id"`
	Action        string `json:"action" db:"action"`
	PreviousState string `json:"previous_state,omitempty" db:"previous_state"`
	DesiredState  string `json:"desired_state" db:"desired_state"`
	Status        string `json:"status" db:"status"`
}

// DriftReport represents a drift detection report
type DriftReport struct {
	ID            string        `json:"id" db:"id"`
	TenantID      string        `json:"tenant_id" db:"tenant_id"`
	DetectedAt    time.Time     `json:"detected_at" db:"detected_at"`
	ResourceCount int           `json:"resource_count" db:"resource_count"`
	DriftedCount  int           `json:"drifted_count" db:"drifted_count"`
	Details       []DriftDetail `json:"details"`
	Status        string        `json:"status" db:"status"`
}

// DriftDetail describes a single drifted field on a resource
type DriftDetail struct {
	ResourceType  string `json:"resource_type"`
	ResourceID    string `json:"resource_id"`
	Field         string `json:"field"`
	ExpectedValue string `json:"expected_value"`
	ActualValue   string `json:"actual_value"`
	Severity      string `json:"severity"`
}

// ApplyPlan describes the planned actions for applying a manifest
type ApplyPlan struct {
	ManifestID    string          `json:"manifest_id"`
	Resources     []PlannedAction `json:"resources"`
	TotalCreates  int             `json:"total_creates"`
	TotalUpdates  int             `json:"total_updates"`
	TotalDeletes  int             `json:"total_deletes"`
	TotalNoChange int             `json:"total_no_change"`
	HasDestructive bool           `json:"has_destructive"`
}

// PlannedAction describes a single planned resource action
type PlannedAction struct {
	ResourceType  string `json:"resource_type"`
	ResourceID    string `json:"resource_id"`
	Action        string `json:"action"`
	Description   string `json:"description"`
	IsDestructive bool   `json:"is_destructive"`
}

// ApplyResult describes the outcome of applying a manifest
type ApplyResult struct {
	ManifestID   string        `json:"manifest_id"`
	Status       string        `json:"status"`
	AppliedCount int           `json:"applied_count"`
	FailedCount  int           `json:"failed_count"`
	Errors       []string      `json:"errors,omitempty"`
	Duration     time.Duration `json:"duration"`
}

// ValidateManifestRequest is the request DTO for validating a manifest
type ValidateManifestRequest struct {
	Content string `json:"content" binding:"required"`
}

// ApplyManifestRequest is the request DTO for applying a manifest
type ApplyManifestRequest struct {
	ManifestID string `json:"manifest_id" binding:"required"`
	DryRun     bool   `json:"dry_run"`
	Force      bool   `json:"force"`
}
