package chaos

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Service provides chaos engineering functionality
type Service struct {
	repo       Repository
	agents     sync.Map // map[experimentID]*Agent
	scheduler  *Scheduler
	config     *ServiceConfig
}

// ServiceConfig holds service configuration
type ServiceConfig struct {
	MaxConcurrentExperiments int
	DefaultBlastRadius       BlastRadius
	SafetyChecksEnabled      bool
}

// DefaultServiceConfig returns default configuration
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		MaxConcurrentExperiments: 5,
		DefaultBlastRadius: BlastRadius{
			MaxAffectedEndpoints:   10,
			MaxAffectedDeliveries:  1000,
			MaxErrorRate:           0.5,
			AutoRollbackThreshold:  0.3,
		},
		SafetyChecksEnabled: true,
	}
}

// NewService creates a new chaos service
func NewService(repo Repository, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}
	
	svc := &Service{
		repo:   repo,
		config: config,
	}
	svc.scheduler = NewScheduler(svc)
	
	return svc
}

// CreateExperiment creates a new chaos experiment
func (s *Service) CreateExperiment(ctx context.Context, tenantID, createdBy string, req *CreateExperimentRequest) (*ChaosExperiment, error) {
	exp := &ChaosExperiment{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		Name:         req.Name,
		Description:  req.Description,
		Type:         req.Type,
		Status:       StatusPending,
		TargetConfig: req.TargetConfig,
		FaultConfig:  req.FaultConfig,
		Schedule:     req.Schedule,
		BlastRadius:  req.BlastRadius,
		Duration:     req.Duration,
		CreatedBy:    createdBy,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Apply default blast radius if not specified
	if exp.BlastRadius.MaxAffectedEndpoints == 0 {
		exp.BlastRadius = s.config.DefaultBlastRadius
	}

	// If scheduled, set status
	if exp.Schedule != nil && !exp.Schedule.StartTime.IsZero() {
		exp.Status = StatusScheduled
	}

	if err := s.repo.SaveExperiment(ctx, exp); err != nil {
		return nil, err
	}

	// Schedule if needed
	if exp.Status == StatusScheduled {
		s.scheduler.Schedule(exp)
	}

	return exp, nil
}

// GetExperiment retrieves an experiment
func (s *Service) GetExperiment(ctx context.Context, tenantID, expID string) (*ChaosExperiment, error) {
	return s.repo.GetExperiment(ctx, tenantID, expID)
}

// ListExperiments lists experiments
func (s *Service) ListExperiments(ctx context.Context, tenantID string, status *ExperimentStatus, limit, offset int) ([]ChaosExperiment, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListExperiments(ctx, tenantID, status, limit, offset)
}

// StartExperiment starts an experiment
func (s *Service) StartExperiment(ctx context.Context, tenantID, expID string) (*ChaosExperiment, error) {
	exp, err := s.repo.GetExperiment(ctx, tenantID, expID)
	if err != nil {
		return nil, err
	}

	if exp.Status != StatusPending && exp.Status != StatusScheduled {
		return nil, fmt.Errorf("experiment cannot be started from status: %s", exp.Status)
	}

	// Check concurrent experiment limit
	runningCount := 0
	s.agents.Range(func(_, _ interface{}) bool {
		runningCount++
		return true
	})

	if runningCount >= s.config.MaxConcurrentExperiments {
		return nil, fmt.Errorf("maximum concurrent experiments reached: %d", s.config.MaxConcurrentExperiments)
	}

	// Start the agent
	agent := NewAgent(exp, s.repo)
	s.agents.Store(exp.ID, agent)

	now := time.Now()
	exp.Status = StatusRunning
	exp.StartedAt = &now
	exp.UpdatedAt = now

	if err := s.repo.SaveExperiment(ctx, exp); err != nil {
		s.agents.Delete(exp.ID)
		return nil, err
	}

	// Run experiment in background
	go func() {
		agent.Run(context.Background())
		s.completeExperiment(context.Background(), exp.TenantID, exp.ID)
	}()

	return exp, nil
}

// StopExperiment stops a running experiment
func (s *Service) StopExperiment(ctx context.Context, tenantID, expID string) (*ChaosExperiment, error) {
	exp, err := s.repo.GetExperiment(ctx, tenantID, expID)
	if err != nil {
		return nil, err
	}

	if exp.Status != StatusRunning {
		return nil, fmt.Errorf("experiment is not running")
	}

	// Stop the agent
	if agentI, ok := s.agents.Load(exp.ID); ok {
		agent := agentI.(*Agent)
		agent.Stop()
	}

	now := time.Now()
	exp.Status = StatusAborted
	exp.CompletedAt = &now
	exp.UpdatedAt = now

	if err := s.repo.SaveExperiment(ctx, exp); err != nil {
		return nil, err
	}

	s.agents.Delete(exp.ID)

	return exp, nil
}

// DeleteExperiment deletes an experiment
func (s *Service) DeleteExperiment(ctx context.Context, tenantID, expID string) error {
	exp, err := s.repo.GetExperiment(ctx, tenantID, expID)
	if err != nil {
		return err
	}

	if exp.Status == StatusRunning {
		return fmt.Errorf("cannot delete running experiment")
	}

	return s.repo.DeleteExperiment(ctx, tenantID, expID)
}

// ShouldInjectFault checks if a delivery should have fault injected
func (s *Service) ShouldInjectFault(ctx context.Context, tenantID, endpointID, deliveryID string) (*FaultInjection, error) {
	var injection *FaultInjection

	s.agents.Range(func(key, value interface{}) bool {
		agent := value.(*Agent)
		if agent.exp.TenantID != tenantID {
			return true
		}

		if agent.shouldAffect(endpointID) {
			injection = agent.getFaultInjection(endpointID, deliveryID)
			return false
		}
		return true
	})

	return injection, nil
}

// RecordFaultResult records the result of a fault injection
func (s *Service) RecordFaultResult(ctx context.Context, tenantID, expID, endpointID, deliveryID string, recovered bool, recoveryTimeMs int64) error {
	event := &ChaosEvent{
		ID:           uuid.New().String(),
		ExperimentID: expID,
		TenantID:     tenantID,
		EndpointID:   endpointID,
		DeliveryID:   deliveryID,
		EventType:    "fault_result",
		Recovered:    recovered,
		RecoveryTime: recoveryTimeMs,
		Timestamp:    time.Now(),
	}

	return s.repo.SaveEvent(ctx, event)
}

// GetResilienceReport generates a resilience report
func (s *Service) GetResilienceReport(ctx context.Context, tenantID string, start, end time.Time) (*ResilienceReport, error) {
	report := &ResilienceReport{
		TenantID:    tenantID,
		GeneratedAt: time.Now(),
		Period:      fmt.Sprintf("%s to %s", start.Format(time.RFC3339), end.Format(time.RFC3339)),
	}

	// Get experiment stats
	stats, err := s.repo.GetExperimentStats(ctx, tenantID, start, end)
	if err != nil {
		return nil, err
	}

	// Get completed experiments
	completed := StatusCompleted
	experiments, count, err := s.repo.ListExperiments(ctx, tenantID, &completed, 100, 0)
	if err != nil {
		return nil, err
	}
	report.ExperimentCount = count

	// Calculate overall score from experiment results
	var totalScore float64
	var scoredCount int
	endpointScores := make(map[string][]float64)

	for _, exp := range experiments {
		if exp.Results != nil && exp.CompletedAt != nil &&
			exp.CompletedAt.After(start) && exp.CompletedAt.Before(end) {
			totalScore += exp.Results.ResilienceScore
			scoredCount++

			for _, er := range exp.Results.ByEndpoint {
				endpointScores[er.EndpointID] = append(endpointScores[er.EndpointID], er.ResilienceScore)
			}
		}
	}

	if scoredCount > 0 {
		report.OverallScore = totalScore / float64(scoredCount)
	}

	// Assign grade
	report.Grade = scoreToGrade(report.OverallScore)

	// Build category scores
	report.ByCategory = []CategoryScore{
		{Category: "Error Handling", Score: report.OverallScore * 0.9, MaxScore: 100, Description: "Ability to handle server errors"},
		{Category: "Timeout Handling", Score: report.OverallScore * 0.85, MaxScore: 100, Description: "Ability to handle timeouts"},
		{Category: "Rate Limit Handling", Score: report.OverallScore * 0.95, MaxScore: 100, Description: "Ability to handle rate limits"},
		{Category: "Network Resilience", Score: report.OverallScore * 0.8, MaxScore: 100, Description: "Ability to handle network issues"},
	}

	// Build endpoint scores
	for epID, scores := range endpointScores {
		var avg float64
		for _, s := range scores {
			avg += s
		}
		avg /= float64(len(scores))
		report.ByEndpoint = append(report.ByEndpoint, EndpointScore{
			EndpointID:  epID,
			Score:       avg,
			Experiments: len(scores),
			LastTested:  time.Now().Add(-24 * time.Hour), // Approximate
		})
	}

	// Add recommendations
	if report.OverallScore < 50 {
		report.Recommendations = append(report.Recommendations, Recommendation{
			Priority:    "high",
			Category:    "general",
			Title:       "Implement Retry Logic",
			Description: "Many endpoints show poor recovery from failures. Implement exponential backoff retry.",
			Impact:      "Could improve resilience score by 20-30 points",
		})
	}

	if avgRecovery, ok := stats["avg_recovery_ms"].(float64); ok && avgRecovery > 5000 {
		report.Recommendations = append(report.Recommendations, Recommendation{
			Priority:    "medium",
			Category:    "performance",
			Title:       "Reduce Recovery Time",
			Description: "Average recovery time is high. Consider reducing retry delays.",
			Impact:      "Could reduce mean time to recovery by 50%",
		})
	}

	// Add strengths/weaknesses
	if report.OverallScore >= 70 {
		report.Strengths = append(report.Strengths, "Good overall resilience to failures")
	}
	if report.OverallScore < 50 {
		report.Weaknesses = append(report.Weaknesses, "Low overall resilience score")
	}

	return report, nil
}

// GetTemplates returns experiment templates
func (s *Service) GetTemplates() []ExperimentTemplate {
	return GetTemplates()
}

// GetEvents retrieves chaos events for an experiment
func (s *Service) GetEvents(ctx context.Context, tenantID, expID string, limit int) ([]ChaosEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.repo.GetEvents(ctx, tenantID, expID, limit)
}

func (s *Service) completeExperiment(ctx context.Context, tenantID, expID string) {
	exp, err := s.repo.GetExperiment(ctx, tenantID, expID)
	if err != nil {
		return
	}

	now := time.Now()
	exp.Status = StatusCompleted
	exp.CompletedAt = &now
	exp.UpdatedAt = now

	// Calculate results
	events, _ := s.repo.GetEvents(ctx, tenantID, expID, 10000)
	exp.Results = calculateResults(events)

	s.repo.SaveExperiment(ctx, exp)
	s.agents.Delete(expID)
}

func calculateResults(events []ChaosEvent) *ExperimentResult {
	result := &ExperimentResult{
		ByEndpoint: make([]EndpointResult, 0),
	}

	endpointStats := make(map[string]*EndpointResult)
	var totalRecoveryTime int64
	var recoveryCount int64

	for _, event := range events {
		result.TotalDeliveries++
		if event.InjectedFault != "" {
			result.AffectedDeliveries++
		}

		if event.Recovered {
			result.SuccessfulRecovery++
			totalRecoveryTime += event.RecoveryTime
			recoveryCount++
		} else if event.EventType == "fault_result" {
			result.FailedRecovery++
		}

		// Per-endpoint stats
		if _, ok := endpointStats[event.EndpointID]; !ok {
			endpointStats[event.EndpointID] = &EndpointResult{EndpointID: event.EndpointID}
		}
		es := endpointStats[event.EndpointID]
		es.Affected++
		if event.Recovered {
			es.Recovered++
			es.AvgRecoveryTime = (es.AvgRecoveryTime*float64(es.Recovered-1) + float64(event.RecoveryTime)) / float64(es.Recovered)
		} else if event.EventType == "fault_result" {
			es.FailedPermanent++
		}
	}

	if recoveryCount > 0 {
		result.AvgRecoveryTimeMs = float64(totalRecoveryTime) / float64(recoveryCount)
	}

	// Calculate resilience score
	if result.AffectedDeliveries > 0 {
		recoveryRate := float64(result.SuccessfulRecovery) / float64(result.AffectedDeliveries)
		result.ResilienceScore = recoveryRate * 100
	}

	// Build endpoint results
	for _, es := range endpointStats {
		if es.Affected > 0 {
			es.RecoveryRate = float64(es.Recovered) / float64(es.Affected)
			es.ResilienceScore = es.RecoveryRate * 100
		}
		result.ByEndpoint = append(result.ByEndpoint, *es)
	}

	// Add observations
	if result.ResilienceScore >= 90 {
		result.Observations = append(result.Observations, "Excellent recovery from injected faults")
	} else if result.ResilienceScore >= 70 {
		result.Observations = append(result.Observations, "Good recovery with room for improvement")
	} else {
		result.Observations = append(result.Observations, "Significant recovery issues detected")
	}

	// Add recommendations
	if result.ResilienceScore < 70 {
		result.Recommendations = append(result.Recommendations, "Implement or improve retry logic")
	}
	if result.AvgRecoveryTimeMs > 10000 {
		result.Recommendations = append(result.Recommendations, "Consider reducing retry delays")
	}

	return result
}

func scoreToGrade(score float64) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 80:
		return "B"
	case score >= 70:
		return "C"
	case score >= 60:
		return "D"
	default:
		return "F"
	}
}

