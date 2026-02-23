-- Delivery Receipt & Processing Confirmation Protocol

CREATE TABLE IF NOT EXISTS delivery_receipts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       VARCHAR(255) NOT NULL,
    delivery_id     VARCHAR(255) NOT NULL,
    webhook_id      VARCHAR(255) NOT NULL,
    endpoint_id     VARCHAR(255) NOT NULL,
    receipt_url     TEXT NOT NULL DEFAULT '',
    status          VARCHAR(50) NOT NULL DEFAULT 'pending',
    http_status     INTEGER NOT NULL DEFAULT 0,
    processing_status VARCHAR(50) NOT NULL DEFAULT '',
    processing_details JSONB DEFAULT '{}',
    confirm_window_sec INTEGER NOT NULL DEFAULT 300,
    confirmed_at    TIMESTAMP WITH TIME ZONE,
    expires_at      TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_receipts_tenant ON delivery_receipts(tenant_id);
CREATE INDEX idx_receipts_delivery ON delivery_receipts(delivery_id);
CREATE INDEX idx_receipts_status ON delivery_receipts(status);
CREATE INDEX idx_receipts_expires ON delivery_receipts(expires_at);
