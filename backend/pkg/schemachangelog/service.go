package schemachangelog

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service implements schema changelog business logic.
type Service struct {
	repo Repository
}

// NewService creates a new schema changelog service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// RegisterSchema registers a new schema version and auto-generates changelog.
func (s *Service) RegisterSchema(tenantID string, req *RegisterSchemaRequest) (*SchemaVersion, *ChangelogEntry, error) {
	if req.EventType == "" || req.Version == "" {
		return nil, nil, fmt.Errorf("event_type and version are required")
	}
	if req.Schema == nil {
		return nil, nil, fmt.Errorf("schema is required")
	}

	sv := &SchemaVersion{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		EventType:   req.EventType,
		Version:     req.Version,
		Schema:      req.Schema,
		Description: req.Description,
		CreatedAt:   time.Now(),
	}

	if err := s.repo.CreateSchemaVersion(sv); err != nil {
		return nil, nil, fmt.Errorf("failed to create schema version: %w", err)
	}

	// Check for previous version to generate changelog
	versions, _ := s.repo.ListSchemaVersions(req.EventType)
	if len(versions) > 1 {
		// Find previous version
		var prev *SchemaVersion
		for _, v := range versions {
			if v.ID != sv.ID {
				if prev == nil || v.CreatedAt.After(prev.CreatedAt) {
					prev = v
				}
			}
		}
		if prev != nil {
			entry, err := s.generateChangelog(tenantID, prev, sv)
			if err == nil {
				return sv, entry, nil
			}
		}
	}

	return sv, nil, nil
}

// GetChangelogs returns all changelogs for an event type.
func (s *Service) GetChangelogs(eventType string) ([]*ChangelogEntry, error) {
	return s.repo.ListChangelogs(eventType)
}

// GetChangelog retrieves a specific changelog.
func (s *Service) GetChangelog(id string) (*ChangelogEntry, error) {
	return s.repo.GetChangelog(id)
}

// GetSchemaVersions returns all versions of a schema.
func (s *Service) GetSchemaVersions(eventType string) ([]*SchemaVersion, error) {
	return s.repo.ListSchemaVersions(eventType)
}

// CompareVersions generates a diff between two schema versions.
func (s *Service) CompareVersions(eventType, fromVersion, toVersion string) (*ChangelogEntry, error) {
	from, err := s.repo.GetSchemaVersion(eventType, fromVersion)
	if err != nil {
		return nil, fmt.Errorf("from version not found: %w", err)
	}
	to, err := s.repo.GetSchemaVersion(eventType, toVersion)
	if err != nil {
		return nil, fmt.Errorf("to version not found: %w", err)
	}
	return s.generateChangelog(from.TenantID, from, to)
}

// CreateMigrationTracking creates migration tracking entries for affected consumers.
func (s *Service) CreateMigrationTracking(tenantID, changelogID string, endpointIDs []string) ([]*ConsumerMigration, error) {
	var migrations []*ConsumerMigration
	for _, epID := range endpointIDs {
		m := &ConsumerMigration{
			ID:          uuid.New().String(),
			TenantID:    tenantID,
			ChangelogID: changelogID,
			EndpointID:  epID,
			Status:      "pending",
		}
		if err := s.repo.CreateMigration(m); err != nil {
			return nil, err
		}
		migrations = append(migrations, m)
	}
	return migrations, nil
}

