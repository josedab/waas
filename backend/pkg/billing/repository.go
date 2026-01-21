package billing

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Repository defines billing data access
type Repository interface {
	// Usage records
	RecordUsage(ctx context.Context, record *UsageRecord) error
	GetUsageSummary(ctx context.Context, tenantID, period string) (*UsageSummary, error)
	GetUsageByResource(ctx context.Context, tenantID, resourceType, period string) ([]UsageRecord, error)

	// Spend tracking
	GetSpendTracker(ctx context.Context, tenantID string, period BillingPeriod) (*SpendTracker, error)
	UpdateSpendTracker(ctx context.Context, tracker *SpendTracker) error
	GetCurrentSpend(ctx context.Context, tenantID string) (float64, error)

	// Budgets
	SaveBudget(ctx context.Context, budget *BudgetConfig) error
	GetBudget(ctx context.Context, tenantID, budgetID string) (*BudgetConfig, error)
	ListBudgets(ctx context.Context, tenantID string) ([]BudgetConfig, error)
	DeleteBudget(ctx context.Context, tenantID, budgetID string) error

	// Alerts
	SaveAlert(ctx context.Context, alert *BillingAlert) error
	GetAlert(ctx context.Context, alertID string) (*BillingAlert, error)
	ListAlerts(ctx context.Context, tenantID string, status *AlertStatus) ([]BillingAlert, error)
	UpdateAlertStatus(ctx context.Context, alertID string, status AlertStatus, ackedBy string) error

	// Optimizations
	SaveOptimization(ctx context.Context, opt *CostOptimization) error
	ListOptimizations(ctx context.Context, tenantID string) ([]CostOptimization, error)
	UpdateOptimizationStatus(ctx context.Context, optID string, status OptimizationStatus) error

	// Invoices
	SaveInvoice(ctx context.Context, invoice *Invoice) error
	GetInvoice(ctx context.Context, tenantID, invoiceID string) (*Invoice, error)
	ListInvoices(ctx context.Context, tenantID string) ([]Invoice, error)

	// Alert config
	SaveAlertConfig(ctx context.Context, config *AlertConfig) error
	GetAlertConfig(ctx context.Context, tenantID string) (*AlertConfig, error)
}

