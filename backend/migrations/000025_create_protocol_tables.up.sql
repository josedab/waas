-- Protocol configs table
CREATE TABLE IF NOT EXISTS protocol_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    endpoint_id UUID NOT NULL,
    protocol VARCHAR(20) NOT NULL,
    target VARCHAR(500) NOT NULL,
    options JSONB DEFAULT '{}',
    headers JSONB DEFAULT '{}',
    tls JSONB,
    auth JSONB,
    timeout INT DEFAULT 30,
    retries INT DEFAULT 3,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_protocol_configs_tenant_id ON protocol_configs(tenant_id);
CREATE INDEX idx_protocol_configs_endpoint_id ON protocol_configs(endpoint_id);
CREATE INDEX idx_protocol_configs_protocol ON protocol_configs(protocol);
CREATE INDEX idx_protocol_configs_enabled ON protocol_configs(enabled);
CREATE INDEX idx_protocol_configs_tenant_endpoint ON protocol_configs(tenant_id, endpoint_id);
