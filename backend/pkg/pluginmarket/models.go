package pluginmarket

import "time"

// PluginType represents the kind of plugin
type PluginType string

const (
	PluginTypeIntegration    PluginType = "integration"
	PluginTypeTransformation PluginType = "transformation"
	PluginTypeFilter         PluginType = "filter"
	PluginTypeRouter         PluginType = "router"
	PluginTypeNotifier       PluginType = "notifier"
)

// PluginStatus represents the lifecycle state of a plugin
type PluginStatus string

const (
	PluginStatusDraft     PluginStatus = "draft"
	PluginStatusReview    PluginStatus = "pending_review"
	PluginStatusPublished PluginStatus = "published"
	PluginStatusSuspended PluginStatus = "suspended"
	PluginStatusArchived  PluginStatus = "archived"
)

// PricingModel for marketplace listings
type PricingModel string

const (
	PricingFree         PricingModel = "free"
	PricingFreemium     PricingModel = "freemium"
	PricingPaid         PricingModel = "paid"
	PricingSubscription PricingModel = "subscription"
	PricingUsageBased   PricingModel = "usage_based"
)

// Plugin represents a marketplace plugin listing
type Plugin struct {
	ID            string       `json:"id" db:"id"`
	Name          string       `json:"name" db:"name"`
	Slug          string       `json:"slug" db:"slug"`
	Description   string       `json:"description" db:"description"`
	LongDesc      string       `json:"long_description" db:"long_description"`
	AuthorID      string       `json:"author_id" db:"author_id"`
	AuthorName    string       `json:"author_name" db:"author_name"`
	Type          PluginType   `json:"type" db:"type"`
	Status        PluginStatus `json:"status" db:"status"`
	Pricing       PricingModel `json:"pricing" db:"pricing"`
	PriceMonthly  float64      `json:"price_monthly,omitempty" db:"price_monthly"`
	Version       string       `json:"version" db:"version"`
	IconURL       string       `json:"icon_url,omitempty" db:"icon_url"`
	SourceURL     string       `json:"source_url,omitempty" db:"source_url"`
	DocURL        string       `json:"documentation_url,omitempty" db:"documentation_url"`
	Tags          []string     `json:"tags" db:"-"`
	Categories    []string     `json:"categories" db:"-"`
	Installs      int64        `json:"installs" db:"installs"`
	AvgRating     float64      `json:"avg_rating" db:"avg_rating"`
	ReviewCount   int          `json:"review_count" db:"review_count"`
	Verified      bool         `json:"verified" db:"verified"`
	Featured      bool         `json:"featured" db:"featured"`
	CreatedAt     time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at" db:"updated_at"`
	PublishedAt   *time.Time   `json:"published_at,omitempty" db:"published_at"`
}

// PluginVersion tracks versions of a plugin
type PluginVersion struct {
	ID           string    `json:"id" db:"id"`
	PluginID     string    `json:"plugin_id" db:"plugin_id"`
	Version      string    `json:"version" db:"version"`
	Changelog    string    `json:"changelog" db:"changelog"`
	MinPlatform  string    `json:"min_platform_version" db:"min_platform_version"`
	Checksum     string    `json:"checksum" db:"checksum"`
	Size         int64     `json:"size_bytes" db:"size_bytes"`
	Downloads    int64     `json:"downloads" db:"downloads"`
	IsLatest     bool      `json:"is_latest" db:"is_latest"`
	ReleasedAt   time.Time `json:"released_at" db:"released_at"`
}

// PluginInstallation tracks which tenants installed which plugins
type PluginInstallation struct {
	ID          string            `json:"id" db:"id"`
	TenantID    string            `json:"tenant_id" db:"tenant_id"`
	PluginID    string            `json:"plugin_id" db:"plugin_id"`
	VersionID   string            `json:"version_id" db:"version_id"`
	Status      string            `json:"status" db:"status"`
	Config      map[string]string `json:"config" db:"-"`
	InstalledAt time.Time         `json:"installed_at" db:"installed_at"`
	UpdatedAt   time.Time         `json:"updated_at" db:"updated_at"`
}

