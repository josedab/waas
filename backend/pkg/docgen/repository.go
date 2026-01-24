package docgen

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"webhook-platform/pkg/database"
)

// Repository defines the interface for docgen storage
type Repository interface {
	CreateDoc(ctx context.Context, doc *WebhookDoc) error
	GetDoc(ctx context.Context, id uuid.UUID) (*WebhookDoc, error)
	UpdateDoc(ctx context.Context, doc *WebhookDoc) error
	ListDocs(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*WebhookDoc, int, error)
	DeleteDoc(ctx context.Context, id uuid.UUID) error

	CreateEventType(ctx context.Context, et *EventTypeDoc) error
	GetEventType(ctx context.Context, id uuid.UUID) (*EventTypeDoc, error)
	UpdateEventType(ctx context.Context, et *EventTypeDoc) error
	ListEventTypes(ctx context.Context, docID uuid.UUID) ([]*EventTypeDoc, error)
	DeleteEventType(ctx context.Context, id uuid.UUID) error

	CreateCodeSample(ctx context.Context, sample *CodeSample) error
	GetCodeSample(ctx context.Context, eventTypeID uuid.UUID, language string) (*CodeSample, error)

	CreateWidget(ctx context.Context, widget *DocWidget) error
	GetWidget(ctx context.Context, id uuid.UUID) (*DocWidget, error)
	IncrementWidgetViewCount(ctx context.Context, id uuid.UUID) error

	GetDocAnalytics(ctx context.Context, docID uuid.UUID) (*DocAnalytics, error)
	RecordDocView(ctx context.Context, docID uuid.UUID) error
}

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// CreateDoc creates a new webhook doc
func (r *PostgresRepository) CreateDoc(ctx context.Context, doc *WebhookDoc) error {
	if doc.ID == uuid.Nil {
		doc.ID = uuid.New()
	}
	doc.CreatedAt = time.Now()
	doc.UpdatedAt = time.Now()
	if doc.Version == "" {
		doc.Version = "1.0.0"
	}

	query := `
		INSERT INTO docgen_docs (id, tenant_id, name, description, version, event_types, base_url, auth_method, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.ExecContext(ctx, query,
		doc.ID, doc.TenantID, doc.Name, doc.Description, doc.Version,
		database.StringArray(doc.EventTypes), doc.BaseURL, doc.AuthMethod,
		doc.CreatedAt, doc.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create doc: %w", err)
	}
	return nil
}

// GetDoc retrieves a webhook doc by ID
func (r *PostgresRepository) GetDoc(ctx context.Context, id uuid.UUID) (*WebhookDoc, error) {
	query := `
		SELECT id, tenant_id, name, description, version, event_types, base_url, auth_method, created_at, updated_at
		FROM docgen_docs WHERE id = $1
	`

	var doc WebhookDoc
	var eventTypes []string
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&doc.ID, &doc.TenantID, &doc.Name, &doc.Description, &doc.Version,
		(*database.StringArray)(&eventTypes), &doc.BaseURL, &doc.AuthMethod,
		&doc.CreatedAt, &doc.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("doc not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get doc: %w", err)
	}
	doc.EventTypes = eventTypes
	return &doc, nil
}

// UpdateDoc updates a webhook doc
func (r *PostgresRepository) UpdateDoc(ctx context.Context, doc *WebhookDoc) error {
	doc.UpdatedAt = time.Now()

	query := `
		UPDATE docgen_docs
		SET name = $2, description = $3, version = $4, event_types = $5, base_url = $6, auth_method = $7, updated_at = $8
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query,
		doc.ID, doc.Name, doc.Description, doc.Version,
		database.StringArray(doc.EventTypes), doc.BaseURL, doc.AuthMethod,
		doc.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update doc: %w", err)
	}
	return nil
}

