-- Event Catalog Tables
-- Feature 2: Event type registry, catalog API, schema extraction

-- Event Types Catalog
CREATE TABLE IF NOT EXISTS event_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    schema_id UUID REFERENCES schemas(id) ON DELETE SET NULL,
    version VARCHAR(50) DEFAULT '1.0.0',
    status VARCHAR(50) DEFAULT 'active' CHECK (status IN ('active', 'deprecated', 'draft')),
    deprecation_message TEXT,
    deprecated_at TIMESTAMP WITH TIME ZONE,
    replacement_event_id UUID REFERENCES event_types(id) ON DELETE SET NULL,
    example_payload JSONB,
    tags TEXT[],
    metadata JSONB DEFAULT '{}',
    documentation_url VARCHAR(500),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, slug, version)
);

-- Event Type Versions (for tracking version history)
CREATE TABLE IF NOT EXISTS event_type_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type_id UUID NOT NULL REFERENCES event_types(id) ON DELETE CASCADE,
    version VARCHAR(50) NOT NULL,
    schema_id UUID REFERENCES schemas(id) ON DELETE SET NULL,
    changelog TEXT,
    is_breaking_change BOOLEAN DEFAULT false,
    published_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    published_by UUID,
    UNIQUE(event_type_id, version)
);

-- Event Categories (for organizing event types)
CREATE TABLE IF NOT EXISTS event_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    description TEXT,
    icon VARCHAR(50),
    color VARCHAR(20),
    parent_id UUID REFERENCES event_categories(id) ON DELETE SET NULL,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, slug)
);

-- Event Subscriptions (which endpoints listen to which events)
CREATE TABLE IF NOT EXISTS event_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id UUID NOT NULL REFERENCES webhook_endpoints(id) ON DELETE CASCADE,
    event_type_id UUID NOT NULL REFERENCES event_types(id) ON DELETE CASCADE,
    filter_expression JSONB,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(endpoint_id, event_type_id)
);

-- Event Documentation (rich documentation for events)
CREATE TABLE IF NOT EXISTS event_documentation (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type_id UUID NOT NULL REFERENCES event_types(id) ON DELETE CASCADE,
    content_type VARCHAR(50) DEFAULT 'markdown',
    content TEXT NOT NULL,
    section VARCHAR(100) DEFAULT 'overview',
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- SDK Generation Configs
CREATE TABLE IF NOT EXISTS sdk_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    language VARCHAR(50) NOT NULL CHECK (language IN ('typescript', 'python', 'go', 'java', 'ruby', 'php', 'csharp')),
    package_name VARCHAR(255),
    version VARCHAR(50),
    config JSONB DEFAULT '{}',
    last_generated_at TIMESTAMP WITH TIME ZONE,
    download_url VARCHAR(500),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, language)
);

-- Indexes
CREATE INDEX idx_event_types_tenant ON event_types(tenant_id);
CREATE INDEX idx_event_types_slug ON event_types(tenant_id, slug);
CREATE INDEX idx_event_types_category ON event_types(category);
CREATE INDEX idx_event_types_status ON event_types(status);
CREATE INDEX idx_event_types_tags ON event_types USING GIN(tags);
CREATE INDEX idx_event_type_versions_event ON event_type_versions(event_type_id);
CREATE INDEX idx_event_subscriptions_endpoint ON event_subscriptions(endpoint_id);
CREATE INDEX idx_event_subscriptions_type ON event_subscriptions(event_type_id);
CREATE INDEX idx_event_categories_tenant ON event_categories(tenant_id);
