-- Zero-Trust Delivery Tables
-- Feature 7: mTLS, certificate pinning, request signing

-- Endpoint Certificates (for mTLS)
CREATE TABLE IF NOT EXISTS endpoint_certificates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id UUID NOT NULL REFERENCES webhook_endpoints(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Certificate data
    certificate_type VARCHAR(50) NOT NULL CHECK (certificate_type IN ('client_cert', 'pinned_cert', 'ca_bundle')),
    certificate_pem TEXT NOT NULL,
    private_key_pem TEXT, -- Only for client certs, encrypted at rest
    
    -- Certificate metadata
    common_name VARCHAR(255),
    issuer VARCHAR(255),
    serial_number VARCHAR(255),
    fingerprint_sha256 VARCHAR(64) NOT NULL,
    
    -- Validity
    not_before TIMESTAMP WITH TIME ZONE NOT NULL,
    not_after TIMESTAMP WITH TIME ZONE NOT NULL,
    is_active BOOLEAN DEFAULT true,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Signing Keys (for request signing)
CREATE TABLE IF NOT EXISTS signing_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Key metadata
    name VARCHAR(255) NOT NULL,
    description TEXT,
    key_type VARCHAR(50) NOT NULL CHECK (key_type IN ('hmac_sha256', 'hmac_sha512', 'rsa_sha256', 'ed25519')),
    
    -- Key material (encrypted at rest)
    public_key TEXT,
    private_key_encrypted TEXT NOT NULL,
    key_id VARCHAR(64) NOT NULL, -- External identifier for key rotation
    
    -- Lifecycle
    status VARCHAR(50) DEFAULT 'active' CHECK (status IN ('active', 'rotating', 'revoked', 'expired')),
    version INTEGER DEFAULT 1,
    expires_at TIMESTAMP WITH TIME ZONE,
    rotated_from UUID REFERENCES signing_keys(id),
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(tenant_id, key_id)
);

-- Endpoint Security Profiles
CREATE TABLE IF NOT EXISTS endpoint_security_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id UUID NOT NULL REFERENCES webhook_endpoints(id) ON DELETE CASCADE UNIQUE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- mTLS settings
    mtls_enabled BOOLEAN DEFAULT false,
    client_certificate_id UUID REFERENCES endpoint_certificates(id),
    verify_server_cert BOOLEAN DEFAULT true,
    
    -- Certificate pinning
    pinning_enabled BOOLEAN DEFAULT false,
    pinning_mode VARCHAR(50) CHECK (pinning_mode IN ('certificate', 'public_key', 'spki')),
    pinned_certificate_ids UUID[],
    
    -- Request signing
    signing_enabled BOOLEAN DEFAULT false,
    signing_key_id UUID REFERENCES signing_keys(id),
    signing_algorithm VARCHAR(50),
    signature_header VARCHAR(100) DEFAULT 'X-Webhook-Signature',
    timestamp_header VARCHAR(100) DEFAULT 'X-Webhook-Timestamp',
    signature_format VARCHAR(50) DEFAULT 'base64' CHECK (signature_format IN ('base64', 'hex')),
    include_body BOOLEAN DEFAULT true,
    include_headers TEXT[],
    
    -- Security policies
    require_https BOOLEAN DEFAULT true,
    min_tls_version VARCHAR(10) DEFAULT '1.2',
    allowed_cipher_suites TEXT[],
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Signature Verification Logs
CREATE TABLE IF NOT EXISTS signature_verifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_id UUID NOT NULL,
    endpoint_id UUID NOT NULL,
    
    -- Verification details
    verification_type VARCHAR(50) NOT NULL CHECK (verification_type IN ('request_signature', 'response_signature', 'certificate')),
    key_id VARCHAR(64),
    algorithm VARCHAR(50),
    
    -- Result
    is_valid BOOLEAN NOT NULL,
    error_code VARCHAR(50),
    error_message TEXT,
    
    -- Debug info
    expected_signature TEXT,
    received_signature TEXT,
    signed_payload_preview TEXT,
    
    verified_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Certificate Rotation History
CREATE TABLE IF NOT EXISTS certificate_rotations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    
    old_certificate_id UUID,
    new_certificate_id UUID NOT NULL REFERENCES endpoint_certificates(id),
    
    rotation_reason VARCHAR(100),
    initiated_by VARCHAR(100),
    
    status VARCHAR(50) DEFAULT 'completed' CHECK (status IN ('pending', 'in_progress', 'completed', 'failed', 'rolled_back')),
    error_message TEXT,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Indexes
CREATE INDEX idx_endpoint_certificates_endpoint ON endpoint_certificates(endpoint_id);
CREATE INDEX idx_endpoint_certificates_fingerprint ON endpoint_certificates(fingerprint_sha256);
CREATE INDEX idx_endpoint_certificates_expiry ON endpoint_certificates(not_after) WHERE is_active = true;
CREATE INDEX idx_signing_keys_tenant ON signing_keys(tenant_id);
CREATE INDEX idx_signing_keys_key_id ON signing_keys(tenant_id, key_id);
CREATE INDEX idx_security_profiles_endpoint ON endpoint_security_profiles(endpoint_id);
CREATE INDEX idx_signature_verifications_delivery ON signature_verifications(delivery_id);
CREATE INDEX idx_certificate_rotations_endpoint ON certificate_rotations(endpoint_id);
