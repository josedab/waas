-- Playground and IDE Tables
-- Feature 3: Webhook IDE/Playground for testing and debugging

-- Playground Sessions
CREATE TABLE IF NOT EXISTS playground_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255),
    description TEXT,
    transformation_code TEXT,
    input_payload JSONB,
    output_payload JSONB,
    last_execution_at TIMESTAMP WITH TIME ZONE,
    execution_count INTEGER DEFAULT 0,
    is_saved BOOLEAN DEFAULT false,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Request Captures (for inspection and replay)
CREATE TABLE IF NOT EXISTS request_captures (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    session_id UUID REFERENCES playground_sessions(id) ON DELETE CASCADE,
    endpoint_id UUID REFERENCES webhook_endpoints(id) ON DELETE SET NULL,
    method VARCHAR(10) NOT NULL DEFAULT 'POST',
    url TEXT,
    headers JSONB,
    body JSONB,
    query_params JSONB,
    response_status INTEGER,
    response_headers JSONB,
    response_body JSONB,
    duration_ms INTEGER,
    error_message TEXT,
    source VARCHAR(50) DEFAULT 'capture' CHECK (source IN ('capture', 'manual', 'replay', 'mock')),
    tags TEXT[],
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Transformation Executions (history of transformation runs)
CREATE TABLE IF NOT EXISTS transformation_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID REFERENCES playground_sessions(id) ON DELETE CASCADE,
    transformation_id UUID REFERENCES transformations(id) ON DELETE SET NULL,
    input_payload JSONB NOT NULL,
    output_payload JSONB,
    transformation_code TEXT,
    execution_time_ms INTEGER,
    memory_used_bytes BIGINT,
    success BOOLEAN NOT NULL,
    error_message TEXT,
    error_stack TEXT,
    logs JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Debug Breakpoints
CREATE TABLE IF NOT EXISTS debug_breakpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES playground_sessions(id) ON DELETE CASCADE,
    line_number INTEGER NOT NULL,
    condition TEXT,
    is_enabled BOOLEAN DEFAULT true,
    hit_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Saved Snippets (reusable code/payloads)
CREATE TABLE IF NOT EXISTS playground_snippets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    snippet_type VARCHAR(50) NOT NULL CHECK (snippet_type IN ('transformation', 'payload', 'headers', 'script')),
    content TEXT NOT NULL,
    language VARCHAR(50) DEFAULT 'javascript',
    tags TEXT[],
    is_public BOOLEAN DEFAULT false,
    use_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_playground_sessions_tenant ON playground_sessions(tenant_id);
CREATE INDEX idx_playground_sessions_expires ON playground_sessions(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_request_captures_tenant ON request_captures(tenant_id);
CREATE INDEX idx_request_captures_session ON request_captures(session_id);
CREATE INDEX idx_request_captures_endpoint ON request_captures(endpoint_id);
CREATE INDEX idx_request_captures_created ON request_captures(created_at);
CREATE INDEX idx_transformation_executions_session ON transformation_executions(session_id);
CREATE INDEX idx_playground_snippets_tenant ON playground_snippets(tenant_id);
CREATE INDEX idx_playground_snippets_type ON playground_snippets(snippet_type);
