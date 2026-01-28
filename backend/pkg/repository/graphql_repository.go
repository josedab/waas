package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"webhook-platform/pkg/models"
)

// GraphQLRepository handles GraphQL schema and subscription persistence
type GraphQLRepository interface {
	// Schema operations
	CreateSchema(ctx context.Context, schema *models.GraphQLSchema) error
	GetSchema(ctx context.Context, id uuid.UUID) (*models.GraphQLSchema, error)
	GetSchemasByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.GraphQLSchema, error)
	UpdateSchema(ctx context.Context, schema *models.GraphQLSchema) error
	DeleteSchema(ctx context.Context, id uuid.UUID) error

	// Subscription operations
	CreateSubscription(ctx context.Context, sub *models.GraphQLSubscription) error
	GetSubscription(ctx context.Context, id uuid.UUID) (*models.GraphQLSubscription, error)
	GetSubscriptionsBySchema(ctx context.Context, schemaID uuid.UUID) ([]*models.GraphQLSubscription, error)
	GetSubscriptionsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.GraphQLSubscription, error)
	GetActiveSubscriptions(ctx context.Context, tenantID uuid.UUID) ([]*models.GraphQLSubscription, error)
	UpdateSubscription(ctx context.Context, sub *models.GraphQLSubscription) error
	DeleteSubscription(ctx context.Context, id uuid.UUID) error

	// Subscription events
	CreateSubscriptionEvent(ctx context.Context, event *models.GraphQLSubscriptionEvent) error
	GetPendingEvents(ctx context.Context, subscriptionID uuid.UUID, limit int) ([]*models.GraphQLSubscriptionEvent, error)
	MarkEventDelivered(ctx context.Context, eventID, deliveryID uuid.UUID) error

	// Federation sources
	AddFederationSource(ctx context.Context, source *models.GraphQLFederationSource) error
	GetFederationSources(ctx context.Context, schemaID uuid.UUID) ([]*models.GraphQLFederationSource, error)
	UpdateFederationSourceHealth(ctx context.Context, id uuid.UUID, status string) error
	DeleteFederationSource(ctx context.Context, id uuid.UUID) error

	// Type mappings
	CreateTypeMapping(ctx context.Context, mapping *models.GraphQLTypeMapping) error
	GetTypeMappings(ctx context.Context, schemaID uuid.UUID) ([]*models.GraphQLTypeMapping, error)
	GetTypeMappingByType(ctx context.Context, schemaID uuid.UUID, graphqlType string) (*models.GraphQLTypeMapping, error)
	DeleteTypeMapping(ctx context.Context, id uuid.UUID) error
}

// PostgresGraphQLRepository implements GraphQLRepository using PostgreSQL
type PostgresGraphQLRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresGraphQLRepository creates a new PostgreSQL-backed GraphQL repository
func NewPostgresGraphQLRepository(pool *pgxpool.Pool) *PostgresGraphQLRepository {
	return &PostgresGraphQLRepository{pool: pool}
}

// Schema operations

func (r *PostgresGraphQLRepository) CreateSchema(ctx context.Context, schema *models.GraphQLSchema) error {
	query := `
		INSERT INTO graphql_schemas (
			id, tenant_id, name, description, schema_sdl, version, status,
			introspection_endpoint, federation_enabled, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)
	`

	if schema.ID == uuid.Nil {
		schema.ID = uuid.New()
	}
	if schema.Status == "" {
		schema.Status = models.GraphQLSchemaStatusActive
	}
	if schema.Version == "" {
		schema.Version = "1.0.0"
	}

	_, err := r.pool.Exec(ctx, query,
		schema.ID, schema.TenantID, schema.Name, schema.Description,
		schema.SchemaSDL, schema.Version, schema.Status,
		schema.IntrospectionEndpoint, schema.FederationEnabled,
	)

	return err
}

