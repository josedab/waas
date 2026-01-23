CREATE TABLE IF NOT EXISTS sla_targets (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    endpoint_id VARCHAR(36),
    name VARCHAR(255) NOT NULL,
    delivery_rate_pct DECIMAL(5,2) NOT NULL DEFAULT 99.9,
    latency_p50_ms INTEGER NOT NULL DEFAULT 0,
    latency_p99_ms INTEGER NOT NULL DEFAULT 0,
    window_minutes INTEGER NOT NULL DEFAULT 60,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sla_breaches (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    target_id VARCHAR(36) NOT NULL REFERENCES sla_targets(id) ON DELETE CASCADE,
    endpoint_id VARCHAR(36),
    breach_type VARCHAR(50) NOT NULL,
    expected_value DECIMAL(10,2) NOT NULL,
    actual_value DECIMAL(10,2) NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'warning',
    resolved_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sla_alert_configs (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    target_id VARCHAR(36) NOT NULL REFERENCES sla_targets(id) ON DELETE CASCADE,
    channels JSONB NOT NULL DEFAULT '[]',
    cooldown_minutes INTEGER NOT NULL DEFAULT 15,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, target_id)
);

CREATE INDEX idx_sla_targets_tenant ON sla_targets(tenant_id);
CREATE INDEX idx_sla_breaches_tenant ON sla_breaches(tenant_id);
CREATE INDEX idx_sla_breaches_active ON sla_breaches(tenant_id) WHERE resolved_at IS NULL;
CREATE INDEX idx_sla_alert_configs_tenant ON sla_alert_configs(tenant_id, target_id);
