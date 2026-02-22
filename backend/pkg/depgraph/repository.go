package depgraph

import "fmt"

// Repository defines the data access interface for the dependency graph.
type Repository interface {
	UpsertDependency(dep *Dependency) error
	GetDependency(id string) (*Dependency, error)
	ListDependencies(tenantID string) ([]*Dependency, error)
	GetDependenciesForEndpoint(endpointID string) ([]*Dependency, error)
	GetConsumers(producerID string) ([]*Dependency, error)
	GetProducers(consumerID string) ([]*Dependency, error)
	DeleteDependency(id string) error

	GetEndpointNode(id string) (*EndpointNode, error)
	ListEndpointNodes(tenantID string) ([]*EndpointNode, error)
	UpsertEndpointNode(node *EndpointNode) error
}

// MemoryRepository provides an in-memory implementation.
type MemoryRepository struct {
	dependencies map[string]*Dependency
	nodes        map[string]*EndpointNode
}

// NewMemoryRepository creates a new in-memory repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		dependencies: make(map[string]*Dependency),
		nodes:        make(map[string]*EndpointNode),
	}
}

func (r *MemoryRepository) UpsertDependency(dep *Dependency) error {
	r.dependencies[dep.ID] = dep
	return nil
}

func (r *MemoryRepository) GetDependency(id string) (*Dependency, error) {
	if d, ok := r.dependencies[id]; ok {
		return d, nil
	}
	return nil, fmt.Errorf("dependency not found: %s", id)
}

func (r *MemoryRepository) ListDependencies(tenantID string) ([]*Dependency, error) {
	var result []*Dependency
	for _, d := range r.dependencies {
		if d.TenantID == tenantID {
			result = append(result, d)
		}
	}
	return result, nil
}

func (r *MemoryRepository) GetDependenciesForEndpoint(endpointID string) ([]*Dependency, error) {
	var result []*Dependency
	for _, d := range r.dependencies {
		if d.ProducerID == endpointID || d.ConsumerID == endpointID {
			result = append(result, d)
		}
	}
	return result, nil
}

func (r *MemoryRepository) GetConsumers(producerID string) ([]*Dependency, error) {
	var result []*Dependency
	for _, d := range r.dependencies {
		if d.ProducerID == producerID {
			result = append(result, d)
		}
	}
	return result, nil
}

func (r *MemoryRepository) GetProducers(consumerID string) ([]*Dependency, error) {
	var result []*Dependency
	for _, d := range r.dependencies {
		if d.ConsumerID == consumerID {
			result = append(result, d)
		}
	}
	return result, nil
}

func (r *MemoryRepository) DeleteDependency(id string) error {
	delete(r.dependencies, id)
	return nil
}

func (r *MemoryRepository) GetEndpointNode(id string) (*EndpointNode, error) {
	if n, ok := r.nodes[id]; ok {
		return n, nil
	}
	return nil, fmt.Errorf("node not found: %s", id)
}

func (r *MemoryRepository) ListEndpointNodes(tenantID string) ([]*EndpointNode, error) {
	var result []*EndpointNode
	for _, n := range r.nodes {
		if n.TenantID == tenantID {
			result = append(result, n)
		}
	}
	return result, nil
}

func (r *MemoryRepository) UpsertEndpointNode(node *EndpointNode) error {
	r.nodes[node.ID] = node
	return nil
}
