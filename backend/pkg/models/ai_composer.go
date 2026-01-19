package models

import (
	"time"

	"github.com/google/uuid"
)

// AIComposerSession represents an AI conversation session
type AIComposerSession struct {
	ID        uuid.UUID              `json:"id" db:"id"`
	TenantID  uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	Status    string                 `json:"status" db:"status"`
	Context   map[string]interface{} `json:"context" db:"context"`
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt time.Time              `json:"updated_at" db:"updated_at"`
	ExpiresAt time.Time              `json:"expires_at" db:"expires_at"`
}

// AIComposerMessage represents a message in an AI conversation
type AIComposerMessage struct {
	ID        uuid.UUID              `json:"id" db:"id"`
	SessionID uuid.UUID              `json:"session_id" db:"session_id"`
	Role      string                 `json:"role" db:"role"`
	Content   string                 `json:"content" db:"content"`
	Metadata  map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
}

// AIComposerGeneratedConfig represents a generated webhook configuration
type AIComposerGeneratedConfig struct {
	ID                 uuid.UUID              `json:"id" db:"id"`
	SessionID          uuid.UUID              `json:"session_id" db:"session_id"`
	TenantID           uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	ConfigType         string                 `json:"config_type" db:"config_type"`
	GeneratedConfig    map[string]interface{} `json:"generated_config" db:"generated_config"`
	TransformationCode string                 `json:"transformation_code,omitempty" db:"transformation_code"`
	ValidationStatus   string                 `json:"validation_status" db:"validation_status"`
	ValidationErrors   []string               `json:"validation_errors" db:"validation_errors"`
	Applied            bool                   `json:"applied" db:"applied"`
	AppliedAt          *time.Time             `json:"applied_at,omitempty" db:"applied_at"`
	CreatedAt          time.Time              `json:"created_at" db:"created_at"`
}

// AIComposerTemplate represents a prompt template for common use cases
type AIComposerTemplate struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	Name           string                 `json:"name" db:"name"`
	Description    string                 `json:"description" db:"description"`
	Category       string                 `json:"category" db:"category"`
	PromptTemplate string                 `json:"prompt_template" db:"prompt_template"`
	ExampleInput   string                 `json:"example_input" db:"example_input"`
	ExampleOutput  map[string]interface{} `json:"example_output" db:"example_output"`
	IsActive       bool                   `json:"is_active" db:"is_active"`
	UsageCount     int                    `json:"usage_count" db:"usage_count"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at" db:"updated_at"`
}

// AIComposerFeedback represents user feedback on generated configs
type AIComposerFeedback struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	SessionID        uuid.UUID  `json:"session_id" db:"session_id"`
	ConfigID         *uuid.UUID `json:"config_id,omitempty" db:"config_id"`
	Rating           int        `json:"rating" db:"rating"`
	FeedbackText     string     `json:"feedback_text" db:"feedback_text"`
	WorkedAsExpected bool       `json:"worked_as_expected" db:"worked_as_expected"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
}

// AIComposerRequest represents a user request to the AI composer
type AIComposerRequest struct {
	Prompt     string                 `json:"prompt" binding:"required"`
	SessionID  *uuid.UUID             `json:"session_id,omitempty"`
	TemplateID *uuid.UUID             `json:"template_id,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
}

// AIComposerResponse represents the AI composer response
type AIComposerResponse struct {
	SessionID       uuid.UUID                  `json:"session_id"`
	Message         string                     `json:"message"`
	GeneratedConfig *AIComposerGeneratedConfig `json:"generated_config,omitempty"`
	Suggestions     []string                   `json:"suggestions,omitempty"`
	NeedsMoreInfo   bool                       `json:"needs_more_info"`
	Questions       []string                   `json:"questions,omitempty"`
}

// Message roles
const (
	AIRoleUser      = "user"
	AIRoleAssistant = "assistant"
	AIRoleSystem    = "system"
)

// Session statuses
const (
	AISessionStatusActive    = "active"
	AISessionStatusCompleted = "completed"
	AISessionStatusExpired   = "expired"
)

// Config types
const (
	AIConfigTypeEndpoint       = "endpoint"
	AIConfigTypeTransformation = "transformation"
	AIConfigTypeFlow           = "flow"
	AIConfigTypeFilter         = "filter"
)

// Validation statuses
const (
	AIValidationPending  = "pending"
	AIValidationValid    = "valid"
	AIValidationInvalid  = "invalid"
	AIValidationWarnings = "warnings"
)
