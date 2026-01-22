package multiregion

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

// Repository defines the multi-region data access interface
type Repository interface {
	// Region management
	CreateRegion(ctx context.Context, region *Region) error
	GetRegion(ctx context.Context, id string) (*Region, error)
	GetRegionByCode(ctx context.Context, code string) (*Region, error)
	ListRegions(ctx context.Context) ([]*Region, error)
	ListActiveRegions(ctx context.Context) ([]*Region, error)
	UpdateRegion(ctx context.Context, region *Region) error
	DeleteRegion(ctx context.Context, id string) error

	// Health tracking
	RecordRegionHealth(ctx context.Context, health *RegionHealth) error
	GetRegionHealth(ctx context.Context, regionID string) (*RegionHealth, error)
	GetAllRegionHealth(ctx context.Context) ([]*RegionHealth, error)

	// Failover events
	CreateFailoverEvent(ctx context.Context, event *FailoverEvent) error
	UpdateFailoverEvent(ctx context.Context, event *FailoverEvent) error
	GetFailoverEvent(ctx context.Context, id string) (*FailoverEvent, error)
	ListFailoverEvents(ctx context.Context, limit int) ([]*FailoverEvent, error)

	// Replication config
	CreateReplicationConfig(ctx context.Context, config *ReplicationConfig) error
	GetReplicationConfig(ctx context.Context, id string) (*ReplicationConfig, error)
	GetReplicationConfigByRegions(ctx context.Context, source, target string) (*ReplicationConfig, error)
	ListReplicationConfigs(ctx context.Context) ([]*ReplicationConfig, error)
	UpdateReplicationConfig(ctx context.Context, config *ReplicationConfig) error

	// Routing policies
	CreateRoutingPolicy(ctx context.Context, policy *RoutingPolicy) error
	GetRoutingPolicy(ctx context.Context, tenantID string) (*RoutingPolicy, error)
	UpdateRoutingPolicy(ctx context.Context, policy *RoutingPolicy) error
	DeleteRoutingPolicy(ctx context.Context, tenantID string) error
}

