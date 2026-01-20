-- CDC Connectors table
CREATE TABLE IF NOT EXISTS cdc_connectors (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'created',
    connection_config JSONB NOT NULL,
    capture_config JSONB NOT NULL,
    webhook_config JSONB NOT NULL,
    offset_config JSONB NOT NULL,
    events_processed BIGINT DEFAULT 0,
    events_failed BIGINT DEFAULT 0,
    bytes_processed BIGINT DEFAULT 0,
    last_event_at TIMESTAMPTZ,
    last_offset JSONB,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cdc_connectors_tenant ON cdc_connectors(tenant_id);
CREATE INDEX idx_cdc_connectors_status ON cdc_connectors(tenant_id, status);
CREATE INDEX idx_cdc_connectors_type ON cdc_connectors(tenant_id, type);

-- CDC Offsets table (for tracking replication position)
CREATE TABLE IF NOT EXISTS cdc_offsets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    connector_id UUID NOT NULL REFERENCES cdc_connectors(id) ON DELETE CASCADE,
    partition_key VARCHAR(255) NOT NULL,
    offset_value JSONB NOT NULL,
    committed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(connector_id, partition_key)
);

CREATE INDEX idx_cdc_offsets_connector ON cdc_offsets(connector_id);

-- CDC Event History table (stores recent events for debugging)
CREATE TABLE IF NOT EXISTS cdc_event_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    connector_id UUID NOT NULL REFERENCES cdc_connectors(id) ON DELETE CASCADE,
    event_id VARCHAR(255) NOT NULL,
    table_name VARCHAR(255) NOT NULL,
    operation VARCHAR(50) NOT NULL,
    key_columns JSONB,
    before_data JSONB,
    after_data JSONB,
    source_timestamp TIMESTAMPTZ,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    webhook_id UUID,
    delivery_status VARCHAR(50),
    error_message TEXT
);

CREATE INDEX idx_cdc_events_connector ON cdc_event_history(connector_id);
CREATE INDEX idx_cdc_events_tenant ON cdc_event_history(tenant_id, processed_at DESC);
CREATE INDEX idx_cdc_events_table ON cdc_event_history(connector_id, table_name);

-- Partition event history by time (optional, for large deployments)
-- CREATE INDEX idx_cdc_events_processed ON cdc_event_history(processed_at);

-- CDC Metrics table (aggregated metrics per hour)
CREATE TABLE IF NOT EXISTS cdc_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    connector_id UUID NOT NULL REFERENCES cdc_connectors(id) ON DELETE CASCADE,
    bucket_start TIMESTAMPTZ NOT NULL,
    events_captured BIGINT DEFAULT 0,
    events_delivered BIGINT DEFAULT 0,
    events_failed BIGINT DEFAULT 0,
    bytes_captured BIGINT DEFAULT 0,
    avg_latency_ms FLOAT,
    p99_latency_ms FLOAT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(connector_id, bucket_start)
);

CREATE INDEX idx_cdc_metrics_connector ON cdc_metrics(connector_id, bucket_start DESC);
CREATE INDEX idx_cdc_metrics_tenant ON cdc_metrics(tenant_id, bucket_start DESC);

-- CDC Schema Snapshots (for schema evolution tracking)
CREATE TABLE IF NOT EXISTS cdc_schema_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    connector_id UUID NOT NULL REFERENCES cdc_connectors(id) ON DELETE CASCADE,
    table_name VARCHAR(255) NOT NULL,
    schema_version INT NOT NULL DEFAULT 1,
    columns JSONB NOT NULL,
    key_columns JSONB NOT NULL,
    captured_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cdc_schema_connector ON cdc_schema_snapshots(connector_id, table_name);
CREATE UNIQUE INDEX idx_cdc_schema_version ON cdc_schema_snapshots(connector_id, table_name, schema_version);