// AcknowledgeMigration marks a migration as acknowledged.
func (s *Service) AcknowledgeMigration(migrationID string) (*ConsumerMigration, error) {
	m, err := s.repo.GetMigration(migrationID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	m.Status = "acknowledged"
	m.AcknowledgedAt = &now
	if err := s.repo.UpdateMigration(m); err != nil {
		return nil, err
	}
	return m, nil
}

// CompleteMigration marks a migration as completed.
func (s *Service) CompleteMigration(migrationID string) (*ConsumerMigration, error) {
	m, err := s.repo.GetMigration(migrationID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	m.Status = "completed"
	m.CompletedAt = &now
	if err := s.repo.UpdateMigration(m); err != nil {
		return nil, err
	}
	return m, nil
}

// GetMigrationStatus returns migration status for a changelog entry.
func (s *Service) GetMigrationStatus(changelogID string) ([]*ConsumerMigration, error) {
	return s.repo.ListMigrations(changelogID)
}

func (s *Service) generateChangelog(tenantID string, from, to *SchemaVersion) (*ChangelogEntry, error) {
	changes := diffSchemas("", from.Schema, to.Schema)

	hasBreaking := false
	for _, c := range changes {
		if c.ChangeType == ChangeBreaking {
			hasBreaking = true
			break
		}
	}

	entry := &ChangelogEntry{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		EventType:   from.EventType,
		FromVersion: from.Version,
		ToVersion:   to.Version,
		Changes:     changes,
		HasBreaking: hasBreaking,
		Summary:     generateSummary(changes),
		CreatedAt:   time.Now(),
	}

	if hasBreaking {
		entry.MigrationGuide = generateMigrationGuide(changes)
	}

	if err := s.repo.CreateChangelog(entry); err != nil {
		return nil, err
	}
	return entry, nil
}

// diffSchemas compares two JSON schemas and produces a list of changes.
func diffSchemas(prefix string, old, new map[string]interface{}) []SchemaChange {
	var changes []SchemaChange

	// Check for removed fields (breaking)
	for key, oldVal := range old {
		path := joinPath(prefix, key)
		if _, exists := new[key]; !exists {
			changes = append(changes, SchemaChange{
				Path:        path,
				ChangeType:  ChangeBreaking,
				Description: fmt.Sprintf("Field '%s' was removed", path),
				OldValue:    fmt.Sprintf("%v", oldVal),
			})
			continue
		}

		newVal := new[key]
		// Type change (breaking)
		oldType := fmt.Sprintf("%T", oldVal)
		newType := fmt.Sprintf("%T", newVal)
		if oldType != newType {
			changes = append(changes, SchemaChange{
				Path:        path,
				ChangeType:  ChangeBreaking,
				Description: fmt.Sprintf("Field '%s' type changed from %s to %s", path, oldType, newType),
				OldValue:    oldType,
				NewValue:    newType,
			})
			continue
		}

		// Recurse into nested objects
		if oldMap, ok := oldVal.(map[string]interface{}); ok {
			if newMap, ok := newVal.(map[string]interface{}); ok {
				changes = append(changes, diffSchemas(path, oldMap, newMap)...)
			}
		}
	}

	// Check for added fields (non-breaking)
	for key := range new {
		path := joinPath(prefix, key)
		if _, exists := old[key]; !exists {
			changes = append(changes, SchemaChange{
				Path:        path,
				ChangeType:  ChangeAddition,
				Description: fmt.Sprintf("New field '%s' added", path),
				NewValue:    fmt.Sprintf("%v", new[key]),
			})
		}
	}

	return changes
}

func joinPath(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

func generateSummary(changes []SchemaChange) string {
	breaking := 0
	additions := 0
	for _, c := range changes {
		switch c.ChangeType {
		case ChangeBreaking:
			breaking++
		case ChangeAddition:
			additions++
		}
	}

	var parts []string
	if breaking > 0 {
		parts = append(parts, fmt.Sprintf("%d breaking change(s)", breaking))
	}
	if additions > 0 {
		parts = append(parts, fmt.Sprintf("%d addition(s)", additions))
	}
	if len(parts) == 0 {
		return "No changes detected"
	}
	return strings.Join(parts, ", ")
}

func generateMigrationGuide(changes []SchemaChange) string {
	var guide strings.Builder
	guide.WriteString("## Migration Guide\n\n")

	for _, c := range changes {
		if c.ChangeType == ChangeBreaking {
			guide.WriteString(fmt.Sprintf("### %s\n", c.Path))
			guide.WriteString(fmt.Sprintf("- **Change**: %s\n", c.Description))
			if c.OldValue != "" {
				guide.WriteString(fmt.Sprintf("- **Previous**: %s\n", c.OldValue))
			}
			if c.NewValue != "" {
				guide.WriteString(fmt.Sprintf("- **New**: %s\n", c.NewValue))
			}
			guide.WriteString("\n")
		}
	}

	return guide.String()
}
