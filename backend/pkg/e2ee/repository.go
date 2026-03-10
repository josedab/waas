package e2ee

import "fmt"

// Repository defines the data access interface for E2EE.
type Repository interface {
	StoreKeyPair(kp *KeyPair) error
	GetActiveKeyPair(endpointID string) (*KeyPair, error)
	GetKeyPairByVersion(endpointID string, version int) (*KeyPair, error)
	ListKeyPairs(endpointID string) ([]*KeyPair, error)
	UpdateKeyPairStatus(id string, status string) error

	AppendAuditEntry(entry *AuditEntry) error
	ListAuditEntries(endpointID string, limit int) ([]*AuditEntry, error)
}

// MemoryRepository provides an in-memory implementation.
type MemoryRepository struct {
	keyPairs map[string]*KeyPair // keyed by ID
	audit    []*AuditEntry
}

// NewMemoryRepository creates a new in-memory repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		keyPairs: make(map[string]*KeyPair),
		audit:    make([]*AuditEntry, 0),
	}
}

func (r *MemoryRepository) StoreKeyPair(kp *KeyPair) error {
	r.keyPairs[kp.ID] = kp
	return nil
}

func (r *MemoryRepository) GetActiveKeyPair(endpointID string) (*KeyPair, error) {
	var best *KeyPair
	for _, kp := range r.keyPairs {
		if kp.EndpointID == endpointID && kp.Status == "active" {
			if best == nil || kp.Version > best.Version {
				best = kp
			}
		}
	}
	if best == nil {
		return nil, fmt.Errorf("no active key pair for endpoint: %s", endpointID)
	}
	return best, nil
}

func (r *MemoryRepository) GetKeyPairByVersion(endpointID string, version int) (*KeyPair, error) {
	for _, kp := range r.keyPairs {
		if kp.EndpointID == endpointID && kp.Version == version {
			return kp, nil
		}
	}
	return nil, fmt.Errorf("key pair not found for endpoint %s version %d", endpointID, version)
}

func (r *MemoryRepository) ListKeyPairs(endpointID string) ([]*KeyPair, error) {
	var result []*KeyPair
	for _, kp := range r.keyPairs {
		if endpointID == "" || kp.EndpointID == endpointID {
			result = append(result, kp)
		}
	}
	return result, nil
}

func (r *MemoryRepository) UpdateKeyPairStatus(id string, status string) error {
	if kp, ok := r.keyPairs[id]; ok {
		kp.Status = status
		return nil
	}
	return fmt.Errorf("key pair not found: %s", id)
}

func (r *MemoryRepository) AppendAuditEntry(entry *AuditEntry) error {
	r.audit = append(r.audit, entry)
	return nil
}

func (r *MemoryRepository) ListAuditEntries(endpointID string, limit int) ([]*AuditEntry, error) {
	var result []*AuditEntry
	for i := len(r.audit) - 1; i >= 0 && len(result) < limit; i-- {
		if r.audit[i].EndpointID == endpointID {
			result = append(result, r.audit[i])
		}
	}
	return result, nil
}
