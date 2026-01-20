-- Event Sourcing Mode Tables
-- Feature 6: Append-only event log with replay

-- Event Store (append-only log)
CREATE TABLE IF NOT EXISTS event_store (
    sequence_id BIGSERIAL PRIMARY KEY,
    event_id UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    stream_id VARCHAR(255) NOT NULL,
    stream_type VARCHAR(100) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    
    -- Event data
    payload JSONB NOT NULL,
    metadata JSONB DEFAULT '{}',
    
    -- Version for optimistic concurrency
    version INTEGER NOT NULL,
    
    -- Causation and correlation
    correlation_id UUID,
    causation_id UUID,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    
    -- Ensure version uniqueness per stream
    UNIQUE(tenant_id, stream_id, version)
);

-- Consumer Checkpoints (for replay)
CREATE TABLE IF NOT EXISTS consumer_checkpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    consumer_group VARCHAR(255) NOT NULL,
    stream_id VARCHAR(255),
    last_sequence_id BIGINT NOT NULL DEFAULT 0,
    last_processed_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(tenant_id, consumer_group, stream_id)
);

-- Snapshots (for state reconstruction optimization)
CREATE TABLE IF NOT EXISTS stream_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    stream_id VARCHAR(255) NOT NULL,
    stream_type VARCHAR(100) NOT NULL,
    version INTEGER NOT NULL,
    state JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(tenant_id, stream_id, version)
);

-- Projections (read models)
CREATE TABLE IF NOT EXISTS projections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Projection query/handler definition
    event_types TEXT[] NOT NULL,
    handler_code TEXT,
    output_schema JSONB,
    
    status VARCHAR(50) DEFAULT 'active' CHECK (status IN ('active', 'paused', 'rebuilding', 'failed')),
    last_sequence_id BIGINT DEFAULT 0,
    last_error TEXT,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(tenant_id, name)
);

-- Projection State (materialized view data)
CREATE TABLE IF NOT EXISTS projection_state (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    projection_id UUID NOT NULL REFERENCES projections(id) ON DELETE CASCADE,
    partition_key VARCHAR(255) NOT NULL,
    state JSONB NOT NULL,
    version INTEGER DEFAULT 1,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(projection_id, partition_key)
);

-- Replay Jobs
CREATE TABLE IF NOT EXISTS replay_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Replay scope
    stream_id VARCHAR(255),
    stream_type VARCHAR(100),
    event_types TEXT[],
    from_sequence_id BIGINT,
    to_sequence_id BIGINT,
    
    -- Target
    target_type VARCHAR(50) NOT NULL CHECK (target_type IN ('endpoint', 'projection', 'consumer')),
    target_id VARCHAR(255) NOT NULL,
    
    -- Progress
    status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'paused', 'completed', 'failed', 'cancelled')),
    current_sequence_id BIGINT,
    events_processed INTEGER DEFAULT 0,
    events_total INTEGER,
    error_message TEXT,
    
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX idx_event_store_tenant_stream ON event_store(tenant_id, stream_id);
CREATE INDEX idx_event_store_tenant_type ON event_store(tenant_id, event_type);
CREATE INDEX idx_event_store_created ON event_store(created_at);
CREATE INDEX idx_event_store_correlation ON event_store(correlation_id) WHERE correlation_id IS NOT NULL;
CREATE INDEX idx_consumer_checkpoints_lookup ON consumer_checkpoints(tenant_id, consumer_group);
CREATE INDEX idx_stream_snapshots_lookup ON stream_snapshots(tenant_id, stream_id);
CREATE INDEX idx_projections_tenant ON projections(tenant_id);
CREATE INDEX idx_replay_jobs_status ON replay_jobs(tenant_id, status);
