CREATE TABLE IF NOT EXISTS protocol_routes (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    source_protocol VARCHAR(50) NOT NULL,
    source_config JSONB NOT NULL DEFAULT '{}',
    dest_protocol VARCHAR(50) NOT NULL,
    dest_config JSONB NOT NULL DEFAULT '{}',
    transform_rule TEXT,
    ordering_guarantee VARCHAR(50) NOT NULL DEFAULT 'none',
    delivery_guarantee VARCHAR(50) NOT NULL DEFAULT 'at_least_once',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS protocol_messages (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    route_id UUID NOT NULL REFERENCES protocol_routes(id),
    source_protocol VARCHAR(50) NOT NULL,
    dest_protocol VARCHAR(50) NOT NULL,
    payload TEXT NOT NULL,
    headers JSONB DEFAULT '{}',
    partition_key VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    translated_payload TEXT,
    error_message TEXT,
    latency_ms BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_protocol_routes_tenant ON protocol_routes(tenant_id);
CREATE INDEX idx_protocol_messages_route ON protocol_messages(route_id);
CREATE INDEX idx_protocol_messages_status ON protocol_messages(tenant_id, status);
