package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"webhook-platform/pkg/database"
	"webhook-platform/pkg/models"
)

// EdgeFunctionsRepository handles persistence for edge functions
type EdgeFunctionsRepository interface {
	// Functions
	CreateFunction(ctx context.Context, fn *models.EdgeFunction) error
	GetFunction(ctx context.Context, id uuid.UUID) (*models.EdgeFunction, error)
	GetFunctionByName(ctx context.Context, tenantID uuid.UUID, name string) (*models.EdgeFunction, error)
	GetFunctionsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.EdgeFunction, error)
	GetActiveFunctions(ctx context.Context, tenantID uuid.UUID) ([]*models.EdgeFunction, error)
	UpdateFunction(ctx context.Context, fn *models.EdgeFunction) error
	UpdateFunctionStatus(ctx context.Context, id uuid.UUID, status string) error
	DeleteFunction(ctx context.Context, id uuid.UUID) error

	// Versions
	CreateVersion(ctx context.Context, version *models.EdgeFunctionVersion) error
	GetVersions(ctx context.Context, functionID uuid.UUID) ([]*models.EdgeFunctionVersion, error)
	GetVersion(ctx context.Context, functionID uuid.UUID, version int) (*models.EdgeFunctionVersion, error)

	// Locations
	GetAllLocations(ctx context.Context) ([]*models.EdgeLocation, error)
	GetLocation(ctx context.Context, id uuid.UUID) (*models.EdgeLocation, error)
	GetLocationByCode(ctx context.Context, code string) (*models.EdgeLocation, error)
	GetActiveLocations(ctx context.Context) ([]*models.EdgeLocation, error)

	// Deployments
	CreateDeployment(ctx context.Context, deployment *models.EdgeFunctionDeployment) error
	GetDeployment(ctx context.Context, id uuid.UUID) (*models.EdgeFunctionDeployment, error)
	GetDeploymentsByFunction(ctx context.Context, functionID uuid.UUID) ([]*models.EdgeFunctionDeployment, error)
	GetActiveDeployment(ctx context.Context, functionID, locationID uuid.UUID) (*models.EdgeFunctionDeployment, error)
	UpdateDeploymentStatus(ctx context.Context, id uuid.UUID, status string, deploymentURL string) error
	UpdateDeploymentHealth(ctx context.Context, id uuid.UUID, healthStatus string) error
	SetDeploymentError(ctx context.Context, id uuid.UUID, errorMsg string) error

	// Triggers
	CreateTrigger(ctx context.Context, trigger *models.EdgeFunctionTrigger) error
	GetTrigger(ctx context.Context, id uuid.UUID) (*models.EdgeFunctionTrigger, error)
	GetTriggersByFunction(ctx context.Context, functionID uuid.UUID) ([]*models.EdgeFunctionTrigger, error)
	GetMatchingTriggers(ctx context.Context, tenantID uuid.UUID, triggerType, eventType string, endpointID uuid.UUID) ([]*models.EdgeFunctionTrigger, error)
	UpdateTrigger(ctx context.Context, trigger *models.EdgeFunctionTrigger) error
	DeleteTrigger(ctx context.Context, id uuid.UUID) error

	// Invocations
	CreateInvocation(ctx context.Context, invocation *models.EdgeFunctionInvocation) error
	GetInvocation(ctx context.Context, id uuid.UUID) (*models.EdgeFunctionInvocation, error)
	GetInvocationsByFunction(ctx context.Context, functionID uuid.UUID, limit int) ([]*models.EdgeFunctionInvocation, error)
	GetRecentInvocations(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.EdgeFunctionInvocation, error)
	CompleteInvocation(ctx context.Context, id uuid.UUID, status string, durationMs, memoryUsed int, errorMsg string) error

	// Metrics
	CreateOrUpdateMetrics(ctx context.Context, metrics *models.EdgeFunctionMetrics) error
	GetMetrics(ctx context.Context, functionID uuid.UUID, since time.Time) ([]*models.EdgeFunctionMetrics, error)

	// Secrets
	CreateSecret(ctx context.Context, secret *models.EdgeFunctionSecret) error
	GetSecrets(ctx context.Context, functionID uuid.UUID) ([]*models.EdgeFunctionSecret, error)
	DeleteSecret(ctx context.Context, id uuid.UUID) error

	// Tests
	CreateTest(ctx context.Context, test *models.EdgeFunctionTest) error
	GetTests(ctx context.Context, functionID uuid.UUID) ([]*models.EdgeFunctionTest, error)

	// Dashboard
	CountFunctions(ctx context.Context, tenantID uuid.UUID) (int, error)
	CountActiveFunctions(ctx context.Context, tenantID uuid.UUID) (int, error)
	CountDeployments(ctx context.Context, tenantID uuid.UUID) (int, error)
	CountInvocations(ctx context.Context, tenantID uuid.UUID, since time.Time) (int64, error)
	GetErrorRate(ctx context.Context, tenantID uuid.UUID, since time.Time) (float64, error)
}

// PostgresEdgeFunctionsRepository implements EdgeFunctionsRepository with PostgreSQL
type PostgresEdgeFunctionsRepository struct {
	pool *pgxpool.Pool
}

