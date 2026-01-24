package intelligence

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
)

// Service implements the AI-powered webhook intelligence engine
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// PredictFailure analyzes an endpoint and predicts failure probability
func (s *Service) PredictFailure(ctx context.Context, tenantID, endpointID string, features *FeatureVector) (*FailurePrediction, error) {
	if features == nil {
		return nil, fmt.Errorf("feature vector is required")
	}

	probability := s.calculateFailureProbability(features)
	risk := s.classifyRisk(probability)
	recommendation := s.generateRecommendation(features, probability)

	prediction := &FailurePrediction{
		ID:                 uuid.New().String(),
		TenantID:           tenantID,
		EndpointID:         endpointID,
		PredictionType:     PredictionFailure,
		FailureProbability: probability,
		RiskLevel:          risk,
		Confidence:         s.calculateConfidence(features),
		Reasons:            s.identifyRiskFactors(features),
		Recommendation:     recommendation,
		Features:           features,
	}

	if err := s.repo.SavePrediction(ctx, prediction); err != nil {
		return nil, fmt.Errorf("failed to save prediction: %w", err)
	}
	return prediction, nil
}

func (s *Service) calculateFailureProbability(f *FeatureVector) float64 {
	score := 0.0

	// Weighted feature contribution (simplified ML scoring)
	if f.FailureRate24h > 0.1 {
		score += f.FailureRate24h * 0.3
	}
	if f.ConsecutiveFailures > 3 {
		score += math.Min(float64(f.ConsecutiveFailures)*0.05, 0.25)
	}
	if f.P99LatencyMs > 5000 {
		score += 0.15
	}
	if f.ResponseTimetrend > 0.5 {
		score += f.ResponseTimetrend * 0.1
	}
	if f.SSLDaysRemaining > 0 && f.SSLDaysRemaining < 7 {
		score += 0.2
	}
	if f.LastSuccessAgo > 3600 {
		score += math.Min(float64(f.LastSuccessAgo)/86400*0.15, 0.15)
	}
	if f.ErrorDiversity > 3 {
		score += 0.1
	}

	return math.Min(score, 1.0)
}

func (s *Service) classifyRisk(probability float64) RiskLevel {
	switch {
	case probability >= 0.8:
		return RiskCritical
	case probability >= 0.5:
		return RiskHigh
	case probability >= 0.2:
		return RiskMedium
	default:
		return RiskLow
	}
}

func (s *Service) calculateConfidence(f *FeatureVector) float64 {
	dataRichness := math.Min(float64(f.EndpointAge)/86400/7, 1.0)
	activityLevel := math.Min(f.RequestsPerMinute/10, 1.0)
	return math.Min(0.5+dataRichness*0.3+activityLevel*0.2, 0.99)
}

func (s *Service) identifyRiskFactors(f *FeatureVector) []string {
	var reasons []string
	if f.FailureRate24h > 0.1 {
		reasons = append(reasons, fmt.Sprintf("High failure rate: %.1f%% in last 24h", f.FailureRate24h*100))
	}
	if f.ConsecutiveFailures > 3 {
		reasons = append(reasons, fmt.Sprintf("%d consecutive failures detected", f.ConsecutiveFailures))
	}
	if f.P99LatencyMs > 5000 {
		reasons = append(reasons, fmt.Sprintf("High P99 latency: %.0fms", f.P99LatencyMs))
	}
	if f.SSLDaysRemaining > 0 && f.SSLDaysRemaining < 7 {
		reasons = append(reasons, fmt.Sprintf("SSL certificate expires in %d days", f.SSLDaysRemaining))
	}
	if f.ResponseTimetrend > 0.5 {
		reasons = append(reasons, "Response time trending upward significantly")
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "Endpoint appears healthy")
	}
	return reasons
}