func (r *PostgresGraphQLRepository) GetSchema(ctx context.Context, id uuid.UUID) (*models.GraphQLSchema, error) {
	query := `
		SELECT id, tenant_id, name, description, schema_sdl, version, status,
		       introspection_endpoint, federation_enabled, created_at, updated_at
		FROM graphql_schemas
		WHERE id = $1
	`

	schema := &models.GraphQLSchema{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&schema.ID, &schema.TenantID, &schema.Name, &schema.Description,
		&schema.SchemaSDL, &schema.Version, &schema.Status,
		&schema.IntrospectionEndpoint, &schema.FederationEnabled,
		&schema.CreatedAt, &schema.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("schema not found: %w", err)
	}

	return schema, nil
}

func (r *PostgresGraphQLRepository) GetSchemasByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.GraphQLSchema, error) {
	query := `
		SELECT id, tenant_id, name, description, schema_sdl, version, status,
		       introspection_endpoint, federation_enabled, created_at, updated_at
		FROM graphql_schemas
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schemas []*models.GraphQLSchema
	for rows.Next() {
		schema := &models.GraphQLSchema{}
		if err := rows.Scan(
			&schema.ID, &schema.TenantID, &schema.Name, &schema.Description,
			&schema.SchemaSDL, &schema.Version, &schema.Status,
			&schema.IntrospectionEndpoint, &schema.FederationEnabled,
			&schema.CreatedAt, &schema.UpdatedAt,
		); err != nil {
			return nil, err
		}
		schemas = append(schemas, schema)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return schemas, nil
}

func (r *PostgresGraphQLRepository) UpdateSchema(ctx context.Context, schema *models.GraphQLSchema) error {
	query := `
		UPDATE graphql_schemas
		SET name = $2, description = $3, schema_sdl = $4, version = $5,
		    status = $6, introspection_endpoint = $7, federation_enabled = $8,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query,
		schema.ID, schema.Name, schema.Description, schema.SchemaSDL,
		schema.Version, schema.Status, schema.IntrospectionEndpoint,
		schema.FederationEnabled,
	)

	return err
}

func (r *PostgresGraphQLRepository) DeleteSchema(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM graphql_schemas WHERE id = $1", id)
	return err
}

// Subscription operations

func (r *PostgresGraphQLRepository) CreateSubscription(ctx context.Context, sub *models.GraphQLSubscription) error {
	query := `
		INSERT INTO graphql_subscriptions (
			id, tenant_id, schema_id, endpoint_id, name, description,
			subscription_query, variables, filter_expression, field_selection,
			transform_js, status, delivery_config, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
			CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)
	`

	if sub.ID == uuid.Nil {
		sub.ID = uuid.New()
	}
	if sub.Status == "" {
		sub.Status = models.GraphQLSubscriptionActive
	}

	variablesJSON, _ := json.Marshal(sub.Variables)
	fieldSelectionJSON, _ := json.Marshal(sub.FieldSelection)
	deliveryConfigJSON, _ := json.Marshal(sub.DeliveryConfig)

	_, err := r.pool.Exec(ctx, query,
		sub.ID, sub.TenantID, sub.SchemaID, sub.EndpointID, sub.Name, sub.Description,
		sub.SubscriptionQuery, variablesJSON, sub.FilterExpression, fieldSelectionJSON,
		sub.TransformJS, sub.Status, deliveryConfigJSON,
	)

	return err
}

func (r *PostgresGraphQLRepository) GetSubscription(ctx context.Context, id uuid.UUID) (*models.GraphQLSubscription, error) {
	query := `
		SELECT id, tenant_id, schema_id, endpoint_id, name, description,
		       subscription_query, variables, filter_expression, field_selection,
		       transform_js, status, delivery_config, created_at, updated_at
		FROM graphql_subscriptions
		WHERE id = $1
	`

	sub := &models.GraphQLSubscription{}
	var variablesJSON, fieldSelectionJSON, deliveryConfigJSON []byte

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&sub.ID, &sub.TenantID, &sub.SchemaID, &sub.EndpointID, &sub.Name, &sub.Description,
		&sub.SubscriptionQuery, &variablesJSON, &sub.FilterExpression, &fieldSelectionJSON,
		&sub.TransformJS, &sub.Status, &deliveryConfigJSON, &sub.CreatedAt, &sub.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("subscription not found: %w", err)
	}

	json.Unmarshal(variablesJSON, &sub.Variables)
	json.Unmarshal(fieldSelectionJSON, &sub.FieldSelection)
	json.Unmarshal(deliveryConfigJSON, &sub.DeliveryConfig)

	return sub, nil
}

