-- Delivery archives for long-term storage and replay
CREATE TABLE IF NOT EXISTS delivery_archives (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    endpoint_id UUID NOT NULL,
    endpoint_url TEXT NOT NULL,
    payload JSONB NOT NULL,
    headers JSONB,
    status VARCHAR(50) NOT NULL,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    last_http_status INTEGER,
    last_error TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Replay snapshots
CREATE TABLE IF NOT EXISTS replay_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    filters JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE
);

-- Snapshot delivery associations
CREATE TABLE IF NOT EXISTS snapshot_deliveries (
    snapshot_id UUID NOT NULL REFERENCES replay_snapshots(id) ON DELETE CASCADE,
    delivery_id UUID NOT NULL,
    PRIMARY KEY (snapshot_id, delivery_id)
);

-- Indexes
CREATE INDEX idx_delivery_archives_tenant_id ON delivery_archives(tenant_id);
CREATE INDEX idx_delivery_archives_endpoint_id ON delivery_archives(endpoint_id);
CREATE INDEX idx_delivery_archives_status ON delivery_archives(tenant_id, status);
CREATE INDEX idx_delivery_archives_created_at ON delivery_archives(tenant_id, created_at);
CREATE INDEX idx_replay_snapshots_tenant_id ON replay_snapshots(tenant_id);
CREATE INDEX idx_replay_snapshots_expires_at ON replay_snapshots(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_snapshot_deliveries_delivery_id ON snapshot_deliveries(delivery_id);
