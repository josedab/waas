-- Rollback Event Sourcing Tables

DROP INDEX IF EXISTS idx_replay_jobs_status;
DROP INDEX IF EXISTS idx_projections_tenant;
DROP INDEX IF EXISTS idx_stream_snapshots_lookup;
DROP INDEX IF EXISTS idx_consumer_checkpoints_lookup;
DROP INDEX IF EXISTS idx_event_store_correlation;
DROP INDEX IF EXISTS idx_event_store_created;
DROP INDEX IF EXISTS idx_event_store_tenant_type;
DROP INDEX IF EXISTS idx_event_store_tenant_stream;

DROP TABLE IF EXISTS replay_jobs;
DROP TABLE IF EXISTS projection_state;
DROP TABLE IF EXISTS projections;
DROP TABLE IF EXISTS stream_snapshots;
DROP TABLE IF EXISTS consumer_checkpoints;
DROP TABLE IF EXISTS event_store;
