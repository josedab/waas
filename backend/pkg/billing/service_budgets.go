package billing

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
)

// CreateBudget creates a new budget
func (s *Service) CreateBudget(ctx context.Context, tenantID string, req *CreateBudgetRequest) (*BudgetConfig, error) {
	budget := &BudgetConfig{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		Name:         req.Name,
		Amount:       req.Amount,
		Currency:     req.Currency,
		Period:       req.Period,
		ResourceType: req.ResourceType,
		WebhookID:    req.WebhookID,
		Alerts:       req.Alerts,
		AutoPause:    req.AutoPause,
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if budget.Currency == "" {
		budget.Currency = "USD"
	}

	if err := s.repo.SaveBudget(ctx, budget); err != nil {
		return nil, fmt.Errorf("save budget: %w", err)
	}

	return budget, nil
}

// GetBudget retrieves a budget
func (s *Service) GetBudget(ctx context.Context, tenantID, budgetID string) (*BudgetConfig, error) {
	return s.repo.GetBudget(ctx, tenantID, budgetID)
}

// ListBudgets lists budgets
func (s *Service) ListBudgets(ctx context.Context, tenantID string) ([]BudgetConfig, error) {
	return s.repo.ListBudgets(ctx, tenantID)
}

// UpdateBudget updates a budget
func (s *Service) UpdateBudget(ctx context.Context, tenantID, budgetID string, req *UpdateBudgetRequest) (*BudgetConfig, error) {
	budget, err := s.repo.GetBudget(ctx, tenantID, budgetID)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		budget.Name = req.Name
	}
	if req.Amount > 0 {
		budget.Amount = req.Amount
	}
	if req.Currency != "" {
		budget.Currency = req.Currency
	}
	if req.Period != "" {
		budget.Period = req.Period
	}
	if req.Alerts != nil {
		budget.Alerts = req.Alerts
	}
	if req.AutoPause != nil {
		budget.AutoPause = *req.AutoPause
	}
	if req.Enabled != nil {
		budget.Enabled = *req.Enabled
	}
	budget.UpdatedAt = time.Now()

	if err := s.repo.SaveBudget(ctx, budget); err != nil {
		return nil, fmt.Errorf("update budget: %w", err)
	}

	return budget, nil
}

// DeleteBudget deletes a budget
func (s *Service) DeleteBudget(ctx context.Context, tenantID, budgetID string) error {
	return s.repo.DeleteBudget(ctx, tenantID, budgetID)
}

// GetAlerts lists alerts
func (s *Service) GetAlerts(ctx context.Context, tenantID string, status *AlertStatus) ([]BillingAlert, error) {
	return s.repo.ListAlerts(ctx, tenantID, status)
}

// AcknowledgeAlert acknowledges an alert
func (s *Service) AcknowledgeAlert(ctx context.Context, alertID, ackedBy string) error {
	return s.repo.UpdateAlertStatus(ctx, alertID, AlertAcked, ackedBy)
}

// GetOptimizations lists cost optimizations
func (s *Service) GetOptimizations(ctx context.Context, tenantID string) ([]CostOptimization, error) {
	return s.repo.ListOptimizations(ctx, tenantID)
}

// ImplementOptimization marks optimization as implemented
func (s *Service) ImplementOptimization(ctx context.Context, tenantID, optID string) error {
	return s.repo.UpdateOptimizationStatus(ctx, optID, OptStatusImplemented)
}

// DismissOptimization dismisses an optimization
func (s *Service) DismissOptimization(ctx context.Context, tenantID, optID string) error {
	return s.repo.UpdateOptimizationStatus(ctx, optID, OptStatusDismissed)
}

