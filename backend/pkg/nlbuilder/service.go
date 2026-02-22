package nlbuilder

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ServiceConfig configures the NL builder service.
type ServiceConfig struct {
	MaxConversationsPerTenant  int
	MaxMessagesPerConversation int
	DefaultRetryPolicy         *RetryPolicySpec
}

// DefaultServiceConfig returns sensible defaults.
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		MaxConversationsPerTenant:  100,
		MaxMessagesPerConversation: 50,
		DefaultRetryPolicy: &RetryPolicySpec{
			MaxRetries:  5,
			Strategy:    "exponential",
			InitialWait: "1s",
			MaxWait:     "1h",
		},
	}
}

// Service implements the NL webhook builder business logic.
type Service struct {
	repo   Repository
	llm    LLMProvider
	config *ServiceConfig
}

// NewService creates a new NL builder service.
func NewService(repo Repository, llm LLMProvider, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}
	if llm == nil {
		llm = &BuiltinIntentParser{}
	}
	return &Service{repo: repo, llm: llm, config: config}
}

// StartConversation creates a new builder conversation.
func (s *Service) StartConversation(tenantID string) (*Conversation, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}

	conv := &Conversation{
		ID:       uuid.New().String(),
		TenantID: tenantID,
		Messages: []ConversationMessage{
			{
				Role:      "system",
				Content:   systemPrompt,
				Timestamp: time.Now(),
			},
			{
				Role:      "assistant",
				Content:   "Hi! I can help you set up webhook endpoints. Describe what you'd like — for example: \"Send order events to https://api.example.com/webhooks with exponential retry\"",
				Timestamp: time.Now(),
			},
		},
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.CreateConversation(conv); err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}
	return conv, nil
}

// Chat processes a user message and returns a response with config updates.
func (s *Service) Chat(tenantID string, req *ChatRequest) (*ChatResponse, error) {
	if req.Message == "" {
		return nil, fmt.Errorf("message is required")
	}

	var conv *Conversation
	var err error

	if req.ConversationID != "" {
		conv, err = s.repo.GetConversation(req.ConversationID)
		if err != nil {
			return nil, fmt.Errorf("conversation not found: %w", err)
		}
	} else {
		conv, err = s.StartConversation(tenantID)
		if err != nil {
			return nil, err
		}
	}

	if len(conv.Messages) >= s.config.MaxMessagesPerConversation {
		return nil, fmt.Errorf("conversation has reached maximum messages (%d)", s.config.MaxMessagesPerConversation)
	}

	// Add user message
	conv.Messages = append(conv.Messages, ConversationMessage{
		Role:      "user",
		Content:   req.Message,
		Timestamp: time.Now(),
	})

	// Parse intent from the message
	intent, err := s.llm.ParseIntent(req.Message)
	if err != nil {
		return nil, fmt.Errorf("failed to parse intent: %w", err)
	}

	// Build or update config based on intent
	config := s.applyIntent(conv.Config, intent)
	conv.Config = config

	// Generate preview
	preview := s.generatePreview(config)

	// Generate reply
	reply := s.generateReply(intent, preview)

	conv.Messages = append(conv.Messages, ConversationMessage{
		Role:      "assistant",
		Content:   reply,
		Timestamp: time.Now(),
	})
	conv.UpdatedAt = time.Now()

	if err := s.repo.UpdateConversation(conv); err != nil {
		return nil, fmt.Errorf("failed to update conversation: %w", err)
	}

	complete := config != nil && config.URL != "" && len(config.EventTypes) > 0 && config.Validated

	return &ChatResponse{
		ConversationID: conv.ID,
		Reply:          reply,
		Intent:         intent,
		Preview:        preview,
		Suggestions:    s.getSuggestions(config),
		Complete:       complete,
	}, nil
}

// GetConversation retrieves a conversation by ID.
func (s *Service) GetConversation(id string) (*Conversation, error) {
	return s.repo.GetConversation(id)
}

// ListConversations returns all conversations for a tenant.
func (s *Service) ListConversations(tenantID string) ([]*Conversation, error) {
	return s.repo.ListConversations(tenantID)
}

