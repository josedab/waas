// Package marketplace provides a self-service connector marketplace
package marketplace

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
)

var (
	ErrListingNotFound     = errors.New("listing not found")
	ErrAlreadyInstalled    = errors.New("listing already installed")
	ErrInvalidSlug         = errors.New("invalid slug format")
	ErrAlreadyReviewed     = errors.New("already reviewed this listing")
	ErrNotInstalled        = errors.New("must install before reviewing")
	ErrVersionNotFound     = errors.New("version not found")
)

// ListingType represents the type of marketplace listing
type ListingType string

const (
	ListingTypeConnector      ListingType = "connector"
	ListingTypeTransformation ListingType = "transformation"
	ListingTypeTemplate       ListingType = "template"
	ListingTypeIntegration    ListingType = "integration"
)

// PricingModel represents pricing options
type PricingModel string

const (
	PricingFree        PricingModel = "free"
	PricingFreemium    PricingModel = "freemium"
	PricingPaid        PricingModel = "paid"
	PricingSubscription PricingModel = "subscription"
	PricingUsageBased  PricingModel = "usage_based"
)

// ListingStatus represents listing lifecycle status
type ListingStatus string

const (
	StatusDraft         ListingStatus = "draft"
	StatusPendingReview ListingStatus = "pending_review"
	StatusPublished     ListingStatus = "published"
	StatusRejected      ListingStatus = "rejected"
	StatusSuspended     ListingStatus = "suspended"
	StatusArchived      ListingStatus = "archived"
)

// Listing represents a marketplace listing
type Listing struct {
	ID                 string       `json:"id"`
	PublisherID        string       `json:"publisher_id"`
	PublisherName      string       `json:"publisher_name"`
	PublisherVerified  bool         `json:"publisher_verified"`
	Name               string       `json:"name"`
	Slug               string       `json:"slug"`
	ShortDescription   string       `json:"short_description"`
	FullDescription    string       `json:"full_description,omitempty"`
	Category           string       `json:"category"`
	Subcategory        string       `json:"subcategory,omitempty"`
	Tags               []string     `json:"tags,omitempty"`
	ListingType        ListingType  `json:"listing_type"`
	IconURL            string       `json:"icon_url,omitempty"`
	BannerURL          string       `json:"banner_url,omitempty"`
	Screenshots        []string     `json:"screenshots,omitempty"`
	Version            string       `json:"version"`
	MinPlatformVersion string       `json:"min_platform_version,omitempty"`
	DocumentationURL   string       `json:"documentation_url,omitempty"`
	SourceURL          string       `json:"source_url,omitempty"`
	PricingModel       PricingModel `json:"pricing_model"`
	PriceCents         int          `json:"price_cents"`
	Currency           string       `json:"currency"`
	InstallCount       int          `json:"install_count"`
	ActiveInstalls     int          `json:"active_installs"`
	Status             ListingStatus `json:"status"`
	ReviewNotes        string       `json:"review_notes,omitempty"`
	PublishedAt        *time.Time   `json:"published_at,omitempty"`
	CreatedAt          time.Time    `json:"created_at"`
	UpdatedAt          time.Time    `json:"updated_at"`

	// Computed fields
	AverageRating float64 `json:"average_rating,omitempty"`
	TotalReviews  int     `json:"total_reviews,omitempty"`
}

// ListingVersion represents a release version
type ListingVersion struct {
	ID                 string     `json:"id"`
	ListingID          string     `json:"listing_id"`
	Version            string     `json:"version"`
	ReleaseNotes       string     `json:"release_notes,omitempty"`
	ArtifactType       string     `json:"artifact_type,omitempty"`
	ArtifactURL        string     `json:"artifact_url,omitempty"`
	ArtifactHash       string     `json:"artifact_hash,omitempty"`
	ArtifactSizeBytes  int        `json:"artifact_size_bytes,omitempty"`
	MinPlatformVersion string     `json:"min_platform_version,omitempty"`
	MaxPlatformVersion string     `json:"max_platform_version,omitempty"`
	BreakingChanges    []string   `json:"breaking_changes,omitempty"`
	Status             string     `json:"status"`
	CreatedAt          time.Time  `json:"created_at"`
}

