-- Anomaly baselines
CREATE TABLE IF NOT EXISTS anomaly_baselines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    endpoint_id UUID REFERENCES webhook_endpoints(id) ON DELETE CASCADE,
    metric_type VARCHAR(50) NOT NULL,
    mean DOUBLE PRECISION NOT NULL,
    std_dev DOUBLE PRECISION NOT NULL,
    min_value DOUBLE PRECISION,
    max_value DOUBLE PRECISION,
    p50 DOUBLE PRECISION,
    p95 DOUBLE PRECISION,
    p99 DOUBLE PRECISION,
    sample_size BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, endpoint_id, metric_type)
);

-- Detected anomalies
CREATE TABLE IF NOT EXISTS anomalies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    endpoint_id UUID REFERENCES webhook_endpoints(id) ON DELETE SET NULL,
    metric_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    current_value DOUBLE PRECISION NOT NULL,
    expected_value DOUBLE PRECISION NOT NULL,
    deviation DOUBLE PRECISION NOT NULL,
    deviation_pct DOUBLE PRECISION NOT NULL,
    description TEXT,
    root_cause TEXT,
    recommendation TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'open',
    detected_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMP WITH TIME ZONE
);

-- Detection configuration
CREATE TABLE IF NOT EXISTS anomaly_detection_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    endpoint_id UUID REFERENCES webhook_endpoints(id) ON DELETE CASCADE,
    metric_type VARCHAR(50) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    sensitivity DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    min_samples INTEGER NOT NULL DEFAULT 30,
    cooldown_minutes INTEGER NOT NULL DEFAULT 15,
    critical_threshold DOUBLE PRECISION NOT NULL DEFAULT 3.0,
    warning_threshold DOUBLE PRECISION NOT NULL DEFAULT 2.0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, endpoint_id, metric_type)
);

-- Alert configuration
CREATE TABLE IF NOT EXISTS anomaly_alert_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    channel VARCHAR(50) NOT NULL,
    config TEXT NOT NULL,
    min_severity VARCHAR(20) NOT NULL DEFAULT 'warning',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Alert history
CREATE TABLE IF NOT EXISTS anomaly_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    anomaly_id UUID NOT NULL REFERENCES anomalies(id) ON DELETE CASCADE,
    channel VARCHAR(50) NOT NULL,
    recipient TEXT,
    status VARCHAR(20) NOT NULL,
    sent_at TIMESTAMP WITH TIME ZONE,
    error TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_anomaly_baselines_tenant ON anomaly_baselines(tenant_id);
CREATE INDEX idx_anomaly_baselines_endpoint ON anomaly_baselines(endpoint_id);
CREATE INDEX idx_anomalies_tenant ON anomalies(tenant_id);
CREATE INDEX idx_anomalies_status ON anomalies(tenant_id, status);
CREATE INDEX idx_anomalies_detected_at ON anomalies(tenant_id, detected_at);
CREATE INDEX idx_anomaly_detection_configs_tenant ON anomaly_detection_configs(tenant_id);
CREATE INDEX idx_anomaly_alert_configs_tenant ON anomaly_alert_configs(tenant_id);
CREATE INDEX idx_anomaly_alerts_anomaly ON anomaly_alerts(anomaly_id);
