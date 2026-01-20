package embed

import (
	"time"
)

// EmbedToken represents an embeddable analytics token
type EmbedToken struct {
	ID             string            `json:"id" db:"id"`
	TenantID       string            `json:"tenant_id" db:"tenant_id"`
	Name           string            `json:"name" db:"name"`
	Token          string            `json:"token" db:"token"`
	Permissions    []Permission      `json:"permissions" db:"permissions"`
	Scopes         EmbedScopes       `json:"scopes" db:"scopes"`
	Theme          *ThemeConfig      `json:"theme,omitempty" db:"theme"`
	ExpiresAt      *time.Time        `json:"expires_at,omitempty" db:"expires_at"`
	AllowedOrigins []string          `json:"allowed_origins" db:"allowed_origins"`
	Metadata       map[string]string `json:"metadata,omitempty" db:"metadata"`
	IsActive       bool              `json:"is_active" db:"is_active"`
	CreatedAt      time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at" db:"updated_at"`
}

// Permission defines what an embed token can access
type Permission string

const (
	PermissionReadDeliveries Permission = "deliveries:read"
	PermissionReadEndpoints  Permission = "endpoints:read"
	PermissionReadMetrics    Permission = "metrics:read"
	PermissionReadEvents     Permission = "events:read"
	PermissionReadActivity   Permission = "activity:read"
	PermissionReadErrors     Permission = "errors:read"
)

// EmbedScopes defines the scope of data accessible
type EmbedScopes struct {
	EndpointIDs []string `json:"endpoint_ids,omitempty"`
	EventTypes  []string `json:"event_types,omitempty"`
	CustomerID  string   `json:"customer_id,omitempty"`
	TimeRange   string   `json:"time_range,omitempty"` // e.g., "24h", "7d", "30d"
}

// ThemeConfig defines the visual theme for embedded components
type ThemeConfig struct {
	Mode            string            `json:"mode"` // light, dark, auto
	PrimaryColor    string            `json:"primary_color"`
	BackgroundColor string            `json:"background_color"`
	TextColor       string            `json:"text_color"`
	BorderRadius    string            `json:"border_radius"`
	FontFamily      string            `json:"font_family"`
	CustomCSS       string            `json:"custom_css,omitempty"`
	Variables       map[string]string `json:"variables,omitempty"`
}

// EmbedComponent represents an embeddable component type
type EmbedComponent string

const (
	ComponentDeliveryStats    EmbedComponent = "delivery_stats"
	ComponentActivityFeed     EmbedComponent = "activity_feed"
	ComponentSuccessRateChart EmbedComponent = "success_rate_chart"
	ComponentLatencyChart     EmbedComponent = "latency_chart"
	ComponentEndpointList     EmbedComponent = "endpoint_list"
	ComponentEventLog         EmbedComponent = "event_log"
	ComponentErrorSummary     EmbedComponent = "error_summary"
	ComponentVolumeChart      EmbedComponent = "volume_chart"
)

// ComponentConfig defines configuration for an embedded component
type ComponentConfig struct {
	Component   EmbedComponent         `json:"component"`
	Width       string                 `json:"width,omitempty"`
	Height      string                 `json:"height,omitempty"`
	Title       string                 `json:"title,omitempty"`
	ShowHeader  bool                   `json:"show_header"`
	RefreshRate int                    `json:"refresh_rate,omitempty"` // seconds
	Options     map[string]interface{} `json:"options,omitempty"`
}

