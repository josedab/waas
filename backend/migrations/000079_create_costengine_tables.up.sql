CREATE TABLE IF NOT EXISTS cost_models (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    compute_cost_per_delivery DECIMAL(10,6) NOT NULL DEFAULT 0.0001,
    bandwidth_cost_per_kb DECIMAL(10,6) NOT NULL DEFAULT 0.00001,
    retry_cost_multiplier DECIMAL(5,2) NOT NULL DEFAULT 1.5,
    storage_cost_per_gb_day DECIMAL(10,6) NOT NULL DEFAULT 0.023,
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS delivery_costs (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    delivery_id UUID,
    endpoint_id UUID,
    event_type VARCHAR(255),
    compute_cost DECIMAL(10,6) NOT NULL DEFAULT 0,
    bandwidth_cost DECIMAL(10,6) NOT NULL DEFAULT 0,
    retry_cost DECIMAL(10,6) NOT NULL DEFAULT 0,
    total_cost DECIMAL(10,6) NOT NULL DEFAULT 0,
    payload_size_bytes BIGINT NOT NULL DEFAULT 0,
    retry_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS cost_anomalies (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    endpoint_id UUID,
    anomaly_type VARCHAR(100) NOT NULL,
    expected_cost DECIMAL(10,6) NOT NULL,
    actual_cost DECIMAL(10,6) NOT NULL,
    deviation_pct DECIMAL(10,2) NOT NULL,
    detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status VARCHAR(50) NOT NULL DEFAULT 'active'
);

CREATE TABLE IF NOT EXISTS cost_budgets (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    monthly_limit DECIMAL(10,2) NOT NULL,
    alert_threshold_pct DECIMAL(5,2) NOT NULL DEFAULT 80,
    current_spend DECIMAL(10,6) NOT NULL DEFAULT 0,
    period VARCHAR(20) NOT NULL DEFAULT 'monthly',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cost_models_tenant ON cost_models(tenant_id);
CREATE INDEX idx_delivery_costs_tenant ON delivery_costs(tenant_id);
CREATE INDEX idx_delivery_costs_endpoint ON delivery_costs(tenant_id, endpoint_id);
CREATE INDEX idx_delivery_costs_created ON delivery_costs(created_at);
CREATE INDEX idx_cost_anomalies_tenant ON cost_anomalies(tenant_id);
CREATE INDEX idx_cost_budgets_tenant ON cost_budgets(tenant_id);
