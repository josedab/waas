CREATE TABLE IF NOT EXISTS inbound_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    verification_secret TEXT,
    verification_header VARCHAR(255),
    verification_algorithm VARCHAR(50) DEFAULT 'hmac-sha256',
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS inbound_routing_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL REFERENCES inbound_sources(id) ON DELETE CASCADE,
    filter_expression TEXT,
    destination_type VARCHAR(50) NOT NULL,
    destination_config JSONB NOT NULL DEFAULT '{}',
    priority INTEGER NOT NULL DEFAULT 0,
    active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS inbound_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL REFERENCES inbound_sources(id),
    tenant_id UUID NOT NULL,
    provider VARCHAR(50) NOT NULL,
    raw_payload TEXT NOT NULL,
    normalized_payload TEXT,
    headers JSONB DEFAULT '{}',
    signature_valid BOOLEAN NOT NULL DEFAULT FALSE,
    status VARCHAR(20) NOT NULL DEFAULT 'received',
    error_message TEXT,
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_inbound_sources_tenant ON inbound_sources(tenant_id);
CREATE INDEX idx_inbound_sources_status ON inbound_sources(tenant_id, status);
CREATE INDEX idx_inbound_routing_rules_source ON inbound_routing_rules(source_id);
CREATE INDEX idx_inbound_events_source ON inbound_events(source_id);
CREATE INDEX idx_inbound_events_tenant ON inbound_events(tenant_id);
CREATE INDEX idx_inbound_events_status ON inbound_events(source_id, status);
CREATE INDEX idx_inbound_events_created ON inbound_events(created_at);
