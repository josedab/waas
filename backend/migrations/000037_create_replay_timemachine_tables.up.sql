-- Event Replay Time Machine tables
-- Full event sourcing with point-in-time replay capabilities

CREATE TABLE event_archive (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    endpoint_id UUID REFERENCES webhook_endpoints(id) ON DELETE SET NULL,
    event_type VARCHAR(255),
    payload JSONB NOT NULL,
    payload_hash VARCHAR(64) NOT NULL,
    headers JSONB DEFAULT '{}',
    source_ip VARCHAR(45),
    received_at TIMESTAMP NOT NULL DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'
);

CREATE TABLE replay_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    filter_criteria JSONB NOT NULL DEFAULT '{}',
    time_range_start TIMESTAMP NOT NULL,
    time_range_end TIMESTAMP NOT NULL,
    target_endpoint_id UUID REFERENCES webhook_endpoints(id),
    transformation_id UUID,
    options JSONB NOT NULL DEFAULT '{}',
    total_events INTEGER DEFAULT 0,
    processed_events INTEGER DEFAULT 0,
    successful_events INTEGER DEFAULT 0,
    failed_events INTEGER DEFAULT 0,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    error_message TEXT,
    created_by UUID,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE replay_job_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    replay_job_id UUID NOT NULL REFERENCES replay_jobs(id) ON DELETE CASCADE,
    event_archive_id UUID NOT NULL REFERENCES event_archive(id),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    original_payload JSONB NOT NULL,
    transformed_payload JSONB,
    delivery_attempt_id UUID,
    error_message TEXT,
    processed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE replay_snapshots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    snapshot_time TIMESTAMP NOT NULL,
    filter_criteria JSONB DEFAULT '{}',
    event_count INTEGER NOT NULL DEFAULT 0,
    size_bytes BIGINT DEFAULT 0,
    storage_location VARCHAR(500),
    expires_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE replay_comparisons (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    original_job_id UUID REFERENCES replay_jobs(id),
    comparison_job_id UUID REFERENCES replay_jobs(id),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    total_events INTEGER DEFAULT 0,
    matching_events INTEGER DEFAULT 0,
    differing_events INTEGER DEFAULT 0,
    diff_report JSONB DEFAULT '{}',
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX idx_event_archive_tenant ON event_archive(tenant_id);
CREATE INDEX idx_event_archive_endpoint ON event_archive(endpoint_id);
CREATE INDEX idx_event_archive_received ON event_archive(received_at DESC);
CREATE INDEX idx_event_archive_type ON event_archive(event_type);
CREATE INDEX idx_event_archive_hash ON event_archive(payload_hash);
CREATE INDEX idx_event_archive_tenant_time ON event_archive(tenant_id, received_at DESC);

CREATE INDEX idx_replay_jobs_tenant ON replay_jobs(tenant_id);
CREATE INDEX idx_replay_jobs_status ON replay_jobs(status);
CREATE INDEX idx_replay_job_events_job ON replay_job_events(replay_job_id);
CREATE INDEX idx_replay_job_events_status ON replay_job_events(status);
CREATE INDEX idx_replay_snapshots_tenant ON replay_snapshots(tenant_id);
CREATE INDEX idx_replay_comparisons_tenant ON replay_comparisons(tenant_id);

-- Partitioning hint for large deployments (comment in for production)
-- ALTER TABLE event_archive PARTITION BY RANGE (received_at);
