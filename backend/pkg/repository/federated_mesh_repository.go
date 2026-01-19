package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"
	"webhook-platform/pkg/models"
)

// FederatedMeshRepository handles persistence for federated mesh
type FederatedMeshRepository interface {
	// Regions
	GetAllRegions(ctx context.Context) ([]*models.Region, error)
	GetRegion(ctx context.Context, id uuid.UUID) (*models.Region, error)
	GetRegionByCode(ctx context.Context, code string) (*models.Region, error)
	GetActiveRegions(ctx context.Context) ([]*models.Region, error)
	UpdateRegionHealth(ctx context.Context, id uuid.UUID, status string) error
	UpdateRegionLoad(ctx context.Context, id uuid.UUID, load int) error

	// Region Clusters
	CreateCluster(ctx context.Context, cluster *models.RegionCluster) error
	GetCluster(ctx context.Context, id uuid.UUID) (*models.RegionCluster, error)
	GetClusters(ctx context.Context) ([]*models.RegionCluster, error)
	AddClusterMember(ctx context.Context, member *models.RegionClusterMember) error
	GetClusterMembers(ctx context.Context, clusterID uuid.UUID) ([]*models.RegionClusterMember, error)

	// Tenant Regions
	CreateTenantRegion(ctx context.Context, tr *models.MeshTenantRegion) error
	GetTenantRegion(ctx context.Context, tenantID uuid.UUID) (*models.MeshTenantRegion, error)
	UpdateTenantRegion(ctx context.Context, tr *models.MeshTenantRegion) error

	// Geo Routing Rules
	CreateRoutingRule(ctx context.Context, rule *models.GeoRoutingRule) error
	GetRoutingRule(ctx context.Context, id uuid.UUID) (*models.GeoRoutingRule, error)
	GetRoutingRulesByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.GeoRoutingRule, error)
	GetEnabledRoutingRules(ctx context.Context, tenantID uuid.UUID) ([]*models.GeoRoutingRule, error)
	UpdateRoutingRule(ctx context.Context, rule *models.GeoRoutingRule) error
	DeleteRoutingRule(ctx context.Context, id uuid.UUID) error

	// Replication Streams
	CreateReplicationStream(ctx context.Context, stream *models.ReplicationStream) error
	GetReplicationStream(ctx context.Context, id uuid.UUID) (*models.ReplicationStream, error)
	GetReplicationStreamsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.ReplicationStream, error)
	UpdateReplicationLag(ctx context.Context, id uuid.UUID, lagMs int64, lastEventID *uuid.UUID) error
	SetReplicationError(ctx context.Context, id uuid.UUID, errorMsg string) error

	// Routing Decisions
	CreateRoutingDecision(ctx context.Context, decision *models.RegionalRoutingDecision) error
	GetRoutingDecisions(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.RegionalRoutingDecision, error)

	// Health Metrics
	CreateHealthMetric(ctx context.Context, metric *models.RegionHealthMetric) error
	GetLatestHealthMetrics(ctx context.Context, regionID uuid.UUID) ([]*models.RegionHealthMetric, error)
	GetHealthMetricHistory(ctx context.Context, regionID uuid.UUID, metricType string, since time.Time) ([]*models.RegionHealthMetric, error)

	// Data Residency Audit
	CreateResidencyAudit(ctx context.Context, audit *models.DataResidencyAudit) error
	GetResidencyAudits(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.DataResidencyAudit, error)

	// Failover Events
	CreateFailoverEvent(ctx context.Context, event *models.FailoverEvent) error
	GetFailoverEvent(ctx context.Context, id uuid.UUID) (*models.FailoverEvent, error)
	GetRecentFailovers(ctx context.Context, limit int) ([]*models.FailoverEvent, error)
	CompleteFailover(ctx context.Context, id uuid.UUID, status string) error

	// Config Sync
	CreateConfigSync(ctx context.Context, sync *models.RegionalConfigSync) error
	GetConfigSync(ctx context.Context, configType string, configID, regionID uuid.UUID) (*models.RegionalConfigSync, error)
	UpdateConfigSyncStatus(ctx context.Context, id uuid.UUID, status string) error
	GetPendingSyncs(ctx context.Context, tenantID uuid.UUID) ([]*models.RegionalConfigSync, error)
}

