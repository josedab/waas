CREATE TABLE IF NOT EXISTS config_manifests (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    content TEXT NOT NULL,
    checksum VARCHAR(128) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    applied_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS config_resources (
    id UUID PRIMARY KEY,
    manifest_id UUID NOT NULL REFERENCES config_manifests(id),
    resource_type VARCHAR(100) NOT NULL,
    resource_id VARCHAR(255) NOT NULL,
    action VARCHAR(50) NOT NULL DEFAULT 'no_change',
    previous_state JSONB,
    desired_state JSONB,
    status VARCHAR(50) NOT NULL DEFAULT 'pending'
);

CREATE TABLE IF NOT EXISTS drift_reports (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resource_count INTEGER NOT NULL DEFAULT 0,
    drifted_count INTEGER NOT NULL DEFAULT 0,
    details JSONB NOT NULL DEFAULT '[]',
    status VARCHAR(50) NOT NULL DEFAULT 'detected'
);

CREATE INDEX idx_config_manifests_tenant ON config_manifests(tenant_id);
CREATE INDEX idx_config_manifests_status ON config_manifests(tenant_id, status);
CREATE INDEX idx_config_resources_manifest ON config_resources(manifest_id);
CREATE INDEX idx_drift_reports_tenant ON drift_reports(tenant_id);
