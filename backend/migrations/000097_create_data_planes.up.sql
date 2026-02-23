-- Multi-Tenant Dedicated Data Planes

CREATE TABLE IF NOT EXISTS data_planes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       VARCHAR(255) NOT NULL UNIQUE,
    plane_type      VARCHAR(50) NOT NULL DEFAULT 'shared',
    status          VARCHAR(50) NOT NULL DEFAULT 'provisioning',
    db_schema       VARCHAR(255) NOT NULL DEFAULT '',
    redis_namespace VARCHAR(255) NOT NULL DEFAULT '',
    worker_pool_id  VARCHAR(255) NOT NULL DEFAULT '',
    config          JSONB NOT NULL DEFAULT '{}',
    region          VARCHAR(100) NOT NULL DEFAULT 'us-east-1',
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_data_planes_tenant ON data_planes(tenant_id);
CREATE INDEX idx_data_planes_status ON data_planes(status);
