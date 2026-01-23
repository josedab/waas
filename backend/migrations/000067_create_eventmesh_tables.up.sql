CREATE TABLE IF NOT EXISTS eventmesh_routes (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    source_filter JSONB,
    targets JSONB NOT NULL DEFAULT '[]',
    priority INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS eventmesh_dead_letter_configs (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    route_id VARCHAR(36) NOT NULL REFERENCES eventmesh_routes(id) ON DELETE CASCADE,
    max_retries INTEGER NOT NULL DEFAULT 3,
    retention_days INTEGER NOT NULL DEFAULT 30,
    alert_on_entry BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, route_id)
);

CREATE TABLE IF NOT EXISTS eventmesh_dead_letter_entries (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    route_id VARCHAR(36) NOT NULL REFERENCES eventmesh_routes(id) ON DELETE CASCADE,
    payload JSONB NOT NULL,
    reason TEXT NOT NULL,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE IF NOT EXISTS eventmesh_executions (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    route_id VARCHAR(36) NOT NULL,
    source_event VARCHAR(255) NOT NULL,
    targets_hit INTEGER NOT NULL DEFAULT 0,
    targets_failed INTEGER NOT NULL DEFAULT 0,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_em_routes_tenant ON eventmesh_routes(tenant_id);
CREATE INDEX idx_em_routes_active ON eventmesh_routes(tenant_id) WHERE is_active = true;
CREATE INDEX idx_em_dl_entries_route ON eventmesh_dead_letter_entries(route_id);
CREATE INDEX idx_em_executions_route ON eventmesh_executions(route_id, created_at DESC);
