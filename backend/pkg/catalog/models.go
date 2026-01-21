package catalog

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventType represents a webhook event type in the catalog
type EventType struct {
	ID                 uuid.UUID       `json:"id" db:"id"`
	TenantID           uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	Name               string          `json:"name" db:"name"`
	Slug               string          `json:"slug" db:"slug"`
	Description        string          `json:"description,omitempty" db:"description"`
	Category           string          `json:"category,omitempty" db:"category"`
	SchemaID           *uuid.UUID      `json:"schema_id,omitempty" db:"schema_id"`
	Version            string          `json:"version" db:"version"`
	Status             string          `json:"status" db:"status"`
	DeprecationMessage string          `json:"deprecation_message,omitempty" db:"deprecation_message"`
	DeprecatedAt       *time.Time      `json:"deprecated_at,omitempty" db:"deprecated_at"`
	ReplacementEventID *uuid.UUID      `json:"replacement_event_id,omitempty" db:"replacement_event_id"`
	ExamplePayload     json.RawMessage `json:"example_payload,omitempty" db:"example_payload"`
	Tags               []string        `json:"tags,omitempty" db:"tags"`
	Metadata           json.RawMessage `json:"metadata,omitempty" db:"metadata"`
	DocumentationURL   string          `json:"documentation_url,omitempty" db:"documentation_url"`
	CreatedAt          time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at" db:"updated_at"`

	// Joined data
	Schema       *EventSchema      `json:"schema,omitempty"`
	Versions     []*EventVersion   `json:"versions,omitempty"`
	Subscribers  int               `json:"subscribers,omitempty"`
	Replacement  *EventType        `json:"replacement,omitempty"`
}

// EventVersion represents a version of an event type
type EventVersion struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	EventTypeID     uuid.UUID  `json:"event_type_id" db:"event_type_id"`
	Version         string     `json:"version" db:"version"`
	SchemaID        *uuid.UUID `json:"schema_id,omitempty" db:"schema_id"`
	Changelog       string     `json:"changelog,omitempty" db:"changelog"`
	IsBreakingChange bool      `json:"is_breaking_change" db:"is_breaking_change"`
	PublishedAt     time.Time  `json:"published_at" db:"published_at"`
	PublishedBy     *uuid.UUID `json:"published_by,omitempty" db:"published_by"`
}

// EventCategory represents a category for organizing event types
type EventCategory struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	TenantID    uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	Name        string     `json:"name" db:"name"`
	Slug        string     `json:"slug" db:"slug"`
	Description string     `json:"description,omitempty" db:"description"`
	Icon        string     `json:"icon,omitempty" db:"icon"`
	Color       string     `json:"color,omitempty" db:"color"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty" db:"parent_id"`
	SortOrder   int        `json:"sort_order" db:"sort_order"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`

	// Joined data
	EventCount int              `json:"event_count,omitempty"`
	Children   []*EventCategory `json:"children,omitempty"`
}

// EventSubscription links an endpoint to an event type
type EventSubscription struct {
	ID               uuid.UUID       `json:"id" db:"id"`
	EndpointID       uuid.UUID       `json:"endpoint_id" db:"endpoint_id"`
	EventTypeID      uuid.UUID       `json:"event_type_id" db:"event_type_id"`
	FilterExpression json.RawMessage `json:"filter_expression,omitempty" db:"filter_expression"`
	IsActive         bool            `json:"is_active" db:"is_active"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
}

// EventDocumentation holds rich documentation for an event type
type EventDocumentation struct {
	ID          uuid.UUID `json:"id" db:"id"`
	EventTypeID uuid.UUID `json:"event_type_id" db:"event_type_id"`
	ContentType string    `json:"content_type" db:"content_type"`
	Content     string    `json:"content" db:"content"`
	Section     string    `json:"section" db:"section"`
	SortOrder   int       `json:"sort_order" db:"sort_order"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// SDKConfig holds configuration for SDK generation
type SDKConfig struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	TenantID        uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	Language        string          `json:"language" db:"language"`
	PackageName     string          `json:"package_name,omitempty" db:"package_name"`
	Version         string          `json:"version,omitempty" db:"version"`
	Config          json.RawMessage `json:"config,omitempty" db:"config"`
	LastGeneratedAt *time.Time      `json:"last_generated_at,omitempty" db:"last_generated_at"`
	DownloadURL     string          `json:"download_url,omitempty" db:"download_url"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at" db:"updated_at"`
}

// EventSchema represents the JSON schema for an event (links to schema registry)
type EventSchema struct {
	ID         uuid.UUID       `json:"id"`
	Name       string          `json:"name"`
	Version    string          `json:"version"`
	Schema     json.RawMessage `json:"schema"`
	Properties []SchemaProperty `json:"properties,omitempty"`
}

// SchemaProperty represents a property in the schema (for documentation)
type SchemaProperty struct {
	Name        string           `json:"name"`
	Type        string           `json:"type"`
	Description string           `json:"description,omitempty"`
	Required    bool             `json:"required"`
	Example     interface{}      `json:"example,omitempty"`
	Properties  []SchemaProperty `json:"properties,omitempty"`
}

// Event status constants
const (
	StatusActive     = "active"
	StatusDeprecated = "deprecated"
	StatusDraft      = "draft"
)

// SDK language constants
const (
	LangTypeScript = "typescript"
	LangPython     = "python"
	LangGo         = "go"
	LangJava       = "java"
	LangRuby       = "ruby"
	LangPHP        = "php"
	LangCSharp     = "csharp"
)

// CatalogSearchParams for searching event types
type CatalogSearchParams struct {
	TenantID   uuid.UUID
	Query      string
	Category   string
	Status     string
	Tags       []string
	Version    string
	Limit      int
	Offset     int
	SortBy     string
	SortOrder  string
}

// CatalogSearchResult represents search results
type CatalogSearchResult struct {
	EventTypes []*EventType `json:"event_types"`
	Total      int          `json:"total"`
	Limit      int          `json:"limit"`
	Offset     int          `json:"offset"`
}
