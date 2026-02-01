package repository

import (
	"context"
	"fmt"
	"time"
	"github.com/josedab/waas/pkg/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type quotaRepository struct {
	db *pgxpool.Pool
}

// NewQuotaRepository creates a new quota repository instance
func NewQuotaRepository(db *pgxpool.Pool) QuotaRepository {
	return &quotaRepository{db: db}
}

// GetOrCreateQuotaUsage gets existing quota usage or creates a new one for the month
func (r *quotaRepository) GetOrCreateQuotaUsage(ctx context.Context, tenantID uuid.UUID, month time.Time) (*models.QuotaUsage, error) {
	// Normalize month to first day of month
	monthStart := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	
	// Try to get existing usage
	usage, err := r.GetQuotaUsageByTenant(ctx, tenantID, monthStart)
	if err == nil {
		return usage, nil
	}
	
	// If not found, create new usage record
	if err == pgx.ErrNoRows {
		usage = &models.QuotaUsage{
			ID:           uuid.New(),
			TenantID:     tenantID,
			Month:        monthStart,
			RequestCount: 0,
			SuccessCount: 0,
			FailureCount: 0,
			OverageCount: 0,
			LastUpdated:  time.Now().UTC(),
			CreatedAt:    time.Now().UTC(),
		}
		
		query := `
			INSERT INTO quota_usage (id, tenant_id, month, request_count, success_count, failure_count, overage_count, last_updated, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
		
		_, err = r.db.Exec(ctx, query, usage.ID, usage.TenantID, usage.Month, usage.RequestCount, usage.SuccessCount, usage.FailureCount, usage.OverageCount, usage.LastUpdated, usage.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to create quota usage: %w", err)
		}
		
		return usage, nil
	}
	
	return nil, fmt.Errorf("failed to get quota usage: %w", err)
}

// UpdateQuotaUsage updates an existing quota usage record
func (r *quotaRepository) UpdateQuotaUsage(ctx context.Context, usage *models.QuotaUsage) error {
	usage.LastUpdated = time.Now().UTC()
	
	query := `
		UPDATE quota_usage 
		SET request_count = $2, 
		    success_count = $3, 
		    failure_count = $4, 
		    overage_count = $5, 
		    last_updated = $6
		WHERE id = $1`
	
	result, err := r.db.Exec(ctx, query, usage.ID, usage.RequestCount, usage.SuccessCount, usage.FailureCount, usage.OverageCount, usage.LastUpdated)
	if err != nil {
		return fmt.Errorf("failed to update quota usage: %w", err)
	}
	
	if result.RowsAffected() == 0 {
		return fmt.Errorf("quota usage not found")
	}
	
	return nil
}

// IncrementUsage atomically increments usage counters
func (r *quotaRepository) IncrementUsage(ctx context.Context, tenantID uuid.UUID, success bool) error {
	monthStart := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
	
	// Use upsert to handle concurrent requests
	query := `
		INSERT INTO quota_usage (id, tenant_id, month, request_count, success_count, failure_count, overage_count, last_updated, created_at)
		VALUES ($1, $2, $3, 1, $4, $5, 0, NOW(), NOW())
		ON CONFLICT (tenant_id, month) 
		DO UPDATE SET 
			request_count = quota_usage.request_count + 1,
			success_count = quota_usage.success_count + $4,
			failure_count = quota_usage.failure_count + $5,
			last_updated = NOW()`
	
	successIncr := 0
	failureIncr := 0
	if success {
		successIncr = 1
	} else {
		failureIncr = 1
	}
	
	_, err := r.db.Exec(ctx, query, uuid.New(), tenantID, monthStart, successIncr, failureIncr)
	if err != nil {
		return fmt.Errorf("failed to increment usage: %w", err)
	}
	
	return nil
}

// GetQuotaUsageByTenant gets quota usage for a specific tenant and month
func (r *quotaRepository) GetQuotaUsageByTenant(ctx context.Context, tenantID uuid.UUID, month time.Time) (*models.QuotaUsage, error) {
	monthStart := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	
	query := `
		SELECT id, tenant_id, month, request_count, success_count, failure_count, overage_count, last_updated, created_at
		FROM quota_usage 
		WHERE tenant_id = $1 AND month = $2`
	
	var usage models.QuotaUsage
	err := r.db.QueryRow(ctx, query, tenantID, monthStart).Scan(
		&usage.ID, &usage.TenantID, &usage.Month, &usage.RequestCount, 
		&usage.SuccessCount, &usage.FailureCount, &usage.OverageCount, 
		&usage.LastUpdated, &usage.CreatedAt)
	if err != nil {
		return nil, err
	}
	
	return &usage, nil
}

// CreateBillingRecord creates a new billing record
func (r *quotaRepository) CreateBillingRecord(ctx context.Context, record *models.BillingRecord) error {
	if record.ID == uuid.Nil {
		record.ID = uuid.New()
	}
	record.CreatedAt = time.Now().UTC()
	record.UpdatedAt = time.Now().UTC()
	
	query := `
		INSERT INTO billing_records (id, tenant_id, billing_period, base_requests, overage_requests, base_amount, overage_amount, total_amount, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	
	_, err := r.db.Exec(ctx, query, record.ID, record.TenantID, record.BillingPeriod, record.BaseRequests, record.OverageRequests, record.BaseAmount, record.OverageAmount, record.TotalAmount, record.Status, record.CreatedAt, record.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create billing record: %w", err)
	}
	
	return nil
}

// GetBillingRecord gets a billing record for a specific tenant and period
func (r *quotaRepository) GetBillingRecord(ctx context.Context, tenantID uuid.UUID, billingPeriod time.Time) (*models.BillingRecord, error) {
	periodStart := time.Date(billingPeriod.Year(), billingPeriod.Month(), 1, 0, 0, 0, 0, time.UTC)
	
	query := `
		SELECT id, tenant_id, billing_period, base_requests, overage_requests, base_amount, overage_amount, total_amount, status, created_at, updated_at
		FROM billing_records 
		WHERE tenant_id = $1 AND billing_period = $2`
	
	var record models.BillingRecord
	err := r.db.QueryRow(ctx, query, tenantID, periodStart).Scan(
		&record.ID, &record.TenantID, &record.BillingPeriod, &record.BaseRequests,
		&record.OverageRequests, &record.BaseAmount, &record.OverageAmount, 
		&record.TotalAmount, &record.Status, &record.CreatedAt, &record.UpdatedAt)
	if err != nil {
		return nil, err
	}
	
	return &record, nil
}

// UpdateBillingRecord updates an existing billing record
func (r *quotaRepository) UpdateBillingRecord(ctx context.Context, record *models.BillingRecord) error {
	record.UpdatedAt = time.Now().UTC()
	
	query := `
		UPDATE billing_records 
		SET base_requests = $2, 
		    overage_requests = $3, 
		    base_amount = $4, 
		    overage_amount = $5, 
		    total_amount = $6, 
		    status = $7, 
		    updated_at = $8
		WHERE id = $1`
	
	result, err := r.db.Exec(ctx, query, record.ID, record.BaseRequests, record.OverageRequests, record.BaseAmount, record.OverageAmount, record.TotalAmount, record.Status, record.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update billing record: %w", err)
	}
	
	if result.RowsAffected() == 0 {
		return fmt.Errorf("billing record not found")
	}
	
	return nil
}

// GetBillingHistory gets billing history for a tenant
func (r *quotaRepository) GetBillingHistory(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*models.BillingRecord, error) {
	query := `
		SELECT id, tenant_id, billing_period, base_requests, overage_requests, base_amount, overage_amount, total_amount, status, created_at, updated_at
		FROM billing_records 
		WHERE tenant_id = $1 
		ORDER BY billing_period DESC 
		LIMIT $2 OFFSET $3`
	
	rows, err := r.db.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get billing history: %w", err)
	}
	defer rows.Close()

	var records []*models.BillingRecord
	for rows.Next() {
		var record models.BillingRecord
		err := rows.Scan(&record.ID, &record.TenantID, &record.BillingPeriod, &record.BaseRequests,
			&record.OverageRequests, &record.BaseAmount, &record.OverageAmount, 
			&record.TotalAmount, &record.Status, &record.CreatedAt, &record.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan billing record: %w", err)
		}
		records = append(records, &record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate billing records: %w", err)
	}
	
	return records, nil
}

// CreateQuotaNotification creates a new quota notification
func (r *quotaRepository) CreateQuotaNotification(ctx context.Context, notification *models.QuotaNotification) error {
	if notification.ID == uuid.Nil {
		notification.ID = uuid.New()
	}
	notification.CreatedAt = time.Now().UTC()
	
	query := `
		INSERT INTO quota_notifications (id, tenant_id, type, threshold, usage_count, quota_limit, sent, sent_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	
	_, err := r.db.Exec(ctx, query, notification.ID, notification.TenantID, notification.Type, notification.Threshold, notification.UsageCount, notification.QuotaLimit, notification.Sent, notification.SentAt, notification.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create quota notification: %w", err)
	}
	
	return nil
}

// GetPendingNotifications gets pending notifications for a tenant
func (r *quotaRepository) GetPendingNotifications(ctx context.Context, tenantID uuid.UUID) ([]*models.QuotaNotification, error) {
	query := `
		SELECT id, tenant_id, type, threshold, usage_count, quota_limit, sent, sent_at, created_at
		FROM quota_notifications 
		WHERE tenant_id = $1 AND sent = false 
		ORDER BY created_at ASC`
	
	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending notifications: %w", err)
	}
	defer rows.Close()

	var notifications []*models.QuotaNotification
	for rows.Next() {
		var notification models.QuotaNotification
		err := rows.Scan(&notification.ID, &notification.TenantID, &notification.Type, &notification.Threshold,
			&notification.UsageCount, &notification.QuotaLimit, &notification.Sent, &notification.SentAt, &notification.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}
		notifications = append(notifications, &notification)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate notifications: %w", err)
	}
	
	return notifications, nil
}

// MarkNotificationSent marks a notification as sent
func (r *quotaRepository) MarkNotificationSent(ctx context.Context, notificationID uuid.UUID) error {
	now := time.Now().UTC()
	
	query := `
		UPDATE quota_notifications 
		SET sent = true, sent_at = $1 
		WHERE id = $2`
	
	result, err := r.db.Exec(ctx, query, now, notificationID)
	if err != nil {
		return fmt.Errorf("failed to mark notification as sent: %w", err)
	}
	
	if result.RowsAffected() == 0 {
		return fmt.Errorf("notification not found")
	}
	
	return nil
}

// GetNotificationHistory gets notification history for a tenant
func (r *quotaRepository) GetNotificationHistory(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*models.QuotaNotification, error) {
	query := `
		SELECT id, tenant_id, type, threshold, usage_count, quota_limit, sent, sent_at, created_at
		FROM quota_notifications 
		WHERE tenant_id = $1 
		ORDER BY created_at DESC 
		LIMIT $2 OFFSET $3`
	
	rows, err := r.db.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification history: %w", err)
	}
	defer rows.Close()

	var notifications []*models.QuotaNotification
	for rows.Next() {
		var notification models.QuotaNotification
		err := rows.Scan(&notification.ID, &notification.TenantID, &notification.Type, &notification.Threshold,
			&notification.UsageCount, &notification.QuotaLimit, &notification.Sent, &notification.SentAt, &notification.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}
		notifications = append(notifications, &notification)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate notifications: %w", err)
	}
	
	return notifications, nil
}