// ListDocs lists webhook docs for a tenant
func (r *PostgresRepository) ListDocs(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*WebhookDoc, int, error) {
	countQuery := `SELECT COUNT(*) FROM docgen_docs WHERE tenant_id = $1`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, tenantID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count docs: %w", err)
	}

	query := `
		SELECT id, tenant_id, name, description, version, event_types, base_url, auth_method, created_at, updated_at
		FROM docgen_docs
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list docs: %w", err)
	}
	defer rows.Close()

	var docs []*WebhookDoc
	for rows.Next() {
		var doc WebhookDoc
		var eventTypes []string
		if err := rows.Scan(
			&doc.ID, &doc.TenantID, &doc.Name, &doc.Description, &doc.Version,
			(*database.StringArray)(&eventTypes), &doc.BaseURL, &doc.AuthMethod,
			&doc.CreatedAt, &doc.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan doc: %w", err)
		}
		doc.EventTypes = eventTypes
		docs = append(docs, &doc)
	}

	return docs, total, nil
}

// DeleteDoc deletes a webhook doc
func (r *PostgresRepository) DeleteDoc(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM docgen_docs WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete doc: %w", err)
	}
	return nil
}

// CreateEventType creates a new event type doc
func (r *PostgresRepository) CreateEventType(ctx context.Context, et *EventTypeDoc) error {
	if et.ID == uuid.Nil {
		et.ID = uuid.New()
	}
	et.CreatedAt = time.Now()
	if et.Version == "" {
		et.Version = "1.0.0"
	}

	query := `
		INSERT INTO docgen_event_types (id, doc_id, name, description, category, payload_schema, example_payload, deprecated, deprecation_notice, version, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.ExecContext(ctx, query,
		et.ID, et.DocID, et.Name, et.Description, et.Category,
		et.PayloadSchema, et.ExamplePayload, et.Deprecated,
		et.DeprecationNotice, et.Version, et.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create event type: %w", err)
	}
	return nil
}

// GetEventType retrieves an event type doc by ID
func (r *PostgresRepository) GetEventType(ctx context.Context, id uuid.UUID) (*EventTypeDoc, error) {
	query := `
		SELECT id, doc_id, name, description, category, payload_schema, example_payload, deprecated, deprecation_notice, version, created_at
		FROM docgen_event_types WHERE id = $1
	`

	var et EventTypeDoc
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&et.ID, &et.DocID, &et.Name, &et.Description, &et.Category,
		&et.PayloadSchema, &et.ExamplePayload, &et.Deprecated,
		&et.DeprecationNotice, &et.Version, &et.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("event type not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get event type: %w", err)
	}
	return &et, nil
}

// UpdateEventType updates an event type doc
func (r *PostgresRepository) UpdateEventType(ctx context.Context, et *EventTypeDoc) error {
	query := `
		UPDATE docgen_event_types
		SET name = $2, description = $3, category = $4, payload_schema = $5, example_payload = $6,
			deprecated = $7, deprecation_notice = $8, version = $9
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query,
		et.ID, et.Name, et.Description, et.Category,
		et.PayloadSchema, et.ExamplePayload, et.Deprecated,
		et.DeprecationNotice, et.Version,
	)
	if err != nil {
		return fmt.Errorf("failed to update event type: %w", err)
	}
	return nil
}

// ListEventTypes lists event type docs for a webhook doc
func (r *PostgresRepository) ListEventTypes(ctx context.Context, docID uuid.UUID) ([]*EventTypeDoc, error) {
	query := `
		SELECT id, doc_id, name, description, category, payload_schema, example_payload, deprecated, deprecation_notice, version, created_at
		FROM docgen_event_types
		WHERE doc_id = $1
		ORDER BY category, name
	`

	rows, err := r.db.QueryContext(ctx, query, docID)
	if err != nil {
		return nil, fmt.Errorf("failed to list event types: %w", err)
	}
	defer rows.Close()

	var eventTypes []*EventTypeDoc
	for rows.Next() {
		var et EventTypeDoc
		if err := rows.Scan(
			&et.ID, &et.DocID, &et.Name, &et.Description, &et.Category,
			&et.PayloadSchema, &et.ExamplePayload, &et.Deprecated,
			&et.DeprecationNotice, &et.Version, &et.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan event type: %w", err)
		}
		eventTypes = append(eventTypes, &et)
	}
	return eventTypes, nil
}

// DeleteEventType deletes an event type doc
func (r *PostgresRepository) DeleteEventType(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM docgen_event_types WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete event type: %w", err)
	}
	return nil
}

// CreateCodeSample creates a new code sample
func (r *PostgresRepository) CreateCodeSample(ctx context.Context, sample *CodeSample) error {
	if sample.ID == uuid.Nil {
		sample.ID = uuid.New()
	}

	query := `
		INSERT INTO docgen_code_samples (id, event_type_id, language, code, framework, description)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (event_type_id, language) DO UPDATE SET code = $4, framework = $5, description = $6
	`

	_, err := r.db.ExecContext(ctx, query,
		sample.ID, sample.EventTypeID, sample.Language,
		sample.Code, sample.Framework, sample.Description,
	)
	if err != nil {
		return fmt.Errorf("failed to create code sample: %w", err)
	}
	return nil
}

// GetCodeSample retrieves a code sample by event type and language
func (r *PostgresRepository) GetCodeSample(ctx context.Context, eventTypeID uuid.UUID, language string) (*CodeSample, error) {
	query := `
		SELECT id, event_type_id, language, code, framework, description
		FROM docgen_code_samples
		WHERE event_type_id = $1 AND language = $2
	`

	var sample CodeSample
	err := r.db.QueryRowContext(ctx, query, eventTypeID, language).Scan(
		&sample.ID, &sample.EventTypeID, &sample.Language,
		&sample.Code, &sample.Framework, &sample.Description,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get code sample: %w", err)
	}
	return &sample, nil
}

// CreateWidget creates a new doc widget
func (r *PostgresRepository) CreateWidget(ctx context.Context, widget *DocWidget) error {
	if widget.ID == uuid.Nil {
		widget.ID = uuid.New()
	}
	widget.CreatedAt = time.Now()

	query := `
		INSERT INTO docgen_widgets (id, tenant_id, doc_id, theme, custom_css, allowed_domains, embed_key, view_count, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.ExecContext(ctx, query,
		widget.ID, widget.TenantID, widget.DocID, widget.Theme, widget.CustomCSS,
		database.StringArray(widget.AllowedDomains), widget.EmbedKey,
		widget.ViewCount, widget.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create widget: %w", err)
	}
	return nil
}

// GetWidget retrieves a doc widget by ID
func (r *PostgresRepository) GetWidget(ctx context.Context, id uuid.UUID) (*DocWidget, error) {
	query := `
		SELECT id, tenant_id, doc_id, theme, custom_css, allowed_domains, embed_key, view_count, created_at
		FROM docgen_widgets WHERE id = $1
	`

	var widget DocWidget
	var allowedDomains []string
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&widget.ID, &widget.TenantID, &widget.DocID, &widget.Theme, &widget.CustomCSS,
		(*database.StringArray)(&allowedDomains), &widget.EmbedKey,
		&widget.ViewCount, &widget.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("widget not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get widget: %w", err)
	}
	widget.AllowedDomains = allowedDomains
	return &widget, nil
}

