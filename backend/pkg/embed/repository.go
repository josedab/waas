package embed

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/josedab/waas/pkg/database"
)

// Repository defines the interface for embed token storage
type Repository interface {
	CreateToken(ctx context.Context, token *EmbedToken) error
	GetToken(ctx context.Context, tenantID, tokenID string) (*EmbedToken, error)
	GetTokenByValue(ctx context.Context, tokenValue string) (*EmbedToken, error)
	ListTokens(ctx context.Context, tenantID string, limit, offset int) ([]EmbedToken, int, error)
	UpdateToken(ctx context.Context, token *EmbedToken) error
	DeleteToken(ctx context.Context, tenantID, tokenID string) error

	CreateSession(ctx context.Context, session *EmbedSession) error
	UpdateSession(ctx context.Context, sessionID string) error
	GetSessionsByToken(ctx context.Context, tokenID string, limit int) ([]EmbedSession, error)

	GetDeliveryStats(ctx context.Context, tenantID string, scopes EmbedScopes) (*DeliveryStats, error)
	GetActivityFeed(ctx context.Context, tenantID string, scopes EmbedScopes, limit int) ([]ActivityItem, error)
	GetChartData(ctx context.Context, tenantID, chartType string, scopes EmbedScopes) (*ChartData, error)
	GetErrorSummary(ctx context.Context, tenantID string, scopes EmbedScopes) (*ErrorSummary, error)
}

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// CreateToken creates a new embed token
func (r *PostgresRepository) CreateToken(ctx context.Context, token *EmbedToken) error {
	if token.ID == "" {
		token.ID = uuid.New().String()
	}

	permissions := make([]string, len(token.Permissions))
	for i, p := range token.Permissions {
		permissions[i] = string(p)
	}

	scopesJSON, _ := json.Marshal(token.Scopes)
	themeJSON, _ := json.Marshal(token.Theme)
	metadataJSON, _ := json.Marshal(token.Metadata)

	query := `
		INSERT INTO embed_tokens 
		(id, tenant_id, name, token, permissions, scopes, theme, expires_at, allowed_origins, metadata, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := r.db.ExecContext(ctx, query,
		token.ID, token.TenantID, token.Name, token.Token,
		database.StringArray(permissions), scopesJSON, themeJSON,
		token.ExpiresAt, database.StringArray(token.AllowedOrigins), metadataJSON,
		token.IsActive, token.CreatedAt, token.UpdatedAt,
	)

	return err
}

// GetToken retrieves an embed token by ID
func (r *PostgresRepository) GetToken(ctx context.Context, tenantID, tokenID string) (*EmbedToken, error) {
	query := `
		SELECT id, tenant_id, name, token, permissions, scopes, theme, expires_at, allowed_origins, metadata, is_active, created_at, updated_at
		FROM embed_tokens
		WHERE id = $1 AND tenant_id = $2
	`

	return r.scanToken(ctx, query, tokenID, tenantID)
}

// GetTokenByValue retrieves an embed token by its value
func (r *PostgresRepository) GetTokenByValue(ctx context.Context, tokenValue string) (*EmbedToken, error) {
	query := `
		SELECT id, tenant_id, name, token, permissions, scopes, theme, expires_at, allowed_origins, metadata, is_active, created_at, updated_at
		FROM embed_tokens
		WHERE token = $1
	`

	return r.scanToken(ctx, query, tokenValue)
}

func (r *PostgresRepository) scanToken(ctx context.Context, query string, args ...interface{}) (*EmbedToken, error) {
	var token EmbedToken
	var permissions []string
	var scopesJSON, themeJSON, metadataJSON []byte
	var expiresAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&token.ID, &token.TenantID, &token.Name, &token.Token,
		(*database.StringArray)(&permissions), &scopesJSON, &themeJSON,
		&expiresAt, (*database.StringArray)(&token.AllowedOrigins), &metadataJSON,
		&token.IsActive, &token.CreatedAt, &token.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	token.Permissions = make([]Permission, len(permissions))
	for i, p := range permissions {
		token.Permissions[i] = Permission(p)
	}
	json.Unmarshal(scopesJSON, &token.Scopes)
	json.Unmarshal(themeJSON, &token.Theme)
	json.Unmarshal(metadataJSON, &token.Metadata)
	if expiresAt.Valid {
		token.ExpiresAt = &expiresAt.Time
	}

	return &token, nil
}

// ListTokens lists embed tokens for a tenant
func (r *PostgresRepository) ListTokens(ctx context.Context, tenantID string, limit, offset int) ([]EmbedToken, int, error) {
	countQuery := `SELECT COUNT(*) FROM embed_tokens WHERE tenant_id = $1`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, tenantID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, tenant_id, name, token, permissions, scopes, theme, expires_at, allowed_origins, metadata, is_active, created_at, updated_at
		FROM embed_tokens
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tokens []EmbedToken
	for rows.Next() {
		var token EmbedToken
		var permissions []string
		var scopesJSON, themeJSON, metadataJSON []byte
		var expiresAt sql.NullTime

		if err := rows.Scan(
			&token.ID, &token.TenantID, &token.Name, &token.Token,
			(*database.StringArray)(&permissions), &scopesJSON, &themeJSON,
			&expiresAt, (*database.StringArray)(&token.AllowedOrigins), &metadataJSON,
			&token.IsActive, &token.CreatedAt, &token.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}

		token.Permissions = make([]Permission, len(permissions))
		for i, p := range permissions {
			token.Permissions[i] = Permission(p)
		}
		json.Unmarshal(scopesJSON, &token.Scopes)
		json.Unmarshal(themeJSON, &token.Theme)
		json.Unmarshal(metadataJSON, &token.Metadata)
		if expiresAt.Valid {
			token.ExpiresAt = &expiresAt.Time
		}

		tokens = append(tokens, token)
	}

	return tokens, total, nil
}

// UpdateToken updates an embed token
func (r *PostgresRepository) UpdateToken(ctx context.Context, token *EmbedToken) error {
	permissions := make([]string, len(token.Permissions))
	for i, p := range token.Permissions {
		permissions[i] = string(p)
	}

	scopesJSON, _ := json.Marshal(token.Scopes)
	themeJSON, _ := json.Marshal(token.Theme)
	metadataJSON, _ := json.Marshal(token.Metadata)

	query := `
		UPDATE embed_tokens
		SET name = $1, token = $2, permissions = $3, scopes = $4, theme = $5, 
		    expires_at = $6, allowed_origins = $7, metadata = $8, is_active = $9, updated_at = $10
		WHERE id = $11 AND tenant_id = $12
	`

	_, err := r.db.ExecContext(ctx, query,
		token.Name, token.Token, database.StringArray(permissions), scopesJSON, themeJSON,
		token.ExpiresAt, database.StringArray(token.AllowedOrigins), metadataJSON,
		token.IsActive, token.UpdatedAt,
		token.ID, token.TenantID,
	)

	return err
}

// DeleteToken deletes an embed token
func (r *PostgresRepository) DeleteToken(ctx context.Context, tenantID, tokenID string) error {
	query := `DELETE FROM embed_tokens WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.ExecContext(ctx, query, tokenID, tenantID)
	return err
}

