package nlbuilder

import (
	"encoding/json"
	"fmt"
	"regexp"
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

// ValidateConfig performs comprehensive validation on a generated config.
func (s *Service) ValidateConfig(config *GeneratedConfig) *ValidationResult {
	result := &ValidationResult{Valid: true, Score: 1.0}

	if config.URL == "" {
		result.Errors = append(result.Errors, ValidationError{Field: "url", Message: "URL is required"})
		result.Valid = false
		result.Score -= 0.3
	} else if !strings.HasPrefix(config.URL, "http://") && !strings.HasPrefix(config.URL, "https://") {
		result.Errors = append(result.Errors, ValidationError{Field: "url", Message: "URL must start with http:// or https://"})
		result.Valid = false
		result.Score -= 0.3
	}

	if len(config.EventTypes) == 0 {
		result.Errors = append(result.Errors, ValidationError{Field: "event_types", Message: "at least one event type is required"})
		result.Valid = false
		result.Score -= 0.2
	}

	if config.URL != "" && !strings.HasPrefix(config.URL, "https://") {
		result.Warnings = append(result.Warnings, "endpoint does not use HTTPS — consider using a secure URL")
		result.Score -= 0.1
	}

	if config.RetryPolicy == nil {
		result.Warnings = append(result.Warnings, "no retry policy configured — deliveries will not be retried on failure")
		result.Score -= 0.05
	} else if config.RetryPolicy.MaxRetries > 15 {
		result.Warnings = append(result.Warnings, "high retry count may cause delivery delays")
		result.Score -= 0.05
	}

	if config.RateLimit > 0 && config.RateLimit < 10 {
		result.Warnings = append(result.Warnings, "very low rate limit may cause delivery backlogs")
	}

	for i, rule := range config.RoutingRules {
		if rule.Name == "" {
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("routing_rules[%d].name", i),
				Message: "routing rule name is required",
			})
			result.Valid = false
		}
		if rule.Destination == "" {
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("routing_rules[%d].destination", i),
				Message: "routing rule destination is required",
			})
			result.Valid = false
		}
	}

	if result.Score < 0 {
		result.Score = 0
	}

	return result
}

// GenerateRoutingRules creates routing rules from a natural language description.
func (s *Service) GenerateRoutingRules(tenantID string, conversationID string, description string) ([]RoutingRule, error) {
	if description == "" {
		return nil, fmt.Errorf("routing description is required")
	}

	msg := strings.ToLower(description)
	var rules []RoutingRule

	// Route by event type
	if strings.Contains(msg, "route") || strings.Contains(msg, "send") {
		eventPattern := regexp.MustCompile(`([a-z]+\.[a-z]+(?:\.[a-z]+)?)`)
		urlPattern := regexp.MustCompile(`https?://[^\s"']+`)

		events := eventPattern.FindAllString(msg, -1)
		urls := urlPattern.FindAllString(description, -1)

		for i, event := range events {
			if isCommonPhrase(event) {
				continue
			}
			dest := ""
			if i < len(urls) {
				dest = urls[i]
			} else if len(urls) > 0 {
				dest = urls[0]
			}
			rules = append(rules, RoutingRule{
				Name:        fmt.Sprintf("route-%s", event),
				Condition:   fmt.Sprintf("event_type == '%s'", event),
				Destination: dest,
				Priority:    i + 1,
			})
		}
	}

	// Failover routing
	if strings.Contains(msg, "failover") || strings.Contains(msg, "fallback") {
		urlPattern := regexp.MustCompile(`https?://[^\s"']+`)
		urls := urlPattern.FindAllString(description, -1)
		for i, url := range urls {
			rules = append(rules, RoutingRule{
				Name:        fmt.Sprintf("failover-%d", i+1),
				Condition:   fmt.Sprintf("attempt >= %d", i+1),
				Destination: url,
				Priority:    i + 1,
			})
		}
	}

	if len(rules) == 0 {
		rules = append(rules, RoutingRule{
			Name:      "default",
			Condition: "true",
			Priority:  1,
		})
	}

	// Update conversation if exists
	if conversationID != "" {
		conv, err := s.repo.GetConversation(conversationID)
		if err == nil && conv.Config != nil {
			conv.Config.RoutingRules = rules
			_ = s.repo.UpdateConversation(conv)
		}
	}

	return rules, nil
}

// GetRefinementSuggestions returns AI-generated suggestions to improve a config.
func (s *Service) GetRefinementSuggestions(config *GeneratedConfig) []RefinementSuggestion {
	var suggestions []RefinementSuggestion

	if config.AuthConfig == nil {
		suggestions = append(suggestions, RefinementSuggestion{
			Category:    "security",
			Description: "Add HMAC signature verification for payload authenticity",
			AutoApply:   false,
		})
	}

	if config.RetryPolicy == nil {
		suggestions = append(suggestions, RefinementSuggestion{
			Category:    "reliability",
			Description: "Add exponential backoff retry policy (5 attempts, 1s initial delay)",
			AutoApply:   true,
		})
	}

	if config.RateLimit == 0 {
		suggestions = append(suggestions, RefinementSuggestion{
			Category:    "performance",
			Description: "Set a rate limit to prevent overwhelming the target endpoint",
			AutoApply:   false,
		})
	}

	if config.Transform == nil {
		suggestions = append(suggestions, RefinementSuggestion{
			Category:    "compatibility",
			Description: "Add a payload transformation to match the target API's expected format",
			AutoApply:   false,
		})
	}

	if len(config.RoutingRules) == 0 && len(config.EventTypes) > 1 {
		suggestions = append(suggestions, RefinementSuggestion{
			Category:    "routing",
			Description: "Add routing rules to send different event types to different destinations",
			AutoApply:   false,
		})
	}

	return suggestions
}
