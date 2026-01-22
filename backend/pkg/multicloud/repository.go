package multicloud

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	"webhook-platform/pkg/database"
)

// Repository handles multi-cloud persistence
type Repository struct {
	db *database.DB
}

// NewRepository creates a new repository
func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

// CreateConnector creates a new cloud connector
func (r *Repository) CreateConnector(ctx context.Context, c *Connector) error {
	configJSON, err := json.Marshal(c.Config)
	if err != nil {
		return err
	}

	tagsJSON, _ := json.Marshal(c.Tags)

	query := `
		INSERT INTO cloud_connectors (tenant_id, name, description, provider, config_encrypted, tags)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, status, created_at, updated_at`

	return r.db.Pool.QueryRow(ctx, query,
		c.TenantID, c.Name, c.Description, c.Provider, configJSON, tagsJSON,
	).Scan(&c.ID, &c.Status, &c.CreatedAt, &c.UpdatedAt)
}

// GetConnector retrieves a connector by ID
func (r *Repository) GetConnector(ctx context.Context, id string) (*Connector, error) {
	var c Connector
	var configJSON, tagsJSON []byte

	query := `
		SELECT id, tenant_id, name, description, provider, config_encrypted,
			status, last_health_check, health_status, error_message, tags,
			created_at, updated_at
		FROM cloud_connectors
		WHERE id = $1`

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.TenantID, &c.Name, &c.Description, &c.Provider, &configJSON,
		&c.Status, &c.LastHealthCheck, &c.HealthStatus, &c.ErrorMessage, &tagsJSON,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrConnectorNotFound
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(configJSON, &c.Config)
	json.Unmarshal(tagsJSON, &c.Tags)

	return &c, nil
}