// PostgresFederatedMeshRepository implements FederatedMeshRepository with PostgreSQL
type PostgresFederatedMeshRepository struct {
	pool *pgxpool.Pool
}

// NewFederatedMeshRepository creates a new repository
func NewFederatedMeshRepository(pool *pgxpool.Pool) FederatedMeshRepository {
	return &PostgresFederatedMeshRepository{pool: pool}
}

// GetAllRegions retrieves all regions
func (r *PostgresFederatedMeshRepository) GetAllRegions(ctx context.Context) ([]*models.Region, error) {
	query := `
		SELECT id, name, code, provider, location, latitude, longitude, status, is_primary,
		       health_status, last_health_check, capacity_limit, current_load, metadata,
		       created_at, updated_at
		FROM regions ORDER BY is_primary DESC, name`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var regions []*models.Region
	for rows.Next() {
		region := &models.Region{}
		var metadataJSON []byte
		if err := rows.Scan(
			&region.ID, &region.Name, &region.Code, &region.Provider, &region.Location,
			&region.Latitude, &region.Longitude, &region.Status, &region.IsPrimary,
			&region.HealthStatus, &region.LastHealthCheck, &region.CapacityLimit,
			&region.CurrentLoad, &metadataJSON, &region.CreatedAt, &region.UpdatedAt,
		); err != nil {
			return nil, err
		}
		json.Unmarshal(metadataJSON, &region.Metadata)
		regions = append(regions, region)
	}

	return regions, nil
}

// GetRegion retrieves a region by ID
func (r *PostgresFederatedMeshRepository) GetRegion(ctx context.Context, id uuid.UUID) (*models.Region, error) {
	query := `
		SELECT id, name, code, provider, location, latitude, longitude, status, is_primary,
		       health_status, last_health_check, capacity_limit, current_load, metadata,
		       created_at, updated_at
		FROM regions WHERE id = $1`

	region := &models.Region{}
	var metadataJSON []byte
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&region.ID, &region.Name, &region.Code, &region.Provider, &region.Location,
		&region.Latitude, &region.Longitude, &region.Status, &region.IsPrimary,
		&region.HealthStatus, &region.LastHealthCheck, &region.CapacityLimit,
		&region.CurrentLoad, &metadataJSON, &region.CreatedAt, &region.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(metadataJSON, &region.Metadata)

	return region, nil
}

// GetRegionByCode retrieves a region by code
func (r *PostgresFederatedMeshRepository) GetRegionByCode(ctx context.Context, code string) (*models.Region, error) {
	query := `
		SELECT id, name, code, provider, location, latitude, longitude, status, is_primary,
		       health_status, last_health_check, capacity_limit, current_load, metadata,
		       created_at, updated_at
		FROM regions WHERE code = $1`

	region := &models.Region{}
	var metadataJSON []byte
	err := r.pool.QueryRow(ctx, query, code).Scan(
		&region.ID, &region.Name, &region.Code, &region.Provider, &region.Location,
		&region.Latitude, &region.Longitude, &region.Status, &region.IsPrimary,
		&region.HealthStatus, &region.LastHealthCheck, &region.CapacityLimit,
		&region.CurrentLoad, &metadataJSON, &region.CreatedAt, &region.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(metadataJSON, &region.Metadata)

	return region, nil
}

// GetActiveRegions retrieves all active regions
func (r *PostgresFederatedMeshRepository) GetActiveRegions(ctx context.Context) ([]*models.Region, error) {
	query := `
		SELECT id, name, code, provider, location, latitude, longitude, status, is_primary,
		       health_status, last_health_check, capacity_limit, current_load, metadata,
		       created_at, updated_at
		FROM regions WHERE status = 'active' ORDER BY is_primary DESC, name`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var regions []*models.Region
	for rows.Next() {
		region := &models.Region{}
		var metadataJSON []byte
		if err := rows.Scan(
			&region.ID, &region.Name, &region.Code, &region.Provider, &region.Location,
			&region.Latitude, &region.Longitude, &region.Status, &region.IsPrimary,
			&region.HealthStatus, &region.LastHealthCheck, &region.CapacityLimit,
			&region.CurrentLoad, &metadataJSON, &region.CreatedAt, &region.UpdatedAt,
		); err != nil {
			return nil, err
		}
		json.Unmarshal(metadataJSON, &region.Metadata)
		regions = append(regions, region)
	}

	return regions, nil
}

// UpdateRegionHealth updates region health status
func (r *PostgresFederatedMeshRepository) UpdateRegionHealth(ctx context.Context, id uuid.UUID, status string) error {
	query := `UPDATE regions SET health_status = $2, last_health_check = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, status)
	return err
}

// UpdateRegionLoad updates region current load
func (r *PostgresFederatedMeshRepository) UpdateRegionLoad(ctx context.Context, id uuid.UUID, load int) error {
	query := `UPDATE regions SET current_load = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, load)
	return err
}

