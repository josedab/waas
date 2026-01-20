-- Feature 7: Self-Healing Endpoint Intelligence
-- ML-based failure prediction and automatic endpoint remediation

-- Endpoint health predictions from ML model
CREATE TABLE IF NOT EXISTS endpoint_health_predictions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    endpoint_id UUID NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    prediction_type VARCHAR(100) NOT NULL, -- failure, degradation, recovery
    probability DECIMAL(5,4) NOT NULL, -- 0.0000 to 1.0000
    confidence DECIMAL(5,4) NOT NULL,
    predicted_time TIMESTAMP WITH TIME ZONE,
    features_used JSONB DEFAULT '{}',
    model_version VARCHAR(50) NOT NULL,
    action_taken VARCHAR(100),
    action_taken_at TIMESTAMP WITH TIME ZONE,
    was_accurate BOOLEAN,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Endpoint behavior patterns for ML training
CREATE TABLE IF NOT EXISTS endpoint_behavior_patterns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    endpoint_id UUID NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    pattern_type VARCHAR(100) NOT NULL, -- response_time, error_rate, availability
    pattern_data JSONB NOT NULL,
    time_window_hours INTEGER NOT NULL DEFAULT 24,
    calculated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Auto-remediation rules
CREATE TABLE IF NOT EXISTS auto_remediation_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    trigger_condition JSONB NOT NULL,
    action_type VARCHAR(100) NOT NULL, -- disable_endpoint, adjust_retry, notify, circuit_break
    action_config JSONB DEFAULT '{}',
    enabled BOOLEAN NOT NULL DEFAULT true,
    cooldown_minutes INTEGER NOT NULL DEFAULT 30,
    last_triggered TIMESTAMP WITH TIME ZONE,
    trigger_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Remediation actions taken
CREATE TABLE IF NOT EXISTS remediation_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    endpoint_id UUID NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    rule_id UUID REFERENCES auto_remediation_rules(id) ON DELETE SET NULL,
    prediction_id UUID REFERENCES endpoint_health_predictions(id) ON DELETE SET NULL,
    action_type VARCHAR(100) NOT NULL,
    action_details JSONB DEFAULT '{}',
    previous_state JSONB,
    new_state JSONB,
    triggered_by VARCHAR(50) NOT NULL, -- auto, manual, prediction
    outcome VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, success, failed, reverted
    reverted_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Endpoint optimization suggestions
CREATE TABLE IF NOT EXISTS endpoint_optimization_suggestions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    endpoint_id UUID NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    suggestion_type VARCHAR(100) NOT NULL, -- retry_config, timeout_adjustment, rate_limit
    current_config JSONB NOT NULL,
    suggested_config JSONB NOT NULL,
    expected_improvement VARCHAR(255),
    confidence DECIMAL(5,4) NOT NULL,
    rationale TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, applied, dismissed
    applied_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Endpoint circuit breaker state
CREATE TABLE IF NOT EXISTS endpoint_circuit_breakers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    endpoint_id UUID NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    state VARCHAR(50) NOT NULL DEFAULT 'closed', -- closed, open, half_open
    failure_count INTEGER NOT NULL DEFAULT 0,
    success_count INTEGER NOT NULL DEFAULT 0,
    last_failure_at TIMESTAMP WITH TIME ZONE,
    last_success_at TIMESTAMP WITH TIME ZONE,
    opened_at TIMESTAMP WITH TIME ZONE,
    half_open_at TIMESTAMP WITH TIME ZONE,
    reset_timeout_seconds INTEGER NOT NULL DEFAULT 60,
    failure_threshold INTEGER NOT NULL DEFAULT 5,
    success_threshold INTEGER NOT NULL DEFAULT 3,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(endpoint_id)
);

-- Indexes for query optimization
CREATE INDEX IF NOT EXISTS idx_health_predictions_endpoint ON endpoint_health_predictions(endpoint_id);
CREATE INDEX IF NOT EXISTS idx_health_predictions_time ON endpoint_health_predictions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_behavior_patterns_endpoint ON endpoint_behavior_patterns(endpoint_id);
CREATE INDEX IF NOT EXISTS idx_remediation_rules_tenant ON auto_remediation_rules(tenant_id);
CREATE INDEX IF NOT EXISTS idx_remediation_actions_endpoint ON remediation_actions(endpoint_id);
CREATE INDEX IF NOT EXISTS idx_remediation_actions_time ON remediation_actions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_optimization_suggestions_endpoint ON endpoint_optimization_suggestions(endpoint_id);
CREATE INDEX IF NOT EXISTS idx_optimization_suggestions_status ON endpoint_optimization_suggestions(status);
CREATE INDEX IF NOT EXISTS idx_circuit_breakers_state ON endpoint_circuit_breakers(state);
