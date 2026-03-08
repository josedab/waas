package billing

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ForecastSpend forecasts spend for a period
func (s *Service) ForecastSpend(ctx context.Context, tenantID string, days int) (*SpendForecast, error) {
	// Get current period usage
	period := time.Now().Format("2006-01")
	summary, err := s.repo.GetUsageSummary(ctx, tenantID, period)
	if err != nil {
		return nil, fmt.Errorf("get usage summary: %w", err)
	}

	// Calculate daily average
	daysPassed := float64(time.Now().Day())
	if daysPassed < 1 {
		daysPassed = 1
	}
	dailyAverage := summary.TotalCost / daysPassed

	// Calculate forecasts
	forecast := &SpendForecast{
		TenantID:       tenantID,
		CurrentSpend:   summary.TotalCost,
		ProjectedSpend: summary.TotalCost + (dailyAverage * float64(days)),
		DailyAverage:   dailyAverage,
		Currency:       summary.Currency,
		Period:         period,
	}

	// Calculate confidence based on data points
	if len(summary.ByDay) >= 7 {
		forecast.Confidence = 0.85
	} else if len(summary.ByDay) >= 3 {
		forecast.Confidence = 0.7
	} else {
		forecast.Confidence = 0.5
	}

	// Calculate trend
	if len(summary.ByDay) >= 3 {
		recent := summary.ByDay[len(summary.ByDay)-3:]
		if len(recent) >= 3 {
			firstAvg := (recent[0].Cost + recent[1].Cost) / 2
			secondAvg := (recent[1].Cost + recent[2].Cost) / 2
			if secondAvg > firstAvg*1.1 {
				forecast.TrendDirection = "increasing"
				forecast.TrendPercent = ((secondAvg - firstAvg) / firstAvg) * 100
			} else if secondAvg < firstAvg*0.9 {
				forecast.TrendDirection = "decreasing"
				forecast.TrendPercent = ((firstAvg - secondAvg) / firstAvg) * 100
			} else {
				forecast.TrendDirection = "stable"
				forecast.TrendPercent = 0
			}
		}
	}

	// Get budget for comparison
	budgets, err := s.repo.ListBudgets(ctx, tenantID)
	if err == nil {
		for _, b := range budgets {
			if b.Enabled && b.Period == BillingPeriod(period[:7]) {
				forecast.BudgetRemaining = b.Amount - summary.TotalCost
				forecast.BudgetUtilization = (summary.TotalCost / b.Amount) * 100
				break
			}
		}
	}

	return forecast, nil
}

// ProjectCost projects future costs based on current usage patterns.
func (s *Service) ProjectCost(ctx context.Context, tenantID uuid.UUID, daysAhead int) (*UsageSummary, error) {
	summary, err := s.GetUsageSummaryForTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	daysPassed := float64(now.Day())
	if daysPassed < 1 {
		daysPassed = 1
	}

	dailyRate := float64(summary.EventsUsed) / daysPassed
	projectedEvents := summary.EventsUsed + int64(dailyRate*float64(daysAhead))

	plan := defaultFreePlan()
	projectedCost := s.CalculateCostForPlan(&plan, projectedEvents)

	summary.ProjectedCost = projectedCost
	return summary, nil
}