func (r *PostgresGraphQLRepository) GetSubscriptionsBySchema(ctx context.Context, schemaID uuid.UUID) ([]*models.GraphQLSubscription, error) {
	query := `
		SELECT id, tenant_id, schema_id, endpoint_id, name, description,
		       subscription_query, variables, filter_expression, field_selection,
		       transform_js, status, delivery_config, created_at, updated_at
		FROM graphql_subscriptions
		WHERE schema_id = $1
		ORDER BY created_at DESC
	`

	return r.querySubscriptions(ctx, query, schemaID)
}

func (r *PostgresGraphQLRepository) GetSubscriptionsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.GraphQLSubscription, error) {
	query := `
		SELECT id, tenant_id, schema_id, endpoint_id, name, description,
		       subscription_query, variables, filter_expression, field_selection,
		       transform_js, status, delivery_config, created_at, updated_at
		FROM graphql_subscriptions
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	return r.querySubscriptions(ctx, query, tenantID)
}

func (r *PostgresGraphQLRepository) GetActiveSubscriptions(ctx context.Context, tenantID uuid.UUID) ([]*models.GraphQLSubscription, error) {
	query := `
		SELECT id, tenant_id, schema_id, endpoint_id, name, description,
		       subscription_query, variables, filter_expression, field_selection,
		       transform_js, status, delivery_config, created_at, updated_at
		FROM graphql_subscriptions
		WHERE tenant_id = $1 AND status = 'active'
		ORDER BY created_at DESC
	`

	return r.querySubscriptions(ctx, query, tenantID)
}

func (r *PostgresGraphQLRepository) querySubscriptions(ctx context.Context, query string, arg interface{}) ([]*models.GraphQLSubscription, error) {
	rows, err := r.pool.Query(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []*models.GraphQLSubscription
	for rows.Next() {
		sub := &models.GraphQLSubscription{}
		var variablesJSON, fieldSelectionJSON, deliveryConfigJSON []byte

		if err := rows.Scan(
			&sub.ID, &sub.TenantID, &sub.SchemaID, &sub.EndpointID, &sub.Name, &sub.Description,
			&sub.SubscriptionQuery, &variablesJSON, &sub.FilterExpression, &fieldSelectionJSON,
			&sub.TransformJS, &sub.Status, &deliveryConfigJSON, &sub.CreatedAt, &sub.UpdatedAt,
		); err != nil {
			return nil, err
		}

		json.Unmarshal(variablesJSON, &sub.Variables)
		json.Unmarshal(fieldSelectionJSON, &sub.FieldSelection)
		json.Unmarshal(deliveryConfigJSON, &sub.DeliveryConfig)

		subs = append(subs, sub)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return subs, nil
}

func (r *PostgresGraphQLRepository) UpdateSubscription(ctx context.Context, sub *models.GraphQLSubscription) error {
	query := `
		UPDATE graphql_subscriptions
		SET name = $2, description = $3, subscription_query = $4, variables = $5,
		    filter_expression = $6, field_selection = $7, transform_js = $8,
		    status = $9, delivery_config = $10, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	variablesJSON, _ := json.Marshal(sub.Variables)
	fieldSelectionJSON, _ := json.Marshal(sub.FieldSelection)
	deliveryConfigJSON, _ := json.Marshal(sub.DeliveryConfig)

	_, err := r.pool.Exec(ctx, query,
		sub.ID, sub.Name, sub.Description, sub.SubscriptionQuery, variablesJSON,
		sub.FilterExpression, fieldSelectionJSON, sub.TransformJS,
		sub.Status, deliveryConfigJSON,
	)

	return err
}

