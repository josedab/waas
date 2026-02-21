package pluginmarket

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ListingStatus represents the lifecycle state of a marketplace listing
type ListingStatus string

const (
	ListingStatusDraft     ListingStatus = "draft"
	ListingStatusPublished ListingStatus = "published"
	ListingStatusSuspended ListingStatus = "suspended"
)

// MarketplaceListing represents a plugin listing in the marketplace
type MarketplaceListing struct {
	ID          string           `json:"id"`
	PluginID    string           `json:"plugin_id"`
	DeveloperID string           `json:"developer_id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Category    string           `json:"category"`
	Tags        []string         `json:"tags,omitempty"`
	Price       float64          `json:"price"`
	Currency    string           `json:"currency"`
	Rating      float64          `json:"rating"`
	Downloads   int64            `json:"downloads"`
	Screenshots []string         `json:"screenshots,omitempty"`
	Versions    []PluginVersionV2 `json:"versions,omitempty"`
	Status      ListingStatus    `json:"status"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// PluginVersionV2 tracks a version of a marketplace plugin listing
type PluginVersionV2 struct {
	Version        string    `json:"version"`
	Changelog      string    `json:"changelog"`
	MinWaaSVersion string    `json:"min_waas_version"`
	Checksum       string    `json:"checksum"`
	DownloadURL    string    `json:"download_url"`
	PublishedAt    time.Time `json:"published_at"`
}

// PluginInstallationV2 tracks installed plugins per tenant
type PluginInstallationV2 struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	ListingID   string    `json:"listing_id"`
	Version     string    `json:"version"`
	Status      string    `json:"status"`
	InstalledAt time.Time `json:"installed_at"`
}

