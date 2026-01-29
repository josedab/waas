CREATE TABLE IF NOT EXISTS inbound_transform_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL REFERENCES inbound_sources(id) ON DELETE CASCADE,
    field_path TEXT NOT NULL,
    transform_type VARCHAR(50) NOT NULL,
    expression TEXT NOT NULL DEFAULT '',
    target_field TEXT NOT NULL DEFAULT '',
    priority INTEGER NOT NULL DEFAULT 0,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS inbound_content_routes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL REFERENCES inbound_sources(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    filter_expression TEXT NOT NULL DEFAULT '',
    destination_type VARCHAR(50) NOT NULL DEFAULT 'http',
    destination_url TEXT NOT NULL,
    headers JSONB DEFAULT '{}',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    priority INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_inbound_transform_rules_source ON inbound_transform_rules(source_id);
CREATE INDEX idx_inbound_content_routes_source ON inbound_content_routes(source_id);
