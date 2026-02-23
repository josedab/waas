package loadtest

import "fmt"

// Repository defines the data access interface for load tests.
type Repository interface {
	CreateTestRun(run *TestRun) error
	GetTestRun(id string) (*TestRun, error)
	ListTestRuns(tenantID string) ([]*TestRun, error)
	UpdateTestRun(run *TestRun) error
	DeleteTestRun(id string) error
}

// MemoryRepository provides an in-memory implementation.
type MemoryRepository struct {
	runs map[string]*TestRun
}

// NewMemoryRepository creates a new in-memory repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{runs: make(map[string]*TestRun)}
}

func (r *MemoryRepository) CreateTestRun(run *TestRun) error {
	r.runs[run.ID] = run
	return nil
}

func (r *MemoryRepository) GetTestRun(id string) (*TestRun, error) {
	if run, ok := r.runs[id]; ok {
		return run, nil
	}
	return nil, fmt.Errorf("test run not found: %s", id)
}

func (r *MemoryRepository) ListTestRuns(tenantID string) ([]*TestRun, error) {
	var result []*TestRun
	for _, run := range r.runs {
		if run.TenantID == tenantID {
			result = append(result, run)
		}
	}
	return result, nil
}

func (r *MemoryRepository) UpdateTestRun(run *TestRun) error {
	r.runs[run.ID] = run
	return nil
}

func (r *MemoryRepository) DeleteTestRun(id string) error {
	delete(r.runs, id)
	return nil
}
