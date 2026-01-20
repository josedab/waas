-- Gateway providers for inbound webhooks
CREATE TABLE IF NOT EXISTS gateway_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    signature_config JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

-- Routing rules for inbound webhooks
CREATE TABLE IF NOT EXISTS gateway_routing_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    provider_id UUID NOT NULL REFERENCES gateway_providers(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    priority INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    conditions JSONB,
    destinations JSONB NOT NULL,
    transform JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Inbound webhooks log
CREATE TABLE IF NOT EXISTS inbound_webhooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    provider_id UUID NOT NULL REFERENCES gateway_providers(id) ON DELETE CASCADE,
    provider_type VARCHAR(50) NOT NULL,
    event_type VARCHAR(255),
    payload JSONB NOT NULL,
    headers JSONB,
    raw_body BYTEA,
    signature_valid BOOLEAN NOT NULL DEFAULT FALSE,
    processed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_gateway_providers_tenant_id ON gateway_providers(tenant_id);
CREATE INDEX idx_gateway_routing_rules_tenant_id ON gateway_routing_rules(tenant_id);
CREATE INDEX idx_gateway_routing_rules_provider_id ON gateway_routing_rules(provider_id);
CREATE INDEX idx_gateway_routing_rules_priority ON gateway_routing_rules(provider_id, priority);
CREATE INDEX idx_inbound_webhooks_tenant_id ON inbound_webhooks(tenant_id);
CREATE INDEX idx_inbound_webhooks_provider_id ON inbound_webhooks(provider_id);
CREATE INDEX idx_inbound_webhooks_created_at ON inbound_webhooks(tenant_id, created_at);
CREATE INDEX idx_inbound_webhooks_event_type ON inbound_webhooks(tenant_id, event_type);