// CreateCluster creates a region cluster
func (r *PostgresFederatedMeshRepository) CreateCluster(ctx context.Context, cluster *models.RegionCluster) error {
	metadataJSON, _ := json.Marshal(cluster.Metadata)

	query := `
		INSERT INTO region_clusters (name, description, primary_region_id, failover_strategy, metadata)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`

	return r.pool.QueryRow(ctx, query,
		cluster.Name, cluster.Description, cluster.PrimaryRegionID,
		cluster.FailoverStrategy, metadataJSON,
	).Scan(&cluster.ID, &cluster.CreatedAt)
}

// GetCluster retrieves a cluster
func (r *PostgresFederatedMeshRepository) GetCluster(ctx context.Context, id uuid.UUID) (*models.RegionCluster, error) {
	query := `
		SELECT id, name, description, primary_region_id, failover_strategy, metadata, created_at
		FROM region_clusters WHERE id = $1`

	cluster := &models.RegionCluster{}
	var metadataJSON []byte
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&cluster.ID, &cluster.Name, &cluster.Description, &cluster.PrimaryRegionID,
		&cluster.FailoverStrategy, &metadataJSON, &cluster.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(metadataJSON, &cluster.Metadata)

	return cluster, nil
}

// GetClusters retrieves all clusters
func (r *PostgresFederatedMeshRepository) GetClusters(ctx context.Context) ([]*models.RegionCluster, error) {
	query := `SELECT id, name, description, primary_region_id, failover_strategy, metadata, created_at FROM region_clusters`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clusters []*models.RegionCluster
	for rows.Next() {
		cluster := &models.RegionCluster{}
		var metadataJSON []byte
		if err := rows.Scan(
			&cluster.ID, &cluster.Name, &cluster.Description, &cluster.PrimaryRegionID,
			&cluster.FailoverStrategy, &metadataJSON, &cluster.CreatedAt,
		); err != nil {
			return nil, err
		}
		json.Unmarshal(metadataJSON, &cluster.Metadata)
		clusters = append(clusters, cluster)
	}

	return clusters, nil
}

// AddClusterMember adds a member to a cluster
func (r *PostgresFederatedMeshRepository) AddClusterMember(ctx context.Context, member *models.RegionClusterMember) error {
	query := `
		INSERT INTO region_cluster_members (cluster_id, region_id, priority, weight)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`

	return r.pool.QueryRow(ctx, query,
		member.ClusterID, member.RegionID, member.Priority, member.Weight,
	).Scan(&member.ID, &member.CreatedAt)
}

