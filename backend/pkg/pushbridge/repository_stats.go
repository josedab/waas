package pushbridge

import (
	"context"
	"time"
)

// GetStats retrieves push statistics
func (r *PostgresRepository) GetStats(ctx context.Context, tenantID string, from, to time.Time) (*PushStats, error) {
	query := `
		SELECT 
			COALESCE(SUM(CASE WHEN status IN ('sent', 'delivered', 'opened') THEN 1 ELSE 0 END), 0) as sent,
			COALESCE(SUM(CASE WHEN status IN ('delivered', 'opened') THEN 1 ELSE 0 END), 0) as delivered,
			COALESCE(SUM(CASE WHEN status = 'opened' THEN 1 ELSE 0 END), 0) as opened,
			COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0) as failed,
			COALESCE(SUM(CASE WHEN status = 'dropped' THEN 1 ELSE 0 END), 0) as dropped
		FROM push_notifications
		WHERE tenant_id = $1 AND created_at BETWEEN $2 AND $3`

	stats := &PushStats{TenantID: tenantID}
	err := r.db.QueryRowContext(ctx, query, tenantID, from, to).Scan(
		&stats.TotalSent, &stats.TotalDelivered, &stats.TotalOpened,
		&stats.TotalFailed, &stats.TotalDropped)
	if err != nil {
		return nil, err
	}

	if stats.TotalSent > 0 {
		stats.DeliveryRate = float64(stats.TotalDelivered) / float64(stats.TotalSent) * 100
		stats.OpenRate = float64(stats.TotalOpened) / float64(stats.TotalDelivered) * 100
	}

	return stats, nil
}

// IncrementStats increments statistics
func (r *PostgresRepository) IncrementStats(ctx context.Context, tenantID string, platform Platform, status DeliveryStatus) error {
	// Stats are calculated from notifications table, no separate stats table needed
	return nil
}
