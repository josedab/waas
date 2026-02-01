package marketplace

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/josedab/waas/pkg/database"
)

// Repository handles marketplace data persistence
type Repository struct {
	db *database.DB
}

// NewRepository creates a new marketplace repository
func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

// CreateListing creates a new marketplace listing
func (r *Repository) CreateListing(ctx context.Context, l *Listing) error {
	query := `
		INSERT INTO marketplace_listings (
			publisher_id, publisher_name, publisher_verified, name, slug,
			short_description, full_description, category, subcategory, tags,
			listing_type, icon_url, banner_url, screenshots, version,
			min_platform_version, documentation_url, source_url,
			pricing_model, price_cents, currency
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21
		)
		RETURNING id, status, install_count, active_installs, created_at, updated_at`

	return r.db.Pool.QueryRow(ctx, query,
		l.PublisherID, l.PublisherName, l.PublisherVerified, l.Name, l.Slug,
		l.ShortDescription, l.FullDescription, l.Category, l.Subcategory, l.Tags,
		l.ListingType, l.IconURL, l.BannerURL, l.Screenshots, l.Version,
		l.MinPlatformVersion, l.DocumentationURL, l.SourceURL,
		l.PricingModel, l.PriceCents, l.Currency,
	).Scan(&l.ID, &l.Status, &l.InstallCount, &l.ActiveInstalls, &l.CreatedAt, &l.UpdatedAt)
}

