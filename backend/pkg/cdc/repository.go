package cdc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Repository defines the interface for CDC storage
type Repository interface {
	// Connector operations
	SaveConnector(ctx context.Context, conn *CDCConnector) error
	GetConnector(ctx context.Context, tenantID, connID string) (*CDCConnector, error)
	ListConnectors(ctx context.Context, tenantID string, status *ConnectorStatus) ([]CDCConnector, error)
	DeleteConnector(ctx context.Context, tenantID, connID string) error
	UpdateConnectorStatus(ctx context.Context, tenantID, connID string, status ConnectorStatus, errMsg string) error

	// Offset operations
	SaveOffset(ctx context.Context, state *OffsetState) error
	GetOffset(ctx context.Context, tenantID, connID string) (*OffsetState, error)

	// Event history
	SaveEventHistory(ctx context.Context, history *EventHistory) error
	GetEventHistory(ctx context.Context, tenantID, connID string, limit, offset int) ([]EventHistory, error)

	// Metrics
	GetConnectorMetrics(ctx context.Context, tenantID, connID string) (*CDCMetrics, error)
	IncrementEventCount(ctx context.Context, tenantID, connID string, success bool) error
}

// PostgresRepository implements Repository for PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) SaveConnector(ctx context.Context, conn *CDCConnector) error {
	connConfigJSON, _ := json.Marshal(conn.ConnectionConfig)
	captureConfigJSON, _ := json.Marshal(conn.CaptureConfig)
	webhookConfigJSON, _ := json.Marshal(conn.WebhookConfig)
	offsetConfigJSON, _ := json.Marshal(conn.OffsetConfig)

	query := `
		INSERT INTO cdc_connectors (
			id, tenant_id, name, type, status, connection_config,
			capture_config, webhook_config, offset_config, last_offset,
			last_event_at, error_message, events_processed, events_failed,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			status = EXCLUDED.status,
			connection_config = EXCLUDED.connection_config,
			capture_config = EXCLUDED.capture_config,
			webhook_config = EXCLUDED.webhook_config,
			offset_config = EXCLUDED.offset_config,
			last_offset = EXCLUDED.last_offset,
			last_event_at = EXCLUDED.last_event_at,
			error_message = EXCLUDED.error_message,
			events_processed = EXCLUDED.events_processed,
			events_failed = EXCLUDED.events_failed,
			updated_at = EXCLUDED.updated_at`

	_, err := r.db.ExecContext(ctx, query,
		conn.ID, conn.TenantID, conn.Name, conn.Type, conn.Status,
		connConfigJSON, captureConfigJSON, webhookConfigJSON, offsetConfigJSON,
		conn.LastOffset, conn.LastEventAt, conn.ErrorMessage,
		conn.EventsProcessed, conn.EventsFailed, conn.CreatedAt, conn.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetConnector(ctx context.Context, tenantID, connID string) (*CDCConnector, error) {
	query := `
		SELECT id, tenant_id, name, type, status, connection_config,
			capture_config, webhook_config, offset_config, last_offset,
			last_event_at, error_message, events_processed, events_failed,
			created_at, updated_at
		FROM cdc_connectors
		WHERE tenant_id = $1 AND id = $2`

	var conn CDCConnector
	var connConfigJSON, captureConfigJSON, webhookConfigJSON, offsetConfigJSON []byte

	err := r.db.QueryRowContext(ctx, query, tenantID, connID).Scan(
		&conn.ID, &conn.TenantID, &conn.Name, &conn.Type, &conn.Status,
		&connConfigJSON, &captureConfigJSON, &webhookConfigJSON, &offsetConfigJSON,
		&conn.LastOffset, &conn.LastEventAt, &conn.ErrorMessage,
		&conn.EventsProcessed, &conn.EventsFailed, &conn.CreatedAt, &conn.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(connConfigJSON, &conn.ConnectionConfig)
	json.Unmarshal(captureConfigJSON, &conn.CaptureConfig)
	json.Unmarshal(webhookConfigJSON, &conn.WebhookConfig)
	json.Unmarshal(offsetConfigJSON, &conn.OffsetConfig)

	return &conn, nil
}

func (r *PostgresRepository) ListConnectors(ctx context.Context, tenantID string, status *ConnectorStatus) ([]CDCConnector, error) {
	query := `
		SELECT id, tenant_id, name, type, status, connection_config,
			capture_config, webhook_config, offset_config, last_offset,
			last_event_at, error_message, events_processed, events_failed,
			created_at, updated_at
		FROM cdc_connectors
		WHERE tenant_id = $1`

	args := []interface{}{tenantID}
	if status != nil {
		query += " AND status = $2"
		args = append(args, *status)
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var connectors []CDCConnector
	for rows.Next() {
		var conn CDCConnector
		var connConfigJSON, captureConfigJSON, webhookConfigJSON, offsetConfigJSON []byte

		err := rows.Scan(
			&conn.ID, &conn.TenantID, &conn.Name, &conn.Type, &conn.Status,
			&connConfigJSON, &captureConfigJSON, &webhookConfigJSON, &offsetConfigJSON,
			&conn.LastOffset, &conn.LastEventAt, &conn.ErrorMessage,
			&conn.EventsProcessed, &conn.EventsFailed, &conn.CreatedAt, &conn.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		json.Unmarshal(connConfigJSON, &conn.ConnectionConfig)
		json.Unmarshal(captureConfigJSON, &conn.CaptureConfig)
		json.Unmarshal(webhookConfigJSON, &conn.WebhookConfig)
		json.Unmarshal(offsetConfigJSON, &conn.OffsetConfig)

		connectors = append(connectors, conn)
	}

	return connectors, nil
}

func (r *PostgresRepository) DeleteConnector(ctx context.Context, tenantID, connID string) error {
	// Delete history first
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM cdc_event_history WHERE tenant_id = $1 AND connector_id = $2",
		tenantID, connID)
	if err != nil {
		return err
	}

	// Delete offsets
	_, err = r.db.ExecContext(ctx,
		"DELETE FROM cdc_offsets WHERE tenant_id = $1 AND connector_id = $2",
		tenantID, connID)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx,
		"DELETE FROM cdc_connectors WHERE tenant_id = $1 AND id = $2",
		tenantID, connID)
	return err
}

func (r *PostgresRepository) UpdateConnectorStatus(ctx context.Context, tenantID, connID string, status ConnectorStatus, errMsg string) error {
	query := `
		UPDATE cdc_connectors
		SET status = $3, error_message = $4, updated_at = $5
		WHERE tenant_id = $1 AND id = $2`

	_, err := r.db.ExecContext(ctx, query, tenantID, connID, status, errMsg, time.Now())
	return err
}

func (r *PostgresRepository) SaveOffset(ctx context.Context, state *OffsetState) error {
	query := `
		INSERT INTO cdc_offsets (
			id, connector_id, tenant_id, "offset", transaction_id,
			committed, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (connector_id) DO UPDATE SET
			"offset" = EXCLUDED."offset",
			transaction_id = EXCLUDED.transaction_id,
			committed = EXCLUDED.committed,
			updated_at = EXCLUDED.updated_at`

	_, err := r.db.ExecContext(ctx, query,
		state.ID, state.ConnectorID, state.TenantID, state.Offset,
		state.TransactionID, state.Committed, state.CreatedAt, state.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetOffset(ctx context.Context, tenantID, connID string) (*OffsetState, error) {
	query := `
		SELECT id, connector_id, tenant_id, "offset", transaction_id,
			committed, created_at, updated_at
		FROM cdc_offsets
		WHERE tenant_id = $1 AND connector_id = $2`

	var state OffsetState
	err := r.db.QueryRowContext(ctx, query, tenantID, connID).Scan(
		&state.ID, &state.ConnectorID, &state.TenantID, &state.Offset,
		&state.TransactionID, &state.Committed, &state.CreatedAt, &state.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &state, nil
}

func (r *PostgresRepository) SaveEventHistory(ctx context.Context, history *EventHistory) error {
	query := `
		INSERT INTO cdc_event_history (
			id, connector_id, tenant_id, event_id, operation, table_name,
			webhook_id, delivery_id, status, error_message, "offset", processed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err := r.db.ExecContext(ctx, query,
		history.ID, history.ConnectorID, history.TenantID, history.EventID,
		history.Operation, history.TableName, history.WebhookID, history.DeliveryID,
		history.Status, history.ErrorMessage, history.Offset, history.ProcessedAt,
	)
	return err
}

func (r *PostgresRepository) GetEventHistory(ctx context.Context, tenantID, connID string, limit, offset int) ([]EventHistory, error) {
	query := `
		SELECT id, connector_id, tenant_id, event_id, operation, table_name,
			webhook_id, delivery_id, status, error_message, "offset", processed_at
		FROM cdc_event_history
		WHERE tenant_id = $1 AND connector_id = $2
		ORDER BY processed_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.db.QueryContext(ctx, query, tenantID, connID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []EventHistory
	for rows.Next() {
		var h EventHistory
		err := rows.Scan(
			&h.ID, &h.ConnectorID, &h.TenantID, &h.EventID, &h.Operation,
			&h.TableName, &h.WebhookID, &h.DeliveryID, &h.Status,
			&h.ErrorMessage, &h.Offset, &h.ProcessedAt,
		)
		if err != nil {
			return nil, err
		}
		history = append(history, h)
	}

	return history, nil
}

func (r *PostgresRepository) GetConnectorMetrics(ctx context.Context, tenantID, connID string) (*CDCMetrics, error) {
	conn, err := r.GetConnector(ctx, tenantID, connID)
	if err != nil {
		return nil, err
	}

	metrics := &CDCMetrics{
		ConnectorID:     conn.ID,
		TenantID:        conn.TenantID,
		Status:          string(conn.Status),
		TotalEvents:     conn.EventsProcessed,
		FailedEvents:    conn.EventsFailed,
		TablesMonitored: len(conn.CaptureConfig.Tables),
	}

	if conn.LastEventAt != nil {
		metrics.LastEventTime = *conn.LastEventAt
		metrics.LagMs = time.Since(*conn.LastEventAt).Milliseconds()
	}

	// Calculate events per second from last hour
	eventsQuery := `
		SELECT COUNT(*) FROM cdc_event_history
		WHERE tenant_id = $1 AND connector_id = $2
		AND processed_at >= $3`

	var recentCount int64
	r.db.QueryRowContext(ctx, eventsQuery, tenantID, connID,
		time.Now().Add(-1*time.Hour)).Scan(&recentCount)

	metrics.EventsPerSecond = float64(recentCount) / 3600.0

	return metrics, nil
}

func (r *PostgresRepository) IncrementEventCount(ctx context.Context, tenantID, connID string, success bool) error {
	var query string
	if success {
		query = `
			UPDATE cdc_connectors
			SET events_processed = events_processed + 1,
				last_event_at = $3,
				updated_at = $3
			WHERE tenant_id = $1 AND id = $2`
	} else {
		query = `
			UPDATE cdc_connectors
			SET events_failed = events_failed + 1,
				updated_at = $3
			WHERE tenant_id = $1 AND id = $2`
	}

	_, err := r.db.ExecContext(ctx, query, tenantID, connID, time.Now())
	return err
}

// SecretStore interface for credential storage
type SecretStore interface {
	StoreConnectionCredentials(ctx context.Context, connID string, password string) error
	GetConnectionCredentials(ctx context.Context, connID string) (string, error)
	DeleteConnectionCredentials(ctx context.Context, connID string) error
}

// InMemorySecretStore for development (use vault in production)
type InMemorySecretStore struct {
	secrets map[string]string
}

func NewInMemorySecretStore() *InMemorySecretStore {
	return &InMemorySecretStore{secrets: make(map[string]string)}
}

func (s *InMemorySecretStore) StoreConnectionCredentials(ctx context.Context, connID string, password string) error {
	s.secrets[connID] = password
	return nil
}

func (s *InMemorySecretStore) GetConnectionCredentials(ctx context.Context, connID string) (string, error) {
	if pwd, ok := s.secrets[connID]; ok {
		return pwd, nil
	}
	return "", fmt.Errorf("credentials not found")
}

func (s *InMemorySecretStore) DeleteConnectionCredentials(ctx context.Context, connID string) error {
	delete(s.secrets, connID)
	return nil
}
