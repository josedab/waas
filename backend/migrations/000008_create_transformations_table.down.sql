-- Drop indexes
DROP INDEX IF EXISTS idx_transformation_logs_created_at;
DROP INDEX IF EXISTS idx_transformation_logs_delivery_id;
DROP INDEX IF EXISTS idx_transformation_logs_transformation_id;
DROP INDEX IF EXISTS idx_endpoint_transformations_transformation_id;
DROP INDEX IF EXISTS idx_endpoint_transformations_endpoint_id;
DROP INDEX IF EXISTS idx_transformations_enabled;
DROP INDEX IF EXISTS idx_transformations_tenant_id;

-- Drop tables
DROP TABLE IF EXISTS transformation_logs;
DROP TABLE IF EXISTS endpoint_transformations;
DROP TABLE IF EXISTS transformations;
