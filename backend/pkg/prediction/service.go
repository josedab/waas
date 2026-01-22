package prediction

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Service provides predictive failure prevention operations
type Service struct {
	repo             Repository
	predictor        Predictor
	extractor        FeatureExtractor
	notifier         Notifier
	collector        MetricsCollector
	healthCalc       HealthCalculator
	mu               sync.RWMutex
	config           *ServiceConfig
	running          bool
	stopCh           chan struct{}
}

// ServiceConfig holds service configuration
type ServiceConfig struct {
	PredictionIntervalSec  int
	MetricRetentionDays    int
	MinDataPointsForPredict int
	DefaultTimeHorizon     time.Duration
	EnableAutoRemediation  bool
	AlertCooldownMinutes   int
}

// DefaultServiceConfig returns default configuration
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		PredictionIntervalSec:   300, // 5 minutes
		MetricRetentionDays:     90,
		MinDataPointsForPredict: 100,
		DefaultTimeHorizon:      24 * time.Hour,
		EnableAutoRemediation:   false,
		AlertCooldownMinutes:    30,
	}
}

// NewService creates a new prediction service
func NewService(repo Repository, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}

	return &Service{
		repo:   repo,
		config: config,
		stopCh: make(chan struct{}),
	}
}

// SetPredictor sets the ML predictor
func (s *Service) SetPredictor(predictor Predictor) {
	s.predictor = predictor
}

// SetFeatureExtractor sets the feature extractor
func (s *Service) SetFeatureExtractor(extractor FeatureExtractor) {
	s.extractor = extractor
}

// SetNotifier sets the notifier
func (s *Service) SetNotifier(notifier Notifier) {
	s.notifier = notifier
}

// SetCollector sets the metrics collector
func (s *Service) SetCollector(collector MetricsCollector) {
	s.collector = collector
}

// SetHealthCalculator sets the health calculator
func (s *Service) SetHealthCalculator(calc HealthCalculator) {
	s.healthCalc = calc
}

// Start starts the prediction service
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.mu.Unlock()

	interval := time.Duration(s.config.PredictionIntervalSec) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.stopCh:
			return nil
		case <-ticker.C:
			s.runPredictionCycle(ctx)
		}
	}
}

// Stop stops the prediction service
func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		close(s.stopCh)
		s.running = false
		s.stopCh = make(chan struct{})
	}
}

// runPredictionCycle runs predictions for all endpoints
func (s *Service) runPredictionCycle(ctx context.Context) {
	if s.predictor == nil || !s.predictor.IsReady() {
		return
	}

	// Get all endpoints
	endpoints, err := s.repo.ListEndpointHealth(ctx, "")
	if err != nil {
		return
	}

	for _, endpoint := range endpoints {
		_ = s.predictForEndpoint(ctx, endpoint.TenantID, endpoint.EndpointID)
	}
}

// predictForEndpoint runs prediction for a single endpoint
func (s *Service) predictForEndpoint(ctx context.Context, tenantID, endpointID string) error {
	// Get historical metrics
	since := time.Now().Add(-7 * 24 * time.Hour) // Last 7 days
	metrics, err := s.repo.GetMetrics(ctx, endpointID, since, "5m")
	if err != nil {
		return err
	}

	if len(metrics) < s.config.MinDataPointsForPredict {
		return ErrInsufficientData
	}

	// Make predictions
	predictions, err := s.predictor.PredictAll(ctx, endpointID, metrics)
	if err != nil {
		return err
	}

	// Process each prediction
	for _, prediction := range predictions {
		prediction.ID = uuid.New().String()
		prediction.TenantID = tenantID
		prediction.PredictedAt = time.Now()

		// Save prediction
		if err := s.repo.SavePrediction(ctx, &prediction); err != nil {
			continue
		}

		// Check alert rules
		s.evaluateAlertRules(ctx, tenantID, &prediction)
	}

	return nil
}

