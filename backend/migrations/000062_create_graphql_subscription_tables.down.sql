-- Drop GraphQL subscription tables

DROP TRIGGER IF EXISTS trigger_graphql_subscriptions_updated ON graphql_subscriptions;
DROP TRIGGER IF EXISTS trigger_graphql_subscription_schemas_updated ON graphql_subscription_schemas;

DROP TABLE IF EXISTS graphql_subscription_stats;
DROP TABLE IF EXISTS graphql_events;
DROP TABLE IF EXISTS graphql_clients;
DROP TABLE IF EXISTS graphql_subscriptions;
DROP TABLE IF EXISTS graphql_subscription_schemas;
