package schemachangelog

import "fmt"

// Repository defines the data access interface for schema changelog.
type Repository interface {
	CreateSchemaVersion(sv *SchemaVersion) error
	GetSchemaVersion(eventType, version string) (*SchemaVersion, error)
	GetLatestSchemaVersion(eventType string) (*SchemaVersion, error)
	ListSchemaVersions(eventType string) ([]*SchemaVersion, error)

	CreateChangelog(entry *ChangelogEntry) error
	GetChangelog(id string) (*ChangelogEntry, error)
	ListChangelogs(eventType string) ([]*ChangelogEntry, error)

	CreateMigration(m *ConsumerMigration) error
	GetMigration(id string) (*ConsumerMigration, error)
	ListMigrations(changelogID string) ([]*ConsumerMigration, error)
	UpdateMigration(m *ConsumerMigration) error
}

// MemoryRepository provides an in-memory implementation.
type MemoryRepository struct {
	schemas    map[string]*SchemaVersion // keyed by eventType:version
	changelogs map[string]*ChangelogEntry
	migrations map[string]*ConsumerMigration
}

// NewMemoryRepository creates a new in-memory repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		schemas:    make(map[string]*SchemaVersion),
		changelogs: make(map[string]*ChangelogEntry),
		migrations: make(map[string]*ConsumerMigration),
	}
}

func (r *MemoryRepository) CreateSchemaVersion(sv *SchemaVersion) error {
	key := sv.EventType + ":" + sv.Version
	r.schemas[key] = sv
	return nil
}

func (r *MemoryRepository) GetSchemaVersion(eventType, version string) (*SchemaVersion, error) {
	key := eventType + ":" + version
	if sv, ok := r.schemas[key]; ok {
		return sv, nil
	}
	return nil, fmt.Errorf("schema version not found: %s:%s", eventType, version)
}

func (r *MemoryRepository) GetLatestSchemaVersion(eventType string) (*SchemaVersion, error) {
	var latest *SchemaVersion
	for _, sv := range r.schemas {
		if sv.EventType == eventType {
			if latest == nil || sv.CreatedAt.After(latest.CreatedAt) {
				latest = sv
			}
		}
	}
	if latest == nil {
		return nil, fmt.Errorf("no schema versions for event type: %s", eventType)
	}
	return latest, nil
}

func (r *MemoryRepository) ListSchemaVersions(eventType string) ([]*SchemaVersion, error) {
	var result []*SchemaVersion
	for _, sv := range r.schemas {
		if sv.EventType == eventType {
			result = append(result, sv)
		}
	}
	return result, nil
}

func (r *MemoryRepository) CreateChangelog(entry *ChangelogEntry) error {
	r.changelogs[entry.ID] = entry
	return nil
}

func (r *MemoryRepository) GetChangelog(id string) (*ChangelogEntry, error) {
	if e, ok := r.changelogs[id]; ok {
		return e, nil
	}
	return nil, fmt.Errorf("changelog not found: %s", id)
}

func (r *MemoryRepository) ListChangelogs(eventType string) ([]*ChangelogEntry, error) {
	var result []*ChangelogEntry
	for _, e := range r.changelogs {
		if e.EventType == eventType {
			result = append(result, e)
		}
	}
	return result, nil
}

func (r *MemoryRepository) CreateMigration(m *ConsumerMigration) error {
	r.migrations[m.ID] = m
	return nil
}

func (r *MemoryRepository) GetMigration(id string) (*ConsumerMigration, error) {
	if m, ok := r.migrations[id]; ok {
		return m, nil
	}
	return nil, fmt.Errorf("migration not found: %s", id)
}

func (r *MemoryRepository) ListMigrations(changelogID string) ([]*ConsumerMigration, error) {
	var result []*ConsumerMigration
	for _, m := range r.migrations {
		if m.ChangelogID == changelogID {
			result = append(result, m)
		}
	}
	return result, nil
}

func (r *MemoryRepository) UpdateMigration(m *ConsumerMigration) error {
	r.migrations[m.ID] = m
	return nil
}
