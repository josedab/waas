CREATE TABLE IF NOT EXISTS mtls_certificates (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    endpoint_id VARCHAR(36),
    domain VARCHAR(255) NOT NULL,
    issuer VARCHAR(255) NOT NULL DEFAULT 'WaaS Internal CA',
    serial_number VARCHAR(64) NOT NULL,
    not_before TIMESTAMP WITH TIME ZONE NOT NULL,
    not_after TIMESTAMP WITH TIME ZONE NOT NULL,
    fingerprint VARCHAR(64) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    auto_renew BOOLEAN NOT NULL DEFAULT true,
    last_renewed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS mtls_tls_policies (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    endpoint_id VARCHAR(36) NOT NULL,
    require_mtls BOOLEAN NOT NULL DEFAULT false,
    min_tls_version VARCHAR(10) NOT NULL DEFAULT '1.2',
    allowed_ciphers JSONB NOT NULL DEFAULT '[]',
    verify_server_cert BOOLEAN NOT NULL DEFAULT true,
    certificate_id VARCHAR(36) REFERENCES mtls_certificates(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, endpoint_id)
);

CREATE INDEX idx_mtls_certs_tenant ON mtls_certificates(tenant_id);
CREATE INDEX idx_mtls_certs_expiry ON mtls_certificates(not_after) WHERE status = 'active';
CREATE INDEX idx_mtls_policies_tenant ON mtls_tls_policies(tenant_id);
