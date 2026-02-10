package costengine

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CostAttributionLevel defines the granularity of cost attribution
type CostAttributionLevel string

const (
	AttributionLevelTenant   CostAttributionLevel = "tenant"
	AttributionLevelEndpoint CostAttributionLevel = "endpoint"
	AttributionLevelEvent    CostAttributionLevel = "event_type"
	AttributionLevelRegion   CostAttributionLevel = "region"
)

// BudgetStatus represents the status of a budget
type BudgetStatus string

const (
	BudgetStatusOK       BudgetStatus = "ok"
	BudgetStatusWarning  BudgetStatus = "warning"
	BudgetStatusCritical BudgetStatus = "critical"
	BudgetStatusExceeded BudgetStatus = "exceeded"
)

// CostAttribution provides per-tenant, per-endpoint cost breakdown
type CostAttribution struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	EndpointID    string    `json:"endpoint_id,omitempty"`
	EventType     string    `json:"event_type,omitempty"`
	Region        string    `json:"region,omitempty"`
	Period        string    `json:"period"` // hourly, daily, monthly
	ComputeCost   float64   `json:"compute_cost"`
	BandwidthCost float64   `json:"bandwidth_cost"`
	StorageCost   float64   `json:"storage_cost"`
	RetryCost     float64   `json:"retry_cost"`
	TotalCost     float64   `json:"total_cost"`
	DeliveryCount int64     `json:"delivery_count"`
	BytesOut      int64     `json:"bytes_out"`
	RetryCount    int64     `json:"retry_count"`
	Timestamp     time.Time `json:"timestamp"`
}

