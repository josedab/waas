CREATE TABLE webhook_endpoints (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    url VARCHAR(2048) NOT NULL,
    secret_hash VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    retry_config JSONB NOT NULL DEFAULT '{"max_attempts": 5, "initial_delay_ms": 1000, "max_delay_ms": 300000, "backoff_multiplier": 2}',
    custom_headers JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_endpoints_tenant ON webhook_endpoints(tenant_id);
CREATE INDEX idx_webhook_endpoints_active ON webhook_endpoints(tenant_id, is_active);
CREATE INDEX idx_webhook_endpoints_url ON webhook_endpoints(url);