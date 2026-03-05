package costing

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines the interface for cost storage
type Repository interface {
	CreateUsageRecord(ctx context.Context, record *UsageRecord) error
	GetUsageSummary(ctx context.Context, tenantID string, startDate, endDate time.Time) (*UsageSummary, error)
	GetUsageByEndpoint(ctx context.Context, tenantID string, startDate, endDate time.Time) (map[string]UsageSummary, error)
	GetDailyCosts(ctx context.Context, tenantID string, startDate, endDate time.Time) ([]DailyCost, error)

	GetCostAllocation(ctx context.Context, tenantID, period, resourceType, resourceID string) (*CostAllocation, error)
	GetCostAllocationsByResource(ctx context.Context, tenantID, period, resourceType string) ([]CostAllocation, error)
	SaveCostAllocation(ctx context.Context, allocation *CostAllocation) error

	CreateBudget(ctx context.Context, budget *Budget) error
	GetBudget(ctx context.Context, tenantID, budgetID string) (*Budget, error)
	ListBudgets(ctx context.Context, tenantID string, limit, offset int) ([]Budget, int, error)
	UpdateBudget(ctx context.Context, budget *Budget) error
	DeleteBudget(ctx context.Context, tenantID, budgetID string) error
}

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// CreateUsageRecord creates a new usage record
func (r *PostgresRepository) CreateUsageRecord(ctx context.Context, record *UsageRecord) error {
	if record.ID == "" {
		record.ID = uuid.New().String()
	}

	metadataJSON, _ := json.Marshal(record.Metadata)

	query := `
		INSERT INTO usage_records (id, tenant_id, endpoint_id, webhook_id, unit, quantity, metadata, recorded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(ctx, query,
		record.ID, record.TenantID, record.EndpointID, record.WebhookID,
		record.Unit, record.Quantity, metadataJSON, record.RecordedAt,
	)

	return err
}

// GetUsageSummary retrieves usage summary for a tenant
func (r *PostgresRepository) GetUsageSummary(ctx context.Context, tenantID string, startDate, endDate time.Time) (*UsageSummary, error) {
	query := `
		SELECT 
			COALESCE(SUM(CASE WHEN unit = 'delivery' THEN quantity ELSE 0 END), 0) as deliveries,
			COALESCE(SUM(CASE WHEN unit = 'byte' THEN quantity ELSE 0 END), 0) as bytes,
			COALESCE(SUM(CASE WHEN unit = 'retry' THEN quantity ELSE 0 END), 0) as retries,
			COALESCE(SUM(CASE WHEN unit = 'transform' THEN quantity ELSE 0 END), 0) as transformations
		FROM usage_records
		WHERE tenant_id = $1 AND recorded_at >= $2 AND recorded_at < $3
	`

	var summary UsageSummary
	err := r.db.QueryRowContext(ctx, query, tenantID, startDate, endDate).Scan(
		&summary.Deliveries, &summary.Bytes, &summary.Retries, &summary.Transformations,
	)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	return &summary, nil
}

// GetUsageByEndpoint retrieves usage grouped by endpoint
func (r *PostgresRepository) GetUsageByEndpoint(ctx context.Context, tenantID string, startDate, endDate time.Time) (map[string]UsageSummary, error) {
	query := `
		SELECT 
			endpoint_id,
			COALESCE(SUM(CASE WHEN unit = 'delivery' THEN quantity ELSE 0 END), 0) as deliveries,
			COALESCE(SUM(CASE WHEN unit = 'byte' THEN quantity ELSE 0 END), 0) as bytes,
			COALESCE(SUM(CASE WHEN unit = 'retry' THEN quantity ELSE 0 END), 0) as retries,
			COALESCE(SUM(CASE WHEN unit = 'transform' THEN quantity ELSE 0 END), 0) as transformations
		FROM usage_records
		WHERE tenant_id = $1 AND recorded_at >= $2 AND recorded_at < $3 AND endpoint_id IS NOT NULL
		GROUP BY endpoint_id
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]UsageSummary)
	for rows.Next() {
		var endpointID string
		var summary UsageSummary

		if err := rows.Scan(&endpointID, &summary.Deliveries, &summary.Bytes, &summary.Retries, &summary.Transformations); err != nil {
			return nil, err
		}

		result[endpointID] = summary
	}

	return result, nil
}