// EmbedSession represents an active embed session
type EmbedSession struct {
	ID        string    `json:"id" db:"id"`
	TokenID   string    `json:"token_id" db:"token_id"`
	TenantID  string    `json:"tenant_id" db:"tenant_id"`
	Origin    string    `json:"origin" db:"origin"`
	UserAgent string    `json:"user_agent" db:"user_agent"`
	IP        string    `json:"ip" db:"ip"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	LastSeen  time.Time `json:"last_seen" db:"last_seen"`
}

// DeliveryStats represents delivery statistics
type DeliveryStats struct {
	TotalDeliveries int64     `json:"total_deliveries"`
	Successful      int64     `json:"successful"`
	Failed          int64     `json:"failed"`
	Pending         int64     `json:"pending"`
	SuccessRate     float64   `json:"success_rate"`
	AvgLatencyMs    int       `json:"avg_latency_ms"`
	Period          string    `json:"period"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ActivityItem represents an activity feed item
type ActivityItem struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Message    string                 `json:"message"`
	Severity   string                 `json:"severity"` // info, warning, error
	EndpointID string                 `json:"endpoint_id,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
}

// ChartDataPoint represents a data point for charts
type ChartDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Label     string    `json:"label,omitempty"`
}

// ChartSeries represents a series of chart data
type ChartSeries struct {
	Name  string           `json:"name"`
	Color string           `json:"color,omitempty"`
	Data  []ChartDataPoint `json:"data"`
}

// ChartData represents chart data for embedded charts
type ChartData struct {
	Title  string        `json:"title"`
	Type   string        `json:"type"` // line, bar, area, pie
	Series []ChartSeries `json:"series"`
	XAxis  string        `json:"x_axis"`
	YAxis  string        `json:"y_axis"`
	Period string        `json:"period"`
}

// ErrorSummary represents error summary data
type ErrorSummary struct {
	TotalErrors int64            `json:"total_errors"`
	ByCategory  map[string]int64 `json:"by_category"`
	ByEndpoint  map[string]int64 `json:"by_endpoint"`
	TopErrors   []ErrorDetail    `json:"top_errors"`
	Period      string           `json:"period"`
}

// ErrorDetail represents error details
type ErrorDetail struct {
	Message    string    `json:"message"`
	Count      int64     `json:"count"`
	LastSeen   time.Time `json:"last_seen"`
	EndpointID string    `json:"endpoint_id,omitempty"`
}

// CreateTokenRequest represents a request to create an embed token
type CreateTokenRequest struct {
	Name           string            `json:"name" binding:"required"`
	Permissions    []Permission      `json:"permissions" binding:"required,min=1"`
	Scopes         EmbedScopes       `json:"scopes"`
	Theme          *ThemeConfig      `json:"theme,omitempty"`
	ExpiresIn      string            `json:"expires_in,omitempty"` // e.g., "30d", "90d"
	AllowedOrigins []string          `json:"allowed_origins"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// UpdateTokenRequest represents a request to update an embed token
type UpdateTokenRequest struct {
	Name           *string           `json:"name,omitempty"`
	Permissions    []Permission      `json:"permissions,omitempty"`
	Scopes         *EmbedScopes      `json:"scopes,omitempty"`
	Theme          *ThemeConfig      `json:"theme,omitempty"`
	AllowedOrigins []string          `json:"allowed_origins,omitempty"`
	IsActive       *bool             `json:"is_active,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// AllPermissions returns all available permissions
func AllPermissions() []Permission {
	return []Permission{
		PermissionReadDeliveries,
		PermissionReadEndpoints,
		PermissionReadMetrics,
		PermissionReadEvents,
		PermissionReadActivity,
		PermissionReadErrors,
	}
}

// AllComponents returns all available components
func AllComponents() []EmbedComponent {
	return []EmbedComponent{
		ComponentDeliveryStats,
		ComponentActivityFeed,
		ComponentSuccessRateChart,
		ComponentLatencyChart,
		ComponentEndpointList,
		ComponentEventLog,
		ComponentErrorSummary,
		ComponentVolumeChart,
	}
}

// DefaultTheme returns the default theme configuration
func DefaultTheme() *ThemeConfig {
	return &ThemeConfig{
		Mode:            "light",
		PrimaryColor:    "#6366f1",
		BackgroundColor: "#ffffff",
		TextColor:       "#1f2937",
		BorderRadius:    "8px",
		FontFamily:      "Inter, system-ui, sans-serif",
	}
}
