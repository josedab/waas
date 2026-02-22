package nlbuilder

import "time"

// ConversationMessage represents a single message in the builder chat.
type ConversationMessage struct {
	Role      string    `json:"role"` // user, assistant, system
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// Conversation tracks a multi-turn builder session.
type Conversation struct {
	ID        string                `json:"id"`
	TenantID  string                `json:"tenant_id"`
	Messages  []ConversationMessage `json:"messages"`
	Config    *GeneratedConfig      `json:"config,omitempty"`
	Status    string                `json:"status"` // active, completed, abandoned
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`
}

// ParsedIntent represents the extracted intent from natural language.
type ParsedIntent struct {
	Action      string           `json:"action"` // create_endpoint, configure_retry, add_transform, set_filter
	EventTypes  []string         `json:"event_types"`
	TargetURL   string           `json:"target_url"`
	RetryPolicy *RetryPolicySpec `json:"retry_policy,omitempty"`
	Transform   *TransformSpec   `json:"transform,omitempty"`
	Filter      *FilterSpec      `json:"filter,omitempty"`
	Confidence  float64          `json:"confidence"` // 0.0 - 1.0
	Ambiguities []string         `json:"ambiguities,omitempty"`
	RawQuery    string           `json:"raw_query"`
}

// RetryPolicySpec describes a retry configuration.
type RetryPolicySpec struct {
	MaxRetries  int    `json:"max_retries"`
	Strategy    string `json:"strategy"`     // exponential, linear, fixed
	InitialWait string `json:"initial_wait"` // e.g. "1s", "5s"
	MaxWait     string `json:"max_wait"`     // e.g. "1h", "24h"
}

// TransformSpec describes a payload transformation.
type TransformSpec struct {
	Language   string            `json:"language"` // javascript, jsonpath, jmespath
	Expression string            `json:"expression"`
	FieldMap   map[string]string `json:"field_map,omitempty"`
}

// FilterSpec describes an event filter.
type FilterSpec struct {
	Conditions []FilterCondition `json:"conditions"`
	Logic      string            `json:"logic"` // and, or
}

// FilterCondition is a single filter rule.
type FilterCondition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"` // eq, neq, contains, gt, lt, exists
	Value    string `json:"value"`
}

// GeneratedConfig is the fully-generated webhook configuration.
type GeneratedConfig struct {
	EndpointName string            `json:"endpoint_name"`
	URL          string            `json:"url"`
	EventTypes   []string          `json:"event_types"`
	RetryPolicy  *RetryPolicySpec  `json:"retry_policy,omitempty"`
	Transform    *TransformSpec    `json:"transform,omitempty"`
	Filter       *FilterSpec       `json:"filter,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	RateLimit    int               `json:"rate_limit,omitempty"`
	Validated    bool              `json:"validated"`
	Warnings     []string          `json:"warnings,omitempty"`
}

// ConfigPreview shows a before/after view of the generated config.
type ConfigPreview struct {
	Config      *GeneratedConfig `json:"config"`
	JSONPreview string           `json:"json_preview"`
	YAMLPreview string           `json:"yaml_preview"`
	Validated   bool             `json:"validated"`
	Errors      []string         `json:"errors,omitempty"`
	Suggestions []string         `json:"suggestions,omitempty"`
}

// ChatRequest is the DTO for sending a message in the builder chat.
type ChatRequest struct {
	ConversationID string `json:"conversation_id"`
	Message        string `json:"message" binding:"required"`
}

// ChatResponse is the response from the builder chat.
type ChatResponse struct {
	ConversationID string         `json:"conversation_id"`
	Reply          string         `json:"reply"`
	Intent         *ParsedIntent  `json:"intent,omitempty"`
	Preview        *ConfigPreview `json:"preview,omitempty"`
	Suggestions    []string       `json:"suggestions,omitempty"`
	Complete       bool           `json:"complete"`
}

// LLMProvider defines the interface for LLM API integration.
type LLMProvider interface {
	Complete(systemPrompt string, messages []ConversationMessage) (string, error)
	ParseIntent(userMessage string) (*ParsedIntent, error)
}
