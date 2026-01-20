-- OTEL configs table
CREATE TABLE IF NOT EXISTS otel_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    service_name VARCHAR(255) NOT NULL DEFAULT 'waas-webhook-service',
    enabled BOOLEAN DEFAULT false,
    traces JSONB DEFAULT '{}',
    metrics JSONB DEFAULT '{}',
    logs JSONB DEFAULT '{}',
    attributes JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_otel_configs_tenant_id ON otel_configs(tenant_id);
CREATE INDEX idx_otel_configs_enabled ON otel_configs(enabled);
CREATE INDEX idx_otel_configs_tenant_enabled ON otel_configs(tenant_id, enabled);