// PluginReviewV2 represents a user review for a marketplace listing
type PluginReviewV2 struct {
	ID        string    `json:"id"`
	ListingID string    `json:"listing_id"`
	TenantID  string    `json:"tenant_id"`
	Rating    int       `json:"rating"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

// MarketplaceSearchParams defines search/filter options for the marketplace
type MarketplaceSearchParams struct {
	Query    string `json:"query,omitempty"`
	Category string `json:"category,omitempty"`
	Tag      string `json:"tag,omitempty"`
	SortBy   string `json:"sort_by,omitempty"` // rating, downloads, date
	Limit    int    `json:"limit,omitempty"`
	Offset   int    `json:"offset,omitempty"`
}

// MarketplaceService manages marketplace listings, installations, and reviews
type MarketplaceService struct {
	mu            sync.RWMutex
	listings      map[string]*MarketplaceListing
	installations map[string][]*PluginInstallationV2 // keyed by tenantID
	reviews       map[string][]*PluginReviewV2       // keyed by listingID
}

// NewMarketplaceService creates a new MarketplaceService
func NewMarketplaceService() *MarketplaceService {
	return &MarketplaceService{
		listings:      make(map[string]*MarketplaceListing),
		installations: make(map[string][]*PluginInstallationV2),
		reviews:       make(map[string][]*PluginReviewV2),
	}
}

// CreateListing adds a new plugin listing in draft status
func (s *MarketplaceService) CreateListing(ctx context.Context, listing *MarketplaceListing) (*MarketplaceListing, error) {
	if listing.Name == "" {
		return nil, fmt.Errorf("listing name is required")
	}
	if listing.DeveloperID == "" {
		return nil, fmt.Errorf("developer ID is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	listing.ID = uuid.New().String()
	listing.Status = ListingStatusDraft
	if listing.Currency == "" {
		listing.Currency = "USD"
	}
	now := time.Now()
	listing.CreatedAt = now
	listing.UpdatedAt = now
	s.listings[listing.ID] = listing
	return listing, nil
}

// Browse returns all published listings with optional filtering and sorting
func (s *MarketplaceService) Browse(ctx context.Context, params *MarketplaceSearchParams) ([]MarketplaceListing, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if params.Limit <= 0 {
		params.Limit = 20
	}

	var results []MarketplaceListing
	for _, l := range s.listings {
		if l.Status != ListingStatusPublished {
			continue
		}
		if params.Category != "" && l.Category != params.Category {
			continue
		}
		if params.Tag != "" && !containsTag(l.Tags, params.Tag) {
			continue
		}
		if params.Query != "" {
			q := strings.ToLower(params.Query)
			if !strings.Contains(strings.ToLower(l.Name), q) && !strings.Contains(strings.ToLower(l.Description), q) {
				continue
			}
		}
		results = append(results, *l)
	}

	switch params.SortBy {
	case "rating":
		sort.Slice(results, func(i, j int) bool { return results[i].Rating > results[j].Rating })
	case "downloads":
		sort.Slice(results, func(i, j int) bool { return results[i].Downloads > results[j].Downloads })
	default:
		sort.Slice(results, func(i, j int) bool { return results[i].CreatedAt.After(results[j].CreatedAt) })
	}

	total := len(results)
	if params.Offset < len(results) {
		results = results[params.Offset:]
	} else {
		results = nil
	}
	if len(results) > params.Limit {
		results = results[:params.Limit]
	}
	return results, total
}

// Search is an alias for Browse with a query term
func (s *MarketplaceService) Search(ctx context.Context, query string) ([]MarketplaceListing, int) {
	return s.Browse(ctx, &MarketplaceSearchParams{Query: query})
}

// GetListing returns a single listing by ID
func (s *MarketplaceService) GetListing(ctx context.Context, id string) (*MarketplaceListing, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	l, ok := s.listings[id]
	if !ok {
		return nil, fmt.Errorf("listing %q not found", id)
	}
	return l, nil
}

// PublishListing transitions a listing from draft to published
func (s *MarketplaceService) PublishListing(ctx context.Context, listingID, developerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	l, ok := s.listings[listingID]
	if !ok {
		return fmt.Errorf("listing %q not found", listingID)
	}
	if l.DeveloperID != developerID {
		return fmt.Errorf("unauthorized: only the listing owner can publish")
	}
	if l.Status != ListingStatusDraft {
		return fmt.Errorf("only draft listings can be published")
	}
	l.Status = ListingStatusPublished
	l.UpdatedAt = time.Now()
	return nil
}

// Install installs a published listing for a tenant
func (s *MarketplaceService) Install(ctx context.Context, tenantID, listingID string) (*PluginInstallationV2, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	l, ok := s.listings[listingID]
	if !ok {
		return nil, fmt.Errorf("listing %q not found", listingID)
	}
	if l.Status != ListingStatusPublished {
		return nil, fmt.Errorf("listing is not published")
	}

	// Check for duplicate installation
	for _, inst := range s.installations[tenantID] {
		if inst.ListingID == listingID && inst.Status == "active" {
			return nil, fmt.Errorf("plugin already installed")
		}
	}

	version := ""
	if len(l.Versions) > 0 {
		version = l.Versions[len(l.Versions)-1].Version
	}
	inst := &PluginInstallationV2{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		ListingID:   listingID,
		Version:     version,
		Status:      "active",
		InstalledAt: time.Now(),
	}
	s.installations[tenantID] = append(s.installations[tenantID], inst)
	l.Downloads++
	return inst, nil
}

// Uninstall removes a plugin installation for a tenant
func (s *MarketplaceService) Uninstall(ctx context.Context, tenantID, listingID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	installs := s.installations[tenantID]
	for _, inst := range installs {
		if inst.ListingID == listingID && inst.Status == "active" {
			inst.Status = "uninstalled"
			return nil
		}
	}
	return fmt.Errorf("plugin is not installed")
}

// GetInstalled returns all active installations for a tenant
func (s *MarketplaceService) GetInstalled(ctx context.Context, tenantID string) []PluginInstallationV2 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var active []PluginInstallationV2
	for _, inst := range s.installations[tenantID] {
		if inst.Status == "active" {
			active = append(active, *inst)
		}
	}
	return active
}

// SubmitReview adds a review for a listing
func (s *MarketplaceService) SubmitReview(ctx context.Context, tenantID, listingID string, rating int, body string) (*PluginReviewV2, error) {
	if rating < 1 || rating > 5 {
		return nil, fmt.Errorf("rating must be between 1 and 5")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	l, ok := s.listings[listingID]
	if !ok {
		return nil, fmt.Errorf("listing %q not found", listingID)
	}

	review := &PluginReviewV2{
		ID:        uuid.New().String(),
		ListingID: listingID,
		TenantID:  tenantID,
		Rating:    rating,
		Body:      body,
		CreatedAt: time.Now(),
	}
	s.reviews[listingID] = append(s.reviews[listingID], review)

	// Recalculate average rating
	reviews := s.reviews[listingID]
	var total int
	for _, r := range reviews {
		total += r.Rating
	}
	l.Rating = float64(total) / float64(len(reviews))
	return review, nil
}

func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}