// PostgresRepository implements Repository with PostgreSQL
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// RecordUsage records a usage event
func (r *PostgresRepository) RecordUsage(ctx context.Context, record *UsageRecord) error {
	query := `
		INSERT INTO billing_usage (
			id, tenant_id, webhook_id, resource_type, quantity,
			unit_cost, total_cost, currency, billing_period, recorded_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.db.ExecContext(ctx, query,
		record.ID, record.TenantID, record.WebhookID, record.ResourceType,
		record.Quantity, record.UnitCost, record.TotalCost, record.Currency,
		record.BillingPeriod, record.RecordedAt)

	return err
}

// GetUsageSummary retrieves usage summary for a period
func (r *PostgresRepository) GetUsageSummary(ctx context.Context, tenantID, period string) (*UsageSummary, error) {
	summary := &UsageSummary{
		TenantID:   tenantID,
		Period:     period,
		Currency:   "USD",
		ByResource: make(map[string]ResourceUsage),
	}

	// Get totals
	query := `
		SELECT 
			SUM(CASE WHEN resource_type = 'webhook_requests' THEN quantity ELSE 0 END) as requests,
			SUM(CASE WHEN resource_type = 'webhook_deliveries' THEN quantity ELSE 0 END) as delivered,
			SUM(CASE WHEN resource_type IN ('retry_attempts') AND quantity < 0 THEN quantity ELSE 0 END) as failed,
			SUM(CASE WHEN resource_type = 'data_transfer_bytes' THEN quantity ELSE 0 END) as bytes,
			COALESCE(SUM(total_cost), 0) as total_cost
		FROM billing_usage
		WHERE tenant_id = $1 AND billing_period = $2`

	err := r.db.QueryRowContext(ctx, query, tenantID, period).Scan(
		&summary.TotalRequests, &summary.TotalDelivered, &summary.TotalFailed,
		&summary.TotalBytes, &summary.TotalCost)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Get breakdown by resource
	breakdownQuery := `
		SELECT resource_type, SUM(quantity), SUM(total_cost)
		FROM billing_usage
		WHERE tenant_id = $1 AND billing_period = $2
		GROUP BY resource_type`

	rows, err := r.db.QueryContext(ctx, breakdownQuery, tenantID, period)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ru ResourceUsage
		if err := rows.Scan(&ru.ResourceType, &ru.Quantity, &ru.Cost); err != nil {
			continue
		}
		summary.ByResource[ru.ResourceType] = ru
	}

	// Get daily breakdown
	dailyQuery := `
		SELECT DATE(recorded_at)::text, SUM(quantity), SUM(total_cost)
		FROM billing_usage
		WHERE tenant_id = $1 AND billing_period = $2 AND resource_type = 'webhook_requests'
		GROUP BY DATE(recorded_at)
		ORDER BY DATE(recorded_at)`

	dailyRows, err := r.db.QueryContext(ctx, dailyQuery, tenantID, period)
	if err == nil {
		defer dailyRows.Close()
		for dailyRows.Next() {
			var du DailyUsage
			if err := dailyRows.Scan(&du.Date, &du.Requests, &du.Cost); err != nil {
				continue
			}
			summary.ByDay = append(summary.ByDay, du)
		}
	}

	return summary, nil
}

// GetUsageByResource retrieves usage for a specific resource type
func (r *PostgresRepository) GetUsageByResource(ctx context.Context, tenantID, resourceType, period string) ([]UsageRecord, error) {
	query := `
		SELECT id, tenant_id, webhook_id, resource_type, quantity,
			   unit_cost, total_cost, currency, billing_period, recorded_at
		FROM billing_usage
		WHERE tenant_id = $1 AND resource_type = $2 AND billing_period = $3
		ORDER BY recorded_at DESC`

	rows, err := r.db.QueryContext(ctx, query, tenantID, resourceType, period)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []UsageRecord
	for rows.Next() {
		var r UsageRecord
		var webhookID sql.NullString
		if err := rows.Scan(&r.ID, &r.TenantID, &webhookID, &r.ResourceType,
			&r.Quantity, &r.UnitCost, &r.TotalCost, &r.Currency,
			&r.BillingPeriod, &r.RecordedAt); err != nil {
			continue
		}
		if webhookID.Valid {
			r.WebhookID = webhookID.String
		}
		records = append(records, r)
	}

	return records, nil
}

// GetSpendTracker retrieves spend tracker
func (r *PostgresRepository) GetSpendTracker(ctx context.Context, tenantID string, period BillingPeriod) (*SpendTracker, error) {
	query := `
		SELECT id, tenant_id, budget_limit, current_spend, currency, period,
			   period_start, period_end, breakdown, alerts, status, updated_at
		FROM billing_spend_trackers
		WHERE tenant_id = $1 AND period = $2
		ORDER BY period_start DESC
		LIMIT 1`

	var tracker SpendTracker
	var breakdownJSON, alertsJSON []byte

	err := r.db.QueryRowContext(ctx, query, tenantID, period).Scan(
		&tracker.ID, &tracker.TenantID, &tracker.BudgetLimit, &tracker.CurrentSpend,
		&tracker.Currency, &tracker.Period, &tracker.PeriodStart, &tracker.PeriodEnd,
		&breakdownJSON, &alertsJSON, &tracker.Status, &tracker.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("spend tracker not found")
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(breakdownJSON, &tracker.Breakdown)
	json.Unmarshal(alertsJSON, &tracker.Alerts)

	return &tracker, nil
}

// UpdateSpendTracker updates spend tracker
func (r *PostgresRepository) UpdateSpendTracker(ctx context.Context, tracker *SpendTracker) error {
	breakdownJSON, _ := json.Marshal(tracker.Breakdown)
	alertsJSON, _ := json.Marshal(tracker.Alerts)

	query := `
		INSERT INTO billing_spend_trackers (
			id, tenant_id, budget_limit, current_spend, currency, period,
			period_start, period_end, breakdown, alerts, status, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (tenant_id, period, period_start) DO UPDATE SET
			current_spend = EXCLUDED.current_spend,
			breakdown = EXCLUDED.breakdown,
			alerts = EXCLUDED.alerts,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at`

	_, err := r.db.ExecContext(ctx, query,
		tracker.ID, tracker.TenantID, tracker.BudgetLimit, tracker.CurrentSpend,
		tracker.Currency, tracker.Period, tracker.PeriodStart, tracker.PeriodEnd,
		breakdownJSON, alertsJSON, tracker.Status, tracker.UpdatedAt)

	return err
}

// GetCurrentSpend gets current period spend
func (r *PostgresRepository) GetCurrentSpend(ctx context.Context, tenantID string) (float64, error) {
	period := time.Now().Format("2006-01")
	query := `
		SELECT COALESCE(SUM(total_cost), 0)
		FROM billing_usage
		WHERE tenant_id = $1 AND billing_period = $2`

	var spend float64
	err := r.db.QueryRowContext(ctx, query, tenantID, period).Scan(&spend)
	return spend, err
}

// SaveBudget saves a budget configuration
func (r *PostgresRepository) SaveBudget(ctx context.Context, budget *BudgetConfig) error {
	alertsJSON, _ := json.Marshal(budget.Alerts)

	query := `
		INSERT INTO billing_budgets (
			id, tenant_id, name, amount, currency, period,
			resource_type, webhook_id, alerts, auto_pause, enabled,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			amount = EXCLUDED.amount,
			currency = EXCLUDED.currency,
			period = EXCLUDED.period,
			resource_type = EXCLUDED.resource_type,
			alerts = EXCLUDED.alerts,
			auto_pause = EXCLUDED.auto_pause,
			enabled = EXCLUDED.enabled,
			updated_at = EXCLUDED.updated_at`

	_, err := r.db.ExecContext(ctx, query,
		budget.ID, budget.TenantID, budget.Name, budget.Amount, budget.Currency,
		budget.Period, budget.ResourceType, budget.WebhookID, alertsJSON,
		budget.AutoPause, budget.Enabled, budget.CreatedAt, budget.UpdatedAt)

	return err
}

// GetBudget retrieves a budget
func (r *PostgresRepository) GetBudget(ctx context.Context, tenantID, budgetID string) (*BudgetConfig, error) {
	query := `
		SELECT id, tenant_id, name, amount, currency, period,
			   resource_type, webhook_id, alerts, auto_pause, enabled,
			   created_at, updated_at
		FROM billing_budgets
		WHERE tenant_id = $1 AND id = $2`

	var budget BudgetConfig
	var alertsJSON []byte
	var resourceType, webhookID sql.NullString

	err := r.db.QueryRowContext(ctx, query, tenantID, budgetID).Scan(
		&budget.ID, &budget.TenantID, &budget.Name, &budget.Amount, &budget.Currency,
		&budget.Period, &resourceType, &webhookID, &alertsJSON, &budget.AutoPause,
		&budget.Enabled, &budget.CreatedAt, &budget.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("budget not found")
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(alertsJSON, &budget.Alerts)
	if resourceType.Valid {
		budget.ResourceType = resourceType.String
	}
	if webhookID.Valid {
		budget.WebhookID = webhookID.String
	}

	return &budget, nil
}

// ListBudgets lists budgets
func (r *PostgresRepository) ListBudgets(ctx context.Context, tenantID string) ([]BudgetConfig, error) {
	query := `
		SELECT id, tenant_id, name, amount, currency, period,
			   resource_type, webhook_id, alerts, auto_pause, enabled,
			   created_at, updated_at
		FROM billing_budgets
		WHERE tenant_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var budgets []BudgetConfig
	for rows.Next() {
		var budget BudgetConfig
		var alertsJSON []byte
		var resourceType, webhookID sql.NullString

		err := rows.Scan(
			&budget.ID, &budget.TenantID, &budget.Name, &budget.Amount, &budget.Currency,
			&budget.Period, &resourceType, &webhookID, &alertsJSON, &budget.AutoPause,
			&budget.Enabled, &budget.CreatedAt, &budget.UpdatedAt)
		if err != nil {
			continue
		}

		json.Unmarshal(alertsJSON, &budget.Alerts)
		if resourceType.Valid {
			budget.ResourceType = resourceType.String
		}
		if webhookID.Valid {
			budget.WebhookID = webhookID.String
		}

		budgets = append(budgets, budget)
	}

	return budgets, nil
}

// DeleteBudget deletes a budget
func (r *PostgresRepository) DeleteBudget(ctx context.Context, tenantID, budgetID string) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM billing_budgets WHERE tenant_id = $1 AND id = $2",
		tenantID, budgetID)
	return err
}

