-- Drop versioning tables

ALTER TABLE webhook_versions DROP CONSTRAINT IF EXISTS fk_versions_schema;
ALTER TABLE webhook_versions DROP CONSTRAINT IF EXISTS fk_versions_replacement;

DROP TABLE IF EXISTS version_usage;
DROP TABLE IF EXISTS version_policies;
DROP TABLE IF EXISTS deprecation_notices;
DROP TABLE IF EXISTS version_migrations;
DROP TABLE IF EXISTS version_subscriptions;
DROP TABLE IF EXISTS webhook_versions;
DROP TABLE IF EXISTS version_schemas;
