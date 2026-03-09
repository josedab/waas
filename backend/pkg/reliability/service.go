package reliability

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
)

// Service manages endpoint reliability scoring and alerting.
type Service struct {
	repo Repository
}

// NewService creates a new reliability service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// GetReliability returns the full reliability report for an endpoint.
func (s *Service) GetReliability(ctx context.Context, tenantID, endpointID string) (*ReliabilityReport, error) {
	score, err := s.ComputeScore(ctx, tenantID, endpointID, 24)
	if err != nil {
		return nil, fmt.Errorf("computing score: %w", err)
	}

	snapshots, err := s.repo.ListSnapshots(ctx, tenantID, endpointID, 168) // 7 days hourly
	if err != nil {
		return nil, fmt.Errorf("listing snapshots: %w", err)
	}

	report := &ReliabilityReport{
		CurrentScore: score,
		Trend: &ReliabilityTrend{
			EndpointID: endpointID,
			Period:     "7d",
			DataPoints: snapshots,
		},
	}

	sla, err := s.repo.GetSLA(ctx, tenantID, endpointID)
	if err == nil && sla != nil {
		report.SLA = &SLAStatus{
			Target:        sla,
			CurrentScore:  score.Score,
			CurrentUptime: score.SuccessRate,
			CurrentP95Ms:  score.LatencyP95Ms,
			IsCompliant:   score.Score >= sla.TargetScore && score.SuccessRate >= sla.TargetUptime && score.LatencyP95Ms <= sla.MaxLatencyP95Ms,
		}
	}

	return report, nil
}

// ComputeScore calculates the reliability score for an endpoint over a window.
func (s *Service) ComputeScore(ctx context.Context, tenantID, endpointID string, windowHours int) (*ReliabilityScore, error) {
	stats, err := s.repo.GetDeliveryStats(ctx, tenantID, endpointID, windowHours)
	if err != nil {
		return nil, fmt.Errorf("getting delivery stats: %w", err)
	}

	score := computeReliabilityScore(stats)

	now := time.Now().UTC()
	rs := &ReliabilityScore{
		ID:                  uuid.New().String(),
		TenantID:            tenantID,
		EndpointID:          endpointID,
		Score:               score,
		Status:              scoreToStatus(score),
		SuccessRate:         safePercent(stats.SuccessfulAttempts, stats.TotalAttempts),
		LatencyP50Ms:        stats.LatencyP50Ms,
		LatencyP95Ms:        stats.LatencyP95Ms,
		LatencyP99Ms:        stats.LatencyP99Ms,
		ConsecutiveFailures: stats.ConsecutiveFailures,
		TotalAttempts:       stats.TotalAttempts,
		SuccessfulAttempts:  stats.SuccessfulAttempts,
		FailedAttempts:      stats.FailedAttempts,
		WindowStart:         now.Add(-time.Duration(windowHours) * time.Hour),
		WindowEnd:           now,
		CreatedAt:           now,
	}

	if err := s.repo.UpsertScore(ctx, rs); err != nil {
		return nil, fmt.Errorf("upserting score: %w", err)
	}

	return rs, nil
}

// TakeSnapshot records an hourly reliability snapshot.
func (s *Service) TakeSnapshot(ctx context.Context, tenantID, endpointID string) (*ScoreSnapshot, error) {
	score, err := s.ComputeScore(ctx, tenantID, endpointID, 1)
	if err != nil {
		return nil, fmt.Errorf("computing score for snapshot: %w", err)
	}

	snapshot := &ScoreSnapshot{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		EndpointID:   endpointID,
		Score:        score.Score,
		SuccessRate:  score.SuccessRate,
		LatencyP50Ms: score.LatencyP50Ms,
		LatencyP95Ms: score.LatencyP95Ms,
		LatencyP99Ms: score.LatencyP99Ms,
		SnapshotAt:   time.Now().UTC(),
		CreatedAt:    time.Now().UTC(),
	}

	if err := s.repo.CreateSnapshot(ctx, snapshot); err != nil {
		return nil, fmt.Errorf("creating snapshot: %w", err)
	}

	return snapshot, nil
}

// SetSLA creates or updates SLA targets for an endpoint.
func (s *Service) SetSLA(ctx context.Context, tenantID, endpointID string, req *CreateSLARequest) (*SLATarget, error) {
	sla := &SLATarget{
		ID:              uuid.New().String(),
		TenantID:        tenantID,
		EndpointID:      endpointID,
		TargetScore:     req.TargetScore,
		TargetUptime:    req.TargetUptime,
		MaxLatencyP95Ms: req.MaxLatencyP95Ms,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	if err := s.repo.UpsertSLA(ctx, sla); err != nil {
		return nil, fmt.Errorf("upserting SLA: %w", err)
	}

	return sla, nil
}

// SetAlertThreshold creates or updates alert thresholds for an endpoint.
func (s *Service) SetAlertThreshold(ctx context.Context, tenantID, endpointID string, req *CreateAlertThresholdRequest) (*AlertThreshold, error) {
	threshold := &AlertThreshold{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		EndpointID:   endpointID,
		MinScore:     req.MinScore,
		MaxLatencyMs: req.MaxLatencyMs,
		MaxFailures:  req.MaxFailures,
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := s.repo.UpsertAlertThreshold(ctx, threshold); err != nil {
		return nil, fmt.Errorf("upserting alert threshold: %w", err)
	}

	return threshold, nil
}

// CheckAlerts evaluates alert thresholds against current scores and returns true if violated.
func (s *Service) CheckAlerts(ctx context.Context, tenantID, endpointID string) (bool, error) {
	threshold, err := s.repo.GetAlertThreshold(ctx, tenantID, endpointID)
	if err != nil || threshold == nil || !threshold.IsActive {
		return false, err
	}

	score, err := s.repo.GetScore(ctx, tenantID, endpointID)
	if err != nil || score == nil {
		return false, err
	}

	violated := score.Score < threshold.MinScore ||
		score.LatencyP95Ms > threshold.MaxLatencyMs ||
		score.ConsecutiveFailures > threshold.MaxFailures

	return violated, nil
}

// computeReliabilityScore calculates 0-100 score from delivery stats.
// Weights: success rate 50%, latency 25%, consecutive failures 25%.
func computeReliabilityScore(stats *DeliveryStats) float64 {
	if stats.TotalAttempts == 0 {
		return 100.0 // No data = assumed healthy
	}

	successRate := safePercent(stats.SuccessfulAttempts, stats.TotalAttempts)

	// Latency score: full marks under 500ms p95, degrades up to 10000ms
	latencyScore := 100.0
	if stats.LatencyP95Ms > 500 {
		latencyScore = math.Max(0, 100.0-float64(stats.LatencyP95Ms-500)/95.0)
	}

	// Consecutive failure penalty: each failure costs 10 points
	failurePenalty := math.Min(100.0, float64(stats.ConsecutiveFailures)*10.0)
	failureScore := 100.0 - failurePenalty

	score := successRate*0.50 + latencyScore*0.25 + failureScore*0.25
	return math.Round(score*100) / 100
}

func scoreToStatus(score float64) ScoreStatus {
	switch {
	case score >= 95:
		return ScoreStatusHealthy
	case score >= 70:
		return ScoreStatusDegraded
	case score > 0:
		return ScoreStatusCritical
	default:
		return ScoreStatusUnknown
	}
}

func safePercent(numerator, denominator int) float64 {
	if denominator == 0 {
		return 100.0
	}
	return float64(numerator) / float64(denominator) * 100.0
}
