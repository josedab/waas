-- Rollback Marketplace Tables

DROP INDEX IF EXISTS idx_featured_listings_type;
DROP INDEX IF EXISTS idx_marketplace_reviews_rating;
DROP INDEX IF EXISTS idx_marketplace_reviews_listing;
DROP INDEX IF EXISTS idx_marketplace_installations_listing;
DROP INDEX IF EXISTS idx_marketplace_installations_tenant;
DROP INDEX IF EXISTS idx_listing_versions_listing;
DROP INDEX IF EXISTS idx_marketplace_listings_slug;
DROP INDEX IF EXISTS idx_marketplace_listings_status;
DROP INDEX IF EXISTS idx_marketplace_listings_category;
DROP INDEX IF EXISTS idx_marketplace_listings_publisher;

DROP TABLE IF EXISTS publisher_payouts;
DROP TABLE IF EXISTS featured_listings;
DROP TABLE IF EXISTS listing_ratings_summary;
DROP TABLE IF EXISTS review_responses;
DROP TABLE IF EXISTS marketplace_reviews;
DROP TABLE IF EXISTS marketplace_installations;
DROP TABLE IF EXISTS listing_config_schemas;
DROP TABLE IF EXISTS listing_versions;
DROP TABLE IF EXISTS marketplace_listings;
