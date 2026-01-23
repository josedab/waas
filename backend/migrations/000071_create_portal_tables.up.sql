CREATE TABLE IF NOT EXISTS portal_configs (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    branding JSONB,
    allowed_origins JSONB NOT NULL DEFAULT '[]',
    features JSONB,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS portal_embed_tokens (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    portal_id VARCHAR(36) NOT NULL REFERENCES portal_configs(id) ON DELETE CASCADE,
    token VARCHAR(128) NOT NULL UNIQUE,
    scopes JSONB NOT NULL DEFAULT '[]',
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS portal_sessions (
    id VARCHAR(36) PRIMARY KEY,
    token_id VARCHAR(36) NOT NULL REFERENCES portal_embed_tokens(id) ON DELETE CASCADE,
    tenant_id VARCHAR(36) NOT NULL,
    user_agent TEXT,
    origin VARCHAR(255),
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_seen_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_portal_configs_tenant ON portal_configs(tenant_id);
CREATE INDEX idx_portal_tokens_portal ON portal_embed_tokens(portal_id);
CREATE INDEX idx_portal_tokens_token ON portal_embed_tokens(token);
CREATE INDEX idx_portal_sessions_tenant ON portal_sessions(tenant_id);