// ApplyConfig finalizes and applies the generated config.
func (s *Service) ApplyConfig(conversationID string) (*GeneratedConfig, error) {
	conv, err := s.repo.GetConversation(conversationID)
	if err != nil {
		return nil, err
	}
	if conv.Config == nil {
		return nil, fmt.Errorf("no config generated yet")
	}
	if !conv.Config.Validated {
		return nil, fmt.Errorf("config has validation errors, please fix them first")
	}
	conv.Status = "completed"
	conv.UpdatedAt = time.Now()
	if err := s.repo.UpdateConversation(conv); err != nil {
		return nil, fmt.Errorf("failed to finalize conversation: %w", err)
	}
	return conv.Config, nil
}

func (s *Service) applyIntent(existing *GeneratedConfig, intent *ParsedIntent) *GeneratedConfig {
	if existing == nil {
		existing = &GeneratedConfig{
			RetryPolicy: s.config.DefaultRetryPolicy,
		}
	}

	switch intent.Action {
	case "create_endpoint":
		if intent.TargetURL != "" {
			existing.URL = intent.TargetURL
		}
		if len(intent.EventTypes) > 0 {
			existing.EventTypes = intent.EventTypes
		}
		if existing.EndpointName == "" && intent.TargetURL != "" {
			existing.EndpointName = "webhook-" + uuid.New().String()[:8]
		}
	case "configure_retry":
		if intent.RetryPolicy != nil {
			existing.RetryPolicy = intent.RetryPolicy
		}
	case "add_transform":
		if intent.Transform != nil {
			existing.Transform = intent.Transform
		}
	case "set_filter":
		if intent.Filter != nil {
			existing.Filter = intent.Filter
		}
	}

	// Validate
	existing.Validated = existing.URL != "" && len(existing.EventTypes) > 0
	existing.Warnings = nil
	if existing.URL != "" && !strings.HasPrefix(existing.URL, "https://") {
		existing.Warnings = append(existing.Warnings, "URL does not use HTTPS — consider using a secure endpoint")
	}

	return existing
}

func (s *Service) generatePreview(config *GeneratedConfig) *ConfigPreview {
	if config == nil {
		return nil
	}
	jsonBytes, _ := json.MarshalIndent(config, "", "  ")
	return &ConfigPreview{
		Config:      config,
		JSONPreview: string(jsonBytes),
		Validated:   config.Validated,
	}
}

func (s *Service) generateReply(intent *ParsedIntent, preview *ConfigPreview) string {
	if intent.Confidence < 0.5 {
		return fmt.Sprintf("I'm not quite sure what you mean. Could you clarify? For example, try: \"Create an endpoint at https://example.com/webhook for order.created events\"")
	}

	var parts []string
	switch intent.Action {
	case "create_endpoint":
		if intent.TargetURL != "" {
			parts = append(parts, fmt.Sprintf("I'll set up an endpoint at **%s**", intent.TargetURL))
		}
		if len(intent.EventTypes) > 0 {
			parts = append(parts, fmt.Sprintf("listening for **%s** events", strings.Join(intent.EventTypes, ", ")))
		}
	case "configure_retry":
		parts = append(parts, "I've updated the retry policy")
	case "add_transform":
		parts = append(parts, "I've added a payload transformation")
	case "set_filter":
		parts = append(parts, "I've configured event filtering")
	}

	reply := strings.Join(parts, " ")
	if preview != nil && preview.Validated {
		reply += ". The configuration looks complete — say **apply** to create the endpoint, or describe any changes."
	} else if preview != nil {
		reply += ". What else would you like to configure?"
	}
	return reply
}

func (s *Service) getSuggestions(config *GeneratedConfig) []string {
	if config == nil {
		return []string{
			"Create an endpoint at https://example.com/webhook",
			"Send order events to my API",
		}
	}
	var suggestions []string
	if config.RetryPolicy == nil {
		suggestions = append(suggestions, "Add exponential retry with 5 attempts")
	}
	if config.Transform == nil {
		suggestions = append(suggestions, "Add a transform to rename fields")
	}
	if config.Filter == nil {
		suggestions = append(suggestions, "Filter events where status is active")
	}
	if config.Validated {
		suggestions = append(suggestions, "Apply this configuration")
	}
	return suggestions
}

const systemPrompt = `You are a webhook configuration assistant. Help users create webhook endpoint configurations by understanding their natural language descriptions. Extract: target URL, event types, retry policies, transformations, and filters. Always confirm before applying.`
