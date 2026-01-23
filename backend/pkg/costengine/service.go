package costengine

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
)

// Service provides cost attribution functionality
type Service struct {
	repo Repository
}

// NewService creates a new cost engine service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateCostModel creates a new cost model
func (s *Service) CreateCostModel(ctx context.Context, tenantID string, req *CreateCostModelRequest) (*CostModel, error) {
	model := &CostModel{
		ID:                     uuid.New().String(),
		TenantID:               tenantID,
		Name:                   req.Name,
		ComputeCostPerDelivery: req.ComputeCostPerDelivery,
		BandwidthCostPerKB:     req.BandwidthCostPerKB,
		RetryCostMultiplier:    req.RetryCostMultiplier,
		StorageCostPerGBDay:    req.StorageCostPerGBDay,
		Currency:               req.Currency,
		IsActive:               true,
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
	}

	if err := s.repo.CreateCostModel(ctx, model); err != nil {
		return nil, fmt.Errorf("failed to create cost model: %w", err)
	}

	return model, nil
}

// GetCostModel retrieves a cost model by ID
func (s *Service) GetCostModel(ctx context.Context, tenantID, modelID string) (*CostModel, error) {
	return s.repo.GetCostModel(ctx, tenantID, modelID)
}

// ListCostModels retrieves all cost models for a tenant
func (s *Service) ListCostModels(ctx context.Context, tenantID string) ([]CostModel, error) {
	return s.repo.ListCostModels(ctx, tenantID)
}

// UpdateCostModel updates an existing cost model
func (s *Service) UpdateCostModel(ctx context.Context, tenantID, modelID string, req *CreateCostModelRequest) (*CostModel, error) {
	model, err := s.repo.GetCostModel(ctx, tenantID, modelID)
	if err != nil {
		return nil, err
	}

	model.Name = req.Name
	model.ComputeCostPerDelivery = req.ComputeCostPerDelivery
	model.BandwidthCostPerKB = req.BandwidthCostPerKB
	model.RetryCostMultiplier = req.RetryCostMultiplier
	model.StorageCostPerGBDay = req.StorageCostPerGBDay
	model.Currency = req.Currency
	model.UpdatedAt = time.Now()

	if err := s.repo.UpdateCostModel(ctx, model); err != nil {
		return nil, fmt.Errorf("failed to update cost model: %w", err)
	}

	return model, nil
}

// RecordDeliveryCost calculates and records the cost of a delivery using the active cost model
func (s *Service) RecordDeliveryCost(ctx context.Context, tenantID string, req *RecordDeliveryCostRequest) (*DeliveryCost, error) {
	model, err := s.repo.GetActiveCostModel(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active cost model: %w", err)
	}

	computeCost := model.ComputeCostPerDelivery
	bandwidthCost := model.BandwidthCostPerKB * (float64(req.PayloadSizeBytes) / 1024.0)
	retryCost := 0.0
	if req.RetryCount > 0 {
		retryCost = computeCost * model.RetryCostMultiplier * float64(req.RetryCount)
	}
	totalCost := computeCost + bandwidthCost + retryCost

	cost := &DeliveryCost{
		ID:               uuid.New().String(),
		TenantID:         tenantID,
		DeliveryID:       req.DeliveryID,
		EndpointID:       req.EndpointID,
		EventType:        req.EventType,
		ComputeCost:      math.Round(computeCost*1e6) / 1e6,
		BandwidthCost:    math.Round(bandwidthCost*1e6) / 1e6,
		RetryCost:        math.Round(retryCost*1e6) / 1e6,
		TotalCost:        math.Round(totalCost*1e6) / 1e6,
		PayloadSizeBytes: req.PayloadSizeBytes,
		RetryCount:       req.RetryCount,
		CreatedAt:        time.Now(),
	}

	if err := s.repo.RecordDeliveryCost(ctx, cost); err != nil {
		return nil, fmt.Errorf("failed to record delivery cost: %w", err)
	}

	return cost, nil
}

// GenerateReport generates a cost report for a given period
func (s *Service) GenerateReport(ctx context.Context, tenantID string, req *GenerateReportRequest) (*CostReport, error) {
	totalCost, deliveryCount, err := s.repo.GetTotalCostByPeriod(ctx, tenantID, req.PeriodStart, req.PeriodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get total cost: %w", err)
	}

	costByEndpoint, err := s.repo.GetCostByEndpoint(ctx, tenantID, req.PeriodStart, req.PeriodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get cost by endpoint: %w", err)
	}

	costByEventType, err := s.repo.GetCostByEventType(ctx, tenantID, req.PeriodStart, req.PeriodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get cost by event type: %w", err)
	}

	costByDay, err := s.repo.GetCostByDay(ctx, tenantID, req.PeriodStart, req.PeriodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get cost by day: %w", err)
	}

	topEndpoints, err := s.repo.GetTopCostEndpoints(ctx, tenantID, req.PeriodStart, req.PeriodEnd, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get top cost endpoints: %w", err)
	}

	var avgCost float64
	if deliveryCount > 0 {
		avgCost = totalCost / float64(deliveryCount)
	}

	report := &CostReport{
		TenantID:           tenantID,
		PeriodStart:        req.PeriodStart,
		PeriodEnd:          req.PeriodEnd,
		TotalCost:          math.Round(totalCost*100) / 100,
		DeliveryCount:      deliveryCount,
		AvgCostPerDelivery: math.Round(avgCost*1e6) / 1e6,
		CostByEndpoint:     costByEndpoint,
		CostByEventType:    costByEventType,
		CostByDay:          costByDay,
		TopCostEndpoints:   topEndpoints,
	}

	return report, nil
}

