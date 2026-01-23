CREATE TABLE IF NOT EXISTS sandbox_environments (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    target_url TEXT NOT NULL,
    masking_rules JSONB NOT NULL DEFAULT '[]',
    ttl_minutes INTEGER NOT NULL DEFAULT 60,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS replay_sessions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    sandbox_id UUID NOT NULL REFERENCES sandbox_environments(id),
    source_event_id VARCHAR(255) NOT NULL,
    original_payload TEXT,
    masked_payload TEXT,
    response_status INTEGER,
    response_body TEXT,
    response_latency_ms BIGINT,
    comparison_result TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sandbox_environments_tenant ON sandbox_environments(tenant_id);
CREATE INDEX idx_sandbox_environments_status ON sandbox_environments(tenant_id, status);
CREATE INDEX idx_replay_sessions_sandbox ON replay_sessions(sandbox_id);
