-- Observability tables for distributed tracing

CREATE TABLE IF NOT EXISTS webhook_spans (
    id VARCHAR(36) PRIMARY KEY,
    trace_id VARCHAR(32) NOT NULL,
    span_id VARCHAR(16) NOT NULL,
    parent_span_id VARCHAR(16),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    webhook_id VARCHAR(36),
    endpoint_id VARCHAR(36),
    delivery_id VARCHAR(36),
    operation_name VARCHAR(255) NOT NULL,
    service_name VARCHAR(255) NOT NULL,
    kind VARCHAR(20) NOT NULL DEFAULT 'internal',
    status VARCHAR(20) NOT NULL DEFAULT 'unset',
    status_message TEXT,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE,
    duration_ms BIGINT,
    attributes JSONB DEFAULT '{}',
    events JSONB DEFAULT '[]',
    links JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_spans_tenant_trace ON webhook_spans(tenant_id, trace_id);
CREATE INDEX idx_spans_tenant_time ON webhook_spans(tenant_id, start_time);
CREATE INDEX idx_spans_trace_id ON webhook_spans(trace_id);
CREATE INDEX idx_spans_parent ON webhook_spans(parent_span_id) WHERE parent_span_id IS NOT NULL;
CREATE INDEX idx_spans_service ON webhook_spans(tenant_id, service_name, start_time);
CREATE INDEX idx_spans_webhook ON webhook_spans(tenant_id, webhook_id) WHERE webhook_id IS NOT NULL;
CREATE INDEX idx_spans_endpoint ON webhook_spans(tenant_id, endpoint_id) WHERE endpoint_id IS NOT NULL;
CREATE INDEX idx_spans_status ON webhook_spans(tenant_id, status, start_time);

CREATE TABLE IF NOT EXISTS otel_export_configs (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    enabled BOOLEAN DEFAULT true,
    protocol VARCHAR(20) NOT NULL,
    endpoint VARCHAR(500) NOT NULL,
    headers JSONB DEFAULT '{}',
    sampling JSONB DEFAULT '{"strategy": "always"}',
    batch_size INTEGER DEFAULT 100,
    timeout_seconds INTEGER DEFAULT 30,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, name)
);

CREATE INDEX idx_otel_configs_tenant ON otel_export_configs(tenant_id);
