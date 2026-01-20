DROP INDEX IF EXISTS idx_chaos_events_timestamp;
DROP INDEX IF EXISTS idx_chaos_events_delivery;
DROP INDEX IF EXISTS idx_chaos_events_exp;
DROP TABLE IF EXISTS chaos_events;

DROP INDEX IF EXISTS idx_chaos_exp_created;
DROP INDEX IF EXISTS idx_chaos_exp_status;
DROP INDEX IF EXISTS idx_chaos_exp_tenant;
DROP TABLE IF EXISTS chaos_experiments;
