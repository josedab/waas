package anomaly

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Repository defines the interface for anomaly storage
type Repository interface {
	// Baseline operations
	GetBaseline(ctx context.Context, tenantID, endpointID string, metricType MetricType) (*Baseline, error)
	SaveBaseline(ctx context.Context, baseline *Baseline) error

	// Anomaly operations
	SaveAnomaly(ctx context.Context, anomaly *Anomaly) error
	GetAnomaly(ctx context.Context, tenantID, anomalyID string) (*Anomaly, error)
	ListAnomalies(ctx context.Context, tenantID string, status string, limit, offset int) ([]Anomaly, int, error)
	UpdateAnomalyStatus(ctx context.Context, tenantID, anomalyID, status string) error

	// Detection config operations
	GetDetectionConfig(ctx context.Context, tenantID, endpointID string, metricType MetricType) (*DetectionConfig, error)
	SaveDetectionConfig(ctx context.Context, config *DetectionConfig) error
	ListDetectionConfigs(ctx context.Context, tenantID string) ([]DetectionConfig, error)

	// Alert config operations
	GetAlertConfigs(ctx context.Context, tenantID string) ([]AlertConfig, error)
	SaveAlertConfig(ctx context.Context, config *AlertConfig) error

	// Alert operations
	SaveAlert(ctx context.Context, alert *AnomalyAlert) error
	ListAlerts(ctx context.Context, tenantID string, anomalyID string, limit, offset int) ([]AnomalyAlert, int, error)

	// Metrics operations
	GetRecentMetrics(ctx context.Context, tenantID, endpointID string, metricType MetricType, duration time.Duration) ([]MetricDataPoint, error)
}

// AlertNotifier defines the interface for sending alerts
type AlertNotifier interface {
	Send(ctx context.Context, alert *AnomalyAlert, anomaly *Anomaly, config *AlertConfig) error
}

// Service provides anomaly detection functionality
type Service struct {
	repo     Repository
	detector *Detector
	notifier AlertNotifier
}

// NewService creates a new anomaly detection service
func NewService(repo Repository, notifier AlertNotifier) *Service {
	return &Service{
		repo:     repo,
		detector: NewDetector(nil),
		notifier: notifier,
	}
}

// CheckMetric checks a metric value for anomalies
func (s *Service) CheckMetric(ctx context.Context, tenantID, endpointID string, metricType MetricType, value float64) (*DetectionResult, error) {
	// Get baseline
	baseline, err := s.repo.GetBaseline(ctx, tenantID, endpointID, metricType)
	if err != nil {
		return nil, fmt.Errorf("failed to get baseline: %w", err)
	}

	if baseline == nil {
		// No baseline yet, can't detect anomalies
		return &DetectionResult{IsAnomaly: false, Description: "No baseline established"}, nil
	}

	// Get detection config
	config, _ := s.repo.GetDetectionConfig(ctx, tenantID, endpointID, metricType)

	// Run detection
	result := s.detector.Detect(value, baseline, config)

	// If anomaly detected, save it and send alerts
	if result.IsAnomaly {
		anomaly := &Anomaly{
			ID:            uuid.New().String(),
			TenantID:      tenantID,
			EndpointID:    endpointID,
			MetricType:    metricType,
			Severity:      result.Severity,
			CurrentValue:  value,
			ExpectedValue: baseline.Mean,
			Deviation:     value - baseline.Mean,
			DeviationPct:  ((value - baseline.Mean) / baseline.Mean) * 100,
			Description:   result.Description,
			Status:        "open",
			DetectedAt:    time.Now(),
		}

		// Generate root cause suggestion
		anomaly.RootCause = s.suggestRootCause(metricType, result)
		anomaly.Recommendation = s.suggestRecommendation(metricType, result)

		if err := s.repo.SaveAnomaly(ctx, anomaly); err != nil {
			// Log but continue
		}

		// Send alerts
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			s.sendAlerts(ctx, anomaly)
		}()
	}

	return result, nil
}

// UpdateBaseline updates the baseline for a metric
func (s *Service) UpdateBaseline(ctx context.Context, tenantID, endpointID string, metricType MetricType, window time.Duration) error {
	// Get recent metrics
	points, err := s.repo.GetRecentMetrics(ctx, tenantID, endpointID, metricType, window)
	if err != nil {
		return fmt.Errorf("failed to get recent metrics: %w", err)
	}

	if len(points) == 0 {
		return nil
	}

	// Get existing baseline
	existing, _ := s.repo.GetBaseline(ctx, tenantID, endpointID, metricType)

	// Calculate new baseline
	baseline := CalculateBaseline(points, existing)
	baseline.TenantID = tenantID
	baseline.EndpointID = endpointID
	baseline.MetricType = metricType
	baseline.UpdatedAt = time.Now()

	if baseline.ID == "" {
		baseline.ID = uuid.New().String()
	}

	if err := s.repo.SaveBaseline(ctx, baseline); err != nil {
		return fmt.Errorf("failed to save baseline: %w", err)
	}

	return nil
}

