-- PII Detection & Payload Masking Engine
-- Stores per-tenant PII detection policies and scan results

CREATE TABLE IF NOT EXISTS pii_detection_policies (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       VARCHAR(255) NOT NULL,
    name            VARCHAR(255) NOT NULL,
    description     TEXT DEFAULT '',
    sensitivity     VARCHAR(50) NOT NULL DEFAULT 'medium',
    categories      JSONB NOT NULL DEFAULT '[]',
    custom_patterns JSONB NOT NULL DEFAULT '[]',
    masking_action  VARCHAR(50) NOT NULL DEFAULT 'mask',
    is_enabled      BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_pii_policies_tenant ON pii_detection_policies(tenant_id);
CREATE INDEX idx_pii_policies_enabled ON pii_detection_policies(tenant_id, is_enabled);

CREATE TABLE IF NOT EXISTS pii_scan_results (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       VARCHAR(255) NOT NULL,
    policy_id       UUID REFERENCES pii_detection_policies(id),
    webhook_id      VARCHAR(255) NOT NULL,
    endpoint_id     VARCHAR(255) NOT NULL,
    event_type      VARCHAR(255) NOT NULL DEFAULT '',
    detections      JSONB NOT NULL DEFAULT '[]',
    fields_scanned  INTEGER NOT NULL DEFAULT 0,
    fields_masked   INTEGER NOT NULL DEFAULT 0,
    masking_action  VARCHAR(50) NOT NULL DEFAULT 'mask',
    original_hash   VARCHAR(128) NOT NULL DEFAULT '',
    masked_hash     VARCHAR(128) NOT NULL DEFAULT '',
    scan_duration_ms INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_pii_scans_tenant ON pii_scan_results(tenant_id);
CREATE INDEX idx_pii_scans_webhook ON pii_scan_results(webhook_id);
CREATE INDEX idx_pii_scans_created ON pii_scan_results(created_at);
