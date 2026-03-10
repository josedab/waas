package capacityplanner

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/utils"
)

// ErrReportNotFound is returned when a capacity report is not found.
var ErrReportNotFound = errors.New("capacity report not found")

// ServiceConfig configures the capacity planner service.
type ServiceConfig struct {
	DefaultLookbackDays int
	HighUtilThreshold   float64
}

// DefaultServiceConfig returns sensible defaults.
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		DefaultLookbackDays: 30,
		HighUtilThreshold:   80.0,
	}
}

// Service implements capacity planning business logic.
type Service struct {
	repo   Repository
	logger *utils.Logger
	config *ServiceConfig
}

// NewService creates a new capacity planner service.
func NewService(repo Repository, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}
	return &Service{
		repo:   repo,
		logger: utils.NewLogger("capacityplanner"),
		config: config,
	}
}

// GenerateReport analyzes traffic and produces a capacity report with projections and recommendations.
func (s *Service) GenerateReport(ctx context.Context, tenantID string, req *GenerateReportRequest) (*CapacityReport, error) {
	usage, err := s.GetCurrentUsage(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current usage: %w", err)
	}

	projections := s.buildProjections(usage)
	recommendations := s.buildRecommendations(usage, projections)
	bottlenecks := s.detectBottlenecks(usage)

	peakUsage := *usage
	peakUsage.DailyAvgRequests = usage.DailyAvgRequests * 1.5
	peakUsage.PeakRequestsPerSec = usage.PeakRequestsPerSec * 2.0

	report := &CapacityReport{
		ID:              uuid.New().String(),
		TenantID:        tenantID,
		CurrentUsage:    *usage,
		PeakUsage:       peakUsage,
		Projections:     projections,
		Recommendations: recommendations,
		Bottlenecks:     bottlenecks,
		GeneratedAt:     time.Now(),
	}

	if err := s.repo.SaveReport(report); err != nil {
		return nil, fmt.Errorf("failed to save report: %w", err)
	}

	s.logger.Info("capacity report generated", map[string]interface{}{
		"tenant_id": tenantID,
		"report_id": report.ID,
	})

	return report, nil
}

// GetReport retrieves a capacity report by ID.
func (s *Service) GetReport(ctx context.Context, reportID string) (*CapacityReport, error) {
	report, err := s.repo.GetReport(reportID)
	if err != nil {
		return nil, ErrReportNotFound
	}
	return report, nil
}

// GetCurrentUsage returns current usage metrics for a tenant.
func (s *Service) GetCurrentUsage(ctx context.Context, tenantID string) (*UsageMetrics, error) {
	end := time.Now()
	start := end.AddDate(0, 0, -s.config.DefaultLookbackDays)

	history, err := s.repo.GetTrafficHistory(tenantID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get traffic history: %w", err)
	}

	usage := &UsageMetrics{}
	if len(history) == 0 {
		return usage, nil
	}

	var totalRPS float64
	var peakRPS float64
	var totalLatency float64
	var maxEndpoints int

	for _, snap := range history {
		totalRPS += snap.RequestsPerSec
		totalLatency += snap.AvgLatencyMs
		if snap.RequestsPerSec > peakRPS {
			peakRPS = snap.RequestsPerSec
		}
		if snap.ActiveEndpoints > maxEndpoints {
			maxEndpoints = snap.ActiveEndpoints
		}
	}

	avgRPS := totalRPS / float64(len(history))
	usage.DailyAvgRequests = math.Round(avgRPS*86400*100) / 100
	usage.PeakRequestsPerSec = math.Round(peakRPS*100) / 100
	usage.ActiveEndpoints = maxEndpoints
	usage.TotalEndpoints = maxEndpoints
	usage.AvgPayloadSizeKB = 2.5
	usage.StorageUsedGB = math.Round(avgRPS*86400*2.5/1e6*100) / 100
	usage.BandwidthUsedGB = math.Round(avgRPS*86400*2.5/1e6*float64(s.config.DefaultLookbackDays)*100) / 100

	return usage, nil
}

// GetTrafficHistory returns traffic snapshots for a tenant within a time range.
func (s *Service) GetTrafficHistory(ctx context.Context, tenantID string, tr TimeRange) ([]TrafficSnapshot, error) {
	snapshots, err := s.repo.GetTrafficHistory(tenantID, tr.Start, tr.End)
	if err != nil {
		return nil, fmt.Errorf("failed to get traffic history: %w", err)
	}
	return snapshots, nil
}

// GetProjections returns growth projections for 30d, 90d, 180d, and 365d.
func (s *Service) GetProjections(ctx context.Context, tenantID string) ([]GrowthProjection, error) {
	usage, err := s.GetCurrentUsage(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current usage: %w", err)
	}
	return s.buildProjections(usage), nil
}

// GetRecommendations returns scaling recommendations for a tenant.
func (s *Service) GetRecommendations(ctx context.Context, tenantID string) ([]ScalingRecommendation, error) {
	usage, err := s.GetCurrentUsage(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current usage: %w", err)
	}
	projections := s.buildProjections(usage)
	return s.buildRecommendations(usage, projections), nil
}

