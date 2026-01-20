-- Feature 10: Serverless Edge Functions
-- Enables custom function deployment at edge for transformations, auth, and enrichment

-- Edge function definitions
CREATE TABLE edge_functions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    runtime VARCHAR(50) NOT NULL DEFAULT 'javascript', -- javascript, typescript, python
    code TEXT NOT NULL,
    entry_point VARCHAR(255) DEFAULT 'handler',
    version INTEGER DEFAULT 1,
    status VARCHAR(50) DEFAULT 'draft', -- draft, deploying, active, deprecated, failed
    timeout_ms INTEGER DEFAULT 5000,
    memory_mb INTEGER DEFAULT 128,
    environment_vars JSONB DEFAULT '{}',
    dependencies JSONB DEFAULT '[]',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deployed_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(tenant_id, name)
);

-- Function versions for rollback support
CREATE TABLE edge_function_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    function_id UUID NOT NULL REFERENCES edge_functions(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    code TEXT NOT NULL,
    entry_point VARCHAR(255) DEFAULT 'handler',
    code_hash VARCHAR(64) NOT NULL,
    change_log TEXT,
    created_by VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(function_id, version)
);

-- Edge locations where functions can be deployed
CREATE TABLE edge_locations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    code VARCHAR(20) NOT NULL UNIQUE,
    region VARCHAR(50) NOT NULL,
    provider VARCHAR(50) NOT NULL, -- cloudflare, fastly, lambda_edge, custom
    status VARCHAR(50) DEFAULT 'active',
    latency_ms INTEGER DEFAULT 0,
    capacity INTEGER DEFAULT 1000,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Function deployments to edge locations
CREATE TABLE edge_function_deployments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    function_id UUID NOT NULL REFERENCES edge_functions(id) ON DELETE CASCADE,
    location_id UUID NOT NULL REFERENCES edge_locations(id),
    version INTEGER NOT NULL,
    status VARCHAR(50) DEFAULT 'pending', -- pending, deploying, active, failed, draining
    deployment_url VARCHAR(500),
    health_check_url VARCHAR(500),
    last_health_check TIMESTAMP WITH TIME ZONE,
    health_status VARCHAR(50) DEFAULT 'unknown',
    error_message TEXT,
    deployed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(function_id, location_id)
);

-- Function triggers (when to invoke)
CREATE TABLE edge_function_triggers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    function_id UUID NOT NULL REFERENCES edge_functions(id) ON DELETE CASCADE,
    trigger_type VARCHAR(50) NOT NULL, -- pre_send, post_receive, transform, authenticate, enrich
    event_types TEXT[] DEFAULT '{}',
    endpoint_ids UUID[] DEFAULT '{}',
    conditions JSONB DEFAULT '{}',
    priority INTEGER DEFAULT 0,
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Function invocations log
CREATE TABLE edge_function_invocations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    function_id UUID NOT NULL,
    deployment_id UUID REFERENCES edge_function_deployments(id),
    trigger_id UUID REFERENCES edge_function_triggers(id),
    tenant_id UUID NOT NULL,
    event_id UUID,
    endpoint_id UUID,
    location_code VARCHAR(20),
    status VARCHAR(50) NOT NULL, -- success, error, timeout, cold_start
    duration_ms INTEGER,
    memory_used_mb INTEGER,
    input_size_bytes INTEGER,
    output_size_bytes INTEGER,
    error_message TEXT,
    cold_start BOOLEAN DEFAULT FALSE,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Function metrics aggregation
CREATE TABLE edge_function_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    function_id UUID NOT NULL REFERENCES edge_functions(id) ON DELETE CASCADE,
    location_id UUID REFERENCES edge_locations(id),
    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    invocation_count INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    error_count INTEGER DEFAULT 0,
    timeout_count INTEGER DEFAULT 0,
    cold_start_count INTEGER DEFAULT 0,
    avg_duration_ms DECIMAL(10, 2),
    p50_duration_ms DECIMAL(10, 2),
    p99_duration_ms DECIMAL(10, 2),
    avg_memory_mb DECIMAL(10, 2),
    total_billed_ms BIGINT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(function_id, location_id, period_start)
);

-- Function secrets (encrypted)
CREATE TABLE edge_function_secrets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    function_id UUID NOT NULL REFERENCES edge_functions(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    encrypted_value TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(function_id, name)
);

-- Function test results
CREATE TABLE edge_function_tests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    function_id UUID NOT NULL REFERENCES edge_functions(id) ON DELETE CASCADE,
    test_name VARCHAR(255) NOT NULL,
    input_payload JSONB NOT NULL,
    expected_output JSONB,
    actual_output JSONB,
    passed BOOLEAN,
    duration_ms INTEGER,
    error_message TEXT,
    executed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes
CREATE INDEX idx_edge_functions_tenant ON edge_functions(tenant_id);
CREATE INDEX idx_edge_functions_status ON edge_functions(status);
CREATE INDEX idx_edge_functions_tenant_status ON edge_functions(tenant_id, status);
CREATE INDEX idx_edge_function_versions_function ON edge_function_versions(function_id);
CREATE INDEX idx_edge_function_deployments_function ON edge_function_deployments(function_id);
CREATE INDEX idx_edge_function_deployments_location ON edge_function_deployments(location_id);
CREATE INDEX idx_edge_function_deployments_status ON edge_function_deployments(status);
CREATE INDEX idx_edge_function_triggers_function ON edge_function_triggers(function_id);
CREATE INDEX idx_edge_function_triggers_type ON edge_function_triggers(trigger_type);
CREATE INDEX idx_edge_function_invocations_function ON edge_function_invocations(function_id);
CREATE INDEX idx_edge_function_invocations_tenant ON edge_function_invocations(tenant_id);
CREATE INDEX idx_edge_function_invocations_started ON edge_function_invocations(started_at);
CREATE INDEX idx_edge_function_metrics_function ON edge_function_metrics(function_id);
CREATE INDEX idx_edge_function_metrics_period ON edge_function_metrics(period_start, period_end);
CREATE INDEX idx_edge_function_tests_function ON edge_function_tests(function_id);

-- Seed edge locations
INSERT INTO edge_locations (name, code, region, provider, status) VALUES
    ('Cloudflare - North America', 'cf-na', 'north-america', 'cloudflare', 'active'),
    ('Cloudflare - Europe', 'cf-eu', 'europe', 'cloudflare', 'active'),
    ('Cloudflare - Asia Pacific', 'cf-ap', 'asia-pacific', 'cloudflare', 'active'),
    ('AWS Lambda@Edge - US East', 'le-use1', 'us-east-1', 'lambda_edge', 'active'),
    ('AWS Lambda@Edge - EU West', 'le-euw1', 'eu-west-1', 'lambda_edge', 'active'),
    ('Fastly - Global', 'fly-global', 'global', 'fastly', 'active'),
    ('Local Runner', 'local', 'local', 'custom', 'active');