// AnalyzeOptimizations finds cost optimization opportunities
func (s *Service) AnalyzeOptimizations(ctx context.Context, tenantID string) ([]CostOptimization, error) {
	var optimizations []CostOptimization

	// Get usage summary for analysis
	period := time.Now().Format("2006-01")
	summary, err := s.repo.GetUsageSummary(ctx, tenantID, period)
	if err != nil {
		return nil, fmt.Errorf("get usage: %w", err)
	}

	// Check for high retry rates
	if summary.TotalRequests > 0 {
		retryRate := float64(summary.TotalFailed) / float64(summary.TotalRequests)
		if retryRate > 0.1 { // More than 10% failures
			retryCost := float64(summary.TotalFailed) * s.pricing.RetryAttemptCost
			optimizations = append(optimizations, CostOptimization{
				ID:               uuid.New().String(),
				TenantID:         tenantID,
				Type:             OptRetryReduction,
				Title:            "High Retry Rate Detected",
				Description:      fmt.Sprintf("%.1f%% of requests are failing and being retried, costing an estimated $%.2f per month", retryRate*100, retryCost),
				EstimatedSavings: retryCost * 0.5, // Assume 50% can be saved
				Currency:         s.pricing.Currency,
				Impact:           "medium",
				Actions: []OptimizationAction{
					{
						Type:        "investigate",
						Label:       "Review Failing Endpoints",
						Description: "Identify endpoints with high failure rates",
					},
					{
						Type:        "configure",
						Label:       "Adjust Retry Policy",
						Description: "Reduce retry attempts for consistently failing endpoints",
					},
				},
				Status:    OptStatusPending,
				CreatedAt: time.Now(),
			})
		}
	}

	// Check for large payload sizes
	if summary.TotalBytes > 0 && summary.TotalRequests > 0 {
		avgSize := float64(summary.TotalBytes) / float64(summary.TotalRequests)
		if avgSize > 100*1024 { // Larger than 100KB average
			compressionSavings := float64(summary.TotalBytes) * 0.6 * (s.pricing.DataTransferCost / (1024 * 1024 * 1024))
			optimizations = append(optimizations, CostOptimization{
				ID:               uuid.New().String(),
				TenantID:         tenantID,
				Type:             OptPayloadCompression,
				Title:            "Enable Payload Compression",
				Description:      fmt.Sprintf("Average payload size is %.1fKB. Compression could reduce transfer costs by up to 60%%", avgSize/1024),
				EstimatedSavings: compressionSavings,
				Currency:         s.pricing.Currency,
				Impact:           "high",
				Actions: []OptimizationAction{
					{
						Type:        "configure",
						Label:       "Enable GZIP Compression",
						Description: "Enable compression for webhook payloads",
					},
				},
				Status:    OptStatusPending,
				CreatedAt: time.Now(),
			})
		}
	}

	// Check for batch delivery opportunities
	if summary.TotalRequests > 1000 {
		batchSavings := float64(summary.TotalRequests) * 0.5 * s.pricing.WebhookRequestCost
		optimizations = append(optimizations, CostOptimization{
			ID:               uuid.New().String(),
			TenantID:         tenantID,
			Type:             OptBatchDelivery,
			Title:            "Consider Batch Delivery",
			Description:      "High request volume detected. Batching multiple events could reduce costs and improve efficiency",
			EstimatedSavings: batchSavings,
			Currency:         s.pricing.Currency,
			Impact:           "medium",
			Actions: []OptimizationAction{
				{
					Type:        "configure",
					Label:       "Enable Batching",
					Description: "Configure batch delivery for high-volume endpoints",
				},
			},
			Status:    OptStatusPending,
			CreatedAt: time.Now(),
		})
	}

	// Sort by estimated savings
	sort.Slice(optimizations, func(i, j int) bool {
		return optimizations[i].EstimatedSavings > optimizations[j].EstimatedSavings
	})

	// Save optimizations
	for i := range optimizations {
		if err := s.repo.SaveOptimization(ctx, &optimizations[i]); err != nil {
			continue
		}
	}

	return optimizations, nil
}

