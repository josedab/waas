-- Drop analytics tables in reverse order

DROP TABLE IF EXISTS alert_history;
DROP TABLE IF EXISTS alert_configs;
DROP FUNCTION IF EXISTS cleanup_realtime_metrics();
DROP TABLE IF EXISTS realtime_metrics;
DROP TABLE IF EXISTS daily_metrics;
DROP TABLE IF EXISTS hourly_metrics;
DROP TABLE IF EXISTS delivery_metrics;