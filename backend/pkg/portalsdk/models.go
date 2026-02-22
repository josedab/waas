package portalsdk

import (
	"encoding/json"
	"time"
)

// Component type constants
const (
	ComponentEndpointManager = "endpoint_manager"
	ComponentEventBrowser    = "event_browser"
	ComponentDeliveryLog     = "delivery_log"
	ComponentMetricsDash     = "metrics_dashboard"
	ComponentAlertConfig     = "alert_config"
	ComponentAPIExplorer     = "api_explorer"
	ComponentLogViewer       = "log_viewer"
	ComponentSubscriptions   = "subscriptions"
)

// Session status constants
const (
	SessionStatusActive  = "active"
	SessionStatusExpired = "expired"
	SessionStatusRevoked = "revoked"
)

// PortalConfig defines the embeddable portal configuration for a tenant.
type PortalConfig struct {
	ID             string         `json:"id" db:"id"`
	TenantID       string         `json:"tenant_id" db:"tenant_id"`
	Name           string         `json:"name" db:"name"`
	AllowedOrigins []string       `json:"allowed_origins"`
	Components     []string       `json:"components"`
	Theme          ThemeConfig    `json:"theme"`
	Features       FeatureConfig  `json:"features"`
	Branding       BrandingConfig `json:"branding"`
	CustomCSS      string         `json:"custom_css,omitempty" db:"custom_css"`
	IsActive       bool           `json:"is_active" db:"is_active"`
	CreatedAt      time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at" db:"updated_at"`
}

// ThemeConfig defines visual theming for the embedded portal.
type ThemeConfig struct {
	PrimaryColor    string `json:"primary_color" yaml:"primary_color"`
	SecondaryColor  string `json:"secondary_color" yaml:"secondary_color"`
	BackgroundColor string `json:"background_color" yaml:"background_color"`
	TextColor       string `json:"text_color" yaml:"text_color"`
	FontFamily      string `json:"font_family" yaml:"font_family"`
	BorderRadius    string `json:"border_radius" yaml:"border_radius"`
	DarkMode        bool   `json:"dark_mode" yaml:"dark_mode"`
}

// FeatureConfig controls which features are available in the portal.
type FeatureConfig struct {
	EndpointManagement bool `json:"endpoint_management"`
	EventBrowsing      bool `json:"event_browsing"`
	DeliveryLogs       bool `json:"delivery_logs"`
	MetricsDashboard   bool `json:"metrics_dashboard"`
	AlertConfiguration bool `json:"alert_configuration"`
	APIExplorer        bool `json:"api_explorer"`
	LogViewer          bool `json:"log_viewer"`
	Subscriptions      bool `json:"subscriptions"`
}

// BrandingConfig defines branding options for white-label embedding.
type BrandingConfig struct {
	LogoURL     string `json:"logo_url,omitempty"`
	FaviconURL  string `json:"favicon_url,omitempty"`
	CompanyName string `json:"company_name,omitempty"`
	SupportURL  string `json:"support_url,omitempty"`
	DocsURL     string `json:"docs_url,omitempty"`
}

// PortalSession represents an authenticated session for an embedded portal.
type PortalSession struct {
	ID           string          `json:"id" db:"id"`
	TenantID     string          `json:"tenant_id" db:"tenant_id"`
	ConfigID     string          `json:"config_id" db:"config_id"`
	CustomerID   string          `json:"customer_id" db:"customer_id"`
	Token        string          `json:"token" db:"token"`
	Permissions  []string        `json:"permissions"`
	Scopes       json.RawMessage `json:"scopes,omitempty" db:"scopes"`
	Origin       string          `json:"origin,omitempty" db:"origin"`
	UserAgent    string          `json:"user_agent,omitempty" db:"user_agent"`
	IPAddress    string          `json:"ip_address,omitempty" db:"ip_address"`
	Status       string          `json:"status" db:"status"`
	ExpiresAt    time.Time       `json:"expires_at" db:"expires_at"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
	LastAccessAt time.Time       `json:"last_access_at" db:"last_access_at"`
}

// SDKBundle represents a generated JavaScript SDK bundle.
type SDKBundle struct {
	ID        string    `json:"id" db:"id"`
	TenantID  string    `json:"tenant_id" db:"tenant_id"`
	ConfigID  string    `json:"config_id" db:"config_id"`
	Version   string    `json:"version" db:"version"`
	Framework string    `json:"framework" db:"framework"`
	BundleURL string    `json:"bundle_url" db:"bundle_url"`
	Checksum  string    `json:"checksum" db:"checksum"`
	SizeBytes int64     `json:"size_bytes" db:"size_bytes"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// PortalUsageStats tracks portal usage metrics.
type PortalUsageStats struct {
	ConfigID        string           `json:"config_id"`
	ActiveSessions  int              `json:"active_sessions"`
	TotalSessions   int64            `json:"total_sessions"`
	UniqueCustomers int64            `json:"unique_customers"`
	ActionsPerDay   int64            `json:"actions_per_day"`
	TopComponents   []ComponentUsage `json:"top_components"`
}

// ComponentUsage tracks usage of individual portal components.
type ComponentUsage struct {
	Component string `json:"component"`
	Views     int64  `json:"views"`
	Actions   int64  `json:"actions"`
}

// Request DTOs

type CreatePortalConfigRequest struct {
	Name           string          `json:"name" binding:"required"`
	AllowedOrigins []string        `json:"allowed_origins" binding:"required"`
	Components     []string        `json:"components,omitempty"`
	Theme          *ThemeConfig    `json:"theme,omitempty"`
	Features       *FeatureConfig  `json:"features,omitempty"`
	Branding       *BrandingConfig `json:"branding,omitempty"`
	CustomCSS      string          `json:"custom_css,omitempty"`
}

type UpdatePortalConfigRequest struct {
	Name           string          `json:"name,omitempty"`
	AllowedOrigins []string        `json:"allowed_origins,omitempty"`
	Components     []string        `json:"components,omitempty"`
	Theme          *ThemeConfig    `json:"theme,omitempty"`
	Features       *FeatureConfig  `json:"features,omitempty"`
	Branding       *BrandingConfig `json:"branding,omitempty"`
	CustomCSS      string          `json:"custom_css,omitempty"`
	IsActive       *bool           `json:"is_active,omitempty"`
}

type CreateSessionRequest struct {
	ConfigID    string   `json:"config_id" binding:"required"`
	CustomerID  string   `json:"customer_id" binding:"required"`
	Permissions []string `json:"permissions,omitempty"`
	ExpiresIn   string   `json:"expires_in,omitempty"`
}

type GenerateSDKRequest struct {
	ConfigID  string `json:"config_id" binding:"required"`
	Framework string `json:"framework" binding:"required"`
}
