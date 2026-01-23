CREATE TABLE IF NOT EXISTS migration_jobs (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    source_platform VARCHAR(100) NOT NULL,
    source_config JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    endpoint_count INTEGER NOT NULL DEFAULT 0,
    migrated_count INTEGER NOT NULL DEFAULT 0,
    failed_count INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS migration_endpoints (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    job_id UUID NOT NULL REFERENCES migration_jobs(id),
    source_id VARCHAR(255) NOT NULL,
    source_url TEXT NOT NULL,
    destination_id VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    mapping_config JSONB,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS parallel_delivery_results (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    job_id UUID NOT NULL REFERENCES migration_jobs(id),
    endpoint_id UUID NOT NULL,
    event_id VARCHAR(255) NOT NULL,
    source_status INTEGER,
    dest_status INTEGER,
    source_latency_ms BIGINT,
    dest_latency_ms BIGINT,
    response_match BOOLEAN NOT NULL DEFAULT FALSE,
    discrepancy TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_migration_jobs_tenant ON migration_jobs(tenant_id);
CREATE INDEX idx_migration_jobs_status ON migration_jobs(tenant_id, status);
CREATE INDEX idx_migration_endpoints_job ON migration_endpoints(job_id);
CREATE INDEX idx_parallel_delivery_results_job ON parallel_delivery_results(job_id);