// PostgresRepository implements Repository for PostgreSQL
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// CreateRegion creates a new region
func (r *PostgresRepository) CreateRegion(ctx context.Context, region *Region) error {
	metadata, _ := json.Marshal(region.Metadata)
	
	query := `
		INSERT INTO regions (id, name, code, endpoint, is_active, is_primary, priority, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	
	_, err := r.db.ExecContext(ctx, query,
		region.ID, region.Name, region.Code, region.Endpoint,
		region.IsActive, region.IsPrimary, region.Priority, metadata,
		region.CreatedAt, region.UpdatedAt)
	
	return err
}

// GetRegion retrieves a region by ID
func (r *PostgresRepository) GetRegion(ctx context.Context, id string) (*Region, error) {
	query := `
		SELECT id, name, code, endpoint, is_active, is_primary, priority, metadata, created_at, updated_at
		FROM regions WHERE id = $1`
	
	var region Region
	var metadata []byte
	
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&region.ID, &region.Name, &region.Code, &region.Endpoint,
		&region.IsActive, &region.IsPrimary, &region.Priority, &metadata,
		&region.CreatedAt, &region.UpdatedAt)
	
	if err != nil {
		return nil, err
	}
	
	json.Unmarshal(metadata, &region.Metadata)
	return &region, nil
}

// GetRegionByCode retrieves a region by code
func (r *PostgresRepository) GetRegionByCode(ctx context.Context, code string) (*Region, error) {
	query := `
		SELECT id, name, code, endpoint, is_active, is_primary, priority, metadata, created_at, updated_at
		FROM regions WHERE code = $1`
	
	var region Region
	var metadata []byte
	
	err := r.db.QueryRowContext(ctx, query, code).Scan(
		&region.ID, &region.Name, &region.Code, &region.Endpoint,
		&region.IsActive, &region.IsPrimary, &region.Priority, &metadata,
		&region.CreatedAt, &region.UpdatedAt)
	
	if err != nil {
		return nil, err
	}
	
	json.Unmarshal(metadata, &region.Metadata)
	return &region, nil
}

// ListRegions lists all regions
func (r *PostgresRepository) ListRegions(ctx context.Context) ([]*Region, error) {
	query := `
		SELECT id, name, code, endpoint, is_active, is_primary, priority, metadata, created_at, updated_at
		FROM regions ORDER BY priority ASC`
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var regions []*Region
	for rows.Next() {
		var region Region
		var metadata []byte
		
		if err := rows.Scan(
			&region.ID, &region.Name, &region.Code, &region.Endpoint,
			&region.IsActive, &region.IsPrimary, &region.Priority, &metadata,
			&region.CreatedAt, &region.UpdatedAt); err != nil {
			return nil, err
		}
		
		json.Unmarshal(metadata, &region.Metadata)
		regions = append(regions, &region)
	}
	
	return regions, rows.Err()
}

// ListActiveRegions lists only active regions
func (r *PostgresRepository) ListActiveRegions(ctx context.Context) ([]*Region, error) {
	query := `
		SELECT id, name, code, endpoint, is_active, is_primary, priority, metadata, created_at, updated_at
		FROM regions WHERE is_active = true ORDER BY priority ASC`
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var regions []*Region
	for rows.Next() {
		var region Region
		var metadata []byte
		
		if err := rows.Scan(
			&region.ID, &region.Name, &region.Code, &region.Endpoint,
			&region.IsActive, &region.IsPrimary, &region.Priority, &metadata,
			&region.CreatedAt, &region.UpdatedAt); err != nil {
			return nil, err
		}
		
		json.Unmarshal(metadata, &region.Metadata)
		regions = append(regions, &region)
	}
	
	return regions, rows.Err()
}

// UpdateRegion updates a region
func (r *PostgresRepository) UpdateRegion(ctx context.Context, region *Region) error {
	metadata, _ := json.Marshal(region.Metadata)
	
	query := `
		UPDATE regions SET name = $2, code = $3, endpoint = $4, is_active = $5, 
		is_primary = $6, priority = $7, metadata = $8, updated_at = $9
		WHERE id = $1`
	
	_, err := r.db.ExecContext(ctx, query,
		region.ID, region.Name, region.Code, region.Endpoint,
		region.IsActive, region.IsPrimary, region.Priority, metadata,
		time.Now())
	
	return err
}

// DeleteRegion deletes a region
func (r *PostgresRepository) DeleteRegion(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM regions WHERE id = $1", id)
	return err
}

// RecordRegionHealth records region health status
func (r *PostgresRepository) RecordRegionHealth(ctx context.Context, health *RegionHealth) error {
	metrics, _ := json.Marshal(health.Metrics)
	
	query := `
		INSERT INTO region_health (region_id, status, last_check, latency_ns, success_rate, 
		active_connections, queue_depth, error_rate, metrics)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (region_id) DO UPDATE SET
		status = EXCLUDED.status, last_check = EXCLUDED.last_check, latency_ns = EXCLUDED.latency_ns,
		success_rate = EXCLUDED.success_rate, active_connections = EXCLUDED.active_connections,
		queue_depth = EXCLUDED.queue_depth, error_rate = EXCLUDED.error_rate, metrics = EXCLUDED.metrics`
	
	_, err := r.db.ExecContext(ctx, query,
		health.RegionID, health.Status, health.LastCheck, health.Latency.Nanoseconds(),
		health.SuccessRate, health.ActiveConnections, health.QueueDepth, health.ErrorRate, metrics)
	
	return err
}

// GetRegionHealth retrieves current health for a region
func (r *PostgresRepository) GetRegionHealth(ctx context.Context, regionID string) (*RegionHealth, error) {
	query := `
		SELECT region_id, status, last_check, latency_ns, success_rate, 
		active_connections, queue_depth, error_rate, metrics
		FROM region_health WHERE region_id = $1`
	
	var health RegionHealth
	var latencyNs int64
	var metrics []byte
	
	err := r.db.QueryRowContext(ctx, query, regionID).Scan(
		&health.RegionID, &health.Status, &health.LastCheck, &latencyNs,
		&health.SuccessRate, &health.ActiveConnections, &health.QueueDepth,
		&health.ErrorRate, &metrics)
	
	if err != nil {
		return nil, err
	}
	
	health.Latency = time.Duration(latencyNs)
	json.Unmarshal(metrics, &health.Metrics)
	return &health, nil
}

// GetAllRegionHealth retrieves health for all regions
func (r *PostgresRepository) GetAllRegionHealth(ctx context.Context) ([]*RegionHealth, error) {
	query := `
		SELECT region_id, status, last_check, latency_ns, success_rate, 
		active_connections, queue_depth, error_rate, metrics
		FROM region_health`
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var healths []*RegionHealth
	for rows.Next() {
		var health RegionHealth
		var latencyNs int64
		var metrics []byte
		
		if err := rows.Scan(
			&health.RegionID, &health.Status, &health.LastCheck, &latencyNs,
			&health.SuccessRate, &health.ActiveConnections, &health.QueueDepth,
			&health.ErrorRate, &metrics); err != nil {
			return nil, err
		}
		
		health.Latency = time.Duration(latencyNs)
		json.Unmarshal(metrics, &health.Metrics)
		healths = append(healths, &health)
	}
	
	return healths, rows.Err()
}

// CreateFailoverEvent creates a failover event
func (r *PostgresRepository) CreateFailoverEvent(ctx context.Context, event *FailoverEvent) error {
	query := `
		INSERT INTO failover_events (id, from_region, to_region, reason, trigger_type, status, 
		started_at, affected_ops, details)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	
	_, err := r.db.ExecContext(ctx, query,
		event.ID, event.FromRegion, event.ToRegion, event.Reason, event.TriggerType,
		event.Status, event.StartedAt, event.AffectedOps, event.Details)
	
	return err
}

