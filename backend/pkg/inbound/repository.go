package inbound

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Repository defines the data access interface for inbound webhooks
type Repository interface {
	// InboundSource CRUD
	CreateSource(ctx context.Context, source *InboundSource) error
	GetSource(ctx context.Context, sourceID string) (*InboundSource, error)
	GetSourceByTenant(ctx context.Context, tenantID, sourceID string) (*InboundSource, error)
	ListSources(ctx context.Context, tenantID string, limit, offset int) ([]InboundSource, int, error)
	UpdateSource(ctx context.Context, source *InboundSource) error
	DeleteSource(ctx context.Context, tenantID, sourceID string) error

	// RoutingRule CRUD
	CreateRoutingRule(ctx context.Context, rule *RoutingRule) error
	GetRoutingRules(ctx context.Context, sourceID string) ([]RoutingRule, error)
	UpdateRoutingRule(ctx context.Context, rule *RoutingRule) error
	DeleteRoutingRule(ctx context.Context, ruleID string) error

	// InboundEvent operations
	CreateEvent(ctx context.Context, event *InboundEvent) error
	GetEvent(ctx context.Context, eventID string) (*InboundEvent, error)
	GetEventByTenant(ctx context.Context, tenantID, eventID string) (*InboundEvent, error)
	ListEventsBySource(ctx context.Context, sourceID string, status string, limit, offset int) ([]InboundEvent, int, error)
	UpdateEventStatus(ctx context.Context, eventID, status, errorMsg string) error

	// DLQ operations
	GetDLQEntries(ctx context.Context, tenantID string, limit, offset int) ([]InboundDLQEntry, error)
	GetDLQEntry(ctx context.Context, tenantID, entryID string) (*InboundDLQEntry, error)
	MarkDLQEntryReplayed(ctx context.Context, entryID string) error

	// Transform rules
	CreateTransformRule(ctx context.Context, rule *TransformRule) error
	ListTransformRules(ctx context.Context, sourceID string) ([]TransformRule, error)
	DeleteTransformRule(ctx context.Context, ruleID string) error

	// Content routes
	CreateContentRoute(ctx context.Context, route *ContentRoute) error
	ListContentRoutes(ctx context.Context, sourceID string) ([]ContentRoute, error)
	DeleteContentRoute(ctx context.Context, routeID string) error

	// Stats and health
	GetProviderHealth(ctx context.Context, sourceID string) (*ProviderHealth, error)
	GetRateLimitConfig(ctx context.Context, sourceID string) (*RateLimitConfig, error)
	GetInboundStats(ctx context.Context, sourceID string) (*InboundStats, error)
}

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateSource(ctx context.Context, source *InboundSource) error {
	query := `INSERT INTO inbound_sources (id, tenant_id, name, provider, verification_secret, verification_header, verification_algorithm, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := r.db.ExecContext(ctx, query,
		source.ID, source.TenantID, source.Name, source.Provider,
		source.VerificationSecret, source.VerificationHeader, source.VerificationAlgorithm,
		source.Status, source.CreatedAt, source.UpdatedAt)
	return err
}

func (r *PostgresRepository) GetSource(ctx context.Context, sourceID string) (*InboundSource, error) {
	var source InboundSource
	err := r.db.GetContext(ctx, &source,
		`SELECT * FROM inbound_sources WHERE id = $1`, sourceID)
	if err != nil {
		return nil, err
	}
	return &source, nil
}

func (r *PostgresRepository) GetSourceByTenant(ctx context.Context, tenantID, sourceID string) (*InboundSource, error) {
	var source InboundSource
	err := r.db.GetContext(ctx, &source,
		`SELECT * FROM inbound_sources WHERE id = $1 AND tenant_id = $2`, sourceID, tenantID)
	if err != nil {
		return nil, err
	}
	return &source, nil
}

func (r *PostgresRepository) ListSources(ctx context.Context, tenantID string, limit, offset int) ([]InboundSource, int, error) {
	var total int
	err := r.db.GetContext(ctx, &total,
		`SELECT COUNT(*) FROM inbound_sources WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return nil, 0, err
	}

	var sources []InboundSource
	err = r.db.SelectContext(ctx, &sources,
		`SELECT * FROM inbound_sources WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		tenantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return sources, total, nil
}

func (r *PostgresRepository) UpdateSource(ctx context.Context, source *InboundSource) error {
	query := `UPDATE inbound_sources SET name = $1, verification_secret = $2, verification_header = $3,
		verification_algorithm = $4, status = $5, updated_at = $6
		WHERE id = $7 AND tenant_id = $8`
	_, err := r.db.ExecContext(ctx, query,
		source.Name, source.VerificationSecret, source.VerificationHeader,
		source.VerificationAlgorithm, source.Status, source.UpdatedAt,
		source.ID, source.TenantID)
	return err
}

func (r *PostgresRepository) DeleteSource(ctx context.Context, tenantID, sourceID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM inbound_sources WHERE id = $1 AND tenant_id = $2`, sourceID, tenantID)
	return err
}