// GetClusterMembers retrieves cluster members
func (r *PostgresFederatedMeshRepository) GetClusterMembers(ctx context.Context, clusterID uuid.UUID) ([]*models.RegionClusterMember, error) {
	query := `
		SELECT m.id, m.cluster_id, m.region_id, m.priority, m.weight, m.created_at
		FROM region_cluster_members m
		WHERE m.cluster_id = $1 ORDER BY m.priority`

	rows, err := r.pool.Query(ctx, query, clusterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*models.RegionClusterMember
	for rows.Next() {
		member := &models.RegionClusterMember{}
		if err := rows.Scan(
			&member.ID, &member.ClusterID, &member.RegionID, &member.Priority, &member.Weight, &member.CreatedAt,
		); err != nil {
			return nil, err
		}
		members = append(members, member)
	}

	return members, nil
}

// CreateTenantRegion creates a tenant region assignment
func (r *PostgresFederatedMeshRepository) CreateTenantRegion(ctx context.Context, tr *models.MeshTenantRegion) error {
	query := `
		INSERT INTO tenant_regions (tenant_id, primary_region_id, allowed_regions, 
		                           data_residency_policy, replication_mode, compliance_frameworks)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		tr.TenantID, tr.PrimaryRegionID, pq.Array(tr.AllowedRegions),
		tr.DataResidencyPolicy, tr.ReplicationMode, pq.Array(tr.ComplianceFrameworks),
	).Scan(&tr.ID, &tr.CreatedAt, &tr.UpdatedAt)
}

// GetTenantRegion retrieves tenant region assignment
func (r *PostgresFederatedMeshRepository) GetTenantRegion(ctx context.Context, tenantID uuid.UUID) (*models.MeshTenantRegion, error) {
	query := `
		SELECT id, tenant_id, primary_region_id, allowed_regions, data_residency_policy,
		       replication_mode, compliance_frameworks, created_at, updated_at
		FROM tenant_regions WHERE tenant_id = $1`

	tr := &models.MeshTenantRegion{}
	err := r.pool.QueryRow(ctx, query, tenantID).Scan(
		&tr.ID, &tr.TenantID, &tr.PrimaryRegionID, pq.Array(&tr.AllowedRegions),
		&tr.DataResidencyPolicy, &tr.ReplicationMode, pq.Array(&tr.ComplianceFrameworks),
		&tr.CreatedAt, &tr.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return tr, nil
}

// UpdateTenantRegion updates tenant region assignment
func (r *PostgresFederatedMeshRepository) UpdateTenantRegion(ctx context.Context, tr *models.MeshTenantRegion) error {
	query := `
		UPDATE tenant_regions 
		SET primary_region_id = $2, allowed_regions = $3, data_residency_policy = $4,
		    replication_mode = $5, compliance_frameworks = $6, updated_at = NOW()
		WHERE tenant_id = $1`

	_, err := r.pool.Exec(ctx, query,
		tr.TenantID, tr.PrimaryRegionID, pq.Array(tr.AllowedRegions),
		tr.DataResidencyPolicy, tr.ReplicationMode, pq.Array(tr.ComplianceFrameworks),
	)
	return err
}

// CreateRoutingRule creates a geo routing rule
func (r *PostgresFederatedMeshRepository) CreateRoutingRule(ctx context.Context, rule *models.GeoRoutingRule) error {
	conditionsJSON, _ := json.Marshal(rule.Conditions)

	query := `
		INSERT INTO geo_routing_rules (tenant_id, name, description, rule_type, priority,
		                               source_regions, target_region_id, conditions, enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		rule.TenantID, rule.Name, rule.Description, rule.RuleType, rule.Priority,
		pq.Array(rule.SourceRegions), rule.TargetRegionID, conditionsJSON, rule.Enabled,
	).Scan(&rule.ID, &rule.CreatedAt, &rule.UpdatedAt)
}

// GetRoutingRule retrieves a routing rule
func (r *PostgresFederatedMeshRepository) GetRoutingRule(ctx context.Context, id uuid.UUID) (*models.GeoRoutingRule, error) {
	query := `
		SELECT id, tenant_id, name, description, rule_type, priority, source_regions,
		       target_region_id, conditions, enabled, created_at, updated_at
		FROM geo_routing_rules WHERE id = $1`

	rule := &models.GeoRoutingRule{}
	var conditionsJSON []byte
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&rule.ID, &rule.TenantID, &rule.Name, &rule.Description, &rule.RuleType,
		&rule.Priority, pq.Array(&rule.SourceRegions), &rule.TargetRegionID,
		&conditionsJSON, &rule.Enabled, &rule.CreatedAt, &rule.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(conditionsJSON, &rule.Conditions)

	return rule, nil
}

// GetRoutingRulesByTenant retrieves all routing rules for a tenant
func (r *PostgresFederatedMeshRepository) GetRoutingRulesByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.GeoRoutingRule, error) {
	query := `
		SELECT id, tenant_id, name, description, rule_type, priority, source_regions,
		       target_region_id, conditions, enabled, created_at, updated_at
		FROM geo_routing_rules WHERE tenant_id = $1 ORDER BY priority`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*models.GeoRoutingRule
	for rows.Next() {
		rule := &models.GeoRoutingRule{}
		var conditionsJSON []byte
		if err := rows.Scan(
			&rule.ID, &rule.TenantID, &rule.Name, &rule.Description, &rule.RuleType,
			&rule.Priority, pq.Array(&rule.SourceRegions), &rule.TargetRegionID,
			&conditionsJSON, &rule.Enabled, &rule.CreatedAt, &rule.UpdatedAt,
		); err != nil {
			return nil, err
		}
		json.Unmarshal(conditionsJSON, &rule.Conditions)
		rules = append(rules, rule)
	}

	return rules, nil
}

