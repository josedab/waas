CREATE TABLE IF NOT EXISTS canary_deployments (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    endpoint_id UUID NOT NULL,
    event_type VARCHAR(255) NOT NULL,
    traffic_pct INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    promotion_rule VARCHAR(50) NOT NULL DEFAULT 'manual',
    rollback_on_error BOOLEAN NOT NULL DEFAULT TRUE,
    error_threshold_pct DECIMAL(5,2) NOT NULL DEFAULT 5.00,
    soak_time_minutes INTEGER NOT NULL DEFAULT 30,
    promoted_at TIMESTAMPTZ,
    rolled_back_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS canary_metrics (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    deployment_id UUID NOT NULL REFERENCES canary_deployments(id),
    window_start TIMESTAMPTZ NOT NULL,
    window_end TIMESTAMPTZ NOT NULL,
    canary_requests INTEGER NOT NULL DEFAULT 0,
    canary_errors INTEGER NOT NULL DEFAULT 0,
    canary_p50_ms BIGINT NOT NULL DEFAULT 0,
    canary_p99_ms BIGINT NOT NULL DEFAULT 0,
    baseline_requests INTEGER NOT NULL DEFAULT 0,
    baseline_errors INTEGER NOT NULL DEFAULT 0,
    baseline_p50_ms BIGINT NOT NULL DEFAULT 0,
    baseline_p99_ms BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX idx_canary_deployments_tenant ON canary_deployments(tenant_id);
CREATE INDEX idx_canary_deployments_status ON canary_deployments(tenant_id, status);
CREATE INDEX idx_canary_metrics_deployment ON canary_metrics(deployment_id);