// NewEdgeFunctionsRepository creates a new repository
func NewEdgeFunctionsRepository(pool *pgxpool.Pool) EdgeFunctionsRepository {
	return &PostgresEdgeFunctionsRepository{pool: pool}
}

// CreateFunction creates an edge function
func (r *PostgresEdgeFunctionsRepository) CreateFunction(ctx context.Context, fn *models.EdgeFunction) error {
	envVarsJSON, _ := json.Marshal(fn.EnvironmentVars)
	metadataJSON, _ := json.Marshal(fn.Metadata)

	query := `
		INSERT INTO edge_functions (tenant_id, name, description, runtime, code, entry_point,
		                            status, timeout_ms, memory_mb, environment_vars, dependencies, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, version, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		fn.TenantID, fn.Name, fn.Description, fn.Runtime, fn.Code, fn.EntryPoint,
		fn.Status, fn.TimeoutMs, fn.MemoryMb, envVarsJSON, database.StringArray(fn.Dependencies), metadataJSON,
	).Scan(&fn.ID, &fn.Version, &fn.CreatedAt, &fn.UpdatedAt)
}

// GetFunction retrieves a function by ID
func (r *PostgresEdgeFunctionsRepository) GetFunction(ctx context.Context, id uuid.UUID) (*models.EdgeFunction, error) {
	query := `
		SELECT id, tenant_id, name, description, runtime, code, entry_point, version, status,
		       timeout_ms, memory_mb, environment_vars, dependencies, metadata, created_at, updated_at, deployed_at
		FROM edge_functions WHERE id = $1`

	fn := &models.EdgeFunction{}
	var envVarsJSON, metadataJSON []byte
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&fn.ID, &fn.TenantID, &fn.Name, &fn.Description, &fn.Runtime, &fn.Code, &fn.EntryPoint,
		&fn.Version, &fn.Status, &fn.TimeoutMs, &fn.MemoryMb, &envVarsJSON, (*database.StringArray)(&fn.Dependencies),
		&metadataJSON, &fn.CreatedAt, &fn.UpdatedAt, &fn.DeployedAt,
	)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(envVarsJSON, &fn.EnvironmentVars)
	json.Unmarshal(metadataJSON, &fn.Metadata)

	return fn, nil
}

// GetFunctionByName retrieves a function by name
func (r *PostgresEdgeFunctionsRepository) GetFunctionByName(ctx context.Context, tenantID uuid.UUID, name string) (*models.EdgeFunction, error) {
	query := `
		SELECT id, tenant_id, name, description, runtime, code, entry_point, version, status,
		       timeout_ms, memory_mb, environment_vars, dependencies, metadata, created_at, updated_at, deployed_at
		FROM edge_functions WHERE tenant_id = $1 AND name = $2`

	fn := &models.EdgeFunction{}
	var envVarsJSON, metadataJSON []byte
	err := r.pool.QueryRow(ctx, query, tenantID, name).Scan(
		&fn.ID, &fn.TenantID, &fn.Name, &fn.Description, &fn.Runtime, &fn.Code, &fn.EntryPoint,
		&fn.Version, &fn.Status, &fn.TimeoutMs, &fn.MemoryMb, &envVarsJSON, (*database.StringArray)(&fn.Dependencies),
		&metadataJSON, &fn.CreatedAt, &fn.UpdatedAt, &fn.DeployedAt,
	)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(envVarsJSON, &fn.EnvironmentVars)
	json.Unmarshal(metadataJSON, &fn.Metadata)

	return fn, nil
}

// GetFunctionsByTenant retrieves functions for a tenant
func (r *PostgresEdgeFunctionsRepository) GetFunctionsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.EdgeFunction, error) {
	query := `
		SELECT id, tenant_id, name, description, runtime, code, entry_point, version, status,
		       timeout_ms, memory_mb, environment_vars, dependencies, metadata, created_at, updated_at, deployed_at
		FROM edge_functions WHERE tenant_id = $1 ORDER BY name`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var functions []*models.EdgeFunction
	for rows.Next() {
		fn := &models.EdgeFunction{}
		var envVarsJSON, metadataJSON []byte
		if err := rows.Scan(
			&fn.ID, &fn.TenantID, &fn.Name, &fn.Description, &fn.Runtime, &fn.Code, &fn.EntryPoint,
			&fn.Version, &fn.Status, &fn.TimeoutMs, &fn.MemoryMb, &envVarsJSON, (*database.StringArray)(&fn.Dependencies),
			&metadataJSON, &fn.CreatedAt, &fn.UpdatedAt, &fn.DeployedAt,
		); err != nil {
			return nil, err
		}
		json.Unmarshal(envVarsJSON, &fn.EnvironmentVars)
		json.Unmarshal(metadataJSON, &fn.Metadata)
		functions = append(functions, fn)
	}

	return functions, nil
}

// GetActiveFunctions retrieves active functions for a tenant
func (r *PostgresEdgeFunctionsRepository) GetActiveFunctions(ctx context.Context, tenantID uuid.UUID) ([]*models.EdgeFunction, error) {
	query := `
		SELECT id, tenant_id, name, description, runtime, code, entry_point, version, status,
		       timeout_ms, memory_mb, environment_vars, dependencies, metadata, created_at, updated_at, deployed_at
		FROM edge_functions WHERE tenant_id = $1 AND status = 'active' ORDER BY name`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var functions []*models.EdgeFunction
	for rows.Next() {
		fn := &models.EdgeFunction{}
		var envVarsJSON, metadataJSON []byte
		if err := rows.Scan(
			&fn.ID, &fn.TenantID, &fn.Name, &fn.Description, &fn.Runtime, &fn.Code, &fn.EntryPoint,
			&fn.Version, &fn.Status, &fn.TimeoutMs, &fn.MemoryMb, &envVarsJSON, (*database.StringArray)(&fn.Dependencies),
			&metadataJSON, &fn.CreatedAt, &fn.UpdatedAt, &fn.DeployedAt,
		); err != nil {
			return nil, err
		}
		json.Unmarshal(envVarsJSON, &fn.EnvironmentVars)
		json.Unmarshal(metadataJSON, &fn.Metadata)
		functions = append(functions, fn)
	}

	return functions, nil
}

// UpdateFunction updates a function
func (r *PostgresEdgeFunctionsRepository) UpdateFunction(ctx context.Context, fn *models.EdgeFunction) error {
	envVarsJSON, _ := json.Marshal(fn.EnvironmentVars)

	query := `
		UPDATE edge_functions 
		SET code = $2, entry_point = $3, timeout_ms = $4, memory_mb = $5, 
		    environment_vars = $6, dependencies = $7, version = version + 1, updated_at = NOW()
		WHERE id = $1
		RETURNING version, updated_at`

	return r.pool.QueryRow(ctx, query,
		fn.ID, fn.Code, fn.EntryPoint, fn.TimeoutMs, fn.MemoryMb,
		envVarsJSON, database.StringArray(fn.Dependencies),
	).Scan(&fn.Version, &fn.UpdatedAt)
}

// UpdateFunctionStatus updates function status
func (r *PostgresEdgeFunctionsRepository) UpdateFunctionStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `UPDATE edge_functions SET status = $2, updated_at = NOW() WHERE id = $1`
	if status == models.FunctionStatusActive {
		query = `UPDATE edge_functions SET status = $2, deployed_at = NOW(), updated_at = NOW() WHERE id = $1`
	}
	_, err := r.pool.Exec(ctx, query, id, status)
	return err
}

// DeleteFunction deletes a function
func (r *PostgresEdgeFunctionsRepository) DeleteFunction(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM edge_functions WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// CreateVersion creates a function version
func (r *PostgresEdgeFunctionsRepository) CreateVersion(ctx context.Context, version *models.EdgeFunctionVersion) error {
	query := `
		INSERT INTO edge_function_versions (function_id, version, code, entry_point, code_hash, change_log, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`

	return r.pool.QueryRow(ctx, query,
		version.FunctionID, version.Version, version.Code, version.EntryPoint,
		version.CodeHash, version.ChangeLog, version.CreatedBy,
	).Scan(&version.ID, &version.CreatedAt)
}

// GetVersions retrieves function versions
func (r *PostgresEdgeFunctionsRepository) GetVersions(ctx context.Context, functionID uuid.UUID) ([]*models.EdgeFunctionVersion, error) {
	query := `
		SELECT id, function_id, version, code, entry_point, code_hash, change_log, created_by, created_at
		FROM edge_function_versions WHERE function_id = $1 ORDER BY version DESC`

	rows, err := r.pool.Query(ctx, query, functionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []*models.EdgeFunctionVersion
	for rows.Next() {
		v := &models.EdgeFunctionVersion{}
		if err := rows.Scan(
			&v.ID, &v.FunctionID, &v.Version, &v.Code, &v.EntryPoint,
			&v.CodeHash, &v.ChangeLog, &v.CreatedBy, &v.CreatedAt,
		); err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}

	return versions, nil
}

// GetVersion retrieves a specific function version
func (r *PostgresEdgeFunctionsRepository) GetVersion(ctx context.Context, functionID uuid.UUID, version int) (*models.EdgeFunctionVersion, error) {
	query := `
		SELECT id, function_id, version, code, entry_point, code_hash, change_log, created_by, created_at
		FROM edge_function_versions WHERE function_id = $1 AND version = $2`

	v := &models.EdgeFunctionVersion{}
	err := r.pool.QueryRow(ctx, query, functionID, version).Scan(
		&v.ID, &v.FunctionID, &v.Version, &v.Code, &v.EntryPoint,
		&v.CodeHash, &v.ChangeLog, &v.CreatedBy, &v.CreatedAt,
	)
	return v, err
}

// GetAllLocations retrieves all edge locations
func (r *PostgresEdgeFunctionsRepository) GetAllLocations(ctx context.Context) ([]*models.EdgeLocation, error) {
	query := `
		SELECT id, name, code, region, provider, status, latency_ms, capacity, metadata, created_at
		FROM edge_locations ORDER BY name`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var locations []*models.EdgeLocation
	for rows.Next() {
		loc := &models.EdgeLocation{}
		var metadataJSON []byte
		if err := rows.Scan(
			&loc.ID, &loc.Name, &loc.Code, &loc.Region, &loc.Provider,
			&loc.Status, &loc.LatencyMs, &loc.Capacity, &metadataJSON, &loc.CreatedAt,
		); err != nil {
			return nil, err
		}
		json.Unmarshal(metadataJSON, &loc.Metadata)
		locations = append(locations, loc)
	}

	return locations, nil
}

// GetLocation retrieves a location by ID
func (r *PostgresEdgeFunctionsRepository) GetLocation(ctx context.Context, id uuid.UUID) (*models.EdgeLocation, error) {
	query := `
		SELECT id, name, code, region, provider, status, latency_ms, capacity, metadata, created_at
		FROM edge_locations WHERE id = $1`

	loc := &models.EdgeLocation{}
	var metadataJSON []byte
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&loc.ID, &loc.Name, &loc.Code, &loc.Region, &loc.Provider,
		&loc.Status, &loc.LatencyMs, &loc.Capacity, &metadataJSON, &loc.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(metadataJSON, &loc.Metadata)

	return loc, nil
}

// GetLocationByCode retrieves a location by code
func (r *PostgresEdgeFunctionsRepository) GetLocationByCode(ctx context.Context, code string) (*models.EdgeLocation, error) {
	query := `
		SELECT id, name, code, region, provider, status, latency_ms, capacity, metadata, created_at
		FROM edge_locations WHERE code = $1`

	loc := &models.EdgeLocation{}
	var metadataJSON []byte
	err := r.pool.QueryRow(ctx, query, code).Scan(
		&loc.ID, &loc.Name, &loc.Code, &loc.Region, &loc.Provider,
		&loc.Status, &loc.LatencyMs, &loc.Capacity, &metadataJSON, &loc.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(metadataJSON, &loc.Metadata)

	return loc, nil
}

// GetActiveLocations retrieves active edge locations
func (r *PostgresEdgeFunctionsRepository) GetActiveLocations(ctx context.Context) ([]*models.EdgeLocation, error) {
	query := `
		SELECT id, name, code, region, provider, status, latency_ms, capacity, metadata, created_at
		FROM edge_locations WHERE status = 'active' ORDER BY name`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var locations []*models.EdgeLocation
	for rows.Next() {
		loc := &models.EdgeLocation{}
		var metadataJSON []byte
		if err := rows.Scan(
			&loc.ID, &loc.Name, &loc.Code, &loc.Region, &loc.Provider,
			&loc.Status, &loc.LatencyMs, &loc.Capacity, &metadataJSON, &loc.CreatedAt,
		); err != nil {
			return nil, err
		}
		json.Unmarshal(metadataJSON, &loc.Metadata)
		locations = append(locations, loc)
	}

	return locations, nil
}

// CreateDeployment creates a function deployment
func (r *PostgresEdgeFunctionsRepository) CreateDeployment(ctx context.Context, deployment *models.EdgeFunctionDeployment) error {
	query := `
		INSERT INTO edge_function_deployments (function_id, location_id, version, status, health_status)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (function_id, location_id) DO UPDATE SET
		    version = EXCLUDED.version, status = EXCLUDED.status, updated_at = NOW()
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		deployment.FunctionID, deployment.LocationID, deployment.Version,
		deployment.Status, deployment.HealthStatus,
	).Scan(&deployment.ID, &deployment.CreatedAt, &deployment.UpdatedAt)
}