// Predict makes predictions for an endpoint on demand
func (s *Service) Predict(ctx context.Context, tenantID string, req *PredictRequest) ([]Prediction, error) {
	if s.predictor == nil {
		return nil, ErrModelNotReady
	}

	// Parse time horizon
	horizon := s.config.DefaultTimeHorizon
	if req.TimeHorizon != "" {
		switch req.TimeHorizon {
		case "1h":
			horizon = time.Hour
		case "6h":
			horizon = 6 * time.Hour
		case "24h":
			horizon = 24 * time.Hour
		case "7d":
			horizon = 7 * 24 * time.Hour
		}
	}

	// Get metrics for prediction window
	since := time.Now().Add(-horizon * 2)
	metrics, err := s.repo.GetMetrics(ctx, req.EndpointID, since, "5m")
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	if len(metrics) < s.config.MinDataPointsForPredict {
		return nil, ErrInsufficientData
	}

	// Make predictions
	var predictions []Prediction
	if len(req.Types) == 0 {
		predictions, err = s.predictor.PredictAll(ctx, req.EndpointID, metrics)
	} else {
		for _, pType := range req.Types {
			pred, err := s.predictor.Predict(ctx, req.EndpointID, pType, metrics)
			if err != nil {
				continue
			}
			predictions = append(predictions, *pred)
		}
	}

	if err != nil {
		return nil, err
	}

	// Add metadata and save
	now := time.Now()
	for i := range predictions {
		predictions[i].ID = uuid.New().String()
		predictions[i].TenantID = tenantID
		predictions[i].PredictedAt = now
		predictions[i].ExpectedTime = now.Add(horizon)
		predictions[i].TimeWindow = horizon / 4 // 25% uncertainty window

		_ = s.repo.SavePrediction(ctx, &predictions[i])
	}

	return predictions, nil
}

// evaluateAlertRules checks if predictions should trigger alerts
func (s *Service) evaluateAlertRules(ctx context.Context, tenantID string, prediction *Prediction) {
	rules, err := s.repo.GetEnabledRules(ctx, tenantID, prediction.Type)
	if err != nil || len(rules) == 0 {
		return
	}

	for _, rule := range rules {
		if !s.shouldTriggerAlert(ctx, tenantID, prediction, &rule) {
			continue
		}

		// Create alert
		alert := &Alert{
			ID:           uuid.New().String(),
			TenantID:     tenantID,
			EndpointID:   prediction.EndpointID,
			PredictionID: prediction.ID,
			Type:         prediction.Type,
			Severity:     rule.Severity,
			Status:       AlertStatusActive,
			Title:        s.generateAlertTitle(prediction),
			Description:  prediction.Description,
			CreatedAt:    time.Now(),
		}

		if err := s.repo.CreateAlert(ctx, alert); err != nil {
			continue
		}

		// Send notifications
		s.sendAlertNotifications(ctx, tenantID, alert, rule.NotificationChannels)
	}
}

// shouldTriggerAlert determines if an alert should be triggered
func (s *Service) shouldTriggerAlert(ctx context.Context, tenantID string, prediction *Prediction, rule *AlertRule) bool {
	// Check probability threshold
	if prediction.Probability < rule.ProbabilityThreshold {
		return false
	}

	// Check confidence threshold
	if rule.ConfidenceThreshold > 0 && prediction.Confidence < rule.ConfidenceThreshold {
		return false
	}

	// Check time window
	if rule.TimeWindowMinutes > 0 {
		expectedWithin := time.Until(prediction.ExpectedTime)
		if expectedWithin > time.Duration(rule.TimeWindowMinutes)*time.Minute {
			return false
		}
	}

	// Check cooldown
	lastAlert, _ := s.repo.GetLastAlertTime(ctx, tenantID, prediction.EndpointID, prediction.Type)
	if lastAlert != nil {
		cooldown := time.Duration(rule.CooldownMinutes) * time.Minute
		if time.Since(*lastAlert) < cooldown {
			return false
		}
	}

	// Check additional conditions
	if len(rule.Conditions) > 0 {
		health, _ := s.repo.GetEndpointHealth(ctx, tenantID, prediction.EndpointID)
		if health != nil && !s.evaluateConditions(rule.Conditions, health) {
			return false
		}
	}

	return true
}

// evaluateConditions evaluates additional rule conditions
func (s *Service) evaluateConditions(conditions []RuleCondition, health *EndpointHealth) bool {
	for _, cond := range conditions {
		var value float64
		switch cond.Field {
		case "health_score":
			value = health.HealthScore
		case "error_rate":
			value = health.ErrorRate
		case "success_rate":
			value = health.SuccessRate
		case "latency_ms":
			value = health.AverageLatencyMs
		default:
			continue
		}

		if !evaluateOperator(cond.Operator, value, cond.Value) {
			return false
		}
	}
	return true
}

// evaluateOperator evaluates a comparison operator
func evaluateOperator(operator string, actual, expected float64) bool {
	switch operator {
	case "lt":
		return actual < expected
	case "gt":
		return actual > expected
	case "eq":
		return actual == expected
	case "lte":
		return actual <= expected
	case "gte":
		return actual >= expected
	default:
		return false
	}
}

