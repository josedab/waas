CREATE TABLE IF NOT EXISTS delivery_traces (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    delivery_id VARCHAR(36) NOT NULL,
    endpoint_id VARCHAR(36) NOT NULL,
    stages JSONB NOT NULL DEFAULT '[]',
    total_duration_ms INTEGER NOT NULL DEFAULT 0,
    final_status VARCHAR(50) NOT NULL DEFAULT 'unknown',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS debug_sessions (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    delivery_id VARCHAR(36) NOT NULL,
    current_step INTEGER NOT NULL DEFAULT 0,
    breakpoints JSONB NOT NULL DEFAULT '[]',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE INDEX idx_traces_tenant ON delivery_traces(tenant_id);
CREATE INDEX idx_traces_delivery ON delivery_traces(delivery_id);
CREATE INDEX idx_traces_endpoint ON delivery_traces(tenant_id, endpoint_id);
CREATE INDEX idx_debug_sessions_tenant ON debug_sessions(tenant_id);
