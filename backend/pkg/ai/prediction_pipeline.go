package ai

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/google/uuid"
)

// PipelineStatus represents the status of a prediction pipeline
type PipelineStatus string

const (
	PipelineStatusActive   PipelineStatus = "active"
	PipelineStatusPaused   PipelineStatus = "paused"
	PipelineStatusTraining PipelineStatus = "training"
	PipelineStatusError    PipelineStatus = "error"
)

// PredictionSeverity indicates prediction urgency
type PredictionSeverity string

const (
	PredictionSeverityCritical PredictionSeverity = "critical"
	PredictionSeverityHigh     PredictionSeverity = "high"
	PredictionSeverityMedium   PredictionSeverity = "medium"
	PredictionSeverityLow      PredictionSeverity = "low"
)

// UnifiedPrediction represents a unified failure prediction across all AI subsystems
type UnifiedPrediction struct {
	ID                string             `json:"id"`
	TenantID          string             `json:"tenant_id"`
	EndpointID        string             `json:"endpoint_id"`
	PredictionType    string             `json:"prediction_type"` // failure, latency, error_rate, capacity
	Severity          PredictionSeverity `json:"severity"`
	Probability       float64            `json:"probability"` // 0-1
	Confidence        float64            `json:"confidence"`  // 0-1
	TimeWindow        time.Duration      `json:"time_window"`
	Factors           []PredictionFactor `json:"factors"`
	Recommendation    string             `json:"recommendation"`
	AutoRemediate     bool               `json:"auto_remediate"`
	RemediationAction *RemediationAction `json:"remediation_action,omitempty"`
	Status            string             `json:"status"` // active, acknowledged, resolved, false_positive
	CreatedAt         time.Time          `json:"created_at"`
	ExpiresAt         time.Time          `json:"expires_at"`
	ResolvedAt        *time.Time         `json:"resolved_at,omitempty"`
}

// PredictionFactor explains what contributed to the prediction
type PredictionFactor struct {
	Name      string  `json:"name"`
	Value     float64 `json:"value"`
	Weight    float64 `json:"weight"`
	Direction string  `json:"direction"` // increasing, decreasing, stable
	Threshold float64 `json:"threshold,omitempty"`
}

// RemediationAction represents an automated response to a prediction
type RemediationAction struct {
	ID           string                 `json:"id"`
	PredictionID string                 `json:"prediction_id"`
	TenantID     string                 `json:"tenant_id"`
	ActionType   string                 `json:"action_type"` // switch_endpoint, adjust_retry, rate_limit, alert, disable
	Description  string                 `json:"description"`
	Config       map[string]interface{} `json:"config"`
	Status       string                 `json:"status"` // pending, executing, completed, failed, reverted
	Result       string                 `json:"result,omitempty"`
	ExecutedAt   *time.Time             `json:"executed_at,omitempty"`
	RevertedAt   *time.Time             `json:"reverted_at,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

// RootCauseReport provides detailed analysis of failures
type RootCauseReport struct {
	ID              string               `json:"id"`
	TenantID        string               `json:"tenant_id"`
	EndpointID      string               `json:"endpoint_id"`
	Title           string               `json:"title"`
	Summary         string               `json:"summary"`
	RootCause       string               `json:"root_cause"`
	Impact          *ImpactAssessment    `json:"impact"`
	Timeline        []TimelineEntry      `json:"timeline"`
	Contributing    []ContributingFactor `json:"contributing_factors"`
	Recommendations []string             `json:"recommendations"`
	Status          string               `json:"status"` // generating, complete, reviewed
	GeneratedAt     time.Time            `json:"generated_at"`
}

// ImpactAssessment quantifies the impact of failures
type ImpactAssessment struct {
	AffectedDeliveries int64   `json:"affected_deliveries"`
	FailureRate        float64 `json:"failure_rate"`
	AvgLatencyIncrease float64 `json:"avg_latency_increase_ms"`
	Duration           string  `json:"duration"`
	SeverityScore      float64 `json:"severity_score"` // 0-10
}

// TimelineEntry represents an event in the failure timeline
type TimelineEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	Event       string    `json:"event"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"`
}

// ContributingFactor explains factors that led to the failure
type ContributingFactor struct {
	Factor       string  `json:"factor"`
	Description  string  `json:"description"`
	Contribution float64 `json:"contribution"` // 0-1 weight
}

