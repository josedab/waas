-- Receiver Dashboard tables
CREATE TABLE IF NOT EXISTS receiver_tokens (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    token TEXT NOT NULL UNIQUE,
    endpoint_ids JSONB NOT NULL DEFAULT '[]',
    label TEXT NOT NULL,
    scopes JSONB NOT NULL DEFAULT '[]',
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_receiver_tokens_tenant ON receiver_tokens(tenant_id);
CREATE INDEX idx_receiver_tokens_token ON receiver_tokens(token);

-- Receiver delivery view (materialized for performance)
CREATE TABLE IF NOT EXISTS receiver_delivery_log (
    id TEXT PRIMARY KEY,
    endpoint_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    status_code INT NOT NULL,
    success BOOLEAN NOT NULL DEFAULT FALSE,
    latency_ms BIGINT NOT NULL DEFAULT 0,
    attempt_count INT NOT NULL DEFAULT 1,
    payload_size_bytes INT NOT NULL DEFAULT 0,
    headers JSONB,
    body TEXT,
    content_type TEXT DEFAULT 'application/json',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_receiver_delivery_log_endpoint ON receiver_delivery_log(endpoint_id);
CREATE INDEX idx_receiver_delivery_log_created ON receiver_delivery_log(created_at);

-- Receiver retry tracking
CREATE TABLE IF NOT EXISTS receiver_retry_status (
    delivery_id TEXT PRIMARY KEY REFERENCES receiver_delivery_log(id),
    endpoint_id TEXT NOT NULL,
    current_state TEXT NOT NULL DEFAULT 'pending',
    attempt_count INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 5,
    next_retry_at TIMESTAMP WITH TIME ZONE,
    last_error TEXT,
    last_attempt_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_receiver_retry_endpoint ON receiver_retry_status(endpoint_id);
CREATE INDEX idx_receiver_retry_state ON receiver_retry_status(current_state);

-- Endpoint health snapshots (computed periodically)
CREATE TABLE IF NOT EXISTS receiver_endpoint_health (
    endpoint_id TEXT NOT NULL,
    period TEXT NOT NULL,
    health_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    success_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
    avg_latency_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    p95_latency_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    total_deliveries INT NOT NULL DEFAULT 0,
    failed_deliveries INT NOT NULL DEFAULT 0,
    active_retries INT NOT NULL DEFAULT 0,
    computed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (endpoint_id, period)
);