func (s *Service) generateRecommendation(f *FeatureVector, probability float64) string {
	if probability >= 0.8 {
		return "CRITICAL: Consider pausing deliveries and investigating endpoint health immediately."
	}
	if f.ConsecutiveFailures > 5 {
		return "Multiple consecutive failures. Consider increasing retry backoff or contacting the endpoint operator."
	}
	if f.P99LatencyMs > 10000 {
		return "Extremely high latency detected. Consider increasing timeout or switching to async delivery."
	}
	if f.SSLDaysRemaining > 0 && f.SSLDaysRemaining < 3 {
		return "SSL certificate expiring very soon. Alert the endpoint operator immediately."
	}
	if f.FailureRate24h > 0.2 {
		return "Elevated failure rate. Review error logs and consider adjusting retry policy."
	}
	return "No immediate action required. Continue monitoring."
}

// DetectAnomalies analyzes metrics to find unusual patterns
func (s *Service) DetectAnomalies(ctx context.Context, tenantID, endpointID string, features *FeatureVector) ([]AnomalyDetection, error) {
	var anomalies []AnomalyDetection

	// Latency anomaly detection (simplified z-score approach)
	if features.AvgLatencyMs > 0 && features.P99LatencyMs > features.AvgLatencyMs*5 {
		anomaly := AnomalyDetection{
			TenantID:    tenantID,
			EndpointID:  endpointID,
			AnomalyType: "latency_spike",
			Description: fmt.Sprintf("P99 latency (%.0fms) is %.1fx the average (%.0fms)", features.P99LatencyMs, features.P99LatencyMs/features.AvgLatencyMs, features.AvgLatencyMs),
			Severity:    RiskHigh,
			Score:       features.P99LatencyMs / features.AvgLatencyMs,
			Baseline:    features.AvgLatencyMs,
			Observed:    features.P99LatencyMs,
			Deviation:   features.P99LatencyMs - features.AvgLatencyMs,
		}
		if err := s.repo.SaveAnomaly(ctx, &anomaly); err == nil {
			anomalies = append(anomalies, anomaly)
		}
	}

	// Failure rate anomaly
	if features.FailureRate24h > features.FailureRate7d*3 && features.FailureRate7d > 0 {
		anomaly := AnomalyDetection{
			TenantID:    tenantID,
			EndpointID:  endpointID,
			AnomalyType: "failure_rate_spike",
			Description: fmt.Sprintf("24h failure rate (%.1f%%) is %.1fx the 7-day average (%.1f%%)", features.FailureRate24h*100, features.FailureRate24h/features.FailureRate7d, features.FailureRate7d*100),
			Severity:    RiskCritical,
			Score:       features.FailureRate24h / features.FailureRate7d,
			Baseline:    features.FailureRate7d,
			Observed:    features.FailureRate24h,
			Deviation:   features.FailureRate24h - features.FailureRate7d,
		}
		if err := s.repo.SaveAnomaly(ctx, &anomaly); err == nil {
			anomalies = append(anomalies, anomaly)
		}
	}

	// Traffic anomaly
	if features.RequestsPerMinute > 100 {
		anomaly := AnomalyDetection{
			TenantID:    tenantID,
			EndpointID:  endpointID,
			AnomalyType: "traffic_spike",
			Description: fmt.Sprintf("Unusually high request rate: %.0f req/min", features.RequestsPerMinute),
			Severity:    RiskMedium,
			Score:       features.RequestsPerMinute / 10,
			Observed:    features.RequestsPerMinute,
		}
		if err := s.repo.SaveAnomaly(ctx, &anomaly); err == nil {
			anomalies = append(anomalies, anomaly)
		}
	}

	return anomalies, nil
}

