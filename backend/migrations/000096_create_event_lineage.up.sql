-- Event Lineage & Provenance Tracker

CREATE TABLE IF NOT EXISTS event_lineage (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       VARCHAR(255) NOT NULL,
    event_id        VARCHAR(255) NOT NULL,
    parent_event_id VARCHAR(255) DEFAULT '',
    event_type      VARCHAR(255) NOT NULL,
    source          VARCHAR(255) NOT NULL DEFAULT '',
    operation       VARCHAR(100) NOT NULL DEFAULT '',
    metadata        JSONB DEFAULT '{}',
    payload_hash    VARCHAR(128) NOT NULL DEFAULT '',
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_lineage_tenant ON event_lineage(tenant_id);
CREATE INDEX idx_lineage_event ON event_lineage(event_id);
CREATE INDEX idx_lineage_parent ON event_lineage(parent_event_id);
CREATE INDEX idx_lineage_created ON event_lineage(created_at);
