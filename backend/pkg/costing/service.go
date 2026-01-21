package costing

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
)

// Service manages cost tracking and attribution
type Service struct {
	repo Repository
}

// NewService creates a new costing service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// RecordUsage records a usage event
func (s *Service) RecordUsage(ctx context.Context, tenantID, endpointID, webhookID string, unit CostUnit, quantity int64, meta UsageMeta) error {
	record := &UsageRecord{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		EndpointID: endpointID,
		WebhookID:  webhookID,
		Unit:       unit,
		Quantity:   quantity,
		Metadata:   meta,
		RecordedAt: time.Now(),
	}

	return s.repo.CreateUsageRecord(ctx, record)
}

// RecordDelivery is a convenience method for recording delivery usage
func (s *Service) RecordDelivery(ctx context.Context, tenantID, endpointID, webhookID string, payloadBytes int64, retryCount int, region string, statusCode int, latencyMs int) error {
	// Record delivery
	if err := s.RecordUsage(ctx, tenantID, endpointID, webhookID, UnitDelivery, 1, UsageMeta{
		PayloadBytes: payloadBytes,
		RetryCount:   retryCount,
		Region:       region,
		StatusCode:   statusCode,
		LatencyMs:    latencyMs,
	}); err != nil {
		return err
	}

	// Record bytes
	if payloadBytes > 0 {
		if err := s.RecordUsage(ctx, tenantID, endpointID, webhookID, UnitByte, payloadBytes, UsageMeta{}); err != nil {
			return err
		}
	}

	// Record retries
	if retryCount > 0 {
		if err := s.RecordUsage(ctx, tenantID, endpointID, webhookID, UnitRetry, int64(retryCount), UsageMeta{}); err != nil {
			return err
		}
	}

	return nil
}