// GetEnabledRoutingRules retrieves enabled routing rules
func (r *PostgresFederatedMeshRepository) GetEnabledRoutingRules(ctx context.Context, tenantID uuid.UUID) ([]*models.GeoRoutingRule, error) {
	query := `
		SELECT id, tenant_id, name, description, rule_type, priority, source_regions,
		       target_region_id, conditions, enabled, created_at, updated_at
		FROM geo_routing_rules WHERE tenant_id = $1 AND enabled = TRUE ORDER BY priority`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*models.GeoRoutingRule
	for rows.Next() {
		rule := &models.GeoRoutingRule{}
		var conditionsJSON []byte
		if err := rows.Scan(
			&rule.ID, &rule.TenantID, &rule.Name, &rule.Description, &rule.RuleType,
			&rule.Priority, pq.Array(&rule.SourceRegions), &rule.TargetRegionID,
			&conditionsJSON, &rule.Enabled, &rule.CreatedAt, &rule.UpdatedAt,
		); err != nil {
			return nil, err
		}
		json.Unmarshal(conditionsJSON, &rule.Conditions)
		rules = append(rules, rule)
	}

	return rules, nil
}

// UpdateRoutingRule updates a routing rule
func (r *PostgresFederatedMeshRepository) UpdateRoutingRule(ctx context.Context, rule *models.GeoRoutingRule) error {
	conditionsJSON, _ := json.Marshal(rule.Conditions)

	query := `
		UPDATE geo_routing_rules 
		SET name = $2, description = $3, rule_type = $4, priority = $5, source_regions = $6,
		    target_region_id = $7, conditions = $8, enabled = $9, updated_at = NOW()
		WHERE id = $1`

	_, err := r.pool.Exec(ctx, query,
		rule.ID, rule.Name, rule.Description, rule.RuleType, rule.Priority,
		pq.Array(rule.SourceRegions), rule.TargetRegionID, conditionsJSON, rule.Enabled,
	)
	return err
}

// DeleteRoutingRule deletes a routing rule
func (r *PostgresFederatedMeshRepository) DeleteRoutingRule(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM geo_routing_rules WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// CreateReplicationStream creates a replication stream
func (r *PostgresFederatedMeshRepository) CreateReplicationStream(ctx context.Context, stream *models.ReplicationStream) error {
	metadataJSON, _ := json.Marshal(stream.Metadata)

	query := `
		INSERT INTO replication_streams (tenant_id, source_region_id, target_region_id, stream_type, status, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		stream.TenantID, stream.SourceRegionID, stream.TargetRegionID,
		stream.StreamType, stream.Status, metadataJSON,
	).Scan(&stream.ID, &stream.CreatedAt, &stream.UpdatedAt)
}

// GetReplicationStream retrieves a replication stream
func (r *PostgresFederatedMeshRepository) GetReplicationStream(ctx context.Context, id uuid.UUID) (*models.ReplicationStream, error) {
	query := `
		SELECT id, tenant_id, source_region_id, target_region_id, stream_type, status,
		       lag_ms, last_replicated_at, last_event_id, error_message, metadata, created_at, updated_at
		FROM replication_streams WHERE id = $1`

	stream := &models.ReplicationStream{}
	var metadataJSON []byte
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&stream.ID, &stream.TenantID, &stream.SourceRegionID, &stream.TargetRegionID,
		&stream.StreamType, &stream.Status, &stream.LagMs, &stream.LastReplicatedAt,
		&stream.LastEventID, &stream.ErrorMessage, &metadataJSON, &stream.CreatedAt, &stream.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(metadataJSON, &stream.Metadata)

	return stream, nil
}

// GetReplicationStreamsByTenant retrieves replication streams for a tenant
func (r *PostgresFederatedMeshRepository) GetReplicationStreamsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.ReplicationStream, error) {
	query := `
		SELECT id, tenant_id, source_region_id, target_region_id, stream_type, status,
		       lag_ms, last_replicated_at, last_event_id, error_message, metadata, created_at, updated_at
		FROM replication_streams WHERE tenant_id = $1`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var streams []*models.ReplicationStream
	for rows.Next() {
		stream := &models.ReplicationStream{}
		var metadataJSON []byte
		if err := rows.Scan(
			&stream.ID, &stream.TenantID, &stream.SourceRegionID, &stream.TargetRegionID,
			&stream.StreamType, &stream.Status, &stream.LagMs, &stream.LastReplicatedAt,
			&stream.LastEventID, &stream.ErrorMessage, &metadataJSON, &stream.CreatedAt, &stream.UpdatedAt,
		); err != nil {
			return nil, err
		}
		json.Unmarshal(metadataJSON, &stream.Metadata)
		streams = append(streams, stream)
	}

	return streams, nil
}

// UpdateReplicationLag updates replication lag
func (r *PostgresFederatedMeshRepository) UpdateReplicationLag(ctx context.Context, id uuid.UUID, lagMs int64, lastEventID *uuid.UUID) error {
	query := `
		UPDATE replication_streams 
		SET lag_ms = $2, last_event_id = $3, last_replicated_at = NOW(), updated_at = NOW()
		WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, lagMs, lastEventID)
	return err
}

