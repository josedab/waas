-- Event Streaming Bridge tables

-- Stream configurations
CREATE TABLE IF NOT EXISTS stream_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    platform VARCHAR(50) NOT NULL, -- kafka, kinesis, pulsar, eventbridge
    direction VARCHAR(10) NOT NULL DEFAULT 'outbound', -- inbound, outbound, bidirectional
    connection_config JSONB NOT NULL DEFAULT '{}',
    topic_config JSONB NOT NULL DEFAULT '{}',
    transform_config JSONB,
    schema_id UUID,
    status VARCHAR(20) NOT NULL DEFAULT 'inactive',
    error_message TEXT,
    metrics JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_stream_name_per_tenant UNIQUE(tenant_id, name)
);

CREATE INDEX idx_stream_configs_tenant ON stream_configs(tenant_id);
CREATE INDEX idx_stream_configs_platform ON stream_configs(tenant_id, platform);
CREATE INDEX idx_stream_configs_status ON stream_configs(tenant_id, status);

-- Stream topics
CREATE TABLE IF NOT EXISTS stream_topics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    stream_config_id UUID NOT NULL REFERENCES stream_configs(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    partition_count INT NOT NULL DEFAULT 1,
    replication_factor INT NOT NULL DEFAULT 1,
    retention_hours INT NOT NULL DEFAULT 168,
    config JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_stream_topics_stream ON stream_topics(stream_config_id);

-- Stream consumers
CREATE TABLE IF NOT EXISTS stream_consumers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    stream_config_id UUID NOT NULL REFERENCES stream_configs(id) ON DELETE CASCADE,
    consumer_group VARCHAR(255) NOT NULL,
    topic_name VARCHAR(255) NOT NULL,
    partition_assignments JSONB DEFAULT '[]',
    offset_tracking JSONB DEFAULT '{}',
    status VARCHAR(20) NOT NULL DEFAULT 'inactive',
    last_poll_at TIMESTAMP WITH TIME ZONE,
    messages_consumed BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_stream_consumers_stream ON stream_consumers(stream_config_id);
CREATE INDEX idx_stream_consumers_group ON stream_consumers(consumer_group);

-- Stream producers
CREATE TABLE IF NOT EXISTS stream_producers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    stream_config_id UUID NOT NULL REFERENCES stream_configs(id) ON DELETE CASCADE,
    topic_name VARCHAR(255) NOT NULL,
    partition_strategy VARCHAR(50) NOT NULL DEFAULT 'round_robin',
    batch_config JSONB DEFAULT '{}',
    compression VARCHAR(20) DEFAULT 'none',
    status VARCHAR(20) NOT NULL DEFAULT 'inactive',
    messages_produced BIGINT NOT NULL DEFAULT 0,
    bytes_produced BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_stream_producers_stream ON stream_producers(stream_config_id);

-- Stream messages (for tracking/replay)
CREATE TABLE IF NOT EXISTS stream_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    stream_config_id UUID NOT NULL REFERENCES stream_configs(id) ON DELETE CASCADE,
    topic_name VARCHAR(255) NOT NULL,
    partition INT NOT NULL DEFAULT 0,
    offset_num BIGINT NOT NULL DEFAULT 0,
    key VARCHAR(500),
    value_hash VARCHAR(64),
    headers JSONB DEFAULT '{}',
    schema_version INT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    sent_at TIMESTAMP WITH TIME ZONE,
    acked_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_stream_messages_stream ON stream_messages(stream_config_id, topic_name);
CREATE INDEX idx_stream_messages_partition ON stream_messages(topic_name, partition, offset_num);
CREATE INDEX idx_stream_messages_status ON stream_messages(tenant_id, status);
CREATE INDEX idx_stream_messages_created ON stream_messages(created_at);

-- Partition the messages table by created_at for better performance
-- (Optional: uncomment for production)
-- CREATE TABLE stream_messages_partitioned (LIKE stream_messages INCLUDING ALL)
-- PARTITION BY RANGE (created_at);

-- Schema registry (extends existing schemas table)
CREATE TABLE IF NOT EXISTS stream_schemas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    subject VARCHAR(255) NOT NULL,
    version INT NOT NULL DEFAULT 1,
    schema_type VARCHAR(20) NOT NULL DEFAULT 'json', -- json, avro, protobuf
    schema_definition TEXT NOT NULL,
    fingerprint VARCHAR(64) NOT NULL,
    compatibility VARCHAR(20) NOT NULL DEFAULT 'backward',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_schema_version UNIQUE(tenant_id, subject, version)
);

CREATE INDEX idx_stream_schemas_subject ON stream_schemas(tenant_id, subject);
CREATE INDEX idx_stream_schemas_fingerprint ON stream_schemas(fingerprint);

-- Trigger for updated_at
CREATE OR REPLACE FUNCTION update_stream_config_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_stream_configs_updated
    BEFORE UPDATE ON stream_configs
    FOR EACH ROW
    EXECUTE FUNCTION update_stream_config_timestamp();

CREATE TRIGGER trigger_stream_consumers_updated
    BEFORE UPDATE ON stream_consumers
    FOR EACH ROW
    EXECUTE FUNCTION update_stream_config_timestamp();

CREATE TRIGGER trigger_stream_producers_updated
    BEFORE UPDATE ON stream_producers
    FOR EACH ROW
    EXECUTE FUNCTION update_stream_config_timestamp();
