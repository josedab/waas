package analyticsembed

import "time"

// WidgetType constants
const (
	WidgetTypeDeliveryChart   = "delivery_chart"
	WidgetTypeErrorBreakdown  = "error_breakdown"
	WidgetTypeLatencyHeatmap  = "latency_heatmap"
	WidgetTypeEndpointHealth  = "endpoint_health"
	WidgetTypeEventTimeline   = "event_timeline"
	WidgetTypeRealtimeCounter = "realtime_counter"
)

// WidgetConfig defines an embeddable analytics widget
type WidgetConfig struct {
	ID                string    `json:"id" db:"id"`
	TenantID          string    `json:"tenant_id" db:"tenant_id"`
	Name              string    `json:"name" db:"name"`
	WidgetType        string    `json:"widget_type" db:"widget_type"`
	DataSource        string    `json:"data_source" db:"data_source"`
	TimeRange         string    `json:"time_range" db:"time_range"`
	RefreshIntervalSec int     `json:"refresh_interval_sec" db:"refresh_interval_sec"`
	CustomCSS         string    `json:"custom_css,omitempty" db:"custom_css"`
	IsPublic          bool      `json:"is_public" db:"is_public"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// EmbedToken represents an authentication token for embedded widgets
type EmbedToken struct {
	ID             string    `json:"id" db:"id"`
	TenantID       string    `json:"tenant_id" db:"tenant_id"`
	WidgetID       string    `json:"widget_id" db:"widget_id"`
	Token          string    `json:"token" db:"token"`
	Scopes         []string  `json:"scopes"`
	AllowedOrigins []string  `json:"allowed_origins"`
	ExpiresAt      time.Time `json:"expires_at" db:"expires_at"`
	IsActive       bool      `json:"is_active" db:"is_active"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// ThemeConfig defines the visual theme for embedded widgets
type ThemeConfig struct {
	ID              string    `json:"id" db:"id"`
	TenantID        string    `json:"tenant_id" db:"tenant_id"`
	PrimaryColor    string    `json:"primary_color" db:"primary_color"`
	SecondaryColor  string    `json:"secondary_color" db:"secondary_color"`
	BackgroundColor string    `json:"background_color" db:"background_color"`
	TextColor       string    `json:"text_color" db:"text_color"`
	FontFamily      string    `json:"font_family" db:"font_family"`
	BorderRadius    string    `json:"border_radius" db:"border_radius"`
	CustomCSS       string    `json:"custom_css,omitempty" db:"custom_css"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// EmbedSnippet contains generated code snippets for embedding a widget
type EmbedSnippet struct {
	WidgetID   string `json:"widget_id"`
	WidgetType string `json:"widget_type"`
	HTML       string `json:"html"`
	React      string `json:"react"`
	JavaScript string `json:"javascript"`
	IframeURL  string `json:"iframe_url"`
}

// WidgetData contains the data payload for a widget
type WidgetData struct {
	WidgetID    string      `json:"widget_id"`
	WidgetType  string      `json:"widget_type"`
	Data        interface{} `json:"data"`
	GeneratedAt time.Time   `json:"generated_at"`
}

// CreateWidgetRequest is the request DTO for creating a widget
type CreateWidgetRequest struct {
	Name               string `json:"name" binding:"required,min=1,max=255"`
	WidgetType         string `json:"widget_type" binding:"required"`
	DataSource         string `json:"data_source" binding:"required"`
	TimeRange          string `json:"time_range" binding:"required"`
	RefreshIntervalSec int    `json:"refresh_interval_sec" binding:"min=0"`
	CustomCSS          string `json:"custom_css,omitempty"`
	IsPublic           bool   `json:"is_public"`
}

// CreateEmbedTokenRequest is the request DTO for generating an embed token
type CreateEmbedTokenRequest struct {
	WidgetID       string   `json:"widget_id" binding:"required"`
	Scopes         []string `json:"scopes" binding:"required,min=1"`
	AllowedOrigins []string `json:"allowed_origins" binding:"required,min=1"`
	ExpiresInHours int      `json:"expires_in_hours" binding:"required,min=1"`
}

// UpdateThemeRequest is the request DTO for updating the theme
type UpdateThemeRequest struct {
	PrimaryColor    string `json:"primary_color"`
	SecondaryColor  string `json:"secondary_color"`
	BackgroundColor string `json:"background_color"`
	TextColor       string `json:"text_color"`
	FontFamily      string `json:"font_family"`
	BorderRadius    string `json:"border_radius"`
	CustomCSS       string `json:"custom_css,omitempty"`
}