// Installation represents a user's installation of a listing
type Installation struct {
	ID               string                 `json:"id"`
	TenantID         string                 `json:"tenant_id"`
	ListingID        string                 `json:"listing_id"`
	VersionID        *string                `json:"version_id,omitempty"`
	InstalledVersion string                 `json:"installed_version"`
	Config           map[string]interface{} `json:"config,omitempty"`
	Status           string                 `json:"status"`
	LastUsedAt       *time.Time             `json:"last_used_at,omitempty"`
	UsageCount       int                    `json:"usage_count"`
	InstalledAt      time.Time              `json:"installed_at"`
	UpdatedAt        time.Time              `json:"updated_at"`

	// Joined data
	Listing *Listing `json:"listing,omitempty"`
}

// Review represents a user review
type Review struct {
	ID               string     `json:"id"`
	ListingID        string     `json:"listing_id"`
	ReviewerID       string     `json:"reviewer_id"`
	ReviewerName     string     `json:"reviewer_name,omitempty"`
	Rating           int        `json:"rating"`
	Title            string     `json:"title,omitempty"`
	Body             string     `json:"body,omitempty"`
	VerifiedPurchase bool       `json:"verified_purchase"`
	HelpfulCount     int        `json:"helpful_count"`
	Status           string     `json:"status"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	
	// Response from publisher
	Response *ReviewResponse `json:"response,omitempty"`
}

// ReviewResponse represents a publisher's response to a review
type ReviewResponse struct {
	ID        string    `json:"id"`
	ReviewID  string    `json:"review_id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RatingsSummary represents aggregated ratings
type RatingsSummary struct {
	ListingID     string    `json:"listing_id"`
	TotalReviews  int       `json:"total_reviews"`
	AverageRating float64   `json:"average_rating"`
	Rating1       int       `json:"rating_1"`
	Rating2       int       `json:"rating_2"`
	Rating3       int       `json:"rating_3"`
	Rating4       int       `json:"rating_4"`
	Rating5       int       `json:"rating_5"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// FeaturedListing represents a featured/promoted listing
type FeaturedListing struct {
	ID           string     `json:"id"`
	ListingID    string     `json:"listing_id"`
	FeatureType  string     `json:"feature_type"`
	Category     string     `json:"category,omitempty"`
	DisplayOrder int        `json:"display_order"`
	StartDate    time.Time  `json:"start_date"`
	EndDate      *time.Time `json:"end_date,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`

	Listing *Listing `json:"listing,omitempty"`
}

// SearchFilters represents marketplace search filters
type SearchFilters struct {
	Query       string       `json:"query,omitempty"`
	Category    string       `json:"category,omitempty"`
	ListingType ListingType  `json:"listing_type,omitempty"`
	PricingModel PricingModel `json:"pricing_model,omitempty"`
	MinRating   float64      `json:"min_rating,omitempty"`
	Tags        []string     `json:"tags,omitempty"`
	SortBy      string       `json:"sort_by,omitempty"` // popular, recent, rating, name
	SortOrder   string       `json:"sort_order,omitempty"` // asc, desc
	Limit       int          `json:"limit,omitempty"`
	Offset      int          `json:"offset,omitempty"`
}

// SearchResult represents marketplace search results
type SearchResult struct {
	Listings   []Listing `json:"listings"`
	TotalCount int       `json:"total_count"`
	Filters    *SearchFilters `json:"filters"`
}

// Service provides marketplace operations
type Service struct {
	slugRegex *regexp.Regexp
}

// NewService creates a new marketplace service
func NewService() *Service {
	return &Service{
		slugRegex: regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`),
	}
}

// GenerateSlug generates a URL-safe slug from a name
func (s *Service) GenerateSlug(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)
	// Replace spaces and underscores with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	// Remove special characters
	re := regexp.MustCompile(`[^a-z0-9-]`)
	slug = re.ReplaceAllString(slug, "")
	// Remove multiple consecutive hyphens
	re = regexp.MustCompile(`-+`)
	slug = re.ReplaceAllString(slug, "-")
	// Trim hyphens from ends
	slug = strings.Trim(slug, "-")
	return slug
}

// ValidateSlug checks if a slug is valid
func (s *Service) ValidateSlug(slug string) error {
	if len(slug) < 3 || len(slug) > 100 {
		return ErrInvalidSlug
	}
	if !s.slugRegex.MatchString(slug) {
		return ErrInvalidSlug
	}
	return nil
}

// ValidateListing validates listing data
func (s *Service) ValidateListing(l *Listing) []string {
	var errors []string

	if l.Name == "" {
		errors = append(errors, "name is required")
	}
	if len(l.Name) > 255 {
		errors = append(errors, "name must be 255 characters or less")
	}
	if l.ShortDescription == "" {
		errors = append(errors, "short_description is required")
	}
	if len(l.ShortDescription) > 500 {
		errors = append(errors, "short_description must be 500 characters or less")
	}
	if l.Category == "" {
		errors = append(errors, "category is required")
	}
	if l.ListingType == "" {
		errors = append(errors, "listing_type is required")
	}
	if l.Version == "" {
		errors = append(errors, "version is required")
	}

	return errors
}

// CanInstall checks if a tenant can install a listing
func (s *Service) CanInstall(ctx context.Context, listing *Listing, tenantID string) (bool, string) {
	if listing.Status != StatusPublished {
		return false, "listing is not published"
	}
	// Additional checks could include:
	// - License verification
	// - Platform version compatibility
	// - Payment status for paid listings
	return true, ""
}

// CanReview checks if a tenant can review a listing
func (s *Service) CanReview(installation *Installation) (bool, string) {
	if installation == nil {
		return false, "must install before reviewing"
	}
	if installation.Status == "uninstalled" {
		return false, "must have active installation"
	}
	return true, ""
}

// CalculateRatingsSummary calculates ratings summary from reviews
func (s *Service) CalculateRatingsSummary(reviews []Review) *RatingsSummary {
	summary := &RatingsSummary{
		UpdatedAt: time.Now(),
	}

	if len(reviews) == 0 {
		return summary
	}

	var total int
	for _, r := range reviews {
		if r.Status != "published" {
			continue
		}
		total += r.Rating
		summary.TotalReviews++
		switch r.Rating {
		case 1:
			summary.Rating1++
		case 2:
			summary.Rating2++
		case 3:
			summary.Rating3++
		case 4:
			summary.Rating4++
		case 5:
			summary.Rating5++
		}
	}

	if summary.TotalReviews > 0 {
		summary.AverageRating = float64(total) / float64(summary.TotalReviews)
	}

	return summary
}

// Categories returns available marketplace categories
func (s *Service) Categories() []Category {
	return []Category{
		{ID: "crm", Name: "CRM & Sales", Icon: "users"},
		{ID: "ecommerce", Name: "E-Commerce", Icon: "shopping-cart"},
		{ID: "payments", Name: "Payments", Icon: "credit-card"},
		{ID: "communication", Name: "Communication", Icon: "message-circle"},
		{ID: "analytics", Name: "Analytics", Icon: "bar-chart"},
		{ID: "devtools", Name: "Developer Tools", Icon: "code"},
		{ID: "security", Name: "Security", Icon: "shield"},
		{ID: "cloud", Name: "Cloud Services", Icon: "cloud"},
		{ID: "productivity", Name: "Productivity", Icon: "zap"},
		{ID: "other", Name: "Other", Icon: "grid"},
	}
}

// Category represents a marketplace category
type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Icon string `json:"icon"`
}

// PublisherProfile represents a marketplace publisher
type PublisherProfile struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Verified       bool      `json:"verified"`
	Description    string    `json:"description,omitempty"`
	Website        string    `json:"website,omitempty"`
	SupportEmail   string    `json:"support_email,omitempty"`
	ListingsCount  int       `json:"listings_count"`
	TotalInstalls  int       `json:"total_installs"`
	AverageRating  float64   `json:"average_rating"`
	JoinedAt       time.Time `json:"joined_at"`
}
