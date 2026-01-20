-- Signature schemes table
CREATE TABLE IF NOT EXISTS signature_schemes (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL,
    algorithm VARCHAR(50) NOT NULL,
    config JSONB NOT NULL,
    key_config JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_signature_schemes_tenant ON signature_schemes(tenant_id);
CREATE INDEX idx_signature_schemes_type ON signature_schemes(tenant_id, type);

-- Signing keys table
CREATE TABLE IF NOT EXISTS signing_keys (
    id UUID PRIMARY KEY,
    scheme_id UUID NOT NULL REFERENCES signature_schemes(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    version INT NOT NULL,
    algorithm VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    secret_key TEXT,
    secret_hash VARCHAR(255),
    public_key TEXT,
    private_key TEXT,
    fingerprint VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    usage_count BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX idx_signing_keys_scheme ON signing_keys(scheme_id);
CREATE INDEX idx_signing_keys_status ON signing_keys(scheme_id, status);
CREATE INDEX idx_signing_keys_tenant ON signing_keys(tenant_id);
CREATE UNIQUE INDEX idx_signing_keys_version ON signing_keys(scheme_id, version);

-- Key rotations table
CREATE TABLE IF NOT EXISTS key_rotations (
    id UUID PRIMARY KEY,
    scheme_id UUID NOT NULL REFERENCES signature_schemes(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    old_key_id UUID NOT NULL REFERENCES signing_keys(id),
    new_key_id UUID NOT NULL REFERENCES signing_keys(id),
    status VARCHAR(50) NOT NULL DEFAULT 'scheduled',
    reason TEXT,
    scheduled_at TIMESTAMPTZ NOT NULL,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    overlap_until TIMESTAMPTZ,
    error TEXT
);

CREATE INDEX idx_key_rotations_scheme ON key_rotations(scheme_id);
CREATE INDEX idx_key_rotations_status ON key_rotations(status, scheduled_at);

-- Signature statistics table
CREATE TABLE IF NOT EXISTS signature_stats (
    scheme_id UUID PRIMARY KEY REFERENCES signature_schemes(id) ON DELETE CASCADE,
    total_signed BIGINT NOT NULL DEFAULT 0,
    total_verified BIGINT NOT NULL DEFAULT 0,
    total_failed BIGINT NOT NULL DEFAULT 0,
    last_signed_at TIMESTAMPTZ,
    last_verified_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Signature audit log (for compliance)
CREATE TABLE IF NOT EXISTS signature_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scheme_id UUID NOT NULL REFERENCES signature_schemes(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    operation VARCHAR(50) NOT NULL, -- 'sign', 'verify', 'rotate', 'revoke'
    key_id UUID,
    success BOOLEAN NOT NULL,
    error_code VARCHAR(50),
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_signature_audit_scheme ON signature_audit_log(scheme_id, created_at DESC);
CREATE INDEX idx_signature_audit_tenant ON signature_audit_log(tenant_id, created_at DESC);

-- Partition audit log by month for large deployments
-- This can be enabled as needed:
-- ALTER TABLE signature_audit_log PARTITION BY RANGE (created_at);
