package callback

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines the data access interface for callback resources
type Repository interface {
	// Callback requests
	CreateCallbackRequest(ctx context.Context, req *CallbackRequest) error
	GetCallbackRequest(ctx context.Context, tenantID, requestID uuid.UUID) (*CallbackRequest, error)
	UpdateCallbackStatus(ctx context.Context, requestID uuid.UUID, status CallbackStatus) error
	ListCallbackRequests(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]CallbackRequest, int, error)

	// Callback responses
	SaveCallbackResponse(ctx context.Context, resp *CallbackResponse) error
	GetCallbackResponse(ctx context.Context, responseID uuid.UUID) (*CallbackResponse, error)
	GetResponseByCorrelation(ctx context.Context, correlationID string) (*CallbackResponse, error)

	// Correlation tracking
	CreateCorrelation(ctx context.Context, entry *CorrelationEntry) error
	GetCorrelation(ctx context.Context, correlationID string) (*CorrelationEntry, error)
	UpdateCorrelation(ctx context.Context, entry *CorrelationEntry) error

	// Long-poll sessions
	CreateLongPollSession(ctx context.Context, session *LongPollSession) error
	GetLongPollSession(ctx context.Context, sessionID uuid.UUID) (*LongPollSession, error)
	UpdateLongPollSession(ctx context.Context, session *LongPollSession) error

	// Patterns
	SavePattern(ctx context.Context, pattern *CallbackPattern) error
	GetPatterns(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]CallbackPattern, int, error)
	GetPattern(ctx context.Context, tenantID, patternID uuid.UUID) (*CallbackPattern, error)

	// Metrics
	GetCallbackMetrics(ctx context.Context, tenantID uuid.UUID) (*CallbackMetrics, error)
}

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateCallbackRequest(ctx context.Context, req *CallbackRequest) error {
	query := `INSERT INTO callback_requests (id, tenant_id, webhook_id, correlation_id, callback_url, payload, headers, timeout_ms, status, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	_, err := r.db.ExecContext(ctx, query,
		req.ID, req.TenantID, req.WebhookID, req.CorrelationID, req.CallbackURL,
		req.Payload, req.Headers, req.TimeoutMs, req.Status, req.CreatedAt, req.ExpiresAt)
	return err
}

func (r *PostgresRepository) GetCallbackRequest(ctx context.Context, tenantID, requestID uuid.UUID) (*CallbackRequest, error) {
	var req CallbackRequest
	err := r.db.GetContext(ctx, &req,
		`SELECT * FROM callback_requests WHERE id = $1 AND tenant_id = $2`, requestID, tenantID)
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *PostgresRepository) UpdateCallbackStatus(ctx context.Context, requestID uuid.UUID, status CallbackStatus) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE callback_requests SET status = $1 WHERE id = $2`, status, requestID)
	return err
}

