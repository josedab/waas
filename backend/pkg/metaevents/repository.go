package metaevents

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Repository defines the interface for meta-events storage
type Repository interface {
	CreateSubscription(ctx context.Context, sub *Subscription) error
	GetSubscription(ctx context.Context, tenantID, subID string) (*Subscription, error)
	ListSubscriptions(ctx context.Context, tenantID string, limit, offset int) ([]Subscription, int, error)
	UpdateSubscription(ctx context.Context, sub *Subscription) error
	DeleteSubscription(ctx context.Context, tenantID, subID string) error
	GetSubscriptionsByEventType(ctx context.Context, tenantID string, eventType EventType) ([]Subscription, error)

	CreateEvent(ctx context.Context, event *MetaEvent) error
	GetEvent(ctx context.Context, tenantID, eventID string) (*MetaEvent, error)
	ListEvents(ctx context.Context, tenantID string, eventType *EventType, limit, offset int) ([]MetaEvent, int, error)

	CreateDelivery(ctx context.Context, delivery *Delivery) error
	ListDeliveries(ctx context.Context, tenantID, subID string, limit, offset int) ([]Delivery, int, error)
}

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// CreateSubscription creates a new subscription
func (r *PostgresRepository) CreateSubscription(ctx context.Context, sub *Subscription) error {
	if sub.ID == "" {
		sub.ID = uuid.New().String()
	}

	eventTypes := make([]string, len(sub.EventTypes))
	for i, et := range sub.EventTypes {
		eventTypes[i] = string(et)
	}

	filtersJSON, _ := json.Marshal(sub.Filters)
	headersJSON, _ := json.Marshal(sub.Headers)
	retryPolicyJSON, _ := json.Marshal(sub.RetryPolicy)

	query := `
		INSERT INTO meta_event_subscriptions 
		(id, tenant_id, name, url, secret, event_types, filters, is_active, headers, retry_policy, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := r.db.ExecContext(ctx, query,
		sub.ID, sub.TenantID, sub.Name, sub.URL, sub.Secret,
		pq.Array(eventTypes), filtersJSON, sub.IsActive, headersJSON, retryPolicyJSON,
		sub.CreatedAt, sub.UpdatedAt,
	)

	return err
}

// GetSubscription retrieves a subscription by ID
func (r *PostgresRepository) GetSubscription(ctx context.Context, tenantID, subID string) (*Subscription, error) {
	query := `
		SELECT id, tenant_id, name, url, secret, event_types, filters, is_active, headers, retry_policy, created_at, updated_at
		FROM meta_event_subscriptions
		WHERE id = $1 AND tenant_id = $2
	`

	var sub Subscription
	var eventTypes []string
	var filtersJSON, headersJSON, retryPolicyJSON []byte

	err := r.db.QueryRowContext(ctx, query, subID, tenantID).Scan(
		&sub.ID, &sub.TenantID, &sub.Name, &sub.URL, &sub.Secret,
		pq.Array(&eventTypes), &filtersJSON, &sub.IsActive, &headersJSON, &retryPolicyJSON,
		&sub.CreatedAt, &sub.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	sub.EventTypes = make([]EventType, len(eventTypes))
	for i, et := range eventTypes {
		sub.EventTypes[i] = EventType(et)
	}
	json.Unmarshal(filtersJSON, &sub.Filters)
	json.Unmarshal(headersJSON, &sub.Headers)
	json.Unmarshal(retryPolicyJSON, &sub.RetryPolicy)

	return &sub, nil
}

// ListSubscriptions lists subscriptions for a tenant
func (r *PostgresRepository) ListSubscriptions(ctx context.Context, tenantID string, limit, offset int) ([]Subscription, int, error) {
	countQuery := `SELECT COUNT(*) FROM meta_event_subscriptions WHERE tenant_id = $1`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, tenantID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, tenant_id, name, url, secret, event_types, filters, is_active, headers, retry_policy, created_at, updated_at
		FROM meta_event_subscriptions
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var subs []Subscription
	for rows.Next() {
		var sub Subscription
		var eventTypes []string
		var filtersJSON, headersJSON, retryPolicyJSON []byte

		if err := rows.Scan(
			&sub.ID, &sub.TenantID, &sub.Name, &sub.URL, &sub.Secret,
			pq.Array(&eventTypes), &filtersJSON, &sub.IsActive, &headersJSON, &retryPolicyJSON,
			&sub.CreatedAt, &sub.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}

		sub.EventTypes = make([]EventType, len(eventTypes))
		for i, et := range eventTypes {
			sub.EventTypes[i] = EventType(et)
		}
		json.Unmarshal(filtersJSON, &sub.Filters)
		json.Unmarshal(headersJSON, &sub.Headers)
		json.Unmarshal(retryPolicyJSON, &sub.RetryPolicy)

		subs = append(subs, sub)
	}

	return subs, total, nil
}

// UpdateSubscription updates a subscription
func (r *PostgresRepository) UpdateSubscription(ctx context.Context, sub *Subscription) error {
	eventTypes := make([]string, len(sub.EventTypes))
	for i, et := range sub.EventTypes {
		eventTypes[i] = string(et)
	}

	filtersJSON, _ := json.Marshal(sub.Filters)
	headersJSON, _ := json.Marshal(sub.Headers)
	retryPolicyJSON, _ := json.Marshal(sub.RetryPolicy)

	query := `
		UPDATE meta_event_subscriptions
		SET name = $1, url = $2, secret = $3, event_types = $4, filters = $5, 
		    is_active = $6, headers = $7, retry_policy = $8, updated_at = $9
		WHERE id = $10 AND tenant_id = $11
	`

	_, err := r.db.ExecContext(ctx, query,
		sub.Name, sub.URL, sub.Secret, pq.Array(eventTypes), filtersJSON,
		sub.IsActive, headersJSON, retryPolicyJSON, sub.UpdatedAt,
		sub.ID, sub.TenantID,
	)

	return err
}

// DeleteSubscription deletes a subscription
func (r *PostgresRepository) DeleteSubscription(ctx context.Context, tenantID, subID string) error {
	query := `DELETE FROM meta_event_subscriptions WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.ExecContext(ctx, query, subID, tenantID)
	return err
}