// SaveAlert saves an alert
func (r *PostgresRepository) SaveAlert(ctx context.Context, alert *BillingAlert) error {
	dataJSON, _ := json.Marshal(alert.Data)
	channelsJSON, _ := json.Marshal(alert.Channels)

	query := `
		INSERT INTO billing_alerts (
			id, tenant_id, type, severity, title, message, data,
			status, channels, sent_at, acked_at, acked_by, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			sent_at = EXCLUDED.sent_at,
			acked_at = EXCLUDED.acked_at,
			acked_by = EXCLUDED.acked_by`

	_, err := r.db.ExecContext(ctx, query,
		alert.ID, alert.TenantID, alert.Type, alert.Severity, alert.Title,
		alert.Message, dataJSON, alert.Status, channelsJSON,
		alert.SentAt, alert.AckedAt, alert.AckedBy, alert.CreatedAt)

	return err
}

// GetAlert retrieves an alert
func (r *PostgresRepository) GetAlert(ctx context.Context, alertID string) (*BillingAlert, error) {
	query := `
		SELECT id, tenant_id, type, severity, title, message, data,
			   status, channels, sent_at, acked_at, acked_by, created_at
		FROM billing_alerts
		WHERE id = $1`

	var alert BillingAlert
	var dataJSON, channelsJSON []byte
	var sentAt, ackedAt sql.NullTime
	var ackedBy sql.NullString

	err := r.db.QueryRowContext(ctx, query, alertID).Scan(
		&alert.ID, &alert.TenantID, &alert.Type, &alert.Severity, &alert.Title,
		&alert.Message, &dataJSON, &alert.Status, &channelsJSON,
		&sentAt, &ackedAt, &ackedBy, &alert.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("alert not found")
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(dataJSON, &alert.Data)
	json.Unmarshal(channelsJSON, &alert.Channels)
	if sentAt.Valid {
		alert.SentAt = &sentAt.Time
	}
	if ackedAt.Valid {
		alert.AckedAt = &ackedAt.Time
	}
	if ackedBy.Valid {
		alert.AckedBy = ackedBy.String
	}

	return &alert, nil
}

// ListAlerts lists alerts
func (r *PostgresRepository) ListAlerts(ctx context.Context, tenantID string, status *AlertStatus) ([]BillingAlert, error) {
	query := `
		SELECT id, tenant_id, type, severity, title, message, data,
			   status, channels, sent_at, acked_at, acked_by, created_at
		FROM billing_alerts
		WHERE tenant_id = $1`
	args := []any{tenantID}

	if status != nil {
		query += " AND status = $2"
		args = append(args, *status)
	}

	query += " ORDER BY created_at DESC LIMIT 100"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []BillingAlert
	for rows.Next() {
		var alert BillingAlert
		var dataJSON, channelsJSON []byte
		var sentAt, ackedAt sql.NullTime
		var ackedBy sql.NullString

		err := rows.Scan(
			&alert.ID, &alert.TenantID, &alert.Type, &alert.Severity, &alert.Title,
			&alert.Message, &dataJSON, &alert.Status, &channelsJSON,
			&sentAt, &ackedAt, &ackedBy, &alert.CreatedAt)
		if err != nil {
			continue
		}

		json.Unmarshal(dataJSON, &alert.Data)
		json.Unmarshal(channelsJSON, &alert.Channels)
		if sentAt.Valid {
			alert.SentAt = &sentAt.Time
		}
		if ackedAt.Valid {
			alert.AckedAt = &ackedAt.Time
		}
		if ackedBy.Valid {
			alert.AckedBy = ackedBy.String
		}

		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// UpdateAlertStatus updates alert status
func (r *PostgresRepository) UpdateAlertStatus(ctx context.Context, alertID string, status AlertStatus, ackedBy string) error {
	var ackedAt *time.Time
	if status == AlertAcked {
		now := time.Now()
		ackedAt = &now
	}

	_, err := r.db.ExecContext(ctx,
		"UPDATE billing_alerts SET status = $1, acked_at = $2, acked_by = $3 WHERE id = $4",
		status, ackedAt, ackedBy, alertID)
	return err
}

// SaveOptimization saves an optimization
func (r *PostgresRepository) SaveOptimization(ctx context.Context, opt *CostOptimization) error {
	actionsJSON, _ := json.Marshal(opt.Actions)

	query := `
		INSERT INTO billing_optimizations (
			id, tenant_id, type, title, description, estimated_savings,
			currency, impact, resource_id, resource_type, actions,
			status, implemented_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			implemented_at = EXCLUDED.implemented_at`

	_, err := r.db.ExecContext(ctx, query,
		opt.ID, opt.TenantID, opt.Type, opt.Title, opt.Description,
		opt.EstimatedSavings, opt.Currency, opt.Impact, opt.ResourceID,
		opt.ResourceType, actionsJSON, opt.Status, opt.ImplementedAt, opt.CreatedAt)

	return err
}

// ListOptimizations lists optimizations
func (r *PostgresRepository) ListOptimizations(ctx context.Context, tenantID string) ([]CostOptimization, error) {
	query := `
		SELECT id, tenant_id, type, title, description, estimated_savings,
			   currency, impact, resource_id, resource_type, actions,
			   status, implemented_at, created_at
		FROM billing_optimizations
		WHERE tenant_id = $1
		ORDER BY estimated_savings DESC`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var opts []CostOptimization
	for rows.Next() {
		var opt CostOptimization
		var actionsJSON []byte
		var resourceID, resourceType sql.NullString
		var implementedAt sql.NullTime

		err := rows.Scan(
			&opt.ID, &opt.TenantID, &opt.Type, &opt.Title, &opt.Description,
			&opt.EstimatedSavings, &opt.Currency, &opt.Impact, &resourceID,
			&resourceType, &actionsJSON, &opt.Status, &implementedAt, &opt.CreatedAt)
		if err != nil {
			continue
		}

		json.Unmarshal(actionsJSON, &opt.Actions)
		if resourceID.Valid {
			opt.ResourceID = resourceID.String
		}
		if resourceType.Valid {
			opt.ResourceType = resourceType.String
		}
		if implementedAt.Valid {
			opt.ImplementedAt = &implementedAt.Time
		}

		opts = append(opts, opt)
	}

	return opts, nil
}

// UpdateOptimizationStatus updates optimization status
func (r *PostgresRepository) UpdateOptimizationStatus(ctx context.Context, optID string, status OptimizationStatus) error {
	var implementedAt *time.Time
	if status == OptStatusImplemented {
		now := time.Now()
		implementedAt = &now
	}

	_, err := r.db.ExecContext(ctx,
		"UPDATE billing_optimizations SET status = $1, implemented_at = $2 WHERE id = $3",
		status, implementedAt, optID)
	return err
}

// SaveInvoice saves an invoice
func (r *PostgresRepository) SaveInvoice(ctx context.Context, invoice *Invoice) error {
	lineItemsJSON, _ := json.Marshal(invoice.LineItems)

	query := `
		INSERT INTO billing_invoices (
			id, tenant_id, number, status, period, subtotal, discount,
			tax, total, currency, line_items, due_date, paid_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			paid_at = EXCLUDED.paid_at`

	_, err := r.db.ExecContext(ctx, query,
		invoice.ID, invoice.TenantID, invoice.Number, invoice.Status,
		invoice.Period, invoice.Subtotal, invoice.Discount, invoice.Tax,
		invoice.Total, invoice.Currency, lineItemsJSON, invoice.DueDate,
		invoice.PaidAt, invoice.CreatedAt)

	return err
}

// GetInvoice retrieves an invoice
func (r *PostgresRepository) GetInvoice(ctx context.Context, tenantID, invoiceID string) (*Invoice, error) {
	query := `
		SELECT id, tenant_id, number, status, period, subtotal, discount,
			   tax, total, currency, line_items, due_date, paid_at, created_at
		FROM billing_invoices
		WHERE tenant_id = $1 AND id = $2`

	var invoice Invoice
	var lineItemsJSON []byte
	var paidAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, tenantID, invoiceID).Scan(
		&invoice.ID, &invoice.TenantID, &invoice.Number, &invoice.Status,
		&invoice.Period, &invoice.Subtotal, &invoice.Discount, &invoice.Tax,
		&invoice.Total, &invoice.Currency, &lineItemsJSON, &invoice.DueDate,
		&paidAt, &invoice.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invoice not found")
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(lineItemsJSON, &invoice.LineItems)
	if paidAt.Valid {
		invoice.PaidAt = &paidAt.Time
	}

	return &invoice, nil
}

// ListInvoices lists invoices
func (r *PostgresRepository) ListInvoices(ctx context.Context, tenantID string) ([]Invoice, error) {
	query := `
		SELECT id, tenant_id, number, status, period, subtotal, discount,
			   tax, total, currency, line_items, due_date, paid_at, created_at
		FROM billing_invoices
		WHERE tenant_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []Invoice
	for rows.Next() {
		var invoice Invoice
		var lineItemsJSON []byte
		var paidAt sql.NullTime

		err := rows.Scan(
			&invoice.ID, &invoice.TenantID, &invoice.Number, &invoice.Status,
			&invoice.Period, &invoice.Subtotal, &invoice.Discount, &invoice.Tax,
			&invoice.Total, &invoice.Currency, &lineItemsJSON, &invoice.DueDate,
			&paidAt, &invoice.CreatedAt)
		if err != nil {
			continue
		}

		json.Unmarshal(lineItemsJSON, &invoice.LineItems)
		if paidAt.Valid {
			invoice.PaidAt = &paidAt.Time
		}

		invoices = append(invoices, invoice)
	}

	return invoices, nil
}

// SaveAlertConfig saves alert configuration
func (r *PostgresRepository) SaveAlertConfig(ctx context.Context, config *AlertConfig) error {
	channelsJSON, _ := json.Marshal(config.Channels)
	recipientsJSON, _ := json.Marshal(config.Recipients)
	scheduleJSON, _ := json.Marshal(config.Schedule)

	query := `
		INSERT INTO billing_alert_configs (
			id, tenant_id, enabled, channels, recipients, schedule,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (tenant_id) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			channels = EXCLUDED.channels,
			recipients = EXCLUDED.recipients,
			schedule = EXCLUDED.schedule,
			updated_at = EXCLUDED.updated_at`

	_, err := r.db.ExecContext(ctx, query,
		config.ID, config.TenantID, config.Enabled, channelsJSON,
		recipientsJSON, scheduleJSON, config.CreatedAt, config.UpdatedAt)

	return err
}

// GetAlertConfig retrieves alert configuration
func (r *PostgresRepository) GetAlertConfig(ctx context.Context, tenantID string) (*AlertConfig, error) {
	query := `
		SELECT id, tenant_id, enabled, channels, recipients, schedule,
			   created_at, updated_at
		FROM billing_alert_configs
		WHERE tenant_id = $1`

	var config AlertConfig
	var channelsJSON, recipientsJSON, scheduleJSON []byte

	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(
		&config.ID, &config.TenantID, &config.Enabled, &channelsJSON,
		&recipientsJSON, &scheduleJSON, &config.CreatedAt, &config.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("alert config not found")
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(channelsJSON, &config.Channels)
	json.Unmarshal(recipientsJSON, &config.Recipients)
	json.Unmarshal(scheduleJSON, &config.Schedule)

	return &config, nil
}

// GenerateBudgetID generates a new budget ID
func GenerateBudgetID() string {
	return uuid.New().String()
}

// GenerateAlertID generates a new alert ID
func GenerateAlertID() string {
	return uuid.New().String()
}
