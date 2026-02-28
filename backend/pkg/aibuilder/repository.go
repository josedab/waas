package aibuilder

import "context"

// Repository defines persistence operations for AI builder conversations.
type Repository interface {
	CreateConversation(ctx context.Context, conv *Conversation) error
	GetConversation(ctx context.Context, tenantID, id string) (*Conversation, error)
	UpdateConversation(ctx context.Context, conv *Conversation) error
	ListConversations(ctx context.Context, tenantID string, limit, offset int) ([]ConversationSummary, error)
	DeleteConversation(ctx context.Context, tenantID, id string) error

	AddMessage(ctx context.Context, msg *Message) error
	ListMessages(ctx context.Context, conversationID string, limit, offset int) ([]Message, error)
}

// memoryRepository is an in-memory implementation for development.
type memoryRepository struct {
	conversations map[string]*Conversation
	messages      map[string][]Message
}

// NewMemoryRepository creates an in-memory repository.
func NewMemoryRepository() Repository {
	return &memoryRepository{
		conversations: make(map[string]*Conversation),
		messages:      make(map[string][]Message),
	}
}

func (r *memoryRepository) CreateConversation(_ context.Context, conv *Conversation) error {
	r.conversations[conv.ID] = conv
	return nil
}

func (r *memoryRepository) GetConversation(_ context.Context, tenantID, id string) (*Conversation, error) {
	conv, ok := r.conversations[id]
	if !ok || conv.TenantID != tenantID {
		return nil, ErrConversationNotFound
	}
	return conv, nil
}

func (r *memoryRepository) UpdateConversation(_ context.Context, conv *Conversation) error {
	if _, ok := r.conversations[conv.ID]; !ok {
		return ErrConversationNotFound
	}
	r.conversations[conv.ID] = conv
	return nil
}

func (r *memoryRepository) ListConversations(_ context.Context, tenantID string, limit, offset int) ([]ConversationSummary, error) {
	var results []ConversationSummary
	for _, conv := range r.conversations {
		if conv.TenantID != tenantID {
			continue
		}
		results = append(results, ConversationSummary{
			ID:        conv.ID,
			Title:     conv.Title,
			Status:    conv.Status,
			Messages:  len(r.messages[conv.ID]),
			CreatedAt: conv.CreatedAt,
			UpdatedAt: conv.UpdatedAt,
		})
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

func (r *memoryRepository) DeleteConversation(_ context.Context, tenantID, id string) error {
	conv, ok := r.conversations[id]
	if !ok || conv.TenantID != tenantID {
		return ErrConversationNotFound
	}
	delete(r.conversations, id)
	delete(r.messages, id)
	return nil
}

func (r *memoryRepository) AddMessage(_ context.Context, msg *Message) error {
	r.messages[msg.ConversationID] = append(r.messages[msg.ConversationID], *msg)
	return nil
}

func (r *memoryRepository) ListMessages(_ context.Context, conversationID string, limit, offset int) ([]Message, error) {
	msgs := r.messages[conversationID]
	if offset >= len(msgs) {
		return nil, nil
	}
	end := offset + limit
	if end > len(msgs) {
		end = len(msgs)
	}
	return msgs[offset:end], nil
}
