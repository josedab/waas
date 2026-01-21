package versioning

import (
	"fmt"
	"time"
)

// Version represents a webhook schema version
type Version struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	WebhookID   string    `json:"webhook_id"`
	Major       int       `json:"major"`
	Minor       int       `json:"minor"`
	Patch       int       `json:"patch"`
	Label       string    `json:"label"` // e.g., "v1.2.3", "2024-01-15"
	SchemaID    string    `json:"schema_id"`
	Status      Status    `json:"status"`
	Changelog   string    `json:"changelog"`
	Breaking    bool      `json:"breaking"`
	
	// Deprecation info
	DeprecatedAt   *time.Time `json:"deprecated_at,omitempty"`
	SunsetAt       *time.Time `json:"sunset_at,omitempty"`
	SunsetPolicy   *SunsetPolicy `json:"sunset_policy,omitempty"`
	Replacement    string    `json:"replacement,omitempty"` // Version ID to migrate to
	
	// Compatibility
	CompatibleWith []string `json:"compatible_with"` // Version IDs
	Transforms     []Transform `json:"transforms"` // Transformations to apply
	
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
}

// Status of a version
type Status string

const (
	StatusDraft      Status = "draft"
	StatusPublished  Status = "published"
	StatusDeprecated Status = "deprecated"
	StatusSunset     Status = "sunset"
)

// SunsetPolicy defines deprecation policy
type SunsetPolicy struct {
	WarningPeriodDays int    `json:"warning_period_days"` // Days before sunset to warn
	GracePeriodDays   int    `json:"grace_period_days"`   // Days after deprecation before sunset
	EnforceAt         time.Time `json:"enforce_at"`       // When to enforce sunset
	Action            SunsetAction `json:"action"`        // What happens at sunset
}

// SunsetAction defines what happens when version is sunset
type SunsetAction string

const (
	ActionReject    SunsetAction = "reject"    // Reject requests to this version
	ActionUpgrade   SunsetAction = "upgrade"   // Auto-upgrade to replacement
	ActionWarn      SunsetAction = "warn"      // Continue but warn
)

// Transform defines a payload transformation between versions
type Transform struct {
	Type        TransformType     `json:"type"`
	SourcePath  string            `json:"source_path,omitempty"`
	TargetPath  string            `json:"target_path,omitempty"`
	Value       any               `json:"value,omitempty"`
	Condition   string            `json:"condition,omitempty"` // CEL expression
	Mapping     map[string]string `json:"mapping,omitempty"`   // For rename/remap
}

// TransformType types of transformations
type TransformType string

const (
	TransformRename    TransformType = "rename"     // Rename field
	TransformRemove    TransformType = "remove"     // Remove field
	TransformAdd       TransformType = "add"        // Add field with default
	TransformRemap     TransformType = "remap"      // Remap values
	TransformCoerce    TransformType = "coerce"     // Type coercion
	TransformFlatten   TransformType = "flatten"    // Flatten nested object
	TransformNest      TransformType = "nest"       // Nest fields
	TransformConvert   TransformType = "convert"    // Custom conversion
)

