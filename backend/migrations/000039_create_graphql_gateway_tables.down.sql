-- Rollback: GraphQL Subscriptions Gateway tables

DROP INDEX IF EXISTS idx_graphql_type_mappings_schema;
DROP INDEX IF EXISTS idx_graphql_federation_sources_schema;
DROP INDEX IF EXISTS idx_graphql_subscription_events_delivered;
DROP INDEX IF EXISTS idx_graphql_subscription_events_sub;
DROP INDEX IF EXISTS idx_graphql_subscriptions_status;
DROP INDEX IF EXISTS idx_graphql_subscriptions_schema;
DROP INDEX IF EXISTS idx_graphql_subscriptions_tenant;
DROP INDEX IF EXISTS idx_graphql_schemas_status;
DROP INDEX IF EXISTS idx_graphql_schemas_tenant;

DROP TABLE IF EXISTS graphql_type_mappings;
DROP TABLE IF EXISTS graphql_federation_sources;
DROP TABLE IF EXISTS graphql_subscription_events;
DROP TABLE IF EXISTS graphql_subscriptions;
DROP TABLE IF EXISTS graphql_schemas;
