-- Regions table
CREATE TABLE IF NOT EXISTS regions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    code VARCHAR(50) NOT NULL UNIQUE,
    endpoint TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    priority INTEGER NOT NULL DEFAULT 0,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Region health tracking
CREATE TABLE IF NOT EXISTS region_health (
    region_id UUID PRIMARY KEY REFERENCES regions(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL,
    last_check TIMESTAMP WITH TIME ZONE NOT NULL,
    latency_ns BIGINT NOT NULL,
    success_rate DOUBLE PRECISION,
    active_connections INTEGER,
    queue_depth INTEGER,
    error_rate DOUBLE PRECISION,
    metrics JSONB
);

-- Failover events
CREATE TABLE IF NOT EXISTS failover_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_region VARCHAR(50) NOT NULL,
    to_region VARCHAR(50) NOT NULL,
    reason VARCHAR(50) NOT NULL,
    trigger_type VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    affected_ops BIGINT DEFAULT 0,
    details TEXT
);

-- Replication configuration
CREATE TABLE IF NOT EXISTS replication_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_region VARCHAR(50) NOT NULL,
    target_region VARCHAR(50) NOT NULL,
    mode VARCHAR(20) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    lag_threshold_ms BIGINT NOT NULL DEFAULT 1000,
    retention_days INTEGER NOT NULL DEFAULT 30,
    tables JSONB,
    last_sync_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(source_region, target_region)
);

-- Routing policies per tenant
CREATE TABLE IF NOT EXISTS routing_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE UNIQUE,
    policy_type VARCHAR(30) NOT NULL,
    primary_region VARCHAR(50) NOT NULL,
    fallback_regions JSONB,
    geo_rules JSONB,
    weights JSONB,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_regions_code ON regions(code);
CREATE INDEX idx_regions_active ON regions(is_active);
CREATE INDEX idx_failover_events_started ON failover_events(started_at DESC);
CREATE INDEX idx_replication_configs_source ON replication_configs(source_region);
CREATE INDEX idx_routing_policies_tenant ON routing_policies(tenant_id);
