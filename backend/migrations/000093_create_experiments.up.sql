-- Webhook A/B Testing & Experimentation Framework

CREATE TABLE IF NOT EXISTS experiments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       VARCHAR(255) NOT NULL,
    name            VARCHAR(255) NOT NULL,
    description     TEXT DEFAULT '',
    status          VARCHAR(50) NOT NULL DEFAULT 'draft',
    event_type      VARCHAR(255) NOT NULL DEFAULT '',
    variants        JSONB NOT NULL DEFAULT '[]',
    success_criteria JSONB NOT NULL DEFAULT '{}',
    winner_variant  VARCHAR(255) DEFAULT '',
    started_at      TIMESTAMP WITH TIME ZONE,
    ended_at        TIMESTAMP WITH TIME ZONE,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_experiments_tenant ON experiments(tenant_id);
CREATE INDEX idx_experiments_status ON experiments(tenant_id, status);

CREATE TABLE IF NOT EXISTS experiment_assignments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    experiment_id   UUID NOT NULL REFERENCES experiments(id) ON DELETE CASCADE,
    webhook_id      VARCHAR(255) NOT NULL,
    variant_id      VARCHAR(255) NOT NULL,
    assigned_at     TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_exp_assignments_experiment ON experiment_assignments(experiment_id);
CREATE INDEX idx_exp_assignments_webhook ON experiment_assignments(webhook_id);

CREATE TABLE IF NOT EXISTS experiment_metrics (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    experiment_id   UUID NOT NULL REFERENCES experiments(id) ON DELETE CASCADE,
    variant_id      VARCHAR(255) NOT NULL,
    total_requests  BIGINT NOT NULL DEFAULT 0,
    success_count   BIGINT NOT NULL DEFAULT 0,
    failure_count   BIGINT NOT NULL DEFAULT 0,
    avg_latency_ms  DOUBLE PRECISION NOT NULL DEFAULT 0,
    p99_latency_ms  DOUBLE PRECISION NOT NULL DEFAULT 0,
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_exp_metrics_experiment ON experiment_metrics(experiment_id);
