package catalog

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"webhook-platform/pkg/database"
)

// Repository handles event catalog data access
type Repository struct {
	db *database.DB
}

// NewRepository creates a new catalog repository
func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

// CreateEventType creates a new event type
func (r *Repository) CreateEventType(ctx context.Context, et *EventType) error {
	if et.ID == uuid.Nil {
		et.ID = uuid.New()
	}
	et.CreatedAt = time.Now()
	et.UpdatedAt = time.Now()
	if et.Version == "" {
		et.Version = "1.0.0"
	}
	if et.Status == "" {
		et.Status = StatusActive
	}

	query := `
		INSERT INTO event_types (id, tenant_id, name, slug, description, category, schema_id, 
			version, status, example_payload, tags, metadata, documentation_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`

	_, err := r.db.Pool.Exec(ctx, query, et.ID, et.TenantID, et.Name, et.Slug, et.Description,
		et.Category, et.SchemaID, et.Version, et.Status, et.ExamplePayload, et.Tags,
		et.Metadata, et.DocumentationURL, et.CreatedAt, et.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create event type: %w", err)
	}
	return nil
}

// GetEventType retrieves an event type by ID
func (r *Repository) GetEventType(ctx context.Context, id uuid.UUID) (*EventType, error) {
	query := `
		SELECT id, tenant_id, name, slug, description, category, schema_id, version, status,
			deprecation_message, deprecated_at, replacement_event_id, example_payload, tags,
			metadata, documentation_url, created_at, updated_at
		FROM event_types WHERE id = $1`

	var et EventType
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&et.ID, &et.TenantID, &et.Name, &et.Slug, &et.Description, &et.Category,
		&et.SchemaID, &et.Version, &et.Status, &et.DeprecationMessage, &et.DeprecatedAt,
		&et.ReplacementEventID, &et.ExamplePayload, &et.Tags, &et.Metadata,
		&et.DocumentationURL, &et.CreatedAt, &et.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get event type: %w", err)
	}
	return &et, nil
}

// GetEventTypeBySlug retrieves an event type by tenant and slug
func (r *Repository) GetEventTypeBySlug(ctx context.Context, tenantID uuid.UUID, slug string) (*EventType, error) {
	query := `
		SELECT id, tenant_id, name, slug, description, category, schema_id, version, status,
			deprecation_message, deprecated_at, replacement_event_id, example_payload, tags,
			metadata, documentation_url, created_at, updated_at
		FROM event_types WHERE tenant_id = $1 AND slug = $2 AND status != 'draft'
		ORDER BY version DESC LIMIT 1`

	var et EventType
	err := r.db.Pool.QueryRow(ctx, query, tenantID, slug).Scan(
		&et.ID, &et.TenantID, &et.Name, &et.Slug, &et.Description, &et.Category,
		&et.SchemaID, &et.Version, &et.Status, &et.DeprecationMessage, &et.DeprecatedAt,
		&et.ReplacementEventID, &et.ExamplePayload, &et.Tags, &et.Metadata,
		&et.DocumentationURL, &et.CreatedAt, &et.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get event type: %w", err)
	}
	return &et, nil
}

// UpdateEventType updates an event type
func (r *Repository) UpdateEventType(ctx context.Context, et *EventType) error {
	et.UpdatedAt = time.Now()

	query := `
		UPDATE event_types SET name = $2, description = $3, category = $4, schema_id = $5,
			status = $6, deprecation_message = $7, deprecated_at = $8, replacement_event_id = $9,
			example_payload = $10, tags = $11, metadata = $12, documentation_url = $13, updated_at = $14
		WHERE id = $1`

	_, err := r.db.Pool.Exec(ctx, query, et.ID, et.Name, et.Description, et.Category,
		et.SchemaID, et.Status, et.DeprecationMessage, et.DeprecatedAt, et.ReplacementEventID,
		et.ExamplePayload, et.Tags, et.Metadata, et.DocumentationURL, et.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update event type: %w", err)
	}
	return nil
}

// DeleteEventType deletes an event type
func (r *Repository) DeleteEventType(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx, "DELETE FROM event_types WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete event type: %w", err)
	}
	return nil
}

