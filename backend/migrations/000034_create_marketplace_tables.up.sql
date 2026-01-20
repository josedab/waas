-- Self-Service Marketplace Tables
-- Feature 10: Connector marketplace with ratings and reviews

-- Marketplace Listings
CREATE TABLE IF NOT EXISTS marketplace_listings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Publisher info
    publisher_id UUID NOT NULL REFERENCES tenants(id),
    publisher_name VARCHAR(255) NOT NULL,
    publisher_verified BOOLEAN DEFAULT false,
    
    -- Listing details
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    short_description VARCHAR(500) NOT NULL,
    full_description TEXT,
    
    -- Categorization
    category VARCHAR(100) NOT NULL,
    subcategory VARCHAR(100),
    tags TEXT[],
    
    -- Type of listing
    listing_type VARCHAR(50) NOT NULL CHECK (listing_type IN ('connector', 'transformation', 'template', 'integration')),
    
    -- Visuals
    icon_url VARCHAR(500),
    banner_url VARCHAR(500),
    screenshots TEXT[],
    
    -- Technical details
    version VARCHAR(50) NOT NULL,
    min_platform_version VARCHAR(50),
    documentation_url VARCHAR(500),
    source_url VARCHAR(500),
    
    -- Pricing
    pricing_model VARCHAR(50) DEFAULT 'free' CHECK (pricing_model IN ('free', 'freemium', 'paid', 'subscription', 'usage_based')),
    price_cents INTEGER DEFAULT 0,
    currency VARCHAR(3) DEFAULT 'USD',
    
    -- Stats
    install_count INTEGER DEFAULT 0,
    active_installs INTEGER DEFAULT 0,
    
    -- Status
    status VARCHAR(50) DEFAULT 'draft' CHECK (status IN ('draft', 'pending_review', 'published', 'rejected', 'suspended', 'archived')),
    review_notes TEXT,
    
    published_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Listing Versions (release history)
CREATE TABLE IF NOT EXISTS listing_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id UUID NOT NULL REFERENCES marketplace_listings(id) ON DELETE CASCADE,
    
    version VARCHAR(50) NOT NULL,
    release_notes TEXT,
    
    -- Artifact
    artifact_type VARCHAR(50) CHECK (artifact_type IN ('config', 'code', 'bundle')),
    artifact_url VARCHAR(500),
    artifact_hash VARCHAR(64),
    artifact_size_bytes INTEGER,
    
    -- Compatibility
    min_platform_version VARCHAR(50),
    max_platform_version VARCHAR(50),
    breaking_changes TEXT[],
    
    status VARCHAR(50) DEFAULT 'active' CHECK (status IN ('active', 'deprecated', 'yanked')),
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(listing_id, version)
);

-- Listing Configuration Schema
CREATE TABLE IF NOT EXISTS listing_config_schemas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id UUID NOT NULL REFERENCES marketplace_listings(id) ON DELETE CASCADE,
    version_id UUID REFERENCES listing_versions(id) ON DELETE CASCADE,
    
    -- Schema definition
    config_schema JSONB NOT NULL,
    ui_schema JSONB, -- For form generation
    default_config JSONB,
    
    -- Secrets handling
    secret_fields TEXT[],
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- User Installations
CREATE TABLE IF NOT EXISTS marketplace_installations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    listing_id UUID NOT NULL REFERENCES marketplace_listings(id) ON DELETE CASCADE,
    version_id UUID REFERENCES listing_versions(id),
    
    -- Installation details
    installed_version VARCHAR(50) NOT NULL,
    config_encrypted JSONB,
    
    -- Status
    status VARCHAR(50) DEFAULT 'active' CHECK (status IN ('active', 'disabled', 'uninstalled', 'pending_update')),
    
    -- Usage tracking
    last_used_at TIMESTAMP WITH TIME ZONE,
    usage_count INTEGER DEFAULT 0,
    
    installed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(tenant_id, listing_id)
);

-- Reviews
CREATE TABLE IF NOT EXISTS marketplace_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id UUID NOT NULL REFERENCES marketplace_listings(id) ON DELETE CASCADE,
    reviewer_id UUID NOT NULL REFERENCES tenants(id),
    reviewer_name VARCHAR(255),
    
    -- Review content
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    title VARCHAR(255),
    body TEXT,
    
    -- Metadata
    verified_purchase BOOLEAN DEFAULT false,
    helpful_count INTEGER DEFAULT 0,
    
    -- Status
    status VARCHAR(50) DEFAULT 'published' CHECK (status IN ('pending', 'published', 'hidden', 'flagged')),
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(listing_id, reviewer_id)
);

-- Review Responses (publisher can respond)
CREATE TABLE IF NOT EXISTS review_responses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    review_id UUID NOT NULL REFERENCES marketplace_reviews(id) ON DELETE CASCADE UNIQUE,
    
    body TEXT NOT NULL,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Ratings Summary (denormalized for performance)
CREATE TABLE IF NOT EXISTS listing_ratings_summary (
    listing_id UUID PRIMARY KEY REFERENCES marketplace_listings(id) ON DELETE CASCADE,
    
    total_reviews INTEGER DEFAULT 0,
    average_rating DECIMAL(3,2) DEFAULT 0,
    
    rating_1 INTEGER DEFAULT 0,
    rating_2 INTEGER DEFAULT 0,
    rating_3 INTEGER DEFAULT 0,
    rating_4 INTEGER DEFAULT 0,
    rating_5 INTEGER DEFAULT 0,
    
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Featured Listings
CREATE TABLE IF NOT EXISTS featured_listings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id UUID NOT NULL REFERENCES marketplace_listings(id) ON DELETE CASCADE,
    
    feature_type VARCHAR(50) NOT NULL CHECK (feature_type IN ('homepage', 'category', 'trending', 'staff_pick', 'new_notable')),
    category VARCHAR(100),
    
    display_order INTEGER DEFAULT 0,
    
    start_date TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    end_date TIMESTAMP WITH TIME ZONE,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Publisher Payouts (for paid listings)
CREATE TABLE IF NOT EXISTS publisher_payouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    publisher_id UUID NOT NULL REFERENCES tenants(id),
    
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    
    gross_revenue_cents INTEGER NOT NULL,
    platform_fee_cents INTEGER NOT NULL,
    net_payout_cents INTEGER NOT NULL,
    
    status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'paid', 'failed')),
    payout_method VARCHAR(50),
    payout_reference VARCHAR(255),
    
    paid_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_marketplace_listings_publisher ON marketplace_listings(publisher_id);
CREATE INDEX idx_marketplace_listings_category ON marketplace_listings(category);
CREATE INDEX idx_marketplace_listings_status ON marketplace_listings(status);
CREATE INDEX idx_marketplace_listings_slug ON marketplace_listings(slug);
CREATE INDEX idx_listing_versions_listing ON listing_versions(listing_id);
CREATE INDEX idx_marketplace_installations_tenant ON marketplace_installations(tenant_id);
CREATE INDEX idx_marketplace_installations_listing ON marketplace_installations(listing_id);
CREATE INDEX idx_marketplace_reviews_listing ON marketplace_reviews(listing_id);
CREATE INDEX idx_marketplace_reviews_rating ON marketplace_reviews(listing_id, rating);
CREATE INDEX idx_featured_listings_type ON featured_listings(feature_type, display_order);
