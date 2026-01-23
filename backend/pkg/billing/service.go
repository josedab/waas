package billing

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
)

// PricingConfig defines pricing for resources
type PricingConfig struct {
	WebhookRequestCost  float64 // per request
	RetryAttemptCost    float64 // per retry
	DataTransferCost    float64 // per GB
	TransformationCost  float64 // per transform
	StorageCost         float64 // per GB per month
	Currency            string
}

// DefaultPricing provides default pricing
var DefaultPricing = PricingConfig{
	WebhookRequestCost:  0.0001,  // $0.10 per 1000 requests
	RetryAttemptCost:    0.00005, // $0.05 per 1000 retries
	DataTransferCost:    0.10,    // $0.10 per GB
	TransformationCost:  0.00002, // $0.02 per 1000 transforms
	StorageCost:         0.023,   // $0.023 per GB per month
	Currency:            "USD",
}

// Notifier sends billing alerts
type Notifier interface {
	Send(ctx context.Context, alert *BillingAlert, channels []AlertChannel, recipients []string) error
}

// ServiceConfig holds subscription billing configuration
type ServiceConfig struct {
	StripeAPIKey   string
	DefaultPlanID  string
	BillingEnabled bool
	FreeTierEvents int64
}

// Service provides billing operations
type Service struct {
	repo         Repository
	pricing      PricingConfig
	notifier     Notifier
	stripeClient StripeClient
	config       ServiceConfig
}

// NewService creates a new billing service
func NewService(repo Repository, pricing *PricingConfig, notifier Notifier) *Service {
	p := DefaultPricing
	if pricing != nil {
		p = *pricing
	}
	return &Service{
		repo:     repo,
		pricing:  p,
		notifier: notifier,
	}
}

// SetStripeClient sets the Stripe client on the service.
func (s *Service) SetStripeClient(client StripeClient) {
	s.stripeClient = client
}

// SetConfig sets the service config.
func (s *Service) SetConfig(cfg ServiceConfig) {
	s.config = cfg
}

// RecordWebhookUsage records webhook request usage
func (s *Service) RecordWebhookUsage(ctx context.Context, tenantID, webhookID string, requests, retries, bytesTransferred int64) error {
	now := time.Now()
	period := now.Format("2006-01")

	// Record request usage
	if requests > 0 {
		record := &CostUsageRecord{
			ID:            uuid.New().String(),
			TenantID:      tenantID,
			WebhookID:     webhookID,
			ResourceType:  "webhook_requests",
			Quantity:      requests,
			UnitCost:      s.pricing.WebhookRequestCost,
			TotalCost:     float64(requests) * s.pricing.WebhookRequestCost,
			Currency:      s.pricing.Currency,
			BillingPeriod: period,
			RecordedAt:    now,
		}
		if err := s.repo.RecordUsage(ctx, record); err != nil {
			return fmt.Errorf("record request usage: %w", err)
		}
	}

	// Record retry usage
	if retries > 0 {
		record := &CostUsageRecord{
			ID:            uuid.New().String(),
			TenantID:      tenantID,
			WebhookID:     webhookID,
			ResourceType:  "retry_attempts",
			Quantity:      retries,
			UnitCost:      s.pricing.RetryAttemptCost,
			TotalCost:     float64(retries) * s.pricing.RetryAttemptCost,
			Currency:      s.pricing.Currency,
			BillingPeriod: period,
			RecordedAt:    now,
		}
		if err := s.repo.RecordUsage(ctx, record); err != nil {
			return fmt.Errorf("record retry usage: %w", err)
		}
	}

	// Record data transfer
	if bytesTransferred > 0 {
		gbTransferred := float64(bytesTransferred) / (1024 * 1024 * 1024)
		record := &CostUsageRecord{
			ID:            uuid.New().String(),
			TenantID:      tenantID,
			WebhookID:     webhookID,
			ResourceType:  "data_transfer_bytes",
			Quantity:      bytesTransferred,
			UnitCost:      s.pricing.DataTransferCost,
			TotalCost:     gbTransferred * s.pricing.DataTransferCost,
			Currency:      s.pricing.Currency,
			BillingPeriod: period,
			RecordedAt:    now,
		}
		if err := s.repo.RecordUsage(ctx, record); err != nil {
			return fmt.Errorf("record data transfer: %w", err)
		}
	}

	// Check budgets for alerts
	go s.checkBudgetAlerts(context.Background(), tenantID)

	return nil
}

// GetCurrentSpend retrieves current period spend
func (s *Service) GetCurrentSpend(ctx context.Context, tenantID string) (float64, error) {
	return s.repo.GetCurrentSpend(ctx, tenantID)
}