// SearchEventTypes searches event types with filters
func (r *Repository) SearchEventTypes(ctx context.Context, params *CatalogSearchParams) (*CatalogSearchResult, error) {
	baseQuery := `FROM event_types WHERE tenant_id = $1`
	args := []interface{}{params.TenantID}
	argIndex := 2

	if params.Query != "" {
		baseQuery += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d OR slug ILIKE $%d)", argIndex, argIndex, argIndex)
		args = append(args, "%"+params.Query+"%")
		argIndex++
	}

	if params.Category != "" {
		baseQuery += fmt.Sprintf(" AND category = $%d", argIndex)
		args = append(args, params.Category)
		argIndex++
	}

	if params.Status != "" {
		baseQuery += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, params.Status)
		argIndex++
	}

	if len(params.Tags) > 0 {
		baseQuery += fmt.Sprintf(" AND tags && $%d", argIndex)
		args = append(args, params.Tags)
		argIndex++
	}

	// Count total
	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	err := r.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count event types: %w", err)
	}

	// Get results
	sortBy := "created_at"
	if params.SortBy != "" {
		sortBy = params.SortBy
	}
	sortOrder := "DESC"
	if params.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	limit := 20
	if params.Limit > 0 && params.Limit <= 100 {
		limit = params.Limit
	}

	selectQuery := fmt.Sprintf(`
		SELECT id, tenant_id, name, slug, description, category, schema_id, version, status,
			deprecation_message, deprecated_at, replacement_event_id, example_payload, tags,
			metadata, documentation_url, created_at, updated_at
		%s ORDER BY %s %s LIMIT %d OFFSET %d`, baseQuery, sortBy, sortOrder, limit, params.Offset)

	rows, err := r.db.Pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search event types: %w", err)
	}
	defer rows.Close()

	var eventTypes []*EventType
	for rows.Next() {
		var et EventType
		err := rows.Scan(&et.ID, &et.TenantID, &et.Name, &et.Slug, &et.Description, &et.Category,
			&et.SchemaID, &et.Version, &et.Status, &et.DeprecationMessage, &et.DeprecatedAt,
			&et.ReplacementEventID, &et.ExamplePayload, &et.Tags, &et.Metadata,
			&et.DocumentationURL, &et.CreatedAt, &et.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event type: %w", err)
		}
		eventTypes = append(eventTypes, &et)
	}

	return &CatalogSearchResult{
		EventTypes: eventTypes,
		Total:      total,
		Limit:      limit,
		Offset:     params.Offset,
	}, nil
}

// CreateEventVersion creates a new version of an event type
func (r *Repository) CreateEventVersion(ctx context.Context, ev *EventVersion) error {
	if ev.ID == uuid.Nil {
		ev.ID = uuid.New()
	}
	ev.PublishedAt = time.Now()

	query := `
		INSERT INTO event_type_versions (id, event_type_id, version, schema_id, changelog, is_breaking_change, published_at, published_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.db.Pool.Exec(ctx, query, ev.ID, ev.EventTypeID, ev.Version, ev.SchemaID,
		ev.Changelog, ev.IsBreakingChange, ev.PublishedAt, ev.PublishedBy)
	if err != nil {
		return fmt.Errorf("failed to create event version: %w", err)
	}
	return nil
}

// ListEventVersions lists all versions of an event type
func (r *Repository) ListEventVersions(ctx context.Context, eventTypeID uuid.UUID) ([]*EventVersion, error) {
	query := `
		SELECT id, event_type_id, version, schema_id, changelog, is_breaking_change, published_at, published_by
		FROM event_type_versions WHERE event_type_id = $1 ORDER BY published_at DESC`

	rows, err := r.db.Pool.Query(ctx, query, eventTypeID)
	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}
	defer rows.Close()

	var versions []*EventVersion
	for rows.Next() {
		var ev EventVersion
		err := rows.Scan(&ev.ID, &ev.EventTypeID, &ev.Version, &ev.SchemaID,
			&ev.Changelog, &ev.IsBreakingChange, &ev.PublishedAt, &ev.PublishedBy)
		if err != nil {
			return nil, fmt.Errorf("failed to scan version: %w", err)
		}
		versions = append(versions, &ev)
	}
	return versions, nil
}

// CreateCategory creates a new event category
func (r *Repository) CreateCategory(ctx context.Context, cat *EventCategory) error {
	if cat.ID == uuid.Nil {
		cat.ID = uuid.New()
	}
	cat.CreatedAt = time.Now()

	query := `
		INSERT INTO event_categories (id, tenant_id, name, slug, description, icon, color, parent_id, sort_order, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.db.Pool.Exec(ctx, query, cat.ID, cat.TenantID, cat.Name, cat.Slug,
		cat.Description, cat.Icon, cat.Color, cat.ParentID, cat.SortOrder, cat.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create category: %w", err)
	}
	return nil
}

// ListCategories lists all categories for a tenant
func (r *Repository) ListCategories(ctx context.Context, tenantID uuid.UUID) ([]*EventCategory, error) {
	query := `
		SELECT c.id, c.tenant_id, c.name, c.slug, c.description, c.icon, c.color, c.parent_id, c.sort_order, c.created_at,
			COUNT(DISTINCT e.id) as event_count
		FROM event_categories c
		LEFT JOIN event_types e ON e.category = c.slug AND e.tenant_id = c.tenant_id
		WHERE c.tenant_id = $1
		GROUP BY c.id
		ORDER BY c.sort_order, c.name`

	rows, err := r.db.Pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list categories: %w", err)
	}
	defer rows.Close()

	var categories []*EventCategory
	for rows.Next() {
		var cat EventCategory
		err := rows.Scan(&cat.ID, &cat.TenantID, &cat.Name, &cat.Slug, &cat.Description,
			&cat.Icon, &cat.Color, &cat.ParentID, &cat.SortOrder, &cat.CreatedAt, &cat.EventCount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, &cat)
	}
	return categories, nil
}

