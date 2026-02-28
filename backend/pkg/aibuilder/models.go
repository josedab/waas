package aibuilder

import "time"

// ConversationStatus represents the state of an AI builder conversation.
type ConversationStatus string

const (
	StatusActive    ConversationStatus = "active"
	StatusCompleted ConversationStatus = "completed"
	StatusAbandoned ConversationStatus = "abandoned"
	StatusError     ConversationStatus = "error"
)

// MessageRole identifies who sent a message in the conversation.
type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleSystem    MessageRole = "system"
)

// IntentType classifies the user's intent in a conversation turn.
type IntentType string

const (
	IntentCreateEndpoint IntentType = "create_endpoint"
	IntentConfigureRetry IntentType = "configure_retry"
	IntentSetupAuth      IntentType = "setup_auth"
	IntentDebugDelivery  IntentType = "debug_delivery"
	IntentListEndpoints  IntentType = "list_endpoints"
	IntentExplainError   IntentType = "explain_error"
	IntentTransformSetup IntentType = "transform_setup"
	IntentGeneral        IntentType = "general"
	IntentUnknown        IntentType = "unknown"
)

// Conversation represents an AI builder session with a user.
type Conversation struct {
	ID        string               `json:"id" db:"id"`
	TenantID  string               `json:"tenant_id" db:"tenant_id"`
	Status    ConversationStatus   `json:"status" db:"status"`
	Title     string               `json:"title" db:"title"`
	Context   *ConversationContext `json:"context" db:"context"`
	CreatedAt time.Time            `json:"created_at" db:"created_at"`
	UpdatedAt time.Time            `json:"updated_at" db:"updated_at"`
	ExpiresAt time.Time            `json:"expires_at" db:"expires_at"`
}

// ConversationContext holds accumulated state across conversation turns.
type ConversationContext struct {
	DetectedIntents []IntentType           `json:"detected_intents"`
	EndpointDraft   *EndpointDraft         `json:"endpoint_draft,omitempty"`
	Variables       map[string]interface{} `json:"variables,omitempty"`
	StepIndex       int                    `json:"step_index"`
}

// EndpointDraft represents a partially-configured webhook endpoint
// built up through conversation.
type EndpointDraft struct {
	URL             string            `json:"url,omitempty"`
	EventTypes      []string          `json:"event_types,omitempty"`
	Description     string            `json:"description,omitempty"`
	RetryPolicy     *RetryPolicyDraft `json:"retry_policy,omitempty"`
	AuthType        string            `json:"auth_type,omitempty"`
	Headers         map[string]string `json:"headers,omitempty"`
	TransformCode   string            `json:"transform_code,omitempty"`
	RateLimitPerSec int               `json:"rate_limit_per_sec,omitempty"`
}

// RetryPolicyDraft holds retry configuration being built through conversation.
type RetryPolicyDraft struct {
	MaxRetries     int    `json:"max_retries"`
	BackoffType    string `json:"backoff_type"` // linear, exponential
	InitialDelayMs int    `json:"initial_delay_ms"`
	MaxDelayMs     int    `json:"max_delay_ms"`
}

// Message represents a single message in the conversation.
type Message struct {
	ID             string      `json:"id" db:"id"`
	ConversationID string      `json:"conversation_id" db:"conversation_id"`
	Role           MessageRole `json:"role" db:"role"`
	Content        string      `json:"content" db:"content"`
	Intent         IntentType  `json:"intent,omitempty" db:"intent"`
	Suggestions    []string    `json:"suggestions,omitempty" db:"-"`
	Actions        []Action    `json:"actions,omitempty" db:"-"`
	CreatedAt      time.Time   `json:"created_at" db:"created_at"`
}

// Action represents an actionable operation suggested or executed by the AI.
type Action struct {
	Type        string                 `json:"type"` // create_endpoint, update_config, test_delivery, show_logs
	Label       string                 `json:"label"`
	Params      map[string]interface{} `json:"params,omitempty"`
	RequireConf bool                   `json:"require_confirmation"`
	Executed    bool                   `json:"executed"`
}

// SendMessageRequest is the API request for sending a user message.
type SendMessageRequest struct {
	Message        string `json:"message" binding:"required"`
	ConversationID string `json:"conversation_id,omitempty"`
	ExecuteAction  string `json:"execute_action,omitempty"`
}

// SendMessageResponse is the API response after processing a message.
type SendMessageResponse struct {
	ConversationID string   `json:"conversation_id"`
	Reply          *Message `json:"reply"`
}

// ConversationSummary is a lightweight view for listing conversations.
type ConversationSummary struct {
	ID        string             `json:"id"`
	Title     string             `json:"title"`
	Status    ConversationStatus `json:"status"`
	Messages  int                `json:"message_count"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
}

// DebugRequest asks the AI to diagnose a specific delivery issue.
type DebugRequest struct {
	DeliveryID string `json:"delivery_id" binding:"required"`
	Question   string `json:"question,omitempty"`
}

// DebugResponse contains AI-generated diagnostic information.
type DebugResponse struct {
	DeliveryID  string   `json:"delivery_id"`
	Summary     string   `json:"summary"`
	RootCause   string   `json:"root_cause"`
	Suggestions []string `json:"suggestions"`
	RelatedDocs []string `json:"related_docs,omitempty"`
}
