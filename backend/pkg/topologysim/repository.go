package topologysim

import "fmt"

// Repository defines the data access interface for topology simulation.
type Repository interface {
	CreateTopology(t *Topology) error
	GetTopology(id string) (*Topology, error)
	ListTopologies(tenantID string) ([]*Topology, error)
	DeleteTopology(id string) error

	StoreResult(r *SimulationResult) error
	GetResult(id string) (*SimulationResult, error)
	ListResults(topologyID string) ([]*SimulationResult, error)
}

// MemoryRepository provides an in-memory implementation.
type MemoryRepository struct {
	topologies map[string]*Topology
	results    map[string]*SimulationResult
}

// NewMemoryRepository creates a new in-memory repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		topologies: make(map[string]*Topology),
		results:    make(map[string]*SimulationResult),
	}
}

func (r *MemoryRepository) CreateTopology(t *Topology) error {
	r.topologies[t.ID] = t
	return nil
}

func (r *MemoryRepository) GetTopology(id string) (*Topology, error) {
	if t, ok := r.topologies[id]; ok {
		return t, nil
	}
	return nil, fmt.Errorf("topology not found: %s", id)
}

func (r *MemoryRepository) ListTopologies(tenantID string) ([]*Topology, error) {
	var result []*Topology
	for _, t := range r.topologies {
		if t.TenantID == tenantID {
			result = append(result, t)
		}
	}
	return result, nil
}

func (r *MemoryRepository) DeleteTopology(id string) error {
	delete(r.topologies, id)
	return nil
}

func (r *MemoryRepository) StoreResult(res *SimulationResult) error {
	r.results[res.ID] = res
	return nil
}

func (r *MemoryRepository) GetResult(id string) (*SimulationResult, error) {
	if res, ok := r.results[id]; ok {
		return res, nil
	}
	return nil, fmt.Errorf("result not found: %s", id)
}

func (r *MemoryRepository) ListResults(topologyID string) ([]*SimulationResult, error) {
	var result []*SimulationResult
	for _, res := range r.results {
		if res.TopologyID == topologyID {
			result = append(result, res)
		}
	}
	return result, nil
}
