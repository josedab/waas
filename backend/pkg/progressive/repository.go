package progressive

import "context"

// Repository defines the data access interface for progressive rollouts.
type Repository interface {
	Create(ctx context.Context, rollout *Rollout) error
	Get(ctx context.Context, tenantID, id string) (*Rollout, error)
	List(ctx context.Context, tenantID string) ([]Rollout, error)
	Update(ctx context.Context, rollout *Rollout) error
	Delete(ctx context.Context, tenantID, id string) error
}

type memoryRepository struct {
	rollouts map[string]*Rollout
}

// NewMemoryRepository creates an in-memory progressive delivery repository.
func NewMemoryRepository() Repository {
	return &memoryRepository{
		rollouts: make(map[string]*Rollout),
	}
}

func (r *memoryRepository) Create(_ context.Context, rollout *Rollout) error {
	r.rollouts[rollout.ID] = rollout
	return nil
}

func (r *memoryRepository) Get(_ context.Context, tenantID, id string) (*Rollout, error) {
	rollout, ok := r.rollouts[id]
	if !ok || rollout.TenantID != tenantID {
		return nil, ErrRolloutNotFound
	}
	return rollout, nil
}

func (r *memoryRepository) List(_ context.Context, tenantID string) ([]Rollout, error) {
	var results []Rollout
	for _, rollout := range r.rollouts {
		if rollout.TenantID == tenantID {
			results = append(results, *rollout)
		}
	}
	return results, nil
}

func (r *memoryRepository) Update(_ context.Context, rollout *Rollout) error {
	if _, ok := r.rollouts[rollout.ID]; !ok {
		return ErrRolloutNotFound
	}
	r.rollouts[rollout.ID] = rollout
	return nil
}

func (r *memoryRepository) Delete(_ context.Context, tenantID, id string) error {
	rollout, ok := r.rollouts[id]
	if !ok || rollout.TenantID != tenantID {
		return ErrRolloutNotFound
	}
	delete(r.rollouts, id)
	return nil
}
