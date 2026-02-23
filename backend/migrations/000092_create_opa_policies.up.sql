-- Programmable Policy Engine (OPA/Rego)
-- Stores Rego policies with versioning for webhook routing, filtering, authorization

CREATE TABLE IF NOT EXISTS opa_policies (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       VARCHAR(255) NOT NULL,
    name            VARCHAR(255) NOT NULL,
    description     TEXT DEFAULT '',
    rego_source     TEXT NOT NULL,
    version         INTEGER NOT NULL DEFAULT 1,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    policy_type     VARCHAR(50) NOT NULL DEFAULT 'routing',
    last_evaluated  TIMESTAMP WITH TIME ZONE,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_opa_policies_tenant ON opa_policies(tenant_id);
CREATE INDEX idx_opa_policies_active ON opa_policies(tenant_id, is_active);
CREATE UNIQUE INDEX idx_opa_policies_name_version ON opa_policies(tenant_id, name, version);

CREATE TABLE IF NOT EXISTS opa_policy_versions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id       UUID NOT NULL REFERENCES opa_policies(id) ON DELETE CASCADE,
    version         INTEGER NOT NULL,
    rego_source     TEXT NOT NULL,
    change_note     TEXT DEFAULT '',
    created_by      VARCHAR(255) NOT NULL DEFAULT 'system',
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_opa_versions_policy ON opa_policy_versions(policy_id);

CREATE TABLE IF NOT EXISTS opa_evaluation_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       VARCHAR(255) NOT NULL,
    policy_id       UUID REFERENCES opa_policies(id),
    decision        BOOLEAN NOT NULL,
    input_hash      VARCHAR(128) NOT NULL DEFAULT '',
    duration_ms     INTEGER NOT NULL DEFAULT 0,
    is_dry_run      BOOLEAN NOT NULL DEFAULT false,
    result          JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_opa_eval_tenant ON opa_evaluation_logs(tenant_id);
CREATE INDEX idx_opa_eval_policy ON opa_evaluation_logs(policy_id);
CREATE INDEX idx_opa_eval_created ON opa_evaluation_logs(created_at);
