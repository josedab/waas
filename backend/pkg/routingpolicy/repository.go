package routingpolicy

import "fmt"

// Repository defines the data access interface for routing policies.
type Repository interface {
	CreatePolicy(p *Policy) error
	GetPolicy(id string) (*Policy, error)
	ListPolicies(tenantID string) ([]*Policy, error)
	UpdatePolicy(p *Policy) error
	DeletePolicy(id string) error

	StoreVersion(v *PolicyVersion) error
	ListVersions(policyID string) ([]*PolicyVersion, error)

	AppendAudit(entry *AuditEntry) error
	ListAudit(policyID string, limit int) ([]*AuditEntry, error)
}

// MemoryRepository provides an in-memory implementation.
type MemoryRepository struct {
	policies map[string]*Policy
	versions map[string][]*PolicyVersion
	audit    []*AuditEntry
}

// NewMemoryRepository creates a new in-memory repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		policies: make(map[string]*Policy),
		versions: make(map[string][]*PolicyVersion),
		audit:    make([]*AuditEntry, 0),
	}
}

func (r *MemoryRepository) CreatePolicy(p *Policy) error {
	r.policies[p.ID] = p
	return nil
}

func (r *MemoryRepository) GetPolicy(id string) (*Policy, error) {
	if p, ok := r.policies[id]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("policy not found: %s", id)
}

func (r *MemoryRepository) ListPolicies(tenantID string) ([]*Policy, error) {
	var result []*Policy
	for _, p := range r.policies {
		if p.TenantID == tenantID {
			result = append(result, p)
		}
	}
	return result, nil
}

func (r *MemoryRepository) UpdatePolicy(p *Policy) error {
	r.policies[p.ID] = p
	return nil
}

func (r *MemoryRepository) DeletePolicy(id string) error {
	delete(r.policies, id)
	return nil
}

func (r *MemoryRepository) StoreVersion(v *PolicyVersion) error {
	r.versions[v.PolicyID] = append(r.versions[v.PolicyID], v)
	return nil
}

func (r *MemoryRepository) ListVersions(policyID string) ([]*PolicyVersion, error) {
	return r.versions[policyID], nil
}

func (r *MemoryRepository) AppendAudit(entry *AuditEntry) error {
	r.audit = append(r.audit, entry)
	return nil
}

func (r *MemoryRepository) ListAudit(policyID string, limit int) ([]*AuditEntry, error) {
	var result []*AuditEntry
	for i := len(r.audit) - 1; i >= 0 && len(result) < limit; i-- {
		if r.audit[i].PolicyID == policyID {
			result = append(result, r.audit[i])
		}
	}
	return result, nil
}