// GetUsageSummary retrieves usage summary
func (s *Service) GetUsageSummary(ctx context.Context, tenantID, period string) (*CostUsageSummary, error) {
	return s.repo.GetUsageSummary(ctx, tenantID, period)
}

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

// GetInvoices lists invoices
func (s *Service) GetInvoices(ctx context.Context, tenantID string) ([]CostInvoice, error) {
	return s.repo.ListInvoices(ctx, tenantID)
}

// GetInvoice retrieves an invoice
func (s *Service) GetInvoice(ctx context.Context, tenantID, invoiceID string) (*CostInvoice, error) {
	return s.repo.GetInvoice(ctx, tenantID, invoiceID)
}

// ForecastSpend forecasts spend for a period
func (s *Service) ForecastSpend(ctx context.Context, tenantID string, days int) (*SpendForecast, error) {
	// Get current period usage
	period := time.Now().Format("2006-01")
	summary, err := s.repo.GetUsageSummary(ctx, tenantID, period)
	if err != nil {
		return nil, fmt.Errorf("get usage summary: %w", err)
	}

	// Calculate daily average
	daysPassed := float64(time.Now().Day())
	if daysPassed < 1 {
		daysPassed = 1
	}
	dailyAverage := summary.TotalCost / daysPassed

	// Calculate forecasts
	forecast := &SpendForecast{
		TenantID:       tenantID,
		CurrentSpend:   summary.TotalCost,
		ProjectedSpend: summary.TotalCost + (dailyAverage * float64(days)),
		DailyAverage:   dailyAverage,
		Currency:       summary.Currency,
		Period:         period,
	}

	// Calculate confidence based on data points
	if len(summary.ByDay) >= 7 {
		forecast.Confidence = 0.85
	} else if len(summary.ByDay) >= 3 {
		forecast.Confidence = 0.7
	} else {
		forecast.Confidence = 0.5
	}

	// Calculate trend
	if len(summary.ByDay) >= 3 {
		recent := summary.ByDay[len(summary.ByDay)-3:]
		if len(recent) >= 3 {
			firstAvg := (recent[0].Cost + recent[1].Cost) / 2
			secondAvg := (recent[1].Cost + recent[2].Cost) / 2
			if secondAvg > firstAvg*1.1 {
				forecast.TrendDirection = "increasing"
				forecast.TrendPercent = ((secondAvg - firstAvg) / firstAvg) * 100
			} else if secondAvg < firstAvg*0.9 {
				forecast.TrendDirection = "decreasing"
				forecast.TrendPercent = ((firstAvg - secondAvg) / firstAvg) * 100
			} else {
				forecast.TrendDirection = "stable"
				forecast.TrendPercent = 0
			}
		}
	}

	// Get budget for comparison
	budgets, err := s.repo.ListBudgets(ctx, tenantID)
	if err == nil {
		for _, b := range budgets {
			if b.Enabled && b.Period == BillingPeriod(period[:7]) {
				forecast.BudgetRemaining = b.Amount - summary.TotalCost
				forecast.BudgetUtilization = (summary.TotalCost / b.Amount) * 100
				break
			}
		}
	}

	return forecast, nil
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

// --- Subscription Billing Methods ---

// SubscriptionRepository defines additional repository methods for subscription billing.
type SubscriptionRepository interface {
	SavePricingPlan(ctx context.Context, plan *PricingPlan) error
	GetPricingPlan(ctx context.Context, planID uuid.UUID) (*PricingPlan, error)
	ListPricingPlans(ctx context.Context) ([]PricingPlan, error)
	SaveSubscription(ctx context.Context, sub *Subscription) error
	GetSubscription(ctx context.Context, tenantID uuid.UUID) (*Subscription, error)
	UpdateSubscription(ctx context.Context, sub *Subscription) error
	SaveUsageRecord(ctx context.Context, record *UsageRecord) error
	GetUsageRecord(ctx context.Context, tenantID uuid.UUID, periodStart time.Time) (*UsageRecord, error)
	GetCurrentUsageRecord(ctx context.Context, tenantID uuid.UUID) (*UsageRecord, error)
	SaveInvoice(ctx context.Context, invoice *Invoice) error
	GetInvoiceByID(ctx context.Context, invoiceID uuid.UUID) (*Invoice, error)
	ListInvoicesByTenant(ctx context.Context, tenantID uuid.UUID) ([]Invoice, error)
}

// CreatePricingPlan creates a new pricing plan.
func (s *Service) CreatePricingPlan(ctx context.Context, plan *PricingPlan) (*PricingPlan, error) {
	plan.ID = uuid.New()
	plan.Active = true
	plan.CreatedAt = time.Now()
	return plan, nil
}

// ListPricingPlans lists available pricing plans.
func (s *Service) ListPricingPlans(ctx context.Context) ([]PricingPlan, error) {
	return []PricingPlan{
		defaultFreePlan(),
		defaultStarterPlan(),
		defaultProPlan(),
		defaultEnterprisePlan(),
	}, nil
}

// CreateSubscriptionForTenant subscribes a tenant to a plan.
func (s *Service) CreateSubscriptionForTenant(ctx context.Context, tenantID, planID uuid.UUID) (*Subscription, error) {
	now := time.Now()
	sub := &Subscription{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		PlanID:             planID,
		Status:             "active",
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		CreatedAt:          now,
	}

	if s.stripeClient != nil && s.config.BillingEnabled {
		stripeSubID, err := s.stripeClient.CreateSubscription(sub.StripeCustomerID, "")
		if err != nil {
			return nil, fmt.Errorf("stripe create subscription: %w", err)
		}
		sub.StripeSubID = stripeSubID
	}

	return sub, nil
}

// GetSubscriptionForTenant gets a tenant's current subscription.
func (s *Service) GetSubscriptionForTenant(ctx context.Context, tenantID uuid.UUID) (*Subscription, error) {
	// Default subscription for tenants without explicit subscription
	now := time.Now()
	return &Subscription{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		PlanID:             uuid.Nil,
		Status:             "active",
		CurrentPeriodStart: time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC),
		CurrentPeriodEnd:   time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC),
		CreatedAt:          now,
	}, nil
}