// FaultInjection represents an injected fault
type FaultInjection struct {
	ExperimentID string
	Type         ExperimentType
	LatencyMs    int
	ErrorCode    int
	ErrorMessage string
	ShouldDrop   bool
}

// Agent executes chaos experiments
type Agent struct {
	exp           *ChaosExperiment
	repo          Repository
	stopCh        chan struct{}
	running       bool
	mu            sync.Mutex
	affectedCount int64
}

// NewAgent creates a new chaos agent
func NewAgent(exp *ChaosExperiment, repo Repository) *Agent {
	return &Agent{
		exp:    exp,
		repo:   repo,
		stopCh: make(chan struct{}),
	}
}

// Run executes the experiment
func (a *Agent) Run(ctx context.Context) {
	a.mu.Lock()
	a.running = true
	a.mu.Unlock()

	timer := time.NewTimer(time.Duration(a.exp.Duration) * time.Second)
	defer timer.Stop()

	select {
	case <-timer.C:
		// Experiment completed naturally
	case <-a.stopCh:
		// Experiment was stopped
	case <-ctx.Done():
		// Context cancelled
	}

	a.mu.Lock()
	a.running = false
	a.mu.Unlock()
}

// Stop stops the agent
func (a *Agent) Stop() {
	close(a.stopCh)
}

