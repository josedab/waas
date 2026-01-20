-- Feature 9: Federated Multi-Region Mesh
-- Enables multi-region deployment with geo-routing and data residency compliance

-- Regions table
CREATE TABLE regions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    code VARCHAR(20) NOT NULL UNIQUE,
    provider VARCHAR(50) NOT NULL, -- aws, gcp, azure
    location VARCHAR(100) NOT NULL,
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    is_primary BOOLEAN DEFAULT FALSE,
    health_status VARCHAR(20) DEFAULT 'healthy',
    last_health_check TIMESTAMP WITH TIME ZONE,
    capacity_limit INTEGER DEFAULT 10000,
    current_load INTEGER DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Region clusters (groupings for failover)
CREATE TABLE region_clusters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    primary_region_id UUID REFERENCES regions(id),
    failover_strategy VARCHAR(50) DEFAULT 'round_robin', -- round_robin, priority, latency_based
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Region cluster members
CREATE TABLE region_cluster_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id UUID NOT NULL REFERENCES region_clusters(id) ON DELETE CASCADE,
    region_id UUID NOT NULL REFERENCES regions(id) ON DELETE CASCADE,
    priority INTEGER DEFAULT 0,
    weight DECIMAL(5, 2) DEFAULT 1.0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(cluster_id, region_id)
);

-- Tenant region assignments (data residency)
CREATE TABLE tenant_regions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    primary_region_id UUID NOT NULL REFERENCES regions(id),
    allowed_regions UUID[] DEFAULT '{}',
    data_residency_policy VARCHAR(50) DEFAULT 'flexible', -- strict, flexible, global
    replication_mode VARCHAR(50) DEFAULT 'async', -- sync, async, none
    compliance_frameworks TEXT[] DEFAULT '{}', -- GDPR, CCPA, HIPAA, etc.
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id)
);

-- Geo-routing rules
CREATE TABLE geo_routing_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    rule_type VARCHAR(50) NOT NULL, -- latency, geofence, load_balance, failover
    priority INTEGER DEFAULT 0,
    source_regions UUID[] DEFAULT '{}',
    target_region_id UUID REFERENCES regions(id),
    conditions JSONB DEFAULT '{}',
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Cross-region replication tracking
CREATE TABLE replication_streams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    source_region_id UUID NOT NULL REFERENCES regions(id),
    target_region_id UUID NOT NULL REFERENCES regions(id),
    stream_type VARCHAR(50) NOT NULL, -- events, configs, state
    status VARCHAR(50) DEFAULT 'active',
    lag_ms BIGINT DEFAULT 0,
    last_replicated_at TIMESTAMP WITH TIME ZONE,
    last_event_id UUID,
    error_message TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, source_region_id, target_region_id, stream_type)
);

-- Regional event routing decisions
CREATE TABLE regional_routing_decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    event_id UUID NOT NULL,
    source_region_id UUID NOT NULL REFERENCES regions(id),
    target_region_id UUID NOT NULL REFERENCES regions(id),
    routing_rule_id UUID REFERENCES geo_routing_rules(id),
    decision_reason VARCHAR(255),
    latency_ms INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Regional health metrics
CREATE TABLE region_health_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    region_id UUID NOT NULL REFERENCES regions(id),
    metric_type VARCHAR(50) NOT NULL, -- latency, error_rate, throughput, capacity
    metric_value DECIMAL(15, 4) NOT NULL,
    recorded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Data residency audit log
CREATE TABLE data_residency_audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    source_region_id UUID REFERENCES regions(id),
    target_region_id UUID REFERENCES regions(id),
    data_type VARCHAR(100),
    compliance_status VARCHAR(50), -- compliant, violation, warning
    details JSONB DEFAULT '{}',
    recorded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Failover events