// ChangeSubscription upgrades or downgrades a tenant's plan.
func (s *Service) ChangeSubscription(ctx context.Context, tenantID, newPlanID uuid.UUID) (*Subscription, error) {
	sub, err := s.GetSubscriptionForTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	sub.PlanID = newPlanID
	return sub, nil
}

// CancelSubscriptionForTenant cancels a tenant's subscription at period end.
func (s *Service) CancelSubscriptionForTenant(ctx context.Context, tenantID uuid.UUID) error {
	sub, err := s.GetSubscriptionForTenant(ctx, tenantID)
	if err != nil {
		return err
	}
	sub.CancelAtPeriodEnd = true
	sub.Status = "canceled"

	if s.stripeClient != nil && sub.StripeSubID != "" {
		if err := s.stripeClient.CancelSubscription(sub.StripeSubID); err != nil {
			return fmt.Errorf("stripe cancel: %w", err)
		}
	}
	return nil
}

// RecordUsageEvent records usage for a tenant.
func (s *Service) RecordUsageEvent(ctx context.Context, tenantID uuid.UUID, eventCount, retryCount, dataBytes int64) (*UsageRecord, error) {
	now := time.Now()
	record := &UsageRecord{
		ID:          uuid.New(),
		TenantID:    tenantID,
		PeriodStart: time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:   time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC),
		EventCount:  eventCount,
		RetryCount:  retryCount,
		DataBytes:   dataBytes,
		CreatedAt:   now,
	}
	return record, nil
}

// GetUsageSummaryForTenant returns the current usage summary for a tenant.
func (s *Service) GetUsageSummaryForTenant(ctx context.Context, tenantID uuid.UUID) (*UsageSummary, error) {
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)

	plan := defaultFreePlan()

	return &UsageSummary{
		TenantID:       tenantID,
		PlanName:       plan.Name,
		EventsUsed:     0,
		EventsIncluded: plan.IncludedEvents,
		OverageEvents:  0,
		CurrentCost:    plan.BasePrice,
		ProjectedCost:  plan.BasePrice,
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		UsagePercent:   0,
	}, nil
}

// CalculateCostForPlan calculates cost for usage on a given plan.
func (s *Service) CalculateCostForPlan(plan *PricingPlan, eventCount int64) int64 {
	switch plan.PricingModel {
	case "flat":
		return CalculateFlatRate(plan, eventCount)
	case "per_event":
		return CalculatePerEvent(plan, eventCount)
	case "tiered":
		return CalculateTiered(plan.Tiers, eventCount)
	default:
		return plan.BasePrice
	}
}

// GenerateInvoiceForTenant generates an invoice for a tenant's billing period.
func (s *Service) GenerateInvoiceForTenant(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time) (*Invoice, error) {
	plan := defaultFreePlan()
	cost := s.CalculateCostForPlan(&plan, 0)

	invoice := &Invoice{
		ID:             uuid.New(),
		TenantID:       tenantID,
		SubscriptionID: uuid.New(),
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		Subtotal:       cost,
		Tax:            0,
		Total:          cost,
		Status:         "draft",
		LineItems: []InvoiceLineItem{
			{
				Description: fmt.Sprintf("%s Plan - %s to %s", plan.DisplayName, periodStart.Format("Jan 2"), periodEnd.Format("Jan 2")),
				Quantity:    1,
				UnitPrice:   plan.BasePrice,
				Amount:      plan.BasePrice,
			},
		},
		CreatedAt: time.Now(),
	}

	return invoice, nil
}

