-- AI Conversational Webhook Builder
CREATE TABLE IF NOT EXISTS ai_builder_conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    title VARCHAR(255) NOT NULL DEFAULT '',
    context JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '24 hours'
);

CREATE TABLE IF NOT EXISTS ai_builder_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES ai_builder_conversations(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL,
    content TEXT NOT NULL,
    intent VARCHAR(50),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ai_builder_conversations_tenant ON ai_builder_conversations(tenant_id);
CREATE INDEX idx_ai_builder_messages_conversation ON ai_builder_messages(conversation_id);

-- Global Edge Delivery Network
CREATE TABLE IF NOT EXISTS edge_network_nodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    region VARCHAR(50) NOT NULL,
    endpoint VARCHAR(512) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'healthy',
    latitude DOUBLE PRECISION DEFAULT 0,
    longitude DOUBLE PRECISION DEFAULT 0,
    capacity INTEGER DEFAULT 1000,
    active_connections INTEGER DEFAULT 0,
    avg_latency_ms DOUBLE PRECISION DEFAULT 0,
    success_rate DOUBLE PRECISION DEFAULT 100,
    last_health_check TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS edge_network_routing_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    priority INTEGER DEFAULT 0,
    strategy VARCHAR(50) NOT NULL,
    target_regions JSONB DEFAULT '[]',
    conditions JSONB DEFAULT '[]',
    fallback_region VARCHAR(50),
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_edge_network_nodes_region ON edge_network_nodes(region);
CREATE INDEX idx_edge_network_routing_rules_tenant ON edge_network_routing_rules(tenant_id);

-- Zero-Downtime Platform Migration Wizard
CREATE TABLE IF NOT EXISTS platform_migrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    source_platform VARCHAR(50) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    config JSONB DEFAULT '{}',
    analysis JSONB,
    progress JSONB DEFAULT '{}',
    error_message TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_platform_migrations_tenant ON platform_migrations(tenant_id);

-- Webhook Security Intelligence Suite
CREATE TABLE IF NOT EXISTS security_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    endpoint_id UUID,
    threat_type VARCHAR(50) NOT NULL,
    threat_level VARCHAR(20) NOT NULL,
    description TEXT NOT NULL,
    source_ip VARCHAR(45),
    payload_snippet TEXT,
    action_taken VARCHAR(20) NOT NULL DEFAULT 'flagged',
    resolved BOOLEAN DEFAULT FALSE,
    detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS security_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    rules JSONB NOT NULL DEFAULT '[]',
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_security_events_tenant ON security_events(tenant_id);
CREATE INDEX idx_security_events_level ON security_events(threat_level);
CREATE INDEX idx_security_policies_tenant ON security_policies(tenant_id);

-- Developer Marketplace & Plugin Ecosystem
CREATE TABLE IF NOT EXISTS marketplace_plugins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    developer_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    description TEXT NOT NULL,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    version VARCHAR(50) NOT NULL,
    icon_url VARCHAR(512),
    source_url VARCHAR(512),
    pricing VARCHAR(20) DEFAULT 'free',
    price_amount_cents INTEGER DEFAULT 0,
    installs INTEGER DEFAULT 0,
    rating DOUBLE PRECISION DEFAULT 0,
    rating_count INTEGER DEFAULT 0,
    tags JSONB DEFAULT '[]',
    manifest JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS plugin_installations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    plugin_id UUID NOT NULL REFERENCES marketplace_plugins(id),
    version VARCHAR(50) NOT NULL,
    config JSONB DEFAULT '{}',
    enabled BOOLEAN DEFAULT TRUE,
    installed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, plugin_id)
);

CREATE TABLE IF NOT EXISTS plugin_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plugin_id UUID NOT NULL REFERENCES marketplace_plugins(id),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    rating INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_marketplace_plugins_status ON marketplace_plugins(status);
CREATE INDEX idx_marketplace_plugins_type ON marketplace_plugins(type);
CREATE INDEX idx_plugin_installations_tenant ON plugin_installations(tenant_id);
CREATE INDEX idx_plugin_reviews_plugin ON plugin_reviews(plugin_id);