// OptimizeRetry suggests optimal retry configuration based on historical data
func (s *Service) OptimizeRetry(ctx context.Context, tenantID, endpointID string, currentRetries int, currentBackoff string, features *FeatureVector) (*RetryOptimization, error) {
	suggestedRetries := currentRetries
	suggestedBackoff := currentBackoff
	improvement := 0.0

	if features.FailureRate24h > 0.3 && currentRetries < 5 {
		suggestedRetries = 5
		improvement += 15.0
	}
	if features.P99LatencyMs > 5000 && currentBackoff == "linear" {
		suggestedBackoff = "exponential"
		improvement += 20.0
	}
	if features.ConsecutiveFailures > 10 && currentRetries > 3 {
		suggestedRetries = currentRetries - 1
		suggestedBackoff = "exponential_with_jitter"
		improvement += 10.0
	}
	if features.AvgLatencyMs < 100 && features.FailureRate7d < 0.01 {
		if currentRetries > 3 {
			suggestedRetries = 3
			improvement += 5.0
		}
	}

	rationale := "Based on delivery pattern analysis: "
	if improvement > 15 {
		rationale += "significant optimization opportunities found in retry strategy."
	} else if improvement > 5 {
		rationale += "minor improvements suggested for retry configuration."
	} else {
		rationale += "current configuration is near-optimal."
	}

	opt := &RetryOptimization{
		TenantID:             tenantID,
		EndpointID:           endpointID,
		CurrentMaxRetries:    currentRetries,
		SuggestedRetries:     suggestedRetries,
		CurrentBackoff:       currentBackoff,
		SuggestedBackoff:     suggestedBackoff,
		EstimatedImprovement: improvement,
		Rationale:            rationale,
		DataPoints:           int(features.EndpointAge / 3600),
	}

	if err := s.repo.SaveOptimization(ctx, opt); err != nil {
		return nil, fmt.Errorf("failed to save optimization: %w", err)
	}
	return opt, nil
}

// ClassifyEvent categorizes a webhook event using pattern matching
func (s *Service) ClassifyEvent(ctx context.Context, tenantID, webhookID, eventType string, payload map[string]any) (*EventClassification, error) {
	category, confidence := s.classifyPayload(eventType, payload)

	classification := &EventClassification{
		TenantID:   tenantID,
		WebhookID:  webhookID,
		EventType:  eventType,
		Category:   category,
		Confidence: confidence,
	}

	if err := s.repo.SaveClassification(ctx, classification); err != nil {
		return nil, fmt.Errorf("failed to save classification: %w", err)
	}
	return classification, nil
}

func (s *Service) classifyPayload(eventType string, payload map[string]any) (EventCategory, float64) {
	// Pattern-based classification (keyword matching heuristic)
	patterns := map[EventCategory][]string{
		CategoryPayment:       {"payment", "charge", "invoice", "refund", "subscription", "billing", "payout"},
		CategoryNotification:  {"notification", "alert", "message", "email", "sms", "push"},
		CategoryDataSync:      {"sync", "update", "create", "delete", "import", "export", "batch"},
		CategoryUserAction:    {"user", "login", "signup", "profile", "session", "click"},
		CategorySystemEvent:   {"system", "deploy", "health", "status", "error", "log"},
		CategorySecurityAlert: {"security", "breach", "suspicious", "blocked", "threat", "vulnerability"},
	}

	for category, keywords := range patterns {
		for _, keyword := range keywords {
			if containsIgnoreCase(eventType, keyword) {
				return category, 0.85
			}
		}
	}

	// Check payload keys
	for category, keywords := range patterns {
		for key := range payload {
			for _, keyword := range keywords {
				if containsIgnoreCase(key, keyword) {
					return category, 0.65
				}
			}
		}
	}

	return CategoryUnknown, 0.3
}

