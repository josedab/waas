CREATE TABLE IF NOT EXISTS cloud_tenants (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL,
    plan VARCHAR(20) NOT NULL DEFAULT 'free',
    region VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'provisioning',
    api_key_hash VARCHAR(255),
    namespace VARCHAR(100) NOT NULL,
    resource_quota JSONB,
    provisioned_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS cloud_usage_metrics (
    tenant_id VARCHAR(36) NOT NULL REFERENCES cloud_tenants(id) ON DELETE CASCADE,
    period VARCHAR(7) NOT NULL,
    deliveries_count INTEGER NOT NULL DEFAULT 0,
    endpoints_count INTEGER NOT NULL DEFAULT 0,
    storage_used_mb DECIMAL(10,2) NOT NULL DEFAULT 0,
    bandwidth_mb DECIMAL(10,2) NOT NULL DEFAULT 0,
    api_calls_count INTEGER NOT NULL DEFAULT 0,
    measured_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (tenant_id, period)
);

CREATE TABLE IF NOT EXISTS cloud_scaling_configs (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL REFERENCES cloud_tenants(id) ON DELETE CASCADE,
    min_workers INTEGER NOT NULL DEFAULT 1,
    max_workers INTEGER NOT NULL DEFAULT 10,
    target_cpu_pct INTEGER NOT NULL DEFAULT 70,
    scale_up_delay_secs INTEGER NOT NULL DEFAULT 60,
    scale_down_delay_secs INTEGER NOT NULL DEFAULT 300,
    enabled BOOLEAN NOT NULL DEFAULT true,
    UNIQUE(tenant_id)
);

CREATE INDEX idx_cloud_tenants_status ON cloud_tenants(status);
CREATE INDEX idx_cloud_tenants_plan ON cloud_tenants(plan);
CREATE INDEX idx_cloud_usage_tenant ON cloud_usage_metrics(tenant_id);
