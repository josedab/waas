package marketplacetpl

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service provides marketplace functionality
type Service struct {
	repo Repository
}

// NewService creates a new marketplace service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateTemplate submits a new template to the marketplace
func (s *Service) CreateTemplate(ctx context.Context, req *CreateTemplateRequest) (*Template, error) {
	template := &Template{
		ID:            uuid.New().String(),
		Name:          req.Name,
		Description:   req.Description,
		Category:      req.Category,
		Source:        req.Source,
		Destination:   req.Destination,
		Transform:     req.Transform,
		RetryPolicy:   req.RetryPolicy,
		SamplePayload: req.SamplePayload,
		Version:       "1.0.0",
		Author:        "community",
		Tags:          req.Tags,
		IsVerified:    false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.repo.CreateTemplate(ctx, template); err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	return template, nil
}

// GetTemplate retrieves a template by ID
func (s *Service) GetTemplate(ctx context.Context, templateID string) (*Template, error) {
	return s.repo.GetTemplate(ctx, templateID)
}

// ListTemplates lists templates with optional category filter
func (s *Service) ListTemplates(ctx context.Context, category string, limit, offset int) ([]Template, int, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.repo.ListTemplates(ctx, category, limit, offset)
}

// SearchTemplates searches templates by query string
func (s *Service) SearchTemplates(ctx context.Context, query string, limit, offset int) ([]Template, int, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.repo.SearchTemplates(ctx, query, limit, offset)
}

// InstallTemplate installs a template for a tenant
func (s *Service) InstallTemplate(ctx context.Context, tenantID, templateID string, req *InstallTemplateRequest) (*Installation, error) {
	template, err := s.repo.GetTemplate(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}

	install := &Installation{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		TemplateID:  template.ID,
		EndpointID:  req.EndpointID,
		Config:      req.Config,
		Status:      InstallStatusActive,
		InstalledAt: time.Now(),
	}

	if err := s.repo.CreateInstallation(ctx, install); err != nil {
		return nil, fmt.Errorf("failed to install template: %w", err)
	}

	s.repo.IncrementInstallCount(ctx, templateID)

	return install, nil
}

// ListInstallations lists all template installations for a tenant
func (s *Service) ListInstallations(ctx context.Context, tenantID string) ([]Installation, error) {
	return s.repo.ListInstallations(ctx, tenantID)
}

// UninstallTemplate removes a template installation
func (s *Service) UninstallTemplate(ctx context.Context, tenantID, installID string) error {
	return s.repo.DeleteInstallation(ctx, tenantID, installID)
}

// SubmitReview submits a review for a template
func (s *Service) SubmitReview(ctx context.Context, tenantID, templateID string, req *SubmitReviewRequest) (*Review, error) {
	review := &Review{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		TemplateID: templateID,
		Rating:     req.Rating,
		Comment:    req.Comment,
		CreatedAt:  time.Now(),
	}

	if err := s.repo.CreateReview(ctx, review); err != nil {
		return nil, fmt.Errorf("failed to submit review: %w", err)
	}

	// Update average rating
	avgRating, _, err := s.repo.GetAverageRating(ctx, templateID)
	if err == nil {
		s.repo.UpdateTemplateRating(ctx, templateID, avgRating)
	}

	return review, nil
}

// ListReviews lists reviews for a template
func (s *Service) ListReviews(ctx context.Context, templateID string, limit, offset int) ([]Review, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.repo.ListReviews(ctx, templateID, limit, offset)
}

// GetStats returns marketplace statistics
func (s *Service) GetStats(ctx context.Context) (*MarketplaceStats, error) {
	return s.repo.GetStats(ctx)
}

// GetBuiltinTemplates returns the curated set of built-in templates
func (s *Service) GetBuiltinTemplates() []Template {
	return []Template{
		{
			ID: "builtin-stripe-slack", Name: "Stripe → Slack", Category: CategoryPayments,
			Source: "stripe", Destination: "slack", IsVerified: true,
			Description: "Forward Stripe payment events to a Slack channel",
			SamplePayload: `{"event":"payment_intent.succeeded","amount":9999}`,
		},
		{
			ID: "builtin-github-pagerduty", Name: "GitHub → PagerDuty", Category: CategoryDevOps,
			Source: "github", Destination: "pagerduty", IsVerified: true,
			Description: "Trigger PagerDuty incidents from GitHub Actions failures",
			SamplePayload: `{"action":"completed","conclusion":"failure"}`,
		},
		{
			ID: "builtin-shopify-email", Name: "Shopify → Email", Category: CategoryEcommerce,
			Source: "shopify", Destination: "email", IsVerified: true,
			Description: "Send email notifications for Shopify order events",
			SamplePayload: `{"topic":"orders/create","order_id":"12345"}`,
		},
		{
			ID: "builtin-stripe-hubspot", Name: "Stripe → HubSpot", Category: CategoryCRM,
			Source: "stripe", Destination: "hubspot", IsVerified: true,
			Description: "Sync Stripe subscription events to HubSpot CRM",
			SamplePayload: `{"event":"customer.subscription.created","customer":"cus_123"}`,
		},
		{
			ID: "builtin-datadog-slack", Name: "Datadog → Slack", Category: CategoryMonitoring,
			Source: "datadog", Destination: "slack", IsVerified: true,
			Description: "Forward Datadog alert notifications to Slack",
			SamplePayload: `{"alert_type":"error","title":"High CPU Usage"}`,
		},
	}
}