// CreateSubscription creates a new event subscription
func (r *Repository) CreateSubscription(ctx context.Context, sub *EventSubscription) error {
	if sub.ID == uuid.Nil {
		sub.ID = uuid.New()
	}
	sub.CreatedAt = time.Now()

	query := `
		INSERT INTO event_subscriptions (id, endpoint_id, event_type_id, filter_expression, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (endpoint_id, event_type_id) DO UPDATE SET filter_expression = $4, is_active = $5`

	_, err := r.db.Pool.Exec(ctx, query, sub.ID, sub.EndpointID, sub.EventTypeID,
		sub.FilterExpression, sub.IsActive, sub.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}
	return nil
}

// DeleteSubscription removes an event subscription
func (r *Repository) DeleteSubscription(ctx context.Context, endpointID, eventTypeID uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx, "DELETE FROM event_subscriptions WHERE endpoint_id = $1 AND event_type_id = $2",
		endpointID, eventTypeID)
	if err != nil {
		return fmt.Errorf("failed to delete subscription: %w", err)
	}
	return nil
}

// ListEndpointSubscriptions lists event subscriptions for an endpoint
func (r *Repository) ListEndpointSubscriptions(ctx context.Context, endpointID uuid.UUID) ([]*EventSubscription, error) {
	query := `
		SELECT id, endpoint_id, event_type_id, filter_expression, is_active, created_at
		FROM event_subscriptions WHERE endpoint_id = $1 AND is_active = true`

	rows, err := r.db.Pool.Query(ctx, query, endpointID)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []*EventSubscription
	for rows.Next() {
		var sub EventSubscription
		err := rows.Scan(&sub.ID, &sub.EndpointID, &sub.EventTypeID, &sub.FilterExpression, &sub.IsActive, &sub.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan subscription: %w", err)
		}
		subs = append(subs, &sub)
	}
	return subs, nil
}

// GetSubscriberCount returns the number of endpoints subscribed to an event type
func (r *Repository) GetSubscriberCount(ctx context.Context, eventTypeID uuid.UUID) (int, error) {
	var count int
	err := r.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM event_subscriptions WHERE event_type_id = $1 AND is_active = true", eventTypeID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count subscribers: %w", err)
	}
	return count, nil
}

// SaveDocumentation saves event documentation
func (r *Repository) SaveDocumentation(ctx context.Context, doc *EventDocumentation) error {
	if doc.ID == uuid.Nil {
		doc.ID = uuid.New()
	}
	doc.CreatedAt = time.Now()
	doc.UpdatedAt = time.Now()

	query := `
		INSERT INTO event_documentation (id, event_type_id, content_type, content, section, sort_order, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (event_type_id, section) DO UPDATE SET content = $4, updated_at = $8`

	_, err := r.db.Pool.Exec(ctx, query, doc.ID, doc.EventTypeID, doc.ContentType,
		doc.Content, doc.Section, doc.SortOrder, doc.CreatedAt, doc.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to save documentation: %w", err)
	}
	return nil
}

// GetDocumentation retrieves all documentation for an event type
func (r *Repository) GetDocumentation(ctx context.Context, eventTypeID uuid.UUID) ([]*EventDocumentation, error) {
	query := `
		SELECT id, event_type_id, content_type, content, section, sort_order, created_at, updated_at
		FROM event_documentation WHERE event_type_id = $1 ORDER BY sort_order`

	rows, err := r.db.Pool.Query(ctx, query, eventTypeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get documentation: %w", err)
	}
	defer rows.Close()

	var docs []*EventDocumentation
	for rows.Next() {
		var doc EventDocumentation
		err := rows.Scan(&doc.ID, &doc.EventTypeID, &doc.ContentType, &doc.Content,
			&doc.Section, &doc.SortOrder, &doc.CreatedAt, &doc.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan documentation: %w", err)
		}
		docs = append(docs, &doc)
	}
	return docs, nil
}