func (r *PostgresRepository) CreateRoutingRule(ctx context.Context, rule *RoutingRule) error {
	query := `INSERT INTO inbound_routing_rules (id, source_id, filter_expression, destination_type, destination_config, priority, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.ExecContext(ctx, query,
		rule.ID, rule.SourceID, rule.FilterExpression, rule.DestinationType,
		rule.DestinationConfig, rule.Priority, rule.Active)
	return err
}

func (r *PostgresRepository) GetRoutingRules(ctx context.Context, sourceID string) ([]RoutingRule, error) {
	var rules []RoutingRule
	err := r.db.SelectContext(ctx, &rules,
		`SELECT * FROM inbound_routing_rules WHERE source_id = $1 ORDER BY priority ASC`, sourceID)
	if err != nil {
		return nil, err
	}
	return rules, nil
}

func (r *PostgresRepository) UpdateRoutingRule(ctx context.Context, rule *RoutingRule) error {
	query := `UPDATE inbound_routing_rules SET filter_expression = $1, destination_type = $2,
		destination_config = $3, priority = $4, active = $5 WHERE id = $6`
	_, err := r.db.ExecContext(ctx, query,
		rule.FilterExpression, rule.DestinationType, rule.DestinationConfig,
		rule.Priority, rule.Active, rule.ID)
	return err
}

func (r *PostgresRepository) DeleteRoutingRule(ctx context.Context, ruleID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM inbound_routing_rules WHERE id = $1`, ruleID)
	return err
}

func (r *PostgresRepository) CreateEvent(ctx context.Context, event *InboundEvent) error {
	query := `INSERT INTO inbound_events (id, source_id, tenant_id, provider, raw_payload, normalized_payload, headers, signature_valid, status, error_message, processed_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err := r.db.ExecContext(ctx, query,
		event.ID, event.SourceID, event.TenantID, event.Provider, event.RawPayload,
		event.NormalizedPayload, event.Headers, event.SignatureValid, event.Status,
		event.ErrorMessage, event.ProcessedAt, event.CreatedAt)
	return err
}

func (r *PostgresRepository) GetEvent(ctx context.Context, eventID string) (*InboundEvent, error) {
	var event InboundEvent
	err := r.db.GetContext(ctx, &event,
		`SELECT * FROM inbound_events WHERE id = $1`, eventID)
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *PostgresRepository) GetEventByTenant(ctx context.Context, tenantID, eventID string) (*InboundEvent, error) {
	var event InboundEvent
	err := r.db.GetContext(ctx, &event,
		`SELECT * FROM inbound_events WHERE id = $1 AND tenant_id = $2`, eventID, tenantID)
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *PostgresRepository) ListEventsBySource(ctx context.Context, sourceID string, status string, limit, offset int) ([]InboundEvent, int, error) {
	var total int
	var err error

	if status != "" {
		err = r.db.GetContext(ctx, &total,
			`SELECT COUNT(*) FROM inbound_events WHERE source_id = $1 AND status = $2`, sourceID, status)
	} else {
		err = r.db.GetContext(ctx, &total,
			`SELECT COUNT(*) FROM inbound_events WHERE source_id = $1`, sourceID)
	}
	if err != nil {
		return nil, 0, err
	}

	var events []InboundEvent
	if status != "" {
		err = r.db.SelectContext(ctx, &events,
			`SELECT * FROM inbound_events WHERE source_id = $1 AND status = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
			sourceID, status, limit, offset)
	} else {
		err = r.db.SelectContext(ctx, &events,
			`SELECT * FROM inbound_events WHERE source_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
			sourceID, limit, offset)
	}
	if err != nil {
		return nil, 0, err
	}

	return events, total, nil
}

func (r *PostgresRepository) UpdateEventStatus(ctx context.Context, eventID, status, errorMsg string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE inbound_events SET status = $1, error_message = $2 WHERE id = $3`,
		status, errorMsg, eventID)
	return err
}