func containsIgnoreCase(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			c1, c2 := s[i+j], substr[j]
			if c1 >= 'A' && c1 <= 'Z' {
				c1 += 32
			}
			if c2 >= 'A' && c2 <= 'Z' {
				c2 += 32
			}
			if c1 != c2 {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// CalculateHealthScore computes an endpoint's overall health score
func (s *Service) CalculateHealthScore(ctx context.Context, tenantID, endpointID string, features *FeatureVector) (*EndpointHealthScore, error) {
	reliability := math.Max(0, 1.0-features.FailureRate24h) * 100
	latency := math.Max(0, 100-features.AvgLatencyMs/100)
	throughput := math.Min(features.RequestsPerMinute/10*100, 100)
	errorRate := math.Max(0, 1.0-features.FailureRate7d) * 100

	overall := reliability*0.4 + latency*0.25 + throughput*0.15 + errorRate*0.2

	trend := "stable"
	if features.ResponseTimetrend > 0.3 {
		trend = "degrading"
	} else if features.ResponseTimetrend < -0.3 {
		trend = "improving"
	}

	predicted := overall
	if trend == "degrading" {
		predicted = math.Max(0, overall-10)
	} else if trend == "improving" {
		predicted = math.Min(100, overall+5)
	}

	score := &EndpointHealthScore{
		EndpointID:        endpointID,
		TenantID:          tenantID,
		OverallScore:      math.Round(overall*10) / 10,
		ReliabilityScore:  math.Round(reliability*10) / 10,
		LatencyScore:      math.Round(latency*10) / 10,
		ThroughputScore:   math.Round(throughput*10) / 10,
		ErrorRateScore:    math.Round(errorRate*10) / 10,
		TrendDirection:    trend,
		PredictedScore24h: math.Round(predicted*10) / 10,
		CalculatedAt:      time.Now(),
	}

	if err := s.repo.SaveHealthScore(ctx, score); err != nil {
		return nil, fmt.Errorf("failed to save health score: %w", err)
	}
	return score, nil
}

// GetDashboard returns all intelligence data for the dashboard
func (s *Service) GetDashboard(ctx context.Context, tenantID string) (*IntelligenceDashboard, error) {
	predictions, _ := s.repo.GetPredictions(ctx, tenantID, true)
	anomalies, _ := s.repo.GetAnomalies(ctx, tenantID, false)
	optimizations, _ := s.repo.GetOptimizations(ctx, tenantID)
	insights, _ := s.repo.GetInsights(ctx, tenantID)
	healthScores, _ := s.repo.GetHealthScores(ctx, tenantID)
	summary, _ := s.repo.GetSummary(ctx, tenantID)

	return &IntelligenceDashboard{
		Predictions:   predictions,
		Anomalies:     anomalies,
		Optimizations: optimizations,
		Insights:      insights,
		HealthScores:  healthScores,
		Summary:       summary,
	}, nil
}

// GetPredictions returns active failure predictions
func (s *Service) GetPredictions(ctx context.Context, tenantID string) ([]FailurePrediction, error) {
	return s.repo.GetPredictions(ctx, tenantID, true)
}

// GetAnomalies returns unacknowledged anomalies
func (s *Service) GetAnomalies(ctx context.Context, tenantID string) ([]AnomalyDetection, error) {
	return s.repo.GetAnomalies(ctx, tenantID, false)
}

// AcknowledgeAnomaly marks an anomaly as acknowledged
func (s *Service) AcknowledgeAnomaly(ctx context.Context, id string) error {
	return s.repo.AcknowledgeAnomaly(ctx, id)
}

// GetOptimizations returns pending retry optimizations
func (s *Service) GetOptimizations(ctx context.Context, tenantID string) ([]RetryOptimization, error) {
	return s.repo.GetOptimizations(ctx, tenantID)
}

// ApplyOptimization marks an optimization as applied
func (s *Service) ApplyOptimization(ctx context.Context, id string) error {
	return s.repo.ApplyOptimization(ctx, id)
}

// GetHealthScores returns all endpoint health scores
func (s *Service) GetHealthScores(ctx context.Context, tenantID string) ([]EndpointHealthScore, error) {
	return s.repo.GetHealthScores(ctx, tenantID)
}

// DismissInsight marks an insight as dismissed
func (s *Service) DismissInsight(ctx context.Context, id string) error {
	return s.repo.DismissInsight(ctx, id)
}
