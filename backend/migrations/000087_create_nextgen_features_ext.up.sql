-- Declarative GitOps sync state tracking
CREATE TABLE IF NOT EXISTS gitops_sync_states (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    manifest_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending_sync',
    message TEXT DEFAULT '',
    last_sync_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    next_sync_at TIMESTAMP WITH TIME ZONE,
    revision INTEGER DEFAULT 1
);

CREATE INDEX idx_gitops_sync_tenant ON gitops_sync_states(tenant_id);
CREATE INDEX idx_gitops_sync_manifest ON gitops_sync_states(manifest_id);

-- DLQ root-cause analysis cache
CREATE TABLE IF NOT EXISTS dlq_root_cause_analyses (
    id TEXT PRIMARY KEY,
    entry_id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    endpoint_id TEXT NOT NULL,
    category TEXT NOT NULL,
    severity TEXT NOT NULL DEFAULT 'medium',
    summary TEXT NOT NULL,
    details TEXT DEFAULT '',
    confidence DOUBLE PRECISION DEFAULT 0,
    suggestions JSONB DEFAULT '[]',
    related_entries JSONB DEFAULT '[]',
    patterns JSONB DEFAULT '[]',
    analyzed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_dlq_rca_tenant ON dlq_root_cause_analyses(tenant_id);
CREATE INDEX idx_dlq_rca_endpoint ON dlq_root_cause_analyses(endpoint_id);

-- Time-travel debug sessions
CREATE TABLE IF NOT EXISTS timetravel_debug_sessions (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    name TEXT NOT NULL,
    event_ids JSONB DEFAULT '[]',
    status TEXT NOT NULL DEFAULT 'active',
    breakpoints JSONB DEFAULT '[]',
    step_history JSONB DEFAULT '[]',
    current_step INTEGER DEFAULT 0,
    context JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_debug_sessions_tenant ON timetravel_debug_sessions(tenant_id);

-- Event catalog auto-discovery
CREATE TABLE IF NOT EXISTS catalog_discovered_events (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    name TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT 'traffic',
    status TEXT NOT NULL DEFAULT 'active',
    sample_payload JSONB,
    inferred_schema JSONB,
    occurrence_count BIGINT DEFAULT 0,
    first_seen_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_seen_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    endpoints JSONB DEFAULT '[]',
    confidence DOUBLE PRECISION DEFAULT 0.5,
    suggested_category TEXT DEFAULT '',
    suggested_tags JSONB DEFAULT '[]',
    UNIQUE(tenant_id, name)
);

CREATE INDEX idx_discovered_events_tenant ON catalog_discovered_events(tenant_id);
CREATE INDEX idx_discovered_events_status ON catalog_discovered_events(tenant_id, status);

-- Adaptive receiver-aware rate limiting
CREATE TABLE IF NOT EXISTS adaptive_receiver_configs (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    endpoint_id TEXT NOT NULL,
    strategy TEXT NOT NULL DEFAULT 'aimd',
    base_rate_per_second DOUBLE PRECISION NOT NULL,
    current_rate DOUBLE PRECISION NOT NULL,
    min_rate DOUBLE PRECISION DEFAULT 1,
    max_rate DOUBLE PRECISION DEFAULT 500,
    increase_step DOUBLE PRECISION DEFAULT 10,
    decrease_multiplier DOUBLE PRECISION DEFAULT 0.5,
    health_threshold DOUBLE PRECISION DEFAULT 0.8,
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, endpoint_id)
);

CREATE INDEX idx_adaptive_configs_tenant ON adaptive_receiver_configs(tenant_id);

CREATE TABLE IF NOT EXISTS adaptive_receiver_health (
    endpoint_id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'unknown',
    avg_response_time_ms DOUBLE PRECISION DEFAULT 0,
    p95_response_time_ms DOUBLE PRECISION DEFAULT 0,
    success_rate DOUBLE PRECISION DEFAULT 0,
    error_rate DOUBLE PRECISION DEFAULT 0,
    rate_limit_hits BIGINT DEFAULT 0,
    consecutive_errors INTEGER DEFAULT 0,
    last_checked_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_success_at TIMESTAMP WITH TIME ZONE,
    PRIMARY KEY (tenant_id, endpoint_id)
);

-- Cost attribution
CREATE TABLE IF NOT EXISTS cost_attributions (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    endpoint_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    period TEXT NOT NULL DEFAULT 'daily',
    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    delivery_count BIGINT DEFAULT 0,
    success_count BIGINT DEFAULT 0,
    failed_count BIGINT DEFAULT 0,
    retry_count BIGINT DEFAULT 0,
    total_cost DOUBLE PRECISION DEFAULT 0,
    delivery_cost DOUBLE PRECISION DEFAULT 0,
    bandwidth_cost DOUBLE PRECISION DEFAULT 0,
    compute_cost DOUBLE PRECISION DEFAULT 0,
    storage_cost DOUBLE PRECISION DEFAULT 0,
    total_bytes_out BIGINT DEFAULT 0,
    UNIQUE(tenant_id, endpoint_id, event_type, period_start)
);

CREATE INDEX idx_cost_attr_tenant ON cost_attributions(tenant_id);
CREATE INDEX idx_cost_attr_period ON cost_attributions(tenant_id, period_start, period_end);

-- Edge dispatch configuration
CREATE TABLE IF NOT EXISTS edge_dispatch_configs (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL UNIQUE,
    strategy TEXT NOT NULL DEFAULT 'lowest_latency',
    preferred_regions JSONB DEFAULT '[]',
    max_latency_ms INTEGER DEFAULT 500,
    enable_failover BOOLEAN DEFAULT TRUE,
    failover_regions JSONB DEFAULT '[]',
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS edge_delivery_metrics (
    node_id TEXT NOT NULL,
    region TEXT NOT NULL DEFAULT '',
    total_deliveries BIGINT DEFAULT 0,
    success_count BIGINT DEFAULT 0,
    failure_count BIGINT DEFAULT 0,
    avg_latency_ms DOUBLE PRECISION DEFAULT 0,
    p50_latency_ms DOUBLE PRECISION DEFAULT 0,
    p99_latency_ms DOUBLE PRECISION DEFAULT 0,
    active_connections INTEGER DEFAULT 0,
    last_delivery_at TIMESTAMP WITH TIME ZONE,
    PRIMARY KEY (node_id)
);