// PredictionPipelineConfig configures the unified prediction pipeline
type PredictionPipelineConfig struct {
	ID                     string             `json:"id"`
	TenantID               string             `json:"tenant_id"`
	Enabled                bool               `json:"enabled"`
	EvaluationIntervalSec  int                `json:"evaluation_interval_sec"`
	PredictionWindowMin    int                `json:"prediction_window_min"`
	AutoRemediationEnabled bool               `json:"auto_remediation_enabled"`
	SeverityThresholds     SeverityThresholds `json:"severity_thresholds"`
	Status                 PipelineStatus     `json:"status"`
	CreatedAt              time.Time          `json:"created_at"`
	UpdatedAt              time.Time          `json:"updated_at"`
}

// SeverityThresholds defines probability thresholds for severity levels
type SeverityThresholds struct {
	Critical float64 `json:"critical"` // >= this is critical
	High     float64 `json:"high"`
	Medium   float64 `json:"medium"`
	Low      float64 `json:"low"`
}

// DefaultSeverityThresholds returns default thresholds
func DefaultSeverityThresholds() SeverityThresholds {
	return SeverityThresholds{
		Critical: 0.9,
		High:     0.7,
		Medium:   0.5,
		Low:      0.3,
	}
}

// IntelligenceDashboard provides a unified view of all AI subsystems
type IntelligenceDashboard struct {
	TenantID           string              `json:"tenant_id"`
	ActivePredictions  []UnifiedPrediction `json:"active_predictions"`
	RecentRemediations []RemediationAction `json:"recent_remediations"`
	RecentReports      []RootCauseReport   `json:"recent_reports"`
	PipelineStatus     PipelineStatus      `json:"pipeline_status"`
	PredictionAccuracy float64             `json:"prediction_accuracy"`
	RemediationSuccess float64             `json:"remediation_success_rate"`
	TotalPredictions   int64               `json:"total_predictions"`
	TotalRemediations  int64               `json:"total_remediations"`
	GeneratedAt        time.Time           `json:"generated_at"`
}

// PredictionPipelineRepository defines storage for the unified pipeline
type PredictionPipelineRepository interface {
	SavePrediction(ctx context.Context, prediction *UnifiedPrediction) error
	GetPrediction(ctx context.Context, tenantID, predictionID string) (*UnifiedPrediction, error)
	ListActivePredictions(ctx context.Context, tenantID string) ([]UnifiedPrediction, error)
	UpdatePredictionStatus(ctx context.Context, predictionID, status string) error
	SaveRemediationAction(ctx context.Context, action *RemediationAction) error
	GetRemediationAction(ctx context.Context, actionID string) (*RemediationAction, error)
	ListRemediationActions(ctx context.Context, tenantID string, limit int) ([]RemediationAction, error)
	UpdateRemediationStatus(ctx context.Context, actionID, status, result string) error
	SaveRootCauseReport(ctx context.Context, report *RootCauseReport) error
	GetRootCauseReport(ctx context.Context, tenantID, reportID string) (*RootCauseReport, error)
	ListRootCauseReports(ctx context.Context, tenantID string, limit int) ([]RootCauseReport, error)
	SavePipelineConfig(ctx context.Context, config *PredictionPipelineConfig) error
	GetPipelineConfig(ctx context.Context, tenantID string) (*PredictionPipelineConfig, error)
	GetPredictionStats(ctx context.Context, tenantID string) (total int64, accuracy float64, err error)
	GetRemediationStats(ctx context.Context, tenantID string) (total int64, successRate float64, err error)
}

// PredictionPipeline is the unified prediction and remediation engine
type PredictionPipeline struct {
	repo       PredictionPipelineRepository
	aiService  *Service
	classifier *Classifier
}

// NewPredictionPipeline creates a new unified prediction pipeline
func NewPredictionPipeline(repo PredictionPipelineRepository, aiService *Service) *PredictionPipeline {
	return &PredictionPipeline{
		repo:       repo,
		aiService:  aiService,
		classifier: NewClassifier(),
	}
}

