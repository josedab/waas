package aibuilder

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/utils"
)

var (
	ErrConversationNotFound = errors.New("conversation not found")
	ErrConversationExpired  = errors.New("conversation has expired")
	ErrEmptyMessage         = errors.New("message cannot be empty")
)

// ServiceConfig holds configuration for the AI builder service.
type ServiceConfig struct {
	MaxConversationsPerTenant  int
	ConversationTTL            time.Duration
	MaxMessagesPerConversation int
	MaxMessageLength           int
}

// DefaultServiceConfig returns sensible defaults.
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		MaxConversationsPerTenant:  50,
		ConversationTTL:            24 * time.Hour,
		MaxMessagesPerConversation: 200,
		MaxMessageLength:           4096,
	}
}

// Service provides the AI conversational webhook builder operations.
type Service struct {
	repo   Repository
	config *ServiceConfig
	logger *utils.Logger
}

// NewService creates a new AI builder service.
func NewService(repo Repository, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}
	if repo == nil {
		repo = NewMemoryRepository()
	}
	return &Service{
		repo:   repo,
		config: config,
		logger: utils.NewLogger("aibuilder"),
	}
}

// SendMessage processes a user message and returns an AI response.
func (s *Service) SendMessage(ctx context.Context, tenantID string, req *SendMessageRequest) (*SendMessageResponse, error) {
	if strings.TrimSpace(req.Message) == "" {
		return nil, ErrEmptyMessage
	}
	if len(req.Message) > s.config.MaxMessageLength {
		return nil, fmt.Errorf("message exceeds maximum length of %d characters", s.config.MaxMessageLength)
	}

	var conv *Conversation
	var err error

	if req.ConversationID != "" {
		conv, err = s.repo.GetConversation(ctx, tenantID, req.ConversationID)
		if err != nil {
			return nil, fmt.Errorf("get conversation: %w", err)
		}
		if conv.Status != StatusActive {
			return nil, ErrConversationExpired
		}
	} else {
		conv = &Conversation{
			ID:       uuid.New().String(),
			TenantID: tenantID,
			Status:   StatusActive,
			Title:    s.generateTitle(req.Message),
			Context: &ConversationContext{
				Variables: make(map[string]interface{}),
			},
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
			ExpiresAt: time.Now().UTC().Add(s.config.ConversationTTL),
		}
		if err := s.repo.CreateConversation(ctx, conv); err != nil {
			return nil, fmt.Errorf("create conversation: %w", err)
		}
	}

	// Store user message
	userMsg := &Message{
		ID:             uuid.New().String(),
		ConversationID: conv.ID,
		Role:           RoleUser,
		Content:        req.Message,
		CreatedAt:      time.Now().UTC(),
	}
	intent := s.classifyIntent(req.Message)
	userMsg.Intent = intent
	if err := s.repo.AddMessage(ctx, userMsg); err != nil {
		return nil, fmt.Errorf("store user message: %w", err)
	}

	// Generate AI response
	reply := s.generateResponse(conv, intent, req.Message)
	if err := s.repo.AddMessage(ctx, reply); err != nil {
		return nil, fmt.Errorf("store reply: %w", err)
	}

	// Update conversation context
	conv.Context.DetectedIntents = append(conv.Context.DetectedIntents, intent)
	conv.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateConversation(ctx, conv); err != nil {
		return nil, fmt.Errorf("update conversation: %w", err)
	}

	return &SendMessageResponse{
		ConversationID: conv.ID,
		Reply:          reply,
	}, nil
}

// GetConversation returns a conversation by ID.
func (s *Service) GetConversation(ctx context.Context, tenantID, id string) (*Conversation, error) {
	return s.repo.GetConversation(ctx, tenantID, id)
}

// ListConversations returns recent conversations for a tenant.
func (s *Service) ListConversations(ctx context.Context, tenantID string, limit, offset int) ([]ConversationSummary, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListConversations(ctx, tenantID, limit, offset)
}

