package selfhealing

import "fmt"

// Repository defines the data access interface for self-healing.
type Repository interface {
	CreateDiscovery(d *EndpointDiscovery) error
	GetDiscovery(id string) (*EndpointDiscovery, error)
	ListDiscoveries(endpointID string) ([]*EndpointDiscovery, error)
	UpdateDiscovery(d *EndpointDiscovery) error

	GetFailureTracker(endpointID string) (*FailureTracker, error)
	UpsertFailureTracker(ft *FailureTracker) error
	ResetFailureTracker(endpointID string) error

	AppendMigrationEvent(evt *MigrationEvent) error
	ListMigrationEvents(tenantID string, limit int) ([]*MigrationEvent, error)
}

// MemoryRepository provides an in-memory implementation.
type MemoryRepository struct {
	discoveries map[string]*EndpointDiscovery
	trackers    map[string]*FailureTracker
	events      []*MigrationEvent
}

// NewMemoryRepository creates a new in-memory repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		discoveries: make(map[string]*EndpointDiscovery),
		trackers:    make(map[string]*FailureTracker),
		events:      make([]*MigrationEvent, 0),
	}
}

func (r *MemoryRepository) CreateDiscovery(d *EndpointDiscovery) error {
	r.discoveries[d.ID] = d
	return nil
}

func (r *MemoryRepository) GetDiscovery(id string) (*EndpointDiscovery, error) {
	if d, ok := r.discoveries[id]; ok {
		return d, nil
	}
	return nil, fmt.Errorf("discovery not found: %s", id)
}

func (r *MemoryRepository) ListDiscoveries(endpointID string) ([]*EndpointDiscovery, error) {
	var result []*EndpointDiscovery
	for _, d := range r.discoveries {
		if d.EndpointID == endpointID {
			result = append(result, d)
		}
	}
	return result, nil
}

func (r *MemoryRepository) UpdateDiscovery(d *EndpointDiscovery) error {
	r.discoveries[d.ID] = d
	return nil
}

func (r *MemoryRepository) GetFailureTracker(endpointID string) (*FailureTracker, error) {
	if ft, ok := r.trackers[endpointID]; ok {
		return ft, nil
	}
	return &FailureTracker{EndpointID: endpointID}, nil
}

func (r *MemoryRepository) UpsertFailureTracker(ft *FailureTracker) error {
	r.trackers[ft.EndpointID] = ft
	return nil
}

func (r *MemoryRepository) ResetFailureTracker(endpointID string) error {
	delete(r.trackers, endpointID)
	return nil
}

func (r *MemoryRepository) AppendMigrationEvent(evt *MigrationEvent) error {
	r.events = append(r.events, evt)
	return nil
}

func (r *MemoryRepository) ListMigrationEvents(tenantID string, limit int) ([]*MigrationEvent, error) {
	var result []*MigrationEvent
	for i := len(r.events) - 1; i >= 0 && len(result) < limit; i-- {
		if r.events[i].TenantID == tenantID {
			result = append(result, r.events[i])
		}
	}
	return result, nil
}