// EvaluateEndpoint runs the prediction pipeline for an endpoint
func (p *PredictionPipeline) EvaluateEndpoint(ctx context.Context, tenantID, endpointID string, recentFailures []DeliveryContext) (*UnifiedPrediction, error) {
	config, err := p.repo.GetPipelineConfig(ctx, tenantID)
	if err != nil || !config.Enabled {
		return nil, nil
	}

	// Classify recent failures to detect patterns
	var factors []PredictionFactor
	categoryCounts := make(map[ErrorCategory]int)
	var totalLatency int64

	for _, failure := range recentFailures {
		classification := p.classifier.Classify(failure.ErrorMessage, failure.HTTPStatus, failure.ResponseBody)
		categoryCounts[classification.Category]++
		totalLatency += failure.Latency
	}

	if len(recentFailures) == 0 {
		return nil, nil
	}

	// Calculate failure probability from patterns
	avgLatency := float64(totalLatency) / float64(len(recentFailures))
	failureRate := float64(len(recentFailures))

	// Build prediction factors
	for cat, count := range categoryCounts {
		factors = append(factors, PredictionFactor{
			Name:      string(cat) + "_errors",
			Value:     float64(count),
			Weight:    float64(count) / failureRate,
			Direction: "increasing",
		})
	}

	if avgLatency > 5000 {
		factors = append(factors, PredictionFactor{
			Name:      "high_latency",
			Value:     avgLatency,
			Weight:    0.3,
			Direction: "increasing",
			Threshold: 5000,
		})
	}

	// Sort factors by weight
	sort.Slice(factors, func(i, j int) bool {
		return factors[i].Weight > factors[j].Weight
	})

	// Calculate overall probability
	probability := calculateProbability(factors, failureRate)
	severity := classifySeverity(probability, config.SeverityThresholds)

	now := time.Now()
	prediction := &UnifiedPrediction{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		EndpointID:     endpointID,
		PredictionType: "failure",
		Severity:       severity,
		Probability:    probability,
		Confidence:     calculateConfidence(len(recentFailures)),
		TimeWindow:     time.Duration(config.PredictionWindowMin) * time.Minute,
		Factors:        factors,
		Recommendation: generateRecommendation(factors, severity),
		AutoRemediate:  config.AutoRemediationEnabled && severity == PredictionSeverityCritical,
		Status:         "active",
		CreatedAt:      now,
		ExpiresAt:      now.Add(time.Duration(config.PredictionWindowMin) * time.Minute),
	}

	if err := p.repo.SavePrediction(ctx, prediction); err != nil {
		return nil, fmt.Errorf("failed to save prediction: %w", err)
	}

	// Auto-remediate if configured
	if prediction.AutoRemediate {
		action := p.createRemediationAction(prediction, factors)
		if err := p.repo.SaveRemediationAction(ctx, action); err == nil {
			prediction.RemediationAction = action
		}
	}

	return prediction, nil
}

func calculateProbability(factors []PredictionFactor, failureRate float64) float64 {
	if len(factors) == 0 {
		return 0
	}

	weightedSum := 0.0
	totalWeight := 0.0
	for _, f := range factors {
		weightedSum += f.Weight * f.Value
		totalWeight += f.Weight
	}

	if totalWeight == 0 {
		return 0
	}

	// Normalize to 0-1 range using sigmoid
	raw := weightedSum / totalWeight
	probability := 1.0 / (1.0 + 1.0/(raw+0.01))
	if probability > 1 {
		probability = 1
	}
	return probability
}

func calculateConfidence(sampleSize int) float64 {
	if sampleSize <= 1 {
		return 0.1
	}
	if sampleSize >= 100 {
		return 0.95
	}
	return 0.1 + (float64(sampleSize)/100.0)*0.85
}

func classifySeverity(probability float64, thresholds SeverityThresholds) PredictionSeverity {
	switch {
	case probability >= thresholds.Critical:
		return PredictionSeverityCritical
	case probability >= thresholds.High:
		return PredictionSeverityHigh
	case probability >= thresholds.Medium:
		return PredictionSeverityMedium
	default:
		return PredictionSeverityLow
	}
}

func generateRecommendation(factors []PredictionFactor, severity PredictionSeverity) string {
	if len(factors) == 0 {
		return "Monitor endpoint for changes"
	}

	topFactor := factors[0].Name
	switch {
	case severity == PredictionSeverityCritical:
		return fmt.Sprintf("Critical: Immediately investigate %s pattern. Consider switching to backup endpoint.", topFactor)
	case severity == PredictionSeverityHigh:
		return fmt.Sprintf("High risk: %s trending upward. Adjust retry strategy or rate limits.", topFactor)
	default:
		return fmt.Sprintf("Monitor %s metric closely.", topFactor)
	}
}