// GetCostReport generates a cost report for a tenant
func (s *Service) GetCostReport(ctx context.Context, tenantID, period string) (*CostReport, error) {
	// Parse period (e.g., "2024-01")
	startDate, endDate, err := parsePeriod(period)
	if err != nil {
		return nil, err
	}

	// Get usage summary
	usage, err := s.repo.GetUsageSummary(ctx, tenantID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Calculate costs
	rates := DefaultRates()
	summary := calculateCosts(usage, rates)

	// Get by-endpoint breakdown
	byEndpoint, err := s.repo.GetCostAllocationsByResource(ctx, tenantID, period, "endpoint")
	if err != nil {
		return nil, err
	}

	// Get daily trend
	dailyTrend, err := s.repo.GetDailyCosts(ctx, tenantID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Generate forecast
	forecast := s.generateForecast(ctx, tenantID, summary.Total)

	return &CostReport{
		TenantID:    tenantID,
		Period:      period,
		StartDate:   startDate,
		EndDate:     endDate,
		Summary:     summary,
		ByEndpoint:  byEndpoint,
		DailyTrend:  dailyTrend,
		Forecast:    forecast,
		GeneratedAt: time.Now(),
	}, nil
}

// GetCostAllocation gets cost allocation for a specific resource
func (s *Service) GetCostAllocation(ctx context.Context, tenantID, period, resourceType, resourceID string) (*CostAllocation, error) {
	return s.repo.GetCostAllocation(ctx, tenantID, period, resourceType, resourceID)
}

// CalculateCostAllocation calculates and stores cost allocations for a period
func (s *Service) CalculateCostAllocation(ctx context.Context, tenantID, period string) error {
	startDate, endDate, err := parsePeriod(period)
	if err != nil {
		return err
	}

	// Get usage by endpoint
	endpointUsage, err := s.repo.GetUsageByEndpoint(ctx, tenantID, startDate, endDate)
	if err != nil {
		return err
	}

	rates := DefaultRates()

	// Calculate and store allocations
	for endpointID, usage := range endpointUsage {
		cost := calculateCosts(&usage, rates)

		allocation := &CostAllocation{
			ID:           uuid.New().String(),
			TenantID:     tenantID,
			Period:       period,
			ResourceType: "endpoint",
			ResourceID:   endpointID,
			Usage:        usage,
			Cost:         cost,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if err := s.repo.SaveCostAllocation(ctx, allocation); err != nil {
			return err
		}
	}

	return nil
}

// Budget operations

// CreateBudget creates a new budget
func (s *Service) CreateBudget(ctx context.Context, tenantID string, req *CreateBudgetRequest) (*Budget, error) {
	budget := &Budget{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		Name:         req.Name,
		Amount:       req.Amount,
		Currency:     req.Currency,
		Period:       req.Period,
		ResourceType: req.ResourceType,
		ResourceID:   req.ResourceID,
		Alerts:       req.Alerts,
		CurrentSpend: 0,
		IsActive:     true,
		StartDate:    req.StartDate,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if budget.StartDate.IsZero() {
		budget.StartDate = time.Now()
	}

	if err := s.repo.CreateBudget(ctx, budget); err != nil {
		return nil, err
	}

	return budget, nil
}

// GetBudget retrieves a budget by ID
func (s *Service) GetBudget(ctx context.Context, tenantID, budgetID string) (*Budget, error) {
	return s.repo.GetBudget(ctx, tenantID, budgetID)
}

// ListBudgets lists budgets for a tenant
func (s *Service) ListBudgets(ctx context.Context, tenantID string, limit, offset int) ([]Budget, int, error) {
	return s.repo.ListBudgets(ctx, tenantID, limit, offset)
}

// UpdateBudget updates a budget
func (s *Service) UpdateBudget(ctx context.Context, tenantID, budgetID string, req *UpdateBudgetRequest) (*Budget, error) {
	budget, err := s.repo.GetBudget(ctx, tenantID, budgetID)
	if err != nil {
		return nil, err
	}
	if budget == nil {
		return nil, fmt.Errorf("budget not found")
	}

	if req.Name != nil {
		budget.Name = *req.Name
	}
	if req.Amount != nil {
		budget.Amount = *req.Amount
	}
	if len(req.Alerts) > 0 {
		budget.Alerts = req.Alerts
	}
	if req.IsActive != nil {
		budget.IsActive = *req.IsActive
	}
	if req.EndDate != nil {
		budget.EndDate = req.EndDate
	}

	budget.UpdatedAt = time.Now()

	if err := s.repo.UpdateBudget(ctx, budget); err != nil {
		return nil, err
	}

	return budget, nil
}

// DeleteBudget deletes a budget
func (s *Service) DeleteBudget(ctx context.Context, tenantID, budgetID string) error {
	return s.repo.DeleteBudget(ctx, tenantID, budgetID)
}

// CheckBudgetAlerts checks and sends budget alerts
func (s *Service) CheckBudgetAlerts(ctx context.Context, tenantID string) ([]Budget, error) {
	budgets, _, err := s.repo.ListBudgets(ctx, tenantID, 100, 0)
	if err != nil {
		return nil, err
	}

	var alertedBudgets []Budget

	for _, budget := range budgets {
		if !budget.IsActive {
			continue
		}

		// Calculate current spend
		spend, err := s.getCurrentSpend(ctx, &budget)
		if err != nil {
			continue
		}

		budget.CurrentSpend = spend
		s.repo.UpdateBudget(ctx, &budget)

		// Check alerts
		percentage := spend / budget.Amount
		for _, alert := range budget.Alerts {
			if percentage >= alert.Threshold {
				alertedBudgets = append(alertedBudgets, budget)
				break
			}
		}
	}

	return alertedBudgets, nil
}

// GetForecast generates a cost forecast
func (s *Service) GetForecast(ctx context.Context, tenantID string) (*CostForecast, error) {
	// Get current month's costs
	now := time.Now()
	period := now.Format("2006-01")

	report, err := s.GetCostReport(ctx, tenantID, period)
	if err != nil {
		return nil, err
	}

	return s.generateForecast(ctx, tenantID, report.Summary.Total), nil
}

// GetUsageStats returns usage statistics
func (s *Service) GetUsageStats(ctx context.Context, tenantID string, startDate, endDate time.Time) (*UsageSummary, error) {
	return s.repo.GetUsageSummary(ctx, tenantID, startDate, endDate)
}

// GetTopEndpoints returns top endpoints by cost
func (s *Service) GetTopEndpoints(ctx context.Context, tenantID, period string, limit int) ([]CostAllocation, error) {
	allocations, err := s.repo.GetCostAllocationsByResource(ctx, tenantID, period, "endpoint")
	if err != nil {
		return nil, err
	}

	// Sort by cost (assuming already sorted) and limit
	if len(allocations) > limit {
		allocations = allocations[:limit]
	}

	return allocations, nil
}

// Helper functions

func (s *Service) getCurrentSpend(ctx context.Context, budget *Budget) (float64, error) {
	startDate := budget.StartDate
	endDate := time.Now()

	usage, err := s.repo.GetUsageSummary(ctx, budget.TenantID, startDate, endDate)
	if err != nil {
		return 0, err
	}

	rates := DefaultRates()
	cost := calculateCosts(usage, rates)

	return cost.Total, nil
}

func (s *Service) generateForecast(ctx context.Context, tenantID string, currentCost float64) *CostForecast {
	now := time.Now()
	daysInMonth := float64(daysInMonth(now))
	currentDay := float64(now.Day())

	// Simple linear projection
	projectedCost := (currentCost / currentDay) * daysInMonth

	// Get previous month for comparison
	prevMonth := now.AddDate(0, -1, 0)
	prevPeriod := prevMonth.Format("2006-01")
	prevCost := 0.0

	if prevReport, err := s.repo.GetCostAllocation(ctx, tenantID, prevPeriod, "tenant", tenantID); err == nil && prevReport != nil {
		prevCost = prevReport.Cost.Total
	}

	percentChange := 0.0
	if prevCost > 0 {
		percentChange = ((projectedCost - prevCost) / prevCost) * 100
	}

	trend := "stable"
	if percentChange > 10 {
		trend = "increasing"
	} else if percentChange < -10 {
		trend = "decreasing"
	}

	return &CostForecast{
		TenantID:       tenantID,
		Period:         now.Format("2006-01"),
		ProjectedCost:  math.Round(projectedCost*100) / 100,
		Confidence:     0.7, // Lower confidence for simple projection
		Trend:          trend,
		PreviousPeriod: prevCost,
		PercentChange:  math.Round(percentChange*100) / 100,
		GeneratedAt:    time.Now(),
	}
}

func calculateCosts(usage *UsageSummary, rates []Rate) CostBreakdown {
	breakdown := CostBreakdown{Currency: "USD"}

	for _, rate := range rates {
		switch rate.Unit {
		case UnitDelivery:
			breakdown.Delivery = float64(usage.Deliveries) * rate.Price
		case UnitByte:
			breakdown.Bandwidth = float64(usage.Bytes) * rate.Price
		case UnitRetry:
			breakdown.Retries = float64(usage.Retries) * rate.Price
		case UnitTransform:
			breakdown.Transformations = float64(usage.Transformations) * rate.Price
		}
	}

	breakdown.Total = breakdown.Delivery + breakdown.Bandwidth + breakdown.Retries + breakdown.Transformations
	return breakdown
}

func parsePeriod(period string) (time.Time, time.Time, error) {
	t, err := time.Parse("2006-01", period)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid period format: %s", period)
	}

	startDate := t
	endDate := t.AddDate(0, 1, 0).Add(-time.Nanosecond)

	return startDate, endDate, nil
}

func daysInMonth(t time.Time) int {
	return t.AddDate(0, 1, -t.Day()).Day()
}