// generateAlertTitle generates an alert title
func (s *Service) generateAlertTitle(prediction *Prediction) string {
	titles := map[PredictionType]string{
		PredictionEndpointFailure:   "Endpoint failure predicted",
		PredictionLatencySpike:      "Latency spike predicted",
		PredictionErrorRateIncrease: "Error rate increase predicted",
		PredictionCapacityExhaustion: "Capacity exhaustion predicted",
		PredictionCertificateExpiry: "Certificate expiry warning",
		PredictionQuotaExhaustion:   "Quota exhaustion predicted",
	}
	if title, ok := titles[prediction.Type]; ok {
		return fmt.Sprintf("%s (%.0f%% probability)", title, prediction.Probability*100)
	}
	return "Prediction alert"
}

// sendAlertNotifications sends notifications for an alert
func (s *Service) sendAlertNotifications(ctx context.Context, tenantID string, alert *Alert, channels []string) {
	if s.notifier == nil {
		return
	}

	for _, channel := range channels {
		config, err := s.repo.GetNotificationConfig(ctx, tenantID, channel)
		if err != nil || !config.Enabled {
			continue
		}

		// Check if severity matches
		if len(config.Severities) > 0 {
			found := false
			for _, sev := range config.Severities {
				if sev == alert.Severity {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		if err := s.notifier.SendAlert(ctx, alert, config); err != nil {
			continue
		}

		// Record notification
		alert.NotificationsSent = append(alert.NotificationsSent, NotificationRecord{
			Channel:   channel,
			Recipient: config.Config["recipient"],
			SentAt:    time.Now(),
			Status:    "sent",
		})
	}

	_ = s.repo.UpdateAlert(ctx, alert)
}

// GetEndpointHealth retrieves health for an endpoint
func (s *Service) GetEndpointHealth(ctx context.Context, tenantID, endpointID string) (*EndpointHealth, error) {
	return s.repo.GetEndpointHealth(ctx, tenantID, endpointID)
}

// ListEndpointHealth lists health for all endpoints
func (s *Service) ListEndpointHealth(ctx context.Context, tenantID string) ([]EndpointHealth, error) {
	return s.repo.ListEndpointHealth(ctx, tenantID)
}

// CalculateEndpointHealth calculates health from recent metrics
func (s *Service) CalculateEndpointHealth(ctx context.Context, tenantID, endpointID string) (*EndpointHealth, error) {
	// Get recent metrics
	since := time.Now().Add(-time.Hour)
	metrics, err := s.repo.GetMetrics(ctx, endpointID, since, "1m")
	if err != nil {
		return nil, err
	}

	if len(metrics) == 0 {
		return nil, ErrInsufficientData
	}

	// Calculate health
	health := &EndpointHealth{
		EndpointID:  endpointID,
		TenantID:    tenantID,
		LastChecked: time.Now(),
	}

	// Calculate aggregates
	var totalSuccessRate, totalLatency, totalErrorRate float64
	var totalRequests, failedRequests int64
	var latencies []float64

	for _, m := range metrics {
		totalSuccessRate += m.SuccessRate
		totalLatency += m.LatencyMs
		totalErrorRate += m.ErrorRate
		latencies = append(latencies, m.LatencyMs)

		// Calculate request counts from status codes
		for code, count := range m.StatusCodes {
			totalRequests += int64(count)
			if code >= 500 {
				failedRequests += int64(count)
			}
		}
	}

	n := float64(len(metrics))
	health.SuccessRate = totalSuccessRate / n
	health.AverageLatencyMs = totalLatency / n
	health.ErrorRate = totalErrorRate / n
	health.TotalRequests = totalRequests
	health.FailedRequests = failedRequests

	// Calculate percentiles
	sort.Float64s(latencies)
	health.P95LatencyMs = percentile(latencies, 95)
	health.P99LatencyMs = percentile(latencies, 99)

	// Calculate health score (0-100)
	health.HealthScore = s.calculateHealthScore(health)

	// Determine status
	if health.HealthScore >= 90 {
		health.CurrentStatus = "healthy"
	} else if health.HealthScore >= 70 {
		health.CurrentStatus = "degraded"
	} else {
		health.CurrentStatus = "unhealthy"
	}

	// Save health
	if err := s.repo.SaveEndpointHealth(ctx, health); err != nil {
		return nil, err
	}

	return health, nil
}

// calculateHealthScore calculates a health score from metrics
func (s *Service) calculateHealthScore(health *EndpointHealth) float64 {
	thresholds := DefaultHealthThresholds()

	// Success rate component (40%)
	successScore := 100.0
	if health.SuccessRate < thresholds.DegradedSuccessRate {
		successScore = (health.SuccessRate / thresholds.DegradedSuccessRate) * 50
	} else if health.SuccessRate < thresholds.HealthySuccessRate {
		successScore = 50 + ((health.SuccessRate - thresholds.DegradedSuccessRate) / (thresholds.HealthySuccessRate - thresholds.DegradedSuccessRate)) * 50
	}

	// Latency component (30%)
	latencyScore := 100.0
	if health.P95LatencyMs > thresholds.DegradedLatencyMs {
		latencyScore = math.Max(0, 100-(health.P95LatencyMs-thresholds.DegradedLatencyMs)/10)
	} else if health.P95LatencyMs > thresholds.HealthyLatencyMs {
		latencyScore = 50 + (1-(health.P95LatencyMs-thresholds.HealthyLatencyMs)/(thresholds.DegradedLatencyMs-thresholds.HealthyLatencyMs))*50
	}

	// Error rate component (30%)
	errorScore := 100.0
	if health.ErrorRate > thresholds.DegradedErrorRate {
		errorScore = math.Max(0, 100-(health.ErrorRate-thresholds.DegradedErrorRate)*1000)
	} else if health.ErrorRate > thresholds.HealthyErrorRate {
		errorScore = 50 + (1-(health.ErrorRate-thresholds.HealthyErrorRate)/(thresholds.DegradedErrorRate-thresholds.HealthyErrorRate))*50
	}

	return math.Min(100, successScore*0.4+latencyScore*0.3+errorScore*0.3)
}

// percentile calculates the nth percentile of a sorted slice
func percentile(sorted []float64, p int) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)*p) / 100)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// GetPrediction retrieves a prediction
func (s *Service) GetPrediction(ctx context.Context, tenantID, predictionID string) (*Prediction, error) {
	pred, err := s.repo.GetPrediction(ctx, predictionID)
	if err != nil {
		return nil, err
	}
	if pred.TenantID != tenantID {
		return nil, ErrPredictionNotFound
	}
	return pred, nil
}

