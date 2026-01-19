package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"webhook-platform/pkg/models"
)

// BidirectionalSyncRepository handles bi-directional sync data persistence
type BidirectionalSyncRepository interface {
	// Config operations
	CreateConfig(ctx context.Context, config *models.WebhookSyncConfig) error
	GetConfig(ctx context.Context, id uuid.UUID) (*models.WebhookSyncConfig, error)
	GetConfigsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.WebhookSyncConfig, error)
	UpdateConfig(ctx context.Context, config *models.WebhookSyncConfig) error
	DeleteConfig(ctx context.Context, id uuid.UUID) error

	// Transaction operations
	CreateTransaction(ctx context.Context, tx *models.SyncTransaction) error
	GetTransaction(ctx context.Context, id uuid.UUID) (*models.SyncTransaction, error)
	GetTransactionByCorrelation(ctx context.Context, correlationID string) (*models.SyncTransaction, error)
	GetTransactionsByConfig(ctx context.Context, configID uuid.UUID, limit int) ([]*models.SyncTransaction, error)
	GetPendingTransactions(ctx context.Context, tenantID uuid.UUID) ([]*models.SyncTransaction, error)
	GetTimedOutTransactions(ctx context.Context) ([]*models.SyncTransaction, error)
	UpdateTransactionState(ctx context.Context, id uuid.UUID, state string, errorMessage string) error
	CompleteTransaction(ctx context.Context, id uuid.UUID, responsePayload map[string]interface{}) error
	IncrementTransactionRetry(ctx context.Context, id uuid.UUID) error
	CountTransactionsToday(ctx context.Context, tenantID uuid.UUID, state string) (int, error)

	// State record operations
	CreateStateRecord(ctx context.Context, record *models.SyncStateRecord) error
	GetStateRecord(ctx context.Context, id uuid.UUID) (*models.SyncStateRecord, error)
	GetStateRecordByResource(ctx context.Context, configID uuid.UUID, resourceType, resourceID string) (*models.SyncStateRecord, error)
	GetStateRecordsByConfig(ctx context.Context, configID uuid.UUID) ([]*models.SyncStateRecord, error)
	GetConflictedRecords(ctx context.Context, tenantID uuid.UUID) ([]*models.SyncStateRecord, error)
	UpdateLocalState(ctx context.Context, id uuid.UUID, state map[string]interface{}) error
	UpdateRemoteState(ctx context.Context, id uuid.UUID, state map[string]interface{}) error
	SetConflict(ctx context.Context, id uuid.UUID, conflictData map[string]interface{}) error
	ResolveConflict(ctx context.Context, id uuid.UUID, resolvedState map[string]interface{}) error
	CountActiveConflicts(ctx context.Context, tenantID uuid.UUID) (int, error)

	// Acknowledgment operations
	CreateAcknowledgment(ctx context.Context, ack *models.SyncAcknowledgment) error
	GetAcknowledgment(ctx context.Context, id uuid.UUID) (*models.SyncAcknowledgment, error)
	GetAcknowledgmentByCorrelation(ctx context.Context, correlationID string) (*models.SyncAcknowledgment, error)
	GetPendingAcknowledgments(ctx context.Context, tenantID uuid.UUID) ([]*models.SyncAcknowledgment, error)
	MarkAcknowledged(ctx context.Context, id uuid.UUID) error
	CountPendingAcks(ctx context.Context, tenantID uuid.UUID) (int, error)

	// Conflict history
	CreateConflictHistory(ctx context.Context, history *models.SyncConflictHistory) error
	GetConflictHistoryByRecord(ctx context.Context, stateRecordID uuid.UUID) ([]*models.SyncConflictHistory, error)
}

// PostgresBidirectionalSyncRepository implements BidirectionalSyncRepository
type PostgresBidirectionalSyncRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresBidirectionalSyncRepository creates a new bi-directional sync repository
func NewPostgresBidirectionalSyncRepository(pool *pgxpool.Pool) *PostgresBidirectionalSyncRepository {
	return &PostgresBidirectionalSyncRepository{pool: pool}
}

// Config operations