// Budget defines spending limits for a tenant
type Budget struct {
	ID               string       `json:"id"`
	TenantID         string       `json:"tenant_id"`
	Name             string       `json:"name"`
	MonthlyLimit     float64      `json:"monthly_limit"`
	DailyLimit       float64      `json:"daily_limit,omitempty"`
	WarningThreshold float64      `json:"warning_threshold"` // 0-1 (e.g., 0.8 = 80%)
	AlertChannels    []string     `json:"alert_channels"`    // email, webhook, slack
	AutoThrottle     bool         `json:"auto_throttle"`
	ThrottlePercent  int          `json:"throttle_percent,omitempty"`
	Status           BudgetStatus `json:"status"`
	CurrentSpend     float64      `json:"current_spend"`
	ForecastedSpend  float64      `json:"forecasted_spend"`
	Enabled          bool         `json:"enabled"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
}

// BudgetAlert represents a budget threshold alert
type BudgetAlert struct {
	ID         string    `json:"id"`
	BudgetID   string    `json:"budget_id"`
	TenantID   string    `json:"tenant_id"`
	Type       string    `json:"type"` // warning, critical, exceeded
	Message    string    `json:"message"`
	Threshold  float64   `json:"threshold"`
	CurrentVal float64   `json:"current_value"`
	Notified   bool      `json:"notified"`
	CreatedAt  time.Time `json:"created_at"`
}

// CostForecast provides cost predictions
type CostForecast struct {
	TenantID         string    `json:"tenant_id"`
	CurrentMonthCost float64   `json:"current_month_cost"`
	ProjectedCost    float64   `json:"projected_cost"`
	DailyAverage     float64   `json:"daily_average"`
	TrendDirection   string    `json:"trend_direction"` // increasing, decreasing, stable
	TrendPercent     float64   `json:"trend_percent"`
	DaysRemaining    int       `json:"days_remaining"`
	GeneratedAt      time.Time `json:"generated_at"`
}

// FinOpsDashboard provides cost visibility
type FinOpsDashboard struct {
	TenantID          string            `json:"tenant_id"`
	CurrentMonthCost  float64           `json:"current_month_cost"`
	PreviousMonthCost float64           `json:"previous_month_cost"`
	CostChange        float64           `json:"cost_change_percent"`
	Forecast          *CostForecast     `json:"forecast"`
	CostByEndpoint    []CostAttribution `json:"cost_by_endpoint"`
	CostByEventType   []CostAttribution `json:"cost_by_event_type"`
	CostByRegion      []CostAttribution `json:"cost_by_region"`
	DailyCosts        []CostAttribution `json:"daily_costs"`
	Budget            *Budget           `json:"budget,omitempty"`
	RecentAlerts      []BudgetAlert     `json:"recent_alerts"`
	GeneratedAt       time.Time         `json:"generated_at"`
}

// CreateFinOpsBudgetRequest represents a request to create a FinOps budget
type CreateFinOpsBudgetRequest struct {
	Name             string   `json:"name" binding:"required"`
	MonthlyLimit     float64  `json:"monthly_limit" binding:"required,gt=0"`
	DailyLimit       float64  `json:"daily_limit,omitempty"`
	WarningThreshold float64  `json:"warning_threshold,omitempty"`
	AlertChannels    []string `json:"alert_channels,omitempty"`
	AutoThrottle     bool     `json:"auto_throttle"`
	ThrottlePercent  int      `json:"throttle_percent,omitempty"`
}

// FinOpsRepository defines storage for cost attribution
type FinOpsRepository interface {
	SaveAttribution(ctx context.Context, attr *CostAttribution) error
	GetAttributions(ctx context.Context, tenantID string, level CostAttributionLevel, start, end time.Time) ([]CostAttribution, error)
	GetTotalCost(ctx context.Context, tenantID string, start, end time.Time) (float64, error)
	GetDailyCosts(ctx context.Context, tenantID string, days int) ([]CostAttribution, error)
	CreateBudget(ctx context.Context, budget *Budget) error
	GetBudget(ctx context.Context, tenantID string) (*Budget, error)
	UpdateBudget(ctx context.Context, budget *Budget) error
	DeleteBudget(ctx context.Context, tenantID string) error
	CreateBudgetAlert(ctx context.Context, alert *BudgetAlert) error
	ListBudgetAlerts(ctx context.Context, tenantID string, limit int) ([]BudgetAlert, error)
}

// FinOpsService provides cost attribution and budget management
type FinOpsService struct {
	repo      FinOpsRepository
	costModel *CostModel
}

// NewFinOpsService creates a new FinOps service
func NewFinOpsService(repo FinOpsRepository, costModel *CostModel) *FinOpsService {
	if costModel == nil {
		costModel = &CostModel{
			ComputeCostPerDelivery: 0.0001,
			BandwidthCostPerKB:     0.00001,
			RetryCostMultiplier:    1.5,
			StorageCostPerGBDay:    0.023,
			Currency:               "USD",
		}
	}
	return &FinOpsService{
		repo:      repo,
		costModel: costModel,
	}
}

// RecordDeliveryCost calculates and records cost for a delivery
func (s *FinOpsService) RecordDeliveryCost(ctx context.Context, tenantID, endpointID, eventType, region string, bytesOut int64, retryCount int) (*CostAttribution, error) {
	computeCost := s.costModel.ComputeCostPerDelivery
	bandwidthCost := float64(bytesOut) / 1024 * s.costModel.BandwidthCostPerKB
	retryCost := float64(retryCount) * s.costModel.RetryCostMultiplier * computeCost
	storageCost := float64(bytesOut) / (1024 * 1024 * 1024) * s.costModel.StorageCostPerGBDay

	now := time.Now()
	attr := &CostAttribution{
		ID:            uuid.New().String(),
		TenantID:      tenantID,
		EndpointID:    endpointID,
		EventType:     eventType,
		Region:        region,
		Period:        "hourly",
		ComputeCost:   computeCost,
		BandwidthCost: bandwidthCost,
		StorageCost:   storageCost,
		RetryCost:     retryCost,
		TotalCost:     computeCost + bandwidthCost + storageCost + retryCost,
		DeliveryCount: 1,
		BytesOut:      bytesOut,
		RetryCount:    int64(retryCount),
		Timestamp:     now,
	}

	if err := s.repo.SaveAttribution(ctx, attr); err != nil {
		return nil, fmt.Errorf("failed to record cost: %w", err)
	}

	// Check budget
	go s.checkBudget(context.Background(), tenantID)

	return attr, nil
}

// checkBudget evaluates budget thresholds and creates alerts
func (s *FinOpsService) checkBudget(ctx context.Context, tenantID string) {
	budget, err := s.repo.GetBudget(ctx, tenantID)
	if err != nil || !budget.Enabled {
		return
	}

	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	currentSpend, err := s.repo.GetTotalCost(ctx, tenantID, startOfMonth, now)
	if err != nil {
		return
	}

	budget.CurrentSpend = currentSpend

	ratio := currentSpend / budget.MonthlyLimit
	switch {
	case ratio >= 1.0:
		budget.Status = BudgetStatusExceeded
		_ = s.repo.CreateBudgetAlert(ctx, &BudgetAlert{
			ID:         uuid.New().String(),
			BudgetID:   budget.ID,
			TenantID:   tenantID,
			Type:       "exceeded",
			Message:    fmt.Sprintf("Monthly budget exceeded: $%.2f / $%.2f", currentSpend, budget.MonthlyLimit),
			Threshold:  budget.MonthlyLimit,
			CurrentVal: currentSpend,
			CreatedAt:  now,
		})
	case ratio >= 0.9:
		budget.Status = BudgetStatusCritical
	case ratio >= budget.WarningThreshold:
		budget.Status = BudgetStatusWarning
	default:
		budget.Status = BudgetStatusOK
	}

	_ = s.repo.UpdateBudget(ctx, budget)
}

// GetDashboard generates the FinOps dashboard
func (s *FinOpsService) GetDashboard(ctx context.Context, tenantID string) (*FinOpsDashboard, error) {
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	prevStart := startOfMonth.AddDate(0, -1, 0)

	currentCost, _ := s.repo.GetTotalCost(ctx, tenantID, startOfMonth, now)
	prevCost, _ := s.repo.GetTotalCost(ctx, tenantID, prevStart, startOfMonth)

	costChange := 0.0
	if prevCost > 0 {
		costChange = ((currentCost - prevCost) / prevCost) * 100
	}

	byEndpoint, _ := s.repo.GetAttributions(ctx, tenantID, AttributionLevelEndpoint, startOfMonth, now)
	if byEndpoint == nil {
		byEndpoint = []CostAttribution{}
	}

	byEventType, _ := s.repo.GetAttributions(ctx, tenantID, AttributionLevelEvent, startOfMonth, now)
	if byEventType == nil {
		byEventType = []CostAttribution{}
	}

	byRegion, _ := s.repo.GetAttributions(ctx, tenantID, AttributionLevelRegion, startOfMonth, now)
	if byRegion == nil {
		byRegion = []CostAttribution{}
	}

	dailyCosts, _ := s.repo.GetDailyCosts(ctx, tenantID, 30)
	if dailyCosts == nil {
		dailyCosts = []CostAttribution{}
	}

	budget, _ := s.repo.GetBudget(ctx, tenantID)
	alerts, _ := s.repo.ListBudgetAlerts(ctx, tenantID, 10)
	if alerts == nil {
		alerts = []BudgetAlert{}
	}

	// Generate forecast
	daysElapsed := now.Day()
	daysInMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()).Day()
	dailyAvg := 0.0
	if daysElapsed > 0 {
		dailyAvg = currentCost / float64(daysElapsed)
	}
	projected := dailyAvg * float64(daysInMonth)

	trendDir := "stable"
	if costChange > 5 {
		trendDir = "increasing"
	} else if costChange < -5 {
		trendDir = "decreasing"
	}

	forecast := &CostForecast{
		TenantID:         tenantID,
		CurrentMonthCost: currentCost,
		ProjectedCost:    projected,
		DailyAverage:     dailyAvg,
		TrendDirection:   trendDir,
		TrendPercent:     costChange,
		DaysRemaining:    daysInMonth - daysElapsed,
		GeneratedAt:      now,
	}

	return &FinOpsDashboard{
		TenantID:          tenantID,
		CurrentMonthCost:  currentCost,
		PreviousMonthCost: prevCost,
		CostChange:        costChange,
		Forecast:          forecast,
		CostByEndpoint:    byEndpoint,
		CostByEventType:   byEventType,
		CostByRegion:      byRegion,
		DailyCosts:        dailyCosts,
		Budget:            budget,
		RecentAlerts:      alerts,
		GeneratedAt:       now,
	}, nil
}

// CreateBudget creates a budget for a tenant
func (s *FinOpsService) CreateBudget(ctx context.Context, tenantID string, req *CreateFinOpsBudgetRequest) (*Budget, error) {
	warnThreshold := req.WarningThreshold
	if warnThreshold == 0 {
		warnThreshold = 0.8
	}
	throttlePercent := req.ThrottlePercent
	if throttlePercent == 0 {
		throttlePercent = 50
	}

	now := time.Now()
	budget := &Budget{
		ID:               uuid.New().String(),
		TenantID:         tenantID,
		Name:             req.Name,
		MonthlyLimit:     req.MonthlyLimit,
		DailyLimit:       req.DailyLimit,
		WarningThreshold: warnThreshold,
		AlertChannels:    req.AlertChannels,
		AutoThrottle:     req.AutoThrottle,
		ThrottlePercent:  throttlePercent,
		Status:           BudgetStatusOK,
		Enabled:          true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.repo.CreateBudget(ctx, budget); err != nil {
		return nil, fmt.Errorf("failed to create budget: %w", err)
	}

	return budget, nil
}

// GetBudget retrieves a tenant's budget
func (s *FinOpsService) GetBudget(ctx context.Context, tenantID string) (*Budget, error) {
	return s.repo.GetBudget(ctx, tenantID)
}

// DeleteBudget deletes a tenant's budget
func (s *FinOpsService) DeleteBudget(ctx context.Context, tenantID string) error {
	return s.repo.DeleteBudget(ctx, tenantID)
}
