package progressive

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/utils"
)

var (
	ErrRolloutNotFound     = errors.New("rollout not found")
	ErrInvalidTrafficSplit = errors.New("traffic split percentages must sum to 100")
	ErrInvalidStrategy     = errors.New("unsupported rollout strategy")
	ErrInvalidTransition   = errors.New("invalid status transition")
)

// ServiceConfig holds configuration for the progressive delivery service.
type ServiceConfig struct {
	MaxRolloutsPerTenant   int
	DefaultBaselinePercent int
	DefaultTargetPercent   int
}

// DefaultServiceConfig returns sensible defaults.
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		MaxRolloutsPerTenant:   50,
		DefaultBaselinePercent: 90,
		DefaultTargetPercent:   10,
	}
}

// Service provides progressive delivery operations.
type Service struct {
	repo   Repository
	config *ServiceConfig
	logger *utils.Logger
}

// NewService creates a new progressive delivery service.
func NewService(repo Repository, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}
	if repo == nil {
		repo = NewMemoryRepository()
	}
	return &Service{repo: repo, config: config, logger: utils.NewLogger("progressive")}
}

// CreateRollout creates a new progressive rollout.
func (s *Service) CreateRollout(ctx context.Context, tenantID string, req *CreateRolloutRequest) (*Rollout, error) {
	if !isValidStrategy(req.Strategy) {
		return nil, ErrInvalidStrategy
	}

	split := req.TrafficSplit
	if split.BaselinePercent == 0 && split.TargetPercent == 0 {
		split.BaselinePercent = s.config.DefaultBaselinePercent
		split.TargetPercent = s.config.DefaultTargetPercent
	}
	if split.BaselinePercent+split.TargetPercent != 100 {
		return nil, ErrInvalidTrafficSplit
	}

	if req.SuccessCriteria.MinSuccessRate <= 0 {
		req.SuccessCriteria.MinSuccessRate = 0.95
	}
	if req.SuccessCriteria.MinSampleSize <= 0 {
		req.SuccessCriteria.MinSampleSize = 100
	}

	now := time.Now().UTC()
	rollout := &Rollout{
		ID:              uuid.New().String(),
		TenantID:        tenantID,
		Name:            req.Name,
		EndpointID:      req.EndpointID,
		Strategy:        req.Strategy,
		Status:          StatusPending,
		TargetConfig:    req.TargetConfig,
		BaselineConfig:  req.BaselineConfig,
		TrafficSplit:    split,
		SuccessCriteria: req.SuccessCriteria,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.Create(ctx, rollout); err != nil {
		return nil, fmt.Errorf("create rollout: %w", err)
	}
	return rollout, nil
}

// GetRollout retrieves a rollout by ID.
func (s *Service) GetRollout(ctx context.Context, tenantID, id string) (*Rollout, error) {
	return s.repo.Get(ctx, tenantID, id)
}

// ListRollouts returns rollouts for a tenant.
func (s *Service) ListRollouts(ctx context.Context, tenantID string) ([]Rollout, error) {
	return s.repo.List(ctx, tenantID)
}

// StartRollout transitions a rollout to active.
func (s *Service) StartRollout(ctx context.Context, tenantID, id string) (*Rollout, error) {
	rollout, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if rollout.Status != StatusPending {
		return nil, fmt.Errorf("%w: can only start a pending rollout", ErrInvalidTransition)
	}

	rollout.Status = StatusActive
	rollout.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, rollout); err != nil {
		return nil, fmt.Errorf("start rollout: %w", err)
	}
	return rollout, nil
}

// UpdateTraffic adjusts the traffic split for an active rollout.
func (s *Service) UpdateTraffic(ctx context.Context, tenantID, id string, req *UpdateTrafficRequest) (*Rollout, error) {
	if req.BaselinePercent+req.TargetPercent != 100 {
		return nil, ErrInvalidTrafficSplit
	}

	rollout, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if rollout.Status != StatusActive {
		return nil, fmt.Errorf("%w: can only update traffic on an active rollout", ErrInvalidTransition)
	}

	rollout.TrafficSplit = TrafficSplit{
		BaselinePercent: req.BaselinePercent,
		TargetPercent:   req.TargetPercent,
	}
	rollout.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, rollout); err != nil {
		return nil, fmt.Errorf("update traffic: %w", err)
	}
	return rollout, nil
}

