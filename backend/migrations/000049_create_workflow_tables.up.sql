-- Workflows table
CREATE TABLE IF NOT EXISTS workflows (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    version INT NOT NULL DEFAULT 1,
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    trigger_config JSONB NOT NULL,
    nodes JSONB NOT NULL DEFAULT '[]',
    edges JSONB NOT NULL DEFAULT '[]',
    variables JSONB NOT NULL DEFAULT '[]',
    settings JSONB NOT NULL,
    canvas JSONB NOT NULL,
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ
);

CREATE INDEX idx_workflows_tenant ON workflows(tenant_id);
CREATE INDEX idx_workflows_status ON workflows(tenant_id, status);
CREATE INDEX idx_workflows_name ON workflows(tenant_id, name);

-- Workflow versions table (for version history)
CREATE TABLE IF NOT EXISTS workflow_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    version INT NOT NULL,
    trigger_config JSONB NOT NULL,
    nodes JSONB NOT NULL,
    edges JSONB NOT NULL,
    variables JSONB NOT NULL,
    settings JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(workflow_id, version)
);

CREATE INDEX idx_workflow_versions_workflow ON workflow_versions(workflow_id);

-- Workflow executions table
CREATE TABLE IF NOT EXISTS workflow_executions (
    id UUID PRIMARY KEY,
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    workflow_name VARCHAR(255) NOT NULL,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    version INT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    trigger_type VARCHAR(50) NOT NULL,
    trigger_data JSONB,
    input JSONB,
    output JSONB,
    variables JSONB,
    node_states JSONB NOT NULL DEFAULT '[]',
    error JSONB,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_workflow_executions_tenant ON workflow_executions(tenant_id);
CREATE INDEX idx_workflow_executions_workflow ON workflow_executions(workflow_id);
CREATE INDEX idx_workflow_executions_status ON workflow_executions(tenant_id, status);
CREATE INDEX idx_workflow_executions_started ON workflow_executions(tenant_id, started_at DESC);

-- Workflow templates table
CREATE TABLE IF NOT EXISTS workflow_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100) NOT NULL,
    tags JSONB NOT NULL DEFAULT '[]',
    thumbnail TEXT,
    workflow JSONB NOT NULL,
    usage_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workflow_templates_category ON workflow_templates(category);

