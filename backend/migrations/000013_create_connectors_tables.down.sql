DROP INDEX IF EXISTS idx_connector_executions_created_at;
DROP INDEX IF EXISTS idx_connector_executions_installed_id;
DROP INDEX IF EXISTS idx_installed_connectors_connector_id;
DROP INDEX IF EXISTS idx_installed_connectors_tenant_id;
DROP TABLE IF EXISTS connector_executions;
DROP TABLE IF EXISTS installed_connectors;
