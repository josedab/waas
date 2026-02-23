-- Event Correlation & Complex Event Processing

CREATE TABLE IF NOT EXISTS correlation_rules (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       VARCHAR(255) NOT NULL,
    name            VARCHAR(255) NOT NULL,
    description     TEXT DEFAULT '',
    trigger_event   VARCHAR(255) NOT NULL,
    follow_event    VARCHAR(255) NOT NULL,
    time_window_sec INTEGER NOT NULL DEFAULT 300,
    match_fields    JSONB NOT NULL DEFAULT '[]',
    composite_event VARCHAR(255) NOT NULL DEFAULT '',
    is_enabled      BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_corr_rules_tenant ON correlation_rules(tenant_id);
CREATE INDEX idx_corr_rules_trigger ON correlation_rules(tenant_id, trigger_event);

CREATE TABLE IF NOT EXISTS correlation_state (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id         UUID NOT NULL REFERENCES correlation_rules(id) ON DELETE CASCADE,
    tenant_id       VARCHAR(255) NOT NULL,
    trigger_event_id VARCHAR(255) NOT NULL,
    match_key       VARCHAR(512) NOT NULL,
    payload         JSONB NOT NULL DEFAULT '{}',
    status          VARCHAR(50) NOT NULL DEFAULT 'pending',
    expires_at      TIMESTAMP WITH TIME ZONE NOT NULL,
    correlated_at   TIMESTAMP WITH TIME ZONE,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_corr_state_match ON correlation_state(rule_id, match_key, status);
CREATE INDEX idx_corr_state_expires ON correlation_state(expires_at);

CREATE TABLE IF NOT EXISTS correlation_matches (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id         UUID NOT NULL REFERENCES correlation_rules(id),
    tenant_id       VARCHAR(255) NOT NULL,
    trigger_event_id VARCHAR(255) NOT NULL,
    follow_event_id  VARCHAR(255) NOT NULL,
    match_key       VARCHAR(512) NOT NULL,
    composite_event_id VARCHAR(255) NOT NULL DEFAULT '',
    matched_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_corr_matches_tenant ON correlation_matches(tenant_id);
CREATE INDEX idx_corr_matches_rule ON correlation_matches(rule_id);
