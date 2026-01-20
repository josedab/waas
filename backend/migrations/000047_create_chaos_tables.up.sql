-- Chaos engineering tables

CREATE TABLE IF NOT EXISTS chaos_experiments (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(30) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    target_config JSONB NOT NULL DEFAULT '{}',
    fault_config JSONB NOT NULL DEFAULT '{}',
    schedule JSONB,
    blast_radius JSONB NOT NULL DEFAULT '{}',
    duration_seconds INTEGER NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    results JSONB,
    created_by VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_chaos_exp_tenant ON chaos_experiments(tenant_id);
CREATE INDEX idx_chaos_exp_status ON chaos_experiments(tenant_id, status);
CREATE INDEX idx_chaos_exp_created ON chaos_experiments(tenant_id, created_at);

CREATE TABLE IF NOT EXISTS chaos_events (
    id VARCHAR(36) PRIMARY KEY,
    experiment_id VARCHAR(36) NOT NULL REFERENCES chaos_experiments(id),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    endpoint_id VARCHAR(36) NOT NULL,
    delivery_id VARCHAR(36),
    event_type VARCHAR(30) NOT NULL,
    injected_fault VARCHAR(30),
    original_state TEXT,
    injected_state TEXT,
    recovered BOOLEAN DEFAULT false,
    recovery_time_ms BIGINT,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE INDEX idx_chaos_events_exp ON chaos_events(tenant_id, experiment_id);
CREATE INDEX idx_chaos_events_delivery ON chaos_events(tenant_id, delivery_id);
CREATE INDEX idx_chaos_events_timestamp ON chaos_events(tenant_id, timestamp);