// SetReplicationError sets a replication error
func (r *PostgresFederatedMeshRepository) SetReplicationError(ctx context.Context, id uuid.UUID, errorMsg string) error {
	query := `UPDATE replication_streams SET status = 'error', error_message = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, errorMsg)
	return err
}

// CreateRoutingDecision creates a routing decision
func (r *PostgresFederatedMeshRepository) CreateRoutingDecision(ctx context.Context, decision *models.RegionalRoutingDecision) error {
	query := `
		INSERT INTO regional_routing_decisions (tenant_id, event_id, source_region_id, target_region_id,
		                                        routing_rule_id, decision_reason, latency_ms)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`

	return r.pool.QueryRow(ctx, query,
		decision.TenantID, decision.EventID, decision.SourceRegionID, decision.TargetRegionID,
		decision.RoutingRuleID, decision.DecisionReason, decision.LatencyMs,
	).Scan(&decision.ID, &decision.CreatedAt)
}

// GetRoutingDecisions retrieves routing decisions
func (r *PostgresFederatedMeshRepository) GetRoutingDecisions(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.RegionalRoutingDecision, error) {
	query := `
		SELECT id, tenant_id, event_id, source_region_id, target_region_id, routing_rule_id,
		       decision_reason, latency_ms, created_at
		FROM regional_routing_decisions WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2`

	rows, err := r.pool.Query(ctx, query, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var decisions []*models.RegionalRoutingDecision
	for rows.Next() {
		d := &models.RegionalRoutingDecision{}
		if err := rows.Scan(
			&d.ID, &d.TenantID, &d.EventID, &d.SourceRegionID, &d.TargetRegionID,
			&d.RoutingRuleID, &d.DecisionReason, &d.LatencyMs, &d.CreatedAt,
		); err != nil {
			return nil, err
		}
		decisions = append(decisions, d)
	}

	return decisions, nil
}

// CreateHealthMetric creates a health metric
func (r *PostgresFederatedMeshRepository) CreateHealthMetric(ctx context.Context, metric *models.RegionHealthMetric) error {
	query := `
		INSERT INTO region_health_metrics (region_id, metric_type, metric_value)
		VALUES ($1, $2, $3)
		RETURNING id, recorded_at`

	return r.pool.QueryRow(ctx, query, metric.RegionID, metric.MetricType, metric.MetricValue).Scan(&metric.ID, &metric.RecordedAt)
}

// GetLatestHealthMetrics retrieves latest health metrics
func (r *PostgresFederatedMeshRepository) GetLatestHealthMetrics(ctx context.Context, regionID uuid.UUID) ([]*models.RegionHealthMetric, error) {
	query := `
		SELECT DISTINCT ON (metric_type) id, region_id, metric_type, metric_value, recorded_at
		FROM region_health_metrics WHERE region_id = $1 ORDER BY metric_type, recorded_at DESC`

	rows, err := r.pool.Query(ctx, query, regionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []*models.RegionHealthMetric
	for rows.Next() {
		m := &models.RegionHealthMetric{}
		if err := rows.Scan(&m.ID, &m.RegionID, &m.MetricType, &m.MetricValue, &m.RecordedAt); err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}

	return metrics, nil
}

// GetHealthMetricHistory retrieves health metric history
func (r *PostgresFederatedMeshRepository) GetHealthMetricHistory(ctx context.Context, regionID uuid.UUID, metricType string, since time.Time) ([]*models.RegionHealthMetric, error) {
	query := `
		SELECT id, region_id, metric_type, metric_value, recorded_at
		FROM region_health_metrics 
		WHERE region_id = $1 AND metric_type = $2 AND recorded_at >= $3
		ORDER BY recorded_at`

	rows, err := r.pool.Query(ctx, query, regionID, metricType, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []*models.RegionHealthMetric
	for rows.Next() {
		m := &models.RegionHealthMetric{}
		if err := rows.Scan(&m.ID, &m.RegionID, &m.MetricType, &m.MetricValue, &m.RecordedAt); err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}

	return metrics, nil
}

// CreateResidencyAudit creates a residency audit entry
func (r *PostgresFederatedMeshRepository) CreateResidencyAudit(ctx context.Context, audit *models.DataResidencyAudit) error {
	detailsJSON, _ := json.Marshal(audit.Details)

	query := `
		INSERT INTO data_residency_audit (tenant_id, event_type, source_region_id, target_region_id,
		                                  data_type, compliance_status, details)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, recorded_at`

	return r.pool.QueryRow(ctx, query,
		audit.TenantID, audit.EventType, audit.SourceRegionID, audit.TargetRegionID,
		audit.DataType, audit.ComplianceStatus, detailsJSON,
	).Scan(&audit.ID, &audit.RecordedAt)
}

// GetResidencyAudits retrieves residency audits
func (r *PostgresFederatedMeshRepository) GetResidencyAudits(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.DataResidencyAudit, error) {
	query := `
		SELECT id, tenant_id, event_type, source_region_id, target_region_id, data_type,
		       compliance_status, details, recorded_at
		FROM data_residency_audit WHERE tenant_id = $1 ORDER BY recorded_at DESC LIMIT $2`

	rows, err := r.pool.Query(ctx, query, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var audits []*models.DataResidencyAudit
	for rows.Next() {
		a := &models.DataResidencyAudit{}
		var detailsJSON []byte
		if err := rows.Scan(
			&a.ID, &a.TenantID, &a.EventType, &a.SourceRegionID, &a.TargetRegionID,
			&a.DataType, &a.ComplianceStatus, &detailsJSON, &a.RecordedAt,
		); err != nil {
			return nil, err
		}
		json.Unmarshal(detailsJSON, &a.Details)
		audits = append(audits, a)
	}

	return audits, nil
}

// CreateFailoverEvent creates a failover event
func (r *PostgresFederatedMeshRepository) CreateFailoverEvent(ctx context.Context, event *models.FailoverEvent) error {
	metadataJSON, _ := json.Marshal(event.Metadata)

	query := `
		INSERT INTO failover_events (cluster_id, from_region_id, to_region_id, trigger_reason,
		                             automatic, status, affected_tenants, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, started_at`

	return r.pool.QueryRow(ctx, query,
		event.ClusterID, event.FromRegionID, event.ToRegionID, event.TriggerReason,
		event.Automatic, event.Status, event.AffectedTenants, metadataJSON,
	).Scan(&event.ID, &event.StartedAt)
}

// GetFailoverEvent retrieves a failover event
func (r *PostgresFederatedMeshRepository) GetFailoverEvent(ctx context.Context, id uuid.UUID) (*models.FailoverEvent, error) {
	query := `
		SELECT id, cluster_id, from_region_id, to_region_id, trigger_reason, automatic,
		       started_at, completed_at, status, affected_tenants, metadata
		FROM failover_events WHERE id = $1`

	event := &models.FailoverEvent{}
	var metadataJSON []byte
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&event.ID, &event.ClusterID, &event.FromRegionID, &event.ToRegionID,
		&event.TriggerReason, &event.Automatic, &event.StartedAt, &event.CompletedAt,
		&event.Status, &event.AffectedTenants, &metadataJSON,
	)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(metadataJSON, &event.Metadata)

	return event, nil
}

// GetRecentFailovers retrieves recent failover events
func (r *PostgresFederatedMeshRepository) GetRecentFailovers(ctx context.Context, limit int) ([]*models.FailoverEvent, error) {
	query := `
		SELECT id, cluster_id, from_region_id, to_region_id, trigger_reason, automatic,
		       started_at, completed_at, status, affected_tenants, metadata
		FROM failover_events ORDER BY started_at DESC LIMIT $1`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*models.FailoverEvent
	for rows.Next() {
		event := &models.FailoverEvent{}
		var metadataJSON []byte
		if err := rows.Scan(
			&event.ID, &event.ClusterID, &event.FromRegionID, &event.ToRegionID,
			&event.TriggerReason, &event.Automatic, &event.StartedAt, &event.CompletedAt,
			&event.Status, &event.AffectedTenants, &metadataJSON,
		); err != nil {
			return nil, err
		}
		json.Unmarshal(metadataJSON, &event.Metadata)
		events = append(events, event)
	}

	return events, nil
}

// CompleteFailover completes a failover event
func (r *PostgresFederatedMeshRepository) CompleteFailover(ctx context.Context, id uuid.UUID, status string) error {
	query := `UPDATE failover_events SET status = $2, completed_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, status)
	return err
}

