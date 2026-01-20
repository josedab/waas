-- Rollback Multi-Cloud Tables

DROP INDEX IF EXISTS idx_cloud_dlq_status;
DROP INDEX IF EXISTS idx_cloud_deliveries_tenant;
DROP INDEX IF EXISTS idx_cloud_deliveries_connector;
DROP INDEX IF EXISTS idx_connector_routes_tenant;
DROP INDEX IF EXISTS idx_connector_routes_connector;
DROP INDEX IF EXISTS idx_cloud_connectors_provider;
DROP INDEX IF EXISTS idx_cloud_connectors_tenant;

DROP TABLE IF EXISTS cloud_dlq;
DROP TABLE IF EXISTS connector_credentials;
DROP TABLE IF EXISTS cloud_deliveries;
DROP TABLE IF EXISTS connector_routes;
DROP TABLE IF EXISTS cloud_connectors;