// ListAlerts returns all capacity alerts for a tenant.
func (s *Service) ListAlerts(ctx context.Context, tenantID string) ([]CapacityAlert, error) {
	alerts, err := s.repo.ListAlerts(tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list alerts: %w", err)
	}
	return alerts, nil
}

// SetAlertThreshold sets the alert threshold for a resource.
func (s *Service) SetAlertThreshold(ctx context.Context, tenantID string, req *SetAlertThresholdRequest) error {
	if err := s.repo.SetAlertThreshold(tenantID, req.Resource, req.ThresholdValue); err != nil {
		return fmt.Errorf("failed to set alert threshold: %w", err)
	}

	s.logger.Info("alert threshold set", map[string]interface{}{
		"tenant_id": tenantID,
		"resource":  req.Resource,
		"threshold": req.ThresholdValue,
	})

	return nil
}

// AcknowledgeAlert removes an alert by ID.
func (s *Service) AcknowledgeAlert(ctx context.Context, alertID string) error {
	if _, err := s.repo.GetAlert(alertID); err != nil {
		return fmt.Errorf("alert not found: %w", err)
	}
	if err := s.repo.DeleteAlert(alertID); err != nil {
		return fmt.Errorf("failed to acknowledge alert: %w", err)
	}
	return nil
}

// buildProjections generates growth projections using simple linear estimation.
func (s *Service) buildProjections(usage *UsageMetrics) []GrowthProjection {
	periods := []struct {
		label      string
		days       int
		confidence float64
	}{
		{"30d", 30, 0.90},
		{"90d", 90, 0.75},
		{"180d", 180, 0.60},
		{"365d", 365, 0.45},
	}

	// Assume a monthly growth rate of 10% for linear estimation
	monthlyGrowthRate := 0.10

	var projections []GrowthProjection
	for _, p := range periods {
		months := float64(p.days) / 30.0
		growthFactor := 1.0 + (monthlyGrowthRate * months)
		growthPct := math.Round((growthFactor-1.0)*10000) / 100

		projections = append(projections, GrowthProjection{
			Period:             p.label,
			ProjectedDailyReqs: math.Round(usage.DailyAvgRequests*growthFactor*100) / 100,
			ProjectedPeakRPS:   math.Round(usage.PeakRequestsPerSec*growthFactor*100) / 100,
			GrowthRatePct:      growthPct,
			ConfidenceLevel:    p.confidence,
		})
	}

	return projections
}

// buildRecommendations generates scaling recommendations based on usage and projections.
func (s *Service) buildRecommendations(usage *UsageMetrics, projections []GrowthProjection) []ScalingRecommendation {
	var recs []ScalingRecommendation

	if usage.PeakRequestsPerSec > 1000 {
		recs = append(recs, ScalingRecommendation{
			Resource:            "compute",
			CurrentCapacity:     fmt.Sprintf("%.0f RPS", usage.PeakRequestsPerSec),
			RecommendedCapacity: fmt.Sprintf("%.0f RPS", usage.PeakRequestsPerSec*2),
			Urgency:             "high",
			Reason:              "Peak RPS exceeds 1000; recommend doubling compute capacity",
			EstimatedCostImpact: math.Round(usage.PeakRequestsPerSec*0.01*100) / 100,
		})
	}

	if usage.StorageUsedGB > 50 {
		recs = append(recs, ScalingRecommendation{
			Resource:            "storage",
			CurrentCapacity:     fmt.Sprintf("%.1f GB", usage.StorageUsedGB),
			RecommendedCapacity: fmt.Sprintf("%.1f GB", usage.StorageUsedGB*1.5),
			Urgency:             "medium",
			Reason:              "Storage usage exceeds 50 GB; plan for growth",
			EstimatedCostImpact: math.Round(usage.StorageUsedGB*0.05*100) / 100,
		})
	}

	if usage.BandwidthUsedGB > 100 {
		recs = append(recs, ScalingRecommendation{
			Resource:            "bandwidth",
			CurrentCapacity:     fmt.Sprintf("%.1f GB", usage.BandwidthUsedGB),
			RecommendedCapacity: fmt.Sprintf("%.1f GB", usage.BandwidthUsedGB*1.5),
			Urgency:             "medium",
			Reason:              "Bandwidth usage exceeds 100 GB; consider CDN or edge caching",
			EstimatedCostImpact: math.Round(usage.BandwidthUsedGB*0.02*100) / 100,
		})
	}

	// Check 90-day projection for proactive scaling
	for _, proj := range projections {
		if proj.Period == "90d" && proj.ProjectedPeakRPS > 5000 {
			recs = append(recs, ScalingRecommendation{
				Resource:            "compute",
				CurrentCapacity:     fmt.Sprintf("%.0f RPS", usage.PeakRequestsPerSec),
				RecommendedCapacity: fmt.Sprintf("%.0f RPS", proj.ProjectedPeakRPS*1.5),
				Urgency:             "low",
				Reason:              "90-day projection exceeds 5000 RPS; plan capacity increase",
				EstimatedCostImpact: math.Round(proj.ProjectedPeakRPS*0.01*100) / 100,
			})
		}
	}

	return recs
}

