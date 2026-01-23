package sla

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
)

// Service provides SLA management functionality
type Service struct {
	repo Repository
}

// NewService creates a new SLA service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateTarget creates a new SLA target
func (s *Service) CreateTarget(ctx context.Context, tenantID string, req *CreateTargetRequest) (*Target, error) {
	if req.DeliveryRatePct < 0 || req.DeliveryRatePct > 100 {
		return nil, fmt.Errorf("delivery_rate_pct must be between 0 and 100")
	}
	if req.WindowMinutes < 1 {
		return nil, fmt.Errorf("window_minutes must be at least 1")
	}

	target := &Target{
		ID:              uuid.New().String(),
		TenantID:        tenantID,
		EndpointID:      req.EndpointID,
		Name:            req.Name,
		DeliveryRatePct: req.DeliveryRatePct,
		LatencyP50Ms:    req.LatencyP50Ms,
		LatencyP99Ms:    req.LatencyP99Ms,
		WindowMinutes:   req.WindowMinutes,
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.repo.CreateTarget(ctx, target); err != nil {
		return nil, fmt.Errorf("failed to create SLA target: %w", err)
	}

	return target, nil
}

// GetTarget retrieves an SLA target by ID
func (s *Service) GetTarget(ctx context.Context, tenantID, targetID string) (*Target, error) {
	return s.repo.GetTarget(ctx, tenantID, targetID)
}

// ListTargets retrieves all SLA targets for a tenant
func (s *Service) ListTargets(ctx context.Context, tenantID string) ([]Target, error) {
	return s.repo.ListTargets(ctx, tenantID)
}

// UpdateTarget updates an SLA target
func (s *Service) UpdateTarget(ctx context.Context, tenantID, targetID string, req *CreateTargetRequest) (*Target, error) {
	target, err := s.repo.GetTarget(ctx, tenantID, targetID)
	if err != nil {
		return nil, err
	}

	target.Name = req.Name
	target.DeliveryRatePct = req.DeliveryRatePct
	target.LatencyP50Ms = req.LatencyP50Ms
	target.LatencyP99Ms = req.LatencyP99Ms
	target.WindowMinutes = req.WindowMinutes
	target.EndpointID = req.EndpointID
	target.UpdatedAt = time.Now()

	if err := s.repo.UpdateTarget(ctx, target); err != nil {
		return nil, fmt.Errorf("failed to update SLA target: %w", err)
	}

	return target, nil
}

// DeleteTarget deletes an SLA target
func (s *Service) DeleteTarget(ctx context.Context, tenantID, targetID string) error {
	return s.repo.DeleteTarget(ctx, tenantID, targetID)
}

// GetComplianceStatus calculates current compliance for a target
func (s *Service) GetComplianceStatus(ctx context.Context, tenantID string, target *Target) (*ComplianceStatus, error) {
	total, success, failed, _, p50, p99, err := s.repo.GetDeliveryStats(ctx, tenantID, target.EndpointID, target.WindowMinutes)
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery stats: %w", err)
	}

	var currentRate float64
	if total > 0 {
		currentRate = float64(success) / float64(total) * 100
	}

	isCompliant := currentRate >= target.DeliveryRatePct
	if target.LatencyP50Ms > 0 && p50 > target.LatencyP50Ms {
		isCompliant = false
	}
	if target.LatencyP99Ms > 0 && p99 > target.LatencyP99Ms {
		isCompliant = false
	}

	now := time.Now()
	return &ComplianceStatus{
		TargetID:        target.ID,
		TargetName:      target.Name,
		EndpointID:      target.EndpointID,
		IsCompliant:     isCompliant,
		CurrentRate:     math.Round(currentRate*100) / 100,
		RequiredRate:    target.DeliveryRatePct,
		CurrentP50Ms:    p50,
		CurrentP99Ms:    p99,
		TotalDeliveries: total,
		SuccessCount:    success,
		FailureCount:    failed,
		WindowStart:     now.Add(-time.Duration(target.WindowMinutes) * time.Minute),
		WindowEnd:       now,
		MeasuredAt:      now,
	}, nil
}

