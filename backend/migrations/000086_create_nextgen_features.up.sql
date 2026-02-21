-- Observability-as-Code Pipeline tables
CREATE TABLE IF NOT EXISTS obs_pipelines (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    version INTEGER DEFAULT 1,
    status TEXT NOT NULL DEFAULT 'draft',
    spec JSONB NOT NULL,
    checksum TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_obs_pipelines_tenant ON obs_pipelines(tenant_id);
CREATE INDEX idx_obs_pipelines_status ON obs_pipelines(tenant_id, status);

CREATE TABLE IF NOT EXISTS obs_pipeline_executions (
    id TEXT PRIMARY KEY,
    pipeline_id TEXT NOT NULL REFERENCES obs_pipelines(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'running',
    metrics_emitted BIGINT DEFAULT 0,
    traces_emitted BIGINT DEFAULT 0,
    logs_emitted BIGINT DEFAULT 0,
    errors JSONB DEFAULT '[]',
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_obs_executions_pipeline ON obs_pipeline_executions(pipeline_id);

CREATE TABLE IF NOT EXISTS obs_alert_events (
    id TEXT PRIMARY KEY,
    pipeline_id TEXT NOT NULL REFERENCES obs_pipelines(id) ON DELETE CASCADE,
    tenant_id TEXT NOT NULL,
    rule_name TEXT NOT NULL,
    severity TEXT NOT NULL DEFAULT 'info',
    message TEXT NOT NULL,
    value DOUBLE PRECISION DEFAULT 0,
    threshold DOUBLE PRECISION DEFAULT 0,
    labels JSONB DEFAULT '{}',
    fired_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    resolved_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_obs_alerts_tenant ON obs_alert_events(tenant_id);
CREATE INDEX idx_obs_alerts_active ON obs_alert_events(tenant_id) WHERE resolved_at IS NULL;

-- Compliance Vault tables
CREATE TABLE IF NOT EXISTS compliance_vault_entries (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    webhook_id TEXT NOT NULL,
    endpoint_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    encrypted_payload BYTEA NOT NULL,
    payload_hash TEXT NOT NULL,
    encryption_algo TEXT NOT NULL DEFAULT 'aes-256-gcm',
    key_id TEXT DEFAULT '',
    metadata JSONB DEFAULT '{}',
    content_type TEXT DEFAULT 'application/json',
    size_bytes BIGINT DEFAULT 0,
    retain_until TIMESTAMP WITH TIME ZONE,
    frameworks JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_vault_entries_tenant ON compliance_vault_entries(tenant_id);
CREATE INDEX idx_vault_entries_webhook ON compliance_vault_entries(tenant_id, webhook_id);
CREATE INDEX idx_vault_entries_expiry ON compliance_vault_entries(expires_at) WHERE expires_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS compliance_retention_policies (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    name TEXT NOT NULL,
    framework TEXT NOT NULL,
    retention_days INTEGER NOT NULL,
    action TEXT NOT NULL DEFAULT 'delete',
    event_type_filter TEXT DEFAULT '',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_retention_policies_tenant ON compliance_retention_policies(tenant_id);

CREATE TABLE IF NOT EXISTS compliance_audit_trail (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    entry_id TEXT DEFAULT '',
    actor_id TEXT NOT NULL,
    actor_type TEXT NOT NULL DEFAULT 'user',
    action TEXT NOT NULL,
    resource TEXT NOT NULL,
    details JSONB DEFAULT '{}',
    ip_address TEXT DEFAULT '',
    user_agent TEXT DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_audit_trail_tenant ON compliance_audit_trail(tenant_id);
CREATE INDEX idx_audit_trail_entry ON compliance_audit_trail(entry_id) WHERE entry_id != '';

CREATE TABLE IF NOT EXISTS compliance_encryption_keys (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    algorithm TEXT NOT NULL DEFAULT 'aes-256-gcm',
    version INTEGER DEFAULT 1,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    rotated_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_encryption_keys_tenant ON compliance_encryption_keys(tenant_id, is_active);

CREATE TABLE IF NOT EXISTS compliance_erasure_requests (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    subject_id TEXT NOT NULL,
    subject_type TEXT NOT NULL,
    reason TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending',
    entries_found INTEGER DEFAULT 0,
    entries_erased INTEGER DEFAULT 0,
    requested_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_erasure_requests_tenant ON compliance_erasure_requests(tenant_id);

-- Portal SDK tables
CREATE TABLE IF NOT EXISTS portal_configs (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    name TEXT NOT NULL,
    allowed_origins JSONB DEFAULT '[]',
    components JSONB DEFAULT '[]',
    theme JSONB DEFAULT '{}',
    features JSONB DEFAULT '{}',
    branding JSONB DEFAULT '{}',
    custom_css TEXT DEFAULT '',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_portal_configs_tenant ON portal_configs(tenant_id);

CREATE TABLE IF NOT EXISTS portal_sessions (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    config_id TEXT NOT NULL REFERENCES portal_configs(id) ON DELETE CASCADE,
    customer_id TEXT NOT NULL,
    token TEXT NOT NULL UNIQUE,
    permissions JSONB DEFAULT '[]',
    scopes JSONB DEFAULT '{}',
    origin TEXT DEFAULT '',
    user_agent TEXT DEFAULT '',
    ip_address TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active',
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_access_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_portal_sessions_token ON portal_sessions(token);
CREATE INDEX idx_portal_sessions_config ON portal_sessions(config_id);
CREATE INDEX idx_portal_sessions_active ON portal_sessions(status) WHERE status = 'active';
