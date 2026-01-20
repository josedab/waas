-- Geographic routing tables
CREATE TABLE IF NOT EXISTS region_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    region VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    endpoint TEXT NOT NULL,
    is_active BOOLEAN DEFAULT true,
    is_primary BOOLEAN DEFAULT false,
    priority INTEGER DEFAULT 0,
    max_concurrent INTEGER DEFAULT 1000,
    health_check JSONB NOT NULL DEFAULT '{}',
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_region_configs_region ON region_configs(region);
CREATE INDEX idx_region_configs_is_active ON region_configs(is_active);
CREATE INDEX idx_region_configs_priority ON region_configs(priority);

-- Endpoint routing configuration
CREATE TABLE IF NOT EXISTS endpoint_routing (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    mode VARCHAR(50) NOT NULL DEFAULT 'auto',
    primary_region VARCHAR(50),
    regions TEXT[] DEFAULT ARRAY['us-east-1'],
    data_residency VARCHAR(50) DEFAULT 'none',
    failover_enabled BOOLEAN DEFAULT true,
    latency_based BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(endpoint_id, tenant_id)
);

CREATE INDEX idx_endpoint_routing_endpoint_id ON endpoint_routing(endpoint_id);
CREATE INDEX idx_endpoint_routing_tenant_id ON endpoint_routing(tenant_id);

-- Region health tracking
CREATE TABLE IF NOT EXISTS region_health (
    region_id VARCHAR(255) PRIMARY KEY,
    is_healthy BOOLEAN DEFAULT true,
    last_check TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    consecutive_ok INTEGER DEFAULT 0,
    consecutive_fail INTEGER DEFAULT 0,
    avg_latency_ms INTEGER DEFAULT 0,
    error_rate DECIMAL(5,4) DEFAULT 0,
    last_error TEXT,
    last_error_at TIMESTAMP WITH TIME ZONE
);

-- Routing decisions log
CREATE TABLE IF NOT EXISTS routing_decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id UUID NOT NULL,
    selected_region VARCHAR(50) NOT NULL,
    reason TEXT,
    latency_ms INTEGER DEFAULT 0,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_routing_decisions_endpoint_id ON routing_decisions(endpoint_id);
CREATE INDEX idx_routing_decisions_selected_region ON routing_decisions(selected_region);
CREATE INDEX idx_routing_decisions_timestamp ON routing_decisions(timestamp);

-- Insert default regions
INSERT INTO region_configs (region, name, endpoint, is_active, is_primary, priority, metadata) VALUES
('us-east-1', 'US East (N. Virginia)', 'https://us-east-1.waas.io', true, true, 1, '{"latitude": 39.0438, "longitude": -77.4874, "country": "US", "continents": ["North America"]}'),
('us-west-2', 'US West (Oregon)', 'https://us-west-2.waas.io', true, false, 2, '{"latitude": 45.8696, "longitude": -119.6880, "country": "US", "continents": ["North America"]}'),
('eu-west-1', 'EU West (Ireland)', 'https://eu-west-1.waas.io', true, false, 3, '{"latitude": 53.3498, "longitude": -6.2603, "country": "IE", "continents": ["Europe"]}'),
('eu-central-1', 'EU Central (Frankfurt)', 'https://eu-central-1.waas.io', true, false, 4, '{"latitude": 50.1109, "longitude": 8.6821, "country": "DE", "continents": ["Europe"]}'),
('ap-south-1', 'Asia Pacific (Mumbai)', 'https://ap-south-1.waas.io', true, false, 5, '{"latitude": 19.0760, "longitude": 72.8777, "country": "IN", "continents": ["Asia"]}'),
('ap-northeast-1', 'Asia Pacific (Tokyo)', 'https://ap-northeast-1.waas.io', true, false, 6, '{"latitude": 35.6762, "longitude": 139.6503, "country": "JP", "continents": ["Asia"]}')
ON CONFLICT (region) DO NOTHING;
