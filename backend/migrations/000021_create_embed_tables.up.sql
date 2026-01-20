-- Embed tokens table
CREATE TABLE IF NOT EXISTS embed_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    token VARCHAR(255) NOT NULL UNIQUE,
    permissions TEXT[] NOT NULL,
    scopes JSONB NOT NULL DEFAULT '{}',
    theme JSONB,
    expires_at TIMESTAMP WITH TIME ZONE,
    allowed_origins TEXT[] DEFAULT ARRAY[]::TEXT[],
    metadata JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_embed_tokens_tenant_id ON embed_tokens(tenant_id);
CREATE INDEX idx_embed_tokens_token ON embed_tokens(token);
CREATE INDEX idx_embed_tokens_is_active ON embed_tokens(is_active);

-- Embed sessions table
CREATE TABLE IF NOT EXISTS embed_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_id UUID NOT NULL REFERENCES embed_tokens(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    origin TEXT,
    user_agent TEXT,
    ip VARCHAR(45),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_embed_sessions_token_id ON embed_sessions(token_id);
CREATE INDEX idx_embed_sessions_tenant_id ON embed_sessions(tenant_id);
CREATE INDEX idx_embed_sessions_last_seen ON embed_sessions(last_seen);
