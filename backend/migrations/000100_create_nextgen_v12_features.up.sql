-- Endpoint reliability scores
CREATE TABLE IF NOT EXISTS endpoint_reliability_scores (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    endpoint_id UUID NOT NULL,
    score DECIMAL(5,2) NOT NULL DEFAULT 100.00,
    status VARCHAR(20) NOT NULL DEFAULT 'unknown',
    success_rate DECIMAL(5,2) NOT NULL DEFAULT 100.00,
    latency_p50_ms INTEGER NOT NULL DEFAULT 0,
    latency_p95_ms INTEGER NOT NULL DEFAULT 0,
    latency_p99_ms INTEGER NOT NULL DEFAULT 0,
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    total_attempts INTEGER NOT NULL DEFAULT 0,
    successful_attempts INTEGER NOT NULL DEFAULT 0,
    failed_attempts INTEGER NOT NULL DEFAULT 0,
    window_start TIMESTAMP WITH TIME ZONE NOT NULL,
    window_end TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_reliability_scores_tenant_endpoint ON endpoint_reliability_scores(tenant_id, endpoint_id);
CREATE INDEX IF NOT EXISTS idx_reliability_scores_status ON endpoint_reliability_scores(status);

-- Hourly reliability score snapshots
CREATE TABLE IF NOT EXISTS reliability_score_snapshots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    endpoint_id UUID NOT NULL,
    score DECIMAL(5,2) NOT NULL,
    success_rate DECIMAL(5,2) NOT NULL,
    latency_p50_ms INTEGER NOT NULL DEFAULT 0,
    latency_p95_ms INTEGER NOT NULL DEFAULT 0,
    latency_p99_ms INTEGER NOT NULL DEFAULT 0,
    snapshot_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_reliability_snapshots_lookup ON reliability_score_snapshots(tenant_id, endpoint_id, snapshot_at DESC);

-- SLA targets
CREATE TABLE IF NOT EXISTS reliability_sla_targets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    endpoint_id UUID NOT NULL,
    target_score DECIMAL(5,2) NOT NULL,
    target_uptime DECIMAL(5,2) NOT NULL,
    max_latency_p95_ms INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_sla_targets_tenant_endpoint ON reliability_sla_targets(tenant_id, endpoint_id);

-- Alert thresholds
CREATE TABLE IF NOT EXISTS reliability_alert_thresholds (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    endpoint_id UUID NOT NULL,
    min_score DECIMAL(5,2) NOT NULL,
    max_latency_ms INTEGER NOT NULL,
    max_failures INTEGER NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_alert_thresholds_tenant_endpoint ON reliability_alert_thresholds(tenant_id, endpoint_id);

-- DLQ entries for intelligent dead letter queue
CREATE TABLE IF NOT EXISTS dlq_entries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    endpoint_id UUID NOT NULL,
    webhook_id UUID NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    headers JSONB NOT NULL DEFAULT '{}',
    error_message TEXT,
    error_type VARCHAR(50),
    root_cause VARCHAR(50),
    failure_cluster_id UUID,
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 5,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    remediation_action TEXT,
    replayed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_dlq_entries_tenant ON dlq_entries(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_dlq_entries_cluster ON dlq_entries(failure_cluster_id);
CREATE INDEX IF NOT EXISTS idx_dlq_entries_root_cause ON dlq_entries(tenant_id, root_cause);

-- DLQ failure clusters
CREATE TABLE IF NOT EXISTS dlq_failure_clusters (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    root_cause VARCHAR(50) NOT NULL,
    pattern TEXT NOT NULL,
    entry_count INTEGER NOT NULL DEFAULT 0,
    first_seen TIMESTAMP WITH TIME ZONE NOT NULL,
    last_seen TIMESTAMP WITH TIME ZONE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_dlq_clusters_tenant ON dlq_failure_clusters(tenant_id, status);

-- Schema registry compatibility checks
CREATE TABLE IF NOT EXISTS schema_compatibility_checks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    schema_id UUID NOT NULL,
    check_mode VARCHAR(20) NOT NULL,
    is_compatible BOOLEAN NOT NULL,
    breaking_changes JSONB DEFAULT '[]',
    checked_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_schema_compat_checks ON schema_compatibility_checks(tenant_id, schema_id);

-- Event catalog entries
CREATE TABLE IF NOT EXISTS event_catalog_entries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    event_type VARCHAR(255) NOT NULL,
    description TEXT,
    schema_id UUID,
    version VARCHAR(20) NOT NULL DEFAULT '1.0.0',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    owner VARCHAR(255),
    tags JSONB DEFAULT '[]',
    examples JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_event_catalog_tenant_type ON event_catalog_entries(tenant_id, event_type);

-- Cloud managed tenant provisioning
CREATE TABLE IF NOT EXISTS cloud_tenant_provisions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    plan_tier VARCHAR(20) NOT NULL DEFAULT 'free',
    stripe_customer_id VARCHAR(255),
    stripe_subscription_id VARCHAR(255),
    api_key_hash VARCHAR(255),
    events_quota INTEGER NOT NULL DEFAULT 1000,
    events_used INTEGER NOT NULL DEFAULT 0,
    storage_quota_bytes BIGINT NOT NULL DEFAULT 104857600,
    storage_used_bytes BIGINT NOT NULL DEFAULT 0,
    is_trial BOOLEAN NOT NULL DEFAULT true,
    trial_ends_at TIMESTAMP WITH TIME ZONE,
    suspended_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_cloud_provisions_tenant ON cloud_tenant_provisions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_cloud_provisions_stripe ON cloud_tenant_provisions(stripe_customer_id);

-- GitOps environment promotions
CREATE TABLE IF NOT EXISTS gitops_environments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    name VARCHAR(50) NOT NULL,
    stage INTEGER NOT NULL DEFAULT 0,
    manifest_id UUID,
    requires_approval BOOLEAN NOT NULL DEFAULT false,
    approved_by VARCHAR(255),
    approved_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_gitops_envs_tenant_name ON gitops_environments(tenant_id, name);

-- Replay archive events
CREATE TABLE IF NOT EXISTS replay_archive_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    webhook_id UUID NOT NULL,
    endpoint_id UUID NOT NULL,
    event_type VARCHAR(255),
    payload JSONB NOT NULL DEFAULT '{}',
    headers JSONB NOT NULL DEFAULT '{}',
    response_status INTEGER,
    response_body TEXT,
    latency_ms INTEGER,
    delivery_status VARCHAR(20),
    context JSONB DEFAULT '{}',
    delivered_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_replay_archive_lookup ON replay_archive_events(tenant_id, endpoint_id, created_at DESC);

-- Inbound provider registry
CREATE TABLE IF NOT EXISTS inbound_provider_registry (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider VARCHAR(50) NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    signature_method VARCHAR(50),
    doc_url TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    config JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_inbound_provider_name ON inbound_provider_registry(provider);

-- Portal SDK configurations
CREATE TABLE IF NOT EXISTS portal_sdk_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    theme JSONB DEFAULT '{}',
    features JSONB DEFAULT '{}',
    branding JSONB DEFAULT '{}',
    custom_domain VARCHAR(255),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_portal_sdk_tenant ON portal_sdk_configs(tenant_id);

-- Webhook test suites
CREATE TABLE IF NOT EXISTS webhook_test_suites (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    test_cases JSONB DEFAULT '[]',
    config JSONB DEFAULT '{}',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    last_run_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_test_suites_tenant ON webhook_test_suites(tenant_id);

-- Webhook test results
CREATE TABLE IF NOT EXISTS webhook_test_results (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    suite_id UUID NOT NULL,
    total_tests INTEGER NOT NULL DEFAULT 0,
    passed INTEGER NOT NULL DEFAULT 0,
    failed INTEGER NOT NULL DEFAULT 0,
    skipped INTEGER NOT NULL DEFAULT 0,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    results JSONB DEFAULT '[]',
    junit_xml TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_test_results_suite ON webhook_test_results(suite_id);

-- Protocol delivery configs for multi-protocol
CREATE TABLE IF NOT EXISTS protocol_delivery_routes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    endpoint_id UUID NOT NULL,
    protocol VARCHAR(20) NOT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    fallback_protocol VARCHAR(20),
    config JSONB DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_protocol_routes_endpoint ON protocol_delivery_routes(tenant_id, endpoint_id);
