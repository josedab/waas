package cloudmanaged

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Repository defines the data access interface for cloud managed resources
type Repository interface {
	// Tenants
	CreateCloudTenant(ctx context.Context, tenant *CloudTenant) error
	GetCloudTenant(ctx context.Context, tenantID string) (*CloudTenant, error)
	UpdateCloudTenant(ctx context.Context, tenant *CloudTenant) error
	ListCloudTenants(ctx context.Context, limit, offset int) ([]CloudTenant, error)

	// Usage
	RecordUsage(ctx context.Context, meter *UsageMeter) error
	GetUsageSummary(ctx context.Context, tenantID, period string) (*UsageSummary, error)
	GetUsageHistory(ctx context.Context, tenantID string, limit int) ([]UsageMeter, error)

	// Billing
	SaveBillingInfo(ctx context.Context, info *BillingInfo) error
	GetBillingInfo(ctx context.Context, tenantID string) (*BillingInfo, error)

	// Onboarding
	SaveOnboardingProgress(ctx context.Context, progress *OnboardingProgress) error
	GetOnboardingProgress(ctx context.Context, tenantID string) (*OnboardingProgress, error)
}

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// CreateCloudTenant creates a new cloud tenant
func (r *PostgresRepository) CreateCloudTenant(ctx context.Context, tenant *CloudTenant) error {
	query := `
		INSERT INTO cloud_tenants (
			id, tenant_id, email, org, plan, status, region,
			webhooks_used, webhooks_limit, storage_used, storage_limit,
			created_at, trial_ends_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	_, err := r.db.ExecContext(ctx, query,
		tenant.ID, tenant.TenantID, tenant.Email, tenant.Org, tenant.Plan,
		tenant.Status, tenant.Region, tenant.WebhooksUsed, tenant.WebhooksLimit,
		tenant.StorageUsed, tenant.StorageLimit, tenant.CreatedAt, tenant.TrialEndsAt)

	return err
}

// GetCloudTenant retrieves a cloud tenant by tenant ID
func (r *PostgresRepository) GetCloudTenant(ctx context.Context, tenantID string) (*CloudTenant, error) {
	query := `
		SELECT id, tenant_id, email, org, plan, status, region,
			   webhooks_used, webhooks_limit, storage_used, storage_limit,
			   created_at, trial_ends_at
		FROM cloud_tenants
		WHERE tenant_id = $1`

	var tenant CloudTenant
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(
		&tenant.ID, &tenant.TenantID, &tenant.Email, &tenant.Org, &tenant.Plan,
		&tenant.Status, &tenant.Region, &tenant.WebhooksUsed, &tenant.WebhooksLimit,
		&tenant.StorageUsed, &tenant.StorageLimit, &tenant.CreatedAt, &tenant.TrialEndsAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("cloud tenant not found")
	}
	if err != nil {
		return nil, err
	}

	return &tenant, nil
}

// UpdateCloudTenant updates an existing cloud tenant
func (r *PostgresRepository) UpdateCloudTenant(ctx context.Context, tenant *CloudTenant) error {
	query := `
		UPDATE cloud_tenants SET
			email = $2, org = $3, plan = $4, status = $5, region = $6,
			webhooks_used = $7, webhooks_limit = $8,
			storage_used = $9, storage_limit = $10,
			trial_ends_at = $11
		WHERE tenant_id = $1`

	_, err := r.db.ExecContext(ctx, query,
		tenant.TenantID, tenant.Email, tenant.Org, tenant.Plan, tenant.Status,
		tenant.Region, tenant.WebhooksUsed, tenant.WebhooksLimit,
		tenant.StorageUsed, tenant.StorageLimit, tenant.TrialEndsAt)

	return err
}

// ListCloudTenants lists cloud tenants with pagination
func (r *PostgresRepository) ListCloudTenants(ctx context.Context, limit, offset int) ([]CloudTenant, error) {
	query := `
		SELECT id, tenant_id, email, org, plan, status, region,
			   webhooks_used, webhooks_limit, storage_used, storage_limit,
			   created_at, trial_ends_at
		FROM cloud_tenants
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tenants []CloudTenant
	for rows.Next() {
		var t CloudTenant
		if err := rows.Scan(
			&t.ID, &t.TenantID, &t.Email, &t.Org, &t.Plan, &t.Status, &t.Region,
			&t.WebhooksUsed, &t.WebhooksLimit, &t.StorageUsed, &t.StorageLimit,
			&t.CreatedAt, &t.TrialEndsAt); err != nil {
			return nil, err
		}
		tenants = append(tenants, t)
	}

	return tenants, rows.Err()
}

