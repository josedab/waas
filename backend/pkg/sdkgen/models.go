package sdkgen

import (
	"encoding/json"
	"time"
)

// Language constants
const (
	LangTypeScript = "typescript"
	LangPython     = "python"
	LangGo         = "go"
	LangJava       = "java"
)

// SchemaDefinition is a simplified JSON Schema for code generation.
type SchemaDefinition struct {
	ID         string                 `json:"id"`
	TenantID   string                 `json:"tenant_id"`
	EventType  string                 `json:"event_type"`
	Version    string                 `json:"version"`
	Schema     json.RawMessage        `json:"schema"`
	Properties map[string]PropertyDef `json:"properties,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

// PropertyDef defines a single property in a schema.
type PropertyDef struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Format      string `json:"format,omitempty"`
}

// GeneratedSDK holds the output of SDK generation.
type GeneratedSDK struct {
	ID        string            `json:"id"`
	TenantID  string            `json:"tenant_id"`
	Language  string            `json:"language"`
	Version   string            `json:"version"`
	Files     map[string]string `json:"files"`
	CreatedAt time.Time         `json:"created_at"`
}

// SDKPortalEntry represents an SDK available for download.
type SDKPortalEntry struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Language    string    `json:"language"`
	Version     string    `json:"version"`
	EventTypes  []string  `json:"event_types"`
	DownloadURL string    `json:"download_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// Request DTOs

type GenerateSDKRequest struct {
	Language    string             `json:"language" binding:"required"`
	EventTypes  []string           `json:"event_types" binding:"required"`
	Schemas     []SchemaDefinition `json:"schemas" binding:"required"`
	PackageName string             `json:"package_name"`
}

type ListSDKsRequest struct {
	Language string `json:"language,omitempty"`
}