// CreateSession creates a new embed session
func (r *PostgresRepository) CreateSession(ctx context.Context, session *EmbedSession) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}

	query := `
		INSERT INTO embed_sessions (id, token_id, tenant_id, origin, user_agent, ip, created_at, last_seen)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(ctx, query,
		session.ID, session.TokenID, session.TenantID, session.Origin,
		session.UserAgent, session.IP, session.CreatedAt, session.LastSeen,
	)

	return err
}

// UpdateSession updates session last seen
func (r *PostgresRepository) UpdateSession(ctx context.Context, sessionID string) error {
	query := `UPDATE embed_sessions SET last_seen = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now(), sessionID)
	return err
}

// GetSessionsByToken retrieves sessions for a token
func (r *PostgresRepository) GetSessionsByToken(ctx context.Context, tokenID string, limit int) ([]EmbedSession, error) {
	query := `
		SELECT id, token_id, tenant_id, origin, user_agent, ip, created_at, last_seen
		FROM embed_sessions
		WHERE token_id = $1
		ORDER BY last_seen DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, tokenID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []EmbedSession
	for rows.Next() {
		var s EmbedSession
		if err := rows.Scan(
			&s.ID, &s.TokenID, &s.TenantID, &s.Origin,
			&s.UserAgent, &s.IP, &s.CreatedAt, &s.LastSeen,
		); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}

	return sessions, nil
}

// GetDeliveryStats retrieves delivery statistics (simplified implementation)
func (r *PostgresRepository) GetDeliveryStats(ctx context.Context, tenantID string, scopes EmbedScopes) (*DeliveryStats, error) {
	// In production, this would query the deliveries table with scope filters
	stats := &DeliveryStats{
		TotalDeliveries: 0,
		Successful:      0,
		Failed:          0,
		Pending:         0,
		SuccessRate:     0,
		AvgLatencyMs:    0,
		Period:          scopes.TimeRange,
		UpdatedAt:       time.Now(),
	}

	// Query total deliveries
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'delivered') as successful,
			COUNT(*) FILTER (WHERE status = 'failed') as failed,
			COUNT(*) FILTER (WHERE status = 'pending') as pending,
			COALESCE(AVG(latency_ms), 0) as avg_latency
		FROM deliveries
		WHERE tenant_id = $1
	`

	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(
		&stats.TotalDeliveries,
		&stats.Successful,
		&stats.Failed,
		&stats.Pending,
		&stats.AvgLatencyMs,
	)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		// Return empty stats if table doesn't exist or error
		return stats, nil
	}

	if stats.TotalDeliveries > 0 {
		stats.SuccessRate = float64(stats.Successful) / float64(stats.TotalDeliveries) * 100
	}

	return stats, nil
}

// GetActivityFeed retrieves activity feed items
func (r *PostgresRepository) GetActivityFeed(ctx context.Context, tenantID string, scopes EmbedScopes, limit int) ([]ActivityItem, error) {
	// Simplified implementation - in production would query multiple sources
	return []ActivityItem{}, nil
}

// GetChartData retrieves chart data
func (r *PostgresRepository) GetChartData(ctx context.Context, tenantID, chartType string, scopes EmbedScopes) (*ChartData, error) {
	// Simplified implementation
	return &ChartData{
		Title:  chartType,
		Type:   "line",
		Series: []ChartSeries{},
		XAxis:  "time",
		YAxis:  "value",
		Period: scopes.TimeRange,
	}, nil
}

// GetErrorSummary retrieves error summary
func (r *PostgresRepository) GetErrorSummary(ctx context.Context, tenantID string, scopes EmbedScopes) (*ErrorSummary, error) {
	// Simplified implementation
	return &ErrorSummary{
		TotalErrors: 0,
		ByCategory:  make(map[string]int64),
		ByEndpoint:  make(map[string]int64),
		TopErrors:   []ErrorDetail{},
		Period:      scopes.TimeRange,
	}, nil
}