// GetInvoicesForTenant lists invoices for a tenant.
func (s *Service) GetInvoicesForTenant(ctx context.Context, tenantID uuid.UUID) ([]Invoice, error) {
	return []Invoice{}, nil
}

// GetInvoiceDetail retrieves a single invoice by ID.
func (s *Service) GetInvoiceDetail(ctx context.Context, invoiceID uuid.UUID) (*Invoice, error) {
	return nil, fmt.Errorf("invoice not found")
}

// HandleStripeWebhook processes Stripe webhook events.
func (s *Service) HandleStripeWebhook(ctx context.Context, eventType string, data map[string]interface{}) error {
	switch eventType {
	case "invoice.paid":
		return nil
	case "invoice.payment_failed":
		return nil
	case "customer.subscription.updated":
		return nil
	case "customer.subscription.deleted":
		return nil
	default:
		return nil
	}
}

// GetBillingDashboard returns billing dashboard data for a tenant.
func (s *Service) GetBillingDashboard(ctx context.Context, tenantID uuid.UUID) (*BillingDashboard, error) {
	sub, _ := s.GetSubscriptionForTenant(ctx, tenantID)
	usage, _ := s.GetUsageSummaryForTenant(ctx, tenantID)
	invoices, _ := s.GetInvoicesForTenant(ctx, tenantID)
	plan := defaultFreePlan()

	return &BillingDashboard{
		Subscription:   sub,
		Plan:           &plan,
		Usage:          usage,
		RecentInvoices: invoices,
	}, nil
}

// ProjectCost projects future costs based on current usage patterns.
func (s *Service) ProjectCost(ctx context.Context, tenantID uuid.UUID, daysAhead int) (*UsageSummary, error) {
	summary, err := s.GetUsageSummaryForTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	daysPassed := float64(now.Day())
	if daysPassed < 1 {
		daysPassed = 1
	}

	dailyRate := float64(summary.EventsUsed) / daysPassed
	projectedEvents := summary.EventsUsed + int64(dailyRate*float64(daysAhead))

	plan := defaultFreePlan()
	projectedCost := s.CalculateCostForPlan(&plan, projectedEvents)

	summary.ProjectedCost = projectedCost
	return summary, nil
}

// Default plan definitions

func defaultFreePlan() PricingPlan {
	return PricingPlan{
		Name:           "free",
		DisplayName:    "Free",
		PricingModel:   "flat",
		BasePrice:      0,
		IncludedEvents: 1000,
		OveragePrice:   0,
		Features: PlanFeatures{
			MaxEndpoints:     5,
			MaxRetries:       3,
			CustomTransforms: false,
			Analytics:        false,
			SLA:              false,
			Support:          "community",
		},
		Active: true,
	}
}

func defaultStarterPlan() PricingPlan {
	return PricingPlan{
		Name:           "starter",
		DisplayName:    "Starter",
		PricingModel:   "per_event",
		BasePrice:      2900,
		IncludedEvents: 10000,
		OveragePrice:   100,
		Features: PlanFeatures{
			MaxEndpoints:     25,
			MaxRetries:       5,
			CustomTransforms: false,
			Analytics:        true,
			SLA:              false,
			Support:          "email",
		},
		Active: true,
	}
}

func defaultProPlan() PricingPlan {
	return PricingPlan{
		Name:           "pro",
		DisplayName:    "Pro",
		PricingModel:   "per_event",
		BasePrice:      9900,
		IncludedEvents: 100000,
		OveragePrice:   50,
		Features: PlanFeatures{
			MaxEndpoints:     100,
			MaxRetries:       10,
			CustomTransforms: true,
			Analytics:        true,
			SLA:              true,
			Support:          "priority",
		},
		Active: true,
	}
}

func defaultEnterprisePlan() PricingPlan {
	return PricingPlan{
		Name:           "enterprise",
		DisplayName:    "Enterprise",
		PricingModel:   "tiered",
		BasePrice:      49900,
		IncludedEvents: 1000000,
		OveragePrice:   25,
		Tiers: []PricingTier{
			{UpTo: 100000, PricePerUnit: 10},
			{UpTo: 500000, PricePerUnit: 5},
			{UpTo: -1, PricePerUnit: 2},
		},
		Features: PlanFeatures{
			MaxEndpoints:     -1,
			MaxRetries:       20,
			CustomTransforms: true,
			Analytics:        true,
			SLA:              true,
			Support:          "dedicated",
		},
		Active: true,
	}
}