// detectBottlenecks identifies resource constraints.
func (s *Service) detectBottlenecks(usage *UsageMetrics) []Bottleneck {
	var bottlenecks []Bottleneck

	computeUtil := (usage.PeakRequestsPerSec / 10000) * 100
	if computeUtil > s.config.HighUtilThreshold {
		bottlenecks = append(bottlenecks, Bottleneck{
			Resource:       "compute",
			CurrentUtilPct: math.Round(computeUtil*100) / 100,
			ThresholdPct:   s.config.HighUtilThreshold,
			Impact:         "Increased latency and potential request drops",
			Mitigation:     "Scale out compute instances or optimize request handling",
		})
	}

	storageUtil := (usage.StorageUsedGB / 500) * 100
	if storageUtil > s.config.HighUtilThreshold {
		bottlenecks = append(bottlenecks, Bottleneck{
			Resource:       "storage",
			CurrentUtilPct: math.Round(storageUtil*100) / 100,
			ThresholdPct:   s.config.HighUtilThreshold,
			Impact:         "Risk of storage exhaustion and data loss",
			Mitigation:     "Increase storage capacity or implement data retention policies",
		})
	}

	return bottlenecks
}

// ForecastCosts generates a cost forecast for a specific cloud provider.
func (s *Service) ForecastCosts(ctx context.Context, tenantID, provider string) (*CloudCostForecast, error) {
	usage, err := s.GetCurrentUsage(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage: %w", err)
	}

	var computeRate, storageRate, bandwidthRate float64
	switch provider {
	case CloudAWS:
		computeRate, storageRate, bandwidthRate = 0.0464, 0.023, 0.09
	case CloudGCP:
		computeRate, storageRate, bandwidthRate = 0.0440, 0.020, 0.08
	case CloudAzure:
		computeRate, storageRate, bandwidthRate = 0.0456, 0.021, 0.087
	default:
		return nil, fmt.Errorf("unsupported provider: %s; use aws, gcp, or azure", provider)
	}

	computeCost := usage.PeakRequestsPerSec * computeRate * 730
	storageCost := usage.StorageUsedGB * storageRate
	bandwidthCost := usage.BandwidthUsedGB * bandwidthRate
	currentTotal := computeCost + storageCost + bandwidthCost

	forecast := &CloudCostForecast{
		Provider:           provider,
		CurrentMonthlyCost: math.Round(currentTotal*100) / 100,
		CostBreakdown: []CostLineItem{
			{Category: "compute", Amount: math.Round(computeCost*100) / 100, Unit: "monthly"},
			{Category: "storage", Amount: math.Round(storageCost*100) / 100, Unit: "monthly"},
			{Category: "bandwidth", Amount: math.Round(bandwidthCost*100) / 100, Unit: "monthly"},
		},
	}

	projections := s.buildProjections(usage)
	for _, p := range projections {
		growthFactor := 1 + p.GrowthRatePct/100
		forecast.ProjectedCosts = append(forecast.ProjectedCosts, ProjectedCost{
			Period:     p.Period,
			Amount:     math.Round(currentTotal*growthFactor*100) / 100,
			Confidence: p.ConfidenceLevel,
		})
	}

	if usage.PeakRequestsPerSec < usage.DailyAvgRequests/86400*3 {
		forecast.Savings = append(forecast.Savings, CostSavingTip{
			Description:     "Use auto-scaling to reduce over-provisioned compute",
			EstimatedSaving: 20,
			Effort:          "medium",
		})
	}
	if usage.StorageUsedGB > 10 {
		forecast.Savings = append(forecast.Savings, CostSavingTip{
			Description:     "Implement data retention policies to reduce storage costs",
			EstimatedSaving: 15,
			Effort:          "low",
		})
	}

	return forecast, nil
}

// GenerateWeeklyDigest creates a weekly summary with key metrics and recommendations.
func (s *Service) GenerateWeeklyDigest(ctx context.Context, tenantID string) (*WeeklyDigest, error) {
	usage, err := s.GetCurrentUsage(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage: %w", err)
	}

	projections := s.buildProjections(usage)
	recommendations := s.buildRecommendations(usage, projections)
	alerts, _ := s.repo.ListAlerts(tenantID)

	// Determine trend
	trend := "stable"
	if len(projections) > 0 && projections[0].GrowthRatePct > 15 {
		trend = "growing"
	} else if len(projections) > 0 && projections[0].GrowthRatePct < -5 {
		trend = "declining"
	}

	// Limit to top 3 recommendations
	topRecs := recommendations
	if len(topRecs) > 3 {
		topRecs = topRecs[:3]
	}

	return &WeeklyDigest{
		TenantID:           tenantID,
		PeriodStart:        time.Now().AddDate(0, 0, -7),
		PeriodEnd:          time.Now(),
		UsageSummary:       *usage,
		TopRecommendations: topRecs,
		Alerts:             alerts,
		TrendDirection:     trend,
		GeneratedAt:        time.Now(),
	}, nil
}