// GetDailyCosts retrieves daily cost data
func (r *PostgresRepository) GetDailyCosts(ctx context.Context, tenantID string, startDate, endDate time.Time) ([]DailyCost, error) {
	query := `
		SELECT 
			DATE(recorded_at) as date,
			COALESCE(SUM(CASE WHEN unit = 'delivery' THEN quantity ELSE 0 END), 0) as deliveries,
			COALESCE(SUM(CASE WHEN unit = 'byte' THEN quantity ELSE 0 END), 0) as bytes,
			COALESCE(SUM(CASE WHEN unit = 'retry' THEN quantity ELSE 0 END), 0) as retries,
			COALESCE(SUM(CASE WHEN unit = 'transform' THEN quantity ELSE 0 END), 0) as transformations
		FROM usage_records
		WHERE tenant_id = $1 AND recorded_at >= $2 AND recorded_at < $3
		GROUP BY DATE(recorded_at)
		ORDER BY date ASC
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dailyCosts []DailyCost
	rates := DefaultRates()

	for rows.Next() {
		var date string
		var summary UsageSummary

		if err := rows.Scan(&date, &summary.Deliveries, &summary.Bytes, &summary.Retries, &summary.Transformations); err != nil {
			return nil, err
		}

		cost := 0.0
		for _, rate := range rates {
			switch rate.Unit {
			case UnitDelivery:
				cost += float64(summary.Deliveries) * rate.Price
			case UnitByte:
				cost += float64(summary.Bytes) * rate.Price
			case UnitRetry:
				cost += float64(summary.Retries) * rate.Price
			case UnitTransform:
				cost += float64(summary.Transformations) * rate.Price
			}
		}

		dailyCosts = append(dailyCosts, DailyCost{
			Date:  date,
			Cost:  cost,
			Usage: summary,
		})
	}

	return dailyCosts, nil
}

// GetCostAllocation retrieves a cost allocation
func (r *PostgresRepository) GetCostAllocation(ctx context.Context, tenantID, period, resourceType, resourceID string) (*CostAllocation, error) {
	query := `
		SELECT id, tenant_id, period, resource_type, resource_id, resource_name, usage, cost, created_at, updated_at
		FROM cost_allocations
		WHERE tenant_id = $1 AND period = $2 AND resource_type = $3 AND resource_id = $4
	`

	var allocation CostAllocation
	var usageJSON, costJSON []byte

	err := r.db.QueryRowContext(ctx, query, tenantID, period, resourceType, resourceID).Scan(
		&allocation.ID, &allocation.TenantID, &allocation.Period, &allocation.ResourceType,
		&allocation.ResourceID, &allocation.ResourceName, &usageJSON, &costJSON,
		&allocation.CreatedAt, &allocation.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(usageJSON, &allocation.Usage)
	json.Unmarshal(costJSON, &allocation.Cost)

	return &allocation, nil
}

// GetCostAllocationsByResource retrieves cost allocations by resource type
func (r *PostgresRepository) GetCostAllocationsByResource(ctx context.Context, tenantID, period, resourceType string) ([]CostAllocation, error) {
	query := `
		SELECT id, tenant_id, period, resource_type, resource_id, resource_name, usage, cost, created_at, updated_at
		FROM cost_allocations
		WHERE tenant_id = $1 AND period = $2 AND resource_type = $3
		ORDER BY (cost->>'total')::NUMERIC DESC
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, period, resourceType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allocations []CostAllocation
	for rows.Next() {
		var allocation CostAllocation
		var usageJSON, costJSON []byte

		if err := rows.Scan(
			&allocation.ID, &allocation.TenantID, &allocation.Period, &allocation.ResourceType,
			&allocation.ResourceID, &allocation.ResourceName, &usageJSON, &costJSON,
			&allocation.CreatedAt, &allocation.UpdatedAt,
		); err != nil {
			return nil, err
		}

		json.Unmarshal(usageJSON, &allocation.Usage)
		json.Unmarshal(costJSON, &allocation.Cost)

		allocations = append(allocations, allocation)
	}

	return allocations, nil
}