-- Insert default templates
INSERT INTO workflow_templates (id, name, description, category, tags, workflow, usage_count) VALUES
(
    gen_random_uuid(),
    'Simple Webhook Relay',
    'Receive a webhook and forward it to another endpoint',
    'basic',
    '["webhook", "relay", "starter"]',
    '{
        "name": "Simple Webhook Relay",
        "description": "Receive and forward webhooks",
        "version": 1,
        "status": "draft",
        "trigger": {"type": "webhook"},
        "nodes": [
            {"id": "start", "type": "start", "name": "Start", "position": {"x": 100, "y": 200}, "outputs": [{"id": "out", "name": "Output"}]},
            {"id": "transform", "type": "transform", "name": "Transform", "position": {"x": 300, "y": 200}, "inputs": [{"id": "in", "name": "Input"}], "outputs": [{"id": "out", "name": "Output"}]},
            {"id": "webhook", "type": "webhook", "name": "Send Webhook", "position": {"x": 500, "y": 200}, "inputs": [{"id": "in", "name": "Input"}], "outputs": [{"id": "out", "name": "Output"}]},
            {"id": "end", "type": "end", "name": "End", "position": {"x": 700, "y": 200}, "inputs": [{"id": "in", "name": "Input"}]}
        ],
        "edges": [
            {"id": "e1", "source": "start", "source_port": "out", "target": "transform", "target_port": "in"},
            {"id": "e2", "source": "transform", "source_port": "out", "target": "webhook", "target_port": "in"},
            {"id": "e3", "source": "webhook", "source_port": "out", "target": "end", "target_port": "in"}
        ],
        "variables": [],
        "settings": {"timeout_seconds": 300, "max_retries": 3, "retry_delay_ms": 1000, "concurrency_limit": 10, "error_handling": "fail_fast", "log_level": "info"},
        "canvas": {"zoom": 1, "pan_x": 0, "pan_y": 0, "grid_size": 20, "snap_to_grid": true}
    }',
    0
),
(
    gen_random_uuid(),
    'Conditional Routing',
    'Route webhooks based on payload content',
    'routing',
    '["webhook", "routing", "condition"]',
    '{
        "name": "Conditional Routing",
        "description": "Route based on conditions",
        "version": 1,
        "status": "draft",
        "trigger": {"type": "webhook"},
        "nodes": [
            {"id": "start", "type": "start", "name": "Start", "position": {"x": 100, "y": 200}, "outputs": [{"id": "out", "name": "Output"}]},
            {"id": "condition", "type": "condition", "name": "Check Type", "position": {"x": 300, "y": 200}, "inputs": [{"id": "in", "name": "Input"}], "outputs": [{"id": "true", "name": "True"}, {"id": "false", "name": "False"}]},
            {"id": "webhook1", "type": "webhook", "name": "Route A", "position": {"x": 500, "y": 100}, "inputs": [{"id": "in", "name": "Input"}], "outputs": [{"id": "out", "name": "Output"}]},
            {"id": "webhook2", "type": "webhook", "name": "Route B", "position": {"x": 500, "y": 300}, "inputs": [{"id": "in", "name": "Input"}], "outputs": [{"id": "out", "name": "Output"}]},
            {"id": "end", "type": "end", "name": "End", "position": {"x": 700, "y": 200}, "inputs": [{"id": "in", "name": "Input"}]}
        ],
        "edges": [
            {"id": "e1", "source": "start", "source_port": "out", "target": "condition", "target_port": "in"},
            {"id": "e2", "source": "condition", "source_port": "true", "target": "webhook1", "target_port": "in"},
            {"id": "e3", "source": "condition", "source_port": "false", "target": "webhook2", "target_port": "in"},
            {"id": "e4", "source": "webhook1", "source_port": "out", "target": "end", "target_port": "in"},
            {"id": "e5", "source": "webhook2", "source_port": "out", "target": "end", "target_port": "in"}
        ],
        "variables": [],
        "settings": {"timeout_seconds": 300, "max_retries": 3, "retry_delay_ms": 1000, "concurrency_limit": 10, "error_handling": "fail_fast", "log_level": "info"},
        "canvas": {"zoom": 1, "pan_x": 0, "pan_y": 0, "grid_size": 20, "snap_to_grid": true}
    }',
    0
),
(
    gen_random_uuid(),
    'Fan-out Pattern',
    'Send webhook to multiple endpoints in parallel',
    'patterns',
    '["webhook", "fanout", "parallel"]',
    '{
        "name": "Fan-out Pattern",
        "description": "Parallel delivery to multiple endpoints",
        "version": 1,
        "status": "draft",
        "trigger": {"type": "webhook"},
        "nodes": [
            {"id": "start", "type": "start", "name": "Start", "position": {"x": 100, "y": 200}, "outputs": [{"id": "out", "name": "Output"}]},
            {"id": "parallel", "type": "parallel", "name": "Fan Out", "position": {"x": 300, "y": 200}, "inputs": [{"id": "in", "name": "Input"}], "outputs": [{"id": "out", "name": "Output"}]},
            {"id": "webhook1", "type": "webhook", "name": "Endpoint 1", "position": {"x": 500, "y": 100}, "inputs": [{"id": "in", "name": "Input"}], "outputs": [{"id": "out", "name": "Output"}]},
            {"id": "webhook2", "type": "webhook", "name": "Endpoint 2", "position": {"x": 500, "y": 200}, "inputs": [{"id": "in", "name": "Input"}], "outputs": [{"id": "out", "name": "Output"}]},
            {"id": "webhook3", "type": "webhook", "name": "Endpoint 3", "position": {"x": 500, "y": 300}, "inputs": [{"id": "in", "name": "Input"}], "outputs": [{"id": "out", "name": "Output"}]},
            {"id": "merge", "type": "merge", "name": "Merge", "position": {"x": 700, "y": 200}, "inputs": [{"id": "in", "name": "Input"}], "outputs": [{"id": "out", "name": "Output"}]},
            {"id": "end", "type": "end", "name": "End", "position": {"x": 900, "y": 200}, "inputs": [{"id": "in", "name": "Input"}]}
        ],
        "edges": [
            {"id": "e1", "source": "start", "source_port": "out", "target": "parallel", "target_port": "in"},
            {"id": "e2", "source": "parallel", "source_port": "out", "target": "webhook1", "target_port": "in"},
            {"id": "e3", "source": "parallel", "source_port": "out", "target": "webhook2", "target_port": "in"},
            {"id": "e4", "source": "parallel", "source_port": "out", "target": "webhook3", "target_port": "in"},
            {"id": "e5", "source": "webhook1", "source_port": "out", "target": "merge", "target_port": "in"},
            {"id": "e6", "source": "webhook2", "source_port": "out", "target": "merge", "target_port": "in"},
            {"id": "e7", "source": "webhook3", "source_port": "out", "target": "merge", "target_port": "in"},
            {"id": "e8", "source": "merge", "source_port": "out", "target": "end", "target_port": "in"}
        ],
        "variables": [],
        "settings": {"timeout_seconds": 300, "max_retries": 3, "retry_delay_ms": 1000, "concurrency_limit": 10, "error_handling": "continue", "log_level": "info"},
        "canvas": {"zoom": 1, "pan_x": 0, "pan_y": 0, "grid_size": 20, "snap_to_grid": true}
    }',
    0
);