// PauseRollout pauses an active rollout.
func (s *Service) PauseRollout(ctx context.Context, tenantID, id string) (*Rollout, error) {
	rollout, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if rollout.Status != StatusActive {
		return nil, fmt.Errorf("%w: can only pause an active rollout", ErrInvalidTransition)
	}

	rollout.Status = StatusPaused
	rollout.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, rollout); err != nil {
		return nil, fmt.Errorf("pause rollout: %w", err)
	}
	return rollout, nil
}

// ResumeRollout resumes a paused rollout.
func (s *Service) ResumeRollout(ctx context.Context, tenantID, id string) (*Rollout, error) {
	rollout, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if rollout.Status != StatusPaused {
		return nil, fmt.Errorf("%w: can only resume a paused rollout", ErrInvalidTransition)
	}

	rollout.Status = StatusActive
	rollout.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, rollout); err != nil {
		return nil, fmt.Errorf("resume rollout: %w", err)
	}
	return rollout, nil
}

// CompleteRollout promotes the target to 100% traffic and marks the rollout as completed.
func (s *Service) CompleteRollout(ctx context.Context, tenantID, id string) (*Rollout, error) {
	rollout, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if rollout.Status != StatusActive {
		return nil, fmt.Errorf("%w: can only complete an active rollout", ErrInvalidTransition)
	}

	rollout.Status = StatusCompleted
	rollout.TrafficSplit = TrafficSplit{BaselinePercent: 0, TargetPercent: 100}
	rollout.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, rollout); err != nil {
		return nil, fmt.Errorf("complete rollout: %w", err)
	}
	return rollout, nil
}

// RollbackRollout sends all traffic back to the baseline and marks the rollout as rolled back.
func (s *Service) RollbackRollout(ctx context.Context, tenantID, id string) (*Rollout, error) {
	rollout, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if rollout.Status != StatusActive && rollout.Status != StatusPaused {
		return nil, fmt.Errorf("%w: can only rollback an active or paused rollout", ErrInvalidTransition)
	}

	rollout.Status = StatusRolledBack
	rollout.TrafficSplit = TrafficSplit{BaselinePercent: 100, TargetPercent: 0}
	rollout.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, rollout); err != nil {
		return nil, fmt.Errorf("rollback rollout: %w", err)
	}
	return rollout, nil
}

// EvaluationResult holds the outcome of evaluating a rollout against its success criteria.
type EvaluationResult struct {
	RolloutID         string  `json:"rollout_id"`
	Passed            bool    `json:"passed"`
	TargetSuccessRate float64 `json:"target_success_rate"`
	TargetAvgLatency  float64 `json:"target_avg_latency_ms"`
	SampleSize        int64   `json:"sample_size"`
	Message           string  `json:"message"`
}

// EvaluateRollout compares the target variant's metrics against the success criteria.
func (s *Service) EvaluateRollout(ctx context.Context, tenantID, id string) (*EvaluationResult, error) {
	rollout, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	target := rollout.Metrics.TargetMetrics
	criteria := rollout.SuccessCriteria

	result := &EvaluationResult{
		RolloutID:         rollout.ID,
		TargetSuccessRate: target.SuccessRate,
		TargetAvgLatency:  target.AvgLatencyMs,
		SampleSize:        target.Requests,
	}

	if target.Requests < int64(criteria.MinSampleSize) {
		result.Message = fmt.Sprintf("insufficient samples: %d < %d", target.Requests, criteria.MinSampleSize)
		return result, nil
	}

	if target.SuccessRate < criteria.MinSuccessRate {
		result.Message = fmt.Sprintf("success rate %.2f below threshold %.2f", target.SuccessRate, criteria.MinSuccessRate)
		return result, nil
	}

	if criteria.MaxLatencyMs > 0 && target.AvgLatencyMs > criteria.MaxLatencyMs {
		result.Message = fmt.Sprintf("avg latency %.2fms exceeds threshold %.2fms", target.AvgLatencyMs, criteria.MaxLatencyMs)
		return result, nil
	}

	result.Passed = true
	result.Message = "all success criteria met"
	return result, nil
}

// DeleteRollout removes a rollout.
func (s *Service) DeleteRollout(ctx context.Context, tenantID, id string) error {
	return s.repo.Delete(ctx, tenantID, id)
}

func isValidStrategy(strategy string) bool {
	switch strategy {
	case StrategyCanary, StrategyBlueGreen, StrategyPercentage, StrategyRingBased:
		return true
	default:
		return false
	}
}
