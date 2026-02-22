package nlbuilder

import "fmt"

// Repository defines the data access interface for NL builder conversations.
type Repository interface {
	CreateConversation(conv *Conversation) error
	GetConversation(id string) (*Conversation, error)
	UpdateConversation(conv *Conversation) error
	ListConversations(tenantID string) ([]*Conversation, error)
	DeleteConversation(id string) error
}

// MemoryRepository provides an in-memory implementation.
type MemoryRepository struct {
	conversations map[string]*Conversation
}

// NewMemoryRepository creates a new in-memory repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		conversations: make(map[string]*Conversation),
	}
}

func (r *MemoryRepository) CreateConversation(conv *Conversation) error {
	r.conversations[conv.ID] = conv
	return nil
}

func (r *MemoryRepository) GetConversation(id string) (*Conversation, error) {
	if c, ok := r.conversations[id]; ok {
		return c, nil
	}
	return nil, fmt.Errorf("conversation not found: %s", id)
}

func (r *MemoryRepository) UpdateConversation(conv *Conversation) error {
	if _, ok := r.conversations[conv.ID]; !ok {
		return fmt.Errorf("conversation not found: %s", conv.ID)
	}
	r.conversations[conv.ID] = conv
	return nil
}

func (r *MemoryRepository) ListConversations(tenantID string) ([]*Conversation, error) {
	var result []*Conversation
	for _, c := range r.conversations {
		if c.TenantID == tenantID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (r *MemoryRepository) DeleteConversation(id string) error {
	delete(r.conversations, id)
	return nil
}