// UpdateFailoverEvent updates a failover event
func (r *PostgresRepository) UpdateFailoverEvent(ctx context.Context, event *FailoverEvent) error {
	query := `
		UPDATE failover_events SET status = $2, completed_at = $3, affected_ops = $4, details = $5
		WHERE id = $1`
	
	_, err := r.db.ExecContext(ctx, query,
		event.ID, event.Status, event.CompletedAt, event.AffectedOps, event.Details)
	
	return err
}

// GetFailoverEvent retrieves a failover event
func (r *PostgresRepository) GetFailoverEvent(ctx context.Context, id string) (*FailoverEvent, error) {
	query := `
		SELECT id, from_region, to_region, reason, trigger_type, status, started_at, 
		completed_at, affected_ops, details
		FROM failover_events WHERE id = $1`
	
	var event FailoverEvent
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&event.ID, &event.FromRegion, &event.ToRegion, &event.Reason, &event.TriggerType,
		&event.Status, &event.StartedAt, &event.CompletedAt, &event.AffectedOps, &event.Details)
	
	if err != nil {
		return nil, err
	}
	
	if event.CompletedAt != nil {
		event.Duration = event.CompletedAt.Sub(event.StartedAt)
	}
	
	return &event, nil
}

// ListFailoverEvents lists recent failover events
func (r *PostgresRepository) ListFailoverEvents(ctx context.Context, limit int) ([]*FailoverEvent, error) {
	query := `
		SELECT id, from_region, to_region, reason, trigger_type, status, started_at, 
		completed_at, affected_ops, details
		FROM failover_events ORDER BY started_at DESC LIMIT $1`
	
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var events []*FailoverEvent
	for rows.Next() {
		var event FailoverEvent
		if err := rows.Scan(
			&event.ID, &event.FromRegion, &event.ToRegion, &event.Reason, &event.TriggerType,
			&event.Status, &event.StartedAt, &event.CompletedAt, &event.AffectedOps, &event.Details); err != nil {
			return nil, err
		}
		
		if event.CompletedAt != nil {
			event.Duration = event.CompletedAt.Sub(event.StartedAt)
		}
		events = append(events, &event)
	}
	
	return events, rows.Err()
}

