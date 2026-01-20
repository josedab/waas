DROP INDEX IF EXISTS idx_otel_configs_tenant;
DROP TABLE IF EXISTS otel_export_configs;

DROP INDEX IF EXISTS idx_spans_status;
DROP INDEX IF EXISTS idx_spans_endpoint;
DROP INDEX IF EXISTS idx_spans_webhook;
DROP INDEX IF EXISTS idx_spans_service;
DROP INDEX IF EXISTS idx_spans_parent;
DROP INDEX IF EXISTS idx_spans_trace_id;
DROP INDEX IF EXISTS idx_spans_tenant_time;
DROP INDEX IF EXISTS idx_spans_tenant_trace;
DROP TABLE IF EXISTS webhook_spans;