// GetSubscriptionsByEventType retrieves active subscriptions matching an event type
func (r *PostgresRepository) GetSubscriptionsByEventType(ctx context.Context, tenantID string, eventType EventType) ([]Subscription, error) {
	query := `
		SELECT id, tenant_id, name, url, secret, event_types, filters, is_active, headers, retry_policy, created_at, updated_at
		FROM meta_event_subscriptions
		WHERE tenant_id = $1 AND is_active = true AND $2 = ANY(event_types)
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, string(eventType))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []Subscription
	for rows.Next() {
		var sub Subscription
		var eventTypes []string
		var filtersJSON, headersJSON, retryPolicyJSON []byte

		if err := rows.Scan(
			&sub.ID, &sub.TenantID, &sub.Name, &sub.URL, &sub.Secret,
			pq.Array(&eventTypes), &filtersJSON, &sub.IsActive, &headersJSON, &retryPolicyJSON,
			&sub.CreatedAt, &sub.UpdatedAt,
		); err != nil {
			return nil, err
		}

		sub.EventTypes = make([]EventType, len(eventTypes))
		for i, et := range eventTypes {
			sub.EventTypes[i] = EventType(et)
		}
		json.Unmarshal(filtersJSON, &sub.Filters)
		json.Unmarshal(headersJSON, &sub.Headers)
		json.Unmarshal(retryPolicyJSON, &sub.RetryPolicy)

		subs = append(subs, sub)
	}

	return subs, nil
}

// CreateEvent creates a new meta-event
func (r *PostgresRepository) CreateEvent(ctx context.Context, event *MetaEvent) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	dataJSON, _ := json.Marshal(event.Data)
	metadataJSON, _ := json.Marshal(event.Metadata)

	query := `
		INSERT INTO meta_events (id, tenant_id, type, source, source_id, data, metadata, occurred_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.ExecContext(ctx, query,
		event.ID, event.TenantID, event.Type, event.Source, event.SourceID,
		dataJSON, metadataJSON, event.OccurredAt, event.CreatedAt,
	)

	return err
}

// GetEvent retrieves an event by ID
func (r *PostgresRepository) GetEvent(ctx context.Context, tenantID, eventID string) (*MetaEvent, error) {
	query := `
		SELECT id, tenant_id, type, source, source_id, data, metadata, occurred_at, delivered_at, created_at
		FROM meta_events
		WHERE id = $1 AND tenant_id = $2
	`

	var event MetaEvent
	var dataJSON, metadataJSON []byte
	var deliveredAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, eventID, tenantID).Scan(
		&event.ID, &event.TenantID, &event.Type, &event.Source, &event.SourceID,
		&dataJSON, &metadataJSON, &event.OccurredAt, &deliveredAt, &event.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(dataJSON, &event.Data)
	json.Unmarshal(metadataJSON, &event.Metadata)
	if deliveredAt.Valid {
		event.DeliveredAt = &deliveredAt.Time
	}

	return &event, nil
}

// ListEvents lists events for a tenant
func (r *PostgresRepository) ListEvents(ctx context.Context, tenantID string, eventType *EventType, limit, offset int) ([]MetaEvent, int, error) {
	var countQuery string
	var countArgs []interface{}
	if eventType != nil {
		countQuery = `SELECT COUNT(*) FROM meta_events WHERE tenant_id = $1 AND type = $2`
		countArgs = []interface{}{tenantID, string(*eventType)}
	} else {
		countQuery = `SELECT COUNT(*) FROM meta_events WHERE tenant_id = $1`
		countArgs = []interface{}{tenantID}
	}

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	var query string
	var args []interface{}
	if eventType != nil {
		query = `
			SELECT id, tenant_id, type, source, source_id, data, metadata, occurred_at, delivered_at, created_at
			FROM meta_events
			WHERE tenant_id = $1 AND type = $2
			ORDER BY created_at DESC
			LIMIT $3 OFFSET $4
		`
		args = []interface{}{tenantID, string(*eventType), limit, offset}
	} else {
		query = `
			SELECT id, tenant_id, type, source, source_id, data, metadata, occurred_at, delivered_at, created_at
			FROM meta_events
			WHERE tenant_id = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{tenantID, limit, offset}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []MetaEvent
	for rows.Next() {
		var event MetaEvent
		var dataJSON, metadataJSON []byte
		var deliveredAt sql.NullTime

		if err := rows.Scan(
			&event.ID, &event.TenantID, &event.Type, &event.Source, &event.SourceID,
			&dataJSON, &metadataJSON, &event.OccurredAt, &deliveredAt, &event.CreatedAt,
		); err != nil {
			return nil, 0, err
		}

		json.Unmarshal(dataJSON, &event.Data)
		json.Unmarshal(metadataJSON, &event.Metadata)
		if deliveredAt.Valid {
			event.DeliveredAt = &deliveredAt.Time
		}

		events = append(events, event)
	}

	return events, total, nil
}

// CreateDelivery creates a delivery record
func (r *PostgresRepository) CreateDelivery(ctx context.Context, delivery *Delivery) error {
	if delivery.ID == "" {
		delivery.ID = uuid.New().String()
	}

	query := `
		INSERT INTO meta_event_deliveries 
		(id, subscription_id, event_id, tenant_id, status, attempt, status_code, response_body, error, latency_ms, next_retry, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := r.db.ExecContext(ctx, query,
		delivery.ID, delivery.SubscriptionID, delivery.EventID, delivery.TenantID,
		delivery.Status, delivery.Attempt, delivery.StatusCode, delivery.ResponseBody,
		delivery.Error, delivery.LatencyMs, delivery.NextRetry, delivery.CreatedAt,
	)

	return err
}

// ListDeliveries lists deliveries for a subscription
func (r *PostgresRepository) ListDeliveries(ctx context.Context, tenantID, subID string, limit, offset int) ([]Delivery, int, error) {
	countQuery := `SELECT COUNT(*) FROM meta_event_deliveries WHERE tenant_id = $1 AND subscription_id = $2`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, tenantID, subID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, subscription_id, event_id, tenant_id, status, attempt, status_code, response_body, error, latency_ms, next_retry, created_at
		FROM meta_event_deliveries
		WHERE tenant_id = $1 AND subscription_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, subID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var deliveries []Delivery
	for rows.Next() {
		var d Delivery
		var nextRetry sql.NullTime

		if err := rows.Scan(
			&d.ID, &d.SubscriptionID, &d.EventID, &d.TenantID,
			&d.Status, &d.Attempt, &d.StatusCode, &d.ResponseBody,
			&d.Error, &d.LatencyMs, &nextRetry, &d.CreatedAt,
		); err != nil {
			return nil, 0, err
		}

		if nextRetry.Valid {
			d.NextRetry = &nextRetry.Time
		}

		deliveries = append(deliveries, d)
	}

	return deliveries, total, nil
}
