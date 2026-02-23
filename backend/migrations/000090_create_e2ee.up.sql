-- E2EE key pairs and audit
CREATE TABLE IF NOT EXISTS e2ee_key_pairs (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    endpoint_id TEXT NOT NULL,
    public_key TEXT NOT NULL,
    private_key_encrypted TEXT NOT NULL,
    algorithm TEXT NOT NULL DEFAULT 'x25519',
    status TEXT NOT NULL DEFAULT 'active',
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    rotated_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE INDEX idx_e2ee_keys_endpoint ON e2ee_key_pairs(endpoint_id);
CREATE INDEX idx_e2ee_keys_status ON e2ee_key_pairs(endpoint_id, status);

CREATE TABLE IF NOT EXISTS e2ee_audit_log (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    endpoint_id TEXT NOT NULL,
    operation TEXT NOT NULL,
    key_version INT NOT NULL,
    success BOOLEAN NOT NULL DEFAULT TRUE,
    details TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_e2ee_audit_endpoint ON e2ee_audit_log(endpoint_id);
CREATE INDEX idx_e2ee_audit_created ON e2ee_audit_log(created_at);
