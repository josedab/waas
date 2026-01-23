CREATE TABLE IF NOT EXISTS schema_definitions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    subject VARCHAR(255) NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    schema_format VARCHAR(50) NOT NULL,
    schema_content TEXT NOT NULL,
    fingerprint VARCHAR(128) NOT NULL,
    description TEXT,
    is_latest BOOLEAN NOT NULL DEFAULT TRUE,
    compatibility_mode VARCHAR(50) NOT NULL DEFAULT 'backward',
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, subject, version)
);

CREATE TABLE IF NOT EXISTS schema_versions (
    id UUID PRIMARY KEY,
    schema_id UUID NOT NULL REFERENCES schema_definitions(id),
    version INTEGER NOT NULL,
    schema_content TEXT NOT NULL,
    change_log TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_schema_definitions_tenant ON schema_definitions(tenant_id);
CREATE INDEX idx_schema_definitions_subject ON schema_definitions(tenant_id, subject);
CREATE INDEX idx_schema_definitions_fingerprint ON schema_definitions(fingerprint);
CREATE INDEX idx_schema_versions_schema ON schema_versions(schema_id);
