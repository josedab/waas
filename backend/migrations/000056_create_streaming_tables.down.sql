-- Drop streaming tables

DROP TRIGGER IF EXISTS trigger_stream_producers_updated ON stream_producers;
DROP TRIGGER IF EXISTS trigger_stream_consumers_updated ON stream_consumers;
DROP TRIGGER IF EXISTS trigger_stream_configs_updated ON stream_configs;
DROP FUNCTION IF EXISTS update_stream_config_timestamp();

DROP TABLE IF EXISTS stream_schemas;
DROP TABLE IF EXISTS stream_messages;
DROP TABLE IF EXISTS stream_producers;
DROP TABLE IF EXISTS stream_consumers;
DROP TABLE IF EXISTS stream_topics;
DROP TABLE IF EXISTS stream_configs;