// GetAnomalies lists detected anomalies
func (s *Service) GetAnomalies(ctx context.Context, tenantID string, status string, limit, offset int) ([]Anomaly, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListAnomalies(ctx, tenantID, status, limit, offset)
}

// ListAnomalies lists anomalies with extended filters
func (s *Service) ListAnomalies(ctx context.Context, tenantID, status, severity, endpointID string, limit, offset int) ([]Anomaly, error) {
	anomalies, _, err := s.repo.ListAnomalies(ctx, tenantID, status, limit, offset)
	return anomalies, err
}

// GetAnomaly retrieves a single anomaly
func (s *Service) GetAnomaly(ctx context.Context, id string) (*Anomaly, error) {
	return s.repo.GetAnomaly(ctx, "", id)
}

// GetBaselines retrieves baselines for a tenant
func (s *Service) GetBaselines(ctx context.Context, tenantID, endpointID string) ([]Baseline, error) {
	// Get baselines for all metric types
	var baselines []Baseline
	metricTypes := []MetricType{MetricTypeErrorRate, MetricTypeLatencyP95, MetricTypeDeliveryRate, MetricTypeRetryRate}

	for _, mt := range metricTypes {
		baseline, err := s.repo.GetBaseline(ctx, tenantID, endpointID, mt)
		if err == nil && baseline != nil {
			baselines = append(baselines, *baseline)
		}
	}
	return baselines, nil
}

// RecalculateBaselines triggers baseline recalculation
func (s *Service) RecalculateBaselines(ctx context.Context, tenantID, endpointID string) error {
	metricTypes := []MetricType{MetricTypeErrorRate, MetricTypeLatencyP95, MetricTypeDeliveryRate, MetricTypeRetryRate}

	for _, mt := range metricTypes {
		s.UpdateBaseline(ctx, tenantID, endpointID, mt, 24*time.Hour)
	}
	return nil
}

// GetDetectionConfig retrieves detection configuration
func (s *Service) GetDetectionConfig(ctx context.Context, tenantID, endpointID, metricType string) (*DetectionConfig, error) {
	return s.repo.GetDetectionConfig(ctx, tenantID, endpointID, MetricType(metricType))
}

// UpdateDetectionConfig updates detection configuration
func (s *Service) UpdateDetectionConfig(ctx context.Context, config *DetectionConfig) error {
	config.UpdatedAt = time.Now()
	return s.repo.SaveDetectionConfig(ctx, config)
}

// ListAlertConfigs lists alert configurations
func (s *Service) ListAlertConfigs(ctx context.Context, tenantID string) ([]AlertConfig, error) {
	return s.repo.GetAlertConfigs(ctx, tenantID)
}

// DeleteAlertConfig deletes an alert configuration (placeholder)
func (s *Service) DeleteAlertConfig(ctx context.Context, id string) error {
	// In production, implement in repository
	return nil
}

// ListAlerts lists sent alerts
func (s *Service) ListAlerts(ctx context.Context, tenantID string, limit, offset int) ([]AnomalyAlert, error) {
	alerts, _, err := s.repo.ListAlerts(ctx, tenantID, "", limit, offset)
	return alerts, err
}

// AcknowledgeAnomaly acknowledges an anomaly
func (s *Service) AcknowledgeAnomaly(ctx context.Context, tenantID, anomalyID string) (*Anomaly, error) {
	if err := s.repo.UpdateAnomalyStatus(ctx, tenantID, anomalyID, "acknowledged"); err != nil {
		return nil, err
	}
	return s.repo.GetAnomaly(ctx, tenantID, anomalyID)
}

// ResolveAnomaly marks an anomaly as resolved
func (s *Service) ResolveAnomaly(ctx context.Context, tenantID, anomalyID string) (*Anomaly, error) {
	if err := s.repo.UpdateAnomalyStatus(ctx, tenantID, anomalyID, "resolved"); err != nil {
		return nil, err
	}
	return s.repo.GetAnomaly(ctx, tenantID, anomalyID)
}

// CreateDetectionConfig creates or updates detection configuration
func (s *Service) CreateDetectionConfig(ctx context.Context, tenantID string, req *CreateDetectionConfigRequest) (*DetectionConfig, error) {
	config := &DetectionConfig{
		ID:                uuid.New().String(),
		TenantID:          tenantID,
		EndpointID:        req.EndpointID,
		MetricType:        req.MetricType,
		Enabled:           true,
		Sensitivity:       req.Sensitivity,
		MinSamples:        req.MinSamples,
		CooldownMinutes:   req.CooldownMinutes,
		CriticalThreshold: req.CriticalThreshold,
		WarningThreshold:  req.WarningThreshold,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// Set defaults
	if config.Sensitivity == 0 {
		config.Sensitivity = 1.0
	}
	if config.MinSamples == 0 {
		config.MinSamples = 30
	}
	if config.CooldownMinutes == 0 {
		config.CooldownMinutes = 15
	}
	if config.CriticalThreshold == 0 {
		config.CriticalThreshold = 3.0
	}
	if config.WarningThreshold == 0 {
		config.WarningThreshold = 2.0
	}

	if err := s.repo.SaveDetectionConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to save detection config: %w", err)
	}

	return config, nil
}