// RecordUsage records a usage metric
func (r *PostgresRepository) RecordUsage(ctx context.Context, meter *UsageMeter) error {
	query := `
		INSERT INTO cloud_usage_meters (id, tenant_id, metric_type, value, period, recorded_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.db.ExecContext(ctx, query,
		meter.ID, meter.TenantID, meter.MetricType, meter.Value,
		meter.Period, meter.RecordedAt)

	return err
}

// GetUsageSummary retrieves aggregated usage for a tenant and period
func (r *PostgresRepository) GetUsageSummary(ctx context.Context, tenantID, period string) (*UsageSummary, error) {
	summary := &UsageSummary{
		TenantID: tenantID,
		Period:   period,
	}

	query := `
		SELECT metric_type, COALESCE(SUM(value), 0)
		FROM cloud_usage_meters
		WHERE tenant_id = $1 AND period = $2
		GROUP BY metric_type`

	rows, err := r.db.QueryContext(ctx, query, tenantID, period)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var metricType string
		var value int64
		if err := rows.Scan(&metricType, &value); err != nil {
			continue
		}
		switch metricType {
		case "webhooks_sent":
			summary.WebhooksSent = value
		case "webhooks_received":
			summary.WebhooksRecvd = value
		case "bandwidth":
			summary.Bandwidth = value
		case "active_endpoints":
			summary.ActiveEndpoints = value
		case "api_requests":
			summary.APIRequests = value
		case "storage_used":
			summary.StorageUsed = value
		}
	}

	return summary, rows.Err()
}

// GetUsageHistory retrieves recent usage meter records for a tenant
func (r *PostgresRepository) GetUsageHistory(ctx context.Context, tenantID string, limit int) ([]UsageMeter, error) {
	query := `
		SELECT id, tenant_id, metric_type, value, period, recorded_at
		FROM cloud_usage_meters
		WHERE tenant_id = $1
		ORDER BY recorded_at DESC
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var meters []UsageMeter
	for rows.Next() {
		var m UsageMeter
		if err := rows.Scan(&m.ID, &m.TenantID, &m.MetricType, &m.Value,
			&m.Period, &m.RecordedAt); err != nil {
			return nil, err
		}
		meters = append(meters, m)
	}

	return meters, rows.Err()
}

// SaveBillingInfo saves or updates billing information
func (r *PostgresRepository) SaveBillingInfo(ctx context.Context, info *BillingInfo) error {
	query := `
		INSERT INTO cloud_billing_info (
			tenant_id, stripe_customer_id, stripe_plan_id,
			payment_method, billing_email, next_billing_date, amount_due
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (tenant_id) DO UPDATE SET
			stripe_customer_id = EXCLUDED.stripe_customer_id,
			stripe_plan_id = EXCLUDED.stripe_plan_id,
			payment_method = EXCLUDED.payment_method,
			billing_email = EXCLUDED.billing_email,
			next_billing_date = EXCLUDED.next_billing_date,
			amount_due = EXCLUDED.amount_due`

	_, err := r.db.ExecContext(ctx, query,
		info.TenantID, info.StripeCustomerID, info.StripePlanID,
		info.PaymentMethod, info.BillingEmail, info.NextBillingDate, info.AmountDue)

	return err
}

// GetBillingInfo retrieves billing information for a tenant
func (r *PostgresRepository) GetBillingInfo(ctx context.Context, tenantID string) (*BillingInfo, error) {
	query := `
		SELECT tenant_id, stripe_customer_id, stripe_plan_id,
			   payment_method, billing_email, next_billing_date, amount_due
		FROM cloud_billing_info
		WHERE tenant_id = $1`

	var info BillingInfo
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(
		&info.TenantID, &info.StripeCustomerID, &info.StripePlanID,
		&info.PaymentMethod, &info.BillingEmail, &info.NextBillingDate, &info.AmountDue)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("billing info not found")
	}
	if err != nil {
		return nil, err
	}

	return &info, nil
}

// SaveOnboardingProgress saves onboarding progress
func (r *PostgresRepository) SaveOnboardingProgress(ctx context.Context, progress *OnboardingProgress) error {
	stepsJSON, _ := json.Marshal(progress.Steps)

	query := `
		INSERT INTO cloud_onboarding (tenant_id, steps, completion_pct, all_completed)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (tenant_id) DO UPDATE SET
			steps = EXCLUDED.steps,
			completion_pct = EXCLUDED.completion_pct,
			all_completed = EXCLUDED.all_completed`

	_, err := r.db.ExecContext(ctx, query,
		progress.TenantID, stepsJSON, progress.CompletionPct, progress.AllCompleted)

	return err
}

// GetOnboardingProgress retrieves onboarding progress for a tenant
func (r *PostgresRepository) GetOnboardingProgress(ctx context.Context, tenantID string) (*OnboardingProgress, error) {
	query := `
		SELECT tenant_id, steps, completion_pct, all_completed
		FROM cloud_onboarding
		WHERE tenant_id = $1`

	var progress OnboardingProgress
	var stepsJSON []byte

	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(
		&progress.TenantID, &stepsJSON, &progress.CompletionPct, &progress.AllCompleted)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("onboarding progress not found")
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(stepsJSON, &progress.Steps)
	return &progress, nil
}

// Ensure imports are used
var _ = time.Now
