package canary

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service provides canary deployment business logic
type Service struct {
	repo Repository
}

// NewService creates a new canary service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateDeployment creates a new canary deployment
func (s *Service) CreateDeployment(ctx context.Context, tenantID string, req *CreateCanaryRequest) (*CanaryDeployment, error) {
	if req.TrafficPct < 1 || req.TrafficPct > 100 {
		return nil, fmt.Errorf("traffic_pct must be between 1 and 100")
	}
	if req.PromotionRule != PromotionRuleManual && req.PromotionRule != PromotionRuleAutomatic {
		return nil, fmt.Errorf("promotion_rule must be manual or automatic")
	}

	now := time.Now()
	deployment := &CanaryDeployment{
		ID:              uuid.New().String(),
		TenantID:        tenantID,
		Name:            req.Name,
		Description:     req.Description,
		EndpointID:      req.EndpointID,
		EventType:       req.EventType,
		TrafficPct:      req.TrafficPct,
		Status:          CanaryStatusPending,
		PromotionRule:   req.PromotionRule,
		RollbackOnError: req.RollbackOnError,
		ErrorThreshold:  req.ErrorThreshold,
		SoakTimeMins:    req.SoakTimeMins,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if deployment.ErrorThreshold == 0 {
		deployment.ErrorThreshold = 5.0
	}
	if deployment.SoakTimeMins == 0 {
		deployment.SoakTimeMins = 30
	}

	if err := s.repo.CreateDeployment(ctx, deployment); err != nil {
		return nil, err
	}

	return deployment, nil
}

// GetDeployment retrieves a canary deployment by ID
func (s *Service) GetDeployment(ctx context.Context, tenantID, deploymentID string) (*CanaryDeployment, error) {
	return s.repo.GetDeployment(ctx, tenantID, deploymentID)
}

// ListDeployments lists canary deployments with optional status filter
func (s *Service) ListDeployments(ctx context.Context, tenantID, status string, limit, offset int) ([]CanaryDeployment, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListDeployments(ctx, tenantID, status, limit, offset)
}

// UpdateTraffic changes the traffic percentage for a canary deployment
func (s *Service) UpdateTraffic(ctx context.Context, tenantID, deploymentID string, trafficPct int) (*CanaryDeployment, error) {
	if trafficPct < 0 || trafficPct > 100 {
		return nil, fmt.Errorf("traffic_pct must be between 0 and 100")
	}

	deployment, err := s.repo.GetDeployment(ctx, tenantID, deploymentID)
	if err != nil {
		return nil, err
	}

	if deployment.Status != CanaryStatusActive && deployment.Status != CanaryStatusPending {
		return nil, fmt.Errorf("cannot update traffic for deployment in status: %s", deployment.Status)
	}

	deployment.TrafficPct = trafficPct
	deployment.UpdatedAt = time.Now()

	if err := s.repo.UpdateDeployment(ctx, deployment); err != nil {
		return nil, err
	}

	return deployment, nil
}

// PromoteCanary promotes the canary to full traffic
func (s *Service) PromoteCanary(ctx context.Context, tenantID, deploymentID string) (*CanaryDeployment, error) {
	deployment, err := s.repo.GetDeployment(ctx, tenantID, deploymentID)
	if err != nil {
		return nil, err
	}

	if deployment.Status != CanaryStatusActive {
		return nil, fmt.Errorf("can only promote active canary deployments, current status: %s", deployment.Status)
	}

	now := time.Now()
	deployment.Status = CanaryStatusPromoted
	deployment.TrafficPct = 100
	deployment.PromotedAt = &now
	deployment.UpdatedAt = now

	if err := s.repo.UpdateDeployment(ctx, deployment); err != nil {
		return nil, err
	}

	return deployment, nil
}

// RollbackCanary rolls back the canary deployment
func (s *Service) RollbackCanary(ctx context.Context, tenantID, deploymentID string) (*CanaryDeployment, error) {
	deployment, err := s.repo.GetDeployment(ctx, tenantID, deploymentID)
	if err != nil {
		return nil, err
	}

	if deployment.Status != CanaryStatusActive && deployment.Status != CanaryStatusPaused {
		return nil, fmt.Errorf("can only rollback active or paused canary deployments, current status: %s", deployment.Status)
	}

	now := time.Now()
	deployment.Status = CanaryStatusRolledBack
	deployment.TrafficPct = 0
	deployment.RolledBackAt = &now
	deployment.UpdatedAt = now

	if err := s.repo.UpdateDeployment(ctx, deployment); err != nil {
		return nil, err
	}

	return deployment, nil
}

// PauseCanary pauses the canary deployment
func (s *Service) PauseCanary(ctx context.Context, tenantID, deploymentID string) (*CanaryDeployment, error) {
	deployment, err := s.repo.GetDeployment(ctx, tenantID, deploymentID)
	if err != nil {
		return nil, err
	}

	if deployment.Status != CanaryStatusActive {
		return nil, fmt.Errorf("can only pause active canary deployments, current status: %s", deployment.Status)
	}

	deployment.Status = CanaryStatusPaused
	deployment.UpdatedAt = time.Now()

	if err := s.repo.UpdateDeployment(ctx, deployment); err != nil {
		return nil, err
	}

	return deployment, nil
}

// ResumeCanary resumes a paused canary deployment
func (s *Service) ResumeCanary(ctx context.Context, tenantID, deploymentID string) (*CanaryDeployment, error) {
	deployment, err := s.repo.GetDeployment(ctx, tenantID, deploymentID)
	if err != nil {
		return nil, err
	}

	if deployment.Status != CanaryStatusPaused {
		return nil, fmt.Errorf("can only resume paused canary deployments, current status: %s", deployment.Status)
	}

	deployment.Status = CanaryStatusActive
	deployment.UpdatedAt = time.Now()

	if err := s.repo.UpdateDeployment(ctx, deployment); err != nil {
		return nil, err
	}

	return deployment, nil
}

// EvaluateCanary compares canary vs baseline metrics and returns a health assessment
func (s *Service) EvaluateCanary(ctx context.Context, tenantID, deploymentID string) (*CanaryComparison, error) {
	deployment, err := s.repo.GetDeployment(ctx, tenantID, deploymentID)
	if err != nil {
		return nil, err
	}

	metrics, err := s.repo.GetLatestMetrics(ctx, tenantID, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("no metrics available for deployment: %w", err)
	}

	comparison := &CanaryComparison{
		DeploymentID: deploymentID,
	}

	// Calculate canary error rate
	if metrics.CanaryRequests > 0 {
		comparison.CanaryErrorRate = float64(metrics.CanaryErrors) / float64(metrics.CanaryRequests) * 100
	}

	// Calculate baseline error rate
	if metrics.BaselineRequests > 0 {
		comparison.BaselineErrorRate = float64(metrics.BaselineErrors) / float64(metrics.BaselineRequests) * 100
	}

	comparison.ErrorRateDelta = comparison.CanaryErrorRate - comparison.BaselineErrorRate

	// Latency comparison using p50
	comparison.CanaryAvgLatency = metrics.CanaryP50Ms
	comparison.BaselineAvgLatency = metrics.BaselineP50Ms
	comparison.LatencyDelta = comparison.CanaryAvgLatency - comparison.BaselineAvgLatency

	// Health assessment
	comparison.IsHealthy = comparison.CanaryErrorRate <= deployment.ErrorThreshold &&
		comparison.ErrorRateDelta <= deployment.ErrorThreshold

	if comparison.IsHealthy {
		comparison.Recommendation = "canary is healthy; safe to promote"
	} else if comparison.CanaryErrorRate > deployment.ErrorThreshold {
		comparison.Recommendation = "canary error rate exceeds threshold; consider rollback"
	} else {
		comparison.Recommendation = "canary error rate delta is elevated; monitor closely"
	}

	return comparison, nil
}

// RecordMetrics saves a metrics snapshot for a canary deployment
func (s *Service) RecordMetrics(ctx context.Context, tenantID string, metrics *CanaryMetrics) error {
	metrics.ID = uuid.New().String()
	metrics.TenantID = tenantID
	return s.repo.SaveMetrics(ctx, metrics)
}