func (r *PostgresRepository) GetDLQEntries(ctx context.Context, tenantID string, limit, offset int) ([]InboundDLQEntry, error) {
	var entries []InboundDLQEntry
	err := r.db.SelectContext(ctx, &entries,
		`SELECT * FROM inbound_dlq WHERE tenant_id = $1 AND replayed = false ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		tenantID, limit, offset)
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func (r *PostgresRepository) GetDLQEntry(ctx context.Context, tenantID, entryID string) (*InboundDLQEntry, error) {
	var entry InboundDLQEntry
	err := r.db.GetContext(ctx, &entry,
		`SELECT * FROM inbound_dlq WHERE id = $1 AND tenant_id = $2`, entryID, tenantID)
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func (r *PostgresRepository) MarkDLQEntryReplayed(ctx context.Context, entryID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE inbound_dlq SET replayed = true, last_attempt = $1 WHERE id = $2`,
		time.Now(), entryID)
	return err
}

func (r *PostgresRepository) GetProviderHealth(ctx context.Context, sourceID string) (*ProviderHealth, error) {
	var health ProviderHealth
	health.SourceID = sourceID

	row := r.db.QueryRowContext(ctx, `SELECT
		s.provider,
		COUNT(e.id) AS events_last_24h,
		COUNT(CASE WHEN e.status = 'failed' THEN 1 END) AS errors_last_24h,
		MAX(e.created_at) AS last_event_at,
		MAX(CASE WHEN e.status = 'failed' THEN e.created_at END) AS last_error_at
		FROM inbound_sources s
		LEFT JOIN inbound_events e ON e.source_id = s.id AND e.created_at > NOW() - INTERVAL '24 hours'
		WHERE s.id = $1
		GROUP BY s.provider`, sourceID)

	var lastEvent, lastError *time.Time
	err := row.Scan(&health.Provider, &health.EventsLast24h, &health.ErrorsLast24h, &lastEvent, &lastError)
	if err != nil {
		return nil, fmt.Errorf("failed to query provider health: %w", err)
	}
	health.LastEventAt = lastEvent
	health.LastErrorAt = lastError

	if health.EventsLast24h > 0 {
		health.SuccessRate = float64(health.EventsLast24h-health.ErrorsLast24h) / float64(health.EventsLast24h) * 100
	}

	health.Status = "healthy"
	if health.SuccessRate < 90 {
		health.Status = "degraded"
	}
	if health.SuccessRate < 50 || health.ConsecutiveErrors > 10 {
		health.Status = "down"
	}

	return &health, nil
}