// GetDashboard builds the full SLA dashboard for a tenant
func (s *Service) GetDashboard(ctx context.Context, tenantID string) (*Dashboard, error) {
	targets, err := s.repo.ListTargets(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list targets: %w", err)
	}

	dashboard := &Dashboard{
		TenantID:       tenantID,
		Targets:        make([]ComplianceStatus, 0, len(targets)),
		BurnRates:      make([]BurnRate, 0, len(targets)),
		ActiveBreaches: make([]Breach, 0),
	}

	compliantCount := 0
	for _, target := range targets {
		if !target.IsActive {
			continue
		}

		status, err := s.GetComplianceStatus(ctx, tenantID, &target)
		if err != nil {
			continue
		}
		dashboard.Targets = append(dashboard.Targets, *status)

		burnRate := s.calculateBurnRate(&target, status)
		dashboard.BurnRates = append(dashboard.BurnRates, burnRate)

		if status.IsCompliant {
			compliantCount++
		}
	}

	if len(dashboard.Targets) > 0 {
		dashboard.OverallScore = float64(compliantCount) / float64(len(dashboard.Targets)) * 100
	}

	breaches, err := s.repo.ListActiveBreaches(ctx, tenantID)
	if err == nil {
		dashboard.ActiveBreaches = breaches
	}

	return dashboard, nil
}

// CheckAndRecordBreaches evaluates all targets and records any breaches
func (s *Service) CheckAndRecordBreaches(ctx context.Context, tenantID string) ([]Breach, error) {
	targets, err := s.repo.ListTargets(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var newBreaches []Breach
	for _, target := range targets {
		if !target.IsActive {
			continue
		}

		status, err := s.GetComplianceStatus(ctx, tenantID, &target)
		if err != nil {
			continue
		}

		if !status.IsCompliant {
			breach := Breach{
				ID:        uuid.New().String(),
				TenantID:  tenantID,
				TargetID:  target.ID,
				EndpointID: target.EndpointID,
				CreatedAt: time.Now(),
			}

			if status.CurrentRate < target.DeliveryRatePct {
				breach.BreachType = BreachTypeDeliveryRate
				breach.ExpectedVal = target.DeliveryRatePct
				breach.ActualVal = status.CurrentRate
				breach.Severity = s.calculateSeverity(target.DeliveryRatePct, status.CurrentRate)
			} else if target.LatencyP99Ms > 0 && status.CurrentP99Ms > target.LatencyP99Ms {
				breach.BreachType = BreachTypeLatencyP99
				breach.ExpectedVal = float64(target.LatencyP99Ms)
				breach.ActualVal = float64(status.CurrentP99Ms)
				breach.Severity = SeverityWarning
			}

			if breach.BreachType != "" {
				if err := s.repo.CreateBreach(ctx, &breach); err == nil {
					newBreaches = append(newBreaches, breach)
				}
			}
		}
	}

	return newBreaches, nil
}

// GetAlertConfig retrieves alert configuration for a target
func (s *Service) GetAlertConfig(ctx context.Context, tenantID, targetID string) (*AlertConfig, error) {
	return s.repo.GetAlertConfig(ctx, tenantID, targetID)
}

// UpdateAlertConfig updates alert configuration for a target
func (s *Service) UpdateAlertConfig(ctx context.Context, tenantID, targetID string, req *UpdateAlertConfigRequest) (*AlertConfig, error) {
	config := &AlertConfig{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		TargetID:     targetID,
		Channels:     req.Channels,
		CooldownMins: req.CooldownMins,
		IsActive:     req.IsActive,
		CreatedAt:    time.Now(),
	}

	if err := s.repo.UpsertAlertConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to update alert config: %w", err)
	}

	return config, nil
}

// GetBreachHistory retrieves historical breaches for a tenant
func (s *Service) GetBreachHistory(ctx context.Context, tenantID string, limit, offset int) ([]Breach, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListBreachHistory(ctx, tenantID, limit, offset)
}

func (s *Service) calculateBurnRate(target *Target, status *ComplianceStatus) BurnRate {
	errorBudget := 100.0 - target.DeliveryRatePct
	if errorBudget <= 0 {
		return BurnRate{TargetID: target.ID, ErrorBudgetPct: 0, IsAtRisk: true}
	}

	currentErrorRate := 100.0 - status.CurrentRate
	burnRate := currentErrorRate / errorBudget

	br := BurnRate{
		TargetID:       target.ID,
		CurrentRate:    math.Round(burnRate*100) / 100,
		ErrorBudgetPct: math.Max(0, math.Round((1-burnRate)*10000)/100),
		IsAtRisk:       burnRate > 1.0,
	}

	if burnRate > 1.0 && status.TotalDeliveries > 0 {
		remainingBudget := errorBudget - currentErrorRate
		if remainingBudget < 0 {
			br.ProjectedBreachIn = "breached"
		} else {
			minutesRemaining := float64(target.WindowMinutes) * (remainingBudget / currentErrorRate)
			br.ProjectedBreachIn = fmt.Sprintf("%.0fm", minutesRemaining)
		}
	}

	return br
}

func (s *Service) calculateSeverity(expected, actual float64) string {
	gap := expected - actual
	if gap > 10.0 {
		return SeverityCritical
	}
	return SeverityWarning
}