func (r *PostgresRepository) ListCallbackRequests(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]CallbackRequest, int, error) {
	var total int
	err := r.db.GetContext(ctx, &total,
		`SELECT COUNT(*) FROM callback_requests WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return nil, 0, err
	}

	var requests []CallbackRequest
	err = r.db.SelectContext(ctx, &requests,
		`SELECT * FROM callback_requests WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		tenantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return requests, total, nil
}

func (r *PostgresRepository) SaveCallbackResponse(ctx context.Context, resp *CallbackResponse) error {
	query := `INSERT INTO callback_responses (id, request_id, correlation_id, status_code, body, headers, received_at, latency_ms)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.ExecContext(ctx, query,
		resp.ID, resp.RequestID, resp.CorrelationID, resp.StatusCode,
		resp.Body, resp.Headers, resp.ReceivedAt, resp.LatencyMs)
	return err
}

func (r *PostgresRepository) GetCallbackResponse(ctx context.Context, responseID uuid.UUID) (*CallbackResponse, error) {
	var resp CallbackResponse
	err := r.db.GetContext(ctx, &resp,
		`SELECT * FROM callback_responses WHERE id = $1`, responseID)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (r *PostgresRepository) GetResponseByCorrelation(ctx context.Context, correlationID string) (*CallbackResponse, error) {
	var resp CallbackResponse
	err := r.db.GetContext(ctx, &resp,
		`SELECT * FROM callback_responses WHERE correlation_id = $1`, correlationID)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (r *PostgresRepository) CreateCorrelation(ctx context.Context, entry *CorrelationEntry) error {
	query := `INSERT INTO callback_correlations (id, correlation_id, tenant_id, request_id, response_id, status, created_at, ttl)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.ExecContext(ctx, query,
		entry.ID, entry.CorrelationID, entry.TenantID, entry.RequestID,
		entry.ResponseID, entry.Status, entry.CreatedAt, entry.TTL)
	return err
}

func (r *PostgresRepository) GetCorrelation(ctx context.Context, correlationID string) (*CorrelationEntry, error) {
	var entry CorrelationEntry
	err := r.db.GetContext(ctx, &entry,
		`SELECT * FROM callback_correlations WHERE correlation_id = $1`, correlationID)
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func (r *PostgresRepository) UpdateCorrelation(ctx context.Context, entry *CorrelationEntry) error {
	query := `UPDATE callback_correlations SET response_id = $1, status = $2 WHERE correlation_id = $3`
	_, err := r.db.ExecContext(ctx, query,
		entry.ResponseID, entry.Status, entry.CorrelationID)
	return err
}

func (r *PostgresRepository) CreateLongPollSession(ctx context.Context, session *LongPollSession) error {
	query := `INSERT INTO callback_longpoll_sessions (id, tenant_id, endpoint_id, filters, timeout_ms, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.ExecContext(ctx, query,
		session.ID, session.TenantID, session.EndpointID, session.Filters,
		session.TimeoutMs, session.Status, session.CreatedAt)
	return err
}

func (r *PostgresRepository) GetLongPollSession(ctx context.Context, sessionID uuid.UUID) (*LongPollSession, error) {
	var session LongPollSession
	err := r.db.GetContext(ctx, &session,
		`SELECT * FROM callback_longpoll_sessions WHERE id = $1`, sessionID)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *PostgresRepository) UpdateLongPollSession(ctx context.Context, session *LongPollSession) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE callback_longpoll_sessions SET status = $1 WHERE id = $2`,
		session.Status, session.ID)
	return err
}

func (r *PostgresRepository) SavePattern(ctx context.Context, pattern *CallbackPattern) error {
	query := `INSERT INTO callback_patterns (id, tenant_id, name, description, request_template, expected_response_schema, timeout_ms, max_retries, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.db.ExecContext(ctx, query,
		pattern.ID, pattern.TenantID, pattern.Name, pattern.Description,
		pattern.RequestTemplate, pattern.ExpectedResponseSchema,
		pattern.TimeoutMs, pattern.MaxRetries, pattern.CreatedAt)
	return err
}

func (r *PostgresRepository) GetPatterns(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]CallbackPattern, int, error) {
	var total int
	err := r.db.GetContext(ctx, &total,
		`SELECT COUNT(*) FROM callback_patterns WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return nil, 0, err
	}

	var patterns []CallbackPattern
	err = r.db.SelectContext(ctx, &patterns,
		`SELECT * FROM callback_patterns WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		tenantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return patterns, total, nil
}

func (r *PostgresRepository) GetPattern(ctx context.Context, tenantID, patternID uuid.UUID) (*CallbackPattern, error) {
	var pattern CallbackPattern
	err := r.db.GetContext(ctx, &pattern,
		`SELECT * FROM callback_patterns WHERE id = $1 AND tenant_id = $2`, patternID, tenantID)
	if err != nil {
		return nil, err
	}
	return &pattern, nil
}

func (r *PostgresRepository) GetCallbackMetrics(ctx context.Context, tenantID uuid.UUID) (*CallbackMetrics, error) {
	var metrics CallbackMetrics

	err := r.db.GetContext(ctx, &metrics.TotalRequests,
		`SELECT COUNT(*) FROM callback_requests WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return nil, err
	}

	if metrics.TotalRequests == 0 {
		return &metrics, nil
	}

	_ = r.db.GetContext(ctx, &metrics.SuccessRate,
		`SELECT COALESCE(COUNT(*) FILTER (WHERE status = 'received')::float / NULLIF(COUNT(*), 0), 0) FROM callback_requests WHERE tenant_id = $1`, tenantID)

	_ = r.db.GetContext(ctx, &metrics.AvgLatencyMs,
		`SELECT COALESCE(AVG(latency_ms), 0) FROM callback_responses cr JOIN callback_requests cq ON cr.request_id = cq.id WHERE cq.tenant_id = $1`, tenantID)

	_ = r.db.GetContext(ctx, &metrics.TimeoutRate,
		`SELECT COALESCE(COUNT(*) FILTER (WHERE status = 'timeout')::float / NULLIF(COUNT(*), 0), 0) FROM callback_requests WHERE tenant_id = $1`, tenantID)

	_ = r.db.GetContext(ctx, &metrics.PendingCallbacks,
		`SELECT COUNT(*) FROM callback_requests WHERE tenant_id = $1 AND status IN ('pending', 'waiting')`, tenantID)

	return &metrics, nil
}
