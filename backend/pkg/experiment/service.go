package experiment

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
)

// Service provides experiment business logic.
type Service struct {
	repo Repository
}

// NewService creates a new experiment service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateExperiment creates a new A/B test.
func (s *Service) CreateExperiment(ctx context.Context, tenantID string, req *CreateExperimentRequest) (*Experiment, error) {
	if len(req.Variants) < 2 {
		return nil, fmt.Errorf("at least 2 variants are required")
	}

	totalPercent := 0
	for _, v := range req.Variants {
		if v.TrafficPercent < 1 || v.TrafficPercent > 100 {
			return nil, fmt.Errorf("traffic_percent must be between 1 and 100 for variant %q", v.ID)
		}
		totalPercent += v.TrafficPercent
	}
	if totalPercent != 100 {
		return nil, fmt.Errorf("variant traffic percentages must sum to 100, got %d", totalPercent)
	}

	if req.SuccessCriteria.ConfidenceLevel <= 0 {
		req.SuccessCriteria.ConfidenceLevel = 0.95
	}
	if req.SuccessCriteria.MinSampleSize <= 0 {
		req.SuccessCriteria.MinSampleSize = 100
	}

	now := time.Now()
	exp := &Experiment{
		ID:              uuid.New().String(),
		TenantID:        tenantID,
		Name:            req.Name,
		Description:     req.Description,
		Status:          StatusDraft,
		EventType:       req.EventType,
		Variants:        req.Variants,
		SuccessCriteria: req.SuccessCriteria,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if s.repo != nil {
		if err := s.repo.CreateExperiment(ctx, exp); err != nil {
			return nil, fmt.Errorf("failed to create experiment: %w", err)
		}
	}

	return exp, nil
}

// StartExperiment transitions an experiment to running.
func (s *Service) StartExperiment(ctx context.Context, tenantID, expID string) (*Experiment, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}

	exp, err := s.repo.GetExperiment(ctx, tenantID, expID)
	if err != nil {
		return nil, err
	}

	if exp.Status != StatusDraft && exp.Status != StatusPaused {
		return nil, fmt.Errorf("experiment must be in draft or paused status to start")
	}

	now := time.Now()
	exp.Status = StatusRunning
	exp.StartedAt = &now
	exp.UpdatedAt = now

	if err := s.repo.UpdateExperiment(ctx, exp); err != nil {
		return nil, fmt.Errorf("failed to start experiment: %w", err)
	}

	return exp, nil
}

// StopExperiment stops a running experiment.
func (s *Service) StopExperiment(ctx context.Context, tenantID, expID string) (*Experiment, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}

	exp, err := s.repo.GetExperiment(ctx, tenantID, expID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	exp.Status = StatusCompleted
	exp.EndedAt = &now
	exp.UpdatedAt = now

	if err := s.repo.UpdateExperiment(ctx, exp); err != nil {
		return nil, fmt.Errorf("failed to stop experiment: %w", err)
	}

	return exp, nil
}

// AssignVariant deterministically assigns a webhook to a variant using
// consistent hashing so the same webhook always gets the same variant.
func (s *Service) AssignVariant(ctx context.Context, experimentID, webhookID string, variants []Variant) string {
	hash := sha256.Sum256([]byte(experimentID + ":" + webhookID))
	bucket := int(binary.BigEndian.Uint32(hash[:4])) % 100

	cumulative := 0
	for _, v := range variants {
		cumulative += v.TrafficPercent
		if bucket < cumulative {
			return v.ID
		}
	}
	return variants[len(variants)-1].ID
}

// GetExperiment retrieves an experiment.
func (s *Service) GetExperiment(ctx context.Context, tenantID, expID string) (*Experiment, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}
	return s.repo.GetExperiment(ctx, tenantID, expID)
}

// ListExperiments lists experiments for a tenant.
func (s *Service) ListExperiments(ctx context.Context, tenantID string) ([]Experiment, error) {
	if s.repo == nil {
		return []Experiment{}, nil
	}
	return s.repo.ListExperiments(ctx, tenantID)
}

// GetResults computes experiment results with statistical significance testing.
func (s *Service) GetResults(ctx context.Context, tenantID, expID string) (*ExperimentResults, error) {
	if s.repo == nil {
		return &ExperimentResults{ExperimentID: expID}, nil
	}

	exp, err := s.repo.GetExperiment(ctx, tenantID, expID)
	if err != nil {
		return nil, err
	}

	metrics, err := s.repo.GetVariantMetrics(ctx, expID)
	if err != nil {
		return nil, err
	}

	results := &ExperimentResults{
		ExperimentID:    expID,
		Status:          exp.Status,
		ConfidenceLevel: exp.SuccessCriteria.ConfidenceLevel,
	}

	variantMap := make(map[string]Variant)
	for _, v := range exp.Variants {
		variantMap[v.ID] = v
	}

	for _, m := range metrics {
		successRate := 0.0
		if m.TotalRequests > 0 {
			successRate = float64(m.SuccessCount) / float64(m.TotalRequests)
		}
		vr := VariantResult{
			VariantID:    m.VariantID,
			VariantName:  variantMap[m.VariantID].Name,
			SuccessRate:  successRate,
			SampleSize:   m.TotalRequests,
			AvgLatencyMs: m.AvgLatencyMs,
		}
		results.Variants = append(results.Variants, vr)
	}

	// Chi-squared test for significance
	if len(results.Variants) >= 2 {
		results.IsSignificant = chiSquaredTest(results.Variants, exp.SuccessCriteria.ConfidenceLevel)
		if results.IsSignificant {
			bestRate := 0.0
			for _, v := range results.Variants {
				if v.SuccessRate > bestRate {
					bestRate = v.SuccessRate
					results.WinnerVariant = v.VariantID
				}
			}
		}
	}

	return results, nil
}

// DeleteExperiment removes an experiment.
func (s *Service) DeleteExperiment(ctx context.Context, tenantID, expID string) error {
	if s.repo == nil {
		return fmt.Errorf("repository not configured")
	}
	return s.repo.DeleteExperiment(ctx, tenantID, expID)
}

// chiSquaredTest performs a basic chi-squared test of independence on variant
// success rates. Returns true if the difference is statistically significant
// at the given confidence level.
func chiSquaredTest(variants []VariantResult, confidence float64) bool {
	if len(variants) < 2 {
		return false
	}

	totalSuccess := 0.0
	totalFailure := 0.0
	for _, v := range variants {
		totalSuccess += float64(v.SampleSize) * v.SuccessRate
		totalFailure += float64(v.SampleSize) * (1 - v.SuccessRate)
	}
	total := totalSuccess + totalFailure
	if total == 0 {
		return false
	}

	chiSq := 0.0
	for _, v := range variants {
		n := float64(v.SampleSize)
		if n == 0 {
			continue
		}
		observed := n * v.SuccessRate
		expected := n * (totalSuccess / total)
		if expected > 0 {
			chiSq += math.Pow(observed-expected, 2) / expected
		}

		observedFail := n * (1 - v.SuccessRate)
		expectedFail := n * (totalFailure / total)
		if expectedFail > 0 {
			chiSq += math.Pow(observedFail-expectedFail, 2) / expectedFail
		}
	}

	// Critical value for df=1 at common confidence levels
	criticalValue := 3.841 // 95%
	if confidence >= 0.99 {
		criticalValue = 6.635
	} else if confidence >= 0.975 {
		criticalValue = 5.024
	}

	return chiSq > criticalValue
}