// ListPredictions lists predictions
func (s *Service) ListPredictions(ctx context.Context, tenantID string, filters *PredictionFilters) (*ListPredictionsResponse, error) {
	if filters == nil {
		filters = &PredictionFilters{Page: 1, PageSize: 20}
	}

	predictions, total, err := s.repo.ListPredictions(ctx, tenantID, filters)
	if err != nil {
		return nil, err
	}

	return &ListPredictionsResponse{
		Predictions: predictions,
		Total:       total,
		Page:        filters.Page,
		PageSize:    filters.PageSize,
		TotalPages:  (total + filters.PageSize - 1) / filters.PageSize,
	}, nil
}

// RecordPredictionOutcome records the actual outcome of a prediction
func (s *Service) RecordPredictionOutcome(ctx context.Context, tenantID, predictionID string, outcome *PredictionOutcome) error {
	prediction, err := s.repo.GetPrediction(ctx, predictionID)
	if err != nil {
		return err
	}
	if prediction.TenantID != tenantID {
		return ErrPredictionNotFound
	}

	outcome.RecordedAt = time.Now()

	// Check if actual time was within window
	if outcome.ActualTime != nil {
		windowStart := prediction.ExpectedTime.Add(-prediction.TimeWindow)
		windowEnd := prediction.ExpectedTime.Add(prediction.TimeWindow)
		outcome.WithinWindow = outcome.ActualTime.After(windowStart) && outcome.ActualTime.Before(windowEnd)
	}

	prediction.Outcome = outcome
	return s.repo.UpdatePrediction(ctx, prediction)
}

// GetAlert retrieves an alert
func (s *Service) GetAlert(ctx context.Context, tenantID, alertID string) (*Alert, error) {
	alert, err := s.repo.GetAlert(ctx, alertID)
	if err != nil {
		return nil, err
	}
	if alert.TenantID != tenantID {
		return nil, ErrAlertNotFound
	}
	return alert, nil
}

// UpdateAlert updates an alert
func (s *Service) UpdateAlert(ctx context.Context, tenantID, alertID, userID string, req *UpdateAlertRequest) (*Alert, error) {
	alert, err := s.repo.GetAlert(ctx, alertID)
	if err != nil {
		return nil, err
	}
	if alert.TenantID != tenantID {
		return nil, ErrAlertNotFound
	}

	now := time.Now()

	if req.Status != nil {
		alert.Status = *req.Status

		switch *req.Status {
		case AlertStatusAcknowledged:
			alert.AcknowledgedAt = &now
			alert.AcknowledgedBy = userID
		case AlertStatusResolved:
			alert.ResolvedAt = &now
			alert.ResolvedBy = userID
			alert.ResolutionNotes = req.ResolutionNotes
		case AlertStatusSuppressed:
			if req.SuppressMinutes > 0 {
				suppressUntil := now.Add(time.Duration(req.SuppressMinutes) * time.Minute)
				alert.SuppressedUntil = &suppressUntil
			}
		}
	}

	if err := s.repo.UpdateAlert(ctx, alert); err != nil {
		return nil, err
	}

	return alert, nil
}

