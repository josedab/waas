package docgen

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// WebhookDoc represents a webhook API documentation resource
type WebhookDoc struct {
	ID          uuid.UUID `json:"id" db:"id"`
	TenantID    uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description,omitempty" db:"description"`
	Version     string    `json:"version" db:"version"`
	EventTypes  []string  `json:"event_types,omitempty" db:"event_types"`
	BaseURL     string    `json:"base_url,omitempty" db:"base_url"`
	AuthMethod  string    `json:"auth_method,omitempty" db:"auth_method"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// EventTypeDoc represents documentation for a specific event type
type EventTypeDoc struct {
	ID                uuid.UUID       `json:"id" db:"id"`
	DocID             uuid.UUID       `json:"doc_id" db:"doc_id"`
	Name              string          `json:"name" db:"name"`
	Description       string          `json:"description,omitempty" db:"description"`
	Category          string          `json:"category,omitempty" db:"category"`
	PayloadSchema     json.RawMessage `json:"payload_schema,omitempty" db:"payload_schema"`
	ExamplePayload    json.RawMessage `json:"example_payload,omitempty" db:"example_payload"`
	Deprecated        bool            `json:"deprecated" db:"deprecated"`
	DeprecationNotice string          `json:"deprecation_notice,omitempty" db:"deprecation_notice"`
	Version           string          `json:"version" db:"version"`
	CreatedAt         time.Time       `json:"created_at" db:"created_at"`
}

// CodeSample represents a generated code sample for handling a webhook event
type CodeSample struct {
	ID          uuid.UUID `json:"id" db:"id"`
	EventTypeID uuid.UUID `json:"event_type_id" db:"event_type_id"`
	Language    string    `json:"language" db:"language"`
	Code        string    `json:"code" db:"code"`
	Framework   string    `json:"framework,omitempty" db:"framework"`
	Description string    `json:"description,omitempty" db:"description"`
}

// SupportedLanguage constants
const (
	LangGo     = "go"
	LangPython = "python"
	LangNodeJS = "nodejs"
	LangJava   = "java"
	LangRuby   = "ruby"
	LangPHP    = "php"
	LangCSharp = "csharp"
	LangCurl   = "curl"
)

// DocWidget represents an embeddable documentation widget configuration
type DocWidget struct {
	ID             uuid.UUID `json:"id" db:"id"`
	TenantID       uuid.UUID `json:"tenant_id" db:"tenant_id"`
	DocID          uuid.UUID `json:"doc_id" db:"doc_id"`
	Theme          string    `json:"theme" db:"theme"`
	CustomCSS      string    `json:"custom_css,omitempty" db:"custom_css"`
	AllowedDomains []string  `json:"allowed_domains,omitempty" db:"allowed_domains"`
	EmbedKey       string    `json:"embed_key" db:"embed_key"`
	ViewCount      int64     `json:"view_count" db:"view_count"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// WidgetTheme constants
const (
	ThemeLight = "light"
	ThemeDark  = "dark"
	ThemeAuto  = "auto"
)

// EventCatalogEntry represents a single entry in the event catalog
type EventCatalogEntry struct {
	EventTypeID uuid.UUID `json:"event_type_id"`
	Name        string    `json:"name"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	Version     string    `json:"version"`
	Deprecated  bool      `json:"deprecated"`
	SchemaURL   string    `json:"schema_url,omitempty"`
	ExampleURL  string    `json:"example_url,omitempty"`
}

// EventCatalog represents a browsable catalog of all event types
type EventCatalog struct {
	TenantID    uuid.UUID            `json:"tenant_id"`
	Entries     []EventCatalogEntry  `json:"entries"`
	Categories  []string             `json:"categories"`
	TotalEvents int                  `json:"total_events"`
}

// DocAnalytics represents analytics for a documentation resource
type DocAnalytics struct {
	DocID          uuid.UUID        `json:"doc_id"`
	Views          int64            `json:"views"`
	UniqueVisitors int64            `json:"unique_visitors"`
	TopEvents      []string         `json:"top_events"`
	AvgTimeOnPage  float64          `json:"avg_time_on_page"`
	WidgetViews    int64            `json:"widget_views"`
}

// CreateDocRequest represents a request to create a webhook doc
type CreateDocRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description,omitempty"`
	Version     string   `json:"version,omitempty"`
	EventTypes  []string `json:"event_types,omitempty"`
	BaseURL     string   `json:"base_url,omitempty"`
	AuthMethod  string   `json:"auth_method,omitempty"`
}

// AddEventTypeRequest represents a request to add an event type to a doc
type AddEventTypeRequest struct {
	Name              string          `json:"name" binding:"required"`
	Description       string          `json:"description,omitempty"`
	Category          string          `json:"category,omitempty"`
	PayloadSchema     json.RawMessage `json:"payload_schema,omitempty"`
	ExamplePayload    json.RawMessage `json:"example_payload,omitempty"`
	Deprecated        bool            `json:"deprecated"`
	DeprecationNotice string          `json:"deprecation_notice,omitempty"`
	Version           string          `json:"version,omitempty"`
}

// GenerateCodeRequest represents a request to generate a code sample
type GenerateCodeRequest struct {
	EventTypeID uuid.UUID `json:"event_type_id" binding:"required"`
	Language    string    `json:"language" binding:"required"`
}

// CreateWidgetRequest represents a request to create a doc widget
type CreateWidgetRequest struct {
	DocID          uuid.UUID `json:"doc_id" binding:"required"`
	Theme          string    `json:"theme,omitempty"`
	CustomCSS      string    `json:"custom_css,omitempty"`
	AllowedDomains []string  `json:"allowed_domains,omitempty"`
}
