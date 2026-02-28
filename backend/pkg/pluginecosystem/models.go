package pluginecosystem

import "time"

// PluginType categorizes plugins by their function.
type PluginType string

const (
	PluginTransform  PluginType = "transformation"
	PluginFilter     PluginType = "filter"
	PluginEnrichment PluginType = "enrichment"
	PluginAuth       PluginType = "authentication"
	PluginNotifier   PluginType = "notification"
	PluginConnector  PluginType = "connector"
	PluginAnalytics  PluginType = "analytics"
)

// PluginStatus tracks the lifecycle state of a plugin.
type PluginStatus string

const (
	PluginDraft     PluginStatus = "draft"
	PluginInReview  PluginStatus = "in_review"
	PluginApproved  PluginStatus = "approved"
	PluginPublished PluginStatus = "published"
	PluginSuspended PluginStatus = "suspended"
	PluginArchived  PluginStatus = "archived"
)

// PricingModel defines how a plugin is priced.
type PricingModel string

const (
	PricingFree         PricingModel = "free"
	PricingOneTime      PricingModel = "one_time"
	PricingSubscription PricingModel = "subscription"
	PricingUsageBased   PricingModel = "usage_based"
)

// Plugin represents a marketplace plugin.
type Plugin struct {
	ID            string          `json:"id" db:"id"`
	DeveloperID   string          `json:"developer_id" db:"developer_id"`
	Name          string          `json:"name" db:"name"`
	Slug          string          `json:"slug" db:"slug"`
	Description   string          `json:"description" db:"description"`
	Type          PluginType      `json:"type" db:"type"`
	Status        PluginStatus    `json:"status" db:"status"`
	Version       string          `json:"version" db:"version"`
	IconURL       string          `json:"icon_url,omitempty" db:"icon_url"`
	SourceURL     string          `json:"source_url,omitempty" db:"source_url"`
	Pricing       PricingModel    `json:"pricing" db:"pricing"`
	PriceAmtCents int             `json:"price_amount_cents,omitempty" db:"price_amount_cents"`
	Installs      int             `json:"installs" db:"installs"`
	Rating        float64         `json:"rating" db:"rating"`
	RatingCount   int             `json:"rating_count" db:"rating_count"`
	Tags          []string        `json:"tags" db:"-"`
	Manifest      *PluginManifest `json:"manifest,omitempty"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
}

// PluginManifest defines the plugin's capabilities and requirements.
type PluginManifest struct {
	Runtime      string                 `json:"runtime"` // javascript, wasm
	EntryPoint   string                 `json:"entry_point"`
	Permissions  []string               `json:"permissions"` // read_payload, write_payload, http_outbound
	EventTypes   []string               `json:"event_types"`
	ConfigSchema map[string]interface{} `json:"config_schema,omitempty"`
}

// PluginInstallation tracks a plugin installed by a tenant.
type PluginInstallation struct {
	ID          string                 `json:"id" db:"id"`
	TenantID    string                 `json:"tenant_id" db:"tenant_id"`
	PluginID    string                 `json:"plugin_id" db:"plugin_id"`
	Version     string                 `json:"version" db:"version"`
	Config      map[string]interface{} `json:"config"`
	Enabled     bool                   `json:"enabled" db:"enabled"`
	InstalledAt time.Time              `json:"installed_at" db:"installed_at"`
}

// PluginReview is a user review of a plugin.
type PluginReview struct {
	ID        string    `json:"id" db:"id"`
	PluginID  string    `json:"plugin_id" db:"plugin_id"`
	TenantID  string    `json:"tenant_id" db:"tenant_id"`
	Rating    int       `json:"rating" db:"rating"`
	Comment   string    `json:"comment" db:"comment"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// DeveloperProfile represents a plugin developer.
type DeveloperProfile struct {
	ID            string    `json:"id" db:"id"`
	TenantID      string    `json:"tenant_id" db:"tenant_id"`
	DisplayName   string    `json:"display_name" db:"display_name"`
	Website       string    `json:"website,omitempty" db:"website"`
	Verified      bool      `json:"verified" db:"verified"`
	PluginCount   int       `json:"plugin_count"`
	TotalInstalls int       `json:"total_installs"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// PublishPluginRequest is the API request for publishing a plugin.
type PublishPluginRequest struct {
	Name          string          `json:"name" binding:"required"`
	Description   string          `json:"description" binding:"required"`
	Type          PluginType      `json:"type" binding:"required"`
	Version       string          `json:"version" binding:"required"`
	Pricing       PricingModel    `json:"pricing"`
	PriceAmtCents int             `json:"price_amount_cents,omitempty"`
	Tags          []string        `json:"tags"`
	Manifest      *PluginManifest `json:"manifest" binding:"required"`
}

// InstallPluginRequest is the API request for installing a plugin.
type InstallPluginRequest struct {
	PluginID string                 `json:"plugin_id" binding:"required"`
	Config   map[string]interface{} `json:"config,omitempty"`
}

// SearchPluginsRequest is the API request for searching plugins.
type SearchPluginsRequest struct {
	Query   string       `json:"query"`
	Type    PluginType   `json:"type,omitempty"`
	Pricing PricingModel `json:"pricing,omitempty"`
	SortBy  string       `json:"sort_by"` // installs, rating, newest
	Limit   int          `json:"limit"`
	Offset  int          `json:"offset"`
}
