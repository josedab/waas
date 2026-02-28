package edgenetwork

import "context"

// Repository defines data access operations for the edge network.
type Repository interface {
	CreateNode(ctx context.Context, node *EdgeNode) error
	GetNode(ctx context.Context, id string) (*EdgeNode, error)
	ListNodes(ctx context.Context, region *Region) ([]EdgeNode, error)
	UpdateNode(ctx context.Context, node *EdgeNode) error
	DeleteNode(ctx context.Context, id string) error

	CreateRoutingRule(ctx context.Context, rule *RoutingRule) error
	GetRoutingRule(ctx context.Context, tenantID, id string) (*RoutingRule, error)
	ListRoutingRules(ctx context.Context, tenantID string) ([]RoutingRule, error)
	DeleteRoutingRule(ctx context.Context, tenantID, id string) error
}

type memoryRepository struct {
	nodes map[string]*EdgeNode
	rules map[string]*RoutingRule
}

// NewMemoryRepository creates an in-memory edge network repository.
func NewMemoryRepository() Repository {
	return &memoryRepository{
		nodes: make(map[string]*EdgeNode),
		rules: make(map[string]*RoutingRule),
	}
}

func (r *memoryRepository) CreateNode(_ context.Context, node *EdgeNode) error {
	r.nodes[node.ID] = node
	return nil
}

func (r *memoryRepository) GetNode(_ context.Context, id string) (*EdgeNode, error) {
	n, ok := r.nodes[id]
	if !ok {
		return nil, ErrNodeNotFound
	}
	return n, nil
}

func (r *memoryRepository) ListNodes(_ context.Context, region *Region) ([]EdgeNode, error) {
	var result []EdgeNode
	for _, n := range r.nodes {
		if region != nil && n.Region != *region {
			continue
		}
		result = append(result, *n)
	}
	return result, nil
}

func (r *memoryRepository) UpdateNode(_ context.Context, node *EdgeNode) error {
	if _, ok := r.nodes[node.ID]; !ok {
		return ErrNodeNotFound
	}
	r.nodes[node.ID] = node
	return nil
}

func (r *memoryRepository) DeleteNode(_ context.Context, id string) error {
	if _, ok := r.nodes[id]; !ok {
		return ErrNodeNotFound
	}
	delete(r.nodes, id)
	return nil
}

func (r *memoryRepository) CreateRoutingRule(_ context.Context, rule *RoutingRule) error {
	r.rules[rule.ID] = rule
	return nil
}

func (r *memoryRepository) GetRoutingRule(_ context.Context, tenantID, id string) (*RoutingRule, error) {
	rule, ok := r.rules[id]
	if !ok || rule.TenantID != tenantID {
		return nil, ErrRuleNotFound
	}
	return rule, nil
}

func (r *memoryRepository) ListRoutingRules(_ context.Context, tenantID string) ([]RoutingRule, error) {
	var result []RoutingRule
	for _, rule := range r.rules {
		if rule.TenantID == tenantID {
			result = append(result, *rule)
		}
	}
	return result, nil
}

func (r *memoryRepository) DeleteRoutingRule(_ context.Context, tenantID, id string) error {
	rule, ok := r.rules[id]
	if !ok || rule.TenantID != tenantID {
		return ErrRuleNotFound
	}
	delete(r.rules, id)
	return nil
}
