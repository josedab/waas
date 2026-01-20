-- Idempotency keys table for request deduplication
CREATE TABLE IF NOT EXISTS idempotency_keys (
    key VARCHAR(255) NOT NULL,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    request_hash VARCHAR(64),
    response JSONB,
    status_code INTEGER,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    is_processing BOOLEAN NOT NULL DEFAULT TRUE,
    PRIMARY KEY (tenant_id, key)
);

-- Index for cleanup queries
CREATE INDEX idx_idempotency_keys_expires_at ON idempotency_keys(expires_at);

-- Index for tenant lookups
CREATE INDEX idx_idempotency_keys_tenant_id ON idempotency_keys(tenant_id);