// PluginReview represents a user review
type PluginReview struct {
	ID        string    `json:"id" db:"id"`
	PluginID  string    `json:"plugin_id" db:"plugin_id"`
	TenantID  string    `json:"tenant_id" db:"tenant_id"`
	Rating    int       `json:"rating" db:"rating"`
	Title     string    `json:"title" db:"title"`
	Body      string    `json:"body" db:"body"`
	Helpful   int       `json:"helpful_count" db:"helpful_count"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// PluginHook defines the interface a plugin can hook into
type PluginHook struct {
	ID          string         `json:"id" db:"id"`
	PluginID    string         `json:"plugin_id" db:"plugin_id"`
	HookPoint   HookPoint      `json:"hook_point" db:"hook_point"`
	Priority    int            `json:"priority" db:"priority"`
	Config      map[string]any `json:"config" db:"-"`
}

// HookPoint defines where in the webhook lifecycle a plugin can execute
type HookPoint string

const (
	HookPreSend       HookPoint = "pre_send"
	HookPostSend      HookPoint = "post_send"
	HookOnFailure     HookPoint = "on_failure"
	HookOnSuccess     HookPoint = "on_success"
	HookTransform     HookPoint = "transform"
	HookValidate      HookPoint = "validate"
	HookRoute         HookPoint = "route"
	HookAuthenticate  HookPoint = "authenticate"
)

// PluginExecResult captures the outcome of a plugin execution
type PluginExecResult struct {
	PluginID    string         `json:"plugin_id"`
	HookPoint   HookPoint      `json:"hook_point"`
	Success     bool           `json:"success"`
	DurationMs  int64          `json:"duration_ms"`
	Output      map[string]any `json:"output,omitempty"`
	Error       string         `json:"error,omitempty"`
	Logs        []string       `json:"logs,omitempty"`
}

// PluginSearchParams for querying the marketplace
type PluginSearchParams struct {
	Query      string       `json:"query"`
	Type       PluginType   `json:"type,omitempty"`
	Category   string       `json:"category,omitempty"`
	Pricing    PricingModel `json:"pricing,omitempty"`
	SortBy     string       `json:"sort_by,omitempty"`
	Verified   *bool        `json:"verified,omitempty"`
	Featured   *bool        `json:"featured,omitempty"`
	Page       int          `json:"page"`
	PageSize   int          `json:"page_size"`
}

// PluginSearchResult wraps paginated search results
type PluginSearchResult struct {
	Plugins    []Plugin `json:"plugins"`
	Total      int      `json:"total"`
	Page       int      `json:"page"`
	PageSize   int      `json:"page_size"`
	TotalPages int      `json:"total_pages"`
}

// PluginStats captures marketplace analytics
type PluginStats struct {
	TotalPlugins      int            `json:"total_plugins"`
	TotalInstalls     int64          `json:"total_installs"`
	ActivePlugins     int            `json:"active_plugins"`
	TopCategories     map[string]int `json:"top_categories"`
	AvgRating         float64        `json:"avg_rating"`
	PublishedLast30d  int            `json:"published_last_30d"`
}

// CreatePluginRequest for new plugin submission
type CreatePluginRequest struct {
	Name         string       `json:"name" binding:"required"`
	Description  string       `json:"description" binding:"required"`
	LongDesc     string       `json:"long_description"`
	Type         PluginType   `json:"type" binding:"required"`
	Pricing      PricingModel `json:"pricing" binding:"required"`
	PriceMonthly float64      `json:"price_monthly,omitempty"`
	Version      string       `json:"version" binding:"required"`
	IconURL      string       `json:"icon_url,omitempty"`
	SourceURL    string       `json:"source_url,omitempty"`
	DocURL       string       `json:"documentation_url,omitempty"`
	Tags         []string     `json:"tags,omitempty"`
	Categories   []string     `json:"categories,omitempty"`
}

// InstallPluginRequest for installing a plugin
type InstallPluginRequest struct {
	PluginID  string            `json:"plugin_id" binding:"required"`
	VersionID string            `json:"version_id,omitempty"`
	Config    map[string]string `json:"config,omitempty"`
}

// CreateReviewRequest for submitting a review
type CreateReviewRequest struct {
	Rating int    `json:"rating" binding:"required,min=1,max=5"`
	Title  string `json:"title" binding:"required"`
	Body   string `json:"body"`
}
