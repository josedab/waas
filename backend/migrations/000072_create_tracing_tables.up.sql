CREATE TABLE IF NOT EXISTS traces (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    trace_id VARCHAR(64) NOT NULL,
    root_span_id VARCHAR(64) NOT NULL,
    service_name VARCHAR(255) NOT NULL,
    operation_name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    span_count INTEGER NOT NULL DEFAULT 0,
    duration_ms BIGINT NOT NULL DEFAULT 0,
    has_errors BOOLEAN NOT NULL DEFAULT FALSE,
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS spans (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    trace_id VARCHAR(64) NOT NULL,
    span_id VARCHAR(64) NOT NULL,
    parent_span_id VARCHAR(64),
    operation_name VARCHAR(255) NOT NULL,
    service_name VARCHAR(255) NOT NULL,
    span_kind VARCHAR(50) NOT NULL,
    status_code VARCHAR(50) NOT NULL DEFAULT 'OK',
    status_message TEXT,
    attributes JSONB DEFAULT '{}',
    events JSONB DEFAULT '[]',
    duration_ms BIGINT NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS trace_propagation_configs (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL UNIQUE,
    inject_headers BOOLEAN NOT NULL DEFAULT TRUE,
    inject_payload BOOLEAN NOT NULL DEFAULT FALSE,
    header_prefix VARCHAR(100) NOT NULL DEFAULT 'traceparent',
    payload_field VARCHAR(100) NOT NULL DEFAULT '_trace',
    sampling_rate DECIMAL(3,2) NOT NULL DEFAULT 1.00,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_traces_tenant_id ON traces(tenant_id);
CREATE INDEX idx_traces_trace_id ON traces(tenant_id, trace_id);
CREATE INDEX idx_traces_status ON traces(tenant_id, status);
CREATE INDEX idx_spans_trace_id ON spans(tenant_id, trace_id);
CREATE INDEX idx_spans_span_id ON spans(tenant_id, span_id);