func (r *PostgresBidirectionalSyncRepository) CreateConfig(ctx context.Context, config *models.WebhookSyncConfig) error {
	query := `
		INSERT INTO webhook_sync_configs (
			id, tenant_id, name, description, outbound_endpoint_id, inbound_event_type,
			sync_mode, timeout_seconds, retry_on_timeout, max_retries, correlation_config,
			enabled, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)
	`

	if config.ID == uuid.Nil {
		config.ID = uuid.New()
	}
	if config.TimeoutSeconds == 0 {
		config.TimeoutSeconds = 30
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	correlationJSON, _ := json.Marshal(config.CorrelationConfig)

	_, err := r.pool.Exec(ctx, query,
		config.ID, config.TenantID, config.Name, config.Description,
		config.OutboundEndpointID, config.InboundEventType, config.SyncMode,
		config.TimeoutSeconds, config.RetryOnTimeout, config.MaxRetries,
		correlationJSON, config.Enabled,
	)

	return err
}

func (r *PostgresBidirectionalSyncRepository) GetConfig(ctx context.Context, id uuid.UUID) (*models.WebhookSyncConfig, error) {
	query := `
		SELECT id, tenant_id, name, description, outbound_endpoint_id, inbound_event_type,
		       sync_mode, timeout_seconds, retry_on_timeout, max_retries, correlation_config,
		       enabled, created_at, updated_at
		FROM webhook_sync_configs WHERE id = $1
	`

	config := &models.WebhookSyncConfig{}
	var correlationJSON []byte

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&config.ID, &config.TenantID, &config.Name, &config.Description,
		&config.OutboundEndpointID, &config.InboundEventType, &config.SyncMode,
		&config.TimeoutSeconds, &config.RetryOnTimeout, &config.MaxRetries,
		&correlationJSON, &config.Enabled, &config.CreatedAt, &config.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("config not found: %w", err)
	}

	json.Unmarshal(correlationJSON, &config.CorrelationConfig)
	return config, nil
}

func (r *PostgresBidirectionalSyncRepository) GetConfigsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.WebhookSyncConfig, error) {
	query := `
		SELECT id, tenant_id, name, description, outbound_endpoint_id, inbound_event_type,
		       sync_mode, timeout_seconds, retry_on_timeout, max_retries, correlation_config,
		       enabled, created_at, updated_at
		FROM webhook_sync_configs WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*models.WebhookSyncConfig
	for rows.Next() {
		config := &models.WebhookSyncConfig{}
		var correlationJSON []byte

		if err := rows.Scan(
			&config.ID, &config.TenantID, &config.Name, &config.Description,
			&config.OutboundEndpointID, &config.InboundEventType, &config.SyncMode,
			&config.TimeoutSeconds, &config.RetryOnTimeout, &config.MaxRetries,
			&correlationJSON, &config.Enabled, &config.CreatedAt, &config.UpdatedAt,
		); err != nil {
			return nil, err
		}

		json.Unmarshal(correlationJSON, &config.CorrelationConfig)
		configs = append(configs, config)
	}

	return configs, nil
}

func (r *PostgresBidirectionalSyncRepository) UpdateConfig(ctx context.Context, config *models.WebhookSyncConfig) error {
	query := `
		UPDATE webhook_sync_configs
		SET name = $2, description = $3, outbound_endpoint_id = $4, inbound_event_type = $5,
		    sync_mode = $6, timeout_seconds = $7, retry_on_timeout = $8, max_retries = $9,
		    correlation_config = $10, enabled = $11, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	correlationJSON, _ := json.Marshal(config.CorrelationConfig)

	_, err := r.pool.Exec(ctx, query,
		config.ID, config.Name, config.Description, config.OutboundEndpointID,
		config.InboundEventType, config.SyncMode, config.TimeoutSeconds,
		config.RetryOnTimeout, config.MaxRetries, correlationJSON, config.Enabled,
	)

	return err
}

func (r *PostgresBidirectionalSyncRepository) DeleteConfig(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM webhook_sync_configs WHERE id = $1", id)
	return err
}

// Transaction operations

func (r *PostgresBidirectionalSyncRepository) CreateTransaction(ctx context.Context, tx *models.SyncTransaction) error {
	query := `
		INSERT INTO sync_transactions (
			id, tenant_id, config_id, correlation_id, outbound_event_id, state,
			request_payload, timeout_at, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)
	`

	if tx.ID == uuid.Nil {
		tx.ID = uuid.New()
	}
	if tx.State == "" {
		tx.State = models.SyncStatesPending
	}

	requestPayloadJSON, _ := json.Marshal(tx.RequestPayload)
	metadataJSON, _ := json.Marshal(tx.Metadata)

	_, err := r.pool.Exec(ctx, query,
		tx.ID, tx.TenantID, tx.ConfigID, tx.CorrelationID, tx.OutboundEventID,
		tx.State, requestPayloadJSON, tx.TimeoutAt, metadataJSON,
	)

	return err
}

