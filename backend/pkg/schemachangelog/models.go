package schemachangelog

import "time"

// SchemaVersion represents a versioned schema for an event type.
type SchemaVersion struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	EventType   string                 `json:"event_type"`
	Version     string                 `json:"version"` // semver
	Schema      map[string]interface{} `json:"schema"`  // JSON Schema
	CreatedAt   time.Time              `json:"created_at"`
	CreatedBy   string                 `json:"created_by"`
	Description string                 `json:"description"`
}

// ChangeType classifies schema changes.
type ChangeType string

const (
	ChangeBreaking    ChangeType = "breaking"
	ChangeNonBreaking ChangeType = "non-breaking"
	ChangeAddition    ChangeType = "addition"
	ChangeDeprecation ChangeType = "deprecation"
)

// SchemaChange describes a single diff between two schema versions.
type SchemaChange struct {
	Path        string     `json:"path"` // JSONPath to changed field
	ChangeType  ChangeType `json:"change_type"`
	Description string     `json:"description"`
	OldValue    string     `json:"old_value,omitempty"`
	NewValue    string     `json:"new_value,omitempty"`
}

// ChangelogEntry is a complete changelog entry between two versions.
type ChangelogEntry struct {
	ID             string         `json:"id"`
	TenantID       string         `json:"tenant_id"`
	EventType      string         `json:"event_type"`
	FromVersion    string         `json:"from_version"`
	ToVersion      string         `json:"to_version"`
	Changes        []SchemaChange `json:"changes"`
	HasBreaking    bool           `json:"has_breaking"`
	Summary        string         `json:"summary"`
	MigrationGuide string         `json:"migration_guide,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
}

// ConsumerMigration tracks a consumer's migration status.
type ConsumerMigration struct {
	ID             string     `json:"id"`
	TenantID       string     `json:"tenant_id"`
	ChangelogID    string     `json:"changelog_id"`
	EndpointID     string     `json:"endpoint_id"`
	EndpointOwner  string     `json:"endpoint_owner"`
	Status         string     `json:"status"` // pending, acknowledged, migrating, completed, skipped
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	Notes          string     `json:"notes,omitempty"`
}

// BreakingChangeNotification is sent to affected consumers.
type BreakingChangeNotification struct {
	ChangelogID       string         `json:"changelog_id"`
	EventType         string         `json:"event_type"`
	FromVersion       string         `json:"from_version"`
	ToVersion         string         `json:"to_version"`
	BreakingChanges   []SchemaChange `json:"breaking_changes"`
	MigrationGuide    string         `json:"migration_guide"`
	DeprecationDate   *time.Time     `json:"deprecation_date,omitempty"`
	AffectedEndpoints []string       `json:"affected_endpoints"`
}

// DeprecationTimeline tracks scheduled deprecations.
type DeprecationTimeline struct {
	EventType      string    `json:"event_type"`
	OldVersion     string    `json:"old_version"`
	NewVersion     string    `json:"new_version"`
	AnnouncedAt    time.Time `json:"announced_at"`
	DeprecationAt  time.Time `json:"deprecation_at"`
	SunsetAt       time.Time `json:"sunset_at"`
	MigratedCount  int       `json:"migrated_count"`
	TotalConsumers int       `json:"total_consumers"`
}

// RegisterSchemaRequest is the DTO for registering a new schema version.
type RegisterSchemaRequest struct {
	EventType   string                 `json:"event_type" binding:"required"`
	Version     string                 `json:"version" binding:"required"`
	Schema      map[string]interface{} `json:"schema" binding:"required"`
	Description string                 `json:"description"`
}
