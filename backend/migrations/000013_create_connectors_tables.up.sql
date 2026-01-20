-- Installed connectors
CREATE TABLE IF NOT EXISTS installed_connectors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    connector_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    config JSONB,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    provider_id UUID REFERENCES gateway_providers(id) ON DELETE SET NULL,
    endpoint_id UUID REFERENCES webhook_endpoints(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Connector executions log
CREATE TABLE IF NOT EXISTS connector_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    installed_connector_id UUID NOT NULL REFERENCES installed_connectors(id) ON DELETE CASCADE,
    event_type VARCHAR(255),
    input_payload BYTEA,
    output_payload BYTEA,
    status VARCHAR(50) NOT NULL,
    error TEXT,
    duration_ms BIGINT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_installed_connectors_tenant_id ON installed_connectors(tenant_id);
CREATE INDEX idx_installed_connectors_connector_id ON installed_connectors(connector_id);
CREATE INDEX idx_connector_executions_installed_id ON connector_executions(installed_connector_id);
CREATE INDEX idx_connector_executions_created_at ON connector_executions(created_at);