func (r *PostgresBidirectionalSyncRepository) GetTransaction(ctx context.Context, id uuid.UUID) (*models.SyncTransaction, error) {
	query := `
		SELECT id, tenant_id, config_id, correlation_id, outbound_event_id, inbound_event_id,
		       state, request_payload, response_payload, request_sent_at, response_received_at,
		       timeout_at, retry_count, error_message, metadata, created_at, updated_at
		FROM sync_transactions WHERE id = $1
	`

	return r.scanTransaction(ctx, query, id)
}

func (r *PostgresBidirectionalSyncRepository) GetTransactionByCorrelation(ctx context.Context, correlationID string) (*models.SyncTransaction, error) {
	query := `
		SELECT id, tenant_id, config_id, correlation_id, outbound_event_id, inbound_event_id,
		       state, request_payload, response_payload, request_sent_at, response_received_at,
		       timeout_at, retry_count, error_message, metadata, created_at, updated_at
		FROM sync_transactions WHERE correlation_id = $1
		ORDER BY created_at DESC LIMIT 1
	`

	return r.scanTransaction(ctx, query, correlationID)
}

func (r *PostgresBidirectionalSyncRepository) scanTransaction(ctx context.Context, query string, arg interface{}) (*models.SyncTransaction, error) {
	tx := &models.SyncTransaction{}
	var requestPayloadJSON, responsePayloadJSON, metadataJSON []byte

	err := r.pool.QueryRow(ctx, query, arg).Scan(
		&tx.ID, &tx.TenantID, &tx.ConfigID, &tx.CorrelationID, &tx.OutboundEventID,
		&tx.InboundEventID, &tx.State, &requestPayloadJSON, &responsePayloadJSON,
		&tx.RequestSentAt, &tx.ResponseReceivedAt, &tx.TimeoutAt, &tx.RetryCount,
		&tx.ErrorMessage, &metadataJSON, &tx.CreatedAt, &tx.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	json.Unmarshal(requestPayloadJSON, &tx.RequestPayload)
	json.Unmarshal(responsePayloadJSON, &tx.ResponsePayload)
	json.Unmarshal(metadataJSON, &tx.Metadata)

	return tx, nil
}

func (r *PostgresBidirectionalSyncRepository) GetTransactionsByConfig(ctx context.Context, configID uuid.UUID, limit int) ([]*models.SyncTransaction, error) {
	query := `
		SELECT id, tenant_id, config_id, correlation_id, outbound_event_id, inbound_event_id,
		       state, request_payload, response_payload, request_sent_at, response_received_at,
		       timeout_at, retry_count, error_message, metadata, created_at, updated_at
		FROM sync_transactions WHERE config_id = $1
		ORDER BY created_at DESC LIMIT $2
	`

	return r.queryTransactions(ctx, query, configID, limit)
}

func (r *PostgresBidirectionalSyncRepository) GetPendingTransactions(ctx context.Context, tenantID uuid.UUID) ([]*models.SyncTransaction, error) {
	query := `
		SELECT id, tenant_id, config_id, correlation_id, outbound_event_id, inbound_event_id,
		       state, request_payload, response_payload, request_sent_at, response_received_at,
		       timeout_at, retry_count, error_message, metadata, created_at, updated_at
		FROM sync_transactions WHERE tenant_id = $1 AND state IN ('pending', 'awaiting_response')
		ORDER BY created_at DESC
	`

	return r.queryTransactions(ctx, query, tenantID)
}

func (r *PostgresBidirectionalSyncRepository) GetTimedOutTransactions(ctx context.Context) ([]*models.SyncTransaction, error) {
	query := `
		SELECT id, tenant_id, config_id, correlation_id, outbound_event_id, inbound_event_id,
		       state, request_payload, response_payload, request_sent_at, response_received_at,
		       timeout_at, retry_count, error_message, metadata, created_at, updated_at
		FROM sync_transactions
		WHERE state = 'awaiting_response' AND timeout_at < CURRENT_TIMESTAMP
	`

	return r.queryTransactions(ctx, query)
}

func (r *PostgresBidirectionalSyncRepository) queryTransactions(ctx context.Context, query string, args ...interface{}) ([]*models.SyncTransaction, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []*models.SyncTransaction
	for rows.Next() {
		tx := &models.SyncTransaction{}
		var requestPayloadJSON, responsePayloadJSON, metadataJSON []byte

		if err := rows.Scan(
			&tx.ID, &tx.TenantID, &tx.ConfigID, &tx.CorrelationID, &tx.OutboundEventID,
			&tx.InboundEventID, &tx.State, &requestPayloadJSON, &responsePayloadJSON,
			&tx.RequestSentAt, &tx.ResponseReceivedAt, &tx.TimeoutAt, &tx.RetryCount,
			&tx.ErrorMessage, &metadataJSON, &tx.CreatedAt, &tx.UpdatedAt,
		); err != nil {
			return nil, err
		}

		json.Unmarshal(requestPayloadJSON, &tx.RequestPayload)
		json.Unmarshal(responsePayloadJSON, &tx.ResponsePayload)
		json.Unmarshal(metadataJSON, &tx.Metadata)
		transactions = append(transactions, tx)
	}

	return transactions, nil
}

func (r *PostgresBidirectionalSyncRepository) UpdateTransactionState(ctx context.Context, id uuid.UUID, state string, errorMessage string) error {
	query := `
		UPDATE sync_transactions
		SET state = $2, error_message = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, id, state, errorMessage)
	return err
}

func (r *PostgresBidirectionalSyncRepository) CompleteTransaction(ctx context.Context, id uuid.UUID, responsePayload map[string]interface{}) error {
	query := `
		UPDATE sync_transactions
		SET state = 'completed', response_payload = $2, response_received_at = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	responsePayloadJSON, _ := json.Marshal(responsePayload)
	_, err := r.pool.Exec(ctx, query, id, responsePayloadJSON)
	return err
}

func (r *PostgresBidirectionalSyncRepository) IncrementTransactionRetry(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE sync_transactions
		SET retry_count = retry_count + 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

func (r *PostgresBidirectionalSyncRepository) CountTransactionsToday(ctx context.Context, tenantID uuid.UUID, state string) (int, error) {
	query := `
		SELECT COUNT(*) FROM sync_transactions
		WHERE tenant_id = $1 AND created_at >= CURRENT_DATE
	`
	args := []interface{}{tenantID}

	if state != "" {
		query += " AND state = $2"
		args = append(args, state)
	}

	var count int
	err := r.pool.QueryRow(ctx, query, args...).Scan(&count)
	return count, err
}

// State record operations

func (r *PostgresBidirectionalSyncRepository) CreateStateRecord(ctx context.Context, record *models.SyncStateRecord) error {
	query := `
		INSERT INTO sync_state_records (
			id, tenant_id, config_id, resource_type, resource_id, local_state,
			sync_status, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		) ON CONFLICT (config_id, resource_type, resource_id) DO UPDATE
		SET local_state = $6, last_local_update = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
	`

	if record.ID == uuid.Nil {
		record.ID = uuid.New()
	}
	if record.SyncStatus == "" {
		record.SyncStatus = models.SyncStatusPendingPush
	}

	localStateJSON, _ := json.Marshal(record.LocalState)

	_, err := r.pool.Exec(ctx, query,
		record.ID, record.TenantID, record.ConfigID, record.ResourceType,
		record.ResourceID, localStateJSON, record.SyncStatus,
	)

	return err
}

func (r *PostgresBidirectionalSyncRepository) GetStateRecord(ctx context.Context, id uuid.UUID) (*models.SyncStateRecord, error) {
	query := `
		SELECT id, tenant_id, config_id, resource_type, resource_id, local_state,
		       remote_state, last_local_update, last_remote_update, sync_status,
		       conflict_data, conflict_resolved_at, created_at, updated_at
		FROM sync_state_records WHERE id = $1
	`

	record := &models.SyncStateRecord{}
	var localStateJSON, remoteStateJSON, conflictDataJSON []byte

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&record.ID, &record.TenantID, &record.ConfigID, &record.ResourceType,
		&record.ResourceID, &localStateJSON, &remoteStateJSON, &record.LastLocalUpdate,
		&record.LastRemoteUpdate, &record.SyncStatus, &conflictDataJSON,
		&record.ConflictResolvedAt, &record.CreatedAt, &record.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	json.Unmarshal(localStateJSON, &record.LocalState)
	json.Unmarshal(remoteStateJSON, &record.RemoteState)
	json.Unmarshal(conflictDataJSON, &record.ConflictData)

	return record, nil
}

func (r *PostgresBidirectionalSyncRepository) GetStateRecordByResource(ctx context.Context, configID uuid.UUID, resourceType, resourceID string) (*models.SyncStateRecord, error) {
	query := `
		SELECT id, tenant_id, config_id, resource_type, resource_id, local_state,
		       remote_state, last_local_update, last_remote_update, sync_status,
		       conflict_data, conflict_resolved_at, created_at, updated_at
		FROM sync_state_records WHERE config_id = $1 AND resource_type = $2 AND resource_id = $3
	`

	record := &models.SyncStateRecord{}
	var localStateJSON, remoteStateJSON, conflictDataJSON []byte

	err := r.pool.QueryRow(ctx, query, configID, resourceType, resourceID).Scan(
		&record.ID, &record.TenantID, &record.ConfigID, &record.ResourceType,
		&record.ResourceID, &localStateJSON, &remoteStateJSON, &record.LastLocalUpdate,
		&record.LastRemoteUpdate, &record.SyncStatus, &conflictDataJSON,
		&record.ConflictResolvedAt, &record.CreatedAt, &record.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	json.Unmarshal(localStateJSON, &record.LocalState)
	json.Unmarshal(remoteStateJSON, &record.RemoteState)
	json.Unmarshal(conflictDataJSON, &record.ConflictData)

	return record, nil
}

func (r *PostgresBidirectionalSyncRepository) GetStateRecordsByConfig(ctx context.Context, configID uuid.UUID) ([]*models.SyncStateRecord, error) {
	query := `
		SELECT id, tenant_id, config_id, resource_type, resource_id, local_state,
		       remote_state, last_local_update, last_remote_update, sync_status,
		       conflict_data, conflict_resolved_at, created_at, updated_at
		FROM sync_state_records WHERE config_id = $1
		ORDER BY updated_at DESC
	`

	return r.queryStateRecords(ctx, query, configID)
}

func (r *PostgresBidirectionalSyncRepository) GetConflictedRecords(ctx context.Context, tenantID uuid.UUID) ([]*models.SyncStateRecord, error) {
	query := `
		SELECT id, tenant_id, config_id, resource_type, resource_id, local_state,
		       remote_state, last_local_update, last_remote_update, sync_status,
		       conflict_data, conflict_resolved_at, created_at, updated_at
		FROM sync_state_records WHERE tenant_id = $1 AND sync_status = 'conflict'
		ORDER BY updated_at DESC
	`

	return r.queryStateRecords(ctx, query, tenantID)
}

func (r *PostgresBidirectionalSyncRepository) queryStateRecords(ctx context.Context, query string, arg interface{}) ([]*models.SyncStateRecord, error) {
	rows, err := r.pool.Query(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*models.SyncStateRecord
	for rows.Next() {
		record := &models.SyncStateRecord{}
		var localStateJSON, remoteStateJSON, conflictDataJSON []byte

		if err := rows.Scan(
			&record.ID, &record.TenantID, &record.ConfigID, &record.ResourceType,
			&record.ResourceID, &localStateJSON, &remoteStateJSON, &record.LastLocalUpdate,
			&record.LastRemoteUpdate, &record.SyncStatus, &conflictDataJSON,
			&record.ConflictResolvedAt, &record.CreatedAt, &record.UpdatedAt,
		); err != nil {
			return nil, err
		}

		json.Unmarshal(localStateJSON, &record.LocalState)
		json.Unmarshal(remoteStateJSON, &record.RemoteState)
		json.Unmarshal(conflictDataJSON, &record.ConflictData)
		records = append(records, record)
	}

	return records, nil
}

func (r *PostgresBidirectionalSyncRepository) UpdateLocalState(ctx context.Context, id uuid.UUID, state map[string]interface{}) error {
	query := `
		UPDATE sync_state_records
		SET local_state = $2, last_local_update = CURRENT_TIMESTAMP,
		    sync_status = 'pending_push', updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	stateJSON, _ := json.Marshal(state)
	_, err := r.pool.Exec(ctx, query, id, stateJSON)
	return err
}

func (r *PostgresBidirectionalSyncRepository) UpdateRemoteState(ctx context.Context, id uuid.UUID, state map[string]interface{}) error {
	query := `
		UPDATE sync_state_records
		SET remote_state = $2, last_remote_update = CURRENT_TIMESTAMP,
		    sync_status = 'synced', updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	stateJSON, _ := json.Marshal(state)
	_, err := r.pool.Exec(ctx, query, id, stateJSON)
	return err
}

func (r *PostgresBidirectionalSyncRepository) SetConflict(ctx context.Context, id uuid.UUID, conflictData map[string]interface{}) error {
	query := `
		UPDATE sync_state_records
		SET sync_status = 'conflict', conflict_data = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	conflictJSON, _ := json.Marshal(conflictData)
	_, err := r.pool.Exec(ctx, query, id, conflictJSON)
	return err
}

func (r *PostgresBidirectionalSyncRepository) ResolveConflict(ctx context.Context, id uuid.UUID, resolvedState map[string]interface{}) error {
	query := `
		UPDATE sync_state_records
		SET local_state = $2, sync_status = 'synced', conflict_data = NULL,
		    conflict_resolved_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	stateJSON, _ := json.Marshal(resolvedState)
	_, err := r.pool.Exec(ctx, query, id, stateJSON)
	return err
}

func (r *PostgresBidirectionalSyncRepository) CountActiveConflicts(ctx context.Context, tenantID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM sync_state_records WHERE tenant_id = $1 AND sync_status = 'conflict'`
	var count int
	err := r.pool.QueryRow(ctx, query, tenantID).Scan(&count)
	return count, err
}

// Acknowledgment operations

func (r *PostgresBidirectionalSyncRepository) CreateAcknowledgment(ctx context.Context, ack *models.SyncAcknowledgment) error {
	query := `
		INSERT INTO sync_acknowledgments (
			id, tenant_id, config_id, event_id, correlation_id, ack_type,
			ack_payload, timeout_at, status, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, CURRENT_TIMESTAMP
		)
	`

	if ack.ID == uuid.Nil {
		ack.ID = uuid.New()
	}
	if ack.Status == "" {
		ack.Status = models.SyncStatesPending
	}

	ackPayloadJSON, _ := json.Marshal(ack.AckPayload)

	_, err := r.pool.Exec(ctx, query,
		ack.ID, ack.TenantID, ack.ConfigID, ack.EventID, ack.CorrelationID,
		ack.AckType, ackPayloadJSON, ack.TimeoutAt, ack.Status,
	)

	return err
}

func (r *PostgresBidirectionalSyncRepository) GetAcknowledgment(ctx context.Context, id uuid.UUID) (*models.SyncAcknowledgment, error) {
	query := `
		SELECT id, tenant_id, config_id, event_id, correlation_id, ack_type,
		       ack_payload, sent_at, acknowledged_at, timeout_at, retry_count, status, created_at
		FROM sync_acknowledgments WHERE id = $1
	`

	ack := &models.SyncAcknowledgment{}
	var ackPayloadJSON []byte

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&ack.ID, &ack.TenantID, &ack.ConfigID, &ack.EventID, &ack.CorrelationID,
		&ack.AckType, &ackPayloadJSON, &ack.SentAt, &ack.AcknowledgedAt,
		&ack.TimeoutAt, &ack.RetryCount, &ack.Status, &ack.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	json.Unmarshal(ackPayloadJSON, &ack.AckPayload)
	return ack, nil
}

func (r *PostgresBidirectionalSyncRepository) GetAcknowledgmentByCorrelation(ctx context.Context, correlationID string) (*models.SyncAcknowledgment, error) {
	query := `
		SELECT id, tenant_id, config_id, event_id, correlation_id, ack_type,
		       ack_payload, sent_at, acknowledged_at, timeout_at, retry_count, status, created_at
		FROM sync_acknowledgments WHERE correlation_id = $1
		ORDER BY created_at DESC LIMIT 1
	`

	ack := &models.SyncAcknowledgment{}
	var ackPayloadJSON []byte

	err := r.pool.QueryRow(ctx, query, correlationID).Scan(
		&ack.ID, &ack.TenantID, &ack.ConfigID, &ack.EventID, &ack.CorrelationID,
		&ack.AckType, &ackPayloadJSON, &ack.SentAt, &ack.AcknowledgedAt,
		&ack.TimeoutAt, &ack.RetryCount, &ack.Status, &ack.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	json.Unmarshal(ackPayloadJSON, &ack.AckPayload)
	return ack, nil
}

func (r *PostgresBidirectionalSyncRepository) GetPendingAcknowledgments(ctx context.Context, tenantID uuid.UUID) ([]*models.SyncAcknowledgment, error) {
	query := `
		SELECT id, tenant_id, config_id, event_id, correlation_id, ack_type,
		       ack_payload, sent_at, acknowledged_at, timeout_at, retry_count, status, created_at
		FROM sync_acknowledgments WHERE tenant_id = $1 AND status = 'pending'
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var acks []*models.SyncAcknowledgment
	for rows.Next() {
		ack := &models.SyncAcknowledgment{}
		var ackPayloadJSON []byte

		if err := rows.Scan(
			&ack.ID, &ack.TenantID, &ack.ConfigID, &ack.EventID, &ack.CorrelationID,
			&ack.AckType, &ackPayloadJSON, &ack.SentAt, &ack.AcknowledgedAt,
			&ack.TimeoutAt, &ack.RetryCount, &ack.Status, &ack.CreatedAt,
		); err != nil {
			return nil, err
		}

		json.Unmarshal(ackPayloadJSON, &ack.AckPayload)
		acks = append(acks, ack)
	}

	return acks, nil
}

func (r *PostgresBidirectionalSyncRepository) MarkAcknowledged(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE sync_acknowledgments
		SET status = 'acknowledged', acknowledged_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

func (r *PostgresBidirectionalSyncRepository) CountPendingAcks(ctx context.Context, tenantID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM sync_acknowledgments WHERE tenant_id = $1 AND status = 'pending'`
	var count int
	err := r.pool.QueryRow(ctx, query, tenantID).Scan(&count)
	return count, err
}

// Conflict history

func (r *PostgresBidirectionalSyncRepository) CreateConflictHistory(ctx context.Context, history *models.SyncConflictHistory) error {
	query := `
		INSERT INTO sync_conflict_history (
			id, tenant_id, state_record_id, local_state, remote_state,
			resolution_strategy, resolved_state, resolved_by, resolved_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, CURRENT_TIMESTAMP
		)
	`

	if history.ID == uuid.Nil {
		history.ID = uuid.New()
	}

	localStateJSON, _ := json.Marshal(history.LocalState)
	remoteStateJSON, _ := json.Marshal(history.RemoteState)
	resolvedStateJSON, _ := json.Marshal(history.ResolvedState)

	_, err := r.pool.Exec(ctx, query,
		history.ID, history.TenantID, history.StateRecordID,
		localStateJSON, remoteStateJSON, history.ResolutionStrategy,
		resolvedStateJSON, history.ResolvedBy,
	)

	return err
}

func (r *PostgresBidirectionalSyncRepository) GetConflictHistoryByRecord(ctx context.Context, stateRecordID uuid.UUID) ([]*models.SyncConflictHistory, error) {
	query := `
		SELECT id, tenant_id, state_record_id, local_state, remote_state,
		       resolution_strategy, resolved_state, resolved_by, resolved_at
		FROM sync_conflict_history WHERE state_record_id = $1
		ORDER BY resolved_at DESC
	`

	rows, err := r.pool.Query(ctx, query, stateRecordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var histories []*models.SyncConflictHistory
	for rows.Next() {
		h := &models.SyncConflictHistory{}
		var localStateJSON, remoteStateJSON, resolvedStateJSON []byte

		if err := rows.Scan(
			&h.ID, &h.TenantID, &h.StateRecordID, &localStateJSON,
			&remoteStateJSON, &h.ResolutionStrategy, &resolvedStateJSON,
			&h.ResolvedBy, &h.ResolvedAt,
		); err != nil {
			return nil, err
		}

		json.Unmarshal(localStateJSON, &h.LocalState)
		json.Unmarshal(remoteStateJSON, &h.RemoteState)
		json.Unmarshal(resolvedStateJSON, &h.ResolvedState)
		histories = append(histories, h)
	}

	return histories, nil
}