func (r *PostgresRepository) GetRateLimitConfig(ctx context.Context, sourceID string) (*RateLimitConfig, error) {
	var config RateLimitConfig
	err := r.db.GetContext(ctx, &config,
		`SELECT source_id, requests_per_minute, burst_size, enabled FROM inbound_rate_limits WHERE source_id = $1`, sourceID)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (r *PostgresRepository) GetInboundStats(ctx context.Context, sourceID string) (*InboundStats, error) {
	var stats InboundStats
	row := r.db.QueryRowContext(ctx, `SELECT
		COUNT(*) AS total_events,
		COUNT(CASE WHEN status = 'validated' THEN 1 END) AS validated_count,
		COUNT(CASE WHEN status = 'routed' THEN 1 END) AS routed_count,
		COUNT(CASE WHEN status = 'failed' THEN 1 END) AS failed_count
		FROM inbound_events WHERE source_id = $1`, sourceID)

	err := row.Scan(&stats.TotalEvents, &stats.ValidatedCount, &stats.RoutedCount, &stats.FailedCount)
	if err != nil {
		return nil, fmt.Errorf("failed to query inbound stats: %w", err)
	}

	// Compute DLQ count
	_ = r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM inbound_dlq WHERE source_id = $1 AND replayed = false`, sourceID).
		Scan(&stats.DLQCount)

	if stats.TotalEvents > 0 {
		stats.SuccessRate = float64(stats.RoutedCount) / float64(stats.TotalEvents) * 100
	}

	return &stats, nil
}

func (r *PostgresRepository) CreateTransformRule(ctx context.Context, rule *TransformRule) error {
	query := `INSERT INTO inbound_transform_rules (id, source_id, field_path, transform_type, expression, target_field, priority, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.ExecContext(ctx, query, rule.ID, rule.SourceID, rule.FieldPath, rule.TransformType, rule.Expression, rule.TargetField, rule.Priority, rule.Active)
	return err
}

func (r *PostgresRepository) ListTransformRules(ctx context.Context, sourceID string) ([]TransformRule, error) {
	query := `SELECT id, source_id, field_path, transform_type, expression, target_field, priority, active FROM inbound_transform_rules WHERE source_id = $1 ORDER BY priority`
	rows, err := r.db.QueryContext(ctx, query, sourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rules []TransformRule
	for rows.Next() {
		var tr TransformRule
		if err := rows.Scan(&tr.ID, &tr.SourceID, &tr.FieldPath, &tr.TransformType, &tr.Expression, &tr.TargetField, &tr.Priority, &tr.Active); err != nil {
			return nil, err
		}
		rules = append(rules, tr)
	}
	return rules, rows.Err()
}

func (r *PostgresRepository) DeleteTransformRule(ctx context.Context, ruleID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM inbound_transform_rules WHERE id = $1`, ruleID)
	return err
}

func (r *PostgresRepository) CreateContentRoute(ctx context.Context, route *ContentRoute) error {
	query := `INSERT INTO inbound_content_routes (id, source_id, name, filter_expression, destination_type, destination_url, headers, active, priority)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	headersJSON, _ := json.Marshal(route.Headers)
	_, err := r.db.ExecContext(ctx, query, route.ID, route.SourceID, route.Name, route.FilterExpression, route.DestinationType, route.DestinationURL, string(headersJSON), route.Active, route.Priority)
	return err
}

func (r *PostgresRepository) ListContentRoutes(ctx context.Context, sourceID string) ([]ContentRoute, error) {
	query := `SELECT id, source_id, name, filter_expression, destination_type, destination_url, headers, active, priority FROM inbound_content_routes WHERE source_id = $1 ORDER BY priority`
	rows, err := r.db.QueryContext(ctx, query, sourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var routes []ContentRoute
	for rows.Next() {
		var cr ContentRoute
		var headersStr string
		if err := rows.Scan(&cr.ID, &cr.SourceID, &cr.Name, &cr.FilterExpression, &cr.DestinationType, &cr.DestinationURL, &headersStr, &cr.Active, &cr.Priority); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(headersStr), &cr.Headers)
		routes = append(routes, cr)
	}
	return routes, rows.Err()
}

func (r *PostgresRepository) DeleteContentRoute(ctx context.Context, routeID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM inbound_content_routes WHERE id = $1`, routeID)
	return err
}