// VersionSchema defines the schema for a version
type VersionSchema struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Format      SchemaFormat           `json:"format"`
	Definition  map[string]any         `json:"definition"` // JSON Schema or Avro
	Examples    []map[string]any       `json:"examples"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// SchemaFormat supported schema formats
type SchemaFormat string

const (
	FormatJSONSchema SchemaFormat = "json_schema"
	FormatAvro       SchemaFormat = "avro"
	FormatProtobuf   SchemaFormat = "protobuf"
)

// CompatibilityResult result of compatibility check
type CompatibilityResult struct {
	Compatible      bool              `json:"compatible"`
	Direction       CompatDirection   `json:"direction"`
	BreakingChanges []BreakingChange  `json:"breaking_changes,omitempty"`
	Warnings        []string          `json:"warnings,omitempty"`
	Transforms      []Transform       `json:"transforms,omitempty"` // Suggested transforms
}

// CompatDirection compatibility direction
type CompatDirection string

const (
	CompatForward   CompatDirection = "forward"   // New consumers can read old
	CompatBackward  CompatDirection = "backward"  // Old consumers can read new
	CompatFull      CompatDirection = "full"      // Both directions
	CompatNone      CompatDirection = "none"      // Not compatible
)

// BreakingChange represents a breaking change
type BreakingChange struct {
	Type        BreakingType `json:"type"`
	Path        string       `json:"path"`
	Description string       `json:"description"`
	Severity    string       `json:"severity"` // high, medium, low
}

// BreakingType types of breaking changes
type BreakingType string

const (
	BreakingFieldRemoved    BreakingType = "field_removed"
	BreakingFieldRequired   BreakingType = "field_now_required"
	BreakingTypeChanged     BreakingType = "type_changed"
	BreakingEnumRemoved     BreakingType = "enum_value_removed"
	BreakingFormatChanged   BreakingType = "format_changed"
	BreakingRename          BreakingType = "field_renamed"
)

// VersionSubscription tracks version subscriptions
type VersionSubscription struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	EndpointID  string    `json:"endpoint_id"`
	VersionID   string    `json:"version_id"`
	WebhookID   string    `json:"webhook_id"`
	Status      SubStatus `json:"status"`
	Pinned      bool      `json:"pinned"` // Don't auto-upgrade
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SubStatus subscription status
type SubStatus string

const (
	SubActive     SubStatus = "active"
	SubMigrating  SubStatus = "migrating"
	SubSuspended  SubStatus = "suspended"
)

// Migration represents a version migration
type Migration struct {
	ID           string           `json:"id"`
	TenantID     string           `json:"tenant_id"`
	WebhookID    string           `json:"webhook_id"`
	FromVersion  string           `json:"from_version"`
	ToVersion    string           `json:"to_version"`
	Status       MigrationStatus  `json:"status"`
	Strategy     MigrationStrategy `json:"strategy"`
	Progress     MigrationProgress `json:"progress"`
	StartedAt    time.Time        `json:"started_at"`
	CompletedAt  *time.Time       `json:"completed_at,omitempty"`
	Error        string           `json:"error,omitempty"`
}

// MigrationStatus status of migration
type MigrationStatus string

const (
	MigStatusPending    MigrationStatus = "pending"
	MigStatusRunning    MigrationStatus = "running"
	MigStatusCompleted  MigrationStatus = "completed"
	MigStatusFailed     MigrationStatus = "failed"
	MigStatusRollback   MigrationStatus = "rollback"
)

// MigrationStrategy how to migrate
type MigrationStrategy string

const (
	StrategyDualWrite  MigrationStrategy = "dual_write"   // Send to both versions
	StrategyGradual    MigrationStrategy = "gradual"      // Gradual rollout
	StrategyCutover    MigrationStrategy = "cutover"      // Immediate switch
	StrategyCanary     MigrationStrategy = "canary"       // Canary deployment
)

// MigrationProgress tracks migration progress
type MigrationProgress struct {
	TotalEndpoints    int     `json:"total_endpoints"`
	MigratedEndpoints int     `json:"migrated_endpoints"`
	FailedEndpoints   int     `json:"failed_endpoints"`
	Percentage        float64 `json:"percentage"`
	CurrentPhase      string  `json:"current_phase"`
}

// DeprecationNotice notice sent to subscribers
type DeprecationNotice struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	VersionID   string    `json:"version_id"`
	EndpointID  string    `json:"endpoint_id"`
	Type        NoticeType `json:"type"`
	Message     string    `json:"message"`
	SentAt      time.Time `json:"sent_at"`
	AckedAt     *time.Time `json:"acked_at,omitempty"`
	Response    string    `json:"response,omitempty"`
}

// NoticeType type of deprecation notice
type NoticeType string

const (
	NoticeDeprecation NoticeType = "deprecation"
	NoticeSunset      NoticeType = "sunset"
	NoticeUrgent      NoticeType = "urgent"
	NoticeReminder    NoticeType = "reminder"
)

// VersionHeader headers for version negotiation
type VersionHeader struct {
	Accept         string // X-API-Version or Accept header
	ContentVersion string // Content-Version header
}

// VersionPolicy tenant-level versioning policy
type VersionPolicy struct {
	ID                    string        `json:"id"`
	TenantID              string        `json:"tenant_id"`
	Enabled               bool          `json:"enabled"`
	DefaultVersion        string        `json:"default_version"` // "latest" or specific version
	RequireVersionHeader  bool          `json:"require_version_header"`
	AllowDeprecated       bool          `json:"allow_deprecated"`
	AutoUpgrade           bool          `json:"auto_upgrade"`
	DeprecationDays       int           `json:"deprecation_days"`
	SunsetDays            int           `json:"sunset_days"`
	NotificationChannels  []string      `json:"notification_channels"`
	CreatedAt             time.Time     `json:"created_at"`
	UpdatedAt             time.Time     `json:"updated_at"`
}

// VersionNegotiator handles version negotiation
type VersionNegotiator struct {
	AcceptHeader     string   `json:"accept_header"`
	AvailableVersions []Version `json:"available_versions"`
	SelectedVersion  *Version `json:"selected_version,omitempty"`
	FallbackVersion  *Version `json:"fallback_version,omitempty"`
}

// VersionComparison compares two versions
type VersionComparison struct {
	Source       *Version          `json:"source"`
	Target       *Version          `json:"target"`
	Comparison   string            `json:"comparison"` // older, newer, same
	AddedFields  []string          `json:"added_fields"`
	RemovedFields []string         `json:"removed_fields"`
	ChangedFields []FieldChange    `json:"changed_fields"`
	CanUpgrade   bool              `json:"can_upgrade"`
	CanDowngrade bool              `json:"can_downgrade"`
}

// FieldChange represents a field change between versions
type FieldChange struct {
	Path       string `json:"path"`
	OldType    string `json:"old_type"`
	NewType    string `json:"new_type"`
	OldFormat  string `json:"old_format,omitempty"`
	NewFormat  string `json:"new_format,omitempty"`
	Breaking   bool   `json:"breaking"`
}

// VersionMetrics usage metrics for versions
type VersionMetrics struct {
	VersionID       string    `json:"version_id"`
	TenantID        string    `json:"tenant_id"`
	TotalRequests   int64     `json:"total_requests"`
	UniqueEndpoints int       `json:"unique_endpoints"`
	LastUsedAt      time.Time `json:"last_used_at"`
	RequestsByDay   []DayCount `json:"requests_by_day"`
}

// DayCount requests per day
type DayCount struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// SemanticVersion parsed semantic version
type SemanticVersion struct {
	Major      int    `json:"major"`
	Minor      int    `json:"minor"`
	Patch      int    `json:"patch"`
	Prerelease string `json:"prerelease,omitempty"`
	Build      string `json:"build,omitempty"`
}

// Compare compares with another semantic version
// Returns -1 if v < other, 0 if v == other, 1 if v > other
func (v SemanticVersion) Compare(other SemanticVersion) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}
	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}
	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}
	return 0
}

// String returns string representation
func (v SemanticVersion) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Prerelease != "" {
		s += "-" + v.Prerelease
	}
	if v.Build != "" {
		s += "+" + v.Build
	}
	return s
}

// Bump increments the version based on bump type
func (v *SemanticVersion) Bump(bumpType string) *SemanticVersion {
	result := &SemanticVersion{
		Major: v.Major,
		Minor: v.Minor,
		Patch: v.Patch,
	}
	
	switch bumpType {
	case "major":
		result.Major++
		result.Minor = 0
		result.Patch = 0
	case "minor":
		result.Minor++
		result.Patch = 0
	case "patch":
		result.Patch++
	}
	
	return result
}

// Type aliases for test compatibility
type VersionStatus = Status
type CompatibilityLevel = CompatDirection
type ChangeType = BreakingType

const (
	VersionDraft      VersionStatus = StatusDraft
	VersionActive     VersionStatus = StatusPublished
	VersionDeprecated VersionStatus = StatusDeprecated
	VersionSunset     VersionStatus = StatusSunset
)

const (
	ChangeAddField    ChangeType = "field_added"
	ChangeRemoveField ChangeType = BreakingFieldRemoved
	ChangeModifyField ChangeType = BreakingTypeChanged
	ChangeRenameField ChangeType = BreakingRename
	ChangeTypeChange  ChangeType = BreakingTypeChanged
)

// WebhookVersion alias for Version
type WebhookVersion = Version

// DeprecationPolicy alias for SunsetPolicy
type DeprecationPolicy struct {
	ID                string     `json:"id"`
	TenantID          string     `json:"tenant_id"`
	WebhookID         string     `json:"webhook_id"`
	Version           string     `json:"version"`
	DeprecationDate   time.Time  `json:"deprecation_date"`
	SunsetDate        time.Time  `json:"sunset_date"`
	Reason            string     `json:"reason"`
	MigrationGuideURL string     `json:"migration_guide_url"`
	NotifySubscribers bool       `json:"notify_subscribers"`
	NotificationSent  bool       `json:"notification_sent"`
	CreatedAt         time.Time  `json:"created_at"`
}

// CompatibilityCheck represents a compatibility check result
type CompatibilityCheck struct {
	ID                 string           `json:"id"`
	TenantID           string           `json:"tenant_id"`
	WebhookID          string           `json:"webhook_id"`
	BaseVersion        string           `json:"base_version"`
	TargetVersion      string           `json:"target_version"`
	Compatibility      CompatDirection  `json:"compatibility"`
	IsCompatible       bool             `json:"is_compatible"`
	BreakingChanges    []SchemaChange   `json:"breaking_changes"`
	NonBreakingChanges []SchemaChange   `json:"non_breaking_changes"`
	CheckedAt          time.Time        `json:"checked_at"`
}

// SchemaChange represents a schema change
type SchemaChange struct {
	Type        BreakingType `json:"type"`
	Path        string       `json:"path"`
	OldType     string       `json:"old_type,omitempty"`
	NewType     string       `json:"new_type,omitempty"`
	Description string       `json:"description"`
	Breaking    bool         `json:"breaking"`
}

// Transformation represents a version transformation
type Transformation struct {
	Type         string `json:"type"`
	From         string `json:"from,omitempty"`
	To           string `json:"to,omitempty"`
	Field        string `json:"field,omitempty"`
	DefaultValue string `json:"default_value,omitempty"`
}

// MigrationPending/Running/etc. status aliases
const (
	MigrationPending   = MigStatusPending
	MigrationRunning   = MigStatusRunning
	MigrationCompleted = MigStatusCompleted
	MigrationFailed    = MigStatusFailed
)
