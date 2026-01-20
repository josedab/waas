-- Push devices table
CREATE TABLE IF NOT EXISTS push_devices (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    user_id VARCHAR(255),
    platform VARCHAR(50) NOT NULL,
    push_token TEXT NOT NULL,
    device_info JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    preferences JSONB NOT NULL DEFAULT '{}',
    tags JSONB NOT NULL DEFAULT '[]',
    metadata JSONB NOT NULL DEFAULT '{}',
    last_active_at TIMESTAMPTZ,
    registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_push_devices_tenant ON push_devices(tenant_id);
CREATE INDEX idx_push_devices_user ON push_devices(tenant_id, user_id);
CREATE INDEX idx_push_devices_platform ON push_devices(tenant_id, platform);
CREATE INDEX idx_push_devices_status ON push_devices(tenant_id, status);
CREATE UNIQUE INDEX idx_push_devices_token ON push_devices(tenant_id, push_token);

-- Push mappings table (webhook to push mapping)
CREATE TABLE IF NOT EXISTS push_mappings (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    webhook_id UUID,
    event_type VARCHAR(255),
    enabled BOOLEAN NOT NULL DEFAULT true,
    config JSONB NOT NULL DEFAULT '{}',
    template JSONB NOT NULL DEFAULT '{}',
    targeting JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_push_mappings_tenant ON push_mappings(tenant_id);
CREATE INDEX idx_push_mappings_webhook ON push_mappings(tenant_id, webhook_id);
CREATE INDEX idx_push_mappings_enabled ON push_mappings(tenant_id, enabled);

-- Push notifications table
CREATE TABLE IF NOT EXISTS push_notifications (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    mapping_id UUID REFERENCES push_mappings(id) ON DELETE SET NULL,
    webhook_id UUID,
    platform VARCHAR(50) NOT NULL,
    device_id UUID NOT NULL REFERENCES push_devices(id) ON DELETE CASCADE,
    push_token TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    payload JSONB NOT NULL,
    response JSONB,
    attempts INT NOT NULL DEFAULT 0,
    last_attempt TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    opened_at TIMESTAMPTZ,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_push_notifications_tenant ON push_notifications(tenant_id);
CREATE INDEX idx_push_notifications_device ON push_notifications(device_id);
CREATE INDEX idx_push_notifications_mapping ON push_notifications(mapping_id);
CREATE INDEX idx_push_notifications_status ON push_notifications(tenant_id, status);
CREATE INDEX idx_push_notifications_created ON push_notifications(tenant_id, created_at DESC);

-- Push offline queue table
CREATE TABLE IF NOT EXISTS push_offline_queue (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    device_id UUID NOT NULL REFERENCES push_devices(id) ON DELETE CASCADE,
    notification JSONB NOT NULL,
    priority INT NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_push_offline_device ON push_offline_queue(device_id);
CREATE INDEX idx_push_offline_expires ON push_offline_queue(expires_at);

-- Push providers table
CREATE TABLE IF NOT EXISTS push_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    provider VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    config JSONB NOT NULL DEFAULT '{}',
    credentials JSONB NOT NULL DEFAULT '{}', -- Encrypted
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, provider)
);

CREATE INDEX idx_push_providers_tenant ON push_providers(tenant_id);

-- Push segments table
CREATE TABLE IF NOT EXISTS push_segments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    query TEXT NOT NULL,
    device_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_push_segments_tenant ON push_segments(tenant_id);
