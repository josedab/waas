-- Drop federation cluster tables

DROP TRIGGER IF EXISTS trigger_federation_replication_configs_updated ON federation_replication_configs;
DROP TRIGGER IF EXISTS trigger_federation_routes_updated ON federation_routes;
DROP TRIGGER IF EXISTS trigger_federation_cluster_credentials_updated ON federation_cluster_credentials;
DROP TRIGGER IF EXISTS trigger_federation_clusters_updated ON federation_clusters;

DROP TABLE IF EXISTS federation_routing_metrics;
DROP TABLE IF EXISTS federation_replication_configs;
DROP TABLE IF EXISTS federation_failover_events;
DROP TABLE IF EXISTS federation_routes;
DROP TABLE IF EXISTS federation_cluster_credentials;
DROP TABLE IF EXISTS federation_clusters;