func (p *PredictionPipeline) createRemediationAction(prediction *UnifiedPrediction, factors []PredictionFactor) *RemediationAction {
	actionType := "alert"
	description := "Alert sent for predicted failure"
	config := map[string]interface{}{"prediction_id": prediction.ID}

	if len(factors) > 0 {
		topFactor := factors[0].Name
		switch {
		case topFactor == "timeout_errors" || topFactor == "high_latency":
			actionType = "adjust_retry"
			description = "Increasing retry backoff due to latency issues"
			config["backoff_multiplier"] = 2.0
		case topFactor == "rate_limit_errors":
			actionType = "rate_limit"
			description = "Reducing send rate to avoid rate limiting"
			config["rate_reduction_percent"] = 50
		case topFactor == "authentication_errors":
			actionType = "alert"
			description = "Authentication failures detected - manual intervention required"
		default:
			actionType = "switch_endpoint"
			description = "Switching to backup endpoint"
		}
	}

	return &RemediationAction{
		ID:           uuid.New().String(),
		PredictionID: prediction.ID,
		TenantID:     prediction.TenantID,
		ActionType:   actionType,
		Description:  description,
		Config:       config,
		Status:       "pending",
		CreatedAt:    time.Now(),
	}
}

// GenerateRootCauseReport creates a root-cause analysis report
func (p *PredictionPipeline) GenerateRootCauseReport(ctx context.Context, tenantID, endpointID string, failures []DeliveryContext) (*RootCauseReport, error) {
	if len(failures) == 0 {
		return nil, fmt.Errorf("no failures to analyze")
	}

	// Classify all failures
	categoryCount := make(map[ErrorCategory]int)
	var timeline []TimelineEntry
	var totalLatency int64

	for _, f := range failures {
		classification := p.classifier.Classify(f.ErrorMessage, f.HTTPStatus, f.ResponseBody)
		categoryCount[classification.Category]++
		totalLatency += f.Latency

		timeline = append(timeline, TimelineEntry{
			Timestamp:   f.Timestamp,
			Event:       string(classification.Category),
			Description: f.ErrorMessage,
			Severity:    string(classification.Category),
		})
	}

	// Sort timeline
	sort.Slice(timeline, func(i, j int) bool {
		return timeline[i].Timestamp.Before(timeline[j].Timestamp)
	})

	// Find dominant category
	var rootCategory ErrorCategory
	maxCount := 0
	for cat, count := range categoryCount {
		if count > maxCount {
			maxCount = count
			rootCategory = cat
		}
	}

	// Build contributing factors
	var contributing []ContributingFactor
	for cat, count := range categoryCount {
		contributing = append(contributing, ContributingFactor{
			Factor:       string(cat),
			Description:  fmt.Sprintf("%d occurrences of %s errors", count, cat),
			Contribution: float64(count) / float64(len(failures)),
		})
	}
	sort.Slice(contributing, func(i, j int) bool {
		return contributing[i].Contribution > contributing[j].Contribution
	})

	avgLatency := float64(totalLatency) / float64(len(failures))
	duration := failures[len(failures)-1].Timestamp.Sub(failures[0].Timestamp)

	report := &RootCauseReport{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		EndpointID: endpointID,
		Title:      fmt.Sprintf("Root Cause Analysis: %s failures", rootCategory),
		Summary:    fmt.Sprintf("Analyzed %d failures over %s. Primary cause: %s (%d occurrences, %.0f%% of total).", len(failures), duration.String(), rootCategory, maxCount, float64(maxCount)/float64(len(failures))*100),
		RootCause:  fmt.Sprintf("Primary failure pattern is %s errors, accounting for %.0f%% of all failures.", rootCategory, float64(maxCount)/float64(len(failures))*100),
		Impact: &ImpactAssessment{
			AffectedDeliveries: int64(len(failures)),
			FailureRate:        1.0,
			AvgLatencyIncrease: avgLatency,
			Duration:           duration.String(),
			SeverityScore:      calculateSeverityScore(len(failures), avgLatency),
		},
		Timeline:        timeline,
		Contributing:    contributing,
		Recommendations: generateReportRecommendations(rootCategory, contributing),
		Status:          "complete",
		GeneratedAt:     time.Now(),
	}

	if err := p.repo.SaveRootCauseReport(ctx, report); err != nil {
		return nil, fmt.Errorf("failed to save report: %w", err)
	}

	return report, nil
}

