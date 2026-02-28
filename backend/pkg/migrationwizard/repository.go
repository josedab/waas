package migrationwizard

import "context"

// Repository defines persistence operations for migrations.
type Repository interface {
	Create(ctx context.Context, m *Migration) error
	Get(ctx context.Context, tenantID, id string) (*Migration, error)
	Update(ctx context.Context, m *Migration) error
	List(ctx context.Context, tenantID string) ([]MigrationSummary, error)
	Delete(ctx context.Context, tenantID, id string) error
}

type memoryRepository struct {
	migrations map[string]*Migration
}

// NewMemoryRepository creates an in-memory migration repository.
func NewMemoryRepository() Repository {
	return &memoryRepository{migrations: make(map[string]*Migration)}
}

func (r *memoryRepository) Create(_ context.Context, m *Migration) error {
	r.migrations[m.ID] = m
	return nil
}

func (r *memoryRepository) Get(_ context.Context, tenantID, id string) (*Migration, error) {
	m, ok := r.migrations[id]
	if !ok || m.TenantID != tenantID {
		return nil, ErrMigrationNotFound
	}
	return m, nil
}

func (r *memoryRepository) Update(_ context.Context, m *Migration) error {
	if _, ok := r.migrations[m.ID]; !ok {
		return ErrMigrationNotFound
	}
	r.migrations[m.ID] = m
	return nil
}

func (r *memoryRepository) List(_ context.Context, tenantID string) ([]MigrationSummary, error) {
	var results []MigrationSummary
	for _, m := range r.migrations {
		if m.TenantID != tenantID {
			continue
		}
		pct := 0.0
		if m.Progress != nil {
			pct = m.Progress.PercentComplete
		}
		results = append(results, MigrationSummary{
			ID: m.ID, Source: m.Source, Status: m.Status, Progress: pct, CreatedAt: m.CreatedAt,
		})
	}
	return results, nil
}

func (r *memoryRepository) Delete(_ context.Context, tenantID, id string) error {
	m, ok := r.migrations[id]
	if !ok || m.TenantID != tenantID {
		return ErrMigrationNotFound
	}
	delete(r.migrations, id)
	return nil
}
