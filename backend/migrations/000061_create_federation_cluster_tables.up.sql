-- Multi-Cloud Federation tables (extends 000054 federation tables)

-- Federation clusters
CREATE TABLE IF NOT EXISTS federation_clusters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    provider VARCHAR(50) NOT NULL, -- aws_eventbridge, azure_eventgrid, gcp_pubsub, kafka, custom
    region VARCHAR(100) NOT NULL,
    zone VARCHAR(100),
    endpoint VARCHAR(500) NOT NULL,
    api_endpoint VARCHAR(500),
    status VARCHAR(20) NOT NULL DEFAULT 'healthy', -- healthy, degraded, unhealthy, draining, offline
    priority INT NOT NULL DEFAULT 100,
    weight INT NOT NULL DEFAULT 100,
    capacity JSONB NOT NULL DEFAULT '{}',
    metrics JSONB NOT NULL DEFAULT '{}',
    config JSONB NOT NULL DEFAULT '{}',
    tags JSONB DEFAULT '{}',
    last_health_check TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_federation_cluster UNIQUE(tenant_id, name)
);

CREATE INDEX idx_federation_clusters_tenant ON federation_clusters(tenant_id);
CREATE INDEX idx_federation_clusters_provider ON federation_clusters(tenant_id, provider);
CREATE INDEX idx_federation_clusters_status ON federation_clusters(tenant_id, status);
CREATE INDEX idx_federation_clusters_region ON federation_clusters(region);

-- Cluster credentials (encrypted)
CREATE TABLE IF NOT EXISTS federation_cluster_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id UUID NOT NULL REFERENCES federation_clusters(id) ON DELETE CASCADE,
    credential_type VARCHAR(50) NOT NULL, -- api_key, service_account, mtls
    encrypted_data BYTEA NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_federation_cluster_credentials ON federation_cluster_credentials(cluster_id);

-- Federation routes
CREATE TABLE IF NOT EXISTS federation_routes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    strategy VARCHAR(50) NOT NULL DEFAULT 'latency', -- latency, geo, round_robin, weighted, failover, cost
    failover_mode VARCHAR(20) NOT NULL DEFAULT 'automatic', -- automatic, manual, none
    clusters JSONB NOT NULL DEFAULT '[]',
    rules JSONB DEFAULT '[]',
    health_check JSONB NOT NULL DEFAULT '{}',
    active BOOLEAN NOT NULL DEFAULT true,
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_federation_route UNIQUE(tenant_id, name)
);

CREATE INDEX idx_federation_routes_tenant ON federation_routes(tenant_id);
CREATE INDEX idx_federation_routes_active ON federation_routes(tenant_id, active);
CREATE INDEX idx_federation_routes_default ON federation_routes(tenant_id, is_default);

-- Failover events
CREATE TABLE IF NOT EXISTS federation_failover_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    route_id UUID NOT NULL REFERENCES federation_routes(id) ON DELETE CASCADE,
    from_cluster_id UUID NOT NULL REFERENCES federation_clusters(id),
    to_cluster_id UUID NOT NULL REFERENCES federation_clusters(id),
    reason TEXT NOT NULL,
    automatic BOOLEAN NOT NULL DEFAULT false,
    duration_ms BIGINT,
    status VARCHAR(20) NOT NULL DEFAULT 'initiated', -- initiated, completed, failed, rolled_back
    initiated_by VARCHAR(255),
    initiated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_federation_failover_events_tenant ON federation_failover_events(tenant_id);
CREATE INDEX idx_federation_failover_events_route ON federation_failover_events(route_id);
CREATE INDEX idx_federation_failover_events_initiated ON federation_failover_events(initiated_at);

-- Replication configs
CREATE TABLE IF NOT EXISTS federation_replication_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    source_cluster_id UUID NOT NULL REFERENCES federation_clusters(id),
    target_cluster_id UUID NOT NULL REFERENCES federation_clusters(id),
    mode VARCHAR(20) NOT NULL DEFAULT 'async', -- sync, async
    lag_threshold_ms BIGINT NOT NULL DEFAULT 5000,
    active BOOLEAN NOT NULL DEFAULT true,
    status JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_replication_config UNIQUE(source_cluster_id, target_cluster_id)
);

CREATE INDEX idx_federation_replication_configs_tenant ON federation_replication_configs(tenant_id);
CREATE INDEX idx_federation_replication_configs_source ON federation_replication_configs(source_cluster_id);

-- Cross-cluster routing metrics
CREATE TABLE IF NOT EXISTS federation_routing_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    cluster_id UUID NOT NULL REFERENCES federation_clusters(id) ON DELETE CASCADE,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    requests_total BIGINT NOT NULL DEFAULT 0,
    requests_success BIGINT NOT NULL DEFAULT 0,
    requests_failed BIGINT NOT NULL DEFAULT 0,
    latency_p50_ms INT,
    latency_p99_ms INT,
    bytes_transferred BIGINT NOT NULL DEFAULT 0,
    
    CONSTRAINT unique_routing_metrics UNIQUE(cluster_id, timestamp)
);

CREATE INDEX idx_federation_routing_metrics_cluster ON federation_routing_metrics(cluster_id, timestamp DESC);
CREATE INDEX idx_federation_routing_metrics_tenant ON federation_routing_metrics(tenant_id, timestamp DESC);

-- Triggers
CREATE TRIGGER trigger_federation_clusters_updated
    BEFORE UPDATE ON federation_clusters
    FOR EACH ROW
    EXECUTE FUNCTION update_stream_config_timestamp();

CREATE TRIGGER trigger_federation_cluster_credentials_updated
    BEFORE UPDATE ON federation_cluster_credentials
    FOR EACH ROW
    EXECUTE FUNCTION update_stream_config_timestamp();

CREATE TRIGGER trigger_federation_routes_updated
    BEFORE UPDATE ON federation_routes
    FOR EACH ROW
    EXECUTE FUNCTION update_stream_config_timestamp();

CREATE TRIGGER trigger_federation_replication_configs_updated
    BEFORE UPDATE ON federation_replication_configs
    FOR EACH ROW
    EXECUTE FUNCTION update_stream_config_timestamp();
