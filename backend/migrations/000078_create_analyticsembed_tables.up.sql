CREATE TABLE IF NOT EXISTS analytics_widgets (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    widget_type VARCHAR(100) NOT NULL,
    data_source VARCHAR(255),
    time_range VARCHAR(100) NOT NULL DEFAULT '24h',
    refresh_interval_sec INTEGER NOT NULL DEFAULT 30,
    custom_css TEXT,
    is_public BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS analytics_embed_tokens (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    widget_id UUID NOT NULL REFERENCES analytics_widgets(id),
    token VARCHAR(512) NOT NULL UNIQUE,
    scopes JSONB NOT NULL DEFAULT '[]',
    allowed_origins JSONB NOT NULL DEFAULT '[]',
    expires_at TIMESTAMPTZ NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS analytics_themes (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL UNIQUE,
    primary_color VARCHAR(50) NOT NULL DEFAULT '#3B82F6',
    secondary_color VARCHAR(50) NOT NULL DEFAULT '#10B981',
    background_color VARCHAR(50) NOT NULL DEFAULT '#FFFFFF',
    text_color VARCHAR(50) NOT NULL DEFAULT '#1F2937',
    font_family VARCHAR(255) NOT NULL DEFAULT 'Inter, sans-serif',
    border_radius VARCHAR(50) NOT NULL DEFAULT '8px',
    custom_css TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_analytics_widgets_tenant ON analytics_widgets(tenant_id);
CREATE INDEX idx_analytics_embed_tokens_token ON analytics_embed_tokens(token);
CREATE INDEX idx_analytics_embed_tokens_widget ON analytics_embed_tokens(widget_id);
