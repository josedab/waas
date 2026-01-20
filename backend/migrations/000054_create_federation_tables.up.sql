-- Federation tables for cross-organization webhook delivery

-- Federation members
CREATE TABLE IF NOT EXISTS federation_members (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    organization_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    domain VARCHAR(255),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    public_key TEXT,
    endpoints JSONB DEFAULT '[]',
    capabilities JSONB DEFAULT '[]',
    trust_level VARCHAR(20) NOT NULL DEFAULT 'none',
    metadata JSONB DEFAULT '{}',
    joined_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    UNIQUE(domain)
);

CREATE INDEX idx_federation_members_tenant ON federation_members(tenant_id);
CREATE INDEX idx_federation_members_domain ON federation_members(domain);
CREATE INDEX idx_federation_members_status ON federation_members(tenant_id, status);

-- Trust relationships
CREATE TABLE IF NOT EXISTS federation_trust (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    source_member_id UUID NOT NULL REFERENCES federation_members(id) ON DELETE CASCADE,
    target_member_id UUID NOT NULL REFERENCES federation_members(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    trust_level VARCHAR(20) NOT NULL DEFAULT 'none',
    permissions JSONB DEFAULT '[]',
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    UNIQUE(source_member_id, target_member_id)
);

CREATE INDEX idx_federation_trust_tenant ON federation_trust(tenant_id);
CREATE INDEX idx_federation_trust_source ON federation_trust(source_member_id);
CREATE INDEX idx_federation_trust_target ON federation_trust(target_member_id);

-- Trust requests
CREATE TABLE IF NOT EXISTS federation_trust_requests (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    requester_id UUID NOT NULL REFERENCES federation_members(id) ON DELETE CASCADE,
    target_member_id UUID NOT NULL REFERENCES federation_members(id) ON DELETE CASCADE,
    requested_level VARCHAR(20) NOT NULL DEFAULT 'basic',
    permissions JSONB DEFAULT '[]',
    message TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    expires_at TIMESTAMP WITH TIME ZONE,
    responded_at TIMESTAMP WITH TIME ZONE,
    response TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_federation_trust_requests_tenant ON federation_trust_requests(tenant_id);
CREATE INDEX idx_federation_trust_requests_status ON federation_trust_requests(tenant_id, status);

-- Event catalogs
CREATE TABLE IF NOT EXISTS federation_catalogs (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    member_id UUID NOT NULL REFERENCES federation_members(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    event_types JSONB NOT NULL DEFAULT '[]',
    version VARCHAR(50) NOT NULL DEFAULT '1.0.0',
    public BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_federation_catalogs_tenant ON federation_catalogs(tenant_id);
CREATE INDEX idx_federation_catalogs_member ON federation_catalogs(member_id);
CREATE INDEX idx_federation_catalogs_public ON federation_catalogs(public) WHERE public = true;

-- Federated subscriptions
CREATE TABLE IF NOT EXISTS federation_subscriptions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    source_member_id UUID NOT NULL REFERENCES federation_members(id) ON DELETE CASCADE,
    target_member_id UUID NOT NULL REFERENCES federation_members(id) ON DELETE CASCADE,
    catalog_id UUID NOT NULL REFERENCES federation_catalogs(id) ON DELETE CASCADE,
    event_types JSONB NOT NULL DEFAULT '[]',
    filter JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    delivery_config JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_federation_subscriptions_tenant ON federation_subscriptions(tenant_id);
CREATE INDEX idx_federation_subscriptions_target ON federation_subscriptions(target_member_id);
CREATE INDEX idx_federation_subscriptions_status ON federation_subscriptions(status);

-- Federated deliveries
CREATE TABLE IF NOT EXISTS federation_deliveries (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    subscription_id UUID NOT NULL REFERENCES federation_subscriptions(id) ON DELETE CASCADE,
    source_member_id UUID NOT NULL,
    target_member_id UUID NOT NULL,
    event_type VARCHAR(255) NOT NULL,
    event_id VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    last_attempt_at TIMESTAMP WITH TIME ZONE,
    next_retry_at TIMESTAMP WITH TIME ZONE,
    error TEXT,
    response_code INTEGER,
    response_body TEXT,
    latency BIGINT,
    delivered_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_federation_deliveries_tenant ON federation_deliveries(tenant_id);
CREATE INDEX idx_federation_deliveries_subscription ON federation_deliveries(subscription_id);
CREATE INDEX idx_federation_deliveries_status ON federation_deliveries(status);
CREATE INDEX idx_federation_deliveries_pending ON federation_deliveries(status, next_retry_at) 
    WHERE status IN ('pending', 'retrying');

-- Federation policies
CREATE TABLE IF NOT EXISTS federation_policies (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) UNIQUE,
    enabled BOOLEAN NOT NULL DEFAULT true,
    auto_accept_trust BOOLEAN NOT NULL DEFAULT false,
    min_trust_level VARCHAR(20) NOT NULL DEFAULT 'basic',
    allowed_domains JSONB DEFAULT '[]',
    blocked_domains JSONB DEFAULT '[]',
    require_encryption BOOLEAN NOT NULL DEFAULT true,
    allow_relay BOOLEAN NOT NULL DEFAULT false,
    max_subscriptions INTEGER NOT NULL DEFAULT 100,
    rate_limit_per_member INTEGER NOT NULL DEFAULT 1000,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Federation crypto keys
CREATE TABLE IF NOT EXISTS federation_keys (
    member_id UUID PRIMARY KEY REFERENCES federation_members(id) ON DELETE CASCADE,
    public_key TEXT NOT NULL,
    algorithm VARCHAR(20) NOT NULL DEFAULT 'ed25519',
    key_id VARCHAR(64) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    revoked BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX idx_federation_keys_key_id ON federation_keys(key_id);