// ListDetectionConfigs lists detection configurations
func (s *Service) ListDetectionConfigs(ctx context.Context, tenantID string) ([]DetectionConfig, error) {
	return s.repo.ListDetectionConfigs(ctx, tenantID)
}

// CreateAlertConfig creates alert configuration
func (s *Service) CreateAlertConfig(ctx context.Context, tenantID string, req *CreateAlertConfigRequest) (*AlertConfig, error) {
	config := &AlertConfig{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        req.Name,
		Channel:     req.Channel,
		Config:      req.Config,
		MinSeverity: req.MinSeverity,
		Enabled:     true,
		CreatedAt:   time.Now(),
	}

	if err := s.repo.SaveAlertConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to save alert config: %w", err)
	}

	return config, nil
}

// GetTrendAnalysis returns trend analysis for a metric
func (s *Service) GetTrendAnalysis(ctx context.Context, tenantID, endpointID string, metricType MetricType) (*TrendAnalysis, error) {
	// Get recent data
	points, err := s.repo.GetRecentMetrics(ctx, tenantID, endpointID, metricType, 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	// Get baseline
	baseline, _ := s.repo.GetBaseline(ctx, tenantID, endpointID, metricType)
	if baseline == nil {
		baseline = &Baseline{Mean: 1.0, StdDev: 0.1} // Defaults
	}

	// Analyze trend
	analysis := s.detector.DetectTrend(points, baseline)
	analysis.TenantID = tenantID
	analysis.EndpointID = endpointID
	analysis.MetricType = metricType

	return analysis, nil
}

func (s *Service) sendAlerts(ctx context.Context, anomaly *Anomaly) {
	if s.notifier == nil {
		return
	}

	configs, err := s.repo.GetAlertConfigs(ctx, anomaly.TenantID)
	if err != nil {
		return
	}

	for _, config := range configs {
		if !config.Enabled {
			continue
		}

		// Check severity threshold
		if !s.severityMeetsThreshold(anomaly.Severity, config.MinSeverity) {
			continue
		}

		alert := &AnomalyAlert{
			ID:        uuid.New().String(),
			TenantID:  anomaly.TenantID,
			AnomalyID: anomaly.ID,
			Channel:   config.Channel,
			Status:    "pending",
			CreatedAt: time.Now(),
		}

		if err := s.notifier.Send(ctx, alert, anomaly, &config); err != nil {
			alert.Status = "failed"
			alert.Error = err.Error()
		} else {
			alert.Status = "sent"
			now := time.Now()
			alert.SentAt = &now
		}

		s.repo.SaveAlert(ctx, alert)
	}
}

func (s *Service) severityMeetsThreshold(severity, threshold Severity) bool {
	severityOrder := map[Severity]int{
		SeverityInfo:     1,
		SeverityWarning:  2,
		SeverityCritical: 3,
	}
	return severityOrder[severity] >= severityOrder[threshold]
}

func (s *Service) suggestRootCause(metricType MetricType, result *DetectionResult) string {
	causes := map[MetricType]string{
		MetricTypeErrorRate:    "Possible endpoint unavailability, network issues, or downstream service errors",
		MetricTypeLatencyP95:   "Increased load, resource constraints, or network latency",
		MetricTypeDeliveryRate: "Queue backlog, processing delays, or rate limiting",
		MetricTypeRetryRate:    "Endpoint reliability issues or timeout configuration",
	}

	if cause, ok := causes[metricType]; ok {
		return cause
	}
	return "Unknown cause - requires investigation"
}

func (s *Service) suggestRecommendation(metricType MetricType, result *DetectionResult) string {
	recommendations := map[MetricType]string{
		MetricTypeErrorRate:    "Check endpoint health, review error logs, verify network connectivity",
		MetricTypeLatencyP95:   "Scale resources, optimize endpoint performance, check for bottlenecks",
		MetricTypeDeliveryRate: "Increase worker capacity, check queue health, review rate limits",
		MetricTypeRetryRate:    "Adjust timeout settings, implement circuit breaker, contact endpoint owner",
	}

	if rec, ok := recommendations[metricType]; ok {
		return rec
	}
	return "Monitor the situation and investigate if it persists"
}

// SlackNotifier sends alerts to Slack
type SlackNotifier struct{}

func (n *SlackNotifier) Send(ctx context.Context, alert *AnomalyAlert, anomaly *Anomaly, config *AlertConfig) error {
	// Parse config
	var slackConfig struct {
		WebhookURL string `json:"webhook_url"`
		Channel    string `json:"channel"`
	}
	json.Unmarshal([]byte(config.Config), &slackConfig)

	if slackConfig.WebhookURL == "" {
		return fmt.Errorf("missing webhook URL")
	}

	// In production, this would send to Slack API
	// For now, just validate config
	return nil
}
