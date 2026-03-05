package georouting

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/josedab/waas/pkg/database"
)

// Repository defines the interface for geo-routing storage
type Repository interface {
	CreateEndpointRouting(ctx context.Context, routing *EndpointRouting) error
	GetEndpointRouting(ctx context.Context, tenantID, endpointID string) (*EndpointRouting, error)
	UpdateEndpointRouting(ctx context.Context, routing *EndpointRouting) error
	DeleteEndpointRouting(ctx context.Context, tenantID, endpointID string) error

	CreateRegionConfig(ctx context.Context, config *RegionConfig) error
	GetRegionConfig(ctx context.Context, regionID string) (*RegionConfig, error)
	ListRegionConfigs(ctx context.Context) ([]RegionConfig, error)
	UpdateRegionConfig(ctx context.Context, config *RegionConfig) error

	GetRegionHealth(ctx context.Context, regionID string) (*RegionHealth, error)
	UpdateRegionHealth(ctx context.Context, health *RegionHealth) error

	GetRoutingStats(ctx context.Context, tenantID, period string) (*RoutingStats, error)
	RecordRoutingDecision(ctx context.Context, decision *RoutingDecision) error

	// Enhanced geo-routing methods
	CreateGeoRegion(ctx context.Context, region *GeoRegion) error
	GetGeoRegion(ctx context.Context, name string) (*GeoRegion, error)
	ListGeoRegions(ctx context.Context) ([]GeoRegion, error)
	UpdateGeoRegion(ctx context.Context, region *GeoRegion) error

	CreateGeoRoutingPolicy(ctx context.Context, policy *GeoRoutingPolicy) error
	GetGeoRoutingPolicy(ctx context.Context, tenantID uuid.UUID) (*GeoRoutingPolicy, error)
	GetGeoRoutingPolicyByID(ctx context.Context, id uuid.UUID) (*GeoRoutingPolicy, error)
	UpdateGeoRoutingPolicy(ctx context.Context, policy *GeoRoutingPolicy) error
	ListGeoRoutingPolicies(ctx context.Context, tenantID uuid.UUID) ([]GeoRoutingPolicy, error)

	GetEndpointRegionConfig(ctx context.Context, endpointID uuid.UUID) (*EndpointRegionConfig, error)
	SaveEndpointRegionConfig(ctx context.Context, config *EndpointRegionConfig) error

	RecordGeoRoutingDecision(ctx context.Context, decision *GeoRoutingDecision) error
	ListGeoRoutingDecisions(ctx context.Context, tenantID uuid.UUID, limit int) ([]GeoRoutingDecision, error)
}

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// CreateEndpointRouting creates a new endpoint routing configuration
func (r *PostgresRepository) CreateEndpointRouting(ctx context.Context, routing *EndpointRouting) error {
	if routing.ID == "" {
		routing.ID = uuid.New().String()
	}

	regions := make([]string, len(routing.Regions))
	for i, region := range routing.Regions {
		regions[i] = string(region)
	}

	query := `
		INSERT INTO endpoint_routing 
		(id, endpoint_id, tenant_id, mode, primary_region, regions, data_residency, failover_enabled, latency_based, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.ExecContext(ctx, query,
		routing.ID, routing.EndpointID, routing.TenantID, routing.Mode,
		routing.PrimaryRegion, database.StringArray(regions), routing.DataResidency,
		routing.FailoverEnabled, routing.LatencyBased,
		routing.CreatedAt, routing.UpdatedAt,
	)

	return err
}

// GetEndpointRouting retrieves routing configuration for an endpoint
func (r *PostgresRepository) GetEndpointRouting(ctx context.Context, tenantID, endpointID string) (*EndpointRouting, error) {
	query := `
		SELECT id, endpoint_id, tenant_id, mode, primary_region, regions, data_residency, failover_enabled, latency_based, created_at, updated_at
		FROM endpoint_routing
		WHERE endpoint_id = $1 AND tenant_id = $2
	`

	var routing EndpointRouting
	var regions []string

	err := r.db.QueryRowContext(ctx, query, endpointID, tenantID).Scan(
		&routing.ID, &routing.EndpointID, &routing.TenantID, &routing.Mode,
		&routing.PrimaryRegion, (*database.StringArray)(&regions), &routing.DataResidency,
		&routing.FailoverEnabled, &routing.LatencyBased,
		&routing.CreatedAt, &routing.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	routing.Regions = make([]Region, len(regions))
	for i, region := range regions {
		routing.Regions[i] = Region(region)
	}

	return &routing, nil
}

// UpdateEndpointRouting updates routing configuration
func (r *PostgresRepository) UpdateEndpointRouting(ctx context.Context, routing *EndpointRouting) error {
	regions := make([]string, len(routing.Regions))
	for i, region := range routing.Regions {
		regions[i] = string(region)
	}

	query := `
		UPDATE endpoint_routing
		SET mode = $1, primary_region = $2, regions = $3, data_residency = $4, 
		    failover_enabled = $5, latency_based = $6, updated_at = $7
		WHERE id = $8 AND tenant_id = $9
	`

	_, err := r.db.ExecContext(ctx, query,
		routing.Mode, routing.PrimaryRegion, database.StringArray(regions), routing.DataResidency,
		routing.FailoverEnabled, routing.LatencyBased, routing.UpdatedAt,
		routing.ID, routing.TenantID,
	)

	return err
}

// DeleteEndpointRouting deletes routing configuration
func (r *PostgresRepository) DeleteEndpointRouting(ctx context.Context, tenantID, endpointID string) error {
	query := `DELETE FROM endpoint_routing WHERE endpoint_id = $1 AND tenant_id = $2`
	_, err := r.db.ExecContext(ctx, query, endpointID, tenantID)
	return err
}

// CreateRegionConfig creates a new region configuration
func (r *PostgresRepository) CreateRegionConfig(ctx context.Context, config *RegionConfig) error {
	if config.ID == "" {
		config.ID = uuid.New().String()
	}

	healthJSON, _ := json.Marshal(config.HealthCheck)
	metadataJSON, _ := json.Marshal(config.Metadata)

	query := `
		INSERT INTO region_configs 
		(id, region, name, endpoint, is_active, is_primary, priority, max_concurrent, health_check, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := r.db.ExecContext(ctx, query,
		config.ID, config.Region, config.Name, config.Endpoint,
		config.IsActive, config.IsPrimary, config.Priority, config.MaxConcurrent,
		healthJSON, metadataJSON,
		config.CreatedAt, config.UpdatedAt,
	)

	return err
}

// GetRegionConfig retrieves a region configuration
func (r *PostgresRepository) GetRegionConfig(ctx context.Context, regionID string) (*RegionConfig, error) {
	query := `
		SELECT id, region, name, endpoint, is_active, is_primary, priority, max_concurrent, health_check, metadata, created_at, updated_at
		FROM region_configs
		WHERE id = $1
	`

	var config RegionConfig
	var healthJSON, metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, regionID).Scan(
		&config.ID, &config.Region, &config.Name, &config.Endpoint,
		&config.IsActive, &config.IsPrimary, &config.Priority, &config.MaxConcurrent,
		&healthJSON, &metadataJSON,
		&config.CreatedAt, &config.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(healthJSON, &config.HealthCheck)
	json.Unmarshal(metadataJSON, &config.Metadata)

	return &config, nil
}

// ListRegionConfigs lists all region configurations
func (r *PostgresRepository) ListRegionConfigs(ctx context.Context) ([]RegionConfig, error) {
	query := `
		SELECT id, region, name, endpoint, is_active, is_primary, priority, max_concurrent, health_check, metadata, created_at, updated_at
		FROM region_configs
		ORDER BY priority ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []RegionConfig
	for rows.Next() {
		var config RegionConfig
		var healthJSON, metadataJSON []byte

		if err := rows.Scan(
			&config.ID, &config.Region, &config.Name, &config.Endpoint,
			&config.IsActive, &config.IsPrimary, &config.Priority, &config.MaxConcurrent,
			&healthJSON, &metadataJSON,
			&config.CreatedAt, &config.UpdatedAt,
		); err != nil {
			return nil, err
		}

		json.Unmarshal(healthJSON, &config.HealthCheck)
		json.Unmarshal(metadataJSON, &config.Metadata)

		configs = append(configs, config)
	}

	return configs, nil
}

// UpdateRegionConfig updates a region configuration
func (r *PostgresRepository) UpdateRegionConfig(ctx context.Context, config *RegionConfig) error {
	healthJSON, _ := json.Marshal(config.HealthCheck)
	metadataJSON, _ := json.Marshal(config.Metadata)

	query := `
		UPDATE region_configs
		SET name = $1, endpoint = $2, is_active = $3, is_primary = $4, 
		    priority = $5, max_concurrent = $6, health_check = $7, metadata = $8, updated_at = $9
		WHERE id = $10
	`

	_, err := r.db.ExecContext(ctx, query,
		config.Name, config.Endpoint, config.IsActive, config.IsPrimary,
		config.Priority, config.MaxConcurrent, healthJSON, metadataJSON, config.UpdatedAt,
		config.ID,
	)

	return err
}

// GetRegionHealth retrieves health status for a region
func (r *PostgresRepository) GetRegionHealth(ctx context.Context, regionID string) (*RegionHealth, error) {
	query := `
		SELECT region_id, is_healthy, last_check, consecutive_ok, consecutive_fail, avg_latency_ms, error_rate, last_error, last_error_at
		FROM region_health
		WHERE region_id = $1
	`

	var health RegionHealth
	var lastErrorAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, regionID).Scan(
		&health.RegionID, &health.IsHealthy, &health.LastCheck,
		&health.ConsecutiveOK, &health.ConsecutiveFail, &health.AvgLatencyMs,
		&health.ErrorRate, &health.LastError, &lastErrorAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if lastErrorAt.Valid {
		health.LastErrorAt = &lastErrorAt.Time
	}

	return &health, nil
}

// UpdateRegionHealth updates health status for a region
func (r *PostgresRepository) UpdateRegionHealth(ctx context.Context, health *RegionHealth) error {
	query := `
		INSERT INTO region_health 
		(region_id, is_healthy, last_check, consecutive_ok, consecutive_fail, avg_latency_ms, error_rate, last_error, last_error_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (region_id) DO UPDATE SET
			is_healthy = EXCLUDED.is_healthy,
			last_check = EXCLUDED.last_check,
			consecutive_ok = EXCLUDED.consecutive_ok,
			consecutive_fail = EXCLUDED.consecutive_fail,
			avg_latency_ms = EXCLUDED.avg_latency_ms,
			error_rate = EXCLUDED.error_rate,
			last_error = EXCLUDED.last_error,
			last_error_at = EXCLUDED.last_error_at
	`

	_, err := r.db.ExecContext(ctx, query,
		health.RegionID, health.IsHealthy, health.LastCheck,
		health.ConsecutiveOK, health.ConsecutiveFail, health.AvgLatencyMs,
		health.ErrorRate, health.LastError, health.LastErrorAt,
	)

	return err
}

// GetRoutingStats retrieves routing statistics
func (r *PostgresRepository) GetRoutingStats(ctx context.Context, tenantID, period string) (*RoutingStats, error) {
	// Simplified implementation - in production would aggregate from routing_decisions table
	return &RoutingStats{
		TenantID: tenantID,
		Period:   period,
		ByRegion: make(map[Region]int64),
		ByMode:   make(map[RoutingMode]int64),
	}, nil
}

// RecordRoutingDecision records a routing decision
func (r *PostgresRepository) RecordRoutingDecision(ctx context.Context, decision *RoutingDecision) error {
	query := `
		INSERT INTO routing_decisions (endpoint_id, selected_region, reason, latency_ms, timestamp)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.ExecContext(ctx, query,
		decision.EndpointID, decision.SelectedRegion, decision.Reason,
		decision.LatencyMs, decision.Timestamp,
	)

	return err
}

// CreateGeoRegion creates a new geo region
func (r *PostgresRepository) CreateGeoRegion(ctx context.Context, region *GeoRegion) error {
	query := `
		INSERT INTO geo_regions (id, name, display_name, provider, latitude, longitude, status, capacity, current_load, avg_latency_ms, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.db.ExecContext(ctx, query,
		region.ID, region.Name, region.DisplayName, region.Provider,
		region.Latitude, region.Longitude, region.Status,
		region.Capacity, region.CurrentLoad, region.AvgLatency, region.CreatedAt,
	)
	return err
}

// GetGeoRegion retrieves a geo region by name
func (r *PostgresRepository) GetGeoRegion(ctx context.Context, name string) (*GeoRegion, error) {
	query := `
		SELECT id, name, display_name, provider, latitude, longitude, status, capacity, current_load, avg_latency_ms, created_at
		FROM geo_regions WHERE name = $1
	`
	var region GeoRegion
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&region.ID, &region.Name, &region.DisplayName, &region.Provider,
		&region.Latitude, &region.Longitude, &region.Status,
		&region.Capacity, &region.CurrentLoad, &region.AvgLatency, &region.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &region, nil
}

// ListGeoRegions lists all geo regions
func (r *PostgresRepository) ListGeoRegions(ctx context.Context) ([]GeoRegion, error) {
	query := `
		SELECT id, name, display_name, provider, latitude, longitude, status, capacity, current_load, avg_latency_ms, created_at
		FROM geo_regions ORDER BY name ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var regions []GeoRegion
	for rows.Next() {
		var region GeoRegion
		if err := rows.Scan(
			&region.ID, &region.Name, &region.DisplayName, &region.Provider,
			&region.Latitude, &region.Longitude, &region.Status,
			&region.Capacity, &region.CurrentLoad, &region.AvgLatency, &region.CreatedAt,
		); err != nil {
			return nil, err
		}
		regions = append(regions, region)
	}
	return regions, rows.Err()
}

// UpdateGeoRegion updates a geo region
func (r *PostgresRepository) UpdateGeoRegion(ctx context.Context, region *GeoRegion) error {
	query := `
		UPDATE geo_regions SET display_name=$1, provider=$2, latitude=$3, longitude=$4, status=$5,
		capacity=$6, current_load=$7, avg_latency_ms=$8
		WHERE id = $9
	`
	_, err := r.db.ExecContext(ctx, query,
		region.DisplayName, region.Provider, region.Latitude, region.Longitude, region.Status,
		region.Capacity, region.CurrentLoad, region.AvgLatency, region.ID,
	)
	return err
}

// CreateGeoRoutingPolicy creates a geo routing policy
func (r *PostgresRepository) CreateGeoRoutingPolicy(ctx context.Context, policy *GeoRoutingPolicy) error {
	dataResJSON, _ := json.Marshal(policy.DataResidencyReq)
	prefJSON, _ := json.Marshal(policy.PreferredRegions)
	foJSON, _ := json.Marshal(policy.FailoverOrder)
	wJSON, _ := json.Marshal(policy.Weights)

	query := `
		INSERT INTO geo_routing_policies (id, tenant_id, name, strategy, data_residency, preferred_regions, failover_order, weights, active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.ExecContext(ctx, query,
		policy.ID, policy.TenantID, policy.Name, policy.Strategy,
		dataResJSON, prefJSON, foJSON, wJSON, policy.Active, policy.CreatedAt,
	)
	return err
}

// GetGeoRoutingPolicy retrieves a geo routing policy by tenant ID
func (r *PostgresRepository) GetGeoRoutingPolicy(ctx context.Context, tenantID uuid.UUID) (*GeoRoutingPolicy, error) {
	query := `
		SELECT id, tenant_id, name, strategy, data_residency, preferred_regions, failover_order, weights, active, created_at
		FROM geo_routing_policies WHERE tenant_id = $1 AND active = true LIMIT 1
	`
	var policy GeoRoutingPolicy
	var dataRes, pref, fo, w []byte
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(
		&policy.ID, &policy.TenantID, &policy.Name, &policy.Strategy,
		&dataRes, &pref, &fo, &w, &policy.Active, &policy.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal(dataRes, &policy.DataResidencyReq)
	json.Unmarshal(pref, &policy.PreferredRegions)
	json.Unmarshal(fo, &policy.FailoverOrder)
	json.Unmarshal(w, &policy.Weights)
	return &policy, nil
}

// GetGeoRoutingPolicyByID retrieves a geo routing policy by its ID
func (r *PostgresRepository) GetGeoRoutingPolicyByID(ctx context.Context, id uuid.UUID) (*GeoRoutingPolicy, error) {
	query := `
		SELECT id, tenant_id, name, strategy, data_residency, preferred_regions, failover_order, weights, active, created_at
		FROM geo_routing_policies WHERE id = $1
	`
	var policy GeoRoutingPolicy
	var dataRes, pref, fo, w []byte
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&policy.ID, &policy.TenantID, &policy.Name, &policy.Strategy,
		&dataRes, &pref, &fo, &w, &policy.Active, &policy.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal(dataRes, &policy.DataResidencyReq)
	json.Unmarshal(pref, &policy.PreferredRegions)
	json.Unmarshal(fo, &policy.FailoverOrder)
	json.Unmarshal(w, &policy.Weights)
	return &policy, nil
}

// UpdateGeoRoutingPolicy updates a geo routing policy
func (r *PostgresRepository) UpdateGeoRoutingPolicy(ctx context.Context, policy *GeoRoutingPolicy) error {
	dataResJSON, _ := json.Marshal(policy.DataResidencyReq)
	prefJSON, _ := json.Marshal(policy.PreferredRegions)
	foJSON, _ := json.Marshal(policy.FailoverOrder)
	wJSON, _ := json.Marshal(policy.Weights)

	query := `
		UPDATE geo_routing_policies SET name=$1, strategy=$2, data_residency=$3, preferred_regions=$4,
		failover_order=$5, weights=$6, active=$7
		WHERE id = $8
	`
	_, err := r.db.ExecContext(ctx, query,
		policy.Name, policy.Strategy, dataResJSON, prefJSON, foJSON, wJSON, policy.Active, policy.ID,
	)
	return err
}

// ListGeoRoutingPolicies lists geo routing policies for a tenant
func (r *PostgresRepository) ListGeoRoutingPolicies(ctx context.Context, tenantID uuid.UUID) ([]GeoRoutingPolicy, error) {
	query := `
		SELECT id, tenant_id, name, strategy, data_residency, preferred_regions, failover_order, weights, active, created_at
		FROM geo_routing_policies WHERE tenant_id = $1 ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []GeoRoutingPolicy
	for rows.Next() {
		var policy GeoRoutingPolicy
		var dataRes, pref, fo, w []byte
		if err := rows.Scan(
			&policy.ID, &policy.TenantID, &policy.Name, &policy.Strategy,
			&dataRes, &pref, &fo, &w, &policy.Active, &policy.CreatedAt,
		); err != nil {
			return nil, err
		}
		json.Unmarshal(dataRes, &policy.DataResidencyReq)
		json.Unmarshal(pref, &policy.PreferredRegions)
		json.Unmarshal(fo, &policy.FailoverOrder)
		json.Unmarshal(w, &policy.Weights)
		policies = append(policies, policy)
	}
	return policies, rows.Err()
}

// GetEndpointRegionConfig retrieves endpoint region config
func (r *PostgresRepository) GetEndpointRegionConfig(ctx context.Context, endpointID uuid.UUID) (*EndpointRegionConfig, error) {
	query := `
		SELECT endpoint_id, primary_region, failover_regions, data_residency
		FROM endpoint_region_configs WHERE endpoint_id = $1
	`
	var config EndpointRegionConfig
	var foRegions []byte
	err := r.db.QueryRowContext(ctx, query, endpointID).Scan(
		&config.EndpointID, &config.PrimaryRegion, &foRegions, &config.DataResidencyRq,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal(foRegions, &config.FailoverRegions)
	return &config, nil
}

// SaveEndpointRegionConfig saves endpoint region config (upsert)
func (r *PostgresRepository) SaveEndpointRegionConfig(ctx context.Context, config *EndpointRegionConfig) error {
	foJSON, _ := json.Marshal(config.FailoverRegions)
	query := `
		INSERT INTO endpoint_region_configs (endpoint_id, primary_region, failover_regions, data_residency)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (endpoint_id) DO UPDATE SET
			primary_region = EXCLUDED.primary_region,
			failover_regions = EXCLUDED.failover_regions,
			data_residency = EXCLUDED.data_residency
	`
	_, err := r.db.ExecContext(ctx, query,
		config.EndpointID, config.PrimaryRegion, foJSON, config.DataResidencyRq,
	)
	return err
}

// RecordGeoRoutingDecision records a geo routing decision
func (r *PostgresRepository) RecordGeoRoutingDecision(ctx context.Context, decision *GeoRoutingDecision) error {
	altJSON, _ := json.Marshal(decision.AlternativeRegions)
	query := `
		INSERT INTO geo_routing_decisions (event_id, selected_region, reason, estimated_latency_ms, alternative_regions)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.ExecContext(ctx, query,
		decision.EventID, decision.SelectedRegion, decision.Reason,
		decision.Latency, altJSON,
	)
	return err
}

// ListGeoRoutingDecisions lists recent geo routing decisions
func (r *PostgresRepository) ListGeoRoutingDecisions(ctx context.Context, tenantID uuid.UUID, limit int) ([]GeoRoutingDecision, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT event_id, selected_region, reason, estimated_latency_ms, alternative_regions
		FROM geo_routing_decisions ORDER BY event_id DESC LIMIT $1
	`
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var decisions []GeoRoutingDecision
	for rows.Next() {
		var d GeoRoutingDecision
		var alt []byte
		if err := rows.Scan(&d.EventID, &d.SelectedRegion, &d.Reason, &d.Latency, &alt); err != nil {
			return nil, err
		}
		json.Unmarshal(alt, &d.AlternativeRegions)
		decisions = append(decisions, d)
	}
	return decisions, rows.Err()
}
