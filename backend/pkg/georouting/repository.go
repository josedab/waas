package georouting

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"webhook-platform/pkg/database"
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

	if err == sql.ErrNoRows {
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

	if err == sql.ErrNoRows {
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

	if err == sql.ErrNoRows {
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
