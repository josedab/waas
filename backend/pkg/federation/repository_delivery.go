package federation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

func (r *PostgresRepository) SaveDelivery(ctx context.Context, delivery *FederatedDelivery) error {
	payloadJSON, err := json.Marshal(delivery.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal delivery payload: %w", err)
	}

	query := `
		INSERT INTO federation_deliveries (
			id, tenant_id, subscription_id, source_member_id, target_member_id,
			event_type, event_id, payload, status, attempts, last_attempt_at,
			next_retry_at, error, response_code, response_body, latency,
			delivered_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			attempts = EXCLUDED.attempts,
			last_attempt_at = EXCLUDED.last_attempt_at,
			next_retry_at = EXCLUDED.next_retry_at,
			error = EXCLUDED.error,
			response_code = EXCLUDED.response_code,
			response_body = EXCLUDED.response_body,
			latency = EXCLUDED.latency,
			delivered_at = EXCLUDED.delivered_at`

	_, err = r.db.ExecContext(ctx, query,
		delivery.ID, delivery.TenantID, delivery.SubscriptionID,
		delivery.SourceMemberID, delivery.TargetMemberID, delivery.EventType,
		delivery.EventID, payloadJSON, delivery.Status, delivery.Attempts,
		delivery.LastAttemptAt, delivery.NextRetryAt, delivery.Error,
		delivery.ResponseCode, delivery.ResponseBody, delivery.Latency,
		delivery.DeliveredAt, delivery.CreatedAt)

	return err
}

// GetDelivery retrieves a delivery
func (r *PostgresRepository) GetDelivery(ctx context.Context, deliveryID string) (*FederatedDelivery, error) {
	query := `
		SELECT id, tenant_id, subscription_id, source_member_id, target_member_id,
			   event_type, event_id, payload, status, attempts, last_attempt_at,
			   next_retry_at, error, response_code, response_body, latency,
			   delivered_at, created_at
		FROM federation_deliveries
		WHERE id = $1`

	var d FederatedDelivery
	var payloadJSON []byte
	var lastAttemptAt, nextRetryAt, deliveredAt sql.NullTime
	var errStr, responseBody sql.NullString
	var responseCode sql.NullInt32

	err := r.db.QueryRowContext(ctx, query, deliveryID).Scan(
		&d.ID, &d.TenantID, &d.SubscriptionID, &d.SourceMemberID, &d.TargetMemberID,
		&d.EventType, &d.EventID, &payloadJSON, &d.Status, &d.Attempts,
		&lastAttemptAt, &nextRetryAt, &errStr, &responseCode, &responseBody,
		&d.Latency, &deliveredAt, &d.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("delivery not found")
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(payloadJSON, &d.Payload); err != nil {
		r.logger.Warn("failed to unmarshal delivery payload", map[string]interface{}{"delivery_id": deliveryID, "error": err.Error()})
	}
	if lastAttemptAt.Valid {
		d.LastAttemptAt = &lastAttemptAt.Time
	}
	if nextRetryAt.Valid {
		d.NextRetryAt = &nextRetryAt.Time
	}
	if deliveredAt.Valid {
		d.DeliveredAt = &deliveredAt.Time
	}
	if errStr.Valid {
		d.Error = errStr.String
	}
	if responseBody.Valid {
		d.ResponseBody = responseBody.String
	}
	if responseCode.Valid {
		d.ResponseCode = int(responseCode.Int32)
	}

	return &d, nil
}

// ListPendingDeliveries lists pending deliveries
func (r *PostgresRepository) ListPendingDeliveries(ctx context.Context, limit int) ([]FederatedDelivery, error) {
	query := `
		SELECT id, tenant_id, subscription_id, source_member_id, target_member_id,
			   event_type, event_id, payload, status, attempts, last_attempt_at,
			   next_retry_at, error, response_code, response_body, latency,
			   delivered_at, created_at
		FROM federation_deliveries
		WHERE status IN ('pending', 'retrying')
			AND (next_retry_at IS NULL OR next_retry_at <= NOW())
		ORDER BY created_at
		LIMIT $1`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanDeliveries(rows)
}

// ListDeliveries lists deliveries
func (r *PostgresRepository) ListDeliveries(ctx context.Context, tenantID, subID string, limit int) ([]FederatedDelivery, error) {
	query := `
		SELECT id, tenant_id, subscription_id, source_member_id, target_member_id,
			   event_type, event_id, payload, status, attempts, last_attempt_at,
			   next_retry_at, error, response_code, response_body, latency,
			   delivered_at, created_at
		FROM federation_deliveries
		WHERE tenant_id = $1 AND subscription_id = $2
		ORDER BY created_at DESC
		LIMIT $3`

	rows, err := r.db.QueryContext(ctx, query, tenantID, subID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanDeliveries(rows)
}

func (r *PostgresRepository) scanDeliveries(rows *sql.Rows) ([]FederatedDelivery, error) {
	var deliveries []FederatedDelivery
	for rows.Next() {
		var d FederatedDelivery
		var payloadJSON []byte
		var lastAttemptAt, nextRetryAt, deliveredAt sql.NullTime
		var errStr, responseBody sql.NullString
		var responseCode sql.NullInt32

		err := rows.Scan(
			&d.ID, &d.TenantID, &d.SubscriptionID, &d.SourceMemberID, &d.TargetMemberID,
			&d.EventType, &d.EventID, &payloadJSON, &d.Status, &d.Attempts,
			&lastAttemptAt, &nextRetryAt, &errStr, &responseCode, &responseBody,
			&d.Latency, &deliveredAt, &d.CreatedAt)
		if err != nil {
			continue
		}

		if err := json.Unmarshal(payloadJSON, &d.Payload); err != nil {
			r.logger.Warn("failed to unmarshal delivery payload", map[string]interface{}{"delivery_id": d.ID, "error": err.Error()})
		}
		if lastAttemptAt.Valid {
			d.LastAttemptAt = &lastAttemptAt.Time
		}
		if nextRetryAt.Valid {
			d.NextRetryAt = &nextRetryAt.Time
		}
		if deliveredAt.Valid {
			d.DeliveredAt = &deliveredAt.Time
		}
		if errStr.Valid {
			d.Error = errStr.String
		}
		if responseBody.Valid {
			d.ResponseBody = responseBody.String
		}
		if responseCode.Valid {
			d.ResponseCode = int(responseCode.Int32)
		}

		deliveries = append(deliveries, d)
	}

	return deliveries, nil
}

// UpdateDeliveryStatus updates delivery status
func (r *PostgresRepository) UpdateDeliveryStatus(ctx context.Context, deliveryID string, status DeliveryStatus, errMsg string, respCode int) error {
	now := time.Now()
	var deliveredAt *time.Time
	if status == DeliverySucceeded {
		deliveredAt = &now
	}

	_, err := r.db.ExecContext(ctx,
		`UPDATE federation_deliveries SET 
			status = $1, last_attempt_at = $2, error = $3, 
			response_code = $4, delivered_at = $5, attempts = attempts + 1
		WHERE id = $6`,
		status, now, errMsg, respCode, deliveredAt, deliveryID)
	return err
}
