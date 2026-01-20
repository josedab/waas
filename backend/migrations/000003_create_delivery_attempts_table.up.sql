CREATE TABLE delivery_attempts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    endpoint_id UUID NOT NULL REFERENCES webhook_endpoints(id) ON DELETE CASCADE,
    payload_hash VARCHAR(64) NOT NULL,
    payload_size INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    http_status INTEGER,
    response_body TEXT,
    error_message TEXT,
    attempt_number INTEGER NOT NULL DEFAULT 1,
    scheduled_at TIMESTAMP NOT NULL DEFAULT NOW(),
    delivered_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_delivery_attempts_endpoint ON delivery_attempts(endpoint_id);
CREATE INDEX idx_delivery_attempts_status ON delivery_attempts(endpoint_id, status);
CREATE INDEX idx_delivery_attempts_scheduled ON delivery_attempts(scheduled_at, status);
CREATE INDEX idx_delivery_attempts_created ON delivery_attempts(created_at);

-- Add constraint to ensure valid status values
ALTER TABLE delivery_attempts ADD CONSTRAINT chk_delivery_status 
    CHECK (status IN ('pending', 'processing', 'delivered', 'failed', 'retrying'));