// CreateConfigSync creates a config sync record
func (r *PostgresFederatedMeshRepository) CreateConfigSync(ctx context.Context, sync *models.RegionalConfigSync) error {
	query := `
		INSERT INTO regional_config_sync (tenant_id, config_type, config_id, region_id, version, sync_status, config_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (config_type, config_id, region_id) DO UPDATE SET
		    version = EXCLUDED.version, sync_status = EXCLUDED.sync_status, 
		    config_hash = EXCLUDED.config_hash, updated_at = NOW()
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		sync.TenantID, sync.ConfigType, sync.ConfigID, sync.RegionID,
		sync.Version, sync.SyncStatus, sync.ConfigHash,
	).Scan(&sync.ID, &sync.CreatedAt, &sync.UpdatedAt)
}

// GetConfigSync retrieves config sync status
func (r *PostgresFederatedMeshRepository) GetConfigSync(ctx context.Context, configType string, configID, regionID uuid.UUID) (*models.RegionalConfigSync, error) {
	query := `
		SELECT id, tenant_id, config_type, config_id, region_id, version, sync_status,
		       last_synced_at, config_hash, created_at, updated_at
		FROM regional_config_sync WHERE config_type = $1 AND config_id = $2 AND region_id = $3`

	sync := &models.RegionalConfigSync{}
	err := r.pool.QueryRow(ctx, query, configType, configID, regionID).Scan(
		&sync.ID, &sync.TenantID, &sync.ConfigType, &sync.ConfigID, &sync.RegionID,
		&sync.Version, &sync.SyncStatus, &sync.LastSyncedAt, &sync.ConfigHash,
		&sync.CreatedAt, &sync.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return sync, err
}

// UpdateConfigSyncStatus updates config sync status
func (r *PostgresFederatedMeshRepository) UpdateConfigSyncStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `UPDATE regional_config_sync SET sync_status = $2, last_synced_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, status)
	return err
}

// GetPendingSyncs retrieves pending syncs
func (r *PostgresFederatedMeshRepository) GetPendingSyncs(ctx context.Context, tenantID uuid.UUID) ([]*models.RegionalConfigSync, error) {
	query := `
		SELECT id, tenant_id, config_type, config_id, region_id, version, sync_status,
		       last_synced_at, config_hash, created_at, updated_at
		FROM regional_config_sync WHERE tenant_id = $1 AND sync_status = 'pending'`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var syncs []*models.RegionalConfigSync
	for rows.Next() {
		sync := &models.RegionalConfigSync{}
		if err := rows.Scan(
			&sync.ID, &sync.TenantID, &sync.ConfigType, &sync.ConfigID, &sync.RegionID,
			&sync.Version, &sync.SyncStatus, &sync.LastSyncedAt, &sync.ConfigHash,
			&sync.CreatedAt, &sync.UpdatedAt,
		); err != nil {
			return nil, err
		}
		syncs = append(syncs, sync)
	}

	return syncs, nil
}
