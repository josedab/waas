package billing

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// PricingConfig defines pricing for resources
type PricingConfig struct {
	WebhookRequestCost float64 // per request
	RetryAttemptCost   float64 // per retry
	DataTransferCost   float64 // per GB
	TransformationCost float64 // per transform
	StorageCost        float64 // per GB per month
	Currency           string
}

// DefaultPricing provides default pricing
var DefaultPricing = PricingConfig{
	WebhookRequestCost: 0.0001,  // $0.10 per 1000 requests
	RetryAttemptCost:   0.00005, // $0.05 per 1000 retries
	DataTransferCost:   0.10,    // $0.10 per GB
	TransformationCost: 0.00002, // $0.02 per 1000 transforms
	StorageCost:        0.023,   // $0.023 per GB per month
	Currency:           "USD",
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
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		s.checkBudgetAlerts(ctx, tenantID)
	}()

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
