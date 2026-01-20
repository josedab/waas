-- Rollback Feature 9: Federated Multi-Region Mesh

DROP TABLE IF EXISTS regional_config_sync;
DROP TABLE IF EXISTS failover_events;
DROP TABLE IF EXISTS data_residency_audit;
DROP TABLE IF EXISTS region_health_metrics;
DROP TABLE IF EXISTS regional_routing_decisions;
DROP TABLE IF EXISTS replication_streams;
DROP TABLE IF EXISTS geo_routing_rules;
DROP TABLE IF EXISTS tenant_regions;
DROP TABLE IF EXISTS region_cluster_members;
DROP TABLE IF EXISTS region_clusters;
DROP TABLE IF EXISTS regions;