// CreateBudget creates a new cost budget
func (s *Service) CreateBudget(ctx context.Context, tenantID string, req *CreateBudgetRequest) (*CostBudget, error) {
	budget := &CostBudget{
		ID:                uuid.New().String(),
		TenantID:          tenantID,
		Name:              req.Name,
		MonthlyLimit:      req.MonthlyLimit,
		AlertThresholdPct: req.AlertThresholdPct,
		CurrentSpend:      0,
		Period:            req.Period,
		IsActive:          true,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := s.repo.CreateBudget(ctx, budget); err != nil {
		return nil, fmt.Errorf("failed to create budget: %w", err)
	}

	return budget, nil
}

// GetBudget retrieves a budget by ID
func (s *Service) GetBudget(ctx context.Context, tenantID, budgetID string) (*CostBudget, error) {
	return s.repo.GetBudget(ctx, tenantID, budgetID)
}

// ListBudgets retrieves all budgets for a tenant
func (s *Service) ListBudgets(ctx context.Context, tenantID string) ([]CostBudget, error) {
	return s.repo.ListBudgets(ctx, tenantID)
}

// UpdateBudget updates an existing budget
func (s *Service) UpdateBudget(ctx context.Context, tenantID, budgetID string, req *CreateBudgetRequest) (*CostBudget, error) {
	budget, err := s.repo.GetBudget(ctx, tenantID, budgetID)
	if err != nil {
		return nil, err
	}

	budget.Name = req.Name
	budget.MonthlyLimit = req.MonthlyLimit
	budget.AlertThresholdPct = req.AlertThresholdPct
	budget.Period = req.Period
	budget.UpdatedAt = time.Now()

	if err := s.repo.UpdateBudget(ctx, budget); err != nil {
		return nil, fmt.Errorf("failed to update budget: %w", err)
	}

	return budget, nil
}

// CheckBudgetAlerts checks all active budgets against their thresholds
func (s *Service) CheckBudgetAlerts(ctx context.Context, tenantID string) ([]CostBudget, error) {
	budgets, err := s.repo.ListBudgets(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list budgets: %w", err)
	}

	periodStart := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
	var alerts []CostBudget
	for _, budget := range budgets {
		if !budget.IsActive {
			continue
		}

		currentSpend, err := s.repo.GetCurrentSpend(ctx, tenantID, periodStart)
		if err != nil {
			continue
		}

		budget.CurrentSpend = currentSpend
		spendPct := (currentSpend / budget.MonthlyLimit) * 100
		if spendPct >= budget.AlertThresholdPct {
			alerts = append(alerts, budget)
		}
	}

	return alerts, nil
}

// DetectAnomalies compares recent costs against historical averages and flags anomalies
func (s *Service) DetectAnomalies(ctx context.Context, tenantID string) ([]CostAnomaly, error) {
	historicalAvg, err := s.repo.GetHistoricalAvgCost(ctx, tenantID, 30)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical average: %w", err)
	}

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	recentCost, _, err := s.repo.GetTotalCostByPeriod(ctx, tenantID, todayStart, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent cost: %w", err)
	}

	var anomalies []CostAnomaly
	if historicalAvg > 0 {
		deviationPct := ((recentCost - historicalAvg) / historicalAvg) * 100

		if deviationPct > 50 {
			anomalyType := AnomalyTypeSpike
			if deviationPct > 200 {
				anomalyType = AnomalyTypeSustainedIncrease
			}

			anomaly := CostAnomaly{
				ID:           uuid.New().String(),
				TenantID:     tenantID,
				AnomalyType:  anomalyType,
				ExpectedCost: math.Round(historicalAvg*100) / 100,
				ActualCost:   math.Round(recentCost*100) / 100,
				DeviationPct: math.Round(deviationPct*100) / 100,
				DetectedAt:   now,
				Status:       AnomalyStatusActive,
			}

			if err := s.repo.CreateAnomaly(ctx, &anomaly); err == nil {
				anomalies = append(anomalies, anomaly)
			}
		}
	}

	existing, err := s.repo.ListAnomalies(ctx, tenantID)
	if err == nil {
		anomalies = append(anomalies, existing...)
	}

	return anomalies, nil
}

// GetCurrentSpend returns the current period spend for a tenant
func (s *Service) GetCurrentSpend(ctx context.Context, tenantID string) (float64, error) {
	periodStart := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
	spend, err := s.repo.GetCurrentSpend(ctx, tenantID, periodStart)
	if err != nil {
		return 0, fmt.Errorf("failed to get current spend: %w", err)
	}
	return math.Round(spend*100) / 100, nil
}