// GetListing retrieves a listing by ID
func (r *Repository) GetListing(ctx context.Context, id string) (*Listing, error) {
	query := `
		SELECT l.id, l.publisher_id, l.publisher_name, l.publisher_verified,
			l.name, l.slug, l.short_description, l.full_description,
			l.category, l.subcategory, l.tags, l.listing_type,
			l.icon_url, l.banner_url, l.screenshots, l.version,
			l.min_platform_version, l.documentation_url, l.source_url,
			l.pricing_model, l.price_cents, l.currency,
			l.install_count, l.active_installs, l.status, l.review_notes,
			l.published_at, l.created_at, l.updated_at,
			COALESCE(rs.average_rating, 0), COALESCE(rs.total_reviews, 0)
		FROM marketplace_listings l
		LEFT JOIN listing_ratings_summary rs ON rs.listing_id = l.id
		WHERE l.id = $1`

	var l Listing
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&l.ID, &l.PublisherID, &l.PublisherName, &l.PublisherVerified,
		&l.Name, &l.Slug, &l.ShortDescription, &l.FullDescription,
		&l.Category, &l.Subcategory, &l.Tags, &l.ListingType,
		&l.IconURL, &l.BannerURL, &l.Screenshots, &l.Version,
		&l.MinPlatformVersion, &l.DocumentationURL, &l.SourceURL,
		&l.PricingModel, &l.PriceCents, &l.Currency,
		&l.InstallCount, &l.ActiveInstalls, &l.Status, &l.ReviewNotes,
		&l.PublishedAt, &l.CreatedAt, &l.UpdatedAt,
		&l.AverageRating, &l.TotalReviews,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrListingNotFound
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// GetListingBySlug retrieves a listing by slug
func (r *Repository) GetListingBySlug(ctx context.Context, slug string) (*Listing, error) {
	query := `
		SELECT l.id, l.publisher_id, l.publisher_name, l.publisher_verified,
			l.name, l.slug, l.short_description, l.full_description,
			l.category, l.subcategory, l.tags, l.listing_type,
			l.icon_url, l.banner_url, l.screenshots, l.version,
			l.min_platform_version, l.documentation_url, l.source_url,
			l.pricing_model, l.price_cents, l.currency,
			l.install_count, l.active_installs, l.status, l.review_notes,
			l.published_at, l.created_at, l.updated_at,
			COALESCE(rs.average_rating, 0), COALESCE(rs.total_reviews, 0)
		FROM marketplace_listings l
		LEFT JOIN listing_ratings_summary rs ON rs.listing_id = l.id
		WHERE l.slug = $1`

	var l Listing
	err := r.db.Pool.QueryRow(ctx, query, slug).Scan(
		&l.ID, &l.PublisherID, &l.PublisherName, &l.PublisherVerified,
		&l.Name, &l.Slug, &l.ShortDescription, &l.FullDescription,
		&l.Category, &l.Subcategory, &l.Tags, &l.ListingType,
		&l.IconURL, &l.BannerURL, &l.Screenshots, &l.Version,
		&l.MinPlatformVersion, &l.DocumentationURL, &l.SourceURL,
		&l.PricingModel, &l.PriceCents, &l.Currency,
		&l.InstallCount, &l.ActiveInstalls, &l.Status, &l.ReviewNotes,
		&l.PublishedAt, &l.CreatedAt, &l.UpdatedAt,
		&l.AverageRating, &l.TotalReviews,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrListingNotFound
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// SearchListings searches marketplace listings
func (r *Repository) SearchListings(ctx context.Context, filters *SearchFilters) (*SearchResult, error) {
	// Build dynamic query
	baseQuery := `
		SELECT l.id, l.publisher_id, l.publisher_name, l.publisher_verified,
			l.name, l.slug, l.short_description, l.full_description,
			l.category, l.subcategory, l.tags, l.listing_type,
			l.icon_url, l.banner_url, l.screenshots, l.version,
			l.min_platform_version, l.documentation_url, l.source_url,
			l.pricing_model, l.price_cents, l.currency,
			l.install_count, l.active_installs, l.status, l.review_notes,
			l.published_at, l.created_at, l.updated_at,
			COALESCE(rs.average_rating, 0), COALESCE(rs.total_reviews, 0)
		FROM marketplace_listings l
		LEFT JOIN listing_ratings_summary rs ON rs.listing_id = l.id
		WHERE l.status = 'published'`

	countQuery := `SELECT COUNT(*) FROM marketplace_listings l 
		LEFT JOIN listing_ratings_summary rs ON rs.listing_id = l.id
		WHERE l.status = 'published'`

	var args []interface{}
	argNum := 1

	// Add filters
	if filters.Query != "" {
		baseQuery += fmt.Sprintf(" AND (l.name ILIKE $%d OR l.short_description ILIKE $%d)", argNum, argNum)
		countQuery += fmt.Sprintf(" AND (l.name ILIKE $%d OR l.short_description ILIKE $%d)", argNum, argNum)
		args = append(args, "%"+filters.Query+"%")
		argNum++
	}

	if filters.Category != "" {
		baseQuery += fmt.Sprintf(" AND l.category = $%d", argNum)
		countQuery += fmt.Sprintf(" AND l.category = $%d", argNum)
		args = append(args, filters.Category)
		argNum++
	}

	if filters.ListingType != "" {
		baseQuery += fmt.Sprintf(" AND l.listing_type = $%d", argNum)
		countQuery += fmt.Sprintf(" AND l.listing_type = $%d", argNum)
		args = append(args, filters.ListingType)
		argNum++
	}

	if filters.PricingModel != "" {
		baseQuery += fmt.Sprintf(" AND l.pricing_model = $%d", argNum)
		countQuery += fmt.Sprintf(" AND l.pricing_model = $%d", argNum)
		args = append(args, filters.PricingModel)
		argNum++
	}

	if filters.MinRating > 0 {
		baseQuery += fmt.Sprintf(" AND COALESCE(rs.average_rating, 0) >= $%d", argNum)
		countQuery += fmt.Sprintf(" AND COALESCE(rs.average_rating, 0) >= $%d", argNum)
		args = append(args, filters.MinRating)
		argNum++
	}

	// Sorting
	orderBy := " ORDER BY "
	switch filters.SortBy {
	case "rating":
		orderBy += "rs.average_rating"
	case "recent":
		orderBy += "l.published_at"
	case "name":
		orderBy += "l.name"
	default:
		orderBy += "l.install_count"
	}

	if filters.SortOrder == "asc" {
		orderBy += " ASC NULLS LAST"
	} else {
		orderBy += " DESC NULLS LAST"
	}

	baseQuery += orderBy

	// Pagination
	limit := filters.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := filters.Offset
	if offset < 0 {
		offset = 0
	}

	baseQuery += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)

	// Get total count
	var totalCount int
	err := r.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, err
	}

	// Get listings
	rows, err := r.db.Pool.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var listings []Listing
	for rows.Next() {
		var l Listing
		err := rows.Scan(
			&l.ID, &l.PublisherID, &l.PublisherName, &l.PublisherVerified,
			&l.Name, &l.Slug, &l.ShortDescription, &l.FullDescription,
			&l.Category, &l.Subcategory, &l.Tags, &l.ListingType,
			&l.IconURL, &l.BannerURL, &l.Screenshots, &l.Version,
			&l.MinPlatformVersion, &l.DocumentationURL, &l.SourceURL,
			&l.PricingModel, &l.PriceCents, &l.Currency,
			&l.InstallCount, &l.ActiveInstalls, &l.Status, &l.ReviewNotes,
			&l.PublishedAt, &l.CreatedAt, &l.UpdatedAt,
			&l.AverageRating, &l.TotalReviews,
		)
		if err != nil {
			return nil, err
		}
		listings = append(listings, l)
	}

	return &SearchResult{
		Listings:   listings,
		TotalCount: totalCount,
		Filters:    filters,
	}, nil
}

// UpdateListingStatus updates a listing's status
func (r *Repository) UpdateListingStatus(ctx context.Context, id string, status ListingStatus, notes string) error {
	query := `
		UPDATE marketplace_listings
		SET status = $2, review_notes = $3, updated_at = NOW(),
			published_at = CASE WHEN $2 = 'published' AND published_at IS NULL THEN NOW() ELSE published_at END
		WHERE id = $1`

	_, err := r.db.Pool.Exec(ctx, query, id, status, notes)
	return err
}

// CreateInstallation records a new installation
func (r *Repository) CreateInstallation(ctx context.Context, i *Installation) error {
	configJSON, _ := json.Marshal(i.Config)

	query := `
		INSERT INTO marketplace_installations (tenant_id, listing_id, version_id, installed_version, config_encrypted)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tenant_id, listing_id) 
		DO UPDATE SET installed_version = $4, config_encrypted = $5, status = 'active', updated_at = NOW()
		RETURNING id, status, installed_at, updated_at`

	err := r.db.Pool.QueryRow(ctx, query,
		i.TenantID, i.ListingID, i.VersionID, i.InstalledVersion, configJSON,
	).Scan(&i.ID, &i.Status, &i.InstalledAt, &i.UpdatedAt)

	if err == nil {
		// Update install count
		r.db.Pool.Exec(ctx,
			"UPDATE marketplace_listings SET install_count = install_count + 1, active_installs = active_installs + 1 WHERE id = $1",
			i.ListingID,
		)
	}

	return err
}

// GetInstallation retrieves an installation
func (r *Repository) GetInstallation(ctx context.Context, tenantID, listingID string) (*Installation, error) {
	var i Installation
	var configJSON []byte

	query := `
		SELECT id, tenant_id, listing_id, version_id, installed_version, config_encrypted,
			status, last_used_at, usage_count, installed_at, updated_at
		FROM marketplace_installations
		WHERE tenant_id = $1 AND listing_id = $2`

	err := r.db.Pool.QueryRow(ctx, query, tenantID, listingID).Scan(
		&i.ID, &i.TenantID, &i.ListingID, &i.VersionID, &i.InstalledVersion, &configJSON,
		&i.Status, &i.LastUsedAt, &i.UsageCount, &i.InstalledAt, &i.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if len(configJSON) > 0 {
		json.Unmarshal(configJSON, &i.Config)
	}

	return &i, nil
}

// ListInstallations lists a tenant's installations
func (r *Repository) ListInstallations(ctx context.Context, tenantID string) ([]Installation, error) {
	query := `
		SELECT i.id, i.tenant_id, i.listing_id, i.version_id, i.installed_version, i.config_encrypted,
			i.status, i.last_used_at, i.usage_count, i.installed_at, i.updated_at,
			l.name, l.slug, l.icon_url, l.version as latest_version
		FROM marketplace_installations i
		JOIN marketplace_listings l ON l.id = i.listing_id
		WHERE i.tenant_id = $1 AND i.status != 'uninstalled'
		ORDER BY i.installed_at DESC`

	rows, err := r.db.Pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var installations []Installation
	for rows.Next() {
		var i Installation
		var configJSON []byte
		var listing Listing
		err := rows.Scan(
			&i.ID, &i.TenantID, &i.ListingID, &i.VersionID, &i.InstalledVersion, &configJSON,
			&i.Status, &i.LastUsedAt, &i.UsageCount, &i.InstalledAt, &i.UpdatedAt,
			&listing.Name, &listing.Slug, &listing.IconURL, &listing.Version,
		)
		if err != nil {
			return nil, err
		}
		if len(configJSON) > 0 {
			json.Unmarshal(configJSON, &i.Config)
		}
		i.Listing = &listing
		installations = append(installations, i)
	}

	return installations, nil
}

// Uninstall marks an installation as uninstalled
func (r *Repository) Uninstall(ctx context.Context, tenantID, listingID string) error {
	query := `
		UPDATE marketplace_installations
		SET status = 'uninstalled', updated_at = NOW()
		WHERE tenant_id = $1 AND listing_id = $2`

	result, err := r.db.Pool.Exec(ctx, query, tenantID, listingID)
	if err != nil {
		return err
	}

	if result.RowsAffected() > 0 {
		r.db.Pool.Exec(ctx,
			"UPDATE marketplace_listings SET active_installs = active_installs - 1 WHERE id = $1 AND active_installs > 0",
			listingID,
		)
	}

	return nil
}

// CreateReview creates a new review
func (r *Repository) CreateReview(ctx context.Context, review *Review) error {
	query := `
		INSERT INTO marketplace_reviews (listing_id, reviewer_id, reviewer_name, rating, title, body, verified_purchase)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, status, helpful_count, created_at, updated_at`

	err := r.db.Pool.QueryRow(ctx, query,
		review.ListingID, review.ReviewerID, review.ReviewerName,
		review.Rating, review.Title, review.Body, review.VerifiedPurchase,
	).Scan(&review.ID, &review.Status, &review.HelpfulCount, &review.CreatedAt, &review.UpdatedAt)

	if err == nil {
		r.updateRatingsSummary(ctx, review.ListingID)
	}

	return err
}

// GetReviews retrieves reviews for a listing
func (r *Repository) GetReviews(ctx context.Context, listingID string, limit, offset int) ([]Review, error) {
	query := `
		SELECT r.id, r.listing_id, r.reviewer_id, r.reviewer_name, r.rating,
			r.title, r.body, r.verified_purchase, r.helpful_count, r.status,
			r.created_at, r.updated_at,
			rr.id, rr.body, rr.created_at
		FROM marketplace_reviews r
		LEFT JOIN review_responses rr ON rr.review_id = r.id
		WHERE r.listing_id = $1 AND r.status = 'published'
		ORDER BY r.helpful_count DESC, r.created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Pool.Query(ctx, query, listingID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []Review
	for rows.Next() {
		var review Review
		var respID, respBody *string
		var respCreated *time.Time

		err := rows.Scan(
			&review.ID, &review.ListingID, &review.ReviewerID, &review.ReviewerName,
			&review.Rating, &review.Title, &review.Body, &review.VerifiedPurchase,
			&review.HelpfulCount, &review.Status, &review.CreatedAt, &review.UpdatedAt,
			&respID, &respBody, &respCreated,
		)
		if err != nil {
			return nil, err
		}

		if respID != nil {
			review.Response = &ReviewResponse{
				ID:        *respID,
				ReviewID:  review.ID,
				Body:      *respBody,
				CreatedAt: *respCreated,
			}
		}

		reviews = append(reviews, review)
	}

	return reviews, nil
}

// updateRatingsSummary recalculates ratings summary
func (r *Repository) updateRatingsSummary(ctx context.Context, listingID string) error {
	query := `
		INSERT INTO listing_ratings_summary (listing_id, total_reviews, average_rating, rating_1, rating_2, rating_3, rating_4, rating_5)
		SELECT 
			$1,
			COUNT(*),
			COALESCE(AVG(rating)::decimal(3,2), 0),
			COUNT(*) FILTER (WHERE rating = 1),
			COUNT(*) FILTER (WHERE rating = 2),
			COUNT(*) FILTER (WHERE rating = 3),
			COUNT(*) FILTER (WHERE rating = 4),
			COUNT(*) FILTER (WHERE rating = 5)
		FROM marketplace_reviews
		WHERE listing_id = $1 AND status = 'published'
		ON CONFLICT (listing_id) DO UPDATE SET
			total_reviews = EXCLUDED.total_reviews,
			average_rating = EXCLUDED.average_rating,
			rating_1 = EXCLUDED.rating_1,
			rating_2 = EXCLUDED.rating_2,
			rating_3 = EXCLUDED.rating_3,
			rating_4 = EXCLUDED.rating_4,
			rating_5 = EXCLUDED.rating_5,
			updated_at = NOW()`

	_, err := r.db.Pool.Exec(ctx, query, listingID)
	return err
}

// GetFeaturedListings retrieves featured listings
func (r *Repository) GetFeaturedListings(ctx context.Context, featureType string, limit int) ([]FeaturedListing, error) {
	query := `
		SELECT f.id, f.listing_id, f.feature_type, f.category, f.display_order,
			f.start_date, f.end_date, f.created_at,
			l.name, l.slug, l.short_description, l.icon_url, l.category as listing_category,
			COALESCE(rs.average_rating, 0), l.install_count
		FROM featured_listings f
		JOIN marketplace_listings l ON l.id = f.listing_id AND l.status = 'published'
		LEFT JOIN listing_ratings_summary rs ON rs.listing_id = l.id
		WHERE f.feature_type = $1
			AND f.start_date <= NOW()
			AND (f.end_date IS NULL OR f.end_date > NOW())
		ORDER BY f.display_order ASC
		LIMIT $2`

	rows, err := r.db.Pool.Query(ctx, query, featureType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var featured []FeaturedListing
	for rows.Next() {
		var f FeaturedListing
		var l Listing
		err := rows.Scan(
			&f.ID, &f.ListingID, &f.FeatureType, &f.Category, &f.DisplayOrder,
			&f.StartDate, &f.EndDate, &f.CreatedAt,
			&l.Name, &l.Slug, &l.ShortDescription, &l.IconURL, &l.Category,
			&l.AverageRating, &l.InstallCount,
		)
		if err != nil {
			return nil, err
		}
		f.Listing = &l
		featured = append(featured, f)
	}

	return featured, nil
}