func (r *PostgresGraphQLRepository) DeleteSubscription(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM graphql_subscriptions WHERE id = $1", id)
	return err
}

// Subscription events

func (r *PostgresGraphQLRepository) CreateSubscriptionEvent(ctx context.Context, event *models.GraphQLSubscriptionEvent) error {
	query := `
		INSERT INTO graphql_subscription_events (
			id, subscription_id, tenant_id, event_type, payload, filtered_payload,
			delivered, received_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP
		)
	`

	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}

	payloadJSON, _ := json.Marshal(event.Payload)
	filteredPayloadJSON, _ := json.Marshal(event.FilteredPayload)

	_, err := r.pool.Exec(ctx, query,
		event.ID, event.SubscriptionID, event.TenantID, event.EventType,
		payloadJSON, filteredPayloadJSON, event.Delivered,
	)

	return err
}

func (r *PostgresGraphQLRepository) GetPendingEvents(ctx context.Context, subscriptionID uuid.UUID, limit int) ([]*models.GraphQLSubscriptionEvent, error) {
	query := `
		SELECT id, subscription_id, tenant_id, event_type, payload, filtered_payload,
		       delivered, delivery_id, received_at, processed_at
		FROM graphql_subscription_events
		WHERE subscription_id = $1 AND delivered = false
		ORDER BY received_at ASC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, subscriptionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*models.GraphQLSubscriptionEvent
	for rows.Next() {
		event := &models.GraphQLSubscriptionEvent{}
		var payloadJSON, filteredPayloadJSON []byte

		if err := rows.Scan(
			&event.ID, &event.SubscriptionID, &event.TenantID, &event.EventType,
			&payloadJSON, &filteredPayloadJSON, &event.Delivered, &event.DeliveryID,
			&event.ReceivedAt, &event.ProcessedAt,
		); err != nil {
			return nil, err
		}

		json.Unmarshal(payloadJSON, &event.Payload)
		json.Unmarshal(filteredPayloadJSON, &event.FilteredPayload)

		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

func (r *PostgresGraphQLRepository) MarkEventDelivered(ctx context.Context, eventID, deliveryID uuid.UUID) error {
	query := `
		UPDATE graphql_subscription_events
		SET delivered = true, delivery_id = $2, processed_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, eventID, deliveryID)
	return err
}

// Federation sources

func (r *PostgresGraphQLRepository) AddFederationSource(ctx context.Context, source *models.GraphQLFederationSource) error {
	query := `
		INSERT INTO graphql_federation_sources (
			id, schema_id, tenant_id, name, endpoint_url, subgraph_sdl,
			auth_config, health_status, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)
	`

	if source.ID == uuid.Nil {
		source.ID = uuid.New()
	}
	if source.HealthStatus == "" {
		source.HealthStatus = models.FederationHealthUnknown
	}

	authConfigJSON, _ := json.Marshal(source.AuthConfig)

	_, err := r.pool.Exec(ctx, query,
		source.ID, source.SchemaID, source.TenantID, source.Name,
		source.EndpointURL, source.SubgraphSDL, authConfigJSON, source.HealthStatus,
	)

	return err
}