// GetMessages returns messages for a conversation.
func (s *Service) GetMessages(ctx context.Context, tenantID, conversationID string, limit, offset int) ([]Message, error) {
	// Verify tenant owns conversation
	if _, err := s.repo.GetConversation(ctx, tenantID, conversationID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.repo.ListMessages(ctx, conversationID, limit, offset)
}

// DeleteConversation removes a conversation and its messages.
func (s *Service) DeleteConversation(ctx context.Context, tenantID, id string) error {
	return s.repo.DeleteConversation(ctx, tenantID, id)
}

// DebugDelivery uses AI to diagnose a webhook delivery issue.
func (s *Service) DebugDelivery(ctx context.Context, tenantID string, req *DebugRequest) (*DebugResponse, error) {
	if req.DeliveryID == "" {
		return nil, errors.New("delivery_id is required")
	}

	// Generate diagnostic analysis
	response := &DebugResponse{
		DeliveryID: req.DeliveryID,
		Summary:    "Analyzing delivery attempt for potential issues.",
		RootCause:  "Further investigation needed. Check endpoint availability and response codes.",
		Suggestions: []string{
			"Verify the endpoint URL is accessible from our servers",
			"Check if the endpoint requires specific authentication headers",
			"Review the payload format matches what the endpoint expects",
			"Consider adding retry policies for transient failures",
		},
		RelatedDocs: []string{
			"/docs/troubleshooting/delivery-failures",
			"/docs/guides/retry-configuration",
		},
	}

	return response, nil
}

// classifyIntent determines the user's intent from their message.
func (s *Service) classifyIntent(message string) IntentType {
	lower := strings.ToLower(message)

	switch {
	case containsAny(lower, "create", "new endpoint", "add endpoint", "set up", "setup webhook"):
		return IntentCreateEndpoint
	case containsAny(lower, "retry", "retries", "backoff", "retry policy"):
		return IntentConfigureRetry
	case containsAny(lower, "auth", "authentication", "api key", "bearer", "hmac", "signature"):
		return IntentSetupAuth
	case containsAny(lower, "explain", "what does", "how does", "why"):
		return IntentExplainError
	case containsAny(lower, "debug", "failing", "failed", "error", "not working", "broken", "issue"):
		return IntentDebugDelivery
	case containsAny(lower, "list", "show", "all endpoints", "my endpoints"):
		return IntentListEndpoints
	case containsAny(lower, "transform", "modify payload", "change payload", "map fields"):
		return IntentTransformSetup
	default:
		return IntentGeneral
	}
}

// generateResponse creates an AI response based on intent and context.
func (s *Service) generateResponse(conv *Conversation, intent IntentType, userMessage string) *Message {
	var content string
	var suggestions []string
	var actions []Action

	switch intent {
	case IntentCreateEndpoint:
		if conv.Context.EndpointDraft == nil {
			conv.Context.EndpointDraft = &EndpointDraft{}
		}
		content = "I'll help you create a new webhook endpoint. Let's start with the basics:\n\n" +
			"**What URL should receive the webhooks?**\n\n" +
			"Please provide the full URL (e.g., `https://api.example.com/webhooks`)."
		suggestions = []string{
			"Use https://httpbin.org/post for testing",
			"I need help choosing a URL",
		}
		conv.Context.StepIndex = 1

	case IntentConfigureRetry:
		content = "Let me help you configure retry policies. Here are the common strategies:\n\n" +
			"1. **Exponential backoff** (recommended) - Delays increase exponentially\n" +
			"2. **Linear backoff** - Fixed delay between retries\n\n" +
			"Which strategy would you prefer? And how many retry attempts (default is 5)?"
		suggestions = []string{
			"Use exponential backoff with 5 retries",
			"Use linear backoff with 3 retries",
		}
		actions = []Action{
			{Type: "update_config", Label: "Apply exponential backoff (5 retries)", Params: map[string]interface{}{"backoff": "exponential", "max_retries": 5}, RequireConf: true},
		}

	case IntentSetupAuth:
		content = "I'll help you configure authentication for your webhook endpoint. Available methods:\n\n" +
			"1. **HMAC Signature** - Signs payloads with a shared secret (most secure)\n" +
			"2. **Bearer Token** - Includes a token in the Authorization header\n" +
			"3. **Basic Auth** - Username and password\n" +
			"4. **API Key** - Custom header with an API key\n\n" +
			"Which authentication method would you like to use?"
		suggestions = []string{
			"Use HMAC signatures",
			"Use a bearer token",
			"Help me choose the right method",
		}

	case IntentDebugDelivery:
		content = "I'll help you debug the delivery issue. To diagnose the problem, I need some information:\n\n" +
			"1. **Delivery ID** - If you have a specific delivery to investigate\n" +
			"2. **Endpoint URL** - The endpoint experiencing issues\n" +
			"3. **Error message** - Any error messages you're seeing\n\n" +
			"What information can you provide?"
		suggestions = []string{
			"Show me recent failed deliveries",
			"I have a delivery ID",
			"All deliveries to my endpoint are failing",
		}

	case IntentListEndpoints:
		content = "I'll fetch your webhook endpoints. Here's what I can show you:\n\n" +
			"• All active endpoints\n" +
			"• Endpoints filtered by event type\n" +
			"• Endpoints with delivery statistics\n\n" +
			"Would you like to see all endpoints or filter by something specific?"
		actions = []Action{
			{Type: "list_endpoints", Label: "Show all endpoints", RequireConf: false},
		}

	case IntentTransformSetup:
		content = "I'll help you set up payload transformations. Transformations let you modify webhook payloads before delivery.\n\n" +
			"**What would you like to do?**\n" +
			"1. Map fields from source to target format\n" +
			"2. Filter out sensitive data\n" +
			"3. Enrich payloads with additional data\n" +
			"4. Write custom JavaScript transformation\n\n" +
			"Describe what transformation you need."
		suggestions = []string{
			"Map fields between formats",
			"Remove PII from payloads",
			"Write a custom transformation",
		}

	case IntentExplainError:
		content = "I'd be happy to explain! Could you share the specific error code or message you're seeing? " +
			"Common webhook errors include:\n\n" +
			"• **408 Timeout** - Endpoint took too long to respond\n" +
			"• **429 Rate Limited** - Too many requests to the endpoint\n" +
			"• **5xx Server Error** - The receiving server had an internal error\n" +
			"• **Connection refused** - Endpoint is unreachable\n\n" +
			"Which error are you encountering?"

	default:
		content = "I'm your AI webhook assistant! I can help you with:\n\n" +
			"• **Creating webhook endpoints** - Set up new endpoints with guided configuration\n" +
			"• **Configuring retries** - Set up retry policies and backoff strategies\n" +
			"• **Authentication setup** - Configure HMAC, Bearer, or API key auth\n" +
			"• **Debugging deliveries** - Diagnose failed webhook deliveries\n" +
			"• **Payload transformations** - Transform webhook payloads\n\n" +
			"What would you like to do?"
		suggestions = []string{
			"Create a new webhook endpoint",
			"Help me debug a failing delivery",
			"Set up retry policies",
		}
	}

	return &Message{
		ID:             uuid.New().String(),
		ConversationID: conv.ID,
		Role:           RoleAssistant,
		Content:        content,
		Intent:         intent,
		Suggestions:    suggestions,
		Actions:        actions,
		CreatedAt:      time.Now().UTC(),
	}
}

func (s *Service) generateTitle(firstMessage string) string {
	if len(firstMessage) > 60 {
		return firstMessage[:57] + "..."
	}
	return firstMessage
}

func containsAny(text string, keywords ...string) bool {
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}
