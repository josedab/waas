package costengine

import (
	"context"
	"time"
)

// Repository defines the data access interface for cost attribution
type Repository interface {
	// Cost models
	CreateCostModel(ctx context.Context, model *CostModel) error
	GetCostModel(ctx context.Context, tenantID, modelID string) (*CostModel, error)
	ListCostModels(ctx context.Context, tenantID string) ([]CostModel, error)
	UpdateCostModel(ctx context.Context, model *CostModel) error
	GetActiveCostModel(ctx context.Context, tenantID string) (*CostModel, error)

	// Delivery costs
	RecordDeliveryCost(ctx context.Context, cost *DeliveryCost) error
	GetDeliveryCostsByPeriod(ctx context.Context, tenantID string, start, end time.Time) ([]DeliveryCost, error)

	// Reports
	GetTotalCostByPeriod(ctx context.Context, tenantID string, start, end time.Time) (float64, int64, error)
	GetCostByEndpoint(ctx context.Context, tenantID string, start, end time.Time) (map[string]float64, error)
	GetCostByEventType(ctx context.Context, tenantID string, start, end time.Time) (map[string]float64, error)
	GetCostByDay(ctx context.Context, tenantID string, start, end time.Time) ([]DailyCost, error)
	GetTopCostEndpoints(ctx context.Context, tenantID string, start, end time.Time, limit int) ([]EndpointCost, error)

	// Budgets
	CreateBudget(ctx context.Context, budget *CostBudget) error
	GetBudget(ctx context.Context, tenantID, budgetID string) (*CostBudget, error)
	ListBudgets(ctx context.Context, tenantID string) ([]CostBudget, error)
	UpdateBudget(ctx context.Context, budget *CostBudget) error

	// Anomalies
	CreateAnomaly(ctx context.Context, anomaly *CostAnomaly) error
	ListAnomalies(ctx context.Context, tenantID string) ([]CostAnomaly, error)
	GetHistoricalAvgCost(ctx context.Context, tenantID string, days int) (float64, error)

	// Spend
	GetCurrentSpend(ctx context.Context, tenantID string, periodStart time.Time) (float64, error)
}