-- Serverless Transform Runtime (FaaS)
CREATE TABLE IF NOT EXISTS faas_functions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    runtime VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    code TEXT NOT NULL,
    version INTEGER DEFAULT 1,
    entry_point VARCHAR(255) DEFAULT 'transform',
    timeout_ms INTEGER DEFAULT 5000,
    memory_limit_mb INTEGER DEFAULT 128,
    env_vars JSONB DEFAULT '{}',
    endpoint_ids JSONB DEFAULT '[]',
    invocations BIGINT DEFAULT 0,
    avg_duration_ms BIGINT DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS faas_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    function_id UUID NOT NULL REFERENCES faas_functions(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    input TEXT,
    output TEXT,
    duration_ms BIGINT,
    memory_used_mb INTEGER,
    success BOOLEAN NOT NULL,
    error TEXT,
    log_output TEXT,
    executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_faas_functions_tenant ON faas_functions(tenant_id);
CREATE INDEX idx_faas_executions_function ON faas_executions(function_id);

-- Webhook A/B Testing & Progressive Delivery
CREATE TABLE IF NOT EXISTS progressive_rollouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    endpoint_id UUID,
    strategy VARCHAR(30) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    target_config JSONB DEFAULT '{}',
    baseline_config JSONB DEFAULT '{}',
    traffic_split JSONB DEFAULT '{"baseline_percent":100,"target_percent":0}',
    success_criteria JSONB DEFAULT '{}',
    metrics JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_progressive_rollouts_tenant ON progressive_rollouts(tenant_id);

-- Self-Healing Endpoint Mesh
CREATE TABLE IF NOT EXISTS endpoint_mesh_nodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    endpoint_id UUID,
    url VARCHAR(512) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'healthy',
    health_score DOUBLE PRECISION DEFAULT 100,
    consecutive_failures INTEGER DEFAULT 0,
    last_success_at TIMESTAMPTZ,
    last_failure_at TIMESTAMPTZ,
    circuit_state JSONB DEFAULT '{"state":"closed"}',
    fallback_node_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS endpoint_mesh_reroute_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    source_node_id UUID NOT NULL,
    target_node_id UUID NOT NULL,
    reason TEXT,
    auto_recovered BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_endpoint_mesh_nodes_tenant ON endpoint_mesh_nodes(tenant_id);
CREATE INDEX idx_endpoint_mesh_reroutes_tenant ON endpoint_mesh_reroute_events(tenant_id);

-- Mobile Webhook Inspector
CREATE TABLE IF NOT EXISTS mobile_devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    device_token VARCHAR(512) NOT NULL,
    platform VARCHAR(10) NOT NULL,
    name VARCHAR(255),
    enabled BOOLEAN DEFAULT TRUE,
    filters JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS mobile_notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    device_id UUID REFERENCES mobile_devices(id),
    type VARCHAR(30) NOT NULL,
    title VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,
    data JSONB DEFAULT '{}',
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    read_at TIMESTAMPTZ
);

CREATE INDEX idx_mobile_devices_tenant ON mobile_devices(tenant_id);
CREATE INDEX idx_mobile_notifications_tenant ON mobile_notifications(tenant_id);

-- Webhook Infrastructure Capacity Planner
CREATE TABLE IF NOT EXISTS capacity_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    current_usage JSONB DEFAULT '{}',
    peak_usage JSONB DEFAULT '{}',
    projections JSONB DEFAULT '[]',
    recommendations JSONB DEFAULT '[]',
    bottlenecks JSONB DEFAULT '[]',
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS capacity_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    resource VARCHAR(100) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    message TEXT NOT NULL,
    current_value DOUBLE PRECISION,
    threshold_value DOUBLE PRECISION,
    acknowledged BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_capacity_reports_tenant ON capacity_reports(tenant_id);
CREATE INDEX idx_capacity_alerts_tenant ON capacity_alerts(tenant_id);