// checkBudgetAlerts checks budgets and triggers alerts
func (s *Service) checkBudgetAlerts(ctx context.Context, tenantID string) {
	budgets, err := s.repo.ListBudgets(ctx, tenantID)
	if err != nil {
		return
	}

	currentSpend, err := s.repo.GetCurrentSpend(ctx, tenantID)
	if err != nil {
		return
	}

	for _, budget := range budgets {
		if !budget.Enabled {
			continue
		}

		utilization := (currentSpend / budget.Amount) * 100

		// Check each alert threshold
		for _, threshold := range budget.Alerts {
			if utilization >= threshold.Percentage {
				s.triggerBudgetAlert(ctx, tenantID, &budget, threshold, currentSpend, utilization)
			}
		}

		// Auto-pause if enabled and over budget
		if budget.AutoPause && currentSpend >= budget.Amount {
			// Trigger critical alert
			alert := &BillingAlert{
				ID:       uuid.New().String(),
				TenantID: tenantID,
				Type:     AlertBudgetExceeded,
				Severity: SeverityCritical,
				Title:    fmt.Sprintf("Budget '%s' Exceeded - Auto-Pause Triggered", budget.Name),
				Message:  fmt.Sprintf("Current spend ($%.2f) has exceeded budget ($%.2f). Webhook delivery has been paused.", currentSpend, budget.Amount),
				Data: AlertData{
					BudgetLimit:  budget.Amount,
					CurrentSpend: currentSpend,
					Percentage:   (currentSpend / budget.Amount) * 100,
				},
				Status:    AlertPending,
				CreatedAt: time.Now(),
			}
			s.repo.SaveAlert(ctx, alert)
		}
	}
}

// triggerBudgetAlert triggers an alert for budget threshold
func (s *Service) triggerBudgetAlert(ctx context.Context, tenantID string, budget *BudgetConfig, threshold AlertThreshold, spend, utilization float64) {
	severity := SeverityInfo
	if threshold.Percentage >= 90 {
		severity = SeverityCritical
	} else if threshold.Percentage >= 75 {
		severity = SeverityWarning
	}

	alert := &BillingAlert{
		ID:       uuid.New().String(),
		TenantID: tenantID,
		Type:     AlertBudgetWarning,
		Severity: severity,
		Title:    fmt.Sprintf("Budget '%s' at %.0f%% Utilization", budget.Name, utilization),
		Message:  fmt.Sprintf("Current spend ($%.2f) has reached %.0f%% of budget ($%.2f)", spend, utilization, budget.Amount),
		Data: AlertData{
			BudgetLimit:  budget.Amount,
			CurrentSpend: spend,
			Percentage:   utilization,
		},
		Channels:  threshold.Channels,
		Status:    AlertPending,
		CreatedAt: time.Now(),
	}

	if err := s.repo.SaveAlert(ctx, alert); err != nil {
		return
	}

	// Send notification
	if s.notifier != nil && len(threshold.Channels) > 0 {
		config, _ := s.repo.GetAlertConfig(ctx, tenantID)
		var recipients []string
		if config != nil {
			recipients = append(recipients, config.Recipients.Emails...)
		}
		s.notifier.Send(ctx, alert, threshold.Channels, recipients)
	}
}