// GetDeployment retrieves a deployment
func (r *PostgresEdgeFunctionsRepository) GetDeployment(ctx context.Context, id uuid.UUID) (*models.EdgeFunctionDeployment, error) {
	query := `
		SELECT id, function_id, location_id, version, status, deployment_url, health_check_url,
		       last_health_check, health_status, error_message, deployed_at, created_at, updated_at
		FROM edge_function_deployments WHERE id = $1`

	d := &models.EdgeFunctionDeployment{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&d.ID, &d.FunctionID, &d.LocationID, &d.Version, &d.Status, &d.DeploymentURL,
		&d.HealthCheckURL, &d.LastHealthCheck, &d.HealthStatus, &d.ErrorMessage,
		&d.DeployedAt, &d.CreatedAt, &d.UpdatedAt,
	)
	return d, err
}

// GetDeploymentsByFunction retrieves deployments for a function
func (r *PostgresEdgeFunctionsRepository) GetDeploymentsByFunction(ctx context.Context, functionID uuid.UUID) ([]*models.EdgeFunctionDeployment, error) {
	query := `
		SELECT d.id, d.function_id, d.location_id, d.version, d.status, d.deployment_url, d.health_check_url,
		       d.last_health_check, d.health_status, d.error_message, d.deployed_at, d.created_at, d.updated_at,
		       l.id, l.name, l.code, l.region, l.provider, l.status, l.latency_ms, l.capacity, l.metadata, l.created_at
		FROM edge_function_deployments d
		JOIN edge_locations l ON d.location_id = l.id
		WHERE d.function_id = $1`

	rows, err := r.pool.Query(ctx, query, functionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deployments []*models.EdgeFunctionDeployment
	for rows.Next() {
		d := &models.EdgeFunctionDeployment{}
		loc := &models.EdgeLocation{}
		var metadataJSON []byte
		if err := rows.Scan(
			&d.ID, &d.FunctionID, &d.LocationID, &d.Version, &d.Status, &d.DeploymentURL,
			&d.HealthCheckURL, &d.LastHealthCheck, &d.HealthStatus, &d.ErrorMessage,
			&d.DeployedAt, &d.CreatedAt, &d.UpdatedAt,
			&loc.ID, &loc.Name, &loc.Code, &loc.Region, &loc.Provider,
			&loc.Status, &loc.LatencyMs, &loc.Capacity, &metadataJSON, &loc.CreatedAt,
		); err != nil {
			return nil, err
		}
		json.Unmarshal(metadataJSON, &loc.Metadata)
		d.Location = loc
		deployments = append(deployments, d)
	}

	return deployments, nil
}

// GetActiveDeployment retrieves active deployment for function at location
func (r *PostgresEdgeFunctionsRepository) GetActiveDeployment(ctx context.Context, functionID, locationID uuid.UUID) (*models.EdgeFunctionDeployment, error) {
	query := `
		SELECT id, function_id, location_id, version, status, deployment_url, health_check_url,
		       last_health_check, health_status, error_message, deployed_at, created_at, updated_at
		FROM edge_function_deployments 
		WHERE function_id = $1 AND location_id = $2 AND status = 'active'`

	d := &models.EdgeFunctionDeployment{}
	err := r.pool.QueryRow(ctx, query, functionID, locationID).Scan(
		&d.ID, &d.FunctionID, &d.LocationID, &d.Version, &d.Status, &d.DeploymentURL,
		&d.HealthCheckURL, &d.LastHealthCheck, &d.HealthStatus, &d.ErrorMessage,
		&d.DeployedAt, &d.CreatedAt, &d.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return d, err
}

// UpdateDeploymentStatus updates deployment status
func (r *PostgresEdgeFunctionsRepository) UpdateDeploymentStatus(ctx context.Context, id uuid.UUID, status string, deploymentURL string) error {
	query := `UPDATE edge_function_deployments SET status = $2, deployment_url = $3, deployed_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, status, deploymentURL)
	return err
}

// UpdateDeploymentHealth updates deployment health
func (r *PostgresEdgeFunctionsRepository) UpdateDeploymentHealth(ctx context.Context, id uuid.UUID, healthStatus string) error {
	query := `UPDATE edge_function_deployments SET health_status = $2, last_health_check = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, healthStatus)
	return err
}

// SetDeploymentError sets deployment error
func (r *PostgresEdgeFunctionsRepository) SetDeploymentError(ctx context.Context, id uuid.UUID, errorMsg string) error {
	query := `UPDATE edge_function_deployments SET status = 'failed', error_message = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, errorMsg)
	return err
}

// CreateTrigger creates a function trigger
func (r *PostgresEdgeFunctionsRepository) CreateTrigger(ctx context.Context, trigger *models.EdgeFunctionTrigger) error {
	conditionsJSON, _ := json.Marshal(trigger.Conditions)

	query := `
		INSERT INTO edge_function_triggers (function_id, trigger_type, event_types, endpoint_ids, conditions, priority, enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		trigger.FunctionID, trigger.TriggerType, database.StringArray(trigger.EventTypes),
		database.UUIDArray(trigger.EndpointIDs), conditionsJSON, trigger.Priority, trigger.Enabled,
	).Scan(&trigger.ID, &trigger.CreatedAt, &trigger.UpdatedAt)
}

// GetTrigger retrieves a trigger
func (r *PostgresEdgeFunctionsRepository) GetTrigger(ctx context.Context, id uuid.UUID) (*models.EdgeFunctionTrigger, error) {
	query := `
		SELECT id, function_id, trigger_type, event_types, endpoint_ids, conditions, priority, enabled, created_at, updated_at
		FROM edge_function_triggers WHERE id = $1`

	t := &models.EdgeFunctionTrigger{}
	var conditionsJSON []byte
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.FunctionID, &t.TriggerType, (*database.StringArray)(&t.EventTypes),
		(*database.UUIDArray)(&t.EndpointIDs), &conditionsJSON, &t.Priority, &t.Enabled,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(conditionsJSON, &t.Conditions)

	return t, nil
}

// GetTriggersByFunction retrieves triggers for a function
func (r *PostgresEdgeFunctionsRepository) GetTriggersByFunction(ctx context.Context, functionID uuid.UUID) ([]*models.EdgeFunctionTrigger, error) {
	query := `
		SELECT id, function_id, trigger_type, event_types, endpoint_ids, conditions, priority, enabled, created_at, updated_at
		FROM edge_function_triggers WHERE function_id = $1 ORDER BY priority`

	rows, err := r.pool.Query(ctx, query, functionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var triggers []*models.EdgeFunctionTrigger
	for rows.Next() {
		t := &models.EdgeFunctionTrigger{}
		var conditionsJSON []byte
		if err := rows.Scan(
			&t.ID, &t.FunctionID, &t.TriggerType, (*database.StringArray)(&t.EventTypes),
			(*database.UUIDArray)(&t.EndpointIDs), &conditionsJSON, &t.Priority, &t.Enabled,
			&t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		json.Unmarshal(conditionsJSON, &t.Conditions)
		triggers = append(triggers, t)
	}

	return triggers, nil
}

// GetMatchingTriggers retrieves triggers matching criteria
func (r *PostgresEdgeFunctionsRepository) GetMatchingTriggers(ctx context.Context, tenantID uuid.UUID, triggerType, eventType string, endpointID uuid.UUID) ([]*models.EdgeFunctionTrigger, error) {
	query := `
		SELECT t.id, t.function_id, t.trigger_type, t.event_types, t.endpoint_ids, t.conditions, t.priority, t.enabled, t.created_at, t.updated_at
		FROM edge_function_triggers t
		JOIN edge_functions f ON t.function_id = f.id
		WHERE f.tenant_id = $1 AND f.status = 'active' AND t.enabled = TRUE AND t.trigger_type = $2
		  AND ($3 = '' OR $3 = ANY(t.event_types) OR ARRAY_LENGTH(t.event_types, 1) IS NULL)
		  AND ($4 = '00000000-0000-0000-0000-000000000000' OR $4 = ANY(t.endpoint_ids) OR ARRAY_LENGTH(t.endpoint_ids, 1) IS NULL)
		ORDER BY t.priority`

	rows, err := r.pool.Query(ctx, query, tenantID, triggerType, eventType, endpointID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var triggers []*models.EdgeFunctionTrigger
	for rows.Next() {
		t := &models.EdgeFunctionTrigger{}
		var conditionsJSON []byte
		if err := rows.Scan(
			&t.ID, &t.FunctionID, &t.TriggerType, (*database.StringArray)(&t.EventTypes),
			(*database.UUIDArray)(&t.EndpointIDs), &conditionsJSON, &t.Priority, &t.Enabled,
			&t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		json.Unmarshal(conditionsJSON, &t.Conditions)
		triggers = append(triggers, t)
	}

	return triggers, nil
}

// UpdateTrigger updates a trigger
func (r *PostgresEdgeFunctionsRepository) UpdateTrigger(ctx context.Context, trigger *models.EdgeFunctionTrigger) error {
	conditionsJSON, _ := json.Marshal(trigger.Conditions)

	query := `
		UPDATE edge_function_triggers 
		SET trigger_type = $2, event_types = $3, endpoint_ids = $4, conditions = $5, priority = $6, enabled = $7, updated_at = NOW()
		WHERE id = $1`

	_, err := r.pool.Exec(ctx, query,
		trigger.ID, trigger.TriggerType, database.StringArray(trigger.EventTypes),
		database.UUIDArray(trigger.EndpointIDs), conditionsJSON, trigger.Priority, trigger.Enabled,
	)
	return err
}

// DeleteTrigger deletes a trigger
func (r *PostgresEdgeFunctionsRepository) DeleteTrigger(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM edge_function_triggers WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// CreateInvocation creates a function invocation
func (r *PostgresEdgeFunctionsRepository) CreateInvocation(ctx context.Context, invocation *models.EdgeFunctionInvocation) error {
	query := `
		INSERT INTO edge_function_invocations (function_id, deployment_id, trigger_id, tenant_id, event_id, 
		                                       endpoint_id, location_code, status, input_size_bytes, cold_start)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, started_at`

	return r.pool.QueryRow(ctx, query,
		invocation.FunctionID, invocation.DeploymentID, invocation.TriggerID, invocation.TenantID,
		invocation.EventID, invocation.EndpointID, invocation.LocationCode, invocation.Status,
		invocation.InputSizeBytes, invocation.ColdStart,
	).Scan(&invocation.ID, &invocation.StartedAt)
}

// GetInvocation retrieves an invocation
func (r *PostgresEdgeFunctionsRepository) GetInvocation(ctx context.Context, id uuid.UUID) (*models.EdgeFunctionInvocation, error) {
	query := `
		SELECT id, function_id, deployment_id, trigger_id, tenant_id, event_id, endpoint_id, location_code,
		       status, duration_ms, memory_used_mb, input_size_bytes, output_size_bytes, error_message,
		       cold_start, started_at, completed_at
		FROM edge_function_invocations WHERE id = $1`

	inv := &models.EdgeFunctionInvocation{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&inv.ID, &inv.FunctionID, &inv.DeploymentID, &inv.TriggerID, &inv.TenantID,
		&inv.EventID, &inv.EndpointID, &inv.LocationCode, &inv.Status, &inv.DurationMs,
		&inv.MemoryUsedMb, &inv.InputSizeBytes, &inv.OutputSizeBytes, &inv.ErrorMessage,
		&inv.ColdStart, &inv.StartedAt, &inv.CompletedAt,
	)
	return inv, err
}

// GetInvocationsByFunction retrieves invocations for a function
func (r *PostgresEdgeFunctionsRepository) GetInvocationsByFunction(ctx context.Context, functionID uuid.UUID, limit int) ([]*models.EdgeFunctionInvocation, error) {
	query := `
		SELECT id, function_id, deployment_id, trigger_id, tenant_id, event_id, endpoint_id, location_code,
		       status, duration_ms, memory_used_mb, input_size_bytes, output_size_bytes, error_message,
		       cold_start, started_at, completed_at
		FROM edge_function_invocations WHERE function_id = $1 ORDER BY started_at DESC LIMIT $2`

	rows, err := r.pool.Query(ctx, query, functionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invocations []*models.EdgeFunctionInvocation
	for rows.Next() {
		inv := &models.EdgeFunctionInvocation{}
		if err := rows.Scan(
			&inv.ID, &inv.FunctionID, &inv.DeploymentID, &inv.TriggerID, &inv.TenantID,
			&inv.EventID, &inv.EndpointID, &inv.LocationCode, &inv.Status, &inv.DurationMs,
			&inv.MemoryUsedMb, &inv.InputSizeBytes, &inv.OutputSizeBytes, &inv.ErrorMessage,
			&inv.ColdStart, &inv.StartedAt, &inv.CompletedAt,
		); err != nil {
			return nil, err
		}
		invocations = append(invocations, inv)
	}

	return invocations, nil
}

// GetRecentInvocations retrieves recent invocations for a tenant
func (r *PostgresEdgeFunctionsRepository) GetRecentInvocations(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.EdgeFunctionInvocation, error) {
	query := `
		SELECT id, function_id, deployment_id, trigger_id, tenant_id, event_id, endpoint_id, location_code,
		       status, duration_ms, memory_used_mb, input_size_bytes, output_size_bytes, error_message,
		       cold_start, started_at, completed_at
		FROM edge_function_invocations WHERE tenant_id = $1 ORDER BY started_at DESC LIMIT $2`

	rows, err := r.pool.Query(ctx, query, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invocations []*models.EdgeFunctionInvocation
	for rows.Next() {
		inv := &models.EdgeFunctionInvocation{}
		if err := rows.Scan(
			&inv.ID, &inv.FunctionID, &inv.DeploymentID, &inv.TriggerID, &inv.TenantID,
			&inv.EventID, &inv.EndpointID, &inv.LocationCode, &inv.Status, &inv.DurationMs,
			&inv.MemoryUsedMb, &inv.InputSizeBytes, &inv.OutputSizeBytes, &inv.ErrorMessage,
			&inv.ColdStart, &inv.StartedAt, &inv.CompletedAt,
		); err != nil {
			return nil, err
		}
		invocations = append(invocations, inv)
	}

	return invocations, nil
}

// CompleteInvocation completes an invocation
func (r *PostgresEdgeFunctionsRepository) CompleteInvocation(ctx context.Context, id uuid.UUID, status string, durationMs, memoryUsed int, errorMsg string) error {
	query := `
		UPDATE edge_function_invocations 
		SET status = $2, duration_ms = $3, memory_used_mb = $4, error_message = $5, completed_at = NOW()
		WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, status, durationMs, memoryUsed, errorMsg)
	return err
}

// CreateOrUpdateMetrics creates or updates function metrics
func (r *PostgresEdgeFunctionsRepository) CreateOrUpdateMetrics(ctx context.Context, metrics *models.EdgeFunctionMetrics) error {
	query := `
		INSERT INTO edge_function_metrics (function_id, location_id, period_start, period_end,
		    invocation_count, success_count, error_count, timeout_count, cold_start_count,
		    avg_duration_ms, p50_duration_ms, p99_duration_ms, avg_memory_mb, total_billed_ms)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (function_id, location_id, period_start) DO UPDATE SET
		    invocation_count = EXCLUDED.invocation_count,
		    success_count = EXCLUDED.success_count,
		    error_count = EXCLUDED.error_count,
		    avg_duration_ms = EXCLUDED.avg_duration_ms
		RETURNING id, created_at`

	return r.pool.QueryRow(ctx, query,
		metrics.FunctionID, metrics.LocationID, metrics.PeriodStart, metrics.PeriodEnd,
		metrics.InvocationCount, metrics.SuccessCount, metrics.ErrorCount, metrics.TimeoutCount,
		metrics.ColdStartCount, metrics.AvgDurationMs, metrics.P50DurationMs, metrics.P99DurationMs,
		metrics.AvgMemoryMb, metrics.TotalBilledMs,
	).Scan(&metrics.ID, &metrics.CreatedAt)
}

// GetMetrics retrieves function metrics
func (r *PostgresEdgeFunctionsRepository) GetMetrics(ctx context.Context, functionID uuid.UUID, since time.Time) ([]*models.EdgeFunctionMetrics, error) {
	query := `
		SELECT id, function_id, location_id, period_start, period_end, invocation_count, success_count,
		       error_count, timeout_count, cold_start_count, avg_duration_ms, p50_duration_ms, p99_duration_ms,
		       avg_memory_mb, total_billed_ms, created_at
		FROM edge_function_metrics WHERE function_id = $1 AND period_start >= $2 ORDER BY period_start`

	rows, err := r.pool.Query(ctx, query, functionID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []*models.EdgeFunctionMetrics
	for rows.Next() {
		m := &models.EdgeFunctionMetrics{}
		if err := rows.Scan(
			&m.ID, &m.FunctionID, &m.LocationID, &m.PeriodStart, &m.PeriodEnd,
			&m.InvocationCount, &m.SuccessCount, &m.ErrorCount, &m.TimeoutCount,
			&m.ColdStartCount, &m.AvgDurationMs, &m.P50DurationMs, &m.P99DurationMs,
			&m.AvgMemoryMb, &m.TotalBilledMs, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}

	return metrics, nil
}

// CreateSecret creates a function secret
func (r *PostgresEdgeFunctionsRepository) CreateSecret(ctx context.Context, secret *models.EdgeFunctionSecret) error {
	query := `
		INSERT INTO edge_function_secrets (function_id, name, encrypted_value)
		VALUES ($1, $2, $3)
		ON CONFLICT (function_id, name) DO UPDATE SET encrypted_value = EXCLUDED.encrypted_value, updated_at = NOW()
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		secret.FunctionID, secret.Name, secret.EncryptedValue,
	).Scan(&secret.ID, &secret.CreatedAt, &secret.UpdatedAt)
}

// GetSecrets retrieves function secrets
func (r *PostgresEdgeFunctionsRepository) GetSecrets(ctx context.Context, functionID uuid.UUID) ([]*models.EdgeFunctionSecret, error) {
	query := `
		SELECT id, function_id, name, encrypted_value, created_at, updated_at
		FROM edge_function_secrets WHERE function_id = $1`

	rows, err := r.pool.Query(ctx, query, functionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var secrets []*models.EdgeFunctionSecret
	for rows.Next() {
		s := &models.EdgeFunctionSecret{}
		if err := rows.Scan(&s.ID, &s.FunctionID, &s.Name, &s.EncryptedValue, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		secrets = append(secrets, s)
	}

	return secrets, nil
}

// DeleteSecret deletes a secret
func (r *PostgresEdgeFunctionsRepository) DeleteSecret(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM edge_function_secrets WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// CreateTest creates a function test
func (r *PostgresEdgeFunctionsRepository) CreateTest(ctx context.Context, test *models.EdgeFunctionTest) error {
	inputJSON, _ := json.Marshal(test.InputPayload)
	expectedJSON, _ := json.Marshal(test.ExpectedOutput)
	actualJSON, _ := json.Marshal(test.ActualOutput)

	query := `
		INSERT INTO edge_function_tests (function_id, test_name, input_payload, expected_output, actual_output, passed, duration_ms, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, executed_at`

	return r.pool.QueryRow(ctx, query,
		test.FunctionID, test.TestName, inputJSON, expectedJSON, actualJSON,
		test.Passed, test.DurationMs, test.ErrorMessage,
	).Scan(&test.ID, &test.ExecutedAt)
}

// GetTests retrieves function tests
func (r *PostgresEdgeFunctionsRepository) GetTests(ctx context.Context, functionID uuid.UUID) ([]*models.EdgeFunctionTest, error) {
	query := `
		SELECT id, function_id, test_name, input_payload, expected_output, actual_output, passed, duration_ms, error_message, executed_at
		FROM edge_function_tests WHERE function_id = $1 ORDER BY executed_at DESC`

	rows, err := r.pool.Query(ctx, query, functionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tests []*models.EdgeFunctionTest
	for rows.Next() {
		t := &models.EdgeFunctionTest{}
		var inputJSON, expectedJSON, actualJSON []byte
		if err := rows.Scan(
			&t.ID, &t.FunctionID, &t.TestName, &inputJSON, &expectedJSON, &actualJSON,
			&t.Passed, &t.DurationMs, &t.ErrorMessage, &t.ExecutedAt,
		); err != nil {
			return nil, err
		}
		json.Unmarshal(inputJSON, &t.InputPayload)
		json.Unmarshal(expectedJSON, &t.ExpectedOutput)
		json.Unmarshal(actualJSON, &t.ActualOutput)
		tests = append(tests, t)
	}

	return tests, nil
}

// CountFunctions counts functions for a tenant
func (r *PostgresEdgeFunctionsRepository) CountFunctions(ctx context.Context, tenantID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM edge_functions WHERE tenant_id = $1`, tenantID).Scan(&count)
	return count, err
}

// CountActiveFunctions counts active functions for a tenant
func (r *PostgresEdgeFunctionsRepository) CountActiveFunctions(ctx context.Context, tenantID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM edge_functions WHERE tenant_id = $1 AND status = 'active'`, tenantID).Scan(&count)
	return count, err
}

// CountDeployments counts deployments for a tenant
func (r *PostgresEdgeFunctionsRepository) CountDeployments(ctx context.Context, tenantID uuid.UUID) (int, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM edge_function_deployments d
		JOIN edge_functions f ON d.function_id = f.id
		WHERE f.tenant_id = $1 AND d.status = 'active'`
	err := r.pool.QueryRow(ctx, query, tenantID).Scan(&count)
	return count, err
}

// CountInvocations counts invocations for a tenant
func (r *PostgresEdgeFunctionsRepository) CountInvocations(ctx context.Context, tenantID uuid.UUID, since time.Time) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM edge_function_invocations WHERE tenant_id = $1 AND started_at >= $2`, tenantID, since).Scan(&count)
	return count, err
}

// GetErrorRate gets error rate for a tenant
func (r *PostgresEdgeFunctionsRepository) GetErrorRate(ctx context.Context, tenantID uuid.UUID, since time.Time) (float64, error) {
	var total, errors int64
	r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM edge_function_invocations WHERE tenant_id = $1 AND started_at >= $2`, tenantID, since).Scan(&total)
	r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM edge_function_invocations WHERE tenant_id = $1 AND started_at >= $2 AND status = 'error'`, tenantID, since).Scan(&errors)

	if total == 0 {
		return 0, nil
	}
	return float64(errors) / float64(total) * 100, nil
}