// IncrementWidgetViewCount increments the view count on a widget
func (r *PostgresRepository) IncrementWidgetViewCount(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE docgen_widgets SET view_count = view_count + 1 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to increment widget view count: %w", err)
	}
	return nil
}

// GetDocAnalytics retrieves analytics for a doc
func (r *PostgresRepository) GetDocAnalytics(ctx context.Context, docID uuid.UUID) (*DocAnalytics, error) {
	analytics := &DocAnalytics{
		DocID:     docID,
		TopEvents: []string{},
	}

	query := `
		SELECT
			COALESCE(SUM(views), 0) as views,
			COALESCE(SUM(unique_visitors), 0) as unique_visitors,
			COALESCE(AVG(avg_time_on_page), 0) as avg_time_on_page
		FROM docgen_analytics
		WHERE doc_id = $1
	`

	err := r.db.QueryRowContext(ctx, query, docID).Scan(
		&analytics.Views,
		&analytics.UniqueVisitors,
		&analytics.AvgTimeOnPage,
	)
	if err != nil && err != sql.ErrNoRows {
		return analytics, nil
	}

	// Get widget views
	widgetQuery := `SELECT COALESCE(SUM(view_count), 0) FROM docgen_widgets WHERE doc_id = $1`
	r.db.QueryRowContext(ctx, widgetQuery, docID).Scan(&analytics.WidgetViews)

	// Get top events
	topQuery := `
		SELECT et.name FROM docgen_event_types et
		WHERE et.doc_id = $1
		ORDER BY et.created_at DESC
		LIMIT 5
	`
	rows, err := r.db.QueryContext(ctx, topQuery, docID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err == nil {
				analytics.TopEvents = append(analytics.TopEvents, name)
			}
		}
	}

	return analytics, nil
}

// RecordDocView records a view for a doc
func (r *PostgresRepository) RecordDocView(ctx context.Context, docID uuid.UUID) error {
	query := `
		INSERT INTO docgen_analytics (doc_id, views, unique_visitors, avg_time_on_page)
		VALUES ($1, 1, 1, 0)
		ON CONFLICT (doc_id) DO UPDATE SET views = docgen_analytics.views + 1
	`

	_, err := r.db.ExecContext(ctx, query, docID)
	if err != nil {
		return fmt.Errorf("failed to record doc view: %w", err)
	}
	return nil
}