// CreateReplicationConfig creates a replication config
func (r *PostgresRepository) CreateReplicationConfig(ctx context.Context, config *ReplicationConfig) error {
	tables, _ := json.Marshal(config.Tables)
	
	query := `
		INSERT INTO replication_configs (id, source_region, target_region, mode, enabled, 
		lag_threshold_ms, retention_days, tables, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	
	_, err := r.db.ExecContext(ctx, query,
		config.ID, config.SourceRegion, config.TargetRegion, config.Mode, config.Enabled,
		config.LagThresholdMs, config.RetentionDays, tables, config.CreatedAt)
	
	return err
}

// GetReplicationConfig retrieves a replication config
func (r *PostgresRepository) GetReplicationConfig(ctx context.Context, id string) (*ReplicationConfig, error) {
	query := `
		SELECT id, source_region, target_region, mode, enabled, lag_threshold_ms, 
		retention_days, tables, last_sync_at, created_at
		FROM replication_configs WHERE id = $1`
	
	var config ReplicationConfig
	var tables []byte
	
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&config.ID, &config.SourceRegion, &config.TargetRegion, &config.Mode, &config.Enabled,
		&config.LagThresholdMs, &config.RetentionDays, &tables, &config.LastSyncAt, &config.CreatedAt)
	
	if err != nil {
		return nil, err
	}
	
	json.Unmarshal(tables, &config.Tables)
	return &config, nil
}

// GetReplicationConfigByRegions retrieves config by source and target regions
func (r *PostgresRepository) GetReplicationConfigByRegions(ctx context.Context, source, target string) (*ReplicationConfig, error) {
	query := `
		SELECT id, source_region, target_region, mode, enabled, lag_threshold_ms, 
		retention_days, tables, last_sync_at, created_at
		FROM replication_configs WHERE source_region = $1 AND target_region = $2`
	
	var config ReplicationConfig
	var tables []byte
	
	err := r.db.QueryRowContext(ctx, query, source, target).Scan(
		&config.ID, &config.SourceRegion, &config.TargetRegion, &config.Mode, &config.Enabled,
		&config.LagThresholdMs, &config.RetentionDays, &tables, &config.LastSyncAt, &config.CreatedAt)
	
	if err != nil {
		return nil, err
	}
	
	json.Unmarshal(tables, &config.Tables)
	return &config, nil
}

// ListReplicationConfigs lists all replication configs
func (r *PostgresRepository) ListReplicationConfigs(ctx context.Context) ([]*ReplicationConfig, error) {
	query := `
		SELECT id, source_region, target_region, mode, enabled, lag_threshold_ms, 
		retention_days, tables, last_sync_at, created_at
		FROM replication_configs ORDER BY created_at DESC`
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var configs []*ReplicationConfig
	for rows.Next() {
		var config ReplicationConfig
		var tables []byte
		
		if err := rows.Scan(
			&config.ID, &config.SourceRegion, &config.TargetRegion, &config.Mode, &config.Enabled,
			&config.LagThresholdMs, &config.RetentionDays, &tables, &config.LastSyncAt, &config.CreatedAt); err != nil {
			return nil, err
		}
		
		json.Unmarshal(tables, &config.Tables)
		configs = append(configs, &config)
	}
	
	return configs, rows.Err()
}

// UpdateReplicationConfig updates a replication config
func (r *PostgresRepository) UpdateReplicationConfig(ctx context.Context, config *ReplicationConfig) error {
	tables, _ := json.Marshal(config.Tables)
	
	query := `
		UPDATE replication_configs SET mode = $2, enabled = $3, lag_threshold_ms = $4, 
		retention_days = $5, tables = $6, last_sync_at = $7
		WHERE id = $1`
	
	_, err := r.db.ExecContext(ctx, query,
		config.ID, config.Mode, config.Enabled, config.LagThresholdMs,
		config.RetentionDays, tables, config.LastSyncAt)
	
	return err
}

// CreateRoutingPolicy creates a routing policy
func (r *PostgresRepository) CreateRoutingPolicy(ctx context.Context, policy *RoutingPolicy) error {
	fallback, _ := json.Marshal(policy.FallbackRegions)
	geoRules, _ := json.Marshal(policy.GeoRules)
	weights, _ := json.Marshal(policy.Weights)
	
	query := `
		INSERT INTO routing_policies (id, tenant_id, policy_type, primary_region, 
		fallback_regions, geo_rules, weights, enabled, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	
	_, err := r.db.ExecContext(ctx, query,
		policy.ID, policy.TenantID, policy.PolicyType, policy.PrimaryRegion,
		fallback, geoRules, weights, policy.Enabled, policy.CreatedAt)
	
	return err
}

// GetRoutingPolicy retrieves a routing policy by tenant
func (r *PostgresRepository) GetRoutingPolicy(ctx context.Context, tenantID string) (*RoutingPolicy, error) {
	query := `
		SELECT id, tenant_id, policy_type, primary_region, fallback_regions, 
		geo_rules, weights, enabled, created_at
		FROM routing_policies WHERE tenant_id = $1`
	
	var policy RoutingPolicy
	var fallback, geoRules, weights []byte
	
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(
		&policy.ID, &policy.TenantID, &policy.PolicyType, &policy.PrimaryRegion,
		&fallback, &geoRules, &weights, &policy.Enabled, &policy.CreatedAt)
	
	if err != nil {
		return nil, err
	}
	
	json.Unmarshal(fallback, &policy.FallbackRegions)
	json.Unmarshal(geoRules, &policy.GeoRules)
	json.Unmarshal(weights, &policy.Weights)
	return &policy, nil
}

// UpdateRoutingPolicy updates a routing policy
func (r *PostgresRepository) UpdateRoutingPolicy(ctx context.Context, policy *RoutingPolicy) error {
	fallback, _ := json.Marshal(policy.FallbackRegions)
	geoRules, _ := json.Marshal(policy.GeoRules)
	weights, _ := json.Marshal(policy.Weights)
	
	query := `
		UPDATE routing_policies SET policy_type = $2, primary_region = $3, 
		fallback_regions = $4, geo_rules = $5, weights = $6, enabled = $7
		WHERE tenant_id = $1`
	
	_, err := r.db.ExecContext(ctx, query,
		policy.TenantID, policy.PolicyType, policy.PrimaryRegion,
		fallback, geoRules, weights, policy.Enabled)
	
	return err
}

// DeleteRoutingPolicy deletes a routing policy
func (r *PostgresRepository) DeleteRoutingPolicy(ctx context.Context, tenantID string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM routing_policies WHERE tenant_id = $1", tenantID)
	return err
}
