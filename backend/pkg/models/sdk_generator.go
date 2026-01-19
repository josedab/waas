package models

import (
	"time"

	"github.com/google/uuid"
)

// SDK generation statuses
const (
	SDKStatusPending    = "pending"
	SDKStatusGenerating = "generating"
	SDKStatusCompleted  = "completed"
	SDKStatusFailed     = "failed"
)

// Supported languages
const (
	SDKLanguageGo         = "go"
	SDKLanguageTypeScript = "typescript"
	SDKLanguagePython     = "python"
	SDKLanguageJava       = "java"
	SDKLanguageRuby       = "ruby"
	SDKLanguagePHP        = "php"
)

// SDKConfiguration represents a white-label SDK configuration
type SDKConfiguration struct {
	ID               uuid.UUID         `json:"id" db:"id"`
	TenantID         uuid.UUID         `json:"tenant_id" db:"tenant_id"`
	Name             string            `json:"name" db:"name"`
	Description      string            `json:"description" db:"description"`
	PackagePrefix    string            `json:"package_prefix" db:"package_prefix"`
	OrganizationName string            `json:"organization_name" db:"organization_name"`
	Branding         SDKBranding       `json:"branding" db:"branding"`
	Languages        []string          `json:"languages" db:"languages"`
	APIBaseURL       string            `json:"api_base_url" db:"api_base_url"`
	Features         SDKFeatures       `json:"features" db:"features"`
	IsActive         bool              `json:"is_active" db:"is_active"`
	CreatedAt        time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at" db:"updated_at"`
}

// SDKBranding holds branding customization options
type SDKBranding struct {
	LogoURL          string            `json:"logo_url,omitempty"`
	PrimaryColor     string            `json:"primary_color,omitempty"`
	DocumentationURL string            `json:"documentation_url,omitempty"`
	SupportEmail     string            `json:"support_email,omitempty"`
	CustomHeaders    map[string]string `json:"custom_headers,omitempty"`
	LicenseType      string            `json:"license_type,omitempty"`
	CopyrightHolder  string            `json:"copyright_holder,omitempty"`
}

// SDKFeatures holds feature flags for SDK generation
type SDKFeatures struct {
	IncludeWebhooks       bool `json:"include_webhooks"`
	IncludeAnalytics      bool `json:"include_analytics"`
	IncludeSignatureVerify bool `json:"include_signature_verify"`
	IncludeRetryLogic     bool `json:"include_retry_logic"`
	IncludeRateLimiting   bool `json:"include_rate_limiting"`
	IncludeAsyncMethods   bool `json:"include_async_methods"`
	IncludeExamples       bool `json:"include_examples"`
	IncludeTests          bool `json:"include_tests"`
}

// SDKGeneration represents a generated SDK artifact
type SDKGeneration struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	ConfigID        uuid.UUID  `json:"config_id" db:"config_id"`
	TenantID        uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	Version         string     `json:"version" db:"version"`
	Language        string     `json:"language" db:"language"`
	Status          string     `json:"status" db:"status"`
	OpenAPISpecHash string     `json:"openapi_spec_hash,omitempty" db:"openapi_spec_hash"`
	ArtifactURL     string     `json:"artifact_url,omitempty" db:"artifact_url"`
	ArtifactSizeBytes int64    `json:"artifact_size_bytes,omitempty" db:"artifact_size_bytes"`
	PackageRegistry string     `json:"package_registry,omitempty" db:"package_registry"`
	PackageName     string     `json:"package_name,omitempty" db:"package_name"`
	GenerationLog   string     `json:"generation_log,omitempty" db:"generation_log"`
	ErrorMessage    string     `json:"error_message,omitempty" db:"error_message"`
	StartedAt       *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
}

// SDKTemplate represents a code generation template
type SDKTemplate struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Language     string    `json:"language" db:"language"`
	TemplateType string    `json:"template_type" db:"template_type"`
	Name         string    `json:"name" db:"name"`
	Content      string    `json:"content" db:"content"`
	Variables    []string  `json:"variables" db:"variables"`
	IsDefault    bool      `json:"is_default" db:"is_default"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// SDKDownload tracks SDK download events
type SDKDownload struct {
	ID           uuid.UUID `json:"id" db:"id"`
	GenerationID uuid.UUID `json:"generation_id" db:"generation_id"`
	TenantID     uuid.UUID `json:"tenant_id" db:"tenant_id"`
	DownloadType string    `json:"download_type" db:"download_type"`
	IPAddress    string    `json:"ip_address" db:"ip_address"`
	UserAgent    string    `json:"user_agent" db:"user_agent"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// Request types

type CreateSDKConfigRequest struct {
	Name             string      `json:"name" binding:"required"`
	Description      string      `json:"description"`
	PackagePrefix    string      `json:"package_prefix" binding:"required"`
	OrganizationName string      `json:"organization_name" binding:"required"`
	Branding         SDKBranding `json:"branding"`
	Languages        []string    `json:"languages" binding:"required"`
	APIBaseURL       string      `json:"api_base_url"`
	Features         SDKFeatures `json:"features"`
}

type GenerateSDKRequest struct {
	ConfigID  string   `json:"config_id" binding:"required"`
	Version   string   `json:"version" binding:"required"`
	Languages []string `json:"languages"` // If empty, use config languages
}

type SDKGenerationResult struct {
	Generation   *SDKGeneration `json:"generation"`
	DownloadURL  string         `json:"download_url,omitempty"`
	Instructions string         `json:"instructions,omitempty"`
}

// OpenAPI schema types for SDK generation
type OpenAPISpec struct {
	OpenAPI string                 `json:"openapi"`
	Info    OpenAPIInfo            `json:"info"`
	Servers []OpenAPIServer        `json:"servers,omitempty"`
	Paths   map[string]interface{} `json:"paths"`
	Components map[string]interface{} `json:"components,omitempty"`
}

type OpenAPIInfo struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version"`
}

type OpenAPIServer struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}