func calculateSeverityScore(failureCount int, avgLatency float64) float64 {
	score := 0.0
	if failureCount > 100 {
		score += 4.0
	} else if failureCount > 10 {
		score += 2.0
	} else {
		score += 1.0
	}

	if avgLatency > 10000 {
		score += 3.0
	} else if avgLatency > 5000 {
		score += 2.0
	} else {
		score += 1.0
	}

	if score > 10 {
		score = 10
	}
	return score
}

func generateReportRecommendations(rootCause ErrorCategory, factors []ContributingFactor) []string {
	var recs []string

	switch rootCause {
	case CategoryNetwork:
		recs = append(recs, "Check network connectivity to endpoint", "Consider adding a CDN or proxy", "Verify DNS resolution is working correctly")
	case CategoryTimeout:
		recs = append(recs, "Increase timeout thresholds", "Implement exponential backoff", "Check endpoint server capacity")
	case CategoryAuth:
		recs = append(recs, "Verify API credentials are valid", "Check token expiration settings", "Ensure signing secrets are correct")
	case CategoryRateLimit:
		recs = append(recs, "Reduce delivery rate", "Implement request queuing", "Contact endpoint provider about rate limits")
	case CategoryServerError:
		recs = append(recs, "Contact endpoint provider about server issues", "Implement circuit breaker pattern", "Add fallback endpoint")
	default:
		recs = append(recs, "Review recent endpoint configuration changes", "Check payload format compatibility", "Monitor for recurring patterns")
	}

	if len(factors) > 1 {
		recs = append(recs, "Multiple failure patterns detected - consider comprehensive endpoint review")
	}

	return recs
}

// GetDashboard returns the unified intelligence dashboard
func (p *PredictionPipeline) GetDashboard(ctx context.Context, tenantID string) (*IntelligenceDashboard, error) {
	predictions, _ := p.repo.ListActivePredictions(ctx, tenantID)
	if predictions == nil {
		predictions = []UnifiedPrediction{}
	}

	remediations, _ := p.repo.ListRemediationActions(ctx, tenantID, 10)
	if remediations == nil {
		remediations = []RemediationAction{}
	}

	reports, _ := p.repo.ListRootCauseReports(ctx, tenantID, 5)
	if reports == nil {
		reports = []RootCauseReport{}
	}

	totalPredictions, accuracy, _ := p.repo.GetPredictionStats(ctx, tenantID)
	totalRemediations, successRate, _ := p.repo.GetRemediationStats(ctx, tenantID)

	pipelineStatus := PipelineStatusActive
	if config, err := p.repo.GetPipelineConfig(ctx, tenantID); err == nil {
		pipelineStatus = config.Status
	}

	return &IntelligenceDashboard{
		TenantID:           tenantID,
		ActivePredictions:  predictions,
		RecentRemediations: remediations,
		RecentReports:      reports,
		PipelineStatus:     pipelineStatus,
		PredictionAccuracy: accuracy,
		RemediationSuccess: successRate,
		TotalPredictions:   totalPredictions,
		TotalRemediations:  totalRemediations,
		GeneratedAt:        time.Now(),
	}, nil
}

// ListPredictions returns active predictions
func (p *PredictionPipeline) ListPredictions(ctx context.Context, tenantID string) ([]UnifiedPrediction, error) {
	return p.repo.ListActivePredictions(ctx, tenantID)
}

// AcknowledgePrediction acknowledges a prediction
func (p *PredictionPipeline) AcknowledgePrediction(ctx context.Context, tenantID, predictionID string) error {
	return p.repo.UpdatePredictionStatus(ctx, predictionID, "acknowledged")
}

// ExecuteRemediation executes a pending remediation action
func (p *PredictionPipeline) ExecuteRemediation(ctx context.Context, actionID string) error {
	now := time.Now()
	action, err := p.repo.GetRemediationAction(ctx, actionID)
	if err != nil {
		return fmt.Errorf("remediation action not found: %w", err)
	}

	if action.Status != "pending" {
		return fmt.Errorf("action is not in pending state: %s", action.Status)
	}

	action.Status = "executing"
	action.ExecutedAt = &now
	if err := p.repo.UpdateRemediationStatus(ctx, actionID, "executing", ""); err != nil {
		log.Printf("ERROR: failed to update remediation status: %v (action=%s)", err, actionID)
	}

	// Execute based on action type (in real implementation, this calls other services)
	result := fmt.Sprintf("Executed %s action at %s", action.ActionType, now.Format(time.RFC3339))
	return p.repo.UpdateRemediationStatus(ctx, actionID, "completed", result)
}