CREATE TABLE failover_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id UUID REFERENCES region_clusters(id),
    from_region_id UUID NOT NULL REFERENCES regions(id),
    to_region_id UUID NOT NULL REFERENCES regions(id),
    trigger_reason VARCHAR(255),
    automatic BOOLEAN DEFAULT TRUE,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(50) DEFAULT 'in_progress',
    affected_tenants INTEGER DEFAULT 0,
    metadata JSONB DEFAULT '{}'
);

-- Regional configuration sync
CREATE TABLE regional_config_sync (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    config_type VARCHAR(100) NOT NULL, -- endpoint, subscription, transformation
    config_id UUID NOT NULL,
    region_id UUID NOT NULL REFERENCES regions(id),
    version INTEGER DEFAULT 1,
    sync_status VARCHAR(50) DEFAULT 'pending', -- pending, synced, conflict, failed
    last_synced_at TIMESTAMP WITH TIME ZONE,
    config_hash VARCHAR(64),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(config_type, config_id, region_id)
);

-- Create indexes
CREATE INDEX idx_regions_status ON regions(status);
CREATE INDEX idx_regions_provider ON regions(provider);
CREATE INDEX idx_tenant_regions_tenant ON tenant_regions(tenant_id);
CREATE INDEX idx_geo_routing_rules_tenant ON geo_routing_rules(tenant_id);
CREATE INDEX idx_geo_routing_rules_enabled ON geo_routing_rules(tenant_id, enabled);
CREATE INDEX idx_replication_streams_tenant ON replication_streams(tenant_id);
CREATE INDEX idx_replication_streams_status ON replication_streams(status);
CREATE INDEX idx_regional_routing_decisions_tenant ON regional_routing_decisions(tenant_id);
CREATE INDEX idx_regional_routing_decisions_event ON regional_routing_decisions(event_id);
CREATE INDEX idx_regional_routing_decisions_created ON regional_routing_decisions(created_at);
CREATE INDEX idx_region_health_metrics_region ON region_health_metrics(region_id);
CREATE INDEX idx_region_health_metrics_recorded ON region_health_metrics(recorded_at);
CREATE INDEX idx_data_residency_audit_tenant ON data_residency_audit(tenant_id);
CREATE INDEX idx_data_residency_audit_recorded ON data_residency_audit(recorded_at);
CREATE INDEX idx_failover_events_cluster ON failover_events(cluster_id);
CREATE INDEX idx_failover_events_started ON failover_events(started_at);
CREATE INDEX idx_regional_config_sync_tenant ON regional_config_sync(tenant_id);
CREATE INDEX idx_regional_config_sync_status ON regional_config_sync(sync_status);

-- Seed default regions
INSERT INTO regions (name, code, provider, location, latitude, longitude, status, is_primary) VALUES
    ('US East (Virginia)', 'us-east-1', 'aws', 'N. Virginia, USA', 38.9072, -77.0369, 'active', TRUE),
    ('US West (Oregon)', 'us-west-2', 'aws', 'Oregon, USA', 45.5152, -122.6784, 'active', FALSE),
    ('EU West (Ireland)', 'eu-west-1', 'aws', 'Dublin, Ireland', 53.3498, -6.2603, 'active', FALSE),
    ('EU Central (Frankfurt)', 'eu-central-1', 'aws', 'Frankfurt, Germany', 50.1109, 8.6821, 'active', FALSE),
    ('Asia Pacific (Tokyo)', 'ap-northeast-1', 'aws', 'Tokyo, Japan', 35.6762, 139.6503, 'active', FALSE),
    ('Asia Pacific (Singapore)', 'ap-southeast-1', 'aws', 'Singapore', 1.3521, 103.8198, 'active', FALSE),
    ('Australia (Sydney)', 'ap-southeast-2', 'aws', 'Sydney, Australia', -33.8688, 151.2093, 'active', FALSE),
    ('South America (São Paulo)', 'sa-east-1', 'aws', 'São Paulo, Brazil', -23.5505, -46.6333, 'active', FALSE);