func (a *Agent) shouldAffect(endpointID string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running {
		return false
	}

	// Check blast radius
	if a.affectedCount >= a.exp.BlastRadius.MaxAffectedDeliveries {
		return false
	}

	// Check if endpoint is targeted
	if len(a.exp.TargetConfig.EndpointIDs) > 0 {
		found := false
		for _, id := range a.exp.TargetConfig.EndpointIDs {
			if id == endpointID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Apply percentage
	if a.exp.TargetConfig.Percentage > 0 && a.exp.TargetConfig.Percentage < 100 {
		if rand.Float64()*100 > a.exp.TargetConfig.Percentage {
			return false
		}
	}

	return true
}

func (a *Agent) getFaultInjection(endpointID, deliveryID string) *FaultInjection {
	a.mu.Lock()
	a.affectedCount++
	a.mu.Unlock()

	injection := &FaultInjection{
		ExperimentID: a.exp.ID,
		Type:         a.exp.Type,
	}

	switch a.exp.Type {
	case ExperimentLatency:
		injection.LatencyMs = a.exp.FaultConfig.LatencyMs
		if a.exp.FaultConfig.LatencyJitter > 0 {
			injection.LatencyMs += rand.IntN(a.exp.FaultConfig.LatencyJitter)
		}
	case ExperimentError:
		if rand.Float64() < a.exp.FaultConfig.ErrorRate {
			injection.ErrorCode = a.exp.FaultConfig.ErrorCode
			injection.ErrorMessage = a.exp.FaultConfig.ErrorMessage
		}
	case ExperimentTimeout:
		injection.LatencyMs = a.exp.FaultConfig.TimeoutMs
	case ExperimentRateLimit:
		injection.ErrorCode = a.exp.FaultConfig.RateLimitCode
		injection.ErrorMessage = fmt.Sprintf("Rate limit exceeded. Retry after %d seconds", a.exp.FaultConfig.RetryAfterSec)
	case ExperimentBlackhole:
		injection.ShouldDrop = true
	case ExperimentPacketLoss:
		if rand.Float64() < a.exp.FaultConfig.PacketLossRate {
			injection.ShouldDrop = true
		}
	}

	// Record event
	event := &ChaosEvent{
		ID:            uuid.New().String(),
		ExperimentID:  a.exp.ID,
		TenantID:      a.exp.TenantID,
		EndpointID:    endpointID,
		DeliveryID:    deliveryID,
		EventType:     "fault_injection",
		InjectedFault: string(a.exp.Type),
		Timestamp:     time.Now(),
	}
	a.repo.SaveEvent(context.Background(), event)

	return injection
}

// Scheduler handles scheduled experiments
type Scheduler struct {
	svc       *Service
	scheduled sync.Map
}

// NewScheduler creates a new scheduler
func NewScheduler(svc *Service) *Scheduler {
	return &Scheduler{svc: svc}
}

// Schedule schedules an experiment
func (s *Scheduler) Schedule(exp *ChaosExperiment) {
	if exp.Schedule == nil || exp.Schedule.StartTime.IsZero() {
		return
	}

	delay := time.Until(exp.Schedule.StartTime)
	if delay <= 0 {
		// Start immediately
		go s.svc.StartExperiment(context.Background(), exp.TenantID, exp.ID)
		return
	}

	timer := time.AfterFunc(delay, func() {
		s.svc.StartExperiment(context.Background(), exp.TenantID, exp.ID)
		s.scheduled.Delete(exp.ID)
	})

	s.scheduled.Store(exp.ID, timer)
}

// Cancel cancels a scheduled experiment
func (s *Scheduler) Cancel(expID string) {
	if timerI, ok := s.scheduled.Load(expID); ok {
		timer := timerI.(*time.Timer)
		timer.Stop()
		s.scheduled.Delete(expID)
	}
}
