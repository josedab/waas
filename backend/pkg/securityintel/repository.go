package securityintel

import "context"

// Repository defines persistence operations for security intelligence data.
type Repository interface {
	CreateEvent(ctx context.Context, event *SecurityEvent) error
	ListEvents(ctx context.Context, tenantID string, limit, offset int) ([]SecurityEvent, error)
	GetEvent(ctx context.Context, tenantID, id string) (*SecurityEvent, error)
	ResolveEvent(ctx context.Context, tenantID, id string) error

	CreatePolicy(ctx context.Context, policy *SecurityPolicy) error
	GetPolicy(ctx context.Context, tenantID, id string) (*SecurityPolicy, error)
	ListPolicies(ctx context.Context, tenantID string) ([]SecurityPolicy, error)
	UpdatePolicy(ctx context.Context, policy *SecurityPolicy) error
	DeletePolicy(ctx context.Context, tenantID, id string) error
}

type memoryRepository struct {
	events   map[string]*SecurityEvent
	policies map[string]*SecurityPolicy
}

// NewMemoryRepository creates an in-memory security intelligence repository.
func NewMemoryRepository() Repository {
	return &memoryRepository{
		events:   make(map[string]*SecurityEvent),
		policies: make(map[string]*SecurityPolicy),
	}
}

func (r *memoryRepository) CreateEvent(_ context.Context, event *SecurityEvent) error {
	r.events[event.ID] = event
	return nil
}

func (r *memoryRepository) ListEvents(_ context.Context, tenantID string, limit, offset int) ([]SecurityEvent, error) {
	var results []SecurityEvent
	for _, e := range r.events {
		if e.TenantID == tenantID {
			results = append(results, *e)
		}
	}
	if offset >= len(results) {
		return nil, nil
	}
	end := offset + limit
	if end > len(results) {
		end = len(results)
	}
	return results[offset:end], nil
}

func (r *memoryRepository) GetEvent(_ context.Context, tenantID, id string) (*SecurityEvent, error) {
	e, ok := r.events[id]
	if !ok || e.TenantID != tenantID {
		return nil, ErrEventNotFound
	}
	return e, nil
}

func (r *memoryRepository) ResolveEvent(_ context.Context, tenantID, id string) error {
	e, ok := r.events[id]
	if !ok || e.TenantID != tenantID {
		return ErrEventNotFound
	}
	e.Resolved = true
	return nil
}

func (r *memoryRepository) CreatePolicy(_ context.Context, policy *SecurityPolicy) error {
	r.policies[policy.ID] = policy
	return nil
}

func (r *memoryRepository) GetPolicy(_ context.Context, tenantID, id string) (*SecurityPolicy, error) {
	p, ok := r.policies[id]
	if !ok || p.TenantID != tenantID {
		return nil, ErrPolicyNotFound
	}
	return p, nil
}

func (r *memoryRepository) ListPolicies(_ context.Context, tenantID string) ([]SecurityPolicy, error) {
	var results []SecurityPolicy
	for _, p := range r.policies {
		if p.TenantID == tenantID {
			results = append(results, *p)
		}
	}
	return results, nil
}

func (r *memoryRepository) UpdatePolicy(_ context.Context, policy *SecurityPolicy) error {
	if _, ok := r.policies[policy.ID]; !ok {
		return ErrPolicyNotFound
	}
	r.policies[policy.ID] = policy
	return nil
}

func (r *memoryRepository) DeletePolicy(_ context.Context, tenantID, id string) error {
	p, ok := r.policies[id]
	if !ok || p.TenantID != tenantID {
		return ErrPolicyNotFound
	}
	delete(r.policies, id)
	return nil
}
