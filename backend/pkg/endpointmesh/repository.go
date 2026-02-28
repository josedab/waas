package endpointmesh

import "fmt"

// Repository defines the data access interface for the endpoint mesh.
type Repository interface {
	CreateNode(node *MeshNode) error
	GetNode(id string) (*MeshNode, error)
	ListNodes(tenantID string) ([]*MeshNode, error)
	UpdateNode(node *MeshNode) error
	DeleteNode(id string) error

	AppendHealthCheck(hc *HealthCheck) error
	ListHealthChecks(nodeID string, limit int) ([]*HealthCheck, error)

	AppendRerouteEvent(evt *RerouteEvent) error
	ListRerouteEvents(tenantID string, limit int) ([]*RerouteEvent, error)
}

// MemoryRepository provides an in-memory implementation.
type MemoryRepository struct {
	nodes         map[string]*MeshNode
	healthChecks  []*HealthCheck
	rerouteEvents []*RerouteEvent
}

// NewMemoryRepository creates a new in-memory repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		nodes:         make(map[string]*MeshNode),
		healthChecks:  make([]*HealthCheck, 0),
		rerouteEvents: make([]*RerouteEvent, 0),
	}
}

func (r *MemoryRepository) CreateNode(node *MeshNode) error {
	r.nodes[node.ID] = node
	return nil
}

func (r *MemoryRepository) GetNode(id string) (*MeshNode, error) {
	if n, ok := r.nodes[id]; ok {
		return n, nil
	}
	return nil, fmt.Errorf("node not found: %s", id)
}

func (r *MemoryRepository) ListNodes(tenantID string) ([]*MeshNode, error) {
	var result []*MeshNode
	for _, n := range r.nodes {
		if n.TenantID == tenantID {
			result = append(result, n)
		}
	}
	return result, nil
}

func (r *MemoryRepository) UpdateNode(node *MeshNode) error {
	if _, ok := r.nodes[node.ID]; !ok {
		return fmt.Errorf("node not found: %s", node.ID)
	}
	r.nodes[node.ID] = node
	return nil
}

func (r *MemoryRepository) DeleteNode(id string) error {
	if _, ok := r.nodes[id]; !ok {
		return fmt.Errorf("node not found: %s", id)
	}
	delete(r.nodes, id)
	return nil
}

func (r *MemoryRepository) AppendHealthCheck(hc *HealthCheck) error {
	r.healthChecks = append(r.healthChecks, hc)
	return nil
}

func (r *MemoryRepository) ListHealthChecks(nodeID string, limit int) ([]*HealthCheck, error) {
	var result []*HealthCheck
	for i := len(r.healthChecks) - 1; i >= 0 && len(result) < limit; i-- {
		if r.healthChecks[i].NodeID == nodeID {
			result = append(result, r.healthChecks[i])
		}
	}
	return result, nil
}

func (r *MemoryRepository) AppendRerouteEvent(evt *RerouteEvent) error {
	r.rerouteEvents = append(r.rerouteEvents, evt)
	return nil
}

func (r *MemoryRepository) ListRerouteEvents(tenantID string, limit int) ([]*RerouteEvent, error) {
	var result []*RerouteEvent
	for i := len(r.rerouteEvents) - 1; i >= 0 && len(result) < limit; i-- {
		if r.rerouteEvents[i].TenantID == tenantID {
			result = append(result, r.rerouteEvents[i])
		}
	}
	return result, nil
}
