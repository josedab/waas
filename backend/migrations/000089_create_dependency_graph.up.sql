-- Webhook dependency graph tables
CREATE TABLE IF NOT EXISTS webhook_dependencies (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    producer_id TEXT NOT NULL,
    consumer_id TEXT NOT NULL,
    event_types JSONB NOT NULL DEFAULT '[]',
    delivery_count BIGINT NOT NULL DEFAULT 0,
    success_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
    avg_latency_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    health_status TEXT NOT NULL DEFAULT 'unknown',
    last_delivery_at TIMESTAMP WITH TIME ZONE,
    discovered_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_refreshed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_deps_tenant ON webhook_dependencies(tenant_id);
CREATE INDEX idx_webhook_deps_producer ON webhook_dependencies(producer_id);
CREATE INDEX idx_webhook_deps_consumer ON webhook_dependencies(consumer_id);
CREATE UNIQUE INDEX idx_webhook_deps_edge ON webhook_dependencies(producer_id, consumer_id);

CREATE TABLE IF NOT EXISTS webhook_endpoint_nodes (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    name TEXT,
    url TEXT,
    node_type TEXT NOT NULL DEFAULT 'unknown',
    health_status TEXT NOT NULL DEFAULT 'unknown',
    event_types JSONB NOT NULL DEFAULT '[]',
    in_degree INT NOT NULL DEFAULT 0,
    out_degree INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_webhook_nodes_tenant ON webhook_endpoint_nodes(tenant_id);
