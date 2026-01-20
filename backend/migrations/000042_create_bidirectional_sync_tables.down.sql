-- Rollback: Bi-Directional Webhook Sync tables

DROP INDEX IF EXISTS idx_sync_conflict_history_record;
DROP INDEX IF EXISTS idx_sync_acknowledgments_correlation;
DROP INDEX IF EXISTS idx_sync_acknowledgments_config;
DROP INDEX IF EXISTS idx_sync_state_records_status;
DROP INDEX IF EXISTS idx_sync_state_records_resource;
DROP INDEX IF EXISTS idx_sync_state_records_config;
DROP INDEX IF EXISTS idx_sync_transactions_timeout;
DROP INDEX IF EXISTS idx_sync_transactions_state;
DROP INDEX IF EXISTS idx_sync_transactions_correlation;
DROP INDEX IF EXISTS idx_sync_transactions_config;
DROP INDEX IF EXISTS idx_sync_configs_enabled;
DROP INDEX IF EXISTS idx_sync_configs_tenant;

DROP TABLE IF EXISTS sync_conflict_history;
DROP TABLE IF EXISTS sync_acknowledgments;
DROP TABLE IF EXISTS sync_state_records;
DROP TABLE IF EXISTS sync_transactions;
DROP TABLE IF EXISTS webhook_sync_configs;