func (r *PostgresGraphQLRepository) GetFederationSources(ctx context.Context, schemaID uuid.UUID) ([]*models.GraphQLFederationSource, error) {
	query := `
		SELECT id, schema_id, tenant_id, name, endpoint_url, subgraph_sdl,
		       auth_config, health_status, last_health_check, created_at, updated_at
		FROM graphql_federation_sources
		WHERE schema_id = $1
		ORDER BY name ASC
	`

	rows, err := r.pool.Query(ctx, query, schemaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []*models.GraphQLFederationSource
	for rows.Next() {
		source := &models.GraphQLFederationSource{}
		var authConfigJSON []byte

		if err := rows.Scan(
			&source.ID, &source.SchemaID, &source.TenantID, &source.Name,
			&source.EndpointURL, &source.SubgraphSDL, &authConfigJSON,
			&source.HealthStatus, &source.LastHealthCheck, &source.CreatedAt, &source.UpdatedAt,
		); err != nil {
			return nil, err
		}

		json.Unmarshal(authConfigJSON, &source.AuthConfig)
		sources = append(sources, source)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sources, nil
}

func (r *PostgresGraphQLRepository) UpdateFederationSourceHealth(ctx context.Context, id uuid.UUID, status string) error {
	query := `
		UPDATE graphql_federation_sources
		SET health_status = $2, last_health_check = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, id, status)
	return err
}

func (r *PostgresGraphQLRepository) DeleteFederationSource(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM graphql_federation_sources WHERE id = $1", id)
	return err
}

// Type mappings

func (r *PostgresGraphQLRepository) CreateTypeMapping(ctx context.Context, mapping *models.GraphQLTypeMapping) error {
	query := `
		INSERT INTO graphql_type_mappings (
			id, schema_id, tenant_id, graphql_type, webhook_event_type,
			field_mappings, auto_generated, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)
	`

	if mapping.ID == uuid.Nil {
		mapping.ID = uuid.New()
	}

	fieldMappingsJSON, _ := json.Marshal(mapping.FieldMappings)

	_, err := r.pool.Exec(ctx, query,
		mapping.ID, mapping.SchemaID, mapping.TenantID, mapping.GraphQLType,
		mapping.WebhookEventType, fieldMappingsJSON, mapping.AutoGenerated,
	)

	return err
}

func (r *PostgresGraphQLRepository) GetTypeMappings(ctx context.Context, schemaID uuid.UUID) ([]*models.GraphQLTypeMapping, error) {
	query := `
		SELECT id, schema_id, tenant_id, graphql_type, webhook_event_type,
		       field_mappings, auto_generated, created_at, updated_at
		FROM graphql_type_mappings
		WHERE schema_id = $1
		ORDER BY graphql_type ASC
	`

	rows, err := r.pool.Query(ctx, query, schemaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mappings []*models.GraphQLTypeMapping
	for rows.Next() {
		mapping := &models.GraphQLTypeMapping{}
		var fieldMappingsJSON []byte

		if err := rows.Scan(
			&mapping.ID, &mapping.SchemaID, &mapping.TenantID, &mapping.GraphQLType,
			&mapping.WebhookEventType, &fieldMappingsJSON, &mapping.AutoGenerated,
			&mapping.CreatedAt, &mapping.UpdatedAt,
		); err != nil {
			return nil, err
		}

		json.Unmarshal(fieldMappingsJSON, &mapping.FieldMappings)
		mappings = append(mappings, mapping)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return mappings, nil
}

func (r *PostgresGraphQLRepository) GetTypeMappingByType(ctx context.Context, schemaID uuid.UUID, graphqlType string) (*models.GraphQLTypeMapping, error) {
	query := `
		SELECT id, schema_id, tenant_id, graphql_type, webhook_event_type,
		       field_mappings, auto_generated, created_at, updated_at
		FROM graphql_type_mappings
		WHERE schema_id = $1 AND graphql_type = $2
	`

	mapping := &models.GraphQLTypeMapping{}
	var fieldMappingsJSON []byte

	err := r.pool.QueryRow(ctx, query, schemaID, graphqlType).Scan(
		&mapping.ID, &mapping.SchemaID, &mapping.TenantID, &mapping.GraphQLType,
		&mapping.WebhookEventType, &fieldMappingsJSON, &mapping.AutoGenerated,
		&mapping.CreatedAt, &mapping.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("type mapping not found: %w", err)
	}

	json.Unmarshal(fieldMappingsJSON, &mapping.FieldMappings)

	return mapping, nil
}

func (r *PostgresGraphQLRepository) DeleteTypeMapping(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM graphql_type_mappings WHERE id = $1", id)
	return err
}
