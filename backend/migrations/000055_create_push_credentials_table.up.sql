-- Add provider credentials table for push notification services
CREATE TABLE IF NOT EXISTS push_provider_credentials (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    credentials JSONB NOT NULL,
    environment VARCHAR(50) NOT NULL DEFAULT 'production',
    is_default BOOLEAN NOT NULL DEFAULT false,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    last_used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_push_provider_credentials_tenant ON push_provider_credentials(tenant_id);
CREATE INDEX idx_push_provider_credentials_provider ON push_provider_credentials(tenant_id, provider);
CREATE UNIQUE INDEX idx_push_provider_credentials_default ON push_provider_credentials(tenant_id, provider) WHERE is_default = true;

COMMENT ON TABLE push_provider_credentials IS 'Stores credentials for push notification providers (FCM, APNs, Web Push, Huawei)';
