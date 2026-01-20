-- Versioning tables for webhook version management

-- Webhook versions
CREATE TABLE IF NOT EXISTS webhook_versions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    webhook_id UUID NOT NULL REFERENCES webhooks(id),
    major INTEGER NOT NULL DEFAULT 1,
    minor INTEGER NOT NULL DEFAULT 0,
    patch INTEGER NOT NULL DEFAULT 0,
    label VARCHAR(50) NOT NULL,
    schema_id UUID,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    changelog TEXT,
    breaking BOOLEAN NOT NULL DEFAULT false,
    deprecated_at TIMESTAMP WITH TIME ZONE,
    sunset_at TIMESTAMP WITH TIME ZONE,
    sunset_policy JSONB,
    replacement UUID,
    compatible_with JSONB DEFAULT '[]',
    transforms JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    published_at TIMESTAMP WITH TIME ZONE,
    
    UNIQUE(tenant_id, webhook_id, label)
);

CREATE INDEX idx_webhook_versions_tenant ON webhook_versions(tenant_id);
CREATE INDEX idx_webhook_versions_webhook ON webhook_versions(tenant_id, webhook_id);
CREATE INDEX idx_webhook_versions_status ON webhook_versions(tenant_id, status);
CREATE INDEX idx_webhook_versions_label ON webhook_versions(tenant_id, webhook_id, label);

-- Version schemas
CREATE TABLE IF NOT EXISTS version_schemas (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    format VARCHAR(20) NOT NULL DEFAULT 'json_schema',
    definition JSONB NOT NULL,
    examples JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_version_schemas_tenant ON version_schemas(tenant_id);

-- Version subscriptions
CREATE TABLE IF NOT EXISTS version_subscriptions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    endpoint_id UUID NOT NULL,
    version_id UUID NOT NULL REFERENCES webhook_versions(id) ON DELETE CASCADE,
    webhook_id UUID NOT NULL REFERENCES webhooks(id),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    pinned BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    UNIQUE(tenant_id, endpoint_id, webhook_id)
);

CREATE INDEX idx_version_subscriptions_tenant ON version_subscriptions(tenant_id);
CREATE INDEX idx_version_subscriptions_version ON version_subscriptions(tenant_id, version_id);
CREATE INDEX idx_version_subscriptions_endpoint ON version_subscriptions(tenant_id, endpoint_id);

-- Version migrations
CREATE TABLE IF NOT EXISTS version_migrations (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    webhook_id UUID NOT NULL REFERENCES webhooks(id),
    from_version UUID NOT NULL REFERENCES webhook_versions(id),
    to_version UUID NOT NULL REFERENCES webhook_versions(id),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    strategy VARCHAR(20) NOT NULL DEFAULT 'gradual',
    progress JSONB DEFAULT '{}',
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    error TEXT
);

CREATE INDEX idx_version_migrations_tenant ON version_migrations(tenant_id);
CREATE INDEX idx_version_migrations_webhook ON version_migrations(tenant_id, webhook_id);
CREATE INDEX idx_version_migrations_status ON version_migrations(status);

-- Deprecation notices
CREATE TABLE IF NOT EXISTS deprecation_notices (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    version_id UUID NOT NULL REFERENCES webhook_versions(id) ON DELETE CASCADE,
    endpoint_id UUID NOT NULL,
    type VARCHAR(20) NOT NULL DEFAULT 'deprecation',
    message TEXT NOT NULL,
    sent_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    acked_at TIMESTAMP WITH TIME ZONE,
    response TEXT
);

CREATE INDEX idx_deprecation_notices_tenant ON deprecation_notices(tenant_id);
CREATE INDEX idx_deprecation_notices_version ON deprecation_notices(tenant_id, version_id);
CREATE INDEX idx_deprecation_notices_endpoint ON deprecation_notices(tenant_id, endpoint_id);

-- Version policies
CREATE TABLE IF NOT EXISTS version_policies (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) UNIQUE,
    enabled BOOLEAN NOT NULL DEFAULT true,
    default_version VARCHAR(50) NOT NULL DEFAULT 'latest',
    require_version_header BOOLEAN NOT NULL DEFAULT false,
    allow_deprecated BOOLEAN NOT NULL DEFAULT true,
    auto_upgrade BOOLEAN NOT NULL DEFAULT false,
    deprecation_days INTEGER NOT NULL DEFAULT 90,
    sunset_days INTEGER NOT NULL DEFAULT 180,
    notification_channels JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Version usage tracking
CREATE TABLE IF NOT EXISTS version_usage (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    version_id UUID NOT NULL REFERENCES webhook_versions(id) ON DELETE CASCADE,
    endpoint_id UUID NOT NULL,
    recorded_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_version_usage_tenant ON version_usage(tenant_id);
CREATE INDEX idx_version_usage_version ON version_usage(tenant_id, version_id);
CREATE INDEX idx_version_usage_recorded ON version_usage(recorded_at);

-- Add foreign key from versions to schemas
ALTER TABLE webhook_versions
    ADD CONSTRAINT fk_versions_schema
    FOREIGN KEY (schema_id) REFERENCES version_schemas(id) ON DELETE SET NULL;

-- Add foreign key for replacement version
ALTER TABLE webhook_versions
    ADD CONSTRAINT fk_versions_replacement
    FOREIGN KEY (replacement) REFERENCES webhook_versions(id) ON DELETE SET NULL;
