CREATE TABLE IF NOT EXISTS marketplace_templates (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    category VARCHAR(50) NOT NULL,
    source VARCHAR(100) NOT NULL,
    destination VARCHAR(100) NOT NULL,
    transform TEXT,
    retry_policy JSONB,
    sample_payload JSONB,
    version VARCHAR(50) NOT NULL DEFAULT '1.0.0',
    author VARCHAR(100) NOT NULL DEFAULT 'community',
    install_count INTEGER NOT NULL DEFAULT 0,
    rating DECIMAL(3,2) NOT NULL DEFAULT 0,
    is_verified BOOLEAN NOT NULL DEFAULT false,
    tags JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS marketplace_installations (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    template_id VARCHAR(36) NOT NULL REFERENCES marketplace_templates(id) ON DELETE CASCADE,
    endpoint_id VARCHAR(36),
    config JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    installed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS marketplace_reviews (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    template_id VARCHAR(36) NOT NULL REFERENCES marketplace_templates(id) ON DELETE CASCADE,
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    comment TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, template_id)
);

CREATE INDEX idx_mp_templates_category ON marketplace_templates(category);
CREATE INDEX idx_mp_templates_search ON marketplace_templates USING gin(to_tsvector('english', name || ' ' || description));
CREATE INDEX idx_mp_installs_tenant ON marketplace_installations(tenant_id);
CREATE INDEX idx_mp_reviews_template ON marketplace_reviews(template_id);