// SaveCostAllocation saves a cost allocation
func (r *PostgresRepository) SaveCostAllocation(ctx context.Context, allocation *CostAllocation) error {
	if allocation.ID == "" {
		allocation.ID = uuid.New().String()
	}

	usageJSON, _ := json.Marshal(allocation.Usage)
	costJSON, _ := json.Marshal(allocation.Cost)

	query := `
		INSERT INTO cost_allocations (id, tenant_id, period, resource_type, resource_id, resource_name, usage, cost, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (tenant_id, period, resource_type, resource_id) DO UPDATE SET
			resource_name = EXCLUDED.resource_name,
			usage = EXCLUDED.usage,
			cost = EXCLUDED.cost,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.db.ExecContext(ctx, query,
		allocation.ID, allocation.TenantID, allocation.Period, allocation.ResourceType,
		allocation.ResourceID, allocation.ResourceName, usageJSON, costJSON,
		allocation.CreatedAt, allocation.UpdatedAt,
	)

	return err
}

// CreateBudget creates a new budget
func (r *PostgresRepository) CreateBudget(ctx context.Context, budget *Budget) error {
	if budget.ID == "" {
		budget.ID = uuid.New().String()
	}

	alertsJSON, _ := json.Marshal(budget.Alerts)

	query := `
		INSERT INTO budgets (id, tenant_id, name, amount, currency, period, resource_type, resource_id, alerts, current_spend, is_active, start_date, end_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	_, err := r.db.ExecContext(ctx, query,
		budget.ID, budget.TenantID, budget.Name, budget.Amount, budget.Currency,
		budget.Period, budget.ResourceType, budget.ResourceID, alertsJSON,
		budget.CurrentSpend, budget.IsActive, budget.StartDate, budget.EndDate,
		budget.CreatedAt, budget.UpdatedAt,
	)

	return err
}

// GetBudget retrieves a budget by ID
func (r *PostgresRepository) GetBudget(ctx context.Context, tenantID, budgetID string) (*Budget, error) {
	query := `
		SELECT id, tenant_id, name, amount, currency, period, resource_type, resource_id, alerts, current_spend, is_active, start_date, end_date, created_at, updated_at
		FROM budgets
		WHERE id = $1 AND tenant_id = $2
	`

	var budget Budget
	var alertsJSON []byte
	var endDate sql.NullTime

	err := r.db.QueryRowContext(ctx, query, budgetID, tenantID).Scan(
		&budget.ID, &budget.TenantID, &budget.Name, &budget.Amount, &budget.Currency,
		&budget.Period, &budget.ResourceType, &budget.ResourceID, &alertsJSON,
		&budget.CurrentSpend, &budget.IsActive, &budget.StartDate, &endDate,
		&budget.CreatedAt, &budget.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(alertsJSON, &budget.Alerts)
	if endDate.Valid {
		budget.EndDate = &endDate.Time
	}

	return &budget, nil
}

// ListBudgets lists budgets for a tenant
func (r *PostgresRepository) ListBudgets(ctx context.Context, tenantID string, limit, offset int) ([]Budget, int, error) {
	countQuery := `SELECT COUNT(*) FROM budgets WHERE tenant_id = $1`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, tenantID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, tenant_id, name, amount, currency, period, resource_type, resource_id, alerts, current_spend, is_active, start_date, end_date, created_at, updated_at
		FROM budgets
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var budgets []Budget
	for rows.Next() {
		var budget Budget
		var alertsJSON []byte
		var endDate sql.NullTime

		if err := rows.Scan(
			&budget.ID, &budget.TenantID, &budget.Name, &budget.Amount, &budget.Currency,
			&budget.Period, &budget.ResourceType, &budget.ResourceID, &alertsJSON,
			&budget.CurrentSpend, &budget.IsActive, &budget.StartDate, &endDate,
			&budget.CreatedAt, &budget.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}

		json.Unmarshal(alertsJSON, &budget.Alerts)
		if endDate.Valid {
			budget.EndDate = &endDate.Time
		}

		budgets = append(budgets, budget)
	}

	return budgets, total, nil
}

// UpdateBudget updates a budget
func (r *PostgresRepository) UpdateBudget(ctx context.Context, budget *Budget) error {
	alertsJSON, _ := json.Marshal(budget.Alerts)

	query := `
		UPDATE budgets
		SET name = $1, amount = $2, alerts = $3, current_spend = $4, is_active = $5, end_date = $6, updated_at = $7
		WHERE id = $8 AND tenant_id = $9
	`

	_, err := r.db.ExecContext(ctx, query,
		budget.Name, budget.Amount, alertsJSON, budget.CurrentSpend,
		budget.IsActive, budget.EndDate, budget.UpdatedAt,
		budget.ID, budget.TenantID,
	)

	return err
}

// DeleteBudget deletes a budget
func (r *PostgresRepository) DeleteBudget(ctx context.Context, tenantID, budgetID string) error {
	query := `DELETE FROM budgets WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.ExecContext(ctx, query, budgetID, tenantID)
	return err
}