// ListAlerts lists alerts
func (s *Service) ListAlerts(ctx context.Context, tenantID string, filters *AlertFilters) (*ListAlertsResponse, error) {
	if filters == nil {
		filters = &AlertFilters{Page: 1, PageSize: 20}
	}

	alerts, total, err := s.repo.ListAlerts(ctx, tenantID, filters)
	if err != nil {
		return nil, err
	}

	return &ListAlertsResponse{
		Alerts:     alerts,
		Total:      total,
		Page:       filters.Page,
		PageSize:   filters.PageSize,
		TotalPages: (total + filters.PageSize - 1) / filters.PageSize,
	}, nil
}

// CreateAlertRule creates an alert rule
func (s *Service) CreateAlertRule(ctx context.Context, tenantID string, req *CreateAlertRuleRequest) (*AlertRule, error) {
	now := time.Now()
	rule := &AlertRule{
		ID:                   uuid.New().String(),
		TenantID:             tenantID,
		Name:                 req.Name,
		Description:          req.Description,
		PredictionType:       req.PredictionType,
		ProbabilityThreshold: req.ProbabilityThreshold,
		ConfidenceThreshold:  req.ConfidenceThreshold,
		TimeWindowMinutes:    req.TimeWindowMinutes,
		Severity:             req.Severity,
		NotificationChannels: req.NotificationChannels,
		CooldownMinutes:      req.CooldownMinutes,
		Conditions:           req.Conditions,
		Enabled:              true,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if err := s.repo.CreateAlertRule(ctx, rule); err != nil {
		return nil, err
	}

	return rule, nil
}

// GetAlertRule retrieves an alert rule
func (s *Service) GetAlertRule(ctx context.Context, tenantID, ruleID string) (*AlertRule, error) {
	return s.repo.GetAlertRule(ctx, tenantID, ruleID)
}

// UpdateAlertRule updates an alert rule
func (s *Service) UpdateAlertRule(ctx context.Context, tenantID string, rule *AlertRule) error {
	rule.UpdatedAt = time.Now()
	return s.repo.UpdateAlertRule(ctx, rule)
}

// DeleteAlertRule deletes an alert rule
func (s *Service) DeleteAlertRule(ctx context.Context, tenantID, ruleID string) error {
	return s.repo.DeleteAlertRule(ctx, tenantID, ruleID)
}

// ListAlertRules lists alert rules
func (s *Service) ListAlertRules(ctx context.Context, tenantID string) ([]AlertRule, error) {
	return s.repo.ListAlertRules(ctx, tenantID)
}

// GetDashboard retrieves dashboard statistics
func (s *Service) GetDashboard(ctx context.Context, tenantID string) (*DashboardStats, error) {
	return s.repo.GetDashboardStats(ctx, tenantID)
}

// SaveNotificationConfig saves a notification configuration
func (s *Service) SaveNotificationConfig(ctx context.Context, tenantID string, config *NotificationConfig) error {
	config.ID = uuid.New().String()
	config.TenantID = tenantID
	config.CreatedAt = time.Now()
	config.UpdatedAt = time.Now()
	return s.repo.SaveNotificationConfig(ctx, config)
}

// ListNotificationConfigs lists notification configurations
func (s *Service) ListNotificationConfigs(ctx context.Context, tenantID string) ([]NotificationConfig, error) {
	return s.repo.ListNotificationConfigs(ctx, tenantID)
}

// TestNotificationConfig tests a notification configuration
func (s *Service) TestNotificationConfig(ctx context.Context, tenantID string, config *NotificationConfig) error {
	if s.notifier == nil {
		return fmt.Errorf("notifier not configured")
	}
	return s.notifier.TestConnection(ctx, config)
}

// RecordMetrics records metrics for an endpoint
func (s *Service) RecordMetrics(ctx context.Context, dataPoint *MetricDataPoint) error {
	return s.repo.SaveMetrics(ctx, dataPoint)
}

// RecordFailure records a failure event
func (s *Service) RecordFailure(ctx context.Context, failure *FailureEvent) error {
	failure.ID = uuid.New().String()
	return s.repo.SaveFailureEvent(ctx, failure)
}
