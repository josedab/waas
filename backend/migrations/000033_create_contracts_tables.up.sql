-- Contract Testing Framework Tables
-- Feature 9: Consumer-driven contract validation

-- Contracts
CREATE TABLE IF NOT EXISTS webhook_contracts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    name VARCHAR(255) NOT NULL,
    description TEXT,
    version VARCHAR(50) NOT NULL,
    
    -- Contract participants
    provider_name VARCHAR(255) NOT NULL,
    consumer_name VARCHAR(255),
    
    -- Contract definition
    event_type VARCHAR(255) NOT NULL,
    schema_format VARCHAR(50) DEFAULT 'json_schema' CHECK (schema_format IN ('json_schema', 'avro', 'protobuf', 'openapi')),
    request_schema JSONB NOT NULL,
    response_schema JSONB,
    
    -- Headers and metadata
    required_headers TEXT[],
    optional_headers TEXT[],
    
    -- Lifecycle
    status VARCHAR(50) DEFAULT 'draft' CHECK (status IN ('draft', 'pending', 'active', 'deprecated', 'archived')),
    published_at TIMESTAMP WITH TIME ZONE,
    deprecated_at TIMESTAMP WITH TIME ZONE,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(tenant_id, name, version)
);

-- Contract Versions (immutable history)
CREATE TABLE IF NOT EXISTS contract_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contract_id UUID NOT NULL REFERENCES webhook_contracts(id) ON DELETE CASCADE,
    version VARCHAR(50) NOT NULL,
    
    -- Snapshot of contract at this version
    schema_snapshot JSONB NOT NULL,
    headers_snapshot JSONB,
    
    -- Changes
    change_summary TEXT,
    breaking_changes TEXT[],
    
    published_by VARCHAR(255),
    published_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(contract_id, version)
);

-- Contract Validations (test results)
CREATE TABLE IF NOT EXISTS contract_validations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contract_id UUID NOT NULL REFERENCES webhook_contracts(id) ON DELETE CASCADE,
    contract_version VARCHAR(50) NOT NULL,
    
    -- Validation context
    validation_type VARCHAR(50) NOT NULL CHECK (validation_type IN ('schema', 'example', 'live_traffic', 'ci_test')),
    source VARCHAR(100),
    
    -- Payload tested
    payload JSONB NOT NULL,
    headers JSONB,
    
    -- Results
    is_valid BOOLEAN NOT NULL,
    errors JSONB,
    warnings JSONB,
    
    -- Timing
    validation_time_ms INTEGER,
    validated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Breaking Change Detections
CREATE TABLE IF NOT EXISTS breaking_change_detections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contract_id UUID NOT NULL REFERENCES webhook_contracts(id) ON DELETE CASCADE,
    
    old_version VARCHAR(50) NOT NULL,
    new_version VARCHAR(50) NOT NULL,
    
    -- Detection results
    has_breaking_changes BOOLEAN NOT NULL,
    change_type VARCHAR(50) CHECK (change_type IN ('field_removed', 'field_type_changed', 'required_added', 'enum_value_removed', 'format_changed', 'other')),
    
    -- Details
    changes JSONB NOT NULL,
    affected_fields TEXT[],
    impact_assessment TEXT,
    
    -- Recommendations
    migration_steps JSONB,
    
    detected_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Consumer Subscriptions (who uses which contracts)
CREATE TABLE IF NOT EXISTS contract_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contract_id UUID NOT NULL REFERENCES webhook_contracts(id) ON DELETE CASCADE,
    
    consumer_id VARCHAR(255) NOT NULL,
    consumer_name VARCHAR(255),
    endpoint_id UUID REFERENCES webhook_endpoints(id) ON DELETE SET NULL,
    
    -- Subscription details
    subscribed_version VARCHAR(50) NOT NULL,
    notification_email VARCHAR(255),
    notify_on_breaking_changes BOOLEAN DEFAULT true,
    notify_on_deprecation BOOLEAN DEFAULT true,
    
    status VARCHAR(50) DEFAULT 'active' CHECK (status IN ('active', 'paused', 'cancelled')),
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(contract_id, consumer_id)
);

-- Contract Test Suites
CREATE TABLE IF NOT EXISTS contract_test_suites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contract_id UUID NOT NULL REFERENCES webhook_contracts(id) ON DELETE CASCADE,
    
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Test configuration
    test_cases JSONB NOT NULL,
    setup_script TEXT,
    teardown_script TEXT,
    
    -- Execution settings
    timeout_seconds INTEGER DEFAULT 30,
    retry_count INTEGER DEFAULT 0,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Contract Test Results
CREATE TABLE IF NOT EXISTS contract_test_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    test_suite_id UUID NOT NULL REFERENCES contract_test_suites(id) ON DELETE CASCADE,
    
    -- Execution context
    run_id VARCHAR(100) NOT NULL,
    environment VARCHAR(50),
    triggered_by VARCHAR(100),
    
    -- Results
    total_tests INTEGER NOT NULL,
    passed INTEGER NOT NULL,
    failed INTEGER NOT NULL,
    skipped INTEGER DEFAULT 0,
    
    -- Details
    test_results JSONB NOT NULL,
    duration_ms INTEGER,
    
    -- Status
    status VARCHAR(50) NOT NULL CHECK (status IN ('running', 'passed', 'failed', 'error')),
    error_message TEXT,
    
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Indexes
CREATE INDEX idx_webhook_contracts_tenant ON webhook_contracts(tenant_id);
CREATE INDEX idx_webhook_contracts_event_type ON webhook_contracts(event_type);
CREATE INDEX idx_webhook_contracts_status ON webhook_contracts(status);
CREATE INDEX idx_contract_versions_contract ON contract_versions(contract_id);
CREATE INDEX idx_contract_validations_contract ON contract_validations(contract_id, validated_at);
CREATE INDEX idx_breaking_changes_contract ON breaking_change_detections(contract_id);
CREATE INDEX idx_contract_subscriptions_contract ON contract_subscriptions(contract_id);
CREATE INDEX idx_contract_subscriptions_consumer ON contract_subscriptions(consumer_id);
CREATE INDEX idx_contract_test_results_suite ON contract_test_results(test_suite_id, started_at);
