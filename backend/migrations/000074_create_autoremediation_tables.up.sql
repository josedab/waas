CREATE TABLE IF NOT EXISTS failure_patterns (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    pattern_name VARCHAR(255) NOT NULL,
    description TEXT,
    event_type VARCHAR(255),
    error_code VARCHAR(100),
    error_message TEXT,
    frequency INTEGER NOT NULL DEFAULT 0,
    first_seen_at TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL,
    occurrence_count INTEGER NOT NULL DEFAULT 1,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    confidence DECIMAL(5,4) NOT NULL DEFAULT 0.0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS remediation_rules (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    pattern_id UUID NOT NULL REFERENCES failure_patterns(id),
    name VARCHAR(255) NOT NULL,
    action_type VARCHAR(100) NOT NULL,
    action_config JSONB NOT NULL DEFAULT '{}',
    is_automatic BOOLEAN NOT NULL DEFAULT FALSE,
    priority INTEGER NOT NULL DEFAULT 0,
    success_count INTEGER NOT NULL DEFAULT 0,
    failure_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS remediation_actions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    rule_id UUID NOT NULL REFERENCES remediation_rules(id),
    pattern_id UUID NOT NULL REFERENCES failure_patterns(id),
    action_type VARCHAR(100) NOT NULL,
    action_details TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    applied_at TIMESTAMPTZ,
    reverted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_failure_patterns_tenant ON failure_patterns(tenant_id);
CREATE INDEX idx_failure_patterns_status ON failure_patterns(tenant_id, status);
CREATE INDEX idx_remediation_rules_pattern ON remediation_rules(pattern_id);
CREATE INDEX idx_remediation_actions_rule ON remediation_actions(rule_id);