// DetectSpendAnomaly detects anomalous spending
func (s *Service) DetectSpendAnomaly(ctx context.Context, tenantID string) (*SpendAnomaly, error) {
	// Get last 30 days of usage
	var dailyCosts []float64
	now := time.Now()
	for i := 30; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)
		period := date.Format("2006-01")
		summary, err := s.repo.GetUsageSummary(ctx, tenantID, period)
		if err != nil {
			continue
		}
		for _, du := range summary.ByDay {
			if du.Date == date.Format("2006-01-02") {
				dailyCosts = append(dailyCosts, du.Cost)
				break
			}
		}
	}

	if len(dailyCosts) < 7 {
		return nil, nil // Not enough data
	}

	// Calculate mean and standard deviation
	mean := 0.0
	for _, c := range dailyCosts {
		mean += c
	}
	mean /= float64(len(dailyCosts))

	variance := 0.0
	for _, c := range dailyCosts {
		variance += (c - mean) * (c - mean)
	}
	variance /= float64(len(dailyCosts))
	stdDev := math.Sqrt(variance)

	// Check today's cost against baseline
	currentSpend, _ := s.repo.GetCurrentSpend(ctx, tenantID)
	todayCost := currentSpend // Simplified - would need daily breakdown

	// Detect anomaly if more than 2 standard deviations from mean
	zScore := (todayCost - mean) / stdDev
	if math.Abs(zScore) > 2 {
		severity := "medium"
		if math.Abs(zScore) > 3 {
			severity = "high"
		}

		anomaly := &SpendAnomaly{
			TenantID:     tenantID,
			CurrentCost:  todayCost,
			ExpectedCost: mean,
			Deviation:    ((todayCost - mean) / mean) * 100,
			ZScore:       zScore,
			Severity:     severity,
			DetectedAt:   time.Now(),
		}

		// Create alert
		alert := &BillingAlert{
			ID:       uuid.New().String(),
			TenantID: tenantID,
			Type:     AlertSpendAnomaly,
			Severity: AlertSeverity(severity),
			Title:    "Spending Anomaly Detected",
			Message:  fmt.Sprintf("Today's spend ($%.2f) is %.1f%% above expected ($%.2f)", todayCost, anomaly.Deviation, mean),
			Data: AlertData{
				CurrentSpend:  todayCost,
				ExpectedSpend: mean,
				AnomalyScore:  zScore,
			},
			Status:    AlertPending,
			CreatedAt: time.Now(),
		}
		s.repo.SaveAlert(ctx, alert)

		return anomaly, nil
	}

	return nil, nil
}

// GetAlertConfig retrieves alert configuration
func (s *Service) GetAlertConfig(ctx context.Context, tenantID string) (*AlertConfig, error) {
	return s.repo.GetAlertConfig(ctx, tenantID)
}

// UpdateAlertConfig updates alert configuration
func (s *Service) UpdateAlertConfig(ctx context.Context, tenantID string, req *UpdateAlertConfigRequest) (*AlertConfig, error) {
	config, err := s.repo.GetAlertConfig(ctx, tenantID)
	if err != nil {
		// Create new config
		config = &AlertConfig{
			ID:        uuid.New().String(),
			TenantID:  tenantID,
			Enabled:   true,
			Channels:  []AlertChannel{ChannelEmail},
			CreatedAt: time.Now(),
		}
	}

	if req.Enabled != nil {
		config.Enabled = *req.Enabled
	}
	if req.Channels != nil {
		config.Channels = req.Channels
	}
	if req.Recipients != nil {
		config.Recipients = *req.Recipients
	}
	if req.Schedule != nil {
		config.Schedule = *req.Schedule
	}
	config.UpdatedAt = time.Now()

	if err := s.repo.SaveAlertConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("save alert config: %w", err)
	}

	return config, nil
}

// UpdateBudgetRequest for updating a budget
type UpdateBudgetRequest struct {
	Name      string           `json:"name"`
	Amount    float64          `json:"amount"`
	Currency  string           `json:"currency"`
	Period    BillingPeriod    `json:"period"`
	Alerts    []AlertThreshold `json:"alerts"`
	AutoPause *bool            `json:"auto_pause"`
	Enabled   *bool            `json:"enabled"`
}

// UpdateAlertConfigRequest for updating alert config
type UpdateAlertConfigRequest struct {
	Enabled    *bool            `json:"enabled"`
	Channels   []AlertChannel   `json:"channels"`
	Recipients *AlertRecipients `json:"recipients"`
	Schedule   *AlertSchedule   `json:"schedule"`
}

// SpendAnomaly represents detected anomaly
type SpendAnomaly struct {
	TenantID     string    `json:"tenant_id"`
	CurrentCost  float64   `json:"current_cost"`
	ExpectedCost float64   `json:"expected_cost"`
	Deviation    float64   `json:"deviation"`
	ZScore       float64   `json:"z_score"`
	Severity     string    `json:"severity"`
	DetectedAt   time.Time `json:"detected_at"`
}
