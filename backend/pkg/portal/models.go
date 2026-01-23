package portal

import "time"

// PortalConfig represents the white-label portal configuration for a tenant
type PortalConfig struct {
	ID             string        `json:"id" db:"id"`
	TenantID       string        `json:"tenant_id" db:"tenant_id"`
	Name           string        `json:"name" db:"name"`
	Branding       *Branding     `json:"branding,omitempty"`
	BrandingJSON   string        `json:"-" db:"branding"`
	AllowedOrigins []string      `json:"allowed_origins"`
	OriginsJSON    string        `json:"-" db:"allowed_origins"`
	Features       *FeatureFlags `json:"features,omitempty"`
	FeaturesJSON   string        `json:"-" db:"features"`
	IsActive       bool          `json:"is_active" db:"is_active"`
	CreatedAt      time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at" db:"updated_at"`
}

// Branding defines custom branding for the embedded portal
type Branding struct {
	LogoURL         string `json:"logo_url,omitempty"`
	PrimaryColor    string `json:"primary_color,omitempty"`
	SecondaryColor  string `json:"secondary_color,omitempty"`
	BackgroundColor string `json:"background_color,omitempty"`
	FontFamily      string `json:"font_family,omitempty"`
	CustomCSS       string `json:"custom_css,omitempty"`
}

// FeatureFlags controls which portal features are visible
type FeatureFlags struct {
	ShowEndpoints   bool `json:"show_endpoints"`
	ShowDeliveries  bool `json:"show_deliveries"`
	ShowAnalytics   bool `json:"show_analytics"`
	ShowTestSender  bool `json:"show_test_sender"`
	AllowCreate     bool `json:"allow_create_endpoints"`
	AllowDelete     bool `json:"allow_delete_endpoints"`
	ShowSLA         bool `json:"show_sla"`
}

// EmbedToken represents a scoped access token for the embedded portal
type EmbedToken struct {
	ID          string    `json:"id" db:"id"`
	TenantID    string    `json:"tenant_id" db:"tenant_id"`
	PortalID    string    `json:"portal_id" db:"portal_id"`
	Token       string    `json:"token" db:"token"`
	Scopes      []string  `json:"scopes"`
	ScopesJSON  string    `json:"-" db:"scopes"`
	ExpiresAt   time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// PortalSession tracks an active embedded portal session
type PortalSession struct {
	ID         string    `json:"id" db:"id"`
	TokenID    string    `json:"token_id" db:"token_id"`
	TenantID   string    `json:"tenant_id" db:"tenant_id"`
	UserAgent  string    `json:"user_agent" db:"user_agent"`
	Origin     string    `json:"origin" db:"origin"`
	StartedAt  time.Time `json:"started_at" db:"started_at"`
	LastSeenAt time.Time `json:"last_seen_at" db:"last_seen_at"`
}

// CreatePortalRequest is the request DTO for creating a portal
type CreatePortalRequest struct {
	Name           string        `json:"name" binding:"required,min=1,max=255"`
	Branding       *Branding     `json:"branding,omitempty"`
	AllowedOrigins []string      `json:"allowed_origins" binding:"required,min=1"`
	Features       *FeatureFlags `json:"features,omitempty"`
}

// CreateTokenRequest is the request DTO for creating an embed token
type CreateTokenRequest struct {
	PortalID  string   `json:"portal_id" binding:"required"`
	Scopes    []string `json:"scopes" binding:"required,min=1"`
	TTLHours  int      `json:"ttl_hours" binding:"min=1,max=8760"`
}

// PortalScope constants
const (
	ScopeEndpointsRead   = "endpoints:read"
	ScopeEndpointsWrite  = "endpoints:write"
	ScopeDeliveriesRead  = "deliveries:read"
	ScopeDeliveriesRetry = "deliveries:retry"
	ScopeAnalyticsRead   = "analytics:read"
	ScopeTestSend        = "test:send"
	ScopeSLARead         = "sla:read"
)

// ValidScopes contains all recognized portal scopes
var ValidScopes = map[string]bool{
	ScopeEndpointsRead:   true,
	ScopeEndpointsWrite:  true,
	ScopeDeliveriesRead:  true,
	ScopeDeliveriesRetry: true,
	ScopeAnalyticsRead:   true,
	ScopeTestSend:        true,
	ScopeSLARead:         true,
}

// IsValidScope checks if a scope string is recognized
func IsValidScope(scope string) bool {
	return ValidScopes[scope]
}

// HasScope checks if a token has the required scope
func HasScope(token *EmbedToken, scope string) bool {
	for _, s := range token.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// EmbedSnippet returns the HTML/JS code to embed the portal
type EmbedSnippet struct {
	HTML      string `json:"html"`
	React     string `json:"react"`
	IFrame    string `json:"iframe"`
}

// PortalEndpointView is the customer-facing view of a webhook endpoint
type PortalEndpointView struct {
	ID          string    `json:"id" db:"id"`
	URL         string    `json:"url" db:"url"`
	Description string    `json:"description" db:"description"`
	EventTypes  []string  `json:"event_types"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// PortalDeliveryView is the delivery log view for portal users
type PortalDeliveryView struct {
	ID            string    `json:"id" db:"id"`
	EndpointID    string    `json:"endpoint_id" db:"endpoint_id"`
	EventType     string    `json:"event_type" db:"event_type"`
	StatusCode    int       `json:"status_code" db:"status_code"`
	Success       bool      `json:"success" db:"success"`
	Attempts      int       `json:"attempts" db:"attempts"`
	LastAttemptAt time.Time `json:"last_attempt_at" db:"last_attempt_at"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// EmbedConfig holds configuration for iframe/React embed
type EmbedConfig struct {
	PortalID       string   `json:"portal_id"`
	Theme          string   `json:"theme"`
	AllowedOrigins []string `json:"allowed_origins"`
	PrimaryColor   string   `json:"primary_color,omitempty"`
	FontFamily     string   `json:"font_family,omitempty"`
	CustomCSS      string   `json:"custom_css,omitempty"`
}

// PortalStats holds usage statistics for the portal
type PortalStats struct {
	TotalEndpoints       int     `json:"total_endpoints"`
	ActiveEndpoints      int     `json:"active_endpoints"`
	TotalDeliveries      int64   `json:"total_deliveries"`
	SuccessfulDeliveries int64   `json:"successful_deliveries"`
	FailedDeliveries     int64   `json:"failed_deliveries"`
	SuccessRate          float64 `json:"success_rate"`
	AvgLatencyMs         float64 `json:"avg_latency_ms"`
}

// UpdatePortalConfigRequest is the request DTO for updating portal configuration
type UpdatePortalConfigRequest struct {
	Name           string        `json:"name,omitempty"`
	Branding       *Branding     `json:"branding,omitempty"`
	AllowedOrigins []string      `json:"allowed_origins,omitempty"`
	Features       *FeatureFlags `json:"features,omitempty"`
	IsActive       *bool         `json:"is_active,omitempty"`
}

// DeliveryFilter holds filter parameters for delivery queries
type DeliveryFilter struct {
	EndpointID string `json:"endpoint_id"`
	Status     string `json:"status"`
	EventType  string `json:"event_type"`
}

// Pagination holds pagination parameters
type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}