// ListConnectors lists connectors for a tenant
func (r *Repository) ListConnectors(ctx context.Context, tenantID string, provider *string) ([]Connector, error) {
	query := `
		SELECT id, tenant_id, name, description, provider, config_encrypted,
			status, last_health_check, health_status, error_message, tags,
			created_at, updated_at
		FROM cloud_connectors
		WHERE tenant_id = $1 AND ($2::text IS NULL OR provider = $2)
		ORDER BY created_at DESC`

	rows, err := r.db.Pool.Query(ctx, query, tenantID, provider)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var connectors []Connector
	for rows.Next() {
		var c Connector
		var configJSON, tagsJSON []byte
		err := rows.Scan(
			&c.ID, &c.TenantID, &c.Name, &c.Description, &c.Provider, &configJSON,
			&c.Status, &c.LastHealthCheck, &c.HealthStatus, &c.ErrorMessage, &tagsJSON,
			&c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		json.Unmarshal(configJSON, &c.Config)
		json.Unmarshal(tagsJSON, &c.Tags)
		connectors = append(connectors, c)
	}

	return connectors, nil
}

// UpdateConnectorStatus updates connector health status
func (r *Repository) UpdateConnectorStatus(ctx context.Context, id string, status, healthStatus, errorMsg string) error {
	query := `
		UPDATE cloud_connectors
		SET status = $2, health_status = $3, error_message = $4, 
			last_health_check = NOW(), updated_at = NOW()
		WHERE id = $1`

	_, err := r.db.Pool.Exec(ctx, query, id, status, healthStatus, errorMsg)
	return err
}

// DeleteConnector deletes a connector
func (r *Repository) DeleteConnector(ctx context.Context, id string) error {
	_, err := r.db.Pool.Exec(ctx, "DELETE FROM cloud_connectors WHERE id = $1", id)
	return err
}

// CreateRoute creates a new connector route
func (r *Repository) CreateRoute(ctx context.Context, route *Route) error {
	sourceJSON, _ := json.Marshal(route.SourceFilter)
	destJSON, _ := json.Marshal(route.DestinationConfig)

	query := `
		INSERT INTO connector_routes (
			tenant_id, connector_id, name, description, source_filter, destination_config,
			transform_enabled, transform_script, is_active, batch_enabled, batch_size, batch_window_seconds
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at`

	return r.db.Pool.QueryRow(ctx, query,
		route.TenantID, route.ConnectorID, route.Name, route.Description,
		sourceJSON, destJSON, route.TransformEnabled, route.TransformScript,
		route.IsActive, route.BatchEnabled, route.BatchSize, route.BatchWindowSec,
	).Scan(&route.ID, &route.CreatedAt, &route.UpdatedAt)
}

// GetRoute retrieves a route by ID
func (r *Repository) GetRoute(ctx context.Context, id string) (*Route, error) {
	var route Route
	var sourceJSON, destJSON []byte

	query := `
		SELECT id, tenant_id, connector_id, name, description, source_filter, destination_config,
			transform_enabled, transform_script, is_active, batch_enabled, batch_size, batch_window_seconds,
			created_at, updated_at
		FROM connector_routes
		WHERE id = $1`

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&route.ID, &route.TenantID, &route.ConnectorID, &route.Name, &route.Description,
		&sourceJSON, &destJSON, &route.TransformEnabled, &route.TransformScript,
		&route.IsActive, &route.BatchEnabled, &route.BatchSize, &route.BatchWindowSec,
		&route.CreatedAt, &route.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(sourceJSON, &route.SourceFilter)
	json.Unmarshal(destJSON, &route.DestinationConfig)

	return &route, nil
}

// ListRoutes lists routes for a connector
func (r *Repository) ListRoutes(ctx context.Context, connectorID string) ([]Route, error) {
	query := `
		SELECT id, tenant_id, connector_id, name, description, source_filter, destination_config,
			transform_enabled, transform_script, is_active, batch_enabled, batch_size, batch_window_seconds,
			created_at, updated_at
		FROM connector_routes
		WHERE connector_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.Pool.Query(ctx, query, connectorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routes []Route
	for rows.Next() {
		var route Route
		var sourceJSON, destJSON []byte
		err := rows.Scan(
			&route.ID, &route.TenantID, &route.ConnectorID, &route.Name, &route.Description,
			&sourceJSON, &destJSON, &route.TransformEnabled, &route.TransformScript,
			&route.IsActive, &route.BatchEnabled, &route.BatchSize, &route.BatchWindowSec,
			&route.CreatedAt, &route.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		json.Unmarshal(sourceJSON, &route.SourceFilter)
		json.Unmarshal(destJSON, &route.DestinationConfig)
		routes = append(routes, route)
	}

	return routes, nil
}

// SaveCloudDelivery saves a cloud delivery record
func (r *Repository) SaveCloudDelivery(ctx context.Context, d *CloudDelivery) error {
	query := `
		INSERT INTO cloud_deliveries (
			tenant_id, connector_id, route_id, original_delivery_id, event_type,
			cloud_message_id, cloud_request_id, payload_hash, payload_size_bytes,
			status, http_status_code, error_code, error_message, sent_at, acknowledged_at, latency_ms
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id, created_at`

	return r.db.Pool.QueryRow(ctx, query,
		d.TenantID, d.ConnectorID, d.RouteID, d.OriginalDeliveryID, d.EventType,
		d.CloudMessageID, d.CloudRequestID, d.PayloadHash, d.PayloadSizeBytes,
		d.Status, d.HTTPStatusCode, d.ErrorCode, d.ErrorMessage,
		d.SentAt, d.AcknowledgedAt, d.LatencyMs,
	).Scan(&d.ID, &d.CreatedAt)
}

// GetCloudDeliveries retrieves cloud deliveries with filters
func (r *Repository) GetCloudDeliveries(ctx context.Context, tenantID string, connectorID *string, limit int, offset int) ([]CloudDelivery, error) {
	query := `
		SELECT id, tenant_id, connector_id, route_id, original_delivery_id, event_type,
			cloud_message_id, cloud_request_id, payload_hash, payload_size_bytes,
			status, http_status_code, error_code, error_message, sent_at, acknowledged_at, latency_ms, created_at
		FROM cloud_deliveries
		WHERE tenant_id = $1 AND ($2::uuid IS NULL OR connector_id = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.db.Pool.Query(ctx, query, tenantID, connectorID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deliveries []CloudDelivery
	for rows.Next() {
		var d CloudDelivery
		err := rows.Scan(
			&d.ID, &d.TenantID, &d.ConnectorID, &d.RouteID, &d.OriginalDeliveryID, &d.EventType,
			&d.CloudMessageID, &d.CloudRequestID, &d.PayloadHash, &d.PayloadSizeBytes,
			&d.Status, &d.HTTPStatusCode, &d.ErrorCode, &d.ErrorMessage,
			&d.SentAt, &d.AcknowledgedAt, &d.LatencyMs, &d.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		deliveries = append(deliveries, d)
	}

	return deliveries, nil
}
