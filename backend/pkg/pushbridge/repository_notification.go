package pushbridge

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SaveNotification saves a notification
func (r *PostgresRepository) SaveNotification(ctx context.Context, notif *PushNotification) error {
	responseJSON, _ := json.Marshal(notif.Response)

	query := `
		INSERT INTO push_notifications (
			id, tenant_id, mapping_id, webhook_id, platform, device_id,
			push_token, status, payload, response, attempts,
			last_attempt, delivered_at, opened_at, error, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			response = EXCLUDED.response,
			attempts = EXCLUDED.attempts,
			last_attempt = EXCLUDED.last_attempt,
			delivered_at = EXCLUDED.delivered_at,
			opened_at = EXCLUDED.opened_at,
			error = EXCLUDED.error`

	_, err := r.db.ExecContext(ctx, query,
		notif.ID, notif.TenantID, notif.MappingID, notif.WebhookID,
		notif.Platform, notif.DeviceID, notif.PushToken, notif.Status,
		notif.Payload, responseJSON, notif.Attempts,
		notif.LastAttempt, notif.DeliveredAt, notif.OpenedAt, notif.Error, notif.CreatedAt)

	return err
}

// GetNotification retrieves a notification
func (r *PostgresRepository) GetNotification(ctx context.Context, notifID string) (*PushNotification, error) {
	query := `
		SELECT id, tenant_id, mapping_id, webhook_id, platform, device_id,
			   push_token, status, payload, response, attempts,
			   last_attempt, delivered_at, opened_at, error, created_at
		FROM push_notifications
		WHERE id = $1`

	var notif PushNotification
	var responseJSON []byte
	var mappingID, webhookID, errMsg sql.NullString
	var lastAttempt, deliveredAt, openedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, notifID).Scan(
		&notif.ID, &notif.TenantID, &mappingID, &webhookID,
		&notif.Platform, &notif.DeviceID, &notif.PushToken, &notif.Status,
		&notif.Payload, &responseJSON, &notif.Attempts,
		&lastAttempt, &deliveredAt, &openedAt, &errMsg, &notif.CreatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("notification not found")
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(responseJSON, &notif.Response)

	if mappingID.Valid {
		notif.MappingID = mappingID.String
	}
	if webhookID.Valid {
		notif.WebhookID = webhookID.String
	}
	if errMsg.Valid {
		notif.Error = errMsg.String
	}
	if lastAttempt.Valid {
		notif.LastAttempt = &lastAttempt.Time
	}
	if deliveredAt.Valid {
		notif.DeliveredAt = &deliveredAt.Time
	}
	if openedAt.Valid {
		notif.OpenedAt = &openedAt.Time
	}

	return &notif, nil
}

// ListNotifications lists notifications
func (r *PostgresRepository) ListNotifications(ctx context.Context, tenantID string, filter *NotificationFilter) ([]PushNotification, error) {
	query := `
		SELECT id, tenant_id, mapping_id, webhook_id, platform, device_id,
			   push_token, status, payload, response, attempts,
			   last_attempt, delivered_at, opened_at, error, created_at
		FROM push_notifications
		WHERE tenant_id = $1`
	args := []any{tenantID}
	argIdx := 2

	if filter != nil {
		if filter.DeviceID != nil {
			query += fmt.Sprintf(" AND device_id = $%d", argIdx)
			args = append(args, *filter.DeviceID)
			argIdx++
		}
		if filter.MappingID != nil {
			query += fmt.Sprintf(" AND mapping_id = $%d", argIdx)
			args = append(args, *filter.MappingID)
			argIdx++
		}
		if filter.Status != nil {
			query += fmt.Sprintf(" AND status = $%d", argIdx)
			args = append(args, *filter.Status)
			argIdx++
		}
		if filter.From != nil {
			query += fmt.Sprintf(" AND created_at >= $%d", argIdx)
			args = append(args, *filter.From)
			argIdx++
		}
		if filter.To != nil {
			query += fmt.Sprintf(" AND created_at <= $%d", argIdx)
			args = append(args, *filter.To)
			argIdx++
		}
	}

	query += " ORDER BY created_at DESC"

	limit := 50
	if filter != nil && filter.Limit > 0 {
		limit = filter.Limit
	}
	query += fmt.Sprintf(" LIMIT %d", limit)

	if filter != nil && filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []PushNotification
	for rows.Next() {
		var notif PushNotification
		var responseJSON []byte
		var mappingID, webhookID, errMsg sql.NullString
		var lastAttempt, deliveredAt, openedAt sql.NullTime

		err := rows.Scan(
			&notif.ID, &notif.TenantID, &mappingID, &webhookID,
			&notif.Platform, &notif.DeviceID, &notif.PushToken, &notif.Status,
			&notif.Payload, &responseJSON, &notif.Attempts,
			&lastAttempt, &deliveredAt, &openedAt, &errMsg, &notif.CreatedAt)
		if err != nil {
			continue
		}

		json.Unmarshal(responseJSON, &notif.Response)

		if mappingID.Valid {
			notif.MappingID = mappingID.String
		}
		if webhookID.Valid {
			notif.WebhookID = webhookID.String
		}
		if errMsg.Valid {
			notif.Error = errMsg.String
		}
		if lastAttempt.Valid {
			notif.LastAttempt = &lastAttempt.Time
		}
		if deliveredAt.Valid {
			notif.DeliveredAt = &deliveredAt.Time
		}
		if openedAt.Valid {
			notif.OpenedAt = &openedAt.Time
		}

		notifications = append(notifications, notif)
	}

	return notifications, nil
}

// UpdateNotificationStatus updates notification status
func (r *PostgresRepository) UpdateNotificationStatus(ctx context.Context, notifID string, status DeliveryStatus, response *ProviderResponse) error {
	responseJSON, _ := json.Marshal(response)

	var deliveredAt, openedAt *time.Time
	now := time.Now()
	if status == DeliveryDelivered {
		deliveredAt = &now
	} else if status == DeliveryOpened {
		openedAt = &now
	}

	query := `
		UPDATE push_notifications SET
			status = $1, response = $2, last_attempt = NOW(),
			delivered_at = COALESCE($3, delivered_at),
			opened_at = COALESCE($4, opened_at),
			attempts = attempts + 1
		WHERE id = $5`

	_, err := r.db.ExecContext(ctx, query, status, responseJSON, deliveredAt, openedAt, notifID)
	return err
}

// GenerateNotificationID generates a new notification ID
func GenerateNotificationID() string {
	return uuid.New().String()
}
