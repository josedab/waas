package pushbridge

import (
	"context"
)

// QueueNotification queues a notification
func (r *PostgresRepository) QueueNotification(ctx context.Context, queue *OfflineQueue) error {
	query := `
		INSERT INTO push_offline_queue (id, tenant_id, device_id, notification, priority, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.db.ExecContext(ctx, query,
		queue.ID, queue.TenantID, queue.DeviceID, queue.Notification,
		queue.Priority, queue.ExpiresAt, queue.CreatedAt)
	return err
}

// GetQueuedNotifications gets queued notifications
func (r *PostgresRepository) GetQueuedNotifications(ctx context.Context, deviceID string, limit int) ([]OfflineQueue, error) {
	query := `
		SELECT id, tenant_id, device_id, notification, priority, expires_at, created_at
		FROM push_offline_queue
		WHERE device_id = $1 AND expires_at > NOW()
		ORDER BY priority DESC, created_at ASC
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, deviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var queued []OfflineQueue
	for rows.Next() {
		var q OfflineQueue
		if err := rows.Scan(&q.ID, &q.TenantID, &q.DeviceID, &q.Notification, &q.Priority, &q.ExpiresAt, &q.CreatedAt); err != nil {
			continue
		}
		queued = append(queued, q)
	}

	return queued, nil
}

// DeleteQueuedNotification deletes a queued notification
func (r *PostgresRepository) DeleteQueuedNotification(ctx context.Context, queueID string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM push_offline_queue WHERE id = $1", queueID)
	return err
}

// CleanExpiredQueue removes expired queue entries
func (r *PostgresRepository) CleanExpiredQueue(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM push_offline_queue WHERE expires_at <= NOW()")
	return err
}
