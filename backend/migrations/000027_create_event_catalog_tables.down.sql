-- Rollback Event Catalog Tables

DROP INDEX IF EXISTS idx_event_categories_tenant;
DROP INDEX IF EXISTS idx_event_subscriptions_type;
DROP INDEX IF EXISTS idx_event_subscriptions_endpoint;
DROP INDEX IF EXISTS idx_event_type_versions_event;
DROP INDEX IF EXISTS idx_event_types_tags;
DROP INDEX IF EXISTS idx_event_types_status;
DROP INDEX IF EXISTS idx_event_types_category;
DROP INDEX IF EXISTS idx_event_types_slug;
DROP INDEX IF EXISTS idx_event_types_tenant;

DROP TABLE IF EXISTS sdk_configs;
DROP TABLE IF EXISTS event_documentation;
DROP TABLE IF EXISTS event_subscriptions;
DROP TABLE IF EXISTS event_categories;
DROP TABLE IF EXISTS event_type_versions;
DROP TABLE IF EXISTS event_types;
