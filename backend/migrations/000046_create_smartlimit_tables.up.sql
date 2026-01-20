-- Smart rate limiting tables

CREATE TABLE IF NOT EXISTS endpoint_behaviors (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    endpoint_id VARCHAR(36) NOT NULL,
    url TEXT,
    window_start TIMESTAMP WITH TIME ZONE NOT NULL,
    window_end TIMESTAMP WITH TIME ZONE NOT NULL,
    total_requests BIGINT DEFAULT 0,
    success_count BIGINT DEFAULT 0,
    rate_limit_count BIGINT DEFAULT 0,
    timeout_count BIGINT DEFAULT 0,
    error_count BIGINT DEFAULT 0,
    avg_latency_ms DOUBLE PRECISION DEFAULT 0,
    p50_latency_ms DOUBLE PRECISION DEFAULT 0,
    p95_latency_ms DOUBLE PRECISION DEFAULT 0,
    p99_latency_ms DOUBLE PRECISION DEFAULT 0,
    max_latency_ms DOUBLE PRECISION DEFAULT 0,
    avg_response_size BIGINT DEFAULT 0,
    status_codes JSONB DEFAULT '{}',
    hourly_pattern JSONB DEFAULT '[]',
    day_of_week_pattern JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_behaviors_tenant_endpoint ON endpoint_behaviors(tenant_id, endpoint_id);
CREATE INDEX idx_behaviors_window ON endpoint_behaviors(tenant_id, endpoint_id, window_end);

CREATE TABLE IF NOT EXISTS adaptive_rate_configs (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    endpoint_id VARCHAR(36) NOT NULL,
    enabled BOOLEAN DEFAULT true,
    mode VARCHAR(20) DEFAULT 'adaptive',
    base_rate_per_sec DOUBLE PRECISION DEFAULT 10,
    min_rate_per_sec DOUBLE PRECISION DEFAULT 1,
    max_rate_per_sec DOUBLE PRECISION DEFAULT 100,
    burst_size INTEGER DEFAULT 10,
    risk_threshold DOUBLE PRECISION DEFAULT 0.7,
    backoff_factor DOUBLE PRECISION DEFAULT 0.5,
    recovery_factor DOUBLE PRECISION DEFAULT 1.1,
    window_seconds INTEGER DEFAULT 60,
    learning_enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, endpoint_id)
);

CREATE INDEX idx_rate_configs_tenant ON adaptive_rate_configs(tenant_id);

CREATE TABLE IF NOT EXISTS rate_limit_states (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    endpoint_id VARCHAR(36) NOT NULL,
    current_rate DOUBLE PRECISION DEFAULT 0,
    allowed_rate DOUBLE PRECISION DEFAULT 0,
    window_start TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    request_count BIGINT DEFAULT 0,
    rate_limit_hits BIGINT DEFAULT 0,
    consecutive_ok INTEGER DEFAULT 0,
    consecutive_fail INTEGER DEFAULT 0,
    last_rate_limit_at TIMESTAMP WITH TIME ZONE,
    retry_after TIMESTAMP WITH TIME ZONE,
    cooldown BOOLEAN DEFAULT false,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, endpoint_id)
);

CREATE INDEX idx_rate_states_tenant ON rate_limit_states(tenant_id);

CREATE TABLE IF NOT EXISTS rate_limit_learning_data (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    endpoint_id VARCHAR(36) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    hour_of_day INTEGER NOT NULL,
    day_of_week INTEGER NOT NULL,
    request_rate DOUBLE PRECISION DEFAULT 0,
    success_rate DOUBLE PRECISION DEFAULT 0,
    avg_latency_ms DOUBLE PRECISION DEFAULT 0,
    rate_limited BOOLEAN DEFAULT false,
    response_code INTEGER
);

CREATE INDEX idx_learning_data_tenant_endpoint ON rate_limit_learning_data(tenant_id, endpoint_id, timestamp);

CREATE TABLE IF NOT EXISTS prediction_models (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    endpoint_id VARCHAR(36) NOT NULL,
    model_type VARCHAR(20) DEFAULT 'linear',
    version INTEGER DEFAULT 1,
    weights JSONB DEFAULT '[]',
    coefficients JSONB DEFAULT '{}',
    features JSONB DEFAULT '[]',
    accuracy DOUBLE PRECISION DEFAULT 0,
    trained_at TIMESTAMP WITH TIME ZONE NOT NULL,
    data_point_count BIGINT DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_prediction_models_tenant ON prediction_models(tenant_id, endpoint_id, is_active);

CREATE TABLE IF NOT EXISTS rate_limit_events (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    endpoint_id VARCHAR(36) NOT NULL,
    delivery_id VARCHAR(36),
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    event_type VARCHAR(20) NOT NULL,
    status_code INTEGER,
    retry_after_seconds INTEGER DEFAULT 0,
    request_rate DOUBLE PRECISION DEFAULT 0,
    headers JSONB DEFAULT '{}'
);

CREATE INDEX idx_rate_events_tenant_endpoint ON rate_limit_events(tenant_id, endpoint_id, timestamp);
CREATE INDEX idx_rate_events_type ON rate_limit_events(tenant_id, event_type, timestamp);
